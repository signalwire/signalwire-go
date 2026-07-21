// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Command pagination-dump is the Go port's PAGINATION-CORPUS program for the
// cross-port pagination differ (porting-sdk/scripts/diff_port_pagination.py).
//
// It runs the shared pagination corpus (porting-sdk/scripts/pagination_corpus.py
// — the single source of truth, mirrored natively below) through the Go SDK's
// REST Paginator and prints ONE JSON object mapping
//
//	corpus-id -> classification
//
// to stdout, where each classification is the shared cross-port shape:
//
//	empty_page_with_next:   { "continued_past_empty": bool, "items_seen": int }
//	repeating_cursor_guard: { "loop_guarded": bool, "hung": bool }
//	exhaustion:             { "terminated": bool, "total_items": int }
//
// The differ builds the golden by running the same corpus against the Python
// reference PaginatedIterator; this program emits the byte-identical
// classifications for a passing port.
//
// Each fixture is driven against an in-process httptest mock that serves the
// fixture's page bodies (data + links.next) FIFO on the list endpoint — exactly
// the capability the mock_signalwire scenario store gives the Python differ, with
// no mock change. The repeating-cursor fixture is walked under a bounded watchdog:
// a paginator with no cycle guard would loop forever, and the watchdog reds it
// LOUD (hung:true) rather than hanging this program.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/rest"
)

// LIST_PATH / the page-cursor URL builder MUST match pagination_corpus.py exactly
// so the armed page bodies are byte-identical across ports.
const listPath = "/api/fabric/addresses"

func nextURL(tok string) string {
	return "http://mock.test" + listPath + "?page_token=" + tok
}

// page is one armed page body: data items + an optional links.next cursor.
type page struct {
	data []map[string]any
	next string // "" => no links.next (terminal page)
}

func (p page) body() map[string]any {
	data := make([]any, len(p.data))
	for i, d := range p.data {
		data[i] = d
	}
	links := map[string]any{}
	if p.next != "" {
		links["next"] = p.next
	}
	return map[string]any{"data": data, "links": links}
}

// fixture is one corpus case: a kind discriminator + the ordered pages to arm.
type fixture struct {
	id    string
	kind  string
	pages []page
}

// CORPUS mirrors pagination_corpus.py CORPUS (the single source of truth).
var corpus = []fixture{
	{
		id:   "empty_page_with_next",
		kind: "empty_page_with_next",
		pages: []page{
			{data: nil, next: nextURL("EP_page2")},
			{data: []map[string]any{{"id": "found-after-empty"}}, next: ""},
		},
	},
	{
		id:   "repeating_cursor_guard",
		kind: "repeating_cursor_guard",
		pages: []page{
			{data: []map[string]any{{"id": "loop-1"}}, next: nextURL("LOOP")},
			{data: []map[string]any{{"id": "loop-2"}}, next: nextURL("LOOP")},
		},
	},
	{
		id:   "exhaustion",
		kind: "exhaustion",
		pages: []page{
			{data: []map[string]any{{"id": "x-1"}, {"id": "x-2"}}, next: nextURL("EX_page2")},
			{data: []map[string]any{{"id": "x-3"}, {"id": "x-4"}}, next: nextURL("EX_page3")},
			{data: []map[string]any{{"id": "x-5"}}, next: ""},
		},
	},
}

// serveFixture stands up an in-process mock that serves the fixture's pages FIFO
// on listPath (the go paginator carries only the next cursor's QUERY into the
// next request, re-hitting listPath — so serving strictly in order is correct).
// A request beyond the armed pages returns the last page again (a terminal, next-
// less page for exhaustion; for the repeating-cursor fixture the paginator's cycle
// guard stops it before that, and the watchdog catches a broken guard).
func serveFixture(f fixture) *httptest.Server {
	var idx int32 = -1
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := int(atomic.AddInt32(&idx, 1))
		var pg page
		if i < len(f.pages) {
			pg = f.pages[i]
		} else {
			// Past the armed sequence: serve a terminal empty page so a
			// mis-guarded walk still terminates the HTTP layer (the classifier
			// separately marks hung via the watchdog).
			pg = page{data: nil, next: ""}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		enc, _ := json.Marshal(pg.body())
		_, _ = w.Write(enc)
	}))
}

// walk drives the port paginator over the served fixture, collecting item ids,
// under a bounded watchdog. Returns (ids, hung).
func walk(srv *httptest.Server) (ids []string, hung bool) {
	client, err := rest.NewRestClient("pagination_proj", "pagination_tok", "mock.invalid")
	if err != nil {
		panic(fmt.Sprintf("pagination-dump: new client: %v", err))
	}
	client.SetBaseURL(srv.URL)

	it := client.Fabric.Addresses.Paginate(context.Background(), nil)

	done := make(chan []string, 1)
	go func() {
		var got []string
		for {
			items, hasMore, werr := it.Next()
			if werr != nil {
				break
			}
			for _, item := range items {
				if id, ok := item["id"].(string); ok {
					got = append(got, id)
				}
			}
			if !hasMore {
				break
			}
		}
		done <- got
	}()

	select {
	case got := <-done:
		return got, false
	case <-time.After(3 * time.Second):
		// No cycle guard → the walk never terminates. Red it LOUD.
		return nil, true
	}
}

func classify(f fixture) map[string]any {
	ids, hung := walk(serveFixture(f))
	switch f.kind {
	case "empty_page_with_next":
		return map[string]any{
			"continued_past_empty": len(ids) == 1 && ids[0] == "found-after-empty",
			"items_seen":           len(ids),
		}
	case "repeating_cursor_guard":
		return map[string]any{
			"loop_guarded": !hung && len(ids) == 2 && ids[0] == "loop-1" && ids[1] == "loop-2",
			"hung":         hung,
		}
	case "exhaustion":
		want := []string{"x-1", "x-2", "x-3", "x-4", "x-5"}
		terminated := !hung && equalIDs(ids, want)
		return map[string]any{
			"terminated":  terminated,
			"total_items": len(ids),
		}
	default:
		panic("pagination-dump: unknown fixture kind " + f.kind)
	}
}

func equalIDs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func main() {
	out := map[string]any{}
	for _, f := range corpus {
		out[f.id] = classify(f)
	}
	enc, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "pagination-dump: marshal:", err)
		os.Exit(1)
	}
	fmt.Println(string(enc))
}
