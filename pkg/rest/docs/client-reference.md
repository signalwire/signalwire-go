# RestClient Reference

## Constructor

<!-- snippet: no-compile illustrative API signature (reference only) -->
```go
func rest.NewRestClient(project, token, space string) (*rest.RestClient, error)
//   project -> SIGNALWIRE_PROJECT_ID
//   token   -> SIGNALWIRE_API_TOKEN
//   space   -> SIGNALWIRE_SPACE
```

<!-- snippet-setup -->
```go
import "github.com/signalwire/signalwire-go/pkg/rest"

// Shared context assumed by the fragments below: a constructed REST client.
var client, err = rest.NewRestClient("project", "token", "space")

var (
	_ = client
	_ = err
)
```

Any empty-string argument falls back to its corresponding environment variable. A non-nil error is returned if any of the three values is still empty after the environment lookup.

Authentication uses HTTP Basic Auth (`project:token`).

## Namespaces

Every API surface is available as a namespace attribute on the client:

### Fabric API

| Field | Description |
|-------|-------------|
| `client.Fabric.SWMLScripts` | SWML script resources (CRUD + addresses) |
| `client.Fabric.SWMLWebhooks` | SWML webhook resources |
| `client.Fabric.AIAgents` | AI agent resources |
| `client.Fabric.RelayApplications` | Relay application resources |
| `client.Fabric.CallFlows` | Call flow resources (+ versions) |
| `client.Fabric.ConferenceRooms` | Conference room resources |
| `client.Fabric.FreeSwitchConnectors` | FreeSWITCH connector resources |
| `client.Fabric.Subscribers` | Subscriber resources (+ SIP endpoints) |
| `client.Fabric.SIPEndpoints` | SIP endpoint resources |
| `client.Fabric.SIPGateways` | SIP gateway resources |
| `client.Fabric.CXMLScripts` | cXML script resources |
| `client.Fabric.CXMLWebhooks` | cXML webhook resources |
| `client.Fabric.CXMLApplications` | cXML application resources (no create) |
| `client.Fabric.Resources` | Generic resource operations |
| `client.Fabric.Addresses` | Fabric addresses (list/get only) |
| `client.Fabric.Tokens` | Subscriber/guest/invite/embed token creation |

### Calling API

| Field | Description |
|-------|-------------|
| `client.Calling` | REST call control -- 37 commands via POST |

### Relay REST Resources

| Field | Description |
|-------|-------------|
| `client.PhoneNumbers` | Phone number management (+ search) |
| `client.Addresses` | Address management |
| `client.Queues` | Queue management (+ members) |
| `client.Recordings` | Recording management |
| `client.NumberGroups` | Number group management (+ memberships) |
| `client.VerifiedCallers` | Verified caller ID management (+ verification flow) |
| `client.SIPProfile` | Project SIP profile (get/update) |
| `client.Lookup` | Phone number lookup |
| `client.ShortCodes` | Short code management |
| `client.ImportedNumbers` | Import external phone numbers |
| `client.MFA` | Multi-factor authentication (SMS/call/verify) |
| `client.Registry` | 10DLC brand/campaign registry |

### Other APIs

| Field | Description |
|-------|-------------|
| `client.Datasphere` | Datasphere document management and semantic search |
| `client.Video` | Video rooms, sessions, recordings, conferences |
| `client.Logs` | Message, voice, fax, and conference logs |
| `client.Project` | API token management |
| `client.PubSub` | PubSub token creation |
| `client.Chat` | Chat token creation |

## Error Handling

```go
import (
	"errors"
	"fmt"
)

_, err = client.Fabric.AIAgents.Get("bad-id")
var restErr *rest.SignalWireRestError
if errors.As(err, &restErr) {
	fmt.Println(restErr.StatusCode) // 404
	fmt.Println(restErr.Body)       // {"error": "not found"}
	fmt.Println(restErr.URL)        // "/api/fabric/resources/ai_agents/bad-id"
	fmt.Println(restErr.Method)     // "GET"
}
```

`*SignalWireRestError` is returned on any non-2xx HTTP response.

### Error Fields

| Field | Type | Description |
|-------|------|-------------|
| `StatusCode` | `int` | HTTP status code |
| `Body` | `string` | Response body (raw JSON or text) |
| `URL` | `string` | Request path |
| `Method` | `string` | HTTP method |

## Client Behavior

- A single `*http.Client` is shared across all namespaces for connection pooling.
- Content-Type is always `application/json`.
- DELETE requests returning 204 return an empty `map[string]any`.
