// SchemaUtils loads the SWML JSON Schema, extracts verb definitions, and
// validates either a single verb config or a complete SWML document.
//
// This SDK bundles a full JSON Schema validator — Draft 2020-12 via
// santhosh-tekuri/jsonschema/v6, compiled in initFullValidator. So const/enum
// VALUES are enforced, not just required properties. FullValidationAvailable
// reports whether that compile succeeded; on failure validation degrades to a
// lightweight check (verb existence + required properties).
//
// SWML_SKIP_SCHEMA_VALIDATION=1 disables validation regardless of the
// constructor argument.

package swml

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/dlclark/regexp2"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// SchemaValidationError is the canonical error type returned when SWML
// schema validation fails.
type SchemaValidationError struct {
	VerbName string
	Errors   []string
}

// NewSchemaValidationError constructs a SchemaValidationError.
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

// ValidationResult is the “(valid, errors)“ pair returned by ValidateVerb
// and ValidateDocument. Errors is empty whenever Valid is true.
type ValidationResult struct {
	Valid  bool
	Errors []string
}

// SchemaUtils holds the loaded SWML schema and its extracted verb table.
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
	// validationEnabled records whether validation is switched on for this
	// instance (constructor argument AND env-var override).
	validationEnabled bool
	// verbs is the extracted verb table keyed by actual verb name
	// (e.g. "ai", "answer", "sip_refer").
	verbs map[string]*VerbInfo
	// schemaValidator is the optional full JSON Schema validator. It holds a
	// compiled *jsonschema.Schema (Draft 2020-12) when full validation is wired
	// and available. nil = lightweight (required-property) fallback.
	schemaValidator any
	// fullValidator is the concretely-typed compiled schema used by the full
	// validation path; kept separate from schemaValidator, which stays `any` so
	// the validator implementation is not part of the exported surface.
	fullValidator *jsonschema.Schema
}

// NewSchemaUtils constructs a SchemaUtils. BOTH inputs are OPTIONAL, matching
// the reference (`schema_path: str | None = None, schema_validation: bool =
// True`): calling it with no arguments uses the embedded schema.json bundled
// with the SDK and leaves validation ENABLED.
//
// WithSchemaUtilsPath overrides the embedded schema. WithSchemaUtilsValidation(false)
// disables validation; the env var SWML_SKIP_SCHEMA_VALIDATION=1/true/yes also
// disables it, regardless of the option.
func NewSchemaUtils(opts ...SchemaUtilsOption) *SchemaUtils {
	// validation defaults to TRUE, mirroring the reference's
	// `schema_validation: bool = True`.
	cfg := schemaUtilsOptions{schemaValidation: true}
	for _, o := range opts {
		o(&cfg)
	}
	envSkip := envBoolish(os.Getenv("SWML_SKIP_SCHEMA_VALIDATION"))
	su := &SchemaUtils{
		schemaPath:        cfg.schemaPath,
		validationEnabled: cfg.schemaValidation && !envSkip,
		verbs:             map[string]*VerbInfo{},
	}
	su.schema = su.LoadSchema()
	su.extractVerbs()
	if su.validationEnabled && len(su.schema) > 0 {
		su.initFullValidator()
	}
	return su
}

// schemaUtilsOptions accumulates SchemaUtils' optional construction inputs.
type schemaUtilsOptions struct {
	schemaPath       string
	schemaValidation bool
}

// SchemaUtilsOption configures a SchemaUtils at construction. The names carry a
// `SchemaUtils` infix because this package's plain `WithSchemaPath` /
// `WithSchemaValidation` are already taken by ServiceOption (service.go); the
// `With` PREFIX is kept because it is the convention every option family in this
// SDK follows and the surface enumerator keys construction params off it.
type SchemaUtilsOption func(*schemaUtilsOptions)

// WithSchemaUtilsPath loads the schema from an explicit path instead of the
// schema.json embedded in the SDK.
func WithSchemaUtilsPath(path string) SchemaUtilsOption {
	return func(o *schemaUtilsOptions) { o.schemaPath = path }
}

// WithSchemaUtilsValidation enables or disables schema validation. It defaults
// to enabled; SWML_SKIP_SCHEMA_VALIDATION disables it regardless.
func WithSchemaUtilsValidation(enabled bool) SchemaUtilsOption {
	return func(o *schemaUtilsOptions) { o.schemaValidation = enabled }
}

func envBoolish(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes":
		return true
	}
	return false
}

// SchemaPath returns the location the schema was loaded from, or "" when the
// embedded schema was used.
func (s *SchemaUtils) SchemaPath() string { return s.schemaPath }

// LoadSchema reads and parses the JSON Schema.
func (s *SchemaUtils) LoadSchema() map[string]any {
	if s.schemaPath != "" {
		return s.loadFromPath(s.schemaPath)
	}
	// Default: use the embedded schema.json bundled with the SDK.
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
	//nolint:gosec // G304: path is an operator-supplied schema path, not attacker
	// input — reading the configured schema file is the intended behavior.
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
			break // first key only
		}
	}
}

// initFullValidator compiles the embedded SWML JSON Schema into a Draft
// 2020-12 validator (santhosh-tekuri/jsonschema/v6). On any compile failure it
// leaves the validator nil so the lightweight required-property check remains
// the fallback.
func (s *SchemaUtils) initFullValidator() {
	if len(s.schema) == 0 {
		return
	}
	// The compiler wants a json-decoded document that uses json.Number for all
	// numbers (jsonschema.UnmarshalJSON sets UseNumber); re-encode our schema
	// map and decode it back through that helper so number semantics match.
	raw, err := json.Marshal(s.schema)
	if err != nil {
		return
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return
	}
	const schemaURL = "mem://swml/schema.json"
	c := jsonschema.NewCompiler()
	// Use an ECMAScript-compatible regexp engine (dlclark/regexp2) instead of
	// Go's RE2. The SWML schema uses negative-lookahead patterns (e.g. a step
	// name pattern "^(?!next$).*$") that RE2 cannot parse. Without this the
	// metaschema pass rejects the schema at compile time and the validator would
	// silently fall back to the lightweight (required-only) check.
	c.UseRegexpEngine(regexp2Engine)
	if err := c.AddResource(schemaURL, doc); err != nil {
		return
	}
	compiled, err := c.Compile(schemaURL)
	if err != nil {
		return
	}
	s.fullValidator = compiled
	s.schemaValidator = compiled
}

// FullValidationAvailable reports whether the full JSON Schema
// validator is wired up.
func (s *SchemaUtils) FullValidationAvailable() bool {
	return s.schemaValidator != nil
}

// GetAllVerbNames returns the sorted list of all known verb names.
func (s *SchemaUtils) GetAllVerbNames() []string {
	out := make([]string, 0, len(s.verbs))
	for k := range s.verbs {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// GetVerbProperties returns the inner “properties[verb_name]“ block
// for a verb, or an empty map when the verb is unknown.
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

// GetVerbRequiredProperties returns the “required“ list for a verb.
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
func (s *SchemaUtils) GetVerbParameters(verbName string) map[string]any {
	inner := s.GetVerbProperties(verbName)
	params, ok := inner["properties"].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return params
}

// ValidateVerb validates a verb config against the schema.
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

// verbTopLevelPropertyNames resolves the set of KNOWN top-level property names
// for a verb's config object, following a single $ref (e.g. AI -> AIObject) and
// UNIONING the branches of an anyOf/oneOf union. Returns (nil, false) only when
// there is genuinely no enumerable closed key-set, so no shallow check applies.
// Mirrors python _verb_top_level_property_names.
//
// The per-verb schemaGapKeys are folded in HERE rather than inside closedKeySet:
// they are a property of the VERB (which emitter writes which undeclared key),
// not of any one schema node, and closedKeySet recurses over $ref/union branches
// where no single verb name is in scope.
func (s *SchemaUtils) verbTopLevelPropertyNames(verbName string) (map[string]struct{}, bool) {
	v, ok := s.verbs[verbName]
	if !ok {
		return nil, false
	}
	props, ok := v.Definition["properties"].(map[string]any)
	if !ok {
		return nil, false
	}
	body, ok := props[verbName].(map[string]any)
	if !ok {
		return nil, false
	}
	known, ok := s.closedKeySet(body, 0)
	if !ok {
		return nil, false
	}
	for _, k := range schemaGapKeys[verbName] {
		known[k] = struct{}{}
	}
	return known, true
}

// maxSchemaResolveDepth bounds $ref / union following so a schema with a
// self-referential $ref cannot spin the resolver. Eight levels is well past
// anything the SWML schema needs (verb body -> $ref -> union branch -> $ref).
const maxSchemaResolveDepth = 8

// closedKeySet resolves ONE schema node to the set of top-level property names
// it closes over, returning (nil, false) when the node has no such enumerable
// closed key-set.
//
// Three node shapes are handled, and the union case is the one that matters:
//
//   - `$ref` — followed into $defs and resolved recursively (ai -> AIObject).
//   - `anyOf` / `oneOf` — resolved BRANCH BY BRANCH and UNIONED. Without this the
//     resolver used to bail on the first `type != "object"` test, because a union
//     node carries no `type` of its own. That bail silently DISENGAGED the
//     closed-key check: ValidateVerbTopLevelKeys got (nil, false) and reported
//     Valid for any key whatsoever. Five verbs in the shipped schema are
//     union-shaped — connect, play, send_sms, sleep, unset — so the check was
//     doing nothing for all of them. A union's known-key set is the union of its
//     object branches' keys: a config satisfying the union satisfies SOME branch,
//     so a key belonging to no branch belongs to no valid document. Non-object
//     branches (sleep's bare `integer`, SWMLVar) contribute no keys and are
//     skipped — they constrain the config to not be an object at all, which is a
//     different check than "which keys may an object config carry".
//   - a plain closed object — its own `properties`.
func (s *SchemaUtils) closedKeySet(body map[string]any, depth int) (map[string]struct{}, bool) {
	if body == nil || depth > maxSchemaResolveDepth {
		return nil, false
	}

	// Follow a $ref (ai -> AIObject) to the node that declares the properties.
	if ref, ok := body["$ref"].(string); ok {
		refName := ref
		if i := strings.LastIndex(ref, "/"); i >= 0 {
			refName = ref[i+1:]
		}
		defs, _ := s.schema["$defs"].(map[string]any)
		rd, ok := defs[refName].(map[string]any)
		if !ok {
			return nil, false
		}
		return s.closedKeySet(rd, depth+1)
	}

	// A union node: resolve every branch and union the ones that yield a set.
	branches, _ := body["anyOf"].([]any)
	if branches == nil {
		branches, _ = body["oneOf"].([]any)
	}
	if branches != nil {
		union := map[string]struct{}{}
		found := false
		for _, b := range branches {
			bm, ok := b.(map[string]any)
			if !ok {
				continue
			}
			keys, ok := s.closedKeySet(bm, depth+1)
			if !ok {
				continue
			}
			found = true
			for k := range keys {
				union[k] = struct{}{}
			}
		}
		if !found {
			// No branch is a closed object (e.g. unset: string | array-of-string).
			// There is no key-set to enforce; the deep validator owns this shape.
			return nil, false
		}
		return union, true
	}

	if t, _ := body["type"].(string); t != "object" {
		return nil, false
	}
	propMap, ok := body["properties"].(map[string]any)
	if !ok {
		return nil, false
	}
	// Only meaningful as a closed-key check when the schema itself closes the
	// object (additionalProperties:false or unevaluatedProperties disallowed).
	closes := false
	if ap, ok := body["additionalProperties"].(bool); ok && !ap {
		closes = true
	}
	if up, ok := body["unevaluatedProperties"].(bool); ok && !up {
		closes = true
	}
	if up, ok := body["unevaluatedProperties"].(map[string]any); ok {
		// The SWML schema closes objects with `unevaluatedProperties: {"not": {}}`
		// — an empty `not` schema that nothing satisfies, so any unevaluated
		// property is rejected.
		if notVal, has := up["not"]; has {
			if m, ok := notVal.(map[string]any); ok && len(m) == 0 {
				closes = true
			}
		}
	}
	if !closes {
		return nil, false
	}
	known := make(map[string]struct{}, len(propMap))
	for k := range propMap {
		known[k] = struct{}{}
	}
	return known, true
}

// schemaGapKeys lists config keys this SDK deliberately emits that the bundled
// schema.json does not (yet) declare. They are accepted by the shallow
// top-level-key check so that binding an emitter to the validating path does not
// DELETE a shipped feature.
//
// This is not an allow-list for invented surface: every entry must be a key an
// emitter here actually writes, cited below. The right long-term fix is in the
// schema, not here — each entry is a SCHEMA GAP awaiting an owner ruling, and
// the entry disappears the moment the schema declares the key.
//
//   - ai.multilingual — SetMultilingual sets aiConfig["multilingual"] at the ai
//     top level (pkg/agent/agent.go:3167-3169, ASR-driven "Mode B", emitted
//     right alongside `languages`). $defs/AIObject is closed over nine keys and
//     multilingual is not among them, so the schema and the emitter genuinely
//     disagree. Rejecting it would drop a shipped feature on the wire; the
//     schema is the side that is behind.
var schemaGapKeys = map[string][]string{
	"ai": {"multilingual"},
}

// ValidateVerbTopLevelKeys is the SHALLOW strict-render check: reject
// unknown/misspelled TOP-LEVEL keys in a verb config against the schema's known
// property set, WITHOUT running the full deep schema (which would false-reject
// legitimate deep emissions such as the ai verb's empty prompt.pom, SWAIG
// defaults, or functions[].web_hook_url/__token). Used for handler verbs (the
// ai verb) whose deep shapes the handler owns. A no-op when validation is
// disabled or when the verb genuinely has no enumerable closed key-set (an open
// object such as `set`, or a union with no object branch such as `unset`).
// Mirrors python validate_verb_top_level_keys.
func (s *SchemaUtils) ValidateVerbTopLevelKeys(verbName string, verbConfig map[string]any) ValidationResult {
	if !s.validationEnabled {
		return ValidationResult{Valid: true, Errors: []string{}}
	}
	if _, ok := s.verbs[verbName]; !ok {
		return ValidationResult{Valid: false, Errors: []string{fmt.Sprintf("Unknown verb: %s", verbName)}}
	}
	known, ok := s.verbTopLevelPropertyNames(verbName)
	if !ok {
		// No enumerable closed key-set — nothing shallow to enforce.
		return ValidationResult{Valid: true, Errors: []string{}}
	}
	var unknown []string
	for k := range verbConfig {
		if _, found := known[k]; !found {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		knownList := make([]string, 0, len(known))
		for k := range known {
			knownList = append(knownList, k)
		}
		sort.Strings(knownList)
		return ValidationResult{Valid: false, Errors: []string{
			fmt.Sprintf("Unknown/misspelled key(s) %v for verb '%s'. Known keys: %v",
				unknown, verbName, knownList),
		}}
	}
	return ValidationResult{Valid: true, Errors: []string{}}
}

func (s *SchemaUtils) validateVerbFull(verbName string, verbConfig map[string]any) ValidationResult {
	if s.fullValidator == nil {
		return s.validateVerbLightweight(verbName, verbConfig)
	}
	// Use lightweight for partial/test schemas that lack the full document
	// structure (no "sections" in properties).
	props, _ := s.schema["properties"].(map[string]any)
	if _, ok := props["sections"]; !ok {
		return s.validateVerbLightweight(verbName, verbConfig)
	}

	// Wrap the verb in a minimal SWML document so the
	// schema's closed-object (unevaluatedProperties) + type + required checks
	// fire against the real document context.
	minimalDoc := map[string]any{
		"version":  "1.0.0",
		"sections": map[string]any{"main": []any{map[string]any{verbName: verbConfig}}},
	}
	// Re-decode through jsonschema.UnmarshalJSON so numbers become json.Number,
	// matching how the schema was compiled (the validator compares kinds).
	raw, err := json.Marshal(minimalDoc)
	if err != nil {
		return ValidationResult{Valid: false, Errors: []string{fmt.Sprintf("Schema validation error for '%s': %v", verbName, err)}}
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return ValidationResult{Valid: false, Errors: []string{fmt.Sprintf("Schema validation error for '%s': %v", verbName, err)}}
	}
	if err := s.fullValidator.Validate(inst); err != nil {
		msg := err.Error()
		if len(msg) > 500 {
			msg = msg[:500] + "..."
		}
		return ValidationResult{Valid: false, Errors: []string{fmt.Sprintf("Schema validation error for '%s': %s", verbName, msg)}}
	}
	return ValidationResult{Valid: true, Errors: []string{}}
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

// ValidateDocument validates a complete SWML document against the schema.
//
// When the full validator is unavailable it returns
// “(false, ["Schema validator not initialized"])“ — an unavailable validator
// is reported as a failure, never silently as a pass.
func (s *SchemaUtils) ValidateDocument(document map[string]any) ValidationResult {
	if s.fullValidator == nil {
		return ValidationResult{Valid: false, Errors: []string{"Schema validator not initialized"}}
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return ValidationResult{Valid: false, Errors: []string{fmt.Sprintf("Document validation error: %v", err)}}
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return ValidationResult{Valid: false, Errors: []string{fmt.Sprintf("Document validation error: %v", err)}}
	}
	if err := s.fullValidator.Validate(inst); err != nil {
		msg := err.Error()
		if len(msg) > 500 {
			msg = msg[:500] + "..."
		}
		return ValidationResult{Valid: false, Errors: []string{fmt.Sprintf("Document validation error: %s", msg)}}
	}
	return ValidationResult{Valid: true, Errors: []string{}}
}

// GenerateMethodSignature renders a Python-style method signature for a verb —
// used by code-gen tooling. The emitted text IS Python source: the annotations
// it prints come from pythonTypeAnnotation.
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

// GenerateMethodBody renders the Python source of a method body for a verb,
// the counterpart to GenerateMethodSignature.
func (s *SchemaUtils) GenerateMethodBody(verbName string) string {
	params := s.GetVerbParameters(verbName)
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, 2+2*len(keys)+1+1+1+1+1+1+1)
	lines = append(lines,
		"        # Prepare the configuration",
		"        config = {}",
	)
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

// pythonTypeAnnotation maps a JSON-Schema parameter definition to the Python
// type-annotation string the code-gen output prints (str / int / float / bool /
// List[…] / Dict[str, Any], falling back to Any).
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

// regexp2Regexp adapts a *regexp2.Regexp to the jsonschema.Regexp interface
// (an ECMAScript-mode engine with lookahead/lookbehind support, which the SWML
// schema's patterns require and Go's RE2 cannot provide).
type regexp2Regexp regexp2.Regexp

// MatchString reports whether the pattern matches anywhere in str. The
// jsonschema.Regexp interface has no error channel, so a regexp2 evaluation
// error (e.g. the backtracking-timeout guard tripping) is treated as "no
// match" — the same conservative outcome as a pattern that simply does not
// match, which keeps a pathological input from failing the whole validation
// with an unrelated engine error.
func (r *regexp2Regexp) MatchString(str string) bool {
	matched, err := (*regexp2.Regexp)(r).MatchString(str)
	return err == nil && matched
}

// String returns the original pattern source the regexp was compiled from,
// which the validator embeds in "does not match pattern ..." error messages.
func (r *regexp2Regexp) String() string {
	return (*regexp2.Regexp)(r).String()
}

// regexp2Engine compiles a pattern with dlclark/regexp2 in ECMAScript mode.
// Wired into the schema compiler via Compiler.UseRegexpEngine so the SWML
// schema's negative-lookahead patterns compile (Go's stdlib RE2 rejects them).
func regexp2Engine(pattern string) (jsonschema.Regexp, error) {
	re, err := regexp2.Compile(pattern, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	return (*regexp2Regexp)(re), nil
}
