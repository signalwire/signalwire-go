// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_compat_tokens.py.
//
// Covers CompatTokens.Create / Update / Delete. Note: CompatTokens.Update
// uses PATCH (not POST) because Python's CompatTokens extends BaseResource.

package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// compatTokensBase is the tokens collection base path for the harness's
// per-test random project (see lamlAccountBase). Function, not const, because
// the AccountSid segment is per-test (parallel isolation).
func compatTokensBase(m *mocktest.Harness) string {
	return lamlAccountBase(m) + "/tokens"
}

// ---------- CompatTokensCreate ----------

func TestCompatTokens_Create_ReturnsToken(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Tokens.Create(map[string]any{
		"Ttl": 3600,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_, hasToken := result["token"]
	_, hasID := result["id"]
	if !hasToken && !hasID {
		t.Errorf("expected token or id, got %v", keys(result))
	}
}

func TestCompatTokens_Create_JournalRecordsPost(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Tokens.Create(map[string]any{
		"Ttl":  3600,
		"Name": "api-key",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	if j.Path != compatTokensBase(mock) {
		t.Errorf("path = %q, want %q", j.Path, compatTokensBase(mock))
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if v, ok := body["Ttl"].(float64); !ok || v != 3600 {
		t.Errorf("Ttl = %v (%T), want 3600", body["Ttl"], body["Ttl"])
	}
	if body["Name"] != "api-key" {
		t.Errorf("Name = %v", body["Name"])
	}
}

// ---------- CompatTokensUpdate ----------

func TestCompatTokens_Update_ReturnsToken(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Tokens.Update("TK_U", map[string]any{
		"Ttl": 7200,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	_, hasToken := result["token"]
	_, hasID := result["id"]
	if !hasToken && !hasID {
		t.Errorf("expected token or id, got %v", keys(result))
	}
}

func TestCompatTokens_Update_JournalRecordsPatch(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Tokens.Update("TK_UU", map[string]any{
		"Ttl": 7200,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := mock.Last(t)
	// CompatTokens.Update uses PATCH (BaseResource.update -> http.patch).
	if j.Method != "PATCH" {
		t.Errorf("method = %q, want PATCH", j.Method)
	}
	wantPath := compatTokensBase(mock) + "/TK_UU"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if v, ok := body["Ttl"].(float64); !ok || v != 7200 {
		t.Errorf("Ttl = %v (%T), want 7200", body["Ttl"], body["Ttl"])
	}
}

// ---------- CompatTokensDelete ----------

func TestCompatTokens_Delete_NoException(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Tokens.Delete("TK_D")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if result == nil {
		t.Error("expected map, got nil")
	}
}

func TestCompatTokens_Delete_JournalRecordsDelete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Tokens.Delete("TK_DEL")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q, want DELETE", j.Method)
	}
	wantPath := compatTokensBase(mock) + "/TK_DEL"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

var _ = mocktest.JournalEntry{}
