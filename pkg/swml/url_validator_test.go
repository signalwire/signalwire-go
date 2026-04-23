package swml

import (
	"testing"
)

// ---------------------------------------------------------------------------
// ValidateURL — SSRF guard
// ---------------------------------------------------------------------------
//
// Test coverage mirrors the Python SDK's logic in url_validator.py.
// DNS-dependent tests (those that resolve real hostnames) are kept to a
// minimum (localhost/127.0.0.1 resolve reliably in any environment without
// network access). Smoke tests for public URLs are skipped because CI
// environments may not have external DNS.

func TestValidateURL_RejectsNonHTTPSchemes(t *testing.T) {
	schemes := []string{
		"file:///etc/passwd",
		"ftp://example.com/file",
		"javascript:alert(1)",
		"data:text/html,<h1>hi</h1>",
	}
	for _, rawURL := range schemes {
		t.Run(rawURL, func(t *testing.T) {
			ok, err := ValidateURL(rawURL, false)
			if ok {
				t.Errorf("ValidateURL(%q) = true, want false", rawURL)
			}
			if err == nil {
				t.Errorf("ValidateURL(%q) returned nil error for invalid scheme", rawURL)
			}
		})
	}
}

func TestValidateURL_RejectsEmptyHostname(t *testing.T) {
	// url.Parse("http://") succeeds but hostname is "".
	ok, err := ValidateURL("http://", false)
	if ok {
		t.Errorf("ValidateURL(\"http://\") = true, want false")
	}
	if err == nil {
		t.Error("ValidateURL(\"http://\") returned nil error for empty hostname")
	}
}

func TestValidateURL_RejectsLoopbackIPv4(t *testing.T) {
	// 127.0.0.1 is in 127.0.0.0/8 — always blocked.
	ok, err := ValidateURL("http://127.0.0.1/secret", false)
	if ok {
		t.Errorf("ValidateURL(loopback IPv4) = true, want false")
	}
	if err == nil {
		t.Error("ValidateURL(loopback IPv4) returned nil error")
	}
}

func TestValidateURL_RejectsLoopbackIPv6(t *testing.T) {
	// ::1 — IPv6 loopback, blocked by ::1/128.
	ok, err := ValidateURL("http://[::1]/secret", false)
	if ok {
		t.Errorf("ValidateURL(loopback IPv6) = true, want false")
	}
	if err == nil {
		t.Error("ValidateURL(loopback IPv6) returned nil error")
	}
}

func TestValidateURL_RejectsRFC1918_10(t *testing.T) {
	ok, err := ValidateURL("http://10.0.0.1/internal", false)
	if ok {
		t.Errorf("ValidateURL(10.x) = true, want false")
	}
	if err == nil {
		t.Error("ValidateURL(10.x) returned nil error")
	}
}

func TestValidateURL_RejectsRFC1918_172(t *testing.T) {
	ok, err := ValidateURL("http://172.16.0.1/internal", false)
	if ok {
		t.Errorf("ValidateURL(172.16.x) = true, want false")
	}
	if err == nil {
		t.Error("ValidateURL(172.16.x) returned nil error")
	}
}

func TestValidateURL_RejectsRFC1918_192(t *testing.T) {
	ok, err := ValidateURL("http://192.168.1.1/internal", false)
	if ok {
		t.Errorf("ValidateURL(192.168.x) = true, want false")
	}
	if err == nil {
		t.Error("ValidateURL(192.168.x) returned nil error")
	}
}

func TestValidateURL_RejectsLinkLocal(t *testing.T) {
	// 169.254.x.x — cloud metadata endpoints (AWS IMDS, GCP, Azure).
	ok, err := ValidateURL("http://169.254.169.254/latest/meta-data/", false)
	if ok {
		t.Errorf("ValidateURL(link-local IMDS) = true, want false")
	}
	if err == nil {
		t.Error("ValidateURL(link-local IMDS) returned nil error")
	}
}

func TestValidateURL_AllowPrivateBypassesSSRFGuard(t *testing.T) {
	// allowPrivate=true short-circuits DNS resolution + CIDR checks.
	// 127.0.0.1 would normally be blocked, but allowPrivate overrides.
	ok, err := ValidateURL("http://127.0.0.1/", true)
	if !ok {
		t.Errorf("ValidateURL(loopback, allowPrivate=true) = false, want true; err: %v", err)
	}
	if err != nil {
		t.Errorf("ValidateURL(loopback, allowPrivate=true) unexpected error: %v", err)
	}
}

func TestValidateURL_EnvVarAllowsPrivate(t *testing.T) {
	for _, val := range []string{"1", "true", "yes", "TRUE", "YES"} {
		t.Run("SWML_ALLOW_PRIVATE_URLS="+val, func(t *testing.T) {
			t.Setenv("SWML_ALLOW_PRIVATE_URLS", val)
			ok, err := ValidateURL("http://127.0.0.1/", false)
			if !ok {
				t.Errorf("ValidateURL with env %q = false, want true; err: %v", val, err)
			}
			if err != nil {
				t.Errorf("ValidateURL with env %q unexpected error: %v", val, err)
			}
		})
	}
}

func TestValidateURL_RejectsLocalhost(t *testing.T) {
	// "localhost" resolves to 127.0.0.1 or ::1 — both in blocked ranges.
	ok, _ := ValidateURL("http://localhost/admin", false)
	if ok {
		t.Errorf("ValidateURL(localhost) = true, want false")
	}
}

// ---------------------------------------------------------------------------
// IsServerlessMode
// ---------------------------------------------------------------------------

func TestIsServerlessMode_FalseInServerMode(t *testing.T) {
	clearExecutionEnv(t)
	if IsServerlessMode() {
		t.Error("IsServerlessMode() = true in plain server mode, want false")
	}
}

func TestIsServerlessMode_TrueInLambdaMode(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "my-func")
	if !IsServerlessMode() {
		t.Error("IsServerlessMode() = false in Lambda mode, want true")
	}
}

func TestIsServerlessMode_TrueInCGIMode(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("GATEWAY_INTERFACE", "CGI/1.1")
	if !IsServerlessMode() {
		t.Error("IsServerlessMode() = false in CGI mode, want true")
	}
}

func TestIsServerlessMode_TrueInGCFMode(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("FUNCTION_TARGET", "myFunction")
	if !IsServerlessMode() {
		t.Error("IsServerlessMode() = false in GCF mode, want true")
	}
}

func TestIsServerlessMode_TrueInAzureMode(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("FUNCTIONS_WORKER_RUNTIME", "go")
	if !IsServerlessMode() {
		t.Error("IsServerlessMode() = false in Azure mode, want true")
	}
}
