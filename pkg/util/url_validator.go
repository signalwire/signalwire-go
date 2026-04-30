// Package util provides cross-cutting helpers used across the Go SDK.
//
// validate_url is the SSRF-prevention guard applied to user-supplied
// URLs before they are fetched.  It must mirror the Python reference
// at signalwire.utils.url_validator.validate_url:
//
//   - require http or https scheme
//   - require a hostname
//   - allow_private bypass (param OR SWML_ALLOW_PRIVATE_URLS env var)
//   - resolve hostname; reject any IP that lands in a blocked network
//
// The blocked-network list is identical across all SDK ports.
package util

import (
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/signalwire/signalwire-go/pkg/logging"
)

// blockedNetworks lists the private / loopback / link-local / cloud-
// metadata ranges every SDK port must reject.  Order matches the
// Python reference for ease of cross-port review.
var blockedNetworks = func() []*net.IPNet {
	cidrs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16", // link-local / cloud metadata
		"0.0.0.0/8",
		"::1/128",
		"fc00::/7",  // IPv6 private (ULA)
		"fe80::/10", // IPv6 link-local
	}
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err == nil {
			out = append(out, n)
		}
	}
	return out
}()

var urlValidatorLogger = logging.New("signalwire.url_validator")

// resolveHost is overridable so tests can inject a fake DNS resolver.
var resolveHost = func(hostname string) ([]net.IP, error) {
	return net.LookupIP(hostname)
}

// ValidateURL reports whether the supplied URL is safe to fetch.
//
// Mirrors Python's validate_url(url, allow_private=False) -> bool.
// Returns false (without raising) for any of:
//
//   - parse failure
//   - scheme not http/https
//   - missing hostname
//   - DNS resolution failure
//   - any resolved IP in a blocked network
//
// When allowPrivate is true, or the SWML_ALLOW_PRIVATE_URLS env var is
// set to "1", "true" or "yes" (case-insensitive), the IP-blocklist
// check is skipped.  Scheme + hostname checks still apply.
//
// This function is projected onto the Python free function name
// validate_url via internal/surface/tables.go.
func ValidateURL(url_ string, allowPrivate bool) bool {
	parsed, err := url.Parse(url_)
	if err != nil {
		urlValidatorLogger.Warn("URL validation error: %v", err)
		return false
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		urlValidatorLogger.Warn("URL rejected: invalid scheme %s", parsed.Scheme)
		return false
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		urlValidatorLogger.Warn("URL rejected: no hostname")
		return false
	}

	if allowPrivate || envAllowsPrivate() {
		return true
	}

	ips, err := resolveHost(hostname)
	if err != nil {
		urlValidatorLogger.Warn("URL rejected: could not resolve hostname %s", hostname)
		return false
	}

	for _, ip := range ips {
		for _, net_ := range blockedNetworks {
			if net_.Contains(ip) {
				urlValidatorLogger.Warn(
					"URL rejected: %s resolves to blocked IP %s (in %s)",
					hostname, ip.String(), net_.String(),
				)
				return false
			}
		}
	}

	return true
}

func envAllowsPrivate() bool {
	v := strings.ToLower(os.Getenv("SWML_ALLOW_PRIVATE_URLS"))
	return v == "1" || v == "true" || v == "yes"
}
