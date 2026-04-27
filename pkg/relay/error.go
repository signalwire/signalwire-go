package relay

import "fmt"

// RelayError is a typed error returned by the RELAY protocol client.
// It carries the numeric code and message from the server so callers can
// programmatically inspect failures via errors.As.
//
// The error format matches Python's RelayError.__str__:
// "RELAY error {code}: {message}".
type RelayError struct {
	Code    int
	Message string
}

// Error implements the built-in error interface.
// Format matches Python's RelayError.__str__: "RELAY error {code}: {message}".
func (e *RelayError) Error() string {
	return fmt.Sprintf("RELAY error %d: %s", e.Code, e.Message)
}

// NewRelayError constructs a RelayError with the given code and message.
func NewRelayError(code int, message string) *RelayError {
	return &RelayError{Code: code, Message: message}
}
