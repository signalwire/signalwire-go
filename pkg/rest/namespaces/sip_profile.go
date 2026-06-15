// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// SIPProfileNamespace provides project SIP profile management (singleton resource).
type SIPProfileNamespace struct {
	Resource
}

// NewSIPProfileNamespace creates a new SIPProfileNamespace.
func NewSIPProfileNamespace(client HTTPClient) *SIPProfileNamespace {
	return &SIPProfileNamespace{
		Resource: Resource{HTTP: client, Base: "/api/relay/rest/sip_profile"},
	}
}

// Get retrieves the project SIP profile.
func (r *SIPProfileNamespace) Get() (map[string]any, error) {
	return r.HTTP.Get(r.Base, nil)
}

// Update modifies the project SIP profile.
func (r *SIPProfileNamespace) Update(data map[string]any) (map[string]any, error) {
	return r.HTTP.Put(r.Base, data)
}

// SIPProfileResource is an alias for SIPProfileNamespace, matching the Python
// class name for cross-SDK parity. Prefer SIPProfileNamespace in new Go code.
type SIPProfileResource = SIPProfileNamespace
