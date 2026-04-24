package namespaces

import (
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// Mock HTTP client for testing
// ---------------------------------------------------------------------------

type mockHTTP struct {
	lastMethod string
	lastPath   string
	lastBody   map[string]any
	lastParams map[string]string
	response   map[string]any
	err        error
}

func (m *mockHTTP) Get(path string, params map[string]string) (map[string]any, error) {
	m.lastMethod = "GET"
	m.lastPath = path
	m.lastParams = params
	return m.response, m.err
}

func (m *mockHTTP) Post(path string, body map[string]any) (map[string]any, error) {
	m.lastMethod = "POST"
	m.lastPath = path
	m.lastBody = body
	return m.response, m.err
}

func (m *mockHTTP) Put(path string, body map[string]any) (map[string]any, error) {
	m.lastMethod = "PUT"
	m.lastPath = path
	m.lastBody = body
	return m.response, m.err
}

func (m *mockHTTP) Patch(path string, body map[string]any) (map[string]any, error) {
	m.lastMethod = "PATCH"
	m.lastPath = path
	m.lastBody = body
	return m.response, m.err
}

func (m *mockHTTP) Delete(path string) error {
	m.lastMethod = "DELETE"
	m.lastPath = path
	return m.err
}

// ---------------------------------------------------------------------------
// Resource path construction
// ---------------------------------------------------------------------------

func TestResource_Path_NoArgs(t *testing.T) {
	r := Resource{Base: "/api/test"}
	if r.Path() != "/api/test" {
		t.Errorf("Path() = %q, want %q", r.Path(), "/api/test")
	}
}

func TestResource_Path_SingleArg(t *testing.T) {
	r := Resource{Base: "/api/test"}
	p := r.Path("123")
	if p != "/api/test/123" {
		t.Errorf("Path(123) = %q, want %q", p, "/api/test/123")
	}
}

func TestResource_Path_MultipleArgs(t *testing.T) {
	r := Resource{Base: "/api/test"}
	p := r.Path("123", "sub", "456")
	expected := "/api/test/123/sub/456"
	if p != expected {
		t.Errorf("Path(123, sub, 456) = %q, want %q", p, expected)
	}
}

// ---------------------------------------------------------------------------
// CrudResource operations
// ---------------------------------------------------------------------------

func TestCrudResource_List(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{"data": []any{}}}
	r := NewCrudResource(mock, "/api/test")
	_, _ = r.List(map[string]string{"page": "1"})
	if mock.lastMethod != "GET" {
		t.Errorf("method = %q, want GET", mock.lastMethod)
	}
	if mock.lastPath != "/api/test" {
		t.Errorf("path = %q, want /api/test", mock.lastPath)
	}
}

func TestCrudResource_Create(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{"id": "new-1"}}
	r := NewCrudResource(mock, "/api/test")
	_, _ = r.Create(map[string]any{"name": "item"})
	if mock.lastMethod != "POST" {
		t.Errorf("method = %q, want POST", mock.lastMethod)
	}
	if mock.lastBody["name"] != "item" {
		t.Errorf("body[name] = %v", mock.lastBody["name"])
	}
}

func TestCrudResource_Get(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{"id": "abc"}}
	r := NewCrudResource(mock, "/api/test")
	_, _ = r.Get("abc")
	if mock.lastPath != "/api/test/abc" {
		t.Errorf("path = %q, want /api/test/abc", mock.lastPath)
	}
}

func TestCrudResource_Update_PATCH(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	r := NewCrudResource(mock, "/api/test")
	_, _ = r.Update("abc", map[string]any{"name": "updated"})
	if mock.lastMethod != "PATCH" {
		t.Errorf("method = %q, want PATCH", mock.lastMethod)
	}
	if mock.lastPath != "/api/test/abc" {
		t.Errorf("path = %q", mock.lastPath)
	}
}

func TestCrudResource_Update_PUT(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	r := NewCrudResourcePUT(mock, "/api/test")
	_, _ = r.Update("abc", map[string]any{"name": "updated"})
	if mock.lastMethod != "PUT" {
		t.Errorf("method = %q, want PUT", mock.lastMethod)
	}
}

func TestCrudResource_Delete(t *testing.T) {
	mock := &mockHTTP{}
	r := NewCrudResource(mock, "/api/test")
	_ = r.Delete("abc")
	if mock.lastMethod != "DELETE" {
		t.Errorf("method = %q, want DELETE", mock.lastMethod)
	}
	if mock.lastPath != "/api/test/abc" {
		t.Errorf("path = %q", mock.lastPath)
	}
}

func TestCrudResource_ListAddresses(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{"data": []any{}}}
	r := NewCrudResource(mock, "/api/test")
	_, _ = r.ListAddresses("abc", nil)
	if mock.lastPath != "/api/test/abc/addresses" {
		t.Errorf("path = %q, want /api/test/abc/addresses", mock.lastPath)
	}
}

func TestCrudResource_ErrorPropagation(t *testing.T) {
	mock := &mockHTTP{err: fmt.Errorf("network error")}
	r := NewCrudResource(mock, "/api/test")
	_, err := r.List(nil)
	if err == nil {
		t.Error("expected error")
	}
}

// ---------------------------------------------------------------------------
// CallingNamespace
// ---------------------------------------------------------------------------

func TestCallingNamespace_Dial(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.Dial(map[string]any{"to": "+1555"})
	if mock.lastMethod != "POST" {
		t.Errorf("method = %q, want POST", mock.lastMethod)
	}
	if mock.lastPath != "/api/calling/calls" {
		t.Errorf("path = %q", mock.lastPath)
	}
	if mock.lastBody["command"] != "dial" {
		t.Errorf("command = %v, want dial", mock.lastBody["command"])
	}
}

func TestCallingNamespace_End(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.End("call-1", map[string]any{})
	if mock.lastBody["command"] != "calling.end" {
		t.Errorf("command = %v, want calling.end", mock.lastBody["command"])
	}
	if mock.lastBody["id"] != "call-1" {
		t.Errorf("id = %v, want call-1", mock.lastBody["id"])
	}
}

func TestCallingNamespace_Play(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.Play("call-1", map[string]any{"url": "audio.mp3"})
	if mock.lastBody["command"] != "calling.play" {
		t.Errorf("command = %v, want calling.play", mock.lastBody["command"])
	}
}

func TestCallingNamespace_PlayPause(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.PlayPause("call-1", nil)
	if mock.lastBody["command"] != "calling.play.pause" {
		t.Errorf("command = %v", mock.lastBody["command"])
	}
}

func TestCallingNamespace_Record(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.Record("call-1", map[string]any{"format": "mp3"})
	if mock.lastBody["command"] != "calling.record" {
		t.Errorf("command = %v", mock.lastBody["command"])
	}
}

func TestCallingNamespace_Transfer(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.Transfer("call-1", map[string]any{})
	if mock.lastBody["command"] != "calling.transfer" {
		t.Errorf("command = %v", mock.lastBody["command"])
	}
}

func TestCallingNamespace_AIMessage(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.AIMessage("call-1", map[string]any{"text": "hello"})
	if mock.lastBody["command"] != "calling.ai_message" {
		t.Errorf("command = %v", mock.lastBody["command"])
	}
}

func TestCallingNamespace_Stream(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.Stream("call-1", map[string]any{"url": "wss://example.com"})
	if mock.lastBody["command"] != "calling.stream" {
		t.Errorf("command = %v", mock.lastBody["command"])
	}
}

func TestCallingNamespace_Denoise(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.Denoise("call-1", nil)
	if mock.lastBody["command"] != "calling.denoise" {
		t.Errorf("command = %v", mock.lastBody["command"])
	}
}

func TestCallingNamespace_Transcribe(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.Transcribe("call-1", nil)
	if mock.lastBody["command"] != "calling.transcribe" {
		t.Errorf("command = %v", mock.lastBody["command"])
	}
}

func TestCallingNamespace_NoCallID(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.Dial(nil)
	if _, ok := mock.lastBody["id"]; ok {
		t.Error("Dial should not include id field")
	}
}

// ---------------------------------------------------------------------------
// FabricNamespace
// ---------------------------------------------------------------------------

func TestFabricNamespace_Init(t *testing.T) {
	mock := &mockHTTP{}
	f := NewFabricNamespace(mock)

	checks := []struct {
		name string
		val  any
	}{
		{"SWMLScripts", f.SWMLScripts},
		{"RelayApplications", f.RelayApplications},
		{"CallFlows", f.CallFlows},
		{"ConferenceRooms", f.ConferenceRooms},
		{"FreeSwitchConnectors", f.FreeSwitchConnectors},
		{"Subscribers", f.Subscribers},
		{"SIPEndpoints", f.SIPEndpoints},
		{"CXMLScripts", f.CXMLScripts},
		{"CXMLApplications", f.CXMLApplications},
		{"SWMLWebhooks", f.SWMLWebhooks},
		{"AIAgents", f.AIAgents},
		{"SIPGateways", f.SIPGateways},
		{"CXMLWebhooks", f.CXMLWebhooks},
		{"Resources", f.Resources},
		{"Addresses", f.Addresses},
		{"Tokens", f.Tokens},
	}

	for _, check := range checks {
		if check.val == nil {
			t.Errorf("Fabric.%s is nil", check.name)
		}
	}
}

func TestCallFlowsResource_ListVersions(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{"data": []any{}}}
	f := NewFabricNamespace(mock)
	_, _ = f.CallFlows.ListVersions("flow-1", nil)
	if mock.lastMethod != "GET" {
		t.Errorf("method = %q, want GET", mock.lastMethod)
	}
	if mock.lastPath == "" {
		t.Error("expected non-empty path")
	}
}

func TestCallFlowsResource_DeployVersion(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	f := NewFabricNamespace(mock)
	_, _ = f.CallFlows.DeployVersion("flow-1", map[string]any{"swml": "{}"})
	if mock.lastMethod != "POST" {
		t.Errorf("method = %q, want POST", mock.lastMethod)
	}
}

func TestSubscribersResource_ListSIPEndpoints(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{"data": []any{}}}
	f := NewFabricNamespace(mock)
	_, _ = f.Subscribers.ListSIPEndpoints("sub-1", nil)
	if mock.lastMethod != "GET" {
		t.Errorf("method = %q", mock.lastMethod)
	}
}

func TestSubscribersResource_CRUD_SIPEndpoint(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	f := NewFabricNamespace(mock)

	_, _ = f.Subscribers.CreateSIPEndpoint("sub-1", map[string]any{"name": "ep"})
	if mock.lastMethod != "POST" {
		t.Errorf("create method = %q, want POST", mock.lastMethod)
	}

	_, _ = f.Subscribers.GetSIPEndpoint("sub-1", "ep-1")
	if mock.lastMethod != "GET" {
		t.Errorf("get method = %q, want GET", mock.lastMethod)
	}

	_, _ = f.Subscribers.UpdateSIPEndpoint("sub-1", "ep-1", map[string]any{"name": "new"})
	if mock.lastMethod != "PATCH" {
		t.Errorf("update method = %q, want PATCH", mock.lastMethod)
	}

	_ = f.Subscribers.DeleteSIPEndpoint("sub-1", "ep-1")
	if mock.lastMethod != "DELETE" {
		t.Errorf("delete method = %q, want DELETE", mock.lastMethod)
	}
}

func TestFabricTokens_CreateSubscriberToken(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{"token": "xyz"}}
	f := NewFabricNamespace(mock)
	_, _ = f.Tokens.CreateSubscriberToken(map[string]any{"subscriber_id": "sub-1"})
	if mock.lastMethod != "POST" {
		t.Errorf("method = %q, want POST", mock.lastMethod)
	}
}

func TestFabricTokens_CreateGuestToken(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{"token": "abc"}}
	f := NewFabricNamespace(mock)
	_, _ = f.Tokens.CreateGuestToken(map[string]any{})
	if mock.lastMethod != "POST" {
		t.Errorf("method = %q, want POST", mock.lastMethod)
	}
}

func TestGenericResources_Operations(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	f := NewFabricNamespace(mock)

	_, _ = f.Resources.List(nil)
	if mock.lastMethod != "GET" {
		t.Errorf("list method = %q", mock.lastMethod)
	}

	_, _ = f.Resources.Get("res-1")
	if mock.lastMethod != "GET" {
		t.Errorf("get method = %q", mock.lastMethod)
	}

	_ = f.Resources.Delete("res-1")
	if mock.lastMethod != "DELETE" {
		t.Errorf("delete method = %q", mock.lastMethod)
	}

	_, _ = f.Resources.ListAddresses("res-1", nil)
	if mock.lastMethod != "GET" {
		t.Errorf("list addresses method = %q", mock.lastMethod)
	}

	_, _ = f.Resources.AssignPhoneRoute("res-1", map[string]any{})
	if mock.lastMethod != "POST" {
		t.Errorf("assign phone route method = %q", mock.lastMethod)
	}
}

func TestFabricAddresses_Operations(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	f := NewFabricNamespace(mock)

	_, _ = f.Addresses.List(nil)
	if mock.lastMethod != "GET" {
		t.Errorf("list method = %q", mock.lastMethod)
	}

	_, _ = f.Addresses.Get("addr-1")
	if mock.lastMethod != "GET" {
		t.Errorf("get method = %q", mock.lastMethod)
	}
}

// ---------------------------------------------------------------------------
// UpdateMethod defaults
// ---------------------------------------------------------------------------

func TestFabricNamespace_PUTResources(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	f := NewFabricNamespace(mock)

	// PUT-update resources
	putResources := []*CrudResource{
		f.SWMLScripts,
		f.RelayApplications,
		f.CallFlows.CrudResource,
		f.FreeSwitchConnectors,
		f.SIPEndpoints,
	}
	for _, r := range putResources {
		if r.UpdateMethod != "PUT" {
			t.Errorf("resource with base %q has UpdateMethod=%q, want PUT", r.Base, r.UpdateMethod)
		}
	}
}

func TestFabricNamespace_PATCHResources(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	f := NewFabricNamespace(mock)

	// PATCH-update resources
	patchResources := []*CrudResource{
		f.SWMLWebhooks.CrudResource,
		f.AIAgents,
		f.SIPGateways,
		f.CXMLWebhooks.CrudResource,
	}
	for _, r := range patchResources {
		if r.UpdateMethod != "PATCH" {
			t.Errorf("resource with base %q has UpdateMethod=%q, want PATCH", r.Base, r.UpdateMethod)
		}
	}
}
