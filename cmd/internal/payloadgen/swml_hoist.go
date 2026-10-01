package payloadgen

import (
	"fmt"
	"regexp"
	"strings"
)

// The schema.json transforms the reference SWML emitter applies before emitting
// (generate_python_rest_types.py: drop_deprecated_swml_verbs, hoist_inline_objects).
// Ported here so the Go surface names the same types the reference does.

// swmlVerbIsDeprecated reports whether a SWML verb wrapper (a SWMLMethod.anyOf
// member) is marked `deprecated: true` — on the wrapper itself or on its verb
// property. Keyed on the schema's annotation only, never on a verb-name list.
func swmlVerbIsDeprecated(wrapper *schema) bool {
	if wrapper == nil {
		return false
	}
	if wrapper.Deprecated {
		return true
	}
	for _, p := range wrapper.Properties {
		if p.sch != nil && p.sch.Deprecated {
			return true
		}
	}
	return false
}

// dropDeprecatedSwmlVerbs removes the deprecated verbs (owner ruling 2026-09-24:
// dial/eval/if stay deprecated and are not exposed): each deprecated wrapper leaves
// the SWMLMethod union and its wrapper $def is not emitted. Returns the new defs +
// order and the dropped verb names.
func dropDeprecatedSwmlVerbs(defs map[string]*schema, order []string) (map[string]*schema, []string, []string) {
	sm := defs["SWMLMethod"]
	if sm == nil {
		return defs, order, nil
	}
	var kept []*schema
	dropped := map[string]bool{}
	var droppedVerbs []string
	for _, arm := range sm.AnyOf {
		wrapper := refNameRaw(arm.Ref)
		if w := defs[wrapper]; w != nil && swmlVerbIsDeprecated(w) {
			dropped[wrapper] = true
			if len(w.Properties) > 0 {
				droppedVerbs = append(droppedVerbs, w.Properties[0].name)
			}
			continue
		}
		kept = append(kept, arm)
	}
	if len(dropped) == 0 {
		return defs, order, nil
	}
	out := map[string]*schema{}
	var outOrder []string
	for _, name := range order {
		if dropped[name] {
			continue
		}
		out[name] = defs[name]
		outOrder = append(outOrder, name)
	}
	nsm := *sm
	nsm.AnyOf = kept
	out["SWMLMethod"] = &nsm
	return out, outOrder, droppedVerbs
}

// presenceOnly reports whether every arm constrains only WHICH keys are present —
// `required` clauses combined by anyOf/oneOf/allOf, and nothing else (schema.json's
// SWML_CHECK_ONLY_ONE_OF / AT_LEAST_ONE_OF rules). Such a combinator adds no key and
// no type, so it must not stop the object from being typed.
func presenceOnly(arms []*schema) bool {
	if len(arms) == 0 {
		return false
	}
	for _, arm := range arms {
		if arm == nil || len(arm.keys) == 0 {
			return false
		}
		for k := range arm.keys {
			switch k {
			case "required", "anyOf", "oneOf", "allOf":
			default:
				return false
			}
		}
		if arm.keys["anyOf"] && !presenceOnly(arm.AnyOf) {
			return false
		}
		if arm.keys["oneOf"] && !presenceOnly(arm.OneOf) {
			return false
		}
		if arm.keys["allOf"] && !presenceOnly(arm.AllOf) {
			return false
		}
	}
	return true
}

// withoutPresenceAllOf returns s minus a presence-only allOf; otherwise s.
func withoutPresenceAllOf(s *schema) *schema {
	if s == nil || !presenceOnly(s.AllOf) {
		return s
	}
	c := *s
	c.AllOf = nil
	c.keys = copyKeys(s.keys)
	delete(c.keys, "allOf")
	return &c
}

func copyKeys(k map[string]bool) map[string]bool {
	out := make(map[string]bool, len(k))
	for kk, v := range k {
		out[kk] = v
	}
	return out
}

// isInlineObject reports an inline object schema with named properties (no $ref,
// no combinator once a presence-only allOf is set aside).
func isInlineObject(s *schema) bool {
	s = withoutPresenceAllOf(s)
	if s == nil || s.Ref != "" || len(s.Properties) == 0 {
		return false
	}
	if s.Type != nil {
		if t, ok := s.Type.(string); !ok || t != "object" {
			return false
		}
	}
	return len(s.AnyOf) == 0 && len(s.OneOf) == 0 && len(s.AllOf) == 0
}

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]+`)

// hoistInlineObjects lifts every inline property-bearing object into its own named
// $def and points a $ref at it, so each becomes a named struct instead of an open
// map. Names are derived from the schema path exactly as the reference does:
//   - a verb wrapper's verb object is <Verb>Config, its descendants <Verb>…;
//   - any other object is <Parent><Key>; an array element adds Item (prefixItems
//     Item<i>), an additionalProperties value adds Value;
//   - in a union with ONE object arm that arm takes the union's name; with several,
//     each takes <Name><ArmTitle> (or <Name>Variant<i> without a title).
//
// A name already declared gets a numeric suffix (2, 3, …). Returns the new defs and
// order (originals first, hoisted after, in walk order).
func hoistInlineObjects(defs map[string]*schema, order []string, verbRoots map[string]string) (map[string]*schema, []string) {
	taken := map[string]bool{}
	for name := range defs {
		taken[name] = true
	}
	hoisted := map[string]*schema{}
	var hoistedOrder []string
	nameOf := map[*schema]string{}

	claim := func(name string) string {
		cand, n := name, 2
		for taken[cand] {
			cand = fmt.Sprintf("%s%d", name, n)
			n++
		}
		taken[cand] = true
		return cand
	}

	var visit func(node *schema, name, prefix string) *schema
	walkChildren := func(node *schema, prefix string) *schema {
		out := *node
		if len(node.Properties) > 0 {
			props := make([]propEntry, len(node.Properties))
			for i, p := range node.Properties {
				props[i] = propEntry{name: p.name, sch: visit(p.sch, prefix+pascal(p.name), prefix+pascal(p.name))}
			}
			out.Properties = props
		}
		if node.Items != nil {
			out.Items = visit(node.Items, prefix+"Item", prefix+"Item")
		}
		if len(node.PrefixItems) > 0 {
			pi := make([]*schema, len(node.PrefixItems))
			for i, p := range node.PrefixItems {
				nm := fmt.Sprintf("%sItem%d", prefix, i+1)
				pi[i] = visit(p, nm, nm)
			}
			out.PrefixItems = pi
		}
		if ap, ok := node.AdditionalProperties.(map[string]any); ok {
			v := visit(parseSchemaFromMap(ap), prefix+"Value", prefix+"Value")
			out.AdditionalProperties = v
		}
		arms := func(list []*schema) []*schema {
			if list == nil {
				return nil
			}
			nObj := 0
			for _, a := range list {
				if isInlineObject(a) {
					nObj++
				}
			}
			res := make([]*schema, len(list))
			for i, arm := range list {
				if nObj > 1 && isInlineObject(arm) {
					suffix := pascal(nonAlnum.ReplaceAllString(arm.Title, " "))
					if suffix == "" {
						suffix = fmt.Sprintf("Variant%d", i+1)
					}
					armName := prefix + suffix
					res[i] = visit(arm, armName, armName)
				} else {
					res[i] = visit(arm, nameOf[node], prefix)
				}
			}
			return res
		}
		out.AnyOf = arms(node.AnyOf)
		out.OneOf = arms(node.OneOf)
		out.AllOf = arms(node.AllOf)
		return &out
	}
	visit = func(node *schema, name, prefix string) *schema {
		if node == nil {
			return nil
		}
		if isInlineObject(node) {
			final := claim(name)
			childPrefix := prefix
			if prefix == name {
				childPrefix = final
			}
			hoistedOrder = append(hoistedOrder, final) // reserve walk order before descending
			hoisted[final] = walkChildren(withoutPresenceAllOf(node), childPrefix)
			return &schema{
				Ref:         "#/$defs/" + final,
				Description: node.Description,
				Title:       node.Title,
				Deprecated:  node.Deprecated,
				XAPIState:   node.XAPIState,
				keys:        map[string]bool{"$ref": true},
			}
		}
		nameOf[node] = name
		return walkChildren(node, prefix)
	}

	out := map[string]*schema{}
	for _, defName := range order {
		sch := defs[defName]
		if sch == nil {
			out[defName] = sch
			continue
		}
		if verb, ok := verbRoots[defName]; ok {
			c := *sch
			props := make([]propEntry, len(sch.Properties))
			copy(props, sch.Properties)
			for i, p := range props {
				if p.name == verb {
					base := pascal(verb)
					props[i] = propEntry{name: p.name, sch: visit(p.sch, base+"Config", base)}
				}
			}
			c.Properties = props
			out[defName] = &c
			continue
		}
		nameOf[sch] = defName
		out[defName] = walkChildren(sch, defName)
	}
	outOrder := append([]string{}, order...)
	for _, name := range hoistedOrder {
		out[name] = hoisted[name]
		outOrder = append(outOrder, name)
	}
	return out, outOrder
}

// swaigEnvelopeTypes are the SWAIG response ENVELOPE types, declared ONCE by the
// SWAIG action file (pkg/swaig, swaig-response.yaml). schema.json carries the same
// two shapes as $defs (a DataMap / Expression `output` is a SwaigResponse
// template), so the SWML verb file would otherwise declare a SECOND SwaigAction /
// SwaigResponse — one wire object, two public types. The SWML file references the
// swaig package's instead and skips the objects hoisted out of them.
var swaigEnvelopeTypes = []string{"SwaigAction", "SwaigResponse"}

// isSwaigEnvelopeOwned reports whether a $def is a SWAIG envelope type or an
// object hoisted out of one (its name continues the envelope name with an
// uppercase letter).
func isSwaigEnvelopeOwned(name string, present map[string]bool) bool {
	for _, n := range swaigEnvelopeTypes {
		if !present[n] {
			continue
		}
		if name == n {
			return true
		}
		if strings.HasPrefix(name, n) && len(name) > len(n) {
			c := name[len(n)]
			if c >= 'A' && c <= 'Z' {
				return true
			}
		}
	}
	return false
}

// verbConfigRef returns the $ref of a verb body's config: the body's own $ref, or
// the one hoisted object arm of an anyOf body. "" when there is no single config.
func verbConfigRef(defs map[string]*schema, inner *schema) string {
	if inner == nil {
		return ""
	}
	if inner.Ref != "" {
		return inner.Ref
	}
	if len(inner.AnyOf) == 0 {
		return ""
	}
	var refs []string
	for _, a := range inner.AnyOf {
		if a != nil && a.Ref != "" && isInlineObject(defs[refNameRaw(a.Ref)]) {
			refs = append(refs, a.Ref)
		}
	}
	if len(refs) == 1 {
		return refs[0]
	}
	return ""
}
