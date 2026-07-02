// Types emitter for cmd/generate-rest.
//
// Emits <ns>_types_generated.go (package namespaces) per REST spec: one Go type
// per components/schemas entry (structs for object schemas, defined-string types
// for enums, scalar-format alias types, `any`/`[]T`/`map[string]any` aliases for
// unions / arrays / open objects), mirroring the TypeScript generator
// (signalwire-typescript/scripts/generate-rest-types.ts generateForSpec /
// declaration / tsType). Plus per-operation Request/Response type aliases where
// the op body/response is INLINE (no $ref).
//
// PORT_PHILOSOPHY_GO.md typing rules:
//
//	required field  = value type T
//	optional field  = pointer *T           (nil = unset, → optional<T> in the oracle)
//	enum            = defined-string type + typed-const block (aws-sdk-go-v2 style)
//	$ref            = the referenced Go type
//	array           = []T
//	open object     = map[string]any
//	scalar formats  = defined-string alias types (docid/uuid/jwt, matching the
//	                  oracle's datasphere_types_generated.docid etc.)
//	oneOf/anyOf     = `any` (Go has no union type; referenced-only alias)
//
// The struct FIELD wire keys come from the schema property name (json tag), so a
// typed struct marshals to the SAME snake wire body the loose map form produced.
package main

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Schema-name sanitisation to a valid Go identifier.
//
// A components/schemas key can carry dots ("Types.StatusCodes.StatusCode400");
// fold every non-identifier rune to "_" (matching the TS generator's tsName +
// the Python reference's ref_name, so the LEAF the surface diff compares is the
// identical token across ports). Go has no lib-global shadowing problem, so the
// TS RESERVED_GLOBALS handling is inert here.
// ---------------------------------------------------------------------------

func typeGoName(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	s := strings.TrimLeft(b.String(), "_")
	if s == "" {
		return "Schema"
	}
	// A leading digit is not a valid identifier start.
	if s[0] >= '0' && s[0] <= '9' {
		s = "Schema_" + s
	}
	return s
}

// refLeafGoName resolves a $ref to the sanitised Go type name of its leaf.
func refLeafGoName(ref string) string {
	return typeGoName(refLeaf(ref))
}

// ---------------------------------------------------------------------------
// schema helpers (ordered-node view; the emitter reads the raw components node).
// ---------------------------------------------------------------------------

// schemaType returns the (first) declared `type:` of a schema node ("" if none).
// A `type:` may be a scalar or a sequence (e.g. ["string","null"]).
func schemaType(node *yaml.Node) (typ string, nullable bool) {
	t := mapChild(node, "type")
	if t == nil {
		return "", false
	}
	if t.Kind == yaml.ScalarNode {
		return t.Value, false
	}
	if t.Kind == yaml.SequenceNode {
		for _, c := range t.Content {
			if c.Value == "null" {
				nullable = true
			} else if typ == "" {
				typ = c.Value
			}
		}
	}
	return typ, nullable
}

func seqChild(node *yaml.Node, key string) *yaml.Node {
	c := mapChild(node, key)
	if c != nil && c.Kind == yaml.SequenceNode {
		return c
	}
	return nil
}

// ---------------------------------------------------------------------------
// goFieldType — the Go type expression for a property/element schema (the value
// side of a struct field, an array element, etc.). Mirrors tsType(): follows
// $ref/allOf/oneOf/enum/array/object. Does NOT apply optional pointer-wrapping —
// the caller adds `*` for optional fields.
// ---------------------------------------------------------------------------

func goFieldType(schemas, node *yaml.Node) string {
	if node == nil {
		return "any"
	}

	// $ref → the referenced Go type (external .json ref → open map).
	if ref := scalarChild(node, "$ref"); ref != "" {
		if !strings.HasPrefix(ref, "#/") && strings.HasSuffix(ref, ".json") {
			return "map[string]any"
		}
		return refLeafGoName(ref)
	}

	// const → the base scalar (Go has no literal types; the wire value is a scalar).
	if mapChild(node, "const") != nil {
		return "string"
	}

	// enum → the base scalar type of an inline enum field (a NAMED enum schema
	// becomes a defined-string type at declaration time; an inline enum on a field
	// carries the same wire scalar, so use the scalar). Determine from `type:`.
	if mapChild(node, "enum") != nil {
		t, _ := schemaType(node)
		return goScalar(t)
	}

	// allOf: a single member (usually a $ref wrapper adding a description) → that
	// member's type; multiple members → an open map (Go can't intersect structs).
	if allOf := seqChild(node, "allOf"); allOf != nil {
		if len(allOf.Content) == 1 {
			return goFieldType(schemas, allOf.Content[0])
		}
		return "map[string]any"
	}

	// oneOf / anyOf → `any` (Go has no union type).
	if seqChild(node, "oneOf") != nil || seqChild(node, "anyOf") != nil {
		return "any"
	}

	t, _ := schemaType(node)
	switch t {
	case "string":
		return "string"
	case "integer":
		return "int"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "null":
		return "any"
	case "array":
		return "[]" + goFieldType(schemas, mapChild(node, "items"))
	case "object", "":
		// object with known properties → an inline open map (the oracle records an
		// inline object property as a plain dict; the TS generator collapses inline
		// method-param objects to Record<string,unknown> the same way). A free-form
		// object → open map. A bare no-type schema → open map (any JSON).
		return "map[string]any"
	default:
		return "any"
	}
}

// goScalar maps an OpenAPI scalar `type:` to a Go scalar type.
func goScalar(t string) string {
	switch t {
	case "integer":
		return "int"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	default:
		return "string"
	}
}

// ---------------------------------------------------------------------------
// Declarations — one Go type per components/schemas entry.
// ---------------------------------------------------------------------------

// isObjectSchema mirrors the TS `isObject` test: type:object (or no type but
// properties present) AND not a oneOf/anyOf/allOf combinator → an interface/struct.
func isObjectSchema(node *yaml.Node) bool {
	if seqChild(node, "oneOf") != nil || seqChild(node, "anyOf") != nil || seqChild(node, "allOf") != nil {
		return false
	}
	t, _ := schemaType(node)
	props := mapChild(node, "properties")
	return (t == "object" || (t == "" && props != nil)) && props != nil && props.Kind == yaml.MappingNode
}

// emitTypeDecl emits the Go declaration for one schema. Object schemas become
// structs (typed fields, json wire-key tags, optional → *T); enums become a
// defined-string type + a typed const block; scalar/array/union schemas become
// alias types.
func emitTypeDecl(b *strings.Builder, schemas *yaml.Node, rawName string, node *yaml.Node) {
	goName := typeGoName(rawName)

	// Object → struct.
	if isObjectSchema(node) {
		emitStructDecl(b, schemas, goName, node)
		return
	}

	// Named enum → defined-string type + const block (only for string enums; a
	// numeric enum keeps its scalar alias).
	if enum := seqChild(node, "enum"); enum != nil {
		t, _ := schemaType(node)
		if t == "string" || t == "" {
			emitEnumDecl(b, goName, enum)
			return
		}
	}

	// Everything else → a type alias (scalar format alias like docid/uuid, an
	// array top-level, an allOf/oneOf/anyOf union, or a free-form object map).
	fmt.Fprintf(b, "type %s %s\n\n", goName, goFieldType(schemas, node))
}

// emitStructDecl emits a Go struct for an object schema. Required fields are
// value-typed; optional fields are pointer-typed (nil = unset). The wire key is
// the schema property name (json tag), so the struct marshals to the same body.
func emitStructDecl(b *strings.Builder, schemas *yaml.Node, goName string, node *yaml.Node) {
	required := map[string]bool{}
	if req := seqChild(node, "required"); req != nil {
		for _, r := range req.Content {
			required[r.Value] = true
		}
	}
	props := mapChild(node, "properties")
	fmt.Fprintf(b, "type %s struct {\n", goName)
	used := map[string]bool{}
	for i := 0; i+1 < len(props.Content); i += 2 {
		wireKey := props.Content[i].Value
		propNode := props.Content[i+1]
		fieldName := structFieldName(wireKey)
		for used[fieldName] {
			fieldName += "_"
		}
		used[fieldName] = true
		typ := goFieldType(schemas, propNode)
		// Optional field → pointer (nil = unset → optional<T>). A field already
		// modelled as a map/slice/any is a nilable reference type; still pointer-
		// wrap value types (string/int/float/bool/defined-name) so the oracle
		// records optional<...>. Slices/maps/any stay as-is (they are nilable and
		// the oracle records list<...>/dict/any without an optional wrapper for the
		// port's reference-typed fields).
		if !required[wireKey] {
			typ = optionalGoType(typ)
		}
		fmt.Fprintf(b, "\t%s %s `json:%q`\n", fieldName, typ, wireKey+",omitempty")
	}
	b.WriteString("}\n\n")
}

// optionalGoType pointer-wraps a value type for an optional field. Reference
// types (slice/map/any) are already nilable and are left unwrapped.
func optionalGoType(typ string) string {
	if strings.HasPrefix(typ, "[]") || strings.HasPrefix(typ, "map[") || typ == "any" {
		return typ
	}
	if strings.HasPrefix(typ, "*") {
		return typ
	}
	return "*" + typ
}

// sdkEnumMarker is the doc-comment sentinel prefixed to an x-sdk-enum-derived
// named enum type. enumerate-surface recognizes it to surface the enum as a
// public class (the reference exports these enums as public API, unlike the
// inline schema enums which are referenced-only defined-string types).
const sdkEnumMarker = "// sdk-enum (x-sdk-enum): surfaced public enum type."

// emitEnumDecl emits `type X string` + a typed const block (aws-sdk-go-v2 style).
// When surfaced is true (an x-sdk-enum-derived public enum), it prepends the
// sdkEnumMarker so enumerate-surface records it as a surface class.
func emitEnumDecl(b *strings.Builder, goName string, enum *yaml.Node) {
	emitEnumDeclX(b, goName, enum, false)
}

func emitEnumDeclX(b *strings.Builder, goName string, enum *yaml.Node, surfaced bool) {
	if surfaced {
		b.WriteString(sdkEnumMarker + "\n")
	}
	fmt.Fprintf(b, "type %s string\n\n", goName)
	if len(enum.Content) == 0 {
		return
	}
	b.WriteString("const (\n")
	used := map[string]bool{}
	for _, v := range enum.Content {
		if v.Value == "" {
			continue
		}
		cname := goName + enumConstSuffix(v.Value)
		for used[cname] {
			cname += "_"
		}
		used[cname] = true
		fmt.Fprintf(b, "\t%s %s = %q\n", cname, goName, v.Value)
	}
	b.WriteString(")\n\n")
}

// enumConstSuffix turns an enum wire value into a PascalCase const-name suffix.
func enumConstSuffix(v string) string {
	parts := strings.FieldsFunc(v, func(r rune) bool {
		return !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	var out strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		out.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	s := out.String()
	if s == "" {
		s = "Value"
	}
	if s[0] >= '0' && s[0] <= '9' {
		s = "V" + s
	}
	return s
}

// ---------------------------------------------------------------------------
// emitTypesFile — the whole <ns>_types_generated.go module.
// ---------------------------------------------------------------------------

// emittedTypeNames tracks Go type names already declared across ALL specs. The
// REST types modules all live in ONE Go package (namespaces), unlike TS/Python
// which put each spec in its own module; the shared SWML-schema types (AI, Cond,
// …) and shared error types (Types_StatusCodes_*) + a handful of cross-spec
// resource types therefore recur across specs. Go forbids re-declaring a type in
// the same package, so each unique type NAME is emitted ONCE (first spec in
// actualSpecDirs order wins). This exactly matches the surface fold: a type the
// reference declares in >1 `_types_generated` module folds to a single
// `gen-type.<Leaf>` symbol, and a port carrying it once matches (the surface diff
// compares leaf NAMES; the field-level bodies of the 6 genuinely-divergent
// cross-spec names are wire-compatible same-endpoint-family shapes and the gates
// never compare their fields). So the dedup keeps the surface + drift both parity-
// clean while satisfying the Go one-package constraint.
// handOwnedTypeNames are generated-schema type names the namespaces package
// ALREADY declares by hand (with richer ergonomics than the generator would
// emit), so the generator must NOT re-declare them (Go forbids the duplicate).
// PhoneCallHandler (the x-sdk-enum public enum) is hand-written in
// pkg/rest/namespaces/call_handler.go with a doc table + AllPhoneCallHandlers()
// helper; it carries the sdk-enum surface marker there so enumerate-surface still
// records it as the reference's relay_rest_types_generated.PhoneCallHandler class.
var handOwnedTypeNames = map[string]bool{
	"PhoneCallHandler": true,
}

func newEmittedTypeNames() map[string]bool {
	m := map[string]bool{}
	for k := range handOwnedTypeNames {
		m[k] = true // pre-seed so the generator skips the hand-owned decls
	}
	return m
}

var emittedTypeNames = newEmittedTypeNames()

func emitTypesFile(sd *specDoc) (string, error) {
	schemas, err := componentsSchemas(sd)
	if err != nil {
		return "", err
	}
	if schemas == nil || schemas.Kind != yaml.MappingNode || len(schemas.Content) == 0 {
		return "", nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, genHeader,
		fmt.Sprintf("Generated REST wire types for the %q namespace spec — one Go type per\n// components/schemas entry (structs for objects, defined-string types for enums,\n// alias types for scalars/arrays/unions). Fields carry json wire-key tags so a\n// typed struct marshals to the same snake body the loose map form produced.\n//\n// Types whose Go name was already declared by an earlier spec (shared SWML-schema\n// types, shared error types) are omitted here — the whole REST types surface lives\n// in ONE Go package, so each unique type name is declared once (see\n// emittedTypeNames in cmd/generate-rest/types.go).", sd.name))
	b.WriteString("\n")
	// Emit in the spec's declared component order (matching the TS/Python
	// generators). Skip a type whose Go name an earlier spec already declared.
	emitted := 0
	for i := 0; i+1 < len(schemas.Content); i += 2 {
		rawName := schemas.Content[i].Value
		node := schemas.Content[i+1]
		goName := typeGoName(rawName)
		// x-sdk-enum markup: emit an ADDITIONAL named public enum type (the reference
		// exports it as public API, e.g. PhoneCallHandler, so callers can write
		// PhoneCallHandler.AiAgent instead of the bare string). Emitted BEFORE the
		// schema's own type so it is declared once; surfaced as a class.
		if xe := scalarChild(node, "x-sdk-enum"); xe != "" {
			en := typeGoName(xe)
			if !emittedTypeNames[en] {
				emittedTypeNames[en] = true
				if enum := seqChild(node, "enum"); enum != nil {
					emitEnumDeclX(&b, en, enum, true)
					emitted++
				}
			}
		}
		if emittedTypeNames[goName] {
			continue
		}
		emittedTypeNames[goName] = true
		emitTypeDecl(&b, schemas, rawName, node)
		emitted++
	}
	if emitted == 0 {
		return "", nil
	}
	return b.String(), nil
}

// ---------------------------------------------------------------------------
// Response / request type resolution for wiring the resource methods.
//
// operationResponseType returns the Go type of an op's 200/201/2XX response
// $ref (the item type), "" when the response is inline / absent. Used to type a
// method return as (*<XResponse>, error). arrayItem is true when the response is
// an array of the ref type (a list<T> response) — the method still returns the
// whole response object, so we type it as the ref (a list wrapper is its own
// schema in these specs); arrayItem is reported for completeness.
// ---------------------------------------------------------------------------

func operationResponseType(sd *specDoc, op opInfo) (goType string, err error) {
	opNode := sd.rawOp(op)
	if opNode == nil {
		return "", nil
	}
	responses := mapChild(opNode, "responses")
	if responses == nil {
		return "", nil
	}
	var ok *yaml.Node
	for _, code := range []string{"200", "201", "2XX"} {
		if n := mapChild(responses, code); n != nil {
			ok = n
			break
		}
	}
	if ok == nil {
		return "", nil
	}
	content := mapChild(ok, "content")
	if content == nil || content.Kind != yaml.MappingNode || len(content.Content) < 2 {
		return "", nil
	}
	sch := mapChild(content.Content[1], "schema")
	if sch == nil {
		return "", nil
	}
	if ref := scalarChild(sch, "$ref"); ref != "" {
		return refLeafGoName(ref), nil
	}
	// array whose items are a $ref → the item type is not the whole response; the
	// spec models list responses as their own wrapper schema, so an inline array
	// response has no named type — fall back to the open map.
	return "", nil
}

// operationBodyFieldTypes returns, for an operation whose body is exploded into
// a params struct, the Go type of each wire field (keyed by wire field name).
// Mirrors operationBodyFields' schema walk but records TYPES, so the params
// struct fields can be typed from the schema (required → value, optional → *T).
func operationBodyFieldTypes(sd *specDoc, op opInfo) (map[string]string, map[string]bool, error) {
	schemas, err := componentsSchemas(sd)
	if err != nil {
		return nil, nil, err
	}
	body := mapChild(sd.rawOp(op), "requestBody")
	if body == nil {
		return nil, nil, nil
	}
	content := mapChild(body, "content")
	if content == nil || content.Kind != yaml.MappingNode || len(content.Content) < 2 {
		return nil, nil, nil
	}
	sch := mapChild(content.Content[1], "schema")
	if sch == nil {
		return nil, nil, nil
	}
	return schemaFieldTypes(schemas, sch)
}

// schemaFieldTypes walks an object (or allOf/union-flattened) schema and returns
// wire-field → Go type + the required set. It follows $ref / allOf / anyOf / oneOf
// so the type map covers every field schemaFields lists. Required semantics match
// the reference's flattenSchema/flattenUnion: an `allOf` MERGES its members'
// required (a field required by any allOf member is required — allOf is an AND),
// while an `anyOf`/`oneOf` INTERSECTS (a field is required only if EVERY variant
// requires it — otherwise a variant that omits it would send an unset value). This
// mirrors the oracle exactly (e.g. calling.dial's anyOf: `from`/`to` required,
// `url`/`swml` optional), so an optional command/body field stays a nilable *T and
// is only sent when set — preserving the wire body.
func schemaFieldTypes(schemas, node *yaml.Node) (map[string]string, map[string]bool, error) {
	types := map[string]string{}
	seen := map[string]bool{}
	var walk func(n *yaml.Node) map[string]bool
	// walk returns the required set contributed by n (merged for allOf, intersected
	// for anyOf/oneOf) and records field types along the way (first-seen wins).
	walk = func(n *yaml.Node) map[string]bool {
		n = resolveSchema(schemas, n)
		if n == nil {
			return map[string]bool{}
		}
		req := map[string]bool{}
		// allOf: AND — merge each member's required into this node's required.
		if lst := seqChild(n, "allOf"); lst != nil {
			for _, br := range lst.Content {
				for k := range walk(br) {
					req[k] = true
				}
			}
		}
		// anyOf/oneOf: OR — a field is required only if required by EVERY branch.
		for _, comb := range []string{"anyOf", "oneOf"} {
			if lst := seqChild(n, comb); lst != nil && len(lst.Content) > 0 {
				var inter map[string]bool
				for i, br := range lst.Content {
					br := walk(br)
					if i == 0 {
						inter = br
						continue
					}
					next := map[string]bool{}
					for k := range inter {
						if br[k] {
							next[k] = true
						}
					}
					inter = next
				}
				for k := range inter {
					req[k] = true
				}
			}
		}
		// This node's own required list.
		if r := seqChild(n, "required"); r != nil {
			for _, x := range r.Content {
				req[x.Value] = true
			}
		}
		// This node's own properties (record types first-seen).
		if props := mapChild(n, "properties"); props != nil && props.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(props.Content); i += 2 {
				name := props.Content[i].Value
				if !seen[name] {
					seen[name] = true
					types[name] = goFieldType(schemas, props.Content[i+1])
				}
			}
		}
		return req
	}
	required := walk(node)
	return types, required, nil
}

// paramFieldType returns the Go type for a params-struct field given its schema
// Go type (from operationBodyFieldTypes) and whether it is required. Required →
// value type; optional → pointer *T (nilable reference types stay unwrapped).
// A field whose type could not be resolved ("") falls back to `any`.
func paramFieldType(goType string, required bool) string {
	if goType == "" {
		return "any"
	}
	if required {
		return goType
	}
	return optionalGoType(goType)
}

// isNilableGoType reports whether a Go type expression can be compared to nil
// (pointer, slice, map, or the empty interface). Value types (string/int/…,
// defined-string enum types) cannot.
func isNilableGoType(t string) bool {
	return strings.HasPrefix(t, "*") || strings.HasPrefix(t, "[]") ||
		strings.HasPrefix(t, "map[") || t == "any"
}

// commandFieldTypes returns wire-field → Go type + required set for a command-
// dispatch command's `params` sub-schema (union-flattened via schemaFieldTypes),
// mirroring commandFields' field walk. Used to type the command params struct.
func commandFieldTypes(sd *specDoc, requestSchema string) (map[string]string, map[string]bool, error) {
	schemas, err := componentsSchemas(sd)
	if err != nil {
		return nil, nil, err
	}
	sch := mapChild(schemas, requestSchema)
	if sch == nil {
		return nil, nil, fmt.Errorf("command request schema %q not in components.schemas", requestSchema)
	}
	sch = resolveSchema(schemas, sch)
	params := mapChild(mapChild(sch, "properties"), "params")
	if params == nil {
		return map[string]string{}, map[string]bool{}, nil
	}
	return schemaFieldTypes(schemas, params)
}
