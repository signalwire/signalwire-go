package builtin

import (
	"fmt"
	"strings"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// SWMLTransferSkill transfers calls between agents using DataMap pattern matching.
type SWMLTransferSkill struct {
	skills.BaseSkill
	toolName           string
	description        string
	paramName          string
	paramDesc          string
	defaultMessage     string
	defaultPostProcess bool
	requiredFields     map[string]string
	transfers          map[string]any
}

// NewSWMLTransfer creates a new SWMLTransferSkill.
func NewSWMLTransfer(params map[string]any) skills.SkillBase {
	return &SWMLTransferSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "swml_transfer",
			SkillDesc: "Transfer calls between agents based on pattern matching",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

func (s *SWMLTransferSkill) SupportsMultipleInstances() bool { return true }

func (s *SWMLTransferSkill) GetInstanceKey() string {
	name := s.GetParamString("tool_name", "transfer_call")
	return "swml_transfer_" + name
}

func (s *SWMLTransferSkill) Setup() bool {
	s.toolName = s.GetParamString("tool_name", "transfer_call")
	s.description = s.GetParamString("description", "Transfer call based on pattern matching")
	s.paramName = s.GetParamString("parameter_name", "transfer_type")
	s.paramDesc = s.GetParamString("parameter_description", "The type of transfer to perform")
	s.defaultMessage = s.GetParamString("default_message", "Please specify a valid transfer type.")
	s.defaultPostProcess = s.GetParamBool("default_post_process", false)

	// Parse required_fields: map[string]string
	s.requiredFields = map[string]string{}
	if rfRaw, ok := s.Params["required_fields"]; ok {
		if rfMap, ok := rfRaw.(map[string]any); ok {
			for k, v := range rfMap {
				if desc, ok := v.(string); ok {
					s.requiredFields[k] = desc
				}
			}
		}
	}

	transfersRaw, ok := s.Params["transfers"]
	if !ok {
		return false
	}
	transfers, ok := transfersRaw.(map[string]any)
	if !ok || len(transfers) == 0 {
		return false
	}
	s.transfers = transfers
	return true
}

func (s *SWMLTransferSkill) RegisterTools() []skills.ToolRegistration {
	// Build enum from transfer keys
	enumVals := make([]string, 0, len(s.transfers))
	for k := range s.transfers {
		enumVals = append(enumVals, k)
	}

	// Build properties: primary param + required_fields
	properties := map[string]any{
		s.paramName: map[string]any{
			"type":        "string",
			"description": s.paramDesc,
			"enum":        enumVals,
		},
	}
	requiredParams := []string{s.paramName}

	for fieldName, fieldDesc := range s.requiredFields {
		properties[fieldName] = map[string]any{
			"type":        "string",
			"description": fieldDesc,
			"required":    true,
		}
		requiredParams = append(requiredParams, fieldName)
	}

	return []skills.ToolRegistration{
		{
			Name:        s.toolName,
			Description: s.description,
			Parameters: map[string]any{
				"type":       "object",
				"properties": properties,
				"required":   requiredParams,
			},
			Handler: s.handleTransfer,
		},
	}
}

func (s *SWMLTransferSkill) handleTransfer(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	transferType, _ := args[s.paramName].(string)
	if transferType == "" {
		result := swaig.NewFunctionResult(s.defaultMessage)
		result.SetPostProcess(s.defaultPostProcess)
		if len(s.requiredFields) > 0 {
			callData := map[string]any{}
			for fieldName := range s.requiredFields {
				if v, ok := args[fieldName]; ok {
					callData[fieldName] = v
				}
			}
			result.UpdateGlobalData(map[string]any{"call_data": callData})
		}
		return result
	}

	configRaw, ok := s.transfers[transferType]
	if !ok {
		result := swaig.NewFunctionResult(s.defaultMessage)
		result.SetPostProcess(s.defaultPostProcess)
		if len(s.requiredFields) > 0 {
			callData := map[string]any{}
			for fieldName := range s.requiredFields {
				if v, ok := args[fieldName]; ok {
					callData[fieldName] = v
				}
			}
			result.UpdateGlobalData(map[string]any{"call_data": callData})
		}
		return result
	}

	config, ok := configRaw.(map[string]any)
	if !ok {
		result := swaig.NewFunctionResult(s.defaultMessage)
		result.SetPostProcess(s.defaultPostProcess)
		return result
	}

	message, _ := config["message"].(string)
	if message == "" {
		message = "Transferring you now..."
	}

	result := swaig.NewFunctionResult(message)

	// Apply per-config post_process; Python defaults it to true for matched transfers
	postProcess := true
	if pp, ok := config["post_process"].(bool); ok {
		postProcess = pp
	}
	result.SetPostProcess(postProcess)

	// Collect required fields into global call_data
	if len(s.requiredFields) > 0 {
		callData := map[string]any{}
		for fieldName := range s.requiredFields {
			if v, ok := args[fieldName]; ok {
				callData[fieldName] = v
			}
		}
		result.UpdateGlobalData(map[string]any{"call_data": callData})
	}

	// Determine transfer type: url (SWML) or address (connect)
	if urlDest, ok := config["url"].(string); ok && urlDest != "" {
		returnMsg, _ := config["return_message"].(string)
		if returnMsg == "" {
			returnMsg = "The transfer is complete. How else can I help you?"
		}
		isFinal := true
		if f, ok := config["final"].(bool); ok {
			isFinal = f
		}
		result.SwmlTransfer(urlDest, returnMsg, isFinal)
	} else if addr, ok := config["address"].(string); ok && addr != "" {
		isFinal := true
		if f, ok := config["final"].(bool); ok {
			isFinal = f
		}
		fromAddr, _ := config["from_addr"].(string)
		result.Connect(addr, isFinal, fromAddr)
	}

	return result
}

// cleanPattern strips regex delimiters (/…/ and /…/i) from a pattern string.
// Returns ("", false) when the pattern is a catch-all (starts with '.') or empty.
func cleanPattern(pattern string) (string, bool) {
	p := pattern
	if strings.HasPrefix(p, "/") {
		p = p[1:]
		if strings.HasSuffix(p, "/i") {
			p = p[:len(p)-2]
		} else if strings.HasSuffix(p, "/") {
			p = p[:len(p)-1]
		}
	}
	if p == "" || strings.HasPrefix(p, ".") {
		return "", false
	}
	return p, true
}

func (s *SWMLTransferSkill) GetHints() []string {
	hints := []string{}

	for pattern := range s.transfers {
		cleaned, ok := cleanPattern(pattern)
		if !ok {
			continue
		}
		if strings.Contains(cleaned, "|") {
			for _, part := range strings.Split(cleaned, "|") {
				part = strings.TrimSpace(part)
				if part != "" {
					hints = append(hints, strings.ToLower(part))
				}
			}
		} else {
			hints = append(hints, strings.ToLower(cleaned))
		}
	}

	// Add common transfer-related words (matching Python order)
	hints = append(hints, "transfer", "connect", "speak to", "talk to")

	return hints
}

func (s *SWMLTransferSkill) GetPromptSections() []map[string]any {
	if len(s.transfers) == 0 {
		return []map[string]any{}
	}

	// Section 1: "Transferring" — list of destinations with cleaned pattern names
	transferBullets := []string{}
	for pattern, configRaw := range s.transfers {
		cleaned, ok := cleanPattern(pattern)
		if !ok {
			continue
		}
		config, _ := configRaw.(map[string]any)
		var destination string
		if urlDest, ok := config["url"].(string); ok && urlDest != "" {
			destination = urlDest
		} else if addr, ok := config["address"].(string); ok {
			destination = addr
		}
		transferBullets = append(transferBullets, fmt.Sprintf(`"%s" - transfers to %s`, cleaned, destination))
	}

	sections := []map[string]any{
		{
			"title":   "Transferring",
			"body":    fmt.Sprintf("You can transfer calls using the %s function with the following destinations:", s.toolName),
			"bullets": transferBullets,
		},
	}

	// Section 2: "Transfer Instructions"
	instructionBullets := []string{
		fmt.Sprintf("Use the %s function when a transfer is needed", s.toolName),
		fmt.Sprintf("Pass the destination type to the '%s' parameter", s.paramName),
	}

	if len(s.requiredFields) > 0 {
		instructionBullets = append(instructionBullets, "You must provide the following information before transferring:")
		for fieldName, fieldDesc := range s.requiredFields {
			instructionBullets = append(instructionBullets, fmt.Sprintf("  - %s: %s", fieldName, fieldDesc))
		}
		instructionBullets = append(instructionBullets, "All required information will be saved under 'call_data' for the next agent")
	}

	instructionBullets = append(instructionBullets,
		"The system will match patterns and handle the transfer automatically",
		"After transfer completes, you'll regain control of the conversation",
	)

	sections = append(sections, map[string]any{
		"title":   "Transfer Instructions",
		"body":    "How to use the transfer capability:",
		"bullets": instructionBullets,
	})

	return sections
}

func (s *SWMLTransferSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["transfers"] = map[string]any{
		"type":        "object",
		"description": "Transfer configurations mapping patterns to destinations",
		"required":    true,
	}
	schema["description"] = map[string]any{
		"type":        "string",
		"description": "Description for the transfer tool",
		"default":     "Transfer call based on pattern matching",
		"required":    false,
	}
	schema["parameter_name"] = map[string]any{
		"type":        "string",
		"description": "Name of the parameter that accepts the transfer type",
		"default":     "transfer_type",
		"required":    false,
	}
	schema["parameter_description"] = map[string]any{
		"type":        "string",
		"description": "Description for the transfer type parameter",
		"default":     "The type of transfer to perform",
		"required":    false,
	}
	schema["default_message"] = map[string]any{
		"type":        "string",
		"description": "Message when no pattern matches",
		"default":     "Please specify a valid transfer type.",
		"required":    false,
	}
	schema["default_post_process"] = map[string]any{
		"type":        "boolean",
		"description": "Whether to process default message with AI",
		"default":     false,
		"required":    false,
	}
	schema["required_fields"] = map[string]any{
		"type":        "object",
		"description": "Additional required fields to collect before transfer",
		"default":     map[string]any{},
		"required":    false,
	}
	return schema
}

func init() {
	skills.RegisterSkill("swml_transfer", NewSWMLTransfer)
}
