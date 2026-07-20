// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

//go:build live

// Plan 6.5 — real-server smoke lane. These tests hit the REAL SignalWire platform
// and are OPT-IN: they compile only under the `live` build tag AND run only when
// SWSDK_LIVE_TESTS=1 and the credentials env vars are present. They catch
// mock↔production drift the mock-backed suites cannot. Run:
//
//	SWSDK_LIVE_TESTS=1 go test -tags live ./tests/ -run TestLive -v
//
// Required env: SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, SIGNALWIRE_SPACE
// (RELAY also honors SIGNALWIRE_JWT_TOKEN). Absent creds → the tests skip cleanly.
package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/relay"
	"github.com/signalwire/signalwire-go/v3/pkg/rest"
	"github.com/signalwire/signalwire-go/v3/pkg/swml"
)

// liveGuard skips the test unless SWSDK_LIVE_TESTS=1 and the auth env vars are set.
func liveGuard(t *testing.T) (project, token, space string) {
	t.Helper()
	if os.Getenv("SWSDK_LIVE_TESTS") != "1" {
		t.Skip("SWSDK_LIVE_TESTS != 1 — skipping real-server smoke")
	}
	project = os.Getenv("SIGNALWIRE_PROJECT_ID")
	token = os.Getenv("SIGNALWIRE_API_TOKEN")
	space = os.Getenv("SIGNALWIRE_SPACE")
	if project == "" || token == "" || space == "" {
		t.Skip("SIGNALWIRE_PROJECT_ID/API_TOKEN/SPACE not all set — skipping real-server smoke")
	}
	return project, token, space
}

// TestLive_RESTAuthAndList proves auth + one REST list against the real platform.
func TestLive_RESTAuthAndList(t *testing.T) {
	project, token, space := liveGuard(t)
	client, err := rest.NewRestClient(project, token, space)
	if err != nil {
		t.Fatalf("NewRestClient: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// One list call: fabric addresses. Auth is exercised implicitly (a 401 would
	// surface as *SignalWireRestError).
	if _, err := client.Fabric.Addresses.List(ctx, map[string]string{"page_size": "1"}); err != nil {
		t.Fatalf("Fabric.Addresses.List: %v", err)
	}
}

// TestLive_SWMLRender proves one SWML document renders (local, but part of the
// smoke: the rendered JSON is what the live platform consumes).
func TestLive_SWMLRender(t *testing.T) {
	liveGuard(t)
	svc := swml.NewService(swml.WithName("live-smoke"))
	greeting := "Hello from the live smoke test."
	if err := svc.Play(swml.PlayOptions{URL: &greeting}); err != nil {
		t.Fatalf("Play: %v", err)
	}
	doc, err := svc.Render()
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if doc == "" {
		t.Fatal("rendered SWML is empty")
	}
}

// TestLive_RELAYConnect proves one RELAY WebSocket connect against the real
// platform, then disconnects.
func TestLive_RELAYConnect(t *testing.T) {
	project, token, space := liveGuard(t)
	client := relay.NewRelayClient(
		relay.WithProject(project),
		relay.WithToken(token),
		relay.WithSpace(space),
		relay.WithContexts("default"),
	)
	if err := client.Connect(); err != nil {
		t.Fatalf("RELAY Connect: %v", err)
	}
	client.Stop()
}
