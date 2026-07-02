package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ConfigLoader loads JSON configuration with ${VAR|default} environment-variable
// substitution. It mirrors signalwire.core.config_loader.ConfigLoader: it reads
// the first available config file from a search-path list, then resolves
// ${VAR|default} references against the process environment on access.
type ConfigLoader struct {
	configPaths []string
	config      map[string]any
	configFile  string
}

// varPattern matches ${VAR} or ${VAR|default}.
var varPattern = regexp.MustCompile(`\$\{([^}|]+)(?:\|([^}]*))?\}`)

// NewConfigLoader creates a ConfigLoader. When configPaths is nil the default
// search paths are used. The first existing, parseable file wins.
func NewConfigLoader(configPaths []string) *ConfigLoader {
	c := &ConfigLoader{configPaths: configPaths}
	if c.configPaths == nil {
		c.configPaths = defaultConfigPaths()
	}
	c.loadConfig()
	return c
}

// defaultConfigPaths returns the default configuration search paths (mirrors the
// Python _get_default_paths list).
func defaultConfigPaths() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"config.json",
		"agent_config.json",
		"swml_config.json",
		".swml/config.json",
		filepath.Join(home, ".swml", "config.json"),
		"/etc/swml/config.json",
	}
}

func (c *ConfigLoader) loadConfig() {
	for _, path := range c.configPaths {
		//nolint:gosec // G304: reading a config file from an operator-configured
		// search path is the intended behaviour of a config loader; the paths are
		// the SDK caller's own (defaults or explicitly passed), not attacker input.
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err != nil {
			continue
		}
		c.config = parsed
		c.configFile = path
		return
	}
}

// HasConfig reports whether a configuration file was loaded.
func (c *ConfigLoader) HasConfig() bool { return c.config != nil }

// GetConfigFile returns the path of the loaded config file, or "" if none.
func (c *ConfigLoader) GetConfigFile() string { return c.configFile }

// GetConfig returns the raw configuration (before variable substitution).
func (c *ConfigLoader) GetConfig() map[string]any {
	if c.config == nil {
		return map[string]any{}
	}
	return c.config
}

// SubstituteVars recursively resolves ${VAR|default} references in strings,
// maps and slices. Resolved scalar strings that look like a bool/int/float are
// converted to that type, matching the Python coercion behaviour. maxDepth
// guards against runaway recursion.
func (c *ConfigLoader) SubstituteVars(value any, maxDepth int) any {
	if maxDepth <= 0 {
		return value
	}
	switch v := value.(type) {
	case string:
		result := varPattern.ReplaceAllStringFunc(v, func(match string) string {
			groups := varPattern.FindStringSubmatch(match)
			name := groups[1]
			def := groups[2]
			if val, ok := os.LookupEnv(name); ok {
				return val
			}
			return def
		})
		lower := strings.ToLower(result)
		if lower == "true" || lower == "false" {
			return lower == "true"
		}
		if isAllDigits(result) {
			if n, err := strconv.Atoi(result); err == nil {
				return n
			}
		}
		if f, err := strconv.ParseFloat(result, 64); err == nil && strings.ContainsAny(result, "0123456789") {
			return f
		}
		return result
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, item := range v {
			out[k] = c.SubstituteVars(item, maxDepth-1)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = c.SubstituteVars(item, maxDepth-1)
		}
		return out
	default:
		return value
	}
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// Get returns a configuration value by dot-notation path (e.g. "security.ssl"),
// with ${VAR|default} substitution applied, or def if the path is absent.
func (c *ConfigLoader) Get(keyPath string, def any) any {
	if c.config == nil {
		return def
	}
	var value any = c.config
	for _, key := range strings.Split(keyPath, ".") {
		m, ok := value.(map[string]any)
		if !ok {
			return def
		}
		next, present := m[key]
		if !present {
			return def
		}
		value = next
	}
	return c.SubstituteVars(value, 10)
}

// GetSection returns an entire configuration section (with substitution), or an
// empty map if absent.
func (c *ConfigLoader) GetSection(section string) map[string]any {
	if c.config == nil {
		return map[string]any{}
	}
	raw, ok := c.config[section]
	if !ok {
		return map[string]any{}
	}
	if sub, ok := c.SubstituteVars(raw, 10).(map[string]any); ok {
		return sub
	}
	return map[string]any{}
}

// MergeWithEnv merges the (substituted) config with environment variables whose
// names start with envPrefix. Config-file values take precedence; a prefixed env
// var whose nested key is absent from the config is folded in (SWML_SSL_ENABLED
// → ssl.enabled).
func (c *ConfigLoader) MergeWithEnv(envPrefix string) map[string]any {
	var result map[string]any
	if c.config != nil {
		if sub, ok := c.SubstituteVars(c.config, 10).(map[string]any); ok {
			result = sub
		}
	}
	if result == nil {
		result = map[string]any{}
	}
	for _, kv := range os.Environ() {
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			continue
		}
		key, val := kv[:eq], kv[eq+1:]
		if !strings.HasPrefix(key, envPrefix) {
			continue
		}
		configKey := strings.ToLower(key[len(envPrefix):])
		if !hasNestedKey(result, configKey) {
			setNestedKey(result, configKey, val)
		}
	}
	return result
}

func hasNestedKey(data map[string]any, keyPath string) bool {
	current := data
	keys := strings.Split(keyPath, "_")
	for i, key := range keys {
		next, ok := current[key]
		if !ok {
			return false
		}
		if i == len(keys)-1 {
			return true
		}
		current, ok = next.(map[string]any)
		if !ok {
			return false
		}
	}
	return true
}

func setNestedKey(data map[string]any, keyPath string, value any) {
	current := data
	keys := strings.Split(keyPath, "_")
	for _, key := range keys[:len(keys)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[key] = next
		}
		current = next
	}
	current[keys[len(keys)-1]] = value
}

// FindConfigFile locates a config file for a service, searching service-specific
// names then generic defaults plus any additionalPaths. Returns "" if none
// exists. Mirrors the Python @staticmethod ConfigLoader.find_config_file.
func FindConfigFile(serviceName string, additionalPaths []string) string {
	var paths []string
	if serviceName != "" {
		paths = append(paths,
			serviceName+"_config.json",
			serviceName+".json",
			filepath.Join(".swml", serviceName+".json"),
		)
	}
	paths = append(paths, additionalPaths...)
	paths = append(paths, defaultConfigPaths()...)
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
