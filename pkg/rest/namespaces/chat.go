// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// ChatNamespace provides Chat token generation.
type ChatNamespace struct {
	Resource
}

// NewChatNamespace creates a new ChatNamespace.
func NewChatNamespace(client HTTPClient) *ChatNamespace {
	return &ChatNamespace{
		Resource: Resource{HTTP: client, Base: "/api/chat/tokens"},
	}
}

// CreateToken creates a Chat token.
func (r *ChatNamespace) CreateToken(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Base, data, nil)
}
