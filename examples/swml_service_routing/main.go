//go:build ignore

// Example: swml_service_routing
//
// SWML service with routing callbacks. Demonstrates registering multiple
// routing callbacks on a single SWMLService to serve different SWML
// content based on the request path (/main, /customer, /product).
package main

import (
	"fmt"
	"net/http"

	"github.com/signalwire/signalwire-go/pkg/swml"
)

func main() {
	svc := swml.NewService(
		swml.WithName("RoutingExample"),
		swml.WithRoute("/main"),
		swml.WithPort(3026),
	)

	// Build the default (main) SWML document
	svc.Answer(nil, nil)
	greeting := "say:Hello from the main service!"
	svc.Play(&greeting, nil, nil, nil, nil, nil, nil)
	svc.Hangup(nil)

	// Register a customer routing callback
	svc.RegisterRoutingCallback("/customer", func(r *http.Request, body map[string]any) map[string]any {
		fmt.Println("Serving customer route")
		doc := swml.NewDocument()
		doc.AddVerb("answer", map[string]any{})
		doc.AddVerb("play", map[string]any{"url": "say:Hello from the customer service!"})
		doc.AddVerb("hangup", map[string]any{})
		return doc.ToMap()
	})

	// Register a product routing callback
	svc.RegisterRoutingCallback("/product", func(r *http.Request, body map[string]any) map[string]any {
		fmt.Println("Serving product route")
		doc := swml.NewDocument()
		doc.AddVerb("answer", map[string]any{})
		doc.AddVerb("play", map[string]any{"url": "say:Hello from the product service!"})
		doc.AddVerb("hangup", map[string]any{})
		return doc.ToMap()
	})

	fmt.Println("Starting RoutingExample on :3026 ...")
	fmt.Println("  Main:     /main")
	fmt.Println("  Customer: /customer")
	fmt.Println("  Product:  /product")

	if err := svc.Serve(); err != nil {
		fmt.Printf("Service error: %v\n", err)
	}
}
