// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_pagination_mock.py.
//
// PaginatedIterator wraps any HttpClient.Get call and walks paged
// responses following the links.next cursor. We test it end-to-end by:
//
//   1. Staging two FIFO scenarios on a known mock endpoint — the first
//      scenario has a ``links.next`` cursor, the second is the terminal page.
//   2. Iterating over a real PaginatedIterator wired to the SDK's
//      HttpClient pointed at the mock.
//   3. Asserting on the items collected and on the journal entries that
//      correspond to the two HTTP fetches.

package namespaces_test

import (
	"testing"

	rest "github.com/signalwire/signalwire-go/pkg/rest"
	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// fabric_addresses_path is stable across spec revisions and the mock
// returns a ``data + links`` shape by default.
const (
	fabricAddressesPath       = "/api/fabric/addresses"
	fabricAddressesEndpointID = "fabric.list_fabric_addresses"
)

// ---------- Constructor/initial state ----------

func TestPaginatedIterator_InitState(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	it := rest.NewPaginatedIterator(
		client.HttpClient(),
		fabricAddressesPath,
		map[string]string{"page_size": "2"},
		"data",
	)
	if it == nil {
		t.Fatal("NewPaginatedIterator returned nil")
	}
	// Constructor must not have fetched anything yet.
	entries := mock.Journal(t)
	if len(entries) != 0 {
		t.Errorf("journal must be empty after constructor; got %d entries", len(entries))
	}
}

// ---------- Pages through all items ----------

func TestPaginatedIterator_NextPagesThroughAllItems(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	// Stage two FIFO scenarios. First page has next cursor; second is terminal.
	mock.PushScenario(t, fabricAddressesEndpointID, 200, map[string]any{
		"data": []any{
			map[string]any{"id": "addr-1", "name": "first"},
			map[string]any{"id": "addr-2", "name": "second"},
		},
		"links": map[string]any{
			"next": "http://example.com/api/fabric/addresses?cursor=page2",
		},
	})
	mock.PushScenario(t, fabricAddressesEndpointID, 200, map[string]any{
		"data": []any{
			map[string]any{"id": "addr-3", "name": "third"},
		},
		"links": map[string]any{},
	})

	it := rest.NewPaginatedIterator(
		client.HttpClient(),
		fabricAddressesPath,
		nil,
		"data",
	)

	var collected []map[string]any
	if err := it.ForEach(func(item map[string]any) error {
		collected = append(collected, item)
		return nil
	}); err != nil {
		t.Fatalf("ForEach: %v", err)
	}

	// Three items total, in order.
	if len(collected) != 3 {
		t.Fatalf("got %d items, want 3: %v", len(collected), collected)
	}
	wantIDs := []string{"addr-1", "addr-2", "addr-3"}
	for i, want := range wantIDs {
		if collected[i]["id"] != want {
			t.Errorf("collected[%d].id = %v, want %s", i, collected[i]["id"], want)
		}
	}

	// Journal must have exactly two GETs at the same path.
	entries := mock.Journal(t)
	var gets []mocktest.JournalEntry
	for _, e := range entries {
		if e.Path == fabricAddressesPath {
			gets = append(gets, e)
		}
	}
	if len(gets) != 2 {
		t.Fatalf("expected 2 paginated GETs, got %d entries: %v", len(gets), gets)
	}
	// The second fetch carries cursor=page2 from the first response's links.next.
	cursorVals := gets[1].QueryParams["cursor"]
	if len(cursorVals) != 1 || cursorVals[0] != "page2" {
		t.Errorf("second fetch missing cursor=page2: %v", gets[1].QueryParams)
	}
}

// ---------- StopIteration when exhausted ----------

func TestPaginatedIterator_NextStopsWhenDone(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	// One terminal page.
	mock.PushScenario(t, fabricAddressesEndpointID, 200, map[string]any{
		"data": []any{
			map[string]any{"id": "only-one"},
		},
		"links": map[string]any{},
	})

	it := rest.NewPaginatedIterator(
		client.HttpClient(),
		fabricAddressesPath,
		nil,
		"data",
	)

	items, hasMore, err := it.Next()
	if err != nil {
		t.Fatalf("first Next: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("first page items = %d, want 1", len(items))
	}
	if items[0]["id"] != "only-one" {
		t.Errorf("first item id = %v", items[0]["id"])
	}
	// Terminal page: hasMore must be false.
	if hasMore {
		t.Error("expected hasMore=false on terminal page")
	}

	// Subsequent Next returns nil items, hasMore=false, no error (Go-idiom
	// equivalent of Python's StopIteration).
	items2, hasMore2, err2 := it.Next()
	if err2 != nil {
		t.Errorf("second Next: %v", err2)
	}
	if items2 != nil {
		t.Errorf("expected nil items after exhaustion, got %v", items2)
	}
	if hasMore2 {
		t.Error("expected hasMore=false after exhaustion")
	}
}

// ---------- ForEach early exit on error ----------

func TestPaginatedIterator_ForEachStopsOnError(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	mock.PushScenario(t, fabricAddressesEndpointID, 200, map[string]any{
		"data": []any{
			map[string]any{"id": "a"},
			map[string]any{"id": "b"},
			map[string]any{"id": "c"},
		},
		"links": map[string]any{},
	})

	it := rest.NewPaginatedIterator(
		client.HttpClient(),
		fabricAddressesPath,
		nil,
		"data",
	)

	stopErr := &stopError{msg: "stop here"}
	visited := 0
	err := it.ForEach(func(item map[string]any) error {
		visited++
		if visited == 2 {
			return stopErr
		}
		return nil
	})
	if err != stopErr {
		t.Errorf("ForEach error = %v, want stopErr", err)
	}
	if visited != 2 {
		t.Errorf("visited = %d, want 2", visited)
	}
}

// stopError is a sentinel used to verify ForEach surfaces the callback's
// returned error without further iteration.
type stopError struct{ msg string }

func (e *stopError) Error() string { return e.msg }

var _ = mocktest.JournalEntry{}
