// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Full success + error coverage for the datasphere.* canonical routes
// (9 endpoints). Each route gets a success test (asserting response body,
// wire method/path, and matched_route == endpoint_id) and an error test
// (asserting *rest.SignalWireRestError StatusCode + journal response_status
// + matched_route).
//
// Routes covered:
//   datasphere.list_documents        GET    /api/datasphere/documents
//   datasphere.create_document       POST   /api/datasphere/documents
//   datasphere.search_documents      POST   /api/datasphere/documents/search
//   datasphere.list_document_chunks  GET    /api/datasphere/documents/{id}/chunks
//   datasphere.get_document_chunk    GET    /api/datasphere/documents/{id}/chunks/{cid}
//   datasphere.delete_document_chunk DELETE /api/datasphere/documents/{id}/chunks/{cid}
//   datasphere.get_document          GET    /api/datasphere/documents/{id}
//   datasphere.update_document       PATCH  /api/datasphere/documents/{id}
//   datasphere.delete_document       DELETE /api/datasphere/documents/{id}

package namespaces_test

import (
	"errors"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest"
	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// ---------- datasphere.list_documents ----------

func TestDatasphereCov_ListDocuments_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Datasphere.Documents.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("missing 'data', got keys %v", keys(body))
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	if j.Path != "/api/datasphere/documents" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.list_documents" {
		t.Errorf("matched_route = %v, want datasphere.list_documents", j.MatchedRoute)
	}
}

func TestDatasphereCov_ListDocuments_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "datasphere.list_documents", 500, map[string]any{"error": "boom"})
	_, err := client.Datasphere.Documents.List(nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("status = %d, want 500", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.list_documents" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 500 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- datasphere.create_document ----------

func TestDatasphereCov_CreateDocument_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Datasphere.Documents.Create(map[string]any{"name": "doc-1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	if j.Path != "/api/datasphere/documents" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.create_document" {
		t.Errorf("matched_route = %v, want datasphere.create_document", j.MatchedRoute)
	}
	reqBody, ok := j.BodyMap()
	if !ok || reqBody["name"] != "doc-1" {
		t.Errorf("request body name = %v", reqBody["name"])
	}
}

func TestDatasphereCov_CreateDocument_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "datasphere.create_document", 422, map[string]any{"error": "invalid"})
	_, err := client.Datasphere.Documents.Create(map[string]any{"name": ""})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 422 {
		t.Errorf("status = %d, want 422", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.create_document" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 422 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- datasphere.search_documents ----------

func TestDatasphereCov_SearchDocuments_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Datasphere.Documents.Search(map[string]any{"query": "hello"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	if j.Path != "/api/datasphere/documents/search" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.search_documents" {
		t.Errorf("matched_route = %v, want datasphere.search_documents", j.MatchedRoute)
	}
	reqBody, ok := j.BodyMap()
	if !ok || reqBody["query"] != "hello" {
		t.Errorf("request body query = %v", reqBody["query"])
	}
}

func TestDatasphereCov_SearchDocuments_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "datasphere.search_documents", 422, map[string]any{"error": "bad query"})
	_, err := client.Datasphere.Documents.Search(map[string]any{"query": ""})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 422 {
		t.Errorf("status = %d, want 422", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.search_documents" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 422 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- datasphere.list_document_chunks ----------

func TestDatasphereCov_ListChunks_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Datasphere.Documents.ListChunks("doc-1", nil)
	if err != nil {
		t.Fatalf("ListChunks: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	if j.Path != "/api/datasphere/documents/doc-1/chunks" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.list_document_chunks" {
		t.Errorf("matched_route = %v, want datasphere.list_document_chunks", j.MatchedRoute)
	}
}

func TestDatasphereCov_ListChunks_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "datasphere.list_document_chunks", 404, map[string]any{"error": "no doc"})
	_, err := client.Datasphere.Documents.ListChunks("missing", nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.list_document_chunks" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- datasphere.get_document_chunk ----------

func TestDatasphereCov_GetChunk_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Datasphere.Documents.GetChunk("doc-1", "chunk-9", nil)
	if err != nil {
		t.Fatalf("GetChunk: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	if j.Path != "/api/datasphere/documents/doc-1/chunks/chunk-9" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.get_document_chunk" {
		t.Errorf("matched_route = %v, want datasphere.get_document_chunk", j.MatchedRoute)
	}
}

func TestDatasphereCov_GetChunk_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "datasphere.get_document_chunk", 404, map[string]any{"error": "no chunk"})
	_, err := client.Datasphere.Documents.GetChunk("doc-1", "missing", nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.get_document_chunk" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- datasphere.delete_document_chunk ----------

func TestDatasphereCov_DeleteChunk_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Datasphere.Documents.DeleteChunk("doc-1", "chunk-9")
	if err != nil {
		t.Fatalf("DeleteChunk: %v", err)
	}
	if body == nil {
		t.Error("expected map (204 normalized to {}), got nil")
	}
	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q, want DELETE", j.Method)
	}
	if j.Path != "/api/datasphere/documents/doc-1/chunks/chunk-9" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.delete_document_chunk" {
		t.Errorf("matched_route = %v, want datasphere.delete_document_chunk", j.MatchedRoute)
	}
}

func TestDatasphereCov_DeleteChunk_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "datasphere.delete_document_chunk", 404, map[string]any{"error": "no chunk"})
	_, err := client.Datasphere.Documents.DeleteChunk("doc-1", "missing")
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.delete_document_chunk" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- datasphere.get_document ----------

func TestDatasphereCov_GetDocument_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Datasphere.Documents.Get("doc-7")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if body == nil {
		t.Error("expected map, got nil")
	}
	j := mock.Last(t)
	if j.Method != "GET" {
		t.Errorf("method = %q, want GET", j.Method)
	}
	if j.Path != "/api/datasphere/documents/doc-7" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.get_document" {
		t.Errorf("matched_route = %v, want datasphere.get_document", j.MatchedRoute)
	}
}

func TestDatasphereCov_GetDocument_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "datasphere.get_document", 404, map[string]any{"error": "not found"})
	_, err := client.Datasphere.Documents.Get("missing")
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.get_document" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- datasphere.update_document (PATCH) ----------

func TestDatasphereCov_UpdateDocument_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Datasphere.Documents.Update("doc-7", map[string]any{"name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	j := mock.Last(t)
	if j.Method != "PATCH" {
		t.Errorf("method = %q, want PATCH", j.Method)
	}
	if j.Path != "/api/datasphere/documents/doc-7" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.update_document" {
		t.Errorf("matched_route = %v, want datasphere.update_document", j.MatchedRoute)
	}
	reqBody, ok := j.BodyMap()
	if !ok || reqBody["name"] != "renamed" {
		t.Errorf("request body name = %v", reqBody["name"])
	}
}

func TestDatasphereCov_UpdateDocument_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "datasphere.update_document", 404, map[string]any{"error": "not found"})
	_, err := client.Datasphere.Documents.Update("missing", map[string]any{"name": "x"})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.update_document" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}

// ---------- datasphere.delete_document ----------

func TestDatasphereCov_DeleteDocument_Success(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Datasphere.Documents.Delete("doc-7")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if body == nil {
		t.Error("expected map (204 normalized to {}), got nil")
	}
	j := mock.Last(t)
	if j.Method != "DELETE" {
		t.Errorf("method = %q, want DELETE", j.Method)
	}
	if j.Path != "/api/datasphere/documents/doc-7" {
		t.Errorf("path = %q", j.Path)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.delete_document" {
		t.Errorf("matched_route = %v, want datasphere.delete_document", j.MatchedRoute)
	}
}

func TestDatasphereCov_DeleteDocument_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "datasphere.delete_document", 404, map[string]any{"error": "not found"})
	_, err := client.Datasphere.Documents.Delete("missing")
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", restErr.StatusCode)
	}
	j := mock.Last(t)
	if j.MatchedRoute == nil || *j.MatchedRoute != "datasphere.delete_document" {
		t.Errorf("matched_route = %v", j.MatchedRoute)
	}
	if j.ResponseStatus == nil || *j.ResponseStatus != 404 {
		t.Errorf("response_status = %v", j.ResponseStatus)
	}
}
