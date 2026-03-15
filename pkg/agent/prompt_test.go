package agent

import (
	"testing"
)

// ---------------------------------------------------------------------------
// SetPromptText isolation
// ---------------------------------------------------------------------------

func TestPromptText_DisablesPom(t *testing.T) {
	a := NewAgentBase()
	if !a.usePom {
		t.Fatal("expected usePom=true by default")
	}
	a.SetPromptText("Hello world")
	if a.usePom {
		t.Error("SetPromptText should disable POM mode")
	}
	if a.promptText != "Hello world" {
		t.Errorf("promptText = %q, want %q", a.promptText, "Hello world")
	}
}

func TestPromptText_GetPromptReturnsString(t *testing.T) {
	a := NewAgentBase()
	a.SetPromptText("raw prompt")
	result := a.GetPrompt()
	s, ok := result.(string)
	if !ok {
		t.Fatalf("GetPrompt returned %T, want string", result)
	}
	if s != "raw prompt" {
		t.Errorf("GetPrompt() = %q, want %q", s, "raw prompt")
	}
}

func TestPromptText_Overwrite(t *testing.T) {
	a := NewAgentBase()
	a.SetPromptText("first")
	a.SetPromptText("second")
	s, _ := a.GetPrompt().(string)
	if s != "second" {
		t.Errorf("expected second prompt, got %q", s)
	}
}

// ---------------------------------------------------------------------------
// POM section operations
// ---------------------------------------------------------------------------

func TestPomAddSection_SetsUsePom(t *testing.T) {
	a := NewAgentBase()
	a.usePom = false
	a.PromptAddSection("Title", "Body", nil)
	if !a.usePom {
		t.Error("PromptAddSection should enable POM mode")
	}
}

func TestPomAddSection_TitleOnly(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("Title", "", nil)
	sections := a.pomSections
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if sections[0]["title"] != "Title" {
		t.Errorf("title = %v, want %q", sections[0]["title"], "Title")
	}
	if _, ok := sections[0]["body"]; ok {
		t.Error("empty body should not be stored")
	}
	if _, ok := sections[0]["bullets"]; ok {
		t.Error("nil bullets should not be stored")
	}
}

func TestPomAddSection_WithBody(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("Title", "Body text", nil)
	if a.pomSections[0]["body"] != "Body text" {
		t.Errorf("body = %v, want %q", a.pomSections[0]["body"], "Body text")
	}
}

func TestPomAddSection_WithBullets(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("Title", "", []string{"a", "b", "c"})
	bullets, ok := a.pomSections[0]["bullets"].([]string)
	if !ok {
		t.Fatalf("bullets type = %T, want []string", a.pomSections[0]["bullets"])
	}
	if len(bullets) != 3 {
		t.Errorf("len(bullets) = %d, want 3", len(bullets))
	}
}

func TestPomAddSection_MultipleSections(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("A", "", nil)
	a.PromptAddSection("B", "", nil)
	a.PromptAddSection("C", "", nil)
	if len(a.pomSections) != 3 {
		t.Errorf("expected 3 sections, got %d", len(a.pomSections))
	}
}

func TestPomHasSection_NotFound(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("Exists", "", nil)
	if a.PromptHasSection("Missing") {
		t.Error("PromptHasSection should return false for non-existent section")
	}
}

func TestPomAddToSection_ExistingBody(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("Title", "Line 1", nil)
	a.PromptAddToSection("Title", "Line 2")
	body, _ := a.pomSections[0]["body"].(string)
	if body != "Line 1\nLine 2" {
		t.Errorf("body = %q, want %q", body, "Line 1\nLine 2")
	}
}

func TestPomAddToSection_EmptyBody(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("Title", "", nil)
	a.PromptAddToSection("Title", "New body")
	body, _ := a.pomSections[0]["body"].(string)
	if body != "New body" {
		t.Errorf("body = %q, want %q", body, "New body")
	}
}

func TestPomAddToSection_NonExistent(t *testing.T) {
	a := NewAgentBase()
	// Should be a no-op when section doesn't exist
	a.PromptAddToSection("Missing", "text")
	if len(a.pomSections) != 0 {
		t.Error("should not create new section")
	}
}

func TestPomAddSubsection_Basic(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("Parent", "", nil)
	a.PromptAddSubsection("Parent", "Child", "child body", []string{"bullet1"})

	subs, ok := a.pomSections[0]["subsections"].([]map[string]any)
	if !ok {
		t.Fatalf("subsections type = %T, want []map[string]any", a.pomSections[0]["subsections"])
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 subsection, got %d", len(subs))
	}
	if subs[0]["title"] != "Child" {
		t.Errorf("subsection title = %v, want %q", subs[0]["title"], "Child")
	}
	if subs[0]["body"] != "child body" {
		t.Errorf("subsection body = %v, want %q", subs[0]["body"], "child body")
	}
}

func TestPomAddSubsection_MultipleChildren(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("Parent", "", nil)
	a.PromptAddSubsection("Parent", "Child1", "", nil)
	a.PromptAddSubsection("Parent", "Child2", "", nil)

	subs, _ := a.pomSections[0]["subsections"].([]map[string]any)
	if len(subs) != 2 {
		t.Errorf("expected 2 subsections, got %d", len(subs))
	}
}

func TestPomAddSubsection_NoParent(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSubsection("Missing", "Child", "", nil)
	if len(a.pomSections) != 0 {
		t.Error("should not create parent or subsection for missing parent")
	}
}

// ---------------------------------------------------------------------------
// SetPromptPom
// ---------------------------------------------------------------------------

func TestSetPromptPom_ReplacesExisting(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("Old", "", nil)
	newPom := []map[string]any{
		{"title": "New1"},
		{"title": "New2"},
	}
	a.SetPromptPom(newPom)
	if len(a.pomSections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(a.pomSections))
	}
	if a.pomSections[0]["title"] != "New1" {
		t.Errorf("first section title = %v", a.pomSections[0]["title"])
	}
	if !a.usePom {
		t.Error("SetPromptPom should enable POM mode")
	}
}

// ---------------------------------------------------------------------------
// GetPrompt modes
// ---------------------------------------------------------------------------

func TestGetPrompt_PomMode(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("S1", "B1", nil)
	result := a.GetPrompt()
	sections, ok := result.([]map[string]any)
	if !ok {
		t.Fatalf("GetPrompt returned %T, want []map[string]any", result)
	}
	if len(sections) != 1 {
		t.Errorf("expected 1 section, got %d", len(sections))
	}
}

func TestGetPrompt_ReturnsCopy(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("S1", "", nil)
	result := a.GetPrompt().([]map[string]any)
	result = append(result, map[string]any{"title": "Extra"})
	// Original should be unmodified
	orig := a.GetPrompt().([]map[string]any)
	if len(orig) != 1 {
		t.Errorf("modifying result should not affect agent; got %d sections", len(orig))
	}
}

// ---------------------------------------------------------------------------
// SetPostPrompt
// ---------------------------------------------------------------------------

func TestSetPostPrompt_Basic(t *testing.T) {
	a := NewAgentBase()
	a.SetPostPrompt("Summarize the call")
	if a.postPrompt != "Summarize the call" {
		t.Errorf("postPrompt = %q, want %q", a.postPrompt, "Summarize the call")
	}
}

func TestSetPostPrompt_EmptyString(t *testing.T) {
	a := NewAgentBase()
	a.SetPostPrompt("something")
	a.SetPostPrompt("")
	if a.postPrompt != "" {
		t.Errorf("postPrompt should be empty, got %q", a.postPrompt)
	}
}

// ---------------------------------------------------------------------------
// POM renders in SWML
// ---------------------------------------------------------------------------

func TestPrompt_RendersAsPomInSWML(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("Role", "You are a helpful assistant", nil)
	doc := a.RenderSWML(nil, nil)

	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	for _, v := range main {
		vm, _ := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			prompt, _ := aiCfg["prompt"].(map[string]any)
			if prompt == nil {
				t.Fatal("expected prompt in AI config")
			}
			pom, ok := prompt["pom"].([]map[string]any)
			if !ok {
				t.Fatal("expected pom key in prompt")
			}
			if len(pom) == 0 {
				t.Fatal("expected non-empty pom array")
			}
			if pom[0]["title"] != "Role" {
				t.Errorf("pom section title = %v", pom[0]["title"])
			}
			return
		}
	}
	t.Fatal("AI verb not found in SWML")
}

func TestPrompt_RendersAsTextInSWML(t *testing.T) {
	a := NewAgentBase()
	a.SetPromptText("You are a bot")
	doc := a.RenderSWML(nil, nil)

	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	for _, v := range main {
		vm, _ := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			prompt, _ := aiCfg["prompt"].(map[string]any)
			if prompt == nil {
				t.Fatal("expected prompt in AI config")
			}
			text, _ := prompt["text"].(string)
			if text != "You are a bot" {
				t.Errorf("prompt text = %q, want %q", text, "You are a bot")
			}
			return
		}
	}
	t.Fatal("AI verb not found in SWML")
}

// ---------------------------------------------------------------------------
// Method chaining
// ---------------------------------------------------------------------------

func TestPromptMethods_ReturnSelf(t *testing.T) {
	a := NewAgentBase()
	ret := a.SetPromptText("x")
	if ret != a {
		t.Error("SetPromptText should return self")
	}
	ret = a.SetPostPrompt("y")
	if ret != a {
		t.Error("SetPostPrompt should return self")
	}
	ret = a.PromptAddSection("T", "", nil)
	if ret != a {
		t.Error("PromptAddSection should return self")
	}
	ret = a.PromptAddToSection("T", "text")
	if ret != a {
		t.Error("PromptAddToSection should return self")
	}
	ret = a.PromptAddSubsection("T", "S", "", nil)
	if ret != a {
		t.Error("PromptAddSubsection should return self")
	}
	ret = a.SetPromptPom(nil)
	if ret != a {
		t.Error("SetPromptPom should return self")
	}
}
