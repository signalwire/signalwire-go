// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_fabric_mock.py.
//
// Closes the audit gaps: addresses, generic resources, SIP-endpoint
// sub-resources on subscribers, the call-flows / conference-rooms addresses
// sub-paths, the full FabricTokens surface, and CXMLApplicationsResource.Create
// deliberate-failure path.

package namespaces_test

import (
	"strings"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// ---------------- FabricAddresses ----------------

func TestFabricAddresses_List(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.Addresses.List(nil)
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
	if j.Path != "/api/fabric/addresses" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "fabric.list_fabric_addresses" {
		got := "<nil>"
		if j.MatchedRoute != nil {
			got = *j.MatchedRoute
		}
		t.Errorf("matched_route = %q, want fabric.list_fabric_addresses", got)
	}
}

func TestFabricAddresses_Get(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.Addresses.Get("addr-9001")
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
	if j.Path != "/api/fabric/addresses/addr-9001" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil (spec gap: address get)")
	}
}

// ---------------- CXMLApplications.Create deliberately fails ----------------

func TestFabricCXMLApplications_CreateRaisesNotImplemented(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.CXMLApplications.Create(map[string]any{"name": "never_built"})
	if err == nil {
		t.Fatal("Create did not return an error - the SDK must refuse this call")
	}
	if !strings.Contains(err.Error(), "cXML applications cannot") {
		t.Errorf("error message = %q, want substring 'cXML applications cannot'", err.Error())
	}
	// Nothing should have hit the wire.
	j := mock.Journal(t)
	if len(j) != 0 {
		t.Errorf("expected no journal entries, got %d: %v", len(j), j)
	}
}

// ---------------- CallFlows.ListAddresses uses singular path ----------------

func TestFabricCallFlows_ListAddressesUsesSingularPath(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CallFlows.ListAddresses("cf-1", nil)
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data' in body")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	const wantPath = "/api/fabric/resources/call_flow/cf-1/addresses"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

// ---------------- ConferenceRooms.ListAddresses uses singular path ----------------

func TestFabricConferenceRooms_ListAddressesUsesSingularPath(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.ConferenceRooms.ListAddresses("cr-1", nil)
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data' in body")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	const wantPath = "/api/fabric/resources/conference_room/cr-1/addresses"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

// ---------------- Subscribers SIP-endpoint per-id ops ----------------

func TestFabricSubscribers_GetSIPEndpoint(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.Subscribers.GetSIPEndpoint("sub-1", "ep-1")
	if err != nil {
		t.Fatalf("GetSIPEndpoint: %v", err)
	}
	if body == nil {
		t.Error("expected map")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	const wantPath = "/api/fabric/resources/subscribers/sub-1/sip_endpoints/ep-1"
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
}

func TestFabricSubscribers_UpdateSIPEndpointUsesPATCH(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Subscribers.UpdateSIPEndpoint("sub-1", "ep-1", map[string]any{
		"username": "renamed",
	})
	if err != nil {
		t.Fatalf("UpdateSIPEndpoint: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "PATCH" {
		t.Errorf("method = %q, want PATCH", j.Method)
	}
	const wantPath = "/api/fabric/resources/subscribers/sub-1/sip_endpoints/ep-1"
	if j.Path != wantPath {
		t.Errorf("path = %q", j.Path)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if body["username"] != "renamed" {
		t.Errorf("username = %v", body["username"])
	}
}

func TestFabricSubscribers_DeleteSIPEndpoint(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.Subscribers.DeleteSIPEndpoint("sub-1", "ep-1")
	if err != nil {
		t.Fatalf("DeleteSIPEndpoint: %v", err)
	}
	if body == nil {
		t.Error("expected map (204 normalized to {})")
	}
	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q", j.Method)
	}
	const wantPath = "/api/fabric/resources/subscribers/sub-1/sip_endpoints/ep-1"
	if j.Path != wantPath {
		t.Errorf("path = %q", j.Path)
	}
}

// ---------------- FabricTokens ----------------

func TestFabricTokens_CreateInviteToken(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Tokens.CreateInviteToken(map[string]any{
		"email": "invitee@example.com",
	})
	if err != nil {
		t.Fatalf("CreateInviteToken: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/fabric/subscriber/invites" {
		t.Errorf("path = %q", j.Path)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if body["email"] != "invitee@example.com" {
		t.Errorf("email = %v", body["email"])
	}
}

func TestFabricTokens_CreateEmbedToken(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Tokens.CreateEmbedToken(map[string]any{
		"allowed_addresses": []string{"addr-1", "addr-2"},
	})
	if err != nil {
		t.Fatalf("CreateEmbedToken: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/fabric/embeds/tokens" {
		t.Errorf("path = %q", j.Path)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	addrs, ok := body["allowed_addresses"].([]any)
	if !ok || len(addrs) != 2 || addrs[0] != "addr-1" || addrs[1] != "addr-2" {
		t.Errorf("allowed_addresses = %v", body["allowed_addresses"])
	}
}

func TestFabricTokens_RefreshSubscriberToken(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Tokens.RefreshSubscriberToken(map[string]any{
		"refresh_token": "abc-123",
	})
	if err != nil {
		t.Fatalf("RefreshSubscriberToken: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/fabric/subscribers/tokens/refresh" {
		t.Errorf("path = %q", j.Path)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if body["refresh_token"] != "abc-123" {
		t.Errorf("refresh_token = %v", body["refresh_token"])
	}
}

// ---------------- GenericResources ----------------

func TestFabricResources_List(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.Resources.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data', got keys %v", keys(body))
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/fabric/resources" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestFabricResources_Get(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Resources.Get("res-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/fabric/resources/res-1" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestFabricResources_Delete(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.Resources.Delete("res-2")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Error("expected map")
	}
	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/fabric/resources/res-2" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestFabricResources_ListAddresses(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.Resources.ListAddresses("res-3", nil)
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data'")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/fabric/resources/res-3/addresses" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestFabricResources_AssignDomainApplication(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Resources.AssignDomainApplication("res-4", map[string]any{
		"domain_application_id": "da-7",
	})
	if err != nil {
		t.Fatalf("AssignDomainApplication: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != "/api/fabric/resources/res-4/domain_applications" {
		t.Errorf("path = %q", j.Path)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if body["domain_application_id"] != "da-7" {
		t.Errorf("domain_application_id = %v", body["domain_application_id"])
	}
}
