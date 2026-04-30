// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

import "fmt"

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
func (r *ShortCodesNamespace) List(params map[string]any) (map[string]any, error) {
	str := make(map[string]string, len(params))
	for k, v := range params {
		str[k] = fmt.Sprint(v)
	}
	return r.HTTP.Get(r.Base, str)
}

// Get retrieves a short code by ID.
func (r *ShortCodesNamespace) Get(id string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(id), nil)
}

// Update modifies a short code by ID.
func (r *ShortCodesNamespace) Update(id string, kwargs map[string]any) (map[string]any, error) {
	return r.HTTP.Put(r.Path(id), kwargs)
}

// ShortCodesResource is an alias for ShortCodesNamespace, matching the Python
// class name for cross-SDK parity. Prefer ShortCodesNamespace in new Go code.
type ShortCodesResource = ShortCodesNamespace
