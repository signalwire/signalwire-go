// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// AddressesNamespace provides address management (no update endpoint).
type AddressesNamespace struct {
	Resource
}

// NewAddressesNamespace creates a new AddressesNamespace.
func NewAddressesNamespace(client HTTPClient) *AddressesNamespace {
	return &AddressesNamespace{
		Resource: Resource{HTTP: client, Base: "/api/relay/rest/addresses"},
	}
}

// List lists all addresses.
func (r *AddressesNamespace) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Create creates a new address.
func (r *AddressesNamespace) Create(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Base, data, nil)
}

// Get retrieves an address by ID.
func (r *AddressesNamespace) Get(id string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(id), nil)
}

// Delete removes an address by ID.
func (r *AddressesNamespace) Delete(id string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(id))
}

// AddressesResource is an alias for AddressesNamespace, matching the Python
// class name for cross-SDK parity. Prefer AddressesNamespace in new Go code.
type AddressesResource = AddressesNamespace
