// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

import "context"

// RequestOptions is the REST request-options envelope (plan 4.2): a single value
// object controlling per-request transport behavior — timeout, an
// idempotency-aware retry policy with exponential backoff, and cooperative
// cancellation. It mirrors the Python reference
// (signalwire.rest._request_options.RequestOptions).
//
// It lives in the namespaces package (not the parent rest package) so the
// generated resource verbs can name it in their `opts ...*RequestOptions` tail
// WITHOUT the rest->namespaces import cycle — exactly like Paginator. The parent
// rest package re-exports it as a transparent type ALIAS
// (`rest.RequestOptions = namespaces.RequestOptions`), so every existing
// `rest.RequestOptions{...}` call site, its client-default seam, and the request
// loop keep working unchanged. The signature enumerator folds the type back to
// `class:signalwire.rest._request_options.RequestOptions` in translateType.
//
// It is supplied at two levels:
//
//   - Client default: NewRestClientWithOptions(..., RequestOptions{...}) (or the
//     HTTPClient's WithRequestOptions seam) stores it and applies it to every
//     request.
//   - Per-request override: each generated verb accepts a trailing optional
//     `opts ...*RequestOptions` that SHALLOW-overrides the client default for
//     that one call — an unset (nil) field falls back to the client default,
//     then the built-in default.
//
// Every field is a pointer so "unset" (nil) is distinguishable from a deliberate
// zero value ("inherit" vs. "0 retries" / "0s timeout"). The Python reference
// uses None for the same reason; Go uses nil pointers.
type RequestOptions struct {
	// Timeout is the max wall-clock duration per attempt (in seconds; a float to
	// match the reference's fractional-second timeouts). Built-in default 30.0s.
	// nil => inherit.
	Timeout *float64

	// Retries is the number of RETRY attempts (total attempts = Retries + 1) on a
	// retryable failure. Built-in default 0. nil => inherit.
	Retries *int

	// RetryOnStatus is the set of HTTP statuses that trigger a retry for an
	// idempotent method. Built-in {429, 500, 502, 503, 504}. nil => inherit.
	RetryOnStatus map[int]bool

	// RetryBackoff is the base duration (seconds) for exponential backoff between
	// retries (backoff * 2^(attempt-1)), honoring Retry-After when present.
	// Built-in 0.5s. nil => inherit.
	RetryBackoff *float64

	// AbortSignal is the cooperative-cancellation primitive: Go's context.Context.
	// Checked before each attempt and threaded onto the request so it also cuts an
	// in-flight send. Built-in nil (no cancellation). nil => inherit.
	AbortSignal context.Context
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
