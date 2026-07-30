package security

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/signalwire/signalwire-go/v3/internal/config"
	"github.com/signalwire/signalwire-go/v3/pkg/logging"
)

// SecurityConfig centralises the SDK's HTTP security settings (SSL, allowed
// hosts, CORS, security headers, HSTS, basic auth), resolved from defaults, then
// SWML_* env vars, then the config file (highest priority). Mirrors
// signalwire.core.security_config.SecurityConfig.
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

// NewSecurityConfig builds a SecurityConfig by resolving, in the reference's
// order: defaults, then SWML_* environment variables, then the config file —
// the config file being HIGHEST priority
// (signalwire.core.security_config.SecurityConfig.__init__).
//
// configFile is an explicit path; when empty the loader auto-discovers one for
// serviceName (serviceName_config.json, serviceName.json, .swml/serviceName.json,
// then the generic defaults), exactly as the reference's
// ConfigLoader.find_config_file does. Pass "" for both to get the plain
// defaults-then-env resolution.
//
// The config-file layer is load-bearing for security: an operator who supplies
// ssl_enabled/ssl_cert_path/ssl_key_path only in a config file must get TLS. A
// config file that is absent, unreadable, unparseable, or has no `security`
// section leaves the env/default resolution untouched (the reference's
// best-effort load).
func NewSecurityConfig(configFile, serviceName string) *SecurityConfig {
	c := &SecurityConfig{}
	c.setDefaults()
	c.LoadFromEnv()
	c.loadConfigFile(configFile, serviceName)
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
	// SWML_DOMAIN is the reference's spelling (security_config.py:
	// SSL_DOMAIN = "SWML_DOMAIN"), and it is what every doc page here has
	// always told operators to set. SWML_SSL_DOMAIN was the name the code
	// actually read — so an operator following the docs, or coming from the
	// Python SDK, set SWML_DOMAIN and got nothing. Read the canonical name
	// first and keep SWML_SSL_DOMAIN as a fallback for anyone who followed
	// the one README table that listed it.
	c.Domain = os.Getenv("SWML_DOMAIN")
	if c.Domain == "" {
		c.Domain = os.Getenv("SWML_SSL_DOMAIN")
	}
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

// loadConfigFile overlays the `security` section of a JSON config file onto the
// already-resolved defaults+env values. It is the HIGHEST-priority layer,
// mirroring the reference's SecurityConfig._load_config_file.
//
// configFile is an explicit path; when empty a file is auto-discovered for
// serviceName. Every key is applied only when PRESENT in the section, so a
// partial `security` block overrides just the keys it names and leaves the rest
// at their env/default values. A missing/unreadable/unparseable file, or one
// without a `security` section, is a silent no-op — the reference behaves the
// same way rather than failing service start-up.
func (c *SecurityConfig) loadConfigFile(configFile, serviceName string) {
	if configFile == "" {
		configFile = config.FindFile(serviceName, nil)
	}
	if configFile == "" {
		return
	}
	loader := config.New([]string{configFile})
	if !loader.HasConfig() {
		return
	}
	section := loader.GetSection("security")
	if len(section) == 0 {
		return
	}
	c.applySSLSection(section)
	c.applyHostsSection(section)
	c.applyHSTSSection(section)
	c.applyAuthSection(section)
}

func (c *SecurityConfig) applySSLSection(section map[string]any) {
	if v, ok := section["ssl_enabled"]; ok {
		c.SSLEnabled = configBool(v, c.SSLEnabled)
	}
	if v, ok := section["ssl_cert_path"]; ok {
		c.SSLCertPath = configString(v, c.SSLCertPath)
	}
	if v, ok := section["ssl_key_path"]; ok {
		c.SSLKeyPath = configString(v, c.SSLKeyPath)
	}
	if v, ok := section["domain"]; ok {
		c.Domain = configString(v, c.Domain)
	}
	if v, ok := section["ssl_verify_mode"]; ok {
		c.SSLVerifyMode = configString(v, c.SSLVerifyMode)
	}
}

func (c *SecurityConfig) applyHostsSection(section map[string]any) {
	if v, ok := section["allowed_hosts"]; ok {
		c.AllowedHosts = configList(v, c.AllowedHosts)
	}
	if v, ok := section["cors_origins"]; ok {
		c.CORSOrigins = configList(v, c.CORSOrigins)
	}
	if v, ok := section["max_request_size"]; ok {
		c.MaxRequestSize = configInt(v, c.MaxRequestSize)
	}
	if v, ok := section["rate_limit"]; ok {
		c.RateLimit = configInt(v, c.RateLimit)
	}
	if v, ok := section["request_timeout"]; ok {
		c.RequestTimeout = configInt(v, c.RequestTimeout)
	}
}

func (c *SecurityConfig) applyHSTSSection(section map[string]any) {
	if v, ok := section["use_hsts"]; ok {
		c.UseHSTS = configBool(v, c.UseHSTS)
	}
	if v, ok := section["hsts_max_age"]; ok {
		c.HSTSMaxAge = configInt(v, c.HSTSMaxAge)
	}
}

func (c *SecurityConfig) applyAuthSection(section map[string]any) {
	auth, ok := section["auth"].(map[string]any)
	if !ok {
		return
	}
	basic, ok := auth["basic"].(map[string]any)
	if !ok {
		return
	}
	if v, ok := basic["user"]; ok {
		c.BasicAuthUser = configString(v, c.BasicAuthUser)
	}
	if v, ok := basic["password"]; ok {
		c.BasicAuthPassword = configString(v, c.BasicAuthPassword)
	}
}

// configBool coerces a config value to bool. JSON gives a real bool; a
// ${VAR|default} substitution can yield the string form, which the loader
// already coerces, but a literal "true"/"1"/"yes" string is honoured too so a
// hand-written config behaves like the equivalent env var.
func configBool(v any, def bool) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		if s == "" {
			return def
		}
		return s == "true" || s == "1" || s == "yes"
	default:
		return def
	}
}

func configString(v any, def string) string {
	if s, ok := v.(string); ok {
		return s
	}
	return def
}

// configInt coerces a config value to int. JSON numbers decode as float64; the
// loader's ${VAR|default} substitution can yield an int or a string.
func configInt(v any, def int) int {
	switch t := v.(type) {
	case int:
		return t
	case float64:
		return int(t)
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return n
		}
	}
	return def
}

// configList coerces a config value to a string slice, accepting either a JSON
// array or the comma-separated string form the env vars use (Python's
// _parse_list takes both).
func configList(v any, def []string) []string {
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				if trimmed := strings.TrimSpace(s); trimmed != "" {
					out = append(out, trimmed)
				}
			}
		}
		return out
	case []string:
		return append([]string(nil), t...)
	case string:
		if t == "*" {
			return []string{"*"}
		}
		return parseList(t)
	default:
		return def
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
// configured.
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
// to configure the HTTPS listener.
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
