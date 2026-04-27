package relay

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Client creation with options
// ---------------------------------------------------------------------------

func TestNewRelayClient_Defaults(t *testing.T) {
	c := NewRelayClient()
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.projectID != "" {
		t.Errorf("expected empty projectID, got %q", c.projectID)
	}
	if c.pending == nil {
		t.Fatal("expected pending map to be initialized")
	}
	if c.calls == nil {
		t.Fatal("expected calls map to be initialized")
	}
	if c.messages == nil {
		t.Fatal("expected messages map to be initialized")
	}
	if c.pendingDials == nil {
		t.Fatal("expected pendingDials map to be initialized")
	}
}

func TestNewRelayClient_WithOptions(t *testing.T) {
	c := NewRelayClient(
		WithProject("proj-123"),
		WithToken("tok-abc"),
		WithSpace("example.signalwire.com"),
		WithContexts("office", "mobile"),
		WithMaxActiveCalls(10),
	)
	if c.projectID != "proj-123" {
		t.Errorf("projectID = %q, want %q", c.projectID, "proj-123")
	}
	if c.token != "tok-abc" {
		t.Errorf("token = %q, want %q", c.token, "tok-abc")
	}
	if c.space != "example.signalwire.com" {
		t.Errorf("space = %q, want %q", c.space, "example.signalwire.com")
	}
	if len(c.contexts) != 2 || c.contexts[0] != "office" || c.contexts[1] != "mobile" {
		t.Errorf("contexts = %v, want [office mobile]", c.contexts)
	}
	if c.maxActiveCalls != 10 {
		t.Errorf("maxActiveCalls = %d, want 10", c.maxActiveCalls)
	}
}

func TestNewRelayClient_WithJWT(t *testing.T) {
	c := NewRelayClient(WithJWT("jwt-token-value"))
	if c.jwtToken != "jwt-token-value" {
		t.Errorf("jwtToken = %q, want %q", c.jwtToken, "jwt-token-value")
	}
}

func TestClient_OnCallHandler(t *testing.T) {
	c := NewRelayClient()
	called := false
	c.OnCall(func(call *Call) {
		called = true
	})
	if c.onCall == nil {
		t.Fatal("expected onCall handler to be set")
	}
	// We can't test invocation without a real connection, but verify the
	// handler was stored.
	_ = called
}

func TestClient_OnMessageHandler(t *testing.T) {
	c := NewRelayClient()
	c.OnMessage(func(msg *Message) {})
	if c.onMessage == nil {
		t.Fatal("expected onMessage handler to be set")
	}
}

// ---------------------------------------------------------------------------
// Event parsing
// ---------------------------------------------------------------------------

func TestRelayEvent_GetString(t *testing.T) {
	e := NewRelayEvent("test", map[string]any{
		"name":  "Alice",
		"count": 42,
	})
	if s := e.GetString("name"); s != "Alice" {
		t.Errorf("GetString(name) = %q, want %q", s, "Alice")
	}
	if s := e.GetString("count"); s != "42" {
		t.Errorf("GetString(count) = %q, want %q", s, "42")
	}
	if s := e.GetString("missing"); s != "" {
		t.Errorf("GetString(missing) = %q, want %q", s, "")
	}
}

func TestRelayEvent_GetInt(t *testing.T) {
	e := NewRelayEvent("test", map[string]any{
		"int_val":    42,
		"float_val":  3.14,
		"string_val": "99",
		"bad_val":    "abc",
	})
	if v := e.GetInt("int_val"); v != 42 {
		t.Errorf("GetInt(int_val) = %d, want 42", v)
	}
	if v := e.GetInt("float_val"); v != 3 {
		t.Errorf("GetInt(float_val) = %d, want 3", v)
	}
	if v := e.GetInt("string_val"); v != 99 {
		t.Errorf("GetInt(string_val) = %d, want 99", v)
	}
	if v := e.GetInt("bad_val"); v != 0 {
		t.Errorf("GetInt(bad_val) = %d, want 0", v)
	}
	if v := e.GetInt("missing"); v != 0 {
		t.Errorf("GetInt(missing) = %d, want 0", v)
	}
}

func TestRelayEvent_GetBool(t *testing.T) {
	e := NewRelayEvent("test", map[string]any{
		"bool_true":   true,
		"bool_false":  false,
		"str_true":    "true",
		"str_one":     "1",
		"int_nonzero": 1,
		"float_zero":  0.0,
	})
	if v := e.GetBool("bool_true"); !v {
		t.Error("GetBool(bool_true) = false, want true")
	}
	if v := e.GetBool("bool_false"); v {
		t.Error("GetBool(bool_false) = true, want false")
	}
	if v := e.GetBool("str_true"); !v {
		t.Error("GetBool(str_true) = false, want true")
	}
	if v := e.GetBool("str_one"); !v {
		t.Error("GetBool(str_one) = false, want true")
	}
	if v := e.GetBool("int_nonzero"); !v {
		t.Error("GetBool(int_nonzero) = false, want true")
	}
	if v := e.GetBool("float_zero"); v {
		t.Error("GetBool(float_zero) = true, want false")
	}
	if v := e.GetBool("missing"); v {
		t.Error("GetBool(missing) = true, want false")
	}
}

func TestRelayEvent_GetMap(t *testing.T) {
	inner := map[string]any{"key": "value"}
	e := NewRelayEvent("test", map[string]any{
		"nested": inner,
		"flat":   "not_a_map",
	})
	m := e.GetMap("nested")
	if m == nil {
		t.Fatal("GetMap(nested) returned nil")
	}
	if m["key"] != "value" {
		t.Errorf("nested[key] = %v, want %q", m["key"], "value")
	}
	if e.GetMap("flat") != nil {
		t.Error("GetMap(flat) should return nil for non-map value")
	}
	if e.GetMap("missing") != nil {
		t.Error("GetMap(missing) should return nil")
	}
}

func TestRelayEvent_NilParams(t *testing.T) {
	e := NewRelayEvent("test", nil)
	if e.Params == nil {
		t.Error("NewRelayEvent should initialize nil params to empty map")
	}
	if s := e.GetString("x"); s != "" {
		t.Errorf("GetString on empty params = %q, want %q", s, "")
	}
	if v := e.GetInt("x"); v != 0 {
		t.Errorf("GetInt on empty params = %d, want 0", v)
	}
	if v := e.GetBool("x"); v {
		t.Error("GetBool on empty params = true, want false")
	}
	if v := e.GetMap("x"); v != nil {
		t.Error("GetMap on empty params should return nil")
	}
}

func TestCallStateEvent(t *testing.T) {
	e := NewCallStateEvent(map[string]any{
		"call_state": "answered",
		"direction":  "inbound",
		"call_id":    "call-1",
		"node_id":    "node-1",
		"tag":        "tag-1",
		"end_reason": "",
	})
	if e.CallState != "answered" {
		t.Errorf("CallState = %q, want %q", e.CallState, "answered")
	}
	if e.Direction != "inbound" {
		t.Errorf("Direction = %q, want %q", e.Direction, "inbound")
	}
	if e.CallID != "call-1" {
		t.Errorf("CallID = %q, want %q", e.CallID, "call-1")
	}
	if e.NodeID != "node-1" {
		t.Errorf("NodeID = %q, want %q", e.NodeID, "node-1")
	}
	if e.EventType != EventCallingCallState {
		t.Errorf("EventType = %q, want %q", e.EventType, EventCallingCallState)
	}
}

func TestCallReceiveEvent(t *testing.T) {
	e := NewCallReceiveEvent(map[string]any{
		"call_state": "ringing",
		"context":    "office",
		"call_id":    "call-2",
		"node_id":    "node-2",
		"tag":        "tag-2",
		"device": map[string]any{
			"type":   "phone",
			"params": map[string]any{"to_number": "+15551234567"},
		},
	})
	if e.CallState != "ringing" {
		t.Errorf("CallState = %q, want %q", e.CallState, "ringing")
	}
	if e.Context != "office" {
		t.Errorf("Context = %q, want %q", e.Context, "office")
	}
	if e.Device == nil {
		t.Fatal("Device should not be nil")
	}
	if e.Device["type"] != "phone" {
		t.Errorf("Device type = %v, want %q", e.Device["type"], "phone")
	}
}

func TestPlayEvent(t *testing.T) {
	e := NewPlayEvent(map[string]any{
		"control_id": "ctrl-1",
		"state":      "finished",
	})
	if e.ControlID != "ctrl-1" {
		t.Errorf("ControlID = %q, want %q", e.ControlID, "ctrl-1")
	}
	if e.State != "finished" {
		t.Errorf("State = %q, want %q", e.State, "finished")
	}
}

func TestRecordEvent(t *testing.T) {
	e := NewRecordEvent(map[string]any{
		"control_id": "ctrl-2",
		"state":      "finished",
		"url":        "https://example.com/recording.wav",
		"duration":   30.0,
		"size":       48000.0,
	})
	if e.URL != "https://example.com/recording.wav" {
		t.Errorf("URL = %q, want expected URL", e.URL)
	}
	if e.Duration != 30.0 {
		t.Errorf("Duration = %v, want 30.0", e.Duration)
	}
	if e.Size != 48000 {
		t.Errorf("Size = %d, want 48000", e.Size)
	}
}

func TestMessageReceiveEvent(t *testing.T) {
	e := NewMessageReceiveEvent(map[string]any{
		"message_id":  "msg-1",
		"direction":   "inbound",
		"from_number": "+15551234567",
		"to_number":   "+15559876543",
		"body":        "Hello!",
		"segments":    1.0,
		"media":       []any{"https://example.com/image.jpg"},
		"tags":        []any{"tag1", "tag2"},
	})
	if e.MessageID != "msg-1" {
		t.Errorf("MessageID = %q, want %q", e.MessageID, "msg-1")
	}
	if e.Body != "Hello!" {
		t.Errorf("Body = %q, want %q", e.Body, "Hello!")
	}
	if len(e.Media) != 1 || e.Media[0] != "https://example.com/image.jpg" {
		t.Errorf("Media = %v, want [https://example.com/image.jpg]", e.Media)
	}
	if len(e.Tags) != 2 || e.Tags[0] != "tag1" {
		t.Errorf("Tags = %v, want [tag1 tag2]", e.Tags)
	}
}

func TestMessageStateEvent(t *testing.T) {
	e := NewMessageStateEvent(map[string]any{
		"message_id":    "msg-2",
		"message_state": "delivered",
		"direction":     "outbound",
		"from_number":   "+15551234567",
		"to_number":     "+15559876543",
	})
	if e.MessageState != "delivered" {
		t.Errorf("MessageState = %q, want %q", e.MessageState, "delivered")
	}
}

func TestAIEvent(t *testing.T) {
	e := NewAIEvent(map[string]any{
		"control_id": "ctrl-ai-1",
		"state":      "finished",
		"result":     map[string]any{"summary": "call ended"},
	})
	if e.ControlID != "ctrl-ai-1" {
		t.Errorf("ControlID = %q, want %q", e.ControlID, "ctrl-ai-1")
	}
	if e.State != "finished" {
		t.Errorf("State = %q, want %q", e.State, "finished")
	}
	if e.Result == nil || e.Result["summary"] != "call ended" {
		t.Errorf("Result = %v, want map with summary", e.Result)
	}
}

// ---------------------------------------------------------------------------
// Action wait/resolve
// ---------------------------------------------------------------------------

func TestAction_ResolveAndWait(t *testing.T) {
	// Use a nil call for unit testing since we won't call Stop().
	a := newAction(nil, "ctrl-1")

	if a.IsDone() {
		t.Error("new action should not be done")
	}
	if a.Completed() {
		t.Error("new action should not be completed")
	}
	if a.Result() != nil {
		t.Error("new action should have nil result")
	}

	event := NewRelayEvent("test.complete", map[string]any{"status": "ok"})

	// Resolve in a goroutine.
	go func() {
		time.Sleep(10 * time.Millisecond)
		a.resolve(event)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := a.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Wait returned nil result")
	}
	if result.GetString("status") != "ok" {
		t.Errorf("result status = %q, want %q", result.GetString("status"), "ok")
	}

	if !a.IsDone() {
		t.Error("action should be done after resolve")
	}
	if !a.Completed() {
		t.Error("action should be completed after resolve")
	}
}

func TestAction_WaitTimeout(t *testing.T) {
	a := newAction(nil, "ctrl-2")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := a.Wait(ctx)
	if err == nil {
		t.Fatal("Wait should return error on timeout")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestAction_OnCompleted(t *testing.T) {
	a := newAction(nil, "ctrl-3")

	var callbackEvent *RelayEvent
	var wg sync.WaitGroup
	wg.Add(1)
	a.OnCompleted(func(e *RelayEvent) {
		callbackEvent = e
		wg.Done()
	})

	event := NewRelayEvent("test.done", map[string]any{"value": "123"})
	a.resolve(event)

	wg.Wait()
	if callbackEvent == nil {
		t.Fatal("OnCompleted callback not called")
	}
	if callbackEvent.GetString("value") != "123" {
		t.Errorf("callback event value = %q, want %q", callbackEvent.GetString("value"), "123")
	}
}

func TestAction_DoubleResolve(t *testing.T) {
	a := newAction(nil, "ctrl-4")

	event1 := NewRelayEvent("first", map[string]any{"n": 1.0})
	event2 := NewRelayEvent("second", map[string]any{"n": 2.0})

	a.resolve(event1)
	a.resolve(event2) // Should be a no-op.

	if a.Result().EventType != "first" {
		t.Errorf("expected first event to win, got %q", a.Result().EventType)
	}
}

func TestPlayAction_Embed(t *testing.T) {
	pa := newPlayAction(nil, "play-ctrl-1")
	if pa.ControlID() != "play-ctrl-1" {
		t.Errorf("PlayAction.ControlID() = %q, want %q", pa.ControlID(), "play-ctrl-1")
	}
	if pa.IsDone() {
		t.Error("new PlayAction should not be done")
	}

	event := NewRelayEvent(EventCallingCallPlay, map[string]any{"state": "finished"})
	pa.resolve(event)
	if !pa.IsDone() {
		t.Error("PlayAction should be done after resolve")
	}
}

// ---------------------------------------------------------------------------
// Call creation
// ---------------------------------------------------------------------------

func TestCall_NewCall(t *testing.T) {
	c := NewRelayClient()
	call := newCall(c, "call-abc", "node-xyz", "tag-123")

	if call.CallID() != "call-abc" {
		t.Errorf("CallID = %q, want %q", call.CallID(), "call-abc")
	}
	if call.NodeID() != "node-xyz" {
		t.Errorf("NodeID = %q, want %q", call.NodeID(), "node-xyz")
	}
	if call.Tag() != "tag-123" {
		t.Errorf("Tag = %q, want %q", call.Tag(), "tag-123")
	}
	if call.State() != CallStateCreated {
		t.Errorf("State = %q, want %q", call.State(), CallStateCreated)
	}
}

func TestCall_StateUpdate(t *testing.T) {
	c := NewRelayClient()
	call := newCall(c, "call-1", "node-1", "tag-1")

	// Simulate a state change event.
	event := NewRelayEvent(EventCallingCallState, map[string]any{
		"call_state": CallStateAnswered,
		"call_id":    "call-1",
	})
	call.dispatchEvent(event)

	if call.State() != CallStateAnswered {
		t.Errorf("State after event = %q, want %q", call.State(), CallStateAnswered)
	}
}

func TestCall_EventHandler(t *testing.T) {
	c := NewRelayClient()
	call := newCall(c, "call-1", "node-1", "tag-1")

	var received *RelayEvent
	call.On(EventCallingCallState, func(e *RelayEvent) {
		received = e
	})

	event := NewRelayEvent(EventCallingCallState, map[string]any{
		"call_state": CallStateRinging,
	})
	call.dispatchEvent(event)

	if received == nil {
		t.Fatal("event handler was not called")
	}
	if received.GetString("call_state") != CallStateRinging {
		t.Errorf("received call_state = %q, want %q", received.GetString("call_state"), CallStateRinging)
	}
}

func TestCall_WaitFor(t *testing.T) {
	c := NewRelayClient()
	call := newCall(c, "call-1", "node-1", "tag-1")

	go func() {
		time.Sleep(10 * time.Millisecond)
		event := NewRelayEvent(EventCallingCallState, map[string]any{
			"call_state": CallStateAnswered,
		})
		call.dispatchEvent(event)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := call.WaitFor(ctx, EventCallingCallState, func(e *RelayEvent) bool {
		return e.GetString("call_state") == CallStateAnswered
	})
	if err != nil {
		t.Fatalf("WaitFor error: %v", err)
	}
	if result.GetString("call_state") != CallStateAnswered {
		t.Errorf("WaitFor result state = %q, want %q", result.GetString("call_state"), CallStateAnswered)
	}
}

func TestCall_WaitForTimeout(t *testing.T) {
	c := NewRelayClient()
	call := newCall(c, "call-1", "node-1", "tag-1")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := call.WaitFor(ctx, EventCallingCallState, func(e *RelayEvent) bool {
		return e.GetString("call_state") == CallStateEnded
	})
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestCall_ActionRegistration(t *testing.T) {
	c := NewRelayClient()
	call := newCall(c, "call-1", "node-1", "tag-1")

	a := newAction(call, "ctrl-abc")
	call.registerAction(a)

	// Resolve through the call.
	event := NewRelayEvent(EventCallingCallPlay, map[string]any{
		"control_id": "ctrl-abc",
		"state":      "finished",
	})
	call.resolveAction("ctrl-abc", event)

	if !a.IsDone() {
		t.Error("action should be done after resolveAction")
	}
}

func TestCall_String(t *testing.T) {
	c := NewRelayClient()
	call := newCall(c, "call-1", "node-1", "tag-1")
	s := call.String()
	if s == "" {
		t.Error("String() should return non-empty")
	}
}

// ---------------------------------------------------------------------------
// Message creation
// ---------------------------------------------------------------------------

func TestMessage_Creation(t *testing.T) {
	msg := newMessage("msg-1", DirectionOutbound, "+15551111111", "+15552222222", "Hello")

	if msg.MessageID() != "msg-1" {
		t.Errorf("MessageID = %q, want %q", msg.MessageID(), "msg-1")
	}
	if msg.Direction() != DirectionOutbound {
		t.Errorf("Direction = %q, want %q", msg.Direction(), DirectionOutbound)
	}
	if msg.FromNumber() != "+15551111111" {
		t.Errorf("FromNumber = %q, want %q", msg.FromNumber(), "+15551111111")
	}
	if msg.ToNumber() != "+15552222222" {
		t.Errorf("ToNumber = %q, want %q", msg.ToNumber(), "+15552222222")
	}
	if msg.Body() != "Hello" {
		t.Errorf("Body = %q, want %q", msg.Body(), "Hello")
	}
	if msg.IsDone() {
		t.Error("new message should not be done")
	}
}

func TestMessage_StateUpdate(t *testing.T) {
	msg := newMessage("msg-1", DirectionOutbound, "+1", "+2", "test")

	// Update to queued (non-terminal).
	msg.updateState(NewRelayEvent(EventMessagingState, map[string]any{
		"state": MessageStateQueued,
	}))
	if msg.State() != MessageStateQueued {
		t.Errorf("State = %q, want %q", msg.State(), MessageStateQueued)
	}
	if msg.IsDone() {
		t.Error("queued is not terminal")
	}

	// Update to delivered (terminal).
	msg.updateState(NewRelayEvent(EventMessagingState, map[string]any{
		"state": MessageStateDelivered,
	}))
	if msg.State() != MessageStateDelivered {
		t.Errorf("State = %q, want %q", msg.State(), MessageStateDelivered)
	}
	if !msg.IsDone() {
		t.Error("delivered is terminal, should be done")
	}
}

func TestMessage_Wait(t *testing.T) {
	msg := newMessage("msg-1", DirectionOutbound, "+1", "+2", "test")

	go func() {
		time.Sleep(10 * time.Millisecond)
		msg.updateState(NewRelayEvent(EventMessagingState, map[string]any{
			"state": MessageStateSent,
		}))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := msg.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait error: %v", err)
	}
	if result == nil {
		t.Fatal("Wait returned nil result")
	}
}

func TestMessage_OnHandler(t *testing.T) {
	msg := newMessage("msg-1", DirectionOutbound, "+1", "+2", "test")

	var received *RelayEvent
	msg.On(func(e *RelayEvent) {
		received = e
	})

	event := NewRelayEvent(EventMessagingState, map[string]any{
		"state": MessageStateInitiated,
	})
	msg.updateState(event)

	if received == nil {
		t.Fatal("On handler was not called")
	}
}

func TestMessage_FailedState(t *testing.T) {
	msg := newMessage("msg-1", DirectionOutbound, "+1", "+2", "test")

	msg.updateState(NewRelayEvent(EventMessagingState, map[string]any{
		"state":  MessageStateFailed,
		"reason": "invalid_number",
	}))

	if msg.State() != MessageStateFailed {
		t.Errorf("State = %q, want %q", msg.State(), MessageStateFailed)
	}
	if msg.Reason() != "invalid_number" {
		t.Errorf("Reason = %q, want %q", msg.Reason(), "invalid_number")
	}
	if !msg.IsDone() {
		t.Error("failed is terminal, should be done")
	}
}

// ---------------------------------------------------------------------------
// Constants defined correctly
// ---------------------------------------------------------------------------

func TestConstants_CallStates(t *testing.T) {
	states := []string{
		CallStateCreated, CallStateRinging, CallStateAnswered,
		CallStateEnding, CallStateEnded,
	}
	expected := []string{"created", "ringing", "answered", "ending", "ended"}
	for i, s := range states {
		if s != expected[i] {
			t.Errorf("call state %d = %q, want %q", i, s, expected[i])
		}
	}
}

func TestConstants_EndReasons(t *testing.T) {
	reasons := []string{
		EndReasonHangup, EndReasonCancel, EndReasonBusy,
		EndReasonNoAnswer, EndReasonDecline, EndReasonError,
	}
	expected := []string{"hangup", "cancel", "busy", "noAnswer", "decline", "error"}
	for i, r := range reasons {
		if r != expected[i] {
			t.Errorf("end reason %d = %q, want %q", i, r, expected[i])
		}
	}
}

func TestConstants_MessageStates(t *testing.T) {
	states := []string{
		MessageStateQueued, MessageStateInitiated, MessageStateSent,
		MessageStateDelivered, MessageStateUndelivered, MessageStateFailed,
		MessageStateReceived,
	}
	expected := []string{
		"queued", "initiated", "sent", "delivered",
		"undelivered", "failed", "received",
	}
	for i, s := range states {
		if s != expected[i] {
			t.Errorf("message state %d = %q, want %q", i, s, expected[i])
		}
	}
}

func TestConstants_EventTypes(t *testing.T) {
	// Verify all 22 calling events plus 2 messaging events.
	callingEvents := []string{
		EventCallingCallState, EventCallingCallReceive,
		EventCallingCallPlay, EventCallingCallRecord,
		EventCallingCallCollect, EventCallingCallConnect,
		EventCallingCallDetect, EventCallingCallFax,
		EventCallingCallTap, EventCallingCallStream,
		EventCallingCallSendDigits, EventCallingCallDial,
		EventCallingCallRefer, EventCallingCallDenoise,
		EventCallingCallPay, EventCallingCallQueue,
		EventCallingCallEcho, EventCallingCallTranscribe,
		EventCallingCallHold, EventCallingCallConference,
		EventCallingCallError, EventCallingCallAI,
	}
	if len(callingEvents) != 22 {
		t.Errorf("expected 22 calling events, got %d", len(callingEvents))
	}
	for _, e := range callingEvents {
		if e == "" {
			t.Error("found empty calling event constant")
		}
		if !contains(e, "calling.") {
			t.Errorf("calling event %q should start with 'calling.'", e)
		}
	}

	// Wire-format alignment with signalwire-python: most calling events use
	// the "calling.call.<verb>" prefix, but EVENT_CALLING_ERROR and
	// EVENT_CONFERENCE in Python use bare "calling.error" / "calling.conference"
	// (relay/constants.py:69-70). The constants below must match what the
	// SignalWire server emits — ParseEvent routes on the literal value.
	if EventCallingCallError != "calling.error" {
		t.Errorf("EventCallingCallError = %q, want %q (matches Python EVENT_CALLING_ERROR)", EventCallingCallError, "calling.error")
	}
	if EventCallingCallConference != "calling.conference" {
		t.Errorf("EventCallingCallConference = %q, want %q (matches Python EVENT_CONFERENCE)", EventCallingCallConference, "calling.conference")
	}

	messagingEvents := []string{EventMessagingReceive, EventMessagingState}
	if len(messagingEvents) != 2 {
		t.Errorf("expected 2 messaging events, got %d", len(messagingEvents))
	}
}

func TestConstants_ProtocolVersion(t *testing.T) {
	if ProtocolVersionMajor != 2 {
		t.Errorf("ProtocolVersionMajor = %d, want 2", ProtocolVersionMajor)
	}
	if ProtocolVersionMinor != 0 {
		t.Errorf("ProtocolVersionMinor = %d, want 0", ProtocolVersionMinor)
	}
	if ProtocolVersionRevision != 0 {
		t.Errorf("ProtocolVersionRevision = %d, want 0", ProtocolVersionRevision)
	}
}

func TestConstants_Directions(t *testing.T) {
	if DirectionInbound != "inbound" {
		t.Errorf("DirectionInbound = %q, want %q", DirectionInbound, "inbound")
	}
	if DirectionOutbound != "outbound" {
		t.Errorf("DirectionOutbound = %q, want %q", DirectionOutbound, "outbound")
	}
}

// ---------------------------------------------------------------------------
// Client internal event routing (unit-testable without WebSocket)
// ---------------------------------------------------------------------------

func TestClient_HandleCallingEvent_InboundCall(t *testing.T) {
	c := NewRelayClient()

	var receivedCall *Call
	var wg sync.WaitGroup
	wg.Add(1)
	c.OnCall(func(call *Call) {
		receivedCall = call
		wg.Done()
	})

	c.handleCallingEvent(EventCallingCallReceive, map[string]any{
		"call_id":    "call-inbound-1",
		"node_id":    "node-1",
		"tag":        "tag-recv-1",
		"call_state": CallStateRinging,
		"context":    "office",
		"device": map[string]any{
			"type": "phone",
		},
	})

	wg.Wait()

	if receivedCall == nil {
		t.Fatal("OnCall handler not called")
	}
	if receivedCall.CallID() != "call-inbound-1" {
		t.Errorf("CallID = %q, want %q", receivedCall.CallID(), "call-inbound-1")
	}

	// Verify call is stored.
	c.mu.RLock()
	stored, ok := c.calls["call-inbound-1"]
	c.mu.RUnlock()
	if !ok {
		t.Fatal("call not stored in client.calls")
	}
	if stored != receivedCall {
		t.Error("stored call does not match received call")
	}
}

func TestClient_HandleCallingEvent_EndedCallCleanup(t *testing.T) {
	c := NewRelayClient()
	call := newCall(c, "call-cleanup", "node-1", "tag-1")
	c.mu.Lock()
	c.calls["call-cleanup"] = call
	c.mu.Unlock()

	c.handleCallingEvent(EventCallingCallState, map[string]any{
		"call_id":    "call-cleanup",
		"call_state": CallStateEnded,
		"end_reason": EndReasonHangup,
	})

	c.mu.RLock()
	_, exists := c.calls["call-cleanup"]
	c.mu.RUnlock()
	if exists {
		t.Error("ended call should be cleaned up from client.calls")
	}
}

func TestClient_HandleCallingEvent_ActionResolve(t *testing.T) {
	c := NewRelayClient()
	call := newCall(c, "call-act", "node-1", "tag-1")
	c.mu.Lock()
	c.calls["call-act"] = call
	c.mu.Unlock()

	a := newAction(call, "ctrl-play-1")
	call.registerAction(a)

	c.handleCallingEvent(EventCallingCallPlay, map[string]any{
		"call_id":    "call-act",
		"control_id": "ctrl-play-1",
		"state":      "finished",
	})

	if !a.IsDone() {
		t.Error("action should be resolved by event")
	}
}

func TestClient_HandleMessagingEvent_Receive(t *testing.T) {
	c := NewRelayClient()

	var receivedMsg *Message
	var wg sync.WaitGroup
	wg.Add(1)
	c.OnMessage(func(msg *Message) {
		receivedMsg = msg
		wg.Done()
	})

	c.handleMessagingEvent(EventMessagingReceive, map[string]any{
		"message_id":  "msg-recv-1",
		"direction":   "inbound",
		"from_number": "+15551234567",
		"to_number":   "+15559876543",
		"body":        "Hi there",
		"segments":    1.0,
	})

	wg.Wait()

	if receivedMsg == nil {
		t.Fatal("OnMessage handler not called")
	}
	if receivedMsg.MessageID() != "msg-recv-1" {
		t.Errorf("MessageID = %q, want %q", receivedMsg.MessageID(), "msg-recv-1")
	}
	if receivedMsg.Body() != "Hi there" {
		t.Errorf("Body = %q, want %q", receivedMsg.Body(), "Hi there")
	}
}

func TestClient_HandleMessagingEvent_State(t *testing.T) {
	c := NewRelayClient()
	msg := newMessage("msg-state-1", DirectionOutbound, "+1", "+2", "test")
	c.mu.Lock()
	c.messages["msg-state-1"] = msg
	c.mu.Unlock()

	c.handleMessagingEvent(EventMessagingState, map[string]any{
		"message_id": "msg-state-1",
		"state":      MessageStateDelivered,
	})

	if msg.State() != MessageStateDelivered {
		t.Errorf("State = %q, want %q", msg.State(), MessageStateDelivered)
	}
	if !msg.IsDone() {
		t.Error("delivered is terminal, should be done")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
