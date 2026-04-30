// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_compat_phone_numbers.py.
//
// Covers the 8 uncovered Compat.PhoneNumbers symbols:
//   - List, Get, Update, Delete (basic CRUD over IncomingPhoneNumbers)
//   - Purchase, ImportNumber (phone-number provisioning)
//   - ListAvailableCountries, SearchTollFree

package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

func TestCompatPhoneNumbers_List(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("returns_paginated_list", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.PhoneNumbers.List(nil)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		raw, ok := result["incoming_phone_numbers"]
		if !ok {
			t.Fatalf("expected 'incoming_phone_numbers', got keys %v", keys(result))
		}
		if _, isList := raw.([]any); !isList {
			t.Errorf("incoming_phone_numbers type = %T, want []any", raw)
		}
	})

	t.Run("journal_records_get", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.PhoneNumbers.List(nil)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "GET" {
			t.Errorf("method = %q, want GET", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/IncomingPhoneNumbers"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
	})
}

func TestCompatPhoneNumbers_Get(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("returns_phone_number_resource", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.PhoneNumbers.Get("PN_TEST")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if _, hasNumber := result["phone_number"]; !hasNumber {
			if _, hasSid := result["sid"]; !hasSid {
				t.Errorf("expected phone_number or sid, got keys %v", keys(result))
			}
		}
	})

	t.Run("journal_records_get_with_sid", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.PhoneNumbers.Get("PN_GET")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "GET" {
			t.Errorf("method = %q, want GET", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/IncomingPhoneNumbers/PN_GET"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
	})
}

func TestCompatPhoneNumbers_Update(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("returns_phone_number_resource", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.PhoneNumbers.Update("PN_U", map[string]any{
			"FriendlyName": "updated",
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if _, hasNumber := result["phone_number"]; !hasNumber {
			if _, hasSid := result["sid"]; !hasSid {
				t.Errorf("expected phone_number or sid, got keys %v", keys(result))
			}
		}
	})

	t.Run("journal_records_post_with_friendly_name", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.PhoneNumbers.Update("PN_UU", map[string]any{
			"FriendlyName": "updated",
			"VoiceUrl":     "https://a.b/v",
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "POST" {
			t.Errorf("method = %q, want POST", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/IncomingPhoneNumbers/PN_UU"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
		body, ok := j.BodyMap()
		if !ok {
			t.Fatalf("body type = %T", j.Body)
		}
		if body["FriendlyName"] != "updated" {
			t.Errorf("FriendlyName = %v, want updated", body["FriendlyName"])
		}
		if body["VoiceUrl"] != "https://a.b/v" {
			t.Errorf("VoiceUrl = %v", body["VoiceUrl"])
		}
	})
}

func TestCompatPhoneNumbers_Delete(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("no_exception_on_delete", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.PhoneNumbers.Delete("PN_D")
		if err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if result == nil {
			t.Error("expected map, got nil")
		}
	})

	t.Run("journal_records_delete", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.PhoneNumbers.Delete("PN_DEL")
		if err != nil {
			t.Fatalf("Delete: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "DELETE" {
			t.Errorf("method = %q, want DELETE", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/IncomingPhoneNumbers/PN_DEL"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
	})
}

func TestCompatPhoneNumbers_Purchase(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("returns_purchased_number", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.PhoneNumbers.Purchase(map[string]any{
			"PhoneNumber": "+15555550100",
		})
		if err != nil {
			t.Fatalf("Purchase: %v", err)
		}
		if _, hasNumber := result["phone_number"]; !hasNumber {
			if _, hasSid := result["sid"]; !hasSid {
				t.Errorf("expected phone_number or sid, got keys %v", keys(result))
			}
		}
	})

	t.Run("journal_records_post_with_phone_number", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.PhoneNumbers.Purchase(map[string]any{
			"PhoneNumber":  "+15555550100",
			"FriendlyName": "Main",
		})
		if err != nil {
			t.Fatalf("Purchase: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "POST" {
			t.Errorf("method = %q, want POST", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/IncomingPhoneNumbers"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
		body, ok := j.BodyMap()
		if !ok {
			t.Fatalf("body type = %T", j.Body)
		}
		if body["PhoneNumber"] != "+15555550100" {
			t.Errorf("PhoneNumber = %v", body["PhoneNumber"])
		}
		if body["FriendlyName"] != "Main" {
			t.Errorf("FriendlyName = %v", body["FriendlyName"])
		}
	})
}

func TestCompatPhoneNumbers_ImportNumber(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("returns_imported_number", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.PhoneNumbers.ImportNumber(map[string]any{
			"PhoneNumber": "+15555550111",
		})
		if err != nil {
			t.Fatalf("ImportNumber: %v", err)
		}
		if _, hasNumber := result["phone_number"]; !hasNumber {
			if _, hasSid := result["sid"]; !hasSid {
				t.Errorf("expected phone_number or sid, got keys %v", keys(result))
			}
		}
	})

	t.Run("journal_records_post_to_imported", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.PhoneNumbers.ImportNumber(map[string]any{
			"PhoneNumber": "+15555550111",
			"VoiceUrl":    "https://a.b/v",
		})
		if err != nil {
			t.Fatalf("ImportNumber: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "POST" {
			t.Errorf("method = %q, want POST", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/ImportedPhoneNumbers"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
		body, ok := j.BodyMap()
		if !ok {
			t.Fatalf("body type = %T", j.Body)
		}
		if body["PhoneNumber"] != "+15555550111" {
			t.Errorf("PhoneNumber = %v", body["PhoneNumber"])
		}
	})
}

func TestCompatPhoneNumbers_ListAvailableCountries(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("returns_countries_collection", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.PhoneNumbers.ListAvailableCountries(nil)
		if err != nil {
			t.Fatalf("ListAvailableCountries: %v", err)
		}
		raw, ok := result["countries"]
		if !ok {
			t.Fatalf("expected 'countries', got keys %v", keys(result))
		}
		if _, isList := raw.([]any); !isList {
			t.Errorf("countries type = %T, want []any", raw)
		}
	})

	t.Run("journal_records_get", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.PhoneNumbers.ListAvailableCountries(nil)
		if err != nil {
			t.Fatalf("ListAvailableCountries: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "GET" {
			t.Errorf("method = %q, want GET", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/AvailablePhoneNumbers"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
	})
}

func TestCompatPhoneNumbers_SearchTollFree(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("returns_available_numbers", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.PhoneNumbers.SearchTollFree("US", map[string]string{
			"AreaCode": "800",
		})
		if err != nil {
			t.Fatalf("SearchTollFree: %v", err)
		}
		raw, ok := result["available_phone_numbers"]
		if !ok {
			t.Fatalf("expected 'available_phone_numbers', got keys %v", keys(result))
		}
		if _, isList := raw.([]any); !isList {
			t.Errorf("available_phone_numbers type = %T", raw)
		}
	})

	t.Run("journal_records_get_with_country_in_path", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.PhoneNumbers.SearchTollFree("US", map[string]string{
			"AreaCode": "888",
		})
		if err != nil {
			t.Fatalf("SearchTollFree: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "GET" {
			t.Errorf("method = %q, want GET", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/AvailablePhoneNumbers/US/TollFree"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
		areaCodes, ok := j.QueryParams["AreaCode"]
		if !ok || len(areaCodes) == 0 || areaCodes[0] != "888" {
			t.Errorf("query AreaCode = %v, want [888]", areaCodes)
		}
	})
}
