// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package rest

import (
	"context"
	"fmt"
	"os"
)

// RestClient is the top-level REST client for the SignalWire platform.
// It provides namespaced access to all SignalWire API domains.
//
// Usage:
//
//	client, err := rest.NewRestClient("project-id", "api-token", "your-space.signalwire.com")
//	// or use environment variables SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, SIGNALWIRE_SPACE
//	client, err := rest.NewRestClient("", "", "")
//
//	agents, err := client.Fabric.AIAgents.List(context.Background(), nil)
//	client.Calling.Play(context.Background(), "call-id", namespaces.CallingNamespacePlayParams{...})
type RestClient struct {
	http      *HTTPClient
	projectID string

	// _GeneratedResourceTree (rest_tree_generated.go) supplies every namespace
	// field — Fabric, Calling, PhoneNumbers, Addresses, …, PubSub, Chat — and
	// promotes them through the embed, so client.Fabric.AIAgents.List(...) etc.
	// resolve exactly as before. The tree is generated from the x-sdk-* markup;
	// this file keeps only the non-spec-derivable bits (auth, HTTP construction,
	// env-var handling, the httpAdapter import-cycle breaker).
	_GeneratedResourceTree
}

// SetBaseURL overrides the base URL used by the underlying HTTPClient.
// Useful for pointing the client at a non-default endpoint such as the
// audit_rest_transport.py harness fixture, a recorded-cassette mock
// server, or a regional endpoint without re-running the constructor.
func (c *RestClient) SetBaseURL(url string) {
	c.http.SetBaseURL(url)
}

// HTTPClient exposes the underlying HTTP transport — the entry point for callers
// that need raw GET/POST access without going through a namespace resource.
func (c *RestClient) HTTPClient() *HTTPClient {
	return c.http
}

// NewRestClient creates a new RestClient. If project, token, or
// space are empty strings the corresponding environment variables are used:
//
//	SIGNALWIRE_PROJECT_ID
//	SIGNALWIRE_API_TOKEN
//	SIGNALWIRE_SPACE
//
// An error is returned when any of the three values is still empty after the
// environment lookup.
func NewRestClient(project, token, space string, opts ...*RequestOptions) (*RestClient, error) {
	if project == "" {
		project = os.Getenv("SIGNALWIRE_PROJECT_ID")
	}
	if token == "" {
		token = os.Getenv("SIGNALWIRE_API_TOKEN")
	}
	if space == "" {
		space = os.Getenv("SIGNALWIRE_SPACE")
	}

	if project == "" || token == "" || space == "" {
		return nil, fmt.Errorf(
			"project, token, and space are required; provide them as arguments " +
				"or set SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, and SIGNALWIRE_SPACE environment variables",
		)
	}

	h := NewHTTPClient(project, token, space, opts...)

	// Wrap the HTTPClient in a namespaces.HTTPClient adapter so namespaces
	// can use it without importing the rest package (avoiding a cycle).
	adapter := &httpAdapter{h}

	c := &RestClient{
		http:      h,
		projectID: project,
	}

	// Wire every namespace resource + container from the generated tree.
	c.wireGeneratedTree(adapter)

	return c, nil
}

// ---------- httpAdapter ----------

// httpAdapter wraps *HTTPClient to satisfy the namespaces.HTTPClient interface.
type httpAdapter struct {
	c *HTTPClient
}

// firstOpt returns the first non-nil per-request *RequestOptions from a generated
// verb's `opts ...*RequestOptions` tail (only the first is honored, mirroring the
// reference's single request_options keyword param). The adapter threads BOTH the
// caller's ctx AND that per-request override down to doRequestContextOpts, so an
// AbortSignal composes with (does not replace) the caller's ctx and a per-request
// timeout/retry policy overrides the client default for that one call.
func firstOpt(opts []*RequestOptions) *RequestOptions {
	for _, o := range opts {
		if o != nil {
			return o
		}
	}
	return nil
}

// Get issues a GET to path with params as the query string, decoding the JSON
// response body into a map. path is relative to the client's space/API base;
// opts carries an optional per-request override (see firstOpt). A non-2xx
// status is returned as an error — the client does not retry.
func (a *httpAdapter) Get(ctx context.Context, path string, params map[string]string, opts ...*RequestOptions) (map[string]any, error) {
	return a.c.doRequestContextOpts(ctx, "GET", path, nil, params, firstOpt(opts))
}

// Post issues a POST to path with body JSON-encoded as the request body and
// params as the query string, decoding the JSON response into a map. A non-2xx
// status is returned as an error.
func (a *httpAdapter) Post(ctx context.Context, path string, body map[string]any, params map[string]string, opts ...*RequestOptions) (map[string]any, error) {
	return a.c.doRequestContextOpts(ctx, "POST", path, body, params, firstOpt(opts))
}

// Put issues a PUT to path with body JSON-encoded as the request body (full
// replacement), decoding the JSON response into a map. It takes no query
// params. A non-2xx status is returned as an error.
func (a *httpAdapter) Put(ctx context.Context, path string, body map[string]any, opts ...*RequestOptions) (map[string]any, error) {
	return a.c.doRequestContextOpts(ctx, "PUT", path, body, nil, firstOpt(opts))
}

// Patch issues a PATCH to path with body JSON-encoded as the request body
// (partial update), decoding the JSON response into a map. It takes no query
// params. A non-2xx status is returned as an error.
func (a *httpAdapter) Patch(ctx context.Context, path string, body map[string]any, opts ...*RequestOptions) (map[string]any, error) {
	return a.c.doRequestContextOpts(ctx, "PATCH", path, body, nil, firstOpt(opts))
}

// Delete issues a DELETE to path with no request body and no query params,
// decoding any JSON response into a map (typically empty for a 204). A non-2xx
// status is returned as an error.
func (a *httpAdapter) Delete(ctx context.Context, path string, opts ...*RequestOptions) (map[string]any, error) {
	return a.c.doRequestContextOpts(ctx, "DELETE", path, nil, nil, firstOpt(opts))
}
