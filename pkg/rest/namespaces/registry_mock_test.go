// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_registry_mock.py.
//
// The 10DLC Campaign Registry namespace exposes four sub-resources:
// brands, campaigns, orders, and numbers.

package namespaces_test

import (
	"context"
	"testing"

	"github.com/signalwire/signalwire-go/v3/pkg/rest/internal/mocktest"
	"github.com/signalwire/signalwire-go/v3/pkg/rest/namespaces"
)

const regBase = "/api/relay/rest/registry/beta"

// ---------- Brands ----------

func TestRegistryBrands_List_ReturnsDict(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	bodyResp, err := client.Registry.Brands.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	body := respMap(t, bodyResp)
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != regBase+"/brands" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil — spec gap: brand list")
	}
}

func TestRegistryBrands_Get_UsesIDInPath(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	bodyResp, err := client.Registry.Brands.Get(context.Background(), "brand-77", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	body := respMap(t, bodyResp)
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != regBase+"/brands/brand-77" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestRegistryBrands_ListCampaigns_UsesBrandSubpath(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	bodyResp, err := client.Registry.Brands.ListCampaigns(context.Background(), "brand-1", nil)
	if err != nil {
		t.Fatalf("ListCampaigns: %v", err)
	}
	body := respMap(t, bodyResp)
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != regBase+"/brands/brand-1/campaigns" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil")
	}
}

func TestRegistryBrands_CreateCampaign_PostsToBrandSubpath(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	bodyResp, err := client.Registry.Brands.CreateCampaign(context.Background(), "brand-2", map[string]any{
		"name":                    "My Campaign",
		"brand_id":                "3fa85f64-5717-4562-b3fc-2c963f66afa6",
		"sms_use_case":            "LOW_VOLUME",
		"description":             "MFA",
		"sample1":                 "Hi John, your appointment is tomorrow. Reply STOP to unsubscribe.",
		"sample2":                 "Your prescription is ready for pickup. Reply STOP to unsubscribe.",
		"message_flow":            "Users opt in via a written form and receive an opt-in message.",
		"opt_out_message":         "You have successfully been opted out. Reply START to opt back in.",
		"help_message":            "For help contact support@example.com. Reply STOP to unsubscribe.",
		"number_pooling_required": false,
		"direct_lending":          false,
		"embedded_link":           false,
		"embedded_phone":          false,
		"age_gated_content":       false,
		"lead_generation":         false,
		"terms_and_conditions":    true,
	})
	if err != nil {
		t.Fatalf("CreateCampaign: %v", err)
	}
	body := respMap(t, bodyResp)
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != regBase+"/brands/brand-2/campaigns" {
		t.Errorf("path = %q", j.Path)
	}
	sent, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if sent["sms_use_case"] != "LOW_VOLUME" {
		t.Errorf("sms_use_case = %v", sent["sms_use_case"])
	}
	if sent["description"] != "MFA" {
		t.Errorf("description = %v", sent["description"])
	}
}

// ---------- Campaigns ----------

func TestRegistryCampaigns_Get_UsesIDInPath(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	bodyResp, err := client.Registry.Campaigns.Get(context.Background(), "camp-1", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	body := respMap(t, bodyResp)
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != regBase+"/campaigns/camp-1" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestRegistryCampaigns_Update_UsesPut(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	bodyResp, err := client.Registry.Campaigns.Update(context.Background(), "camp-2", namespaces.RegistryCampaignsUpdateParams{Extras: map[string]any{
		"name": "Updated Campaign",
	}})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	body := respMap(t, bodyResp)
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "PUT" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != regBase+"/campaigns/camp-2" {
		t.Errorf("path = %q", j.Path)
	}
	sent, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if sent["name"] != "Updated Campaign" {
		t.Errorf("name = %v", sent["name"])
	}
}

func TestRegistryCampaigns_ListNumbers_UsesNumbersSubpath(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	bodyResp, err := client.Registry.Campaigns.ListNumbers(context.Background(), "camp-3", nil)
	if err != nil {
		t.Fatalf("ListNumbers: %v", err)
	}
	body := respMap(t, bodyResp)
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != regBase+"/campaigns/camp-3/numbers" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil")
	}
}

func TestRegistryCampaigns_CreateOrder_PostsToOrdersSubpath(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	bodyResp, err := client.Registry.Campaigns.CreateOrder(context.Background(), "camp-4", namespaces.RegistryCampaignsCreateOrderParams{
		PhoneNumbers: []string{"pn-1", "pn-2"},
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	body := respMap(t, bodyResp)
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != regBase+"/campaigns/camp-4/orders" {
		t.Errorf("path = %q", j.Path)
	}
	sent, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	rawNumbers, ok := sent["phone_numbers"].([]any)
	if !ok {
		t.Fatalf("phone_numbers type = %T", sent["phone_numbers"])
	}
	if len(rawNumbers) != 2 || rawNumbers[0] != "pn-1" || rawNumbers[1] != "pn-2" {
		t.Errorf("phone_numbers = %v, want [pn-1 pn-2]", rawNumbers)
	}
}

// ---------- Orders ----------

func TestRegistryOrders_Get_UsesIDInPath(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	bodyResp, err := client.Registry.Orders.Get(context.Background(), "order-1", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	body := respMap(t, bodyResp)
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != regBase+"/orders/order-1" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil — spec gap: order retrieve")
	}
}

// ---------- Numbers ----------

func TestRegistryNumbers_Delete_UsesIDInPath(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	bodyResp, err := client.Registry.Numbers.Delete(context.Background(), "num-1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	body := respMap(t, bodyResp)
	if body == nil {
		t.Error("expected map, got nil")
	}

	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q", j.Method)
	}
	if j.Path != regBase+"/numbers/num-1" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil")
	}
}

var _ = mocktest.JournalEntry{}
