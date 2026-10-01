package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Cross-spec type-name collisions.
//
// The whole REST types surface lives in ONE Go package, so a components/schemas
// name declared by two specs is emitted once. That is right when both specs carry
// the SAME schema (the shared error types, the SWML-schema types fabric and
// calling both embed). It is WRONG when the two schemas differ: the later spec's
// methods would decode into the earlier spec's struct (video's
// VideoStreams.Get returned calling's SWML `Stream` verb struct; fabric's SIP
// endpoint methods returned relay-rest's SipEndpoint). The reference has one
// module per spec, so it never conflates them.
//
// computeTypeRenames finds every such pair and gives the LATER spec's type a
// spec-qualified Go name (<Ns><Name>: VideoStream, FabricSipEndpoint), which that
// spec's own declarations and $refs then use. The enumerators fold the qualified
// name back to the reference leaf (GeneratedTypeRenames), so parity compares the
// same class the reference declares.

// curRenames is the rename map of the spec currently being emitted (rawGoName ->
// qualified Go name). Package-level, like emittedTypeNames, so the emit helpers
// (refLeafGoName) reach it without threading the spec through every signature.
var curRenames map[string]string

// generatedTypeRenames records qualified Go name -> reference leaf for the
// enumerator fold table (internal/surface/gen_type_renames_generated.go).
var generatedTypeRenames = map[string]string{}

// localTypeName is typeGoName for a schema declared or referenced from the spec
// being emitted, applying that spec's collision renames.
func localTypeName(raw string) string {
	n := typeGoName(raw)
	if r, ok := curRenames[n]; ok {
		return r
	}
	return n
}

// specSchemas loads a spec document's components/schemas mapping node.
func specSchemas(rawPath string) (*yaml.Node, error) {
	raw, err := os.ReadFile(rawPath) //nolint:gosec // G304: developer-run codegen reading a spec path derived from $PORTING_SDK.
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return mapChild(mapChild(rootOf(&doc), "components"), "schemas"), nil
}

// canonSchema renders a schema node as a canonical string for structural
// comparison: prose keys dropped, mapping keys sorted, and every same-file $ref
// rewritten to the Go name it resolves to in its spec (so two textually equal
// schemas that reference differently-renamed types still compare unequal).
func canonSchema(n *yaml.Node, renames map[string]string) string {
	if n == nil {
		return "nil"
	}
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) > 0 {
			return canonSchema(n.Content[0], renames)
		}
		return ""
	case yaml.MappingNode:
		type kv struct{ k, v string }
		var kvs []kv
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := n.Content[i].Value
			switch k {
			case "description", "examples", "example", "title", "x-provenance":
				continue
			}
			v := n.Content[i+1]
			if k == "$ref" && strings.HasPrefix(v.Value, "#/") {
				name := typeGoName(refLeaf(v.Value))
				if r, ok := renames[name]; ok {
					name = r
				}
				kvs = append(kvs, kv{k, "ref:" + name})
				continue
			}
			kvs = append(kvs, kv{k, canonSchema(v, renames)})
		}
		sort.Slice(kvs, func(i, j int) bool { return kvs[i].k < kvs[j].k })
		var b strings.Builder
		b.WriteString("{")
		for _, e := range kvs {
			fmt.Fprintf(&b, "%q:%s,", e.k, e.v)
		}
		b.WriteString("}")
		return b.String()
	case yaml.SequenceNode:
		var b strings.Builder
		b.WriteString("[")
		for _, c := range n.Content {
			b.WriteString(canonSchema(c, renames))
			b.WriteString(",")
		}
		b.WriteString("]")
		return b.String()
	default:
		return fmt.Sprintf("%q", n.Value)
	}
}

// nsPrefix is the Go identifier prefix for a spec dir (relay-rest -> RelayRest).
func nsPrefix(ns string) string {
	var b strings.Builder
	for _, p := range strings.FieldsFunc(ns, func(r rune) bool { return r == '-' || r == '_' }) {
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return b.String()
}

// computeTypeRenames returns, per spec name, the collision renames (see the file
// comment). specOrder is the emission order (first spec keeps the bare name).
// Iterates to a fixpoint because renaming a type can make a schema that
// references it differ from its namesake in an earlier spec.
func computeTypeRenames(specOrder []string, rawPaths map[string]string) (map[string]map[string]string, error) {
	schemasBy := map[string]*yaml.Node{}
	for _, ns := range specOrder {
		s, err := specSchemas(rawPaths[ns])
		if err != nil {
			return nil, err
		}
		schemasBy[ns] = s
	}
	renames := map[string]map[string]string{}
	for _, ns := range specOrder {
		renames[ns] = map[string]string{}
	}
	for range 8 {
		changed := false
		claimed := map[string]string{} // Go name -> canonical schema of its declarer
		for _, ns := range specOrder {
			s := schemasBy[ns]
			if s == nil || s.Kind != yaml.MappingNode {
				continue
			}
			for i := 0; i+1 < len(s.Content); i += 2 {
				name := typeGoName(s.Content[i].Value)
				canon := canonSchema(s.Content[i+1], renames[ns])
				if _, renamed := renames[ns][name]; renamed {
					continue
				}
				prev, ok := claimed[name]
				if !ok {
					claimed[name] = canon
					continue
				}
				if prev != canon {
					renames[ns][name] = nsPrefix(ns) + name
					changed = true
				}
			}
		}
		if !changed {
			return renames, nil
		}
	}
	return nil, fmt.Errorf("computeTypeRenames: no fixpoint after 8 passes")
}

// emitTypeRenamesTable renders internal/surface/gen_type_renames_generated.go.
func emitTypeRenamesTable() string {
	var b strings.Builder
	b.WriteString(`// Code generated by cmd/generate-rest; DO NOT EDIT.
//
// AUTO-GENERATED from porting-sdk/rest-apis/ — regenerate with:
//   go run ./cmd/generate-rest
//
// GeneratedTypeRenames maps a spec-qualified generated REST type name (a type
// whose components/schemas name collides with a DIFFERENT schema of the same
// name in an earlier spec, so it cannot share the one Go package's bare name) to
// the reference leaf it stands for. The enumerators fold the qualified name back
// to that leaf, so the class compares with the reference's per-spec module class.

package surface

// GeneratedTypeRenames: qualified Go type name -> reference leaf.
var GeneratedTypeRenames = map[string]string{
`)
	keys := make([]string, 0, len(generatedTypeRenames))
	for k := range generatedTypeRenames {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "\t%q: %q,\n", k, generatedTypeRenames[k])
	}
	b.WriteString("}\n")
	return b.String()
}
