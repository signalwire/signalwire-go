package skills

import (
	"testing"
)

// subSkillLoader is a skill whose Setup() loads a SECOND skill through the same
// SkillManager — the sub-skill pattern. Setup() is caller-supplied code
// (SkillBase is a public interface, skill_base.go:37), and LoadSkill used to
// invoke it while holding sm.mu.Lock() (manager.go:56, deferred unlock at 57),
// so this call reenters a non-reentrant write lock and self-deadlocks.
type subSkillLoader struct {
	BaseSkill
	mgr       *SkillManager
	subName   string
	subLoadOK bool
	subErr    string
	setupRan  bool
}

func (s *subSkillLoader) Setup() bool {
	s.setupRan = true
	if s.mgr != nil && s.subName != "" {
		s.subLoadOK, s.subErr = s.mgr.LoadSkill(newTestSkill(s.subName))
	}
	return true
}

func (s *subSkillLoader) RegisterTools() []ToolRegistration { return nil }

// panicSkill panics from Setup(). Go has no mutex poisoning, but the manager
// must still be usable afterwards — the deferred unlock has to survive the
// unwind and the half-registered skill must not be left in the map.
type panicSkill struct {
	BaseSkill
}

func (s *panicSkill) Setup() bool                       { panic("skill setup exploded") }
func (s *panicSkill) RegisterTools() []ToolRegistration { return nil }

// cleanupLoader loads another skill from Cleanup(), which UnloadSkill invokes
// under the same write lock (manager.go:104). Same reentrancy, different verb —
// the brief only named Setup().
type cleanupLoader struct {
	BaseSkill
	mgr     *SkillManager
	subName string
	ran     bool
}

func (s *cleanupLoader) Setup() bool                       { return true }
func (s *cleanupLoader) RegisterTools() []ToolRegistration { return nil }
func (s *cleanupLoader) Cleanup() {
	s.ran = true
	if s.mgr != nil && s.subName != "" {
		s.mgr.LoadSkill(newTestSkill(s.subName))
	}
}

// TestLoadSkill_SubSkillDuringSetupDoesNotDeadlock is the SITE 1 reproduction.
//
// sm.mu is a sync.RWMutex and Lock() is not reentrant, so a Setup() that calls
// back into LoadSkill blocks forever on a lock its own caller holds. Go's
// runtime deadlock detector does NOT fire (it requires every goroutine asleep;
// the test-timeout goroutine stays runnable), so in production this hangs with
// no diagnostic at all.
func TestLoadSkill_SubSkillDuringSetupDoesNotDeadlock(t *testing.T) {
	sm := NewSkillManager()
	parent := &subSkillLoader{
		BaseSkill: BaseSkill{SkillName: "parent", SkillDesc: "loads a sub-skill"},
		mgr:       sm,
		subName:   "child",
	}

	ok, errMsg := sm.LoadSkill(parent)
	if !ok {
		t.Fatalf("parent load failed: %s", errMsg)
	}
	if !parent.setupRan {
		t.Fatal("parent Setup() never ran")
	}

	// The sub-skill load must have been ATTEMPTED and must have succeeded:
	// a reentrant load is legal once the lock is not held across Setup().
	if !parent.subLoadOK {
		t.Errorf("sub-skill load failed: %s", parent.subErr)
	}

	// Both skills are registered and the manager still works.
	if !sm.HasSkill("parent") {
		t.Error("parent not registered")
	}
	if !sm.HasSkill("child") {
		t.Error("child (sub-skill) not registered")
	}
}

// TestUnloadSkill_ReentrantCleanupDoesNotDeadlock covers the Cleanup() half.
func TestUnloadSkill_ReentrantCleanupDoesNotDeadlock(t *testing.T) {
	sm := NewSkillManager()
	s := &cleanupLoader{
		BaseSkill: BaseSkill{SkillName: "cleanuper", SkillDesc: "loads from Cleanup"},
		mgr:       sm,
		subName:   "from_cleanup",
	}
	if ok, msg := sm.LoadSkill(s); !ok {
		t.Fatalf("load failed: %s", msg)
	}

	if !sm.UnloadSkill("cleanuper") {
		t.Fatal("UnloadSkill returned false")
	}
	if !s.ran {
		t.Error("Cleanup() never ran")
	}
	if sm.HasSkill("cleanuper") {
		t.Error("cleanuper still present after unload")
	}
	if !sm.HasSkill("from_cleanup") {
		t.Error("skill loaded from Cleanup() was not registered")
	}
}

// TestLoadSkill_PanickingSetupLeavesManagerUsable pins the post-panic contract:
// the lock is released, nothing half-registered is left behind, and subsequent
// operations succeed rather than block.
func TestLoadSkill_PanickingSetupLeavesManagerUsable(t *testing.T) {
	sm := NewSkillManager()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected Setup() to panic")
			}
		}()
		//nolint:errcheck // panics before returning
		sm.LoadSkill(&panicSkill{BaseSkill: BaseSkill{SkillName: "boom", SkillDesc: "panics"}})
	}()

	// The panicking skill must NOT be registered.
	if sm.HasSkill("boom") {
		t.Error("panicking skill was registered anyway")
	}

	// The manager must still be usable — if the write lock leaked, this blocks.
	if ok, msg := sm.LoadSkill(newTestSkill("after_panic")); !ok {
		t.Fatalf("load after panic failed: %s", msg)
	}
	if !sm.HasSkill("after_panic") {
		t.Error("post-panic load did not register")
	}
	if got := sm.ListLoadedSkills(); len(got) != 1 {
		t.Errorf("expected 1 loaded skill after panic, got %v", got)
	}
}
