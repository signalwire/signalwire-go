// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// ShortCodesNamespace provides short code management (read + update only).
type ShortCodesNamespace struct {
	Resource
}

// NewShortCodesNamespace creates a new ShortCodesNamespace.
func NewShortCodesNamespace(client HTTPClient) *ShortCodesNamespace {
	return &ShortCodesNamespace{
		Resource: Resource{HTTP: client, Base: "/api/relay/rest/short_codes"},
	}
}

// List lists all short codes.
func (r *ShortCodesNamespace) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Get retrieves a short code by ID.
func (r *ShortCodesNamespace) Get(id string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(id), nil)
}

// Update modifies a short code by ID.
func (r *ShortCodesNamespace) Update(id string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Put(r.Path(id), data)
}
