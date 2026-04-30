// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_compat_calls_streams.py.
//
// Each Go subtest mirrors one Python test and asserts on both the SDK
// response shape and the wire request the mock journaled.

package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// TestCompatCalls_StartStream covers Calls.StartStream → POST /Calls/{sid}/Streams.
func TestCompatCalls_StartStream(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("returns_stream_resource", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.Calls.StartStream("CA_TEST", map[string]any{
			"Url":  "wss://example.com/stream",
			"Name": "my-stream",
		})
		if err != nil {
			t.Fatalf("StartStream: %v", err)
		}
		if result == nil {
			t.Fatal("StartStream returned nil result")
		}
		if _, hasSid := result["sid"]; !hasSid {
			if _, hasName := result["name"]; !hasName {
				t.Errorf("expected stream sid or name in body, got keys %v", keys(result))
			}
		}
	})

	t.Run("journal_records_post_to_streams_collection", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.Calls.StartStream("CA_JX1", map[string]any{
			"Url":  "wss://a.b/s",
			"Name": "strm-x",
		})
		if err != nil {
			t.Fatalf("StartStream: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "POST" {
			t.Errorf("method = %q, want POST", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/Calls/CA_JX1/Streams"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
		body, ok := j.BodyMap()
		if !ok {
			t.Fatalf("expected JSON body, got %T: %v", j.Body, j.Body)
		}
		if body["Url"] != "wss://a.b/s" {
			t.Errorf("body[Url] = %v, want wss://a.b/s", body["Url"])
		}
		if body["Name"] != "strm-x" {
			t.Errorf("body[Name] = %v, want strm-x", body["Name"])
		}
	})
}

// TestCompatCalls_StopStream covers Calls.StopStream → POST .../Streams/{stream_sid}.
func TestCompatCalls_StopStream(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("returns_stream_resource_with_status", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.Calls.StopStream("CA_T1", "ST_T1", map[string]any{
			"Status": "stopped",
		})
		if err != nil {
			t.Fatalf("StopStream: %v", err)
		}
		if result == nil {
			t.Fatal("StopStream returned nil result")
		}
		if _, hasSid := result["sid"]; !hasSid {
			if _, hasStatus := result["status"]; !hasStatus {
				t.Errorf("expected stream sid or status in body, got keys %v", keys(result))
			}
		}
	})

	t.Run("journal_records_post_to_specific_stream", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.Calls.StopStream("CA_S1", "ST_S1", map[string]any{
			"Status": "stopped",
		})
		if err != nil {
			t.Fatalf("StopStream: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "POST" {
			t.Errorf("method = %q, want POST", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/Calls/CA_S1/Streams/ST_S1"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
		body, ok := j.BodyMap()
		if !ok {
			t.Fatalf("expected JSON body, got %T: %v", j.Body, j.Body)
		}
		if body["Status"] != "stopped" {
			t.Errorf("body[Status] = %v, want stopped", body["Status"])
		}
	})
}

// TestCompatCalls_UpdateRecording covers Calls.UpdateRecording.
func TestCompatCalls_UpdateRecording(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("returns_recording_resource", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.Calls.UpdateRecording("CA_T2", "RE_T2", map[string]any{
			"Status": "paused",
		})
		if err != nil {
			t.Fatalf("UpdateRecording: %v", err)
		}
		if result == nil {
			t.Fatal("UpdateRecording returned nil result")
		}
		if _, hasSid := result["sid"]; !hasSid {
			if _, hasStatus := result["status"]; !hasStatus {
				t.Errorf("expected recording sid or status in body, got keys %v", keys(result))
			}
		}
	})

	t.Run("journal_records_post_to_specific_recording", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.Calls.UpdateRecording("CA_R1", "RE_R1", map[string]any{
			"Status": "paused",
		})
		if err != nil {
			t.Fatalf("UpdateRecording: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "POST" {
			t.Errorf("method = %q, want POST", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/Calls/CA_R1/Recordings/RE_R1"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
		body, ok := j.BodyMap()
		if !ok {
			t.Fatalf("expected JSON body, got %T: %v", j.Body, j.Body)
		}
		if body["Status"] != "paused" {
			t.Errorf("body[Status] = %v, want paused", body["Status"])
		}
	})
}

// keys returns the keys of m in unspecified order — used for assertion error
// messages.
func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
