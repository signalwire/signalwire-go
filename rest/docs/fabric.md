# Fabric Resources

The Fabric API (`/api/fabric`) manages all resource types in your SignalWire project. Every resource type supports CRUD operations and address listing.

The examples below import the resource-parameter structs from
`github.com/signalwire/signalwire-go/v3/pkg/rest/namespaces` and use a small helper
to set optional pointer fields:

<!-- snippet: no-compile illustrative API signature (reference only) -->
```go
func ptr[T any](v T) *T { return &v }
```

<!-- snippet-setup -->
```go
import (
	"context"

	"github.com/signalwire/signalwire-go/v3/pkg/rest"
	"github.com/signalwire/signalwire-go/v3/pkg/rest/namespaces"
)

// Shared context the fragments below assume: a constructed REST client and a
// phone-number SID for the binding examples. (The `ptr` helper above is
// illustrative; runnable fragments take the address of a local variable.)
var client, err = rest.NewRestClient("project", "token", "space")
var pnID = "pn-uuid"

var (
	_ = client
	_ = err
	_ = pnID
	_ = namespaces.Uuid("")
	_ = context.Background
)
```

## Standard CRUD Pattern

All 13 resource types share the same methods:

```go
// List all resources of this type
items, err := client.Fabric.AIAgents.List(context.Background(), nil)
items, err = client.Fabric.AIAgents.List(context.Background(), map[string]string{"page_number": "2", "page_size": "10"})

// Create a new resource
agent, err := client.Fabric.AIAgents.Create(context.Background(), map[string]any{
	"name":   "Support Bot",
	"prompt": map[string]any{"text": "You are a helpful support agent."},
})

// Get a resource by ID
agent, err = client.Fabric.AIAgents.Get(context.Background(), "agent-uuid")

// Update a resource
_, err = client.Fabric.AIAgents.Update(context.Background(), "agent-uuid", map[string]any{"name": "Updated Name"})

// Delete a resource
_, err = client.Fabric.AIAgents.Delete(context.Background(), "agent-uuid")

// List addresses assigned to this resource
addresses, err := client.Fabric.AIAgents.ListAddresses(context.Background(), "agent-uuid", nil)

_, _, _ = items, agent, addresses
```

## Resource Types

### PUT-Update Resources

These resources use `PUT` for updates (full replacement):

| Attribute | API Path |
|-----------|----------|
| `Fabric.SWMLScripts` | `/api/fabric/resources/swml_scripts` |
| `Fabric.RelayApplications` | `/api/fabric/resources/relay_applications` |
| `Fabric.CallFlows` | `/api/fabric/resources/call_flows` |
| `Fabric.ConferenceRooms` | `/api/fabric/resources/conference_rooms` |
| `Fabric.FreeSwitchConnectors` | `/api/fabric/resources/freeswitch_connectors` |
| `Fabric.Subscribers` | `/api/fabric/resources/subscribers` |
| `Fabric.SIPEndpoints` | `/api/fabric/resources/sip_endpoints` |
| `Fabric.CXMLScripts` | `/api/fabric/resources/cxml_scripts` |
| `Fabric.CXMLApplications` | `/api/fabric/resources/cxml_applications` |

### PATCH-Update Resources

These resources use `PATCH` for updates (partial update):

| Attribute | API Path |
|-----------|----------|
| `Fabric.SWMLWebhooks` | `/api/fabric/resources/swml_webhooks` |
| `Fabric.AIAgents` | `/api/fabric/resources/ai_agents` |
| `Fabric.SIPGateways` | `/api/fabric/resources/sip_gateways` |
| `Fabric.CXMLWebhooks` | `/api/fabric/resources/cxml_webhooks` |

## Call Flows -- Extra Methods

Call flows support version management:

```go
// List all versions of a call flow
versions, err := client.Fabric.CallFlows.ListVersions(context.Background(), "call-flow-uuid", nil)

// Deploy a new version
_, err = client.Fabric.CallFlows.DeployVersion(context.Background(), "call-flow-uuid", map[string]any{"document_version": 3})

_ = versions
```

## Subscribers -- SIP Endpoints

Subscribers have nested SIP endpoint management:

```go
// List subscriber's SIP endpoints
endpoints, err := client.Fabric.Subscribers.ListSIPEndpoints(context.Background(), "subscriber-uuid", nil)

// Create a SIP endpoint for a subscriber
callerId := "+15551234567"
endpoint, err := client.Fabric.Subscribers.CreateSIPEndpoint(context.Background(), 
	"subscriber-uuid",
	namespaces.SubscribersResourceCreateSIPEndpointParams{
		Username: "user1",
		Password: "secret",
		CallerID: &callerId,
	},
)

// Get a specific SIP endpoint
endpoint, err = client.Fabric.Subscribers.GetSIPEndpoint(context.Background(), "subscriber-uuid", "endpoint-uuid", nil)

// Update a SIP endpoint (uses PATCH)
newCallerId := "+15559876543"
_, err = client.Fabric.Subscribers.UpdateSIPEndpoint(context.Background(), 
	"subscriber-uuid", "endpoint-uuid",
	namespaces.SubscribersResourceUpdateSIPEndpointParams{
		CallerID: &newCallerId,
	},
)

// Delete a SIP endpoint
_, err = client.Fabric.Subscribers.DeleteSIPEndpoint(context.Background(), "subscriber-uuid", "endpoint-uuid")

_, _ = endpoints, endpoint
```

## cXML Applications

cXML applications support list/get/update/delete but not create:

```go
apps, err := client.Fabric.CXMLApplications.List(context.Background(), nil)
app, err := client.Fabric.CXMLApplications.Get(context.Background(), "app-uuid", nil)
voiceUrl := "https://example.com/voice"
_, err = client.Fabric.CXMLApplications.Update(context.Background(), "app-uuid", namespaces.CxmlApplicationsResourceUpdateParams{
	VoiceURL: &voiceUrl,
})
_, err = client.Fabric.CXMLApplications.Delete(context.Background(), "app-uuid")

// There is no Create method on CXMLApplications -- creation is not supported.

_, _ = apps, app
```

## Generic Resources

Operate on any resource type by ID:

```go
// List all resources across all types
allResources, err := client.Fabric.Resources.List(context.Background(), nil)

// Get any resource by ID
resource, err := client.Fabric.Resources.Get(context.Background(), "resource-uuid", nil)

// Delete any resource
_, err = client.Fabric.Resources.Delete(context.Background(), "resource-uuid")

// List addresses for any resource
addresses, err := client.Fabric.Resources.ListAddresses(context.Background(), "resource-uuid", nil)

// Assign a resource as a domain application handler
_, err = client.Fabric.Resources.AssignDomainApplication(context.Background(), 
	"resource-uuid",
	namespaces.GenericResourcesAssignDomainApplicationParams{DomainApplicationID: "da-uuid"},
)

_, _, _ = allResources, resource, addresses
```

> **Note:** `AssignPhoneRoute` is deprecated for the common binding cases.
> It applies only to a narrow set of legacy resource types and does NOT
> work for `swml_webhook`, `cxml_webhook`, or `ai_agent`. To bind a phone
> number to a webhook/agent/flow, configure the phone number directly via
> the `client.PhoneNumbers.Set*` helpers (see [Phone-number binding](#phone-number-binding) below).

## Phone-number binding

Routing an inbound phone number to a call handler (SWML webhook, cXML
webhook, AI agent, call flow, RELAY app/topic) is configured on the
**phone number**, not on a Fabric resource. Setting `call_handler` + the
handler-specific companion field on the phone number causes the server
to auto-materialize the matching Fabric resource.

Use the typed helpers on `client.PhoneNumbers`:

```go
// SWML webhook (your backend returns SWML per call)
_, err = client.PhoneNumbers.SetSwmlWebhook(context.Background(), pnID, "https://example.com/swml", nil)

// cXML / LAML webhook (Twilio-compat); optional fallback + status URLs
fallbackURL := "https://example.com/fallback.xml"
statusURL := "https://example.com/status"
_, err = client.PhoneNumbers.SetCxmlWebhook(context.Background(), pnID, "https://example.com/voice.xml", &fallbackURL, &statusURL, nil)

// Existing cXML application by ID
_, err = client.PhoneNumbers.SetCxmlApplication(context.Background(), pnID, "app-uuid", nil)

// AI Agent by ID
_, err = client.PhoneNumbers.SetAiAgent(context.Background(), pnID, "agent-uuid", nil)

// Call flow (optionally pin a version)
version := "current_deployed"
_, err = client.PhoneNumbers.SetCallFlow(context.Background(), pnID, "flow-uuid", &version, nil)

// Relay application (named routing)
_, err = client.PhoneNumbers.SetRelayApplication(context.Background(), pnID, "my-relay-app", nil)

// Relay topic (RELAY client subscription)
_, err = client.PhoneNumbers.SetRelayTopic(context.Background(), pnID, "office", nil, nil)
```

The `namespaces.PhoneCallHandler` type exposes the full enum of wire values
(`PhoneCallHandlerRelayScript`, `PhoneCallHandlerLamlWebhooks`, `…AiAgent`,
etc.). For unusual combinations, call `client.PhoneNumbers.Update` directly
with `call_handler` + the companion field in the body.

**Do not** call `client.Fabric.SWMLWebhooks.Create` or
`client.Fabric.CXMLWebhooks.Create` as a binding primitive — those produce
orphan resources. The corresponding list/get/update/delete operations
remain unchanged.

## Fabric Addresses

Read-only access to all fabric addresses:

```go
// List all addresses (filter by type or display_name)
addresses, err := client.Fabric.Addresses.List(context.Background(), map[string]string{"type": "room"})

// Get a specific address
address, err := client.Fabric.Addresses.Get(context.Background(), "address-uuid")

_, _ = addresses, address
```

## Tokens

Create tokens for subscribers, guests, invites, and embeds. Fields whose types
are ID/JWT aliases (`address_id`, `allowed_addresses`, `refresh_token`) are set
through the `Extras` map:

```go
// Subscriber token
password := "secret"
token, err := client.Fabric.Tokens.CreateSubscriberToken(context.Background(), namespaces.FabricTokensCreateSubscriberTokenParams{
	Reference: "user@example.com",
	Password:  &password,
})

// Refresh a subscriber token
refreshed, err := client.Fabric.Tokens.RefreshSubscriberToken(context.Background(), namespaces.FabricTokensRefreshSubscriberTokenParams{
	Extras: map[string]any{"refresh_token": "existing-refresh-token"},
})

// Guest token
guestToken, err := client.Fabric.Tokens.CreateGuestToken(context.Background(), namespaces.FabricTokensCreateGuestTokenParams{
	Extras: map[string]any{
		"allowed_addresses": []string{"address-uuid-1", "address-uuid-2"},
	},
})

// Subscriber invite token
inviteToken, err := client.Fabric.Tokens.CreateInviteToken(context.Background(), namespaces.FabricTokensCreateInviteTokenParams{
	Extras: map[string]any{"address_id": "address-uuid"},
})

// Click-to-call embed token
embedToken, err := client.Fabric.Tokens.CreateEmbedToken(context.Background(), namespaces.FabricTokensCreateEmbedTokenParams{
	Token: "embed-source-token",
})

_, _, _, _, _ = token, refreshed, guestToken, inviteToken, embedToken
```
