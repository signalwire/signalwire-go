// Package relay implements real-time WebSocket call control over the
// SignalWire Blade protocol (JSON-RPC 2.0). It provides a RELAY client
// that manages WebSocket connections, authentication, and event routing
// for calling, messaging, and other real-time communication primitives.
package relay

// Protocol version for the SignalWire Blade protocol.
const (
	ProtocolVersionMajor    = 2
	ProtocolVersionMinor    = 0
	ProtocolVersionRevision = 0
)

// AgentString is the SDK identifier the connect frame announces to the
// RELAY server. Mirrors Python's AGENT_STRING at relay/constants.py:5,
// translated to identify this language's port. Sent as
// `params.agent` on signalwire.connect.
const AgentString = "signalwire-agents-go/1.0"

// Call states represent the lifecycle of a call.
const (
	CallStateCreated = "created"
	CallStateRinging = "ringing"
	CallStateAnswered = "answered"
	CallStateEnding  = "ending"
	CallStateEnded   = "ended"
)

// Call end reasons indicate why a call ended.
const (
	EndReasonHangup   = "hangup"
	EndReasonCancel   = "cancel"
	EndReasonBusy     = "busy"
	EndReasonNoAnswer = "noAnswer"
	EndReasonDecline  = "decline"
	EndReasonError    = "error"
)

// Message states represent the lifecycle of an SMS/MMS message.
const (
	MessageStateQueued      = "queued"
	MessageStateInitiated   = "initiated"
	MessageStateSent        = "sent"
	MessageStateDelivered   = "delivered"
	MessageStateUndelivered = "undelivered"
	MessageStateFailed      = "failed"
	MessageStateReceived    = "received"
)

// Event types for calling events.
const (
	EventCallingCallState      = "calling.call.state"
	EventCallingCallReceive    = "calling.call.receive"
	EventCallingCallPlay       = "calling.call.play"
	EventCallingCallRecord     = "calling.call.record"
	EventCallingCallCollect    = "calling.call.collect"
	EventCallingCallConnect    = "calling.call.connect"
	EventCallingCallDetect     = "calling.call.detect"
	EventCallingCallFax        = "calling.call.fax"
	EventCallingCallTap        = "calling.call.tap"
	EventCallingCallStream     = "calling.call.stream"
	EventCallingCallSendDigits = "calling.call.send_digits"
	EventCallingCallDial       = "calling.call.dial"
	EventCallingCallRefer      = "calling.call.refer"
	EventCallingCallDenoise    = "calling.call.denoise"
	EventCallingCallPay        = "calling.call.pay"
	EventCallingCallQueue      = "calling.call.queue"
	EventCallingCallEcho       = "calling.call.echo"
	EventCallingCallTranscribe = "calling.call.transcribe"
	EventCallingCallHold       = "calling.call.hold"
	EventCallingCallConference = "calling.conference"
	EventCallingCallError      = "calling.error"
	EventCallingCallAI         = "calling.call.ai"
)

// Event types for messaging events.
const (
	EventMessagingReceive = "messaging.receive"
	EventMessagingState   = "messaging.state"
)

// Authorization-state event used to refresh reconnect material. Mirrors
// Python's EVENT_AUTHORIZATION_STATE = "signalwire.authorization.state"
// (relay/constants.py:16).
const EventAuthorizationState = "signalwire.authorization.state"

// Blade/SignalWire internal method constants.
const (
	MethodSignalWireConnect = "signalwire.connect"
	MethodSignalWirePing    = "signalwire.ping"
	MethodSignalWireEvent   = "signalwire.event"
	MethodCalling           = "calling"
	MethodMessaging         = "messaging"
)

// Call directions.
const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"
)
