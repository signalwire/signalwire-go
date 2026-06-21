// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Full success + error coverage for the small canonical spec groups
// (14 endpoints). Each route gets a success test (asserting response body,
// wire method/path, and matched_route == endpoint_id) and an error test
// (asserting *rest.SignalWireRestError StatusCode + journal response_status
// + matched_route).
//
// Routes covered:
//   project.create_token         POST   /api/project/tokens
//   project.update_token         PATCH  /api/project/tokens/{token_id}
//   project.delete_token         DELETE /api/project/tokens/{token_id}
//   voice.list_voice_logs        GET    /api/voice/logs
//   voice.get_voice_log          GET    /api/voice/logs/{id}
//   voice.list_voice_log_events  GET    /api/voice/logs/{id}/events
//   fax.list_fax_logs            GET    /api/fax/logs
//   fax.get_fax_log              GET    /api/fax/logs/{id}
//   message.list_message_logs    GET    /api/messaging/logs
//   message.get_message_log      GET    /api/messaging/logs/{id}
//   logs.list_conferences        GET    /api/logs/conferences
//   calling.call-commands        POST   /api/calling/calls
//   chat.create_chat_token       POST   /api/chat/tokens
//   pubsub.create_token          POST   /api/pubsub/tokens

package namespaces_test

import (
	"errors"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest"
	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// ---------- project.create_token ----------

func TestSmallSpec_ProjectCreateToken_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Project.Tokens.Create(map[string]any{"name": "tok-1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	if j.Path != "/api/project/tokens" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "project.create_token" {
		t.Errorf("matched_route = %v, want project.create_token", j.MatchedRoute)
	}
	reqBody, ok := j.BodyMap()
	if !ok || reqBody["name"] != "tok-1" {
		t.Errorf("request body name = %v", reqBody["name"])
	}
}

func TestSmallSpec_ProjectCreateToken_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "project.create_token", 422, map[string]any{"error": "invalid"})
	_, err := client.Project.Tokens.Create(map[string]any{})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 422 {
		t.Errorf("status = %d, want 422", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "project.create_token" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 422 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- project.update_token (PATCH) ----------

func TestSmallSpec_ProjectUpdateToken_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Project.Tokens.Update("tok-7", map[string]any{"name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "PATCH" {
		t.Errorf("method = %q, want PATCH", j.Method)
	}
	if j.Path != "/api/project/tokens/tok-7" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "project.update_token" {
		t.Errorf("matched_route = %v, want project.update_token", j.MatchedRoute)
	}
	reqBody, ok := j.BodyMap()
	if !ok || reqBody["name"] != "renamed" {
		t.Errorf("request body name = %v", reqBody["name"])
	}
}

func TestSmallSpec_ProjectUpdateToken_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "project.update_token", 404, map[string]any{"error": "not found"})
	_, err := client.Project.Tokens.Update("missing", map[string]any{"name": "x"})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "project.update_token" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- project.delete_token ----------

func TestSmallSpec_ProjectDeleteToken_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Project.Tokens.Delete("tok-7")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Error("expected map (204 normalized to {}), got nil")
	}
	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q, want DELETE", j.Method)
	}
	if j.Path != "/api/project/tokens/tok-7" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "project.delete_token" {
		t.Errorf("matched_route = %v, want project.delete_token", j.MatchedRoute)
	}
}

func TestSmallSpec_ProjectDeleteToken_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "project.delete_token", 404, map[string]any{"error": "not found"})
	_, err := client.Project.Tokens.Delete("missing")
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "project.delete_token" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- voice.list_voice_logs ----------

func TestSmallSpec_VoiceListLogs_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Logs.Voice.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	if j.Path != "/api/voice/logs" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "voice.list_voice_logs" {
		t.Errorf("matched_route = %v, want voice.list_voice_logs", j.MatchedRoute)
	}
}

func TestSmallSpec_VoiceListLogs_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "voice.list_voice_logs", 500, map[string]any{"error": "boom"})
	_, err := client.Logs.Voice.List(nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d, want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "voice.list_voice_logs" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- voice.get_voice_log ----------

func TestSmallSpec_VoiceGetLog_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Logs.Voice.Get("vl-99")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	if j.Path != "/api/voice/logs/vl-99" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "voice.get_voice_log" {
		t.Errorf("matched_route = %v, want voice.get_voice_log", j.MatchedRoute)
	}
}

func TestSmallSpec_VoiceGetLog_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "voice.get_voice_log", 404, map[string]any{"error": "not found"})
	_, err := client.Logs.Voice.Get("missing")
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "voice.get_voice_log" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- voice.list_voice_log_events ----------

func TestSmallSpec_VoiceListLogEvents_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Logs.Voice.ListEvents("vl-99", nil)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	if j.Path != "/api/voice/logs/vl-99/events" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "voice.list_voice_log_events" {
		t.Errorf("matched_route = %v, want voice.list_voice_log_events", j.MatchedRoute)
	}
}

func TestSmallSpec_VoiceListLogEvents_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "voice.list_voice_log_events", 404, map[string]any{"error": "no log"})
	_, err := client.Logs.Voice.ListEvents("missing", nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "voice.list_voice_log_events" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- fax.list_fax_logs ----------

func TestSmallSpec_FaxListLogs_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Logs.Fax.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	if j.Path != "/api/fax/logs" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "fax.list_fax_logs" {
		t.Errorf("matched_route = %v, want fax.list_fax_logs", j.MatchedRoute)
	}
}

func TestSmallSpec_FaxListLogs_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fax.list_fax_logs", 500, map[string]any{"error": "boom"})
	_, err := client.Logs.Fax.List(nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d, want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "fax.list_fax_logs" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- fax.get_fax_log ----------

func TestSmallSpec_FaxGetLog_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Logs.Fax.Get("fl-7")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	if j.Path != "/api/fax/logs/fl-7" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "fax.get_fax_log" {
		t.Errorf("matched_route = %v, want fax.get_fax_log", j.MatchedRoute)
	}
}

func TestSmallSpec_FaxGetLog_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fax.get_fax_log", 404, map[string]any{"error": "not found"})
	_, err := client.Logs.Fax.Get("missing")
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "fax.get_fax_log" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- message.list_message_logs ----------

func TestSmallSpec_MessageListLogs_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Logs.Messages.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	if j.Path != "/api/messaging/logs" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "message.list_message_logs" {
		t.Errorf("matched_route = %v, want message.list_message_logs", j.MatchedRoute)
	}
}

func TestSmallSpec_MessageListLogs_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "message.list_message_logs", 500, map[string]any{"error": "boom"})
	_, err := client.Logs.Messages.List(nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d, want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "message.list_message_logs" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- message.get_message_log ----------

func TestSmallSpec_MessageGetLog_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Logs.Messages.Get("ml-42")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	if j.Path != "/api/messaging/logs/ml-42" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "message.get_message_log" {
		t.Errorf("matched_route = %v, want message.get_message_log", j.MatchedRoute)
	}
}

func TestSmallSpec_MessageGetLog_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "message.get_message_log", 404, map[string]any{"error": "not found"})
	_, err := client.Logs.Messages.Get("missing")
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "message.get_message_log" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- logs.list_conferences ----------

func TestSmallSpec_LogsListConferences_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Logs.Conferences.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	if j.Path != "/api/logs/conferences" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "logs.list_conferences" {
		t.Errorf("matched_route = %v, want logs.list_conferences", j.MatchedRoute)
	}
}

func TestSmallSpec_LogsListConferences_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "logs.list_conferences", 500, map[string]any{"error": "boom"})
	_, err := client.Logs.Conferences.List(nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d, want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "logs.list_conferences" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- calling.call-commands ----------

func TestSmallSpec_CallingDial_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Calling.Dial(map[string]any{
		"url": "https://example.com/swml",
		"to":  "+15551234567",
	})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	if _, ok := body["id"]; !ok {
		t.Errorf("response missing 'id', got keys %v", keys(body))
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	if j.Path != "/api/calling/calls" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "calling.call-commands" {
		t.Errorf("matched_route = %v, want calling.call-commands", j.MatchedRoute)
	}
	reqBody, ok := j.BodyMap()
	if !ok || reqBody["command"] != "dial" {
		t.Errorf("request body command = %v, want dial", reqBody["command"])
	}
}

func TestSmallSpec_CallingDial_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "calling.call-commands", 422, map[string]any{"error": "invalid"})
	_, err := client.Calling.Dial(map[string]any{"command": "dial"})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 422 {
		t.Errorf("status = %d, want 422", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "calling.call-commands" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 422 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- chat.create_chat_token ----------

func TestSmallSpec_ChatCreateToken_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Chat.CreateToken(map[string]any{
		"channels": map[string]any{"room": map[string]any{"read": true}},
	})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	if j.Path != "/api/chat/tokens" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "chat.create_chat_token" {
		t.Errorf("matched_route = %v, want chat.create_chat_token", j.MatchedRoute)
	}
}

func TestSmallSpec_ChatCreateToken_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "chat.create_chat_token", 422, map[string]any{"error": "invalid"})
	_, err := client.Chat.CreateToken(map[string]any{})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 422 {
		t.Errorf("status = %d, want 422", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "chat.create_chat_token" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 422 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- pubsub.create_token ----------

func TestSmallSpec_PubSubCreateToken_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.PubSub.CreateToken(map[string]any{
		"channels": map[string]any{"updates": map[string]any{"read": true}},
	})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	if j.Path != "/api/pubsub/tokens" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "pubsub.create_token" {
		t.Errorf("matched_route = %v, want pubsub.create_token", j.MatchedRoute)
	}
}

func TestSmallSpec_PubSubCreateToken_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "pubsub.create_token", 422, map[string]any{"error": "invalid"})
	_, err := client.PubSub.CreateToken(map[string]any{})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 422 {
		t.Errorf("status = %d, want 422", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "pubsub.create_token" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 422 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}
