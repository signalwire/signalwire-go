package skills

import (
	"errors"
	"sort"
	"sync"
)

// registry holds all registered skill factories.
var (
	registry   = make(map[string]func(params map[string]any) SkillBase)
	registryMu sync.RWMutex
)

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

// AddSkillDirectory is a stub that documents the Go compilation model constraint.
// In Go, skills must be registered at compile time via RegisterSkill called from
// an init() function. Dynamic directory scanning (as in Python) is not possible
// because Go compiles to a static binary. To add third-party skills, import their
// package (which calls RegisterSkill in init()) or call RegisterSkill directly
// from main() before starting the agent.
func AddSkillDirectory(path string) error {
	if path == "" {
		return errors.New("AddSkillDirectory: path must be non-empty")
	}
	return errors.New(
		"AddSkillDirectory: dynamic skill loading from directories is not supported in Go. " +
			"Import the skill package (which calls RegisterSkill in init()) or call " +
			"skills.RegisterSkill() directly from main() instead",
	)
}
