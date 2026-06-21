// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Success+error REST coverage for the eight fabric `*_addresses` sub-routes
// that became reachable once FabricResourcePUT, AutoMaterializedWebhookResource
// and SubscribersResource were given ListAddresses (via the CrudWithAddresses
// embed) to match Python's FabricResourcePUT(CrudWithAddresses).
//
// Each route is GET {base}/{id}/addresses and journals matched_route
// fabric.list_<type>_addresses. The helpers fabAssertSuccess / fabAssertError
// live in fabric_coverage_mock_test.go (same package).
package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// ---------- cxml_scripts ----------

func TestFabricAddrCov_CXMLScripts_ListAddresses(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CXMLScripts.ListAddresses("cs-1", nil)
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data'")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/cxml_scripts/cs-1/addresses", "fabric.list_cxml_script_addresses")
}

func TestFabricAddrCov_CXMLScripts_ListAddresses_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_cxml_script_addresses", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CXMLScripts.ListAddresses("missing", nil)
	e := fabAssertError(t, mock, err, 404, "fabric.list_cxml_script_addresses")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ---------- cxml_webhooks ----------

func TestFabricAddrCov_CXMLWebhooks_ListAddresses(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.CXMLWebhooks.ListAddresses("cw-1", nil)
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data'")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/cxml_webhooks/cw-1/addresses", "fabric.list_cxml_webhook_addresses")
}

func TestFabricAddrCov_CXMLWebhooks_ListAddresses_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_cxml_webhook_addresses", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.CXMLWebhooks.ListAddresses("missing", nil)
	e := fabAssertError(t, mock, err, 404, "fabric.list_cxml_webhook_addresses")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ---------- freeswitch_connectors ----------

func TestFabricAddrCov_FreeSwitchConnectors_ListAddresses(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.FreeSwitchConnectors.ListAddresses("fc-1", nil)
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data'")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/freeswitch_connectors/fc-1/addresses", "fabric.list_freeswitch_connector_addresses")
}

func TestFabricAddrCov_FreeSwitchConnectors_ListAddresses_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_freeswitch_connector_addresses", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.FreeSwitchConnectors.ListAddresses("missing", nil)
	e := fabAssertError(t, mock, err, 404, "fabric.list_freeswitch_connector_addresses")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ---------- relay_applications ----------

func TestFabricAddrCov_RelayApplications_ListAddresses(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.RelayApplications.ListAddresses("ra-1", nil)
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data'")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/relay_applications/ra-1/addresses", "fabric.list_relay_application_addresses")
}

func TestFabricAddrCov_RelayApplications_ListAddresses_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_relay_application_addresses", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.RelayApplications.ListAddresses("missing", nil)
	e := fabAssertError(t, mock, err, 404, "fabric.list_relay_application_addresses")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ---------- sip_endpoints ----------

func TestFabricAddrCov_SIPEndpoints_ListAddresses(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.SIPEndpoints.ListAddresses("se-1", nil)
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data'")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/sip_endpoints/se-1/addresses", "fabric.list_sip_endpoint_addresses")
}

func TestFabricAddrCov_SIPEndpoints_ListAddresses_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_sip_endpoint_addresses", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.SIPEndpoints.ListAddresses("missing", nil)
	e := fabAssertError(t, mock, err, 404, "fabric.list_sip_endpoint_addresses")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ---------- swml_scripts ----------

func TestFabricAddrCov_SWMLScripts_ListAddresses(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.SWMLScripts.ListAddresses("ss-1", nil)
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data'")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/swml_scripts/ss-1/addresses", "fabric.list_swml_script_addresses")
}

func TestFabricAddrCov_SWMLScripts_ListAddresses_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_swml_script_addresses", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.SWMLScripts.ListAddresses("missing", nil)
	e := fabAssertError(t, mock, err, 404, "fabric.list_swml_script_addresses")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ---------- swml_webhooks ----------

func TestFabricAddrCov_SWMLWebhooks_ListAddresses(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.SWMLWebhooks.ListAddresses("sw-1", nil)
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data'")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/swml_webhooks/sw-1/addresses", "fabric.list_swml_webhook_addresses")
}

func TestFabricAddrCov_SWMLWebhooks_ListAddresses_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_swml_webhook_addresses", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.SWMLWebhooks.ListAddresses("missing", nil)
	e := fabAssertError(t, mock, err, 404, "fabric.list_swml_webhook_addresses")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

// ---------- subscribers ----------

func TestFabricAddrCov_Subscribers_ListAddresses(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Fabric.Subscribers.ListAddresses("sub-1", nil)
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("missing 'data'")
	}
	fabAssertSuccess(t, mock, "GET", "/api/fabric/resources/subscribers/sub-1/addresses", "fabric.list_subscriber_addresses")
}

func TestFabricAddrCov_Subscribers_ListAddresses_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "fabric.list_subscriber_addresses", 404, map[string]any{"error": "not found"})
	_, err := client.Fabric.Subscribers.ListAddresses("missing", nil)
	e := fabAssertError(t, mock, err, 404, "fabric.list_subscriber_addresses")
	if e.StatusCode != 404 {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}
