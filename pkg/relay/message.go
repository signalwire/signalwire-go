package relay

import (
	"context"
	"sync"
)

// Message represents an SMS/MMS message tracked through its lifecycle.
type Message struct {
	messageID  string
	context    string
	direction  string
	fromNumber string
	toNumber   string
	body       string
	media      []string
	segments   int
	state      string
	reason     string
	tags       []string

	done   chan struct{}
	result *RelayEvent
	mu     sync.Mutex

	completed   bool
	onCompleted func(*Message)

	eventHandlers []func(*RelayEvent)
}

// newMessage creates a new Message with the given identifiers.
func newMessage(messageID, direction, from, to, body string) *Message {
	return &Message{
		messageID:  messageID,
		direction:  direction,
		fromNumber: from,
		toNumber:   to,
		body:       body,
		done:       make(chan struct{}),
	}
}

// MessageID returns the unique message identifier.
func (m *Message) MessageID() string { return m.messageID }

// Context returns the RELAY context on which this message was received.
func (m *Message) Context() string { return m.context }

// Direction returns "inbound" or "outbound".
func (m *Message) Direction() string { return m.direction }

// FromNumber returns the sender number.
func (m *Message) FromNumber() string { return m.fromNumber }

// ToNumber returns the recipient number.
func (m *Message) ToNumber() string { return m.toNumber }

// Body returns the text body of the message.
func (m *Message) Body() string { return m.body }

// Media returns the list of media URLs attached to the message.
func (m *Message) Media() []string { return m.media }

// Segments returns the number of SMS segments.
func (m *Message) Segments() int { return m.segments }

// State returns the current message state.
func (m *Message) State() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// Reason returns the failure reason if the message failed.
func (m *Message) Reason() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reason
}

// Tags returns the tags associated with the message.
func (m *Message) Tags() []string { return m.tags }

// Result returns the terminal RelayEvent if the message has reached a terminal
// state, or nil if not yet done. This is the non-blocking equivalent of
// Python's Message.result property.
func (m *Message) Result() *RelayEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.completed {
		return m.result
	}
	return nil
}

// Wait blocks until the message reaches a terminal state or the context
// is cancelled. Returns the final event or the context error.
func (m *Message) Wait(ctx context.Context) (*RelayEvent, error) {
	select {
	case <-m.done:
		m.mu.Lock()
		defer m.mu.Unlock()
		return m.result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// IsDone returns true if the message has reached a terminal state.
func (m *Message) IsDone() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.completed
}

// On registers an event handler called when message state changes.
func (m *Message) On(handler func(*RelayEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventHandlers = append(m.eventHandlers, handler)
}

// updateState is called internally when a message state event is received.
func (m *Message) updateState(event *RelayEvent) {
	m.mu.Lock()
	newState := event.GetString("state")
	m.state = newState
	if r := event.GetString("reason"); r != "" {
		m.reason = r
	}
	handlers := make([]func(*RelayEvent), len(m.eventHandlers))
	copy(handlers, m.eventHandlers)

	terminal := isTerminalMessageState(newState)
	var onCompleted func(*Message)
	if terminal && !m.completed {
		m.completed = true
		m.result = event
		onCompleted = m.onCompleted
	}
	m.mu.Unlock()

	for _, h := range handlers {
		h(event)
	}

	if terminal {
		if onCompleted != nil {
			go onCompleted(m)
		}
		select {
		case <-m.done:
			// Already closed.
		default:
			close(m.done)
		}
	}
}

// isTerminalMessageState returns true for message states that represent
// a final state (no further transitions expected).
func isTerminalMessageState(state string) bool {
	switch state {
	case MessageStateSent, MessageStateDelivered,
		MessageStateUndelivered, MessageStateFailed,
		MessageStateReceived:
		return true
	}
	return false
}
