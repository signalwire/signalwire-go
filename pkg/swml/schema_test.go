package swml

import (
	"testing"
)

func TestGetSchema(t *testing.T) {
	schema, err := GetSchema()
	if err != nil {
		t.Fatalf("GetSchema failed: %v", err)
	}
	if schema == nil {
		t.Fatal("schema is nil")
	}
}

func TestSchemaVerbCount(t *testing.T) {
	schema, err := GetSchema()
	if err != nil {
		t.Fatalf("GetSchema failed: %v", err)
	}
	// Assert the schema LOADED and is self-consistent, not a frozen headcount. A
	// hardcoded literal has to be edited by every PR that adds a verb upstream
	// (ai_sidecar took it 38 -> 39) and never caught a real defect -- the python
	// reference has no equivalent assertion, it only logs the count.
	count := schema.VerbCount()
	if count != len(schema.GetAllVerbNames()) {
		t.Errorf("VerbCount = %d disagrees with GetAllVerbNames = %d", count, len(schema.GetAllVerbNames()))
	}
	if count < 38 {
		t.Errorf("schema looks truncated: VerbCount = %d, want >= 38", count)
	}
}

func TestSchemaKnownVerbs(t *testing.T) {
	schema, err := GetSchema()
	if err != nil {
		t.Fatalf("GetSchema failed: %v", err)
	}

	expectedVerbs := []string{
		"answer", "ai", "amazon_bedrock", "cond", "connect",
		"denoise", "enter_queue", "execute", "goto", "label",
		"live_transcribe", "live_translate", "hangup", "join_room",
		"join_conference", "play", "prompt", "receive_fax", "record",
		"record_call", "request", "return", "sip_refer", "send_digits",
		"send_fax", "send_sms", "set", "sleep", "stop_denoise",
		"stop_record_call", "stop_tap", "switch", "tap", "transfer",
		"unset", "pay", "detect_machine", "user_event",
	}

	for _, verb := range expectedVerbs {
		if !schema.IsValidVerb(verb) {
			t.Errorf("schema should recognize verb %q", verb)
		}
	}
}

func TestSchemaGetVerb(t *testing.T) {
	schema, err := GetSchema()
	if err != nil {
		t.Fatalf("GetSchema failed: %v", err)
	}

	// Test known verb
	info, ok := schema.GetVerb("sip_refer")
	if !ok {
		t.Fatal("sip_refer should exist")
	}
	if info.Name != "sip_refer" {
		t.Errorf("Name = %q, want %q", info.Name, "sip_refer")
	}
	if info.SchemaName != "SIPRefer" {
		t.Errorf("SchemaName = %q, want %q", info.SchemaName, "SIPRefer")
	}

	// Test AI verb
	info, ok = schema.GetVerb("ai")
	if !ok {
		t.Fatal("ai should exist")
	}
	if info.SchemaName != "AI" {
		t.Errorf("SchemaName = %q, want %q", info.SchemaName, "AI")
	}

	// Test unknown verb
	_, ok = schema.GetVerb("nonexistent")
	if ok {
		t.Error("nonexistent verb should not be found")
	}
}

func TestSchemaGetAllVerbNames(t *testing.T) {
	schema, err := GetSchema()
	if err != nil {
		t.Fatalf("GetSchema failed: %v", err)
	}

	names := schema.GetAllVerbNames()
	if len(names) < 38 {
		t.Errorf("schema looks truncated: GetAllVerbNames returned %d, want >= 38", len(names))
	}

	// Check a few are present
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	for _, expected := range []string{"ai", "play", "answer", "hangup", "sleep"} {
		if !nameSet[expected] {
			t.Errorf("missing expected verb %q", expected)
		}
	}
}
