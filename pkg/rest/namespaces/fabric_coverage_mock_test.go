// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Full success+error REST coverage for the `fabric` spec group.
//
// For every canonical fabric.* route reachable through the Go SDK surface this
// file provides BOTH a success (2xx) test — asserting the response plus the
// journaled Method/Path/MatchedRoute — AND an error test that arms a non-2xx
// scenario via mock.PushScenario and asserts the call returns a
// *rest.SignalWireRestError with the expected StatusCode, alongside the
// journaled ResponseStatus + MatchedRoute.
//
// GAPS (not covered here — no Go SDK surface or malformed canonical route):
//   - fabric.list_dialogflow_agents / get / update / delete / list_addresses (5)
//     — dialogflow_agents has no resource on the Go FabricNamespace.
//   - fabric.list_sip_gateway_addresses (1) — canonical path doubles the segment
//     (.../sip_gateways/resources/sip_gateways/{id}/addresses); the SDK builds a
//     plain .../sip_gateways/{id}/addresses, which matches no canonical route.
//   - fabric.assign_resource_sip_endpoint (1) — canonical path doubles the segment
//     (.../sip_endpoints/resources/{id}/sip_endpoints); no SDK method builds it.
//   - the *_addresses sub-routes for cxml_scripts, cxml_webhooks,
//     freeswitch_connectors, relay_applications, sip_endpoints, subscribers,
//     swml_scripts, swml_webhooks (8) — these resources are backed by Go's
//     FabricResourcePUT (= plain CrudResource), AutoMaterializedWebhookResource,
//     or SubscribersResource, NONE of which exposes ListAddresses. Python's
//     FabricResourcePUT(CrudWithAddresses) DOES, so list_addresses is reachable
//     there; this is a Go SDK drift, reported by the task. Without a source
//     change these 8 canonical routes cannot be reached on the correct path.

package namespaces_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest"
	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// fabAssertSuccess checks the most-recent journal entry for a successful call:
// method + path + matched_route all match. It returns the entry so the calling
// test can make a further in-body assertion on the response.
func fabAssertSuccess(t *testing.T, mock *mocktest.Harness, wantMethod, wantPath, wantRoute string) mocktest.JournalEntry {
	t.Helper()
	j := mock.Last(t)
	if j.Method != wantMethod {
		t.Errorf("method = %q, want %q", j.Method, wantMethod)
	}
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != wantRoute {
		got := "<nil>"
		if j.MatchedRoute != nil {
			got = *j.MatchedRoute
		}
		t.Errorf("matched_route = %q, want %q", got, wantRoute)
	}
	return j
}

// fabAssertError checks that err is a *rest.SignalWireRestError carrying
// wantStatus, and that the journaled entry recorded that status against
// wantRoute. It returns restErr so the calling test can assert on it in-body.
func fabAssertError(t *testing.T, mock *mocktest.Harness, err error, wantStatus int, wantRoute string) *rest.SignalWireRestError {
	t.Helper()
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *rest.SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != wantStatus {
		t.Errorf("StatusCode = %d, want %d", restErr.StatusCode, wantStatus)
	}
	j := mock.Last(t)
	if j.ResponseStatus == nil {
		t.Errorf("response_status = <nil>, want %d", wantStatus)
	} else if *j.ResponseStatus != wantStatus {
		t.Errorf("response_status = %d, want %d", *j.ResponseStatus, wantStatus)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != wantRoute {
		got := "<nil>"
		if j.MatchedRoute != nil {
			got = *j.MatchedRoute
		}
		t.Errorf("matched_route = %q, want %q", got, wantRoute)
	}
	return restErr
}

// ============================ Addresses ============================

func TestFabricCov_Addresses_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.Addresses.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data', keys=%v", keys(body))
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/addresses", "fabric.list_fabric_addresses")
}

func TestFabricCov_Addresses_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_fabric_addresses", 500, map[string]any{"error": "boom"})
	_, err := client.Fabric.Addresses.List(nil)
	e := fabAssertError(t, mock, err, 500, "fabric.list_fabric_addresses")
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Addresses_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.Addresses.Get("addr-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/addresses/addr-1", "fabric.get_fabric_address")
}

func TestFabricCov_Addresses_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_fabric_address", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.Addresses.Get("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_fabric_address")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ============================ Tokens ============================

func TestFabricCov_Tokens_CreateEmbed(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Tokens.CreateEmbedToken(map[string]any{"allowed_addresses": []string{"a"}})
	if err != nil {
		t.Fatalf("CreateEmbedToken: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/embeds/tokens", "fabric.create_embeds_token")
	if b, ok := j.BodyMap(); !ok || b["allowed_addresses"] == nil {
		t.Errorf("body = %v", j.Body)
	}
}

func TestFabricCov_Tokens_CreateEmbed_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_embeds_token", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.Tokens.CreateEmbedToken(map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_embeds_token")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Tokens_CreateGuest(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Tokens.CreateGuestToken(map[string]any{"allowed_addresses": []string{"a"}})
	if err != nil {
		t.Fatalf("CreateGuestToken: %v", err)
	}
	fabAssertSuccess(t, mock, "POST", "/api/fabric/guests/tokens", "fabric.create_subscriber_guest_token")
}

func TestFabricCov_Tokens_CreateGuest_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_subscriber_guest_token", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.Tokens.CreateGuestToken(map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_subscriber_guest_token")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Tokens_CreateInvite(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Tokens.CreateInviteToken(map[string]any{"email": "x@example.com"})
	if err != nil {
		t.Fatalf("CreateInviteToken: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/subscriber/invites", "fabric.create_subscriber_invite_token")
	if b, _ := j.BodyMap(); b["email"] != "x@example.com" {
		t.Errorf("email = %v", b["email"])
	}
}

func TestFabricCov_Tokens_CreateInvite_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_subscriber_invite_token", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.Tokens.CreateInviteToken(map[string]any{"email": "x"})
	e := fabAssertError(t, mock, err, 422, "fabric.create_subscriber_invite_token")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Tokens_CreateSubscriber(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Tokens.CreateSubscriberToken(map[string]any{"reference": "r1"})
	if err != nil {
		t.Fatalf("CreateSubscriberToken: %v", err)
	}
	fabAssertSuccess(t, mock, "POST", "/api/fabric/subscribers/tokens", "fabric.create_subscriber_token")
}

func TestFabricCov_Tokens_CreateSubscriber_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_subscriber_token", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.Tokens.CreateSubscriberToken(map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_subscriber_token")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Tokens_RefreshSubscriber(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Tokens.RefreshSubscriberToken(map[string]any{"refresh_token": "abc"})
	if err != nil {
		t.Fatalf("RefreshSubscriberToken: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/subscribers/tokens/refresh", "fabric.refresh_subscriber_token")
	if b, _ := j.BodyMap(); b["refresh_token"] != "abc" {
		t.Errorf("refresh_token = %v", b["refresh_token"])
	}
}

func TestFabricCov_Tokens_RefreshSubscriber_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.refresh_subscriber_token", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.Tokens.RefreshSubscriberToken(map[string]any{"refresh_token": "x"})
	e := fabAssertError(t, mock, err, 422, "fabric.refresh_subscriber_token")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ============================ Generic Resources ============================

func TestFabricCov_Resources_List(t *testing.T) {
	t.Parallel()
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
		t.Fatalf("missing 'data', keys=%v", keys(body))
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources", "fabric.list_resources")
}

func TestFabricCov_Resources_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_resources", 500, map[string]any{"error": "boom"})
	_, err := client.Fabric.Resources.List(nil)
	e := fabAssertError(t, mock, err, 500, "fabric.list_resources")
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Resources_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.Resources.Get("res-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/res-1", "fabric.get_resource")
}

func TestFabricCov_Resources_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_resource", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.Resources.Get("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_resource")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Resources_Delete(t *testing.T) {
	t.Parallel()
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
		t.Fatal("expected map (204 normalized)")
	}
	fabAssertSuccess(t, mock, "DELETE", "/api/fabric/resources/res-2", "fabric.delete_resource")
}

func TestFabricCov_Resources_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.delete_resource", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.Resources.Delete("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.delete_resource")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Resources_ListAddresses(t *testing.T) {
	t.Parallel()
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
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/res-3/addresses", "fabric.list_resource_addresses")
}

func TestFabricCov_Resources_ListAddresses_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_resource_addresses", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.Resources.ListAddresses("missing", nil)
	e := fabAssertError(t, mock, err, 404, "fabric.list_resource_addresses")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Resources_AssignDomainApplication(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Resources.AssignDomainApplication("res-4", map[string]any{"domain_application_id": "da-7"})
	if err != nil {
		t.Fatalf("AssignDomainApplication: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/res-4/domain_applications", "fabric.assign_resource_domain_application")
	if b, _ := j.BodyMap(); b["domain_application_id"] != "da-7" {
		t.Errorf("domain_application_id = %v", b["domain_application_id"])
	}
}

func TestFabricCov_Resources_AssignDomainApplication_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.assign_resource_domain_application", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.Resources.AssignDomainApplication("res-4", map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.assign_resource_domain_application")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Resources_AssignPhoneRoute(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Resources.AssignPhoneRoute("res-5", map[string]any{"phone_number": "+15550001111"})
	if err != nil {
		t.Fatalf("AssignPhoneRoute: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/res-5/phone_routes", "fabric.assign_resource_phone_route")
	if b, _ := j.BodyMap(); b["phone_number"] != "+15550001111" {
		t.Errorf("phone_number = %v", b["phone_number"])
	}
}

func TestFabricCov_Resources_AssignPhoneRoute_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.assign_resource_phone_route", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.Resources.AssignPhoneRoute("res-5", map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.assign_resource_phone_route")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ============================ AI Agents (CrudWithAddresses, PATCH) ============================

func TestFabricCov_AIAgents_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.AIAgents.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data', keys=%v", keys(body))
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/ai_agents", "fabric.list_ai_agents")
}

func TestFabricCov_AIAgents_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_ai_agents", 500, map[string]any{"error": "boom"})
	_, err := client.Fabric.AIAgents.List(nil)
	e := fabAssertError(t, mock, err, 500, "fabric.list_ai_agents")
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_AIAgents_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.AIAgents.Create(map[string]any{"name": "agent-1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/ai_agents", "fabric.create_ai_agent")
	if b, _ := j.BodyMap(); b["name"] != "agent-1" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_AIAgents_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_ai_agent", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.AIAgents.Create(map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_ai_agent")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_AIAgents_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.AIAgents.Get("agent-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/ai_agents/agent-1", "fabric.get_ai_agent")
}

func TestFabricCov_AIAgents_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_ai_agent", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.AIAgents.Get("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_ai_agent")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_AIAgents_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.AIAgents.Update("agent-1", map[string]any{"name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := fabAssertSuccess(t, mock, "PATCH", "/api/fabric/resources/ai_agents/agent-1", "fabric.update_ai_agent")
	if b, _ := j.BodyMap(); b["name"] != "renamed" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_AIAgents_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.update_ai_agent", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.AIAgents.Update("missing", map[string]any{"name": "x"})
	e := fabAssertError(t, mock, err, 404, "fabric.update_ai_agent")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_AIAgents_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.AIAgents.Delete("agent-2")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map")
	}
	fabAssertSuccess(t, mock, "DELETE", "/api/fabric/resources/ai_agents/agent-2", "fabric.delete_ai_agent")
}

func TestFabricCov_AIAgents_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.delete_ai_agent", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.AIAgents.Delete("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.delete_ai_agent")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_AIAgents_ListAddresses(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.AIAgents.ListAddresses("agent-3", nil)
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data'")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/ai_agents/agent-3/addresses", "fabric.list_ai_agent_addresses")
}

func TestFabricCov_AIAgents_ListAddresses_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_ai_agent_addresses", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.AIAgents.ListAddresses("missing", nil)
	e := fabAssertError(t, mock, err, 404, "fabric.list_ai_agent_addresses")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ============================ Call Flows (PUT update; singular sub-paths) ============================

func TestFabricCov_CallFlows_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CallFlows.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data', keys=%v", keys(body))
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/call_flows", "fabric.list_call_flows")
}

func TestFabricCov_CallFlows_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_call_flows", 500, map[string]any{"error": "boom"})
	_, err := client.Fabric.CallFlows.List(nil)
	e := fabAssertError(t, mock, err, 500, "fabric.list_call_flows")
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CallFlows_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.CallFlows.Create(map[string]any{"name": "cf"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/call_flows", "fabric.create_call_flow")
	if b, _ := j.BodyMap(); b["name"] != "cf" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_CallFlows_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_call_flow", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.CallFlows.Create(map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_call_flow")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CallFlows_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CallFlows.Get("cf-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/call_flows/cf-1", "fabric.get_call_flow")
}

func TestFabricCov_CallFlows_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_call_flow", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CallFlows.Get("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_call_flow")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CallFlows_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.CallFlows.Update("cf-1", map[string]any{"name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := fabAssertSuccess(t, mock, "PUT", "/api/fabric/resources/call_flows/cf-1", "fabric.update_call_flow")
	if b, _ := j.BodyMap(); b["name"] != "renamed" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_CallFlows_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.update_call_flow", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CallFlows.Update("missing", map[string]any{"name": "x"})
	e := fabAssertError(t, mock, err, 404, "fabric.update_call_flow")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CallFlows_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CallFlows.Delete("cf-2")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map")
	}
	fabAssertSuccess(t, mock, "DELETE", "/api/fabric/resources/call_flows/cf-2", "fabric.delete_call_flow")
}

func TestFabricCov_CallFlows_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.delete_call_flow", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CallFlows.Delete("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.delete_call_flow")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CallFlows_ListAddresses(t *testing.T) {
	t.Parallel()
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
		t.Errorf("missing 'data'")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/call_flow/cf-1/addresses", "fabric.list_call_flow_addresses")
}

func TestFabricCov_CallFlows_ListAddresses_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_call_flow_addresses", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CallFlows.ListAddresses("missing", nil)
	e := fabAssertError(t, mock, err, 404, "fabric.list_call_flow_addresses")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CallFlows_ListVersions(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CallFlows.ListVersions("cf-1", nil)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data'")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/call_flow/cf-1/versions", "fabric.list_call_flow_versions")
}

func TestFabricCov_CallFlows_ListVersions_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_call_flow_versions", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CallFlows.ListVersions("missing", nil)
	e := fabAssertError(t, mock, err, 404, "fabric.list_call_flow_versions")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CallFlows_DeployVersion(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.CallFlows.DeployVersion("cf-1", map[string]any{"version": "v2"})
	if err != nil {
		t.Fatalf("DeployVersion: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/call_flow/cf-1/versions", "fabric.deploy_call_flow_version")
	if b, _ := j.BodyMap(); b["version"] != "v2" {
		t.Errorf("version = %v", b["version"])
	}
}

func TestFabricCov_CallFlows_DeployVersion_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.deploy_call_flow_version", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.CallFlows.DeployVersion("cf-1", map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.deploy_call_flow_version")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ============================ Conference Rooms (PUT update; singular addresses) ============================

func TestFabricCov_ConferenceRooms_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.ConferenceRooms.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data', keys=%v", keys(body))
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/conference_rooms", "fabric.list_conference_rooms")
}

func TestFabricCov_ConferenceRooms_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_conference_rooms", 500, map[string]any{"error": "boom"})
	_, err := client.Fabric.ConferenceRooms.List(nil)
	e := fabAssertError(t, mock, err, 500, "fabric.list_conference_rooms")
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_ConferenceRooms_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.ConferenceRooms.Create(map[string]any{"name": "cr"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/conference_rooms", "fabric.create_conference_room")
	if b, _ := j.BodyMap(); b["name"] != "cr" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_ConferenceRooms_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_conference_room", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.ConferenceRooms.Create(map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_conference_room")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_ConferenceRooms_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.ConferenceRooms.Get("cr-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/conference_rooms/cr-1", "fabric.get_conference_room")
}

func TestFabricCov_ConferenceRooms_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_conference_room", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.ConferenceRooms.Get("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_conference_room")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_ConferenceRooms_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.ConferenceRooms.Update("cr-1", map[string]any{"name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := fabAssertSuccess(t, mock, "PUT", "/api/fabric/resources/conference_rooms/cr-1", "fabric.update_conference_room")
	if b, _ := j.BodyMap(); b["name"] != "renamed" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_ConferenceRooms_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.update_conference_room", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.ConferenceRooms.Update("missing", map[string]any{"name": "x"})
	e := fabAssertError(t, mock, err, 404, "fabric.update_conference_room")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_ConferenceRooms_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.ConferenceRooms.Delete("cr-2")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map")
	}
	fabAssertSuccess(t, mock, "DELETE", "/api/fabric/resources/conference_rooms/cr-2", "fabric.delete_conference_room")
}

func TestFabricCov_ConferenceRooms_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.delete_conference_room", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.ConferenceRooms.Delete("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.delete_conference_room")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_ConferenceRooms_ListAddresses(t *testing.T) {
	t.Parallel()
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
		t.Errorf("missing 'data'")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/conference_room/cr-1/addresses", "fabric.list_conference_room_addresses")
}

func TestFabricCov_ConferenceRooms_ListAddresses_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_conference_room_addresses", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.ConferenceRooms.ListAddresses("missing", nil)
	e := fabAssertError(t, mock, err, 404, "fabric.list_conference_room_addresses")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ============================ CXML Applications (PUT update; Create disallowed) ============================

func TestFabricCov_CXMLApplications_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CXMLApplications.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data', keys=%v", keys(body))
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/cxml_applications", "fabric.list_cxml_applications")
}

func TestFabricCov_CXMLApplications_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_cxml_applications", 500, map[string]any{"error": "boom"})
	_, err := client.Fabric.CXMLApplications.List(nil)
	e := fabAssertError(t, mock, err, 500, "fabric.list_cxml_applications")
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CXMLApplications_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CXMLApplications.Get("app-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/cxml_applications/app-1", "fabric.get_cxml_application")
}

func TestFabricCov_CXMLApplications_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_cxml_application", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CXMLApplications.Get("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_cxml_application")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CXMLApplications_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.CXMLApplications.Update("app-1", map[string]any{"name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := fabAssertSuccess(t, mock, "PUT", "/api/fabric/resources/cxml_applications/app-1", "fabric.update_cxml_application")
	if b, _ := j.BodyMap(); b["name"] != "renamed" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_CXMLApplications_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.update_cxml_application", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CXMLApplications.Update("missing", map[string]any{"name": "x"})
	e := fabAssertError(t, mock, err, 404, "fabric.update_cxml_application")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CXMLApplications_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CXMLApplications.Delete("app-2")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map")
	}
	fabAssertSuccess(t, mock, "DELETE", "/api/fabric/resources/cxml_applications/app-2", "fabric.delete_cxml_application")
}

func TestFabricCov_CXMLApplications_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.delete_cxml_application", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CXMLApplications.Delete("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.delete_cxml_application")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CXMLApplications_ListAddresses(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CXMLApplications.ListAddresses("app-3", nil)
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data'")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/cxml_applications/app-3/addresses", "fabric.list_cxml_application_addresses")
}

func TestFabricCov_CXMLApplications_ListAddresses_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_cxml_application_addresses", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CXMLApplications.ListAddresses("missing", nil)
	e := fabAssertError(t, mock, err, 404, "fabric.list_cxml_application_addresses")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// CXMLApplications.Create is deliberately disallowed by the SDK (mirrors
// Python's NotImplementedError). It is NOT a canonical route; assert the
// real refusal behavior and that nothing reached the wire.
func TestFabricCov_CXMLApplications_CreateRefused(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.CXMLApplications.Create(map[string]any{"name": "x"})
	if err == nil {
		t.Fatal("Create must return an error - cXML applications cannot be created via this API")
	}
	if !strings.Contains(err.Error(), "cXML applications cannot") {
		t.Errorf("error = %q, want substring 'cXML applications cannot'", err.Error())
	}
	if j := mock.Journal(t); len(j) != 0 {
		t.Errorf("expected no journal entries, got %d", len(j))
	}
}

// ============================ CXML Scripts (PUT update) ============================

func TestFabricCov_CXMLScripts_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CXMLScripts.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data', keys=%v", keys(body))
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/cxml_scripts", "fabric.list_cxml_scripts")
}

func TestFabricCov_CXMLScripts_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_cxml_scripts", 500, map[string]any{"error": "boom"})
	_, err := client.Fabric.CXMLScripts.List(nil)
	e := fabAssertError(t, mock, err, 500, "fabric.list_cxml_scripts")
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CXMLScripts_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.CXMLScripts.Create(map[string]any{"name": "s"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/cxml_scripts", "fabric.create_cxml_script")
	if b, _ := j.BodyMap(); b["name"] != "s" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_CXMLScripts_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_cxml_script", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.CXMLScripts.Create(map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_cxml_script")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CXMLScripts_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CXMLScripts.Get("s-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/cxml_scripts/s-1", "fabric.get_cxml_script")
}

func TestFabricCov_CXMLScripts_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_cxml_script", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CXMLScripts.Get("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_cxml_script")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CXMLScripts_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.CXMLScripts.Update("s-1", map[string]any{"name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := fabAssertSuccess(t, mock, "PUT", "/api/fabric/resources/cxml_scripts/s-1", "fabric.update_cxml_script")
	if b, _ := j.BodyMap(); b["name"] != "renamed" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_CXMLScripts_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.update_cxml_script", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CXMLScripts.Update("missing", map[string]any{"name": "x"})
	e := fabAssertError(t, mock, err, 404, "fabric.update_cxml_script")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CXMLScripts_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CXMLScripts.Delete("s-2")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map")
	}
	fabAssertSuccess(t, mock, "DELETE", "/api/fabric/resources/cxml_scripts/s-2", "fabric.delete_cxml_script")
}

func TestFabricCov_CXMLScripts_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.delete_cxml_script", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CXMLScripts.Delete("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.delete_cxml_script")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ============================ CXML Webhooks (PATCH update; auto-materialized Create) ============================

func TestFabricCov_CXMLWebhooks_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CXMLWebhooks.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data', keys=%v", keys(body))
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/cxml_webhooks", "fabric.list_cxml_webhooks")
}

func TestFabricCov_CXMLWebhooks_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_cxml_webhooks", 500, map[string]any{"error": "boom"})
	_, err := client.Fabric.CXMLWebhooks.List(nil)
	e := fabAssertError(t, mock, err, 500, "fabric.list_cxml_webhooks")
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CXMLWebhooks_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.CXMLWebhooks.Create(map[string]any{"name": "w"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/cxml_webhooks", "fabric.create_cxml_webhook")
	if b, _ := j.BodyMap(); b["name"] != "w" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_CXMLWebhooks_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_cxml_webhook", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.CXMLWebhooks.Create(map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_cxml_webhook")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CXMLWebhooks_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CXMLWebhooks.Get("w-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/cxml_webhooks/w-1", "fabric.get_cxml_webhook")
}

func TestFabricCov_CXMLWebhooks_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_cxml_webhook", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CXMLWebhooks.Get("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_cxml_webhook")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CXMLWebhooks_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.CXMLWebhooks.Update("w-1", map[string]any{"name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := fabAssertSuccess(t, mock, "PATCH", "/api/fabric/resources/cxml_webhooks/w-1", "fabric.update_cxml_webhook")
	if b, _ := j.BodyMap(); b["name"] != "renamed" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_CXMLWebhooks_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.update_cxml_webhook", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CXMLWebhooks.Update("missing", map[string]any{"name": "x"})
	e := fabAssertError(t, mock, err, 404, "fabric.update_cxml_webhook")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_CXMLWebhooks_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CXMLWebhooks.Delete("w-2")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map")
	}
	fabAssertSuccess(t, mock, "DELETE", "/api/fabric/resources/cxml_webhooks/w-2", "fabric.delete_cxml_webhook")
}

func TestFabricCov_CXMLWebhooks_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.delete_cxml_webhook", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CXMLWebhooks.Delete("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.delete_cxml_webhook")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ============================ FreeSwitch Connectors (PUT update) ============================

func TestFabricCov_FreeSwitchConnectors_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.FreeSwitchConnectors.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data', keys=%v", keys(body))
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/freeswitch_connectors", "fabric.list_freeswitch_connectors")
}

func TestFabricCov_FreeSwitchConnectors_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_freeswitch_connectors", 500, map[string]any{"error": "boom"})
	_, err := client.Fabric.FreeSwitchConnectors.List(nil)
	e := fabAssertError(t, mock, err, 500, "fabric.list_freeswitch_connectors")
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_FreeSwitchConnectors_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.FreeSwitchConnectors.Create(map[string]any{"name": "fc"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/freeswitch_connectors", "fabric.create_freeswitch_connector")
	if b, _ := j.BodyMap(); b["name"] != "fc" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_FreeSwitchConnectors_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_freeswitch_connector", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.FreeSwitchConnectors.Create(map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_freeswitch_connector")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_FreeSwitchConnectors_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.FreeSwitchConnectors.Get("fc-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/freeswitch_connectors/fc-1", "fabric.get_freeswitch_connector")
}

func TestFabricCov_FreeSwitchConnectors_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_freeswitch_connector", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.FreeSwitchConnectors.Get("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_freeswitch_connector")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_FreeSwitchConnectors_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.FreeSwitchConnectors.Update("fc-1", map[string]any{"name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := fabAssertSuccess(t, mock, "PUT", "/api/fabric/resources/freeswitch_connectors/fc-1", "fabric.update_freeswitch_connector")
	if b, _ := j.BodyMap(); b["name"] != "renamed" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_FreeSwitchConnectors_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.update_freeswitch_connector", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.FreeSwitchConnectors.Update("missing", map[string]any{"name": "x"})
	e := fabAssertError(t, mock, err, 404, "fabric.update_freeswitch_connector")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_FreeSwitchConnectors_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.FreeSwitchConnectors.Delete("fc-2")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map")
	}
	fabAssertSuccess(t, mock, "DELETE", "/api/fabric/resources/freeswitch_connectors/fc-2", "fabric.delete_freeswitch_connector")
}

func TestFabricCov_FreeSwitchConnectors_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.delete_freeswitch_connector", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.FreeSwitchConnectors.Delete("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.delete_freeswitch_connector")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ============================ Relay Applications (PUT update) ============================

func TestFabricCov_RelayApplications_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.RelayApplications.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data', keys=%v", keys(body))
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/relay_applications", "fabric.list_relay_applications")
}

func TestFabricCov_RelayApplications_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_relay_applications", 500, map[string]any{"error": "boom"})
	_, err := client.Fabric.RelayApplications.List(nil)
	e := fabAssertError(t, mock, err, 500, "fabric.list_relay_applications")
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_RelayApplications_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.RelayApplications.Create(map[string]any{"name": "ra"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/relay_applications", "fabric.create_relay_application")
	if b, _ := j.BodyMap(); b["name"] != "ra" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_RelayApplications_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_relay_application", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.RelayApplications.Create(map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_relay_application")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_RelayApplications_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.RelayApplications.Get("ra-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/relay_applications/ra-1", "fabric.get_relay_application")
}

func TestFabricCov_RelayApplications_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_relay_application", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.RelayApplications.Get("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_relay_application")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_RelayApplications_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.RelayApplications.Update("ra-1", map[string]any{"name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := fabAssertSuccess(t, mock, "PUT", "/api/fabric/resources/relay_applications/ra-1", "fabric.update_relay_application")
	if b, _ := j.BodyMap(); b["name"] != "renamed" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_RelayApplications_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.update_relay_application", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.RelayApplications.Update("missing", map[string]any{"name": "x"})
	e := fabAssertError(t, mock, err, 404, "fabric.update_relay_application")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_RelayApplications_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.RelayApplications.Delete("ra-2")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map")
	}
	fabAssertSuccess(t, mock, "DELETE", "/api/fabric/resources/relay_applications/ra-2", "fabric.delete_relay_application")
}

func TestFabricCov_RelayApplications_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.delete_relay_application", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.RelayApplications.Delete("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.delete_relay_application")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ============================ SIP Endpoints (PUT update) ============================

func TestFabricCov_SIPEndpoints_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	// NOTE: the mock synthesizes this list response as a top-level JSON ARRAY
	// (`[{"data":[...]}]`) from the spec, which the SDK's map[string]any return
	// type cannot hold, so List returns a non-nil unmarshal error even though
	// the request succeeded (HTTP 200, journaled below). This is a mock
	// response-synthesis artifact, not an SDK bug — assert success on the
	// journal entry (method/path/route + 2xx) rather than the parsed body.
	_, _ = client.Fabric.SIPEndpoints.List(nil)
	j := fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/sip_endpoints", "fabric.list_sip_endpoints")
	if j.ResponseStatus == nil || *j.ResponseStatus < 200 || *j.ResponseStatus >= 300 {
		t.Errorf("response_status = %v, want 2xx", j.ResponseStatus)
	}
}

func TestFabricCov_SIPEndpoints_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_sip_endpoints", 500, map[string]any{"error": "boom"})
	_, err := client.Fabric.SIPEndpoints.List(nil)
	e := fabAssertError(t, mock, err, 500, "fabric.list_sip_endpoints")
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SIPEndpoints_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.SIPEndpoints.Create(map[string]any{"username": "u"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/sip_endpoints", "fabric.create_sip_endpoint")
	if b, _ := j.BodyMap(); b["username"] != "u" {
		t.Errorf("username = %v", b["username"])
	}
}

func TestFabricCov_SIPEndpoints_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_sip_endpoint", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.SIPEndpoints.Create(map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_sip_endpoint")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SIPEndpoints_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.SIPEndpoints.Get("se-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/sip_endpoints/se-1", "fabric.get_sip_endpoint")
}

func TestFabricCov_SIPEndpoints_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_sip_endpoint", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.SIPEndpoints.Get("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_sip_endpoint")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SIPEndpoints_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.SIPEndpoints.Update("se-1", map[string]any{"username": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := fabAssertSuccess(t, mock, "PUT", "/api/fabric/resources/sip_endpoints/se-1", "fabric.update_sip_endpoint")
	if b, _ := j.BodyMap(); b["username"] != "renamed" {
		t.Errorf("username = %v", b["username"])
	}
}

func TestFabricCov_SIPEndpoints_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.update_sip_endpoint", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.SIPEndpoints.Update("missing", map[string]any{"username": "x"})
	e := fabAssertError(t, mock, err, 404, "fabric.update_sip_endpoint")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SIPEndpoints_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.SIPEndpoints.Delete("se-2")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map")
	}
	fabAssertSuccess(t, mock, "DELETE", "/api/fabric/resources/sip_endpoints/se-2", "fabric.delete_sip_endpoint")
}

func TestFabricCov_SIPEndpoints_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.delete_sip_endpoint", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.SIPEndpoints.Delete("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.delete_sip_endpoint")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ============================ SIP Gateways (PATCH update) ============================

func TestFabricCov_SIPGateways_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.SIPGateways.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data', keys=%v", keys(body))
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/sip_gateways", "fabric.list_sip_gateways")
}

func TestFabricCov_SIPGateways_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_sip_gateways", 500, map[string]any{"error": "boom"})
	_, err := client.Fabric.SIPGateways.List(nil)
	e := fabAssertError(t, mock, err, 500, "fabric.list_sip_gateways")
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SIPGateways_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.SIPGateways.Create(map[string]any{"name": "gw"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/sip_gateways", "fabric.create_sip_gateway")
	if b, _ := j.BodyMap(); b["name"] != "gw" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_SIPGateways_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_sip_gateway", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.SIPGateways.Create(map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_sip_gateway")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SIPGateways_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.SIPGateways.Get("gw-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/sip_gateways/gw-1", "fabric.get_sip_gateway")
}

func TestFabricCov_SIPGateways_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_sip_gateway", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.SIPGateways.Get("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_sip_gateway")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SIPGateways_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.SIPGateways.Update("gw-1", map[string]any{"name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := fabAssertSuccess(t, mock, "PATCH", "/api/fabric/resources/sip_gateways/gw-1", "fabric.update_sip_gateway")
	if b, _ := j.BodyMap(); b["name"] != "renamed" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_SIPGateways_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.update_sip_gateway", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.SIPGateways.Update("missing", map[string]any{"name": "x"})
	e := fabAssertError(t, mock, err, 404, "fabric.update_sip_gateway")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SIPGateways_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.SIPGateways.Delete("gw-2")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map")
	}
	fabAssertSuccess(t, mock, "DELETE", "/api/fabric/resources/sip_gateways/gw-2", "fabric.delete_sip_gateway")
}

func TestFabricCov_SIPGateways_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.delete_sip_gateway", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.SIPGateways.Delete("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.delete_sip_gateway")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ============================ Subscribers (PUT update) + SIP endpoint sub-resources ============================

func TestFabricCov_Subscribers_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.Subscribers.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data', keys=%v", keys(body))
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/subscribers", "fabric.list_subscribers")
}

func TestFabricCov_Subscribers_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_subscribers", 500, map[string]any{"error": "boom"})
	_, err := client.Fabric.Subscribers.List(nil)
	e := fabAssertError(t, mock, err, 500, "fabric.list_subscribers")
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Subscribers_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Subscribers.Create(map[string]any{"email": "s@example.com"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/subscribers", "fabric.create_subscriber")
	if b, _ := j.BodyMap(); b["email"] != "s@example.com" {
		t.Errorf("email = %v", b["email"])
	}
}

func TestFabricCov_Subscribers_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_subscriber", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.Subscribers.Create(map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_subscriber")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Subscribers_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.Subscribers.Get("sub-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/subscribers/sub-1", "fabric.get_subscriber")
}

func TestFabricCov_Subscribers_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_subscriber", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.Subscribers.Get("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_subscriber")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Subscribers_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Subscribers.Update("sub-1", map[string]any{"email": "new@example.com"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := fabAssertSuccess(t, mock, "PUT", "/api/fabric/resources/subscribers/sub-1", "fabric.update_subscriber")
	if b, _ := j.BodyMap(); b["email"] != "new@example.com" {
		t.Errorf("email = %v", b["email"])
	}
}

func TestFabricCov_Subscribers_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.update_subscriber", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.Subscribers.Update("missing", map[string]any{"email": "x"})
	e := fabAssertError(t, mock, err, 404, "fabric.update_subscriber")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Subscribers_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.Subscribers.Delete("sub-2")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map")
	}
	fabAssertSuccess(t, mock, "DELETE", "/api/fabric/resources/subscribers/sub-2", "fabric.delete_subscriber")
}

func TestFabricCov_Subscribers_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.delete_subscriber", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.Subscribers.Delete("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.delete_subscriber")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Subscribers_ListSIPEndpoints(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.Subscribers.ListSIPEndpoints("sub-1", nil)
	if err != nil {
		t.Fatalf("ListSIPEndpoints: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data'")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/subscribers/sub-1/sip_endpoints", "fabric.list_subscriber_sip_endpoints")
}

func TestFabricCov_Subscribers_ListSIPEndpoints_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_subscriber_sip_endpoints", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.Subscribers.ListSIPEndpoints("missing", nil)
	e := fabAssertError(t, mock, err, 404, "fabric.list_subscriber_sip_endpoints")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Subscribers_CreateSIPEndpoint(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Subscribers.CreateSIPEndpoint("sub-1", map[string]any{"username": "u"})
	if err != nil {
		t.Fatalf("CreateSIPEndpoint: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/subscribers/sub-1/sip_endpoints", "fabric.create_subscriber_sip_endpoint")
	if b, _ := j.BodyMap(); b["username"] != "u" {
		t.Errorf("username = %v", b["username"])
	}
}

func TestFabricCov_Subscribers_CreateSIPEndpoint_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_subscriber_sip_endpoint", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.Subscribers.CreateSIPEndpoint("sub-1", map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_subscriber_sip_endpoint")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Subscribers_GetSIPEndpoint(t *testing.T) {
	t.Parallel()
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
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/subscribers/sub-1/sip_endpoints/ep-1", "fabric.get_subscriber_sip_endpoint")
}

func TestFabricCov_Subscribers_GetSIPEndpoint_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_subscriber_sip_endpoint", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.Subscribers.GetSIPEndpoint("sub-1", "missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_subscriber_sip_endpoint")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Subscribers_UpdateSIPEndpoint(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.Subscribers.UpdateSIPEndpoint("sub-1", "ep-1", map[string]any{"username": "renamed"})
	if err != nil {
		t.Fatalf("UpdateSIPEndpoint: %v", err)
	}
	j := fabAssertSuccess(t, mock, "PATCH", "/api/fabric/resources/subscribers/sub-1/sip_endpoints/ep-1", "fabric.update_subscriber_sip_endpoint")
	if b, _ := j.BodyMap(); b["username"] != "renamed" {
		t.Errorf("username = %v", b["username"])
	}
}

func TestFabricCov_Subscribers_UpdateSIPEndpoint_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.update_subscriber_sip_endpoint", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.Subscribers.UpdateSIPEndpoint("sub-1", "missing", map[string]any{"username": "x"})
	e := fabAssertError(t, mock, err, 404, "fabric.update_subscriber_sip_endpoint")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_Subscribers_DeleteSIPEndpoint(t *testing.T) {
	t.Parallel()
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
		t.Fatal("expected map")
	}
	fabAssertSuccess(t, mock, "DELETE", "/api/fabric/resources/subscribers/sub-1/sip_endpoints/ep-1", "fabric.delete_subscriber_sip_endpoint")
}

func TestFabricCov_Subscribers_DeleteSIPEndpoint_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.delete_subscriber_sip_endpoint", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.Subscribers.DeleteSIPEndpoint("sub-1", "missing")
	e := fabAssertError(t, mock, err, 404, "fabric.delete_subscriber_sip_endpoint")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ============================ SWML Scripts (PUT update) ============================

func TestFabricCov_SWMLScripts_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	// NOTE: the mock synthesizes this list response as a top-level JSON ARRAY
	// (`[{"data":[...]}]`) from the spec, which the SDK's map[string]any return
	// type cannot hold, so List returns a non-nil unmarshal error even though
	// the request succeeded (HTTP 200, journaled below). This is a mock
	// response-synthesis artifact, not an SDK bug — assert success on the
	// journal entry (method/path/route + 2xx) rather than the parsed body.
	_, _ = client.Fabric.SWMLScripts.List(nil)
	j := fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/swml_scripts", "fabric.list_swml_scripts")
	if j.ResponseStatus == nil || *j.ResponseStatus < 200 || *j.ResponseStatus >= 300 {
		t.Errorf("response_status = %v, want 2xx", j.ResponseStatus)
	}
}

func TestFabricCov_SWMLScripts_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_swml_scripts", 500, map[string]any{"error": "boom"})
	_, err := client.Fabric.SWMLScripts.List(nil)
	e := fabAssertError(t, mock, err, 500, "fabric.list_swml_scripts")
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SWMLScripts_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.SWMLScripts.Create(map[string]any{"name": "s"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/swml_scripts", "fabric.create_swml_script")
	if b, _ := j.BodyMap(); b["name"] != "s" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_SWMLScripts_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_swml_script", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.SWMLScripts.Create(map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_swml_script")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SWMLScripts_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.SWMLScripts.Get("s-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/swml_scripts/s-1", "fabric.get_swml_script")
}

func TestFabricCov_SWMLScripts_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_swml_script", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.SWMLScripts.Get("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_swml_script")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SWMLScripts_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.SWMLScripts.Update("s-1", map[string]any{"name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := fabAssertSuccess(t, mock, "PUT", "/api/fabric/resources/swml_scripts/s-1", "fabric.update_swml_script")
	if b, _ := j.BodyMap(); b["name"] != "renamed" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_SWMLScripts_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.update_swml_script", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.SWMLScripts.Update("missing", map[string]any{"name": "x"})
	e := fabAssertError(t, mock, err, 404, "fabric.update_swml_script")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SWMLScripts_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.SWMLScripts.Delete("s-2")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map")
	}
	fabAssertSuccess(t, mock, "DELETE", "/api/fabric/resources/swml_scripts/s-2", "fabric.delete_swml_script")
}

func TestFabricCov_SWMLScripts_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.delete_swml_script", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.SWMLScripts.Delete("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.delete_swml_script")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ============================ SWML Webhooks (PATCH update; auto-materialized Create) ============================

func TestFabricCov_SWMLWebhooks_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.SWMLWebhooks.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data', keys=%v", keys(body))
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/swml_webhooks", "fabric.list_swml_webhooks")
}

func TestFabricCov_SWMLWebhooks_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_swml_webhooks", 500, map[string]any{"error": "boom"})
	_, err := client.Fabric.SWMLWebhooks.List(nil)
	e := fabAssertError(t, mock, err, 500, "fabric.list_swml_webhooks")
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SWMLWebhooks_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.SWMLWebhooks.Create(map[string]any{"name": "w"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := fabAssertSuccess(t, mock, "POST", "/api/fabric/resources/swml_webhooks", "fabric.create_swml_webhook")
	if b, _ := j.BodyMap(); b["name"] != "w" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_SWMLWebhooks_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.create_swml_webhook", 422, map[string]any{"error": "bad"})
	_, err := client.Fabric.SWMLWebhooks.Create(map[string]any{"x": 1})
	e := fabAssertError(t, mock, err, 422, "fabric.create_swml_webhook")
	if e.StatusCode != 422 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SWMLWebhooks_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.SWMLWebhooks.Get("w-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/swml_webhooks/w-1", "fabric.get_swml_webhook")
}

func TestFabricCov_SWMLWebhooks_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.get_swml_webhook", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.SWMLWebhooks.Get("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.get_swml_webhook")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SWMLWebhooks_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Fabric.SWMLWebhooks.Update("w-1", map[string]any{"name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := fabAssertSuccess(t, mock, "PATCH", "/api/fabric/resources/swml_webhooks/w-1", "fabric.update_swml_webhook")
	if b, _ := j.BodyMap(); b["name"] != "renamed" {
		t.Errorf("name = %v", b["name"])
	}
}

func TestFabricCov_SWMLWebhooks_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.update_swml_webhook", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.SWMLWebhooks.Update("missing", map[string]any{"name": "x"})
	e := fabAssertError(t, mock, err, 404, "fabric.update_swml_webhook")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestFabricCov_SWMLWebhooks_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.SWMLWebhooks.Delete("w-2")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Fatal("expected map")
	}
	fabAssertSuccess(t, mock, "DELETE", "/api/fabric/resources/swml_webhooks/w-2", "fabric.delete_swml_webhook")
}

func TestFabricCov_SWMLWebhooks_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.delete_swml_webhook", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.SWMLWebhooks.Delete("missing")
	e := fabAssertError(t, mock, err, 404, "fabric.delete_swml_webhook")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}
