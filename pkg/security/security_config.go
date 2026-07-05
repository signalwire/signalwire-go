package security

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/signalwire/signalwire-go/pkg/logging"
)

// SecurityConfig centralises the SDK's HTTP security settings (SSL, allowed
// hosts, CORS, security headers, HSTS, basic auth), loaded from SWML_* env vars
// with sane defaults. Mirrors signalwire.core.security_config.SecurityConfig.
type SecurityConfig struct {
	SSLEnabled    bool
	SSLCertPath   string
	SSLKeyPath    string
	Domain        string
	SSLVerifyMode string

	AllowedHosts   []string
	CORSOrigins    []string
	MaxRequestSize int
	RateLimit      int
	RequestTimeout int
	UseHSTS        bool
	HSTSMaxAge     int

	BasicAuthUser     string
	BasicAuthPassword string
}

// NewSecurityConfig builds a SecurityConfig from defaults then overlays SWML_*
// environment variables (env takes precedence over defaults).
func NewSecurityConfig() *SecurityConfig {
	c := &SecurityConfig{}
	c.setDefaults()
	c.LoadFromEnv()
	return c
}

func (c *SecurityConfig) setDefaults() {
	c.SSLEnabled = false
	c.SSLVerifyMode = "CERT_REQUIRED"
	c.AllowedHosts = []string{"*"}
	c.CORSOrigins = []string{"*"}
	c.MaxRequestSize = 10 * 1024 * 1024
	c.RateLimit = 60
	c.RequestTimeout = 30
	c.UseHSTS = true
	c.HSTSMaxAge = 31536000
}

func parseList(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// LoadFromEnv (re)loads configuration from SWML_* environment variables.
func (c *SecurityConfig) LoadFromEnv() {
	sslEnabled := strings.ToLower(os.Getenv("SWML_SSL_ENABLED"))
	c.SSLEnabled = sslEnabled == "true" || sslEnabled == "1" || sslEnabled == "yes"
	c.SSLCertPath = os.Getenv("SWML_SSL_CERT_PATH")
	c.SSLKeyPath = os.Getenv("SWML_SSL_KEY_PATH")
	c.Domain = os.Getenv("SWML_SSL_DOMAIN")
	if v, ok := os.LookupEnv("SWML_SSL_VERIFY_MODE"); ok {
		c.SSLVerifyMode = v
	}
	if v, ok := os.LookupEnv("SWML_ALLOWED_HOSTS"); ok {
		c.AllowedHosts = parseList(v)
	}
	if v, ok := os.LookupEnv("SWML_CORS_ORIGINS"); ok {
		c.CORSOrigins = parseList(v)
	}
	c.MaxRequestSize = envInt("SWML_MAX_REQUEST_SIZE", c.MaxRequestSize)
	c.RateLimit = envInt("SWML_RATE_LIMIT", c.RateLimit)
	c.RequestTimeout = envInt("SWML_REQUEST_TIMEOUT", c.RequestTimeout)
	if v, ok := os.LookupEnv("SWML_USE_HSTS"); ok {
		c.UseHSTS = strings.ToLower(v) != "false"
	}
	c.HSTSMaxAge = envInt("SWML_HSTS_MAX_AGE", c.HSTSMaxAge)
	if v, ok := os.LookupEnv("SWML_BASIC_AUTH_USER"); ok {
		c.BasicAuthUser = v
	}
	if v, ok := os.LookupEnv("SWML_BASIC_AUTH_PASSWORD"); ok {
		c.BasicAuthPassword = v
	}
}

// ValidateSSLConfig checks the SSL configuration, returning (valid, errorMsg).
func (c *SecurityConfig) ValidateSSLConfig() (bool, string) {
	if !c.SSLEnabled {
		return true, ""
	}
	if c.SSLCertPath == "" {
		return false, "SSL enabled but SWML_SSL_CERT_PATH not set"
	}
	if c.SSLKeyPath == "" {
		return false, "SSL enabled but SWML_SSL_KEY_PATH not set"
	}
	if _, err := os.Stat(c.SSLCertPath); err != nil {
		return false, "SSL certificate file not found: " + c.SSLCertPath
	}
	if _, err := os.Stat(c.SSLKeyPath); err != nil {
		return false, "SSL key file not found: " + c.SSLKeyPath
	}
	return true, ""
}

// GetBasicAuth returns the configured basic-auth credentials, defaulting the
// username to "signalwire". A random password is generated (once) when none is
// configured, matching the Python fallback.
func (c *SecurityConfig) GetBasicAuth() (string, string) {
	username := c.BasicAuthUser
	if username == "" {
		username = "signalwire"
	}
	if c.BasicAuthPassword == "" {
		c.BasicAuthPassword = NewSessionManager(0).CreateSession("")
	}
	return username, c.BasicAuthPassword
}

// GetSecurityHeaders returns the security headers to add to responses. When
// isHTTPS and HSTS is enabled the Strict-Transport-Security header is included.
func (c *SecurityConfig) GetSecurityHeaders(isHTTPS bool) map[string]string {
	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	if isHTTPS && c.UseHSTS {
		headers["Strict-Transport-Security"] = fmt.Sprintf("max-age=%d; includeSubDomains", c.HSTSMaxAge)
	}
	return headers
}

// ShouldAllowHost reports whether the host is allowed by the allow-list ("*"
// permits any host).
func (c *SecurityConfig) ShouldAllowHost(host string) bool {
	for _, h := range c.AllowedHosts {
		if h == "*" {
			return true
		}
		if h == host {
			return true
		}
	}
	return false
}

// GetCORSConfig returns the CORS configuration.
func (c *SecurityConfig) GetCORSConfig() map[string]any {
	return map[string]any{
		"allow_origins":     c.CORSOrigins,
		"allow_credentials": true,
		"allow_methods":     []string{"*"},
		"allow_headers":     []string{"*"},
	}
}

// GetSSLContextKwargs returns the SSL parameters (primitive path strings) used
// to configure the HTTPS listener, mirroring Python's get_ssl_context_kwargs.
// The returned map is the primitive-dict form of the SSLCertPath/SSLKeyPath
// fields — the Go server feeds these into crypto/tls via swml.WithTLS. Returns
// an empty map when SSL is disabled or the SSL config fails validation.
func (c *SecurityConfig) GetSSLContextKwargs() map[string]any {
	if !c.SSLEnabled {
		return map[string]any{}
	}
	if ok, _ := c.ValidateSSLConfig(); !ok {
		return map[string]any{}
	}
	return map[string]any{
		"ssl_certfile": c.SSLCertPath,
		"ssl_keyfile":  c.SSLKeyPath,
	}
}

// GetURLScheme returns "https" when SSL is enabled, otherwise "http".
func (c *SecurityConfig) GetURLScheme() string {
	if c.SSLEnabled {
		return "https"
	}
	return "http"
}

// LogConfig logs a summary of the effective security configuration.
func (c *SecurityConfig) LogConfig(serviceName string) {
	logging.New("SecurityConfig").Info(
		"security config for %s: ssl=%v allowed_hosts=%v cors_origins=%v hsts=%v",
		serviceName, c.SSLEnabled, c.AllowedHosts, c.CORSOrigins, c.UseHSTS)
}
