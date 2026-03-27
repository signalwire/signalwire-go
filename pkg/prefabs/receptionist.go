package prefabs

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// Department describes a destination the receptionist can transfer to.
type Department struct {
	Name         string // e.g. "sales"
	Description  string // what the department handles
	Number       string // phone number or SWML transfer destination
	TransferSWML bool   // true if Number is a SWML destination (uses SwmlTransfer)
}

// ReceptionistOptions configures a new ReceptionistAgent.
type ReceptionistOptions struct {
	Name        string
	Route       string
	Departments []Department
	Greeting    string
}

// ReceptionistAgent greets callers and routes them to the appropriate department.
type ReceptionistAgent struct {
	*agent.AgentBase
	departments []Department
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// NewReceptionistAgent creates an agent that greets callers and transfers
// them to the appropriate department.
func NewReceptionistAgent(opts ReceptionistOptions) *ReceptionistAgent {
	name := opts.Name
	if name == "" {
		name = "receptionist"
	}
	route := opts.Route
	if route == "" {
		route = "/receptionist"
	}
	greeting := opts.Greeting
	if greeting == "" {
		greeting = "Thank you for calling. How can I help you today?"
	}

	base := agent.NewAgentBase(
		agent.WithName(name),
		agent.WithRoute(route),
	)

	ra := &ReceptionistAgent{
		AgentBase:   base,
		departments: opts.Departments,
	}

	// ---- Prompt ----
	base.PromptAddSection("Personality",
		"You are a friendly and professional receptionist. You speak clearly and efficiently while maintaining a warm, helpful tone.",
		nil,
	)
	base.PromptAddSection("Goal",
		"Your goal is to greet callers, collect their basic information, and transfer them to the appropriate department.",
		nil,
	)
	base.PromptAddSection("Instructions", "", []string{
		fmt.Sprintf("Begin by greeting the caller with: '%s'", greeting),
		"Collect their name and a brief description of their needs.",
		"Based on their needs, determine which department would be most appropriate.",
		"Use the collect_caller_info function when you have their name and reason for calling.",
		"Use the transfer_call function to transfer them to the appropriate department.",
		"Before transferring, always confirm with the caller that they're being transferred to the right department.",
		"If a caller's request doesn't clearly match a department, ask follow-up questions to clarify.",
	})

	// Department list in the prompt
	deptBullets := make([]string, len(opts.Departments))
	for i, d := range opts.Departments {
		deptBullets[i] = fmt.Sprintf("%s: %s", d.Name, d.Description)
	}
	base.PromptAddSection("Available Departments", "", deptBullets)

	// Post-prompt for JSON summary
	base.SetPostPrompt(`Return a JSON summary of the conversation:
{
    "caller_name": "CALLER'S NAME",
    "reason": "REASON FOR CALLING",
    "department": "DEPARTMENT TRANSFERRED TO",
    "satisfaction": "high/medium/low"
}`)

	// ---- Global data ----
	deptMaps := make([]map[string]any, len(opts.Departments))
	for i, d := range opts.Departments {
		deptMaps[i] = map[string]any{
			"name":          d.Name,
			"description":   d.Description,
			"number":        d.Number,
			"transfer_swml": d.TransferSWML,
		}
	}
	base.SetGlobalData(map[string]any{
		"departments": deptMaps,
		"caller_info": map[string]any{},
	})

	// ---- Tools ----
	ra.registerTools()

	return ra
}

// ---------------------------------------------------------------------------
// Tool registration
// ---------------------------------------------------------------------------

func (ra *ReceptionistAgent) registerTools() {
	// collect_caller_info ------------------------------------------------
	ra.DefineTool(agent.ToolDefinition{
		Name:        "collect_caller_info",
		Description: "Collect the caller's information for routing",
		Parameters: map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The caller's name",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "The reason for the call",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			callerName, _ := args["name"].(string)
			reason, _ := args["reason"].(string)

			result := swaig.NewFunctionResult(
				fmt.Sprintf("Thank you, %s. I've noted that you're calling about %s.", callerName, reason),
			)
			result.UpdateGlobalData(map[string]any{
				"caller_info": map[string]any{
					"name":   callerName,
					"reason": reason,
				},
			})
			return result
		},
	})

	// Build department name enum for parameter description
	deptNames := make([]string, len(ra.departments))
	for i, d := range ra.departments {
		deptNames[i] = d.Name
	}

	// transfer_call ----------------------------------------------------
	ra.DefineTool(agent.ToolDefinition{
		Name:        "transfer_call",
		Description: "Transfer the caller to the appropriate department",
		Parameters: map[string]any{
			"department": map[string]any{
				"type":        "string",
				"description": "The department to transfer to",
				"enum":        deptNames,
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			deptName, _ := args["department"].(string)

			// Look up the department
			var dept *Department
			for i := range ra.departments {
				if ra.departments[i].Name == deptName {
					dept = &ra.departments[i]
					break
				}
			}
			if dept == nil {
				return swaig.NewFunctionResult(
					fmt.Sprintf("Sorry, I couldn't find the %s department.", deptName),
				)
			}

			// Get caller name from global data for a friendly message
			globalData, _ := rawData["global_data"].(map[string]any)
			callerInfo, _ := globalData["caller_info"].(map[string]any)
			callerName, _ := callerInfo["name"].(string)
			if callerName == "" {
				callerName = "the caller"
			}

			result := swaig.NewFunctionResult(
				fmt.Sprintf("I'll transfer you to our %s department now. Thank you for calling, %s!", deptName, callerName),
			)
			result.SetPostProcess(true)

			if dept.TransferSWML {
				result.SwmlTransfer(dept.Number, "Transferring you now.", true)
			} else {
				result.Connect(dept.Number, true, "")
			}

			return result
		},
	})
}
