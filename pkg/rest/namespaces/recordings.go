// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

import "fmt"

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

// List lists all recordings. params may contain values of any type (matching
// Python's **data); non-string values are stringified via fmt.Sprintf before
// being sent as query parameters.
func (r *RecordingsNamespace) List(params map[string]any) (map[string]any, error) {
	var strParams map[string]string
	if len(params) > 0 {
		strParams = make(map[string]string, len(params))
		for k, v := range params {
			if s, ok := v.(string); ok {
				strParams[k] = s
			} else {
				strParams[k] = fmt.Sprintf("%v", v)
			}
		}
	}
	return r.HTTP.Get(r.Base, strParams)
}

// Get retrieves a recording by ID.
func (r *RecordingsNamespace) Get(id string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(id), nil)
}

// Delete removes a recording by ID.
func (r *RecordingsNamespace) Delete(id string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(id))
}

// RecordingsResource is an alias for RecordingsNamespace, matching the Python
// class name for cross-SDK parity. Prefer RecordingsNamespace in new Go code.
type RecordingsResource = RecordingsNamespace
