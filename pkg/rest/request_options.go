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

	"github.com/signalwire/signalwire-go/v3/pkg/rest/namespaces"
)

// RequestOptions is the REST request-options envelope. It is a transparent type
// ALIAS to namespaces.RequestOptions: the struct + its Merge method live in the
// namespaces package so the generated resource verbs can name it in their
// `opts ...*RequestOptions` tail WITHOUT the rest->namespaces import cycle
// (exactly like Paginator). The alias keeps every existing `rest.RequestOptions`
// call site, the client-default seam, and the request loop below working
// unchanged, and keeps the type's public spelling under the rest package. The
// full doc + field semantics live on the namespaces definition.
type RequestOptions = namespaces.RequestOptions

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

// Merge lives on namespaces.RequestOptions (this type's alias target); Resolve
// below invokes it through the alias.

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
