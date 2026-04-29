// Example: swmlservice_swaig_standalone
//
// Proves that swml.Service — by itself, with NO agent.AgentBase — can host
// SWAIG functions and serve them on its own /swaig endpoint.
//
// This is the path you take when you want a SWAIG-callable HTTP service that
// isn't an `<ai>` agent: the SWAIG verb is a generic LLM-tool surface and
// swml.Service is the host. agent.AgentBase is just a swml.Service composition
// that *also* layers in prompts, AI config, dynamic config, and token
// validation.
//
// Run:
//
//	go run examples/swmlservice_swaig_standalone/main.go
//
// Then exercise the endpoints:
//
//	curl -u u:p http://localhost:3000/standalone        # GET SWML doc
//	curl -u u:p http://localhost:3000/standalone/swaig \
//	    -H 'Content-Type: application/json' \
//	    -d '{"function":"lookup_competitor","argument":{"parsed":[{"competitor":"ACME"}]}}'
//
// Or drive it through the swaig-test CLI:
//
//	go run cmd/swaig-test/main.go --url http://u:p@localhost:3000/standalone --list-tools
//	go run cmd/swaig-test/main.go --url http://u:p@localhost:3000/standalone \
//	    --exec lookup_competitor --param competitor=ACME
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/swml"
)

func main() {
	svc := swml.NewService(
		swml.WithName("standalone"),
		swml.WithRoute("/standalone"),
		swml.WithBasicAuth("u", "p"),
		swml.WithPort(3000),
	)

	// 1. Build a minimal SWML document. Any verbs are fine — the SWAIG HTTP
	//    surface is independent of what the document contains.
	if err := svc.Answer(nil, nil); err != nil {
		fmt.Printf("answer error: %v\n", err)
		return
	}
	if err := svc.Hangup(nil); err != nil {
		fmt.Printf("hangup error: %v\n", err)
		return
	}

	// 2. Register a SWAIG function. DefineTool lives on swml.Service, not
	//    just AgentBase. The handler receives parsed arguments plus the raw
	//    POST body.
	svc.DefineTool(&swml.ToolDefinition{
		Name: "lookup_competitor",
		Description: "Look up competitor pricing by company name. Use this when " +
			"the user asks how a competitor's price compares to ours.",
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
					"%s pricing is $99/seat; we're $79/seat.", competitor,
				),
			}
		},
		Secure: false, // standalone services don't validate session tokens by default
	})

	pretty, err := svc.RenderPretty()
	if err != nil {
		fmt.Printf("render error: %v\n", err)
		return
	}
	fmt.Println("SWML Document:")
	fmt.Println(pretty)

	fmt.Println("\nStarting standalone SWAIG service on :3000/standalone ...")
	if err := svc.Serve(); err != nil {
		fmt.Printf("Service error: %v\n", err)
	}
}
