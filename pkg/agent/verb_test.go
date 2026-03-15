package agent

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Pre-answer verbs
// ---------------------------------------------------------------------------

func TestPreAnswerVerbs_Add(t *testing.T) {
	a := NewAgentBase()
	a.AddPreAnswerVerb("play", map[string]any{"url": "https://example.com/ring.mp3"})
	if len(a.preAnswerVerbs) != 1 {
		t.Fatalf("expected 1 pre-answer verb, got %d", len(a.preAnswerVerbs))
	}
	if a.preAnswerVerbs[0].Name != "play" {
		t.Errorf("verb name = %q, want %q", a.preAnswerVerbs[0].Name, "play")
	}
}

func TestPreAnswerVerbs_Multiple(t *testing.T) {
	a := NewAgentBase()
	a.AddPreAnswerVerb("play", map[string]any{"url": "a.mp3"})
	a.AddPreAnswerVerb("sleep", map[string]any{"duration": 1000})
	if len(a.preAnswerVerbs) != 2 {
		t.Errorf("expected 2 pre-answer verbs, got %d", len(a.preAnswerVerbs))
	}
}

func TestPreAnswerVerbs_Clear(t *testing.T) {
	a := NewAgentBase()
	a.AddPreAnswerVerb("play", nil)
	a.ClearPreAnswerVerbs()
	if len(a.preAnswerVerbs) != 0 {
		t.Errorf("expected 0 pre-answer verbs after clear, got %d", len(a.preAnswerVerbs))
	}
}

func TestPreAnswerVerbs_RenderBeforeAnswer(t *testing.T) {
	a := NewAgentBase()
	a.AddPreAnswerVerb("play", map[string]any{"url": "ring.mp3"})
	doc := a.RenderSWML(nil, nil)

	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	// First verb should be the pre-answer play
	if len(main) < 2 {
		t.Fatalf("expected at least 2 verbs in main, got %d", len(main))
	}
	first, _ := main[0].(map[string]any)
	if _, ok := first["play"]; !ok {
		t.Errorf("first verb should be play, got %v", first)
	}
	second, _ := main[1].(map[string]any)
	if _, ok := second["answer"]; !ok {
		t.Errorf("second verb should be answer, got %v", second)
	}
}

// ---------------------------------------------------------------------------
// Post-answer verbs
// ---------------------------------------------------------------------------

func TestPostAnswerVerbs_Add(t *testing.T) {
	a := NewAgentBase()
	a.AddPostAnswerVerb("record_call", map[string]any{"format": "wav"})
	if len(a.postAnswerVerbs) != 1 {
		t.Fatalf("expected 1 post-answer verb, got %d", len(a.postAnswerVerbs))
	}
}

func TestPostAnswerVerbs_Clear(t *testing.T) {
	a := NewAgentBase()
	a.AddPostAnswerVerb("play", nil)
	a.ClearPostAnswerVerbs()
	if len(a.postAnswerVerbs) != 0 {
		t.Errorf("expected 0 post-answer verbs after clear, got %d", len(a.postAnswerVerbs))
	}
}

func TestPostAnswerVerbs_RenderAfterAnswer(t *testing.T) {
	a := NewAgentBase()
	a.AddPostAnswerVerb("play", map[string]any{"url": "welcome.mp3"})
	doc := a.RenderSWML(nil, nil)

	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	// Should be: answer, play, ai
	if len(main) < 3 {
		t.Fatalf("expected at least 3 verbs, got %d", len(main))
	}

	// Answer should come first
	if _, ok := main[0].(map[string]any)["answer"]; !ok {
		t.Error("first verb should be answer")
	}
	// Post-answer play should be between answer and ai
	if _, ok := main[1].(map[string]any)["play"]; !ok {
		t.Error("second verb should be the post-answer play")
	}
	// AI should be last
	if _, ok := main[2].(map[string]any)["ai"]; !ok {
		t.Error("third verb should be ai")
	}
}

// ---------------------------------------------------------------------------
// Post-AI verbs
// ---------------------------------------------------------------------------

func TestPostAiVerbs_Add(t *testing.T) {
	a := NewAgentBase()
	a.AddPostAiVerb("hangup", map[string]any{"reason": "complete"})
	if len(a.postAiVerbs) != 1 {
		t.Fatalf("expected 1 post-AI verb, got %d", len(a.postAiVerbs))
	}
}

func TestPostAiVerbs_Clear(t *testing.T) {
	a := NewAgentBase()
	a.AddPostAiVerb("hangup", nil)
	a.ClearPostAiVerbs()
	if len(a.postAiVerbs) != 0 {
		t.Errorf("expected 0 post-AI verbs after clear, got %d", len(a.postAiVerbs))
	}
}

func TestPostAiVerbs_RenderAfterAi(t *testing.T) {
	a := NewAgentBase()
	a.AddPostAiVerb("hangup", map[string]any{})
	doc := a.RenderSWML(nil, nil)

	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	lastVerb, _ := main[len(main)-1].(map[string]any)
	if _, ok := lastVerb["hangup"]; !ok {
		t.Errorf("last verb should be hangup, got %v", lastVerb)
	}
}

// ---------------------------------------------------------------------------
// Answer config
// ---------------------------------------------------------------------------

func TestAddAnswerVerb_MergesConfig(t *testing.T) {
	a := NewAgentBase()
	a.AddAnswerVerb(map[string]any{"ring_tone": true})
	if a.answerConfig["ring_tone"] != true {
		t.Error("expected ring_tone=true in answerConfig")
	}
}

func TestAddAnswerVerb_MultipleMerges(t *testing.T) {
	a := NewAgentBase()
	a.AddAnswerVerb(map[string]any{"ring_tone": true})
	a.AddAnswerVerb(map[string]any{"max_duration": 7200})
	if a.answerConfig["ring_tone"] != true {
		t.Error("ring_tone should persist")
	}
	if a.answerConfig["max_duration"] != 7200 {
		t.Errorf("max_duration = %v, want 7200", a.answerConfig["max_duration"])
	}
}

func TestAnswerConfig_RendersInSWML(t *testing.T) {
	a := NewAgentBase()
	a.AddAnswerVerb(map[string]any{"ring_tone": true})
	doc := a.RenderSWML(nil, nil)

	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	for _, v := range main {
		vm, _ := v.(map[string]any)
		if answerCfg, ok := vm["answer"].(map[string]any); ok {
			if answerCfg["ring_tone"] != true {
				t.Errorf("expected ring_tone=true in answer, got %v", answerCfg)
			}
			// Default max_duration should also be present
			if answerCfg["max_duration"] == nil {
				t.Error("expected max_duration in answer config")
			}
			return
		}
	}
	t.Fatal("answer verb not found in SWML")
}

// ---------------------------------------------------------------------------
// Auto-answer
// ---------------------------------------------------------------------------

func TestAutoAnswer_Default(t *testing.T) {
	a := NewAgentBase()
	if !a.autoAnswer {
		t.Error("expected autoAnswer=true by default")
	}
}

func TestAutoAnswer_Disabled(t *testing.T) {
	a := NewAgentBase(WithAutoAnswer(false))
	doc := a.RenderSWML(nil, nil)

	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	for _, v := range main {
		vm, _ := v.(map[string]any)
		if _, ok := vm["answer"]; ok {
			t.Error("answer verb should not be present when autoAnswer=false")
		}
	}
}

// ---------------------------------------------------------------------------
// Record call
// ---------------------------------------------------------------------------

func TestRecordCall_Enabled(t *testing.T) {
	a := NewAgentBase(WithRecordCall(true))
	doc := a.RenderSWML(nil, nil)

	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	found := false
	for _, v := range main {
		vm, _ := v.(map[string]any)
		if cfg, ok := vm["record_call"].(map[string]any); ok {
			found = true
			if cfg["format"] != "mp4" {
				t.Errorf("record format = %v, want mp4", cfg["format"])
			}
			if cfg["stereo"] != true {
				t.Errorf("record stereo = %v, want true", cfg["stereo"])
			}
		}
	}
	if !found {
		t.Error("record_call verb not found in SWML")
	}
}

func TestRecordCall_CustomFormat(t *testing.T) {
	a := NewAgentBase(WithRecordCall(true), WithRecordFormat("wav"), WithRecordStereo(false))
	doc := a.RenderSWML(nil, nil)

	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	for _, v := range main {
		vm, _ := v.(map[string]any)
		if cfg, ok := vm["record_call"].(map[string]any); ok {
			if cfg["format"] != "wav" {
				t.Errorf("record format = %v, want wav", cfg["format"])
			}
			if cfg["stereo"] != false {
				t.Errorf("record stereo = %v, want false", cfg["stereo"])
			}
			return
		}
	}
	t.Fatal("record_call not found")
}

// ---------------------------------------------------------------------------
// Method chaining
// ---------------------------------------------------------------------------

func TestVerbMethods_ReturnSelf(t *testing.T) {
	a := NewAgentBase()
	if a.AddPreAnswerVerb("x", nil) != a {
		t.Error("AddPreAnswerVerb should return self")
	}
	if a.AddAnswerVerb(nil) != a {
		t.Error("AddAnswerVerb should return self")
	}
	if a.AddPostAnswerVerb("x", nil) != a {
		t.Error("AddPostAnswerVerb should return self")
	}
	if a.AddPostAiVerb("x", nil) != a {
		t.Error("AddPostAiVerb should return self")
	}
	if a.ClearPreAnswerVerbs() != a {
		t.Error("ClearPreAnswerVerbs should return self")
	}
	if a.ClearPostAnswerVerbs() != a {
		t.Error("ClearPostAnswerVerbs should return self")
	}
	if a.ClearPostAiVerbs() != a {
		t.Error("ClearPostAiVerbs should return self")
	}
}
