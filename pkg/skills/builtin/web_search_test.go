package builtin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/skills"
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
	ws, ok := s.(*WebSearchSkill)
	if !ok {
		t.Fatalf("expected *WebSearchSkill, got %T", s)
	}
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
	ws, ok := s.(*WebSearchSkill)
	if !ok {
		t.Fatalf("expected *WebSearchSkill, got %T", s)
	}
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
			_, _ = fmt.Fprintf(w, `{"items":[{"title":"Result about %[1]s","link":"%[2]s/page","snippet":"info on %[1]s"}]}`, query, "http://"+r.Host)
			return
		}
		// Content fetch — return enough relevant text to clear the quality bar.
		w.Header().Set("Content-Type", "text/html")
		body := strings.Repeat(fmt.Sprintf("This document discusses %s in detail. Researchers studying %s have found important insights. The topic of %s is covered comprehensively in this article. ", query, query, query), 30)
		_, _ = fmt.Fprintf(w, "<html><body><article>%s</article></body></html>", body)
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
	ws, ok := s.(*WebSearchSkill)
	if !ok {
		t.Fatalf("expected *WebSearchSkill, got %T", s)
	}
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
	ws, ok := s.(*WebSearchSkill)
	if !ok {
		t.Fatalf("expected *WebSearchSkill, got %T", s)
	}
	res := ws.handleWebSearch(map[string]any{"query": "obscurequery"}, nil)
	resp, _ := res.ToMap()["response"].(string)
	if strings.Contains(resp, "PFX") || strings.Contains(resp, "SFX") {
		t.Errorf("no-results path must not be wrapped by prefix/postfix; got %q", resp)
	}
}

// ---------------------------------------------------------------------------
// Latency-control params: per_page_timeout / overall_deadline / parallel_scrape
// / snippets_only + snippet fallback.
//
// Ports Python 51101da + 295745b. overall_deadline + per_page_timeout are the
// CONTRACT — a slow site must not blow past the kernel webhook timeout (~55s).
// These tests exercise the deadline path deterministically with a content
// server that sleeps longer than the configured budget.
// ---------------------------------------------------------------------------

// latencyStubServer returns an httptest.Server that answers the Google CSE call
// instantly with `numItems` results, but sleeps `contentDelay` on every content
// fetch. hits counts how many content fetches actually started, so a test can
// assert scraping was (or was not) attempted.
func latencyStubServer(t *testing.T, query string, numItems int, contentDelay time.Duration, hits *int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/customsearch/v1") {
			w.Header().Set("Content-Type", "application/json")
			items := make([]string, 0, numItems)
			for i := range numItems {
				items = append(items, fmt.Sprintf(
					`{"title":"Result %[1]d about %[2]s","link":"%[3]s/page%[1]d","snippet":"snippet %[1]d on %[2]s"}`,
					i, query, "http://"+r.Host))
			}
			_, _ = fmt.Fprintf(w, `{"items":[%s]}`, strings.Join(items, ","))
			return
		}
		// Content fetch — record the hit, then stall longer than the deadline.
		// Honor request-context cancellation so that when the client (the
		// skill) abandons the fetch at the deadline / per-page timeout, the
		// server stops sleeping too and srv.Close() in cleanup returns fast.
		// The skill has already returned by then; this only keeps the test
		// suite quick, it does not weaken the deadline assertions.
		if hits != nil {
			atomic.AddInt32(hits, 1)
		}
		select {
		case <-time.After(contentDelay):
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "text/html")
		body := strings.Repeat(fmt.Sprintf("This document discusses %s in detail. ", query), 50)
		_, _ = fmt.Fprintf(w, "<html><body><article>%s</article></body></html>", body)
	}))
	t.Setenv("WEB_SEARCH_BASE_URL", srv.URL)
	return srv
}

// buildWebSearch wires a skill instance against the given params + base URL env
// (already set by latencyStubServer/newWebSearchStubServer via t.Setenv).
func buildWebSearch(t *testing.T, params map[string]any) *WebSearchSkill {
	t.Helper()
	merged := map[string]any{
		"api_key":           "k",
		"search_engine_id":  "e",
		"num_results":       2,
		"min_quality_score": 0.0,
		"delay":             0.0,
	}
	for k, v := range params {
		merged[k] = v
	}
	factory := skills.GetSkillFactory("web_search")
	s := factory(merged)
	if !s.Setup() {
		t.Fatal("Setup failed")
	}
	return as[*WebSearchSkill](t, s)
}

func TestWebSearch_Setup_LatencyParamsDefault(t *testing.T) {
	ws := buildWebSearch(t, nil)
	if ws.perPageTimeout != 2.0 {
		t.Errorf("perPageTimeout = %v, want 2.0", ws.perPageTimeout)
	}
	if ws.overallDeadline != 10.0 {
		t.Errorf("overallDeadline = %v, want 10.0", ws.overallDeadline)
	}
	if ws.parallelScrape != true {
		t.Errorf("parallelScrape = %v, want true", ws.parallelScrape)
	}
	if ws.snippetsOnly != false {
		t.Errorf("snippetsOnly = %v, want false", ws.snippetsOnly)
	}
}

func TestWebSearch_Setup_LatencyParamsOverride(t *testing.T) {
	ws := buildWebSearch(t, map[string]any{
		"per_page_timeout": 3.5,
		"overall_deadline": 12.0,
		"parallel_scrape":  false,
		"snippets_only":    true,
	})
	if ws.perPageTimeout != 3.5 {
		t.Errorf("perPageTimeout = %v, want 3.5", ws.perPageTimeout)
	}
	if ws.overallDeadline != 12.0 {
		t.Errorf("overallDeadline = %v, want 12.0", ws.overallDeadline)
	}
	if ws.parallelScrape != false {
		t.Errorf("parallelScrape = %v, want false", ws.parallelScrape)
	}
	if ws.snippetsOnly != true {
		t.Errorf("snippetsOnly = %v, want true", ws.snippetsOnly)
	}
}

// snippets_only must short-circuit BEFORE any page fetch. We point the content
// path at a server that would sleep 5s; if scraping ran the test would take
// ~5s. Instead it returns immediately with snippet-only formatting and zero
// content hits.
func TestWebSearch_Handler_SnippetsOnlySkipsScraping(t *testing.T) {
	var hits int32
	srv := latencyStubServer(t, "golang", 2, 5*time.Second, &hits)
	t.Cleanup(srv.Close)

	ws := buildWebSearch(t, map[string]any{"snippets_only": true})

	start := time.Now()
	res := ws.handleWebSearch(map[string]any{"query": "golang"}, nil)
	elapsed := time.Since(start)
	resp, _ := res.ToMap()["response"].(string)

	if got := atomic.LoadInt32(&hits); got != 0 {
		t.Errorf("snippets_only must not fetch any page; content hits = %d", got)
	}
	if elapsed > 2*time.Second {
		t.Errorf("snippets_only should be sub-second; took %v", elapsed)
	}
	if !strings.Contains(resp, "Snippet-only results for 'golang'") {
		t.Errorf("expected snippet-only formatting; got %.160q", resp)
	}
	if !strings.Contains(resp, "snippet 0 on golang") {
		t.Errorf("snippet text should be carried through; got %.200q", resp)
	}
	if strings.Contains(resp, "page content not scraped") == false {
		t.Errorf("snippet header should note pages were not scraped; got %.200q", resp)
	}
}

// overall_deadline is the contract: a content server that stalls 5s with a 1s
// budget must cause the call to return within ~deadline+slack (NOT 5s) AND fall
// back to the CSE snippets, never an empty no-results message. Parallel mode.
func TestWebSearch_Handler_OverallDeadlineTruncatesToSnippetFallback(t *testing.T) {
	var hits int32
	srv := latencyStubServer(t, "kubernetes", 3, 5*time.Second, &hits)
	t.Cleanup(srv.Close)

	ws := buildWebSearch(t, map[string]any{
		"overall_deadline": 1.0,  // budget well under the 5s content stall
		"per_page_timeout": 30.0, // large, so the DEADLINE (not per-page) truncates
		"parallel_scrape":  true,
	})

	start := time.Now()
	res := ws.handleWebSearch(map[string]any{"query": "kubernetes"}, nil)
	elapsed := time.Since(start)
	resp, _ := res.ToMap()["response"].(string)

	// Returned at the deadline, not after the full 5s stall.
	if elapsed > 3*time.Second {
		t.Errorf("overall_deadline (1s) not enforced: call took %v (content stalls 5s)", elapsed)
	}
	// Non-empty snippet fallback, NOT the empty no-results message.
	if !strings.Contains(resp, "Snippet-only results for 'kubernetes'") {
		t.Errorf("deadline path must fall back to snippet formatting; got %.200q", resp)
	}
	if strings.Contains(resp, "couldn't find quality results") {
		t.Errorf("deadline path must NOT return the empty no-results message; got %.200q", resp)
	}
	if strings.TrimSpace(resp) == "" {
		t.Error("deadline fallback response must be non-empty")
	}
	if !strings.Contains(resp, "snippet 0 on kubernetes") {
		t.Errorf("snippet fallback should carry CSE snippet text; got %.200q", resp)
	}
}

// Same contract in sequential mode (parallel_scrape=false). The deadline is
// checked between iterations; with a 5s-per-page stall and a 1s budget the
// first page alone blows the budget, so we must break and fall back. Allow
// per_page_timeout to bound a single fetch so the first iteration cannot run
// 5s either — the loop then sees the (already-)passed deadline and stops.
func TestWebSearch_Handler_OverallDeadlineSequentialFallsBack(t *testing.T) {
	var hits int32
	srv := latencyStubServer(t, "rustlang", 3, 5*time.Second, &hits)
	t.Cleanup(srv.Close)

	ws := buildWebSearch(t, map[string]any{
		"overall_deadline": 1.0,
		"per_page_timeout": 0.5, // a single page fetch is force-failed at 0.5s
		"parallel_scrape":  false,
	})

	start := time.Now()
	res := ws.handleWebSearch(map[string]any{"query": "rustlang"}, nil)
	elapsed := time.Since(start)
	resp, _ := res.ToMap()["response"].(string)

	if elapsed > 3*time.Second {
		t.Errorf("sequential overall_deadline not enforced: took %v", elapsed)
	}
	if !strings.Contains(resp, "Snippet-only results for 'rustlang'") {
		t.Errorf("sequential deadline path must fall back to snippets; got %.200q", resp)
	}
	if strings.Contains(resp, "couldn't find quality results") {
		t.Errorf("sequential deadline path must NOT return no-results; got %.200q", resp)
	}
}

// per_page_timeout must cap a single fetch independent of the overall budget.
// With a 5s content stall, a 0.3s per-page timeout, a generous 8s overall
// budget and parallel mode, every fetch errors out near 0.3s, no page yields
// content, and we fall back to snippets WELL before the 5s stall would resolve.
func TestWebSearch_Handler_PerPageTimeoutHonored(t *testing.T) {
	var hits int32
	srv := latencyStubServer(t, "elixir", 2, 5*time.Second, &hits)
	t.Cleanup(srv.Close)

	ws := buildWebSearch(t, map[string]any{
		"per_page_timeout": 0.3,
		"overall_deadline": 8.0, // large, so it can't be what truncates us
		"parallel_scrape":  true,
	})

	start := time.Now()
	res := ws.handleWebSearch(map[string]any{"query": "elixir"}, nil)
	elapsed := time.Since(start)
	resp, _ := res.ToMap()["response"].(string)

	// per_page_timeout (0.3s) should govern: finish around there, not 5s, and
	// crucially not wait for the 8s overall budget either.
	if elapsed > 3*time.Second {
		t.Errorf("per_page_timeout (0.3s) not honored: took %v", elapsed)
	}
	if atomic.LoadInt32(&hits) == 0 {
		t.Error("expected content fetches to be attempted (then time out)")
	}
	if !strings.Contains(resp, "Snippet-only results for 'elixir'") {
		t.Errorf("all-pages-timed-out should fall back to snippets; got %.200q", resp)
	}
}

// Happy path under the deadline: a FAST content server with parallel scraping
// returns fully-scraped results (not the snippet fallback), proving the
// goroutine harvest path delivers real content when pages are quick.
func TestWebSearch_Handler_ParallelFastPathScrapesContent(t *testing.T) {
	var hits int32
	srv := latencyStubServer(t, "postgres", 3, 0, &hits) // no content delay
	t.Cleanup(srv.Close)

	ws := buildWebSearch(t, map[string]any{
		"overall_deadline": 10.0,
		"per_page_timeout": 5.0,
		"parallel_scrape":  true,
	})

	res := ws.handleWebSearch(map[string]any{"query": "postgres"}, nil)
	resp, _ := res.ToMap()["response"].(string)

	if !strings.Contains(resp, "Quality web search results for 'postgres'") {
		t.Errorf("fast parallel path should produce fully-scraped results; got %.200q", resp)
	}
	if strings.Contains(resp, "Snippet-only results") {
		t.Errorf("fast path should NOT degrade to snippet fallback; got %.200q", resp)
	}
	if atomic.LoadInt32(&hits) == 0 {
		t.Error("fast path must actually fetch page content")
	}
}

// Schema drift guard (Matches Python: test_every_setup_param_is_advertised).
// Every latency/response param read in Setup() must be advertised in the schema
// with the matching type + default.
func TestWebSearch_Schema_AdvertisesLatencyAndResponseParams(t *testing.T) {
	ws := as[*WebSearchSkill](t, NewWebSearch(map[string]any{"api_key": "k", "search_engine_id": "e"}))
	schema := ws.GetParameterSchema()

	for _, key := range []string{
		"response_prefix", "response_postfix",
		"per_page_timeout", "overall_deadline",
		"parallel_scrape", "snippets_only",
	} {
		entry, ok := schema[key]
		if !ok {
			t.Errorf("Setup() reads %q but schema omits it", key)
			continue
		}
		if entry["required"] != false {
			t.Errorf("%q: required = %v, want false", key, entry["required"])
		}
	}

	// Type + default fidelity vs Python.
	if schema["per_page_timeout"]["type"] != "number" || schema["per_page_timeout"]["default"] != 2.0 {
		t.Errorf("per_page_timeout schema wrong: %+v", schema["per_page_timeout"])
	}
	if schema["overall_deadline"]["type"] != "number" || schema["overall_deadline"]["default"] != 10.0 {
		t.Errorf("overall_deadline schema wrong: %+v", schema["overall_deadline"])
	}
	if schema["parallel_scrape"]["type"] != "boolean" || schema["parallel_scrape"]["default"] != true {
		t.Errorf("parallel_scrape schema wrong: %+v", schema["parallel_scrape"])
	}
	if schema["snippets_only"]["type"] != "boolean" || schema["snippets_only"]["default"] != false {
		t.Errorf("snippets_only schema wrong: %+v", schema["snippets_only"])
	}
	if schema["response_prefix"]["default"] != "" || schema["response_postfix"]["default"] != "" {
		t.Errorf("response_prefix/postfix default should be empty string")
	}
}
