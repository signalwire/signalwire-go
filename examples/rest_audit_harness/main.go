// Example: rest_audit_harness
//
// Audit-only harness — drives a single REST operation against the loopback
// fixture spun up by porting-sdk's audit_rest_transport.py. Reads:
//
//   - REST_OPERATION       e.g. "calling.list_calls", "messaging.send"
//   - REST_FIXTURE_URL     "http://127.0.0.1:NNNN"
//   - REST_OPERATION_ARGS  JSON dict (forwarded to the operation)
//   - SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, SIGNALWIRE_SPACE
//
// Behavior: constructs a RestClient, overrides its base URL with
// REST_FIXTURE_URL, dispatches the operation, prints the parsed
// response as JSON to stdout, exits 0 on success / non-zero on error.
//
// Operation mapping: the audit operations are dotted names that don't
// always 1:1 with Go method names. The mapping below routes each dotted
// name to the appropriate Go REST method:
//
//   - calling.list_calls          → Compat.Calls.List   (LAML endpoint)
//   - messaging.send              → Compat.Messages.Create
//   - phone_numbers.list          → PhoneNumbers.List
//   - fabric.subscribers.list     → Fabric.Subscribers.List
//   - compatibility.calls.list    → Compat.Calls.List
//
// The Calling namespace in Go (and Python) is for relay-native command
// dispatch (POST /api/calling/calls), not for listing LAML-style calls.
// The audit's `calling.list_calls` is interpreted as the LAML endpoint
// because that's the only "list calls" operation the SDK exposes.
//
// Not for production use.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/pkg/rest"
)

func main() {
	op := os.Getenv("REST_OPERATION")
	if op == "" {
		die("REST_OPERATION required")
	}
	fixtureURL := os.Getenv("REST_FIXTURE_URL")
	if fixtureURL == "" {
		die("REST_FIXTURE_URL required")
	}

	rawArgs := os.Getenv("REST_OPERATION_ARGS")
	if rawArgs == "" {
		rawArgs = "{}"
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		die(fmt.Sprintf("REST_OPERATION_ARGS not JSON: %v", err))
	}

	// Construct the client. NewRestClient reads SIGNALWIRE_* env vars
	// when its arguments are empty. The audit sets all three, so the
	// constructor succeeds.
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		die(fmt.Sprintf("NewRestClient: %v", err))
	}

	// Override the base URL to point at the audit fixture. Without this
	// the client would hit https://127.0.0.1 (no port) which the audit
	// fixture wouldn't see.
	client.SetBaseURL(fixtureURL)

	var result map[string]any
	var opErr error
	switch op {
	case "calling.list_calls", "compatibility.calls.list":
		// LAML calls listing — Go's Compat.Calls embeds CrudResource
		// whose List method issues GET against the resource path.
		result, opErr = client.Compat.Calls.List(stringParams(args))
	case "messaging.send":
		// Send SMS — Compat.Messages.Create matches Twilio's POST
		// /Messages with To/From/Body in form fields. The audit
		// fixture serves the canned response on any POST.
		result, opErr = client.Compat.Messages.Create(remapMessagingArgs(args))
	case "phone_numbers.list":
		result, opErr = client.PhoneNumbers.List(stringParams(args))
	case "fabric.subscribers.list":
		result, opErr = client.Fabric.Subscribers.List(stringParams(args))
	default:
		die(fmt.Sprintf("unknown REST_OPERATION: %s", op))
	}

	if opErr != nil {
		die(fmt.Sprintf("operation %s failed: %v", op, opErr))
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(result); err != nil {
		die(fmt.Sprintf("encode result: %v", err))
	}
}

// stringParams converts a JSON-decoded args map (values may be int,
// float64, bool, etc.) to the map[string]string format Go's REST
// namespaces expect for query parameters.
func stringParams(args map[string]any) map[string]string {
	if len(args) == 0 {
		return nil
	}
	out := make(map[string]string, len(args))
	for k, v := range args {
		out[k] = fmt.Sprint(v)
	}
	return out
}

// remapMessagingArgs maps the audit's Python-style messaging args to
// the Twilio LAML form-field shape that Compat.Messages.Create expects:
// from_ → From, to → To, body → Body.
func remapMessagingArgs(args map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range args {
		switch k {
		case "from_", "from":
			out["From"] = v
		case "to":
			out["To"] = v
		case "body":
			out["Body"] = v
		default:
			out[k] = v
		}
	}
	return out
}

func die(msg string) {
	fmt.Fprintf(os.Stderr, "rest_audit_harness: %s\n", msg)
	os.Exit(1)
}
