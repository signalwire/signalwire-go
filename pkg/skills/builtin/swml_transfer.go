package builtin

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// SWMLTransferSkill transfers calls between agents using DataMap pattern matching.
type SWMLTransferSkill struct {
	skills.BaseSkill
	toolName    string
	description string
	paramName   string
	transfers   map[string]any
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

	return []skills.ToolRegistration{
		{
			Name:        s.toolName,
			Description: s.description,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					s.paramName: map[string]any{
						"type":        "string",
						"description": "The type of transfer to perform",
						"enum":        enumVals,
					},
				},
				"required": []string{s.paramName},
			},
			Handler: s.handleTransfer,
		},
	}
}

func (s *SWMLTransferSkill) handleTransfer(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	transferType, _ := args[s.paramName].(string)
	if transferType == "" {
		return swaig.NewFunctionResult("Please specify a valid transfer type.")
	}

	configRaw, ok := s.transfers[transferType]
	if !ok {
		return swaig.NewFunctionResult("Please specify a valid transfer type.")
	}

	config, ok := configRaw.(map[string]any)
	if !ok {
		return swaig.NewFunctionResult("Invalid transfer configuration.")
	}

	message, _ := config["message"].(string)
	if message == "" {
		message = "Transferring you now..."
	}

	result := swaig.NewFunctionResult(message)

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

func (s *SWMLTransferSkill) GetHints() []string {
	hints := []string{"transfer", "connect", "speak to", "talk to"}
	for k := range s.transfers {
		hints = append(hints, k)
	}
	return hints
}

func (s *SWMLTransferSkill) GetPromptSections() []map[string]any {
	var bullets []string
	for pattern := range s.transfers {
		bullets = append(bullets, fmt.Sprintf("'%s' - available transfer destination", pattern))
	}
	return []map[string]any{
		{
			"title": "Call Transfer",
			"body":  "You can transfer calls using the " + s.toolName + " function.",
			"bullets": append(bullets,
				"Use "+s.toolName+" when a transfer is needed",
				"Pass the destination type to the '"+s.paramName+"' parameter",
			),
		},
	}
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
	return schema
}

func init() {
	skills.RegisterSkill("swml_transfer", NewSWMLTransfer)
}
