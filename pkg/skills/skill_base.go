// Package skills provides the skills system for SignalWire AI agents.
// Skills are modular capabilities that can be loaded into agents to provide
// tools, prompt sections, speech hints, and global data.
package skills

import (
	"github.com/signalwire/signalwire-go/pkg/logging"
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
	// Logger is a named logger for this skill instance. It is initialized
	// automatically when Name() is first called via NewBaseSkill, or can be
	// set explicitly. Mirrors Python SkillBase.logger.
	Logger *logging.Logger
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

// ShouldSkipPrompt returns true if the "skip_prompt" parameter is set to true.
// Concrete skill overrides of GetPromptSections should call this helper before
// returning prompt content, mirroring Python's get_prompt_sections() guard.
func (b *BaseSkill) ShouldSkipPrompt() bool {
	return b.GetParamBool("skip_prompt", false)
}

// GetPromptSections returns nil (no prompt sections by default).
// When skip_prompt is set to true in Params, returns nil even for concrete
// overrides — concrete overrides that inject prompt sections MUST call
// ShouldSkipPrompt() and return nil (or an empty slice) when it is true.
func (b *BaseSkill) GetPromptSections() []map[string]any {
	if b.ShouldSkipPrompt() {
		return nil
	}
	return nil
}

// Cleanup is a no-op by default.
func (b *BaseSkill) Cleanup() {}

// GetInstanceKey returns a unique key for tracking this skill instance.
// When SupportsMultipleInstances() returns true, the key is composed of
// the skill name and the "tool_name" parameter (defaulting to the skill name),
// matching Python's get_instance_key() behavior for multi-instance skills.
// When SupportsMultipleInstances() returns false, returns the skill name.
func (b *BaseSkill) GetInstanceKey() string {
	if b.SupportsMultipleInstances() {
		toolName := b.GetParamString("tool_name", b.SkillName)
		return b.SkillName + "_" + toolName
	}
	return b.SkillName
}

// GetParameterSchema returns the common parameters available to all skills.
// The "tool_name" parameter is only included when SupportsMultipleInstances()
// returns true, matching Python's conditional inclusion in get_parameter_schema().
func (b *BaseSkill) GetParameterSchema() map[string]map[string]any {
	schema := map[string]map[string]any{
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
	}
	if b.SupportsMultipleInstances() {
		schema["tool_name"] = map[string]any{
			"type":        "string",
			"description": "Custom name for this skill instance (for multiple instances)",
			"default":     b.SkillName,
			"required":    false,
		}
	}
	return schema
}

// GetSkillNamespace returns the namespaced key used to store this skill
// instance's state in agent global_data. Uses the "prefix" parameter if set,
// otherwise falls back to the instance key. Mirrors Python's _get_skill_namespace().
//
// Example: a skill named "datasphere" with no prefix returns "skill:datasphere".
// With prefix "kb" it returns "skill:kb".
func (b *BaseSkill) GetSkillNamespace() string {
	if prefix := b.GetParamString("prefix", ""); prefix != "" {
		return "skill:" + prefix
	}
	return "skill:" + b.GetInstanceKey()
}

// GetSkillData reads this skill instance's namespaced state from rawData.
// rawData is the raw_data map passed to SWAIG function handlers, expected
// to contain a "global_data" key. Returns an empty map when not found.
// Mirrors Python's get_skill_data(raw_data).
func (b *BaseSkill) GetSkillData(rawData map[string]any) map[string]any {
	namespace := b.GetSkillNamespace()
	globalData, _ := rawData["global_data"].(map[string]any)
	if globalData == nil {
		return map[string]any{}
	}
	if data, ok := globalData[namespace].(map[string]any); ok {
		return data
	}
	return map[string]any{}
}

// UpdateSkillData writes this skill instance's namespaced state into result's
// global_data via result.UpdateGlobalData(). Returns result for method chaining.
// Mirrors Python's update_skill_data(result, data).
func (b *BaseSkill) UpdateSkillData(result *swaig.FunctionResult, data map[string]any) *swaig.FunctionResult {
	namespace := b.GetSkillNamespace()
	result.UpdateGlobalData(map[string]any{namespace: data})
	return result
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
