package agent

import (
	"github.com/signalwire/signalwire-go/v3/internal/config"
)

// ConfigLoader loads JSON configuration with ${VAR|default} environment-variable
// substitution. It mirrors signalwire.core.config_loader.ConfigLoader: it reads
// the first available config file from a search-path list, then resolves
// ${VAR|default} references against the process environment on access.
//
// The mechanics live in internal/config so pkg/security can reuse the exact
// same loader for SecurityConfig's config-file layer without an import cycle
// (pkg/agent already imports pkg/security). This type is the public façade;
// there is one implementation, not two.
type ConfigLoader struct {
	inner *config.Loader
}

// NewConfigLoader creates a ConfigLoader. When configPaths is nil the default
// search paths are used. The first existing, parseable file wins.
func NewConfigLoader(configPaths []string) *ConfigLoader {
	return &ConfigLoader{inner: config.New(configPaths)}
}

// ConfigPaths returns the search paths this loader considered, in order — the
// caller's list, or the defaults when none was supplied (Python: config_paths).
// A copy is returned so a caller cannot mutate the loader's search order.
func (c *ConfigLoader) ConfigPaths() []string { return c.inner.ConfigPaths() }

// HasConfig reports whether a configuration file was loaded.
func (c *ConfigLoader) HasConfig() bool { return c.inner.HasConfig() }

// GetConfigFile returns the path of the loaded config file, or "" if none.
func (c *ConfigLoader) GetConfigFile() string { return c.inner.ConfigFile() }

// GetConfig returns the raw configuration (before variable substitution).
func (c *ConfigLoader) GetConfig() map[string]any { return c.inner.Config() }

// SubstituteVars recursively resolves ${VAR|default} references in strings,
// maps and slices. Resolved scalar strings that look like a bool/int/float are
// converted to that type, matching the Python coercion behaviour. maxDepth
// guards against runaway recursion.
func (c *ConfigLoader) SubstituteVars(value any, maxDepth int) any {
	return c.inner.SubstituteVars(value, maxDepth)
}

// Get returns a configuration value by dot-notation path (e.g. "security.ssl"),
// with ${VAR|default} substitution applied, or def if the path is absent.
func (c *ConfigLoader) Get(keyPath string, def any) any { return c.inner.Get(keyPath, def) }

// GetSection returns an entire configuration section (with substitution), or an
// empty map if absent.
func (c *ConfigLoader) GetSection(section string) map[string]any {
	return c.inner.GetSection(section)
}

// MergeWithEnv merges the (substituted) config with environment variables whose
// names start with envPrefix. Config-file values take precedence; a prefixed env
// var whose nested key is absent from the config is folded in (SWML_SSL_ENABLED
// → ssl.enabled).
func (c *ConfigLoader) MergeWithEnv(envPrefix string) map[string]any {
	return c.inner.MergeWithEnv(envPrefix)
}

// FindConfigFile locates a config file for a service, searching service-specific
// names then generic defaults plus any additionalPaths. Returns "" if none
// exists. Mirrors the Python @staticmethod ConfigLoader.find_config_file.
func FindConfigFile(serviceName string, additionalPaths []string) string {
	return config.FindFile(serviceName, additionalPaths)
}
