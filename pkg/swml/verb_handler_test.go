// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package swml

import (
	"testing"
)

// --- Service registry: RegisterVerbHandler / GetVerbHandler / HasVerbHandler ---

// stubVerbHandler implements VerbHandler for registry tests.
type stubVerbHandler struct {
	name string
}

func (s *stubVerbHandler) GetVerbName() string { return s.name }
func (s *stubVerbHandler) ValidateConfig(map[string]any) (bool, []string) {
	return true, nil
}
func (s *stubVerbHandler) BuildConfig(params map[string]any) (map[string]any, error) {
	return params, nil
}

func TestServiceRegisterVerbHandlerStoresAndLooksUp(t *testing.T) {
	svc := NewService(WithName("test"))
	h := &stubVerbHandler{name: "custom_verb"}

	if svc.HasVerbHandler("custom_verb") {
		t.Fatalf("HasVerbHandler should be false before registration")
	}
	if got := svc.GetVerbHandler("custom_verb"); got != nil {
		t.Fatalf("GetVerbHandler should return nil before registration, got %#v", got)
	}

	svc.RegisterVerbHandler(h)

	if !svc.HasVerbHandler("custom_verb") {
		t.Fatalf("HasVerbHandler should be true after registration")
	}
	got := svc.GetVerbHandler("custom_verb")
	if got != h {
		t.Fatalf("GetVerbHandler returned %#v, want %p", got, h)
	}
}

func TestServiceRegisterVerbHandlerReplacesPrevious(t *testing.T) {
	svc := NewService(WithName("test"))
	first := &stubVerbHandler{name: "dup"}
	second := &stubVerbHandler{name: "dup"}

	svc.RegisterVerbHandler(first)
	svc.RegisterVerbHandler(second)

	got := svc.GetVerbHandler("dup")
	if got != second {
		t.Fatalf("GetVerbHandler returned %p, expected the second handler %p", got, second)
	}
}

func TestServiceAIHandlerRegisteredByDefault(t *testing.T) {
	// NewService should pre-register the AIVerbHandler for the "ai" verb —
	// matches Python's SWMLVerbHandlerRegistry __init__ which populates
	// AIVerbHandler automatically.
	svc := NewService(WithName("test"))
	if !svc.HasVerbHandler("ai") {
		t.Fatalf(`expected default "ai" verb handler registered by NewService`)
	}
	h := svc.GetVerbHandler("ai")
	if _, ok := h.(*AIVerbHandler); !ok {
		t.Fatalf(`default "ai" handler should be *AIVerbHandler, got %T`, h)
	}
}

// --- AIVerbHandler.ValidateConfig ---

func TestAIVerbHandlerValidateConfigMissingPrompt(t *testing.T) {
	h := NewAIVerbHandler()
	valid, errs := h.ValidateConfig(map[string]any{})
	if valid {
		t.Fatalf("ValidateConfig should reject config without prompt")
	}
	if len(errs) == 0 {
		t.Fatalf("expected error messages, got none")
	}
}

func TestAIVerbHandlerValidateConfigPromptNotObject(t *testing.T) {
	h := NewAIVerbHandler()
	valid, errs := h.ValidateConfig(map[string]any{"prompt": "bare-string"})
	if valid {
		t.Fatalf("ValidateConfig should reject a bare-string prompt")
	}
	if len(errs) == 0 {
		t.Fatalf("expected error messages for bare-string prompt")
	}
}

func TestAIVerbHandlerValidateConfigAcceptsText(t *testing.T) {
	h := NewAIVerbHandler()
	valid, errs := h.ValidateConfig(map[string]any{
		"prompt": map[string]any{"text": "hello"},
	})
	if !valid {
		t.Fatalf("ValidateConfig should accept text prompt, got errors: %v", errs)
	}
}

func TestAIVerbHandlerValidateConfigAcceptsPOM(t *testing.T) {
	h := NewAIVerbHandler()
	valid, errs := h.ValidateConfig(map[string]any{
		"prompt": map[string]any{"pom": []any{map[string]any{"title": "x"}}},
	})
	if !valid {
		t.Fatalf("ValidateConfig should accept pom prompt, got errors: %v", errs)
	}
}

func TestAIVerbHandlerValidateConfigTextAndPOMMutuallyExclusive(t *testing.T) {
	h := NewAIVerbHandler()
	valid, _ := h.ValidateConfig(map[string]any{
		"prompt": map[string]any{"text": "hi", "pom": []any{}},
	})
	if valid {
		t.Fatalf("ValidateConfig should reject having both text and pom")
	}
}

// --- AIVerbHandler.BuildConfig ---

func TestAIVerbHandlerBuildConfigAlwaysEmitsParams(t *testing.T) {
	// Regression guard for issue #118 — Python always emits
	// config["params"] = {} even when no extra kwargs are supplied.
	h := NewAIVerbHandler()
	cfg, err := h.BuildConfig(map[string]any{"prompt_text": "hi"})
	if err != nil {
		t.Fatalf("BuildConfig returned error: %v", err)
	}
	params, present := cfg["params"]
	if !present {
		t.Fatalf(`expected "params" key in BuildConfig output, got keys: %v`, mapKeys(cfg))
	}
	m, ok := params.(map[string]any)
	if !ok {
		t.Fatalf(`"params" should be map[string]any, got %T`, params)
	}
	if len(m) != 0 {
		t.Fatalf(`expected empty "params" map, got %v`, m)
	}
}

func TestAIVerbHandlerBuildConfigCollectsExtraKwargs(t *testing.T) {
	h := NewAIVerbHandler()
	cfg, err := h.BuildConfig(map[string]any{
		"prompt_text": "hi",
		"tenant_id":   "abc",
		"custom_flag": true,
	})
	if err != nil {
		t.Fatalf("BuildConfig returned error: %v", err)
	}
	params, _ := cfg["params"].(map[string]any)
	if params["tenant_id"] != "abc" {
		t.Fatalf(`params["tenant_id"] = %v, want "abc"`, params["tenant_id"])
	}
	if params["custom_flag"] != true {
		t.Fatalf(`params["custom_flag"] = %v, want true`, params["custom_flag"])
	}
}

func TestAIVerbHandlerBuildConfigPromotesWellKnownKeys(t *testing.T) {
	// languages/hints/pronounce/global_data go to the top level, not into params.
	// Matches Python AIVerbHandler.build_config.
	h := NewAIVerbHandler()
	cfg, err := h.BuildConfig(map[string]any{
		"prompt_text": "hi",
		"languages":   []any{"en"},
		"hints":       []any{"foo"},
		"pronounce":   []any{},
		"global_data": map[string]any{"k": "v"},
	})
	if err != nil {
		t.Fatalf("BuildConfig returned error: %v", err)
	}
	for _, k := range []string{"languages", "hints", "pronounce", "global_data"} {
		if _, ok := cfg[k]; !ok {
			t.Errorf("expected top-level key %q in output", k)
		}
	}
	// None of those should end up under params.
	params, _ := cfg["params"].(map[string]any)
	for _, k := range []string{"languages", "hints", "pronounce", "global_data"} {
		if _, present := params[k]; present {
			t.Errorf("top-level key %q must not appear in params", k)
		}
	}
}

func TestAIVerbHandlerBuildConfigRejectsMissingPrompt(t *testing.T) {
	h := NewAIVerbHandler()
	_, err := h.BuildConfig(map[string]any{})
	if err == nil {
		t.Fatalf("expected error when neither prompt_text nor prompt_pom supplied")
	}
}

func TestAIVerbHandlerBuildConfigRejectsBothPrompts(t *testing.T) {
	h := NewAIVerbHandler()
	_, err := h.BuildConfig(map[string]any{
		"prompt_text": "hi",
		"prompt_pom":  []any{},
	})
	if err == nil {
		t.Fatalf("expected error when both prompt_text and prompt_pom supplied")
	}
}

func mapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
