package prefabs

import (
	"fmt"
	"strings"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// Amenity describes a venue amenity with its hours, location, and extra details.
type Amenity struct {
	Hours    string
	Location string
	Details  string
}

// ConciergeOptions configures a new ConciergeAgent.
type ConciergeOptions struct {
	Name                string
	Route               string
	VenueName           string
	Services            []string
	Amenities           map[string]Amenity
	Hours               string   // general hours of operation
	SpecialInstructions []string // optional additional instructions appended to the default list
	WelcomeMessage      string   // optional static greeting spoken at the start of the call
}

// ConciergeAgent acts as a virtual concierge for a venue, answering questions
// about amenities, services, hours, and directions.
type ConciergeAgent struct {
	*agent.AgentBase
	venueName string
	services  []string
	amenities map[string]Amenity
	hours     string
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// NewConciergeAgent creates an agent that provides concierge services for a venue.
func NewConciergeAgent(opts ConciergeOptions) *ConciergeAgent {
	name := opts.Name
	if name == "" {
		name = "concierge"
	}
	route := opts.Route
	if route == "" {
		route = "/concierge"
	}
	hours := opts.Hours
	if hours == "" {
		hours = "9 AM - 5 PM"
	}

	base := agent.NewAgentBase(
		agent.WithName(name),
		agent.WithRoute(route),
	)

	ca := &ConciergeAgent{
		AgentBase: base,
		venueName: opts.VenueName,
		services:  opts.Services,
		amenities: opts.Amenities,
		hours:     hours,
	}

	// ---- Prompt ----
	base.PromptAddSection("Personality",
		fmt.Sprintf("You are a professional and helpful virtual concierge for %s.", opts.VenueName),
		nil,
	)
	base.PromptAddSection("Goal",
		"Provide exceptional service by helping users with information, recommendations, and booking assistance.",
		nil,
	)
	instructions := []string{
		"Be warm and welcoming but professional at all times.",
		"Provide accurate information about amenities, services, and operating hours.",
		"Offer to help with reservations and bookings when appropriate.",
		"Answer questions concisely with specific, relevant details.",
	}
	instructions = append(instructions, opts.SpecialInstructions...)
	base.PromptAddSection("Instructions", "", instructions)

	// Services section
	if len(opts.Services) > 0 {
		base.PromptAddSection("Available Services",
			fmt.Sprintf("The following services are available: %s", strings.Join(opts.Services, ", ")),
			nil,
		)
	}

	// Amenities section with subsections
	if len(opts.Amenities) > 0 {
		base.PromptAddSection("Amenities",
			"Information about available amenities:",
			nil,
		)
		for aName, a := range opts.Amenities {
			var lines []string
			if a.Hours != "" {
				lines = append(lines, "Hours: "+a.Hours)
			}
			if a.Location != "" {
				lines = append(lines, "Location: "+a.Location)
			}
			if a.Details != "" {
				lines = append(lines, "Details: "+a.Details)
			}
			base.PromptAddSubsection("Amenities", titleCase(aName), strings.Join(lines, "\n"), nil)
		}
	}

	// Hours of operation
	base.PromptAddSection("Hours of Operation",
		fmt.Sprintf("General hours: %s", hours),
		nil,
	)

	// ---- Post-prompt ----
	base.SetPostPrompt(`Return a JSON summary of this interaction:
{
    "topic": "MAIN_TOPIC",
    "service_requested": "SPECIFIC_SERVICE_REQUESTED_OR_null",
    "questions_answered": ["QUESTION_1", "QUESTION_2"],
    "follow_up_needed": true/false
}`)

	// ---- Welcome message ----
	if opts.WelcomeMessage != "" {
		base.SetParam("static_greeting", opts.WelcomeMessage)
		base.SetParam("static_greeting_no_barge", true)
	}

	// ---- Global data ----
	amenityMaps := make(map[string]any, len(opts.Amenities))
	for k, a := range opts.Amenities {
		amenityMaps[k] = map[string]any{
			"hours":    a.Hours,
			"location": a.Location,
			"details":  a.Details,
		}
	}
	base.SetGlobalData(map[string]any{
		"venue_name": opts.VenueName,
		"services":   opts.Services,
		"amenities":  amenityMaps,
		"hours":      hours,
	})

	// ---- Hints ----
	hints := []string{opts.VenueName}
	hints = append(hints, opts.Services...)
	for k := range opts.Amenities {
		hints = append(hints, k)
	}
	base.AddHints(hints)

	// ---- Tools ----
	ca.registerTools()

	return ca
}

// ---------------------------------------------------------------------------
// Tool registration
// ---------------------------------------------------------------------------

func (ca *ConciergeAgent) registerTools() {
	// check_availability -----------------------------------------------
	ca.DefineTool(agent.ToolDefinition{
		Name:        "check_availability",
		Description: "Check availability for a service on a specific date and time",
		Parameters: map[string]any{
			"service": map[string]any{
				"type":        "string",
				"description": "The service to check (e.g., spa, restaurant)",
			},
			"date": map[string]any{
				"type":        "string",
				"description": "The date to check (YYYY-MM-DD format)",
			},
			"time": map[string]any{
				"type":        "string",
				"description": "The time to check (HH:MM format, 24-hour)",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			service := strings.ToLower(strings.TrimSpace(args["service"].(string)))
			date, _ := args["date"].(string)
			time, _ := args["time"].(string)

			// Check if the service is offered
			found := false
			for _, s := range ca.services {
				if strings.ToLower(s) == service {
					found = true
					break
				}
			}

			if found {
				return swaig.NewFunctionResult(
					fmt.Sprintf("Yes, %s is available on %s at %s. Would you like to make a reservation?", service, date, time),
				)
			}

			return swaig.NewFunctionResult(
				fmt.Sprintf("I'm sorry, we don't offer %s at %s. Our available services are: %s.",
					service, ca.venueName, strings.Join(ca.services, ", ")),
			)
		},
	})

	// get_directions ---------------------------------------------------
	ca.DefineTool(agent.ToolDefinition{
		Name:        "get_directions",
		Description: "Get directions to a specific location or amenity within the venue",
		Parameters: map[string]any{
			"location": map[string]any{
				"type":        "string",
				"description": "The location or amenity to get directions to",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			dest := strings.ToLower(strings.TrimSpace(args["location"].(string)))

			if amenity, ok := ca.amenities[dest]; ok && amenity.Location != "" {
				return swaig.NewFunctionResult(
					fmt.Sprintf("The %s is located at %s. From the main entrance, follow the signs to %s.",
						dest, amenity.Location, amenity.Location),
				)
			}

			return swaig.NewFunctionResult(
				fmt.Sprintf("I don't have specific directions to %s. You can ask our staff at the front desk for assistance.", dest),
			)
		},
	})
}

// titleCase returns s with the first letter of each word capitalised.
// This avoids the deprecated strings.Title.
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
