// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// VerifiedCallersNamespace provides verified caller ID management with
// verification flow.
type VerifiedCallersNamespace struct {
	*CrudResource
}

// NewVerifiedCallersNamespace creates a new VerifiedCallersNamespace.
func NewVerifiedCallersNamespace(client HTTPClient) *VerifiedCallersNamespace {
	return &VerifiedCallersNamespace{
		CrudResource: NewCrudResourcePUT(client, "/api/relay/rest/verified_caller_ids"),
	}
}

// RedialVerification redials the verification call for a caller ID.
func (r *VerifiedCallersNamespace) RedialVerification(callerID string) (map[string]any, error) {
	return r.HTTP.Post(r.Path(callerID, "verification"), nil, nil)
}

// SubmitVerification submits a verification code for a caller ID.
func (r *VerifiedCallersNamespace) SubmitVerification(callerID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Put(r.Path(callerID, "verification"), data)
}
