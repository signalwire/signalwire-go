package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Client is the main RELAY WebSocket client that manages a persistent
// connection to SignalWire, handles Blade/JSON-RPC 2.0 authentication,
// event dispatch, and exposes high-level Dial/SendMessage methods.
type Client struct {
	projectID      string
	token          string
	jwtToken       string
	space          string
	conn           *websocket.Conn
	mu             sync.RWMutex
	pending        map[string]chan json.RawMessage // JSON-RPC id -> response channel
	calls          map[string]*Call                // call_id -> Call
	messages       map[string]*Message             // message_id -> Message
	pendingDials   map[string]chan *Call            // tag -> dial result
	protocol           string
	authState          string
	authorizationState string // signalwire.authorization.state event payload
	contexts           []string
	maxActiveCalls     int

	onCall    func(*Call)
	onMessage func(*Message)
	onEvent   func(eventType string, params map[string]any)

	logger  *log.Logger
	running atomic.Bool

	// Shutdown management
	ctx    context.Context
	cancel context.CancelFunc

	// Reconnect
	reconnectBackoff time.Duration
	maxBackoff       time.Duration
}

// NewRelayClient creates a new RELAY Client with the given options.
//
// After explicit options are applied, any remaining unset auth/space
// fields fall back to SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN,
// SIGNALWIRE_JWT_TOKEN, SIGNALWIRE_SPACE, and RELAY_MAX_ACTIVE_CALLS
// environment variables — matching Python RelayClient.__init__'s
// automatic env-var fallback (relay/client.py:115-119). Explicit
// options always win.
func NewRelayClient(opts ...ClientOption) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		pending:          make(map[string]chan json.RawMessage),
		calls:            make(map[string]*Call),
		messages:         make(map[string]*Message),
		pendingDials:     make(map[string]chan *Call),
		logger:           log.Default(),
		ctx:              ctx,
		cancel:           cancel,
		reconnectBackoff: 1 * time.Second,
		maxBackoff:       30 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.applyEnvDefaults()
	return c
}

// OnCall registers a handler invoked for each inbound call.
func (c *Client) OnCall(handler func(*Call)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onCall = handler
}

// OnMessage registers a handler invoked for each inbound message.
func (c *Client) OnMessage(handler func(*Message)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onMessage = handler
}

// OnEvent registers a handler invoked for every inbound `signalwire.event`
// frame, AFTER type-specific routing (call, messaging) has run. The
// handler receives the raw event_type string and params map. This is
// the lowest-level event hook — most callers should use OnCall or
// OnMessage instead. Mirrors Python RelayClient's public event-tap
// surface used by integration tests.
func (c *Client) OnEvent(handler func(eventType string, params map[string]any)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onEvent = handler
}

// RelayProtocol returns the server-assigned protocol string received during
// authentication. Mirrors Python's relay_protocol property. The value is empty
// until after a successful Connect/Run.
func (c *Client) RelayProtocol() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.protocol
}

// ProjectID returns the configured project ID. Mirrors Python's public
// client.project attribute, allowing callers to read back the value
// supplied via WithProject(...) or the SIGNALWIRE_PROJECT_ID env var.
func (c *Client) ProjectID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.projectID
}

// Token returns the configured API token. Mirrors Python's public
// client.token attribute, allowing callers to read back the value
// supplied via WithToken(...) or the SIGNALWIRE_API_TOKEN env var.
func (c *Client) Token() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// JWTToken returns the configured JWT. Mirrors Python's public
// client.jwt_token attribute, allowing callers to read back the value
// supplied via WithJWT(...).
func (c *Client) JWTToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.jwtToken
}

// Space returns the configured SignalWire space hostname. Mirrors Python's
// public client.host attribute (Python uses the term "host"; Go uses
// "space" because that's the more accurate noun — see WithSpace).
func (c *Client) Space() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.space
}

// Contexts returns a copy of the configured RELAY contexts. Mirrors
// Python's public client.contexts attribute. The returned slice is a
// copy — mutating it does not affect the client.
func (c *Client) Contexts() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]string(nil), c.contexts...)
}

// AuthorizationState returns the most recent encrypted authorization
// state blob received via signalwire.authorization.state events.
// Mirrors Python's RelayClient._authorization_state used during
// reconnection (relay/client.py:174). Empty until the server pushes
// such an event.
func (c *Client) AuthorizationState() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.authorizationState
}

// Connect establishes the WebSocket connection to SignalWire. This is the
// public equivalent of the internal connect() method, mirroring Python's
// async connect() which is also used in the async-with context manager.
// In most cases callers should use Run() which calls Connect internally and
// then drives the read loop.
func (c *Client) Connect() error {
	return c.connect()
}

// Execute sends a JSON-RPC request over the WebSocket and waits for the
// response. Mirrors Python's async execute(method, params) which is the
// public arbitrary-RPC surface used by callers that need low-level access.
func (c *Client) Execute(method string, params map[string]any) (json.RawMessage, error) {
	return c.execute(method, params)
}

// Notify sends a JSON-RPC notification (no `id`, no response expected)
// with the given method and params. Used for fire-and-forget frames
// such as the client-side `signalwire.event` ACK pattern that some
// integration fixtures expect. Returns any write error from the
// underlying socket.
func (c *Client) Notify(method string, params map[string]any) error {
	frame := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	return c.writeJSON(frame)
}

// Receive subscribes to additional contexts for inbound events after the
// client is already connected. Sends signalwire.receive on the assigned
// protocol. Mirrors Python's async receive(contexts).
func (c *Client) Receive(contexts ...string) error {
	if len(contexts) == 0 {
		return nil
	}
	_, err := c.execute("signalwire.receive", map[string]any{
		"contexts": contexts,
	})
	return err
}

// Unreceive unsubscribes from contexts for inbound events. Sends
// signalwire.unreceive on the assigned protocol. Mirrors Python's async
// unreceive(contexts).
func (c *Client) Unreceive(contexts ...string) error {
	if len(contexts) == 0 {
		return nil
	}
	_, err := c.execute("signalwire.unreceive", map[string]any{
		"contexts": contexts,
	})
	return err
}

// Run connects to SignalWire, authenticates, subscribes to configured
// contexts, and starts the read loop. It blocks until Stop is called
// or the context is cancelled.
//
// Important: the read loop must be running before subscribeContexts()
// is called. subscribeContexts() executes a JSON-RPC request whose
// response is delivered through the read loop's pending-id channel
// machinery. If the read loop isn't running, the JSON-RPC reply has no
// reader and the request times out (30s). Hence we start readLoop in a
// goroutine BEFORE the subscribe call, then block on a done channel
// here.
func (c *Client) Run() error {
	if err := c.connect(); err != nil {
		return fmt.Errorf("relay connect: %w", err)
	}

	if err := c.authenticate(); err != nil {
		return fmt.Errorf("relay authenticate: %w", err)
	}

	c.running.Store(true)
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.readLoop()
	}()

	if err := c.subscribeContexts(); err != nil {
		c.running.Store(false)
		c.cancel()
		<-done
		return fmt.Errorf("relay subscribe contexts: %w", err)
	}

	<-done
	return nil
}

// Authenticate runs the signalwire.connect handshake and stores the
// server-issued protocol string. Mirrors Python's RelayClient.connect()
// auth phase. Use Connect first to establish the WebSocket; this call
// reads the auth response synchronously (the read loop has not yet
// started so no other reader is contending for the socket).
func (c *Client) Authenticate() error {
	return c.authenticate()
}

// StartReadLoop spawns the read goroutine and marks the client running.
// Mirrors the goroutine-spawn portion of Run() — call it after
// Authenticate() and before any Execute() call so JSON-RPC responses
// have a reader. Pair with Stop() to terminate.
func (c *Client) StartReadLoop() {
	c.running.Store(true)
	go c.readLoop()
}

// SubscribeContexts subscribes to whatever contexts were configured via
// WithContexts. No-op when the contexts slice is empty. Used by the
// mock-relay test helper which drives connect/auth/read-loop manually.
func (c *Client) SubscribeContexts() error {
	return c.subscribeContexts()
}

// Stop gracefully shuts down the client connection.
//
// Equivalent to Python's RelayClient.disconnect() (relay/client.py:286).
// Python users porting code can search for "disconnect" and find this
// method by its rename.
func (c *Client) Stop() {
	c.running.Store(false)
	c.cancel()
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn != nil {
		_ = conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		_ = conn.Close()
	}
}

// Dial initiates an outbound call to the given device list. The devices
// parameter is a list of serial/parallel device groups (same structure as
// the Blade calling.dial devices field).
//
// Mirrors Python's RelayClient.dial(devices, *, tag=None, max_duration=None,
// dial_timeout=None). The calling.dial RPC response only contains
// {"code": "200", "message": "Dialing"} — no call_id. The real call_id and
// node_id arrive via subsequent calling.call.dial events keyed by tag.
// This method waits for that event so the returned Call always has valid
// identifiers.
//
// To pass a caller-supplied tag, use WithDialTag. Without it the SDK
// generates a UUID, matching Python's tag = tag or str(uuid.uuid4()).
func (c *Client) Dial(devices [][]map[string]any, opts ...DialOption) (*Call, error) {
	params := map[string]any{
		"devices": devices,
	}
	for _, opt := range opts {
		opt(params)
	}

	// If no caller-supplied tag, mint one. Mirrors Python's
	//   dial_tag = tag or str(uuid.uuid4())
	tag, _ := params["tag"].(string)
	if tag == "" {
		tag = uuid.New().String()
		params["tag"] = tag
	}

	ch := make(chan *Call, 1)
	c.mu.Lock()
	c.pendingDials[tag] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pendingDials, tag)
		c.mu.Unlock()
	}()

	dialTimeout := 120 * time.Second
	if v, ok := params["_dial_timeout"]; ok {
		if d, ok := v.(time.Duration); ok && d > 0 {
			dialTimeout = d
		}
		delete(params, "_dial_timeout")
	}

	_, err := c.execute("calling.dial", params)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	select {
	case call := <-ch:
		if call == nil {
			return nil, NewRelayError(-1, fmt.Sprintf("Dial failed (tag=%s)", tag))
		}
		return call, nil
	case <-c.ctx.Done():
		return nil, c.ctx.Err()
	case <-time.After(dialTimeout):
		return nil, NewRelayError(-1, fmt.Sprintf("Dial timed out waiting for answer (tag=%s)", tag))
	}
}

// SendMessage sends an SMS/MMS message and returns a Message that can be
// used to track delivery.
//
// The context option (WithMessageContext) sets the routing context for the
// message; it defaults to the relay protocol when omitted, matching Python SDK
// behaviour. The on_completed option (WithMessageOnCompleted) registers a
// callback fired when the message reaches a terminal state.
func (c *Client) SendMessage(to, from, body string, opts ...MessageOption) (*Message, error) {
	params := map[string]any{
		"to_number":   to,
		"from_number": from,
		"body":        body,
	}
	for _, opt := range opts {
		opt(params)
	}

	// Extract internal-only options that must not be sent over the wire.
	var onCompleted func(*Message, *RelayEvent)
	if v, ok := params["_on_completed"]; ok {
		onCompleted, _ = v.(func(*Message, *RelayEvent))
		delete(params, "_on_completed")
	}

	// Apply default context (relay protocol) when not explicitly set.
	if _, hasCtx := params["context"]; !hasCtx {
		c.mu.RLock()
		proto := c.protocol
		c.mu.RUnlock()
		if proto != "" {
			params["context"] = proto
		}
	}

	resp, err := c.execute("messaging.send", params)
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	var result struct {
		MessageID string `json:"message_id"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("send message parse response: %w", err)
	}

	msg := newMessage(result.MessageID, DirectionOutbound, from, to, body)
	// Mirrors Python: outbound messages start in "queued" state
	// (relay/client.py:468, Message(... state="queued")).
	msg.state = MessageStateQueued
	if onCompleted != nil {
		msg.onCompleted = onCompleted
	}

	c.mu.Lock()
	c.messages[result.MessageID] = msg
	c.mu.Unlock()

	return msg, nil
}

// ---------------------------------------------------------------------------
// Internal: connection management
// ---------------------------------------------------------------------------

// connect establishes the WebSocket connection to SignalWire.
//
// The URL is built as `<scheme>://<host>/api/relay/ws`. By default the
// scheme is wss and the host is derived from the configured space
// (with a `.signalwire.com` suffix appended when the space is a bare
// subdomain). For testing — including the audit_relay_handshake.py
// fixture — both can be overridden via env vars:
//
//   - SIGNALWIRE_RELAY_HOST   (e.g. "127.0.0.1:5050")
//   - SIGNALWIRE_RELAY_SCHEME (e.g. "ws" for plain WebSocket)
//
// When the host env var is set it takes precedence over the configured
// space; when scheme is set it overrides "wss". This mirrors the
// per-port harness contract documented in the porting-sdk
// SUBAGENT_PLAYBOOK.
func (c *Client) connect() error {
	host := os.Getenv("SIGNALWIRE_RELAY_HOST")
	if host == "" {
		host = c.space
		if !strings.Contains(host, ".") {
			host = host + ".signalwire.com"
		}
	}
	scheme := os.Getenv("SIGNALWIRE_RELAY_SCHEME")
	if scheme == "" {
		scheme = "wss"
	}
	url := fmt.Sprintf("%s://%s/api/relay/ws", scheme, host)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	header := http.Header{}
	conn, _, err := dialer.DialContext(c.ctx, url, header)
	if err != nil {
		return fmt.Errorf("websocket dial %s: %w", url, err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	return nil
}

// authenticate sends signalwire.connect with credentials and reads the
// authentication response containing the JWT and protocol.
func (c *Client) authenticate() error {
	id := uuid.New().String()

	connectParams := map[string]any{
		"version": map[string]any{
			"major":    ProtocolVersionMajor,
			"minor":    ProtocolVersionMinor,
			"revision": ProtocolVersionRevision,
		},
		// Mirrors Python connect_params at relay/client.py:265-267:
		// {"version": ..., "agent": AGENT_STRING, "event_acks": True}.
		// event_acks=true tells the server to expect JSON-RPC ACKs for
		// each pushed signalwire.event. If we omit this, the server may
		// fall back to fire-and-forget event delivery.
		"agent":      AgentString,
		"event_acks": true,
	}

	c.mu.RLock()
	if c.contexts != nil && len(c.contexts) > 0 {
		// Python sends contexts on the connect frame itself (the same
		// `subscriptions` set the connect result echoes back). Mirrors
		// relay/client.py where the contexts list flows into connect.
		connectParams["contexts"] = append([]string(nil), c.contexts...)
	}
	if c.protocol != "" {
		// Reconnect-with-protocol path: the protocol string the server
		// previously issued goes on the wire so the server can resume
		// the session. Mirrors Python's _relay_protocol-on-reconnect.
		connectParams["protocol"] = c.protocol
	}
	if c.authorizationState != "" {
		connectParams["authorization_state"] = c.authorizationState
	}
	c.mu.RUnlock()

	if c.jwtToken != "" {
		connectParams["authentication"] = map[string]any{
			"jwt_token": c.jwtToken,
		}
	} else {
		connectParams["authentication"] = map[string]any{
			"project": c.projectID,
			"token":   c.token,
		}
	}

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  MethodSignalWireConnect,
		"params":  connectParams,
	}

	ch := make(chan json.RawMessage, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	if err := c.writeJSON(req); err != nil {
		return fmt.Errorf("write connect: %w", err)
	}

	// Read the authentication response directly (we are not in the
	// read loop yet).
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read auth response: %w", err)
		}

		var msg struct {
			ID     string          `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		if msg.ID == id {
			c.mu.Lock()
			delete(c.pending, id)
			c.mu.Unlock()

			if msg.Error != nil {
				return NewRelayError(msg.Error.Code, msg.Error.Message)
			}

			var authResult struct {
				Protocol     string `json:"protocol"`
				JWTToken     string `json:"jwt_token"`
				Authorization struct {
					Project string `json:"project"`
				} `json:"authorization"`
			}
			if err := json.Unmarshal(msg.Result, &authResult); err != nil {
				return fmt.Errorf("parse auth result: %w", err)
			}
			c.mu.Lock()
			c.protocol = authResult.Protocol
			c.jwtToken = authResult.JWTToken
			c.authState = "authenticated"
			c.mu.Unlock()
			return nil
		}
	}
}

// subscribeContexts subscribes to configured contexts for inbound events.
func (c *Client) subscribeContexts() error {
	if len(c.contexts) == 0 {
		return nil
	}

	_, err := c.execute("signalwire.receive", map[string]any{
		"contexts": c.contexts,
	})
	return err
}

// readLoop is the main message processing goroutine. It reads JSON-RPC
// messages from the WebSocket and dispatches them to the appropriate
// handlers.
func (c *Client) readLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()
		if conn == nil {
			return
		}

		_, raw, err := conn.ReadMessage()
		if err != nil {
			if !c.running.Load() {
				return
			}
			c.logger.Printf("relay read error: %v", err)
			c.reconnect()
			return
		}

		var msg struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      string          `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
			Result  json.RawMessage `json:"result"`
			Error   *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			c.logger.Printf("relay unmarshal error: %v", err)
			continue
		}

		// JSON-RPC response to one of our requests.
		if msg.Result != nil || msg.Error != nil {
			c.mu.RLock()
			ch, ok := c.pending[msg.ID]
			c.mu.RUnlock()
			if ok {
				if msg.Error != nil {
					errJSON, _ := json.Marshal(msg.Error)
					ch <- errJSON
				} else {
					ch <- msg.Result
				}
				c.mu.Lock()
				delete(c.pending, msg.ID)
				c.mu.Unlock()
			}
			continue
		}

		// Server-initiated request/notification.
		switch msg.Method {
		case MethodSignalWirePing:
			c.handlePing(msg.ID)

		case MethodSignalWireEvent:
			// ACK the event immediately.
			c.ackEvent(msg.ID)

			var eventParams struct {
				EventType string         `json:"event_type"`
				Params    map[string]any `json:"params"`
			}
			if err := json.Unmarshal(msg.Params, &eventParams); err != nil {
				c.logger.Printf("relay event unmarshal error: %v", err)
				continue
			}
			c.handleEvent(eventParams.EventType, eventParams.Params)

		default:
			c.logger.Printf("relay unknown method: %s", msg.Method)
		}
	}
}

// execute sends a JSON-RPC request and waits for the response.
func (c *Client) execute(method string, params map[string]any) (json.RawMessage, error) {
	id := uuid.New().String()

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}

	// Add protocol to params if we have one.
	c.mu.RLock()
	if c.protocol != "" {
		params["protocol"] = c.protocol
	}
	c.mu.RUnlock()

	ch := make(chan json.RawMessage, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	if err := c.writeJSON(req); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		// Check if the response is an error.
		var errResp struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(resp, &errResp); err == nil && errResp.Code != 0 {
			return nil, NewRelayError(errResp.Code, errResp.Message)
		}
		return resp, nil
	case <-c.ctx.Done():
		return nil, c.ctx.Err()
	case <-time.After(30 * time.Second):
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("execute %s: timeout", method)
	}
}

// handleEvent routes events to the appropriate call, message, or action.
// After type-specific routing it also invokes the generic OnEvent hook
// (if registered) so callers can observe every event regardless of
// whether it matched a known correlation.
func (c *Client) handleEvent(eventType string, params map[string]any) {
	switch {
	case eventType == EventAuthorizationState:
		// Mirrors Python relay/client.py:796. Encrypted blob the
		// server hands back so a reconnect can resume seamlessly.
		if as, _ := params["authorization_state"].(string); as != "" {
			c.mu.Lock()
			c.authorizationState = as
			c.mu.Unlock()
		}
	case strings.HasPrefix(eventType, "calling."):
		c.handleCallingEvent(eventType, params)
	case strings.HasPrefix(eventType, "messaging."):
		c.handleMessagingEvent(eventType, params)
	default:
		c.logger.Printf("relay unhandled event: %s", eventType)
	}

	// Fire the generic event hook last so it sees the same dispatch
	// the call/message hooks did. The lock is taken in a tight scope
	// so the user's callback can call back into Client (e.g. Execute)
	// without deadlocking.
	c.mu.RLock()
	hook := c.onEvent
	c.mu.RUnlock()
	if hook != nil {
		hook(eventType, params)
	}
}

// handleCallingEvent processes calling.* events.
func (c *Client) handleCallingEvent(eventType string, params map[string]any) {
	event := NewRelayEvent(eventType, params)

	// Extract correlation identifiers.
	callID, _ := params["call_id"].(string)
	controlID, _ := params["control_id"].(string)
	tag, _ := params["tag"].(string)

	// Handle call receive (inbound) events - create new call.
	if eventType == EventCallingCallReceive {
		nodeID, _ := params["node_id"].(string)
		if tag == "" {
			tag = uuid.New().String()
		}
		call := newCall(c, callID, nodeID, tag)
		// Populate direction from event so .Direction() reads correctly
		// even before any subsequent state event.
		if dir, _ := params["direction"].(string); dir != "" {
			call.direction = dir
		}
		if dev, _ := params["device"].(map[string]any); dev != nil {
			call.device = dev
		}
		if ctxStr, _ := params["context"].(string); ctxStr != "" {
			call.context = ctxStr
		}
		c.mu.Lock()
		c.calls[callID] = call
		handler := c.onCall
		c.mu.Unlock()

		call.dispatchEvent(event)

		if handler != nil {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						c.logger.Printf("relay: on_call handler panicked: %v", r)
					}
				}()
				handler(call)
			}()
		}
		return
	}

	// Handle calling.call.dial events — these carry the winner via
	// nested params.call.{call_id,node_id}, NOT a top-level call_id.
	// Mirrors Python's _handle_dial_event at relay/client.py:945.
	if eventType == EventCallingCallDial {
		dialState, _ := params["dial_state"].(string)
		callInfo, _ := params["call"].(map[string]any)
		var winnerCallID, winnerNodeID string
		var winnerDevice map[string]any
		if callInfo != nil {
			winnerCallID, _ = callInfo["call_id"].(string)
			winnerNodeID, _ = callInfo["node_id"].(string)
			winnerDevice, _ = callInfo["device"].(map[string]any)
		}

		c.mu.RLock()
		ch, hasPending := c.pendingDials[tag]
		c.mu.RUnlock()

		if !hasPending {
			// Stale event — still dispatch to any matching call.
			if winnerCallID != "" {
				c.mu.RLock()
				call, ok := c.calls[winnerCallID]
				c.mu.RUnlock()
				if ok {
					call.dispatchEvent(event)
				}
			}
			return
		}

		switch dialState {
		case "answered":
			c.mu.Lock()
			call, exists := c.calls[winnerCallID]
			if !exists {
				call = newCall(c, winnerCallID, winnerNodeID, tag)
				call.direction = DirectionOutbound
				call.state = CallStateAnswered
				if winnerDevice != nil {
					call.device = winnerDevice
				}
				c.calls[winnerCallID] = call
			} else if winnerNodeID != "" && call.nodeID == "" {
				call.nodeID = winnerNodeID
			}
			c.mu.Unlock()

			select {
			case ch <- call:
			default:
			}
			call.dispatchEvent(event)

		case "failed":
			// Signal failure by closing the channel without a value
			// after dispatching the event. Dial() detects this via the
			// channel-recv idiom: a closed channel returns the zero
			// value of *Call, which is nil. We instead resolve the
			// future via a sentinel — easier to send a nil-bearing
			// Call wrapper. Since `chan *Call` can carry nil, push
			// nil to indicate failure.
			select {
			case ch <- nil:
			default:
			}
		}
		return
	}

	// Handle calling.call.state events with a matching pending dial tag.
	// Some servers emit state events for the winner before the dial
	// event itself; in that case the call_id is meaningful and we
	// pre-create the Call so it's already in c.calls when the dial
	// event lands.
	if eventType == EventCallingCallState && tag != "" {
		c.mu.RLock()
		_, hasPending := c.pendingDials[tag]
		_, hasCall := c.calls[callID]
		c.mu.RUnlock()

		if hasPending && !hasCall && callID != "" {
			nodeID, _ := params["node_id"].(string)
			call := newCall(c, callID, nodeID, tag)
			call.direction = DirectionOutbound
			c.mu.Lock()
			c.calls[callID] = call
			c.mu.Unlock()
			call.dispatchEvent(event)
			return
		}
	}

	// Route to existing call.
	c.mu.RLock()
	call, ok := c.calls[callID]
	c.mu.RUnlock()

	if !ok {
		return
	}

	// Resolve action by control_id if present.
	if controlID != "" {
		call.resolveAction(controlID, event)
	}

	call.dispatchEvent(event)

	// Clean up ended calls.
	if eventType == EventCallingCallState {
		state, _ := params["call_state"].(string)
		if state == CallStateEnded {
			c.mu.Lock()
			delete(c.calls, callID)
			c.mu.Unlock()
		}
	}
}

// handleMessagingEvent processes messaging.* events.
func (c *Client) handleMessagingEvent(eventType string, params map[string]any) {
	event := NewRelayEvent(eventType, params)

	if eventType == EventMessagingReceive {
		c.mu.RLock()
		handler := c.onMessage
		c.mu.RUnlock()

		if handler != nil {
			rcv := NewMessageReceiveEvent(params)
			msg := newMessage(rcv.MessageID, rcv.Direction, rcv.FromNumber, rcv.ToNumber, rcv.Body)
			msg.context = rcv.Context
			msg.media = rcv.Media
			msg.segments = rcv.Segments
			msg.tags = rcv.Tags
			msg.state = MessageStateReceived
			go handler(msg)
		}
		return
	}

	if eventType == EventMessagingState {
		messageID, _ := params["message_id"].(string)
		c.mu.RLock()
		msg, ok := c.messages[messageID]
		c.mu.RUnlock()
		if ok {
			msg.updateState(event)
		}
	}
}

// handlePing responds to signalwire.ping keep-alive messages.
func (c *Client) handlePing(id string) {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  map[string]any{"timestamp": time.Now().Unix()},
	}
	if err := c.writeJSON(resp); err != nil {
		c.logger.Printf("relay ping response error: %v", err)
	}
}

// ackEvent sends a JSON-RPC result ACK for a received event.
func (c *Client) ackEvent(id string) {
	if id == "" {
		return
	}
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  map[string]any{},
	}
	if err := c.writeJSON(resp); err != nil {
		c.logger.Printf("relay ack error: %v", err)
	}
}

// reconnect attempts to re-establish the WebSocket connection with
// exponential backoff.
func (c *Client) reconnect() {
	if !c.running.Load() {
		return
	}

	backoff := c.reconnectBackoff
	for c.running.Load() {
		c.logger.Printf("relay reconnecting in %v...", backoff)

		select {
		case <-time.After(backoff):
		case <-c.ctx.Done():
			return
		}

		if err := c.connect(); err != nil {
			c.logger.Printf("relay reconnect failed: %v", err)
			backoff *= 2
			if backoff > c.maxBackoff {
				backoff = c.maxBackoff
			}
			continue
		}

		if err := c.authenticate(); err != nil {
			c.logger.Printf("relay re-auth failed: %v", err)
			backoff *= 2
			if backoff > c.maxBackoff {
				backoff = c.maxBackoff
			}
			continue
		}

		if err := c.subscribeContexts(); err != nil {
			c.logger.Printf("relay re-subscribe failed: %v", err)
		}

		c.logger.Printf("relay reconnected")
		go c.readLoop()
		return
	}
}

// writeJSON safely writes a JSON message to the WebSocket connection.
func (c *Client) writeJSON(v any) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return fmt.Errorf("no websocket connection")
	}
	// Use a write lock to ensure only one goroutine writes at a time.
	// gorilla/websocket connections are not safe for concurrent writes.
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}
