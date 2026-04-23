// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// QueuesNamespace provides queue management with member operations.
type QueuesNamespace struct {
	*CrudResource
}

// NewQueuesNamespace creates a new QueuesNamespace.
func NewQueuesNamespace(client HTTPClient) *QueuesNamespace {
	return &QueuesNamespace{
		CrudResource: NewCrudResourcePUT(client, "/api/relay/rest/queues"),
	}
}

// ListMembers lists members of a queue.
func (r *QueuesNamespace) ListMembers(queueID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(queueID, "members"), params)
}

// GetNextMember retrieves the next member in the queue.
func (r *QueuesNamespace) GetNextMember(queueID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(queueID, "members", "next"), nil)
}

// GetMember retrieves a specific member from a queue.
func (r *QueuesNamespace) GetMember(queueID, memberID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(queueID, "members", memberID), nil)
}

// QueuesResource is an alias for QueuesNamespace, matching the Python class name
// for cross-SDK parity. Prefer QueuesNamespace in new Go code.
type QueuesResource = QueuesNamespace
