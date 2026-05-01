// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_video_mock.py.
//
// Exercises the Video API surface that wasn't reached by the legacy
// namespaces_test.go cases: room sessions, room recordings, conference
// tokens, conference streams, and individual stream lifecycle.

package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// ---------- Rooms — streams sub-resource ----------

func TestVideoRooms_ListStreams_ReturnsDataCollection(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.Rooms.ListStreams("room-1", nil)
	if err != nil {
		t.Fatalf("ListStreams: %v", err)
	}
	data, ok := body["data"]
	if !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	if _, isList := data.([]any); !isList {
		t.Errorf("data type = %T", data)
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/rooms/room-1/streams" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil — spec gap: rooms streams list")
	}
}

func TestVideoRooms_CreateStream_PostsKwargsInBody(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.Rooms.CreateStream("room-1", map[string]any{
		"url": "rtmp://example.com/live",
	})
	if err != nil {
		t.Fatalf("CreateStream: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/rooms/room-1/streams" {
		t.Errorf("path = %q", j.Path)
	}
	sent, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if sent["url"] != "rtmp://example.com/live" {
		t.Errorf("url = %v", sent["url"])
	}
}

// ---------- Room Sessions ----------

func TestVideoRoomSessions_List_ReturnsDataCollection(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.RoomSessions.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	data, ok := body["data"]
	if !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	if _, isList := data.([]any); !isList {
		t.Errorf("data type = %T", data)
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_sessions" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestVideoRoomSessions_Get_ReturnsSessionObject(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.RoomSessions.Get("sess-abc")
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
	if j.Path != "/api/video/room_sessions/sess-abc" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil")
	}
}

func TestVideoRoomSessions_ListEvents_UsesEventsSubpath(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.RoomSessions.ListEvents("sess-1", nil)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	data, ok := body["data"]
	if !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	if _, isList := data.([]any); !isList {
		t.Errorf("data type = %T", data)
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_sessions/sess-1/events" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestVideoRoomSessions_ListRecordings_UsesRecordingsSubpath(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.RoomSessions.ListRecordings("sess-2", nil)
	if err != nil {
		t.Fatalf("ListRecordings: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data' in %v", keys(body))
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_sessions/sess-2/recordings" {
		t.Errorf("path = %q", j.Path)
	}
}

// ---------- Room Recordings ----------

func TestVideoRoomRecordings_List_ReturnsDataCollection(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.RoomRecordings.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	data, ok := body["data"]
	if !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	if _, isList := data.([]any); !isList {
		t.Errorf("data type = %T", data)
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_recordings" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestVideoRoomRecordings_Get_ReturnsSingleRecording(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.RoomRecordings.Get("rec-xyz")
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
	if j.Path != "/api/video/room_recordings/rec-xyz" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestVideoRoomRecordings_Delete_ReturnsEmptyDictFor204(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.RoomRecordings.Delete("rec-del")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_recordings/rec-del" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil")
	}
}

func TestVideoRoomRecordings_ListEvents_UsesEventsSubpath(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.RoomRecordings.ListEvents("rec-1", nil)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data' in %v", keys(body))
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_recordings/rec-1/events" {
		t.Errorf("path = %q", j.Path)
	}
}

// ---------- Conferences — sub-collections ----------

func TestVideoConferences_ListConferenceTokens(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.Conferences.ListConferenceTokens("conf-1", nil)
	if err != nil {
		t.Fatalf("ListConferenceTokens: %v", err)
	}
	data, ok := body["data"]
	if !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	if _, isList := data.([]any); !isList {
		t.Errorf("data type = %T", data)
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/conferences/conf-1/conference_tokens" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestVideoConferences_ListStreams(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.Conferences.ListStreams("conf-2", nil)
	if err != nil {
		t.Fatalf("ListStreams: %v", err)
	}
	data, ok := body["data"]
	if !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	if _, isList := data.([]any); !isList {
		t.Errorf("data type = %T", data)
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/conferences/conf-2/streams" {
		t.Errorf("path = %q", j.Path)
	}
}

// ---------- Conference Tokens (top-level) ----------

func TestVideoConferenceTokens_Get_ReturnsSingleToken(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.ConferenceTokens.Get("tok-1")
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
	if j.Path != "/api/video/conference_tokens/tok-1" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil")
	}
}

func TestVideoConferenceTokens_Reset_PostsToResetSubpath(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.ConferenceTokens.Reset("tok-2")
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/conference_tokens/tok-2/reset" {
		t.Errorf("path = %q", j.Path)
	}
	// reset is a no-body POST — body should be nil or an empty/empty map.
	switch v := j.Body.(type) {
	case nil:
		// ok
	case map[string]any:
		if len(v) != 0 {
			t.Errorf("expected empty body, got %v", v)
		}
	case string:
		if v != "" {
			t.Errorf("expected empty body string, got %q", v)
		}
	default:
		t.Errorf("unexpected body type %T", v)
	}
}

// ---------- Streams (top-level) ----------

func TestVideoStreams_Get_ReturnsStream(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.Streams.Get("stream-1")
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
	if j.Path != "/api/video/streams/stream-1" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestVideoStreams_Update_UsesPutWithKwargs(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.Streams.Update("stream-2", map[string]any{
		"url": "rtmp://example.com/new",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "PUT" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/streams/stream-2" {
		t.Errorf("path = %q", j.Path)
	}
	sent, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if sent["url"] != "rtmp://example.com/new" {
		t.Errorf("url = %v", sent["url"])
	}
}

func TestVideoStreams_Delete(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Video.Streams.Delete("stream-3")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/streams/stream-3" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil")
	}
}

var _ = mocktest.JournalEntry{}
