# RestClient Reference

## Constructor

```go
client, err := rest.NewRestClient(project, token, host string) (*RestClient, error)
```

All parameters fall back to their corresponding environment variables when passed as empty strings. An error is returned if any are missing.

Authentication uses HTTP Basic Auth (`project:token`).

```go
// Explicit credentials
client, err := rest.NewRestClient("your-project-id", "your-api-token", "example.signalwire.com")

// From environment variables
client, err := rest.NewRestClient("", "", "")
```

## Namespaces

Every API surface is available as a namespace attribute on the client:

### Fabric API

| Attribute | Description |
|-----------|-------------|
| `client.Fabric.SWMLScripts` | SWML script resources (CRUD + addresses) |
| `client.Fabric.SWMLWebhooks` | SWML webhook resources |
| `client.Fabric.AIAgents` | AI agent resources |
| `client.Fabric.RelayApplications` | Relay application resources |
| `client.Fabric.CallFlows` | Call flow resources (+ versions) |
| `client.Fabric.ConferenceRooms` | Conference room resources |
| `client.Fabric.FreeSWITCHConnectors` | FreeSWITCH connector resources |
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

| Attribute | Description |
|-----------|-------------|
| `client.Calling` | REST call control -- 37 commands via POST |

### Relay REST Resources

| Attribute | Description |
|-----------|-------------|
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

| Attribute | Description |
|-----------|-------------|
| `client.Datasphere` | Datasphere document management and semantic search |
| `client.Video` | Video rooms, sessions, recordings, conferences |
| `client.Logs` | Message, voice, fax, and conference logs |
| `client.Project` | API token management |
| `client.PubSub` | PubSub token creation |
| `client.Chat` | Chat token creation |
| `client.Compat` | Twilio-compatible LAML API |

## Error Handling

```go
import (
	"errors"
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/rest"
)

agent, err := client.Fabric.AIAgents.Get("bad-id")
if err != nil {
	var restErr *rest.SignalWireRestError
	if errors.As(err, &restErr) {
		fmt.Println(restErr.StatusCode) // 404
		fmt.Println(restErr.Body)       // map[error:not found]
		fmt.Println(restErr.URL)        // "/api/fabric/resources/ai_agents/bad-id"
		fmt.Println(restErr.Method)     // "GET"
	}
}
```

`SignalWireRestError` is returned on any non-2xx HTTP response.

### Error Fields

| Field | Type | Description |
|-------|------|-------------|
| `StatusCode` | `int` | HTTP status code |
| `Body` | `any` | Response body (parsed JSON map or raw string) |
| `URL` | `string` | Request path |
| `Method` | `string` | HTTP method |

## Session Behavior

- A single `http.Client` is shared across all namespaces for connection pooling.
- Content-Type is always `application/json`.
- User-Agent is `signalwire-go-rest/1.0`.
- DELETE requests returning 204 return an empty map.

## Usage Patterns

### Parallel API calls with goroutines

```go
var wg sync.WaitGroup

wg.Add(2)

var agents map[string]any
var numbers map[string]any

go func() {
	defer wg.Done()
	agents, _ = client.Fabric.AIAgents.List(nil)
}()

go func() {
	defer wg.Done()
	numbers, _ = client.PhoneNumbers.List(nil)
}()

wg.Wait()
fmt.Printf("Agents: %v\nNumbers: %v\n", agents, numbers)
```

### Error handling with retries

```go
import "time"

var result map[string]any
var err error

for attempt := 0; attempt < 3; attempt++ {
	result, err = client.Fabric.AIAgents.Get("agent-uuid")
	if err == nil {
		break
	}

	var restErr *rest.SignalWireRestError
	if errors.As(err, &restErr) && restErr.StatusCode >= 500 {
		time.Sleep(time.Duration(attempt+1) * time.Second)
		continue
	}
	break // don't retry client errors
}
```
