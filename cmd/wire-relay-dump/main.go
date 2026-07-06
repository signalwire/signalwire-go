// Command wire-relay-dump is the Go port's WIRE-RELAY dump program for the
// cross-port relay differ (porting-sdk/scripts/diff_port_wire_relay.py).
//
// It captures, for each wire_relay_corpus case, the observable RELAY artifact:
//   - verb   : the {method, params} JSON-RPC frame a Call verb (or an Action
//     control-op) hands to the wire.
//   - client : the {method, params} frame a RelayClient call (execute / dial /
//     send_message) sends.
//   - event  : the decoded fields a typed event decoder extracts from a payload.
//
// It prints ONE JSON object mapping case-id -> artifact to stdout; the differ
// canonicalizes both sides (normalizing the random control_id to a sentinel) and
// byte-compares against the python oracle. Only stdout carries JSON.
//
// Frame capture: verb/client verbs send over a real *websocket.Conn, so this
// program stands up a tiny in-process mock RELAY WS server on a loopback port
// (pointed to via SIGNALWIRE_RELAY_HOST/SCHEME), completes the connect
// handshake, records each calling.*/messaging.* frame, and replies with a canned
// success. Event decoding is pure (no wire).
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/wire-relay-dump
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/signalwire/signalwire-go/pkg/relay"
)

const (
	node = "node-abc"
	call = "call-xyz"
	cid  = "ctl-123"
)

// mockRelay is a minimal in-process RELAY WebSocket server. It completes the
// signalwire.connect handshake, records every calling.*/messaging.* frame, and
// replies to each request with a canned success so the client's verbs proceed.
type mockRelay struct {
	upgrader websocket.Upgrader
	srv      *http.Server

	mu      sync.Mutex
	frames  map[string]map[string]any // method -> latest params
	conns   []*websocket.Conn
	dialArm string // when set, resolve dial for this tag on the next calling.dial
}

func newMockRelay() *mockRelay {
	return &mockRelay{
		upgrader: websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }},
		frames:   map[string]map[string]any{},
	}
}

func (m *mockRelay) lastFrame(method string) map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.frames[method]
}

func (m *mockRelay) handle(w http.ResponseWriter, r *http.Request) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	m.mu.Lock()
	m.conns = append(m.conns, conn)
	m.mu.Unlock()
	go m.readConn(conn)
}

func (m *mockRelay) readConn(conn *websocket.Conn) {
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg struct {
			ID     string          `json:"id"`
			Method string          `json:"method"`
			Params map[string]any  `json:"params"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		// An ACK/response to a server push (has result, no method) — ignore.
		if msg.Method == "" {
			continue
		}
		switch msg.Method {
		case "signalwire.connect":
			// Reply with protocol "default" (SendMessage defaults context to it).
			m.reply(conn, msg.ID, map[string]any{
				"protocol":  "default",
				"sessionid": "sess-1",
			})
		case "signalwire.receive":
			m.reply(conn, msg.ID, map[string]any{"code": "200"})
		default:
			// Record the frame's params (a calling.* / messaging.* verb).
			m.mu.Lock()
			m.frames[msg.Method] = msg.Params
			armed := m.dialArm
			m.mu.Unlock()

			switch msg.Method {
			case "calling.dial":
				m.reply(conn, msg.ID, map[string]any{"code": "200", "message": "Dialing"})
				if armed != "" {
					// Resolve dial: push calling.call.dial answered for the tag.
					go m.pushDialAnswered(conn, armed)
				}
			case "messaging.send":
				m.reply(conn, msg.ID, map[string]any{"code": "200", "message_id": "msg-1"})
			default:
				m.reply(conn, msg.ID, map[string]any{"code": "200"})
			}
		}
	}
}

func (m *mockRelay) reply(conn *websocket.Conn, id string, result map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

// push sends a server-initiated signalwire.event to the client.
func (m *mockRelay) push(conn *websocket.Conn, eventType string, params map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = conn.WriteJSON(map[string]any{
		"jsonrpc": "2.0",
		"id":      "evt-" + eventType,
		"method":  "signalwire.event",
		"params":  map[string]any{"event_type": eventType, "params": params},
	})
}

func (m *mockRelay) pushDialAnswered(conn *websocket.Conn, tag string) {
	time.Sleep(20 * time.Millisecond)
	m.push(conn, "calling.call.dial", map[string]any{
		"tag":        tag,
		"dial_state": "answered",
		"call":       map[string]any{"call_id": call, "node_id": node},
	})
}

func (m *mockRelay) pushInboundCall(conn *websocket.Conn) {
	m.push(conn, "calling.call.receive", map[string]any{
		"call_id":    call,
		"node_id":    node,
		"direction":  "inbound",
		"call_state": "created",
	})
}

func main() {
	mock := newMockRelay()

	// Bind a free loopback port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "wire-relay-dump: listen: %v\n", err)
		os.Exit(1)
	}
	addr := ln.Addr().String()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/relay/ws", mock.handle)
	mock.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = mock.srv.Serve(ln) }()

	if err := os.Setenv("SIGNALWIRE_RELAY_HOST", addr); err != nil {
		fmt.Fprintf(os.Stderr, "wire-relay-dump: setenv host: %v\n", err)
		os.Exit(1)
	}
	if err := os.Setenv("SIGNALWIRE_RELAY_SCHEME", "ws"); err != nil {
		fmt.Fprintf(os.Stderr, "wire-relay-dump: setenv scheme: %v\n", err)
		os.Exit(1)
	}

	if err := run(mock); err != nil {
		fmt.Fprintf(os.Stderr, "wire-relay-dump: %v\n", err)
		_ = mock.srv.Close()
		os.Exit(1)
	}
	_ = mock.srv.Close()
}

// run performs the capture and encodes the result — split out so main can hold
// no defers past the os.Exit paths (gocritic exitAfterDefer).
func run(mock *mockRelay) error {
	out := map[string]any{}

	// ---- event decoders (pure — no wire) ----
	decodeEvents(out)

	// ---- verb + client frames (over the mock wire) ----
	if err := captureFrames(mock, out); err != nil {
		return fmt.Errorf("capture: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	return nil
}

// frame wraps a captured params map as {method, params}.
func frame(method string, params map[string]any) map[string]any {
	return map[string]any{"method": method, "params": params}
}

func captureFrames(mock *mockRelay, out map[string]any) error {
	client := relay.NewRelayClient(
		relay.WithProject("proj-1"),
		relay.WithToken("tok-1"),
		relay.WithSpace("mock"),
	)

	// A Call to drive the verb cases arrives via OnCall after we push an
	// inbound receive event.
	callCh := make(chan *relay.Call, 1)
	client.OnCall(func(c *relay.Call) { callCh <- c })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runErr := make(chan error, 1)
	go func() { runErr <- client.RunContext(ctx) }()

	// Wait for the client to connect, then push the inbound call.
	if !waitConn(mock) {
		return fmt.Errorf("client did not connect")
	}
	conn := mock.conns[0]
	mock.pushInboundCall(conn)

	var c *relay.Call
	select {
	case c = <-callCh:
	case <-time.After(3 * time.Second):
		return fmt.Errorf("inbound call not delivered to OnCall")
	}

	// Small helper: run a verb, wait for its frame to land, capture it.
	settle := func() { time.Sleep(60 * time.Millisecond) }

	// relay_play
	c.Play([]map[string]any{{"type": "audio", "params": map[string]any{"url": "https://x/a.mp3"}}},
		relay.WithPlayVolume(5.0), relay.WithPlayControlID(cid))
	settle()
	out["relay_play"] = frame("calling.play", mock.lastFrame("calling.play"))

	// relay_play_tts
	c.PlayTTS("Hello world", relay.WithTTSVoice("en-US-Neural"))
	settle()
	out["relay_play_tts"] = frame("calling.play", mock.lastFrame("calling.play"))

	// relay_record
	c.Record(relay.WithRecordAudio(map[string]any{"format": "mp3", "beep": true}),
		relay.WithRecordControlID(cid))
	settle()
	out["relay_record"] = frame("calling.record", mock.lastFrame("calling.record"))

	// relay_connect
	_ = c.Connect(
		[][]map[string]any{{{"type": "phone", "params": map[string]any{"to_number": "+15551112222"}}}},
		relay.WithConnectRingback([]map[string]any{{"type": "ringtone", "params": map[string]any{"name": "us"}}}),
		relay.WithConnectTag("leg-1"),
		relay.WithConnectMaxDuration(3600),
	)
	settle()
	out["relay_connect"] = frame("calling.connect", mock.lastFrame("calling.connect"))

	// relay_collect
	pr := true
	it := 5.0
	c.Collect(&relay.CollectParams{
		Digits:         map[string]any{"max": 4, "terminators": "#"},
		Speech:         map[string]any{"language": "en-US"},
		InitialTimeout: &it,
		PartialResults: &pr,
		ControlID:      cid,
	})
	settle()
	out["relay_collect"] = frame("calling.collect", mock.lastFrame("calling.collect"))

	// relay_prompt (play_and_collect)
	c.PromptTTS("Enter your PIN", map[string]any{"digits": map[string]any{"max": 4}},
		relay.WithTTSVoice("en-US-Neural"))
	settle()
	out["relay_prompt"] = frame("calling.play_and_collect", mock.lastFrame("calling.play_and_collect"))

	// relay_detect
	dt := 30.0
	c.Detect(map[string]any{"type": "machine", "params": map[string]any{"initial_timeout": 4.0}}, &dt, cid)
	settle()
	out["relay_detect"] = frame("calling.detect", mock.lastFrame("calling.detect"))

	// relay_detect_amd
	c.DetectAnsweringMachine(
		relay.WithAMDInitialTimeout(4.0),
		relay.WithAMDMachineWordsThreshold(6),
		relay.WithAMDTimeout(30.0),
	)
	settle()
	out["relay_detect_amd"] = frame("calling.detect", mock.lastFrame("calling.detect"))

	// relay_tap
	c.Tap(
		map[string]any{"type": "audio", "params": map[string]any{"direction": "both"}},
		map[string]any{"type": "ws", "params": map[string]any{"uri": "wss://x/tap"}},
		cid,
	)
	settle()
	out["relay_tap"] = frame("calling.tap", mock.lastFrame("calling.tap"))

	// relay_send_fax
	c.SendFax("https://x/doc.pdf", "+15550001111",
		relay.WithFaxHeaderInfo("Hdr"), relay.WithFaxControlID(cid))
	settle()
	out["relay_send_fax"] = frame("calling.send_fax", mock.lastFrame("calling.send_fax"))

	// ---- control-ops (Action methods) ----
	// relay_play_stop
	pa := c.Play([]map[string]any{{"type": "audio", "params": map[string]any{"url": "https://x/a.mp3"}}},
		relay.WithPlayControlID(cid))
	settle()
	_ = pa.Stop()
	settle()
	out["relay_play_stop"] = frame("calling.play.stop", mock.lastFrame("calling.play.stop"))

	// relay_play_pause
	pa2 := c.Play([]map[string]any{{"type": "audio", "params": map[string]any{"url": "https://x/a.mp3"}}},
		relay.WithPlayControlID(cid))
	settle()
	_ = pa2.Pause("silence")
	settle()
	out["relay_play_pause"] = frame("calling.play.pause", mock.lastFrame("calling.play.pause"))

	// relay_record_resume
	ra := c.Record(relay.WithRecordAudio(map[string]any{"format": "mp3"}), relay.WithRecordControlID(cid))
	settle()
	_ = ra.Resume()
	settle()
	out["relay_record_resume"] = frame("calling.record.resume", mock.lastFrame("calling.record.resume"))

	// relay_play_volume
	pa3 := c.Play([]map[string]any{{"type": "audio", "params": map[string]any{"url": "https://x/a.mp3"}}},
		relay.WithPlayControlID(cid))
	settle()
	_ = pa3.Volume(3.5)
	settle()
	out["relay_play_volume"] = frame("calling.play.volume", mock.lastFrame("calling.play.volume"))

	// ---- RelayClient-level frames ----
	// relay_client_execute
	_, _ = client.Execute("calling.answer", map[string]any{"node_id": node, "call_id": call})
	settle()
	out["relay_client_execute"] = frame("calling.answer", mock.lastFrame("calling.answer"))

	// relay_send_message
	_, _ = client.SendMessage("+15551112222", "+15553334444", "hi",
		relay.WithMessageTags([]string{"t1"}))
	settle()
	out["relay_send_message"] = frame("messaging.send", mock.lastFrame("messaging.send"))

	// relay_dial (arm the dial-answer push for tag "dial-1")
	mock.mu.Lock()
	mock.dialArm = "dial-1"
	mock.mu.Unlock()
	_, _ = client.Dial(
		[][]map[string]any{{{"type": "phone", "params": map[string]any{"to_number": "+15551112222"}}}},
		relay.WithDialTag("dial-1"), relay.WithDialMaxDuration(600),
	)
	settle()
	out["relay_dial"] = frame("calling.dial", mock.lastFrame("calling.dial"))

	client.Stop()
	cancel()
	select {
	case <-runErr:
	case <-time.After(2 * time.Second):
	}
	return nil
}

func waitConn(mock *mockRelay) bool {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mock.mu.Lock()
		n := len(mock.conns)
		mock.mu.Unlock()
		if n > 0 {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// decodeEvents runs the pure typed-event decoders.
func decodeEvents(out map[string]any) {
	// relay_evt_queue
	q := relay.NewQueueEvent(map[string]any{
		"call_id": call, "control_id": cid, "status": "waiting",
		"id": "q-42", "name": "support", "position": 3, "size": 10,
	})
	out["relay_evt_queue"] = map[string]any{
		"control_id": q.ControlID,
		"status":     q.Status,
		"queue_id":   q.QueueID,
		"queue_name": q.QueueName,
		"position":   q.Position,
		"size":       q.Size,
	}

	// relay_evt_record
	rec := relay.NewRecordEvent(map[string]any{
		"call_id": call, "control_id": cid, "state": "finished",
		// Numbers arrive from the wire as JSON float64; mirror that so the
		// decoder's float64 type-assertions (rec["size"].(float64)) succeed.
		"record": map[string]any{"url": "https://x/rec.mp3", "duration": 12.5, "size": float64(4096)},
	})
	out["relay_evt_record"] = map[string]any{
		"control_id": rec.ControlID,
		"state":      rec.State,
		"url":        rec.URL,
		"duration":   rec.Duration,
		"size":       rec.Size,
	}

	// relay_evt_state_dispatch (parse_event -> CallStateEvent)
	obj := relay.ParseEvent(map[string]any{
		"event_type": "calling.call.state",
		"params": map[string]any{
			"call_id": call, "call_state": "answered", "direction": "inbound", "end_reason": "",
		},
	})
	stateOut := map[string]any{"_class": className(obj)}
	if cse, ok := obj.(*relay.CallStateEvent); ok {
		stateOut["call_id"] = cse.CallID
		stateOut["call_state"] = cse.CallState
		stateOut["direction"] = cse.Direction
	}
	out["relay_evt_state_dispatch"] = stateOut

	// relay_evt_collect
	col := relay.NewCollectEvent(map[string]any{
		"call_id": call, "control_id": cid, "state": "finished",
		"result": map[string]any{"type": "digit", "params": map[string]any{"digits": "1234"}},
		"final":  true,
	})
	var final any
	if col.Final != nil {
		final = *col.Final
	}
	out["relay_evt_collect"] = map[string]any{
		"control_id": col.ControlID,
		"state":      col.State,
		"result":     col.Result,
		"final":      final,
	}
}

// className returns the bare type name (no pointer/package) of a decoded event,
// mirroring Python's type(obj).__name__.
func className(v any) string {
	t := reflect.TypeOf(v)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil {
		return ""
	}
	return t.Name()
}
