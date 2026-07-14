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
import (
	"context"

	"github.com/signalwire/signalwire-go/pkg/rest"
)

// Shared context assumed by the fragments below: a constructed REST client.
var client, err = rest.NewRestClient("project", "token", "space")

var (
	_ = client
	_ = err
	_ = context.Background
)
```

All parameters fall back to their corresponding environment variables when passed as empty strings. An error is returned if any are missing.

Authentication uses HTTP Basic Auth (`project:token`).

```go
// Explicit credentials
client, err = rest.NewRestClient("your-project-id", "your-api-token", "example.signalwire.com")

// From environment variables
client, err = rest.NewRestClient("", "", "")
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

## Error Handling

```go
import (
	"errors"
	"fmt"
)

agent, err := client.Fabric.AIAgents.Get(context.Background(), "bad-id")
if err != nil {
	var restErr *rest.SignalWireRestError
	if errors.As(err, &restErr) {
		fmt.Println(restErr.StatusCode) // 404
		fmt.Println(restErr.Body)       // `{"error":"not found"}` (raw response body)
		fmt.Println(restErr.URL)        // "/api/fabric/resources/ai_agents/bad-id"
		fmt.Println(restErr.Method)     // "GET"
	}
}
_ = agent
```

`SignalWireRestError` is returned on any non-2xx HTTP response.

### Error Fields

| Field | Type | Description |
|-------|------|-------------|
| `StatusCode` | `int` | HTTP status code |
| `Body` | `string` | Raw response body |
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
import (
	"fmt"
	"sync"
)

var wg sync.WaitGroup

wg.Add(2)

var agents map[string]any
var numbers map[string]any

go func() {
	defer wg.Done()
	agents, _ = client.Fabric.AIAgents.List(context.Background(), nil)
}()

go func() {
	defer wg.Done()
	numbers, _ = client.PhoneNumbers.List(context.Background(), nil)
}()

wg.Wait()
fmt.Printf("Agents: %v\nNumbers: %v\n", agents, numbers)
```

### Error handling with retries

```go
import (
	"errors"
	"time"
)

var result map[string]any
var getErr error

for attempt := 0; attempt < 3; attempt++ {
	result, getErr = client.Fabric.AIAgents.Get(context.Background(), "agent-uuid")
	if getErr == nil {
		break
	}

	var restErr *rest.SignalWireRestError
	if errors.As(getErr, &restErr) && restErr.StatusCode >= 500 {
		time.Sleep(time.Duration(attempt+1) * time.Second)
		continue
	}
	break // don't retry client errors
}
_ = result
```
