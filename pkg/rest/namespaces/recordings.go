// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// RecordingsNamespace provides recording management (read-only + delete).
type RecordingsNamespace struct {
	Resource
}

// NewRecordingsNamespace creates a new RecordingsNamespace.
func NewRecordingsNamespace(client HTTPClient) *RecordingsNamespace {
	return &RecordingsNamespace{
		Resource: Resource{HTTP: client, Base: "/api/relay/rest/recordings"},
	}
}

// List lists all recordings.
func (r *RecordingsNamespace) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Get retrieves a recording by ID.
func (r *RecordingsNamespace) Get(id string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(id), nil)
}

// Delete removes a recording by ID.
func (r *RecordingsNamespace) Delete(id string) error {
	return r.HTTP.Delete(r.Path(id))
}
