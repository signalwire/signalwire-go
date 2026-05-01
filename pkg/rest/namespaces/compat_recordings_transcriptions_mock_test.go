// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_compat_recordings_transcriptions.py.
//
// Both resources expose the same surface (List/Get/Delete) and use the
// account-scoped LAML path. Twelve gap entries total.

package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

const compatRecsBase = "/api/laml/2010-04-01/Accounts/test_proj/Recordings"
const compatTransBase = "/api/laml/2010-04-01/Accounts/test_proj/Transcriptions"

// ---------- CompatRecordings ----------

func TestCompatRecordings_List_ReturnsPaginated(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Recordings.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	recs, ok := result["recordings"]
	if !ok {
		t.Fatalf("expected 'recordings' key, got %v", keys(result))
	}
	if _, isList := recs.([]any); !isList {
		t.Errorf("recordings type = %T, want []any", recs)
	}
}

func TestCompatRecordings_List_JournalRecordsGet(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Recordings.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	if j.Path != compatRecsBase {
		t.Errorf("path = %q, want %q", j.Path, compatRecsBase)
	}
}

func TestCompatRecordings_Get_ReturnsRecording(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Recordings.Get("RE_TEST")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	_, hasSid := result["sid"]
	_, hasCallSid := result["call_sid"]
	if !hasSid && !hasCallSid {
		t.Errorf("expected sid or call_sid, got %v", keys(result))
	}
}

func TestCompatRecordings_Get_JournalRecordsGetWithSid(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Recordings.Get("RE_GET")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	const wantPath = compatRecsBase + "/RE_GET"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

func TestCompatRecordings_Delete_NoException(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Recordings.Delete("RE_D")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if result == nil {
		t.Error("expected map, got nil")
	}
}

func TestCompatRecordings_Delete_JournalRecordsDelete(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Recordings.Delete("RE_DEL")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q, want DELETE", j.Method)
	}
	const wantPath = compatRecsBase + "/RE_DEL"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

// ---------- CompatTranscriptions ----------

func TestCompatTranscriptions_List_ReturnsPaginated(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Transcriptions.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	trans, ok := result["transcriptions"]
	if !ok {
		t.Fatalf("expected 'transcriptions' key, got %v", keys(result))
	}
	if _, isList := trans.([]any); !isList {
		t.Errorf("transcriptions type = %T, want []any", trans)
	}
}

func TestCompatTranscriptions_List_JournalRecordsGet(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Transcriptions.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	if j.Path != compatTransBase {
		t.Errorf("path = %q, want %q", j.Path, compatTransBase)
	}
}

func TestCompatTranscriptions_Get_ReturnsTranscription(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Transcriptions.Get("TR_TEST")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	_, hasSid := result["sid"]
	_, hasDuration := result["duration"]
	if !hasSid && !hasDuration {
		t.Errorf("expected sid or duration, got %v", keys(result))
	}
}

func TestCompatTranscriptions_Get_JournalRecordsGetWithSid(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Transcriptions.Get("TR_GET")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	const wantPath = compatTransBase + "/TR_GET"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

func TestCompatTranscriptions_Delete_NoException(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Transcriptions.Delete("TR_D")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if result == nil {
		t.Error("expected map, got nil")
	}
}

func TestCompatTranscriptions_Delete_JournalRecordsDelete(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Transcriptions.Delete("TR_DEL")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q, want DELETE", j.Method)
	}
	const wantPath = compatTransBase + "/TR_DEL"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

var _ = mocktest.JournalEntry{}
