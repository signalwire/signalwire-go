// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Package swml — URL validation utility to prevent SSRF attacks.
// Ported from signalwire/utils/url_validator.py.
package swml

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

// blockedNetworks lists private/reserved IP ranges that should be blocked to
// prevent SSRF attacks. Mirrors the Python SDK's _BLOCKED_NETWORKS list:
//
//	10.0.0.0/8       — RFC 1918 private
//	172.16.0.0/12    — RFC 1918 private
//	192.168.0.0/16   — RFC 1918 private
//	127.0.0.0/8      — loopback
//	169.254.0.0/16   — link-local / cloud metadata (AWS IMDS, GCP, Azure)
//	0.0.0.0/8        — "this" network
//	::1/128          — IPv6 loopback
//	fc00::/7         — IPv6 unique-local (RFC 4193)
//	fe80::/10        — IPv6 link-local
var blockedNetworks []*net.IPNet

func init() {
	cidrs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"0.0.0.0/8",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			// Compile-time constants — panic on bad CIDR is intentional so
			// tests catch mis-edits immediately.
			panic(fmt.Sprintf("swml: bad blocked-network CIDR %q: %v", cidr, err))
		}
		blockedNetworks = append(blockedNetworks, network)
	}
}

// ValidateURL reports whether rawURL is safe to fetch (i.e. does not point to
// a private or internal resource). It returns an error describing why the URL
// was rejected, or nil if the URL is acceptable.
//
// Behavior mirrors Python's validate_url(url, allow_private=False):
//   - Only http and https schemes are accepted.
//   - A non-empty hostname is required.
//   - When allowPrivate is false AND the SWML_ALLOW_PRIVATE_URLS env var is
//     not set to "1", "true", or "yes" (case-insensitive), every IP address
//     that the hostname resolves to is checked against the nine blocked CIDR
//     ranges above. If any resolved IP falls in a blocked range the URL is
//     rejected.
//
// Go idiom: returns (bool, error) instead of a bare bool so callers can log
// or propagate the rejection reason. Returning (false, nil) never happens —
// err is always non-nil when the bool is false.
func ValidateURL(rawURL string, allowPrivate bool) (bool, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false, fmt.Errorf("url rejected: parse error: %w", err)
	}

	// Require http or https scheme.
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return false, fmt.Errorf("url rejected: invalid scheme %q", parsed.Scheme)
	}

	// Must have a hostname.
	hostname := parsed.Hostname()
	if hostname == "" {
		return false, fmt.Errorf("url rejected: no hostname")
	}

	// If allowPrivate is set, or the env-var override is active, skip
	// the SSRF-guard entirely (matches Python's short-circuit).
	envVal := strings.ToLower(os.Getenv("SWML_ALLOW_PRIVATE_URLS"))
	if allowPrivate || envVal == "1" || envVal == "true" || envVal == "yes" {
		return true, nil
	}

	// Resolve hostname to IP addresses (mirrors socket.getaddrinfo).
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return false, fmt.Errorf("url rejected: could not resolve hostname %q: %w", hostname, err)
	}

	// Check every resolved IP against the blocked CIDR ranges.
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			// Unparseable address — skip (mirrors Python's ValueError continue).
			continue
		}
		for _, network := range blockedNetworks {
			if network.Contains(ip) {
				return false, fmt.Errorf(
					"url rejected: %q resolves to blocked IP %s (in %s)",
					hostname, addr, network,
				)
			}
		}
	}

	return true, nil
}
