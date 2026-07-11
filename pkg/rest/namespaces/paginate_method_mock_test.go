// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed tests for the resource-level Paginate() method — the Go-idiom
// equivalent of Python's ReadResource.paginate() (signalwire/rest/_base.py). It
// wires the resource layer to the Paginator so a caller pages through a list
// endpoint without hand-building the path + links.next cursor loop.
//
// Both surfaces are covered:
//   - the GENERATED read-only subclass form (client.Fabric.Addresses.Paginate),
//     synthesized directly on the resource (FabricAddresses embeds the method-less
//     Resource), and
//   - the base CrudResource.Paginate form, inherited by every CRUD resource
//     (client.Fabric.AIAgents embeds CrudWithAddresses -> CrudResource).
//
// Each test stages two FIFO pages — the first carrying a links.next cursor, the
// second terminal — pages through, and asserts every item is yielded in order and
// that the second fetch carried the cursor parsed from the first response.

package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

const aiAgentsPath = "/api/fabric/resources/ai_agents"

// pushCursorPages stages a cursor page then a terminal page on endpointID, whose
// links.next points at nextURL (parsed for the second fetch's query).
func pushCursorPages(t *testing.T, mock *mocktest.Harness, endpointID, nextURL string) {
	t.Helper()
	mock.PushScenario(t, endpointID, 200, map[string]any{
		"data": []any{
			map[string]any{"id": "row-1", "name": "first"},
			map[string]any{"id": "row-2", "name": "second"},
		},
		"links": map[string]any{"next": nextURL},
	})
	mock.PushScenario(t, endpointID, 200, map[string]any{
		"data":  []any{map[string]any{"id": "row-3", "name": "third"}},
		"links": map[string]any{},
	})
}

// collectAll pages a paginator to exhaustion, returning every item id in order.
type pager interface {
	Next() ([]map[string]any, bool, error)
}

func collectIDs(t *testing.T, it pager) []string {
	t.Helper()
	var ids []string
	for {
		items, hasMore, err := it.Next()
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		for _, m := range items {
			id, _ := m["id"].(string)
			ids = append(ids, id)
		}
		if !hasMore {
			return ids
		}
	}
}

func assertRowsAndCursor(t *testing.T, ids []string, mock *mocktest.Harness, path string) {
	t.Helper()
	want := []string{"row-1", "row-2", "row-3"}
	if len(ids) != len(want) {
		t.Fatalf("collected %v, want %v", ids, want)
	}
	for i, w := range want {
		if ids[i] != w {
			t.Errorf("ids[%d]=%q want %q", i, ids[i], w)
		}
	}
	var gets []mocktest.JournalEntry
	for _, e := range mock.Journal(t) {
		if e.Path == path {
			gets = append(gets, e)
		}
	}
	if len(gets) != 2 {
		t.Fatalf("expected 2 paginated GETs at %s, got %d", path, len(gets))
	}
	if cv := gets[1].QueryParams["cursor"]; len(cv) != 1 || cv[0] != "page2" {
		t.Errorf("second fetch missing cursor=page2: %v", gets[1].QueryParams)
	}
}

// TestReadResourcePaginate_WalksAllPages drives the generated read-only
// subclass's Paginate() (FabricAddresses).
func TestReadResourcePaginate_WalksAllPages(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	pushCursorPages(t, mock, fabricAddressesEndpointID,
		"http://example.com/api/fabric/addresses?cursor=page2")

	it := client.Fabric.Addresses.Paginate(nil)
	if it == nil {
		t.Fatal("Paginate returned nil")
	}
	// Construction must not fetch.
	if n := len(mock.Journal(t)); n != 0 {
		t.Fatalf("Paginate must not fetch on construction; journal has %d entries", n)
	}

	ids := collectIDs(t, it)
	assertRowsAndCursor(t, ids, mock, fabricAddressesPath)
}

// TestCrudResourcePaginate_WalksAllPages drives the base CrudResource.Paginate()
// inherited by a CRUD resource (AIAgents).
func TestCrudResourcePaginate_WalksAllPages(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	pushCursorPages(t, mock, "fabric.list_ai_agents",
		"http://example.com/api/fabric/resources/ai_agents?cursor=page2")

	it := client.Fabric.AIAgents.Paginate(nil)
	if it == nil {
		t.Fatal("Paginate returned nil")
	}
	ids := collectIDs(t, it)
	assertRowsAndCursor(t, ids, mock, aiAgentsPath)
}
