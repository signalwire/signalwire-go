package skills

import (
	"fmt"
	"os"
	"sync"
)

// SkillManager manages the lifecycle of loaded skill instances.
type SkillManager struct {
	loadedSkills map[string]SkillBase
	mu           sync.RWMutex
}

// NewSkillManager creates a new SkillManager.
func NewSkillManager() *SkillManager {
	return &SkillManager{
		loadedSkills: make(map[string]SkillBase),
	}
}

// LoadSkill validates environment variables, calls Setup, and registers the skill.
// Returns (success bool, errorMessage string).
//
// When a skill with the same instance key is already loaded, the behavior
// depends on SupportsMultipleInstances():
//   - false (default): returns (false, error) — duplicate is an error.
//   - true: returns (true, "") — duplicate instance is silently accepted,
//     matching Python's SkillManager.load_skill() warning-and-continue behavior.
func (sm *SkillManager) LoadSkill(skill SkillBase) (bool, string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := skill.GetInstanceKey()

	// Check if already loaded
	if _, exists := sm.loadedSkills[key]; exists {
		if skill.SupportsMultipleInstances() {
			// Multi-instance skill: duplicate instance key is acceptable.
			// Python warns and returns True, "". Mirror that here.
			return true, ""
		}
		return false, fmt.Sprintf("skill '%s' is already loaded and does not support multiple instances", key)
	}

	// Validate required environment variables
	for _, envVar := range skill.RequiredEnvVars() {
		if os.Getenv(envVar) == "" {
			return false, fmt.Sprintf("missing required environment variable: %s", envVar)
		}
	}

	// Call Setup
	if !skill.Setup() {
		return false, fmt.Sprintf("skill '%s' setup failed", skill.Name())
	}

	sm.loadedSkills[key] = skill
	return true, ""
}

// UnloadSkill removes a skill by its instance key. Returns true if found and removed.
func (sm *SkillManager) UnloadSkill(key string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, exists := sm.loadedSkills[key]
	if !exists {
		return false
	}

	skill.Cleanup()
	delete(sm.loadedSkills, key)
	return true
}

// ListLoadedSkills returns the instance keys of all loaded skills.
func (sm *SkillManager) ListLoadedSkills() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	keys := make([]string, 0, len(sm.loadedSkills))
	for k := range sm.loadedSkills {
		keys = append(keys, k)
	}
	return keys
}

// HasSkill returns true if a skill with the given instance key is loaded.
func (sm *SkillManager) HasSkill(key string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	_, exists := sm.loadedSkills[key]
	return exists
}

// GetSkill returns the skill with the given instance key, or nil if not found.
func (sm *SkillManager) GetSkill(key string) SkillBase {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.loadedSkills[key]
}
