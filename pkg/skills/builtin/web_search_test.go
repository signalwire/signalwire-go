package builtin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/skills"
)

// ---------------------------------------------------------------------------
// response_prefix / response_postfix
//
// Ports Python commit 8aad242 (prefix/postfix to web_search skill). The
// wrapper is applied only on the success path — error and no-results
// branches are not wrapped. Wire keys stay snake_case to match the SWML
// contract.
// ---------------------------------------------------------------------------

func TestWebSearch_Setup_ReadsPrefixPostfixParams(t *testing.T) {
	factory := skills.GetSkillFactory("web_search")
	s := factory(map[string]any{
		"api_key":          "k",
		"search_engine_id": "e",
		"response_prefix":  "PREFIX",
		"response_postfix": "POSTFIX",
	})
	if !s.Setup() {
		t.Fatal("Setup failed")
	}
	ws := s.(*WebSearchSkill)
	if ws.responsePrefix != "PREFIX" {
		t.Errorf("responsePrefix = %q, want PREFIX", ws.responsePrefix)
	}
	if ws.responsePostfix != "POSTFIX" {
		t.Errorf("responsePostfix = %q, want POSTFIX", ws.responsePostfix)
	}
}

func TestWebSearch_Setup_DefaultsPrefixPostfixToEmpty(t *testing.T) {
	factory := skills.GetSkillFactory("web_search")
	s := factory(map[string]any{
		"api_key":          "k",
		"search_engine_id": "e",
	})
	if !s.Setup() {
		t.Fatal("Setup failed")
	}
	ws := s.(*WebSearchSkill)
	if ws.responsePrefix != "" {
		t.Errorf("default responsePrefix = %q, want empty", ws.responsePrefix)
	}
	if ws.responsePostfix != "" {
		t.Errorf("default responsePostfix = %q, want empty", ws.responsePostfix)
	}
}

// newWebSearchStubServer returns an httptest.Server that satisfies BOTH the
// Google CSE API call (path /customsearch/v1) and the per-result content fetch
// (any other path) used by extractHTMLContent. The fetched HTML is intentionally
// long, content-rich, and matches the query so the quality threshold is met.
func newWebSearchStubServer(t *testing.T, query string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/customsearch/v1") {
			// One result that points back at this same server for content.
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"items":[{"title":"Result about %[1]s","link":"%[2]s/page","snippet":"info on %[1]s"}]}`, query, "http://"+r.Host)
			return
		}
		// Content fetch — return enough relevant text to clear the quality bar.
		w.Header().Set("Content-Type", "text/html")
		body := strings.Repeat(fmt.Sprintf("This document discusses %s in detail. Researchers studying %s have found important insights. The topic of %s is covered comprehensively in this article. ", query, query, query), 30)
		fmt.Fprintf(w, "<html><body><article>%s</article></body></html>", body)
	}))
	t.Setenv("WEB_SEARCH_BASE_URL", srv.URL)
	return srv
}

func runWebSearchHandler(t *testing.T, params map[string]any, query string) string {
	t.Helper()
	srv := newWebSearchStubServer(t, query)
	t.Cleanup(srv.Close)

	// Force defaults that make the test deterministic and fast.
	merged := map[string]any{
		"api_key":           "k",
		"search_engine_id":  "e",
		"num_results":       1,
		"oversample_factor": 1.0,
		"delay":             0.0,
		"min_quality_score": 0.0, // any content from the stub passes
	}
	for k, v := range params {
		merged[k] = v
	}

	factory := skills.GetSkillFactory("web_search")
	s := factory(merged)
	if !s.Setup() {
		t.Fatal("Setup failed")
	}
	ws := s.(*WebSearchSkill)
	res := ws.handleWebSearch(map[string]any{"query": query}, nil)
	m := res.ToMap()
	resp, _ := m["response"].(string)
	return resp
}

func TestWebSearch_Handler_WrapsSuccessWithPrefixAndPostfix(t *testing.T) {
	resp := runWebSearchHandler(t, map[string]any{
		"response_prefix":  "BEFORE-RESULTS",
		"response_postfix": "AFTER-RESULTS",
	}, "golang")
	if !strings.HasPrefix(resp, "BEFORE-RESULTS\n\n") {
		t.Errorf("response should start with prefix + two newlines; got %.80q", resp)
	}
	if !strings.HasSuffix(resp, "\n\nAFTER-RESULTS") {
		t.Errorf("response should end with two newlines + postfix; got %.80q", resp[len(resp)-80:])
	}
	if !strings.Contains(resp, "Quality web search results for 'golang'") {
		t.Errorf("response should contain the standard header between prefix and postfix; got %q", resp)
	}
}

func TestWebSearch_Handler_NoWrappingWhenPrefixPostfixEmpty(t *testing.T) {
	// Compare against an explicit-prefix-only run on the same query so any
	// natural trailing newlines in the body cancel out; only the prefix
	// itself differs between the two responses.
	resp := runWebSearchHandler(t, nil, "rust")
	if !strings.HasPrefix(resp, "Quality web search results for 'rust'") {
		t.Errorf("unwrapped response must start with the standard header; got %.80q", resp)
	}
	// Run again with a distinctive prefix and confirm the unwrapped form
	// does NOT contain that marker — i.e. wrapping is the only delta.
	wrapped := runWebSearchHandler(t, map[string]any{
		"response_prefix":  "PFX-MARKER",
		"response_postfix": "SFX-MARKER",
	}, "rust")
	if strings.Contains(resp, "PFX-MARKER") || strings.Contains(resp, "SFX-MARKER") {
		t.Errorf("unwrapped response should not contain marker text from wrapped run; got %q", resp)
	}
	if !strings.Contains(wrapped, "PFX-MARKER") || !strings.Contains(wrapped, "SFX-MARKER") {
		t.Errorf("wrapped response should contain both markers; got %q", wrapped)
	}
}

func TestWebSearch_Handler_PrefixOnlyWraps(t *testing.T) {
	resp := runWebSearchHandler(t, map[string]any{
		"response_prefix": "HEADS-UP",
	}, "kubernetes")
	if !strings.HasPrefix(resp, "HEADS-UP\n\nQuality web search results for 'kubernetes'") {
		t.Errorf("prefix-only response should start with prefix then header; got %.120q", resp)
	}
}

func TestWebSearch_Handler_PostfixOnlyWraps(t *testing.T) {
	resp := runWebSearchHandler(t, map[string]any{
		"response_postfix": "DISCLAIMER",
	}, "python")
	if !strings.HasSuffix(resp, "\n\nDISCLAIMER") {
		t.Errorf("postfix-only response should end with two newlines + postfix; got %.120q", resp[len(resp)-120:])
	}
	if !strings.HasPrefix(resp, "Quality web search results for 'python'") {
		t.Errorf("postfix-only response should start with the standard header; got %.80q", resp)
	}
}

// Error path is unchanged — prefix/postfix do not wrap the error message.
// We can't easily induce searchGoogle failure via the stub (it would still
// return 200), but the no-results branch is a related "non-success" path
// that must remain unwrapped.
func TestWebSearch_Handler_NoResultsNotWrappedByPrefixPostfix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CSE returns zero items → handler hits no-results branch.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("WEB_SEARCH_BASE_URL", srv.URL)

	factory := skills.GetSkillFactory("web_search")
	s := factory(map[string]any{
		"api_key":           "k",
		"search_engine_id":  "e",
		"num_results":       1,
		"oversample_factor": 1.0,
		"delay":             0.0,
		"response_prefix":   "PFX",
		"response_postfix":  "SFX",
	})
	if !s.Setup() {
		t.Fatal("Setup failed")
	}
	ws := s.(*WebSearchSkill)
	res := ws.handleWebSearch(map[string]any{"query": "obscurequery"}, nil)
	resp, _ := res.ToMap()["response"].(string)
	if strings.Contains(resp, "PFX") || strings.Contains(resp, "SFX") {
		t.Errorf("no-results path must not be wrapped by prefix/postfix; got %q", resp)
	}
}
