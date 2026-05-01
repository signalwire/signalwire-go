// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_compat_conferences.py.
//
// Drives client.Compat.Conferences.* against the mock_signalwire HTTP server.
// Covers: Conference itself (List/Get/Update), Participants (Get/Update/Remove),
// Recordings (List/Get/Update/Delete), Streams (Start/Stop).

package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

const compatConfBase = "/api/laml/2010-04-01/Accounts/test_proj/Conferences"

// ---------- Conference itself ----------

func TestCompatConferences_List_ReturnsPaginatedList(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	confs, ok := result["conferences"]
	if !ok {
		t.Fatalf("expected 'conferences' key, got %v", keys(result))
	}
	if _, isList := confs.([]any); !isList {
		t.Errorf("conferences type = %T, want []any", confs)
	}
	if _, ok := result["page"].(float64); !ok {
		// JSON numbers decode to float64 in Go. Some responses may return int.
		if _, ok := result["page"]; !ok {
			t.Errorf("expected 'page' key, got %v", keys(result))
		}
	}
}

func TestCompatConferences_List_JournalRecordsGet(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Conferences.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	if j.Path != compatConfBase {
		t.Errorf("path = %q, want %q", j.Path, compatConfBase)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil — spec gap: conferences.list")
	}
}

func TestCompatConferences_Get_ReturnsConferenceResource(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.Get("CF_TEST")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	_, hasFn := result["friendly_name"]
	_, hasStatus := result["status"]
	if !hasFn && !hasStatus {
		t.Errorf("expected friendly_name or status, got %v", keys(result))
	}
}

func TestCompatConferences_Get_JournalRecordsGetWithSid(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Conferences.Get("CF_GETSID")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	const wantPath = compatConfBase + "/CF_GETSID"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

func TestCompatConferences_Update_ReturnsUpdatedConference(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.Update("CF_X", map[string]any{
		"Status": "completed",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	_, hasFn := result["friendly_name"]
	_, hasStatus := result["status"]
	if !hasFn && !hasStatus {
		t.Errorf("expected friendly_name or status, got %v", keys(result))
	}
}

func TestCompatConferences_Update_JournalRecordsPostWithStatus(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Conferences.Update("CF_UPD", map[string]any{
		"Status":      "completed",
		"AnnounceUrl": "https://a.b",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	const wantPath = compatConfBase + "/CF_UPD"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if body["Status"] != "completed" {
		t.Errorf("Status = %v, want completed", body["Status"])
	}
	if body["AnnounceUrl"] != "https://a.b" {
		t.Errorf("AnnounceUrl = %v", body["AnnounceUrl"])
	}
}

// ---------- Participants ----------

func TestCompatConferences_GetParticipant_ReturnsParticipant(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.GetParticipant("CF_P", "CA_P")
	if err != nil {
		t.Fatalf("GetParticipant: %v", err)
	}
	_, hasCallSid := result["call_sid"]
	_, hasConfSid := result["conference_sid"]
	if !hasCallSid && !hasConfSid {
		t.Errorf("expected call_sid or conference_sid, got %v", keys(result))
	}
}

func TestCompatConferences_GetParticipant_JournalRecordsGet(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Conferences.GetParticipant("CF_GP", "CA_GP")
	if err != nil {
		t.Fatalf("GetParticipant: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	const wantPath = compatConfBase + "/CF_GP/Participants/CA_GP"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

func TestCompatConferences_UpdateParticipant_ReturnsParticipant(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.UpdateParticipant("CF_UP", "CA_UP", map[string]any{
		"Muted": true,
	})
	if err != nil {
		t.Fatalf("UpdateParticipant: %v", err)
	}
	_, hasCallSid := result["call_sid"]
	_, hasConfSid := result["conference_sid"]
	if !hasCallSid && !hasConfSid {
		t.Errorf("expected call_sid or conference_sid, got %v", keys(result))
	}
}

func TestCompatConferences_UpdateParticipant_JournalRecordsPostWithMute(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Conferences.UpdateParticipant("CF_M", "CA_M", map[string]any{
		"Muted": true,
		"Hold":  false,
	})
	if err != nil {
		t.Fatalf("UpdateParticipant: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	const wantPath = compatConfBase + "/CF_M/Participants/CA_M"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if body["Muted"] != true {
		t.Errorf("Muted = %v, want true", body["Muted"])
	}
	if body["Hold"] != false {
		t.Errorf("Hold = %v, want false", body["Hold"])
	}
}

func TestCompatConferences_RemoveParticipant_ReturnsObject(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.RemoveParticipant("CF_R", "CA_R")
	if err != nil {
		t.Fatalf("RemoveParticipant: %v", err)
	}
	if result == nil {
		t.Error("expected map, got nil")
	}
}

func TestCompatConferences_RemoveParticipant_JournalRecordsDelete(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Conferences.RemoveParticipant("CF_RM", "CA_RM")
	if err != nil {
		t.Fatalf("RemoveParticipant: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q, want DELETE", j.Method)
	}
	const wantPath = compatConfBase + "/CF_RM/Participants/CA_RM"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

// ---------- Recordings ----------

func TestCompatConferences_ListRecordings_ReturnsPaginated(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.ListRecordings("CF_LR", nil)
	if err != nil {
		t.Fatalf("ListRecordings: %v", err)
	}
	recs, ok := result["recordings"]
	if !ok {
		t.Fatalf("expected 'recordings' key, got %v", keys(result))
	}
	if _, isList := recs.([]any); !isList {
		t.Errorf("recordings type = %T, want []any", recs)
	}
}

func TestCompatConferences_ListRecordings_JournalRecordsGet(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Conferences.ListRecordings("CF_LRX", nil)
	if err != nil {
		t.Fatalf("ListRecordings: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	const wantPath = compatConfBase + "/CF_LRX/Recordings"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

func TestCompatConferences_GetRecording_ReturnsRecording(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.GetRecording("CF_GR", "RE_GR")
	if err != nil {
		t.Fatalf("GetRecording: %v", err)
	}
	_, hasSid := result["sid"]
	_, hasCallSid := result["call_sid"]
	if !hasSid && !hasCallSid {
		t.Errorf("expected sid or call_sid, got %v", keys(result))
	}
}

func TestCompatConferences_GetRecording_JournalRecordsGet(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Conferences.GetRecording("CF_GRX", "RE_GRX")
	if err != nil {
		t.Fatalf("GetRecording: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	const wantPath = compatConfBase + "/CF_GRX/Recordings/RE_GRX"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

func TestCompatConferences_UpdateRecording_ReturnsRecording(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.UpdateRecording("CF_URC", "RE_URC", map[string]any{
		"Status": "paused",
	})
	if err != nil {
		t.Fatalf("UpdateRecording: %v", err)
	}
	_, hasSid := result["sid"]
	_, hasStatus := result["status"]
	if !hasSid && !hasStatus {
		t.Errorf("expected sid or status, got %v", keys(result))
	}
}

func TestCompatConferences_UpdateRecording_JournalRecordsPost(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Conferences.UpdateRecording("CF_UR", "RE_UR", map[string]any{
		"Status": "paused",
	})
	if err != nil {
		t.Fatalf("UpdateRecording: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	const wantPath = compatConfBase + "/CF_UR/Recordings/RE_UR"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if body["Status"] != "paused" {
		t.Errorf("Status = %v, want paused", body["Status"])
	}
}

func TestCompatConferences_DeleteRecording_NoException(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.DeleteRecording("CF_DR", "RE_DR")
	if err != nil {
		t.Fatalf("DeleteRecording: %v", err)
	}
	if result == nil {
		t.Error("expected map, got nil")
	}
}

func TestCompatConferences_DeleteRecording_JournalRecordsDelete(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Conferences.DeleteRecording("CF_DRX", "RE_DRX")
	if err != nil {
		t.Fatalf("DeleteRecording: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q, want DELETE", j.Method)
	}
	const wantPath = compatConfBase + "/CF_DRX/Recordings/RE_DRX"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

// ---------- Streams ----------

func TestCompatConferences_StartStream_ReturnsStream(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.StartStream("CF_SS", map[string]any{
		"Url": "wss://a.b/s",
	})
	if err != nil {
		t.Fatalf("StartStream: %v", err)
	}
	_, hasSid := result["sid"]
	_, hasName := result["name"]
	if !hasSid && !hasName {
		t.Errorf("expected sid or name, got %v", keys(result))
	}
}

func TestCompatConferences_StartStream_JournalRecordsPost(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Conferences.StartStream("CF_SSX", map[string]any{
		"Url":  "wss://a.b/s",
		"Name": "strm",
	})
	if err != nil {
		t.Fatalf("StartStream: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	const wantPath = compatConfBase + "/CF_SSX/Streams"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if body["Url"] != "wss://a.b/s" {
		t.Errorf("Url = %v", body["Url"])
	}
}

func TestCompatConferences_StopStream_ReturnsStream(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.StopStream("CF_TS", "ST_TS", map[string]any{
		"Status": "stopped",
	})
	if err != nil {
		t.Fatalf("StopStream: %v", err)
	}
	_, hasSid := result["sid"]
	_, hasStatus := result["status"]
	if !hasSid && !hasStatus {
		t.Errorf("expected sid or status, got %v", keys(result))
	}
}

func TestCompatConferences_StopStream_JournalRecordsPost(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Conferences.StopStream("CF_TSX", "ST_TSX", map[string]any{
		"Status": "stopped",
	})
	if err != nil {
		t.Fatalf("StopStream: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	const wantPath = compatConfBase + "/CF_TSX/Streams/ST_TSX"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if body["Status"] != "stopped" {
		t.Errorf("Status = %v, want stopped", body["Status"])
	}
}

// keep mock package referenced in case build refactors a test out
var _ = mocktest.JournalEntry{}
