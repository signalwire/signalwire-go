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
// It stands up an AgentBase with a `lookup_order` tool (secure=false) and a
// `charge_account` tool (secure=true) whose handlers RECORD what they received,
// mounts the agent's real /swaig route over a live httptest HTTP server, POSTs
// each corpus fixture body to it over a genuine HTTP round trip (the dispatch
// path the platform actually drives — NOT an in-process handler call, which is
// exactly the blind spot that hid the GO-7 bug), and prints ONE JSON object
// mapping each fixture id to its kind-specific classification:
//
//	unwrap fixtures -> {"args_unwrapped": bool, "handler_saw_real_args": bool}
//	token fixtures  -> {"handler_invoked": bool, "refused": bool}
//
// For the unwrap fixtures, args_unwrapped is true when the handler was handed a
// FLAT args dict whose keys are the real argument names (order_id/customer, NOT
// parsed/raw), and handler_saw_real_args is true when every expected key->value
// the fixture armed round-tripped to the handler. A port that passes the nested
// {parsed,raw} envelope through reds BOTH (the pre-GO-7 go behavior).
//
// For the token fixtures, the secure tool is invoked with a VALID / FORGED /
// ABSENT `__token` query param and we record, INDEPENDENTLY, whether the tool
// body actually ran (handler_invoked) and whether the endpoint short-circuited
// with a token-refusal response (refused). Recording them independently is what
// keeps a port that neither runs nor refuses (a 500, an unknown-function error)
// distinguishable from a genuine refusal. The refusal is a RESPONSE-BODY
// short-circuit at HTTP 200, not a status code, so `refused` is derived from the
// response text — never from the status.
//
// The corpus is embedded here byte-identically to swaig_http_corpus.py; the
// differ keys our output by fixture id against the python oracle.
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
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/signalwire/signalwire-go/v3/pkg/agent"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
)

// The stable tool names every port registers (swaig_http_corpus.FUNCTION /
// .SECURE_FUNCTION). SECURE_FUNCTION is registered with secure=true and is used
// ONLY by the token fixtures, so the unwrap fixtures keep exercising a
// non-secure tool exactly as they always have (no token path in their way).
const (
	functionName       = "lookup_order"
	secureFunctionName = "charge_account"
)

// tokenParam is the query-param name the platform puts the per-tool token in
// (swaig_http_corpus.TOKEN_PARAM). This IS the wire contract.
const tokenParam = "__token"

// tokenCallID is the session id token fixtures mint/validate against
// (swaig_http_corpus.TOKEN_CALL_ID). It rides in the BODY, because that is where
// the reference reads it (core/swml_service.py :877) — NOT the query string.
const tokenCallID = "corpus-call-7fc2"

// forgedToken mirrors swaig_http_corpus.FORGED_TOKEN verbatim: base64url of
// "<call_id>.<function>.<far-future-expiry>.<nonce>.<garbage-signature>". It
// DECODES and SPLITS into the 5 expected parts and is not expired, so a port
// cannot pass the forged fixture via a length/format/expiry check alone; only an
// actual signature verification rejects it.
const forgedToken = "Y29ycHVzLWNhbGwtN2ZjMi5jaGFyZ2VfYWNjb3VudC40MTAyNDQ0ODAwLmRlYWRiZWVmZGVh" +
	"ZGJlZWYuMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAw" +
	"MDAwMDAwMDAwMDAw"

// basic-auth creds we set explicitly so we can authenticate the POST.
const (
	authUser = "u"
	authPass = "p"
)

// Fixture kinds — the driver discriminator (swaig_http_corpus per-case `kind`).
const (
	kindPlatformNested = "platform_nested"
	kindFlatArguments  = "flat_arguments"
	kindToken          = "token"
)

// Token directives (swaig_http_corpus per-case `token`). A DIRECTIVE, never a
// literal: a valid token is an HMAC keyed by this agent's per-process session
// secret and it expires, so each driver MINTS ITS OWN.
const (
	tokenValid  = "valid"
	tokenForged = "forged"
	tokenAbsent = "absent"
)

// fixture mirrors one swaig_http_corpus.CORPUS entry.
type fixture struct {
	id       string
	kind     string
	function string
	// token is the directive, set only on kind == kindToken.
	token string
	args  map[string]any // the real args the handler MUST end up receiving
	body  map[string]any // the exact SWAIG POST body (kind-specific envelope)
}

// The real args each fixture arms (swaig_http_corpus._NESTED_ARGS etc). Distinct
// values per fixture so a handler that leaks another fixture's args (or an empty
// dict) is caught.
var (
	nestedArgs      = map[string]any{"order_id": "ORD-3007", "customer": "acme-42"}
	flatArgs        = map[string]any{"order_id": "FLAT-9911"}
	tokenValidArgs  = map[string]any{"order_id": "TOKVALID-5501"}
	tokenForgedArgs = map[string]any{"order_id": "TOKFORGED-5502"}
	tokenAbsentArgs = map[string]any{"order_id": "TOKABSENT-5503"}
)

// nestedBody / flatArgumentsBody / tokenBody mirror the corpus body builders.
func nestedBody(function string, args map[string]any) map[string]any {
	raw, err := json.Marshal(args)
	if err != nil {
		panic(err) // fixture args are static, plain JSON — marshal never fails
	}
	return map[string]any{
		"function": function,
		"argument": map[string]any{"parsed": []any{args}, "raw": string(raw)},
	}
}

func flatArgumentsBody(function string, args map[string]any) map[string]any {
	return map[string]any{"function": function, "arguments": args}
}

// tokenBody is the PLATFORM-nested shape (the real platform always sends nested)
// PLUS the call_id the reference reads from the BODY and binds validation to.
// Putting call_id only in the query would silently invalidate every token.
func tokenBody(function string, args map[string]any, callID string) map[string]any {
	body := nestedBody(function, args)
	body["call_id"] = callID
	return body
}

var corpus = []fixture{
	// platform_nested — the shape the real platform sends. A correct handler
	// unwraps argument.parsed[0] -> {order_id, customer}.
	{
		id: "platform_nested", kind: kindPlatformNested, function: functionName,
		args: nestedArgs, body: nestedBody(functionName, nestedArgs),
	},
	// flat_arguments — the {"arguments":{...}} fallback python + the platform accept.
	{
		id: "flat_arguments", kind: kindFlatArguments, function: functionName,
		args: flatArgs, body: flatArgumentsBody(functionName, flatArgs),
	},
	// token_valid — the secure tool invoked with a token WE minted for
	// (secureFunctionName, tokenCallID). The endpoint must accept it and RUN the
	// tool. A port that refuses every token (or cannot mint) reds HERE.
	{
		id: "token_valid", kind: kindToken, function: secureFunctionName,
		token: tokenValid, args: tokenValidArgs,
		body: tokenBody(secureFunctionName, tokenValidArgs, tokenCallID),
	},
	// token_forged — THE SECURITY FIXTURE. Same secure tool, same call_id, but a
	// well-formed yet UNSIGNED token. The reference REFUSES and never calls the
	// handler. A port that MINTS __token but never VALIDATES it inbound runs the
	// handler anyway and reds exactly here.
	{
		id: "token_forged", kind: kindToken, function: secureFunctionName,
		token: tokenForged, args: tokenForgedArgs,
		body: tokenBody(secureFunctionName, tokenForgedArgs, tokenCallID),
	},
	// token_absent — the same secure tool with NO __token query param at all.
	// The golden records what the ORACLE DOES; a port that diverges in EITHER
	// direction surfaces as a red for a human to adjudicate.
	{
		id: "token_absent", kind: kindToken, function: secureFunctionName,
		token: tokenAbsent, args: tokenAbsentArgs,
		body: tokenBody(secureFunctionName, tokenAbsentArgs, tokenCallID),
	},
}

// unwrapClassification is the artifact for the platform_nested / flat_arguments
// fixtures.
type unwrapClassification struct {
	ArgsUnwrapped      bool `json:"args_unwrapped"`
	HandlerSawRealArgs bool `json:"handler_saw_real_args"`
}

// tokenClassification is the artifact for the token fixtures. The two booleans
// are recorded INDEPENDENTLY (see the file comment).
type tokenClassification struct {
	HandlerInvoked bool `json:"handler_invoked"`
	Refused        bool `json:"refused"`
}

func main() {
	os.Exit(run())
}

func run() int {
	// A single agent + recorder shared across fixtures. The handlers record what
	// they were handed / whether they ran at all.
	var mu sync.Mutex
	var received map[string]any
	var invoked bool

	record := func(args map[string]any) *swaig.FunctionResult {
		mu.Lock()
		received = args
		invoked = true
		mu.Unlock()
		return swaig.NewFunctionResult("ok")
	}

	a := agent.NewAgentBase(agent.WithBasicAuth(authUser, authPass))
	// ToolDefinition.Secure is a tri-state *bool whose nil means SECURE (the
	// A1 secure-by-default contract), so BOTH flags are set explicitly here
	// rather than relying on the zero value.
	insecure, secure := false, true
	a.DefineTool(agent.ToolDefinition{
		Name:        functionName,
		Description: "record the args the handler received",
		Secure:      &insecure,
		Handler: func(args map[string]any, _ map[string]any) *swaig.FunctionResult {
			return record(args)
		},
	})
	// The SECOND tool the token fixtures target. secure=true is what arms the
	// inbound __token check at all — a secure=false tool would never refuse ANY
	// token, so the token fixtures would be vacuous against it.
	a.DefineTool(agent.ToolDefinition{
		Name:        secureFunctionName,
		Description: "secure probe tool that records whether it ran",
		Secure:      &secure,
		Handler: func(args map[string]any, _ map[string]any) *swaig.FunctionResult {
			return record(args)
		},
	})

	srv := httptest.NewServer(a.AsRouter())
	defer srv.Close()

	out := map[string]any{}
	for _, f := range corpus {
		mu.Lock()
		received = nil
		invoked = false
		mu.Unlock()

		query, err := tokenQuery(a, f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "swaig-http-dump: fixture %s: %v\n", f.id, err)
			return 1
		}

		respBody, err := postSwaig(srv.URL, f.body, query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "swaig-http-dump: fixture %s: %v\n", f.id, err)
			return 1
		}

		mu.Lock()
		got := received
		ran := invoked
		mu.Unlock()

		if f.kind == kindToken {
			out[f.id] = classifyToken(ran, respBody)
			continue
		}
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

// tokenQuery resolves a fixture's token DIRECTIVE into the query params to send.
// Non-token fixtures and the ABSENT directive get an EMPTY url.Values — never a
// bare nil alongside a nil error, which would be an ambiguous "no result / no
// error" signal (nilnil). url.Values is a map, so an empty one IS the valid
// representation of "send no query params": postSwaig's len(query) > 0 guard
// treats it byte-identically to nil, so the two original fixtures are untouched.
//
// "valid" must be MINTED here, not read from a constant: the token is an
// HMAC-SHA256 keyed by this agent's SessionManager secret (a per-process random
// value) and it expires, so no literal could ever be valid.
func tokenQuery(a *agent.AgentBase, f fixture) (url.Values, error) {
	if f.kind != kindToken {
		return url.Values{}, nil
	}
	switch f.token {
	case tokenAbsent:
		return url.Values{}, nil
	case tokenForged:
		return url.Values{tokenParam: []string{forgedToken}}, nil
	case tokenValid:
		minted := a.CreateToolToken(f.function, tokenCallID)
		if minted == "" {
			return nil, fmt.Errorf("minting a valid token for %q returned empty", f.function)
		}
		return url.Values{tokenParam: []string{minted}}, nil
	default:
		return nil, fmt.Errorf("unknown token directive %q", f.token)
	}
}

// classify derives the unwrap-fixture artifact from what the handler received.
//
//	args_unwrapped        — the received map's keys are the real arg names, i.e.
//	                        it is NOT the {parsed,raw} envelope (no "parsed"/"raw"
//	                        top-level keys) and it carries at least one expected key.
//	handler_saw_real_args — every expected key->value round-tripped exactly.
func classify(expected, got map[string]any) unwrapClassification {
	if got == nil {
		return unwrapClassification{}
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
	return unwrapClassification{ArgsUnwrapped: unwrapped, HandlerSawRealArgs: sawReal}
}

// classifyToken derives the token-fixture artifact.
//
//	handler_invoked — the tool body ACTUALLY RAN (our recording handler fired).
//	refused         — the endpoint short-circuited with a token-refusal response
//	                  instead of running the tool. Mirrors the differ's own
//	                  predicate (diff_port_swaig_http._classify_token): not
//	                  invoked AND the response mentions "token" AND
//	                  "invalid"/"expired". Derived from the RESPONSE BODY, never
//	                  the status — the refusal is a 200 + FunctionResult body.
func classifyToken(invoked bool, respBody []byte) tokenClassification {
	text := strings.ToLower(string(respBody))
	refused := !invoked &&
		strings.Contains(text, "token") &&
		(strings.Contains(text, "invalid") || strings.Contains(text, "expired"))
	return tokenClassification{HandlerInvoked: invoked, Refused: refused}
}

// postSwaig POSTs a SWAIG body to the agent's /swaig route over HTTP with basic
// auth (plus any token query params) and returns the response body.
//
// A non-200 is NOT an error here: the token contract's refusal is a 200 body,
// but a port that (wrongly) refuses with a status code must still be
// classifiable as {invoked:false, refused:false-or-true} rather than aborting
// the whole dump. The unwrap fixtures still surface a non-200 correctly, since
// a handler that never ran leaves `received` nil.
func postSwaig(baseURL string, body map[string]any, query url.Values) ([]byte, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}
	target := baseURL + "/swaig"
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req, err := http.NewRequest("POST", target, bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.SetBasicAuth(authUser, authPass)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST /swaig: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read /swaig response: %w", err)
	}
	return respBody, nil
}
