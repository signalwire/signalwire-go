// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Package rest provides a REST client for the SignalWire platform APIs.
//
// It includes an HTTP transport layer, generic CRUD resource abstractions,
// paginated iteration, and namespaced sub-clients for each API domain.
package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/signalwire/signalwire-go/pkg/logging"
)

// userAgent is the User-Agent header sent with every request.
const userAgent = "signalwire-go-rest/1.0"

// ---------- SignalWireRestError ----------

// SignalWireRestError is returned when the SignalWire REST API responds with a
// non-2xx status code.
type SignalWireRestError struct {
	StatusCode int
	Body       string
	URL        string
	Method     string
}

// Error implements the error interface.
func (e *SignalWireRestError) Error() string {
	return fmt.Sprintf("%s %s returned %d: %s", e.Method, e.URL, e.StatusCode, e.Body)
}

// NewSignalWireRestError constructs a SignalWireRestError, substituting
// "GET" as the method when method is empty — matches Python's default.
func NewSignalWireRestError(statusCode int, body, url, method string) *SignalWireRestError {
	if method == "" {
		method = "GET"
	}
	return &SignalWireRestError{StatusCode: statusCode, Body: body, URL: url, Method: method}
}

// ---------- HttpClient ----------

// HttpClient is a thin wrapper around net/http that provides Basic Auth,
// JSON encoding/decoding, and standard headers for SignalWire API calls.
type HttpClient struct {
	baseURL    string
	projectID  string
	token      string
	httpClient *http.Client
	logger     *logging.Logger
}

// NewHttpClient creates a new HttpClient configured for the given SignalWire
// space. The baseURL is constructed as "https://<space>".
func NewHttpClient(projectID, token, space string) *HttpClient {
	return &HttpClient{
		baseURL:   "https://" + space,
		projectID: projectID,
		token:     token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logging.New("rest_client"),
	}
}

// BaseURL returns the base URL used by this client.
func (c *HttpClient) BaseURL() string {
	return c.baseURL
}

// Get performs an HTTP GET request. params are added as query-string
// parameters.
func (c *HttpClient) Get(path string, params map[string]string) (map[string]any, error) {
	return c.doRequest("GET", path, nil, params)
}

// Post performs an HTTP POST request with a JSON body. Optional params are
// appended to the URL as query-string parameters.
func (c *HttpClient) Post(path string, body map[string]any, params map[string]string) (map[string]any, error) {
	return c.doRequest("POST", path, body, params)
}

// Put performs an HTTP PUT request with a JSON body.
func (c *HttpClient) Put(path string, body map[string]any) (map[string]any, error) {
	return c.doRequest("PUT", path, body, nil)
}

// Patch performs an HTTP PATCH request with a JSON body.
func (c *HttpClient) Patch(path string, body map[string]any) (map[string]any, error) {
	return c.doRequest("PATCH", path, body, nil)
}

// Delete performs an HTTP DELETE request. It returns the parsed response body
// (or an empty map for 204 No Content) and any error.
func (c *HttpClient) Delete(path string) (map[string]any, error) {
	return c.doRequest("DELETE", path, nil, nil)
}

// doRequest is the shared request execution method. It sets Basic Auth,
// Content-Type, Accept, and User-Agent headers. A 204 No Content response
// returns an empty map. Non-2xx responses return a *SignalWireRestError.
func (c *HttpClient) doRequest(method, path string, body any, params map[string]string) (map[string]any, error) {
	// Build URL
	reqURL := c.baseURL + path
	if len(params) > 0 {
		q := url.Values{}
		for k, v := range params {
			q.Set(k, v)
		}
		reqURL += "?" + q.Encode()
	}

	// Encode body
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("json marshal: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	c.logger.Debug("REST request %s %s", method, path)

	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	// Headers
	req.SetBasicAuth(c.projectID, c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// Non-2xx error
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &SignalWireRestError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
			URL:        path,
			Method:     method,
		}
	}

	// 204 No Content or empty body
	if resp.StatusCode == 204 || len(respBody) == 0 {
		return map[string]any{}, nil
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	return result, nil
}

// ---------- CrudResource ----------

// CrudResource provides standard List, Create, Get, Update, Delete operations
// against a REST collection endpoint. Update defaults to PATCH; set
// UpdateMethod to "PUT" to override.
type CrudResource struct {
	Client       *HttpClient
	Path         string
	UpdateMethod string // "PATCH" (default) or "PUT"
}

// NewCrudResource creates a CrudResource for the given path. The default
// update method is PATCH.
func NewCrudResource(client *HttpClient, path string) *CrudResource {
	return &CrudResource{
		Client:       client,
		Path:         path,
		UpdateMethod: "PATCH",
	}
}

// NewCrudResourcePUT creates a CrudResource that uses PUT for updates.
func NewCrudResourcePUT(client *HttpClient, path string) *CrudResource {
	return &CrudResource{
		Client:       client,
		Path:         path,
		UpdateMethod: "PUT",
	}
}

// subPath joins additional path segments onto the resource base path.
func (r *CrudResource) subPath(parts ...string) string {
	return r.Path + "/" + strings.Join(parts, "/")
}

// List retrieves all items from the collection. Optional query parameters can
// be provided. The raw JSON response map is returned.
func (r *CrudResource) List(params map[string]string) (map[string]any, error) {
	return r.Client.Get(r.Path, params)
}

// Create sends a POST request to create a new resource.
func (r *CrudResource) Create(data map[string]any) (map[string]any, error) {
	return r.Client.Post(r.Path, data, nil)
}

// Get retrieves a single resource by ID.
func (r *CrudResource) Get(id string) (map[string]any, error) {
	return r.Client.Get(r.subPath(id), nil)
}

// Update modifies an existing resource by ID using the configured update
// method (PATCH or PUT).
func (r *CrudResource) Update(id string, data map[string]any) (map[string]any, error) {
	path := r.subPath(id)
	if r.UpdateMethod == "PUT" {
		return r.Client.Put(path, data)
	}
	return r.Client.Patch(path, data)
}

// Delete removes a resource by ID. It returns the parsed response body
// (or an empty map for 204 No Content) and any error.
func (r *CrudResource) Delete(id string) (map[string]any, error) {
	return r.Client.Delete(r.subPath(id))
}

// ---------- PaginatedIterator ----------

// PaginatedIterator walks through paginated API responses one page at a time.
// Each call to Next returns the items from the current page, a boolean
// indicating whether more pages exist, and any error encountered.
type PaginatedIterator struct {
	client  *HttpClient
	path    string
	params  map[string]string
	dataKey string
	done    bool
}

// NewPaginatedIterator creates a new iterator for the given endpoint.
// dataKey is the JSON key that holds the array of items (typically "data").
func NewPaginatedIterator(client *HttpClient, path string, params map[string]string, dataKey string) *PaginatedIterator {
	if params == nil {
		params = map[string]string{}
	}
	if dataKey == "" {
		dataKey = "data"
	}
	return &PaginatedIterator{
		client:  client,
		path:    path,
		params:  params,
		dataKey: dataKey,
	}
}

// Next fetches the next page of results. It returns the items from the page,
// a boolean hasMore that is true when additional pages remain, and any error.
// When there are no more pages, it returns nil, false, nil.
func (p *PaginatedIterator) Next() ([]map[string]any, bool, error) {
	if p.done {
		return nil, false, nil
	}

	resp, err := p.client.Get(p.path, p.params)
	if err != nil {
		return nil, false, err
	}

	// Extract items
	var items []map[string]any
	if raw, ok := resp[p.dataKey]; ok {
		if arr, ok := raw.([]any); ok {
			for _, v := range arr {
				if m, ok := v.(map[string]any); ok {
					items = append(items, m)
				}
			}
		}
	}

	// Check for next page link
	if linksRaw, ok := resp["links"]; ok {
		if links, ok := linksRaw.(map[string]any); ok {
			if nextRaw, ok := links["next"]; ok {
				if nextURL, ok := nextRaw.(string); ok && nextURL != "" && len(items) > 0 {
					// Parse query parameters from the next URL
					parsed, err := url.Parse(nextURL)
					if err == nil {
						newParams := map[string]string{}
						for k, v := range parsed.Query() {
							if len(v) > 0 {
								newParams[k] = v[0]
							}
						}
						p.params = newParams
						return items, true, nil
					}
				}
			}
		}
	}

	// No more pages
	p.done = true
	return items, false, nil
}

// ForEach calls fn for every item across all pages. It fetches pages lazily
// via Next and invokes fn once per item in the order they are returned.
// Iteration stops early if fn returns a non-nil error (that error is returned
// to the caller) or when Next signals there are no more pages.
func (p *PaginatedIterator) ForEach(fn func(map[string]any) error) error {
	for {
		items, hasMore, err := p.Next()
		if err != nil {
			return err
		}
		for _, item := range items {
			if err := fn(item); err != nil {
				return err
			}
		}
		if !hasMore {
			return nil
		}
	}
}
