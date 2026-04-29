// Example: relay_audit_harness
//
// Audit-only harness — runs a RelayClient against a local fixture that
// speaks JSON-RPC 2.0 + the SignalWire RELAY handshake. Used by
// porting-sdk/scripts/audit_relay_handshake.py to prove the Go RELAY
// client opens a real WebSocket, runs the connect handshake, subscribes
// to a context, and dispatches an inbound event end-to-end.
//
// Contract (from porting-sdk SUBAGENT_PLAYBOOK.md):
//
//   - Reads SIGNALWIRE_RELAY_HOST    (e.g. "127.0.0.1:5050")
//   - Reads SIGNALWIRE_RELAY_SCHEME  (e.g. "ws" — fixture serves plain
//     WebSocket; the SDK honors this env var in its connect URL builder)
//   - Reads SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, SIGNALWIRE_CONTEXTS
//   - Connects, subscribes, waits up to 5s for one inbound event
//   - When the event arrives, sends a JSON-RPC `signalwire.event`
//     notification back so the fixture sees the dispatch ack
//   - Exits 0 on success, non-zero on error
//
// Not for production use. The harness's whole purpose is to give the
// audit a small, fast binary to drive its fixture against.
package main

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/signalwire/signalwire-go/pkg/relay"
)

func main() {
	contexts := strings.Split(os.Getenv("SIGNALWIRE_CONTEXTS"), ",")
	cleaned := contexts[:0]
	for _, c := range contexts {
		if c = strings.TrimSpace(c); c != "" {
			cleaned = append(cleaned, c)
		}
	}
	if len(cleaned) == 0 {
		cleaned = []string{"audit_ctx"}
	}

	client := relay.NewRelayClient(
		relay.WithProject(os.Getenv("SIGNALWIRE_PROJECT_ID")),
		relay.WithToken(os.Getenv("SIGNALWIRE_API_TOKEN")),
		// Space is required by the option helpers but ignored by the
		// connect() URL builder when SIGNALWIRE_RELAY_HOST is set.
		relay.WithSpace("audit"),
		relay.WithContexts(cleaned...),
	)

	var eventReceived atomic.Bool
	client.OnEvent(func(eventType string, params map[string]any) {
		// The audit fixture marks "event_dispatched" only on a frame
		// whose method == "signalwire.event" coming from the client.
		// Send that frame so the fixture sees the dispatch ack.
		_ = client.Notify("signalwire.event", map[string]any{
			"event_type": eventType,
			"params":     params,
			"audit_ack":  true,
		})
		eventReceived.Store(true)
	})

	// Run() is blocking: connect + authenticate + subscribe + read loop.
	// Drive it from a goroutine so we can exit cleanly when the event
	// arrives or the deadline expires.
	runErr := make(chan error, 1)
	go func() { runErr <- client.Run() }()

	deadline := time.After(5 * time.Second)

waitLoop:
	for {
		select {
		case <-deadline:
			break waitLoop
		case err := <-runErr:
			// Run returned before we saw the event — the client
			// disconnected (server closed). Bail.
			fmt.Fprintf(os.Stderr, "relay_audit_harness: Run() returned early: %v\n", err)
			os.Exit(1)
		default:
			if eventReceived.Load() {
				break waitLoop
			}
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Close gracefully — fixture observes the WS close frame.
	client.Stop()

	if !eventReceived.Load() {
		fmt.Fprintln(os.Stderr, "relay_audit_harness: timed out waiting for inbound event")
		os.Exit(1)
	}
	fmt.Println("relay_audit_harness: ok")
	os.Exit(0)
}
