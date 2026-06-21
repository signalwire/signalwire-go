package security

import (
	"reflect"
	"testing"
)

// ---------------------------------------------------------------------------
// FilterSensitiveHeaders
// ---------------------------------------------------------------------------

func TestFilterSensitiveHeaders_RemovesSensitiveCaseInsensitive(t *testing.T) {
	in := map[string]string{
		"Authorization":       "Bearer secret",
		"Cookie":              "session=abc",
		"X-Api-Key":           "key123",
		"Proxy-Authorization": "Basic xyz",
		"Set-Cookie":          "session=def",
		"Content-Type":        "application/json",
		"X-Request-Id":        "req-42",
	}
	got := FilterSensitiveHeaders(in)
	want := map[string]string{
		"Content-Type": "application/json",
		"X-Request-Id": "req-42",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FilterSensitiveHeaders() = %v, want %v", got, want)
	}
}

func TestFilterSensitiveHeaders_LowerAndUpperCaseKeysBothStripped(t *testing.T) {
	in := map[string]string{
		"authorization": "a",
		"AUTHORIZATION": "b", // distinct map key, also sensitive
		"x-api-key":     "c",
		"keep":          "d",
	}
	got := FilterSensitiveHeaders(in)
	if _, ok := got["authorization"]; ok {
		t.Errorf("lowercase authorization should be stripped")
	}
	if _, ok := got["AUTHORIZATION"]; ok {
		t.Errorf("uppercase AUTHORIZATION should be stripped")
	}
	if _, ok := got["x-api-key"]; ok {
		t.Errorf("x-api-key should be stripped")
	}
	if v, ok := got["keep"]; !ok || v != "d" {
		t.Errorf("non-sensitive 'keep' should be preserved, got %q ok=%v", v, ok)
	}
	if len(got) != 1 {
		t.Errorf("expected exactly 1 surviving header, got %d: %v", len(got), got)
	}
}

func TestFilterSensitiveHeaders_PreservesOriginalCasingOfKeptKeys(t *testing.T) {
	in := map[string]string{"X-Custom-Header": "v"}
	got := FilterSensitiveHeaders(in)
	if v, ok := got["X-Custom-Header"]; !ok || v != "v" {
		t.Fatalf("kept key should preserve original casing, got %v", got)
	}
}

func TestFilterSensitiveHeaders_ReturnsCopyNotMutatingInput(t *testing.T) {
	in := map[string]string{"Authorization": "secret", "Keep": "v"}
	got := FilterSensitiveHeaders(in)
	got["Keep"] = "mutated"
	if in["Keep"] != "v" {
		t.Fatalf("input map was mutated: %v", in)
	}
	if _, ok := in["Authorization"]; !ok {
		t.Fatalf("input map lost a key: %v", in)
	}
}

func TestFilterSensitiveHeaders_EmptyAndNil(t *testing.T) {
	if got := FilterSensitiveHeaders(nil); got == nil || len(got) != 0 {
		t.Errorf("nil input should yield non-nil empty map, got %#v", got)
	}
	if got := FilterSensitiveHeaders(map[string]string{}); got == nil || len(got) != 0 {
		t.Errorf("empty input should yield empty map, got %#v", got)
	}
}

// ---------------------------------------------------------------------------
// RedactURL
// ---------------------------------------------------------------------------

func TestRedactURL_MasksPassword(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"https with creds", "https://user:secret@host/path", "https://user:****@host/path"},
		{"http with creds", "http://alice:p4ssw0rd@example.com", "http://alice:****@example.com"},
		{"wss with creds", "wss://u:s@relay.signalwire.com/", "wss://u:****@relay.signalwire.com/"},
		{"no credentials unchanged", "https://example.com/path?x=1", "https://example.com/path?x=1"},
		{"userinfo without password unchanged", "https://user@host/path", "https://user@host/path"},
		{"empty unchanged", "", ""},
		{"non-url string unchanged", "just a string", "just a string"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := RedactURL(c.in); got != c.want {
				t.Errorf("RedactURL(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestRedactURL_DoesNotLeakSecret(t *testing.T) {
	got := RedactURL("https://user:topsecret@host")
	if got == "https://user:topsecret@host" {
		t.Fatalf("password was not redacted: %q", got)
	}
	if want := "https://user:****@host"; got != want {
		t.Fatalf("RedactURL = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// IsValidHostname
// ---------------------------------------------------------------------------

func TestIsValidHostname(t *testing.T) {
	cases := []struct {
		name string
		host string
		want bool
	}{
		{"plain hostname", "example.com", true},
		{"subdomain", "api.signalwire.com", true},
		{"with port", "example.com:8080", true},
		{"ipv4", "10.0.0.1", true},
		{"empty rejected", "", false},
		{"space rejected", "exa mple.com", false},
		{"leading space rejected", " example.com", false},
		{"forward slash rejected", "example.com/path", false},
		{"backslash rejected", "example.com\\path", false},
		{"tab rejected", "example\t.com", false},
		{"newline rejected", "example.com\n", false},
		{"null byte rejected", "example\x00.com", false},
		{"del control char rejected", "example\x7f.com", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsValidHostname(c.host); got != c.want {
				t.Errorf("IsValidHostname(%q) = %v, want %v", c.host, got, c.want)
			}
		})
	}
}
