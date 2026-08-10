//go:build swexample

// Example: dynamic_swml_service
//
// SWML service with a routing callback. A routing callback inspects the POST
// body and headers and returns a route string to redirect the request (HTTP
// 307) — for example, sending VIP callers to a dedicated priority endpoint — or
// nil to serve the default document. This mirrors the reference
// SWMLService.register_routing_callback (body, headers) -> route | None.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/signalwire/signalwire-go/v3/pkg/swml"
)

// must stops the example on a verb error. Each svc.* call returns an error when
// the verb is rejected (bad arguments, unknown verb); ignoring it would render a
// document that is silently missing a step, which is the last thing a demo
// should teach.
func must(what string, err error) {
	if err != nil {
		fmt.Printf("%s failed: %v\n", what, err)
		os.Exit(1)
	}
}

func main() {
	svc := swml.NewService(
		swml.WithName("DynamicGreeting"),
		swml.WithRoute("/greeting"),
		swml.WithPort(3024),
	)

	// Build a default SWML document
	must("answer", svc.Answer(nil, nil))
	greeting := "say:Hello, thank you for calling our service."
	must("play", svc.Play(swml.PlayOptions{URL: &greeting}))
	must("prompt", svc.Prompt(map[string]any{
		"play":        "say:Press 1 for sales, 2 for support, or 3 to leave a message.",
		"max_digits":  1,
		"terminators": "#",
	}))
	must("hangup", svc.Hangup(nil))

	// Register a routing callback: redirect VIP callers to a priority endpoint;
	// everyone else falls through to the default document.
	svc.RegisterRoutingCallback(func(body map[string]any, headers map[string]any) *string {
		callerType, _ := body["caller_type"].(string)
		if strings.ToLower(callerType) == "vip" {
			route := "/priority"
			return &route
		}
		return nil // no redirect — serve the default document
	}, "/greeting")

	fmt.Println("Starting DynamicGreeting on :3024/greeting ...")
	fmt.Println("  Default: generic menu")
	fmt.Println("  VIP:     POST {\"caller_type\":\"vip\"} -> 307 redirect to /priority")

	if err := svc.Serve(); err != nil {
		fmt.Printf("Service error: %v\n", err)
	}
}
