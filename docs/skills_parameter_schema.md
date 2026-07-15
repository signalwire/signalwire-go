# Skills Parameter Schema System

This guide explains the parameter schema system for SignalWire AI Agents SDK skills, which enables GUI configuration tools and programmatic skill discovery.

## Overview

The parameter schema system allows skills to declare their configurable parameters with metadata including types, descriptions, default values, and security hints. This enables:

- **GUI Configuration Tools** - Automatically generate configuration forms
- **API Documentation** - Document all available parameters
- **Validation** - Type checking and constraint validation
- **Security** - Mark sensitive parameters as hidden
- **Environment Variables** - Indicate which parameters can be sourced from environment

## Using the Schema System

### Getting All Skills Schema

Use the `skills.ListSkillsWithParams()` function to get a complete schema of all available skills:

<!-- snippet-setup -->
```go
// Shared context the fragments below assume: `schema` is one skill's parameter
// schema (parameter name -> metadata), as produced by
// skills.ListSkillsWithParams()[<name>].
var schema = map[string]map[string]any{}

var (
	_ = schema
)
```

```go
import "github.com/signalwire/signalwire-go/v3/pkg/skills"

// Get complete schema for all skills.
// Returns map[string]map[string]map[string]any keyed by skill name.
allSchema := skills.ListSkillsWithParams()
_ = allSchema
```

The returned structure, expressed as JSON, looks like:

```json
{
    "web_search": {
        "name": "web_search",
        "description": "Search the web for information using Google Custom Search API",
        "version": "1.0.0",
        "supports_multiple_instances": true,
        "required_env_vars": [],
        "parameters": {
            "api_key": {
                "type": "string",
                "description": "Google Custom Search API key",
                "required": true,
                "hidden": true,
                "env_var": "GOOGLE_SEARCH_API_KEY"
            },
            "search_engine_id": {
                "type": "string",
                "description": "Google Custom Search Engine ID",
                "required": true,
                "hidden": true,
                "env_var": "GOOGLE_SEARCH_ENGINE_ID"
            },
            "num_results": {
                "type": "integer",
                "description": "Default number of search results to return",
                "default": 1,
                "required": false,
                "min": 1,
                "max": 10
            }
        }
    },
    "datetime": {
        "name": "datetime",
        "description": "Get current date, time, and timezone information",
        "version": "1.0.0",
        "supports_multiple_instances": false,
        "required_env_vars": [],
        "parameters": {
            "swaig_fields": {
                "type": "object",
                "description": "Additional SWAIG function metadata to merge into tool definitions",
                "default": {},
                "required": false
            }
        }
    }
}
```

### Using Schema for GUI Configuration

Here's an example of how to use the schema to generate a configuration form:

```go
package main

import (
    "fmt"
    "strings"

    "github.com/signalwire/signalwire-go/v3/pkg/skills"
)

// generateFormField builds an HTML form field from a parameter's schema info.
func generateFormField(paramName string, paramInfo map[string]any) string {
    var b strings.Builder
    fmt.Fprintf(&b, "<div class=\"form-group\">\n")
    fmt.Fprintf(&b, "  <label for=\"%s\">%v</label>\n", paramName, paramInfo["description"])

    // Mark required fields
    required := ""
    if req, _ := paramInfo["required"].(bool); req {
        required = "required"
    }

    // Hide sensitive fields
    inputType := "text"
    if hidden, _ := paramInfo["hidden"].(bool); hidden {
        inputType = "password"
    }

    // Handle different types
    switch paramInfo["type"] {
    case "string":
        def, _ := paramInfo["default"]
        fmt.Fprintf(&b, "  <input type=\"%s\" id=\"%s\" name=\"%s\" value=\"%v\" %s>\n",
            inputType, paramName, paramName, def, required)
    case "integer":
        def := paramInfo["default"]
        minVal := ""
        if v, ok := paramInfo["min"]; ok {
            minVal = fmt.Sprintf("min=\"%v\"", v)
        }
        maxVal := ""
        if v, ok := paramInfo["max"]; ok {
            maxVal = fmt.Sprintf("max=\"%v\"", v)
        }
        fmt.Fprintf(&b, "  <input type=\"number\" id=\"%s\" name=\"%s\" value=\"%v\" %s %s %s>\n",
            paramName, paramName, def, minVal, maxVal, required)
    case "boolean":
        checked := ""
        if def, _ := paramInfo["default"].(bool); def {
            checked = "checked"
        }
        fmt.Fprintf(&b, "  <input type=\"checkbox\" id=\"%s\" name=\"%s\" %s>\n",
            paramName, paramName, checked)
    }

    // Show environment variable hint
    if envVar, ok := paramInfo["env_var"]; ok {
        fmt.Fprintf(&b, "  <small>Can also be set via %v environment variable</small>\n", envVar)
    }

    b.WriteString("</div>\n")
    return b.String()
}

func printWebSearchForm() {
    // Get skills schema
    schema := skills.ListSkillsWithParams()

    // Generate an HTML form for the web_search skill.
    // schema[name] is map[string]map[string]any, so ["parameters"] is a map[string]any.
    webSearchParams := schema["web_search"]["parameters"]

    fmt.Println("<form>")
    for paramName, info := range webSearchParams {
        paramInfo, _ := info.(map[string]any)
        fmt.Print(generateFormField(paramName, paramInfo))
    }
    fmt.Println("</form>")
}

func main() { printWebSearchForm() }
```

### Programmatic Skill Configuration

Use the schema to validate and configure skills programmatically:

```go
package main

import (
    "fmt"

    "github.com/signalwire/signalwire-go/v3/pkg/agent"
    "github.com/signalwire/signalwire-go/v3/pkg/skills"
)

func newMyAgent() (*agent.AgentBase, error) {
    a := agent.NewAgentBase(agent.WithName("my-agent"))

    // Get schema to validate configuration
    schema := skills.ListSkillsWithParams()

    // Configure web_search skill with validation
    webSearchParams := map[string]any{
        "api_key":            "your-api-key",
        "search_engine_id":   "your-engine-id",
        "num_results":        3,
        "max_content_length": 3000,
    }

    // Validate required parameters
    webSearchSchema := schema["web_search"]["parameters"]
    for param, info := range webSearchSchema {
        paramInfo, _ := info.(map[string]any)
        required, _ := paramInfo["required"].(bool)
        if _, present := webSearchParams[param]; required && !present {
            return nil, fmt.Errorf("missing required parameter: %s", param)
        }
    }

    // Add skill with validated parameters
    a.AddSkill("web_search", webSearchParams)
    return a, nil
}

func main() { _, _ = newMyAgent() }
```

## Parameter Schema Reference

Each parameter in the schema can have the following properties:

| Property | Type | Description |
|----------|------|-------------|
| `type` | string | Parameter type: "string", "integer", "number", "boolean", "object", "array" |
| `description` | string | Human-readable description of the parameter |
| `default` | any | Default value if not provided |
| `required` | boolean | Whether the parameter is required (default: false) |
| `hidden` | boolean | Whether to hide this field in UIs (for secrets/API keys) |
| `env_var` | string | Environment variable that can provide this value |
| `enum` | array | List of allowed values (for string types) |
| `min` | number | Minimum value (for numeric types) |
| `max` | number | Maximum value (for numeric types) |

## Implementing Parameter Schema in Skills

To add parameter schema support to a skill, override the `GetParameterSchema()`
method and merge in the base schema (which supplies the common parameters):

```go
package mycustomskill

import (
    "github.com/signalwire/signalwire-go/v3/pkg/skills"
)

type MyCustomSkill struct {
    skills.BaseSkill
    apiEndpoint string
    apiKey      string
    timeout     int
}

func NewMyCustomSkill(params map[string]any) skills.SkillBase {
    return &MyCustomSkill{
        BaseSkill: skills.BaseSkill{
            SkillName: "my_custom_skill",
            SkillDesc: "My custom skill",
            SkillVer:  "1.0.0",
            Params:    params,
        },
    }
}

func (s *MyCustomSkill) GetParameterSchema() map[string]map[string]any {
    // Get base schema (includes common parameters)
    schema := s.BaseSkill.GetParameterSchema()

    // Add skill-specific parameters
    schema["api_endpoint"] = map[string]any{
        "type":        "string",
        "description": "API endpoint URL",
        "required":    true,
        "default":     "https://api.example.com",
    }
    schema["api_key"] = map[string]any{
        "type":        "string",
        "description": "API authentication key",
        "required":    true,
        "hidden":      true,          // Mark as sensitive
        "env_var":     "MY_API_KEY",  // Can be set via environment
    }
    schema["timeout"] = map[string]any{
        "type":        "integer",
        "description": "Request timeout in seconds",
        "default":     30,
        "required":    false,
        "min":         1,
        "max":         300,
    }
    schema["retry_count"] = map[string]any{
        "type":        "integer",
        "description": "Number of retries on failure",
        "default":     3,
        "required":    false,
        "min":         0,
        "max":         10,
    }
    schema["output_format"] = map[string]any{
        "type":        "string",
        "description": "Output format for results",
        "default":     "json",
        "required":    false,
        "enum":        []string{"json", "xml", "text"}, // Allowed values
    }
    schema["enable_cache"] = map[string]any{
        "type":        "boolean",
        "description": "Enable response caching",
        "default":     true,
        "required":    false,
    }

    return schema
}

func (s *MyCustomSkill) Setup() bool {
    // Access parameters via the BaseSkill helpers
    s.apiEndpoint = s.GetParamString("api_endpoint", "")
    s.apiKey = s.GetParamString("api_key", "")
    s.timeout = s.GetParamInt("timeout", 30)
    // ... etc
    return true
}

// RegisterTools registers the skill's tools with the agent (required by SkillBase).
func (s *MyCustomSkill) RegisterTools() []skills.ToolRegistration {
    return nil
}
```

## Common Parameter Patterns

### API Keys and Secrets

Always mark sensitive parameters as `hidden` and provide an `env_var` option:

```go
schema["api_key"] = map[string]any{
    "type":        "string",
    "description": "API key for authentication",
    "required":    true,
    "hidden":      true,
    "env_var":     "SERVICE_API_KEY",
}
```

### Numeric Parameters with Constraints

Use `min` and `max` to enforce valid ranges:

```go
schema["port"] = map[string]any{
    "type":        "integer",
    "description": "Server port number",
    "default":     8080,
    "required":    false,
    "min":         1,
    "max":         65535,
}
```

### Enumerated Values

Use `enum` to restrict to specific values:

```go
schema["log_level"] = map[string]any{
    "type":        "string",
    "description": "Logging level",
    "default":     "info",
    "required":    false,
    "enum":        []string{"debug", "info", "warning", "error"},
}
```

### Optional Features

Use boolean parameters for optional features:

```go
schema["enable_analytics"] = map[string]any{
    "type":        "boolean",
    "description": "Enable analytics tracking",
    "default":     false,
    "required":    false,
}
```

## Base Parameters

All skills automatically inherit these base parameters from `skills.BaseSkill`:

- **`swaig_fields`** (object) - Additional SWAIG function metadata to merge into tool definitions
- **`tool_name`** (string) - Custom name for skill instances (only for skills whose `SupportsMultipleInstances()` returns `true`)

## Examples

### Simple Skill (No Parameters)

Skills like `datetime` and `math` that don't need configuration simply inherit the
base schema. Since `skills.BaseSkill` already provides `GetParameterSchema()`, you
don't override it at all:

```go
import "github.com/signalwire/signalwire-go/v3/pkg/skills"

// No GetParameterSchema override needed — the embedded skills.BaseSkill
// supplies the base schema (swaig_fields, tool_name).
type DateTimeSkill struct {
    skills.BaseSkill
}

var _ DateTimeSkill
```

### Complex Skill (Many Parameters)

Skills like `web_search` with multiple configuration options:

<!-- snippet: no-compile illustrative GetParameterSchema method on WebSearchSkill (type defined elsewhere; entries elided with /* ... */) -->
```go
func (s *WebSearchSkill) GetParameterSchema() map[string]map[string]any {
    schema := s.BaseSkill.GetParameterSchema()

    // API credentials (hidden)
    schema["api_key"] = map[string]any{ /* ... */ }
    schema["api_secret"] = map[string]any{ /* ... */ }

    // Configuration options
    schema["timeout"] = map[string]any{ /* ... */ }
    schema["retry_count"] = map[string]any{ /* ... */ }

    // Feature flags
    schema["enable_cache"] = map[string]any{ /* ... */ }
    schema["debug_mode"] = map[string]any{ /* ... */ }

    // Customization
    schema["response_template"] = map[string]any{ /* ... */ }
    schema["error_messages"] = map[string]any{ /* ... */ }

    return schema
}
```

## Best Practices

1. **Always provide descriptions** - Make parameters self-documenting
2. **Set sensible defaults** - Allow skills to work with minimal configuration
3. **Mark secrets as hidden** - Protect sensitive information in UIs
4. **Use appropriate types** - Enable proper validation and UI controls
5. **Document environment variables** - Show alternative configuration methods
6. **Validate in setup()** - Ensure all required parameters are present
7. **Support backward compatibility** - Handle deprecated parameters gracefully

## Future Enhancements

The parameter schema system is designed to be extensible. Future enhancements may include:

- **Conditional parameters** - Show/hide based on other parameter values
- **Complex validation** - Cross-parameter validation rules
- **Nested schemas** - Support for complex object parameters
- **Internationalization** - Localized descriptions and error messages
- **Runtime parameter updates** - Modify configuration without restart