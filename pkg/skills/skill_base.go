// Package skills provides the skills system for SignalWire AI agents.
// Skills are modular capabilities that can be loaded into agents to provide
// tools, prompt sections, speech hints, and global data.
package skills

import (
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// SkillBase defines the interface that all skills must implement.
type SkillBase interface {
	// Name returns the unique skill identifier.
	Name() string
	// Description returns a human-readable description of the skill.
	Description() string
	// Version returns the semantic version of the skill.
	Version() string
	// RequiredEnvVars returns environment variable names that must be set.
	RequiredEnvVars() []string
	// SupportsMultipleInstances returns whether multiple instances are allowed.
	SupportsMultipleInstances() bool
	// Setup validates configuration and initializes the skill.
	// Returns true if setup was successful.
	Setup() bool
	// RegisterTools returns tool registrations for this skill.
	RegisterTools() []ToolRegistration
	// GetHints returns speech recognition hints for this skill.
	GetHints() []string
	// GetGlobalData returns data to add to the agent's global context.
	GetGlobalData() map[string]any
	// GetPromptSections returns prompt sections to inject into the agent.
	GetPromptSections() []map[string]any
	// Cleanup releases resources when the skill is unloaded.
	Cleanup()
	// GetInstanceKey returns a unique key for tracking this skill instance.
	GetInstanceKey() string
	// GetParameterSchema returns metadata about all parameters the skill accepts.
	GetParameterSchema() map[string]map[string]any
}

// ToolRegistration describes a tool that a skill wants to register with the agent.
type ToolRegistration struct {
	Name        string
	Description string
	Parameters  map[string]any
	Handler     swaig.ToolHandler
	Secure      bool
	Fillers     map[string][]string
	SwaigFields map[string]any
}

// BaseSkill provides default implementations for the SkillBase interface.
// Concrete skills should embed this struct and override methods as needed.
type BaseSkill struct {
	SkillName string
	SkillDesc string
	SkillVer  string
	Params    map[string]any
}

// Name returns the skill name.
func (b *BaseSkill) Name() string { return b.SkillName }

// Description returns the skill description.
func (b *BaseSkill) Description() string { return b.SkillDesc }

// Version returns the skill version, defaulting to "1.0.0".
func (b *BaseSkill) Version() string {
	if b.SkillVer == "" {
		return "1.0.0"
	}
	return b.SkillVer
}

// RequiredEnvVars returns nil (no required env vars by default).
func (b *BaseSkill) RequiredEnvVars() []string { return nil }

// SupportsMultipleInstances returns false by default.
func (b *BaseSkill) SupportsMultipleInstances() bool { return false }

// GetHints returns nil (no hints by default).
func (b *BaseSkill) GetHints() []string { return nil }

// GetGlobalData returns nil (no global data by default).
func (b *BaseSkill) GetGlobalData() map[string]any { return nil }

// GetPromptSections returns nil (no prompt sections by default).
func (b *BaseSkill) GetPromptSections() []map[string]any { return nil }

// Cleanup is a no-op by default.
func (b *BaseSkill) Cleanup() {}

// GetInstanceKey returns the skill name as the default instance key.
func (b *BaseSkill) GetInstanceKey() string { return b.SkillName }

// GetParameterSchema returns the common parameters available to all skills.
func (b *BaseSkill) GetParameterSchema() map[string]map[string]any {
	return map[string]map[string]any{
		"swaig_fields": {
			"type":        "object",
			"description": "Additional SWAIG function metadata to merge into tool definitions",
			"default":     map[string]any{},
			"required":    false,
		},
		"skip_prompt": {
			"type":        "boolean",
			"description": "If true, the skill will not inject its default prompt section into the POM",
			"default":     false,
			"required":    false,
		},
		"tool_name": {
			"type":        "string",
			"description": "Custom name for this skill instance",
			"default":     b.SkillName,
			"required":    false,
		},
	}
}

// GetParam retrieves a parameter value from the skill's Params map.
// Returns the value and true if found, or nil and false otherwise.
func (b *BaseSkill) GetParam(key string) (any, bool) {
	if b.Params == nil {
		return nil, false
	}
	v, ok := b.Params[key]
	return v, ok
}

// GetParamString retrieves a string parameter, returning the default if not found.
func (b *BaseSkill) GetParamString(key, defaultVal string) string {
	if b.Params == nil {
		return defaultVal
	}
	v, ok := b.Params[key]
	if !ok {
		return defaultVal
	}
	s, ok := v.(string)
	if !ok {
		return defaultVal
	}
	return s
}

// GetParamInt retrieves an integer parameter, returning the default if not found.
func (b *BaseSkill) GetParamInt(key string, defaultVal int) int {
	if b.Params == nil {
		return defaultVal
	}
	v, ok := b.Params[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	default:
		return defaultVal
	}
}

// GetParamFloat retrieves a float parameter, returning the default if not found.
func (b *BaseSkill) GetParamFloat(key string, defaultVal float64) float64 {
	if b.Params == nil {
		return defaultVal
	}
	v, ok := b.Params[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return defaultVal
	}
}

// GetParamBool retrieves a boolean parameter, returning the default if not found.
func (b *BaseSkill) GetParamBool(key string, defaultVal bool) bool {
	if b.Params == nil {
		return defaultVal
	}
	v, ok := b.Params[key]
	if !ok {
		return defaultVal
	}
	bv, ok := v.(bool)
	if !ok {
		return defaultVal
	}
	return bv
}
