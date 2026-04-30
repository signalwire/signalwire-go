// Parity tests for ValidateURL (projects onto Python
// signalwire.utils.url_validator.validate_url).
//
// Mirrors signalwire-python/tests/unit/utils/test_url_validator.py.
// DNS resolution is stubbed via resolveHost so the suite is hermetic.
package util

import (
	"errors"
	"net"
	"os"
	"testing"
)

func withResolver(t *testing.T, ips []net.IP, err error) func() {
	t.Helper()
	prev := resolveHost
	resolveHost = func(string) ([]net.IP, error) {
		if err != nil {
			return nil, err
		}
		return ips, nil
	}
	return func() { resolveHost = prev }
}

func withEnv(t *testing.T, key, value string) func() {
	t.Helper()
	prev, had := os.LookupEnv(key)
	if value == "" {
		_ = os.Unsetenv(key)
	} else {
		_ = os.Setenv(key, value)
	}
	return func() {
		if had {
			_ = os.Setenv(key, prev)
		} else {
			_ = os.Unsetenv(key)
		}
	}
}

// --- Scheme ----------------------------------------------------------------

func TestValidateURL_HTTPSchemeAllowed(t *testing.T) {
	defer withResolver(t, []net.IP{net.ParseIP("1.2.3.4")}, nil)()
	if !ValidateURL("http://example.com", false) {
		t.Fatal("http scheme should be allowed for public IP")
	}
}

func TestValidateURL_HTTPSSchemeAllowed(t *testing.T) {
	defer withResolver(t, []net.IP{net.ParseIP("1.2.3.4")}, nil)()
	if !ValidateURL("https://example.com", false) {
		t.Fatal("https scheme should be allowed for public IP")
	}
}

func TestValidateURL_FTPSchemeRejected(t *testing.T) {
	if ValidateURL("ftp://example.com", false) {
		t.Fatal("ftp scheme must be rejected")
	}
}

func TestValidateURL_FileSchemeRejected(t *testing.T) {
	if ValidateURL("file:///etc/passwd", false) {
		t.Fatal("file scheme must be rejected")
	}
}

func TestValidateURL_JavaScriptSchemeRejected(t *testing.T) {
	if ValidateURL("javascript:alert(1)", false) {
		t.Fatal("javascript scheme must be rejected")
	}
}

// --- Hostname --------------------------------------------------------------

func TestValidateURL_NoHostnameRejected(t *testing.T) {
	if ValidateURL("http://", false) {
		t.Fatal("URL with no hostname must be rejected")
	}
}

func TestValidateURL_HostnameUnresolvableRejected(t *testing.T) {
	defer withResolver(t, nil, errors.New("gaierror: no such host"))()
	if ValidateURL("http://nonexistent.invalid", false) {
		t.Fatal("unresolvable hostname must be rejected")
	}
}

// --- Blocked ranges --------------------------------------------------------

func TestValidateURL_LoopbackIPv4Rejected(t *testing.T) {
	defer withResolver(t, []net.IP{net.ParseIP("127.0.0.1")}, nil)()
	if ValidateURL("http://localhost", false) {
		t.Fatal("127.0.0.1 must be rejected (loopback)")
	}
}

func TestValidateURL_RFC1918_10_Rejected(t *testing.T) {
	defer withResolver(t, []net.IP{net.ParseIP("10.0.0.5")}, nil)()
	if ValidateURL("http://internal", false) {
		t.Fatal("10.0.0.5 must be rejected (RFC1918)")
	}
}

func TestValidateURL_RFC1918_192_Rejected(t *testing.T) {
	defer withResolver(t, []net.IP{net.ParseIP("192.168.1.1")}, nil)()
	if ValidateURL("http://router", false) {
		t.Fatal("192.168.1.1 must be rejected (RFC1918)")
	}
}

func TestValidateURL_RFC1918_172_Rejected(t *testing.T) {
	defer withResolver(t, []net.IP{net.ParseIP("172.16.0.1")}, nil)()
	if ValidateURL("http://corp", false) {
		t.Fatal("172.16.0.1 must be rejected (RFC1918)")
	}
}

func TestValidateURL_LinkLocalMetadataRejected(t *testing.T) {
	defer withResolver(t, []net.IP{net.ParseIP("169.254.169.254")}, nil)()
	if ValidateURL("http://metadata", false) {
		t.Fatal("169.254.169.254 must be rejected (cloud metadata)")
	}
}

func TestValidateURL_ZeroIPRejected(t *testing.T) {
	defer withResolver(t, []net.IP{net.ParseIP("0.0.0.0")}, nil)()
	if ValidateURL("http://void", false) {
		t.Fatal("0.0.0.0 must be rejected (0.0.0.0/8)")
	}
}

func TestValidateURL_IPv6LoopbackRejected(t *testing.T) {
	defer withResolver(t, []net.IP{net.ParseIP("::1")}, nil)()
	if ValidateURL("http://[::1]", false) {
		t.Fatal("::1 must be rejected (IPv6 loopback)")
	}
}

func TestValidateURL_IPv6LinkLocalRejected(t *testing.T) {
	defer withResolver(t, []net.IP{net.ParseIP("fe80::1")}, nil)()
	if ValidateURL("http://link-local", false) {
		t.Fatal("fe80::1 must be rejected (IPv6 link-local)")
	}
}

func TestValidateURL_IPv6PrivateRejected(t *testing.T) {
	defer withResolver(t, []net.IP{net.ParseIP("fc00::1")}, nil)()
	if ValidateURL("http://ipv6-private", false) {
		t.Fatal("fc00::1 must be rejected (IPv6 ULA)")
	}
}

func TestValidateURL_PublicIPAllowed(t *testing.T) {
	defer withResolver(t, []net.IP{net.ParseIP("8.8.8.8")}, nil)()
	if !ValidateURL("http://dns.google", false) {
		t.Fatal("8.8.8.8 must be allowed (public)")
	}
}

// --- allow_private bypass --------------------------------------------------

func TestValidateURL_AllowPrivateParamBypassesCheck(t *testing.T) {
	// No resolver override — bypass means we never call DNS at all.
	if !ValidateURL("http://10.0.0.5", true) {
		t.Fatal("allow_private=true must bypass IP check")
	}
}

func TestValidateURL_EnvVarBypassesCheck(t *testing.T) {
	defer withEnv(t, "SWML_ALLOW_PRIVATE_URLS", "true")()
	if !ValidateURL("http://10.0.0.5", false) {
		t.Fatal("SWML_ALLOW_PRIVATE_URLS=true must bypass IP check")
	}
}

func TestValidateURL_EnvVarYesBypassesCheck(t *testing.T) {
	defer withEnv(t, "SWML_ALLOW_PRIVATE_URLS", "YES")()
	if !ValidateURL("http://10.0.0.5", false) {
		t.Fatal("SWML_ALLOW_PRIVATE_URLS=YES must bypass IP check (case-insensitive)")
	}
}

func TestValidateURL_EnvVar1BypassesCheck(t *testing.T) {
	defer withEnv(t, "SWML_ALLOW_PRIVATE_URLS", "1")()
	if !ValidateURL("http://10.0.0.5", false) {
		t.Fatal("SWML_ALLOW_PRIVATE_URLS=1 must bypass IP check")
	}
}

func TestValidateURL_EnvVarFalseDoesNotBypass(t *testing.T) {
	defer withEnv(t, "SWML_ALLOW_PRIVATE_URLS", "false")()
	defer withResolver(t, []net.IP{net.ParseIP("10.0.0.5")}, nil)()
	if ValidateURL("http://internal", false) {
		t.Fatal("SWML_ALLOW_PRIVATE_URLS=false must not bypass IP check")
	}
}

func TestValidateURL_BlockedNetworksHasAllNine(t *testing.T) {
	if got, want := len(blockedNetworks), 9; got != want {
		t.Fatalf("blockedNetworks len = %d, want %d (all 9 SSRF ranges)", got, want)
	}
}
