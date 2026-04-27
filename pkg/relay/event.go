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
	// CallID is the call identifier, populated from the "call_id" wire key.
	// Python base class always carries this field.
	CallID string
	// Timestamp is the event timestamp (float for subsecond precision),
	// populated from the "timestamp" wire key.
	Timestamp float64
}

// NewRelayEvent creates a new RelayEvent from the given type and params.
func NewRelayEvent(eventType string, params map[string]any) *RelayEvent {
	if params == nil {
		params = make(map[string]any)
	}
	e := &RelayEvent{
		EventType: eventType,
		Params:    params,
	}
	e.CallID = e.GetString("call_id")
	e.Timestamp = e.GetFloat64("timestamp")
	return e
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

// GetFloat64 returns the float64 value for a key in params, or 0.0 if missing/wrong type.
// This preserves subsecond precision for duration and timestamp fields.
func (e *RelayEvent) GetFloat64(key string) float64 {
	if e.Params == nil {
		return 0.0
	}
	v, ok := e.Params[key]
	if !ok {
		return 0.0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	default:
		return 0.0
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

// GetBoolPtr returns a *bool for a key in params, or nil if the key is absent.
// This matches Python's Optional[bool] = None semantics.
func (e *RelayEvent) GetBoolPtr(key string) *bool {
	if e.Params == nil {
		return nil
	}
	v, ok := e.Params[key]
	if !ok {
		return nil
	}
	var result bool
	switch b := v.(type) {
	case bool:
		result = b
	case string:
		result = b == "true" || b == "1"
	case int:
		result = b != 0
	case float64:
		result = b != 0
	default:
		return nil
	}
	return &result
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

// GetStringSlice returns a []string for a key in params whose wire value is []any.
// Returns nil if absent or wrong type. Matches Python list[str] field behavior.
func (e *RelayEvent) GetStringSlice(key string) []string {
	if e.Params == nil {
		return nil
	}
	v, ok := e.Params[key]
	if !ok {
		return nil
	}
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Specific event types constructed from raw params
// ---------------------------------------------------------------------------

// CallStateEvent represents a calling.call.state event.
type CallStateEvent struct {
	*RelayEvent
	CallState string
	EndReason string
	Direction string
	Device    map[string]any
	CallID    string
	NodeID    string
	Tag       string
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
	Direction string
	Device    map[string]any
	Context   string
	Tag       string
	CallID    string
	NodeID    string
	ProjectID string
	SegmentID string
}

// NewCallReceiveEvent constructs a CallReceiveEvent from raw params.
func NewCallReceiveEvent(params map[string]any) *CallReceiveEvent {
	e := &CallReceiveEvent{
		RelayEvent: NewRelayEvent(EventCallingCallReceive, params),
	}
	e.CallState = e.GetString("call_state")
	e.Direction = e.GetString("direction")
	e.Device = e.GetMap("device")
	// SIP-originated receive events carry routing under "protocol" instead of
	// "context". Mirrors Python relay/event.py CallReceiveEvent.from_payload:
	//   context=p.get("context", p.get("protocol", ""))
	// Use a key-presence check (not an empty-string check) so an explicitly
	// empty "context" wins over a present "protocol", matching Python's
	// dict.get default semantics.
	if _, ok := e.Params["context"]; ok {
		e.Context = e.GetString("context")
	} else {
		e.Context = e.GetString("protocol")
	}
	e.Tag = e.GetString("tag")
	e.CallID = e.GetString("call_id")
	e.NodeID = e.GetString("node_id")
	e.ProjectID = e.GetString("project_id")
	e.SegmentID = e.GetString("segment_id")
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
	// Duration is float64 (matching Python's float) to preserve subsecond precision.
	Duration float64
	Size     int
	// Record is the raw nested record dict from the wire payload, matching Python's record field.
	Record map[string]any
}

// NewRecordEvent constructs a RecordEvent from raw params.
// URL, Duration, and Size are extracted from the nested "record" dict first,
// falling back to top-level params — matching Python's from_payload behavior.
func NewRecordEvent(params map[string]any) *RecordEvent {
	e := &RecordEvent{
		RelayEvent: NewRelayEvent(EventCallingCallRecord, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	e.Record = e.GetMap("record")

	// Mirror Python: rec.get("url", p.get("url", "")), etc.
	if e.Record != nil {
		if u, ok := e.Record["url"].(string); ok && u != "" {
			e.URL = u
		} else {
			e.URL = e.GetString("url")
		}
		if d, ok := e.Record["duration"].(float64); ok {
			e.Duration = d
		} else {
			e.Duration = e.GetFloat64("duration")
		}
		if s, ok := e.Record["size"].(float64); ok {
			e.Size = int(s)
		} else {
			e.Size = e.GetInt("size")
		}
	} else {
		e.URL = e.GetString("url")
		e.Duration = e.GetFloat64("duration")
		e.Size = e.GetInt("size")
	}
	return e
}

// CollectEvent represents a calling.call.collect event.
type CollectEvent struct {
	*RelayEvent
	ControlID string
	State     string
	Result    map[string]any
	// Final is a *bool matching Python's Optional[bool] = None semantics.
	Final *bool
}

// NewCollectEvent constructs a CollectEvent from raw params.
func NewCollectEvent(params map[string]any) *CollectEvent {
	e := &CollectEvent{
		RelayEvent: NewRelayEvent(EventCallingCallCollect, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	e.Result = e.GetMap("result")
	e.Final = e.GetBoolPtr("final")
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
	State     string
	Tap       map[string]any
	// Device is the tap device dict, matching Python's device field.
	Device map[string]any
}

// NewTapEvent constructs a TapEvent from raw params.
func NewTapEvent(params map[string]any) *TapEvent {
	e := &TapEvent{
		RelayEvent: NewRelayEvent(EventCallingCallTap, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	e.Tap = e.GetMap("tap")
	e.Device = e.GetMap("device")
	return e
}

// StreamEvent represents a calling.call.stream event.
type StreamEvent struct {
	*RelayEvent
	ControlID string
	State     string
	// URL is the stream URL, matching Python's url field.
	URL string
	// Name is the stream name, matching Python's name field.
	Name string
}

// NewStreamEvent constructs a StreamEvent from raw params.
func NewStreamEvent(params map[string]any) *StreamEvent {
	e := &StreamEvent{
		RelayEvent: NewRelayEvent(EventCallingCallStream, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	e.URL = e.GetString("url")
	e.Name = e.GetString("name")
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
	// DialState reads wire key "dial_state" matching Python's dial_state field.
	// (Replaces the previous State field which incorrectly read "state".)
	DialState string
	// Call is the nested call dict, matching Python's call field.
	Call map[string]any
}

// NewDialEvent constructs a DialEvent from raw params.
func NewDialEvent(params map[string]any) *DialEvent {
	e := &DialEvent{
		RelayEvent: NewRelayEvent(EventCallingCallDial, params),
	}
	e.Tag = e.GetString("tag")
	e.CallID = e.GetString("call_id")
	e.NodeID = e.GetString("node_id")
	e.DialState = e.GetString("dial_state")
	e.Call = e.GetMap("call")
	return e
}

// ReferEvent represents a calling.call.refer event.
type ReferEvent struct {
	*RelayEvent
	ControlID              string
	State                  string
	SIPReferTo             string
	SIPReferResponseCode   string
	SIPNotifyResponseCode  string
}

// NewReferEvent constructs a ReferEvent from raw params.
func NewReferEvent(params map[string]any) *ReferEvent {
	e := &ReferEvent{
		RelayEvent: NewRelayEvent(EventCallingCallRefer, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	e.SIPReferTo = e.GetString("sip_refer_to")
	e.SIPReferResponseCode = e.GetString("sip_refer_response_code")
	e.SIPNotifyResponseCode = e.GetString("sip_notify_response_code")
	return e
}

// DenoiseEvent represents a calling.call.denoise event.
type DenoiseEvent struct {
	*RelayEvent
	ControlID string
	State     string
	// Denoised matches Python's denoised bool field.
	Denoised bool
}

// NewDenoiseEvent constructs a DenoiseEvent from raw params.
func NewDenoiseEvent(params map[string]any) *DenoiseEvent {
	e := &DenoiseEvent{
		RelayEvent: NewRelayEvent(EventCallingCallDenoise, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	e.Denoised = e.GetBool("denoised")
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
	// Status reads wire key "status" matching Python's status field.
	// (Replaces the previous State field which incorrectly read "state".)
	Status string
	// QueueName reads wire key "name" matching Python's queue_name = p.get("name", "").
	// (Previously read "queue_name" which was wrong.)
	QueueName string
	// QueueID reads wire key "id" matching Python's queue_id = p.get("id", "").
	QueueID  string
	Position int
	Size     int
}

// NewQueueEvent constructs a QueueEvent from raw params.
func NewQueueEvent(params map[string]any) *QueueEvent {
	e := &QueueEvent{
		RelayEvent: NewRelayEvent(EventCallingCallQueue, params),
	}
	e.ControlID = e.GetString("control_id")
	e.Status = e.GetString("status")
	e.QueueName = e.GetString("name")
	e.QueueID = e.GetString("id")
	e.Position = e.GetInt("position")
	e.Size = e.GetInt("size")
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
	// URL is the transcription recording URL, matching Python's url field.
	URL string
	// RecordingID is the recording identifier, matching Python's recording_id field.
	RecordingID string
	// Duration is float64 for subsecond precision, matching Python's duration: float field.
	Duration float64
	// Size is the recording size in bytes, matching Python's size field.
	Size int
}

// NewTranscribeEvent constructs a TranscribeEvent from raw params.
func NewTranscribeEvent(params map[string]any) *TranscribeEvent {
	e := &TranscribeEvent{
		RelayEvent: NewRelayEvent(EventCallingCallTranscribe, params),
	}
	e.ControlID = e.GetString("control_id")
	e.State = e.GetString("state")
	e.Text = e.GetString("text")
	e.URL = e.GetString("url")
	e.RecordingID = e.GetString("recording_id")
	e.Duration = e.GetFloat64("duration")
	e.Size = e.GetInt("size")
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
	ConferenceID string
	Name         string
	// Status reads wire key "status" matching Python's status field.
	// (Replaces the previous State field which incorrectly read "state".)
	Status string
}

// NewConferenceEvent constructs a ConferenceEvent from raw params.
func NewConferenceEvent(params map[string]any) *ConferenceEvent {
	e := &ConferenceEvent{
		RelayEvent: NewRelayEvent(EventCallingCallConference, params),
	}
	e.ControlID = e.GetString("control_id")
	e.ConferenceID = e.GetString("conference_id")
	e.Name = e.GetString("name")
	e.Status = e.GetString("status")
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
	MessageID    string
	Context      string
	Direction    string
	FromNumber   string
	ToNumber     string
	Body         string
	Media        []string
	Segments     int
	Tags         []string
	// MessageState matches Python's message_state field.
	MessageState string
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
	e.MessageState = e.GetString("message_state")

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
	// MessageState reads wire key "message_state" matching Python's message_state field.
	// (Replaces the previous State field which incorrectly read "state".)
	MessageState string
	Reason       string
	Direction    string
	FromNumber   string
	ToNumber     string
	// Context matches Python's context field.
	Context string
	// Body matches Python's body field.
	Body string
	// Media matches Python's media: list[str] field.
	Media []string
	// Segments matches Python's segments: int field.
	Segments int
	// Tags matches Python's tags: list[str] field.
	Tags []string
}

// NewMessageStateEvent constructs a MessageStateEvent from raw params.
func NewMessageStateEvent(params map[string]any) *MessageStateEvent {
	e := &MessageStateEvent{
		RelayEvent: NewRelayEvent(EventMessagingState, params),
	}
	e.MessageID = e.GetString("message_id")
	e.MessageState = e.GetString("message_state")
	e.Reason = e.GetString("reason")
	e.Direction = e.GetString("direction")
	e.FromNumber = e.GetString("from_number")
	e.ToNumber = e.GetString("to_number")
	e.Context = e.GetString("context")
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

// ParseEvent parses a raw signalwire event payload dict into a typed event
// object. It reads "event_type" from the top-level payload and "params" as
// the inner parameter map, then dispatches to the appropriate typed
// constructor. If the event_type is not recognised, a plain *RelayEvent is
// returned. Callers can type-assert or type-switch on the result to access
// the concrete event fields.
//
// This mirrors Python's relay.event.parse_event(payload).
func ParseEvent(payload map[string]any) any {
	eventType, _ := payload["event_type"].(string)
	var params map[string]any
	if p, ok := payload["params"].(map[string]any); ok {
		params = p
	} else {
		params = make(map[string]any)
	}

	switch eventType {
	case EventCallingCallState:
		return NewCallStateEvent(params)
	case EventCallingCallReceive:
		return NewCallReceiveEvent(params)
	case EventCallingCallPlay:
		return NewPlayEvent(params)
	case EventCallingCallRecord:
		return NewRecordEvent(params)
	case EventCallingCallCollect:
		return NewCollectEvent(params)
	case EventCallingCallConnect:
		return NewConnectEvent(params)
	case EventCallingCallDetect:
		return NewDetectEvent(params)
	case EventCallingCallFax:
		return NewFaxEvent(params)
	case EventCallingCallTap:
		return NewTapEvent(params)
	case EventCallingCallStream:
		return NewStreamEvent(params)
	case EventCallingCallSendDigits:
		return NewSendDigitsEvent(params)
	case EventCallingCallDial:
		return NewDialEvent(params)
	case EventCallingCallRefer:
		return NewReferEvent(params)
	case EventCallingCallDenoise:
		return NewDenoiseEvent(params)
	case EventCallingCallPay:
		return NewPayEvent(params)
	case EventCallingCallQueue:
		return NewQueueEvent(params)
	case EventCallingCallEcho:
		return NewEchoEvent(params)
	case EventCallingCallTranscribe:
		return NewTranscribeEvent(params)
	case EventCallingCallHold:
		return NewHoldEvent(params)
	case EventCallingCallConference:
		return NewConferenceEvent(params)
	case EventCallingCallError:
		return NewCallingErrorEvent(params)
	case EventCallingCallAI:
		return NewAIEvent(params)
	case EventMessagingReceive:
		return NewMessageReceiveEvent(params)
	case EventMessagingState:
		return NewMessageStateEvent(params)
	default:
		return NewRelayEvent(eventType, params)
	}
}
