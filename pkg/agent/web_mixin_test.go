package agent

import (
	"net/http"
	"reflect"
	"testing"
)

// ---------------------------------------------------------------------------
// OnRequest / OnSwmlRequest — Python parity tests for WebMixin hooks
//
// Python parity:
//
//	tests/unit/core/mixins/test_web_mixin.py::
//	  test_on_request_delegates_to_on_swml_request
//	  test_on_swml_request_called
//
// Go has no method overriding via embedded structs alone — instead a
// function-field hook (SetOnSwmlRequestHook) is the idiomatic way to inject
// custom behavior into AgentBase.OnSwmlRequest. This mirrors how the .NET
// port exposes virtual methods and how Python exposes a subclass-overridable
// method.
// ---------------------------------------------------------------------------

func TestOnRequest_DelegatesToOnSwmlRequest(t *testing.T) {
	a := NewAgentBase(WithName("t"))

	want := map[string]any{"custom": true}
	var seenRequestData map[string]any
	var seenCallbackPath string
	a.SetOnSwmlRequestHook(func(rd map[string]any, cb string, r *http.Request) map[string]any {
		seenRequestData = rd
		seenCallbackPath = cb
		return want
	})

	rd := map[string]any{"data": "val"}
	got := a.OnRequest(rd, "/cb")

	if !reflect.DeepEqual(seenRequestData, rd) {
		t.Errorf("hook did not receive request_data; got=%v want=%v", seenRequestData, rd)
	}
	if seenCallbackPath != "/cb" {
		t.Errorf("hook did not receive callback_path; got=%q want=%q", seenCallbackPath, "/cb")
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("OnRequest did not return hook result; got=%v want=%v", got, want)
	}
}

func TestOnRequest_DefaultReturnsNil(t *testing.T) {
	a := NewAgentBase(WithName("t"))
	if got := a.OnRequest(nil, ""); got != nil {
		t.Errorf("default OnRequest should return nil; got=%v", got)
	}
}

func TestOnSwmlRequest_DefaultReturnsNil(t *testing.T) {
	a := NewAgentBase(WithName("t"))
	if got := a.OnSwmlRequest(nil, "", nil); got != nil {
		t.Errorf("default OnSwmlRequest should return nil; got=%v", got)
	}
}

func TestOnSwmlRequest_HookInvoked(t *testing.T) {
	a := NewAgentBase(WithName("t"))
	called := false
	a.SetOnSwmlRequestHook(func(rd map[string]any, cb string, r *http.Request) map[string]any {
		called = true
		return nil
	})
	a.OnSwmlRequest(map[string]any{}, "", nil)
	if !called {
		t.Error("registered hook was not invoked by OnSwmlRequest")
	}
}
