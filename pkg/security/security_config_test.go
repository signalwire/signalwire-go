package security

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetSSLContextKwargsPrimitives asserts that the TLS config surfaces the
// cert/key paths as primitive path strings under the same keys the Python
// reference's get_ssl_context_kwargs returns ({ssl_certfile, ssl_keyfile}).
func TestGetSSLContextKwargsPrimitives(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")
	if err := os.WriteFile(certPath, []byte("cert"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}

	c := &SecurityConfig{
		SSLEnabled:  true,
		SSLCertPath: certPath,
		SSLKeyPath:  keyPath,
	}
	kwargs := c.GetSSLContextKwargs()

	cert, ok := kwargs["ssl_certfile"].(string)
	if !ok || cert != certPath {
		t.Fatalf("ssl_certfile = %v, want %s", kwargs["ssl_certfile"], certPath)
	}
	key, ok := kwargs["ssl_keyfile"].(string)
	if !ok || key != keyPath {
		t.Fatalf("ssl_keyfile = %v, want %s", kwargs["ssl_keyfile"], keyPath)
	}
}

// TestGetSSLContextKwargsDisabled mirrors Python: an empty map when SSL is off.
func TestGetSSLContextKwargsDisabled(t *testing.T) {
	c := &SecurityConfig{SSLEnabled: false}
	if got := c.GetSSLContextKwargs(); len(got) != 0 {
		t.Fatalf("GetSSLContextKwargs() with SSL disabled = %v, want empty", got)
	}
}

// ---------------------------------------------------------------------------
// Config-file layer (defaults -> env -> config file, config file HIGHEST).
// Mirrors the reference's SecurityConfig.__init__ ordering
// (_set_defaults(); load_from_env(); _load_config_file(...)).
// ---------------------------------------------------------------------------

// writeSecurityConfigFile writes a JSON config file with the given `security`
// body and returns its path.
func writeSecurityConfigFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "svc_config.json")
	if err := os.WriteFile(path, []byte(`{"security":`+body+`}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// clearSecurityEnv unsets the SWML_* vars these tests must not inherit.
func clearSecurityEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"SWML_SSL_ENABLED", "SWML_SSL_CERT_PATH", "SWML_SSL_KEY_PATH",
		"SWML_DOMAIN", "SWML_SSL_DOMAIN", "SWML_ALLOWED_HOSTS",
		"SWML_CORS_ORIGINS", "SWML_RATE_LIMIT", "SWML_USE_HSTS",
		"SWML_BASIC_AUTH_USER", "SWML_BASIC_AUTH_PASSWORD",
	} {
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("unset %s: %v", k, err)
		}
	}
}

// TestConfigFileSuppliesSSL: with NOTHING in the environment, the config file
// alone must turn TLS on and supply the cert/key. This is the resolution an
// operator relies on when they configure TLS through a file.
func TestConfigFileSuppliesSSL(t *testing.T) {
	clearSecurityEnv(t)
	path := writeSecurityConfigFile(t, `{
		"ssl_enabled": true,
		"ssl_cert_path": "/etc/ssl/from-file.crt",
		"ssl_key_path": "/etc/ssl/from-file.key",
		"domain": "file.example.com"
	}`)

	c := NewSecurityConfig(path, "")

	if !c.SSLEnabled {
		t.Error("SSLEnabled = false; the config file's ssl_enabled:true was ignored")
	}
	if c.SSLCertPath != "/etc/ssl/from-file.crt" {
		t.Errorf("SSLCertPath = %q, want /etc/ssl/from-file.crt", c.SSLCertPath)
	}
	if c.SSLKeyPath != "/etc/ssl/from-file.key" {
		t.Errorf("SSLKeyPath = %q, want /etc/ssl/from-file.key", c.SSLKeyPath)
	}
	if c.Domain != "file.example.com" {
		t.Errorf("Domain = %q, want file.example.com", c.Domain)
	}
	if c.GetURLScheme() != "https" {
		t.Errorf("GetURLScheme() = %q, want https", c.GetURLScheme())
	}
}

// TestConfigFileOverridesEnv pins the PRECEDENCE: the config file is the
// highest-priority layer, so its values beat the SWML_* env vars.
func TestConfigFileOverridesEnv(t *testing.T) {
	clearSecurityEnv(t)
	t.Setenv("SWML_SSL_ENABLED", "true")
	t.Setenv("SWML_SSL_CERT_PATH", "/from/env.crt")
	t.Setenv("SWML_SSL_KEY_PATH", "/from/env.key")
	t.Setenv("SWML_DOMAIN", "env.example.com")

	path := writeSecurityConfigFile(t, `{
		"ssl_cert_path": "/from/file.crt",
		"ssl_key_path": "/from/file.key",
		"domain": "file.example.com"
	}`)

	c := NewSecurityConfig(path, "")

	if c.SSLCertPath != "/from/file.crt" {
		t.Errorf("SSLCertPath = %q; the env value won over the config file", c.SSLCertPath)
	}
	if c.SSLKeyPath != "/from/file.key" {
		t.Errorf("SSLKeyPath = %q; the env value won over the config file", c.SSLKeyPath)
	}
	if c.Domain != "file.example.com" {
		t.Errorf("Domain = %q; the env value won over the config file", c.Domain)
	}
	// ssl_enabled is absent from the file, so the env value survives — a
	// partial security block overrides only the keys it names.
	if !c.SSLEnabled {
		t.Error("SSLEnabled = false; an absent config key wrongly reset the env value")
	}
}

// TestConfigFileSSLEnabledFalseOverridesEnv: an explicit `ssl_enabled: false`
// really does turn TLS off, even when the env asked for it. The reference
// distinguishes "absent" from "explicitly false"
// (`if "ssl_enabled" in security_config:`); so must this.
func TestConfigFileSSLEnabledFalseOverridesEnv(t *testing.T) {
	clearSecurityEnv(t)
	t.Setenv("SWML_SSL_ENABLED", "true")

	path := writeSecurityConfigFile(t, `{"ssl_enabled": false}`)

	c := NewSecurityConfig(path, "")
	if c.SSLEnabled {
		t.Error("SSLEnabled = true; the config file's explicit ssl_enabled:false was ignored")
	}
	if c.GetURLScheme() != "http" {
		t.Errorf("GetURLScheme() = %q, want http", c.GetURLScheme())
	}
}

// TestConfigFileNonSSLSection covers the rest of the security block: hosts,
// CORS, limits, HSTS and basic auth all resolve off the file too.
func TestConfigFileNonSSLSection(t *testing.T) {
	clearSecurityEnv(t)
	path := writeSecurityConfigFile(t, `{
		"allowed_hosts": ["a.example.com", "b.example.com"],
		"cors_origins": "https://app.example.com",
		"max_request_size": 2048,
		"rate_limit": 7,
		"request_timeout": 11,
		"use_hsts": false,
		"hsts_max_age": 60,
		"auth": {"basic": {"user": "alice", "password": "s3cr3t"}}
	}`)

	c := NewSecurityConfig(path, "")

	if len(c.AllowedHosts) != 2 || c.AllowedHosts[0] != "a.example.com" {
		t.Errorf("AllowedHosts = %v, want [a.example.com b.example.com]", c.AllowedHosts)
	}
	if c.ShouldAllowHost("evil.example.com") {
		t.Error("ShouldAllowHost(evil) = true; the config file's allow-list was ignored")
	}
	if !c.ShouldAllowHost("a.example.com") {
		t.Error("ShouldAllowHost(a.example.com) = false; the allow-list did not apply")
	}
	if len(c.CORSOrigins) != 1 || c.CORSOrigins[0] != "https://app.example.com" {
		t.Errorf("CORSOrigins = %v, want [https://app.example.com]", c.CORSOrigins)
	}
	if c.MaxRequestSize != 2048 {
		t.Errorf("MaxRequestSize = %d, want 2048", c.MaxRequestSize)
	}
	if c.RateLimit != 7 {
		t.Errorf("RateLimit = %d, want 7", c.RateLimit)
	}
	if c.RequestTimeout != 11 {
		t.Errorf("RequestTimeout = %d, want 11", c.RequestTimeout)
	}
	if c.UseHSTS {
		t.Error("UseHSTS = true; the config file's use_hsts:false was ignored")
	}
	if _, ok := c.GetSecurityHeaders(true)["Strict-Transport-Security"]; ok {
		t.Error("HSTS header present with use_hsts:false")
	}
	if c.BasicAuthUser != "alice" || c.BasicAuthPassword != "s3cr3t" {
		t.Errorf("basic auth = %q/%q, want alice/s3cr3t", c.BasicAuthUser, c.BasicAuthPassword)
	}
}

// TestConfigFileAbsentIsNoOp: no config file at all leaves the env resolution
// untouched (the reference's best-effort load), and a bad path does not panic.
func TestConfigFileAbsentIsNoOp(t *testing.T) {
	clearSecurityEnv(t)
	t.Setenv("SWML_SSL_ENABLED", "true")
	t.Setenv("SWML_SSL_CERT_PATH", "/from/env.crt")

	c := NewSecurityConfig(filepath.Join(t.TempDir(), "nope.json"), "")
	if !c.SSLEnabled || c.SSLCertPath != "/from/env.crt" {
		t.Errorf("a missing config file perturbed the env resolution: enabled=%v cert=%q",
			c.SSLEnabled, c.SSLCertPath)
	}
}

// TestDomainReadsCanonicalEnvVar: the reference names this var SWML_DOMAIN
// (security_config.py SSL_DOMAIN = "SWML_DOMAIN"), which is also what every doc
// page here tells operators to set. The code used to read only SWML_SSL_DOMAIN,
// so following the docs silently produced an empty domain.
func TestDomainReadsCanonicalEnvVar(t *testing.T) {
	clearSecurityEnv(t)
	t.Setenv("SWML_DOMAIN", "canonical.example.com")

	c := NewSecurityConfig("", "")
	if c.Domain != "canonical.example.com" {
		t.Errorf("Domain = %q, want canonical.example.com (SWML_DOMAIN was ignored)", c.Domain)
	}
}
