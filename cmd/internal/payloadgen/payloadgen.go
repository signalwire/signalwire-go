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

func refName(ref string) string {
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
	default:
		return "signalwire.core.swaig_request_generated"
	}
}

type gen struct {
	module    string
	refModule map[string]string
}

func (g *gen) classRef(name string) string {
	mod := g.module
	if m, ok := g.refModule[name]; ok {
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
		if !strings.HasPrefix(s.Ref, "#/") && strings.HasSuffix(s.Ref, ".json") {
			return "dict<string,any>"
		}
		return g.classRef(refName(s.Ref))
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
func EmitSwaigRequest(raw []byte) (string, error) {
	schemas, _, err := loadYAMLSchemas(raw)
	if err != nil {
		return "", err
	}
	req := schemas["SwaigRequest"]
	if req == nil {
		return "", fmt.Errorf("swaig-request.yaml: missing SwaigRequest")
	}
	g := &gen{module: canonModule("swaig_request"), refModule: map[string]string{}}

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

	body := strings.Join(decls, "\n")
	src := fmt.Sprintf(genHeaderTmpl,
		"generate-swaig-payloads",
		"porting-sdk/swaig-specs/swaig-request.yaml",
		"The SWAIG function-webhook REQUEST payload — the body a SWAIG function\n// handler RECEIVES. Open shape: every field optional; extra server keys tolerated.",
		"swaig") + "\n" + body
	return src, nil
}

// EmitPostPrompt mirrors generate_post_prompt: one decl per component schema.
func EmitPostPrompt(raw []byte) (string, error) {
	schemas, order, err := loadYAMLSchemas(raw)
	if err != nil {
		return "", err
	}
	g := &gen{
		module: canonModule("post_prompt"),
		refModule: map[string]string{
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
		"generate-swaig-payloads",
		"porting-sdk/swaig-specs/post-prompt.yaml",
		"The post-prompt callback payload — the call summary + enriched call log the\n// agent's post-prompt / onSummary handler RECEIVES. Open shape; extra keys tolerated.",
		"swaig") + "\n" + body
	return src, nil
}

// EmitSwaigActions mirrors generate_swaig_actions: one <Verb>Action struct per
// object-shaped action value.
func EmitSwaigActions(raw []byte) (string, error) {
	schemas, _, err := loadYAMLSchemas(raw)
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
		"generate-swaig-payloads",
		"porting-sdk/swaig-specs/swaig-response.yaml",
		"The typed SWAIG response-action CONFIG types (one <Verb>Action per object-\n// shaped action value). The ergonomic builder methods live on FunctionResult.",
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
func EmitSwmlVerbs(raw []byte) (string, error) {
	defs, order, err := loadJSONDefs(raw)
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
	body := strings.Join(decls, "\n")
	src := fmt.Sprintf(genHeaderTmpl,
		"generate-swml-verbs",
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
