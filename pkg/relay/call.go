package relay

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// Call represents an active voice call managed through the RELAY client.
// It exposes methods for every calling operation supported by SignalWire,
// grouped into logical categories: lifecycle, audio, recording, bridging,
// DTMF, detection, fax, tap, streaming, conferencing, AI, hold/denoise,
// room/queue, pay, and transcription.
type Call struct {
	callID  string
	nodeID  string
	tag     string
	client  *Client
	state   string
	mu      sync.Mutex
	actions map[string]*Action

	eventHandlers map[string][]func(*RelayEvent)
	waiters       []waiter
}

// waiter is an internal struct for WaitFor callers.
type waiter struct {
	eventType string
	predicate func(*RelayEvent) bool
	ch        chan *RelayEvent
}

// newCall creates a new Call tied to a RELAY client.
func newCall(client *Client, callID, nodeID, tag string) *Call {
	return &Call{
		callID:        callID,
		nodeID:        nodeID,
		tag:           tag,
		client:        client,
		state:         CallStateCreated,
		actions:       make(map[string]*Action),
		eventHandlers: make(map[string][]func(*RelayEvent)),
	}
}

// CallID returns the unique call identifier assigned by the server.
func (c *Call) CallID() string { return c.callID }

// NodeID returns the node handling this call.
func (c *Call) NodeID() string { return c.nodeID }

// Tag returns the client-generated correlation tag.
func (c *Call) Tag() string { return c.tag }

// State returns the current call state.
func (c *Call) State() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// ---------------------------------------------------------------------------
// Event handling
// ---------------------------------------------------------------------------

// On registers a handler for a specific event type on this call.
func (c *Call) On(eventType string, handler func(*RelayEvent)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.eventHandlers[eventType] = append(c.eventHandlers[eventType], handler)
}

// WaitFor blocks until an event matching the given type and predicate is
// received, or the context expires.
func (c *Call) WaitFor(ctx context.Context, eventType string, predicate func(*RelayEvent) bool) (*RelayEvent, error) {
	ch := make(chan *RelayEvent, 1)
	c.mu.Lock()
	c.waiters = append(c.waiters, waiter{
		eventType: eventType,
		predicate: predicate,
		ch:        ch,
	})
	c.mu.Unlock()

	select {
	case ev := <-ch:
		return ev, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// dispatchEvent is called internally to route an event through handlers
// and waiters.
func (c *Call) dispatchEvent(event *RelayEvent) {
	c.mu.Lock()

	// Update call state for state events.
	if event.EventType == EventCallingCallState {
		if s := event.GetString("call_state"); s != "" {
			c.state = s
		}
	}

	// Copy handlers for this event type.
	handlers := make([]func(*RelayEvent), len(c.eventHandlers[event.EventType]))
	copy(handlers, c.eventHandlers[event.EventType])

	// Check waiters.
	remaining := c.waiters[:0]
	for _, w := range c.waiters {
		if w.eventType == event.EventType && (w.predicate == nil || w.predicate(event)) {
			select {
			case w.ch <- event:
			default:
			}
		} else {
			remaining = append(remaining, w)
		}
	}
	c.waiters = remaining

	c.mu.Unlock()

	for _, h := range handlers {
		h(event)
	}
}

// resolveAction resolves a pending action by control ID.
func (c *Call) resolveAction(controlID string, event *RelayEvent) {
	c.mu.Lock()
	a, ok := c.actions[controlID]
	c.mu.Unlock()
	if ok {
		a.resolve(event)
	}
}

// registerAction stores an action so it can be resolved later.
func (c *Call) registerAction(a *Action) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.actions[a.controlID] = a
}

// newControlID generates a unique control identifier.
func newControlID() string {
	return uuid.New().String()
}

// ---------------------------------------------------------------------------
// Lifecycle methods
// ---------------------------------------------------------------------------

// Answer answers an inbound call.
func (c *Call) Answer() error {
	_, err := c.client.execute("calling.answer", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
	})
	return err
}

// Hangup ends the call with the given reason.
func (c *Call) Hangup(reason string) error {
	if reason == "" {
		reason = EndReasonHangup
	}
	_, err := c.client.execute("calling.end", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
		"reason":  reason,
	})
	return err
}

// Pass passes the call to the next context handler without answering.
func (c *Call) Pass() error {
	_, err := c.client.execute("calling.pass", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
	})
	return err
}

// Transfer transfers the call to another destination.
func (c *Call) Transfer(dest map[string]any) error {
	params := map[string]any{
		"node_id":     c.nodeID,
		"call_id":     c.callID,
		"destination": dest,
	}
	_, err := c.client.execute("calling.transfer", params)
	return err
}

// ---------------------------------------------------------------------------
// Audio: Play, PlayAndCollect, Collect
// ---------------------------------------------------------------------------

// Play starts playing media on the call and returns a PlayAction.
func (c *Call) Play(media []map[string]any, opts ...PlayOption) *PlayAction {
	controlID := newControlID()
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
		"play":       media,
	}
	for _, opt := range opts {
		opt(params)
	}

	action := newPlayAction(c, controlID)
	c.registerAction(action.Action)

	go func() {
		_, err := c.client.execute("calling.play", params)
		if err != nil {
			action.resolve(NewRelayEvent(EventCallingCallPlay, map[string]any{
				"control_id": controlID,
				"state":      "error",
				"error":      err.Error(),
			}))
		}
	}()

	return action
}

// PlayAndCollect plays media while collecting input (DTMF or speech).
func (c *Call) PlayAndCollect(media []map[string]any, collect map[string]any, opts ...PlayOption) *CollectAction {
	controlID := newControlID()
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
		"play":       media,
		"collect":    collect,
	}
	for _, opt := range opts {
		opt(params)
	}

	action := newCollectAction(c, controlID)
	c.registerAction(action.Action)

	go func() {
		_, err := c.client.execute("calling.play_and_collect", params)
		if err != nil {
			action.resolve(NewRelayEvent(EventCallingCallCollect, map[string]any{
				"control_id": controlID,
				"state":      "error",
				"error":      err.Error(),
			}))
		}
	}()

	return action
}

// Collect starts collecting user input without playing media.
func (c *Call) Collect(collect map[string]any) *StandaloneCollectAction {
	controlID := newControlID()
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
		"collect":    collect,
	}

	action := newStandaloneCollectAction(c, controlID)
	c.registerAction(action.Action)

	go func() {
		_, err := c.client.execute("calling.collect", params)
		if err != nil {
			action.resolve(NewRelayEvent(EventCallingCallCollect, map[string]any{
				"control_id": controlID,
				"state":      "error",
				"error":      err.Error(),
			}))
		}
	}()

	return action
}

// ---------------------------------------------------------------------------
// Recording
// ---------------------------------------------------------------------------

// Record starts recording the call and returns a RecordAction.
func (c *Call) Record(opts ...RecordOption) *RecordAction {
	controlID := newControlID()
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
	}
	for _, opt := range opts {
		opt(params)
	}

	action := newRecordAction(c, controlID)
	c.registerAction(action.Action)

	go func() {
		_, err := c.client.execute("calling.record", params)
		if err != nil {
			action.resolve(NewRelayEvent(EventCallingCallRecord, map[string]any{
				"control_id": controlID,
				"state":      "error",
				"error":      err.Error(),
			}))
		}
	}()

	return action
}

// ---------------------------------------------------------------------------
// Bridging / Connect
// ---------------------------------------------------------------------------

// Connect bridges this call to one or more devices.
func (c *Call) Connect(devices [][]map[string]any, opts ...ConnectOption) error {
	params := map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
		"devices": devices,
	}
	for _, opt := range opts {
		opt(params)
	}
	_, err := c.client.execute("calling.connect", params)
	return err
}

// Disconnect tears down a previously established bridge.
func (c *Call) Disconnect() error {
	_, err := c.client.execute("calling.disconnect", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
	})
	return err
}

// ---------------------------------------------------------------------------
// DTMF
// ---------------------------------------------------------------------------

// SendDigits sends DTMF digits on the call.
func (c *Call) SendDigits(digits string) error {
	controlID := newControlID()
	_, err := c.client.execute("calling.send_digits", map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
		"digits":     digits,
	})
	return err
}

// ---------------------------------------------------------------------------
// Detection
// ---------------------------------------------------------------------------

// Detect starts a detection operation (e.g., answering machine detection).
func (c *Call) Detect(detect map[string]any, timeout int) *DetectAction {
	controlID := newControlID()
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
		"detect":     detect,
		"timeout":    timeout,
	}

	action := newDetectAction(c, controlID)
	c.registerAction(action.Action)

	go func() {
		_, err := c.client.execute("calling.detect", params)
		if err != nil {
			action.resolve(NewRelayEvent(EventCallingCallDetect, map[string]any{
				"control_id": controlID,
				"state":      "error",
				"error":      err.Error(),
			}))
		}
	}()

	return action
}

// ---------------------------------------------------------------------------
// Fax
// ---------------------------------------------------------------------------

// SendFax sends a fax document on the call.
func (c *Call) SendFax(document string, identity string) *FaxAction {
	controlID := newControlID()
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
		"document":   document,
		"identity":   identity,
	}

	action := newFaxAction(c, controlID)
	c.registerAction(action.Action)

	go func() {
		_, err := c.client.execute("calling.send_fax", params)
		if err != nil {
			action.resolve(NewRelayEvent(EventCallingCallFax, map[string]any{
				"control_id": controlID,
				"state":      "error",
				"error":      err.Error(),
			}))
		}
	}()

	return action
}

// ReceiveFax starts receiving a fax on the call.
func (c *Call) ReceiveFax() *FaxAction {
	controlID := newControlID()
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
	}

	action := newFaxAction(c, controlID)
	c.registerAction(action.Action)

	go func() {
		_, err := c.client.execute("calling.receive_fax", params)
		if err != nil {
			action.resolve(NewRelayEvent(EventCallingCallFax, map[string]any{
				"control_id": controlID,
				"state":      "error",
				"error":      err.Error(),
			}))
		}
	}()

	return action
}

// ---------------------------------------------------------------------------
// Tap
// ---------------------------------------------------------------------------

// Tap starts tapping the call audio to an external destination.
func (c *Call) Tap(tap, device map[string]any) *TapAction {
	controlID := newControlID()
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
		"tap":        tap,
		"device":     device,
	}

	action := newTapAction(c, controlID)
	c.registerAction(action.Action)

	go func() {
		_, err := c.client.execute("calling.tap", params)
		if err != nil {
			action.resolve(NewRelayEvent(EventCallingCallTap, map[string]any{
				"control_id": controlID,
				"state":      "error",
				"error":      err.Error(),
			}))
		}
	}()

	return action
}

// ---------------------------------------------------------------------------
// Streaming
// ---------------------------------------------------------------------------

// Stream starts streaming call audio to a WebSocket URL.
func (c *Call) Stream(url string, opts ...StreamOption) *StreamAction {
	controlID := newControlID()
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
		"url":        url,
	}
	for _, opt := range opts {
		opt(params)
	}

	action := newStreamAction(c, controlID)
	c.registerAction(action.Action)

	go func() {
		_, err := c.client.execute("calling.stream", params)
		if err != nil {
			action.resolve(NewRelayEvent(EventCallingCallStream, map[string]any{
				"control_id": controlID,
				"state":      "error",
				"error":      err.Error(),
			}))
		}
	}()

	return action
}

// ---------------------------------------------------------------------------
// Conferencing
// ---------------------------------------------------------------------------

// JoinConference joins the call to a named conference.
func (c *Call) JoinConference(name string, opts ...ConferenceOption) error {
	params := map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
		"name":    name,
	}
	for _, opt := range opts {
		opt(params)
	}
	_, err := c.client.execute("calling.conference", params)
	return err
}

// LeaveConference removes the call from a conference.
func (c *Call) LeaveConference(confID string) error {
	_, err := c.client.execute("calling.conference.leave", map[string]any{
		"node_id":       c.nodeID,
		"call_id":       c.callID,
		"conference_id": confID,
	})
	return err
}

// ---------------------------------------------------------------------------
// AI
// ---------------------------------------------------------------------------

// AI starts an AI session on the call.
func (c *Call) AI(opts ...AIOption) *AIAction {
	controlID := newControlID()
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
	}
	for _, opt := range opts {
		opt(params)
	}

	action := newAIAction(c, controlID)
	c.registerAction(action.Action)

	go func() {
		_, err := c.client.execute("calling.ai", params)
		if err != nil {
			action.resolve(NewRelayEvent(EventCallingCallAI, map[string]any{
				"control_id": controlID,
				"state":      "error",
				"error":      err.Error(),
			}))
		}
	}()

	return action
}

// AmazonBedrock starts an AI session using Amazon Bedrock.
func (c *Call) AmazonBedrock(opts ...AIOption) *AIAction {
	merged := append([]AIOption{WithAIEngine("amazon_bedrock")}, opts...)
	return c.AI(merged...)
}

// AIMessage sends a text message within an active AI session.
func (c *Call) AIMessage(controlID, text, role string) error {
	_, err := c.client.execute("calling.ai.message", map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
		"text":       text,
		"role":       role,
	})
	return err
}

// AIHold places the AI-controlled call on hold.
func (c *Call) AIHold(controlID string, timeout int) error {
	_, err := c.client.execute("calling.ai.hold", map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
		"timeout":    timeout,
	})
	return err
}

// AIUnhold removes the call from AI hold.
func (c *Call) AIUnhold(controlID string) error {
	_, err := c.client.execute("calling.ai.unhold", map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
	})
	return err
}

// ---------------------------------------------------------------------------
// Hold / Denoise
// ---------------------------------------------------------------------------

// Hold places the call on hold.
func (c *Call) Hold() error {
	_, err := c.client.execute("calling.hold", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
	})
	return err
}

// Unhold takes the call off hold.
func (c *Call) Unhold() error {
	_, err := c.client.execute("calling.unhold", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
	})
	return err
}

// Denoise starts noise reduction on the call.
func (c *Call) Denoise() error {
	_, err := c.client.execute("calling.denoise", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
	})
	return err
}

// DenoiseStop stops noise reduction on the call.
func (c *Call) DenoiseStop() error {
	_, err := c.client.execute("calling.denoise.stop", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
	})
	return err
}

// ---------------------------------------------------------------------------
// Room / Queue
// ---------------------------------------------------------------------------

// JoinRoom joins the call to a named room.
func (c *Call) JoinRoom(name string) error {
	_, err := c.client.execute("calling.room.join", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
		"name":    name,
	})
	return err
}

// LeaveRoom removes the call from the current room.
func (c *Call) LeaveRoom() error {
	_, err := c.client.execute("calling.room.leave", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
	})
	return err
}

// QueueEnter places the call in a named queue.
func (c *Call) QueueEnter(name string) error {
	_, err := c.client.execute("calling.queue.enter", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
		"name":    name,
	})
	return err
}

// QueueLeave removes the call from the named queue.
func (c *Call) QueueLeave(name string) error {
	_, err := c.client.execute("calling.queue.leave", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
		"name":    name,
	})
	return err
}

// ---------------------------------------------------------------------------
// Digit Binding / User Event / Echo
// ---------------------------------------------------------------------------

// BindDigit binds a DTMF digit sequence to trigger a method call.
func (c *Call) BindDigit(digits, method string, params map[string]any) error {
	p := map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
		"digits":  digits,
		"method":  method,
	}
	if params != nil {
		p["params"] = params
	}
	_, err := c.client.execute("calling.bind_digit", p)
	return err
}

// ClearDigitBindings clears all DTMF digit bindings.
func (c *Call) ClearDigitBindings() error {
	_, err := c.client.execute("calling.clear_digit_bindings", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
	})
	return err
}

// UserEvent sends a user-defined event on the call.
func (c *Call) UserEvent(event map[string]any) error {
	_, err := c.client.execute("calling.user_event", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
		"event":   event,
	})
	return err
}

// Echo starts echo mode on the call (echo audio back to the caller).
func (c *Call) Echo(timeout int) error {
	_, err := c.client.execute("calling.echo", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
		"timeout": timeout,
	})
	return err
}

// ---------------------------------------------------------------------------
// Pay
// ---------------------------------------------------------------------------

// Pay starts a pay session on the call.
func (c *Call) Pay(connectorURL string, amount string, currency string) *PayAction {
	controlID := newControlID()
	params := map[string]any{
		"node_id":       c.nodeID,
		"call_id":       c.callID,
		"control_id":    controlID,
		"connector_url": connectorURL,
		"amount":        amount,
		"currency":      currency,
	}

	action := newPayAction(c, controlID)
	c.registerAction(action.Action)

	go func() {
		_, err := c.client.execute("calling.pay", params)
		if err != nil {
			action.resolve(NewRelayEvent(EventCallingCallPay, map[string]any{
				"control_id": controlID,
				"state":      "error",
				"error":      err.Error(),
			}))
		}
	}()

	return action
}

// ---------------------------------------------------------------------------
// Transcribe
// ---------------------------------------------------------------------------

// Transcribe starts real-time transcription on the call.
func (c *Call) Transcribe(statusURL string) *TranscribeAction {
	controlID := newControlID()
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
	}
	if statusURL != "" {
		params["status_url"] = statusURL
	}

	action := newTranscribeAction(c, controlID)
	c.registerAction(action.Action)

	go func() {
		_, err := c.client.execute("calling.transcribe", params)
		if err != nil {
			action.resolve(NewRelayEvent(EventCallingCallTranscribe, map[string]any{
				"control_id": controlID,
				"state":      "error",
				"error":      err.Error(),
			}))
		}
	}()

	return action
}

// ---------------------------------------------------------------------------
// String representation
// ---------------------------------------------------------------------------

// String returns a human-readable representation of the call.
func (c *Call) String() string {
	return fmt.Sprintf("Call{id=%s, node=%s, tag=%s, state=%s}", c.callID, c.nodeID, c.tag, c.State())
}
