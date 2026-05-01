// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_compat_misc.py.
//
// Covers the Compat resources with single-method gaps:
//   - CompatApplications.Update
//   - CompatLamlBins.Update

package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// ---------- CompatApplicationsUpdate ----------

func TestCompatApplications_Update_ReturnsApplication(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Applications.Update("AP_U", map[string]any{
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

func TestCompatApplications_Update_JournalRecordsPost(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.Applications.Update("AP_UU", map[string]any{
		"FriendlyName": "renamed",
		"VoiceUrl":     "https://a.b/v",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/Applications/AP_UU"
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
	if body["VoiceUrl"] != "https://a.b/v" {
		t.Errorf("VoiceUrl = %v", body["VoiceUrl"])
	}
}

// ---------- CompatLamlBinsUpdate ----------

func TestCompatLamlBins_Update_ReturnsLamlBin(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.LamlBins.Update("LB_U", map[string]any{
		"FriendlyName": "updated",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	_, hasFn := result["friendly_name"]
	_, hasSid := result["sid"]
	_, hasContents := result["contents"]
	if !hasFn && !hasSid && !hasContents {
		t.Errorf("expected friendly_name, sid, or contents, got %v", keys(result))
	}
}

func TestCompatLamlBins_Update_JournalRecordsPost(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	_, err := client.Compat.LamlBins.Update("LB_UU", map[string]any{
		"FriendlyName": "renamed",
		"Contents":     "<Response/>",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/LamlBins/LB_UU"
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
	if body["Contents"] != "<Response/>" {
		t.Errorf("Contents = %v", body["Contents"])
	}
}

// keep mocktest reference for build hygiene
var _ = mocktest.JournalEntry{}
