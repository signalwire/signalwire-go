package capabilities

import (
	"reflect"
	"testing"
)

func body(caps map[string]any) map[string]any {
	return map[string]any{"vars": map[string]any{"userVariables": map[string]any{"capabilities": caps}}}
}

func TestDeclaredCapabilitiesFromBody(t *testing.T) {
	b := body(map[string]any{"display_content": true, "transcript": true, "chat_handoff": false})
	if got := DeclaredCapabilities(b); !reflect.DeepEqual(got, []string{"display_content", "transcript"}) {
		t.Errorf("DeclaredCapabilities = %v", got)
	}
	if !HasCapability(b, "display_content") || HasCapability(b, "chat_handoff") {
		t.Error("HasCapability disagrees with the declaration")
	}
}

func TestDeclaredCapabilitiesFromUserVariables(t *testing.T) {
	uv := map[string]any{"capabilities": map[string]any{"transcript": true}}
	if got := DeclaredCapabilities(uv); !reflect.DeepEqual(got, []string{"transcript"}) {
		t.Errorf("DeclaredCapabilities(user vars) = %v", got)
	}
}

func TestAbsenceMeansNo(t *testing.T) {
	for _, b := range []any{nil, 42, "x", map[string]any{}, map[string]any{"vars": "bad"},
		body(nil), map[string]any{"vars": map[string]any{"userVariables": map[string]any{"capabilities": []any{"x"}}}}} {
		if got := DeclaredCapabilities(b); len(got) != 0 {
			t.Errorf("DeclaredCapabilities(%v) = %v, want empty", b, got)
		}
		if HasCapability(b, "display_content") {
			t.Errorf("HasCapability(%v) = true", b)
		}
	}
	if got := UserVariables(map[string]any{"vars": map[string]any{}}); len(got) != 0 {
		t.Errorf("UserVariables = %v, want empty", got)
	}
}
