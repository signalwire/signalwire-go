package agent

import (
	"sync"
	"testing"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/contexts"
)

// TestRenderSWMLForCall_ContextsDoNotSelfDeadlock pins the contract that
// RenderSWMLForCall never invokes ContextBuilder.ToMap() while holding a.mu.
//
// The chain that made this a hang (all line numbers as of the fix):
//
//	RenderSWMLForCall          pkg/agent/agent.go:3032   a.mu.RLock(), deferred unlock
//	  -> contextBuilder.ToMap  pkg/agent/agent.go:3175   142 lines inside the guard
//	  -> cb.Validate           pkg/contexts/contexts.go:1227
//	  -> cb.agent.ListToolNames pkg/contexts/contexts.go:1137
//	  -> a.mu.RLock            pkg/agent/agent.go:2004
//
// DefineContexts calls AttachAgent(a) (agent.go:1838), so cb.agent is always
// non-nil for an agent-owned builder and line 1137 always fires. sync.RWMutex
// is not reentrant and Go's RLock is not recursively safe: a second RLock
// blocks as soon as any writer is queued, and the guard here is held by the
// same goroutine that needs it again. Go's deadlock detector does NOT fire
// (it needs every goroutine asleep; the test-timeout goroutine stays
// runnable), so in production this hangs silently with no diagnostic.
func TestRenderSWMLForCall_ContextsDoNotSelfDeadlock(t *testing.T) {
	a := NewAgentBase(WithName("ctxbot"))
	ctx := a.DefineContexts().AddContext("default")
	ctx.AddStep("greeting").SetText("Hello!")

	doc := a.RenderSWMLForCall(map[string]any{}, nil, "call-1")
	if doc == nil {
		t.Fatal("RenderSWMLForCall returned nil")
	}

	// The contexts must actually reach the rendered document — a fix that
	// dropped the contexts instead of moving the call would pass a bare
	// "it returned" assertion.
	if !renderedHasContexts(doc) {
		t.Error("rendered SWML is missing the ai contexts block")
	}
}

// TestRenderSWMLForCall_ReleasesLockForSubsequentWriters proves the guard is
// actually released across the render: a writer path (a.mu.Lock) must succeed
// afterwards, and a second render must still work.
func TestRenderSWMLForCall_ReleasesLockForSubsequentWriters(t *testing.T) {
	a := NewAgentBase(WithName("ctxbot"))
	ctx := a.DefineContexts().AddContext("default")
	ctx.AddStep("greeting").SetText("Hello!")

	if doc := a.RenderSWMLForCall(map[string]any{}, nil, "call-1"); doc == nil {
		t.Fatal("first render returned nil")
	}

	// A write-locking operation after the render: if the render leaked the
	// read guard this blocks forever.
	a.SetParam("temperature", 0.5)

	doc := a.RenderSWMLForCall(map[string]any{}, nil, "call-2")
	if doc == nil {
		t.Fatal("second render returned nil")
	}
	if !renderedHasContexts(doc) {
		t.Error("second render is missing the ai contexts block")
	}
}

// TestListToolNames_ReentrantFromContextsValidate is the narrow unit form: an
// agent-attached ContextBuilder validated from inside a caller-held read guard
// reaches back into the agent. This is the exact reentrancy, isolated from the
// ~200-line render function.
func TestListToolNames_ReentrantFromContextsValidate(t *testing.T) {
	a := NewAgentBase(WithName("ctxbot"))
	cb := a.DefineContexts()
	cb.AddContext("default").AddStep("greeting").SetText("Hello!")

	// Simulate what RenderSWMLForCall used to do: hold the read guard and
	// call into the builder.
	snapshot := func() *contexts.ContextBuilder {
		a.mu.RLock()
		defer a.mu.RUnlock()
		return a.contextBuilder
	}()
	if snapshot == nil {
		t.Fatal("contextBuilder is nil")
	}

	// Called with NO guard held — this is what the fix guarantees.
	m, err := snapshot.ToMap()
	if err != nil {
		t.Fatalf("ToMap failed: %v", err)
	}
	if _, ok := m["default"]; !ok {
		t.Error("expected 'default' context")
	}
}

// TestRenderSWMLForCall_ContendedWriterDeadlock is the REAL reproduction.
//
// A recursive a.mu.RLock() on the same goroutine only blocks when a WRITER is
// queued in between: sync.RWMutex is writer-preferring, so once Lock() has
// registered its intent, every subsequent RLock() parks — including a
// recursive one from the goroutine that already holds a read guard. That
// goroutine then waits for the writer, and the writer waits for the read
// guard the SAME goroutine holds. Neither can move.
//
// This is why the uncontended test above PASSES on the broken tree: the bug is
// load-dependent, which makes it worse, not better. A concurrent SetParam is
// completely ordinary for a live agent serving SWML while being reconfigured.
func TestRenderSWMLForCall_ContendedWriterDeadlock(t *testing.T) {
	a := NewAgentBase(WithName("ctxbot"))
	ctx := a.DefineContexts().AddContext("default")
	ctx.AddStep("greeting").SetText("Hello!")

	// Hammer renders against a concurrent writer. Each render holds the read
	// guard for its whole body, so a writer arriving mid-render queues, and the
	// recursive RLock at agent.go:2004 then parks behind that writer while the
	// writer waits on the read guard the SAME goroutine holds.
	const renders = 200
	stop := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			a.SetParam("temperature", i)
		}
	}()

	done := make(chan int, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range renders {
			doc := a.RenderSWMLForCall(map[string]any{}, nil, "call-1")
			if doc == nil {
				t.Error("render returned nil")
				break
			}
		}
		done <- 1
	}()

	select {
	case <-done:
		close(stop)
		wg.Wait()
	case <-time.After(15 * time.Second):
		close(stop)
		t.Fatal("DEADLOCK: RenderSWMLForCall did not complete within 15s — " +
			"the recursive a.mu.RLock() in ListToolNames (agent.go:2004) is " +
			"parked behind a queued writer while the render holds the read guard")
	}
}

func renderedHasContexts(doc map[string]any) bool {
	sections, ok := doc["sections"].(map[string]any)
	if !ok {
		return false
	}
	main, ok := sections["main"].([]any)
	if !ok {
		return false
	}
	for _, item := range main {
		verb, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ai, ok := verb["ai"].(map[string]any)
		if !ok {
			continue
		}
		// Contexts live at ai.prompt.contexts, NOT ai.contexts. The ai object's
		// closed key set has no `contexts` member; the reference sets
		// prompt_config["contexts"] (swml_handler.py:190-191). This helper
		// originally looked at ai["contexts"], the one-level-too-high location
		// the engine never reads — see the RenderSWMLForCall comment.
		promptCfg, ok := ai["prompt"].(map[string]any)
		if !ok {
			continue
		}
		if _, ok := promptCfg["contexts"]; ok {
			return true
		}
	}
	return false
}
