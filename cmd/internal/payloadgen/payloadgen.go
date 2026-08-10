// Package payloadgen holds the shared schema model + emission machinery for the
// typed READ-side SWAIG and SWML-verb payload generators. Its logic is lifted
// verbatim from the previously-consolidated cmd/generate-payloads command; the
// two split commands (cmd/generate-swaig-payloads, cmd/generate-swml-verbs) each
// call the exported EmitX entry points below, so the SWAIG and SWML surfaces are
// emitted byte-for-byte identically to the old consolidated generator.
//
// Open-shaped READ payloads: every field is optional (a Go pointer / slice / map
// / any zero value = absent), and every named struct is a plain struct the runtime
// unmarshals JSON into with extra server keys tolerated. The Go RUNTIME type of a
// field is the most faithful compilable shape; the AUDIT-canonical type (what the
// cross-port drift gate compares) is written into a `gen:"..."` struct tag so
// cmd/enumerate-signatures records it verbatim.
package payloadgen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/signalwire/signalwire-go/v3/cmd/internal/overlay"
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

// parseSchemaNode converts a decoded map into an ordered schema. It walks a
// yaml.Node (which preserves key order) so `properties` field order is stable.
func parseSchemaNode(node *yaml.Node) *schema {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
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

func typeName(name string) string {
	cleaned := strings.TrimLeft(nonIdent.ReplaceAllString(name, "_"), "_")
	if cleaned == "" {
		cleaned = "Schema"
	}
	if !isIdentStart(cleaned[0]) {
		cleaned = "Schema" + cleaned
	}
	if goReserved[cleaned] {
		cleaned += "_"
	}
	return cleaned
}

func isIdentStart(b byte) bool {
	return b == '_' || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

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

// ---------------------------------------------------------------------------
// Cross-file $ref resolution
// ---------------------------------------------------------------------------
//
// A ref of the form `<file>.yaml#/components/schemas/<Name>` points at a schema in a
// SIBLING spec file, whose types this generator emits into a different output FILE.
// Resolving it means three things, all of which must happen or the emitted file is
// broken:
//  1. VERIFY the target spec file exists and actually declares <Name> (else fail).
//  2. Return the Go type name for <Name>.
//  3. RECORD what the emitting file needs to reference the name.
//
// Before this existed only step 2 happened, via last-path-segment-wins: `refName`
// took the text after the final "/" and threw the file part away, so
// `swaig-response.yaml#/components/schemas/SwaigResponse` and a same-file
// `#/components/schemas/SwaigResponse` were indistinguishable. That emitted a name
// nothing declared — `undefined: SwaigResponse` at compile time — or, worse, silently
// bound to an unrelated same-named local schema. There is deliberately NO per-schema
// special case here: register a file once and every ref into it resolves.
//
// Step 3 is a NO-OP for Go specifically, and that is a property of the output layout
// rather than a shortcut: all three swaig-specs files emit into the SAME Go package
// (pkg/swaig), so a name declared by a sibling file is already in scope with no import
// statement. Python needs `if TYPE_CHECKING: from <module> import <Name>` and
// TypeScript needs an `import type`; Go needs nothing. The resolver still records the
// (module, name) pair because the canonical AUDIT type must name the module that
// DECLARES the schema — see crossFileModules and gen.refModule — and because if a
// future spec file ever emitted into a different Go package this is the hook that
// would have to grow an import.

// crossFileModules maps a spec FILE NAME to the canonical module that hosts its
// schemas, mirroring CROSS_FILE_MODULES in porting-sdk's
// generate_python_rest_types.py. A cross-file ref into a file that is not registered
// is an ERROR, not a silent widening to map[string]any: adding a new cross-file link
// is a deliberate act, not something that degrades quietly.
var crossFileModules = map[string]string{
	"swaig-request.yaml":  canonModule("swaig_request"),
	"swaig-response.yaml": canonModule("swaig_actions"),
}

// crossFileSearchDirs are the porting-sdk-relative dirs searched for a ref target.
var crossFileSearchDirs = []string{"swaig-specs"}

// crossFileResolver verifies cross-file $refs against the on-disk spec tree.
//
// specRoot is the porting-sdk root. It may be EMPTY, which means "no spec tree
// available": generate-swml-verbs reads schema.json and has no cross-file refs at all,
// and the *_generated.go-preserving skip path in the generator mains runs without a
// resolved porting-sdk. With an empty specRoot a cross-file ref is a hard error rather
// than an unverified pass — an unverifiable ref must not resolve.
type crossFileResolver struct {
	specRoot string
	// declared memoizes file name -> set of schema names it declares, so a spec is
	// read and parsed once per generator run however many refs point into it.
	declared map[string]map[string]bool
	// resolved records file name -> resolved Go type names, for the emitter's benefit
	// (see step 3 above) and so a caller can assert what a run actually resolved.
	resolved map[string]map[string]bool
}

func newCrossFileResolver(specRoot string) *crossFileResolver {
	return &crossFileResolver{
		specRoot: specRoot,
		declared: map[string]map[string]bool{},
		resolved: map[string]map[string]bool{},
	}
}

// moduleOf reports the module a cross-file-resolved type name was declared in, for the
// canonical audit tag. Only names this run actually resolved through a verified
// cross-file ref are present — an unresolved name falls back to the emitting module.
func (r *crossFileResolver) moduleOf(name string) (string, bool) {
	for mod, names := range r.resolved {
		if names[name] {
			return mod, true
		}
	}
	return "", false
}

// isCrossFileRef is true for `<file>#/<pointer>` — a ref with BOTH a file part and a
// pointer. A bare `#/components/schemas/X` (same document) and a whole-file
// `SWMLObject.json` (no pointer) are both NOT cross-file refs.
func isCrossFileRef(ref string) bool {
	i := strings.Index(ref, "#")
	return i > 0 && i < len(ref)-1
}

// isWholeFileJSONRef is true for a ref naming a whole JSON DOCUMENT with no pointer
// into it (`SWMLObject.json`) — a nested value this generator emits no type for, so an
// opaque map is the correct shape. Deliberately false for `x.json#/…`: that HAS a
// pointer, so it is a schema reference and belongs to the verifying resolver, not to
// the opaque fallback. (This is the `"#" not in ref` half of the reference generator's
// same guard, which go's copy had dropped.)
func isWholeFileJSONRef(ref string) bool {
	return !strings.HasPrefix(ref, "#/") && strings.HasSuffix(ref, ".json") && !strings.Contains(ref, "#")
}

// schemasDeclaredIn loads a sibling spec by file name and returns the schema names it
// declares, or an error naming every path searched.
func (r *crossFileResolver) schemasDeclaredIn(fileName string) (map[string]bool, error) {
	if got, ok := r.declared[fileName]; ok {
		return got, nil
	}
	if r.specRoot == "" {
		return nil, fmt.Errorf("cross-file $ref target %q cannot be verified: no spec root "+
			"available to this generator run", fileName)
	}
	var tried []string
	for _, d := range crossFileSearchDirs {
		p := filepath.Join(r.specRoot, d, fileName)
		tried = append(tried, p)
		// The file name is not free-form: resolve() rejects anything absent from the
		// closed crossFileModules registry before calling this, so p is <spec root> +
		// a fixed search dir + one of a fixed set of names.
		raw, err := os.ReadFile(p) //nolint:gosec // G304: developer-run codegen reading a spec/source path derived from the repo root or $PORTING_SDK, not from untrusted input.
		if err != nil {
			continue
		}
		schemas, _, err := loadYAMLSchemas(raw)
		if err != nil {
			return nil, fmt.Errorf("cross-file $ref target %q: %w", p, err)
		}
		names := make(map[string]bool, len(schemas))
		for n := range schemas {
			names[n] = true
		}
		r.declared[fileName] = names
		return names, nil
	}
	return nil, fmt.Errorf("cross-file $ref target spec file not found: %q (looked in: %s)",
		fileName, strings.Join(tried, ", "))
}

// resolve turns `<file>.yaml#/components/schemas/<Name>` into the Go type name for
// <Name>, having verified that <file> exists and declares it. It fails — naming both
// the file and the schema — on all three failure modes (unsupported pointer,
// unregistered file, undeclared schema) rather than falling back to a name nothing
// defines.
func (r *crossFileResolver) resolve(ref string) (string, error) {
	fileName, pointer, _ := strings.Cut(ref, "#")
	const want = "/components/schemas/"
	if !strings.HasPrefix(pointer, want) {
		return "", fmt.Errorf("unsupported cross-file $ref pointer %q in %q; only "+
			"'#%s<Name>' is resolvable", pointer, ref, want)
	}
	schemaName := pointer[len(want):]
	if schemaName == "" || strings.Contains(schemaName, "/") {
		return "", fmt.Errorf("unsupported cross-file $ref pointer %q in %q; only "+
			"'#%s<Name>' is resolvable", pointer, ref, want)
	}
	module, ok := crossFileModules[fileName]
	if !ok {
		return "", fmt.Errorf("cross-file $ref into unregistered spec file %q (schema %q); "+
			"add it to crossFileModules with the module that hosts its schemas", fileName, schemaName)
	}
	declared, err := r.schemasDeclaredIn(fileName)
	if err != nil {
		return "", err
	}
	if !declared[schemaName] {
		names := make([]string, 0, len(declared))
		for n := range declared {
			names = append(names, n)
		}
		sort.Strings(names)
		joined := strings.Join(names, ", ")
		if joined == "" {
			joined = "<none>"
		}
		return "", fmt.Errorf("cross-file $ref names schema %q which does not exist in %q "+
			"(it declares: %s)", schemaName, fileName, joined)
	}
	name := typeName(schemaName)
	if r.resolved[module] == nil {
		r.resolved[module] = map[string]bool{}
	}
	r.resolved[module][name] = true
	return name, nil
}

// refName resolves a $ref to the Go type name it names.
//
// A SAME-FILE ref (`#/components/schemas/X`) is the local schema X. A CROSS-FILE ref
// goes through the verifying resolver above. Errors are accumulated on the gen rather
// than returned, because refName is called from deep inside the type-expression
// builders (canonicalType / goType) whose signatures are `(*schema) string`; the
// emitting Emit* entry point checks g.err before returning a source string, so a
// failure is reported and NOTHING is written — it never degrades to a bad name.
func (g *gen) refName(ref string) string {
	if isCrossFileRef(ref) {
		name, err := g.crossFile.resolve(ref)
		if err != nil {
			g.fail(err)
			// The returned name is never emitted (the Emit* entry point aborts on
			// g.err); it exists only so the in-flight expression builder can finish.
			return "InvalidCrossFileRef"
		}
		return name
	}
	seg := ref
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		seg = ref[i+1:]
	}
	return typeName(seg)
}

// ---------------------------------------------------------------------------
// Canonical (audit) type
// ---------------------------------------------------------------------------

func canonModule(kind string) string {
	switch kind {
	case "swml":
		return "signalwire.core.swml_verbs_generated"
	case "post_prompt":
		return "signalwire.core.post_prompt_generated"
	case "swaig_request":
		return "signalwire.core.swaig_request_generated"
	case "swaig_actions":
		return "signalwire.core.swaig_actions_generated"
	default:
		return "signalwire.core.swaig_request_generated"
	}
}

type gen struct {
	module    string
	refModule map[string]string
	// overlay is the SDK-surface policy (x-sdk-overlay.yaml). Set on the SWML-verb
	// path where AIParams surfaces; nil (no-op) for the SWAIG payload paths.
	overlay *overlay.Overlay
	// crossFile verifies `<file>.yaml#/components/schemas/<Name>` refs against the
	// on-disk spec tree. Always non-nil (newGen); a nil specRoot inside it makes any
	// cross-file ref a hard error rather than an unverified pass.
	crossFile *crossFileResolver
	// err is the first cross-file resolution failure seen while building type
	// expressions. The Emit* entry points return it instead of a source string.
	err error
}

// newGen builds a gen with the cross-file resolver wired to specRoot (the porting-sdk
// root, or "" when this generator has no spec tree — see crossFileResolver).
func newGen(module, specRoot string, refModule map[string]string, ov *overlay.Overlay) *gen {
	if refModule == nil {
		refModule = map[string]string{}
	}
	return &gen{module: module, refModule: refModule, overlay: ov, crossFile: newCrossFileResolver(specRoot)}
}

// fail records the first resolution error; later ones are dropped so the reported
// error is the root one rather than whichever expression finished last.
func (g *gen) fail(err error) {
	if g.err == nil {
		g.err = err
	}
}

// classRef is the canonical AUDIT type for a named schema: `class:<module>.<Name>`.
//
// The module is the one that DECLARES the schema, which for a cross-file ref is NOT
// the emitting module. That mapping comes from the resolver — it recorded (module,
// name) when it verified the ref — so crossFileModules is the single source for it and
// no hand-map has to repeat it. refModule remains for the SAME-FILE case where a
// schema is declared in a sibling output file of the same package (post-prompt.yaml's
// local SwaigRequest / SwaigArgument refs, emitted into swaig_request_generated.go).
func (g *gen) classRef(name string) string {
	mod := g.module
	if m, ok := g.refModule[name]; ok {
		mod = m
	}
	if m, ok := g.crossFile.moduleOf(name); ok {
		mod = m
	}
	return "class:" + mod + "." + typeName(name)
}

func (g *gen) canonicalType(s *schema) string {
	if s == nil {
		return "any"
	}
	if len(s.XSDKEnumLiteral) > 0 {
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
		// A WHOLE-FILE JSON ref (`SWMLObject.json`, no `#` pointer) names a document
		// this generator emits no types for — correctly an opaque nested value. A
		// ref WITH a pointer is a schema reference and goes to refName, which
		// verifies the cross-file case instead of guessing at its last segment.
		if isWholeFileJSONRef(s.Ref) {
			return "dict<string,any>"
		}
		return g.classRef(g.refName(s.Ref))
	}
	if s.Const != nil {
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
// Go runtime type
// ---------------------------------------------------------------------------

func (g *gen) goType(s *schema) string {
	if s == nil {
		return "any"
	}
	if len(s.XSDKEnumLiteral) > 0 || s.XSDKWiden {
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
		if isWholeFileJSONRef(s.Ref) {
			return "map[string]any"
		}
		return "*" + g.refName(s.Ref)
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
	var node yaml.Node
	_ = node.Encode(m)
	return parseSchemaNode(&node)
}

// ---------------------------------------------------------------------------
// Declaration emission
// ---------------------------------------------------------------------------

func (g *gen) declaration(name string, s *schema) string {
	ident := typeName(name)
	var b strings.Builder
	if doc := firstLine(s.Description); doc != "" {
		fmt.Fprintf(&b, "// %s %s\n", ident, doc)
	}
	if s.isObject() && len(s.Properties) > 0 {
		fmt.Fprintf(&b, "type %s struct {\n", ident)
		for _, p := range s.Properties {
			// SDK-surface policy from the single overlay (x-sdk-overlay.yaml), matched
			// by (wire field, SPEC schema name). `name` is the schema name as it
			// appears in the spec ($defs key) — NOT the emitted Go ident. Hidden →
			// drop from the surface entirely (still on the wire); deprecated → emit
			// with a Go // Deprecated: doc comment.
			if g.overlay.Hidden(p.name, name) {
				continue
			}
			canon := g.canonicalType(p.sch)
			got := g.goType(p.sch)
			fn := fieldName(p.name)
			tag := fmt.Sprintf("`json:%q gen:%q`", p.name+",omitempty", canon)
			if g.overlay.Deprecated(p.name, name) {
				fmt.Fprintf(&b, "\t// Deprecated: %s\n", p.name)
			} else if fd := firstLine(p.sch.Description); fd != "" {
				fmt.Fprintf(&b, "\t// %s %s\n", fn, fd)
			}
			fmt.Fprintf(&b, "\t%s %s %s\n", fn, got, tag)
		}
		b.WriteString("}\n")
		return b.String()
	}
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

func loadYAMLSchemas(raw []byte) (map[string]*schema, []string, error) {
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

func loadJSONDefs(raw []byte) (map[string]*schema, []string, error) {
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

// genHeaderTmpl is parameterized by the emitting command name (%[1]s) so each
// generated file names the command that PRODUCES it: the SWAIG emitters pass
// "generate-swaig-payloads", EmitSwmlVerbs passes "generate-swml-verbs". Remaining
// slots: %[2]s = source spec path, %[3]s = file description, %[4]s = package.
const genHeaderTmpl = `// Code generated by cmd/%[1]s; DO NOT EDIT.
//
// AUTO-GENERATED from %[2]s — regenerate with:
//   go run ./cmd/%[1]s
//
// %[3]s

package %[4]s
`

// EmitSwaigRequest mirrors generate_swaig_request: SwaigRequest (+ the inline
// `argument` lifted to SwaigArgument). raw is the swaig-request.yaml bytes.
func EmitSwaigRequest(raw []byte, specRoot string) (string, error) {
	schemas, _, err := loadYAMLSchemas(raw)
	if err != nil {
		return "", err
	}
	req := schemas["SwaigRequest"]
	if req == nil {
		return "", fmt.Errorf("swaig-request.yaml: missing SwaigRequest")
	}
	g := newGen(canonModule("swaig_request"), specRoot, nil, nil)

	var decls []string
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
	if g.err != nil {
		return "", g.err
	}

	body := strings.Join(decls, "\n")
	src := fmt.Sprintf(genHeaderTmpl,
		"generate-swaig-payloads",
		"porting-sdk/swaig-specs/swaig-request.yaml",
		"The SWAIG function-webhook REQUEST payload — the body a SWAIG function\n// handler RECEIVES. Open shape: every field optional; extra server keys tolerated.",
		"swaig") + "\n" + body
	return src, nil
}

// EmitPostPrompt mirrors generate_post_prompt: one decl per component schema.
func EmitPostPrompt(raw []byte, specRoot string) (string, error) {
	schemas, order, err := loadYAMLSchemas(raw)
	if err != nil {
		return "", err
	}
	// SwaigRequest / SwaigArgument are declared in swaig_request_generated.go — the
	// same Go PACKAGE, a different output FILE — so a bare `#/components/schemas/
	// SwaigRequest` ref here compiles, but the audit tag must still name the module
	// that declares it. The cross-file SwaigResponse / SwaigAction entries that used
	// to sit alongside them are GONE: the verifying resolver records their module from
	// crossFileModules, so listing them here would duplicate that mapping in a second
	// place free to drift from it.
	g := newGen(canonModule("post_prompt"), specRoot, map[string]string{
		"SwaigRequest":  canonModule("swaig_request"),
		"SwaigArgument": canonModule("swaig_request"),
	}, nil)
	var decls []string
	for _, name := range order {
		decls = append(decls, g.declaration(name, schemas[name]))
	}
	if g.err != nil {
		return "", g.err
	}
	body := strings.Join(decls, "\n")
	src := fmt.Sprintf(genHeaderTmpl,
		"generate-swaig-payloads",
		"porting-sdk/swaig-specs/post-prompt.yaml",
		"The post-prompt callback payload — the call summary + enriched call log the\n// agent's post-prompt / onSummary handler RECEIVES. Open shape; extra keys tolerated.",
		"swaig") + "\n" + body
	return src, nil
}

// EmitSwaigActions mirrors generate_swaig_actions: one <Verb>Action struct per
// object-shaped action value.
func EmitSwaigActions(raw []byte, specRoot string) (string, error) {
	schemas, _, err := loadYAMLSchemas(raw)
	if err != nil {
		return "", err
	}
	sa := schemas["SwaigAction"]
	if sa == nil || len(sa.Properties) == 0 {
		return "", fmt.Errorf("swaig-response.yaml: missing SwaigAction.properties")
	}
	g := newGen(canonModule("swaig_actions"), specRoot, nil, nil)
	isObj := func(s *schema) bool {
		if s == nil {
			return false
		}
		ts, _ := s.Type.(string)
		return ts == "object" && len(s.Properties) > 0
	}
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
	// envProps carries one entry per action verb for the SwaigAction ENVELOPE. Every
	// inline-object branch that lifted to a named <Verb>Action struct is replaced by
	// a local $ref to it, so the ENVELOPE field's canonical type NAMES the lifted
	// class — `union<string,class:….ContextSwitchAction>` — exactly as the reference
	// records it. Leaving the inline object in place would widen the audit type to
	// `dict<string,any>` and read as a missing field to the DRIFT gate.
	envProps := make([]propEntry, 0, len(verbs))
	for _, verb := range verbs {
		s := verbSchema[verb]
		var branches []*schema
		if len(s.OneOf) > 0 {
			branches = s.OneOf
		} else if isObj(s) {
			branches = []*schema{s}
		}
		objIdx := 0
		// envBranches mirrors `branches` with each lifted object swapped for its $ref.
		envBranches := make([]*schema, 0, len(branches))
		for _, b := range branches {
			if !isObj(b) {
				envBranches = append(envBranches, b)
				continue
			}
			objIdx++
			name := pascal(verb) + "Action"
			if objIdx != 1 {
				name += fmt.Sprintf("%d", objIdx)
			}
			decls = append(decls, g.declaration(name, &schema{Type: "object", Properties: b.Properties}))
			envBranches = append(envBranches, &schema{Ref: "#/components/schemas/" + name})
		}
		switch {
		case objIdx == 0:
			// Nothing lifted (scalar / array / open object): the spec schema already
			// canonicalizes correctly.
			envProps = append(envProps, propEntry{name: verb, sch: s})
		case len(s.OneOf) == 0:
			// A single object-with-properties: one named struct, referenced directly.
			envProps = append(envProps, propEntry{
				name: verb,
				sch:  &schema{Ref: envBranches[0].Ref, Description: s.Description},
			})
		default:
			// A union: keep it a union, with the object branches now naming their
			// lifted structs. goType still widens the Go field to `any` (Go has no
			// sum type) while the canonical tag stays precise.
			envProps = append(envProps, propEntry{
				name: verb,
				sch:  &schema{OneOf: envBranches, Description: s.Description},
			})
		}
	}
	// The response ENVELOPE types. SwaigAction is the action OBJECT (one or more
	// action keys set at once — the engine dispatches every recognized key), and
	// SwaigResponse is the {response, action, post_process} body a handler returns.
	// They live here, alongside the per-action value types, because this is the
	// module that owns swaig-response.yaml — which is what makes
	// `swaig-response.yaml#/components/schemas/SwaigResponse` resolvable from
	// post-prompt.yaml (see EmitPostPrompt's refModule).
	decls = append(decls, g.declaration("SwaigAction", &schema{
		Type: "object", Properties: envProps, Description: sa.Description,
	}))
	if sr := schemas["SwaigResponse"]; sr != nil {
		decls = append(decls, g.declaration("SwaigResponse", sr))
	} else {
		return "", fmt.Errorf("swaig-response.yaml: missing SwaigResponse")
	}
	if g.err != nil {
		return "", g.err
	}
	body := strings.Join(decls, "\n")
	src := fmt.Sprintf(genHeaderTmpl,
		"generate-swaig-payloads",
		"porting-sdk/swaig-specs/swaig-response.yaml",
		"The typed SWAIG response-action CONFIG types (one <Verb>Action per object-\n// shaped action value) plus the SwaigAction/SwaigResponse envelope. The\n// ergonomic builder methods live on FunctionResult.",
		"swaig") + "\n" + body
	return src, nil
}

// handWrittenVerbs are verbs this port hand-writes with richer ergonomics; they
// are excluded from the <Verb>Config flatten (matches the reference hand_written
// set).
var handWrittenVerbs = map[string]bool{
	"answer": true, "hangup": true, "ai": true, "play": true, "say": true,
}

// EmitSwmlVerbs mirrors generate_swml_verbs: one decl per schema.json $defs entry
// (object -> struct; else -> alias) + the flattened <Verb>Config structs from
// SWMLMethod.anyOf. raw is the schema.json bytes.
func EmitSwmlVerbs(raw []byte, ov *overlay.Overlay, specRoot string) (string, error) {
	defs, order, err := loadJSONDefs(raw)
	if err != nil {
		return "", err
	}
	g := newGen(canonModule("swml"), specRoot, nil, ov)
	var decls []string
	declared := map[string]bool{}
	emit := func(name string, s *schema) {
		if declared[name] {
			return
		}
		declared[name] = true
		decls = append(decls, g.declaration(name, s))
	}
	for _, name := range order {
		emit(name, defs[name])
	}
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
	if g.err != nil {
		return "", g.err
	}
	body := strings.Join(decls, "\n")
	src := fmt.Sprintf(genHeaderTmpl,
		"generate-swml-verbs",
		"porting-sdk/schema.json ($defs)",
		"The typed SWML verb CONFIG surface: one struct per schema.json $defs entry\n// (object -> struct; non-object -> defined-type alias) + the flattened <Verb>Config\n// payload shapes the SWML builder verb methods accept. Open shape; extra keys tolerated.",
		"swml") + "\n" + body
	return src, nil
}

// refNameRaw is the RAW (un-typeName'd) final segment of a ref, used to index the SWML
// schema.json `$defs` map by its literal key.
//
// SAME-DOCUMENT refs only, and unlike refName it must stay raw because the caller looks
// the result up in a map keyed by the spec's own spelling. schema.json's one non-local
// ref is the whole-file `SWMLObject.json` (no pointer), which isWholeFileJSONRef routes
// to an opaque map before reaching here. A ref with a file part AND a pointer would have
// its file part discarded, so it is refused rather than mis-indexed.
func refNameRaw(ref string) string {
	if isCrossFileRef(ref) {
		// Callers index `defs` with the result and treat a miss as "nothing to flatten",
		// so returning a discarded-file-part name here would degrade silently — the
		// exact failure mode the resolver above exists to remove. The SWML emitter has
		// no cross-file link to resolve; if one is ever added, this is the hook that
		// must grow the verifying path.
		panic(fmt.Sprintf("payloadgen: cross-file $ref %q reached refNameRaw, which "+
			"resolves same-document $defs keys only. Give the SWML emitter the verifying "+
			"crossFileResolver path rather than letting the file part be discarded", ref))
	}
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		return ref[i+1:]
	}
	return ref
}

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
