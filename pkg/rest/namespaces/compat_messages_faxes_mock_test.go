// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_compat_messages_faxes.py.
//
// Covers Compat.Messages and Compat.Faxes media + update endpoints.

package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// ----------------- Messages -----------------

func TestCompatMessages_Update(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("returns_message_resource", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.Messages.Update("MM_TEST", map[string]any{
			"Body": "updated body",
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if _, hasBody := result["body"]; !hasBody {
			if _, hasSid := result["sid"]; !hasSid {
				t.Errorf("expected body or sid, got keys %v", keys(result))
			}
		}
	})

	t.Run("journal_records_post_to_message", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.Messages.Update("MM_U1", map[string]any{
			"Body":   "x",
			"Status": "canceled",
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "POST" {
			t.Errorf("method = %q, want POST", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/Messages/MM_U1"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
		body, ok := j.BodyMap()
		if !ok {
			t.Fatalf("body type = %T", j.Body)
		}
		if body["Body"] != "x" {
			t.Errorf("Body = %v, want x", body["Body"])
		}
		if body["Status"] != "canceled" {
			t.Errorf("Status = %v, want canceled", body["Status"])
		}
	})
}

func TestCompatMessages_GetMedia(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("returns_media_resource", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.Messages.GetMedia("MM_GM", "ME_GM")
		if err != nil {
			t.Fatalf("GetMedia: %v", err)
		}
		if _, hasContentType := result["content_type"]; !hasContentType {
			if _, hasSid := result["sid"]; !hasSid {
				t.Errorf("expected content_type or sid, got keys %v", keys(result))
			}
		}
	})

	t.Run("journal_records_get_to_media_path", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.Messages.GetMedia("MM_X", "ME_X")
		if err != nil {
			t.Fatalf("GetMedia: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "GET" {
			t.Errorf("method = %q, want GET", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/Messages/MM_X/Media/ME_X"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
	})
}

func TestCompatMessages_DeleteMedia(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("no_exception_on_delete", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.Messages.DeleteMedia("MM_DM", "ME_DM")
		if err != nil {
			t.Fatalf("DeleteMedia: %v", err)
		}
		if result == nil {
			t.Errorf("expected map, got nil")
		}
	})

	t.Run("journal_records_delete", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.Messages.DeleteMedia("MM_D", "ME_D")
		if err != nil {
			t.Fatalf("DeleteMedia: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "DELETE" {
			t.Errorf("method = %q, want DELETE", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/Messages/MM_D/Media/ME_D"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
	})
}

// ----------------- Faxes -----------------

func TestCompatFaxes_Update(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("returns_fax_resource", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.Faxes.Update("FX_U", map[string]any{
			"Status": "canceled",
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if _, hasStatus := result["status"]; !hasStatus {
			if _, hasDirection := result["direction"]; !hasDirection {
				t.Errorf("expected status or direction, got keys %v", keys(result))
			}
		}
	})

	t.Run("journal_records_post_with_status", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.Faxes.Update("FX_U2", map[string]any{
			"Status": "canceled",
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "POST" {
			t.Errorf("method = %q, want POST", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/Faxes/FX_U2"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
		body, ok := j.BodyMap()
		if !ok {
			t.Fatalf("body type = %T", j.Body)
		}
		if body["Status"] != "canceled" {
			t.Errorf("Status = %v, want canceled", body["Status"])
		}
	})
}

func TestCompatFaxes_ListMedia(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("returns_paginated_list", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.Faxes.ListMedia("FX_LM", nil)
		if err != nil {
			t.Fatalf("ListMedia: %v", err)
		}
		if _, ok := result["media"]; !ok {
			if _, ok := result["fax_media"]; !ok {
				t.Errorf("expected 'media' or 'fax_media', got keys %v", keys(result))
			}
		}
	})

	t.Run("journal_records_get_to_fax_media", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.Faxes.ListMedia("FX_LM_X", nil)
		if err != nil {
			t.Fatalf("ListMedia: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "GET" {
			t.Errorf("method = %q, want GET", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/Faxes/FX_LM_X/Media"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
	})
}

func TestCompatFaxes_GetMedia(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("returns_fax_media_resource", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.Faxes.GetMedia("FX_GM", "ME_GM")
		if err != nil {
			t.Fatalf("GetMedia: %v", err)
		}
		if _, hasContentType := result["content_type"]; !hasContentType {
			if _, hasSid := result["sid"]; !hasSid {
				t.Errorf("expected content_type or sid, got keys %v", keys(result))
			}
		}
	})

	t.Run("journal_records_get_to_specific_media", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.Faxes.GetMedia("FX_G", "ME_G")
		if err != nil {
			t.Fatalf("GetMedia: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "GET" {
			t.Errorf("method = %q, want GET", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/Faxes/FX_G/Media/ME_G"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
	})
}

func TestCompatFaxes_DeleteMedia(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	t.Run("no_exception_on_delete", func(t *testing.T) {
		mock.Reset(t)
		result, err := client.Compat.Faxes.DeleteMedia("FX_DM", "ME_DM")
		if err != nil {
			t.Fatalf("DeleteMedia: %v", err)
		}
		if result == nil {
			t.Error("expected map, got nil")
		}
	})

	t.Run("journal_records_delete", func(t *testing.T) {
		mock.Reset(t)
		_, err := client.Compat.Faxes.DeleteMedia("FX_D", "ME_D")
		if err != nil {
			t.Fatalf("DeleteMedia: %v", err)
		}
		j := mock.Last(t)
		if j.Method != "DELETE" {
			t.Errorf("method = %q, want DELETE", j.Method)
		}
		const wantPath = "/api/laml/2010-04-01/Accounts/test_proj/Faxes/FX_D/Media/ME_D"
		if j.Path != wantPath {
			t.Errorf("path = %q, want %q", j.Path, wantPath)
		}
	})
}
