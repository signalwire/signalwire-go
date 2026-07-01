// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Full success+error REST coverage for the video spec group, mirroring the
// proven python/java/ts suites. Every coverable canonical video.* route gets a
// SUCCESS test (asserts response + journal Method/Path/MatchedRoute) and an
// ERROR test (PushScenario 4xx/5xx -> *rest.SignalWireRestError + journal
// ResponseStatus/MatchedRoute).
//
// Gaps (not faked, same as python/java/ts):
//   - video.list_logs / video.get_log: no logs accessor in the Go SDK
//     (VideoNamespace has no Logs field), matching python/java/ts.
//   - video.get_room (GET /rooms/{id}) is wire-identical to
//     video.get_room_by_name (GET /rooms/{name}); the mock always resolves
//     GET /rooms/X to get_room_by_name, so get_room is unhittable. We cover
//     get_room_by_name (via Rooms.Get) instead.

package namespaces_test

import (
	"errors"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest"
	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// matchedRoute returns the dereferenced matched route or "<nil>".
func videoCovRoute(mr *string) string {
	if mr == nil {
		return "<nil>"
	}
	return *mr
}

// ---------------- conference_tokens ----------------

func TestVideoCov_GetConferenceToken(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.ConferenceTokens.Get("ct-1", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/conference_tokens/ct-1" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.get_conference_token" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_GetConferenceToken_NotFound(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.get_conference_token", 404, map[string]any{"error": "not found"})
	_, err := client.Video.ConferenceTokens.Get("missing", nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.get_conference_token" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_ResetConferenceToken(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.ConferenceTokens.Reset("ct-2")
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/conference_tokens/ct-2/reset" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.reset_conference_token" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_ResetConferenceToken_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.reset_conference_token", 422, map[string]any{"error": "x"})
	_, err := client.Video.ConferenceTokens.Reset("ct-2")
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 422 {
		t.Errorf("status = %d want 422", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.reset_conference_token" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 422 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------------- conferences ----------------

func TestVideoCov_CreateConference(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Conferences.Create(map[string]any{"name": "conf-a"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/conferences" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.create_video_conference" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	sent, ok := j.BodyMap()
	if !ok || sent["name"] != "conf-a" {
		t.Errorf("body = %v", j.Body)
	}
}

func TestVideoCov_CreateConference_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.create_video_conference", 422, map[string]any{"error": "x"})
	_, err := client.Video.Conferences.Create(map[string]any{"name": "bad"})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 422 {
		t.Errorf("status = %d want 422", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.create_video_conference" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 422 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_ListConferences(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Conferences.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/conferences" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.list_video_conferences" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_ListConferences_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.list_video_conferences", 500, map[string]any{"error": "x"})
	_, err := client.Video.Conferences.List(nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.list_video_conferences" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_GetConference(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Conferences.Get("conf-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/conferences/conf-1" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.get_video_conference" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_GetConference_NotFound(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.get_video_conference", 404, map[string]any{"error": "not found"})
	_, err := client.Video.Conferences.Get("missing")
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.get_video_conference" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_UpdateConference(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Conferences.Update("conf-1", map[string]any{"name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "PUT" {
		t.Errorf("method = %q want PUT", j.Method)
	}
	if j.Path != "/api/video/conferences/conf-1" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.update_video_conference" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	sent, ok := j.BodyMap()
	if !ok || sent["name"] != "renamed" {
		t.Errorf("body = %v", j.Body)
	}
}

func TestVideoCov_UpdateConference_NotFound(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.update_video_conference", 404, map[string]any{"error": "not found"})
	_, err := client.Video.Conferences.Update("missing", map[string]any{"name": "x"})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.update_video_conference" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_DeleteConference(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Conferences.Delete("conf-1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/conferences/conf-1" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.delete_video_conference" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_DeleteConference_NotFound(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.delete_video_conference", 404, map[string]any{"error": "not found"})
	_, err := client.Video.Conferences.Delete("missing")
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.delete_video_conference" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_ListConferenceTokens(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Conferences.ListConferenceTokens("conf-1", nil)
	if err != nil {
		t.Fatalf("ListConferenceTokens: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/conferences/conf-1/conference_tokens" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.list_conference_tokens" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_ListConferenceTokens_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.list_conference_tokens", 500, map[string]any{"error": "x"})
	_, err := client.Video.Conferences.ListConferenceTokens("conf-1", nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.list_conference_tokens" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_ListConferenceStreams(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Conferences.ListStreams("conf-1", nil)
	if err != nil {
		t.Fatalf("ListStreams: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/conferences/conf-1/streams" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.list_conference_streams" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_ListConferenceStreams_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.list_conference_streams", 500, map[string]any{"error": "x"})
	_, err := client.Video.Conferences.ListStreams("conf-1", nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.list_conference_streams" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_CreateConferenceStream(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Conferences.CreateStream("conf-1", map[string]any{
		"url": "rtmp://example.com/live",
	})
	if err != nil {
		t.Fatalf("CreateStream: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/conferences/conf-1/streams" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.create_conference_stream" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	sent, ok := j.BodyMap()
	if !ok || sent["url"] != "rtmp://example.com/live" {
		t.Errorf("body = %v", j.Body)
	}
}

func TestVideoCov_CreateConferenceStream_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.create_conference_stream", 422, map[string]any{"error": "x"})
	_, err := client.Video.Conferences.CreateStream("conf-1", map[string]any{"url": "bad"})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 422 {
		t.Errorf("status = %d want 422", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.create_conference_stream" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 422 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------------- room_recordings ----------------

func TestVideoCov_ListRoomRecordings(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.RoomRecordings.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_recordings" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.list_room_recordings" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_ListRoomRecordings_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.list_room_recordings", 500, map[string]any{"error": "x"})
	_, err := client.Video.RoomRecordings.List(nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.list_room_recordings" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_GetRoomRecording(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.RoomRecordings.Get("rec-1", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_recordings/rec-1" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.get_room_recording" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_GetRoomRecording_NotFound(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.get_room_recording", 404, map[string]any{"error": "not found"})
	_, err := client.Video.RoomRecordings.Get("missing", nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.get_room_recording" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_DeleteRoomRecording(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.RoomRecordings.Delete("rec-1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_recordings/rec-1" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.delete_room_recording" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_DeleteRoomRecording_NotFound(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.delete_room_recording", 404, map[string]any{"error": "not found"})
	_, err := client.Video.RoomRecordings.Delete("missing")
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.delete_room_recording" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_ListRoomRecordingEvents(t *testing.T) {
	t.Parallel()
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
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_recordings/rec-1/events" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.list_room_recording_events" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_ListRoomRecordingEvents_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.list_room_recording_events", 500, map[string]any{"error": "x"})
	_, err := client.Video.RoomRecordings.ListEvents("rec-1", nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.list_room_recording_events" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------------- room_sessions ----------------

func TestVideoCov_ListRoomSessions(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.RoomSessions.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_sessions" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.list_room_sessions" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_ListRoomSessions_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.list_room_sessions", 500, map[string]any{"error": "x"})
	_, err := client.Video.RoomSessions.List(nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.list_room_sessions" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_GetRoomSession(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.RoomSessions.Get("sess-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_sessions/sess-1" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.get_room_session" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_GetRoomSession_NotFound(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.get_room_session", 404, map[string]any{"error": "not found"})
	_, err := client.Video.RoomSessions.Get("missing")
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.get_room_session" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_ListRoomSessionEvents(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.RoomSessions.ListEvents("sess-1", nil)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_sessions/sess-1/events" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.list_room_session_events" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_ListRoomSessionEvents_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.list_room_session_events", 500, map[string]any{"error": "x"})
	_, err := client.Video.RoomSessions.ListEvents("sess-1", nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.list_room_session_events" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_ListRoomSessionMembers(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.RoomSessions.ListMembers("sess-1", nil)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_sessions/sess-1/members" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.list_room_session_members" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_ListRoomSessionMembers_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.list_room_session_members", 500, map[string]any{"error": "x"})
	_, err := client.Video.RoomSessions.ListMembers("sess-1", nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.list_room_session_members" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_ListRoomSessionRecordings(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.RoomSessions.ListRecordings("sess-1", nil)
	if err != nil {
		t.Fatalf("ListRecordings: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_sessions/sess-1/recordings" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.list_room_session_recordings" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_ListRoomSessionRecordings_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.list_room_session_recordings", 500, map[string]any{"error": "x"})
	_, err := client.Video.RoomSessions.ListRecordings("sess-1", nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.list_room_session_recordings" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------------- room_tokens ----------------

func TestVideoCov_CreateRoomToken(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.RoomTokens.Create(map[string]any{"room_name": "demo"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/room_tokens" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.create_room_token" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	sent, ok := j.BodyMap()
	if !ok || sent["room_name"] != "demo" {
		t.Errorf("body = %v", j.Body)
	}
}

func TestVideoCov_CreateRoomToken_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.create_room_token", 422, map[string]any{"error": "x"})
	_, err := client.Video.RoomTokens.Create(map[string]any{"room_name": "bad"})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 422 {
		t.Errorf("status = %d want 422", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.create_room_token" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 422 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------------- rooms ----------------

func TestVideoCov_CreateRoom(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Rooms.Create(map[string]any{"name": "room-a"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/rooms" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.create_room" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	sent, ok := j.BodyMap()
	if !ok || sent["name"] != "room-a" {
		t.Errorf("body = %v", j.Body)
	}
}

func TestVideoCov_CreateRoom_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.create_room", 422, map[string]any{"error": "x"})
	_, err := client.Video.Rooms.Create(map[string]any{"name": "bad"})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 422 {
		t.Errorf("status = %d want 422", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.create_room" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 422 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_ListRooms(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Rooms.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/rooms" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.list_rooms" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_ListRooms_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.list_rooms", 500, map[string]any{"error": "x"})
	_, err := client.Video.Rooms.List(nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.list_rooms" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// GetRoomByName: GET /rooms/{id}. The mock resolves GET /rooms/X to
// video.get_room_by_name (get_room is the routing-collision gap).
func TestVideoCov_GetRoomByName(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Rooms.Get("my-room")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/rooms/my-room" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.get_room_by_name" {
		t.Errorf("matched_route = %q want video.get_room_by_name", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_GetRoomByName_NotFound(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.get_room_by_name", 404, map[string]any{"error": "not found"})
	_, err := client.Video.Rooms.Get("missing")
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.get_room_by_name" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_UpdateRoom(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Rooms.Update("room-1", map[string]any{"display_name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "PUT" {
		t.Errorf("method = %q want PUT", j.Method)
	}
	if j.Path != "/api/video/rooms/room-1" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.update_room" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	sent, ok := j.BodyMap()
	if !ok || sent["display_name"] != "renamed" {
		t.Errorf("body = %v", j.Body)
	}
}

func TestVideoCov_UpdateRoom_NotFound(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.update_room", 404, map[string]any{"error": "not found"})
	_, err := client.Video.Rooms.Update("missing", map[string]any{"display_name": "x"})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.update_room" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_DeleteRoom(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Rooms.Delete("room-1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/rooms/room-1" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.delete_room" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_DeleteRoom_NotFound(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.delete_room", 404, map[string]any{"error": "not found"})
	_, err := client.Video.Rooms.Delete("missing")
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.delete_room" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_ListRoomStreams(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Rooms.ListStreams("room-1", nil)
	if err != nil {
		t.Fatalf("ListStreams: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data' in %v", keys(body))
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/rooms/room-1/streams" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.list_room_streams" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_ListRoomStreams_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.list_room_streams", 500, map[string]any{"error": "x"})
	_, err := client.Video.Rooms.ListStreams("room-1", nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.list_room_streams" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_CreateRoomStream(t *testing.T) {
	t.Parallel()
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
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/rooms/room-1/streams" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.create_room_stream" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	sent, ok := j.BodyMap()
	if !ok || sent["url"] != "rtmp://example.com/live" {
		t.Errorf("body = %v", j.Body)
	}
}

func TestVideoCov_CreateRoomStream_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.create_room_stream", 422, map[string]any{"error": "x"})
	_, err := client.Video.Rooms.CreateStream("room-1", map[string]any{"url": "bad"})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 422 {
		t.Errorf("status = %d want 422", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.create_room_stream" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 422 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------------- streams ----------------

func TestVideoCov_GetStream(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Streams.Get("stream-1", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/streams/stream-1" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.get_stream" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_GetStream_NotFound(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.get_stream", 404, map[string]any{"error": "not found"})
	_, err := client.Video.Streams.Get("missing", nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.get_stream" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_UpdateStream(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Streams.Update("stream-1", map[string]any{"url": "rtmp://example.com/new"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "PUT" {
		t.Errorf("method = %q want PUT", j.Method)
	}
	if j.Path != "/api/video/streams/stream-1" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.update_stream" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	sent, ok := j.BodyMap()
	if !ok || sent["url"] != "rtmp://example.com/new" {
		t.Errorf("body = %v", j.Body)
	}
}

func TestVideoCov_UpdateStream_NotFound(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.update_stream", 404, map[string]any{"error": "not found"})
	_, err := client.Video.Streams.Update("missing", map[string]any{"url": "x"})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.update_stream" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

func TestVideoCov_DeleteStream(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Video.Streams.Delete("stream-1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/video/streams/stream-1" {
		t.Errorf("path = %q", j.Path)
	}
	if videoCovRoute(j.MatchedRoute) != "video.delete_stream" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
}

func TestVideoCov_DeleteStream_NotFound(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "video.delete_stream", 404, map[string]any{"error": "not found"})
	_, err := client.Video.Streams.Delete("missing")
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if videoCovRoute(j.MatchedRoute) != "video.delete_stream" {
		t.Errorf("matched_route = %q", videoCovRoute(j.MatchedRoute))
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}
