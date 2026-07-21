// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Command swaig-http-dump is the Go port's SWAIG-HTTP-INVOKE dump program for
// the cross-port behavioral differ (porting-sdk/scripts/diff_port_swaig_http.py,
// corpus porting-sdk/scripts/swaig_http_corpus.py).
//
// It stands up an AgentBase with a `lookup_order` tool whose handler RECORDS the
// args it received, mounts the agent's real /swaig route over a live httptest
// HTTP server, POSTs each corpus fixture body to it over a genuine HTTP round
// trip (the dispatch path the platform actually drives — NOT an in-process
// handler call, which is exactly the blind spot that hid the GO-7 bug), and
// prints ONE JSON object mapping
//
//	fixture_id -> {"args_unwrapped": bool, "handler_saw_real_args": bool}
//
// to stdout. args_unwrapped is true when the handler was handed a FLAT args dict
// whose keys are the real argument names (order_id/customer, NOT parsed/raw).
// handler_saw_real_args is true when every expected key->value the fixture armed
// round-tripped to the handler. A port that passes the nested {parsed,raw}
// envelope through reds BOTH (the pre-GO-7 go behavior). The corpus is embedded
// here byte-identically to swaig_http_corpus.py; the differ keys our output by
// fixture id against the python oracle.
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/swaig-http-dump
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"

	"github.com/signalwire/signalwire-go/v3/pkg/agent"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
)

// The stable tool name every port registers (swaig_http_corpus.FUNCTION).
const functionName = "lookup_order"

// basic-auth creds we set explicitly so we can authenticate the POST.
const (
	authUser = "u"
	authPass = "p"
)

// fixture mirrors one swaig_http_corpus.CORPUS entry.
type fixture struct {
	id   string
	args map[string]any // the real args the handler MUST end up receiving
	body map[string]any // the exact SWAIG POST body (kind-specific envelope)
}

// The real args each fixture arms (swaig_http_corpus._NESTED_ARGS / _FLAT_ARGS).
var nestedArgs = map[string]any{"order_id": "ORD-3007", "customer": "acme-42"}
var flatArgs = map[string]any{"order_id": "FLAT-9911"}

// nestedBody / flatArgumentsBody mirror the corpus body builders.
func nestedBody(args map[string]any) map[string]any {
	raw, err := json.Marshal(args)
	if err != nil {
		panic(err) // fixture args are static, plain JSON — marshal never fails
	}
	return map[string]any{
		"function": functionName,
		"argument": map[string]any{"parsed": []any{args}, "raw": string(raw)},
	}
}

func flatArgumentsBody(args map[string]any) map[string]any {
	return map[string]any{"function": functionName, "arguments": args}
}

var corpus = []fixture{
	// platform_nested — the shape the real platform sends. A correct handler
	// unwraps argument.parsed[0] -> {order_id, customer}.
	{id: "platform_nested", args: nestedArgs, body: nestedBody(nestedArgs)},
	// flat_arguments — the {"arguments":{...}} fallback python + the platform accept.
	{id: "flat_arguments", args: flatArgs, body: flatArgumentsBody(flatArgs)},
}

// classification is the per-fixture artifact the differ byte-compares.
type classification struct {
	ArgsUnwrapped      bool `json:"args_unwrapped"`
	HandlerSawRealArgs bool `json:"handler_saw_real_args"`
}

func main() {
	os.Exit(run())
}

func run() int {
	// A single agent + recorder shared across fixtures. The handler records the
	// args map it was handed on each call.
	var mu sync.Mutex
	var received map[string]any

	a := agent.NewAgentBase(agent.WithBasicAuth(authUser, authPass))
	a.DefineTool(agent.ToolDefinition{
		Name:        functionName,
		Description: "record the args the handler received",
		Handler: func(args map[string]any, _ map[string]any) *swaig.FunctionResult {
			mu.Lock()
			received = args
			mu.Unlock()
			return swaig.NewFunctionResult("ok")
		},
	})

	srv := httptest.NewServer(a.AsRouter())
	defer srv.Close()

	out := map[string]classification{}
	for _, f := range corpus {
		mu.Lock()
		received = nil
		mu.Unlock()

		if err := postSwaig(srv.URL, f.body); err != nil {
			fmt.Fprintf(os.Stderr, "swaig-http-dump: fixture %s: %v\n", f.id, err)
			return 1
		}

		mu.Lock()
		got := received
		mu.Unlock()

		out[f.id] = classify(f.args, got)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "swaig-http-dump: encode: %v\n", err)
		return 1
	}
	return 0
}

// classify derives the fixture artifact from what the handler received.
//
//	args_unwrapped        — the received map's keys are the real arg names, i.e.
//	                        it is NOT the {parsed,raw} envelope (no "parsed"/"raw"
//	                        top-level keys) and it carries at least one expected key.
//	handler_saw_real_args — every expected key->value round-tripped exactly.
func classify(expected, got map[string]any) classification {
	if got == nil {
		return classification{}
	}
	// A leaked envelope has "parsed"/"raw" keys and none of the real arg names.
	_, hasParsed := got["parsed"]
	_, hasRaw := got["raw"]
	envelopeLeaked := hasParsed || hasRaw

	sawReal := true
	sawAnyExpectedKey := false
	for k, want := range expected {
		gv, ok := got[k]
		if ok {
			sawAnyExpectedKey = true
		}
		if !ok || fmt.Sprint(gv) != fmt.Sprint(want) {
			sawReal = false
		}
	}
	unwrapped := !envelopeLeaked && sawAnyExpectedKey
	return classification{ArgsUnwrapped: unwrapped, HandlerSawRealArgs: sawReal}
}

// postSwaig POSTs a SWAIG body to the agent's /swaig route over HTTP with basic
// auth and drains the response.
func postSwaig(baseURL string, body map[string]any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
	req, err := http.NewRequest("POST", baseURL+"/swaig", bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.SetBasicAuth(authUser, authPass)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST /swaig: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("POST /swaig -> %d", resp.StatusCode)
	}
	return nil
}
