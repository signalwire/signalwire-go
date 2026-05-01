// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_logs_mock.py.
//
// The Logs namespace fans out across four spec docs (message/voice/fax/logs)
// because each kind of log lives at a different sub-API.

package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// ---------- Message Logs ----------

func TestMessageLogs_List_ReturnsDict(t *testing.T) {
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
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/messaging/logs" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "message.list_message_logs" {
		got := "<nil>"
		if j.MatchedRoute != nil {
			got = *j.MatchedRoute
		}
		t.Errorf("matched_route = %q, want message.list_message_logs", got)
	}
}

func TestMessageLogs_Get_UsesIDInPath(t *testing.T) {
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
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/messaging/logs/ml-42" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil — spec gap: message log retrieve")
	}
}

// ---------- Voice Logs ----------

func TestVoiceLogs_List_ReturnsDict(t *testing.T) {
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
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/voice/logs" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "voice.list_voice_logs" {
		got := "<nil>"
		if j.MatchedRoute != nil {
			got = *j.MatchedRoute
		}
		t.Errorf("matched_route = %q, want voice.list_voice_logs", got)
	}
}

func TestVoiceLogs_Get_UsesIDInPath(t *testing.T) {
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
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/voice/logs/vl-99" {
		t.Errorf("path = %q", j.Path)
	}
}

// ---------- Fax Logs ----------

func TestFaxLogs_List_ReturnsDict(t *testing.T) {
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
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/fax/logs" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "fax.list_fax_logs" {
		got := "<nil>"
		if j.MatchedRoute != nil {
			got = *j.MatchedRoute
		}
		t.Errorf("matched_route = %q, want fax.list_fax_logs", got)
	}
}

func TestFaxLogs_Get_UsesIDInPath(t *testing.T) {
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
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/fax/logs/fl-7" {
		t.Errorf("path = %q", j.Path)
	}
}

// ---------- Conference Logs ----------

func TestConferenceLogs_List_ReturnsDict(t *testing.T) {
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
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/logs/conferences" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "logs.list_conferences" {
		got := "<nil>"
		if j.MatchedRoute != nil {
			got = *j.MatchedRoute
		}
		t.Errorf("matched_route = %q, want logs.list_conferences", got)
	}
}

var _ = mocktest.JournalEntry{}
