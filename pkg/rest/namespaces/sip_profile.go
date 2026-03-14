// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// SipProfileNamespace provides project SIP profile management (singleton resource).
type SipProfileNamespace struct {
	Resource
}

// NewSipProfileNamespace creates a new SipProfileNamespace.
func NewSipProfileNamespace(client HTTPClient) *SipProfileNamespace {
	return &SipProfileNamespace{
		Resource: Resource{HTTP: client, Base: "/api/relay/rest/sip_profile"},
	}
}

// Get retrieves the project SIP profile.
func (r *SipProfileNamespace) Get() (map[string]any, error) {
	return r.HTTP.Get(r.Base, nil)
}

// Update modifies the project SIP profile.
func (r *SipProfileNamespace) Update(data map[string]any) (map[string]any, error) {
	return r.HTTP.Put(r.Base, data)
}
