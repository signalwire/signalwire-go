package skills

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
)

// registry holds all registered skill factories.
var (
	registry   = make(map[string]func(params map[string]any) SkillBase)
	registryMu sync.RWMutex
)

// SkillRegistry is the per-instance Python-parity surface mirroring
// `signalwire.skills.registry.SkillRegistry`. Each instance owns its
// own list of external skill directories, validated and de-duplicated
// on insert. The package-level `RegisterSkill` / `GetSkillFactory` /
// `ListSkills` functions remain the canonical Go API for static
// compile-time skill registration; `SkillRegistry` exists so the
// `add_skill_directory` parity case has a real owning object the
// audit and downstream callers can hold.
type SkillRegistry struct {
	mu             sync.Mutex
	externalPaths  []string
}

// NewSkillRegistry constructs a new SkillRegistry. The Python reference
// uses a singleton-per-module (`skill_registry`); Go callers can either
// construct their own via NewSkillRegistry() or use the global
// `globalRegistry` accessed through the package-level helpers.
func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{externalPaths: nil}
}

// globalRegistry backs the package-level `AddSkillDirectory` shim so
// the free-function call path and the instance-method call path share
// state, matching Python's `signalwire.add_skill_directory` →
// `skill_registry.add_skill_directory` delegation.
var globalRegistry = NewSkillRegistry()

// AddSkillDirectory adds a directory to search for skills. Mirrors
// Python's `SkillRegistry.add_skill_directory`: validates that the
// path exists and is a directory, then appends it (de-duplicated) to
// the registry's external paths list. Returns an error (the Go analog
// of Python's `ValueError`) for non-existent paths or non-directories.
func (r *SkillRegistry) AddSkillDirectory(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Skill directory does not exist: %s", path)
		}
		return fmt.Errorf("AddSkillDirectory: stat %q: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("Path is not a directory: %s", path)
	}
	for _, existing := range r.externalPaths {
		if existing == path {
			return nil
		}
	}
	r.externalPaths = append(r.externalPaths, path)
	return nil
}

// ExternalPaths returns a copy of the registered external skill
// directories. Parity surface for Python's `_external_paths`.
func (r *SkillRegistry) ExternalPaths() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.externalPaths))
	copy(out, r.externalPaths)
	return out
}

// RegisterSkill registers a skill factory function by name.
// This is typically called from init() functions in builtin skill packages.
func RegisterSkill(name string, factory func(params map[string]any) SkillBase) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// GetSkillFactory returns the factory function for a registered skill name.
// Returns nil if the skill is not registered.
func GetSkillFactory(name string) func(params map[string]any) SkillBase {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[name]
}

// ListSkills returns sorted names of all registered skills.
func ListSkills() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ListSkillsWithParams returns the complete parameter schema for all registered skills.
// It instantiates each skill with nil params to obtain its GetParameterSchema output.
// This mirrors Python's skill_registry.get_all_skills_schema().
// The returned map has skill names as keys and their parameter schemas as values.
func ListSkillsWithParams() map[string]map[string]map[string]any {
	registryMu.RLock()
	defer registryMu.RUnlock()
	result := make(map[string]map[string]map[string]any, len(registry))
	for name, factory := range registry {
		instance := factory(nil)
		result[name] = instance.GetParameterSchema()
	}
	return result
}

// AddSkillDirectory is a package-level shim that delegates to the
// shared `globalRegistry.AddSkillDirectory`. It mirrors Python's
// `signalwire.add_skill_directory` (which delegates to the module
// singleton `signalwire.skills.registry.skill_registry`).
//
// The path is validated (must exist and be a directory) and added
// (de-duplicated) to the global registry's external-paths list. Note
// that Go compiles to a static binary; dynamic on-disk skill loading
// is not implemented here, but the path-tracking surface is — so
// tools that introspect "what external directories has this agent
// registered?" get the same answer they'd get on the Python side.
func AddSkillDirectory(path string) error {
	if path == "" {
		return errors.New("AddSkillDirectory: path must be non-empty")
	}
	return globalRegistry.AddSkillDirectory(path)
}
