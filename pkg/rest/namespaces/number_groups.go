// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// NumberGroupsNamespace provides number group management with membership operations.
type NumberGroupsNamespace struct {
	*CrudResource
}

// NewNumberGroupsNamespace creates a new NumberGroupsNamespace.
func NewNumberGroupsNamespace(client HTTPClient) *NumberGroupsNamespace {
	return &NumberGroupsNamespace{
		CrudResource: NewCrudResourcePUT(client, "/api/relay/rest/number_groups"),
	}
}

// ListMemberships lists number group memberships for a group.
func (r *NumberGroupsNamespace) ListMemberships(groupID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(groupID, "number_group_memberships"), params)
}

// AddMembership adds a number to a group.
func (r *NumberGroupsNamespace) AddMembership(groupID string, kwargs map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(groupID, "number_group_memberships"), kwargs, nil)
}

// GetMembership retrieves a specific membership by ID.
func (r *NumberGroupsNamespace) GetMembership(membershipID string) (map[string]any, error) {
	return r.HTTP.Get("/api/relay/rest/number_group_memberships/"+membershipID, nil)
}

// DeleteMembership removes a membership by ID.
func (r *NumberGroupsNamespace) DeleteMembership(membershipID string) (map[string]any, error) {
	return r.HTTP.Delete("/api/relay/rest/number_group_memberships/" + membershipID)
}

// NumberGroupsResource is an alias for NumberGroupsNamespace, matching the
// Python class name for cross-SDK parity.
type NumberGroupsResource = NumberGroupsNamespace
