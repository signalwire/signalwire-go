// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

import "strings"

// ---------- CallFlowsResource ----------

// CallFlowsResource extends CrudResource with version management and a
// singular sub-resource path convention.
type CallFlowsResource struct {
	*CrudResource
}

// ListAddresses lists addresses for a call flow (uses singular "call_flow" path).
func (r *CallFlowsResource) ListAddresses(id string, params map[string]string) (map[string]any, error) {
	path := strings.Replace(r.Base, "/call_flows", "/call_flow", 1)
	return r.HTTP.Get(path+"/"+id+"/addresses", params)
}

// ListVersions lists versions of a call flow.
func (r *CallFlowsResource) ListVersions(id string, params map[string]string) (map[string]any, error) {
	path := strings.Replace(r.Base, "/call_flows", "/call_flow", 1)
	return r.HTTP.Get(path+"/"+id+"/versions", params)
}

// DeployVersion deploys a new version of a call flow.
func (r *CallFlowsResource) DeployVersion(id string, data map[string]any) (map[string]any, error) {
	path := strings.Replace(r.Base, "/call_flows", "/call_flow", 1)
	return r.HTTP.Post(path+"/"+id+"/versions", data)
}

// ---------- ConferenceRoomsResource ----------

// ConferenceRoomsResource uses singular "conference_room" for sub-resource paths.
type ConferenceRoomsResource struct {
	*CrudResource
}

// ListAddresses lists addresses for a conference room.
func (r *ConferenceRoomsResource) ListAddresses(id string, params map[string]string) (map[string]any, error) {
	path := strings.Replace(r.Base, "/conference_rooms", "/conference_room", 1)
	return r.HTTP.Get(path+"/"+id+"/addresses", params)
}

// ---------- SubscribersResource ----------

// SubscribersResource extends CrudResource with SIP endpoint management.
type SubscribersResource struct {
	*CrudResource
}

// ListSIPEndpoints lists SIP endpoints for a subscriber.
func (r *SubscribersResource) ListSIPEndpoints(subscriberID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(subscriberID, "sip_endpoints"), params)
}

// CreateSIPEndpoint creates a SIP endpoint for a subscriber.
func (r *SubscribersResource) CreateSIPEndpoint(subscriberID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(subscriberID, "sip_endpoints"), data)
}

// GetSIPEndpoint retrieves a SIP endpoint for a subscriber.
func (r *SubscribersResource) GetSIPEndpoint(subscriberID, endpointID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(subscriberID, "sip_endpoints", endpointID), nil)
}

// UpdateSIPEndpoint updates a SIP endpoint for a subscriber.
func (r *SubscribersResource) UpdateSIPEndpoint(subscriberID, endpointID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Patch(r.Path(subscriberID, "sip_endpoints", endpointID), data)
}

// DeleteSIPEndpoint deletes a SIP endpoint from a subscriber.
func (r *SubscribersResource) DeleteSIPEndpoint(subscriberID, endpointID string) error {
	return r.HTTP.Delete(r.Path(subscriberID, "sip_endpoints", endpointID))
}

// ---------- GenericResources ----------

// GenericResources provides operations across all fabric resource types.
type GenericResources struct {
	Resource
}

// List lists all generic resources.
func (r *GenericResources) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Get retrieves a generic resource by ID.
func (r *GenericResources) Get(id string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(id), nil)
}

// Delete removes a generic resource by ID.
func (r *GenericResources) Delete(id string) error {
	return r.HTTP.Delete(r.Path(id))
}

// ListAddresses lists addresses for a generic resource.
func (r *GenericResources) ListAddresses(id string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(id, "addresses"), params)
}

// AssignPhoneRoute assigns a phone route to a resource.
func (r *GenericResources) AssignPhoneRoute(id string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(id, "phone_routes"), data)
}

// AssignDomainApplication assigns a domain application to a resource.
func (r *GenericResources) AssignDomainApplication(id string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(id, "domain_applications"), data)
}

// ---------- FabricAddresses ----------

// FabricAddresses provides read-only access to fabric addresses.
type FabricAddresses struct {
	Resource
}

// List lists all fabric addresses.
func (r *FabricAddresses) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Get retrieves a fabric address by ID.
func (r *FabricAddresses) Get(id string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(id), nil)
}

// ---------- FabricTokens ----------

// FabricTokens provides subscriber, guest, invite, and embed token creation.
type FabricTokens struct {
	Resource
}

// CreateSubscriberToken creates a subscriber token.
func (r *FabricTokens) CreateSubscriberToken(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path("subscribers", "tokens"), data)
}

// RefreshSubscriberToken refreshes a subscriber token.
func (r *FabricTokens) RefreshSubscriberToken(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path("subscribers", "tokens", "refresh"), data)
}

// CreateInviteToken creates an invite token.
func (r *FabricTokens) CreateInviteToken(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path("subscriber", "invites"), data)
}

// CreateGuestToken creates a guest token.
func (r *FabricTokens) CreateGuestToken(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path("guests", "tokens"), data)
}

// CreateEmbedToken creates an embed token.
func (r *FabricTokens) CreateEmbedToken(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path("embeds", "tokens"), data)
}

// ---------- FabricNamespace ----------

// FabricNamespace groups all Fabric API resource types.
type FabricNamespace struct {
	// PUT-update resources
	SWMLScripts           *CrudResource
	RelayApplications     *CrudResource
	CallFlows             *CallFlowsResource
	ConferenceRooms       *ConferenceRoomsResource
	FreeSwitchConnectors  *CrudResource
	Subscribers           *SubscribersResource
	SIPEndpoints          *CrudResource
	CXMLScripts           *CrudResource
	CXMLApplications      *CrudResource

	// PATCH-update resources
	SWMLWebhooks *CrudResource
	AIAgents     *CrudResource
	SIPGateways  *CrudResource
	CXMLWebhooks *CrudResource

	// Special resources
	Resources *GenericResources
	Addresses *FabricAddresses
	Tokens    *FabricTokens
}

// NewFabricNamespace creates a new FabricNamespace with all sub-resources initialized.
func NewFabricNamespace(client HTTPClient) *FabricNamespace {
	base := "/api/fabric/resources"

	return &FabricNamespace{
		// PUT-update resources
		SWMLScripts:          NewCrudResourcePUT(client, base+"/swml_scripts"),
		RelayApplications:    NewCrudResourcePUT(client, base+"/relay_applications"),
		CallFlows:            &CallFlowsResource{NewCrudResourcePUT(client, base+"/call_flows")},
		ConferenceRooms:      &ConferenceRoomsResource{NewCrudResourcePUT(client, base+"/conference_rooms")},
		FreeSwitchConnectors: NewCrudResourcePUT(client, base+"/freeswitch_connectors"),
		Subscribers:          &SubscribersResource{NewCrudResourcePUT(client, base+"/subscribers")},
		SIPEndpoints:         NewCrudResourcePUT(client, base+"/sip_endpoints"),
		CXMLScripts:          NewCrudResourcePUT(client, base+"/cxml_scripts"),
		CXMLApplications:     NewCrudResourcePUT(client, base+"/cxml_applications"),

		// PATCH-update resources
		SWMLWebhooks: NewCrudResource(client, base+"/swml_webhooks"),
		AIAgents:     NewCrudResource(client, base+"/ai_agents"),
		SIPGateways:  NewCrudResource(client, base+"/sip_gateways"),
		CXMLWebhooks: NewCrudResource(client, base+"/cxml_webhooks"),

		// Special resources
		Resources: &GenericResources{Resource{HTTP: client, Base: base}},
		Addresses: &FabricAddresses{Resource{HTTP: client, Base: "/api/fabric/addresses"}},
		Tokens:    &FabricTokens{Resource{HTTP: client, Base: "/api/fabric"}},
	}
}
