// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// PhoneNumbersNamespace provides phone number management with search.
type PhoneNumbersNamespace struct {
	*CrudResource
}

// NewPhoneNumbersNamespace creates a new PhoneNumbersNamespace.
func NewPhoneNumbersNamespace(client HTTPClient) *PhoneNumbersNamespace {
	return &PhoneNumbersNamespace{
		CrudResource: NewCrudResourcePUT(client, "/api/relay/rest/phone_numbers"),
	}
}

// Search searches for available phone numbers with optional filter parameters
// such as area_code, contains, starts_with, etc.
func (r *PhoneNumbersNamespace) Search(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path("search"), params)
}
