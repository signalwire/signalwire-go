// Command enumerate-signatures emits port_signatures.json — the
// canonical, signature-level cousin of port_surface.json. Same shape
// as porting-sdk/python_signatures.json (surface_schema_v2.json),
// driven by the same StructTable / FreeFnTable / FactoryInit lookup
// tables shared with cmd/enumerate-surface.
//
// This is the Go half of Phase 3 of the cross-language signature audit
// (see porting-sdk/SIGNATURE_AUDIT_PLAN.md). The pipeline:
//
//  1. Walk pkg/**/*.go via go/ast, collect every public method's source-
//     level signature (param names, type expressions, return types).
//  2. For each Go struct in surface.StructTable, translate Go method
//     signatures onto the corresponding Python class+method using the
//     same name-translation logic as enumerate-surface.
//  3. Translate Go source-level type expressions to canonical types
//     (string, int, optional<T>, list<T>, dict<K,V>, callable<...>,
//     class:<dotted>, ...) via porting-sdk/type_aliases.yaml.
//  4. Emit port_signatures.json validated against
//     porting-sdk/surface_schema_v2.json.
//
// Type translation deliberately uses source-level names (no go/types
// resolution). The SDK uses standard Go imports throughout — no aliased
// imports of stdlib types — so source spellings are unambiguous. If a
// future code change introduces an aliased type, the adapter raises
// loud failure and the alias table or vocabulary gets extended.
//
// Usage:
//
//	go run ./cmd/enumerate-signatures             # write port_signatures.json
//	go run ./cmd/enumerate-signatures --strict    # fail on any unknown type
//	go run ./cmd/enumerate-signatures --stdout
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	surfacepkg "github.com/signalwire/signalwire-go/internal/surface"
)

var (
	structTable = surfacepkg.StructTable
	freeFnTable = surfacepkg.FreeFnTable
	factoryInit = surfacepkg.FactoryInit
)

// kwargsTailMethods lists the fully-qualified Python reference methods whose
// FINAL parameter is a `**kwargs`/`**params` var_keyword tail rather than a
// concrete positional argument. Go models such a tail as a trailing
// `params map[string]any` / `extra map[string]any` bag; the Python oracle now
// (porting-sdk #58) STRIPS the var_keyword tail from the extracted signature,
// and the cross-port signature checker only excuses a trailing tail the port
// still carries when it is `required: false`. So for these — and ONLY these —
// methods the enumerator reclassifies the trailing bag param to a var_keyword
// tail (required:false), matching the reference's `**kwargs` idiom. This is a
// reconciliation table (the var_keyword-tail analog of a rename table), keyed by
// the reference QN so a genuine positional `params: dict` argument (e.g.
// AIConfigMixin.set_language_params(self, code, params: dict) — a REAL required
// positional, NOT **kwargs) is left untouched and keeps comparing EQUAL. The
// generated-REST resource methods carry their own `sig.restResource` var_keyword
// handling below and are not listed here.
//
// Each entry was verified against the reference source: the Python def ends in
// `**params: Any` / `**kwargs: Any`.
var kwargsTailMethods = map[string]bool{
	"signalwire.core.mixins.ai_config_mixin.AIConfigMixin.set_prompt_llm_params":      true, // def set_prompt_llm_params(self, **params: Any)
	"signalwire.core.mixins.ai_config_mixin.AIConfigMixin.set_post_prompt_llm_params": true, // def set_post_prompt_llm_params(self, **params: Any)
	"signalwire.core.swml_handler.SWMLVerbHandler.build_config":                       true, // def build_config(self, **kwargs: Any)
	"signalwire.core.swml_builder.SWMLBuilder.ai":                                     true, // def ai(self, ..., swaig=None, **kwargs)
	"signalwire.rest._base.CrudWithAddresses.list_addresses":                          true, // def list_addresses(self, resource_id, **params: Any)
}

// optionalTailVariadicMethods lists the fully-qualified Python reference methods
// whose FINAL parameter is a single OPTIONAL scalar (`behavior: str | None = None`)
// that Go idiomatically models as a trailing variadic `...string`. The RELAY
// pause controls (PlayAction/RecordAction/CollectAction, projected from the
// PausableAction mixin — porting-sdk @5744580) take `pause(behavior: str | None =
// None)`; Go spells "an optional single behavior" as `Pause(behavior ...string)`
// (call with 0 or 1 arg). The raw enumerator would translate `...string` to
// `list<string>` required:true, which mismatches the reference's `optional<string>`
// required:false. For THESE — and only these — methods, reclassify the trailing
// variadic to `optional<string>` required:false so the port compares EQUAL. This
// is an idiom reconciliation table (the optional-scalar analog of the var_keyword
// tail table above), keyed by reference QN; a genuine required `[]string` /
// multi-arg variadic elsewhere is untouched. Verified against the reference:
// signalwire/relay/call.py PausableAction.pause(self, behavior: str | None = None).
var optionalTailVariadicMethods = map[string]bool{
	"signalwire.relay.call.PlayAction.pause":    true,
	"signalwire.relay.call.RecordAction.pause":  true,
	"signalwire.relay.call.CollectAction.pause": true,
}

// paramsStructField is one field of a generated-REST params struct (§5/§4a).
type paramsStructField struct {
	name    string // exported Go field name (e.g. "QueryString", "Extras")
	typeStr string // source-level type expression (e.g. "any", "map[string]any")
}

// paramsStructFields maps a generated-REST `<...>Params` struct's SHORT type name
// to its ordered fields. Populated while parsing pkg/rest/namespaces/
// *_resources_generated.go. The signature enumerator UNFOLDS these fields back
// into the flat keyword param set the Python oracle records, so collapsing the old
// flat-positional operation/command params into a named Go options struct is a pure
// call-site reshape and keeps port_signatures.json byte-identical (drift 0).
var paramsStructFields = map[string][]paramsStructField{}

// genTypeModule maps a generated REST wire-type name (declared in a
// pkg/rest/namespaces/*_types_generated.go file) to its canonical Python module
// `signalwire.rest.namespaces.<ns>_types_generated`. Populated while parsing
// those files. translateType consults it so a field/return referencing a
// generated type (including the LOWERCASE scalar-format aliases docid/uuid/jwt,
// which the leading-uppercase class-ref fallback would otherwise reject) resolves
// to `class:signalwire.rest.namespaces.<ns>_types_generated.<Name>`. The shared
// diff tool folds that to `gen:<Name>` and compares by leaf, matching the
// reference's per-namespace `<ns>_types_generated.<Name>` exactly. A name shared
// across specs (deduped to one Go decl) keeps whichever ns declared it — the leaf
// fold makes the module path immaterial to the comparison.
var genTypeModule = map[string]string{}

// scalarAliasLeaf folds an EXPORTED generated scalar-format alias type name back to
// the reference's lowercase canonical leaf. The REST generator exports every type
// (uuid→Uuid, docid→Docid, jwt→Jwt, play_url→Play_url) so a public struct field
// doesn't leak a private type; the Python oracle records these scalar-format aliases
// under their lowercase names (relay_rest_types_generated.uuid, datasphere…docid,
// fabric…jwt). Folding the leaf here keeps the class ref parity-identical to the
// oracle while the Go source stays idiomatic (exported). A name not in this map is
// returned unchanged.
var scalarAliasLeaf = map[string]string{
	"Uuid":     "uuid",
	"Docid":    "docid",
	"Jwt":      "jwt",
	"Play_url": "play_url",
}

// genLeaf returns the canonical leaf name for a generated type name (folding the
// exported scalar-format aliases back to their lowercase oracle leaf).
func genLeaf(t string) string {
	if leaf, ok := scalarAliasLeaf[t]; ok {
		return leaf
	}
	return t
}

// ---------------------------------------------------------------------------
// AST walking — collects signatures, not just names
// ---------------------------------------------------------------------------

type goParam struct {
	name    string // canonical Go name (already snake-style? No, Go uses camelCase; we'll snake_case at translation time)
	typeStr string // source-level type expression
}

type goSignature struct {
	pkg     string
	name    string
	params  []goParam
	returns string // source-level type expression of the canonical return; "" → void
	isField bool   // true when this signature was synthesized from a struct field, not a method
	// restResource marks a method on a generated REST resource class
	// (pkg/rest/namespaces/*_resources_generated.go). Its named params-struct
	// (`params <Recv><Method>Params`) is UNFOLDED back into the Python reference's
	// flat keyword set (see toCanonicalSignature + paramsStructFields).
	restResource bool
}

type goFunc = goSignature // free function

type goStructFacts struct {
	pkg     string
	name    string
	methods map[string]*goSignature
	// embeds holds the SHORT type names of the struct's anonymous (embedded)
	// fields whose declared methods are PROMOTED onto this struct — e.g. a
	// generated REST resource embeds `*CrudResource` / `*CrudWithAddresses`,
	// promoting their Create/Update/Get/List/Delete. When a StructTable-listed
	// goMethod is not declared directly on the struct, it is resolved through
	// this embed chain and the promoted method's SIGNATURE is used for the
	// projection, attributed to the subclass. Only the short type name is
	// stored; the embed chain lives in the same package (namespaces).
	embeds []string
}

// ---------------------------------------------------------------------------
// Generated-payload (SWAIG + SWML-verb) interface-field emission (D3)
//
// The generated READ-side payload structs (cmd/generate-payloads output) are NOT
// in the StructTable — they carry no ergonomic method surface, only typed wire
// FIELDS. Without special handling the drift gate can't SEE them and reports
// every field the Python/TS reference records as `missing-port`. So, SCOPED to
// the generated-payload files only (by filename), we emit each struct's exported
// FIELDS as zero-arg members — matching the reference `_is_sdk_class_type` rule:
// only a CLASS-typed field (a `$ref` to another generated payload struct, a list
// of one, or a union carrying one) is part of the cross-port surface; a
// primitive/plain-dict field is Python-internal scaffolding the reference skips.
//
// Each generated field carries a `gen:"<canonical-audit-type>"` struct tag (Go
// has no union type, so a `union<int,class:SWMLVar>` field is `any` at runtime
// but its exact audit shape lives in the tag). We read that tag verbatim as the
// member's canonical return type — the tag is the single source of truth, so the
// (lossy) Go static type never has to be re-derived here.
//
// The MODULE names below end in the `_generated` markers the shared diff tool
// folds to the stable `gen-payload` token (diff_port_signatures.py
// _GEN_PAYLOAD_MODULE_MARKERS), so a payload class keys as `gen-payload.<Class>.
// <field>` cross-port regardless of which file/package a port groups it in.
// ---------------------------------------------------------------------------

// genPayloadModule maps a generated-payload file's base name to the canonical
// module it is recorded under. Only files listed here are interface-walked (the
// scope restriction — no other struct leaks into the payload oracle).
var genPayloadModule = map[string]string{
	"swaig_request_generated.go": "signalwire.core.swaig_request_generated",
	"post_prompt_generated.go":   "signalwire.core.post_prompt_generated",
	"swaig_actions_generated.go": "signalwire.core.swaig_actions_generated",
	"swml_verbs_generated.go":    "signalwire.core.swml_verbs_generated",
}

// genPayloadFacts collects the class-typed members of the generated-payload
// structs, keyed by canonical module -> class -> member -> canonical return.
type genPayloadFacts struct {
	// members[module][class][member] = canonical return type (from the gen: tag)
	members map[string]map[string]map[string]string
}

func newGenPayloadFacts() *genPayloadFacts {
	return &genPayloadFacts{members: map[string]map[string]map[string]string{}}
}

func (g *genPayloadFacts) add(module, class, member, ret string) {
	if g.members[module] == nil {
		g.members[module] = map[string]map[string]string{}
	}
	if g.members[module][class] == nil {
		g.members[module][class] = map[string]string{}
	}
	g.members[module][class][member] = ret
}

// genTagRe extracts the `gen:"..."` value from a struct-field tag literal.
var genTagRe = regexp.MustCompile(`gen:"([^"]*)"`)

// jsonTagRe extracts the wire key from the `json:"key,..."` tag — the member is
// keyed by the WIRE name (snake_case), matching how the reference records the
// TypedDict field (its member name IS the wire key).
var jsonTagRe = regexp.MustCompile(`json:"([^",]*)`)

// isGenClassType applies the reference `_is_sdk_class_type` rule to a canonical
// type string: emit the field only when it carries a class ref.
func isGenClassType(canon string) bool {
	return strings.Contains(canon, "class:")
}

func walk(root string) (map[string]*goStructFacts, map[string]*goFunc, *genPayloadFacts, error) {
	structs := map[string]*goStructFacts{}
	funcs := map[string]*goFunc{}
	payloads := newGenPayloadFacts()

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			n := info.Name()
			if strings.HasPrefix(n, ".") || n == "vendor" || n == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		return parseFile(path, structs, funcs, payloads)
	})
	return structs, funcs, payloads, err
}

// collectGenPayload walks a generated-payload file's structs and records each
// exported, class-typed field as a member (keyed by wire name).
func collectGenPayload(file *ast.File, module string, payloads *genPayloadFacts) {
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || !ast.IsExported(ts.Name.Name) {
				continue
			}
			st, isStruct := ts.Type.(*ast.StructType)
			if !isStruct || st.Fields == nil {
				continue
			}
			class := ts.Name.Name
			for _, f := range st.Fields.List {
				if len(f.Names) == 0 || f.Tag == nil {
					continue
				}
				tag := f.Tag.Value
				gm := genTagRe.FindStringSubmatch(tag)
				if gm == nil {
					continue
				}
				canon := gm[1]
				// Only class-typed fields are part of the cross-port surface
				// (the reference _is_sdk_class_type rule).
				if !isGenClassType(canon) {
					continue
				}
				member := ""
				if jm := jsonTagRe.FindStringSubmatch(tag); jm != nil {
					member = jm[1]
				}
				if member == "" {
					continue
				}
				payloads.add(module, class, member, canon)
			}
		}
	}
}

func parseFile(path string, structs map[string]*goStructFacts, funcs map[string]*goFunc, payloads *genPayloadFacts) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	// Generated-payload files are interface-walked separately (D3) and NOT fed
	// into the StructTable-driven method projection (they carry no method
	// surface, only typed wire fields).
	if module, ok := genPayloadModule[filepath.Base(path)]; ok {
		collectGenPayload(file, module, payloads)
		return nil
	}
	pkgName := file.Name.Name
	// Generated REST resource files carry the exploded-typed operation +
	// command-dispatch methods (§5). Their body-field params are keyword-only in
	// the Python reference; mark them so buildSignature captures which params are
	// exploded body fields and toCanonicalSignature reclassifies their kinds.
	base := filepath.Base(path)
	isRestResource := strings.HasSuffix(base, "_resources_generated.go") &&
		strings.Contains(filepath.ToSlash(path), "pkg/rest/namespaces/")
	// Generated REST wire-type files (<ns>_types_generated.go): record every
	// declared type name → its canonical <ns>_types_generated module, so a
	// field/return referencing it resolves to the folded class ref (see
	// genTypeModule). The <ns> is the file base with the suffix stripped.
	if strings.HasSuffix(base, "_types_generated.go") &&
		strings.Contains(filepath.ToSlash(path), "pkg/rest/namespaces/") {
		ns := strings.TrimSuffix(base, "_types_generated.go")
		module := "signalwire.rest.namespaces." + ns + "_types_generated"
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				// Record every generated type name (first-declaring ns wins for a
				// cross-spec-deduped name; the leaf fold makes the module immaterial).
				if _, seen := genTypeModule[ts.Name.Name]; !seen {
					genTypeModule[ts.Name.Name] = module
				}
			}
		}
		return nil
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ast.IsExported(ts.Name.Name) {
					continue
				}
				st, isStruct := ts.Type.(*ast.StructType)
				if !isStruct {
					continue
				}
				// Record generated-REST params structs' fields (§5/§4a) so the
				// signature enumerator can UNFOLD `params <...>Params` back into the
				// flat keyword set the oracle records (drift-neutral). Scoped to
				// `*Params` structs in the generated resource files.
				if isRestResource && strings.HasSuffix(ts.Name.Name, "Params") && st.Fields != nil {
					var fields []paramsStructField
					for _, f := range st.Fields.List {
						typeStr := exprString(f.Type)
						for _, n := range f.Names {
							fields = append(fields, paramsStructField{name: n.Name, typeStr: typeStr})
						}
					}
					paramsStructFields[ts.Name.Name] = fields
				}
				key := pkgName + "." + ts.Name.Name
				if _, present := structs[key]; !present {
					structs[key] = &goStructFacts{
						pkg:     pkgName,
						name:    ts.Name.Name,
						methods: map[string]*goSignature{},
					}
				}
				// Record anonymous (embedded) fields so promoted methods can be
				// resolved through the embed chain during projection.
				structs[key].embeds = append(structs[key].embeds, embeddedTypeNames(st)...)
				// Project exported struct fields (e.g. RestClient.Calling
				// *namespaces.CallingNamespace) as zero-arg accessor
				// methods so the cross-language audit sees them. Matches
				// the Python reference adapter, which emits typed
				// instance attributes the same way.
				if st.Fields != nil {
					for _, f := range st.Fields.List {
						if len(f.Names) == 0 {
							continue
						}
						typeStr := exprString(f.Type)
						for _, n := range f.Names {
							if !ast.IsExported(n.Name) {
								continue
							}
							if _, exists := structs[key].methods[n.Name]; exists {
								continue
							}
							structs[key].methods[n.Name] = &goSignature{
								pkg:     pkgName,
								name:    n.Name,
								params:  []goParam{},
								returns: typeStr,
								isField: true,
							}
						}
					}
				}
			}
		case *ast.FuncDecl:
			if !ast.IsExported(d.Name.Name) {
				continue
			}
			sig := buildSignature(pkgName, d)
			if isRestResource {
				sig.restResource = true
			}
			if d.Recv == nil || len(d.Recv.List) == 0 {
				funcs[pkgName+"."+d.Name.Name] = sig
				continue
			}
			recv := recvTypeName(d.Recv.List[0].Type)
			if recv == "" || !ast.IsExported(recv) {
				continue
			}
			key := pkgName + "." + recv
			if _, present := structs[key]; !present {
				structs[key] = &goStructFacts{
					pkg:     pkgName,
					name:    recv,
					methods: map[string]*goSignature{},
				}
			}
			structs[key].methods[d.Name.Name] = sig
		}
	}
	return nil
}

func buildSignature(pkg string, fd *ast.FuncDecl) *goSignature {
	sig := &goSignature{pkg: pkg, name: fd.Name.Name, params: []goParam{}}
	if fd.Type.Params != nil {
		for _, field := range fd.Type.Params.List {
			typeStr := exprString(field.Type)
			if len(field.Names) == 0 {
				// Anonymous param: treat as positional with index name
				sig.params = append(sig.params, goParam{name: fmt.Sprintf("p%d", len(sig.params)), typeStr: typeStr})
				continue
			}
			for _, n := range field.Names {
				sig.params = append(sig.params, goParam{name: n.Name, typeStr: typeStr})
			}
		}
	}
	if fd.Type.Results != nil && len(fd.Type.Results.List) > 0 {
		// Flatten the result list to individual source-level type strings
		// (a single `f.Names` entry may still name one type).
		var rets []string
		for _, f := range fd.Type.Results.List {
			ts := exprString(f.Type)
			n := 1
			if len(f.Names) > 0 {
				n = len(f.Names)
			}
			for range n {
				rets = append(rets, ts)
			}
		}
		// A genuine multi-value return whose values are ALL non-error is a real
		// tuple (e.g. HandleRequest's (int, map[string]string, string) →
		// tuple<int,dict<string,string>,string>); emit it as a tuple so it
		// compares EQUAL to the reference's tuple return. Otherwise take the
		// first result — multi-return Go funcs typically pair a value with an
		// `error` (mapped to `any`, not part of the Python signature).
		if len(rets) > 1 && rets[len(rets)-1] != "error" {
			sig.returns = "tuple(" + strings.Join(rets, ",") + ")"
		} else {
			sig.returns = rets[0]
		}
	}
	return sig
}

// exprString renders an ast.Expr as the canonical Go source string.
func exprString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprString(t.Elt)
		}
		return "[" + exprString(t.Len) + "]" + exprString(t.Elt)
	case *ast.MapType:
		return "map[" + exprString(t.Key) + "]" + exprString(t.Value)
	case *ast.InterfaceType:
		if len(t.Methods.List) == 0 {
			return "interface{}"
		}
		return "interface{...}"
	case *ast.FuncType:
		return funcTypeString(t)
	case *ast.ChanType:
		return "chan " + exprString(t.Value)
	case *ast.Ellipsis:
		return "..." + exprString(t.Elt)
	case *ast.IndexExpr:
		return exprString(t.X) + "[" + exprString(t.Index) + "]"
	case *ast.IndexListExpr:
		parts := make([]string, len(t.Indices))
		for i, ix := range t.Indices {
			parts[i] = exprString(ix)
		}
		return exprString(t.X) + "[" + strings.Join(parts, ",") + "]"
	case *ast.BasicLit:
		return t.Value
	}
	return fmt.Sprintf("<unhandled:%T>", e)
}

func funcTypeString(t *ast.FuncType) string {
	var args []string
	if t.Params != nil {
		for _, f := range t.Params.List {
			ts := exprString(f.Type)
			n := 1
			if len(f.Names) > 0 {
				n = len(f.Names)
			}
			for range n {
				args = append(args, ts)
			}
		}
	}
	var results []string
	if t.Results != nil {
		for _, f := range t.Results.List {
			ts := exprString(f.Type)
			n := 1
			if len(f.Names) > 0 {
				n = len(f.Names)
			}
			for range n {
				results = append(results, ts)
			}
		}
	}
	return "func(" + strings.Join(args, ",") + ") (" + strings.Join(results, ",") + ")"
}

func recvTypeName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return recvTypeName(e.X)
	case *ast.IndexExpr:
		return recvTypeName(e.X)
	case *ast.IndexListExpr:
		return recvTypeName(e.X)
	}
	return ""
}

// embeddedTypeNames returns the SHORT type names of a struct's anonymous
// (embedded) fields — the fields with no explicit name. Only embeds whose type
// resolves to a bare/pointer identifier in the same package are recorded (e.g.
// `*CrudResource`, `CrudWithAddresses`); qualified selector embeds (pkg.Type)
// carry no promoted SDK method surface we project and are skipped.
func embeddedTypeNames(st *ast.StructType) []string {
	if st.Fields == nil {
		return nil
	}
	var out []string
	for _, f := range st.Fields.List {
		if len(f.Names) != 0 {
			continue // named field, not an embed
		}
		if name := recvTypeName(f.Type); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// resolvePromotedMethod returns the promoted method signature for goMethod if it
// is declared on one of facts' embedded base structs (transitively), else nil.
// Go promotes an embedded field's methods onto the embedder, so a StructTable
// entry that lists e.g. `Create` for a generated REST resource embedding
// `*CrudResource` is supplied by CrudResource's own `Create` signature. The
// embed chain is walked in the same package; cycles are guarded by a visited
// set.
func resolvePromotedMethod(structs map[string]*goStructFacts, facts *goStructFacts, goMethod string) *goSignature {
	visited := map[string]struct{}{}
	var search func(f *goStructFacts) *goSignature
	search = func(f *goStructFacts) *goSignature {
		for _, embed := range f.embeds {
			base, ok := structs[f.pkg+"."+embed]
			if !ok {
				continue
			}
			key := base.pkg + "." + base.name
			if _, seen := visited[key]; seen {
				continue
			}
			visited[key] = struct{}{}
			if sig, present := base.methods[goMethod]; present {
				return sig
			}
			if sig := search(base); sig != nil {
				return sig
			}
		}
		return nil
	}
	return search(facts)
}

// ---------------------------------------------------------------------------
// Type translation
// ---------------------------------------------------------------------------

type translationFailure struct {
	context string
	reason  string
}

func loadAliases(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Aliases struct {
			Go map[string]string `yaml:"go"`
		} `yaml:"aliases"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return doc.Aliases.Go, nil
}

// goLocalAliases holds Go-specific named-type → canonical-type expansions that
// the shared porting-sdk/type_aliases.yaml does not carry (they name Go-only
// SDK types). Applied on top of the loaded aliases (loaded entries win).
var goLocalAliases = map[string]string{
	// swml.RoutingCallback = func(body, headers map[string]any) *string.
	"RoutingCallback":      "callable<list<dict<string,any>,dict<string,any>>,optional<string>>",
	"swml.RoutingCallback": "callable<list<dict<string,any>,dict<string,any>>,optional<string>>",
	// swaig.ToolHandler / swaig.TypedHandler are SWAIG tool-handler func types;
	// the reference types the create_typed_handler_wrapper func + return as a
	// bare callable. Expand to the canonical callable so they compare EQUAL.
	"ToolHandler":        "callable<list<any>,any>",
	"swaig.ToolHandler":  "callable<list<any>,any>",
	"TypedHandler":       "callable<list<any>,any>",
	"swaig.TypedHandler": "callable<list<any>,any>",
}

// closedSetUnions maps the Go defined-string closed-set types (and their
// bare/qualified spellings) to the canonical union<class:...,string> the
// audit vocabulary expects. The string member absorbs against the reference's
// plain `str`, so typing these params adds zero signature drift while giving
// Go callers typed constants. See the Tier-1 block in translateType.
var closedSetUnions = map[string]string{
	"skills.SkillName":      "union<class:signalwire.skills.SkillName,string>",
	"SkillName":             "union<class:signalwire.skills.SkillName,string>",
	"swaig.RecordFormat":    "union<class:signalwire.swaig.RecordFormat,string>",
	"RecordFormat":          "union<class:signalwire.swaig.RecordFormat,string>",
	"swaig.RecordDirection": "union<class:signalwire.swaig.RecordDirection,string>",
	"RecordDirection":       "union<class:signalwire.swaig.RecordDirection,string>",
	"swaig.TapDirection":    "union<class:signalwire.swaig.TapDirection,string>",
	"TapDirection":          "union<class:signalwire.swaig.TapDirection,string>",
	"swaig.Codec":           "union<class:signalwire.swaig.Codec,string>",
	"Codec":                 "union<class:signalwire.swaig.Codec,string>",
	"relay.TTSGender":       "union<class:signalwire.relay.TTSGender,string>",
	"TTSGender":             "union<class:signalwire.relay.TTSGender,string>",
	"logging.LogLevel":      "union<class:signalwire.logging.LogLevel,string>",
	"LogLevel":              "union<class:signalwire.logging.LogLevel,string>",
}

// translateType maps a source-level Go type expression to the canonical
// vocabulary. Returns ("", failure) when the type can't be translated;
// the caller decides whether to fail loudly or skip.
func translateType(t string, aliases map[string]string, ctx string) (string, *translationFailure) {
	t = strings.TrimSpace(t)
	if t == "" {
		return "void", nil
	}
	// Tier-1 closed-set defined string types. Each is a Go `type X string`
	// with typed constants used at a user-facing closed-set boundary
	// (skill name, record format, TTS gender, log-level name). The typed
	// param gives autocomplete + call-site typo checking, but Go auto-converts
	// untyped string-constant literals, so a bare "datetime" / "wav" / "female"
	// / "debug" still compiles — preserving parity with the reference's plain
	// `str`. Each is emitted as a union (the typed-name OR a string), mirroring
	// the PHP backed-enum proofs; the `string` member keeps drift 0 against the
	// reference's str. Both the qualified (pkg.Type) and bare (Type) spellings
	// are matched because the enumerator sees source-level expressions from
	// either the defining package or an importer.
	if canon, ok := closedSetUnions[t]; ok {
		return canon, nil
	}
	// context.Context is Go's idiomatic deadline/cancellation carrier — the
	// idiomatic Go expression of "this call may take an optional timeout" (the
	// PORT_PHILOSOPHY_GO ctx-cancelled-loop idiom). Where the Python reference
	// expresses the same capability it uses a `timeout: float = None` param, so
	// record a `ctx context.Context` param as the reference's canonical
	// `optional<float>` timeout slot — the idiom is reconciled in the recorded
	// surface (the param analog of the StructTable method-name mapping), not left
	// as an untyped `any` or papered over with an omission. Emitted OPTIONAL (a Go
	// ctx is always cancellable but never obligates a deadline, matching Python's
	// `timeout=None`), so on methods whose reference has no timeout it is absorbed
	// as an optional extra param (functional-parity tolerance), not a mismatch.
	if t == "context.Context" {
		return "optional<float>", nil
	}
	// Pointer: canonical interpretation is optional<T> for value types,
	// class:<T> for struct types. Without go/types we can't tell; default
	// to optional<...> and rely on alias table to resolve known names.
	if strings.HasPrefix(t, "*") {
		inner, fail := translateType(t[1:], aliases, ctx)
		if fail != nil {
			return "", fail
		}
		// For class references, drop the optional wrapper — Python
		// reference doesn't mark passed objects as optional.
		if strings.HasPrefix(inner, "class:") {
			return inner, nil
		}
		return "optional<" + inner + ">", nil
	}
	// Variadic: ...T → list<T>
	if strings.HasPrefix(t, "...") {
		inner, fail := translateType(t[3:], aliases, ctx)
		if fail != nil {
			return "", fail
		}
		return "list<" + inner + ">", nil
	}
	// Slice: []T → list<T>; []byte → bytes (handled by alias table)
	if strings.HasPrefix(t, "[]") {
		if t == "[]byte" {
			return "bytes", nil
		}
		inner, fail := translateType(t[2:], aliases, ctx)
		if fail != nil {
			return "", fail
		}
		return "list<" + inner + ">", nil
	}
	// Channel: chan T → not in canonical vocab; fail loud
	if strings.HasPrefix(t, "chan ") {
		return "", &translationFailure{context: ctx, reason: "chan types have no canonical equivalent: " + t}
	}
	// Map: map[K]V → dict<K,V>
	if strings.HasPrefix(t, "map[") {
		// Find matching closing bracket at depth 0
		depth := 0
		var split int
		for i := 4; i < len(t); i++ {
			ch := t[i]
			if ch == '[' {
				depth++
			} else if ch == ']' {
				if depth == 0 {
					split = i
					break
				}
				depth--
			}
		}
		if split == 0 {
			return "", &translationFailure{context: ctx, reason: "malformed map type: " + t}
		}
		k, fail := translateType(t[4:split], aliases, ctx)
		if fail != nil {
			return "", fail
		}
		v, fail := translateType(t[split+1:], aliases, ctx)
		if fail != nil {
			return "", fail
		}
		return "dict<" + k + "," + v + ">", nil
	}
	// Interface {} → any
	if t == "interface{}" || t == "any" {
		return "any", nil
	}
	if strings.HasPrefix(t, "interface{") {
		return "any", nil
	}
	// Multi-value return marker tuple(a,b,c) → tuple<a,b,c>. Emitted by
	// extractSignature for a genuine all-non-error multi-return (e.g.
	// HandleRequest's (int, dict<string,string>, string)).
	if strings.HasPrefix(t, "tuple(") && strings.HasSuffix(t, ")") {
		inner := t[len("tuple(") : len(t)-1]
		parts := splitTopLevelCommas(inner)
		canonParts := make([]string, 0, len(parts))
		for _, p := range parts {
			c, fail := translateType(strings.TrimSpace(p), aliases, ctx)
			if fail != nil {
				return "", fail
			}
			canonParts = append(canonParts, c)
		}
		return "tuple<" + strings.Join(canonParts, ",") + ">", nil
	}
	// Function type → callable<list<args>,ret>
	if strings.HasPrefix(t, "func(") {
		return translateFunc(t, aliases, ctx)
	}
	// Direct alias hit
	if v, ok := aliases[t]; ok {
		return v, nil
	}
	// Lowercase generated REST scalar-format alias (docid/uuid/jwt): resolve to the
	// folded gen-type class ref. Done BEFORE the generic/selector/uppercase paths
	// (those all require an uppercase leading rune, so a lowercase alias would
	// otherwise fall through to the unknown-type failure). A real SDK class never has
	// a lowercase leading rune, so this cannot hijack one; uppercase generated names
	// are resolved LAST (after StructTable) so a real SDK class of the same name wins.
	if module, ok := genTypeModule[t]; ok && !(len(t) > 0 && t[0] >= 'A' && t[0] <= 'Z') {
		return "class:" + module + "." + genLeaf(t), nil
	}
	// Generic instantiation: Foo[T,U] → translate Foo, drop type args
	// (Python reference doesn't carry generic instantiations in signatures)
	if i := strings.Index(t, "["); i > 0 && strings.HasSuffix(t, "]") {
		return translateType(t[:i], aliases, ctx)
	}
	// Selector expression: pkg.Name. Try alias for the full name first;
	// fall back to a class reference using the right-hand side.
	if dot := strings.LastIndex(t, "."); dot > 0 {
		// Whole thing in alias table?
		if v, ok := aliases[t]; ok {
			return v, nil
		}
		short := t[dot+1:]
		if v, ok := aliases[short]; ok {
			return v, nil
		}
		// Check if it's an SDK class — look up by short name in StructTable
		// (the same translation enumerate-surface uses)
		if classRef := lookupClassRef(t); classRef != "" {
			return classRef, nil
		}
		// No clear class but starts with uppercase — best-effort class ref
		if len(short) > 0 && short[0] >= 'A' && short[0] <= 'Z' {
			return "class:" + t, nil
		}
	}
	// Bare identifier — could be a struct in the same package.
	if len(t) > 0 && t[0] >= 'A' && t[0] <= 'Z' {
		// A real SDK class (StructTable) wins — a generated REST type name that
		// COLLIDES with an SDK class (e.g. the SWML-schema types AI/Cond/DataMap/
		// Section that the fabric/calling specs embed AND that the hand SWML/agent
		// surface also declares) must keep the SDK class ref when referenced from a
		// hand method; the generated struct is a distinct same-named wire type only
		// referenced from the (non-signature-enumerated) types module.
		if classRef := lookupClassRefByShort(t); classRef != "" {
			return classRef, nil
		}
		// Uppercase generated REST wire type not shadowed by any SDK class (e.g.
		// SearchResponse, CallResponse, SWMLObject, ChunkListResponse): fold to its
		// gen-type class ref so a resource method's typed return/param records the
		// real complex type (→ gen:<Name>), matching the oracle.
		if module, ok := genTypeModule[t]; ok {
			return "class:" + module + "." + genLeaf(t), nil
		}
		return "class:" + t, nil
	}
	return "", &translationFailure{context: ctx, reason: "unknown type: " + t}
}

func translateFunc(t string, aliases map[string]string, ctx string) (string, *translationFailure) {
	// Format produced by funcTypeString: "func(<args>) (<results>)"
	if !strings.HasPrefix(t, "func(") {
		return "", &translationFailure{context: ctx, reason: "not a func type: " + t}
	}
	rest := t[len("func("):]
	// Find matching closing paren for args
	depth := 1
	var argEnd int
	for i := range len(rest) {
		switch rest[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				argEnd = i
				goto found
			}
		}
	}
	return "", &translationFailure{context: ctx, reason: "unbalanced func args: " + t}
found:
	argList := rest[:argEnd]
	resultPart := strings.TrimSpace(rest[argEnd+1:])
	resultPart = strings.TrimPrefix(resultPart, "(")
	resultPart = strings.TrimSuffix(resultPart, ")")

	var canonArgs []string
	if argList != "" {
		for _, a := range splitTopLevelCommas(argList) {
			c, fail := translateType(strings.TrimSpace(a), aliases, ctx)
			if fail != nil {
				return "", fail
			}
			canonArgs = append(canonArgs, c)
		}
	}

	canonRet := "void"
	if resultPart != "" {
		// First result only (matches Method handling)
		results := splitTopLevelCommas(resultPart)
		if len(results) > 0 {
			c, fail := translateType(strings.TrimSpace(results[0]), aliases, ctx)
			if fail != nil {
				return "", fail
			}
			canonRet = c
		}
	}
	return "callable<list<" + strings.Join(canonArgs, ",") + ">," + canonRet + ">", nil
}

func splitTopLevelCommas(s string) []string {
	var out []string
	var buf strings.Builder
	depth := 0
	for _, ch := range s {
		switch ch {
		case '(', '[', '<':
			depth++
		case ')', ']', '>':
			depth--
		}
		if ch == ',' && depth == 0 {
			out = append(out, buf.String())
			buf.Reset()
			continue
		}
		buf.WriteRune(ch)
	}
	if buf.Len() > 0 {
		out = append(out, buf.String())
	}
	return out
}

// lookupClassRef tries to resolve a Go selector expression like
// “relay.Call“ or “agent.AgentBase“ to the canonical
// “class:signalwire.<...>.<Class>“ form using StructTable.
func lookupClassRef(sel string) string {
	if targets, ok := structTable[sel]; ok && len(targets) > 0 {
		return "class:" + targets[0].Module + "." + targets[0].Class
	}
	return ""
}

// lookupClassRefByShort searches StructTable for any entry whose name
// matches `short` (case-sensitive). Used when the source-level type is
// just `Foo` without the package qualifier (because it's in the same package).
func lookupClassRefByShort(short string) string {
	for k, targets := range structTable {
		if !strings.HasSuffix(k, "."+short) {
			continue
		}
		if len(targets) > 0 {
			return "class:" + targets[0].Module + "." + targets[0].Class
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Building the canonical inventory
// ---------------------------------------------------------------------------

type sigDoc struct {
	Version       string                        `json:"version"`
	GeneratedFrom string                        `json:"generated_from"`
	Modules       map[string]sigModuleInventory `json:"modules"`
}

type sigModuleInventory struct {
	Classes   map[string]sigClassEntry      `json:"classes,omitempty"`
	Functions map[string]canonicalSignature `json:"functions,omitempty"`
}

type sigClassEntry struct {
	Methods map[string]canonicalSignature `json:"methods"`
}

type canonicalSignature struct {
	Params  []canonicalParam `json:"params"`
	Returns string           `json:"returns"`
}

type canonicalParam struct {
	Name     string `json:"name"`
	Kind     string `json:"kind,omitempty"`
	Type     string `json:"type,omitempty"`
	Required *bool  `json:"required,omitempty"`
	// Default is a JSON value; keep as raw to preserve nulls.
	Default json.RawMessage `json:"default,omitempty"`
}

func boolPtr(b bool) *bool { return &b }

func goNameToSnake(s string) string {
	// Reuse enumerate-surface's pascal_to_snake by inlining the canonical
	// rule: insert _ between lowercase→uppercase and uppercase→Aa boundaries.
	var out strings.Builder
	for i, r := range s {
		if i > 0 {
			prev := rune(s[i-1])
			if (isUpper(r) && isLower(prev)) ||
				(isUpper(r) && i+1 < len(s) && isLower(rune(s[i+1])) && isUpper(prev)) {
				out.WriteRune('_')
			}
		}
		out.WriteRune(toLower(r))
	}
	return out.String()
}
func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func isLower(r rune) bool { return r >= 'a' && r <= 'z' }
func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}

// goFieldToPython converts a Go exported struct field name to its
// Python-canonical snake_case form, with corrections for SDK-specific
// abbreviations that don't snake-case naturally (e.g. “MFA“ -> “mfa“,
// “PubSub“ -> “pubsub“).
// isPrimitive returns true for Go primitive types (string, int, bool,
// etc.) — these should not be projected as SDK class accessor methods.
func isPrimitive(t string) bool {
	switch t {
	case "string", "bool", "byte", "rune", "error",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"uintptr", "float32", "float64",
		"complex64", "complex128", "any", "interface{}":
		return true
	}
	return false
}

func goFieldToPython(s string) string {
	switch s {
	case "MFA":
		return "mfa"
	case "PubSub":
		return "pubsub"
	case "FreeSwitchConnectors":
		return "freeswitch_connectors"
	case "SIPEndpoints":
		return "sip_endpoints"
	case "SIPGateways":
		return "sip_gateways"
	case "SWMLScripts":
		return "swml_scripts"
	case "SWMLWebhooks":
		return "swml_webhooks"
	case "CXMLScripts":
		return "cxml_scripts"
	case "CXMLApplications":
		return "cxml_applications"
	case "CXMLWebhooks":
		return "cxml_webhooks"
	}
	return goNameToSnake(s)
}

func toCanonicalSignature(sig *goSignature, aliases map[string]string, isMethod bool, isCtor bool, ctx string) (canonicalSignature, []translationFailure) {
	var failures []translationFailure
	params := []canonicalParam{}
	// Both regular methods and constructors take an implicit self in
	// the canonical Python shape.  Go factory functions (NewX) lift
	// into __init__ slots without a receiver, so we add self here so
	// param-count matches Python's reference signature.
	if isMethod || isCtor {
		params = append(params, canonicalParam{Name: "self", Kind: "self"})
	}
	for pi, p := range sig.params {
		// A reconciliation-table method's FINAL bag param is the Python
		// reference's `**kwargs`/`**params` var_keyword tail (stripped from the
		// oracle by porting-sdk #58). Emit it as var_keyword required:false so the
		// checker's drop-tail excusal applies — the Go trailing `params`/`extra`
		// map is the idiomatic spelling of the reference `**kwargs`. Scoped to the
		// LAST param and to a dict bag, so a leading positional dict is untouched.
		if pi == len(sig.params)-1 && kwargsTailMethods[ctx] &&
			(p.name == "params" || p.name == "extra" || p.name == "kwargs") &&
			strings.HasPrefix(p.typeStr, "map[string]") {
			params = append(params, canonicalParam{
				Name: p.name, Kind: "var_keyword", Type: "any",
				Required: boolPtr(false), Default: json.RawMessage("{}"),
			})
			continue
		}
		// A pause control's trailing variadic `...string` is the Go idiom for the
		// reference's optional scalar `behavior: str | None = None`. Reclassify it
		// to `optional<string>` required:false so it compares EQUAL (see
		// optionalTailVariadicMethods). Scoped to the LAST param and to a `...string`.
		if pi == len(sig.params)-1 && optionalTailVariadicMethods[ctx] &&
			p.typeStr == "...string" {
			params = append(params, canonicalParam{
				Name: goNameToSnake(p.name), Type: "optional<string>",
				Required: boolPtr(false),
			})
			continue
		}
		// §5/§4a: a generated-REST operation/command method takes its wire-body
		// fields as a named params STRUCT (`params <Recv><Method>Params`) instead of
		// flat positionals. UNFOLD that struct back into the flat keyword set the
		// Python oracle records so port_signatures.json is byte-identical to the old
		// flat form (pure call-site reshape → drift 0). Each non-Extras field →
		// keyword; the `Extras` field → keyword + a synthetic `**kwargs` var_keyword
		// tail (the exact shape the flat `extras map[string]any` param produced). A
		// GET query `params map[string]string` is NOT a params struct — it still
		// falls through to the `**params` var_keyword handling below.
		if sig.restResource {
			if fields, ok := paramsStructFields[p.typeStr]; ok {
				for _, f := range fields {
					fCanon, fail := translateType(f.typeStr, aliases, ctx+"["+f.name+"]")
					if fail != nil {
						failures = append(failures, *fail)
						continue
					}
					if f.name == "Extras" {
						params = append(params, canonicalParam{
							Name: "extras", Kind: "keyword", Type: fCanon,
							Required: boolPtr(true),
						})
						params = append(params, canonicalParam{
							Name: "kwargs", Kind: "var_keyword", Type: "any",
							Required: boolPtr(false), Default: json.RawMessage("{}"),
						})
						continue
					}
					params = append(params, canonicalParam{
						Name: goNameToSnake(f.name), Kind: "keyword", Type: fCanon,
						Required: boolPtr(true),
					})
				}
				continue
			}
		}
		canon, fail := translateType(p.typeStr, aliases, ctx+"["+p.name+"]")
		if fail != nil {
			failures = append(failures, *fail)
			continue
		}
		cp := canonicalParam{
			Name:     goNameToSnake(p.name),
			Type:     canon,
			Required: boolPtr(true), // Go has no defaults; every param is required
		}
		// §5: reclassify the remaining generated-REST params to the Python
		// reference's kinds. Leading path-id positionals stay positional; a GET
		// query `params` / set_methods `extra` object becomes a single `**params` /
		// `**extra` (var_keyword) tail. This makes the loose Go surface compare
		// COUNT + KIND clean against the closed Python reference.
		if sig.restResource && (p.name == "params" || p.name == "extra") {
			params = append(params, canonicalParam{
				Name: p.name, Kind: "var_keyword", Type: "any",
				Required: boolPtr(false), Default: json.RawMessage("{}"),
			})
			continue
		}
		params = append(params, cp)
	}
	returns := "void"
	if isCtor {
		returns = "void"
	} else {
		canon, fail := translateType(sig.returns, aliases, ctx+"[->]")
		if fail != nil {
			failures = append(failures, *fail)
		} else {
			returns = canon
		}
	}
	return canonicalSignature{Params: params, Returns: returns}, failures
}

func build(structs map[string]*goStructFacts, funcs map[string]*goFunc, payloads *genPayloadFacts, aliases map[string]string) (sigDoc, []translationFailure) {
	out := sigDoc{
		Version: "2",
		Modules: map[string]sigModuleInventory{},
	}
	var failures []translationFailure

	ensureModule := func(mod string) sigModuleInventory {
		if inv, ok := out.Modules[mod]; ok {
			return inv
		}
		return sigModuleInventory{
			Classes:   map[string]sigClassEntry{},
			Functions: map[string]canonicalSignature{},
		}
	}
	addClassMethod := func(mod, cls, method string, sig canonicalSignature) {
		inv := ensureModule(mod)
		if inv.Classes == nil {
			inv.Classes = map[string]sigClassEntry{}
		}
		entry, ok := inv.Classes[cls]
		if !ok {
			entry = sigClassEntry{Methods: map[string]canonicalSignature{}}
		}
		entry.Methods[method] = sig
		inv.Classes[cls] = entry
		out.Modules[mod] = inv
	}
	addFunction := func(mod, name string, sig canonicalSignature) {
		inv := ensureModule(mod)
		if inv.Functions == nil {
			inv.Functions = map[string]canonicalSignature{}
		}
		inv.Functions[name] = sig
		out.Modules[mod] = inv
	}

	// --- 1. Project struct methods onto Python classes ---
	for key, facts := range structs {
		targets, ok := structTable[key]
		if !ok {
			continue
		}
		for _, target := range targets {
			for goMethod, pyMethod := range target.Methods {
				if strings.HasPrefix(goMethod, "New") {
					if fn, present := funcs[facts.pkg+"."+goMethod]; present {
						sig, fails := toCanonicalSignature(fn, aliases, false, true, fmt.Sprintf("%s.%s.%s", target.Module, target.Class, pyMethod))
						failures = append(failures, fails...)
						addClassMethod(target.Module, target.Class, pyMethod, sig)
					}
					continue
				}
				mSig, present := facts.methods[goMethod]
				if !present {
					// Not declared directly — resolve through the embed chain
					// (promoted method). SCOPED to StructTable-listed methods:
					// the Methods map is the allowlist of what to project; the
					// embed resolution only SUPPLIES the promoted method's
					// signature. Arbitrary promoted methods not listed here are
					// never projected (no surface flood).
					mSig = resolvePromotedMethod(structs, facts, goMethod)
				}
				if mSig != nil {
					sig, fails := toCanonicalSignature(mSig, aliases, true, false, fmt.Sprintf("%s.%s.%s", target.Module, target.Class, pyMethod))
					failures = append(failures, fails...)
					addClassMethod(target.Module, target.Class, pyMethod, sig)
				}
			}
			// Synthetic methods: Python members the port expresses through a
			// package-level factory rather than a same-named Go method. The
			// ``from_payload`` classmethod on every relay event is the canonical
			// case — Go's ``New<Event>(params map[string]any)`` factory IS the
			// from_payload constructor (build the typed event from the raw
			// payload dict). We emit the reference-shaped classmethod signature
			// ``(cls, payload: dict<string,any>) -> class:<Module>.<Class>`` so
			// the signature audit sees the member the surface audit already
			// projects via ClassTarget.SyntheticMethods. Other synthetics
			// (``__init__``, ``from_json``, …) are covered by factoryInit /
			// FreeFnTable or documented in PORT_SIGNATURE_OMISSIONS.md.
			for _, syn := range target.SyntheticMethods {
				if syn != "from_payload" {
					continue
				}
				if _, already := target.Methods[syn]; already {
					continue
				}
				addClassMethod(target.Module, target.Class, "from_payload", canonicalSignature{
					Params: []canonicalParam{
						{Name: "cls", Kind: "cls"},
						{Name: "payload", Type: "dict<string,any>", Required: boolPtr(true)},
					},
					Returns: "class:" + target.Module + "." + target.Class,
				})
			}

			// Auto-emit exported fields whose type is an SDK class
			// (``*namespaces.FabricNamespace``, ``*FooClient``, etc.)
			// as zero-arg accessor methods. Mirrors the Python reference
			// adapter's instance-attribute projection for the same
			// composition pattern (RestClient.fabric, RestClient.calling).
			for goField, fSig := range facts.methods {
				if !fSig.isField {
					continue
				}
				if _, alreadyMapped := target.Methods[goField]; alreadyMapped {
					continue
				}
				// Only project fields whose return type is an SDK class
				// reference — primitive-typed state fields are filtered
				// (matches the Python adapter's _is_sdk_class_type rule).
				// Accept either ``namespaces.FabricNamespace`` (qualified)
				// or ``SubscribersResource`` (intra-package, identified by
				// leading uppercase).
				ret := strings.TrimPrefix(fSig.returns, "*")
				if ret == "" {
					continue
				}
				if !strings.Contains(ret, ".") && !(ret[0] >= 'A' && ret[0] <= 'Z') {
					continue
				}
				if isPrimitive(ret) {
					continue
				}
				pyName := goFieldToPython(goField)
				sig, fails := toCanonicalSignature(fSig, aliases, true, false, fmt.Sprintf("%s.%s.%s", target.Module, target.Class, pyName))
				failures = append(failures, fails...)
				addClassMethod(target.Module, target.Class, pyName, sig)
			}
		}
	}

	// --- 2. factoryInit: lift function as __init__ ---
	for goFn, spec := range factoryInit {
		fn, present := funcs[goFn]
		if !present {
			continue
		}
		targets, ok := structTable[spec.StructKey]
		if !ok {
			continue
		}
		for _, target := range targets {
			sig, fails := toCanonicalSignature(fn, aliases, false, true, fmt.Sprintf("%s.%s.__init__", target.Module, target.Class))
			failures = append(failures, fails...)
			addClassMethod(target.Module, target.Class, "__init__", sig)
		}
	}

	// --- 3. Free functions ---
	for key, fn := range funcs {
		if target, ok := freeFnTable[key]; ok {
			sig, fails := toCanonicalSignature(fn, aliases, false, false, fmt.Sprintf("%s.%s", target.Module, target.Name))
			failures = append(failures, fails...)
			addFunction(target.Module, target.Name, sig)
		}
	}

	// --- 4. Generated-payload interface fields (D3) ---
	// Each class-typed field of a generated payload struct is a zero-arg member
	// returning its canonical (gen: tag) type, under a module that folds to
	// gen-payload in the shared diff tool. This is what makes the SWAIG + SWML
	// read-side payloads (cmd/generate-payloads) visible to the drift gate.
	if payloads != nil {
		for module, classes := range payloads.members {
			for class, members := range classes {
				for member, ret := range members {
					addClassMethod(module, class, member, canonicalSignature{
						Params:  []canonicalParam{{Name: "self", Kind: "self"}},
						Returns: ret,
					})
				}
			}
		}
	}

	// Sort modules + classes + methods deterministically
	sortedMods := map[string]sigModuleInventory{}
	keys := make([]string, 0, len(out.Modules))
	for k := range out.Modules {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		inv := out.Modules[k]
		// Drop empty classes/functions maps so the JSON shape matches the
		// Python reference (omitempty handles this via tags but only at
		// the top level; we want methods sorted too).
		newInv := sigModuleInventory{}
		if len(inv.Classes) > 0 {
			newInv.Classes = inv.Classes
		}
		if len(inv.Functions) > 0 {
			newInv.Functions = inv.Functions
		}
		sortedMods[k] = newInv
	}
	out.Modules = sortedMods
	return out, failures
}

// ---------------------------------------------------------------------------
// Repo helpers
// ---------------------------------------------------------------------------

func goSHA(repo string) string {
	cmd := exec.Command("git", "-C", repo, "rev-parse", "HEAD")
	o, err := cmd.Output()
	if err != nil {
		return "N/A"
	}
	return strings.TrimSpace(string(o))
}

func findRepoRoot(start string) (string, error) {
	cwd := start
	for {
		if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			return cwd, nil
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			return "", fmt.Errorf("no go.mod found above %s", start)
		}
		cwd = parent
	}
}

// ---------------------------------------------------------------------------
// CLI
// ---------------------------------------------------------------------------

func run() error {
	var (
		outputPath  = flag.String("output", "port_signatures.json", "Write JSON to this path")
		aliasesPath = flag.String("aliases", "", "Path to porting-sdk/type_aliases.yaml (autodetected if empty)")
		strict      = flag.Bool("strict", false, "Exit non-zero on any translation failure")
		stdoutFlag  = flag.Bool("stdout", false, "Print to stdout")
	)
	flag.Parse()

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	repoRoot, err := findRepoRoot(cwd)
	if err != nil {
		return err
	}
	pkgRoot := filepath.Join(repoRoot, "pkg")

	aliasFile := *aliasesPath
	if aliasFile == "" {
		// Autodetect: try sibling porting-sdk
		candidates := []string{
			filepath.Join(repoRoot, "..", "porting-sdk", "type_aliases.yaml"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				aliasFile = c
				break
			}
		}
	}
	if aliasFile == "" {
		return fmt.Errorf("type_aliases.yaml not found; pass --aliases")
	}
	aliases, err := loadAliases(aliasFile)
	if err != nil {
		return fmt.Errorf("loadAliases: %w", err)
	}
	// Go-local named-type expansions the shared type_aliases.yaml doesn't carry.
	// swml.RoutingCallback is `func(body, headers map[string]any) *string`; the
	// reference types the routing callback_fn as
	// callable<list<dict<string,any>,dict<string,any>>,optional<string>>. Expand
	// the named type to that canonical callable so RegisterRoutingCallback's
	// callback_fn param compares EQUAL to the reference (idiom reconciled in the
	// alias table, not via an omission).
	for k, v := range goLocalAliases {
		if _, exists := aliases[k]; !exists {
			aliases[k] = v
		}
	}

	structs, funcs, payloads, err := walk(pkgRoot)
	if err != nil {
		return fmt.Errorf("walk: %w", err)
	}

	doc, failures := build(structs, funcs, payloads, aliases)
	doc.GeneratedFrom = fmt.Sprintf("signalwire-go @ %s (go/ast walker)", goSHA(repoRoot))

	if len(failures) > 0 {
		fmt.Fprintf(os.Stderr, "enumerate-signatures: %d translation failure(s)\n", len(failures))
		for i, f := range failures {
			if i >= 30 {
				fmt.Fprintf(os.Stderr, "  ... (%d more)\n", len(failures)-30)
				break
			}
			fmt.Fprintf(os.Stderr, "  - at %s: %s\n", f.context, f.reason)
		}
		if *strict {
			return fmt.Errorf("translation failures with --strict")
		}
	}

	rendered, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	rendered = append(rendered, '\n')

	if *stdoutFlag {
		_, err := os.Stdout.Write(rendered)
		return err
	}
	return os.WriteFile(*outputPath, rendered, 0o644)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
