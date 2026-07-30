// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package lambda_test

// Inbound __token validation over the SERVERLESS transport.
//
// The token contract holds on EVERY transport, not just plain HTTP: serverless
// is not a weaker envelope, it is the identical contract over a different one.
// pkg/agent/swaig_token_validation_test.go pins the HTTP half; this file pins
// the lambda half, so a regression that gated only the net/http router — or an
// adapter change that dropped the query string on the way in — is caught.
//
// The credential split, verified end to end:
//
//	__token  -> the QUERY STRING (RawQueryString on a Function URL / API
//	            Gateway v2 event, which the adapter folds back into r.URL)
//	call_id  -> the POST BODY (read back via raw_data["call_id"])
//
// The contract, mirroring the pinned goldens in
// porting-sdk/scripts/http_corpus.py:
//
//	valid token   -> handler RUNS,          not refused, 200
//	forged token  -> handler does NOT run,  REFUSED,     200 + FunctionResult
//	absent token  -> handler does NOT run,  REFUSED      (fail-CLOSED)
//	call_id absent-> REFUSED (a token can only be validated against a call_id)
//	insecure tool -> RUNS ungated in all of the above
//
// The refusal is a 200 + FunctionResult BODY, never an HTTP error status: the
// engine (mod_openai) has no handling for a SWAIG refusal status code.

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"github.com/signalwire/signalwire-go/v3/pkg/agent"
	"github.com/signalwire/signalwire-go/v3/pkg/lambda"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
)

const (
	slUser = "sl-u"
	slPass = "sl-p"
	slCall = "sl-call-4417"
	// slHandlerMark is the handler's own output. Its presence in the response
	// body is the ONLY proof the handler ran — a refusal never contains it.
	slHandlerMark = "SERVERLESS-HANDLER-RAN"
)

// slForged is a well-formed but UNSIGNED token: it base64url-decodes and splits
// into the expected parts and is not expired, so only an actual signature
// verification rejects it. Byte-identical to swaig_http_corpus.FORGED_TOKEN.
const slForged = "Y29ycHVzLWNhbGwtN2ZjMi5jaGFyZ2VfYWNjb3VudC40MTAyNDQ0ODAwLmRlYWRiZWVmZGVh" +
	"ZGJlZWYuMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAw" +
	"MDAwMDAwMDAwMDAw"

// slProbe stands up an agent with one secure and one insecure tool behind the
// lambda adapter and drives SWAIG calls through it.
type slProbe struct {
	agent *agent.AgentBase
	h     *lambda.Handler
}

func newSLProbe(t *testing.T) *slProbe {
	t.Helper()
	clearAWSEnv(t)

	a := agent.NewAgentBase(
		agent.WithName("sl-probe"),
		agent.WithRoute("/"),
		agent.WithBasicAuth(slUser, slPass),
	)
	handler := func(map[string]any, map[string]any) *swaig.FunctionResult {
		return swaig.NewFunctionResult(slHandlerMark)
	}
	secure, insecure := true, false
	a.DefineTool(agent.ToolDefinition{
		Name: "secure_tool", Description: "secure probe",
		Secure: &secure, Parameters: map[string]any{}, Handler: handler,
	})
	a.DefineTool(agent.ToolDefinition{
		Name: "insecure_tool", Description: "insecure probe",
		Secure: &insecure, Parameters: map[string]any{}, Handler: handler,
	})
	return &slProbe{agent: a, h: lambda.NewHandler(a.AsRouter())}
}

// call drives one serverless SWAIG invocation and reports (handlerRan, status,
// body). token=="" omits the query string entirely; callID=="" omits call_id
// from the body.
func (p *slProbe) call(t *testing.T, fn, token, callID string) (bool, int, string) {
	t.Helper()

	body := `{"function":"` + fn + `","argument":{"parsed":[{}]}`
	if callID != "" {
		body += `,"call_id":"` + callID + `"`
	}
	body += `}`

	rawQuery := ""
	if token != "" {
		rawQuery = url.Values{"__token": {token}}.Encode()
	}

	resp, err := p.h.HandleFunctionURL(context.Background(), events.LambdaFunctionURLRequest{
		RawPath:        "/swaig",
		RawQueryString: rawQuery,
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "POST"},
		},
		Headers: map[string]string{
			"authorization": basicAuthHeader(slUser, slPass),
			"content-type":  "application/json",
		},
		Body: body,
	})
	if err != nil {
		t.Fatalf("HandleFunctionURL: %v", err)
	}
	return strings.Contains(resp.Body, slHandlerMark), resp.StatusCode, resp.Body
}

// mint produces a genuine token from the SAME agent instance the probe drives.
// The token is an HMAC keyed by that agent's per-process SessionManager secret
// and it expires, so a literal could never be valid.
func (p *slProbe) mint(t *testing.T, fn, callID string) string {
	t.Helper()
	tok := p.agent.CreateToolToken(fn, callID)
	if tok == "" {
		t.Fatalf("CreateToolToken(%q, %q) returned empty — cannot test the valid path", fn, callID)
	}
	return tok
}

func TestServerlessSecureToolAcceptsValidToken(t *testing.T) {
	p := newSLProbe(t)
	ran, status, body := p.call(t, "secure_tool", p.mint(t, "secure_tool", slCall), slCall)

	if !ran {
		t.Errorf("valid token over lambda must RUN the secure handler; body=%q", body)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	if strings.Contains(body, "security token") {
		t.Errorf("valid token must not be refused; body=%q", body)
	}
}

func TestServerlessSecureToolRefusesForgedToken(t *testing.T) {
	p := newSLProbe(t)
	ran, status, body := p.call(t, "secure_tool", slForged, slCall)

	if ran {
		t.Errorf("forged token over lambda must NOT run the secure handler; body=%q", body)
	}
	if status != http.StatusOK {
		t.Errorf("refusal status = %d, want 200 — the refusal is a FunctionResult body, "+
			"not an HTTP error (the engine has no handling for a refusal status)", status)
	}
	if !strings.Contains(body, "security token") {
		t.Errorf("refusal body must name the token as invalid/expired; got %q", body)
	}
}

func TestServerlessSecureToolRefusesAbsentToken(t *testing.T) {
	p := newSLProbe(t)
	// No query string at all — the fail-CLOSED case. Omitting the credential
	// must never be weaker than presenting a wrong one.
	ran, status, body := p.call(t, "secure_tool", "", slCall)

	if ran {
		t.Errorf("absent token over lambda must NOT run the secure handler (fail-closed); body=%q", body)
	}
	if status != http.StatusOK {
		t.Errorf("refusal status = %d, want 200", status)
	}
	if !strings.Contains(body, "security token") {
		t.Errorf("refusal body must name the token as invalid/expired; got %q", body)
	}
}

func TestServerlessSecureToolRefusesWhenCallIDAbsent(t *testing.T) {
	p := newSLProbe(t)
	// A genuine token for slCall, but the body carries NO call_id. A token can
	// only be validated against a call_id, so a missing one is UNVALIDATED —
	// never a bypass.
	ran, status, body := p.call(t, "secure_tool", p.mint(t, "secure_tool", slCall), "")

	if ran {
		t.Errorf("missing call_id over lambda must NOT run the secure handler; body=%q", body)
	}
	if status != http.StatusOK {
		t.Errorf("refusal status = %d, want 200", status)
	}
	if !strings.Contains(body, "security token") {
		t.Errorf("refusal body must name the token as invalid/expired; got %q", body)
	}
}

// TestServerlessInsecureToolRunsUngated pins that the gate is not a blanket
// one. An insecure tool runs in every token state — valid, forged, absent, and
// with no call_id. A fix that refuses everything is not a fix.
func TestServerlessInsecureToolRunsUngated(t *testing.T) {
	p := newSLProbe(t)

	cases := []struct {
		name   string
		token  string
		callID string
	}{
		{"valid_token", p.mint(t, "insecure_tool", slCall), slCall},
		{"forged_token", slForged, slCall},
		{"absent_token", "", slCall},
		{"no_call_id", p.mint(t, "insecure_tool", slCall), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ran, status, body := p.call(t, "insecure_tool", tc.token, tc.callID)
			if !ran {
				t.Errorf("insecure tool must run ungated (%s); body=%q", tc.name, body)
			}
			if status != http.StatusOK {
				t.Errorf("status = %d, want 200", status)
			}
		})
	}
}
