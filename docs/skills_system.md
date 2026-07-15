# SignalWire Agents Skills System

The SignalWire Agents SDK now includes a modular skills system that lets you add capabilities to your agents with simple one-liner calls and configurable parameters.

## What's New

Instead of manually implementing every agent capability, you can now:

<!-- snippet-setup -->
```go
import (
	"fmt"

	"github.com/signalwire/signalwire-go/v3/pkg/agent"
)

// Shared context the fragments below assume.
var a = agent.NewAgentBase()

var (
	_ = a
	_ = fmt.Sprint
)
```

```go
import (
    // Blank-import the built-in skills so their init() functions register them.
    _ "github.com/signalwire/signalwire-go/v3/pkg/skills/all"
)

// Create an agent
a = agent.NewAgentBase(agent.WithName("My Assistant"))

// Add skills with one-liners!
a.AddSkill("web_search", nil) // Web search capability with default settings
a.AddSkill("datetime", nil)   // Current date/time info
a.AddSkill("math", nil)       // Mathematical calculations

// Add skills with custom parameters!
a.AddSkill("web_search", map[string]any{
    "num_results": 3,   // Get 3 search results instead of default 1
    "delay":       0.5, // Add 0.5s delay between requests instead of default 0
})

// Your agent now has all these capabilities automatically
```

## Architecture

The skills system consists of:

### Core Infrastructure
- **`skills.SkillBase`** - The interface every skill implements (embed `skills.BaseSkill` for the defaults) with parameter support
- **`skills.SkillManager`** - Handles loading/unloading and lifecycle management with parameters
- **`AgentBase.AddSkill()`** - Simple method to add skills to agents with optional parameters

### Discovery & Registry
- **Skill registry** - Built-in and third-party skills register themselves via `skills.RegisterSkill` from their package `init()`
- **Blank-import registration** - Skills become available when their package is blank-imported (e.g. `_ "github.com/signalwire/signalwire-go/v3/pkg/skills/all"` for the built-ins)
- **Validation** - Checks required environment variables

### Built-in Skills
- **`web_search`** - Google Custom Search API integration with web scraping
- **`datetime`** - Current date/time information with timezone support
- **`math`** - Basic mathematical calculations

## Available Skills

### Web Search (`web_search`)
Search the internet and extract content from web pages.

**Requirements:**
- Environment variables: `GOOGLE_SEARCH_API_KEY`, `GOOGLE_SEARCH_ENGINE_ID`

**Parameters:**
- `num_results` (default: 1) - Number of search results to retrieve (1-10)
- `delay` (default: 0) - Delay in seconds between web requests

**Tools provided:**
- `web_search(query, num_results)` - Search and scrape web content

**Usage examples:**
```go
// Default: fast single result
a.AddSkill("web_search", nil)

// Custom: multiple results with delay
a.AddSkill("web_search", map[string]any{
    "num_results": 3,
    "delay":       0.5,
})

// Speed optimized: single result, no delay
a.AddSkill("web_search", map[string]any{
    "num_results": 1,
    "delay":       0,
})
```

### Date/Time (`datetime`)  
Get current date and time information.

**Requirements:** None (built-in; timezone handling uses the Go standard library)

**Parameters:** None (no configurable parameters)

**Tools provided:**
- `get_current_time(timezone)` - Current time in any timezone
- `get_current_date(timezone)` - Current date in any timezone

### Math (`math`)
Perform mathematical calculations.

**Requirements:** None

**Parameters:** None (no configurable parameters)

**Tools provided:**
- `calculate(expression)` - Evaluate mathematical expressions safely

### Native Vector Search (`native_vector_search`)
Search a knowledge base hosted on a **remote search server** using vector similarity
and keyword search. The Go SDK skill is **remote-only**: it connects to a running
search server over HTTP and does not build or read any local index files. There is
no local `.swsearch` index, no SQLite/pgvector backend, and no index-building CLI in
the Go SDK -- the search server is operated separately and exposes `/health` and
`/search` HTTP endpoints.

**Requirements:**
- A reachable remote search server (the `remote_url` parameter is required).
- No additional Go packages -- the skill is a built-in and uses only the standard
  library HTTP client.

**Parameters:**
- `remote_url` (required) - URL of the remote search server, e.g.
  `http://localhost:8001` or `http://user:pass@host:8001` (embedded credentials are
  sent as HTTP Basic auth). The URL is validated for SSRF protection; private and
  loopback addresses are rejected unless `SWML_ALLOW_PRIVATE_URLS` is set.
- `index_name` (default: "default") - Name of the index to query on the remote server
- `tool_name` (default: "search_knowledge") - Custom name for the search tool
- `count` (default: 5) - Number of search results to return (1-20)
- `similarity_threshold` (default: 0.0) - Minimum similarity score (0.0 = no limit, 1.0 = exact match)
- `tags` (optional) - List of tags to filter search results
- `response_prefix` (optional) - Text to prepend to responses
- `response_postfix` (optional) - Text to append to responses
- `max_content_length` (default: 32768) - Maximum total response size in characters
- `no_results_message` (default: `"No information found for '{query}'"`) - Message when no results are found; `{query}` is substituted
- `hints` (optional) - Additional speech-recognition hints
- `description` (default: "Search the knowledge base for information") - Tool description shown to the AI

**Tools provided:**
- `search_knowledge(query, count)` (or your custom `tool_name`) - Search the remote index

**Usage examples:**
```go
// Connect to a remote search server
a.AddSkill("native_vector_search", map[string]any{
    "remote_url": "http://localhost:8001",
    "index_name": "knowledge",
})

// Custom tool name, more results, and a similarity floor
a.AddSkill("native_vector_search", map[string]any{
    "remote_url":           "http://search.internal:8001",
    "index_name":           "docs",
    "tool_name":            "search_docs",
    "count":                10,
    "similarity_threshold": 0.5,
})

// Multiple instances querying different indexes on the same server
a.AddSkill("native_vector_search", map[string]any{
    "remote_url": "http://localhost:8001",
    "index_name": "examples",
    "tool_name":  "search_examples",
})
```

### SWML Transfer (`swml_transfer`)
Transfer calls between agents using pattern matching.

**Requirements:** None (no additional packages or environment variables required)

**Parameters:**
- `tool_name` (default: "transfer_call") - Custom name for the transfer function
- `description` (default: "Transfer call based on pattern matching") - Tool description
- `parameter_name` (default: "transfer_type") - Name of the parameter for the transfer function
- `parameter_description` (default: "The type of transfer to perform") - Parameter description
- `transfers` (required) - Dictionary mapping regex patterns to transfer configurations:
  - Pattern (key): Regex pattern to match (e.g., "/sales/i")
  - Configuration (value): Dictionary with:
    - `url` (required): Transfer destination URL
    - `message` (optional): Pre-transfer message
    - `return_message` (optional): Post-transfer message
    - `post_process` (optional, default: True): Enable post-processing
- `default_message` (default: "Please specify a valid transfer type.") - Message when no pattern matches
- `default_post_process` (default: False) - Post-processing flag for default case
- `required_fields` (default: {}) - Object mapping field names to descriptions for data collection before transfer

**Tools provided:**
- `transfer_call(transfer_type, ...required_fields)` (or custom tool_name) - Transfer based on pattern matching with optional required fields

**Usage examples:**
```go
// Simple transfer between departments
a.AddSkill("swml_transfer", map[string]any{
    "tool_name": "transfer_to_department",
    "transfers": map[string]any{
        "/sales/i": map[string]any{
            "url":            "https://example.com/sales",
            "message":        "Transferring to sales...",
            "return_message": "Sales transfer complete.",
        },
        "/support/i": map[string]any{
            "url":            "https://example.com/support",
            "message":        "Transferring to support...",
            "return_message": "Support transfer complete.",
        },
    },
})

// Multiple instances for different transfer types
a.AddSkill("swml_transfer", map[string]any{
    "tool_name":      "route_call",
    "parameter_name": "department",
    "transfers": map[string]any{
        "/sales|billing/i": map[string]any{
            "url":          "https://api.company.com/sales",
            "message":      "Connecting to sales team...",
            "post_process": true,
        },
        "/technical|support/i": map[string]any{
            "url":          "https://api.company.com/support",
            "message":      "Connecting to support team...",
            "post_process": true,
        },
    },
    "default_message": "Would you like sales or support?",
})
```

## Usage Examples

### Basic Usage
```go
import (
    // Blank-import the built-in skills so their init() functions register them.
    _ "github.com/signalwire/signalwire-go/v3/pkg/skills/all"
)

// Create agent and add skills
a = agent.NewAgentBase(agent.WithName("Assistant"), agent.WithRoute("/assistant"))
a.AddSkill("datetime", nil)
a.AddSkill("math", nil)
a.AddSkill("web_search", nil) // Uses defaults: 1 result, no delay

// Start the agent
a.Run()
```

### Skills with Custom Parameters
```go
// Create agent
a = agent.NewAgentBase(agent.WithName("Research Assistant"), agent.WithRoute("/research"))

// Add web search optimized for research (more results)
a.AddSkill("web_search", map[string]any{
    "num_results": 5,   // Get more comprehensive results
    "delay":       1.0, // Be respectful to websites
})

// Add other skills without parameters
a.AddSkill("datetime", nil)
a.AddSkill("math", nil)

// Start the agent
a.Run()
```

### Different Parameter Configurations
```go
// Speed-optimized for quick responses
a.AddSkill("web_search", map[string]any{
    "num_results": 1,
    "delay":       0,
})

// Comprehensive research mode
a.AddSkill("web_search", map[string]any{
    "num_results": 5,
    "delay":       1.0,
})

// Balanced approach
a.AddSkill("web_search", map[string]any{
    "num_results": 3,
    "delay":       0.5,
})
```

### Check Available Skills
```go
import (
    "github.com/signalwire/signalwire-go/v3/pkg/skills"
)

// List all discovered skills with their parameter schemas. The schema for each
// skill is a map of parameter name -> parameter metadata (type/description/
// default/required).
for name, schema := range skills.ListSkillsWithParams() {
    fmt.Printf("- %s (%d parameters)\n", name, len(schema))
    for param, meta := range schema {
        fmt.Printf("    %s: %v\n", param, meta["description"])
    }
}
```

### Runtime Skill Management
```go
a = agent.NewAgentBase(agent.WithName("Dynamic Agent"))

// Add skills with different configurations
a.AddSkill("math", nil)
a.AddSkill("datetime", nil)
a.AddSkill("web_search", map[string]any{"num_results": 2, "delay": 0.3})

// Check what's loaded
fmt.Println("Loaded skills:", a.ListSkills())

// Remove a skill
a.RemoveSkill("math")

// Check if specific skill is loaded
if a.HasSkill("datetime") {
    fmt.Println("Date/time capabilities available")
}
```

## Creating Custom Skills

Create a new skill by defining a struct that embeds `skills.BaseSkill` and
implements the `SkillBase` interface. Register it from the package's `init()` so a
blank-import makes it available:

```go
package myskill // mymodule/myskill/skill.go

import (
    "fmt"

    "github.com/signalwire/signalwire-go/v3/pkg/skills"
    "github.com/signalwire/signalwire-go/v3/pkg/swaig"
)

type MyCustomSkill struct {
    skills.BaseSkill
    maxItems   int
    timeout    int
    retryCount int
}

func NewMyCustomSkill(params map[string]any) skills.SkillBase {
    return &MyCustomSkill{
        BaseSkill: skills.BaseSkill{
            SkillName: "my_skill",
            SkillDesc: "Does something awesome with configurable parameters",
            SkillVer:  "1.0.0",
            Params:    params,
        },
    }
}

// Env vars validated by the SkillManager before Setup runs. (Go compiles its
// dependencies in, so there is no runtime package validation.)
func (s *MyCustomSkill) RequiredEnvVars() []string { return []string{"API_KEY"} }

func (s *MyCustomSkill) Setup() bool {
    // Use parameters with defaults
    s.maxItems = s.GetParamInt("max_items", 10)
    s.timeout = s.GetParamInt("timeout", 30)
    s.retryCount = s.GetParamInt("retry_count", 3)
    return true
}

func (s *MyCustomSkill) RegisterTools() []skills.ToolRegistration {
    return []skills.ToolRegistration{{
        Name:        "my_function",
        Description: fmt.Sprintf("Does something cool (max %d items)", s.maxItems),
        Parameters: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "input": map[string]any{
                    "type":        "string",
                    "description": "Input parameter",
                },
            },
        },
        Handler: s.handleMyFunction,
    }}
}

func (s *MyCustomSkill) handleMyFunction(args, rawData map[string]any) *swaig.FunctionResult {
    // Use s.maxItems, s.timeout, s.retryCount in your logic
    return swaig.NewFunctionResult(fmt.Sprintf("Processed with max_items=%d", s.maxItems))
}

// Speech-recognition hints
func (s *MyCustomSkill) GetHints() []string { return []string{"custom", "skill", "awesome"} }

// Prompt sections to add to the agent
func (s *MyCustomSkill) GetPromptSections() []map[string]any {
    return []map[string]any{{
        "title": "Custom Capability",
        "body":  fmt.Sprintf("You can do custom things with my_skill (configured for %d items).", s.maxItems),
    }}
}

func init() { skills.RegisterSkill("my_skill", NewMyCustomSkill) }
```

Blank-import the package once so its `init()` registers the skill, then use it like
any built-in:
<!-- snippet: no-compile references a hypothetical user module (github.com/you/mymodule/myskill) that does not exist -->
```go
import _ "github.com/you/mymodule/myskill"

// Use defaults
a.AddSkill("my_skill", nil)

// Use custom parameters
a.AddSkill("my_skill", map[string]any{
    "max_items":   20,
    "timeout":     60,
    "retry_count": 5,
})
```

## Quick Start

1. **Add the SDK to your module:**
   ```bash
   go get github.com/signalwire/signalwire-go/v3
   ```

2. **Run the demo:**
   ```bash
   go run examples/skills_demo/main.go
   ```

3. **For web search, set environment variables:**
   ```bash
   export GOOGLE_SEARCH_API_KEY="your_api_key"
   export GOOGLE_SEARCH_ENGINE_ID="your_engine_id"
   ```

## Testing

Test the skills system with parameters. The demo agent under
`examples/skills_demo/main.go` loads several skills and prints the registered set:

```go
package main

import (
    "fmt"

    "github.com/signalwire/signalwire-go/v3/pkg/agent"
    "github.com/signalwire/signalwire-go/v3/pkg/skills"

    _ "github.com/signalwire/signalwire-go/v3/pkg/skills/all"
)

func main() {
    // Show discovered skills
    fmt.Println("Available skills:", skills.ListSkills())

    // Create agent and load skills with parameters
    a := agent.NewAgentBase(agent.WithName("Test"), agent.WithRoute("/test"))
    a.AddSkill("datetime", nil)
    a.AddSkill("math", nil)
    a.AddSkill("web_search", map[string]any{"num_results": 2, "delay": 0.5})

    fmt.Println("Loaded skills:", a.ListSkills())
    fmt.Println("Skills system with parameters working!")
}
```

## Benefits

- **One-liner integration** - `a.AddSkill("skill_name", nil)`
- **Configurable parameters** - `a.AddSkill("skill_name", map[string]any{"param": "value"})`
- **Automatic discovery** - Drop skills in the directory and they're available
- **Dependency validation** - Checks packages and environment variables
- **Modular architecture** - Skills are self-contained and reusable
- **Extensible** - Easy to create custom skills with parameters
- **Clean separation** - Skills don't interfere with each other
- **Performance tuning** - Configure skills for speed vs. comprehensiveness

## Migration Guide

**Before (manual implementation):**
<!-- snippet: no-compile illustrative "old way" pseudo-code (setupGoogleSearch / webSearchHandler are hypothetical user helpers) -->
```go
// Had to manually implement every capability
a = agent.NewAgentBase(agent.WithName("WebSearchAgent"))
setupGoogleSearch(a)
a.DefineTool(agent.ToolDefinition{
    Name:    "web_search",
    Handler: webSearchHandler,
    // Lots of manual code...
})
```

**After (skills system with parameters):**
```go
// Simple one-liner with custom configuration
a = agent.NewAgentBase(agent.WithName("WebSearchAgent"))
a.AddSkill("web_search", map[string]any{
    "num_results": 3,   // Get more results
    "delay":       0.5, // Be respectful to servers
})
// Done! Full web search capability with custom settings.
```

The skills system makes SignalWire agents more modular, maintainable, and configurable. 