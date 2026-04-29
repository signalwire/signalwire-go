package swml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWithConfigFile_AppliesSecuritySection verifies that WithConfigFile
// reads a YAML file's security section and applies each documented field
// to the underlying Service. This is the round-trip proof that
// WithConfigFile is real (not the no-op stub the audit flagged before).
func TestWithConfigFile_AppliesSecuritySection(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "service.yaml")
	yaml := `# Test SignalWire SDK config
security:
  ssl_cert_path: /tmp/cert.pem
  ssl_key_path: /tmp/key.pem
  domain: example.com
  auth:
    basic:
      user: alice
      password: s3cr3t
    bearer_token: tok-abc-123
    api_key: key-xyz-789
    api_key_header: X-Custom-Key
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	s := NewService(WithConfigFile(cfgPath))

	if s.tlsCertFile != "/tmp/cert.pem" {
		t.Errorf("tlsCertFile = %q; want /tmp/cert.pem", s.tlsCertFile)
	}
	if s.tlsKeyFile != "/tmp/key.pem" {
		t.Errorf("tlsKeyFile = %q; want /tmp/key.pem", s.tlsKeyFile)
	}
	if s.Domain != "example.com" {
		t.Errorf("Domain = %q; want example.com", s.Domain)
	}
	if s.basicAuthUser != "alice" {
		t.Errorf("basicAuthUser = %q; want alice", s.basicAuthUser)
	}
	if s.basicAuthPassword != "s3cr3t" {
		t.Errorf("basicAuthPassword = %q; want s3cr3t", s.basicAuthPassword)
	}
	if s.bearerToken != "tok-abc-123" {
		t.Errorf("bearerToken = %q; want tok-abc-123", s.bearerToken)
	}
	if s.apiKey != "key-xyz-789" {
		t.Errorf("apiKey = %q; want key-xyz-789", s.apiKey)
	}
	if s.apiKeyHeader != "X-Custom-Key" {
		t.Errorf("apiKeyHeader = %q; want X-Custom-Key", s.apiKeyHeader)
	}
}

// TestWithConfigFile_DefaultsAPIKeyHeader confirms that omitting
// api_key_header in the YAML causes the loader to default to "X-API-Key",
// matching Python's SecurityConfig default.
func TestWithConfigFile_DefaultsAPIKeyHeader(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "service.yaml")
	yaml := `security:
  auth:
    api_key: only-the-key
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	s := NewService(WithConfigFile(cfgPath))

	if s.apiKey != "only-the-key" {
		t.Errorf("apiKey = %q; want only-the-key", s.apiKey)
	}
	if s.apiKeyHeader != "X-API-Key" {
		t.Errorf("apiKeyHeader = %q; want default X-API-Key", s.apiKeyHeader)
	}
}

// TestWithConfigFile_MissingFile_NoCrash verifies that a missing config
// file does NOT crash NewService — it logs to stderr and continues with
// the previously-set values. Mirrors Python's "best-effort" load.
func TestWithConfigFile_MissingFile_NoCrash(t *testing.T) {
	// Capture stderr so the warning doesn't pollute test output AND so we
	// can assert the warning was emitted.
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w

	s := NewService(
		WithBasicAuth("preset-user", "preset-password"),
		WithConfigFile("/nonexistent/path/to/file.yaml"),
	)

	w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	stderr := string(buf[:n])

	if !strings.Contains(stderr, "WithConfigFile") || !strings.Contains(stderr, "failed to read") {
		t.Errorf("expected stderr warning about failed read; got %q", stderr)
	}
	// preset values must survive the failed config-file load
	if s.basicAuthUser != "preset-user" {
		t.Errorf("basicAuthUser = %q; want preset-user (preserved)", s.basicAuthUser)
	}
	if s.basicAuthPassword != "preset-password" {
		t.Errorf("basicAuthPassword = %q; want preset-password (preserved)", s.basicAuthPassword)
	}
}

// TestWithConfigFile_InvalidYAML_NoCrash verifies that a malformed YAML
// file is also handled gracefully.
func TestWithConfigFile_InvalidYAML_NoCrash(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "broken.yaml")
	if err := os.WriteFile(cfgPath, []byte("security: : : not valid yaml :"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w

	s := NewService(WithBearerToken("preset-token"), WithConfigFile(cfgPath))

	w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	stderr := string(buf[:n])

	if !strings.Contains(stderr, "WithConfigFile") || !strings.Contains(stderr, "parse YAML") {
		t.Errorf("expected stderr warning about YAML parse; got %q", stderr)
	}
	if s.bearerToken != "preset-token" {
		t.Errorf("bearerToken = %q; want preset-token (preserved)", s.bearerToken)
	}
}

// TestWithConfigFile_EmptyPath_NoOp verifies that WithConfigFile("") is
// silently ignored — useful for callers passing an env-var value that
// may be unset.
func TestWithConfigFile_EmptyPath_NoOp(t *testing.T) {
	s := NewService(WithBasicAuth("u", "p"), WithConfigFile(""))
	if s.basicAuthUser != "u" || s.basicAuthPassword != "p" {
		t.Errorf("WithConfigFile(\"\") perturbed previously-set basic auth: %q/%q",
			s.basicAuthUser, s.basicAuthPassword)
	}
}
