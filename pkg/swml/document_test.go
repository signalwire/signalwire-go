package swml

import (
	"encoding/json"
	"testing"
)

func TestNewDocument(t *testing.T) {
	doc := NewDocument()
	if doc.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", doc.Version, "1.0.0")
	}
	if _, ok := doc.Sections["main"]; !ok {
		t.Error("missing 'main' section")
	}
}

func TestDocumentReset(t *testing.T) {
	doc := NewDocument()
	if err := doc.AddVerb("play", map[string]any{"url": "test"}); err != nil {
		t.Fatalf("AddVerb: %v", err)
	}
	doc.AddSection("extra")

	doc.Reset()

	if len(doc.Sections) != 1 {
		t.Errorf("sections count = %d, want 1", len(doc.Sections))
	}
	if len(doc.Sections["main"]) != 0 {
		t.Error("main section should be empty after reset")
	}
}

func TestAddSection(t *testing.T) {
	doc := NewDocument()
	if !doc.AddSection("custom") {
		t.Error("AddSection should return true for new section")
	}
	if doc.AddSection("custom") {
		t.Error("AddSection should return false for existing section")
	}
	if !doc.HasSection("custom") {
		t.Error("HasSection should return true after adding")
	}
}

func TestAddVerb(t *testing.T) {
	doc := NewDocument()
	err := doc.AddVerb("play", map[string]any{"url": "https://example.com/audio.mp3"})
	if err != nil {
		t.Fatalf("AddVerb failed: %v", err)
	}

	verbs := doc.GetVerbs("main")
	if len(verbs) != 1 {
		t.Fatalf("expected 1 verb, got %d", len(verbs))
	}
	if _, ok := verbs[0]["play"]; !ok {
		t.Error("verb should have 'play' key")
	}
}

func TestAddVerbEmptyName(t *testing.T) {
	doc := NewDocument()
	err := doc.AddVerb("", map[string]any{})
	if err == nil {
		t.Error("expected error for empty verb name")
	}
}

func TestAddVerbToSection(t *testing.T) {
	doc := NewDocument()
	// Adding to non-existent section creates it
	err := doc.AddVerbToSection("custom", "play", map[string]any{"url": "test"})
	if err != nil {
		t.Fatalf("AddVerbToSection failed: %v", err)
	}
	if !doc.HasSection("custom") {
		t.Error("section should be auto-created")
	}
	verbs := doc.GetVerbs("custom")
	if len(verbs) != 1 {
		t.Fatalf("expected 1 verb in custom section, got %d", len(verbs))
	}
}

func TestGetVerbsNonExistent(t *testing.T) {
	doc := NewDocument()
	verbs := doc.GetVerbs("nonexistent")
	if verbs != nil {
		t.Error("expected nil for non-existent section")
	}
}

func TestToMap(t *testing.T) {
	doc := NewDocument()
	if err := doc.AddVerb("answer", map[string]any{"max_duration": 300}); err != nil {
		t.Fatalf("AddVerb answer: %v", err)
	}
	if err := doc.AddVerb("play", map[string]any{"url": "https://example.com/audio.mp3"}); err != nil {
		t.Fatalf("AddVerb play: %v", err)
	}

	m := doc.ToMap()
	if m["version"] != "1.0.0" {
		t.Errorf("version = %v, want 1.0.0", m["version"])
	}
	sections, ok := m["sections"].(map[string]any)
	if !ok {
		t.Fatal("sections should be a map")
	}
	main, ok := sections["main"].([]any)
	if !ok {
		t.Fatal("main should be an array")
	}
	if len(main) != 2 {
		t.Errorf("main verbs = %d, want 2", len(main))
	}
}

func TestRender(t *testing.T) {
	doc := NewDocument()
	if err := doc.AddVerb("answer", map[string]any{}); err != nil {
		t.Fatalf("AddVerb answer: %v", err)
	}
	if err := doc.AddVerb("hangup", map[string]any{}); err != nil {
		t.Fatalf("AddVerb hangup: %v", err)
	}

	rendered, err := doc.Render()
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(rendered), &parsed); err != nil {
		t.Fatalf("rendered document is not valid JSON: %v", err)
	}
	if parsed["version"] != "1.0.0" {
		t.Errorf("version = %v, want 1.0.0", parsed["version"])
	}
}

func TestMarshalJSON(t *testing.T) {
	doc := NewDocument()
	if err := doc.AddVerb("play", map[string]any{"url": "test"}); err != nil {
		t.Fatalf("AddVerb: %v", err)
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if parsed["version"] != "1.0.0" {
		t.Error("MarshalJSON should produce valid SWML")
	}
}
