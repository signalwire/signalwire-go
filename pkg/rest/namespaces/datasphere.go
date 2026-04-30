// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// DatasphereDocuments provides document management with search and chunk
// operations for the Datasphere API.
type DatasphereDocuments struct {
	*CrudResource
}

// Search performs a semantic search across documents.
func (r *DatasphereDocuments) Search(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path("search"), data, nil)
}

// ListChunks lists chunks for a specific document.
func (r *DatasphereDocuments) ListChunks(documentID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(documentID, "chunks"), params)
}

// GetChunk retrieves a specific chunk from a document.
func (r *DatasphereDocuments) GetChunk(documentID, chunkID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(documentID, "chunks", chunkID), nil)
}

// DeleteChunk deletes a specific chunk from a document.
func (r *DatasphereDocuments) DeleteChunk(documentID, chunkID string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(documentID, "chunks", chunkID))
}

// DatasphereNamespace groups Datasphere API resources.
type DatasphereNamespace struct {
	Documents *DatasphereDocuments
}

// NewDatasphereNamespace creates a new DatasphereNamespace.
func NewDatasphereNamespace(client HTTPClient) *DatasphereNamespace {
	return &DatasphereNamespace{
		Documents: &DatasphereDocuments{
			CrudResource: NewCrudResource(client, "/api/datasphere/documents"),
		},
	}
}
