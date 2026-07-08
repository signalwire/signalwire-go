// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

import (
	"strings"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// mockHTTP extension: call recording for regression tests
// ---------------------------------------------------------------------------

// callRecorder is a fuller HTTPClient mock that records every call in order
// (the namespaces_test mockHTTP only retains the last call). Used by tests
// that need to assert exact call counts and sequencing.
type callRecorder struct {
	mu    sync.Mutex
	calls []recordedCall
	resp  map[string]any
}

type recordedCall struct {
	Method string
	Path   string
	Body   map[string]any
	Params map[string]string
}

func (m *callRecorder) record(c recordedCall) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, c)
}

func (m *callRecorder) Calls() []recordedCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]recordedCall, len(m.calls))
	copy(out, m.calls)
	return out
}

func (m *callRecorder) Get(path string, params map[string]string) (map[string]any, error) {
	m.record(recordedCall{Method: "GET", Path: path, Params: params})
	return m.resp, nil
}
func (m *callRecorder) Post(path string, body map[string]any, params map[string]string) (map[string]any, error) {
	m.record(recordedCall{Method: "POST", Path: path, Body: body, Params: params})
	return m.resp, nil
}
func (m *callRecorder) Put(path string, body map[string]any) (map[string]any, error) {
	m.record(recordedCall{Method: "PUT", Path: path, Body: body})
	return m.resp, nil
}
func (m *callRecorder) Patch(path string, body map[string]any) (map[string]any, error) {
	m.record(recordedCall{Method: "PATCH", Path: path, Body: body})
	return m.resp, nil
}
func (m *callRecorder) Delete(path string) (map[string]any, error) {
	m.record(recordedCall{Method: "DELETE", Path: path})
	return m.resp, nil
}

// ---------------------------------------------------------------------------
// PhoneCallHandler enum contract
// ---------------------------------------------------------------------------

func TestPhoneCallHandler_AllWireValues(t *testing.T) {
	// Every call_handler value accepted by the API must be in the enum.
	wantSet := map[string]bool{
		"relay_script":      true,
		"laml_webhooks":     true,
		"laml_application":  true,
		"ai_agent":          true,
		"call_flow":         true,
		"relay_application": true,
		"relay_topic":       true,
		"relay_context":     true,
		"relay_connector":   true,
		"video_room":        true,
		"dialogflow":        true,
	}
	got := AllPhoneCallHandlers()
	if len(got) != len(wantSet) {
		t.Fatalf("AllPhoneCallHandlers length = %d, want %d", len(got), len(wantSet))
	}
	seen := map[string]bool{}
	for _, h := range got {
		s := string(h)
		if !wantSet[s] {
			t.Errorf("unexpected handler value %q", s)
		}
		if seen[s] {
			t.Errorf("duplicate handler value %q", s)
		}
		seen[s] = true
	}
	for want := range wantSet {
		if !seen[want] {
			t.Errorf("missing handler value %q", want)
		}
	}
}

func TestPhoneCallHandler_IsString(t *testing.T) {
	// PhoneCallHandler is a string-typed alias so it converts directly.
	if string(PhoneCallHandlerRelayScript) != "relay_script" {
		t.Errorf("PhoneCallHandlerRelayScript = %q, want relay_script",
			string(PhoneCallHandlerRelayScript))
	}
	if string(PhoneCallHandlerAiAgent) != "ai_agent" {
		t.Errorf("PhoneCallHandlerAiAgent = %q, want ai_agent",
			string(PhoneCallHandlerAiAgent))
	}
}

// ---------------------------------------------------------------------------
// PhoneNumbersNamespace: typed binding helpers
// ---------------------------------------------------------------------------

const phoneBase = "/api/relay/rest/phone_numbers"

func newPhoneNumbers() (*PhoneNumbersNamespace, *callRecorder) {
	mock := &callRecorder{resp: map[string]any{}}
	return NewPhoneNumbersNamespace(mock), mock
}

func TestPhoneNumbers_Update_UsesPUT(t *testing.T) {
	pn, mock := newPhoneNumbers()
	_, _ = pn.Update("pn-1", map[string]any{"name": "Main"})
	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	if calls[0].Method != "PUT" {
		t.Errorf("method = %q, want PUT", calls[0].Method)
	}
	if calls[0].Path != phoneBase+"/pn-1" {
		t.Errorf("path = %q, want %s/pn-1", calls[0].Path, phoneBase)
	}
}

func TestSetSwmlWebhook_HappyPath(t *testing.T) {
	pn, mock := newPhoneNumbers()
	_, _ = pn.SetSwmlWebhook("pn-1", "https://example.com/swml")
	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	c := calls[0]
	if c.Method != "PUT" || c.Path != phoneBase+"/pn-1" {
		t.Errorf("got %s %s, want PUT %s/pn-1", c.Method, c.Path, phoneBase)
	}
	if c.Body["call_handler"] != "relay_script" {
		t.Errorf("call_handler = %v, want relay_script", c.Body["call_handler"])
	}
	if c.Body["call_relay_script_url"] != "https://example.com/swml" {
		t.Errorf("call_relay_script_url = %v", c.Body["call_relay_script_url"])
	}
}

func TestSetSwmlWebhook_Extra(t *testing.T) {
	pn, mock := newPhoneNumbers()
	_, _ = pn.SetSwmlWebhook("pn-1", "https://example.com/swml", map[string]any{
		"name": "Support Line",
	})
	body := mock.Calls()[0].Body
	if body["name"] != "Support Line" {
		t.Errorf("name = %v, want Support Line", body["name"])
	}
	if body["call_handler"] != "relay_script" {
		t.Errorf("call_handler = %v", body["call_handler"])
	}
}

func TestSetCxmlWebhook_Minimal(t *testing.T) {
	pn, mock := newPhoneNumbers()
	_, _ = pn.SetCxmlWebhook("pn-1", "https://example.com/voice.xml", nil, nil)
	body := mock.Calls()[0].Body
	if body["call_handler"] != "laml_webhooks" {
		t.Errorf("call_handler = %v, want laml_webhooks", body["call_handler"])
	}
	if body["call_request_url"] != "https://example.com/voice.xml" {
		t.Errorf("call_request_url = %v", body["call_request_url"])
	}
	if _, ok := body["call_fallback_url"]; ok {
		t.Error("call_fallback_url should not be set in minimal form")
	}
	if _, ok := body["call_status_callback_url"]; ok {
		t.Error("call_status_callback_url should not be set in minimal form")
	}
}

func TestSetCxmlWebhook_WithFallbackAndStatus(t *testing.T) {
	pn, mock := newPhoneNumbers()
	fallback := "https://example.com/fallback.xml"
	status := "https://example.com/status"
	_, _ = pn.SetCxmlWebhook("pn-1", "https://example.com/voice.xml", &fallback, &status)
	body := mock.Calls()[0].Body
	want := map[string]any{
		"call_handler":             "laml_webhooks",
		"call_request_url":         "https://example.com/voice.xml",
		"call_fallback_url":        "https://example.com/fallback.xml",
		"call_status_callback_url": "https://example.com/status",
	}
	for k, v := range want {
		if body[k] != v {
			t.Errorf("body[%q] = %v, want %v", k, body[k], v)
		}
	}
	if len(body) != len(want) {
		t.Errorf("body has %d keys, want %d: %v", len(body), len(want), body)
	}
}

func TestSetCxmlApplication_HappyPath(t *testing.T) {
	pn, mock := newPhoneNumbers()
	_, _ = pn.SetCxmlApplication("pn-1", "app-1")
	body := mock.Calls()[0].Body
	if body["call_handler"] != "laml_application" {
		t.Errorf("call_handler = %v, want laml_application", body["call_handler"])
	}
	if body["call_laml_application_id"] != "app-1" {
		t.Errorf("call_laml_application_id = %v", body["call_laml_application_id"])
	}
}

func TestSetAiAgent_HappyPath(t *testing.T) {
	pn, mock := newPhoneNumbers()
	_, _ = pn.SetAiAgent("pn-1", "agent-1")
	body := mock.Calls()[0].Body
	if body["call_handler"] != "ai_agent" {
		t.Errorf("call_handler = %v, want ai_agent", body["call_handler"])
	}
	if body["call_ai_agent_id"] != "agent-1" {
		t.Errorf("call_ai_agent_id = %v", body["call_ai_agent_id"])
	}
}

func TestSetCallFlow_Minimal(t *testing.T) {
	pn, mock := newPhoneNumbers()
	_, _ = pn.SetCallFlow("pn-1", "cf-1", nil)
	body := mock.Calls()[0].Body
	if body["call_handler"] != "call_flow" {
		t.Errorf("call_handler = %v, want call_flow", body["call_handler"])
	}
	if body["call_flow_id"] != "cf-1" {
		t.Errorf("call_flow_id = %v", body["call_flow_id"])
	}
	if _, ok := body["call_flow_version"]; ok {
		t.Error("call_flow_version should not be set when Version is empty")
	}
}

func TestSetCallFlow_WithVersion(t *testing.T) {
	pn, mock := newPhoneNumbers()
	version := "current_deployed"
	_, _ = pn.SetCallFlow("pn-1", "cf-1", &version)
	body := mock.Calls()[0].Body
	if body["call_flow_version"] != "current_deployed" {
		t.Errorf("call_flow_version = %v, want current_deployed", body["call_flow_version"])
	}
}

func TestSetRelayApplication_HappyPath(t *testing.T) {
	pn, mock := newPhoneNumbers()
	_, _ = pn.SetRelayApplication("pn-1", "my-app")
	body := mock.Calls()[0].Body
	if body["call_handler"] != "relay_application" {
		t.Errorf("call_handler = %v, want relay_application", body["call_handler"])
	}
	if body["call_relay_application"] != "my-app" {
		t.Errorf("call_relay_application = %v", body["call_relay_application"])
	}
}

func TestSetRelayTopic_Minimal(t *testing.T) {
	pn, mock := newPhoneNumbers()
	_, _ = pn.SetRelayTopic("pn-1", "office", nil)
	body := mock.Calls()[0].Body
	if body["call_handler"] != "relay_topic" {
		t.Errorf("call_handler = %v, want relay_topic", body["call_handler"])
	}
	if body["call_relay_topic"] != "office" {
		t.Errorf("call_relay_topic = %v", body["call_relay_topic"])
	}
}

func TestSetRelayTopic_WithStatusCallback(t *testing.T) {
	pn, mock := newPhoneNumbers()
	statusCB := "https://example.com/status"
	_, _ = pn.SetRelayTopic("pn-1", "office", &statusCB)
	body := mock.Calls()[0].Body
	if body["call_relay_topic_status_callback_url"] != "https://example.com/status" {
		t.Errorf("call_relay_topic_status_callback_url = %v",
			body["call_relay_topic_status_callback_url"])
	}
}

// ---------------------------------------------------------------------------
// Regression: post-mortem anti-patterns
// ---------------------------------------------------------------------------

// TestBindingRegression_NoFabricWebhookCreate pins the contract that the
// correct happy-path binding is a single PUT to /api/relay/rest/phone_numbers/{sid}
// — no call to fabric.swml_webhooks.create, no assign_phone_route. These
// were the two traps found in the phone-binding post-mortem.
func TestBindingRegression_NoFabricWebhookCreate(t *testing.T) {
	pn, mock := newPhoneNumbers()
	_, _ = pn.SetSwmlWebhook("pn-1", "https://example.com/swml")

	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want exactly 1 (full binding flow is a single PUT)", len(calls))
	}
	c := calls[0]
	if c.Method != "PUT" {
		t.Errorf("method = %q, want PUT", c.Method)
	}
	if c.Path != phoneBase+"/pn-1" {
		t.Errorf("path = %q, want %s/pn-1", c.Path, phoneBase)
	}
	// No /api/fabric/resources/swml_webhooks POST
	if strings.Contains(c.Path, "/api/fabric/resources/swml_webhooks") {
		t.Errorf("SetSwmlWebhook should not hit fabric.SwmlWebhooks.Create: path=%q", c.Path)
	}
	// No /phone_routes POST
	if strings.Contains(c.Path, "/phone_routes") {
		t.Errorf("SetSwmlWebhook should not hit AssignPhoneRoute: path=%q", c.Path)
	}
}

// TestBindingRegression_WireLevelFormStillWorks — callers who bypass the
// helpers and pass call_handler directly to Update get the same result.
func TestBindingRegression_WireLevelFormStillWorks(t *testing.T) {
	pn, mock := newPhoneNumbers()
	_, _ = pn.Update("pn-1", map[string]any{
		"call_handler":          "relay_script",
		"call_relay_script_url": "https://example.com/swml",
	})
	body := mock.Calls()[0].Body
	if body["call_handler"] != "relay_script" {
		t.Errorf("call_handler = %v", body["call_handler"])
	}
	if body["call_relay_script_url"] != "https://example.com/swml" {
		t.Errorf("call_relay_script_url = %v", body["call_relay_script_url"])
	}
}

// TestBindingRegression_EnumConstantMatchesWireValue pins that using the
// typed constant produces the same wire value as the raw string.
func TestBindingRegression_EnumConstantMatchesWireValue(t *testing.T) {
	pn, mock := newPhoneNumbers()
	_, _ = pn.Update("pn-1", map[string]any{
		"call_handler":          string(PhoneCallHandlerRelayScript),
		"call_relay_script_url": "https://example.com/swml",
	})
	body := mock.Calls()[0].Body
	if body["call_handler"] != "relay_script" {
		t.Errorf("call_handler = %v, want relay_script (from PhoneCallHandlerRelayScript)",
			body["call_handler"])
	}
}

// TestHelperCoverage pins that every expected helper exists on the
// PhoneNumbersNamespace.
func TestHelperCoverage(t *testing.T) {
	pn, _ := newPhoneNumbers()
	// These are method-value lookups; they fail to compile if any helper
	// is renamed or removed.
	_ = pn.SetSwmlWebhook
	_ = pn.SetCxmlWebhook
	_ = pn.SetCxmlApplication
	_ = pn.SetAiAgent
	_ = pn.SetCallFlow
	_ = pn.SetRelayApplication
	_ = pn.SetRelayTopic
}
