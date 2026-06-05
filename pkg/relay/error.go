package relay

import (
	"errors"
	"fmt"
)

// Package-level sentinel errors for the conditions the relay client returns.
//
// These are Go-idiomatic, errors.Is-able markers (a Go-port addition — the
// Python reference uses RelayError + bare exceptions and has no equivalent
// sentinel set). Every code path that produces one of these conditions wraps
// the sentinel with %w, so callers can branch with errors.Is rather than
// scraping error strings:
//
//	if _, err := c.Dial(devices); errors.Is(err, relay.ErrDialTimeout) {
//	    // retry / give up on the dial specifically
//	}
//
// They are documented in PORT_ADDITIONS.md.
var (
	// ErrNotConnected is returned (wrapped) when an operation requires a live
	// WebSocket connection but none has been established (or it has been torn
	// down by Stop). Mirrors the "no websocket connection" failure mode.
	ErrNotConnected = errors.New("relay: not connected")

	// ErrDialTimeout is returned (wrapped) when Dial does not receive the
	// answering calling.call.dial event before its dial-timeout elapses.
	ErrDialTimeout = errors.New("relay: dial timed out")

	// ErrDialFailed is returned (wrapped) when the server reports the dial
	// reached a terminal "failed" dial_state (no device answered).
	ErrDialFailed = errors.New("relay: dial failed")

	// ErrExecuteTimeout is returned (wrapped) when a JSON-RPC request issued
	// via execute/Execute does not receive a response within its deadline.
	ErrExecuteTimeout = errors.New("relay: execute timed out")
)

// RelayError is a typed error returned by the RELAY protocol client.
// It carries the numeric code and message from the server so callers can
// programmatically inspect failures via errors.As.
//
// The error format matches Python's RelayError.__str__:
// "RELAY error {code}: {message}".
//
// A RelayError may also carry a wrapped sentinel (one of the Err* values
// above) so the same value satisfies BOTH errors.As(&RelayError) and
// errors.Is(err, ErrDialTimeout): the typed error preserves the server
// code/message, and the sentinel gives callers a stable, string-free branch.
type RelayError struct {
	Code    int
	Message string
	wrapped error // optional sentinel for errors.Is; nil for plain server errors
}

// Error implements the built-in error interface.
// Format matches Python's RelayError.__str__: "RELAY error {code}: {message}".
func (e *RelayError) Error() string {
	return fmt.Sprintf("RELAY error %d: %s", e.Code, e.Message)
}

// Unwrap exposes the optional wrapped sentinel so errors.Is can match it.
// Returns nil for a plain server-issued RelayError (no sentinel attached).
func (e *RelayError) Unwrap() error {
	return e.wrapped
}

// NewRelayError constructs a RelayError with the given code and message.
func NewRelayError(code int, message string) *RelayError {
	return &RelayError{Code: code, Message: message}
}

// newRelayErrorWrapping constructs a RelayError that also wraps a sentinel,
// so it matches errors.Is(err, sentinel) while keeping the "RELAY error N: …"
// string and the *RelayError type intact for existing errors.As callers.
func newRelayErrorWrapping(code int, message string, sentinel error) *RelayError {
	return &RelayError{Code: code, Message: message, wrapped: sentinel}
}
