// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// ---------- RegistryBrands ----------

// RegistryBrands provides 10DLC brand management.
type RegistryBrands struct {
	Resource
}

// List lists all brands.
func (r *RegistryBrands) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Create creates a new brand.
func (r *RegistryBrands) Create(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Base, data)
}

// Get retrieves a brand by ID.
func (r *RegistryBrands) Get(brandID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(brandID), nil)
}

// ListCampaigns lists campaigns for a brand.
func (r *RegistryBrands) ListCampaigns(brandID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(brandID, "campaigns"), params)
}

// CreateCampaign creates a campaign under a brand.
func (r *RegistryBrands) CreateCampaign(brandID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(brandID, "campaigns"), data)
}

// ---------- RegistryCampaigns ----------

// RegistryCampaigns provides 10DLC campaign management.
type RegistryCampaigns struct {
	Resource
}

// Get retrieves a campaign by ID.
func (r *RegistryCampaigns) Get(campaignID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(campaignID), nil)
}

// Update modifies a campaign by ID.
func (r *RegistryCampaigns) Update(campaignID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Put(r.Path(campaignID), data)
}

// ListNumbers lists numbers assigned to a campaign.
func (r *RegistryCampaigns) ListNumbers(campaignID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(campaignID, "numbers"), params)
}

// ListOrders lists orders for a campaign.
func (r *RegistryCampaigns) ListOrders(campaignID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(campaignID, "orders"), params)
}

// CreateOrder creates a number assignment order for a campaign.
func (r *RegistryCampaigns) CreateOrder(campaignID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(campaignID, "orders"), data)
}

// ---------- RegistryOrders ----------

// RegistryOrders provides 10DLC assignment order management.
type RegistryOrders struct {
	Resource
}

// Get retrieves an order by ID.
func (r *RegistryOrders) Get(orderID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(orderID), nil)
}

// ---------- RegistryNumbers ----------

// RegistryNumbers provides 10DLC number assignment management.
type RegistryNumbers struct {
	Resource
}

// Delete removes a number assignment.
func (r *RegistryNumbers) Delete(numberID string) error {
	return r.HTTP.Delete(r.Path(numberID))
}

// ---------- RegistryNamespace ----------

// RegistryNamespace groups all 10DLC Campaign Registry resources.
type RegistryNamespace struct {
	Brands    *RegistryBrands
	Campaigns *RegistryCampaigns
	Orders    *RegistryOrders
	Numbers   *RegistryNumbers
}

// NewRegistryNamespace creates a new RegistryNamespace with all sub-resources.
func NewRegistryNamespace(client HTTPClient) *RegistryNamespace {
	base := "/api/relay/rest/registry/beta"
	return &RegistryNamespace{
		Brands:    &RegistryBrands{Resource{HTTP: client, Base: base + "/brands"}},
		Campaigns: &RegistryCampaigns{Resource{HTTP: client, Base: base + "/campaigns"}},
		Orders:    &RegistryOrders{Resource{HTTP: client, Base: base + "/orders"}},
		Numbers:   &RegistryNumbers{Resource{HTTP: client, Base: base + "/numbers"}},
	}
}
