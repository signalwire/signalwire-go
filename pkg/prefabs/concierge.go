package prefabs

import (
	"fmt"
	"sort"
	"strings"

	"github.com/signalwire/signalwire-go/v3/pkg/agent"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
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
	Name      string
	Route     string
	VenueName string
	Services  []string
	Amenities map[string]Amenity
	// Hours is the single-line shorthand for HoursOfOperation: it is equivalent to
	// HoursOfOperation{"default": Hours}. Ignored when HoursOfOperation is set.
	Hours string
	// HoursOfOperation is the labelled operating-hours map the reference takes
	// (`hours_of_operation: dict[str, str]`), e.g.
	// {"weekdays": "9 AM - 6 PM", "saturday": "10 AM - 2 PM"}. Each entry renders
	// as its own "<Label>: <hours>" line in the Hours of Operation prompt section,
	// exactly as the reference does. Defaults to {"default": "9 AM - 5 PM"}.
	HoursOfOperation    map[string]string
	SpecialInstructions []string // optional additional instructions appended to the default list
	WelcomeMessage      string   // optional static greeting spoken at the start of the call
}

// ConciergeAgent acts as a virtual concierge for a venue, answering questions
// about amenities, services, hours, and directions.
type ConciergeAgent struct {
	*agent.AgentBase
	venueName           string
	services            []string
	amenities           map[string]Amenity
	hoursOfOperation    map[string]string
	specialInstructions []string
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
	// hours_of_operation is a LABELLED MAP in the reference; Hours is the
	// single-line shorthand for {"default": Hours}. Default matches the
	// reference's `hours_of_operation or {"default": "9 AM - 5 PM"}`.
	hoursOfOperation := opts.HoursOfOperation
	if len(hoursOfOperation) == 0 {
		label := opts.Hours
		if label == "" {
			label = "9 AM - 5 PM"
		}
		hoursOfOperation = map[string]string{"default": label}
	}

	base := agent.NewAgentBase(
		agent.WithName(name),
		agent.WithRoute(route),
	)

	ca := &ConciergeAgent{
		AgentBase:           base,
		venueName:           opts.VenueName,
		services:            opts.Services,
		amenities:           opts.Amenities,
		hoursOfOperation:    hoursOfOperation,
		specialInstructions: opts.SpecialInstructions,
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
	instructions := make([]string, 0, 4+len(opts.SpecialInstructions))
	instructions = append(instructions,
		"Be warm and welcoming but professional at all times.",
		"Provide accurate information about amenities, services, and operating hours.",
		"Offer to help with reservations and bookings when appropriate.",
		"Answer questions concisely with specific, relevant details.",
	)
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

	// Hours of operation. The reference joins one "<Title>: <hours>" line per
	// map entry with no prefix (concierge.py:132-136) — match it exactly, and
	// sort the keys so the rendered document is deterministic (Go map iteration
	// order is randomised, which would otherwise make the prompt unstable).
	hoursLabels := make([]string, 0, len(hoursOfOperation))
	for label := range hoursOfOperation {
		hoursLabels = append(hoursLabels, label)
	}
	sort.Strings(hoursLabels)
	hoursLines := make([]string, 0, len(hoursLabels))
	for _, label := range hoursLabels {
		hoursLines = append(hoursLines, titleCase(label)+": "+hoursOfOperation[label])
	}
	base.PromptAddSection("Hours of Operation", strings.Join(hoursLines, "\n"), nil)

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
		"hours":      hoursOfOperation,
	})

	// ---- Hints ----
	hints := make([]string, 0, 1+len(opts.Services)+len(opts.Amenities))
	hints = append(hints, opts.VenueName)
	hints = append(hints, opts.Services...)
	for k := range opts.Amenities {
		hints = append(hints, k)
	}
	base.AddHints(hints)

	// ---- Tools ----
	ca.registerTools()

	// ---- Summary callback ----
	ca.AgentBase.OnSummary(ca.OnSummary)

	return ca
}

// ---------------------------------------------------------------------------
// Configuration readers
// ---------------------------------------------------------------------------
//
// The reference stores each constructor argument as a public attribute
// (concierge.py:75-79), so a caller who configured the agent can read the
// configuration back. These are the Go accessors over the same state.

// VenueName returns the venue this concierge represents (Python: venue_name).
func (ca *ConciergeAgent) VenueName() string { return ca.venueName }

// Services returns the list of services offered (Python: services). A copy is
// returned so a caller cannot mutate the agent's configured list.
func (ca *ConciergeAgent) Services() []string {
	return append([]string(nil), ca.services...)
}

// Amenities returns the configured amenities keyed by name (Python: amenities).
// A copy is returned so a caller cannot mutate the agent's configuration.
func (ca *ConciergeAgent) Amenities() map[string]Amenity {
	out := make(map[string]Amenity, len(ca.amenities))
	for k, v := range ca.amenities {
		out[k] = v
	}
	return out
}

// HoursOfOperation returns the labelled operating hours (Python:
// hours_of_operation). A copy is returned so a caller cannot mutate the agent's
// configuration.
func (ca *ConciergeAgent) HoursOfOperation() map[string]string {
	out := make(map[string]string, len(ca.hoursOfOperation))
	for k, v := range ca.hoursOfOperation {
		out[k] = v
	}
	return out
}

// SpecialInstructions returns the extra instructions appended to the agent's
// Instructions prompt section (Python: special_instructions). A copy is returned
// so a caller cannot mutate the agent's configuration.
func (ca *ConciergeAgent) SpecialInstructions() []string {
	return append([]string(nil), ca.specialInstructions...)
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
		Handler: ca.CheckAvailability,
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
		Handler: ca.GetDirections,
	})
}

// CheckAvailability handles the "check_availability" tool: it checks whether a
// requested service is offered by the venue on a given date and time.
func (ca *ConciergeAgent) CheckAvailability(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
	serviceRaw, _ := args["service"].(string)
	service := strings.ToLower(strings.TrimSpace(serviceRaw))
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
}

// GetDirections handles the "get_directions" tool: it returns directions to a
// named location or amenity within the venue.
func (ca *ConciergeAgent) GetDirections(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
	destRaw, _ := args["location"].(string)
	dest := strings.ToLower(strings.TrimSpace(destRaw))

	if amenity, ok := ca.amenities[dest]; ok && amenity.Location != "" {
		return swaig.NewFunctionResult(
			fmt.Sprintf("The %s is located at %s. From the main entrance, follow the signs to %s.",
				dest, amenity.Location, amenity.Location),
		)
	}

	return swaig.NewFunctionResult(
		fmt.Sprintf("I don't have specific directions to %s. You can ask our staff at the front desk for assistance.", dest),
	)
}

// OnSummary is the summary hook for the concierge agent. It matches the
// agent.SummaryCallback signature and is registered via
// ca.AgentBase.OnSummary in the constructor. There is currently no
// concierge-specific summary logic (the post-prompt already emits a JSON
// summary), so this is a no-op placeholder that mirrors Python's on_summary
// surface.
func (ca *ConciergeAgent) OnSummary(summary map[string]any, rawData map[string]any) {
	_ = summary
	_ = rawData
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
