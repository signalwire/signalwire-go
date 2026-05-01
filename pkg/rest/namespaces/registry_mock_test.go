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
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

const regBase = "/api/relay/rest/registry/beta"

// ---------- Brands ----------

func TestRegistryBrands_List_ReturnsDict(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Registry.Brands.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
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
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Registry.Brands.Get("brand-77")
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
	if j.Path != regBase+"/brands/brand-77" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestRegistryBrands_ListCampaigns_UsesBrandSubpath(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Registry.Brands.ListCampaigns("brand-1", nil)
	if err != nil {
		t.Fatalf("ListCampaigns: %v", err)
	}
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
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Registry.Brands.CreateCampaign("brand-2", map[string]any{
		"usecase":     "LOW_VOLUME",
		"description": "MFA",
	})
	if err != nil {
		t.Fatalf("CreateCampaign: %v", err)
	}
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
	if sent["usecase"] != "LOW_VOLUME" {
		t.Errorf("usecase = %v", sent["usecase"])
	}
	if sent["description"] != "MFA" {
		t.Errorf("description = %v", sent["description"])
	}
}

// ---------- Campaigns ----------

func TestRegistryCampaigns_Get_UsesIDInPath(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Registry.Campaigns.Get("camp-1")
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
	if j.Path != regBase+"/campaigns/camp-1" {
		t.Errorf("path = %q", j.Path)
	}
}

func TestRegistryCampaigns_Update_UsesPut(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Registry.Campaigns.Update("camp-2", map[string]any{
		"description": "Updated",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
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
	if sent["description"] != "Updated" {
		t.Errorf("description = %v", sent["description"])
	}
}

func TestRegistryCampaigns_ListNumbers_UsesNumbersSubpath(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Registry.Campaigns.ListNumbers("camp-3", nil)
	if err != nil {
		t.Fatalf("ListNumbers: %v", err)
	}
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
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Registry.Campaigns.CreateOrder("camp-4", map[string]any{
		"numbers": []string{"pn-1", "pn-2"},
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
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
	rawNumbers, ok := sent["numbers"].([]any)
	if !ok {
		t.Fatalf("numbers type = %T", sent["numbers"])
	}
	if len(rawNumbers) != 2 || rawNumbers[0] != "pn-1" || rawNumbers[1] != "pn-2" {
		t.Errorf("numbers = %v, want [pn-1 pn-2]", rawNumbers)
	}
}

// ---------- Orders ----------

func TestRegistryOrders_Get_UsesIDInPath(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Registry.Orders.Get("order-1")
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
	if j.Path != regBase+"/orders/order-1" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil — spec gap: order retrieve")
	}
}

// ---------- Numbers ----------

func TestRegistryNumbers_Delete_UsesIDInPath(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	body, err := client.Registry.Numbers.Delete("num-1")
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
	if j.Path != regBase+"/numbers/num-1" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil {
		t.Error("matched_route is nil")
	}
}

var _ = mocktest.JournalEntry{}
