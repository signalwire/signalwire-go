// SchemaUtils — Go port of signalwire.utils.schema_utils.SchemaUtils.
//
// Loads the SWML JSON Schema, extracts verb definitions, and validates
// either a single verb config or a complete SWML document.  Validation
// is lightweight (verb existence + required-property check) when
// run without a JSON Schema validator dependency; the surface mirrors
// the Python reference so cross-language audits see identical methods.
//
// The Go SDK does not currently bundle a JSON Schema validator, so
// full_validation_available is gated on whether one has been wired
// in (see schemaValidator field).  Lightweight mode is the default
// and matches Python's behaviour when jsonschema_rs is unavailable.
//
// SWML_SKIP_SCHEMA_VALIDATION=1 disables validation regardless of the
// constructor argument, mirroring Python's env-var override.

package swml

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// SchemaValidationError is the canonical error type raised when SWML
// schema validation fails.  Mirrors Python's SchemaValidationError.
type SchemaValidationError struct {
	VerbName string
	Errors   []string
}

// NewSchemaValidationError constructs a SchemaValidationError.
// Mirrors Python's SchemaValidationError.__init__(verb_name, errors).
func NewSchemaValidationError(verbName string, errors []string) *SchemaValidationError {
	return &SchemaValidationError{VerbName: verbName, Errors: errors}
}

// Error renders the validation failure as a single string.
func (e *SchemaValidationError) Error() string {
	return fmt.Sprintf(
		"Schema validation failed for '%s': %s",
		e.VerbName, strings.Join(e.Errors, "; "),
	)
}

// ValidationResult mirrors Python's ``Tuple[bool, List[str]]`` return
// shape used by ValidateVerb / ValidateDocument.
//
// The cross-language type alias table maps this struct to the canonical
// ``tuple<bool,list<string>>`` so audits accept it as Python-shaped.
type ValidationResult struct {
	Valid  bool
	Errors []string
}

// SchemaUtils is the Go port of signalwire.utils.schema_utils.SchemaUtils.
//
// Construction rules:
//   - schemaPath empty + SWML_SKIP_SCHEMA_VALIDATION unset → load embedded.
//   - schemaPath set → load from file.
//   - schemaValidation=false OR SWML_SKIP_SCHEMA_VALIDATION=1 → no full validator
//     (lightweight required-property check still runs).
type SchemaUtils struct {
	// schema is the parsed JSON Schema document.
	schema map[string]any
	// schemaPath is the resolved location the schema was loaded from
	// (or "" when the embedded schema was used).
	schemaPath string
	// validationEnabled mirrors Python's _validation_enabled.
	validationEnabled bool
	// verbs is the extracted verb table keyed by actual verb name
	// (e.g. "ai", "answer", "sip_refer").
	verbs map[string]*VerbInfo
	// schemaValidator is the optional full JSON Schema validator.
	// Currently nil in the Go port — extend by wiring a validator
	// (e.g. github.com/santhosh-tekuri/jsonschema/v5) here.
	schemaValidator any
}

// NewSchemaUtils constructs a SchemaUtils.
// Mirrors Python's ``SchemaUtils(schema_path, schema_validation=True)``.
//
// Pass schemaPath="" to use the embedded schema.json bundled with the SDK.
// schemaValidation=false disables validation; the env var
// SWML_SKIP_SCHEMA_VALIDATION=1/true/yes also disables it.
func NewSchemaUtils(schemaPath string, schemaValidation bool) *SchemaUtils {
	envSkip := envBoolish(os.Getenv("SWML_SKIP_SCHEMA_VALIDATION"))
	su := &SchemaUtils{
		schemaPath:        schemaPath,
		validationEnabled: schemaValidation && !envSkip,
		verbs:             map[string]*VerbInfo{},
	}
	su.schema = su.LoadSchema()
	su.extractVerbs()
	if su.validationEnabled && len(su.schema) > 0 {
		su.initFullValidator()
	}
	return su
}

func envBoolish(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes":
		return true
	}
	return false
}

// LoadSchema reads and parses the JSON Schema.
// Mirrors Python's ``load_schema()``.
func (s *SchemaUtils) LoadSchema() map[string]any {
	if s.schemaPath != "" {
		return s.loadFromPath(s.schemaPath)
	}
	// Default: use embedded schema (matches Python's _get_default_schema_path).
	data, err := schemaFS.ReadFile("schema.json")
	if err != nil {
		return map[string]any{}
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return map[string]any{}
	}
	return raw
}

func (s *SchemaUtils) loadFromPath(path string) map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{}
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return map[string]any{}
	}
	return raw
}

func (s *SchemaUtils) extractVerbs() {
	defs, ok := s.schema["$defs"].(map[string]any)
	if !ok {
		return
	}
	swmlMethod, ok := defs["SWMLMethod"].(map[string]any)
	if !ok {
		return
	}
	anyOf, ok := swmlMethod["anyOf"].([]any)
	if !ok {
		return
	}
	for _, ref := range anyOf {
		refMap, ok := ref.(map[string]any)
		if !ok {
			continue
		}
		refStr, ok := refMap["$ref"].(string)
		if !ok {
			continue
		}
		// "#/$defs/SIPRefer" -> "SIPRefer"
		const prefix = "#/$defs/"
		if !strings.HasPrefix(refStr, prefix) {
			continue
		}
		schemaName := refStr[len(prefix):]
		defn, ok := defs[schemaName].(map[string]any)
		if !ok {
			continue
		}
		props, ok := defn["properties"].(map[string]any)
		if !ok {
			continue
		}
		for actualVerb := range props {
			s.verbs[actualVerb] = &VerbInfo{
				Name:       actualVerb,
				SchemaName: schemaName,
				Definition: defn,
			}
			break // first key only — matches Python
		}
	}
}

// initFullValidator wires up the full JSON Schema validator when
// available.  In the Go port this currently does nothing — extend
// here when a validator dependency is added.
func (s *SchemaUtils) initFullValidator() {
	// Reserved for future full-validator wiring.  Lightweight validation
	// (required-property check) runs unconditionally below.
	s.schemaValidator = nil
}

// FullValidationAvailable reports whether the full JSON Schema
// validator is wired up.  Mirrors Python's full_validation_available.
func (s *SchemaUtils) FullValidationAvailable() bool {
	return s.schemaValidator != nil
}

// GetAllVerbNames returns the sorted list of all known verb names.
// Mirrors Python's ``get_all_verb_names()``.
func (s *SchemaUtils) GetAllVerbNames() []string {
	out := make([]string, 0, len(s.verbs))
	for k := range s.verbs {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// GetVerbProperties returns the inner ``properties[verb_name]`` block
// for a verb, or an empty map when the verb is unknown.
// Mirrors Python's ``get_verb_properties(verb_name)``.
func (s *SchemaUtils) GetVerbProperties(verbName string) map[string]any {
	v, ok := s.verbs[verbName]
	if !ok {
		return map[string]any{}
	}
	props, ok := v.Definition["properties"].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	inner, ok := props[verbName].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return inner
}

// GetVerbRequiredProperties returns the ``required`` list for a verb.
// Mirrors Python's ``get_verb_required_properties(verb_name)``.
func (s *SchemaUtils) GetVerbRequiredProperties(verbName string) []string {
	inner := s.GetVerbProperties(verbName)
	req, ok := inner["required"].([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(req))
	for _, r := range req {
		if s, ok := r.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// GetVerbParameters returns the parameter-definition block used for
// codegen — verb_props["properties"].
// Mirrors Python's ``get_verb_parameters(verb_name)``.
func (s *SchemaUtils) GetVerbParameters(verbName string) map[string]any {
	inner := s.GetVerbProperties(verbName)
	params, ok := inner["properties"].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return params
}

// ValidateVerb validates a verb config against the schema.
// Mirrors Python's ``validate_verb(verb_name, verb_config)``.
//
// When validation is disabled returns Valid=true.  When the verb name
// is unknown returns Valid=false with a single "Unknown verb" error.
// Otherwise runs the full validator if available, falling back to
// the lightweight required-property check.
func (s *SchemaUtils) ValidateVerb(verbName string, verbConfig map[string]any) ValidationResult {
	if !s.validationEnabled {
		return ValidationResult{Valid: true, Errors: []string{}}
	}
	if _, ok := s.verbs[verbName]; !ok {
		return ValidationResult{Valid: false, Errors: []string{fmt.Sprintf("Unknown verb: %s", verbName)}}
	}
	if s.schemaValidator != nil {
		return s.validateVerbFull(verbName, verbConfig)
	}
	return s.validateVerbLightweight(verbName, verbConfig)
}

func (s *SchemaUtils) validateVerbFull(verbName string, verbConfig map[string]any) ValidationResult {
	// Reserved for full-validator integration.  Lightweight fallback
	// keeps behaviour identical to Python's lightweight path until
	// a validator is wired in.
	return s.validateVerbLightweight(verbName, verbConfig)
}

func (s *SchemaUtils) validateVerbLightweight(verbName string, verbConfig map[string]any) ValidationResult {
	required := s.GetVerbRequiredProperties(verbName)
	errors := []string{}
	for _, prop := range required {
		if _, ok := verbConfig[prop]; !ok {
			errors = append(errors, fmt.Sprintf("Missing required property '%s' for verb '%s'", prop, verbName))
		}
	}
	return ValidationResult{Valid: len(errors) == 0, Errors: errors}
}

// ValidateDocument validates a complete SWML document against the
// schema.  Mirrors Python's ``validate_document(document)``.
//
// When the full validator is unavailable Python returns
// ``(False, ["Schema validator not initialized"])``; the Go port
// matches that contract bit-for-bit.
func (s *SchemaUtils) ValidateDocument(document map[string]any) ValidationResult {
	if s.schemaValidator == nil {
		return ValidationResult{Valid: false, Errors: []string{"Schema validator not initialized"}}
	}
	// Reserved for full-validator integration.
	return ValidationResult{Valid: true, Errors: []string{}}
}

// GenerateMethodSignature renders a Python-style method signature
// for a verb — used by code-gen tooling.  Mirrors Python's
// ``generate_method_signature(verb_name)``.
func (s *SchemaUtils) GenerateMethodSignature(verbName string) string {
	params := s.GetVerbParameters(verbName)
	required := map[string]bool{}
	for _, r := range s.GetVerbRequiredProperties(verbName) {
		required[r] = true
	}
	parts := []string{"self"}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		t := pythonTypeAnnotation(params[name])
		if required[name] {
			parts = append(parts, fmt.Sprintf("%s: %s", name, t))
		} else {
			parts = append(parts, fmt.Sprintf("%s: Optional[%s] = None", name, t))
		}
	}
	parts = append(parts, "**kwargs")
	docstring := fmt.Sprintf(
		"\"\"\"\n        Add the %s verb to the current document\n        \n",
		verbName,
	)
	for _, name := range keys {
		desc := ""
		if d, ok := params[name].(map[string]any); ok {
			if dv, ok := d["description"].(string); ok {
				desc = strings.ReplaceAll(dv, "\n", " ")
				desc = strings.TrimSpace(desc)
			}
		}
		docstring += fmt.Sprintf("        Args:\n            %s: %s\n", name, desc)
	}
	docstring += "        \n        Returns:\n            True if the verb was added successfully, False otherwise\n        \"\"\"\n"
	return fmt.Sprintf("def %s(%s) -> bool:\n%s", verbName, strings.Join(parts, ", "), docstring)
}

// GenerateMethodBody renders a Python-style method body for a verb.
// Mirrors Python's ``generate_method_body(verb_name)``.
func (s *SchemaUtils) GenerateMethodBody(verbName string) string {
	params := s.GetVerbParameters(verbName)
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := []string{
		"        # Prepare the configuration",
		"        config = {}",
	}
	for _, name := range keys {
		lines = append(lines, fmt.Sprintf("        if %s is not None:", name))
		lines = append(lines, fmt.Sprintf("            config['%s'] = %s", name, name))
	}
	lines = append(lines, "        # Add any additional parameters from kwargs")
	lines = append(lines, "        for key, value in kwargs.items():")
	lines = append(lines, "            if value is not None:")
	lines = append(lines, "                config[key] = value")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("        # Add the %s verb", verbName))
	lines = append(lines, fmt.Sprintf("        return self.add_verb('%s', config)", verbName))
	return strings.Join(lines, "\n")
}

// pythonTypeAnnotation maps a JSON-Schema parameter definition to a
// Python type annotation string, mirroring Python's ``_get_type_annotation``.
func pythonTypeAnnotation(def any) string {
	d, ok := def.(map[string]any)
	if !ok {
		return "Any"
	}
	switch t, _ := d["type"].(string); t {
	case "string":
		return "str"
	case "integer":
		return "int"
	case "number":
		return "float"
	case "boolean":
		return "bool"
	case "array":
		item := "Any"
		if items, ok := d["items"].(map[string]any); ok {
			item = pythonTypeAnnotation(items)
		}
		return "List[" + item + "]"
	case "object":
		return "Dict[str, Any]"
	default:
		if _, has := d["anyOf"]; has {
			return "Any"
		}
		if _, has := d["oneOf"]; has {
			return "Any"
		}
		if _, has := d["$ref"]; has {
			return "Any"
		}
		return "Any"
	}
}
