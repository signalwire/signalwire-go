package namespaces

import (
	"context"
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
	lastOpts   *RequestOptions // first per-request override the verb forwarded (GO-1)
	response   map[string]any
	err        error
}

// firstMockOpt records the first non-nil per-request override the verb forwarded,
// so a test can assert request_options threads through (GO-1 / PY-7).
func firstMockOpt(opts []*RequestOptions) *RequestOptions {
	for _, o := range opts {
		if o != nil {
			return o
		}
	}
	return nil
}

func (m *mockHTTP) Get(_ context.Context, path string, params map[string]string, opts ...*RequestOptions) (map[string]any, error) {
	m.lastMethod = "GET"
	m.lastPath = path
	m.lastParams = params
	m.lastOpts = firstMockOpt(opts)
	return m.response, m.err
}

func (m *mockHTTP) Post(_ context.Context, path string, body map[string]any, params map[string]string, opts ...*RequestOptions) (map[string]any, error) {
	m.lastMethod = "POST"
	m.lastPath = path
	m.lastBody = body
	m.lastParams = params
	m.lastOpts = firstMockOpt(opts)
	return m.response, m.err
}

func (m *mockHTTP) Put(_ context.Context, path string, body map[string]any, opts ...*RequestOptions) (map[string]any, error) {
	m.lastMethod = "PUT"
	m.lastPath = path
	m.lastBody = body
	m.lastOpts = firstMockOpt(opts)
	return m.response, m.err
}

func (m *mockHTTP) Patch(_ context.Context, path string, body map[string]any, opts ...*RequestOptions) (map[string]any, error) {
	m.lastMethod = "PATCH"
	m.lastPath = path
	m.lastBody = body
	m.lastOpts = firstMockOpt(opts)
	return m.response, m.err
}

func (m *mockHTTP) Delete(_ context.Context, path string, opts ...*RequestOptions) (map[string]any, error) {
	m.lastMethod = "DELETE"
	m.lastPath = path
	m.lastOpts = firstMockOpt(opts)
	return m.response, m.err
}

// ---------------------------------------------------------------------------
// request_options threading (GO-1 / PY-7): a per-request *RequestOptions passed
// to a generated verb reaches the HTTP layer (never the wire body). The mock
// captures the first forwarded override in lastOpts.
// ---------------------------------------------------------------------------

func TestRequestOptions_ThreadsThroughCrudVerbs(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	r := NewCrudResource(mock, "/api/things")
	to := 12.5
	ro := &RequestOptions{Timeout: &to}

	cases := []struct {
		name string
		call func()
	}{
		{"List", func() { _, _ = r.List(context.Background(), nil, ro) }},
		{"Get", func() { _, _ = r.Get(context.Background(), "id-1", ro) }},
		{"Create", func() { _, _ = r.Create(context.Background(), map[string]any{"a": 1}, ro) }},
		{"Update", func() { _, _ = r.Update(context.Background(), "id-1", map[string]any{"a": 1}, ro) }},
		{"Delete", func() { _, _ = r.Delete(context.Background(), "id-1", ro) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mock.lastOpts = nil
			c.call()
			if mock.lastOpts != ro {
				t.Errorf("%s: request_options not threaded to HTTP layer (lastOpts=%v)", c.name, mock.lastOpts)
			}
			// Never serialized into the wire body.
			if mock.lastBody != nil {
				if _, leaked := mock.lastBody["request_options"]; leaked {
					t.Errorf("%s: request_options leaked into wire body", c.name)
				}
			}
		})
	}
}

func TestRequestOptions_OmittedIsNil(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	r := NewCrudResource(mock, "/api/things")
	mock.lastOpts = &RequestOptions{} // sentinel to prove the verb overwrites it
	_, _ = r.Get(context.Background(), "id-1")
	if mock.lastOpts != nil {
		t.Errorf("no request_options passed, but HTTP layer saw %v", mock.lastOpts)
	}
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
	_, _ = r.List(context.Background(), map[string]string{"page": "1"})
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
	_, _ = r.Create(context.Background(), map[string]any{"name": "item"})
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
	_, _ = r.Get(context.Background(), "abc")
	if mock.lastPath != "/api/test/abc" {
		t.Errorf("path = %q, want /api/test/abc", mock.lastPath)
	}
}

func TestCrudResource_Update_PATCH(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	r := NewCrudResource(mock, "/api/test")
	_, _ = r.Update(context.Background(), "abc", map[string]any{"name": "updated"})
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
	_, _ = r.Update(context.Background(), "abc", map[string]any{"name": "updated"})
	if mock.lastMethod != "PUT" {
		t.Errorf("method = %q, want PUT", mock.lastMethod)
	}
}

func TestCrudResource_Delete(t *testing.T) {
	mock := &mockHTTP{}
	r := NewCrudResource(mock, "/api/test")
	_, _ = r.Delete(context.Background(), "abc")
	if mock.lastMethod != "DELETE" {
		t.Errorf("method = %q, want DELETE", mock.lastMethod)
	}
	if mock.lastPath != "/api/test/abc" {
		t.Errorf("path = %q", mock.lastPath)
	}
}

func TestCrudWithAddresses_ListAddresses(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{"data": []any{}}}
	r := NewCrudWithAddresses(mock, "/api/test")
	_, _ = r.ListAddresses(context.Background(), "abc", nil)
	if mock.lastPath != "/api/test/abc/addresses" {
		t.Errorf("path = %q, want /api/test/abc/addresses", mock.lastPath)
	}
}

func TestCrudResource_ErrorPropagation(t *testing.T) {
	mock := &mockHTTP{err: fmt.Errorf("network error")}
	r := NewCrudResource(mock, "/api/test")
	_, err := r.List(context.Background(), nil)
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
	_, _ = c.Dial(context.Background(), CallingNamespaceDialParams{Extras: map[string]any{"to": "+1555"}})
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
	_, _ = c.End(context.Background(), "call-1", CallingNamespaceEndParams{})
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
	_, _ = c.Play(context.Background(), "call-1", CallingNamespacePlayParams{Extras: map[string]any{"url": "audio.mp3"}})
	if mock.lastBody["command"] != "calling.play" {
		t.Errorf("command = %v, want calling.play", mock.lastBody["command"])
	}
}

func TestCallingNamespace_PlayPause(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.PlayPause(context.Background(), "call-1", CallingNamespacePlayPauseParams{})
	if mock.lastBody["command"] != "calling.play.pause" {
		t.Errorf("command = %v", mock.lastBody["command"])
	}
}

func TestCallingNamespace_Record(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.Record(context.Background(), "call-1", CallingNamespaceRecordParams{Extras: map[string]any{"format": "mp3"}})
	if mock.lastBody["command"] != "calling.record" {
		t.Errorf("command = %v", mock.lastBody["command"])
	}
}

func TestCallingNamespace_Transfer(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.Transfer(context.Background(), "call-1", CallingNamespaceTransferParams{})
	if mock.lastBody["command"] != "calling.transfer" {
		t.Errorf("command = %v", mock.lastBody["command"])
	}
}

func TestCallingNamespace_AIMessage(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.AIMessage(context.Background(), "call-1", CallingNamespaceAIMessageParams{Extras: map[string]any{"text": "hello"}})
	if mock.lastBody["command"] != "calling.ai_message" {
		t.Errorf("command = %v", mock.lastBody["command"])
	}
}

func TestCallingNamespace_Stream(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.Stream(context.Background(), "call-1", CallingNamespaceStreamParams{Extras: map[string]any{"url": "wss://example.com"}})
	if mock.lastBody["command"] != "calling.stream" {
		t.Errorf("command = %v", mock.lastBody["command"])
	}
}

func TestCallingNamespace_Denoise(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.Denoise(context.Background(), "call-1", CallingNamespaceDenoiseParams{})
	if mock.lastBody["command"] != "calling.denoise" {
		t.Errorf("command = %v", mock.lastBody["command"])
	}
}

func TestCallingNamespace_Transcribe(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.Transcribe(context.Background(), "call-1", CallingNamespaceTranscribeParams{})
	if mock.lastBody["command"] != "calling.transcribe" {
		t.Errorf("command = %v", mock.lastBody["command"])
	}
}

func TestCallingNamespace_NoCallID(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	c := NewCallingNamespace(mock)
	_, _ = c.Dial(context.Background(), CallingNamespaceDialParams{})
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
	_, _ = f.CallFlows.ListVersions(context.Background(), "flow-1", nil)
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
	_, _ = f.CallFlows.DeployVersion(context.Background(), "flow-1", map[string]any{"swml": "{}"})
	if mock.lastMethod != "POST" {
		t.Errorf("method = %q, want POST", mock.lastMethod)
	}
}

func TestSubscribersResource_ListSIPEndpoints(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{"data": []any{}}}
	f := NewFabricNamespace(mock)
	_, _ = f.Subscribers.ListSIPEndpoints(context.Background(), "sub-1", nil)
	if mock.lastMethod != "GET" {
		t.Errorf("method = %q", mock.lastMethod)
	}
}

func TestSubscribersResource_CRUD_SIPEndpoint(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	f := NewFabricNamespace(mock)

	_, _ = f.Subscribers.CreateSIPEndpoint(context.Background(), "sub-1", SubscribersResourceCreateSIPEndpointParams{Extras: map[string]any{"name": "ep"}})
	if mock.lastMethod != "POST" {
		t.Errorf("create method = %q, want POST", mock.lastMethod)
	}

	_, _ = f.Subscribers.GetSIPEndpoint(context.Background(), "sub-1", "ep-1", nil)
	if mock.lastMethod != "GET" {
		t.Errorf("get method = %q, want GET", mock.lastMethod)
	}

	_, _ = f.Subscribers.UpdateSIPEndpoint(context.Background(), "sub-1", "ep-1", SubscribersResourceUpdateSIPEndpointParams{Extras: map[string]any{"name": "new"}})
	if mock.lastMethod != "PATCH" {
		t.Errorf("update method = %q, want PATCH", mock.lastMethod)
	}

	_, _ = f.Subscribers.DeleteSIPEndpoint(context.Background(), "sub-1", "ep-1")
	if mock.lastMethod != "DELETE" {
		t.Errorf("delete method = %q, want DELETE", mock.lastMethod)
	}
}

func TestFabricTokens_CreateSubscriberToken(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{"token": "xyz"}}
	f := NewFabricNamespace(mock)
	_, _ = f.Tokens.CreateSubscriberToken(context.Background(), FabricTokensCreateSubscriberTokenParams{Extras: map[string]any{"subscriber_id": "sub-1"}})
	if mock.lastMethod != "POST" {
		t.Errorf("method = %q, want POST", mock.lastMethod)
	}
}

func TestFabricTokens_CreateGuestToken(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{"token": "abc"}}
	f := NewFabricNamespace(mock)
	_, _ = f.Tokens.CreateGuestToken(context.Background(), FabricTokensCreateGuestTokenParams{})
	if mock.lastMethod != "POST" {
		t.Errorf("method = %q, want POST", mock.lastMethod)
	}
}

func TestGenericResources_Operations(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	f := NewFabricNamespace(mock)

	_, _ = f.Resources.List(context.Background(), nil)
	if mock.lastMethod != "GET" {
		t.Errorf("list method = %q", mock.lastMethod)
	}

	_, _ = f.Resources.Get(context.Background(), "res-1", nil)
	if mock.lastMethod != "GET" {
		t.Errorf("get method = %q", mock.lastMethod)
	}

	_, _ = f.Resources.Delete(context.Background(), "res-1")
	if mock.lastMethod != "DELETE" {
		t.Errorf("delete method = %q", mock.lastMethod)
	}

	_, _ = f.Resources.ListAddresses(context.Background(), "res-1", nil)
	if mock.lastMethod != "GET" {
		t.Errorf("list addresses method = %q", mock.lastMethod)
	}

	_, _ = f.Resources.AssignPhoneRoute(context.Background(), "res-1", GenericResourcesAssignPhoneRouteParams{})
	if mock.lastMethod != "POST" {
		t.Errorf("assign phone route method = %q", mock.lastMethod)
	}
}

func TestFabricAddresses_Operations(t *testing.T) {
	mock := &mockHTTP{response: map[string]any{}}
	f := NewFabricNamespace(mock)

	_, _ = f.Addresses.List(context.Background(), nil)
	if mock.lastMethod != "GET" {
		t.Errorf("list method = %q", mock.lastMethod)
	}

	_, _ = f.Addresses.Get(context.Background(), "addr-1")
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
		f.SWMLScripts.CrudResource,
		f.RelayApplications.CrudResource,
		f.CallFlows.CrudResource,
		f.FreeSwitchConnectors.CrudResource,
		f.SIPEndpoints.CrudResource,
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
		f.AIAgents.CrudResource,
		f.SIPGateways.CrudResource,
		f.CXMLWebhooks.CrudResource,
	}
	for _, r := range patchResources {
		if r.UpdateMethod != "PATCH" {
			t.Errorf("resource with base %q has UpdateMethod=%q, want PATCH", r.Base, r.UpdateMethod)
		}
	}
}
