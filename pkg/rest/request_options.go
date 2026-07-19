// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package rest

import (
	"context"
	"strings"
)

// RequestOptions is the REST request-options envelope (plan 4.2): a single value
// object controlling per-request transport behavior — timeout, an
// idempotency-aware retry policy with exponential backoff, and cooperative
// cancellation. It mirrors the Python reference
// (signalwire.rest._request_options.RequestOptions).
//
// It is supplied at two levels:
//
//   - Client default: NewRestClientWithOptions(..., RequestOptions{...}) (or the
//     HTTPClient's WithRequestOptions seam) stores it on the HTTPClient and
//     applies it to every request.
//   - Per-request override: each ...WithOptions verb accepts an optional
//     *RequestOptions that SHALLOW-overrides the client default for that one call —
//     an unset (nil) field falls back to the client default, then the built-in
//     default.
//
// Every field is a pointer so "unset" (nil) is distinguishable from a deliberate
// zero value ("inherit" vs. "0 retries" / "0s timeout"). The Python reference uses
// None for the same reason; Go uses nil pointers.
//
// AbortSignal fidelity is per-port idiom. Go's cancellation primitive IS
// context.Context, and Go's REST surface is already ctx-first — so the request
// loop threads AbortSignal onto the outgoing HTTP request's context. A cancelled
// context is checked BEFORE each attempt (surfacing the typed transport error
// without a send) AND cuts an in-flight request (net/http observes the ctx). When
// AbortSignal is nil the request uses the caller's own ctx (the ...Context verbs)
// or context.Background() (the plain verbs), unchanged.
type RequestOptions struct {
	// Timeout is the max wall-clock duration per attempt (in seconds; a float to
	// match the reference's fractional-second timeouts). On exceed the request
	// raises the transport-error type (SignalWireRestError with Transport=true).
	// Built-in default 30.0s. nil => inherit.
	Timeout *float64

	// Retries is the number of RETRY attempts (total attempts = Retries + 1) on a
	// retryable failure. Built-in default 0 (opt-in resilience — the no-retry
	// behavior stays the default; a caller opts into retries). nil => inherit.
	Retries *int

	// RetryOnStatus is the set of HTTP statuses that trigger a retry for an
	// idempotent method. Built-in {429, 500, 502, 503, 504}. nil => inherit.
	RetryOnStatus map[int]bool

	// RetryBackoff is the base duration (seconds) for exponential backoff between
	// retries (backoff * 2^(attempt-1)), honoring Retry-After when present.
	// Built-in 0.5s. nil => inherit.
	RetryBackoff *float64

	// AbortSignal is the cooperative-cancellation primitive: Go's context.Context.
	// Checked before each attempt (a cancelled ctx raises the transport error
	// before the send) and threaded onto the request so it also cuts an in-flight
	// send. Built-in nil (no cancellation). nil => inherit.
	AbortSignal context.Context
}

// The built-in defaults (the contract floor). A nil field on a RequestOptions
// means "inherit"; these are what an unset field resolves to at apply-time.
const (
	defaultTimeoutSeconds = 30.0
	defaultRetries        = 0
	defaultRetryBackoff   = 0.5
)

// defaultRetryOnStatus is the built-in retryable-status set for idempotent
// methods. A fresh copy is handed out so callers can't mutate the shared default.
func defaultRetryOnStatus() map[int]bool {
	return map[int]bool{429: true, 500: true, 502: true, 503: true, 504: true}
}

// Merge returns a copy of o with any SET (non-nil) field of override applied.
// This is the per-request-over-client-default shallow merge: a nil field on
// override leaves o's value intact. A nil o is treated as an empty RequestOptions.
func (o *RequestOptions) Merge(override *RequestOptions) RequestOptions {
	var out RequestOptions
	if o != nil {
		out = *o
	}
	if override == nil {
		return out
	}
	if override.Timeout != nil {
		out.Timeout = override.Timeout
	}
	if override.Retries != nil {
		out.Retries = override.Retries
	}
	if override.RetryOnStatus != nil {
		out.RetryOnStatus = override.RetryOnStatus
	}
	if override.RetryBackoff != nil {
		out.RetryBackoff = override.RetryBackoff
	}
	if override.AbortSignal != nil {
		out.AbortSignal = override.AbortSignal
	}
	return out
}

// EffectiveOptions is a RequestOptions with every field resolved to a concrete
// value — produced by Resolve, so the request loop reads concrete values
// without re-checking defaults on every attempt. Mirrors the reference's
// _EffectiveOptions.
type EffectiveOptions struct {
	timeout       float64
	retries       int
	retryOnStatus map[int]bool
	retryBackoff  float64
	abortSignal   context.Context
}

// Resolve resolves the effective options: per-request over client-default over
// built-in. A nil field inherits the next level down; the built-in defaults are
// the floor. The result has every field concrete. Mirrors the reference resolve().
func Resolve(clientDefault, perRequest *RequestOptions) EffectiveOptions {
	merged := clientDefault.Merge(perRequest)
	eff := EffectiveOptions{
		timeout:       defaultTimeoutSeconds,
		retries:       defaultRetries,
		retryOnStatus: defaultRetryOnStatus(),
		retryBackoff:  defaultRetryBackoff,
		abortSignal:   merged.AbortSignal,
	}
	if merged.Timeout != nil {
		eff.timeout = *merged.Timeout
	}
	if merged.Retries != nil {
		eff.retries = *merged.Retries
	}
	if merged.RetryOnStatus != nil {
		eff.retryOnStatus = merged.RetryOnStatus
	}
	if merged.RetryBackoff != nil {
		eff.retryBackoff = *merged.RetryBackoff
	}
	return eff
}

// idempotentMethods are the methods with no server-side side effect — safe to
// retry on any retryable status. POST/PATCH are excluded: they may create/mutate,
// so they retry ONLY on 429/503 (the Retry-After-bearing throttles, which mean
// "the request was NOT processed"), never blindly on 500/502/504, to avoid
// duplicate side effects. This asymmetry is part of the pinned contract.
var idempotentMethods = map[string]bool{
	"GET": true, "PUT": true, "DELETE": true, "HEAD": true, "OPTIONS": true,
}

// StatusIsRetryable reports whether an HTTP status for method should trigger a
// retry. Idempotent methods (GET/PUT/DELETE) retry on the full retryOnStatus set;
// non-idempotent methods (POST/PATCH) retry only on 429/503. Mirrors the
// reference status_is_retryable().
func StatusIsRetryable(method string, status int, eff EffectiveOptions) bool {
	if !eff.retryOnStatus[status] {
		return false
	}
	if idempotentMethods[strings.ToUpper(method)] {
		return true
	}
	// Non-idempotent: only the throttle statuses (which carry Retry-After and
	// mean "the request was NOT processed, back off").
	return status == 429 || status == 503
}
