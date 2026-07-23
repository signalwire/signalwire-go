package contexts

import "testing"

// STRICT-RENDER contexts-level (Wave-2 P#5, GAP2 / r5 F3): a step whose
// SetFunctions([...]) whitelist names a function that is neither a registered
// SWAIG tool nor a reserved native tool is a DANGLING reference and must raise
// at validation time. Mirrors the contexts cases of signalwire-python
// tests/unit/core/test_swml_strict_render.py.
//
// The dangling-ref check only runs when a real tool universe is attached, so
// these tests use a fakeToolLister to supply the registered tool names — the
// Go analogue of an AgentBase with DefineTool'd tools attached via
// ContextBuilder.AttachAgent.

type fakeToolLister struct{ names []string }

func (f fakeToolLister) ListToolNames() []string { return f.names }

// builder wires a ContextBuilder with the given registered tool names attached,
// so Validate() knows the tool universe.
func builderWithTools(names ...string) *ContextBuilder {
	cb := NewContextBuilder()
	cb.AttachAgent(fakeToolLister{names: names})
	return cb
}

func TestStrictDanglingFunctionRefRaises(t *testing.T) {
	cb := builderWithTools("order_status")
	st := cb.AddContext("default").AddStep("help")
	st.SetText("help")
	// get_datetime is neither registered nor a reserved native -> dangling.
	st.SetFunctions([]string{"order_status", "get_datetime"})
	if _, err := cb.ToMap(); err == nil {
		t.Fatal("dangling step-function ref 'get_datetime' must raise, got nil error")
	}
}

func TestStrictRegisteredFunctionRefRenders(t *testing.T) {
	cb := builderWithTools("order_status")
	st := cb.AddContext("default").AddStep("help")
	st.SetText("help")
	st.SetFunctions([]string{"order_status"})
	if _, err := cb.ToMap(); err != nil {
		t.Fatalf("step referencing a registered tool must render, got error: %v", err)
	}
}

func TestStrictReservedNativeToolRefAllowed(t *testing.T) {
	// next_step / change_context are reserved natives, not dangling — even with
	// no user tools registered.
	cb := builderWithTools()
	st := cb.AddContext("default").AddStep("help")
	st.SetText("help")
	st.SetFunctions([]string{"next_step", "change_context"})
	if _, err := cb.ToMap(); err != nil {
		t.Fatalf("reserved native tool refs must render, got error: %v", err)
	}
}

func TestStrictFunctionsNoneAndEmptyRender(t *testing.T) {
	// "none" (string) and [] (explicit disable-all) are not reference lists and
	// must never be treated as dangling.
	for _, funcs := range []any{"none", []string{}} {
		cb := builderWithTools()
		st := cb.AddContext("default").AddStep("help")
		st.SetText("help")
		st.SetFunctions(funcs)
		if _, err := cb.ToMap(); err != nil {
			t.Fatalf("SetFunctions(%v) must render (disable-all, not dangling), got error: %v", funcs, err)
		}
	}
}

func TestStrictDanglingValidContextRaises(t *testing.T) {
	// valid_contexts referencing an undefined context must raise (already
	// enforced; pinned so a refactor can't loosen it).
	cb := builderWithTools()
	st := cb.AddContext("default").AddStep("help")
	st.SetText("help")
	st.SetValidContexts([]string{"nowhere"})
	if _, err := cb.ToMap(); err == nil {
		t.Fatal("valid_contexts ref to undefined context 'nowhere' must raise, got nil error")
	}
}
