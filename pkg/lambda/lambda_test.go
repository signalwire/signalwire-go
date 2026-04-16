package lambda

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// echoHandler answers any request with a JSON body describing exactly what
// it received. Tests assert on this echoed shape to verify the adapter is
// translating Lambda events into http.Requests faithfully.
func echoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		out := map[string]any{
			"method":      r.Method,
			"path":        r.URL.Path,
			"query":       r.URL.RawQuery,
			"body":        string(bodyBytes),
			"auth_header": r.Header.Get("Authorization"),
			"xyz":         r.Header.Get("X-Custom"),
			"cookie":      r.Header.Get("Cookie"),
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("Set-Cookie", "a=1")
		w.Header().Add("Set-Cookie", "b=2")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(out)
	})
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNewHandler_PanicsOnNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when http.Handler is nil")
		}
	}()
	_ = NewHandler(nil)
}

func TestNewHandler_ReturnsNonNil(t *testing.T) {
	if h := NewHandler(echoHandler()); h == nil {
		t.Error("NewHandler returned nil")
	}
}

// ---------------------------------------------------------------------------
// Function URL dispatch
// ---------------------------------------------------------------------------

func TestHandleFunctionURL_PassesMethodPathAndQuery(t *testing.T) {
	h := NewHandler(echoHandler())
	req := events.LambdaFunctionURLRequest{
		RawPath:        "/my-agent/swaig",
		RawQueryString: "agent_id=123&session=abc",
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "POST"},
		},
		Headers: map[string]string{"content-type": "application/json"},
		Body:    `{"function":"greet"}`,
	}

	resp, err := h.HandleFunctionURL(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleFunctionURL: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var echoed map[string]any
	if err := json.Unmarshal([]byte(resp.Body), &echoed); err != nil {
		t.Fatalf("response body not JSON: %v", err)
	}
	if echoed["method"] != "POST" {
		t.Errorf("method = %v, want POST", echoed["method"])
	}
	if echoed["path"] != "/my-agent/swaig" {
		t.Errorf("path = %v, want /my-agent/swaig", echoed["path"])
	}
	if echoed["query"] != "agent_id=123&session=abc" {
		t.Errorf("query = %v, want agent_id=123&session=abc", echoed["query"])
	}
	if echoed["body"] != `{"function":"greet"}` {
		t.Errorf("body = %v, want JSON payload", echoed["body"])
	}
}

func TestHandleFunctionURL_CopiesHeaders(t *testing.T) {
	h := NewHandler(echoHandler())
	req := events.LambdaFunctionURLRequest{
		RawPath: "/",
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "GET"},
		},
		Headers: map[string]string{
			"authorization": "Basic dXNlcjpwYXNz",
			"x-custom":      "hello",
		},
	}
	resp, err := h.HandleFunctionURL(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleFunctionURL: %v", err)
	}
	var echoed map[string]any
	_ = json.Unmarshal([]byte(resp.Body), &echoed)
	if echoed["auth_header"] != "Basic dXNlcjpwYXNz" {
		t.Errorf("auth_header = %v", echoed["auth_header"])
	}
	if echoed["xyz"] != "hello" {
		t.Errorf("X-Custom = %v", echoed["xyz"])
	}
}

func TestHandleFunctionURL_FoldsCookiesIntoHeader(t *testing.T) {
	h := NewHandler(echoHandler())
	req := events.LambdaFunctionURLRequest{
		RawPath: "/",
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "GET"},
		},
		Cookies: []string{"session=xyz", "theme=dark"},
	}
	resp, err := h.HandleFunctionURL(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleFunctionURL: %v", err)
	}
	var echoed map[string]any
	_ = json.Unmarshal([]byte(resp.Body), &echoed)
	got, _ := echoed["cookie"].(string)
	if !strings.Contains(got, "session=xyz") || !strings.Contains(got, "theme=dark") {
		t.Errorf("folded cookie header = %q, want both cookies", got)
	}
}

func TestHandleFunctionURL_DecodesBase64Body(t *testing.T) {
	h := NewHandler(echoHandler())
	payload := `{"hello":"world"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(payload))
	req := events.LambdaFunctionURLRequest{
		RawPath: "/",
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "POST"},
		},
		Body:            encoded,
		IsBase64Encoded: true,
	}
	resp, err := h.HandleFunctionURL(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleFunctionURL: %v", err)
	}
	var echoed map[string]any
	_ = json.Unmarshal([]byte(resp.Body), &echoed)
	if echoed["body"] != payload {
		t.Errorf("decoded body = %v, want %q", echoed["body"], payload)
	}
}

func TestHandleFunctionURL_InvalidBase64ReturnsError(t *testing.T) {
	h := NewHandler(echoHandler())
	req := events.LambdaFunctionURLRequest{
		RawPath: "/",
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "POST"},
		},
		Body:            "not valid base64!!!",
		IsBase64Encoded: true,
	}
	_, err := h.HandleFunctionURL(context.Background(), req)
	if err == nil {
		t.Error("expected base64 decode error, got nil")
	}
}

func TestHandleFunctionURL_DefaultsMethodAndPath(t *testing.T) {
	// Handler should not crash when optional fields are zero.
	h := NewHandler(echoHandler())
	req := events.LambdaFunctionURLRequest{
		// Method and RawPath intentionally left empty
	}
	resp, err := h.HandleFunctionURL(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleFunctionURL: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 (defaults applied)", resp.StatusCode)
	}
	var echoed map[string]any
	_ = json.Unmarshal([]byte(resp.Body), &echoed)
	if echoed["method"] != "GET" {
		t.Errorf("default method = %v, want GET", echoed["method"])
	}
	if echoed["path"] != "/" {
		t.Errorf("default path = %v, want /", echoed["path"])
	}
}

// ---------------------------------------------------------------------------
// Response-side conversion
// ---------------------------------------------------------------------------

func TestHandleFunctionURL_SetCookieMovedToCookiesField(t *testing.T) {
	h := NewHandler(echoHandler())
	req := events.LambdaFunctionURLRequest{
		RawPath: "/",
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "GET"},
		},
	}
	resp, err := h.HandleFunctionURL(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleFunctionURL: %v", err)
	}
	// echoHandler adds TWO Set-Cookie headers. They must end up in the
	// Cookies slice, NOT comma-joined into a single Set-Cookie header.
	if len(resp.Cookies) != 2 {
		t.Errorf("Cookies length = %d, want 2; Cookies=%v", len(resp.Cookies), resp.Cookies)
	}
	if _, ok := resp.Headers["Set-Cookie"]; ok {
		t.Errorf("Set-Cookie should have moved to Cookies field, headers=%v", resp.Headers)
	}
}

func TestHandleFunctionURL_BinaryResponseIsBase64Encoded(t *testing.T) {
	// A handler returning invalid UTF-8 must come out base64-encoded.
	binary := []byte{0xff, 0xfe, 0xfd}
	h := NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(binary)
	}))
	req := events.LambdaFunctionURLRequest{
		RawPath: "/",
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "GET"},
		},
	}
	resp, err := h.HandleFunctionURL(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleFunctionURL: %v", err)
	}
	if !resp.IsBase64Encoded {
		t.Fatal("IsBase64Encoded = false, want true for binary body")
	}
	decoded, err := base64.StdEncoding.DecodeString(resp.Body)
	if err != nil {
		t.Fatalf("response body is not valid base64: %v", err)
	}
	if string(decoded) != string(binary) {
		t.Errorf("decoded body = %x, want %x", decoded, binary)
	}
}

// ---------------------------------------------------------------------------
// API Gateway V2 dispatch
// ---------------------------------------------------------------------------

func TestHandleAPIGatewayV2_BasicRoundTrip(t *testing.T) {
	h := NewHandler(echoHandler())
	req := events.APIGatewayV2HTTPRequest{
		RawPath: "/my-agent/swaig",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: "POST"},
		},
		Body: `{"function":"noop"}`,
	}
	resp, err := h.HandleAPIGatewayV2(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleAPIGatewayV2: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var echoed map[string]any
	_ = json.Unmarshal([]byte(resp.Body), &echoed)
	if echoed["path"] != "/my-agent/swaig" {
		t.Errorf("path = %v, want /my-agent/swaig", echoed["path"])
	}
}
