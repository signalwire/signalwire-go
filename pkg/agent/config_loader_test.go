// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Tests for the public agent.ConfigLoader façade — the Go spelling of
// signalwire.core.config_loader.ConfigLoader. The mechanics live in
// internal/config (shared with pkg/security so the two cannot drift); these
// tests pin the public behaviour, including the ${VAR|default} substitution and
// its type coercion, so a change to the shared loader that broke this surface
// would be caught here rather than in a downstream service.

package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func writeJSON(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile %s: %v", name, err)
	}
	return path
}

func TestConfigLoader_LoadsAndReadsSections(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "svc.json", `{
		"service": {"name": "my-service", "port": 8080},
		"security": {"ssl_enabled": true, "ssl_cert_path": "/c.pem"}
	}`)

	c := NewConfigLoader([]string{path})

	if !c.HasConfig() {
		t.Fatal("HasConfig() = false after loading a valid file")
	}
	if c.GetConfigFile() != path {
		t.Errorf("GetConfigFile() = %q, want %q", c.GetConfigFile(), path)
	}
	if got := c.Get("service.name", nil); got != "my-service" {
		t.Errorf("Get(service.name) = %v, want my-service", got)
	}
	if got := c.Get("service.missing", "fallback"); got != "fallback" {
		t.Errorf("Get(missing) = %v, want the default", got)
	}
	sec := c.GetSection("security")
	if sec["ssl_enabled"] != true {
		t.Errorf("GetSection(security)[ssl_enabled] = %v, want true", sec["ssl_enabled"])
	}
	if len(c.GetSection("nope")) != 0 {
		t.Error("GetSection of an absent section is not empty")
	}
	if len(c.GetConfig()) != 2 {
		t.Errorf("GetConfig() has %d top-level keys, want 2", len(c.GetConfig()))
	}
	if paths := c.ConfigPaths(); len(paths) != 1 || paths[0] != path {
		t.Errorf("ConfigPaths() = %v, want [%s]", paths, path)
	}
}

// TestConfigLoader_SubstituteVars pins the ${VAR|default} syntax AND the
// scalar-coercion the reference performs (a resolved "true"/"8080" becomes a
// bool/int, not a string).
func TestConfigLoader_SubstituteVars(t *testing.T) {
	t.Setenv("CFGTEST_PRESENT", "from-env")

	c := NewConfigLoader([]string{filepath.Join(t.TempDir(), "absent.json")})

	if got := c.SubstituteVars("${CFGTEST_PRESENT}", 10); got != "from-env" {
		t.Errorf("substitution of a set var = %v, want from-env", got)
	}
	if got := c.SubstituteVars("${CFGTEST_ABSENT|fallback}", 10); got != "fallback" {
		t.Errorf("substitution of an unset var = %v, want fallback", got)
	}
	if got := c.SubstituteVars("${CFGTEST_ABSENT|true}", 10); got != true {
		t.Errorf("bool coercion = %#v, want true (bool)", got)
	}
	if got := c.SubstituteVars("${CFGTEST_ABSENT|8080}", 10); got != 8080 {
		t.Errorf("int coercion = %#v, want 8080 (int)", got)
	}
	nested := c.SubstituteVars(map[string]any{
		"a": "${CFGTEST_PRESENT}",
		"b": []any{"${CFGTEST_ABSENT|x}"},
	}, 10)
	m, ok := nested.(map[string]any)
	if !ok {
		t.Fatalf("nested substitution returned %T, want map", nested)
	}
	if m["a"] != "from-env" {
		t.Errorf("nested map value = %v, want from-env", m["a"])
	}
	list, ok := m["b"].([]any)
	if !ok || len(list) != 1 || list[0] != "x" {
		t.Errorf("nested slice value = %v, want [x]", m["b"])
	}
}

// TestConfigLoader_FirstExistingFileWins mirrors the reference: the loader walks
// the search path and stops at the first readable, parseable file.
func TestConfigLoader_FirstExistingFileWins(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.json")
	broken := writeJSON(t, dir, "broken.json", `{not json`)
	good := writeJSON(t, dir, "good.json", `{"service": {"name": "second"}}`)

	// A missing file and an unparseable one are both skipped, not fatal.
	c := NewConfigLoader([]string{missing, broken, good})
	if c.GetConfigFile() != good {
		t.Errorf("GetConfigFile() = %q, want %q (missing/broken should be skipped)",
			c.GetConfigFile(), good)
	}
	if got := c.Get("service.name", nil); got != "second" {
		t.Errorf("Get(service.name) = %v, want second", got)
	}
}

func TestConfigLoader_NoConfigIsEmptyNotNil(t *testing.T) {
	c := NewConfigLoader([]string{filepath.Join(t.TempDir(), "nope.json")})
	if c.HasConfig() {
		t.Error("HasConfig() = true with no readable file")
	}
	if c.GetConfigFile() != "" {
		t.Errorf("GetConfigFile() = %q, want empty", c.GetConfigFile())
	}
	if c.GetConfig() == nil || len(c.GetConfig()) != 0 {
		t.Errorf("GetConfig() = %v, want a non-nil empty map", c.GetConfig())
	}
	if c.Get("anything", "def") != "def" {
		t.Error("Get on an unloaded config did not return the default")
	}
}

// TestConfigLoader_MergeWithEnv: config-file values win; a prefixed env var
// whose nested key is absent from the config is folded in.
func TestConfigLoader_MergeWithEnv(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "m.json", `{"ssl": {"enabled": "from-file"}}`)
	t.Setenv("SWML_SSL_ENABLED", "from-env")
	t.Setenv("SWML_NEW_KEY", "folded")

	merged := NewConfigLoader([]string{path}).MergeWithEnv("SWML_")

	ssl, ok := merged["ssl"].(map[string]any)
	if !ok {
		t.Fatalf("merged[ssl] = %T, want map", merged["ssl"])
	}
	if ssl["enabled"] != "from-file" {
		t.Errorf("merged ssl.enabled = %v; the env value beat the config file", ssl["enabled"])
	}
	newKey, ok := merged["new"].(map[string]any)
	if !ok {
		t.Fatalf("merged[new] = %T, want map (SWML_NEW_KEY should fold to new.key)", merged["new"])
	}
	if newKey["key"] != "folded" {
		t.Errorf("merged new.key = %v, want folded", newKey["key"])
	}
}

// TestFindConfigFile: service-specific names are searched before the generic
// defaults (Python: ConfigLoader.find_config_file).
func TestFindConfigFile(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	if got := FindConfigFile("mysvc", nil); got != "" {
		t.Errorf("FindConfigFile with nothing on disk = %q, want empty", got)
	}

	writeJSON(t, dir, "config.json", `{}`)
	if got := FindConfigFile("mysvc", nil); got != "config.json" {
		t.Errorf("FindConfigFile fell back to %q, want config.json", got)
	}

	// The service-specific name outranks the generic default.
	writeJSON(t, dir, "mysvc_config.json", `{}`)
	if got := FindConfigFile("mysvc", nil); got != "mysvc_config.json" {
		t.Errorf("FindConfigFile = %q, want mysvc_config.json (service-specific wins)", got)
	}
}

// TestConfigLoader_OmittedDefaults covers the DECLINED-argument path of the two
// wrappers that carry a non-zero reference default. Every other test in this
// file supplies the argument explicitly, which proves nothing about what an
// omitting caller gets — and the two defaults below are exactly where the port
// used to diverge (a zero maxDepth returned the value UNSUBSTITUTED; an empty
// envPrefix matched EVERY environment variable).
func TestConfigLoader_OmittedDefaults(t *testing.T) {
	t.Run("substitute_vars max_depth omitted defaults to 10", func(t *testing.T) {
		t.Setenv("CFGTEST_OMIT", "resolved")
		c := NewConfigLoader([]string{filepath.Join(t.TempDir(), "absent.json")})

		// The reference's `substitute_vars(value)` — max_depth omitted.
		if got := c.SubstituteVars("${CFGTEST_OMIT}", 0); got != "resolved" {
			t.Errorf("SubstituteVars with maxDepth omitted = %v, want resolved "+
				"(the reference's max_depth=10 default must apply)", got)
		}
		// It must agree with the explicit reference default.
		if got, want := c.SubstituteVars("${CFGTEST_OMIT}", 0),
			c.SubstituteVars("${CFGTEST_OMIT}", 10); got != want {
			t.Errorf("omitted maxDepth = %v, explicit 10 = %v; they must agree", got, want)
		}
	})

	t.Run("merge_with_env env_prefix omitted defaults to SWML_", func(t *testing.T) {
		dir := t.TempDir()
		path := writeJSON(t, dir, "m.json", `{"ssl": {"enabled": "from-file"}}`)
		t.Setenv("SWML_OMIT_KEY", "folded")

		// The reference's `merge_with_env()` — env_prefix omitted.
		merged := NewConfigLoader([]string{path}).MergeWithEnv("")

		omit, ok := merged["omit"].(map[string]any)
		if !ok {
			t.Fatalf("merged[omit] = %T, want map — the omitted env_prefix must "+
				"default to SWML_ so SWML_OMIT_KEY folds to omit.key", merged["omit"])
		}
		if omit["key"] != "folded" {
			t.Errorf("merged omit.key = %v, want folded", omit["key"])
		}
		// An omitted prefix must NOT behave like a bare "" prefix, which would
		// fold every unrelated environment variable into the merged config.
		if _, leaked := merged["path"]; leaked {
			t.Error("omitted env_prefix folded in PATH — it matched every env var " +
				"instead of defaulting to SWML_")
		}
	})
}
