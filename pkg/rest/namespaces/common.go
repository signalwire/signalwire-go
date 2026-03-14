// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Package namespaces contains the individual API namespace implementations
// for the SignalWire REST client.
package namespaces

import "strings"

// HTTPClient is the interface that namespace implementations use to make HTTP
// requests. It is satisfied by the httpAdapter in the parent rest package,
// which prevents an import cycle.
type HTTPClient interface {
	Get(path string, params map[string]string) (map[string]any, error)
	Post(path string, body map[string]any) (map[string]any, error)
	Put(path string, body map[string]any) (map[string]any, error)
	Patch(path string, body map[string]any) (map[string]any, error)
	Delete(path string) error
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
func (r *CrudResource) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Create sends a POST request to create a new resource.
func (r *CrudResource) Create(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Base, data)
}

// Get retrieves a single resource by ID.
func (r *CrudResource) Get(id string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(id), nil)
}

// Update modifies an existing resource by ID.
func (r *CrudResource) Update(id string, data map[string]any) (map[string]any, error) {
	p := r.Path(id)
	if r.UpdateMethod == "PUT" {
		return r.HTTP.Put(p, data)
	}
	return r.HTTP.Patch(p, data)
}

// Delete removes a resource by ID.
func (r *CrudResource) Delete(id string) error {
	return r.HTTP.Delete(r.Path(id))
}

// ListAddresses lists addresses associated with a resource (for fabric
// resources that support it).
func (r *CrudResource) ListAddresses(id string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(id, "addresses"), params)
}
