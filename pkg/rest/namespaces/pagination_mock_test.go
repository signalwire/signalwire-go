// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_pagination_mock.py.
//
// These drive the LIVE Paginator — the value returned by a resource's Paginate()
// accessor (client.Fabric.Addresses.Paginate) — through the SDK's real HTTPClient
// pointed at the mock. The multi-page cursor walk + page_token round-trip is
// covered by paginate_method_mock_test.go; here we cover the two remaining
// behaviors: StopIteration-on-exhaustion and ForEach early-exit on a callback
// error. (The former orphan rest.PaginatedIterator these tests exercised was
// retired in plan 6.2-go — the resource-wired Paginator is the one users get.)

package namespaces_test

import (
	"context"
	"errors"
	"testing"

	"github.com/signalwire/signalwire-go/v3/pkg/rest/internal/mocktest"
)

// fabric_addresses_path is stable across spec revisions and the mock
// returns a “data + links“ shape by default.
const (
	fabricAddressesPath       = "/api/fabric/addresses"
	fabricAddressesEndpointID = "fabric.list_fabric_addresses"
)

// ---------- StopIteration when exhausted ----------

func TestPaginator_NextStopsWhenDone(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	// One terminal page.
	mock.PushScenario(t, fabricAddressesEndpointID, 200, map[string]any{
		"data":  []any{map[string]any{"id": "only-one"}},
		"links": map[string]any{},
	})

	it := client.Fabric.Addresses.Paginate(context.Background(), nil)

	items, hasMore, err := it.Next()
	if err != nil {
		t.Fatalf("first Next: %v", err)
	}
	if len(items) != 1 || items[0]["id"] != "only-one" {
		t.Fatalf("first page = %v, want [only-one]", items)
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

func TestPaginator_ForEachStopsOnError(t *testing.T) {
	t.Parallel()
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

	it := client.Fabric.Addresses.Paginate(context.Background(), nil)

	stopErr := &stopError{msg: "stop here"}
	visited := 0
	err := it.ForEach(func(item map[string]any) error {
		visited++
		if visited == 2 {
			return stopErr
		}
		return nil
	})
	if !errors.Is(err, stopErr) {
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
