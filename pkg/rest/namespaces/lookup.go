// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// LookupNamespace provides phone number lookup (carrier, CNAM).
type LookupNamespace struct {
	Resource
}

// NewLookupNamespace creates a new LookupNamespace.
func NewLookupNamespace(client HTTPClient) *LookupNamespace {
	return &LookupNamespace{
		Resource: Resource{HTTP: client, Base: "/api/relay/rest/lookup"},
	}
}

// PhoneNumber looks up information about a phone number.
// The e164 parameter should be the number in E.164 format.
// Optional params can include "include" for additional data (e.g., "carrier").
func (r *LookupNamespace) PhoneNumber(e164 string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path("phone_number", e164), params)
}
