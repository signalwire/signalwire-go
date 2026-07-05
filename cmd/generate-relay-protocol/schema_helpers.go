// Schema-node + Go-type-emission helpers for cmd/generate-relay-protocol.
//
// These are the exact minimal subset of the REST types emitter
// (cmd/generate-rest: main.go YAML helpers + types.go type emission) that the
// RELAY protocol emitter needs. They are duplicated here — rather than shared
// through an internal package — because the REST generator uses them pervasively
// across its own internal call sites; keeping this command self-contained lets
// the two generators evolve independently while both reproduce the SAME
// object-vs-alias struct emission (so pkg/relay/protocol_types_generated.go is
// byte-identical whether emitted by the old consolidated generate-rest or by
// this split command). Verbatim copies; the drift/GEN-FRESH gates prove parity.
package main

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Ordered YAML-node view (from cmd/generate-rest/main.go).
// ---------------------------------------------------------------------------

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

func scalarChild(node *yaml.Node, key string) string {
	c := mapChild(node, key)
	if c == nil {
		return ""
	}
	return c.Value
}

func rootOf(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}
	return doc
}

// ---------------------------------------------------------------------------
// Identifiers (from cmd/generate-rest/main.go + types.go).
// ---------------------------------------------------------------------------

var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true, "interface": true,
	"map": true, "package": true, "range": true, "return": true, "select": true,
	"struct": true, "switch": true, "type": true, "var": true,
}

func escapeIdent(s string) string {
	if goKeywords[s] {
		return s + "_"
	}
	return s
}

func pascal(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' || r == '.' })
	var b strings.Builder
	for _, w := range parts {
		if w == "" {
			continue
		}
		b.WriteString(strings.ToUpper(w[:1]) + w[1:])
	}
	return b.String()
}

func goParamName(field string) string {
	parts := strings.Split(field, "_")
	var b strings.Builder
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			b.WriteString(p)
		} else {
			b.WriteString(strings.ToUpper(p[:1]) + p[1:])
		}
	}
	s := b.String()
	if s == "" {
		s = "field"
	}
	return escapeIdent(s)
}

func structFieldName(field string) string {
	s := goParamName(field)
	r := []rune(s)
	if len(r) > 0 && r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 32
	}
	return string(r)
}

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
	if s[0] >= '0' && s[0] <= '9' {
		s = "Schema_" + s
	}
	return s
}

func refLeaf(ref string) string {
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		return ref[i+1:]
	}
	return ref
}

func refLeafGoName(ref string) string {
	return typeGoName(refLeaf(ref))
}

// ---------------------------------------------------------------------------
// Schema helpers (from cmd/generate-rest/types.go).
// ---------------------------------------------------------------------------

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

func goFieldType(schemas, node *yaml.Node) string {
	if node == nil {
		return "any"
	}
	if ref := scalarChild(node, "$ref"); ref != "" {
		if !strings.HasPrefix(ref, "#/") && strings.HasSuffix(ref, ".json") {
			return "map[string]any"
		}
		return refLeafGoName(ref)
	}
	if mapChild(node, "const") != nil {
		return "string"
	}
	if mapChild(node, "enum") != nil {
		t, _ := schemaType(node)
		return goScalar(t)
	}
	if allOf := seqChild(node, "allOf"); allOf != nil {
		if len(allOf.Content) == 1 {
			return goFieldType(schemas, allOf.Content[0])
		}
		return "map[string]any"
	}
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
		return "map[string]any"
	default:
		return "any"
	}
}

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

func isObjectSchema(node *yaml.Node) bool {
	if seqChild(node, "oneOf") != nil || seqChild(node, "anyOf") != nil || seqChild(node, "allOf") != nil {
		return false
	}
	t, _ := schemaType(node)
	props := mapChild(node, "properties")
	return (t == "object" || (t == "" && props != nil)) &&
		props != nil && props.Kind == yaml.MappingNode && len(props.Content) > 0
}

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
		if !required[wireKey] {
			typ = optionalGoType(typ)
		}
		fmt.Fprintf(b, "\t%s %s `json:%q`\n", fieldName, typ, wireKey+",omitempty")
	}
	b.WriteString("}\n\n")
}

func optionalGoType(typ string) string {
	if strings.HasPrefix(typ, "[]") || strings.HasPrefix(typ, "map[") || typ == "any" {
		return typ
	}
	if strings.HasPrefix(typ, "*") {
		return typ
	}
	return "*" + typ
}
