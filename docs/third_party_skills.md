# Third-Party Skills Integration Guide

This guide explains how to create and integrate third-party skills with the SignalWire AI Agents SDK. The SDK supports multiple methods for loading external skills, making it easy to extend agent capabilities without modifying the core SDK.

## Overview

Because the Go SDK compiles to a single static binary, third-party skills are
integrated at **build time** by importing their package. A package registers its
skill(s) from `init()` (via `skills.RegisterSkill`), so a blank import is all that
is needed to make a skill available. There is no runtime directory scanning or
plugin loading. The integration approaches are:

1. **Direct Registration** - Register a skill factory from your own code with `skills.RegisterSkill`
2. **Package Registration** - Blank-import a skill package so its `init()` registers it
3. **Published Modules** - `go get` a module that provides skills, then blank-import its package(s)
4. **Directory Tracking** - `skills.AddSkillDirectory` records external skill directories for introspection only (it does not load code — the binary is static)

All registered third-party skills are discovered and indexed the same way as
built-in skills, appearing in `skills.ListSkillsWithParams()` output with their
parameter schemas.

## Creating a Third-Party Skill

Third-party skills follow the same structure as built-in skills: a struct that
embeds `skills.BaseSkill` and implements the `SkillBase` interface, registered from
`init()`. Here's a minimal example:

```go
// mymodule/weather/skill.go
package weather

import (
    "strings"

    "github.com/signalwire/signalwire-go/pkg/skills"
    "github.com/signalwire/signalwire-go/pkg/swaig"
)

// WeatherSkill is a custom weather information skill.
type WeatherSkill struct {
    skills.BaseSkill
    apiKey       string
    units        string
    cacheTimeout int
}

func NewWeatherSkill(params map[string]any) skills.SkillBase {
    return &WeatherSkill{
        BaseSkill: skills.BaseSkill{
            SkillName: "weather",
            SkillDesc: "Get weather information for any location",
            SkillVer:  "1.0.0",
            Params:    params,
        },
    }
}

// GetParameterSchema defines the configuration parameters.
func (s *WeatherSkill) GetParameterSchema() map[string]map[string]any {
    schema := s.BaseSkill.GetParameterSchema()

    schema["api_key"] = map[string]any{
        "type":        "string",
        "description": "Weather API key",
        "required":    true,
        "hidden":      true,
        "env_var":     "WEATHER_API_KEY",
    }
    schema["units"] = map[string]any{
        "type":        "string",
        "description": "Temperature units",
        "default":     "celsius",
        "required":    false,
        "enum":        []string{"celsius", "fahrenheit", "kelvin"},
    }
    schema["cache_timeout"] = map[string]any{
        "type":        "integer",
        "description": "Cache timeout in seconds",
        "default":     300,
        "required":    false,
        "min":         0,
        "max":         3600,
    }

    return schema
}

// Setup initializes the skill. Env-var requirements are validated by the
// SkillManager via RequiredEnvVars; Go has no runtime package validation.
func (s *WeatherSkill) Setup() bool {
    s.apiKey = s.GetParamString("api_key", "")
    if s.apiKey == "" {
        return false
    }
    s.units = s.GetParamString("units", "celsius")
    s.cacheTimeout = s.GetParamInt("cache_timeout", 300)
    return true
}

// RegisterTools registers the weather tools with the agent.
func (s *WeatherSkill) RegisterTools() []skills.ToolRegistration {
    return []skills.ToolRegistration{{
        Name:        "get_weather",
        Description: "Get current weather for a location",
        Parameters: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "location": map[string]any{
                    "type":        "string",
                    "description": "City name or coordinates",
                },
            },
        },
        Handler: s.handleGetWeather,
    }}
}

func (s *WeatherSkill) handleGetWeather(args, rawData map[string]any) *swaig.FunctionResult {
    location, _ := args["location"].(string)
    location = strings.TrimSpace(location)

    if location == "" {
        return swaig.NewFunctionResult("Please provide a location")
    }

    // Implementation would call weather API here. This is just an example.
    unit := strings.ToUpper(s.units[:1])
    return swaig.NewFunctionResult("The weather in " + location + " is sunny and 22°" + unit)
}

func init() { skills.RegisterSkill("weather", NewWeatherSkill) }
```

## Integration Methods

### Method 1: Direct Registration

If you define a skill in your own program, register its factory directly with
`skills.RegisterSkill` (typically from the package's `init()`, or from `main` before
you build agents):

```go
import (
    "github.com/signalwire/signalwire-go/pkg/agent"
    "github.com/signalwire/signalwire-go/pkg/skills"

    "github.com/you/mymodule/weather" // exports NewWeatherSkill
)

func newMyAgent() *agent.AgentBase {
    // Register the skill factory globally.
    skills.RegisterSkill("weather", weather.NewWeatherSkill)

    a := agent.NewAgentBase(agent.WithName("my-agent"))

    // Add the registered skill
    a.AddSkill("weather", map[string]any{
        "api_key": "your-api-key",
        "units":   "fahrenheit",
    })
    return a
}
```

### Method 2: Package Registration (blank import)

If the skill package registers itself from `init()` (the recommended pattern —
see the `init()` line in the example above), you only need to blank-import the
package. Its `init()` runs at program start and calls `skills.RegisterSkill` for
you:

```go
import (
    // Blank-import each skill package so its init() registers the skill.
    _ "github.com/you/myskills/weather"
    _ "github.com/you/myskills/stock_market"
    _ "github.com/you/myskills/translation"
)

// Now use any skill that was registered by the imported packages.
// a.AddSkill("weather", map[string]any{"api_key": "..."})
// a.AddSkill("stock_market", map[string]any{"api_key": "..."})
```

Because Go links skills in at build time, there is no on-disk directory scan.
`skills.AddSkillDirectory(path)` exists but only *tracks* a path for introspection
tools — it does not load code from disk (the binary is static):

```go
// Record an external skill directory for introspection/tooling.
// This does NOT load or execute skills — importing the package does.
_ = skills.AddSkillDirectory("/opt/custom_skills")
```

### Method 3: Published Modules

Publish your skills as a Go module and depend on it like any other library. Consumers
`go get` the module and blank-import the package(s):

```bash
go get github.com/you/my-signalwire-skills@latest
```

```go
import (
    // Each imported package registers its skill via init().
    _ "github.com/you/my-signalwire-skills/weather"
    _ "github.com/you/my-signalwire-skills/stock"
    _ "github.com/you/my-signalwire-skills/translate"
)

// Registered skills are now available:
// a.AddSkill("weather", map[string]any{"api_key": "..."})
```

### Method 4: Introspection-Only Directory Tracking

Go has no `SIGNALWIRE_SKILL_PATHS`-style dynamic loading — a static binary cannot
load skill code from a directory at runtime. To make a skill available you must
import its package (Methods 2 and 3). `skills.AddSkillDirectory` only records paths
so tooling can report which external directories are in play:

```go
// Track directories for introspection tools; loading still happens via import.
_ = skills.AddSkillDirectory("/opt/my_skills")
_ = skills.AddSkillDirectory("/home/user/custom_skills")
```

## Module Structure

A skills module is a normal Go module: one package per skill (the package name
typically matches the skill's `SkillName`). Each package registers its skill from
`init()`:

```text
github.com/you/my-signalwire-skills/
├── go.mod
├── weather/                 # Package (matches SkillName "weather")
│   ├── skill.go            # Required: skill struct + init() registration
│   └── README.md           # Optional: Documentation
├── translation/
│   ├── skill.go
│   └── resources/          # Optional: Additional embedded files
│       └── languages.json  # (embed with //go:embed)
└── stock_market/
    └── skill.go
```

## Skill Discovery and Schema

Third-party skills are fully integrated with the SDK's discovery system once their
package has been imported:

```go
import (
    "fmt"

    "github.com/signalwire/signalwire-go/pkg/skills"

    _ "github.com/you/my-signalwire-skills/weather" // registers "weather"
)

// Get all skills including third-party ones.
allSkills := skills.ListSkillsWithParams()

fmt.Printf("%v\n", allSkills["weather"])
```

The `weather` entry, expressed as JSON:

```json
{
    "name": "weather",
    "description": "Get weather information for any location",
    "version": "1.0.0",
    "supports_multiple_instances": false,
    "required_env_vars": [],
    "parameters": {
        "api_key": {
            "type": "string",
            "description": "Weather API key",
            "required": true,
            "hidden": true,
            "env_var": "WEATHER_API_KEY"
        },
        "units": {
            "type": "string",
            "description": "Temperature units",
            "default": "celsius",
            "required": false,
            "enum": ["celsius", "fahrenheit", "kelvin"]
        }
    }
}
```

## Best Practices

### 1. Skill Naming

- Use lowercase, underscore-separated names
- Choose unique names to avoid conflicts with built-in skills
- Match the package name to `SkillName` for clarity

### 2. Parameter Design

- Always implement `GetParameterSchema()` for GUI compatibility
- Mark sensitive parameters as `hidden`
- Provide sensible defaults
- Use `env_var` for parameters that can come from environment

### 3. Error Handling

```go
func (s *WeatherSkill) Setup() bool {
    // Validate required parameters
    s.apiKey = s.GetParamString("api_key", "")
    if s.apiKey == "" {
        s.Logger.Error("API key is required")
        return false
    }

    // Test connectivity
    if err := s.testAPIConnection(); err != nil {
        s.Logger.Error("failed to connect to API: %v", err)
        return false
    }

    return true
}
```

### 4. Documentation

Include a README.md in your skill package directory:

````markdown
# Weather Skill

Provides weather information for any location.

## Configuration

- `api_key` (required): Your weather API key
- `units` (optional): Temperature units (celsius, fahrenheit, kelvin)
- `cache_timeout` (optional): Cache timeout in seconds

## Usage

```go
a.AddSkill("weather", map[string]any{
    "api_key": "your-api-key",
    "units":   "fahrenheit",
})
```
````

## Advanced Features

### Multiple Instances

Support multiple instances of your skill by returning `true` from
`SupportsMultipleInstances()` and providing a unique `GetInstanceKey()`:

```go
func (s *WeatherSkill) SupportsMultipleInstances() bool { return true }

func (s *WeatherSkill) GetInstanceKey() string {
    service := s.GetParamString("service", "default")
    return s.Name() + "_" + service
}
```

Usage:

```go
// Add multiple weather services
a.AddSkill("weather", map[string]any{
    "tool_name": "openweather",
    "service":   "openweathermap",
    "api_key":   "key1",
})

a.AddSkill("weather", map[string]any{
    "tool_name": "weatherapi",
    "service":   "weatherapi",
    "api_key":   "key2",
})
```

### Dynamic Tool Names

Customize tool names for better agent prompts:

```go
func (s *WeatherSkill) RegisterTools() []skills.ToolRegistration {
    toolName := s.GetParamString("tool_name", "get_weather")
    service := s.GetParamString("service", "default")

    return []skills.ToolRegistration{{
        Name:        toolName,
        Description: "Get weather using " + service,
        Parameters:  map[string]any{ /* ... */ },
        Handler:     s.handleGetWeather,
    }}
}
```

### Skill Dependencies

A skill has no back-reference to the agent, so it can't inspect which other skills
are loaded from inside `Setup()`. Enforce cross-skill dependencies at wiring time
instead — add the prerequisite skill first and check `HasSkill` on the agent before
adding the dependent one:

```go
func addWeather(a *agent.AgentBase) error {
    // The weather skill depends on the translation skill.
    a.AddSkill("translation", nil)
    if !a.HasSkill("translation") {
        return fmt.Errorf("weather skill requires the translation skill")
    }
    a.AddSkill("weather", map[string]any{"api_key": "..."})
    return nil
}
```

## Testing Third-Party Skills

Test your skills before distribution with the standard `testing` package:

```go
// skill_test.go
package weather

import (
    "testing"

    "github.com/signalwire/signalwire-go/pkg/agent"
    "github.com/signalwire/signalwire-go/pkg/skills"
)

func TestSkillRegistration(t *testing.T) {
    // Register the skill factory directly.
    skills.RegisterSkill("weather", NewWeatherSkill)

    a := agent.NewAgentBase(agent.WithName("test-agent"))
    a.AddSkill("weather", map[string]any{"api_key": "test-key"})

    if !a.HasSkill("weather") {
        t.Fatal("expected weather skill to be loaded")
    }
}

func TestParameterSchema(t *testing.T) {
    s := NewWeatherSkill(nil)
    schema := s.GetParameterSchema()

    apiKey, ok := schema["api_key"]
    if !ok {
        t.Fatal("expected api_key in schema")
    }
    if req, _ := apiKey["required"].(bool); !req {
        t.Error("api_key should be required")
    }
    if hidden, _ := apiKey["hidden"].(bool); !hidden {
        t.Error("api_key should be hidden")
    }
}
```

## Troubleshooting

### Skill Not Found

If your skill isn't being discovered:

1. Make sure the skill's package is blank-imported somewhere in the build (an unused import is elided by the compiler; a blank import `_ "…/weather"` keeps it and runs its `init()`)
2. Verify the name passed to `skills.RegisterSkill` matches the name you pass to `AddSkill`
3. Confirm the package's `init()` actually calls `skills.RegisterSkill`
4. Check logs for setup errors (a skill whose `Setup()` returns false fails to load)

### Registration Not Running

A skill registers itself from `init()`, which only runs if the package is imported.
If nothing in your program references the package, add a blank import to force it:

```go
import (
    // Force the package's init() to run and register its skill.
    _ "github.com/you/myskills/weather"
)
```

### Verifying Registered Skills

List every registered skill (built-in and third-party) to confirm yours is present:

```go
import (
    "fmt"

    "github.com/signalwire/signalwire-go/pkg/skills"
)

func printRegisteredSkills() {
    fmt.Println("Registered skills:", skills.ListSkills())
}
```

## Example: Complete Third-Party Skill Package

Here's a complete example of a distributable Go skills module:

```text
my-signalwire-skills/
├── go.mod
├── README.md
├── weather/
│   ├── skill.go        # WeatherSkill + init() registration
│   ├── skill_test.go
│   └── utils.go
└── translation/
    ├── skill.go        # TranslationSkill + init() registration
    └── skill_test.go
```

The `go.mod` declares the module path and its dependency on the SDK:

```text
module github.com/yourname/my-signalwire-skills

go 1.22

require github.com/signalwire/signalwire-go v1.0.12
```

Each package registers its skill from `init()`, e.g. in `weather/skill.go`:

```go
func init() { skills.RegisterSkill("weather", NewWeatherSkill) }
```

Install and use — `go get` the module, then blank-import each skill package so its
`init()` registers the skill:

```bash
go get github.com/yourname/my-signalwire-skills@latest
```

```go
package main

import (
    "github.com/signalwire/signalwire-go/pkg/agent"

    // Blank-import the built-in skills and the third-party ones.
    _ "github.com/signalwire/signalwire-go/pkg/skills/all"
    _ "github.com/yourname/my-signalwire-skills/translation"
    _ "github.com/yourname/my-signalwire-skills/weather"
)

func main() {
    a := agent.NewAgentBase(agent.WithName("my-agent"))
    a.AddSkill("weather", map[string]any{"api_key": "..."})
    a.AddSkill("translate", map[string]any{"api_key": "..."})
    a.Run()
}
```