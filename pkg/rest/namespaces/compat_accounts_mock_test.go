// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_compat_accounts.py.
//
// Drives client.Compat.Accounts.* against the mock_signalwire HTTP server.
// Each test asserts on both the SDK return value and the recorded request
// journal so neither half is allowed to drift.

package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

const compatAccountsBase = "/api/laml/2010-04-01/Accounts"

// ---------- CompatAccountsCreate ----------

func TestCompatAccounts_Create_ReturnsAccountResource(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Accounts.Create(map[string]any{
		"FriendlyName": "Sub-A",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, ok := result["friendly_name"]; !ok {
		t.Errorf("missing 'friendly_name' in %v", keys(result))
	}
}

func TestCompatAccounts_Create_JournalRecordsPost(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Accounts.Create(map[string]any{
		"FriendlyName": "Sub-B",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	if j.Path != compatAccountsBase {
		t.Errorf("path = %q, want %q", j.Path, compatAccountsBase)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if body["FriendlyName"] != "Sub-B" {
		t.Errorf("FriendlyName = %v, want Sub-B", body["FriendlyName"])
	}
	if j.ResponseStatus == nil {
		t.Error("response_status is nil")
	} else if s := *j.ResponseStatus; s < 200 || s >= 400 {
		t.Errorf("response_status = %d, want 2xx/3xx", s)
	}
}

// ---------- CompatAccountsGet ----------

func TestCompatAccounts_Get_ReturnsAccountForSid(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Accounts.Get("AC123")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if _, ok := result["friendly_name"]; !ok {
		t.Errorf("missing 'friendly_name' in %v", keys(result))
	}
}

func TestCompatAccounts_Get_JournalRecordsGetWithSid(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Accounts.Get("AC_SAMPLE_SID")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	const wantPath = compatAccountsBase + "/AC_SAMPLE_SID"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil — spec gap: account-get should match a route")
	}
}

// ---------- CompatAccountsUpdate ----------

func TestCompatAccounts_Update_ReturnsUpdatedAccount(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Accounts.Update("AC123", map[string]any{
		"FriendlyName": "Renamed",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, ok := result["friendly_name"]; !ok {
		t.Errorf("missing 'friendly_name' in %v", keys(result))
	}
}

func TestCompatAccounts_Update_JournalRecordsPostToAccountPath(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Accounts.Update("AC_X", map[string]any{
		"FriendlyName": "NewName",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := mock.Last(t)
	// Twilio-compat update is POST (not PATCH/PUT).
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	const wantPath = compatAccountsBase + "/AC_X"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if body["FriendlyName"] != "NewName" {
		t.Errorf("FriendlyName = %v, want NewName", body["FriendlyName"])
	}
}
