// Package overlay loads the SDK-surface policy overlay
// (porting-sdk/rest-apis/x-sdk-overlay.yaml) — the single authoritative source,
// NOT wire truth, that says which spec fields the SDKs hide (drop from the
// surface entirely) or deprecate (emit but flag). It mirrors the reference
// implementation in porting-sdk/scripts/generate_python_rest_types.py.
//
// The specs (schema.json + rest-apis/*/openapi.yaml) stay pure wire truth; the
// hide/deprecate policy lives ONLY here so one place governs every emission site
// wherever a field surfaces.
//
// MATCHING: each rule has a field name and an OPTIONAL scope. The scope is
// matched against the field's containing SPEC SCHEMA NAME — the $defs/<name> key
// in schema.json and the components/schemas/<name> key in the openapi.yaml files
// — NOT the language-idiomatic Go type name a generator later emits. Callers must
// pass the spec schema name (the key they are iterating), not the emitted struct
// name. An unscoped rule matches everywhere; a scoped rule only in its schema.
package overlay

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// rule is one (field, scope) pair; scope == "" means "matches in every schema".
type rule struct {
	field string
	scope string
}

// Overlay is the parsed policy: the hidden + deprecated rule sets.
type Overlay struct {
	hidden     []rule
	deprecated []rule
}

type overlayFile struct {
	Hidden     []entry `yaml:"hidden"`
	Deprecated []entry `yaml:"deprecated"`
}

type entry struct {
	Field string `yaml:"field"`
	Scope string `yaml:"scope"`
}

func toRules(entries []entry) []rule {
	out := make([]rule, 0, len(entries))
	for _, e := range entries {
		if e.Field == "" {
			continue
		}
		out = append(out, rule{field: e.Field, scope: e.Scope})
	}
	return out
}

// Load reads x-sdk-overlay.yaml from the given porting-sdk root. A missing file
// yields an empty (no-op) overlay so generation still works without porting-sdk.
func Load(psdk string) (*Overlay, error) {
	path := filepath.Join(psdk, "rest-apis", "x-sdk-overlay.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Overlay{}, nil
		}
		return nil, err
	}
	var f overlayFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, err
	}
	return &Overlay{hidden: toRules(f.Hidden), deprecated: toRules(f.Deprecated)}, nil
}

func match(rules []rule, field, schemaName string) bool {
	for _, r := range rules {
		if r.field == field && (r.scope == "" || r.scope == schemaName) {
			return true
		}
	}
	return false
}

// Hidden reports whether the field in the given SPEC schema is dropped from the
// SDK surface entirely (still on the wire).
func (o *Overlay) Hidden(field, schemaName string) bool {
	if o == nil {
		return false
	}
	return match(o.hidden, field, schemaName)
}

// Deprecated reports whether the field in the given SPEC schema is emitted but
// flagged deprecated.
func (o *Overlay) Deprecated(field, schemaName string) bool {
	if o == nil {
		return false
	}
	return match(o.deprecated, field, schemaName)
}
