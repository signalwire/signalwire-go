// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// MFANamespace provides multi-factor authentication via SMS or phone call.
type MFANamespace struct {
	Resource
}

// NewMFANamespace creates a new MFANamespace.
func NewMFANamespace(client HTTPClient) *MFANamespace {
	return &MFANamespace{
		Resource: Resource{HTTP: client, Base: "/api/relay/rest/mfa"},
	}
}

// SMS initiates MFA verification via SMS.
func (r *MFANamespace) SMS(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path("sms"), data, nil)
}

// Call initiates MFA verification via phone call.
func (r *MFANamespace) Call(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path("call"), data, nil)
}

// Verify verifies an MFA token for a given request ID.
func (r *MFANamespace) Verify(requestID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(requestID, "verify"), data, nil)
}
