package builtin

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/skills"
)

// TestNativeVectorSearch_RemoteHTTP is Tier-2 behavioral contract #4:
// native_vector_search in remote mode must make a REAL HTTP POST to
// <remote_url>/search with the query in the JSON body, and format the mock's
// returned [{content,score,metadata}] into the FunctionResult — NOT return a
// hardcoded "[Would query…]" / "In production this would…" stub string.
//
// The mock HTTP server is an httptest.Server, which binds a FREE loopback port
// and is torn down via defer srv.Close(). Because it binds 127.0.0.1, the
// skill's SSRF guard (validateRemoteURL) would reject it, so we set the
// documented SWML_ALLOW_PRIVATE_URLS escape hatch for the duration of the test
// (this mirrors Python's env-var handling for test environments).
func TestNativeVectorSearch_RemoteHTTP(t *testing.T) {
	t.Setenv("SWML_ALLOW_PRIVATE_URLS", "1")

	var searchHits int32
	var gotBody atomic.Value // string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "ok")
		case "/search":
			atomic.AddInt32(&searchHits, 1)
			if r.Method != http.MethodPost {
				t.Errorf("search request method = %s, want POST", r.Method)
			}
			raw, _ := io.ReadAll(r.Body)
			gotBody.Store(string(raw))
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]any{
					{
						"content":  "The mitochondria is the powerhouse of the cell.",
						"score":    0.91,
						"metadata": map[string]any{"filename": "biology.md", "section": "Cells"},
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	factory := skills.GetSkillFactory("native_vector_search")
	s := factory(map[string]any{
		"remote_url": srv.URL,
		"tool_name":  "search_knowledge",
		"index_name": "biology",
	})
	if !s.Setup() {
		t.Fatal("Setup failed — remote health check did not pass against the mock server")
	}

	tools := s.RegisterTools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	handler := tools[0].Handler
	if handler == nil {
		t.Fatal("search tool handler is nil")
	}

	res := handler(map[string]any{"query": "what is a mitochondria"}, nil)
	if res == nil {
		t.Fatal("search handler returned nil FunctionResult")
	}

	// (1) A real POST to /search must have happened.
	if n := atomic.LoadInt32(&searchHits); n != 1 {
		t.Fatalf("mock /search POST count = %d, want 1 (no real HTTP call — stub?)", n)
	}

	// (2) The query must be in the POST body.
	body, _ := gotBody.Load().(string)
	var sent map[string]any
	if err := json.Unmarshal([]byte(body), &sent); err != nil {
		t.Fatalf("search request body is not JSON: %v; body=%q", err, body)
	}
	if sent["query"] != "what is a mitochondria" {
		t.Errorf("POST body query = %v, want %q", sent["query"], "what is a mitochondria")
	}
	if sent["index_name"] != "biology" {
		t.Errorf("POST body index_name = %v, want biology", sent["index_name"])
	}

	// (3) The mock's result must be formatted into the response — not a stub string.
	resp := res.Response()
	if strings.Contains(resp, "Would query") || strings.Contains(resp, "In production") {
		t.Fatalf("response is a stub string, not real results: %q", resp)
	}
	if !strings.Contains(resp, "powerhouse of the cell") {
		t.Errorf("response does not contain the mock result content; got %q", resp)
	}
	if !strings.Contains(resp, "biology.md") {
		t.Errorf("response does not contain the result metadata filename; got %q", resp)
	}
}
