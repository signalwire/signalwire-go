// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// ProjectTokens provides project API token management.
type ProjectTokens struct {
	Resource
}

// Create creates a new project API token.
func (r *ProjectTokens) Create(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Base, data, nil)
}

// Update modifies a project API token.
func (r *ProjectTokens) Update(tokenID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Patch(r.Path(tokenID), data)
}

// Delete removes a project API token.
func (r *ProjectTokens) Delete(tokenID string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(tokenID))
}

// ProjectNamespace groups project management resources.
type ProjectNamespace struct {
	Tokens *ProjectTokens
}

// NewProjectNamespace creates a new ProjectNamespace.
func NewProjectNamespace(client HTTPClient) *ProjectNamespace {
	return &ProjectNamespace{
		Tokens: &ProjectTokens{Resource{HTTP: client, Base: "/api/project/tokens"}},
	}
}
