// Command generate-payloads emits the typed READ-side SWAIG + SWML-verb
// payloads (SWAIG_PIPELINE §4, SESSION_CHANGESET_FOR_PORTS §D) from the
// authoritative vendored specs in porting-sdk:
//
//	swaig-specs/swaig-request.yaml   -> pkg/swaig/swaig_request_generated.go
//	swaig-specs/post-prompt.yaml     -> pkg/swaig/post_prompt_generated.go
//	swaig-specs/swaig-response.yaml  -> pkg/swaig/swaig_actions_generated.go
//	schema.json ($defs)              -> pkg/swml/swml_verbs_generated.go
//
// These are the payloads the agent RECEIVES (function-webhook request body,
// post-prompt / onSummary callback summary) plus the SWML verb CONFIG shapes
// and the SWAIG response-action CONFIG shapes the FunctionResult builder
// accepts. They mirror the Python reference emitters (generate_swaig_request /
// generate_post_prompt / generate_swaig_actions / generate_swml_verbs in
// porting-sdk/scripts/generate_python_rest_types.py) and the TypeScript
// generateSwaigContracts / generateSwaigActions / generateSwmlVerbs — same
// STRUCTURE, expressed in idiomatic Go.
//
// Open-shaped READ payloads: every field is optional (a Go pointer / slice /
// map / any zero value = absent), and every named struct is a plain struct the
// runtime unmarshals JSON into with extra server keys tolerated (encoding/json
// ignores unknown keys by default). The Go RUNTIME type of each field is the
// most faithful compilable shape (a `*Ref` pointer, a `[]Ref` slice, a
// `map[string]any`, `any` for a scalar|ref union). The AUDIT-canonical type
// (what the cross-port drift gate compares) is written into a `gen:"..."` struct
// tag so cmd/enumerate-signatures records it verbatim — Go has no union type, so
// a `union<int,class:SWMLVar>` field is `any` at runtime but carries its exact
// canonical shape in the tag. The tag is the single source of truth for the
// audit; it decouples the (lossy) Go static type from the (faithful) wire shape.
//
// GEN-FRESH-gated: `--check` reproduces the committed *_generated.go and exits
// non-zero if any differs. Resolves porting-sdk via $PORTING_SDK or sibling.
//
// Usage:
//
//	go run ./cmd/generate-payloads          # (re)write the *_generated.go files
//	go run ./cmd/generate-payloads --check  # GEN-FRESH: fail if any is stale
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Spec model — a minimal JSON-Schema / OpenAPI schema node.
// ---------------------------------------------------------------------------

// schema is an ordered view of a schema node. YAML/JSON object key order is
// preserved for `properties` (so the emitted field order is stable and matches
// the spec) via the ordered decode below.
type schema struct {
	Ref                  string
	Type                 any // string or []any
	Const                any
	Enum                 []any
	OneOf                []*schema
	AnyOf                []*schema
	AllOf                []*schema
	Items                *schema
	Properties           []propEntry // ordered
	AdditionalProperties any
	Required             []string
	Description          string
	Nullable             bool
	XSDKEnumLiteral      []any
	XSDKWiden            bool
	raw                  map[string]any
}

type propEntry struct {
	name string
	sch  *schema
}

// parseSchema converts a decoded map into an ordered schema. It walks a
// yaml.Node (which preserves key order) so `properties` field order is stable.
func parseSchemaNode(node *yaml.Node) *schema {
	if node == nil {
		return nil
	}
	// Resolve document node to its content.
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		// Non-object schema fragment (rare); decode generically.
		return &schema{}
	}
	s := &schema{raw: map[string]any{}}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		val := node.Content[i+1]
		switch key {
		case "$ref":
			s.Ref = val.Value
		case "type":
			var t any
			_ = val.Decode(&t)
			s.Type = t
		case "const":
			var c any
			_ = val.Decode(&c)
			s.Const = c
		case "enum":
			_ = val.Decode(&s.Enum)
		case "oneOf":
			s.OneOf = parseSchemaList(val)
		case "anyOf":
			s.AnyOf = parseSchemaList(val)
		case "allOf":
			s.AllOf = parseSchemaList(val)
		case "items":
			s.Items = parseSchemaNode(val)
		case "properties":
			s.Properties = parsePropList(val)
		case "additionalProperties":
			var ap any
			_ = val.Decode(&ap)
			s.AdditionalProperties = ap
		case "required":
			_ = val.Decode(&s.Required)
		case "description":
			s.Description = val.Value
		case "nullable":
			_ = val.Decode(&s.Nullable)
		case "x-sdk-enum-literal":
			_ = val.Decode(&s.XSDKEnumLiteral)
		case "x-sdk-widen":
			_ = val.Decode(&s.XSDKWiden)
		}
	}
	return s
}

func parseSchemaList(node *yaml.Node) []*schema {
	if node == nil || node.Kind != yaml.SequenceNode {
		return nil
	}
	out := make([]*schema, 0, len(node.Content))
	for _, c := range node.Content {
		out = append(out, parseSchemaNode(c))
	}
	return out
}

func parsePropList(node *yaml.Node) []propEntry {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	out := make([]propEntry, 0, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		out = append(out, propEntry{name: node.Content[i].Value, sch: parseSchemaNode(node.Content[i+1])})
	}
	return out
}

// typeStr returns the singular `type` string when the schema declares one, or
// the first non-null of a multi-type list.
func (s *schema) typeStr() string {
	switch t := s.Type.(type) {
	case string:
		return t
	case []any:
		for _, x := range t {
			if str, ok := x.(string); ok && str != "null" {
				return str
			}
		}
	}
	return ""
}

func (s *schema) typeList() []string {
	switch t := s.Type.(type) {
	case string:
		return []string{t}
	case []any:
		out := []string{}
		for _, x := range t {
			if str, ok := x.(string); ok {
				out = append(out, str)
			}
		}
		return out
	}
	return nil
}

func (s *schema) isObject() bool {
	ts := ""
	if str, ok := s.Type.(string); ok {
		ts = str
	}
	return (ts == "object" || (s.Type == nil && len(s.Properties) > 0)) &&
		len(s.OneOf) == 0 && len(s.AnyOf) == 0 && len(s.AllOf) == 0
}

// ---------------------------------------------------------------------------
// Naming
// ---------------------------------------------------------------------------

var nonIdent = regexp.MustCompile(`[^A-Za-z0-9_]`)

var goReserved = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true, "interface": true,
	"map": true, "package": true, "range": true, "return": true, "select": true,
	"struct": true, "switch": true, "type": true, "var": true,
}

// pascal splits on separators and PascalCases (mirrors the reference `pascal`).
func pascal(s string) string {
	parts := regexp.MustCompile(`[_\-\s.]`).Split(s, -1)
	var b strings.Builder
	for _, w := range parts {
		if w == "" {
			continue
		}
		b.WriteString(strings.ToUpper(w[:1]) + w[1:])
	}
	return b.String()
}

// typeName is the Go/type identifier for a schema NAME (a $def / component
// name). The audit compares class refs by LEAF name, so the leaf must equal the
// reference's `py_name(name)` leaf. The reference keeps names like `Record` /
// `Set` as-is (they aren't Python-reserved and Go doesn't reserve them either),
// so no built-in-collision rename is needed for this schema set (verified: no
// generated name collides with a Go keyword or an existing pkg symbol).
func typeName(name string) string {
	cleaned := strings.TrimLeft(nonIdent.ReplaceAllString(name, "_"), "_")
	if cleaned == "" {
		cleaned = "Schema"
	}
	if !isIdentStart(cleaned[0]) {
		cleaned = "Schema" + cleaned
	}
	// A generated name that lands on a Go keyword gets a trailing underscore
	// (the Go analog of the reference's reserved-word suffix). None do in the
	// current spec set, but keep the guard so a future spec addition fails safe
	// rather than emitting invalid Go.
	if goReserved[cleaned] {
		cleaned += "_"
	}
	return cleaned
}

func isIdentStart(b byte) bool {
	return b == '_' || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

// fieldName is the exported Go struct-field identifier for a wire key. It is
// only used for the Go static field; the wire key is preserved in the json tag.
func fieldName(wireKey string) string {
	p := pascal(wireKey)
	if p == "" {
		p = "Field"
	}
	if !isIdentStart(p[0]) {
		p = "F" + p
	}
	return p
}

func refName(ref string) string {
	seg := ref
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		seg = ref[i+1:]
	}
	return typeName(seg)
}

// ---------------------------------------------------------------------------
// Canonical (audit) type — mirrors the enumerator vocabulary + the reference's
// py_type canonicalization, so the drift gate compares the SAME shape python
// records. Written into the `gen:"..."` struct tag.
// ---------------------------------------------------------------------------

// canonModule is the class-ref module prefix. The diff tool compares class refs
// by LEAF name, so the prefix is cosmetic; use the reference's module so the
// emitted tags read identically to python's recorded types.
func canonModule(kind string) string {
	switch kind {
	case "swml":
		return "signalwire.core.swml_verbs_generated"
	case "post_prompt":
		return "signalwire.core.post_prompt_generated"
	case "swaig_request":
		return "signalwire.core.swaig_request_generated"
	default:
		return "signalwire.core.swaig_request_generated"
	}
}

// gen is the resolution context for one output module: the module used for local
// class refs, and the set of names declared in THIS document (so a ref to a name
// declared elsewhere still uses the right module — but leaf comparison makes the
// module cosmetic anyway).
type gen struct {
	module string // canon module for local refs
	// refModule maps a schema NAME to the canon module it is declared in. A ref
	// to an out-of-module name (post_prompt -> swaig_request's SwaigRequest)
	// records that name's home module. Cosmetic (leaf compare), but faithful.
	refModule map[string]string
}

// classRef renders a `$ref` (or a bare declared name) as `class:<module>.<Leaf>`.
func (g *gen) classRef(name string) string {
	mod := g.module
	if m, ok := g.refModule[name]; ok {
		mod = m
	}
	return "class:" + mod + "." + typeName(name)
}

// canonicalType maps a schema node to the audit-canonical type string. Mirrors
// the reference py_type: x-sdk markup first, then $ref / const / enum / allOf /
// union / typed scalar / array / object, with unions rendered as `union<...>`.
func (g *gen) canonicalType(s *schema) string {
	if s == nil {
		return "any"
	}
	// Field-markup overrides (formulaic-enrichment-as-markup).
	if len(s.XSDKEnumLiteral) > 0 {
		// A closed literal set of scalars — the enumerator absorbs a closed
		// string set to `string` (the reference records the same for a
		// Literal[...] of strings).
		return "string"
	}
	if s.XSDKWiden {
		switch s.typeStr() {
		case "integer":
			return "int"
		case "number":
			return "float"
		case "boolean":
			return "bool"
		default:
			return "string"
		}
	}
	if s.Ref != "" {
		// External file ref (SWMLObject.json) -> opaque object.
		if !strings.HasPrefix(s.Ref, "#/") && strings.HasSuffix(s.Ref, ".json") {
			return "dict<string,any>"
		}
		return g.classRef(refName(s.Ref))
	}
	if s.Const != nil {
		// A single const scalar — the reference records a Literal, which the
		// enumerator canonicalizes to `string` (proven against the oracle).
		return "string"
	}
	if len(s.Enum) > 0 {
		return "string"
	}
	if len(s.AllOf) == 1 {
		return g.canonicalType(s.AllOf[0])
	}
	if len(s.AllOf) > 1 {
		return "dict<string,any>"
	}
	if u := union(s); u != nil {
		return g.canonicalUnion(u)
	}
	// Multi-type list (e.g. ["string","null"]).
	tl := s.typeList()
	if len(tl) > 1 {
		parts := []string{}
		hasNull := false
		for _, t := range tl {
			if t == "null" {
				hasNull = true
				continue
			}
			sub := *s
			sub.Type = t
			parts = append(parts, g.canonicalType(&sub))
		}
		joined := dedupUnion(parts)
		if hasNull && joined != "any" {
			return "optional<" + joined + ">"
		}
		return joined
	}
	return g.canonicalScalar(s)
}

func (g *gen) canonicalScalar(s *schema) string {
	wrap := func(t string) string {
		if s.Nullable && t != "any" {
			return "optional<" + t + ">"
		}
		return t
	}
	switch s.typeStr() {
	case "string":
		return wrap("string")
	case "integer":
		return wrap("int")
	case "number":
		return wrap("float")
	case "boolean":
		return wrap("bool")
	case "null":
		return "optional<any>"
	case "array":
		return wrap("list<" + g.canonicalType(s.Items) + ">")
	case "object", "":
		if len(s.Properties) > 0 {
			return wrap("dict<string,any>")
		}
		if ap, ok := s.AdditionalProperties.(map[string]any); ok {
			apSch := parseSchemaFromMap(ap)
			return wrap("dict<string," + g.canonicalType(apSch) + ">")
		}
		return wrap("dict<string,any>")
	}
	return "any"
}

// canonicalUnion renders a oneOf/anyOf as `union<...>`, threading a `null`
// branch into an `optional<...>` on the sibling members (matches the reference:
// `int | None | SWMLVar` records `union<optional<int>,class:SWMLVar>`).
func (g *gen) canonicalUnion(branches []*schema) string {
	var members []string
	nullSeen := false
	for _, b := range branches {
		if b != nil && b.typeStr() == "null" && b.Ref == "" {
			nullSeen = true
			continue
		}
		members = append(members, g.canonicalType(b))
	}
	if nullSeen && len(members) > 0 {
		// Apply optional<> to the FIRST scalar member (mirrors the left-nested
		// `int | None` the reference forms: `union<optional<int>,...>`).
		for i, m := range members {
			if !strings.HasPrefix(m, "class:") && !strings.HasPrefix(m, "optional<") {
				members[i] = "optional<" + m + ">"
				nullSeen = false
				break
			}
		}
	}
	joined := dedupUnion(members)
	if nullSeen && joined != "any" {
		return "optional<" + joined + ">"
	}
	return joined
}

func dedupUnion(parts []string) string {
	seen := map[string]bool{}
	var out []string
	for _, p := range parts {
		if p == "" {
			continue
		}
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return "any"
	}
	if len(out) == 1 {
		return out[0]
	}
	return "union<" + strings.Join(out, ",") + ">"
}

func union(s *schema) []*schema {
	if len(s.OneOf) > 0 {
		return s.OneOf
	}
	if len(s.AnyOf) > 0 {
		return s.AnyOf
	}
	return nil
}

// ---------------------------------------------------------------------------
// Go runtime type — the most faithful COMPILABLE Go type for a field. Only used
// for the static struct field; the audit reads the gen: tag, not this.
// ---------------------------------------------------------------------------

func (g *gen) goType(s *schema) string {
	if s == nil {
		return "any"
	}
	if len(s.XSDKEnumLiteral) > 0 || s.XSDKWiden {
		// A closed/widened scalar surface: model as the base Go scalar.
		if s.XSDKWiden {
			switch s.typeStr() {
			case "integer":
				return "int"
			case "number":
				return "float64"
			case "boolean":
				return "bool"
			}
		}
		return "string"
	}
	if s.Ref != "" {
		if !strings.HasPrefix(s.Ref, "#/") && strings.HasSuffix(s.Ref, ".json") {
			return "map[string]any"
		}
		return "*" + refName(s.Ref)
	}
	if s.Const != nil || len(s.Enum) > 0 {
		return "string"
	}
	if len(s.AllOf) == 1 {
		return g.goType(s.AllOf[0])
	}
	if len(s.AllOf) > 1 {
		return "map[string]any"
	}
	if union(s) != nil {
		// Go has no sum type; `any` round-trips every branch and the audit type
		// lives in the gen: tag.
		return "any"
	}
	tl := s.typeList()
	if len(tl) > 1 {
		return "any"
	}
	switch s.typeStr() {
	case "string":
		return "string"
	case "integer":
		return "int"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		return "[]" + g.goType(s.Items)
	case "object", "":
		if len(s.Properties) > 0 {
			return "map[string]any"
		}
		if ap, ok := s.AdditionalProperties.(map[string]any); ok {
			return "map[string]" + g.goType(parseSchemaFromMap(ap))
		}
		return "map[string]any"
	}
	return "any"
}

func parseSchemaFromMap(m map[string]any) *schema {
	// Re-encode to a yaml.Node to reuse the ordered parser (small, rare path).
	var node yaml.Node
	_ = node.Encode(m)
	return parseSchemaNode(&node)
}

// ---------------------------------------------------------------------------
// Declaration emission
// ---------------------------------------------------------------------------

// declaration emits a Go declaration for one named schema: an object-with-props
// -> a struct; anything else -> a defined-type alias (so a `$ref` to it
// resolves). Each struct field carries a json tag (wire key + omitempty) and a
// gen tag (audit-canonical type).
func (g *gen) declaration(name string, s *schema) string {
	ident := typeName(name)
	var b strings.Builder
	if doc := firstLine(s.Description); doc != "" {
		fmt.Fprintf(&b, "// %s %s\n", ident, doc)
	}
	if s.isObject() && len(s.Properties) > 0 {
		fmt.Fprintf(&b, "type %s struct {\n", ident)
		for _, p := range s.Properties {
			canon := g.canonicalType(p.sch)
			got := g.goType(p.sch)
			fn := fieldName(p.name)
			tag := fmt.Sprintf("`json:%q gen:%q`", p.name+",omitempty", canon)
			if fd := firstLine(p.sch.Description); fd != "" {
				fmt.Fprintf(&b, "\t// %s %s\n", fn, fd)
			}
			fmt.Fprintf(&b, "\t%s %s %s\n", fn, got, tag)
		}
		b.WriteString("}\n")
		return b.String()
	}
	// Non-object schema -> a defined type alias. Use the Go runtime type; a
	// oneOf/anyOf-of-refs alias becomes `any` (the union of object shapes).
	fmt.Fprintf(&b, "type %s %s\n", ident, g.goType(s))
	return b.String()
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// ---------------------------------------------------------------------------
// Spec loading
// ---------------------------------------------------------------------------

func loadYAMLSchemas(path string) (map[string]*schema, []string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, nil, err
	}
	root := &doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	comps := mapChild(root, "components")
	schemasNode := mapChild(comps, "schemas")
	return orderedSchemas(schemasNode)
}

func loadJSONDefs(path string) (map[string]*schema, []string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, nil, err
	}
	root := &doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	defsNode := mapChild(root, "$defs")
	return orderedSchemas(defsNode)
}

func mapChild(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func orderedSchemas(node *yaml.Node) (map[string]*schema, []string, error) {
	out := map[string]*schema{}
	var order []string
	if node == nil || node.Kind != yaml.MappingNode {
		return out, order, nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		name := node.Content[i].Value
		out[name] = parseSchemaNode(node.Content[i+1])
		order = append(order, name)
	}
	return out, order, nil
}

// ---------------------------------------------------------------------------
// Emitters (one per spec) — mirror the reference generate_* functions.
// ---------------------------------------------------------------------------

const genHeaderTmpl = `// Code generated by cmd/generate-payloads; DO NOT EDIT.
//
// AUTO-GENERATED from %s — regenerate with:
//   go run ./cmd/generate-payloads
//
// %s

package %s
`

// emitSwaigRequest mirrors generate_swaig_request: SwaigRequest (+ the inline
// `argument` lifted to SwaigArgument).
func emitSwaigRequest(specPath string) (string, error) {
	schemas, _, err := loadYAMLSchemas(specPath)
	if err != nil {
		return "", err
	}
	req := schemas["SwaigRequest"]
	if req == nil {
		return "", fmt.Errorf("swaig-request.yaml: missing SwaigRequest")
	}
	g := &gen{module: canonModule("swaig_request"), refModule: map[string]string{}}

	var decls []string
	// Lift the inline `argument` object to a named SwaigArgument, then rewrite
	// the field to a $ref (so the struct field records class:SwaigArgument).
	props := make([]propEntry, 0, len(req.Properties))
	for _, p := range req.Properties {
		if p.name == "argument" && len(p.sch.Properties) > 0 {
			decls = append(decls, g.declaration("SwaigArgument", &schema{Type: "object", Properties: p.sch.Properties}))
			props = append(props, propEntry{name: "argument", sch: &schema{Ref: "#/components/schemas/SwaigArgument"}})
		} else {
			props = append(props, p)
		}
	}
	decls = append(decls, g.declaration("SwaigRequest", &schema{Type: "object", Properties: props, Description: req.Description}))

	body := strings.Join(decls, "\n")
	src := fmt.Sprintf(genHeaderTmpl,
		"porting-sdk/swaig-specs/swaig-request.yaml",
		"The SWAIG function-webhook REQUEST payload — the body a SWAIG function\n// handler RECEIVES. Open shape: every field optional; extra server keys tolerated.",
		"swaig") + "\n" + body
	return src, nil
}

// emitPostPrompt mirrors generate_post_prompt: one decl per component schema.
// SwaigRequest is declared in swaig_request_generated (same package), so its ref
// resolves locally.
func emitPostPrompt(specPath string) (string, error) {
	schemas, order, err := loadYAMLSchemas(specPath)
	if err != nil {
		return "", err
	}
	g := &gen{
		module: canonModule("post_prompt"),
		refModule: map[string]string{
			// SwaigRequest lives in the request module (leaf compare makes this
			// cosmetic, but keep it faithful to the reference recording).
			"SwaigRequest":  canonModule("swaig_request"),
			"SwaigArgument": canonModule("swaig_request"),
		},
	}
	var decls []string
	for _, name := range order {
		if name == "SwaigRequest" {
			continue // declared in swaig_request_generated.go (same package)
		}
		decls = append(decls, g.declaration(name, schemas[name]))
	}
	body := strings.Join(decls, "\n")
	src := fmt.Sprintf(genHeaderTmpl,
		"porting-sdk/swaig-specs/post-prompt.yaml",
		"The post-prompt callback payload — the call summary + enriched call log the\n// agent's post-prompt / onSummary handler RECEIVES. Open shape; extra keys tolerated.",
		"swaig") + "\n" + body
	return src, nil
}

// emitSwaigActions mirrors generate_swaig_actions: one <Verb>Action struct per
// object-shaped action value. (Only the CONFIG TYPE surface; the ergonomic
// builder methods live on FunctionResult.)
func emitSwaigActions(specPath string) (string, error) {
	schemas, _, err := loadYAMLSchemas(specPath)
	if err != nil {
		return "", err
	}
	sa := schemas["SwaigAction"]
	if sa == nil || len(sa.Properties) == 0 {
		return "", fmt.Errorf("swaig-response.yaml: missing SwaigAction.properties")
	}
	g := &gen{module: canonModule("swaig_request"), refModule: map[string]string{}}
	isObj := func(s *schema) bool {
		if s == nil {
			return false
		}
		ts, _ := s.Type.(string)
		return ts == "object" && len(s.Properties) > 0
	}
	// Sort verb keys for deterministic output (matches the reference sort).
	var verbs []string
	for _, p := range sa.Properties {
		verbs = append(verbs, p.name)
	}
	sort.Strings(verbs)
	verbSchema := map[string]*schema{}
	for _, p := range sa.Properties {
		verbSchema[p.name] = p.sch
	}
	var decls []string
	for _, verb := range verbs {
		s := verbSchema[verb]
		var branches []*schema
		if len(s.OneOf) > 0 {
			branches = s.OneOf
		} else if isObj(s) {
			branches = []*schema{s}
		}
		objIdx := 0
		for _, b := range branches {
			if !isObj(b) {
				continue
			}
			objIdx++
			name := pascal(verb) + "Action"
			if objIdx != 1 {
				name += fmt.Sprintf("%d", objIdx)
			}
			decls = append(decls, g.declaration(name, &schema{Type: "object", Properties: b.Properties}))
		}
	}
	body := strings.Join(decls, "\n")
	src := fmt.Sprintf(genHeaderTmpl,
		"porting-sdk/swaig-specs/swaig-response.yaml",
		"The typed SWAIG response-action CONFIG types (one <Verb>Action per object-\n// shaped action value). The ergonomic builder methods live on FunctionResult.",
		"swaig") + "\n" + body
	return src, nil
}

// handWrittenVerbs are verbs this port hand-writes with richer ergonomics; they
// are excluded from the <Verb>Config flatten (matches the reference hand_written
// set). Only affects which Config decls are emitted — every $defs schema is
// still declared unconditionally.
var handWrittenVerbs = map[string]bool{
	"answer": true, "hangup": true, "ai": true, "play": true, "say": true,
}

// emitSwmlVerbs mirrors generate_swml_verbs: one decl per schema.json $defs
// entry (object -> struct; else -> alias) + the flattened <Verb>Config structs
// from SWMLMethod.anyOf. (The verb METHOD surface — the reference's `_SwmlVerbs`
// protocol — is `_`-prefixed and NOT part of the cross-port oracle, so it is not
// emitted here; the config TYPE surface is what the drift gate compares.)
func emitSwmlVerbs(schemaPath string) (string, error) {
	defs, order, err := loadJSONDefs(schemaPath)
	if err != nil {
		return "", err
	}
	g := &gen{module: canonModule("swml"), refModule: map[string]string{}}
	var decls []string
	declared := map[string]bool{}
	emit := func(name string, s *schema) {
		if declared[name] {
			return
		}
		declared[name] = true
		decls = append(decls, g.declaration(name, s))
	}
	// 1. One declaration per $defs schema (so every $ref resolves).
	for _, name := range order {
		emit(name, defs[name])
	}
	// 2. Walk SWMLMethod.anyOf -> flatten each non-hand-written verb's inner
	//    schema into a <Verb>Config struct.
	if sm := defs["SWMLMethod"]; sm != nil {
		for _, ref := range sm.AnyOf {
			wrapper := refNameRaw(ref.Ref)
			wdef := defs[wrapper]
			if wdef == nil || len(wdef.Properties) == 0 {
				continue
			}
			verb := wdef.Properties[0].name
			if handWrittenVerbs[verb] {
				continue
			}
			inner := wdef.Properties[0].sch
			if inner.typeStr() == "string" || inner.Ref != "" {
				continue
			}
			hasInlineProps := inner.typeStr() == "object" && len(inner.Properties) > 0
			if len(inner.OneOf) == 0 && !hasInlineProps {
				continue
			}
			props := flattenUnion(defs, inner)
			if len(props) == 0 {
				continue
			}
			cfgName := pascal(verb) + "Config"
			desc := firstLine(inner.Description)
			if desc == "" {
				desc = "Add the " + verb + " verb."
			}
			emit(cfgName, &schema{Type: "object", Properties: props, Description: desc})
		}
	}
	body := strings.Join(decls, "\n")
	src := fmt.Sprintf(genHeaderTmpl,
		"porting-sdk/schema.json ($defs)",
		"The typed SWML verb CONFIG surface: one struct per schema.json $defs entry\n// (object -> struct; non-object -> defined-type alias) + the flattened <Verb>Config\n// payload shapes the SWML builder verb methods accept. Open shape; extra keys tolerated.",
		"swml") + "\n" + body
	return src, nil
}

func refNameRaw(ref string) string {
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		return ref[i+1:]
	}
	return ref
}

// flattenUnion resolves a (possibly $ref / allOf / oneOf) object schema to the
// UNION of its variants' properties (mirrors the reference `_flatten_union`).
func flattenUnion(defs map[string]*schema, s *schema) []propEntry {
	seen := map[string]bool{}
	var out []propEntry
	var walk func(s *schema)
	add := func(props []propEntry) {
		for _, p := range props {
			if !seen[p.name] {
				seen[p.name] = true
				out = append(out, p)
			}
		}
	}
	walk = func(s *schema) {
		if s == nil {
			return
		}
		if s.Ref != "" {
			walk(defs[refNameRaw(s.Ref)])
			return
		}
		for _, sub := range s.AllOf {
			walk(sub)
		}
		add(s.Properties)
		for _, sub := range s.OneOf {
			walk(sub)
		}
		for _, sub := range s.AnyOf {
			walk(sub)
		}
	}
	walk(s)
	return out
}

// ---------------------------------------------------------------------------
// Driver
// ---------------------------------------------------------------------------

type outputFile struct {
	path string
	src  func(psdk string) (string, error)
}

func gofmtSrc(src string) ([]byte, error) {
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return nil, fmt.Errorf("gofmt: %w\n---\n%s", err, src)
	}
	return formatted, nil
}

func resolvePortingSDK(repoRoot string) (string, error) {
	if p := os.Getenv("PORTING_SDK"); p != "" {
		if _, err := os.Stat(filepath.Join(p, "swaig-specs")); err == nil {
			return p, nil
		}
	}
	cand := filepath.Join(repoRoot, "..", "porting-sdk")
	if _, err := os.Stat(filepath.Join(cand, "swaig-specs")); err == nil {
		return filepath.Abs(cand)
	}
	return "", fmt.Errorf("porting-sdk not found (set $PORTING_SDK or clone adjacent)")
}

func findRepoRoot(start string) (string, error) {
	cur := start
	for {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("no go.mod above %s", start)
		}
		cur = parent
	}
}

func run() error {
	check := flag.Bool("check", false, "GEN-FRESH: exit non-zero if any generated file is stale")
	flag.Parse()

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	repoRoot, err := findRepoRoot(cwd)
	if err != nil {
		return err
	}
	psdk, err := resolvePortingSDK(repoRoot)
	if err != nil {
		if *check {
			return fmt.Errorf("generate-payloads --check: %w", err)
		}
		fmt.Fprintf(os.Stderr, "generate-payloads: %v — skipping (committed files kept)\n", err)
		return nil
	}

	outputs := []outputFile{
		{
			path: filepath.Join(repoRoot, "pkg", "swaig", "swaig_request_generated.go"),
			src: func(p string) (string, error) {
				return emitSwaigRequest(filepath.Join(p, "swaig-specs", "swaig-request.yaml"))
			},
		},
		{
			path: filepath.Join(repoRoot, "pkg", "swaig", "post_prompt_generated.go"),
			src: func(p string) (string, error) {
				return emitPostPrompt(filepath.Join(p, "swaig-specs", "post-prompt.yaml"))
			},
		},
		{
			path: filepath.Join(repoRoot, "pkg", "swaig", "swaig_actions_generated.go"),
			src: func(p string) (string, error) {
				return emitSwaigActions(filepath.Join(p, "swaig-specs", "swaig-response.yaml"))
			},
		},
		{
			path: filepath.Join(repoRoot, "pkg", "swml", "swml_verbs_generated.go"),
			src:  func(p string) (string, error) { return emitSwmlVerbs(filepath.Join(p, "schema.json")) },
		},
	}

	var stale []string
	for _, o := range outputs {
		src, err := o.src(psdk)
		if err != nil {
			return err
		}
		formatted, err := gofmtSrc(src)
		if err != nil {
			return fmt.Errorf("%s: %w", o.path, err)
		}
		if *check {
			existing, err := os.ReadFile(o.path)
			if err != nil || !bytes.Equal(existing, formatted) {
				stale = append(stale, o.path)
			}
			continue
		}
		if err := os.WriteFile(o.path, formatted, 0o644); err != nil {
			return err
		}
		fmt.Printf("generated %s\n", o.path)
	}

	if *check && len(stale) > 0 {
		fmt.Fprintf(os.Stderr, "\nGEN-FRESH FAIL: %d generated payload file(s) stale — run `go run ./cmd/generate-payloads` and commit:\n", len(stale))
		for _, f := range stale {
			fmt.Fprintf(os.Stderr, "  - %s\n", f)
		}
		return fmt.Errorf("stale generated files")
	}
	if *check {
		fmt.Println("GEN-FRESH: generated payloads match the canonical specs.")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
