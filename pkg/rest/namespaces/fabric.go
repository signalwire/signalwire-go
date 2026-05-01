// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

import (
	"fmt"
	"strings"
	"sync"

	"github.com/signalwire/signalwire-go/pkg/logging"
)

// deprecationLogger carries the runtime warnings emitted by deprecated
// methods. It is a package-level logger so tests can swap in a sink via
// SetDeprecationLogger.
var (
	deprecationLoggerMu sync.RWMutex
	deprecationLogger   = logging.New("rest_deprecation")

	// warnOnce tracks which deprecation messages have already been emitted
	// so a long-running program doesn't spam identical warnings. Keyed by
	// a stable id per call site (not by caller identity).
	warnOnceMu sync.Mutex
	warnOnce   = map[string]bool{}
)

// emitDeprecationWarning logs a deprecation warning at most once per key.
// The key scopes the "once" guarantee (e.g. "AssignPhoneRoute"), so each
// deprecated method fires once per process lifetime regardless of how many
// times it's called.
func emitDeprecationWarning(key, msg string) {
	warnOnceMu.Lock()
	if warnOnce[key] {
		warnOnceMu.Unlock()
		return
	}
	warnOnce[key] = true
	warnOnceMu.Unlock()

	deprecationLoggerMu.RLock()
	logger := deprecationLogger
	deprecationLoggerMu.RUnlock()
	logger.Warn("%s", msg)
}

// SetDeprecationLogger replaces the package-level deprecation logger. The
// previous logger is returned so tests can restore it. Passing nil is a
// no-op.
func SetDeprecationLogger(l *logging.Logger) *logging.Logger {
	if l == nil {
		return nil
	}
	deprecationLoggerMu.Lock()
	defer deprecationLoggerMu.Unlock()
	prev := deprecationLogger
	deprecationLogger = l
	return prev
}

// ResetDeprecationWarnOnce clears the "once" tracking set so deprecation
// warnings fire again. Test-only helper.
func ResetDeprecationWarnOnce() {
	warnOnceMu.Lock()
	defer warnOnceMu.Unlock()
	warnOnce = map[string]bool{}
}

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
	return r.HTTP.Post(path+"/"+id+"/versions", data, nil)
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
	return r.HTTP.Post(r.Path(subscriberID, "sip_endpoints"), data, nil)
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
func (r *SubscribersResource) DeleteSIPEndpoint(subscriberID, endpointID string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(subscriberID, "sip_endpoints", endpointID))
}

// ---------- CxmlApplicationsResource ----------

// CxmlApplicationsResource exposes the fabric cXML applications sub-resource.
// Create is explicitly disallowed — cXML applications cannot be created via
// this API. This mirrors Python's CxmlApplicationsResource.create raising
// NotImplementedError (fabric.py:90).
type CxmlApplicationsResource struct {
	*CrudResource
}

// Create always returns an error — cXML applications cannot be created
// via this API. The params argument is accepted for API parity with other
// CRUD resources but is reported in the error so the caller can see what
// payload was rejected. To create a new cXML application use a different
// API surface or the SignalWire dashboard.
//
// Mirrors Python's CxmlApplicationsResource.create raising
// NotImplementedError (signalwire/rest/namespaces/fabric.py:90).
func (r *CxmlApplicationsResource) Create(params map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf(
		"cXML applications cannot be created via this API (received %d field(s); use the SignalWire dashboard)",
		len(params),
	)
}

// ListAddresses lists addresses for a cXML application.
func (r *CxmlApplicationsResource) ListAddresses(id string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(id, "addresses"), params)
}

// ---------- AutoMaterializedWebhook resources ----------

// AutoMaterializedWebhookResource is a Fabric webhook resource that is
// normally auto-created by the phone_numbers.Set*Webhook helpers. Exposed
// for backwards compatibility: list/get/update/delete work as usual, but
// Create now emits a deprecation warning because creating a webhook
// resource directly produces an orphan that isn't bound to any phone
// number.
type AutoMaterializedWebhookResource struct {
	*CrudResource
	// helperName is the recommended replacement helper, inserted into the
	// deprecation message. E.g. "phone_numbers.SetSwmlWebhook(sid, url)".
	helperName string
	// deprecationKey scopes the "once" guarantee for this resource's
	// Create deprecation warning.
	deprecationKey string
}

// Create sends a POST to create a new webhook resource.
//
// Deprecated: Creating a webhook Fabric resource directly produces an orphan
// not bound to any phone number. Use phone_numbers.SetSwmlWebhook or
// phone_numbers.SetCxmlWebhook instead; setting call_handler on the phone
// number causes the server to auto-materialize the webhook resource. See
// porting-sdk's phone-binding.md.
func (r *AutoMaterializedWebhookResource) Create(data map[string]any) (map[string]any, error) {
	emitDeprecationWarning(
		r.deprecationKey,
		"Creating a webhook Fabric resource directly produces an orphan not "+
			"bound to any phone number. Use "+r.helperName+" instead; it "+
			"updates the phone number and the server auto-materializes the "+
			"resource. See porting-sdk's phone-binding.md.",
	)
	return r.CrudResource.Create(data)
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
func (r *GenericResources) Delete(id string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(id))
}

// ListAddresses lists addresses for a generic resource.
func (r *GenericResources) ListAddresses(id string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(id, "addresses"), params)
}

// AssignPhoneRoute assigns a phone route to a resource.
//
// Deprecated: This endpoint (POST /api/fabric/resources/{id}/phone_routes)
// accepts only a narrow set of legacy resource types as the attach target.
// It does NOT work for swml_webhook / cxml_webhook / ai_agent bindings —
// those are configured on the phone number and the Fabric resource is
// auto-materialized. Use phone_numbers.SetSwmlWebhook, SetCxmlWebhook,
// SetAiAgent, etc. instead. See porting-sdk's phone-binding.md.
func (r *GenericResources) AssignPhoneRoute(id string, data map[string]any) (map[string]any, error) {
	emitDeprecationWarning(
		"GenericResources.AssignPhoneRoute",
		"AssignPhoneRoute does not bind phone numbers to "+
			"swml_webhook/cxml_webhook/ai_agent resources — those are "+
			"configured via phone_numbers.SetSwmlWebhook / SetCxmlWebhook "+
			"/ SetAiAgent. This method applies only to a narrow set of "+
			"legacy resource types. See porting-sdk's phone-binding.md.",
	)
	return r.HTTP.Post(r.Path(id, "phone_routes"), data, nil)
}

// AssignDomainApplication assigns a domain application to a resource.
func (r *GenericResources) AssignDomainApplication(id string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(id, "domain_applications"), data, nil)
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
	return r.HTTP.Post(r.Path("subscribers", "tokens"), data, nil)
}

// RefreshSubscriberToken refreshes a subscriber token.
func (r *FabricTokens) RefreshSubscriberToken(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path("subscribers", "tokens", "refresh"), data, nil)
}

// CreateInviteToken creates an invite token.
func (r *FabricTokens) CreateInviteToken(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path("subscriber", "invites"), data, nil)
}

// CreateGuestToken creates a guest token.
func (r *FabricTokens) CreateGuestToken(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path("guests", "tokens"), data, nil)
}

// CreateEmbedToken creates an embed token.
func (r *FabricTokens) CreateEmbedToken(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path("embeds", "tokens"), data, nil)
}

// FabricResourcePUT is the Python class name for a CrudResource that uses
// PUT for updates. Go aliases CrudResource here so the cross-language
// audit sees the same type name on both sides without requiring a
// distinct struct.
type FabricResourcePUT = CrudResource

// FabricResource is the Python class name for a CrudResource that exposes
// the addresses sub-resource. Go aliases CrudWithAddresses here for the
// same reason as FabricResourcePUT.
type FabricResource = CrudWithAddresses

// SwmlWebhooksResource is the Python class name for the auto-materialized
// SWML webhook resource. Go aliases AutoMaterializedWebhookResource here.
type SwmlWebhooksResource = AutoMaterializedWebhookResource

// CxmlWebhooksResource is the Python class name for the auto-materialized
// CXML webhook resource. Go aliases AutoMaterializedWebhookResource here.
type CxmlWebhooksResource = AutoMaterializedWebhookResource

// ---------- FabricNamespace ----------

// FabricNamespace groups all Fabric API resource types.
type FabricNamespace struct {
	// PUT-update resources
	SWMLScripts          *FabricResourcePUT
	RelayApplications    *FabricResourcePUT
	CallFlows            *CallFlowsResource
	ConferenceRooms      *ConferenceRoomsResource
	FreeSwitchConnectors *FabricResourcePUT
	Subscribers          *SubscribersResource
	SIPEndpoints         *FabricResourcePUT
	CXMLScripts          *FabricResourcePUT
	CXMLApplications     *CxmlApplicationsResource

	// PATCH-update resources
	//
	// SWMLWebhooks and CXMLWebhooks are auto-materialized: prefer
	// PhoneNumbers.SetSwmlWebhook / SetCxmlWebhook for creation. Direct
	// .Create still works for backcompat but emits a deprecation warning.
	SWMLWebhooks *SwmlWebhooksResource
	AIAgents     *FabricResource
	SIPGateways  *FabricResource
	CXMLWebhooks *CxmlWebhooksResource

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
		CXMLApplications:     &CxmlApplicationsResource{NewCrudResourcePUT(client, base+"/cxml_applications")},

		// PATCH-update resources
		SWMLWebhooks: &AutoMaterializedWebhookResource{
			CrudResource:   NewCrudResource(client, base+"/swml_webhooks"),
			helperName:     "phone_numbers.SetSwmlWebhook(sid, url)",
			deprecationKey: "SWMLWebhooks.Create",
		},
		AIAgents:    NewCrudWithAddresses(client, base+"/ai_agents"),
		SIPGateways: NewCrudWithAddresses(client, base+"/sip_gateways"),
		CXMLWebhooks: &AutoMaterializedWebhookResource{
			CrudResource:   NewCrudResource(client, base+"/cxml_webhooks"),
			helperName:     "phone_numbers.SetCxmlWebhook(sid, url, opts)",
			deprecationKey: "CXMLWebhooks.Create",
		},

		// Special resources
		Resources: &GenericResources{Resource{HTTP: client, Base: base}},
		Addresses: &FabricAddresses{Resource{HTTP: client, Base: "/api/fabric/addresses"}},
		Tokens:    &FabricTokens{Resource{HTTP: client, Base: "/api/fabric"}},
	}
}
