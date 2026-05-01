// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_compat_queues.py.
//
// Covers CompatQueues.Update / ListMembers / GetMember / DequeueMember.

package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

const compatQueuesBase = "/api/laml/2010-04-01/Accounts/test_proj/Queues"

// ---------- CompatQueuesUpdate ----------

func TestCompatQueues_Update_ReturnsQueue(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Queues.Update("QU_U", map[string]any{
		"FriendlyName": "updated",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	_, hasFn := result["friendly_name"]
	_, hasSid := result["sid"]
	if !hasFn && !hasSid {
		t.Errorf("expected friendly_name or sid, got %v", keys(result))
	}
}

func TestCompatQueues_Update_JournalRecordsPost(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Queues.Update("QU_UU", map[string]any{
		"FriendlyName": "renamed",
		"MaxSize":      200,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	const wantPath = compatQueuesBase + "/QU_UU"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if body["FriendlyName"] != "renamed" {
		t.Errorf("FriendlyName = %v", body["FriendlyName"])
	}
	// JSON numbers decode to float64 in Go.
	if v, ok := body["MaxSize"].(float64); !ok || v != 200 {
		t.Errorf("MaxSize = %v (%T), want 200", body["MaxSize"], body["MaxSize"])
	}
}

// ---------- CompatQueuesListMembers ----------

func TestCompatQueues_ListMembers_ReturnsPaginated(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Queues.ListMembers("QU_LM", nil)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	members, ok := result["queue_members"]
	if !ok {
		t.Fatalf("expected 'queue_members', got %v", keys(result))
	}
	if _, isList := members.([]any); !isList {
		t.Errorf("queue_members type = %T, want []any", members)
	}
}

func TestCompatQueues_ListMembers_JournalRecordsGet(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Queues.ListMembers("QU_LMX", nil)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	const wantPath = compatQueuesBase + "/QU_LMX/Members"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

// ---------- CompatQueuesGetMember ----------

func TestCompatQueues_GetMember_ReturnsMember(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Queues.GetMember("QU_GM", "CA_GM")
	if err != nil {
		t.Fatalf("GetMember: %v", err)
	}
	_, hasCallSid := result["call_sid"]
	_, hasQueueSid := result["queue_sid"]
	if !hasCallSid && !hasQueueSid {
		t.Errorf("expected call_sid or queue_sid, got %v", keys(result))
	}
}

func TestCompatQueues_GetMember_JournalRecordsGet(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Queues.GetMember("QU_GMX", "CA_GMX")
	if err != nil {
		t.Fatalf("GetMember: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	const wantPath = compatQueuesBase + "/QU_GMX/Members/CA_GMX"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

// ---------- CompatQueuesDequeueMember ----------

func TestCompatQueues_DequeueMember_ReturnsMember(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Queues.DequeueMember("QU_DM", "CA_DM", map[string]any{
		"Url": "https://a.b",
	})
	if err != nil {
		t.Fatalf("DequeueMember: %v", err)
	}
	_, hasCallSid := result["call_sid"]
	_, hasQueueSid := result["queue_sid"]
	if !hasCallSid && !hasQueueSid {
		t.Errorf("expected call_sid or queue_sid, got %v", keys(result))
	}
}

func TestCompatQueues_DequeueMember_JournalRecordsPost(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Queues.DequeueMember("QU_DMX", "CA_DMX", map[string]any{
		"Url":    "https://a.b/url",
		"Method": "POST",
	})
	if err != nil {
		t.Fatalf("DequeueMember: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	const wantPath = compatQueuesBase + "/QU_DMX/Members/CA_DMX"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if body["Url"] != "https://a.b/url" {
		t.Errorf("Url = %v", body["Url"])
	}
	if body["Method"] != "POST" {
		t.Errorf("Method = %v", body["Method"])
	}
}

var _ = mocktest.JournalEntry{}
