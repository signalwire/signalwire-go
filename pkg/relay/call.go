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
	callID    string
	nodeID    string
	tag       string
	client    *Client
	state     string
	projectID string
	context   string
	direction string
	device    map[string]any
	segmentID string
	mu        sync.Mutex
	actions   map[string]*Action

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

// ProjectID returns the project ID associated with this call.
func (c *Call) ProjectID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.projectID
}

// Context returns the RELAY context this call was received on.
func (c *Call) Context() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.context
}

// Direction returns the call direction ("inbound" or "outbound").
func (c *Call) Direction() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.direction
}

// Device returns the device map describing the endpoint of this call.
func (c *Call) Device() map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.device
}

// SegmentID returns the segment identifier for this call leg.
func (c *Call) SegmentID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.segmentID
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

// WaitForEnded blocks until the call reaches the "ended" state, or the
// context expires. This mirrors Python's wait_for_ended() which awaits
// the _ended asyncio.Future.
func (c *Call) WaitForEnded(ctx context.Context) (*RelayEvent, error) {
	return c.WaitFor(ctx, EventCallingCallState, func(e *RelayEvent) bool {
		return e.GetString("call_state") == CallStateEnded
	})
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
		if d := event.GetString("direction"); d != "" && c.direction == "" {
			c.direction = d
		}
		if dev := event.GetMap("device"); dev != nil && c.device == nil {
			c.device = dev
		}
	}

	// Populate metadata from inbound receive events.
	if event.EventType == EventCallingCallReceive {
		if ctx := event.GetString("context"); ctx != "" && c.context == "" {
			c.context = ctx
		}
		if dev := event.GetMap("device"); dev != nil && c.device == nil {
			c.device = dev
		}
		if pid := event.GetString("project_id"); pid != "" && c.projectID == "" {
			c.projectID = pid
		}
		if seg := event.GetString("segment_id"); seg != "" && c.segmentID == "" {
			c.segmentID = seg
		}
		if d := event.GetString("direction"); d != "" && c.direction == "" {
			c.direction = d
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

// Transfer transfers call control to another RELAY app or SWML script.
// The dest parameter is the destination context/URL string, sent as the
// "dest" key to the server (matches Python's transfer(dest: str) behavior).
func (c *Call) Transfer(dest string) error {
	_, err := c.client.execute("calling.transfer", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
		"dest":    dest,
	})
	return err
}

// ---------------------------------------------------------------------------
// SIP Refer
// ---------------------------------------------------------------------------

// Refer transfers a SIP call to an external SIP endpoint via a REFER request.
// statusURL is optional (empty string omits it), matching Python's
// refer(device, *, status_url).
func (c *Call) Refer(device map[string]any, statusURL string) error {
	params := map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
		"device":  device,
	}
	if statusURL != "" {
		params["status_url"] = statusURL
	}
	_, err := c.client.execute("calling.refer", params)
	return err
}

// ---------------------------------------------------------------------------
// Live Transcribe / Translate
// ---------------------------------------------------------------------------

// LiveTranscribe starts or stops live transcription on the call. The action
// map describes the transcription operation (e.g. {"type": "start"}).
// Matches Python's live_transcribe(action, **kwargs).
func (c *Call) LiveTranscribe(action map[string]any) error {
	_, err := c.client.execute("calling.live_transcribe", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
		"action":  action,
	})
	return err
}

// LiveTranslate starts or stops live translation on the call. The action map
// describes the translation operation. statusURL is optional (empty string
// omits it), matching Python's live_translate(action, *, status_url).
func (c *Call) LiveTranslate(action map[string]any, statusURL string) error {
	params := map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
		"action":  action,
	}
	if statusURL != "" {
		params["status_url"] = statusURL
	}
	_, err := c.client.execute("calling.live_translate", params)
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

// CollectParams holds named parameters for the Collect method, matching
// Python's collect() named arguments.
type CollectParams struct {
	// Digits configures DTMF digit collection.
	Digits map[string]any
	// Speech configures speech recognition collection.
	Speech map[string]any
	// InitialTimeout is the number of seconds to wait for first input.
	InitialTimeout *float64
	// PartialResults enables streaming partial results as input is gathered.
	PartialResults *bool
	// Continuous enables continuous collection after a result is received.
	Continuous *bool
	// SendStartOfInput signals when the user begins speaking/pressing.
	SendStartOfInput *bool
	// StartInputTimers controls whether input timers start immediately.
	StartInputTimers *bool
}

// Collect starts collecting user input without playing media. The params
// argument exposes named fields that mirror Python's collect() parameters.
// Pass a nil CollectParams to send an empty collect body.
func (c *Call) Collect(params *CollectParams) *StandaloneCollectAction {
	controlID := newControlID()
	collect := map[string]any{}
	if params != nil {
		if params.Digits != nil {
			collect["digits"] = params.Digits
		}
		if params.Speech != nil {
			collect["speech"] = params.Speech
		}
		if params.InitialTimeout != nil {
			collect["initial_timeout"] = *params.InitialTimeout
		}
		if params.PartialResults != nil {
			collect["partial_results"] = *params.PartialResults
		}
		if params.Continuous != nil {
			collect["continuous"] = *params.Continuous
		}
		if params.SendStartOfInput != nil {
			collect["send_start_of_input"] = *params.SendStartOfInput
		}
		if params.StartInputTimers != nil {
			collect["start_input_timers"] = *params.StartInputTimers
		}
	}
	rpcParams := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
		"collect":    collect,
	}

	action := newStandaloneCollectAction(c, controlID)
	c.registerAction(action.Action)

	go func() {
		_, err := c.client.execute("calling.collect", rpcParams)
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
// timeout is optional; pass nil to omit it from the request (matches Python's
// optional float timeout parameter).
func (c *Call) Detect(detect map[string]any, timeout *float64) *DetectAction {
	controlID := newControlID()
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
		"detect":     detect,
	}
	if timeout != nil {
		params["timeout"] = *timeout
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

// SendFax sends a fax document on the call. Use WithFaxHeaderInfo to include
// a fax header string (matches Python's send_fax header_info parameter).
func (c *Call) SendFax(document string, identity string, opts ...FaxOption) *FaxAction {
	controlID := newControlID()
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
		"document":   document,
	}
	if identity != "" {
		params["identity"] = identity
	}
	for _, opt := range opts {
		opt(params)
	}

	action := newFaxAction(c, controlID, "send_fax")
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

	action := newFaxAction(c, controlID, "receive_fax")
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
	_, err := c.client.execute("calling.join_conference", params)
	return err
}

// LeaveConference removes the call from a conference.
func (c *Call) LeaveConference(confID string) error {
	_, err := c.client.execute("calling.leave_conference", map[string]any{
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

// AIMessage sends a text message within an active AI session. All parameters
// are optional, matching Python's ai_message(*, message_text=None, role=None,
// reset=None, global_data=None). Pass "" for controlID/text/role and nil for
// reset/globalData to omit them from the wire payload (Python omits the key
// entirely when the argument is None).
func (c *Call) AIMessage(controlID, text, role string, reset map[string]any, globalData map[string]any) error {
	params := map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
	}
	if controlID != "" {
		params["control_id"] = controlID
	}
	if text != "" {
		params["message_text"] = text
	}
	if role != "" {
		params["role"] = role
	}
	if reset != nil {
		params["reset"] = reset
	}
	if globalData != nil {
		params["global_data"] = globalData
	}
	_, err := c.client.execute("calling.ai_message", params)
	return err
}

// AIHold places the AI-controlled call on hold. controlID, timeout and prompt
// are all optional — pass "" to omit any of them, matching Python's
// ai_hold(*, timeout: Optional[str] = None, prompt: Optional[str] = None)
// which has no control_id parameter and only writes keys conditionally.
func (c *Call) AIHold(controlID string, timeout string, prompt string) error {
	params := map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
	}
	if controlID != "" {
		params["control_id"] = controlID
	}
	if timeout != "" {
		params["timeout"] = timeout
	}
	if prompt != "" {
		params["prompt"] = prompt
	}
	_, err := c.client.execute("calling.ai_hold", params)
	return err
}

// AIUnhold removes the call from AI hold. prompt is optional (empty string
// omits it), matching Python's ai_unhold(*, prompt: str|None).
func (c *Call) AIUnhold(controlID string, prompt string) error {
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": controlID,
	}
	if prompt != "" {
		params["prompt"] = prompt
	}
	_, err := c.client.execute("calling.ai_unhold", params)
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

// JoinRoom joins the call to a named room. statusURL is optional (empty
// string omits it), matching Python's join_room(name, *, status_url).
func (c *Call) JoinRoom(name string, statusURL string) error {
	params := map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
		"name":    name,
	}
	if statusURL != "" {
		params["status_url"] = statusURL
	}
	_, err := c.client.execute("calling.join_room", params)
	return err
}

// LeaveRoom removes the call from the current room.
func (c *Call) LeaveRoom() error {
	_, err := c.client.execute("calling.leave_room", map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
	})
	return err
}

// QueueEnter places the call in a named queue. statusURL is optional (empty
// string omits it), matching Python's queue_enter(queue_name, *, control_id,
// status_url) at signalwire/relay/call.py:1268. A per-request control_id is
// generated so the server can correlate this action with subsequent events.
func (c *Call) QueueEnter(name string, statusURL string) error {
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": newControlID(),
		"queue_name": name,
	}
	if statusURL != "" {
		params["status_url"] = statusURL
	}
	_, err := c.client.execute("calling.queue.enter", params)
	return err
}

// QueueLeave removes the call from the named queue. queueID and statusURL are
// optional (empty string omits each), matching Python's
// queue_leave(queue_name, *, control_id, queue_id, status_url) at
// signalwire/relay/call.py:1287. A per-request control_id is generated.
func (c *Call) QueueLeave(name string, queueID string, statusURL string) error {
	params := map[string]any{
		"node_id":    c.nodeID,
		"call_id":    c.callID,
		"control_id": newControlID(),
		"queue_name": name,
	}
	if queueID != "" {
		params["queue_id"] = queueID
	}
	if statusURL != "" {
		params["status_url"] = statusURL
	}
	_, err := c.client.execute("calling.queue.leave", params)
	return err
}

// ---------------------------------------------------------------------------
// Digit Binding / User Event / Echo
// ---------------------------------------------------------------------------

// BindDigit binds a DTMF digit sequence to trigger a RELAY method.
// bindParams, realm, and maxTriggers are optional (nil/zero-value omits them),
// matching Python's bind_digit(digits, bind_method, *, bind_params, realm, max_triggers).
func (c *Call) BindDigit(digits, method string, bindParams map[string]any, realm string, maxTriggers int) error {
	p := map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
		"digits":  digits,
		"method":  method,
	}
	if bindParams != nil {
		p["params"] = bindParams
	}
	if realm != "" {
		p["realm"] = realm
	}
	if maxTriggers > 0 {
		p["max_triggers"] = maxTriggers
	}
	_, err := c.client.execute("calling.bind_digit", p)
	return err
}

// ClearDigitBindings clears all DTMF digit bindings, optionally filtered
// by realm. Pass an empty string to clear all realms (matches Python's
// clear_digit_bindings(*, realm)).
func (c *Call) ClearDigitBindings(realm string) error {
	params := map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
	}
	if realm != "" {
		params["realm"] = realm
	}
	_, err := c.client.execute("calling.clear_digit_bindings", params)
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
// Both timeout and statusURL are optional (nil omits them), matching
// Python's echo(*, timeout: float|None, status_url).
func (c *Call) Echo(timeout *float64, statusURL string) error {
	params := map[string]any{
		"node_id": c.nodeID,
		"call_id": c.callID,
	}
	if timeout != nil {
		params["timeout"] = *timeout
	}
	if statusURL != "" {
		params["status_url"] = statusURL
	}
	_, err := c.client.execute("calling.echo", params)
	return err
}

// ---------------------------------------------------------------------------
// Pay
// ---------------------------------------------------------------------------

// Pay starts a payment collection session on the call. Use PayOption
// functional options to supply any of the 20+ optional parameters that
// Python's pay() exposes (input_method, status_url, payment_method, timeout,
// max_attempts, security_code, postal_code, min_postal_code_length,
// token_type, charge_amount, currency, language, voice, description,
// valid_card_types, parameters, prompts).
func (c *Call) Pay(connectorURL string, opts ...PayOption) *PayAction {
	controlID := newControlID()
	params := map[string]any{
		"node_id":                c.nodeID,
		"call_id":                c.callID,
		"control_id":             controlID,
		"payment_connector_url":  connectorURL,
	}
	for _, opt := range opts {
		opt(params)
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
