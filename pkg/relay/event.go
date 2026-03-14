package relay

import (
	"fmt"
	"strconv"
)

// RelayEvent is the base event type for all events received from the
// SignalWire RELAY service. It carries an event type string and a generic
// parameter map that can be queried via helper methods.
type RelayEvent struct {
	EventType string
	Params    map[string]any
}

// NewRelayEvent creates a new RelayEvent from the given type and params.
func NewRelayEvent(eventType string, params map[string]any) *RelayEvent {
	if params == nil {
		params = make(map[string]any)
	}
	return &RelayEvent{
		EventType: eventType,
		Params:    params,
	}
}

// GetString returns the string value for a key in params, or "" if missing/wrong type.
func (e *RelayEvent) GetString(key string) string {
	if e.Params == nil {
		return ""
	}
	v, ok := e.Params[key]
	if !ok {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case fmt.Stringer:
		return s.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetInt returns the integer value for a key in params, or 0 if missing/wrong type.
func (e *RelayEvent) GetInt(key string) int {
	if e.Params == nil {
		return 0
	}
	v, ok := e.Params[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	default:
		return 0
	}
}

// GetBool returns the boolean value for a key in params, or false if missing/wrong type.
func (e *RelayEvent) GetBool(key string) bool {
	if e.Params == nil {
		return false
	}
	v, ok := e.Params[key]
	if !ok {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return b == "true" || b == "1"
	case int:
		return b != 0
	case float64:
		return b != 0
	default:
		return false
	}
}

// GetMap returns the nested map for a key in params, or nil if missing/wrong type.
func (e *RelayEvent) GetMap(key string) map[string]any {
	if e.Params == nil {
		return nil
	}
	v, ok := e.Params[key]
	if !ok {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

// ---------------------------------------------------------------------------
// Specific event types constructed from raw params
// ---------------------------------------------------------------------------

// CallStateEvent represents a calling.call.state event.
type CallStateEvent struct {
	*RelayEvent
	CallState  string
	EndReason  string
	Direction  string
	Device     map[string]any
	CallID     string
	NodeID     string
	Tag        string
}

// NewCallStateEvent constructs a CallStateEvent from raw params.
func NewCallStateEvent(params map[string]any) *CallStateEvent {
	e := &CallStateEvent{
		RelayEvent: NewRelayEvent(EventCallingCallState, params),
	}
	e.CallState = e.GetString("call_state")
	e.EndReason = e.GetString("end_reason")
	e.Direction = e.GetString("direction")
	e.Device = e.GetMap("device")
	e.CallID = e.GetString("call_id")
	e.NodeID = e.GetString("node_id")
	e.Tag = e.GetString("tag")
	return e
}

// CallReceiveEvent represents a calling.call.receive event for inbound calls.
type CallReceiveEvent struct {
	*RelayEvent
	CallState string
	Device    map[string]any
	Context   string
	Tag       string
	CallID    string
	NodeID    string
}

// NewCallReceiveEvent constructs a CallReceiveEvent from raw params.
func NewCallReceiveEvent(params map[string]any) *CallReceiveEvent {
	e := &CallReceiveEvent{
		RelayEvent: NewRelayEvent(EventCallingCallReceive, params),
	}
	e.CallState = e.GetString("call_state")
	e.Device = e.GetMap("device")
	e.Context = e.GetString("context")
	e.Tag = e.GetString("tag")
	e.CallID = e.GetString("call_id")
	e.NodeID = e.GetString("node_id")
	return e
}

// PlayEvent represents a calling.call.play event.
type PlayEvent struct {
	*RelayEvent
	ControlID string
	State     string
}

// NewPlayEvent constructs a PlayEvent from raw params.
func NewPlayEvent(params map[string]any) *PlayEvent {
	e := &PlayEvent{
		RelayEvent: NewRelayEvent(EventCallingCallPlay, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	return e
}

// RecordEvent represents a calling.call.record event.
type RecordEvent struct {
	*RelayEvent
	ControlID string
	State     string
	URL       string
	Duration  int
	Size      int
}

// NewRecordEvent constructs a RecordEvent from raw params.
func NewRecordEvent(params map[string]any) *RecordEvent {
	e := &RecordEvent{
		RelayEvent: NewRelayEvent(EventCallingCallRecord, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	e.URL = e.GetString("url")
	e.Duration = e.GetInt("duration")
	e.Size = e.GetInt("size")
	return e
}

// CollectEvent represents a calling.call.collect event.
type CollectEvent struct {
	*RelayEvent
	ControlID string
	Result    map[string]any
}

// NewCollectEvent constructs a CollectEvent from raw params.
func NewCollectEvent(params map[string]any) *CollectEvent {
	e := &CollectEvent{
		RelayEvent: NewRelayEvent(EventCallingCallCollect, params),
	}
	e.ControlID = e.GetString("control_id")
	e.Result = e.GetMap("result")
	return e
}

// ConnectEvent represents a calling.call.connect event.
type ConnectEvent struct {
	*RelayEvent
	ConnectState string
	Peer         map[string]any
}

// NewConnectEvent constructs a ConnectEvent from raw params.
func NewConnectEvent(params map[string]any) *ConnectEvent {
	e := &ConnectEvent{
		RelayEvent: NewRelayEvent(EventCallingCallConnect, params),
	}
	e.ConnectState = e.GetString("connect_state")
	e.Peer = e.GetMap("peer")
	return e
}

// DetectEvent represents a calling.call.detect event.
type DetectEvent struct {
	*RelayEvent
	ControlID string
	Detect    map[string]any
}

// NewDetectEvent constructs a DetectEvent from raw params.
func NewDetectEvent(params map[string]any) *DetectEvent {
	e := &DetectEvent{
		RelayEvent: NewRelayEvent(EventCallingCallDetect, params),
	}
	e.ControlID = e.GetString("control_id")
	e.Detect = e.GetMap("detect")
	return e
}

// FaxEvent represents a calling.call.fax event.
type FaxEvent struct {
	*RelayEvent
	ControlID string
	Fax       map[string]any
}

// NewFaxEvent constructs a FaxEvent from raw params.
func NewFaxEvent(params map[string]any) *FaxEvent {
	e := &FaxEvent{
		RelayEvent: NewRelayEvent(EventCallingCallFax, params),
	}
	e.ControlID = e.GetString("control_id")
	e.Fax = e.GetMap("fax")
	return e
}

// TapEvent represents a calling.call.tap event.
type TapEvent struct {
	*RelayEvent
	ControlID string
	Tap       map[string]any
}

// NewTapEvent constructs a TapEvent from raw params.
func NewTapEvent(params map[string]any) *TapEvent {
	e := &TapEvent{
		RelayEvent: NewRelayEvent(EventCallingCallTap, params),
	}
	e.ControlID = e.GetString("control_id")
	e.Tap = e.GetMap("tap")
	return e
}

// StreamEvent represents a calling.call.stream event.
type StreamEvent struct {
	*RelayEvent
	ControlID string
	State     string
}

// NewStreamEvent constructs a StreamEvent from raw params.
func NewStreamEvent(params map[string]any) *StreamEvent {
	e := &StreamEvent{
		RelayEvent: NewRelayEvent(EventCallingCallStream, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	return e
}

// SendDigitsEvent represents a calling.call.send_digits event.
type SendDigitsEvent struct {
	*RelayEvent
	ControlID string
	State     string
}

// NewSendDigitsEvent constructs a SendDigitsEvent from raw params.
func NewSendDigitsEvent(params map[string]any) *SendDigitsEvent {
	e := &SendDigitsEvent{
		RelayEvent: NewRelayEvent(EventCallingCallSendDigits, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	return e
}

// DialEvent represents a calling.call.dial event.
type DialEvent struct {
	*RelayEvent
	Tag    string
	CallID string
	NodeID string
	State  string
}

// NewDialEvent constructs a DialEvent from raw params.
func NewDialEvent(params map[string]any) *DialEvent {
	e := &DialEvent{
		RelayEvent: NewRelayEvent(EventCallingCallDial, params),
	}
	e.Tag = e.GetString("tag")
	e.CallID = e.GetString("call_id")
	e.NodeID = e.GetString("node_id")
	e.State = e.GetString("state")
	return e
}

// ReferEvent represents a calling.call.refer event.
type ReferEvent struct {
	*RelayEvent
	ControlID string
	State     string
}

// NewReferEvent constructs a ReferEvent from raw params.
func NewReferEvent(params map[string]any) *ReferEvent {
	e := &ReferEvent{
		RelayEvent: NewRelayEvent(EventCallingCallRefer, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	return e
}

// DenoiseEvent represents a calling.call.denoise event.
type DenoiseEvent struct {
	*RelayEvent
	ControlID string
	State     string
}

// NewDenoiseEvent constructs a DenoiseEvent from raw params.
func NewDenoiseEvent(params map[string]any) *DenoiseEvent {
	e := &DenoiseEvent{
		RelayEvent: NewRelayEvent(EventCallingCallDenoise, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	return e
}

// PayEvent represents a calling.call.pay event.
type PayEvent struct {
	*RelayEvent
	ControlID string
	State     string
	Result    map[string]any
}

// NewPayEvent constructs a PayEvent from raw params.
func NewPayEvent(params map[string]any) *PayEvent {
	e := &PayEvent{
		RelayEvent: NewRelayEvent(EventCallingCallPay, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	e.Result = e.GetMap("result")
	return e
}

// QueueEvent represents a calling.call.queue event.
type QueueEvent struct {
	*RelayEvent
	ControlID string
	State     string
	QueueName string
}

// NewQueueEvent constructs a QueueEvent from raw params.
func NewQueueEvent(params map[string]any) *QueueEvent {
	e := &QueueEvent{
		RelayEvent: NewRelayEvent(EventCallingCallQueue, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	e.QueueName = e.GetString("queue_name")
	return e
}

// EchoEvent represents a calling.call.echo event.
type EchoEvent struct {
	*RelayEvent
	ControlID string
	State     string
}

// NewEchoEvent constructs an EchoEvent from raw params.
func NewEchoEvent(params map[string]any) *EchoEvent {
	e := &EchoEvent{
		RelayEvent: NewRelayEvent(EventCallingCallEcho, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	return e
}

// TranscribeEvent represents a calling.call.transcribe event.
type TranscribeEvent struct {
	*RelayEvent
	ControlID string
	State     string
	Text      string
}

// NewTranscribeEvent constructs a TranscribeEvent from raw params.
func NewTranscribeEvent(params map[string]any) *TranscribeEvent {
	e := &TranscribeEvent{
		RelayEvent: NewRelayEvent(EventCallingCallTranscribe, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	e.Text = e.GetString("text")
	return e
}

// HoldEvent represents a calling.call.hold event.
type HoldEvent struct {
	*RelayEvent
	ControlID string
	State     string
}

// NewHoldEvent constructs a HoldEvent from raw params.
func NewHoldEvent(params map[string]any) *HoldEvent {
	e := &HoldEvent{
		RelayEvent: NewRelayEvent(EventCallingCallHold, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	return e
}

// ConferenceEvent represents a calling.call.conference event.
type ConferenceEvent struct {
	*RelayEvent
	ControlID    string
	State        string
	ConferenceID string
}

// NewConferenceEvent constructs a ConferenceEvent from raw params.
func NewConferenceEvent(params map[string]any) *ConferenceEvent {
	e := &ConferenceEvent{
		RelayEvent: NewRelayEvent(EventCallingCallConference, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	e.ConferenceID = e.GetString("conference_id")
	return e
}

// CallingErrorEvent represents a calling.call.error event.
type CallingErrorEvent struct {
	*RelayEvent
	Code        string
	Message     string
	Description string
}

// NewCallingErrorEvent constructs a CallingErrorEvent from raw params.
func NewCallingErrorEvent(params map[string]any) *CallingErrorEvent {
	e := &CallingErrorEvent{
		RelayEvent: NewRelayEvent(EventCallingCallError, params),
	}
	e.Code = e.GetString("code")
	e.Message = e.GetString("message")
	e.Description = e.GetString("description")
	return e
}

// MessageReceiveEvent represents a messaging.receive event.
type MessageReceiveEvent struct {
	*RelayEvent
	MessageID   string
	Context     string
	Direction   string
	FromNumber  string
	ToNumber    string
	Body        string
	Media       []string
	Segments    int
	Tags        []string
}

// NewMessageReceiveEvent constructs a MessageReceiveEvent from raw params.
func NewMessageReceiveEvent(params map[string]any) *MessageReceiveEvent {
	e := &MessageReceiveEvent{
		RelayEvent: NewRelayEvent(EventMessagingReceive, params),
	}
	e.MessageID = e.GetString("message_id")
	e.Context = e.GetString("context")
	e.Direction = e.GetString("direction")
	e.FromNumber = e.GetString("from_number")
	e.ToNumber = e.GetString("to_number")
	e.Body = e.GetString("body")
	e.Segments = e.GetInt("segments")

	if mediaRaw, ok := params["media"]; ok {
		if mediaSlice, ok := mediaRaw.([]any); ok {
			for _, m := range mediaSlice {
				if s, ok := m.(string); ok {
					e.Media = append(e.Media, s)
				}
			}
		}
	}

	if tagsRaw, ok := params["tags"]; ok {
		if tagsSlice, ok := tagsRaw.([]any); ok {
			for _, t := range tagsSlice {
				if s, ok := t.(string); ok {
					e.Tags = append(e.Tags, s)
				}
			}
		}
	}

	return e
}

// MessageStateEvent represents a messaging.state event.
type MessageStateEvent struct {
	*RelayEvent
	MessageID  string
	State      string
	Reason     string
	Direction  string
	FromNumber string
	ToNumber   string
}

// NewMessageStateEvent constructs a MessageStateEvent from raw params.
func NewMessageStateEvent(params map[string]any) *MessageStateEvent {
	e := &MessageStateEvent{
		RelayEvent: NewRelayEvent(EventMessagingState, params),
	}
	e.MessageID = e.GetString("message_id")
	e.State = e.GetString("state")
	e.Reason = e.GetString("reason")
	e.Direction = e.GetString("direction")
	e.FromNumber = e.GetString("from_number")
	e.ToNumber = e.GetString("to_number")
	return e
}

// AIEvent represents a calling.call.ai event.
type AIEvent struct {
	*RelayEvent
	ControlID string
	State     string
	Result    map[string]any
}

// NewAIEvent constructs an AIEvent from raw params.
func NewAIEvent(params map[string]any) *AIEvent {
	e := &AIEvent{
		RelayEvent: NewRelayEvent(EventCallingCallAI, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	e.Result = e.GetMap("result")
	return e
}
