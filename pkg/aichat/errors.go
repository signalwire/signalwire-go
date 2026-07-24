// Copyright (c) 2026 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package aichat

import "fmt"

// AIChatError is the base error for AI Chat service failures. Every typed variant
// carries the JSON-RPC error Code (or 0 with HasCode false when the failure rode
// the success envelope, as with SummaryError) and the server Message.
//
// Go has no exception-class hierarchy, so the typed family is modelled as error
// values wrapping a common *AIChatError: a caller unwraps the whole family with
// errors.As(err, &target) where target is either **AIChatError (any AI-Chat
// failure) or a specific typed error (e.g. **ConversationNotFoundError). Branching
// on Code is also supported.
type AIChatError struct {
	// Code is the JSON-RPC error code. HasCode is false when the failure rode the
	// JSON-RPC success envelope (SummaryError), where no code exists.
	Code int
	// HasCode distinguishes a real code of 0 from "no code" (Go has no int|None).
	HasCode bool
	// Message is the server-provided error message (without the "[code]" prefix).
	Message string
}

// Error implements the error interface, mirroring the Python reference's
// "[code] message" formatting. When HasCode is false the code renders as <nil>,
// matching the reference's None.
func (e *AIChatError) Error() string {
	if e.HasCode {
		return fmt.Sprintf("[%d] %s", e.Code, e.Message)
	}
	return fmt.Sprintf("[<nil>] %s", e.Message)
}

// AuthenticationError is a missing/rejected identity (HTTP 401 / JSON-RPC -32009).
type AuthenticationError struct{ *AIChatError }

// ConversationNotFoundError means the conversation does not exist in this project
// (-32001).
type ConversationNotFoundError struct{ *AIChatError }

// RateLimitError means a project or conversation rate limit was hit
// (-32005 / -32006).
type RateLimitError struct{ *AIChatError }

// ChatInProgressError means another message is being processed for this
// conversation (-32007).
type ChatInProgressError struct{ *AIChatError }

// SummaryError means summary generation failed. summarize returns EXACTLY ONE of
// {summary} (success) or {error} (generation failed), and the failure rides the
// JSON-RPC success envelope — not an error object — so it never reaches the
// error-code mapping. Surfaced as its own type so a failed summary can't
// masquerade as an empty string. Code is 0 with HasCode false (no JSON-RPC code).
type SummaryError struct{ *AIChatError }

// newTypedError builds the typed error variant for a JSON-RPC error code, falling
// back to the base *AIChatError for an unmapped code. The returned value is an
// error whose concrete type errors.As can match (e.g. *ConversationNotFoundError),
// AND whose embedded *AIChatError errors.As can also reach via Unwrap.
func newTypedError(code int, message string) error {
	base := &AIChatError{Code: code, HasCode: true, Message: message}
	switch code {
	case -32001:
		return &ConversationNotFoundError{base}
	case -32005, -32006:
		return &RateLimitError{base}
	case -32007:
		return &ChatInProgressError{base}
	case -32009:
		return &AuthenticationError{base}
	default:
		return base
	}
}

// Unwrap on each typed variant returns the embedded *AIChatError, so
// errors.As(err, &(*AIChatError)) matches ANY variant in the family — the Go
// equivalent of catching the base class.
func (e *AuthenticationError) Unwrap() error       { return e.AIChatError }
func (e *ConversationNotFoundError) Unwrap() error { return e.AIChatError }
func (e *RateLimitError) Unwrap() error            { return e.AIChatError }
func (e *ChatInProgressError) Unwrap() error       { return e.AIChatError }
func (e *SummaryError) Unwrap() error              { return e.AIChatError }
