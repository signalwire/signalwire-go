//go:build ignore

// Example: Upload a document to Datasphere and run a semantic search.
//
// Set these env vars (or pass them directly to NewRestClient):
//
//	SIGNALWIRE_PROJECT_ID   - your SignalWire project ID
//	SIGNALWIRE_API_TOKEN    - your SignalWire API token
//	SIGNALWIRE_SPACE        - your SignalWire space (e.g. example.signalwire.com)
//
// For full HTTP debug output:
//
//	SIGNALWIRE_LOG_LEVEL=debug
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/signalwire/signalwire-go/pkg/rest"
)

func main() {
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}

	// 1. Upload a document (a publicly accessible text file)
	fmt.Println("Uploading document to Datasphere...")
	doc, err := client.Datasphere.Documents.Create(map[string]any{
		"url":  "https://filesamples.com/samples/document/txt/sample3.txt",
		"tags": []string{"support", "demo"},
	})
	if err != nil {
		fmt.Printf("  Upload failed: %v\n", err)
		return
	}
	docID := doc["id"].(string)
	fmt.Printf("  Document created: %s (status: %v)\n", docID, doc["status"])

	// 2. Wait for vectorization to complete
	fmt.Println("\nWaiting for document to be vectorized...")
	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)
		docStatus, err := client.Datasphere.Documents.Get(docID)
		if err != nil {
			fmt.Printf("  Poll error: %v\n", err)
			continue
		}
		status, _ := docStatus["status"].(string)
		fmt.Printf("  Poll %d: status=%s\n", i+1, status)
		if status == "completed" {
			fmt.Printf("  Vectorized! Chunks: %v\n", docStatus["number_of_chunks"])
			break
		}
		if status == "error" || status == "failed" {
			fmt.Printf("  Document processing failed: %s\n", status)
			client.Datasphere.Documents.Delete(docID)
			return
		}
		if i == 29 {
			fmt.Println("  Timed out waiting for vectorization.")
			client.Datasphere.Documents.Delete(docID)
			return
		}
	}

	// 3. List chunks
	fmt.Printf("\nListing chunks for document %s...\n", docID)
	chunks, err := client.Datasphere.Documents.ListChunks(docID, nil)
	if err != nil {
		fmt.Printf("  List chunks failed: %v\n", err)
	} else if data, ok := chunks["data"].([]any); ok {
		limit := 5
		if len(data) < limit {
			limit = len(data)
		}
		for _, c := range data[:limit] {
			if m, ok := c.(map[string]any); ok {
				content, _ := m["content"].(string)
				if len(content) > 80 {
					content = content[:80]
				}
				fmt.Printf("  - Chunk %v: %s...\n", m["id"], content)
			}
		}
	}

	// 4. Semantic search across all documents
	fmt.Println("\nSearching Datasphere...")
	results, err := client.Datasphere.Documents.Search(map[string]any{
		"query_string": "lorem ipsum dolor sit amet",
		"count":        3,
	})
	if err != nil {
		fmt.Printf("  Search failed: %v\n", err)
	} else if chunkList, ok := results["chunks"].([]any); ok {
		for _, c := range chunkList {
			if m, ok := c.(map[string]any); ok {
				text, _ := m["text"].(string)
				if len(text) > 100 {
					text = text[:100]
				}
				fmt.Printf("  - %s...\n", text)
			}
		}
	}

	// 5. Clean up
	fmt.Printf("\nDeleting document %s...\n", docID)
	if err := client.Datasphere.Documents.Delete(docID); err != nil {
		fmt.Printf("  Delete failed: %v\n", err)
	} else {
		fmt.Println("  Deleted.")
	}
}
