// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from the remaining sections of
// signalwire-python/tests/unit/rest/test_small_namespaces_mock.py.
//
// The existing small_mock_test.go already covers the addresses + recordings
// sections from this Python file. This file picks up the rest:
//   - short_codes (list/get/update)
//   - imported_numbers (create)
//   - mfa (call)
//   - sip_profile (update)
//   - number_groups (list_memberships / delete_membership)
//   - project.tokens (update / delete)
//   - datasphere (get_chunk)
//   - queues (get_member)

package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// ---------- Short Codes ----------

func TestShortCodes_List(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.ShortCodes.List(map[string]any{"page_size": 20})
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
	if j.Path != "/api/relay/rest/short_codes" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestShortCodes_Get(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.ShortCodes.Get("sc-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if _, ok := body["id"]; !ok {
		t.Errorf("missing 'id' in %v", keys(body))
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/relay/rest/short_codes/sc-1" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestShortCodes_Update(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.ShortCodes.Update("sc-1", map[string]any{
		"name": "Marketing SMS",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, ok := body["id"]; !ok {
		t.Errorf("missing 'id' in %v", keys(body))
	}

	j := mock.Last(t)
	// short_codes uses PUT for update.
	if j.Method != "PUT" {
		t.Errorf("method = %q, want PUT", j.Method)
	}
	if j.Path != "/api/relay/rest/short_codes/sc-1" {
		t.Errorf("path = %q", j.Path)
	}
	sent, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if sent["name"] != "Marketing SMS" {
		t.Errorf("name = %v", sent["name"])
	}
}

// ---------- Imported Numbers ----------

func TestImportedNumbers_Create(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.ImportedNumbers.Create(map[string]any{
		"number":       "+15551234567",
		"sip_username": "alice",
		"sip_password": "secret",
		"sip_proxy":    "sip.example.com",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, ok := body["id"]; !ok {
		t.Errorf("missing 'id' in %v", keys(body))
	}

	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/relay/rest/imported_phone_numbers" {
		t.Errorf("path = %q", j.Path)
	}
	sent, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if sent["number"] != "+15551234567" {
		t.Errorf("number = %v", sent["number"])
	}
	if sent["sip_username"] != "alice" {
		t.Errorf("sip_username = %v", sent["sip_username"])
	}
	if sent["sip_proxy"] != "sip.example.com" {
		t.Errorf("sip_proxy = %v", sent["sip_proxy"])
	}
}

// ---------- MFA — voice channel ----------

func TestMFA_Call(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.MFA.Call(map[string]any{
		"to":      "+15551234567",
		"from_":   "+15559876543",
		"message": "Your code is {code}",
	})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if _, ok := body["id"]; !ok {
		t.Errorf("missing 'id' in %v", keys(body))
	}

	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/relay/rest/mfa/call" {
		t.Errorf("path = %q", j.Path)
	}
	sent, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if sent["to"] != "+15551234567" {
		t.Errorf("to = %v", sent["to"])
	}
	if sent["from_"] != "+15559876543" {
		t.Errorf("from_ = %v", sent["from_"])
	}
	if sent["message"] != "Your code is {code}" {
		t.Errorf("message = %v", sent["message"])
	}
}

// ---------- SIP Profile ----------

func TestSipProfile_Update(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.SipProfile.Update(map[string]any{
		"domain":         "myco.sip.signalwire.com",
		"default_codecs": []string{"PCMU", "PCMA"},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	_, hasDomain := body["domain"]
	_, hasCodecs := body["default_codecs"]
	if !hasDomain && !hasCodecs {
		t.Errorf("expected domain or default_codecs, got %v", keys(body))
	}

	j := mock.Last(t)
	if j.Method != "PUT" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/relay/rest/sip_profile" {
		t.Errorf("path = %q", j.Path)
	}
	sent, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if sent["domain"] != "myco.sip.signalwire.com" {
		t.Errorf("domain = %v", sent["domain"])
	}
	codecs, ok := sent["default_codecs"].([]any)
	if !ok {
		t.Fatalf("default_codecs type = %T", sent["default_codecs"])
	}
	if len(codecs) != 2 || codecs[0] != "PCMU" || codecs[1] != "PCMA" {
		t.Errorf("default_codecs = %v, want [PCMU PCMA]", codecs)
	}
}

// ---------- Number Groups ----------

func TestNumberGroups_ListMemberships(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.NumberGroups.ListMemberships("ng-1", map[string]string{
		"page_size": "10",
	})
	if err != nil {
		t.Fatalf("ListMemberships: %v", err)
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
	if j.Path != "/api/relay/rest/number_groups/ng-1/number_group_memberships" {
		t.Errorf("path = %q", j.Path)
	}
	if got := j.QueryParams["page_size"]; len(got) != 1 || got[0] != "10" {
		t.Errorf("query page_size = %v, want [10]", got)
	}
}

func TestNumberGroups_DeleteMembership(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.NumberGroups.DeleteMembership("mem-1")
	if err != nil {
		t.Fatalf("DeleteMembership: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/relay/rest/number_group_memberships/mem-1" {
		t.Errorf("path = %q", j.Path)
	}
	if j.ResponseStatus == nil {
		t.Error("response_status is nil")
	} else if s := *j.ResponseStatus; s != 200 && s != 202 && s != 204 {
		t.Errorf("response_status = %d, want 200/202/204", s)
	}
}

// ---------- Project tokens ----------

func TestProjectTokens_Update(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Project.Tokens.Update("tok-1", map[string]any{
		"name": "renamed-token",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, ok := body["id"]; !ok {
		t.Errorf("missing 'id' in %v", keys(body))
	}

	j := mock.Last(t)
	if j.Method != "PATCH" {
		t.Errorf("method = %q, want PATCH", j.Method)
	}
	if j.Path != "/api/project/tokens/tok-1" {
		t.Errorf("path = %q", j.Path)
	}
	sent, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if sent["name"] != "renamed-token" {
		t.Errorf("name = %v", sent["name"])
	}
}

func TestProjectTokens_Delete(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Project.Tokens.Delete("tok-1")
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
	if j.Path != "/api/project/tokens/tok-1" {
		t.Errorf("path = %q", j.Path)
	}
	if j.ResponseStatus == nil {
		t.Error("response_status is nil")
	} else if s := *j.ResponseStatus; s != 200 && s != 202 && s != 204 {
		t.Errorf("response_status = %d, want 200/202/204", s)
	}
}

// ---------- Datasphere ----------

func TestDatasphere_GetChunk(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Datasphere.Documents.GetChunk("doc-1", "chunk-99")
	if err != nil {
		t.Fatalf("GetChunk: %v", err)
	}
	if _, ok := body["id"]; !ok {
		t.Errorf("missing 'id' in %v", keys(body))
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/datasphere/documents/doc-1/chunks/chunk-99" {
		t.Errorf("path = %q", j.Path)
	}
}

// ---------- Queues — get_member ----------

func TestQueues_GetMember(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Queues.GetMember("q-1", "mem-7")
	if err != nil {
		t.Fatalf("GetMember: %v", err)
	}
	_, hasQ := body["queue_id"]
	_, hasC := body["call_id"]
	if !hasQ && !hasC {
		t.Errorf("expected queue_id or call_id, got %v", keys(body))
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/relay/rest/queues/q-1/members/mem-7" {
		t.Errorf("path = %q", j.Path)
	}
}

var _ = mocktest.JournalEntry{}
