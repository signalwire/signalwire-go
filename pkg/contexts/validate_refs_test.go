// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package contexts

import (
	"strings"
	"testing"
)

// Negative-case tests for the cross-reference validation loops added in
// this PR: initial_step, step.valid_steps, context.valid_contexts,
// step.valid_contexts. Mirrors Python SWML_ContextBuilder validation
// (tests/unit/core/test_contexts.py).

// --- initial_step ---

func TestValidateInitialStepUnknown(t *testing.T) {
	cb := NewContextBuilder()
	ctx := cb.AddContext("default")
	ctx.AddStep("greet").SetText("hi")
	ctx.SetInitialStep("nonexistent")

	err := cb.Validate()
	if err == nil {
		t.Fatal("expected error when initial_step references an unknown step")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention the unknown step name, got: %v", err)
	}
}

func TestValidateInitialStepValid(t *testing.T) {
	cb := NewContextBuilder()
	ctx := cb.AddContext("default")
	ctx.AddStep("greet").SetText("hi")
	ctx.AddStep("collect").SetText("give me your name")
	ctx.SetInitialStep("collect")

	if err := cb.Validate(); err != nil {
		t.Fatalf("expected valid initial_step to pass, got: %v", err)
	}
}

// --- step.valid_steps ---

func TestValidateStepValidStepsReferencesUnknownStep(t *testing.T) {
	cb := NewContextBuilder()
	ctx := cb.AddContext("default")
	ctx.AddStep("greet").SetText("hi").SetValidSteps([]string{"no_such_step"})

	err := cb.Validate()
	if err == nil {
		t.Fatal("expected error when valid_steps references unknown step")
	}
	if !strings.Contains(err.Error(), "no_such_step") {
		t.Errorf("error should mention the unknown step, got: %v", err)
	}
}

func TestValidateStepValidStepsAllowsNextKeyword(t *testing.T) {
	// "next" is the special keyword meaning advance to the next sequential
	// step — it must NOT be rejected as an unknown step name.
	cb := NewContextBuilder()
	ctx := cb.AddContext("default")
	ctx.AddStep("greet").SetText("hi").SetValidSteps([]string{"next"})

	if err := cb.Validate(); err != nil {
		t.Fatalf(`"next" keyword must be accepted in valid_steps, got: %v`, err)
	}
}

func TestValidateStepValidStepsAllowsKnownSteps(t *testing.T) {
	cb := NewContextBuilder()
	ctx := cb.AddContext("default")
	ctx.AddStep("greet").SetText("hi").SetValidSteps([]string{"collect"})
	ctx.AddStep("collect").SetText("give me your name")

	if err := cb.Validate(); err != nil {
		t.Fatalf("valid cross-step reference should pass, got: %v", err)
	}
}

// --- context.valid_contexts (context-level) ---

func TestValidateContextValidContextsReferencesUnknownContext(t *testing.T) {
	cb := NewContextBuilder()
	a := cb.AddContext("sales")
	a.AddStep("s1").SetText("hi")
	a.SetValidContexts([]string{"no_such_context"})

	// Also add a second context so the single-context-must-be-default rule
	// doesn't mask this failure.
	b := cb.AddContext("support")
	b.AddStep("s1").SetText("hi")

	err := cb.Validate()
	if err == nil {
		t.Fatal("expected error when context valid_contexts references unknown context")
	}
	if !strings.Contains(err.Error(), "no_such_context") {
		t.Errorf("error should mention the unknown context name, got: %v", err)
	}
}

func TestValidateContextValidContextsAcceptsKnownContexts(t *testing.T) {
	cb := NewContextBuilder()
	a := cb.AddContext("sales")
	a.AddStep("s1").SetText("hi")
	a.SetValidContexts([]string{"support"})
	b := cb.AddContext("support")
	b.AddStep("s1").SetText("hi")

	if err := cb.Validate(); err != nil {
		t.Fatalf("known context in valid_contexts should pass, got: %v", err)
	}
}

// --- step.valid_contexts (step-level) ---

func TestValidateStepValidContextsReferencesUnknownContext(t *testing.T) {
	cb := NewContextBuilder()
	a := cb.AddContext("sales")
	a.AddStep("s1").SetText("hi").SetValidContexts([]string{"no_such_context"})
	b := cb.AddContext("support")
	b.AddStep("s1").SetText("hi")

	err := cb.Validate()
	if err == nil {
		t.Fatal("expected error when step valid_contexts references unknown context")
	}
	if !strings.Contains(err.Error(), "no_such_context") {
		t.Errorf("error should mention the unknown context name, got: %v", err)
	}
}

func TestValidateStepValidContextsAcceptsKnownContexts(t *testing.T) {
	cb := NewContextBuilder()
	a := cb.AddContext("sales")
	a.AddStep("s1").SetText("hi").SetValidContexts([]string{"support"})
	b := cb.AddContext("support")
	b.AddStep("s1").SetText("hi")

	if err := cb.Validate(); err != nil {
		t.Fatalf("known context in step valid_contexts should pass, got: %v", err)
	}
}
