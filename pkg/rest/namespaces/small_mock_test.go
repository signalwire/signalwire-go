// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from the addresses + recordings sections
// of signalwire-python/tests/unit/rest/test_small_namespaces_mock.py.
//
// Picked for the pilot because each namespace has only a handful of methods
// — addresses (list/get/create/delete) and recordings (list/get/delete) —
// and they exercise the spec-synthesised "data" collection shape that the
// rest of the small Relay namespaces share.

package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// ----------------- Addresses -----------------

func TestAddresses_List(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Addresses.List(map[string]string{"page_size": "10"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	data, ok := body["data"]
	if !ok {
		t.Fatalf("missing 'data', got keys %v", keys(body))
	}
	if _, isList := data.([]any); !isList {
		t.Errorf("data type = %T", data)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/relay/rest/addresses" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil")
	}
	if got := j.QueryParams["page_size"]; len(got) != 1 || got[0] != "10" {
		t.Errorf("query page_size = %v, want [10]", got)
	}
}

func TestAddresses_Create(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Addresses.Create(map[string]any{
		"address_type": "commercial",
		"first_name":   "Ada",
		"last_name":    "Lovelace",
		"country":      "US",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, ok := body["id"]; !ok {
		t.Errorf("response missing 'id', got keys %v", keys(body))
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/relay/rest/addresses" {
		t.Errorf("path = %q", j.Path)
	}
	sent, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if sent["address_type"] != "commercial" {
		t.Errorf("address_type = %v", sent["address_type"])
	}
	if sent["first_name"] != "Ada" {
		t.Errorf("first_name = %v", sent["first_name"])
	}
	if sent["country"] != "US" {
		t.Errorf("country = %v", sent["country"])
	}
}

func TestAddresses_Get(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Addresses.Get("addr-123")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if _, ok := body["id"]; !ok {
		t.Errorf("response missing 'id'")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/relay/rest/addresses/addr-123" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestAddresses_Delete(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Addresses.Delete("addr-123")
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
	if j.Path != "/api/relay/rest/addresses/addr-123" {
		t.Errorf("path = %q", j.Path)
	}
	if j.ResponseStatus == nil {
		t.Error("response_status is nil")
	} else if s := *j.ResponseStatus; s != 200 && s != 202 && s != 204 {
		t.Errorf("response_status = %d, want 200/202/204", s)
	}
}

// ----------------- Recordings -----------------

func TestRecordings_List(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Recordings.List(map[string]any{"page_size": 5})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	data, ok := body["data"]
	if !ok {
		t.Fatalf("missing 'data', got keys %v", keys(body))
	}
	if _, isList := data.([]any); !isList {
		t.Errorf("data type = %T", data)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/relay/rest/recordings" {
		t.Errorf("path = %q", j.Path)
	}
	if got := j.QueryParams["page_size"]; len(got) != 1 || got[0] != "5" {
		t.Errorf("query page_size = %v, want [5]", got)
	}
}

func TestRecordings_Get(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Recordings.Get("rec-123")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if _, ok := body["id"]; !ok {
		t.Errorf("response missing 'id'")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/relay/rest/recordings/rec-123" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestRecordings_Delete(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Recordings.Delete("rec-123")
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
	if j.Path != "/api/relay/rest/recordings/rec-123" {
		t.Errorf("path = %q", j.Path)
	}
	if j.ResponseStatus == nil {
		t.Error("response_status is nil")
	} else if s := *j.ResponseStatus; s != 200 && s != 202 && s != 204 {
		t.Errorf("response_status = %d, want 200/202/204", s)
	}
}
