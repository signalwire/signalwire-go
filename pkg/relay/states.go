package relay

// This file adds typed, defined-string kinds for the three RELAY lifecycle
// vocabularies — CallState, DialState, MessageState — ALONGSIDE the existing
// bare-string consts in constants.go and the bare-string accessors
// (Call.State, Message.State, DialEvent.DialState). It is purely additive:
// the string consts and string accessors stay canonical for parity with the
// Python reference (relay/constants.py uses bare str), and every typed value
// is a string subtype whose underlying value is byte-identical to the wire
// token, so nothing changes on the wire.
//
// These three vocabularies mirror values EMITTED BY THE SERVER, which can grow
// over time (a new lifecycle phase, a new delivery state). They are therefore
// modeled as defined-string kinds with named consts plus IsKnown()/IsTerminal()
// predicates rather than an exhaustive closed enum: an unrecognized
// server-emitted value flows through as a CallState/DialState/MessageState
// whose .IsKnown() is false, instead of being rejected — adding a value is not
// a breaking change. The three vocabularies are deliberately distinct Go types
// so they can never be conflated (CallState != DialState != MessageState):
// passing a DialState where a CallState is expected fails to compile.
//
// Grounding:
//   - CallState  — Python relay/constants.py CALL_STATES (created..ended).
//   - DialState  — Python relay/client.py:950 dial_state docstring
//     (dialing | answered | failed); :1006 notes "dialing" is progress.
//   - MessageState — Python relay/constants.py MESSAGE_STATE_* +
//     MESSAGE_TERMINAL_STATES (delivered, undelivered, failed).

// ---------------------------------------------------------------------------
// CallState
// ---------------------------------------------------------------------------

// CallState is the lifecycle state of a Call as a typed defined-string kind.
// The underlying string is the exact wire token (e.g. "answered"), so a
// CallState is interchangeable with the bare string the reference uses.
//
// Server-emitted and growable: prefer IsKnown()/IsTerminal() over an
// exhaustive switch so an unrecognized future state does not break callers.
type CallState string

// Call lifecycle states. These equal the bare-string CallState* consts in
// constants.go (created < ringing < answered < ending < ended) — the typed
// kind is an alternative spelling, not a replacement.
const (
	CallCreated  CallState = CallState(CallStateCreated)
	CallRinging  CallState = CallState(CallStateRinging)
	CallAnswered CallState = CallState(CallStateAnswered)
	CallEnding   CallState = CallState(CallStateEnding)
	CallEnded    CallState = CallState(CallStateEnded)
)

// String returns the wire token (identical to the underlying string).
func (s CallState) String() string { return string(s) }

// IsKnown reports whether s is one of the documented call lifecycle states.
// A false result means the server emitted a state this SDK build doesn't name
// yet — the value is still carried, just not recognized.
func (s CallState) IsKnown() bool {
	switch s {
	case CallCreated, CallRinging, CallAnswered, CallEnding, CallEnded:
		return true
	}
	return false
}

// IsTerminal reports whether s is a terminal call state (no further
// transitions expected). For a call the single terminal state is "ended",
// matching Python's CALL_STATE_ENDED gate in relay/call.py (the _ended future
// resolves only on "ended").
func (s CallState) IsTerminal() bool {
	return s == CallEnded
}

// ---------------------------------------------------------------------------
// DialState
// ---------------------------------------------------------------------------

// DialState is the outcome state of a calling.dial operation as a typed
// defined-string kind, read from the wire "dial_state" field. It is a SEPARATE
// vocabulary from CallState (and from the connect/bridge states): a dial
// progresses dialing -> answered (winner found) or dialing -> failed (no device
// answered). Grounded in Python relay/client.py:950.
//
// Distinct Go type from CallState/MessageState so the three can never be mixed.
type DialState string

// Dial outcome states. "dialing" is a non-terminal progress state; "answered"
// and "failed" are the two terminal outcomes that resolve/reject the dial.
const (
	DialDialing  DialState = "dialing"
	DialAnswered DialState = "answered"
	DialFailed   DialState = "failed"
)

// String returns the wire token (identical to the underlying string).
func (s DialState) String() string { return string(s) }

// IsKnown reports whether s is one of the documented dial outcome states.
func (s DialState) IsKnown() bool {
	switch s {
	case DialDialing, DialAnswered, DialFailed:
		return true
	}
	return false
}

// IsTerminal reports whether s is a terminal dial outcome — "answered" (a
// winning call was found) or "failed" (no device answered). "dialing" is
// progress and is NOT terminal. Mirrors Python's _handle_dial_event, which
// resolves the dial future on "answered", rejects it on "failed", and treats
// "dialing" as a progress event that doesn't settle the future
// (relay/client.py:976-1006).
func (s DialState) IsTerminal() bool {
	return s == DialAnswered || s == DialFailed
}

// ---------------------------------------------------------------------------
// MessageState
// ---------------------------------------------------------------------------

// MessageState is the delivery lifecycle state of a Message as a typed
// defined-string kind. The underlying string is the exact wire token (the
// "message_state" field), so it is interchangeable with the bare string the
// reference uses.
//
// Server-emitted and growable: prefer IsKnown()/IsTerminal().
// A SEPARATE vocabulary from CallState/DialState — distinct Go type.
type MessageState string

// Message delivery states. These equal the bare-string MessageState* consts in
// constants.go.
const (
	MsgQueued      MessageState = MessageState(MessageStateQueued)
	MsgInitiated   MessageState = MessageState(MessageStateInitiated)
	MsgSent        MessageState = MessageState(MessageStateSent)
	MsgDelivered   MessageState = MessageState(MessageStateDelivered)
	MsgUndelivered MessageState = MessageState(MessageStateUndelivered)
	MsgFailed      MessageState = MessageState(MessageStateFailed)
	MsgReceived    MessageState = MessageState(MessageStateReceived)
)

// String returns the wire token (identical to the underlying string).
func (s MessageState) String() string { return string(s) }

// IsKnown reports whether s is one of the documented message delivery states.
func (s MessageState) IsKnown() bool {
	switch s {
	case MsgQueued, MsgInitiated, MsgSent, MsgDelivered,
		MsgUndelivered, MsgFailed, MsgReceived:
		return true
	}
	return false
}

// IsTerminal reports whether s is a terminal OUTBOUND delivery state — one of
// delivered, undelivered, failed — matching Python's MESSAGE_TERMINAL_STATES
// (relay/constants.py). These are the states on which an outbound send settles.
//
// Note: the inbound terminal state "received" is intentionally NOT counted
// here, to mirror Python exactly (received is excluded from
// MESSAGE_TERMINAL_STATES — the inbound flow doesn't await). The internal
// helper isTerminalMessageState (message.go) additionally treats "received" as
// terminal so a Wait() on an inbound message returns immediately; that is a
// separate, behavior-only concern and is deliberately not folded into this
// grounded predicate.
func (s MessageState) IsTerminal() bool {
	switch s {
	case MsgDelivered, MsgUndelivered, MsgFailed:
		return true
	}
	return false
}
