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

// NewConfigLoader creates a ConfigLoader. `configPaths` is OPTIONAL, matching
// the reference (`config_paths: list[str] | None = None`): call it with no
// arguments — or with an empty list — to use the default search paths. The
// first existing, parseable file wins.
func NewConfigLoader(configPaths ...string) *ConfigLoader {
	// An EMPTY variadic must reach config.New as nil, not as a non-nil empty
	// slice: config.New substitutes DefaultPaths() only on nil, so
	// `NewConfigLoader()` and `NewConfigLoader(pathsThatWereNil...)` would
	// otherwise search nothing at all instead of the defaults.
	if len(configPaths) == 0 {
		configPaths = nil
	}
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
// coerced to that type. maxDepth
// guards against runaway recursion.
// maxDepth is OPTIONAL: 0 substitutes the reference's `max_depth: int = 10`, so
// `SubstituteVars(v, 0)` is the reference's one-argument call. Without the
// substitution a zero maxDepth returned the value UNSUBSTITUTED — the opposite
// of what omitting the argument means in the reference.
func (c *ConfigLoader) SubstituteVars(value any, maxDepth int) any {
	if maxDepth == 0 {
		maxDepth = 10
	}
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
// envPrefix is OPTIONAL: the empty string substitutes the reference's
// `env_prefix: str = "SWML_"`. Without it, an omitting caller got a bare ""
// prefix, which matches EVERY environment variable.
func (c *ConfigLoader) MergeWithEnv(envPrefix string) map[string]any {
	if envPrefix == "" {
		envPrefix = "SWML_"
	}
	return c.inner.MergeWithEnv(envPrefix)
}

// FindConfigFile locates a config file for a service, searching service-specific
// names then generic defaults plus any additionalPaths. Returns "" if none
// exists.
// ConfigLoader.find_config_file(service_name=None, additional_paths=None).
//
// Both parameters are optional and the delegate honours their zero values:
// config.FindFile skips the service-specific candidate names when serviceName
// is "" and falls through to the generic defaults, and appends nothing for a nil
// additionalPaths. This wrapper delegates in one line, so no guard in THIS body
// carries that fact.
//
//sw:param serviceName optional
//sw:param additionalPaths optional
func FindConfigFile(serviceName string, additionalPaths []string) string {
	return config.FindFile(serviceName, additionalPaths)
}
