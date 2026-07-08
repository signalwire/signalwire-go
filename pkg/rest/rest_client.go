// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package rest

import (
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
//	agents, err := client.Fabric.AIAgents.List(nil)
//	client.Calling.Play("call-id", map[string]any{"play": [...]})
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

// HTTPClient exposes the underlying HTTP transport. It is the public form
// of Python's “signalwire_client._http“ and is the entry point used by
// helpers like PaginatedIterator that need raw GET access without going
// through a namespace resource.
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
func NewRestClient(project, token, space string) (*RestClient, error) {
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

	h := NewHTTPClient(project, token, space)

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

func (a *httpAdapter) Get(path string, params map[string]string) (map[string]any, error) {
	return a.c.Get(path, params)
}
func (a *httpAdapter) Post(path string, body map[string]any, params map[string]string) (map[string]any, error) {
	return a.c.Post(path, body, params)
}
func (a *httpAdapter) Put(path string, body map[string]any) (map[string]any, error) {
	return a.c.Put(path, body)
}
func (a *httpAdapter) Patch(path string, body map[string]any) (map[string]any, error) {
	return a.c.Patch(path, body)
}
func (a *httpAdapter) Delete(path string) (map[string]any, error) {
	return a.c.Delete(path)
}
