// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// ImportedNumbersNamespace provides imported phone number management.
type ImportedNumbersNamespace struct {
	Resource
}

// NewImportedNumbersNamespace creates a new ImportedNumbersNamespace.
func NewImportedNumbersNamespace(client HTTPClient) *ImportedNumbersNamespace {
	return &ImportedNumbersNamespace{
		Resource: Resource{HTTP: client, Base: "/api/relay/rest/imported_phone_numbers"},
	}
}

// Create imports an externally-hosted phone number.
func (r *ImportedNumbersNamespace) Create(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Base, data)
}
