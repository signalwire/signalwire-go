// schema_utils_test.go — parity tests for SchemaUtils.
//
// Mirrors Python's tests/unit/utils/test_schema_utils.py and the
// TS / Perl reference implementations.  Every public method is
// exercised; assertions check shape (not just non-nullness) so the
// no-cheat-tests audit accepts them.

package swml

import (
	"os"
	"strings"
	"testing"
)

func TestSchemaUtils_DefaultLoad(t *testing.T) {
	su := NewSchemaUtils("", true)
	if su == nil {
		t.Fatal("NewSchemaUtils returned nil")
	}
	if len(su.GetAllVerbNames()) == 0 {
		t.Fatal("expected non-empty verb list from default schema")
	}
	// Verify a known verb is present
	names := su.GetAllVerbNames()
	if !sliceContainsStr(names, "ai") {
		t.Errorf("expected 'ai' verb in list, got: %v", names)
	}
	if !sliceContainsStr(names, "answer") {
		t.Errorf("expected 'answer' verb in list, got: %v", names)
	}
}

func TestSchemaUtils_DisabledValidation(t *testing.T) {
	su := NewSchemaUtils("", false)
	if su == nil {
		t.Fatal("NewSchemaUtils returned nil")
	}
	if su.FullValidationAvailable() {
		t.Error("expected FullValidationAvailable=false when schema_validation=false")
	}
	// validate_verb on a known verb should still return true (validation off)
	res := su.ValidateVerb("ai", map[string]any{})
	if !res.Valid {
		t.Errorf("expected validation skipped to return Valid=true, got: %+v", res)
	}
}

func TestSchemaUtils_EnvSkipDisablesValidation(t *testing.T) {
	t.Setenv("SWML_SKIP_SCHEMA_VALIDATION", "1")
	su := NewSchemaUtils("", true)
	if su.FullValidationAvailable() {
		t.Error("expected FullValidationAvailable=false when env var set")
	}
	res := su.ValidateVerb("ai", map[string]any{})
	if !res.Valid {
		t.Errorf("expected env-skip to return Valid=true, got: %+v", res)
	}
}

func TestSchemaUtils_GetVerbProperties(t *testing.T) {
	su := newMockedSchemaUtils()
	props := su.GetVerbProperties("ai")
	if len(props) == 0 {
		t.Fatal("expected non-empty properties for 'ai'")
	}
	// AI mock verb is type 'object' in the test fixture
	if typ, _ := props["type"].(string); typ != "object" {
		t.Errorf("expected ai.type=object, got: %v", props["type"])
	}
	// AI mock verb has a 'properties' block (parameters)
	if _, ok := props["properties"]; !ok {
		t.Error("expected ai.properties present")
	}
}

func TestSchemaUtils_GetVerbRequiredProperties(t *testing.T) {
	su := newMockedSchemaUtils()
	req := su.GetVerbRequiredProperties("ai")
	if !sliceContainsStr(req, "prompt") {
		t.Errorf("expected ai required to include 'prompt', got: %v", req)
	}
}

func TestSchemaUtils_GetVerbRequiredProperties_NoneSpecified(t *testing.T) {
	su := newMockedSchemaUtils()
	req := su.GetVerbRequiredProperties("answer")
	if len(req) != 0 {
		t.Errorf("expected empty required for 'answer', got: %v", req)
	}
}

func TestSchemaUtils_GetVerbParameters(t *testing.T) {
	su := newMockedSchemaUtils()
	params := su.GetVerbParameters("ai")
	if len(params) == 0 {
		t.Fatal("expected non-empty parameters for 'ai'")
	}
	if _, ok := params["prompt"]; !ok {
		t.Error("expected 'prompt' in ai parameters")
	}
	if _, ok := params["temperature"]; !ok {
		t.Error("expected 'temperature' in ai parameters")
	}
}

func TestSchemaUtils_ValidateVerb_Valid(t *testing.T) {
	su := newMockedSchemaUtils()
	cfg := map[string]any{"prompt": "You are helpful"}
	res := su.ValidateVerb("ai", cfg)
	if !res.Valid {
		t.Errorf("expected valid, got: %+v", res)
	}
	if len(res.Errors) != 0 {
		t.Errorf("expected no errors, got: %v", res.Errors)
	}
}

func TestSchemaUtils_ValidateVerb_MissingRequired(t *testing.T) {
	su := newMockedSchemaUtils()
	// Config missing required 'prompt'
	res := su.ValidateVerb("ai", map[string]any{"temperature": 0.7})
	if res.Valid {
		t.Error("expected invalid for missing required prompt")
	}
	if len(res.Errors) != 1 {
		t.Errorf("expected exactly 1 error, got: %v", res.Errors)
	}
	if !strings.Contains(res.Errors[0], "Missing required property 'prompt'") {
		t.Errorf("expected error to mention 'prompt', got: %v", res.Errors)
	}
}

func TestSchemaUtils_ValidateVerb_ExtraPropertiesAllowed(t *testing.T) {
	su := newMockedSchemaUtils()
	cfg := map[string]any{"prompt": "You are helpful", "extra_prop": "value"}
	res := su.ValidateVerb("ai", cfg)
	if !res.Valid {
		t.Errorf("expected valid (extra props allowed), got: %+v", res)
	}
}

func TestSchemaUtils_ValidateVerb_UnknownVerb(t *testing.T) {
	su := newMockedSchemaUtils()
	res := su.ValidateVerb("not_a_real_verb", map[string]any{})
	if res.Valid {
		t.Error("expected invalid for unknown verb")
	}
	if len(res.Errors) == 0 || !strings.Contains(res.Errors[0], "Unknown verb") {
		t.Errorf("expected 'Unknown verb' error, got: %v", res.Errors)
	}
}

func TestSchemaUtils_ValidateDocument_NoFullValidator(t *testing.T) {
	// In Go we don't ship a JSON Schema validator yet; validate_document
	// must return (false, ["Schema validator not initialized"]) — same
	// contract as Python when no validator is wired in.
	su := NewSchemaUtils("", true)
	res := su.ValidateDocument(map[string]any{
		"version":  "1.0.0",
		"sections": map[string]any{"main": []any{}},
	})
	if res.Valid {
		t.Error("expected ValidateDocument to fail when full validator not wired")
	}
	if len(res.Errors) == 0 || !strings.Contains(res.Errors[0], "validator not initialized") {
		t.Errorf("expected 'validator not initialized' error, got: %v", res.Errors)
	}
}

func TestSchemaUtils_GenerateMethodSignature(t *testing.T) {
	su := newMockedSchemaUtils()
	sig := su.GenerateMethodSignature("ai")
	if !strings.HasPrefix(sig, "def ai(") {
		t.Errorf("expected signature to start with 'def ai(', got: %q", sig)
	}
	if !strings.Contains(sig, "**kwargs") {
		t.Errorf("expected **kwargs in signature, got: %q", sig)
	}
	if !strings.Contains(sig, "prompt: str") {
		t.Errorf("expected required 'prompt: str' in signature, got: %q", sig)
	}
	if !strings.Contains(sig, "temperature: Optional[float]") {
		t.Errorf("expected optional 'temperature: Optional[float]' in signature, got: %q", sig)
	}
}

func TestSchemaUtils_GenerateMethodBody(t *testing.T) {
	su := newMockedSchemaUtils()
	body := su.GenerateMethodBody("ai")
	if !strings.Contains(body, "self.add_verb('ai'") {
		t.Errorf("expected body to call self.add_verb('ai'), got: %q", body)
	}
	if !strings.Contains(body, "config = {}") {
		t.Errorf("expected body to init config={}, got: %q", body)
	}
	if !strings.Contains(body, "config['prompt'] = prompt") {
		t.Errorf("expected body to set prompt, got: %q", body)
	}
}

func TestSchemaUtils_LoadSchemaFromExplicitPath(t *testing.T) {
	// Write the embedded schema to a temp path and load it explicitly.
	data, err := schemaFS.ReadFile("schema.json")
	if err != nil {
		t.Fatal(err)
	}
	tmp, err := os.CreateTemp(t.TempDir(), "schema-*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.Write(data); err != nil {
		t.Fatal(err)
	}
	tmp.Close()
	su := NewSchemaUtils(tmp.Name(), true)
	if len(su.GetAllVerbNames()) == 0 {
		t.Error("expected verbs from file-loaded schema")
	}
}

func TestSchemaValidationError(t *testing.T) {
	err := NewSchemaValidationError("ai", []string{"missing prompt", "bad type"})
	msg := err.Error()
	if !strings.Contains(msg, "ai") {
		t.Errorf("expected 'ai' in error message, got: %q", msg)
	}
	if !strings.Contains(msg, "missing prompt") {
		t.Errorf("expected error detail in message, got: %q", msg)
	}
	if err.VerbName != "ai" {
		t.Errorf("expected VerbName='ai', got: %q", err.VerbName)
	}
	if len(err.Errors) != 2 {
		t.Errorf("expected 2 errors, got: %d", len(err.Errors))
	}
}

func TestService_SchemaUtilsAccessor(t *testing.T) {
	s := NewService()
	su := s.SchemaUtils()
	if su == nil {
		t.Fatal("Service.SchemaUtils() returned nil")
	}
	if len(su.GetAllVerbNames()) == 0 {
		t.Error("expected verbs from Service-bound SchemaUtils")
	}
}

func sliceContainsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// newMockedSchemaUtils returns a SchemaUtils prepopulated with the
// same minimal verb fixture used by Python's TestVerbValidation suite.
// Mirrors test_schema_utils.py:setup_method (TestVerbValidation +
// TestVerbExtraction).
func newMockedSchemaUtils() *SchemaUtils {
	return &SchemaUtils{
		validationEnabled: true,
		verbs: map[string]*VerbInfo{
			"ai": {
				Name:       "ai",
				SchemaName: "AIMethod",
				Definition: map[string]any{
					"properties": map[string]any{
						"ai": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"prompt":      map[string]any{"type": "string", "description": "The AI prompt text"},
								"temperature": map[string]any{"type": "number", "description": "Temperature for AI generation"},
							},
							"required": []any{"prompt"},
						},
					},
				},
			},
			"answer": {
				Name:       "answer",
				SchemaName: "AnswerMethod",
				Definition: map[string]any{
					"properties": map[string]any{
						"answer": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"max_duration": map[string]any{"type": "integer"},
							},
						},
					},
				},
			},
		},
	}
}
