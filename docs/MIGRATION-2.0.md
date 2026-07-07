# Migrating to SignalWire SDK 2.0

## Module Path Change

Update your `go.mod`:
```
// Before
require github.com/signalwire/signalwire-agents-go v1.x.x

// After
require github.com/signalwire/signalwire-go v2.0.0
```

Then run:
```bash
go mod tidy
```

## Import Changes

<!-- snippet: no-compile illustrative before/after showing the removed v1 module path signalwire-agents-go -->
```go
// Before
import (
    "github.com/signalwire/signalwire-agents-go/pkg/agent"
    "github.com/signalwire/signalwire-agents-go/pkg/rest"
    "github.com/signalwire/signalwire-agents-go/pkg/swml"
)

client := rest.NewSignalWireClient(projectID, token, spaceURL)

// After
import (
    "github.com/signalwire/signalwire-go/pkg/agent"
    "github.com/signalwire/signalwire-go/pkg/rest"
    "github.com/signalwire/signalwire-go/pkg/swml"
)

client := rest.NewRestClient(projectID, token, spaceURL)
```

## Class Renames

| Before | After |
|--------|-------|
| `SignalWireClient` | `RestClient` |
| `NewSignalWireClient` | `NewRestClient` |

## Quick Migration

Find and replace in your project:
```bash
# Update all import paths
find . -name '*.go' -exec sed -i 's|signalwire-agents-go|signalwire-go|g' {} +

# Rename client constructors and types
find . -name '*.go' -exec sed -i 's/NewSignalWireClient/NewRestClient/g' {} +
find . -name '*.go' -exec sed -i 's/SignalWireClient/RestClient/g' {} +

# Update go.mod module path
sed -i 's|signalwire-agents-go|signalwire-go|g' go.mod
go mod tidy
```

## What Didn't Change

- All method signatures (SetPromptText, DefineTool, AddSkill, etc.)
- All parameter structs
- SWML output format
- RELAY protocol
- REST API paths
- Skills, contexts, DataMap -- all the same
