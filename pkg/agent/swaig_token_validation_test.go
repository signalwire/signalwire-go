// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package agent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
)

// Inbound __token validation on POST /swaig.
//
// The port MINTED per-tool `__token` query params into every secure tool's
// web_hook_url but never VALIDATED one coming back in, so a secure tool ran for
// a FORGED token and for NO token at all. That is a `secure` flag that permits
// anonymous calls.
//
// The contract mirrors the reference (signalwire-python
// core/agent_base.py:_swaig_pre_dispatch) and the pinned goldens in
// porting-sdk/scripts/swaig_http_corpus.py:
//
//	token_valid   {handler_invoked: true,  refused: false}
//	token_forged  {handler_invoked: false, refused: true}
//	token_absent  {handler_invoked: false, refused: true}   <- FAIL-CLOSED
//
// The refusal is a 200 + FunctionResult body, NOT an HTTP error status: the
// engine (mod_openai) has no handling for a SWAIG refusal status, so the tool
// reports that it cannot execute and the model relays it.

const (
	tokTestUser = "u"
	tokTestPass = "p"
	tokTestCall = "test-call-9c31"
)

// tokForged is a well-formed but UNSIGNED token: it base64url-decodes and
// splits into the 5 expected parts and is not expired, so only an actual
// signature verification rejects it (a length/format/expiry check would pass
// it). Byte-identical to swaig_http_corpus.FORGED_TOKEN.
const tokForged = "Y29ycHVzLWNhbGwtN2ZjMi5jaGFyZ2VfYWNjb3VudC40MTAyNDQ0ODAwLmRlYWRiZWVmZGVh" +
	"ZGJlZWYuMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAw" +
	"MDAwMDAwMDAwMDAw"

// tokenProbe stands up an agent with one secure and one insecure tool whose
// handlers record whether they ran, mounts the REAL router over httptest, and
// POSTs the platform-nested SWAIG body with the given query params.
type tokenProbe struct {
	agent   *AgentBase
	srv     *httptest.Server
	invoked bool
}

func newTokenProbe(t *testing.T, secureName, insecureName string) *tokenProbe {
	t.Helper()
	p := &tokenProbe{}
	p.agent = NewAgentBase(WithName("token-probe"), WithBasicAuth(tokTestUser, tokTestPass))

	record := func(map[string]any, map[string]any) *swaig.FunctionResult {
		p.invoked = true
		return swaig.NewFunctionResult("ok")
	}
	secure, insecure := true, false
	p.agent.DefineTool(ToolDefinition{
		Name: secureName, Description: "secure probe", Secure: &secure, Handler: record,
	})
	p.agent.DefineTool(ToolDefinition{
		Name: insecureName, Description: "insecure probe", Secure: &insecure, Handler: record,
	})

	p.srv = httptest.NewServer(p.agent.AsRouter())
	t.Cleanup(p.srv.Close)
	return p
}

// post drives one SWAIG call and returns (handlerRan, statusCode, bodyText).
// The body carries call_id where the reference reads it (the BODY, not the
// query string) — a token can only be validated against a call_id.
func (p *tokenProbe) post(t *testing.T, function string, query url.Values, callID string) (bool, int, string) {
	t.Helper()
	p.invoked = false

	args := map[string]any{"order_id": "PROBE-1"}
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	body := map[string]any{
		"function": function,
		"argument": map[string]any{"parsed": []any{args}, "raw": string(raw)},
	}
	if callID != "" {
		body["call_id"] = callID
	}
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	target := p.srv.URL + "/swaig"
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req, err := http.NewRequest("POST", target, bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.SetBasicAuth(tokTestUser, tokTestPass)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /swaig: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var out bytes.Buffer
	if _, err := out.ReadFrom(resp.Body); err != nil {
		t.Fatalf("read response: %v", err)
	}
	return p.invoked, resp.StatusCode, out.String()
}

// isRefusal mirrors the cross-port differ's predicate
// (diff_port_swaig_http._classify_token): a refusal is a response mentioning
// "token" AND "invalid"/"expired". Derived from the BODY, never the status.
func isRefusal(bodyText string) bool {
	t := strings.ToLower(bodyText)
	return strings.Contains(t, "token") &&
		(strings.Contains(t, "invalid") || strings.Contains(t, "expired"))
}

// A secure tool called with a VALID minted token RUNS. A fix that refuses
// everything is not a fix.
func TestSwaigSecureToolAcceptsValidToken(t *testing.T) {
	p := newTokenProbe(t, "charge_account", "lookup_order")

	minted := p.agent.CreateToolToken("charge_account", tokTestCall)
	if minted == "" {
		t.Fatalf("minting a valid token returned empty")
	}
	q := url.Values{"__token": []string{minted}}

	ran, status, body := p.post(t, "charge_account", q, tokTestCall)
	if !ran {
		t.Errorf("secure tool with a VALID token: handler_invoked=false, want true (body=%q)", body)
	}
	if isRefusal(body) {
		t.Errorf("secure tool with a VALID token: refused=true, want false (body=%q)", body)
	}
	if status != http.StatusOK {
		t.Errorf("secure tool with a VALID token: status=%d, want 200", status)
	}
}

// THE SECURITY FIXTURE. A well-formed but unsigned token must be REFUSED and
// the handler must never run.
func TestSwaigSecureToolRefusesForgedToken(t *testing.T) {
	p := newTokenProbe(t, "charge_account", "lookup_order")
	q := url.Values{"__token": []string{tokForged}}

	ran, status, body := p.post(t, "charge_account", q, tokTestCall)
	if ran {
		t.Errorf("secure tool with a FORGED token: handler_invoked=true, want false — the tool RAN for an unsigned token (body=%q)", body)
	}
	if !isRefusal(body) {
		t.Errorf("secure tool with a FORGED token: refused=false, want true — response does not report an invalid/expired token (body=%q)", body)
	}
	// The refusal is a 200 + FunctionResult body, not an error status: the
	// engine has no handling for a SWAIG refusal status code.
	if status != http.StatusOK {
		t.Errorf("secure tool with a FORGED token: status=%d, want 200 (the refusal is a body short-circuit, not an error status)", status)
	}
}

// FAIL-CLOSED. Omitting the credential must never be weaker than presenting a
// wrong one, or `secure` would be a flag that permits anonymous calls.
func TestSwaigSecureToolRefusesAbsentToken(t *testing.T) {
	p := newTokenProbe(t, "charge_account", "lookup_order")

	ran, status, body := p.post(t, "charge_account", nil, tokTestCall)
	if ran {
		t.Errorf("secure tool with NO token: handler_invoked=true, want false — an absent credential ran a secure tool anonymously (body=%q)", body)
	}
	if !isRefusal(body) {
		t.Errorf("secure tool with NO token: refused=false, want true — response does not report an invalid/expired token (body=%q)", body)
	}
	if status != http.StatusOK {
		t.Errorf("secure tool with NO token: status=%d, want 200 (the refusal is a body short-circuit, not an error status)", status)
	}
}

// A token can only be validated against a call_id; without one there is nothing
// to check it against, so the reference treats it as UNVALIDATED and a secure
// tool is refused. Guards against "no call_id" becoming a validation bypass.
func TestSwaigSecureToolRefusesWhenCallIDAbsent(t *testing.T) {
	p := newTokenProbe(t, "charge_account", "lookup_order")

	minted := p.agent.CreateToolToken("charge_account", tokTestCall)
	if minted == "" {
		t.Fatalf("minting a valid token returned empty")
	}
	q := url.Values{"__token": []string{minted}}

	// Same token, but the body carries NO call_id.
	ran, _, body := p.post(t, "charge_account", q, "")
	if ran {
		t.Errorf("secure tool with a token but NO call_id: handler_invoked=true, want false — nothing to validate the token against (body=%q)", body)
	}
	if !isRefusal(body) {
		t.Errorf("secure tool with a token but NO call_id: refused=false, want true (body=%q)", body)
	}
}

// An INSECURE tool is not gated. It runs with no token at all — the token check
// must not become a blanket gate on every tool.
func TestSwaigInsecureToolRunsWithoutToken(t *testing.T) {
	p := newTokenProbe(t, "charge_account", "lookup_order")

	ran, status, body := p.post(t, "lookup_order", nil, tokTestCall)
	if !ran {
		t.Errorf("INSECURE tool with no token: handler_invoked=false, want true — an insecure tool is not gated (body=%q)", body)
	}
	if isRefusal(body) {
		t.Errorf("INSECURE tool with no token: refused=true, want false (body=%q)", body)
	}
	if status != http.StatusOK {
		t.Errorf("INSECURE tool with no token: status=%d, want 200", status)
	}
}

// The reference reads `__token` first and falls back to a plain `token` query
// param. Both spellings must reach the validator.
func TestSwaigSecureToolAcceptsPlainTokenParamFallback(t *testing.T) {
	p := newTokenProbe(t, "charge_account", "lookup_order")

	minted := p.agent.CreateToolToken("charge_account", tokTestCall)
	if minted == "" {
		t.Fatalf("minting a valid token returned empty")
	}
	q := url.Values{"token": []string{minted}}

	ran, _, body := p.post(t, "charge_account", q, tokTestCall)
	if !ran {
		t.Errorf("secure tool with a valid token in the `token` fallback param: handler_invoked=false, want true (body=%q)", body)
	}
	if isRefusal(body) {
		t.Errorf("secure tool with a valid token in the `token` fallback param: refused=true, want false (body=%q)", body)
	}
}
