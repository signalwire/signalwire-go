// Example: swmlservice_ai_sidecar
//
// Proves that swml.Service can emit the `ai_sidecar` verb, register SWAIG
// tools the sidecar's LLM can call, and dispatch them end-to-end — without
// any agent.AgentBase code path.
//
// The `ai_sidecar` verb runs an AI listener alongside an in-progress call
// (real-time copilot, transcription analyzer, compliance monitor, etc.). It
// is NOT an agent — it does not own the call. So the right host is
// swml.Service, not agent.AgentBase.
//
// Run:
//
//	go run examples/swmlservice_ai_sidecar/main.go
//
// What this serves:
//
//	GET  /sales-sidecar           → SWML doc with the ai_sidecar verb
//	POST /sales-sidecar/swaig     → SWAIG tool dispatch (used by the sidecar's LLM)
//	POST /sales-sidecar/events    → optional event sink for sidecar lifecycle events
//
// Drive the SWAIG path through the swaig-test CLI:
//
//	go run cmd/swaig-test/main.go --url http://u:p@localhost:3000/sales-sidecar --list-tools
//	go run cmd/swaig-test/main.go --url http://u:p@localhost:3000/sales-sidecar \
//	    --exec lookup_competitor --param competitor=ACME
package main

import (
	"fmt"
	"net/http"

	"github.com/signalwire/signalwire-go/pkg/swml"
)

func main() {
	publicURL := "https://your-host.example.com/sales-sidecar"

	svc := swml.NewService(
		swml.WithName("sales-sidecar"),
		swml.WithRoute("/sales-sidecar"),
		swml.WithBasicAuth("u", "p"),
		swml.WithPort(3000),
	)

	// 1. Emit any SWML — including ai_sidecar. swml.Service's
	//    AddVerbToSection accepts arbitrary verb dicts, so new platform
	//    verbs work without an SDK release.
	if err := svc.Answer(nil, nil); err != nil {
		fmt.Printf("answer error: %v\n", err)
		return
	}

	if err := svc.GetDocument().AddVerbToSection("main", "ai_sidecar", map[string]any{
		"prompt": "You are a real-time sales copilot. Listen to the call " +
			"and surface competitor pricing comparisons when relevant.",
		"lang":      "en-US",
		"direction": []string{"remote-caller", "local-caller"},
		// Where the sidecar POSTs lifecycle/transcription events. Optional —
		// remove this key if you don't need an event sink.
		"url": publicURL + "/events",
		// Where the sidecar's LLM POSTs SWAIG tool calls. This service's
		// /swaig route is what answers them. NOTE: UPPERCASE SWAIG key.
		"SWAIG": map[string]any{
			"defaults": map[string]any{"web_hook_url": publicURL + "/swaig"},
		},
	}); err != nil {
		fmt.Printf("ai_sidecar error: %v\n", err)
		return
	}

	if err := svc.Hangup(nil); err != nil {
		fmt.Printf("hangup error: %v\n", err)
		return
	}

	// 2. Register tools the sidecar's LLM can call. Same DefineTool you'd
	//    use on AgentBase — it lives on swml.Service.
	svc.DefineTool(&swml.ToolDefinition{
		Name: "lookup_competitor",
		Description: "Look up competitor pricing by company name. The sidecar " +
			"should call this whenever the caller mentions a competitor.",
		Parameters: map[string]any{
			"competitor": map[string]any{
				"type":        "string",
				"description": "The competitor's company name, e.g. 'ACME'.",
			},
		},
		Handler: func(args map[string]any, raw map[string]any) any {
			competitor, _ := args["competitor"].(string)
			if competitor == "" {
				competitor = "<unknown>"
			}
			return map[string]any{
				"response": fmt.Sprintf(
					"Pricing for %s: $99/seat. Our equivalent plan is "+
						"$79/seat with the same SLA.", competitor,
				),
			}
		},
		Secure: false,
	})

	// 3. (Optional) Mount an event sink for ai_sidecar lifecycle events at
	//    POST /sales-sidecar/events. Remove this if you don't need it; the
	//    sidecar runtime POSTs each event as JSON.
	svc.RegisterRoutingCallback("/events", func(r *http.Request, body map[string]any) map[string]any {
		eventType, _ := body["type"].(string)
		if eventType == "" {
			eventType = "<unknown>"
		}
		fmt.Printf("[sidecar event] type=%s body=%v\n", eventType, body)
		return nil // nil means: fall through to default doc; non-nil overrides it.
	})

	pretty, err := svc.RenderPretty()
	if err != nil {
		fmt.Printf("render error: %v\n", err)
		return
	}
	fmt.Println("SWML Document:")
	fmt.Println(pretty)

	fmt.Println("\nStarting sales-sidecar service on :3000/sales-sidecar ...")
	if err := svc.Serve(); err != nil {
		fmt.Printf("Service error: %v\n", err)
	}
}
