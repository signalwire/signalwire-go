# Fabric Resources

The Fabric API (`/api/fabric`) manages all resource types in your SignalWire project. Every resource type supports CRUD operations and address listing.

## Standard CRUD Pattern

All 13 resource types share the same methods:

```go
// List all resources of this type
items, _ := client.Fabric.AIAgents.List(nil)
items, _ = client.Fabric.AIAgents.List(map[string]string{"page": "2", "page_size": "10"})

// Create a new resource
agent, _ := client.Fabric.AIAgents.Create(map[string]any{
	"name":   "Support Bot",
	"prompt": map[string]any{"text": "You are a helpful support agent."},
})

// Get a resource by ID
agent, _ = client.Fabric.AIAgents.Get("agent-uuid")

// Update a resource
client.Fabric.AIAgents.Update("agent-uuid", map[string]any{"name": "Updated Name"})

// Delete a resource
client.Fabric.AIAgents.Delete("agent-uuid")

// List addresses assigned to this resource
addresses, _ := client.Fabric.AIAgents.ListAddresses("agent-uuid")
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
| `Fabric.FreeSWITCHConnectors` | `/api/fabric/resources/freeswitch_connectors` |
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
versions, _ := client.Fabric.CallFlows.ListVersions("call-flow-uuid")

// Deploy a new version
client.Fabric.CallFlows.DeployVersion("call-flow-uuid", 3)
```

## Subscribers -- SIP Endpoints

Subscribers have nested SIP endpoint management:

```go
// List subscriber's SIP endpoints
endpoints, _ := client.Fabric.Subscribers.ListSIPEndpoints("subscriber-uuid")

// Create a SIP endpoint for a subscriber
endpoint, _ := client.Fabric.Subscribers.CreateSIPEndpoint("subscriber-uuid", map[string]any{
	"username":  "user1",
	"password":  "secret",
	"caller_id": "+15551234567",
})

// Get a specific SIP endpoint
endpoint, _ = client.Fabric.Subscribers.GetSIPEndpoint("subscriber-uuid", "endpoint-uuid")

// Update a SIP endpoint (uses PATCH)
client.Fabric.Subscribers.UpdateSIPEndpoint("subscriber-uuid", "endpoint-uuid", map[string]any{
	"caller_id": "+15559876543",
})

// Delete a SIP endpoint
client.Fabric.Subscribers.DeleteSIPEndpoint("subscriber-uuid", "endpoint-uuid")
```

## cXML Applications

cXML applications support list/get/update/delete but not create:

```go
apps, _ := client.Fabric.CXMLApplications.List(nil)
app, _ := client.Fabric.CXMLApplications.Get("app-uuid")
client.Fabric.CXMLApplications.Update("app-uuid", map[string]any{
	"voice_url": "https://example.com/voice",
})
client.Fabric.CXMLApplications.Delete("app-uuid")

// This returns an error (not supported):
// client.Fabric.CXMLApplications.Create(...)
```

## Generic Resources

Operate on any resource type by ID:

```go
// List all resources across all types
allResources, _ := client.Fabric.Resources.List(nil)

// Get any resource by ID
resource, _ := client.Fabric.Resources.Get("resource-uuid")

// Delete any resource
client.Fabric.Resources.Delete("resource-uuid")

// List addresses for any resource
addresses, _ := client.Fabric.Resources.ListAddresses("resource-uuid")

// Assign a resource to a phone route
client.Fabric.Resources.AssignPhoneRoute("resource-uuid", "route-uuid")

// Assign a resource as a domain application handler
client.Fabric.Resources.AssignDomainApplication("resource-uuid", "da-uuid")
```

## Fabric Addresses

Read-only access to all fabric addresses:

```go
// List all addresses (filter by type or display_name)
addresses, _ := client.Fabric.Addresses.List(map[string]string{"type": "room"})

// Get a specific address
address, _ := client.Fabric.Addresses.Get("address-uuid")
```

## Tokens

Create tokens for subscribers, guests, invites, and embeds:

```go
// Subscriber token
token, _ := client.Fabric.Tokens.CreateSubscriberToken(map[string]any{
	"reference": "user@example.com",
	"password":  "secret",
})

// Refresh a subscriber token
refreshed, _ := client.Fabric.Tokens.RefreshSubscriberToken("existing-refresh-token")

// Guest token
token, _ = client.Fabric.Tokens.CreateGuestToken(map[string]any{
	"allowed_addresses": []string{"address-uuid-1", "address-uuid-2"},
	"expire_at":         "2025-12-31T23:59:59Z",
})

// Subscriber invite token
token, _ = client.Fabric.Tokens.CreateInviteToken(map[string]any{
	"address_id": "address-uuid",
	"expires_at": "2025-12-31T23:59:59Z",
})

// Click-to-call embed token
token, _ = client.Fabric.Tokens.CreateEmbedToken("embed-source-token")
```
