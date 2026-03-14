// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// PubSubNamespace provides PubSub token generation.
type PubSubNamespace struct {
	Resource
}

// NewPubSubNamespace creates a new PubSubNamespace.
func NewPubSubNamespace(client HTTPClient) *PubSubNamespace {
	return &PubSubNamespace{
		Resource: Resource{HTTP: client, Base: "/api/pubsub/tokens"},
	}
}

// CreateToken creates a PubSub token.
func (r *PubSubNamespace) CreateToken(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Base, data)
}
