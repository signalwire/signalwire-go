package agent

import (
	"testing"
)

// Parity: signalwire-python/tests/unit/core/test_agent_base.py
//
//	TestAgentBaseTokenMethods.test_validate_tool_token
//	TestAgentBaseTokenMethods.test_create_tool_token
//
// These tests mirror the Python contract for StateMixin.validate_tool_token
// (rejects unknown functions, swallows panics, returns false on failure)
// and StateMixin._create_tool_token (returns "" on failure, otherwise the
// SessionManager-issued token).

func TestCreateToolToken_RoundTrip(t *testing.T) {
	a := NewAgentBase(WithName("test_agent"))
	a.DefineTool(ToolDefinition{Name: "test_tool", Description: "t"})

	token := a.CreateToolToken("test_tool", "call_123")
	if token == "" {
		t.Fatalf("CreateToolToken returned empty string; expected a non-empty token")
	}

	if !a.ValidateToolToken("test_tool", token, "call_123") {
		t.Errorf("ValidateToolToken rejected the token we just created")
	}
}

func TestValidateToolToken_RejectsUnknownFunction(t *testing.T) {
	a := NewAgentBase(WithName("test_agent"))
	// No tool registered.

	if a.ValidateToolToken("not_registered", "any_token", "call_123") {
		t.Errorf("ValidateToolToken returned true for unregistered function; expected false")
	}
}

func TestValidateToolToken_RejectsBadToken(t *testing.T) {
	a := NewAgentBase(WithName("test_agent"))
	a.DefineTool(ToolDefinition{Name: "test_tool", Description: "t"})

	if a.ValidateToolToken("test_tool", "garbage_token_value", "call_123") {
		t.Errorf("ValidateToolToken accepted a garbage token; expected false")
	}
}

func TestValidateToolToken_RejectsWrongCallID(t *testing.T) {
	a := NewAgentBase(WithName("test_agent"))
	a.DefineTool(ToolDefinition{Name: "test_tool", Description: "t"})

	token := a.CreateToolToken("test_tool", "call_A")
	if token == "" {
		t.Fatalf("CreateToolToken returned empty string")
	}

	if a.ValidateToolToken("test_tool", token, "call_B") {
		t.Errorf("ValidateToolToken accepted token bound to a different call_id; expected false")
	}
}
