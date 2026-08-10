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
	// agent is the agent this manager loads skills for. Read via Agent();
	// the reference's SkillManager.agent (skill_manager.py). LoadSkill
	// propagates it to each skill so a skill can configure its agent.
	agent SkillAgent
}

// NewSkillManager creates a new SkillManager. Pass the owning agent so loaded
// skills can reach it (the reference takes it as SkillManager(agent)); pass nil
// for a standalone manager.
func NewSkillManager(agent ...SkillAgent) *SkillManager {
	sm := &SkillManager{
		loadedSkills: make(map[string]SkillBase),
	}
	if len(agent) > 0 {
		sm.agent = agent[0]
	}
	return sm
}

// Agent returns the agent this manager loads skills for, or nil when the manager
// is standalone (Python: SkillManager.agent).
func (sm *SkillManager) Agent() SkillAgent {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.agent
}

// SetAgent records the owning agent. Skills loaded after this call receive it.
func (sm *SkillManager) SetAgent(a SkillAgent) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.agent = a
}

// LoadSkill validates environment variables, calls Setup, and registers the skill.
// Returns (success bool, errorMessage string).
//
// When a skill with the same instance key is already loaded, the behavior
// depends on SupportsMultipleInstances():
//   - false (default): returns (false, error) — duplicate is an error.
//   - true: returns (true, "") — duplicate instance is silently accepted,
//     matching Python's SkillManager.load_skill() warning-and-continue behavior.
//
// No caller-supplied method is invoked while sm.mu is held. Every method on
// SkillBase is caller-supplied (SkillBase is a public interface, skill_base.go:37),
// and sm.mu is a non-reentrant sync.RWMutex, so any of them that calls back into
// the manager — LoadSkill, UnloadSkill, HasSkill, GetSkill, ListLoadedSkills —
// would self-deadlock on a lock its own caller still holds. The live case is a
// skill that loads a SUB-SKILL from its Setup(): AgentBase.AddSkill
// (pkg/agent/agent.go:2662) calls LoadSkill, so `skill.Setup() -> agent.AddSkill
// -> LoadSkill` hangs. Go's deadlock detector does not fire for it (it needs
// every goroutine asleep), so in production it hangs silently.
//
// Reentrant loads are ALLOWED, not rejected. The reference does not serialise
// this at all — signalwire-python's SkillManager.load_skill takes no lock, doing
// setup() (skill_manager.py:166) and then loaded_skills[key] = ...
// (skill_manager.py:189) unsynchronised — so "register and set up atomically" is
// not part of the contract and a sentinel that rejected a nested load would
// invent a failure mode no other port has. The lock's job here is only to keep
// the loadedSkills MAP consistent.
//
// Moving Setup() out does open a check-then-act window on the duplicate key, so
// the key is re-checked under the lock before registering (see below). That
// makes concurrent duplicate loads lose the race cleanly instead of silently
// overwriting a registered skill.
func (sm *SkillManager) LoadSkill(skill SkillBase) (bool, string) {
	// Caller-supplied, so read outside the lock.
	key := skill.GetInstanceKey()
	multi := skill.SupportsMultipleInstances()
	requiredEnv := skill.RequiredEnvVars()

	duplicate := func() bool {
		sm.mu.RLock()
		defer sm.mu.RUnlock()
		_, exists := sm.loadedSkills[key]
		return exists
	}()
	if duplicate {
		if multi {
			// Multi-instance skill: duplicate instance key is acceptable.
			// Python warns and returns True, "". Mirror that here.
			return true, ""
		}
		return false, fmt.Sprintf("skill '%s' is already loaded and does not support multiple instances", key)
	}

	// Validate required environment variables
	for _, envVar := range requiredEnv {
		if os.Getenv(envVar) == "" {
			return false, fmt.Sprintf("missing required environment variable: %s", envVar)
		}
	}

	// Hand the skill its agent BEFORE Setup, so Setup can already configure the
	// agent — the reference assigns self.agent in SkillBase.__init__, i.e. before
	// any setup work runs.
	agent := sm.Agent()
	if agent != nil {
		skill.SetAgent(agent)
	}

	// Call Setup with NO lock held: it may load sub-skills.
	if !skill.Setup() {
		return false, fmt.Sprintf("skill '%s' setup failed", skill.Name())
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()
	// Re-check under the write lock: another goroutine may have registered this
	// key while Setup() ran, and so may Setup() itself via a sub-skill load.
	if _, exists := sm.loadedSkills[key]; exists {
		if multi {
			return true, ""
		}
		return false, fmt.Sprintf("skill '%s' is already loaded and does not support multiple instances", key)
	}
	sm.loadedSkills[key] = skill
	return true, ""
}

// UnloadSkill removes a skill by its instance key. Returns true if found and removed.
//
// Cleanup() is caller-supplied and is invoked with the lock RELEASED, for the
// same reason as Setup() in LoadSkill: a Cleanup() that touches the manager
// would otherwise self-deadlock on the non-reentrant sm.mu. The entry is removed
// from the map BEFORE Cleanup() runs, so a reentrant HasSkill/GetSkill during
// cleanup sees the skill as already gone (it is being torn down) and the removal
// cannot be lost if Cleanup() panics.
func (sm *SkillManager) UnloadSkill(key string) bool {
	skill, exists := func() (SkillBase, bool) {
		sm.mu.Lock()
		defer sm.mu.Unlock()
		s, ok := sm.loadedSkills[key]
		if !ok {
			return nil, false
		}
		delete(sm.loadedSkills, key)
		return s, true
	}()
	if !exists {
		return false
	}

	skill.Cleanup()
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
