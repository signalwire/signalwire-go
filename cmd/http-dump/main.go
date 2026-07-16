// Command http-dump is the Go port's HTTP dump program for the cross-port HTTP
// differ (porting-sdk/scripts/diff_port_http.py).
//
// For each http_corpus case it feeds a synthetic request into the Go SDK's
// framework-free dispatch core (SWMLService.HandleRequest, ExtractSIPUsername,
// the webhook Validate middleware, and the lambda serverless adapter) and prints
// ONE JSON object mapping
//
//	case-id -> reduced-artifact
//
// to stdout, reduced to the same shape the python oracle emits. The differ
// canonicalizes both sides and byte-compares. Only stdout carries JSON.
//
// The corpus sentinels (__AUTH__/__AUTH_BAD__ Basic headers, __SIG__ webhook
// signature, __REDIRECT_CB__ routing callback, __HELLO_HANDLER__ SWAIG handler,
// __JSON__: lambda body prefix) are materialized here as the oracle materializes
// them, so the interop cases are reproducible.
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/http-dump
package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // webhook Scheme A is defined as HMAC-SHA1; matches the wire spec.
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/aws/aws-lambda-go/events"

	"github.com/signalwire/signalwire-go/v3/pkg/agent"
	"github.com/signalwire/signalwire-go/v3/pkg/lambda"
	"github.com/signalwire/signalwire-go/v3/pkg/security"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
	"github.com/signalwire/signalwire-go/v3/pkg/swml"
)

const (
	user       = "user"
	password   = "pass"
	signingKey = "PSK-fixed-signing-key"
	whURL      = "https://agent.example.com/webhook"
	whBody     = `{"event":"call.created","id":"abc"}`
)

func basicAuth(u, p string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(u+":"+p))
}

func webhookSig(url, body, key string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(url + body))
	return hex.EncodeToString(mac.Sum(nil))
}

// observeResponse reduces a (status, headers, body) triple to a comparable
// artifact — the Go mirror of diff_port_http._observe_response.
func observeResponse(status int, headers map[string]string, bodyStr, kind string) map[string]any {
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := map[string]any{"status": status, "header_keys": keys}
	if loc, ok := headers["Location"]; ok {
		out["location"] = loc
	}
	if wa, ok := headers["WWW-Authenticate"]; ok {
		out["www_authenticate"] = wa
	}
	if kind == "response_full" {
		if bodyStr == "" {
			out["body"] = ""
		} else {
			var parsed any
			if err := json.Unmarshal([]byte(bodyStr), &parsed); err == nil {
				out["body"] = parsed
			} else {
				out["body"] = bodyStr
			}
		}
	}
	return out
}

func newSWMLService() *swml.Service {
	return swml.NewService(
		swml.WithName("demo"),
		swml.WithRoute("/swml"),
		swml.WithBasicAuth(user, password),
	)
}

func main() {
	out := map[string]any{}

	// ---- handle_request: 200 SWML happy path ----
	{
		svc := newSWMLService()
		status, headers, body := svc.HandleRequest("GET", "http://localhost:3000/swml",
			map[string]string{"Authorization": basicAuth(user, password)}, nil)
		out["http_handle_request_200_swml"] = observeResponse(status, headers, body, "response_full")
	}
	// ---- handle_request: 401 no auth ----
	{
		svc := newSWMLService()
		status, headers, body := svc.HandleRequest("GET", "http://localhost:3000/swml",
			map[string]string{}, nil)
		out["http_handle_request_401_no_auth"] = observeResponse(status, headers, body, "response_full")
	}
	// ---- handle_request: 401 bad password (status+headers only) ----
	{
		svc := newSWMLService()
		status, headers, body := svc.HandleRequest("GET", "http://localhost:3000/swml",
			map[string]string{"Authorization": basicAuth(user, "wrong")}, nil)
		out["http_handle_request_401_bad_password"] = observeResponse(status, headers, body, "response_status_headers")
	}
	// ---- handle_request: 307 redirect via routing callback ----
	{
		svc := newSWMLService()
		svc.RegisterRoutingCallback("/sip", redirectCB)
		status, headers, body := svc.HandleRequest("POST", "http://localhost:3000/swml/sip",
			map[string]string{"Authorization": basicAuth(user, password)},
			map[string]any{"call": map[string]any{"to": "sip:redirect-me@space"}})
		out["http_handle_request_307_redirect"] = observeResponse(status, headers, body, "response_full")
	}
	// ---- handle_request: callback returns nil -> normal 200 SWML ----
	{
		svc := newSWMLService()
		svc.RegisterRoutingCallback("/sip", redirectCB)
		status, headers, body := svc.HandleRequest("POST", "http://localhost:3000/swml/sip",
			map[string]string{"Authorization": basicAuth(user, password)},
			map[string]any{"call": map[string]any{"to": "sip:keep@space"}})
		out["http_handle_request_callback_passthrough_200"] = observeResponse(status, headers, body, "response_full")
	}

	// ---- extract_sip_username: pure extractor ----
	out["http_extract_sip_username_sip"] = extractUsername(map[string]any{"call": map[string]any{"to": "sip:alice@agents.signalwire.com"}})
	out["http_extract_sip_username_tel"] = extractUsername(map[string]any{"call": map[string]any{"to": "tel:+15551234567"}})
	out["http_extract_sip_username_plain"] = extractUsername(map[string]any{"call": map[string]any{"to": "support"}})
	out["http_extract_sip_username_missing"] = extractUsername(map[string]any{"vars": map[string]any{}})

	// ---- webhook validate ----
	out["http_webhook_validate_ok"] = webhookDecision("POST", whURL, whBody,
		map[string]string{"x-signalwire-signature": webhookSig(whURL, whBody, signingKey)}, signingKey)
	badSig := ""
	for range 5 {
		badSig += "deadbeef"
	}
	out["http_webhook_validate_bad_sig"] = webhookDecision("POST", whURL, whBody,
		map[string]string{"x-signalwire-signature": badSig}, signingKey)
	out["http_webhook_validate_missing_sig"] = webhookDecision("POST", whURL, whBody,
		map[string]string{}, signingKey)
	out["http_webhook_validate_twilio_alias"] = webhookDecision("POST", whURL, whBody,
		map[string]string{"x-twilio-signature": webhookSig(whURL, whBody, signingKey)}, signingKey)

	// ---- serverless (lambda) ----
	out["http_serverless_lambda_swaig"] = serverlessSwaig()
	out["http_serverless_lambda_noauth_401"] = serverlessNoAuth()

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "http-dump: encode failed: %v\n", err)
		os.Exit(1)
	}
}

// redirectCB redirects one specific 'to', else passes through (nil).
func redirectCB(body map[string]any, _ map[string]any) *string {
	call, _ := body["call"].(map[string]any)
	to, _ := call["to"].(string)
	if to == "sip:redirect-me@space" {
		r := "/other-route"
		return &r
	}
	return nil
}

func extractUsername(body map[string]any) map[string]any {
	u := swml.ExtractSIPUsername(body)
	if u == "" {
		return map[string]any{"username": nil}
	}
	return map[string]any{"username": u}
}

func webhookDecision(method, url, body string, headers map[string]string, key string) map[string]any {
	rej := security.Validate(method, url, headers, body, key)
	if rej == nil {
		return map[string]any{"decision": "pass"}
	}
	return map[string]any{"decision": "reject", "status": rej.Status}
}

// serverlessSwaig drives the lambda adapter for the /swaig dispatch case. The
// agent is built at route "/" so the event's root-relative "/swaig" path routes
// correctly — matching Python's serverless dispatch, which strips the route and
// treats rawPath as agent-relative.
func serverlessSwaig() map[string]any {
	a := agent.NewAgentBase(
		agent.WithName("demo"),
		agent.WithRoute("/"),
		agent.WithBasicAuth(user, password),
	)
	a.DefineTool(agent.ToolDefinition{
		Name:        "say_hello",
		Description: "greet",
		Parameters:  map[string]any{},
		Handler: func(_ map[string]any, _ map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("hello there")
		},
	})
	h := lambda.NewHandler(a.AsRouter())
	req := events.LambdaFunctionURLRequest{
		RawPath: "/swaig",
		Headers: map[string]string{
			"authorization": basicAuth(user, password),
			"content-type":  "application/json",
		},
		Body: `{"function":"say_hello","argument":{"parsed":[{}]},"call_id":"c1"}`,
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "POST"},
		},
	}
	resp, err := h.HandleFunctionURL(context.Background(), req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http-dump: serverless swaig failed: %v\n", err)
		os.Exit(1)
	}
	return reduceLambda(resp)
}

func serverlessNoAuth() map[string]any {
	a := agent.NewAgentBase(
		agent.WithName("demo"),
		agent.WithRoute("/"),
		agent.WithBasicAuth(user, password),
	)
	h := lambda.NewHandler(a.AsRouter())
	req := events.LambdaFunctionURLRequest{
		RawPath: "/",
		Headers: map[string]string{},
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "GET"},
		},
	}
	resp, err := h.HandleFunctionURL(context.Background(), req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http-dump: serverless noauth failed: %v\n", err)
		os.Exit(1)
	}
	return reduceLambda(resp)
}

// reduceLambda reduces a lambda response to {status, body} with the body parsed
// as JSON — mirroring the oracle's serverless_result observer.
func reduceLambda(resp events.LambdaFunctionURLResponse) map[string]any {
	var body any = resp.Body
	if resp.Body != "" {
		var parsed any
		if err := json.Unmarshal([]byte(resp.Body), &parsed); err == nil {
			body = parsed
		}
	}
	return map[string]any{"status": resp.StatusCode, "body": body}
}
