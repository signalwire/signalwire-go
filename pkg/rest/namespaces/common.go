// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Package namespaces contains the individual API namespace implementations
// for the SignalWire REST client.
package namespaces

import (
	"context"
	"encoding/json"
	"strings"
)

// HTTPClient is the interface that namespace implementations use to make HTTP
// requests. It is satisfied by the httpAdapter in the parent rest package,
// which prevents an import cycle.
//
// Every method takes a leading context.Context (Go's idiomatic deadline/
// cancellation carrier): the resource methods thread the caller's ctx down to
// the HTTP request, so a call like client.Fabric.Addresses.List(ctx, nil) is
// cancellable. ctx is compile-time-only plumbing — it is NEVER serialized into
// the request body or query, so the wire bytes are unchanged.
type HTTPClient interface {
	Get(ctx context.Context, path string, params map[string]string) (map[string]any, error)
	Post(ctx context.Context, path string, body map[string]any, params map[string]string) (map[string]any, error)
	Put(ctx context.Context, path string, body map[string]any) (map[string]any, error)
	Patch(ctx context.Context, path string, body map[string]any) (map[string]any, error)
	Delete(ctx context.Context, path string) (map[string]any, error)
}

// Resource is a helper for building sub-paths from a base path.
type Resource struct {
	HTTP HTTPClient
	Base string
}

// Path joins additional segments onto the base path.
func (r *Resource) Path(parts ...string) string {
	if len(parts) == 0 {
		return r.Base
	}
	return r.Base + "/" + strings.Join(parts, "/")
}

// CrudResource provides standard List, Create, Get, Update, Delete operations
// against a REST collection endpoint within a namespace.
type CrudResource struct {
	Resource
	UpdateMethod string // "PATCH" (default) or "PUT"
}

// NewCrudResource creates a CrudResource with PATCH as the update method.
func NewCrudResource(client HTTPClient, path string) *CrudResource {
	return &CrudResource{
		Resource:     Resource{HTTP: client, Base: path},
		UpdateMethod: "PATCH",
	}
}

// NewCrudResourcePUT creates a CrudResource that uses PUT for updates.
func NewCrudResourcePUT(client HTTPClient, path string) *CrudResource {
	return &CrudResource{
		Resource:     Resource{HTTP: client, Base: path},
		UpdateMethod: "PUT",
	}
}

// List retrieves all items from the collection.
func (r *CrudResource) List(ctx context.Context, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(ctx, r.Base, params)
}

// Paginate returns a Paginator that walks EVERY page of this resource's list
// endpoint, following the response's links.next cursor so callers no longer
// hand-build the path + token loop. List returns a single raw page (the server's
// first response); Paginate follows the cursor and yields each item.
//
// Equivalent to the Python SDK's ReadResource.paginate(**params); data_key is
// fixed to "data".
//
//	it := client.Fabric.Addresses.Paginate(nil)
//	for {
//	    items, hasMore, err := it.Next()
//	    if err != nil { return err }
//	    // ... use items ...
//	    if !hasMore { break }
//	}
//
// (Or use it.ForEach for the item-at-a-time idiom.)
func (r *CrudResource) Paginate(ctx context.Context, params map[string]string) *Paginator {
	return NewPaginator(ctx, r.HTTP, r.Base, params, "data")
}

// Create sends a POST request to create a new resource.
func (r *CrudResource) Create(ctx context.Context, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(ctx, r.Base, data, nil)
}

// Get retrieves a single resource by ID.
func (r *CrudResource) Get(ctx context.Context, id string) (map[string]any, error) {
	return r.HTTP.Get(ctx, r.Path(id), nil)
}

// Update modifies an existing resource by ID.
func (r *CrudResource) Update(ctx context.Context, id string, data map[string]any) (map[string]any, error) {
	p := r.Path(id)
	if r.UpdateMethod == "PUT" {
		return r.HTTP.Put(ctx, p, data)
	}
	return r.HTTP.Patch(ctx, p, data)
}

// Delete removes a resource by ID. It returns the parsed response body
// (or an empty map for 204 No Content) and any error.
func (r *CrudResource) Delete(ctx context.Context, id string) (map[string]any, error) {
	return r.HTTP.Delete(ctx, r.Path(id))
}

// CrudWithAddresses extends CrudResource with the nested addresses endpoint.
// Matches Python's CrudWithAddresses at _base.py:109-113.
// Only resources that explicitly support the addresses sub-resource should
// embed this type; plain CrudResource does not expose ListAddresses.
type CrudWithAddresses struct {
	*CrudResource
}

// NewCrudWithAddresses constructs a CrudWithAddresses backed by a PATCH-default
// CrudResource. Use NewCrudWithAddressesPUT for resources that update via PUT.
func NewCrudWithAddresses(client HTTPClient, path string) *CrudWithAddresses {
	return &CrudWithAddresses{NewCrudResource(client, path)}
}

// NewCrudWithAddressesPUT constructs a CrudWithAddresses backed by a PUT-update
// CrudResource.
func NewCrudWithAddressesPUT(client HTTPClient, path string) *CrudWithAddresses {
	return &CrudWithAddresses{NewCrudResourcePUT(client, path)}
}

// ListAddresses lists addresses associated with the resource identified by id.
func (r *CrudWithAddresses) ListAddresses(ctx context.Context, id string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(ctx, r.Path(id, "addresses"), params)
}

// decodeResult re-marshals a base-method map[string]any response into a typed
// generated wire struct T. The generated REST resource methods return
// (*T, error) for operations whose spec declares a typed ($ref) 200/201 response
// (the closed typed-output surface — PORT_PHILOSOPHY_GO.md §4 / TYPED_SURFACE_
// STRATEGY §4); this converts the loose map the HTTP layer decodes into the typed
// value. The wire bytes are unchanged — the HTTP layer already parsed the JSON
// into a map, and this round-trips that map through the same encoding/json into
// the struct's json-tagged fields. A base-method error is passed through
// untouched (the typed pointer is nil on error, matching Go convention).
func decodeResult[T any](m map[string]any, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	raw, mErr := json.Marshal(m)
	if mErr != nil {
		return nil, mErr
	}
	var out T
	if uErr := json.Unmarshal(raw, &out); uErr != nil {
		return nil, uErr
	}
	return &out, nil
}

// mergeExtra merges optional extra-fields maps into body. It is used by the
// generated Set* wrappers (SetSwmlWebhook, SetCxmlWebhook, …) to funnel their
// variadic extra-map tail into the update body. Kept here (a hand base file) so
// the generated resource files can call it without owning it.
func mergeExtra(body map[string]any, extra []map[string]any) {
	if len(extra) == 0 {
		return
	}
	for _, m := range extra {
		for k, v := range m {
			body[k] = v
		}
	}
}
