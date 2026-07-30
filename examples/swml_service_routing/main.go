//go:build swexample

// Example: swml_service_routing
//
// SWML service with routing callbacks. A routing callback inspects the POST
// body and headers and returns a route string to redirect the request (HTTP
// 307), or nil to serve the default document. This mirrors the reference
// SWMLService.register_routing_callback (body, headers) -> route | None.
package main

import (
	"fmt"
	"os"

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
		swml.WithName("RoutingExample"),
		swml.WithRoute("/main"),
		swml.WithPort(3026),
	)

	// Build the default (main) SWML document
	must("answer", svc.Answer(nil, nil))
	greeting := "say:Hello from the main service!"
	must("play", svc.Play(swml.PlayOptions{URL: &greeting}))
	must("hangup", svc.Hangup(nil))

	// Register a routing callback at /dispatch: inspect the body and redirect
	// callers to a dedicated endpoint based on the "department" field.
	svc.RegisterRoutingCallback(func(body map[string]any, headers map[string]any) *string {
		dept, _ := body["department"].(string)
		switch dept {
		case "customer":
			route := "/customer"
			return &route
		case "product":
			route := "/product"
			return &route
		default:
			return nil // no redirect — serve the default document
		}
	}, "/dispatch")

	fmt.Println("Starting RoutingExample on :3026 ...")
	fmt.Println("  Main:     /main")
	fmt.Println("  Dispatch: POST /dispatch {\"department\":\"customer\"|\"product\"} -> 307 redirect")

	if err := svc.Serve(); err != nil {
		fmt.Printf("Service error: %v\n", err)
	}
}
