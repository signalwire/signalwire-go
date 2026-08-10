package skills

import (
	"sort"
	"testing"
)

// reentrantSchemaSkill is a SkillBase whose GetParameterSchema calls back into
// the package-level registry. A skill that wants to describe its parameters in
// terms of what else is registered (a "depends on skill X" schema hint) is a
// legitimate thing for third-party skill code to do, and it must not deadlock.
type reentrantSchemaSkill struct {
	BaseSkill
	observed []string
}

func (s *reentrantSchemaSkill) Setup() bool { return true }

func (s *reentrantSchemaSkill) RegisterTools() []ToolRegistration { return nil }

func (s *reentrantSchemaSkill) GetParameterSchema() map[string]map[string]any {
	// Re-enter the registry from inside caller-supplied code. Under the old
	// implementation ListSkillsWithParams held registryMu.RLock() across this
	// call, so this RegisterSkill (a WRITE lock on a non-reentrant mutex)
	// blocked forever on the read lock its own caller still held.
	RegisterSkill("reentrant_registered_from_schema", func(params map[string]any) SkillBase {
		return newTestSkill("reentrant_registered_from_schema")
	})
	s.observed = ListSkills()
	return map[string]map[string]any{
		"reentrant": {"type": "boolean"},
	}
}

// unregister removes a name from the package-level registry, taking the write
// lock the same way production code does. If a prior call leaked the lock, this
// helper is itself where the test wedges — which is a true report either way.
func unregister(names ...string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	for _, n := range names {
		delete(registry, n)
	}
}

// snapshotRegistryNames records the registry keys so a test can restore the
// package-level global it necessarily mutates.
func snapshotRegistryNames() map[string]bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make(map[string]bool, len(registry))
	for n := range registry {
		out[n] = true
	}
	return out
}

// TestListSkillsWithParams_ReentrantFactoryDoesNotDeadlock is the deadlock
// regression. ListSkillsWithParams invokes both a caller-supplied factory and
// a caller-supplied GetParameterSchema; neither may run while the registry
// lock is held, because either one may legitimately call back into the
// registry. registryMu is a sync.RWMutex, so a reentrant WRITE (RegisterSkill)
// from inside an RLock-holding read is an unconditional self-deadlock.
//
// Reverting the registry.go fix makes this test HANG until go test -timeout
// fires and dumps every goroutine.
func TestListSkillsWithParams_ReentrantFactoryDoesNotDeadlock(t *testing.T) {
	before := snapshotRegistryNames()
	t.Cleanup(func() {
		registryMu.Lock()
		defer registryMu.Unlock()
		for n := range registry {
			if !before[n] {
				delete(registry, n)
			}
		}
	})

	RegisterSkill("reentrant_schema_skill", func(params map[string]any) SkillBase {
		return &reentrantSchemaSkill{
			BaseSkill: BaseSkill{SkillName: "reentrant_schema_skill", SkillDesc: "reentrant"},
		}
	})

	schemas := ListSkillsWithParams()
	if _, ok := schemas["reentrant_schema_skill"]; !ok {
		t.Fatalf("reentrant_schema_skill missing from schema map; got %d entries", len(schemas))
	}
	if _, ok := schemas["reentrant_schema_skill"]["reentrant"]; !ok {
		t.Error("reentrant skill's own schema key not returned")
	}

	// The reentrant registration really took effect, proving the callback ran
	// with the lock released rather than being skipped.
	if GetSkillFactory("reentrant_registered_from_schema") == nil {
		t.Error("reentrant RegisterSkill from inside GetParameterSchema did not take effect")
	}
}

// TestListSkillsWithParams_ReentrantFactoryConstruction covers the other
// caller-supplied call site in the same loop: the factory itself, not the
// schema method, re-entering the registry.
func TestListSkillsWithParams_ReentrantFactoryConstruction(t *testing.T) {
	before := snapshotRegistryNames()
	t.Cleanup(func() {
		registryMu.Lock()
		defer registryMu.Unlock()
		for n := range registry {
			if !before[n] {
				delete(registry, n)
			}
		}
	})

	RegisterSkill("reentrant_ctor_skill", func(params map[string]any) SkillBase {
		// A factory that consults the registry while constructing.
		_ = ListSkills()
		RegisterSkill("reentrant_registered_from_ctor", func(p map[string]any) SkillBase {
			return newTestSkill("reentrant_registered_from_ctor")
		})
		return newTestSkill("reentrant_ctor_skill")
	})

	schemas := ListSkillsWithParams()
	if _, ok := schemas["reentrant_ctor_skill"]; !ok {
		t.Fatalf("reentrant_ctor_skill missing from schema map; got %d entries", len(schemas))
	}
	if GetSkillFactory("reentrant_registered_from_ctor") == nil {
		t.Error("reentrant RegisterSkill from inside the factory did not take effect")
	}
}

// TestRegistryUsableAfterFactoryPanic pins the contract the rust twin proved:
// a caller-supplied factory that panics must not leave the registry unusable.
// Go has no mutex poisoning, so the cascade rust suffered cannot happen here;
// what this pins is that the lock is not left HELD across the unwind and that
// every subsequent registry operation still completes.
func TestRegistryUsableAfterFactoryPanic(t *testing.T) {
	before := snapshotRegistryNames()
	t.Cleanup(func() {
		registryMu.Lock()
		defer registryMu.Unlock()
		for n := range registry {
			if !before[n] {
				delete(registry, n)
			}
		}
	})

	RegisterSkill("panicking_factory_skill", func(params map[string]any) SkillBase {
		panic("caller-supplied factory blew up")
	})

	func() {
		defer func() {
			if rec := recover(); rec == nil {
				t.Error("expected the factory panic to propagate to the caller")
			}
		}()
		_ = ListSkillsWithParams()
	}()

	// The registry must still be fully usable: a write, a read, and a listing.
	unregister("panicking_factory_skill")
	if GetSkillFactory("panicking_factory_skill") != nil {
		t.Error("unregister after panic did not take effect (write lock unavailable?)")
	}
	RegisterSkill("after_panic_skill", func(params map[string]any) SkillBase {
		return newTestSkill("after_panic_skill")
	})
	if GetSkillFactory("after_panic_skill") == nil {
		t.Fatal("RegisterSkill after a factory panic did not take effect")
	}
	names := ListSkills()
	if !sort.StringsAreSorted(names) {
		t.Error("ListSkills returned unsorted names after a factory panic")
	}
	found := false
	for _, n := range names {
		if n == "after_panic_skill" {
			found = true
		}
	}
	if !found {
		t.Error("after_panic_skill absent from ListSkills after a factory panic")
	}

	// And the schema walk itself works again once the bad factory is gone.
	if _, err := func() (m map[string]map[string]map[string]any, err error) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("ListSkillsWithParams panicked after recovery: %v", rec)
			}
		}()
		return ListSkillsWithParams(), nil
	}(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
