# SignalWire SWML Service Guide

## Table of Contents
- [Introduction](#introduction)
- [Installation](#installation)
- [Basic Usage](#basic-usage)
- [Centralized Logging System](#centralized-logging-system)
- [SWML Document Creation](#swml-document-creation)
- [Verb Handling](#verb-handling)
- [Web Service Features](#web-service-features)
- [Custom Routing Callbacks](#custom-routing-callbacks)
- [Advanced Usage](#advanced-usage)
- [API Reference](#api-reference)
- [Examples](#examples)

## Introduction

The `swml.Service` type (package `github.com/signalwire/signalwire-go/v3/pkg/swml`) provides a foundation for creating and serving SignalWire Markup Language (SWML) documents. It underpins all SignalWire services, including AI Agents, and handles common tasks such as:

- SWML document creation and manipulation
- Schema validation
- Web service functionality
- Authentication
- Centralized logging

The type is designed to be embedded or configured for specific use cases, while providing a full set of capabilities out of the box.

## Installation

The `swml.Service` type is part of the SignalWire AI Agents Go SDK. Add it to your module with `go get`:

```bash
go get github.com/signalwire/signalwire-go/v3
```

Then import the `swml` package:

<!-- snippet: no-compile illustrative import statement only -->
```go
import "github.com/signalwire/signalwire-go/v3/pkg/swml"
```

## Basic Usage

Here's a simple example of creating an SWML service. A service is created with
functional options and its document is built with typed verb methods:

```go
package main

import "github.com/signalwire/signalwire-go/v3/pkg/swml"

func main() {
	svc := swml.NewService(
		swml.WithName("voice-service"),
		swml.WithRoute("/voice"),
		swml.WithHost("0.0.0.0"),
		swml.WithPort(3000),
	)

	// Build the SWML document with typed verb methods.
	svc.Answer(nil, nil)

	greeting := "say:Hello, thank you for calling our service."
	svc.Play(swml.PlayOptions{URL: &greeting})

	svc.Hangup(nil)

	// Serve the document over HTTP (blocks).
	if err := svc.Serve(); err != nil {
		panic(err)
	}
}
```

For any verb without a dedicated typed method, use the generic
`ExecuteVerb(verbName string, config any)`:

<!-- snippet-setup -->
```go
import "github.com/signalwire/signalwire-go/v3/pkg/swml"

// Shared service established in prose above.
var svc = swml.NewService(swml.WithName("svc"))
var document = ""

var (
	_ = svc
	_ = document
)
```

```go
svc.ExecuteVerb("answer", map[string]any{})
svc.ExecuteVerb("play", map[string]any{
	"url": "say:Hello, thank you for calling our service.",
})
svc.ExecuteVerb("hangup", map[string]any{})
```

## Centralized Logging System

The `swml.Service` type includes a centralized structured logging system (package `github.com/signalwire/signalwire-go/v3/pkg/logging`) that provides consistent, level-based logs. The logger is configured automatically, so you don't need to set it up in each service or example.

### How It Works

1. The `logging` package configures itself from the environment on first use
2. Each `swml.Service` instance exposes a logger bound to its service name via the `Logger` field
3. All logs include contextual information like service name and log level
4. Log level and mode are controlled by `SIGNALWIRE_LOG_LEVEL` and `SIGNALWIRE_LOG_MODE`

### Using the Logger

Every `swml.Service` instance has a `Logger` field that can be used for logging. Each method takes a `printf`-style format string and arguments:

```go
someOperation := func() error { return nil } // your operation

// Basic logging
svc.Logger.Info("service started")

// Logging with context
svc.Logger.Debug("document created, size=%d", len(document))

// Error logging
if err := someOperation(); err != nil {
	svc.Logger.Error("operation failed: %s", err)
}
```

### Log Levels

The following log levels are available (in increasing order of severity):
- `debug`: Detailed information for debugging
- `info`: General information about operation
- `warning`: Warning about potential issues
- `error`: Error information when operations fail
- `critical`: Critical error that might cause the application to terminate

### Suppressing Logs

To suppress logs when running a service, set the log level via the environment before starting:

```bash
export SIGNALWIRE_LOG_LEVEL=warning  # Only show warnings and above
```

You can also disable logging output entirely with the log mode:

```bash
export SIGNALWIRE_LOG_MODE=off
```

## SWML Document Creation

The `SWMLService` class provides methods for creating and manipulating SWML documents.

### Document Structure

SWML documents have the following basic structure:

```json
{
  "version": "1.0.0",
  "sections": {
    "main": [
      { "verb1": { /* configuration */ } },
      { "verb2": { /* configuration */ } }
    ],
    "section1": [
      { "verb3": { /* configuration */ } }
    ]
  }
}
```

### Document Methods

- `ResetDocument()`: Reset the document to an empty state
- `ExecuteVerb(verbName string, config any) error`: Add a verb to the main section
- `AddSection(name string) bool`: Add a new section
- `ExecuteVerbToSection(section, verbName string, config any) error`: Add a verb to a specific section
- `GetDocument() *Document`: Get the current document model
- `Render() (string, error)`: Get the current document as a JSON string (`RenderPretty` for indented output)

### Common Verb Shortcuts

In addition to the generic `ExecuteVerb`, `swml.Service` provides typed methods for common verbs: `Answer`, `Play`, `Say`, `Record`, `Connect`, `Switch`, `Prompt`, `Hangup`, and many more (see the [API Reference](#api-reference)).

## Verb Handling

The `SWMLService` class provides validation for SWML verbs using the SignalWire schema.

### Verb Validation

When adding a verb, the service validates it against the schema to ensure it has the correct structure and parameters. `ExecuteVerb` returns an `error` when validation fails.

```go
// This will validate the configuration against the schema
svc.ExecuteVerb("play", map[string]any{
	"url":    "say:Hello, world!",
	"volume": 5,
})

// This returns a non-nil error (invalid parameter)
if err := svc.ExecuteVerb("play", map[string]any{
	"invalid_param": "value",
}); err != nil {
	svc.Logger.Error("verb validation failed: %s", err)
}
```

### Custom Verb Handlers

You can register custom verb handlers for specialized verb processing. A handler implements the `swml.VerbHandler` interface:

```go
package main

import "github.com/signalwire/signalwire-go/v3/pkg/swml"

// CustomPlayHandler implements swml.VerbHandler for the "play" verb.
type CustomPlayHandler struct{}

func (h CustomPlayHandler) GetVerbName() string { return "play" }

func (h CustomPlayHandler) ValidateConfig(config map[string]any) (bool, []string) {
	// Custom validation logic
	return true, nil
}

func (h CustomPlayHandler) BuildConfig(params map[string]any) (map[string]any, error) {
	// Custom configuration building
	return params, nil
}

func main() {
	svc := swml.NewService(swml.WithName("my-service"))

	// Register it on the service.
	svc.RegisterVerbHandler(CustomPlayHandler{})
}
```

## Web Service Features

The `SWMLService` class includes built-in web service capabilities for serving SWML documents.

### Endpoints

By default, a service provides the following endpoints:

- `GET /route`: Return the SWML document
- `POST /route`: Process request data and return the SWML document
- `GET /route/`: Same as above but with trailing slash
- `POST /route/`: Same as above but with trailing slash

Where `route` is the route path specified when creating the service.

### Authentication

Basic authentication is automatically set up for all endpoints. Credentials are generated if not provided, or can be specified with the `WithBasicAuth` option:

```go
svc = swml.NewService(
	swml.WithName("my-service"),
	swml.WithBasicAuth("username", "password"),
)
```

You can also set credentials using environment variables:
- `SWML_BASIC_AUTH_USER`
- `SWML_BASIC_AUTH_PASSWORD`

### Dynamic SWML Generation

To customize SWML documents based on request data, register a routing callback
(see below) or build the document based on the parsed POST body. The framework
dispatches each request through `HandleRequest`, which consults registered
routing callbacks and then serves the document. A common pattern is to inspect
the request body and rebuild the document before serving:

```go
package main

import "github.com/signalwire/signalwire-go/v3/pkg/swml"

// buildDocument (re)builds the SWML document from the parsed request body.
func buildDocument(svc *swml.Service, requestData map[string]any) {
	// Reset the document to start fresh.
	svc.ResetDocument()
	svc.Answer(nil, nil)

	// Add custom verbs based on the request data.
	if callerType, _ := requestData["caller_type"].(string); callerType == "vip" {
		vip := "say:Welcome VIP caller!"
		svc.Play(swml.PlayOptions{URL: &vip})
	} else {
		std := "say:Welcome caller!"
		svc.Play(swml.PlayOptions{URL: &std})
	}
}

func main() {
	svc := swml.NewService(swml.WithName("my-service"))
	buildDocument(svc, map[string]any{})
}
```

## Custom Routing Callbacks

The `SWMLService` class allows you to register custom routing callbacks that can examine incoming requests and determine where they should be routed.

### Registering a Routing Callback

You can use the `RegisterRoutingCallback` method to register a function that will be called to process requests to a specific path. The callback receives the parsed body and headers and returns a `*string`: a non-nil route to redirect to (HTTP 307), or `nil` to process normally:

```go
// A RoutingCallback inspects the body and headers and returns a route to
// redirect to (*string), or nil to process the request normally.
svc.RegisterRoutingCallback("/customer", func(body map[string]any, headers map[string]any) *string {
	// Example: route based on a field in the request body.
	if customerID, ok := body["customer_id"].(string); ok {
		route := "/customer/" + customerID
		return &route
	}
	// Process the request normally.
	return nil
})
```

### How Routing Works

1. When a request is received at the registered path, the routing callback is executed
2. The callback inspects the request and can decide whether to redirect it
3. If the callback returns a non-nil route (`*string`), the request is redirected with HTTP 307 (temporary redirect)
4. If the callback returns `nil`, the request is processed normally and the current document is served

### Serving Different Content for Different Paths

To serve different content for different paths, register a routing callback per
path that redirects (returns a route) to a dedicated endpoint hosting the right
document. Each endpoint can be a separate `swml.Service` mounted on its own route
(see the [`AgentServer`](#) multi-service pattern), or the callback can redirect
to a distinct URL that serves the appropriate SWML:

```go
svc.RegisterRoutingCallback("/dispatch", func(body map[string]any, headers map[string]any) *string {
	// Redirect to a dedicated endpoint based on the request.
	if _, ok := body["customer_id"]; ok {
		route := "/customer"
		return &route
	}
	if _, ok := body["product_id"]; ok {
		route := "/product"
		return &route
	}
	// No redirect — serve the default document.
	return nil
})
```

### Example: Multi-Section Service

Here's an example of a service that uses routing callbacks to redirect different
types of requests to dedicated endpoints. The main service builds a default
document, and the callback at `/dispatch` redirects to `/customer` or `/product`
based on the request body:

```go
package main

import "github.com/signalwire/signalwire-go/v3/pkg/swml"

func main() {
	svc := swml.NewService(
		swml.WithName("multi-section"),
		swml.WithRoute("/main"),
	)

	// Build the main (default) document.
	svc.Answer(nil, nil)
	greeting := "say:Hello from the main service!"
	svc.Play(swml.PlayOptions{URL: &greeting})
	svc.Hangup(nil)

	// Register a routing callback at /dispatch that redirects based on the body.
	svc.RegisterRoutingCallback("/dispatch", func(body map[string]any, headers map[string]any) *string {
		if customerID, ok := body["customer_id"].(string); ok {
			// In a real implementation, redirect to a customer-specific endpoint.
			svc.Logger.Info("routing request for customer ID: %s", customerID)
			route := "/customer"
			return &route
		}
		if productID, ok := body["product_id"].(string); ok {
			svc.Logger.Info("routing request for product ID: %s", productID)
			route := "/product"
			return &route
		}
		// No redirect — serve the default document.
		return nil
	})

	if err := svc.Serve(); err != nil {
		panic(err)
	}
}
```

In this example:
1. The service builds a default document served at `/main`
2. A routing callback at `/dispatch` inspects the request body
3. The callback returns a route (`*string`) to redirect the caller (HTTP 307), or `nil` to serve the default
4. Dedicated endpoints (e.g. `/customer`, `/product`) serve their own documents

See `examples/swml_service_routing/main.go` for a complete runnable version.

## Advanced Usage

### Mounting into a Larger HTTP Application

`AsRouter` returns an `http.Handler` for the service that you can mount into a
larger `net/http` application:

```go
import "net/http"

svc = swml.NewService(swml.WithName("my-service"))

mux := http.NewServeMux()
mux.Handle("/voice/", http.StripPrefix("/voice", svc.AsRouter()))

http.ListenAndServe(":8080", mux)
```

### Schema Path Customization

You can specify a custom path to the schema file with the `WithSchemaPath` option:

```go
svc = swml.NewService(
	swml.WithName("my-service"),
	swml.WithSchemaPath("/path/to/schema.json"),
)
```

## API Reference

### Constructor Options

- `WithName(name)`: Service name/identifier (required)
- `WithRoute(route)`: HTTP route path (default: "/")
- `WithHost(host)`: Host to bind to (default: "0.0.0.0")
- `WithPort(port)`: Port to bind to (default: 3000)
- `WithBasicAuth(user, password)`: Optional basic-auth credentials
- `WithSchemaPath(path)`: Optional path to schema.json

Log verbosity is controlled by the `SIGNALWIRE_LOG_LEVEL` / `SIGNALWIRE_LOG_MODE`
environment variables rather than a constructor option.

### Document Methods

- `ResetDocument()`
- `ExecuteVerb(verbName, config)`
- `AddSection(name)`
- `ExecuteVerbToSection(section, verbName, config)`
- `GetDocument()`
- `Render()` / `RenderPretty()`

### Service Methods

- `AsRouter() http.Handler`: Get an `http.Handler` for the service
- `Serve() error`: Start the service
- `Stop() error`: Stop the service
- `GetBasicAuthCredentials() (string, string)`: Get the basic auth credentials
- `OnRequest(requestData, callbackPath)`: Called when SWML is requested
- `RegisterRoutingCallback(path, cb)`: Register a callback for request routing

### Verb Helper Methods

- `ExecuteVerb(verbName, config)`: Add any SWML verb with configuration; typed methods (`Answer`, `Play`, `Record`, `Connect`, `Switch`, ...) cover the common verbs

## Examples

### Basic Voicemail Service

```go
package main

import "github.com/signalwire/signalwire-go/v3/pkg/swml"

func main() {
	svc := swml.NewService(
		swml.WithName("voicemail"),
		swml.WithRoute("/voicemail"),
		swml.WithHost("0.0.0.0"),
		swml.WithPort(3000),
	)

	buildVoicemailDocument(svc)

	if err := svc.Serve(); err != nil {
		panic(err)
	}
}

// buildVoicemailDocument builds the voicemail SWML document.
func buildVoicemailDocument(svc *swml.Service) {
	// Reset the document.
	svc.ResetDocument()

	// Answer the call.
	svc.Answer(nil, nil)

	// Play the greeting.
	greeting := "say:Hello, you've reached the voicemail service. Please leave a message after the beep."
	svc.Play(swml.PlayOptions{URL: &greeting})

	// Play a beep.
	beep := "https://example.com/beep.wav"
	svc.Play(swml.PlayOptions{URL: &beep})

	// Record the message.
	svc.Record(map[string]any{
		"format":      "mp3",
		"stereo":      false,
		"max_length":  120, // 2 minutes max
		"terminators": "#",
	})

	// Thank the caller.
	thanks := "say:Thank you for your message. Goodbye!"
	svc.Play(swml.PlayOptions{URL: &thanks})

	// Hang up.
	svc.Hangup(nil)

	svc.Logger.Debug("voicemail document built")
}
```

### Dynamic Call Routing Service

This example builds the document from the parsed request body, then serves it.
`buildCallRouterDocument` inspects the `department` field and connects the caller
to the right number:

```go
package main

import (
	"fmt"
	"strings"

	"github.com/signalwire/signalwire-go/v3/pkg/swml"
)

func main() {
	svc := swml.NewService(swml.WithName("call-router"))
	buildCallRouterDocument(svc, map[string]any{})
}

// buildCallRouterDocument builds the routing document from the request body.
func buildCallRouterDocument(svc *swml.Service, requestData map[string]any) {
	// If there's no request data, keep the default document.
	if len(requestData) == 0 {
		svc.Logger.Debug("no request data, using default")
		return
	}

	// Create a new document.
	svc.ResetDocument()
	svc.Answer(nil, nil)

	// Get routing parameters.
	department, _ := requestData["department"].(string)
	department = strings.ToLower(department)

	// Greeting.
	greeting := fmt.Sprintf("say:Thank you for calling our %s department. Please hold.", department)
	svc.Play(swml.PlayOptions{URL: &greeting})

	// Route based on department.
	phoneNumbers := map[string]string{
		"sales":   "+15551112222",
		"support": "+15553334444",
		"billing": "+15555556666",
	}
	toNumber, ok := phoneNumbers[department]
	if !ok {
		toNumber = "+15559990000"
	}

	// Connect to the department.
	svc.Connect(map[string]any{
		"to":              toNumber,
		"timeout":         30,
		"answer_on_bridge": true,
	})

	// Fallback message and hangup.
	fallback := "say:We're sorry, but all of our agents are currently busy. Please try again later."
	svc.Play(swml.PlayOptions{URL: &fallback})
	svc.Hangup(nil)
}
```

For more examples, see the `examples` directory (e.g. `examples/swml_service`,
`examples/swml_service_routing`) in the SignalWire AI Agents Go SDK repository.