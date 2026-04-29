// Tests for the SWAIG_LIST_TOOLS introspection hook used by
// `swaig-test --example NAME`. The hook is a payload-builder + sentinel
// emitter on swml.Service.Serve(); when SWAIG_LIST_TOOLS is set the
// service prints the registry between sentinels and exits before binding.
//
// We test the helper (BuildSwaigListToolsPayload, maybeEmitListToolsSentinels)
// rather than Serve() itself so the test does not actually call os.Exit.
// Serve()'s exit branch is exercised end-to-end by the examples-binary
// invocation in the CLI integration test (cmd/swaig-test/main_test.go).

package swml

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestBuildSwaigListToolsPayload_EmptyRegistry(t *testing.T) {
	svc := NewService(WithName("svc"), WithBasicAuth("u", "p"))
	raw, err := svc.BuildSwaigListToolsPayload()
	if err != nil {
		t.Fatalf("BuildSwaigListToolsPayload err: %v", err)
	}
	var out struct {
		Tools []map[string]any `json:"tools"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Tools == nil {
		t.Fatal("tools key missing")
	}
	if len(out.Tools) != 0 {
		t.Errorf("expected empty tool list, got %d entries", len(out.Tools))
	}
}

func TestBuildSwaigListToolsPayload_WithTools(t *testing.T) {
	svc := NewService(WithName("svc"), WithBasicAuth("u", "p"))
	svc.DefineTool(&ToolDefinition{
		Name:        "alpha",
		Description: "Alpha tool",
		Parameters: map[string]any{
			"x": map[string]any{"type": "string", "description": "an x"},
		},
	})
	svc.DefineTool(&ToolDefinition{
		Name:        "bravo",
		Description: "Bravo tool",
	})

	raw, err := svc.BuildSwaigListToolsPayload()
	if err != nil {
		t.Fatalf("BuildSwaigListToolsPayload err: %v", err)
	}
	var out struct {
		Tools []map[string]any `json:"tools"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(out.Tools))
	}
	// Insertion order preserved.
	if out.Tools[0]["function"] != "alpha" {
		t.Errorf("tools[0].function = %v, want alpha", out.Tools[0]["function"])
	}
	if out.Tools[0]["description"] != "Alpha tool" {
		t.Errorf("tools[0].description = %v, want %q", out.Tools[0]["description"], "Alpha tool")
	}
	if _, ok := out.Tools[0]["parameters"].(map[string]any); !ok {
		t.Errorf("tools[0] missing parameters map")
	}
	if out.Tools[1]["function"] != "bravo" {
		t.Errorf("tools[1].function = %v, want bravo", out.Tools[1]["function"])
	}
	// bravo has no Parameters; should NOT include the key.
	if _, present := out.Tools[1]["parameters"]; present {
		t.Errorf("tools[1] should omit parameters when none registered")
	}
}

func TestMaybeEmitListToolsSentinels_NoEnvVar(t *testing.T) {
	t.Setenv("SWAIG_LIST_TOOLS", "")
	svc := NewService(WithName("svc"), WithBasicAuth("u", "p"))

	stdout, restore := captureStdout(t)
	defer restore()
	emitted := svc.maybeEmitListToolsSentinels()
	out := stdout()

	if emitted {
		t.Error("emitted=true with env var unset, want false")
	}
	if out != "" {
		t.Errorf("stdout = %q, want empty", out)
	}
}

func TestMaybeEmitListToolsSentinels_EnvVarSet(t *testing.T) {
	t.Setenv("SWAIG_LIST_TOOLS", "1")
	svc := NewService(WithName("svc"), WithBasicAuth("u", "p"))
	svc.DefineTool(&ToolDefinition{
		Name:        "lookup_competitor",
		Description: "Look up competitor pricing",
		Parameters: map[string]any{
			"competitor": map[string]any{"type": "string"},
		},
	})

	stdout, restore := captureStdout(t)
	defer restore()
	emitted := svc.maybeEmitListToolsSentinels()
	out := stdout()

	if !emitted {
		t.Fatal("emitted=false with env var set, want true")
	}
	if !strings.Contains(out, "__SWAIG_TOOLS_BEGIN__") {
		t.Errorf("missing BEGIN sentinel: %q", out)
	}
	if !strings.Contains(out, "__SWAIG_TOOLS_END__") {
		t.Errorf("missing END sentinel: %q", out)
	}
	if !strings.Contains(out, "lookup_competitor") {
		t.Errorf("payload missing tool name: %q", out)
	}
	// Payload between sentinels must be valid JSON.
	begin := strings.Index(out, "__SWAIG_TOOLS_BEGIN__")
	end := strings.Index(out, "__SWAIG_TOOLS_END__")
	if begin < 0 || end < 0 || end <= begin {
		t.Fatalf("malformed sentinel ordering: begin=%d end=%d", begin, end)
	}
	jsonSlice := strings.TrimSpace(out[begin+len("__SWAIG_TOOLS_BEGIN__") : end])
	var decoded map[string]any
	if err := json.Unmarshal([]byte(jsonSlice), &decoded); err != nil {
		t.Errorf("payload not valid JSON: %v\npayload: %q", err, jsonSlice)
	}
	tools, _ := decoded["tools"].([]any)
	if len(tools) != 1 {
		t.Errorf("expected 1 tool in payload, got %d", len(tools))
	}
}

// captureStdout redirects os.Stdout to a pipe and returns (reader, restore).
// Calling reader() drains current contents; restore() puts the original
// stdout back.
func captureStdout(t *testing.T) (read func() string, restore func()) {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	read = func() string {
		_ = w.Close()
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		os.Stdout = orig
		return buf.String()
	}
	restore = func() {
		os.Stdout = orig
	}
	return read, restore
}
