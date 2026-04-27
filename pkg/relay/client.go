package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	protocol       string
	authState      string
	contexts       []string
	maxActiveCalls int

	onCall    func(*Call)
	onMessage func(*Message)

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

// Run connects to SignalWire, authenticates, subscribes to configured
// contexts, and starts the read loop. It blocks until Stop is called
// or the context is cancelled.
func (c *Client) Run() error {
	if err := c.connect(); err != nil {
		return fmt.Errorf("relay connect: %w", err)
	}

	if err := c.authenticate(); err != nil {
		return fmt.Errorf("relay authenticate: %w", err)
	}

	if err := c.subscribeContexts(); err != nil {
		return fmt.Errorf("relay subscribe contexts: %w", err)
	}

	c.running.Store(true)
	c.readLoop()
	return nil
}

// Stop gracefully shuts down the client connection.
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
// the Blade calling.begin devices field).
func (c *Client) Dial(devices [][]map[string]any, opts ...DialOption) (*Call, error) {
	tag := uuid.New().String()
	params := map[string]any{
		"tag":     tag,
		"devices": devices,
	}
	for _, opt := range opts {
		opt(params)
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

	_, err := c.execute("calling.begin", params)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	select {
	case call := <-ch:
		return call, nil
	case <-c.ctx.Done():
		return nil, c.ctx.Err()
	case <-time.After(60 * time.Second):
		return nil, fmt.Errorf("dial: timeout waiting for call creation")
	}
}

// SendMessage sends an SMS/MMS message and returns a Message that can be
// used to track delivery.
func (c *Client) SendMessage(to, from, body string, opts ...MessageOption) (*Message, error) {
	params := map[string]any{
		"to_number":   to,
		"from_number": from,
		"body":        body,
	}
	for _, opt := range opts {
		opt(params)
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

	c.mu.Lock()
	c.messages[result.MessageID] = msg
	c.mu.Unlock()

	return msg, nil
}

// ---------------------------------------------------------------------------
// Internal: connection management
// ---------------------------------------------------------------------------

// connect establishes the WebSocket connection to SignalWire.
func (c *Client) connect() error {
	host := c.space
	if !strings.Contains(host, ".") {
		host = host + ".signalwire.com"
	}
	url := fmt.Sprintf("wss://%s/api/relay/ws", host)

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
	}

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
func (c *Client) handleEvent(eventType string, params map[string]any) {
	switch {
	case strings.HasPrefix(eventType, "calling."):
		c.handleCallingEvent(eventType, params)
	case strings.HasPrefix(eventType, "messaging."):
		c.handleMessagingEvent(eventType, params)
	default:
		c.logger.Printf("relay unhandled event: %s", eventType)
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
		c.mu.Lock()
		c.calls[callID] = call
		handler := c.onCall
		c.mu.Unlock()

		call.dispatchEvent(event)

		if handler != nil {
			go handler(call)
		}
		return
	}

	// Handle dial state events - resolve pending dials.
	if eventType == EventCallingCallDial || (eventType == EventCallingCallState && tag != "") {
		c.mu.RLock()
		ch, hasPending := c.pendingDials[tag]
		_, hasCall := c.calls[callID]
		c.mu.RUnlock()

		if hasPending && !hasCall && callID != "" {
			nodeID, _ := params["node_id"].(string)
			call := newCall(c, callID, nodeID, tag)
			c.mu.Lock()
			c.calls[callID] = call
			c.mu.Unlock()

			select {
			case ch <- call:
			default:
			}

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
