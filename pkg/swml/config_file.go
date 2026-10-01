package swml

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/signalwire/signalwire-go/v3/pkg/security"
)

// applyResolvedSecurity hoists a fully-resolved security.SecurityConfig onto the
// service. SecurityConfig has already applied the reference's precedence —
// defaults, then SWML_* env vars, then the config file (highest) — so this is
// the Go spelling of SWMLService.__init__'s
//
//	self.ssl_enabled   = self.security.ssl_enabled
//	self.domain        = self.security.domain
//	self.ssl_cert_path = self.security.ssl_cert_path
//	self.ssl_key_path  = self.security.ssl_key_path
//
// Explicit constructor options win over the resolved values (a value already set
// by WithTLS / WithDomain / WithBasicAuth is not overwritten), matching the
// reference's serve(ssl_enabled=…, ssl_cert=…) overrides.
func (s *Service) applyResolvedSecurity(cfg *security.SecurityConfig) {
	if s.Domain == "" {
		s.Domain = cfg.Domain
	}
	if s.tlsCertFile == "" {
		s.tlsCertFile = cfg.SSLCertPath
	}
	if s.tlsKeyFile == "" {
		s.tlsKeyFile = cfg.SSLKeyPath
	}
	s.sslEnabled = cfg.SSLEnabled
	if s.basicAuthUser == "" {
		s.basicAuthUser = cfg.BasicAuthUser
	}
	if s.basicAuthPassword == "" {
		s.basicAuthPassword = cfg.BasicAuthPassword
	}
}

// configFileSchema is the YAML structure read from a SecurityConfig-compatible
// config file. Only the security section is consumed; other sections are
// ignored so that callers can co-locate unrelated configuration in the same
// file. Mirrors signalwire/core/security_config.py _load_config_file.
//
// The reference's config files are JSON; yaml.v3 parses JSON as a strict subset,
// so both spellings work here. The canonical JSON keys are also read by
// security.SecurityConfig, which is what supplies ssl_enabled and the rest of
// the security block; this struct covers the auth fields that are specific to
// the Go service (bearer token / API key) and are not part of the reference's
// SecurityConfig.
type configFileSchema struct {
	Security struct {
		SSLEnabled  *bool  `yaml:"ssl_enabled"`
		SSLCertPath string `yaml:"ssl_cert_path"`
		SSLKeyPath  string `yaml:"ssl_key_path"`
		Domain      string `yaml:"domain"`
		Auth        struct {
			Basic struct {
				User     string `yaml:"user"`
				Password string `yaml:"password"`
			} `yaml:"basic"`
			BearerToken  string `yaml:"bearer_token"`
			APIKey       string `yaml:"api_key"`
			APIKeyHeader string `yaml:"api_key_header"`
		} `yaml:"auth"`
	} `yaml:"security"`
}

// applyConfigFile reads the YAML/JSON file at path and applies its security
// section to s. If the file cannot be read or parsed, the service's logger
// records a warning and the function returns without mutating s — matching
// Python's "best-effort" behaviour where a missing config file is logged but
// service start-up continues.
//
// Note: applyConfigFile is invoked from NewService. At that point s.Logger may
// not be allocated yet, so warnings are written to stderr via fmt.Fprintf.
func applyConfigFile(s *Service, path string) {
	//nolint:gosec // G304: path is the operator-supplied WithConfigFile argument,
	// not attacker input — reading the configured file is the intended behavior.
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"signalwire-go: WithConfigFile(%q) failed to read file: %v; ignoring\n",
			path, err)
		return
	}
	var cfg configFileSchema
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr,
			"signalwire-go: WithConfigFile(%q) failed to parse YAML: %v; ignoring\n",
			path, err)
		return
	}

	sec := cfg.Security
	if sec.SSLCertPath != "" {
		s.tlsCertFile = sec.SSLCertPath
	}
	if sec.SSLKeyPath != "" {
		s.tlsKeyFile = sec.SSLKeyPath
	}
	// ssl_enabled is a pointer so an absent key leaves the env/default value
	// alone while an explicit `ssl_enabled: false` really does turn TLS off —
	// the reference distinguishes the two the same way
	// (`if "ssl_enabled" in security_config:`).
	if sec.SSLEnabled != nil {
		s.sslEnabled = *sec.SSLEnabled
		// An explicit config-file switch is authoritative over an earlier
		// WithTLS: it is the operator's highest-priority statement of intent,
		// matching the reference's config-file-wins precedence.
		s.sslEnabledExplicit = *sec.SSLEnabled
	}
	if sec.Domain != "" {
		s.Domain = sec.Domain
	}
	if sec.Auth.Basic.User != "" {
		s.basicAuthUser = sec.Auth.Basic.User
	}
	if sec.Auth.Basic.Password != "" {
		s.basicAuthPassword = sec.Auth.Basic.Password
	}
	if sec.Auth.BearerToken != "" {
		s.bearerToken = sec.Auth.BearerToken
	}
	if sec.Auth.APIKey != "" {
		s.apiKey = sec.Auth.APIKey
		hdr := sec.Auth.APIKeyHeader
		if hdr == "" {
			hdr = "X-API-Key"
		}
		s.apiKeyHeader = hdr
	}
}
