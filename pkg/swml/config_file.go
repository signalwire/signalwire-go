package swml

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// configFileSchema is the YAML structure read from a SecurityConfig-compatible
// config file. Only the security section is consumed; other sections are
// ignored so that callers can co-locate unrelated configuration in the same
// file. Mirrors signalwire/core/security_config.py _load_config_file.
type configFileSchema struct {
	Security struct {
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

// applyConfigFile reads the YAML file at path and applies its security
// section to s. If the file cannot be read or parsed, the service's logger
// records a warning and the function returns without mutating s — matching
// Python's "best-effort" behaviour where a missing config file is logged but
// service start-up continues.
//
// Note: applyConfigFile is invoked from inside the WithConfigFile option,
// which executes during NewService. At that point s.Logger is not yet
// allocated, so warnings are written to stderr via fmt.Fprintf.
func applyConfigFile(s *Service, path string) {
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
