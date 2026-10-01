// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package rest

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/v3/pkg/rest/namespaces"
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
	// patHTTP is the Personal-Access-Token transport behind client.Space (nil
	// when no token was configured).
	patHTTP *HTTPClient

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
	if c.patHTTP != nil {
		c.patHTTP.SetBaseURL(url)
	}
}

// HTTPClient exposes the underlying HTTP transport — the entry point for callers
// that need raw GET/POST access without going through a namespace resource.
func (c *RestClient) HTTPClient() *HTTPClient {
	return c.http
}

// RestClientOption configures NewRestClient beyond its three auth positionals
// (the reference's request_options / personal_access_token keyword params).
type RestClientOption func(*restClientConfig)

// restClientConfig collects the RestClientOption settings.
type restClientConfig struct {
	requestOptions      *RequestOptions
	personalAccessToken string
}

// WithRequestOptions sets the CLIENT-DEFAULT request-options envelope (timeout /
// retries / backoff / abort) applied to every request unless a per-request
// *RequestOptions overrides it.
func WithRequestOptions(o *RequestOptions) RestClientOption {
	return func(c *restClientConfig) { c.requestOptions = o }
}

// WithPersonalAccessToken sets the Personal Access Token (a user's "pat_..."
// token) that authenticates client.Space — the Space Administration API, which
// prime-rails serves only to a PAT (HTTP Basic with an empty username).
func WithPersonalAccessToken(pat string) RestClientOption {
	return func(c *restClientConfig) { c.personalAccessToken = pat }
}

// NewRestClient creates a new RestClient for one project, one space's
// administration API, or both. Each value falls back to its environment
// variable when empty:
//
//	SIGNALWIRE_PROJECT_ID
//	SIGNALWIRE_API_TOKEN
//	SIGNALWIRE_SPACE
//	SIGNALWIRE_PERSONAL_ACCESS_TOKEN   (WithPersonalAccessToken)
//
// project + token authenticate every project-scoped resource; a personal access
// token authenticates client.Space. Either credential, or both, may be given;
// calling a resource whose credential is missing returns an error.
//
//	admin, err := rest.NewRestClient("", "", "your-space.signalwire.com",
//		rest.WithPersonalAccessToken("pat_..."))
//	members, err := admin.Space.Members.List(ctx, nil)
//
// An error is returned when space is still empty after the environment lookup,
// or when neither a complete project + token pair nor a personal access token
// is available.
func NewRestClient(project, token, space string, opts ...RestClientOption) (*RestClient, error) {
	var cfg restClientConfig
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	if project == "" {
		project = os.Getenv("SIGNALWIRE_PROJECT_ID")
	}
	if token == "" {
		token = os.Getenv("SIGNALWIRE_API_TOKEN")
	}
	if space == "" {
		space = os.Getenv("SIGNALWIRE_SPACE")
	}
	pat := cfg.personalAccessToken
	if pat == "" {
		pat = os.Getenv("SIGNALWIRE_PERSONAL_ACCESS_TOKEN")
	}

	hasProject := project != "" && token != ""
	if space == "" || (!hasProject && pat == "") {
		return nil, fmt.Errorf(
			"project, token, and space are required; provide them as arguments " +
				"or set SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, and SIGNALWIRE_SPACE environment variables " +
				"(or, for client.Space only, space and SIGNALWIRE_PERSONAL_ACCESS_TOKEN)",
		)
	}

	var reqOpts []*RequestOptions
	if cfg.requestOptions != nil {
		reqOpts = append(reqOpts, cfg.requestOptions)
	}
	h := NewHTTPClient(project, token, space, reqOpts...)

	c := &RestClient{
		http:      h,
		projectID: project,
	}

	// Wrap each HTTPClient in a namespaces.HTTPClient adapter so namespaces
	// can use it without importing the rest package (avoiding a cycle). A
	// missing credential becomes an adapter that fails every call with a clear
	// error instead of sending unauthenticated requests.
	var projectAdapter, patAdapter namespaces.HTTPClient
	if hasProject {
		projectAdapter = &httpAdapter{h}
	} else {
		projectAdapter = missingCredential{"project and token are required for this resource " +
			"(SIGNALWIRE_PROJECT_ID / SIGNALWIRE_API_TOKEN); this client has only " +
			"a personal access token, which authenticates client.Space"}
	}
	if pat != "" {
		// A Personal Access Token is HTTP Basic with an EMPTY username
		// (prime-rails API::Space::BaseController -> Authenticators::PersonalAccessToken).
		patHTTP := NewHTTPClient("", pat, space, reqOpts...)
		c.patHTTP = patHTTP
		patAdapter = &httpAdapter{patHTTP}
	} else {
		patAdapter = missingCredential{"a personal access token is required for client.Space " +
			"(SIGNALWIRE_PERSONAL_ACCESS_TOKEN)"}
	}

	// Wire every namespace resource + container from the generated tree.
	c.wireGeneratedTree(projectAdapter, patAdapter)

	return c, nil
}

// missingCredential is the namespaces.HTTPClient wired for a resource whose
// credential this client was not given: every call fails with msg.
type missingCredential struct{ msg string }

func (m missingCredential) err() error { return errors.New(m.msg) }

// Get fails with the missing-credential error.
func (m missingCredential) Get(context.Context, string, map[string]string, ...*RequestOptions) (map[string]any, error) {
	return nil, m.err()
}

// Post fails with the missing-credential error.
func (m missingCredential) Post(context.Context, string, map[string]any, map[string]string, ...*RequestOptions) (map[string]any, error) {
	return nil, m.err()
}

// Put fails with the missing-credential error.
func (m missingCredential) Put(context.Context, string, map[string]any, ...*RequestOptions) (map[string]any, error) {
	return nil, m.err()
}

// Patch fails with the missing-credential error.
func (m missingCredential) Patch(context.Context, string, map[string]any, ...*RequestOptions) (map[string]any, error) {
	return nil, m.err()
}

// Delete fails with the missing-credential error.
func (m missingCredential) Delete(context.Context, string, ...*RequestOptions) (map[string]any, error) {
	return nil, m.err()
}

// GetText fails with the missing-credential error.
func (m missingCredential) GetText(context.Context, string, map[string]string, map[string]string, ...*RequestOptions) (string, error) {
	return "", m.err()
}

// GetRedirectLocation fails with the missing-credential error.
func (m missingCredential) GetRedirectLocation(context.Context, string, map[string]string, ...*RequestOptions) (string, error) {
	return "", m.err()
}

// PostWithHeaders fails with the missing-credential error.
func (m missingCredential) PostWithHeaders(context.Context, string, map[string]any, map[string]string, map[string]string, ...*RequestOptions) (map[string]any, error) {
	return nil, m.err()
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
	m, _, err := a.c.doRequestContextOpts(ctx, "GET", path, nil, params, firstOpt(opts), nil, responseJSON)
	return m, err
}

// Post issues a POST to path with body JSON-encoded as the request body and
// params as the query string, decoding the JSON response into a map. A non-2xx
// status is returned as an error.
func (a *httpAdapter) Post(ctx context.Context, path string, body map[string]any, params map[string]string, opts ...*RequestOptions) (map[string]any, error) {
	m, _, err := a.c.doRequestContextOpts(ctx, "POST", path, body, params, firstOpt(opts), nil, responseJSON)
	return m, err
}

// Put issues a PUT to path with body JSON-encoded as the request body (full
// replacement), decoding the JSON response into a map. It takes no query
// params. A non-2xx status is returned as an error.
func (a *httpAdapter) Put(ctx context.Context, path string, body map[string]any, opts ...*RequestOptions) (map[string]any, error) {
	m, _, err := a.c.doRequestContextOpts(ctx, "PUT", path, body, nil, firstOpt(opts), nil, responseJSON)
	return m, err
}

// Patch issues a PATCH to path with body JSON-encoded as the request body
// (partial update), decoding the JSON response into a map. It takes no query
// params. A non-2xx status is returned as an error.
func (a *httpAdapter) Patch(ctx context.Context, path string, body map[string]any, opts ...*RequestOptions) (map[string]any, error) {
	m, _, err := a.c.doRequestContextOpts(ctx, "PATCH", path, body, nil, firstOpt(opts), nil, responseJSON)
	return m, err
}

// Delete issues a DELETE to path with no request body and no query params,
// decoding any JSON response into a map (typically empty for a 204). A non-2xx
// status is returned as an error.
func (a *httpAdapter) Delete(ctx context.Context, path string, opts ...*RequestOptions) (map[string]any, error) {
	m, _, err := a.c.doRequestContextOpts(ctx, "DELETE", path, nil, nil, firstOpt(opts), nil, responseJSON)
	return m, err
}

// GetText issues a GET whose success body is a non-JSON media type and returns
// it as text, sending headers (the media type's Accept) on this request.
func (a *httpAdapter) GetText(ctx context.Context, path string, params map[string]string, headers map[string]string, opts ...*RequestOptions) (string, error) {
	_, text, err := a.c.doRequestContextOpts(ctx, "GET", path, nil, params, firstOpt(opts), headers, responseText)
	return text, err
}

// GetRedirectLocation issues a GET whose success IS a redirect and returns its
// Location without following it.
func (a *httpAdapter) GetRedirectLocation(ctx context.Context, path string, params map[string]string, opts ...*RequestOptions) (string, error) {
	_, loc, err := a.c.doRequestContextOpts(ctx, "GET", path, nil, params, firstOpt(opts), nil, responseRedirect)
	return loc, err
}

// PostWithHeaders issues a POST carrying extra request headers (an operation's
// declared header parameters, e.g. Idempotency-Key).
func (a *httpAdapter) PostWithHeaders(ctx context.Context, path string, body map[string]any, params map[string]string, headers map[string]string, opts ...*RequestOptions) (map[string]any, error) {
	m, _, err := a.c.doRequestContextOpts(ctx, "POST", path, body, params, firstOpt(opts), headers, responseJSON)
	return m, err
}
