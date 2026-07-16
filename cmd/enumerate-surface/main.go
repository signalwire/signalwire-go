// Command enumerate-surface emits a JSON snapshot of the Go SDK's public API
// translated into Python-reference symbol names.
//
// The output (“port_surface.json“) is compared against
// “porting-sdk/python_surface.json“ by “diff_port_surface.py“ to detect
// unexcused drift.  Each Go struct is mapped onto a (python_module,
// python_class) pair and each Go method onto a python method name — so that
// “AgentBase.SetPromptText“ is emitted as
// “signalwire.core.mixins.prompt_mixin.PromptMixin.set_prompt_text“.  The
// same Go struct may contribute to multiple Python classes (“AgentBase“ is
// scattered across every mixin in the Python tree).
//
// Usage:
//
//	go run ./cmd/enumerate-surface            # writes port_surface.json
//	go run ./cmd/enumerate-surface --stdout   # print to stdout
//	go run ./cmd/enumerate-surface --check    # compare with existing file
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
	"sort"
	"strings"

	surfacepkg "github.com/signalwire/signalwire-go/v3/internal/surface"
)

// Re-export internal/surface package symbols under the previous local
// names so the existing main.go body keeps working without churn.
var (
	structTable = surfacepkg.StructTable
	freeFnTable = surfacepkg.FreeFnTable
	factoryInit = surfacepkg.FactoryInit
)

// genType records one generated REST wire-type surface class: the canonical
// `<ns>_types_generated` module + the (object) type name. The Python/TS
// references surface every OBJECT schema of a spec's components/schemas as a
// method-less class under `<ns>_types_generated`; the Go types modules
// (pkg/rest/namespaces/*_types_generated.go) carry the identical set, so they are
// emitted into the surface the same way (matched by leaf via the surface diff's
// `gen-type` fold for cross-module duplicates, or by full module path for a
// single-module type). Enum / scalar-alias / union-alias types are NOT surface
// classes (the reference records only interfaces/structs), matching Go: only
// `type X struct { … }` decls are collected here.
type genType struct {
	module string
	name   string
}

var genTypeSurface []genType

// sdkEnumSurfaceMarker is the doc-comment sentinel the types generator prepends to
// an x-sdk-enum-derived public enum type (cmd/generate-rest/types.go sdkEnumMarker).
// Its presence marks an enum type as surfaced public API (a surface class), while
// inline schema-enum defined-string types carry no marker and stay referenced-only.
const sdkEnumSurfaceMarker = "sdk-enum: surfaced public enum type."

// scanMarkedEnumTypes parses a hand-written namespaces file and returns each
// exported type whose doc comment carries the sdkEnumSurfaceMarker together with
// an explicit `module=<python-module>` (the hand-owned x-sdk-enum public enum).
// The module is read from the marker line so the surfaced class lands under the
// reference's `<ns>_types_generated` module even though the hand file has no ns
// in its name.
func scanMarkedEnumTypes(path string) []genType {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil
	}
	var out []genType
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE || gd.Doc == nil {
			continue
		}
		doc := gd.Doc.Text()
		if !strings.Contains(doc, sdkEnumSurfaceMarker) {
			continue
		}
		module := ""
		for _, field := range strings.Fields(doc) {
			if strings.HasPrefix(field, "module=") {
				module = strings.TrimPrefix(field, "module=")
				break
			}
		}
		if module == "" {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || !ast.IsExported(ts.Name.Name) {
				continue
			}
			out = append(out, genType{module: module, name: ts.Name.Name})
		}
	}
	return out
}

// --- AST walker -------------------------------------------------------------

// goStructFacts is the raw Go inventory for a single struct.
type goStructFacts struct {
	pkg     string
	name    string
	methods map[string]struct{}
	// embeds holds the SHORT type names of the struct's anonymous (embedded)
	// fields whose declared methods are promoted onto this struct — e.g. a
	// generated REST resource embeds `*CrudResource` / `*CrudWithAddresses`,
	// which promotes their Create/Update/Get/List/Delete. Recorded so that a
	// StructTable-listed goMethod not declared directly on the struct can be
	// RESOLVED through the embed chain (see resolvePromotedMethod). Only the
	// short type name is stored; the embed chain lives in the same package
	// (namespaces), so the base is looked up by `<pkg>.<embed>`.
	embeds []string
	// paramsPlumbing marks a generated-REST `<...>Params` options struct (§5/§4a):
	// call-shape plumbing for the named operation/command params, NOT oracle
	// surface. Excluded from port_additions_actual.json so it never shows up as a
	// SURFACE-DIFF addition (it's a pure call-site convenience type, generated
	// alongside the method, carrying no method surface of its own).
	paramsPlumbing bool
}

// walk parses every .go file under root and returns the collected inventory.
func walk(root string) (map[string]*goStructFacts, map[string]struct{}, error) {
	structs := map[string]*goStructFacts{}
	funcs := map[string]struct{}{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		return parseFile(path, structs, funcs)
	})
	return structs, funcs, err
}

// parseFile extracts exported struct types, exported methods and exported
// free functions from a single .go file.
func parseFile(path string, structs map[string]*goStructFacts, funcs map[string]struct{}) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	pkgName := file.Name.Name
	base := filepath.Base(path)
	// Hand-written namespaces types carrying the sdk-enum surface marker with an
	// explicit `module=...` (the one hand-owned x-sdk-enum public enum,
	// PhoneCallHandler in call_handler.go): surface the type under the named module.
	// The generator emits the OTHER x-sdk-enum types into their <ns>_types_generated
	// file (handled by the types-file branch); this covers the hand-owned one only.
	if pkgName == "namespaces" &&
		strings.Contains(filepath.ToSlash(path), "pkg/rest/namespaces/") &&
		!strings.HasSuffix(base, "_generated.go") && !strings.HasSuffix(base, "_test.go") {
		if names := scanMarkedEnumTypes(path); len(names) > 0 {
			genTypeSurface = append(genTypeSurface, names...)
		}
	}
	// Generated REST wire-type files (<ns>_types_generated.go): collect each OBJECT
	// struct as a surface class under `signalwire.rest.namespaces.<ns>_types_
	// generated` (see genTypeSurface). These are handled here and NOT fed into the
	// StructTable-driven projection (they carry no ergonomic method surface) nor the
	// port-additions inventory (they ARE canonical reference surface, emitted below).
	if strings.HasSuffix(base, "_types_generated.go") &&
		strings.Contains(filepath.ToSlash(path), "pkg/rest/namespaces/") {
		ns := strings.TrimSuffix(base, "_types_generated.go")
		module := "signalwire.rest.namespaces." + ns + "_types_generated"
		// Re-parse WITH comments so the x-sdk-enum surface marker (sdkEnumSurfaceMarker,
		// a doc comment the generator prepends to an exported public enum type) is
		// visible: those enum types ARE surface classes (the reference exports them as
		// public API), unlike the inline schema-enum defined-string types.
		cfset := token.NewFileSet()
		cfile, cerr := parser.ParseFile(cfset, path, nil, parser.ParseComments|parser.SkipObjectResolution)
		if cerr != nil {
			return fmt.Errorf("parse (comments) %s: %w", path, cerr)
		}
		for _, decl := range cfile.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ast.IsExported(ts.Name.Name) {
					continue
				}
				_, isStruct := ts.Type.(*ast.StructType)
				surfacedEnum := gd.Doc != nil && strings.Contains(gd.Doc.Text(), sdkEnumSurfaceMarker)
				// Surface classes are OBJECT structs and x-sdk-enum public enum types
				// (the reference surfaces interfaces + its exported enums; inline
				// schema enums / scalar / union aliases are referenced-only).
				if !isStruct && !surfacedEnum {
					continue
				}
				genTypeSurface = append(genTypeSurface, genType{module: module, name: ts.Name.Name})
			}
		}
		return nil
	}
	// Generated RELAY WS protocol types (pkg/relay/protocol_types_generated.go,
	// package relay): each OBJECT struct is a surface class under the reference's
	// `signalwire.relay.protocol_types_generated` module. The empty-object methods
	// (calling.call, signalwire.disconnect result) are `map[string]any` aliases, NOT
	// structs, so they are not surfaced — matching the reference (123 structs). Handled
	// here (and NOT fed into the StructTable projection nor the port-additions
	// inventory) exactly like the REST `_types_generated` files above.
	if base == "protocol_types_generated.go" && pkgName == "relay" &&
		strings.Contains(filepath.ToSlash(path), "pkg/relay/") {
		const module = "signalwire.relay.protocol_types_generated"
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
				if _, isStruct := ts.Type.(*ast.StructType); !isStruct {
					continue
				}
				genTypeSurface = append(genTypeSurface, genType{module: module, name: ts.Name.Name})
			}
		}
		return nil
	}

	// Generated CORE SWML/SWAIG typed-payload files (the D-workstream payloads).
	// cmd/generate-payloads emits these under pkg/swml / pkg/swaig with a
	// `<name>_generated.go` suffix (NOT the REST `_types_generated.go` convention),
	// carrying one Go struct per components/schemas entry of the SWML verb / SWAIG
	// request / post-prompt specs. The Python reference surfaces the identical set
	// under `signalwire.core.<name>_generated`. RECONCILE-IN-EMIT: record each OBJECT
	// struct as a method-less surface class under that canonical module (the surface
	// diff's gen-type fold reconciles leaves the reference duplicates across modules;
	// the module-unique *Config / Omit* / Pick* / PostPrompt* / *Action / SwaigRequest
	// types match under their own module). Emitted here (like the REST/RELAY generated
	// types) and NOT fed into the StructTable projection nor the port-additions
	// inventory (they ARE canonical reference surface).
	if genCoreModule := coreGeneratedModule(base, filepath.ToSlash(path)); genCoreModule != "" {
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
				if _, isStruct := ts.Type.(*ast.StructType); !isStruct {
					continue
				}
				genTypeSurface = append(genTypeSurface, genType{module: genCoreModule, name: ts.Name.Name})
			}
		}
		return nil
	}

	isRestResource := strings.HasSuffix(base, "_resources_generated.go") &&
		strings.Contains(filepath.ToSlash(path), "pkg/rest/namespaces/")
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
				key := pkgName + "." + ts.Name.Name
				if _, present := structs[key]; !present {
					structs[key] = &goStructFacts{
						pkg:     pkgName,
						name:    ts.Name.Name,
						methods: map[string]struct{}{},
					}
				}
				// §5/§4a: mark generated-REST params-struct plumbing so it is
				// excluded from the SURFACE-DIFF additions inventory.
				if isRestResource && strings.HasSuffix(ts.Name.Name, "Params") {
					structs[key].paramsPlumbing = true
				}
				// Record anonymous (embedded) fields so promoted methods can be
				// resolved through the embed chain during projection.
				structs[key].embeds = append(structs[key].embeds, embeddedTypeNames(st)...)
			}
		case *ast.FuncDecl:
			if !ast.IsExported(d.Name.Name) {
				continue
			}
			if d.Recv == nil || len(d.Recv.List) == 0 {
				funcs[pkgName+"."+d.Name.Name] = struct{}{}
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
					methods: map[string]struct{}{},
				}
			}
			structs[key].methods[d.Name.Name] = struct{}{}
		}
	}
	return nil
}

// coreGeneratedModule maps a generated core SWML/SWAIG typed-payload file (by
// base name + slash path) to its Python-canonical `signalwire.core.<name>_generated`
// module, or "" if the file is not one of them. These are emitted by
// cmd/generate-payloads under pkg/swml / pkg/swaig. swaig_actions_generated.go
// folds into the same swaig_actions_generated module (its PlaybackBgAction /
// TransferAction are the reference's swaig_actions_generated classes).
func coreGeneratedModule(base, slashPath string) string {
	switch {
	case base == "swml_verbs_generated.go" && strings.Contains(slashPath, "pkg/swml/"):
		return "signalwire.core.swml_verbs_generated"
	case base == "post_prompt_generated.go" && strings.Contains(slashPath, "pkg/swaig/"):
		return "signalwire.core.post_prompt_generated"
	case base == "swaig_request_generated.go" && strings.Contains(slashPath, "pkg/swaig/"):
		return "signalwire.core.swaig_request_generated"
	case base == "swaig_actions_generated.go" && strings.Contains(slashPath, "pkg/swaig/"):
		return "signalwire.core.swaig_actions_generated"
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

// recvTypeName extracts the base type name from a method receiver.
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

// --- Emission ---------------------------------------------------------------

// surface is the final JSON shape.  Matches “python_surface.json“.
type surface struct {
	Version       string                     `json:"version"`
	GeneratedFrom string                     `json:"generated_from"`
	Modules       map[string]moduleInventory `json:"modules"`
}

type moduleInventory struct {
	Classes   map[string][]string `json:"classes"`
	Functions []string            `json:"functions"`
}

// baseSkillProvides is the set of Go methods the embedded skills.BaseSkill
// supplies as defaults (pkg/skills/skill_base.go), promoted onto every concrete
// built-in skill struct. Used to accept a skill-contract method that the skill
// does not override but inherits (the qualified cross-package embed the walker
// cannot resolve automatically).
var baseSkillProvides = map[string]bool{
	"GetHints":           true,
	"Cleanup":            true,
	"GetParameterSchema": true,
	"GetInstanceKey":     true,
	"GetGlobalData":      true,
	"GetPromptSections":  true,
}

// skillLeafToGoMethod reverse-maps a Python-canonical skill-contract method leaf
// to the Go member that satisfies it (declared override or BaseSkill-promoted).
// These are the fixed SkillBase contract methods; the mapping is the inverse of
// goNameToSnake for the specific SDK-initialism-free names in play.
func skillLeafToGoMethod(leaf string) string {
	switch leaf {
	case "register_tools":
		return "RegisterTools"
	case "get_hints":
		return "GetHints"
	case "setup":
		return "Setup"
	case "cleanup":
		return "Cleanup"
	case "get_parameter_schema":
		return "GetParameterSchema"
	case "get_instance_key":
		return "GetInstanceKey"
	case "get_global_data":
		return "GetGlobalData"
	case "get_prompt_sections":
		return "GetPromptSections"
	}
	panic(fmt.Sprintf("enumerate-surface: no Go member mapping for skill contract leaf %q", leaf))
}

// promotedMethodExists reports whether goMethod is declared on one of facts'
// embedded base structs (transitively). Go promotes an embedded field's methods
// onto the embedder, so a StructTable entry that lists e.g. `Create` for a
// generated REST resource which embeds `*CrudResource` is satisfied by
// CrudResource's own `Create`. The embed chain is walked in the same package
// (the Crud bases live alongside the resources in namespaces); cycles are
// guarded by a visited set.
func promotedMethodExists(structs map[string]*goStructFacts, facts *goStructFacts, goMethod string) bool {
	visited := map[string]struct{}{}
	var search func(f *goStructFacts) bool
	search = func(f *goStructFacts) bool {
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
			if _, present := base.methods[goMethod]; present {
				return true
			}
			if search(base) {
				return true
			}
		}
		return false
	}
	return search(facts)
}

// build turns (goStructs, goFuncs) into a Python-reference surface driven by
// the translation tables.
func build(structs map[string]*goStructFacts, funcs map[string]struct{}) surface {
	out := surface{
		Version: "1",
		Modules: map[string]moduleInventory{},
	}
	ensure := func(mod string) moduleInventory {
		if inv, ok := out.Modules[mod]; ok {
			return inv
		}
		inv := moduleInventory{
			Classes:   map[string][]string{},
			Functions: []string{},
		}
		out.Modules[mod] = inv
		return inv
	}
	addMethod := func(mod, cls, method string) {
		inv := ensure(mod)
		if _, present := inv.Classes[cls]; !present {
			inv.Classes[cls] = []string{}
		}
		for _, m := range inv.Classes[cls] {
			if m == method {
				return
			}
		}
		inv.Classes[cls] = append(inv.Classes[cls], method)
		out.Modules[mod] = inv
	}
	addClass := func(mod, cls string) {
		inv := ensure(mod)
		if _, present := inv.Classes[cls]; !present {
			inv.Classes[cls] = []string{}
			out.Modules[mod] = inv
		}
	}
	addFunction := func(mod, name string) {
		inv := ensure(mod)
		for _, f := range inv.Functions {
			if f == name {
				return
			}
		}
		inv.Functions = append(inv.Functions, name)
		out.Modules[mod] = inv
	}

	// --- 1. Project Go structs onto Python classes ------------------------
	for key, facts := range structs {
		targets, ok := structTable[key]
		if !ok {
			// Port-only struct. Recorded in port_additions_actual.json
			// for cross-checking against PORT_ADDITIONS.md (see
			// computePortAdditions). Don't emit into the canonical
			// surface here — that path is for Python-mapped symbols.
			_ = facts
			continue
		}
		for _, target := range targets {
			addClass(target.Module, target.Class)
			for goMethod, pyMethod := range target.Methods {
				if strings.HasPrefix(goMethod, "New") {
					// Factory constructor lives as a free function;
					// emit only if the matching Go ``New<X>`` exists.
					if _, present := funcs[facts.pkg+"."+goMethod]; present {
						addMethod(target.Module, target.Class, pyMethod)
					}
					continue
				}
				if _, present := facts.methods[goMethod]; present {
					addMethod(target.Module, target.Class, pyMethod)
					continue
				}
				// Not declared directly on the struct — resolve through the
				// embed chain (promoted method). SCOPED to StructTable-listed
				// methods: the Methods map is the allowlist of what to project;
				// the embed resolution only SUPPLIES the fact that a promoted
				// method exists. Arbitrary promoted methods not listed here are
				// never projected (no surface flood).
				if promotedMethodExists(structs, facts, goMethod) {
					addMethod(target.Module, target.Class, pyMethod)
				}
			}
			for _, synthetic := range target.SyntheticMethods {
				addMethod(target.Module, target.Class, synthetic)
			}
			_ = target.Alias // already added via addClass above.
		}
	}

	// --- 2. Honour factoryInit (non-New<Struct> constructors) -------------
	for goFn, spec := range factoryInit {
		if _, present := funcs[goFn]; !present {
			continue
		}
		targets, ok := structTable[spec.StructKey]
		if !ok {
			continue
		}
		for _, target := range targets {
			addMethod(target.Module, target.Class, "__init__")
		}
	}

	// --- 3. Project Go free functions onto Python module-level functions --
	for key := range funcs {
		if target, ok := freeFnTable[key]; ok {
			addFunction(target.Module, target.Name)
		}
	}

	// --- 3a. Built-in skill contract projection ---------------------------
	// Each Go built-in *Skill struct (pkg/skills/builtin/*.go) embeds
	// skills.BaseSkill and overrides a subset of the SkillBase contract; the
	// rest is promoted from BaseSkill. So the concrete struct genuinely PROVIDES
	// every method the Python reference records for it. Project each onto its
	// Python-canonical `signalwire.skills.<name>.skill.<Class>` with the
	// reference's exact per-skill method set (RECONCILE-IN-EMIT — symbol PRESENT,
	// compares EQUAL — not omitted). Verify each mapped method is actually
	// present on the struct (declared or promoted) so a renamed/removed skill
	// member fails loud instead of emitting a phantom.
	for _, sc := range surfacepkg.SkillContractTable {
		facts, ok := structs[sc.GoStruct]
		if !ok {
			panic(fmt.Sprintf("enumerate-surface: skill struct %q in SkillContractTable not found in walk", sc.GoStruct))
		}
		addClass(sc.Module, sc.ClassName)
		for _, leaf := range sc.Methods {
			goMethod := skillLeafToGoMethod(leaf)
			// The method is satisfied either by a direct override on the skill
			// struct or by the embedded skills.BaseSkill default. BaseSkill lives
			// in a DIFFERENT package (`skills`) via a QUALIFIED embed
			// (`skills.BaseSkill`), which the same-package embed walker cannot
			// resolve — so a promoted BaseSkill contract method is accepted via
			// the known BaseSkill-provided set (verified against
			// pkg/skills/skill_base.go). A non-BaseSkill leaf that isn't declared
			// on the struct fails loud.
			if _, declared := facts.methods[goMethod]; !declared && !baseSkillProvides[goMethod] {
				panic(fmt.Sprintf("enumerate-surface: skill %s expects Go method %q (for %q) but it is neither declared nor a BaseSkill default", sc.GoStruct, goMethod, leaf))
			}
			addMethod(sc.Module, sc.ClassName, leaf)
		}
		for _, syn := range sc.Synthetic {
			addMethod(sc.Module, sc.ClassName, syn)
		}
	}

	// --- 3b. Generated REST wire types (<ns>_types_generated) -------------
	// Each collected object struct is a method-less surface class under its
	// `<ns>_types_generated` module (matching the Python/TS reference, which
	// surfaces every object schema of a spec). The surface diff folds a leaf the
	// reference duplicates across modules to `gen-type.<Leaf>` and keeps a single-
	// module type under its own module — both compare clean against this emission.
	for _, gt := range genTypeSurface {
		addClass(gt.module, gt.name)
	}

	// --- 4. Normalise output ----------------------------------------------
	for mod, inv := range out.Modules {
		for cls, methods := range inv.Classes {
			sort.Strings(methods)
			inv.Classes[cls] = methods
		}
		sort.Strings(inv.Functions)
		out.Modules[mod] = inv
	}
	return out
}

// PortAdditions is the JSON shape written to port_additions_actual.json.
// Each entry records a Go-only public symbol that wasn't projected into the
// Python-canonical surface (because it has no entry in StructTable /
// FreeFnTable). diff_port_surface.py reads this file alongside
// PORT_ADDITIONS.md and fails CI when an entry isn't documented there.
type PortAdditions struct {
	Version   string   `json:"version"`
	Generated string   `json:"generated_from"`
	Structs   []string `json:"structs"`
	Functions []string `json:"functions"`
}

// computePortAdditions walks the parsed Go inventory, keeps only the
// genuinely-public exports that have no entry in the translation tables,
// and emits the list in canonical order. Methods on unmapped structs are
// implicitly covered by listing the struct itself. Factory “New<Struct>“
// constructors paired with a mapped struct are already projected as
// __init__ and not listed here.
func computePortAdditions(structs map[string]*goStructFacts, funcs map[string]struct{}, repo string) PortAdditions {
	var addStructs []string
	for key, facts := range structs {
		if _, ok := structTable[key]; ok {
			continue
		}
		// §5/§4a: generated-REST params structs are call-shape plumbing, not oracle
		// surface — never list them as SURFACE-DIFF additions.
		if facts.paramsPlumbing {
			continue
		}
		addStructs = append(addStructs, key)
	}
	sort.Strings(addStructs)

	var addFuncs []string
	for key := range funcs {
		if _, ok := freeFnTable[key]; ok {
			continue
		}
		if _, ok := factoryInit[key]; ok {
			continue
		}
		dot := strings.Index(key, ".")
		if dot > 0 && strings.HasPrefix(key[dot+1:], "New") {
			pkgPart := key[:dot]
			structName := key[dot+4:] // strip "<pkg>.New"
			if _, ok := structTable[pkgPart+"."+structName]; ok {
				continue
			}
		}
		addFuncs = append(addFuncs, key)
	}
	sort.Strings(addFuncs)
	return PortAdditions{
		Version:   "1",
		Generated: fmt.Sprintf("signalwire-go @ %s", goSHA(repo)),
		Structs:   addStructs,
		Functions: addFuncs,
	}
}

// buildGoSurface turns (goStructs, goFuncs) into a surface file keyed on the
// **native** Go struct + method names.  Unlike “build“ — which translates
// everything onto the Python reference's dotted path — this captures the
// exact identifiers a Go doc or example would use (“AgentBase.DefineTool“,
// “RestClient“, “RunAgent“).  Used by “audit_docs.py“ on the Go port
// so that method-call references resolve against the actual surface.
//
// Shape matches “port_surface.json“ but the module name is the short Go
// package, the class is the exported struct, and methods are the exported
// Go method names.
func buildGoSurface(structs map[string]*goStructFacts, funcs map[string]struct{}) surface {
	out := surface{
		Version: "1",
		Modules: map[string]moduleInventory{},
	}
	ensure := func(mod string) moduleInventory {
		if inv, ok := out.Modules[mod]; ok {
			return inv
		}
		inv := moduleInventory{
			Classes:   map[string][]string{},
			Functions: []string{},
		}
		out.Modules[mod] = inv
		return inv
	}
	// Every exported struct becomes a class; every exported method becomes
	// a member.  Unexported or port-only symbols are included — ``audit_docs.py``
	// only cares that *some* reference resolves, not that the inventory
	// matches a reference layout.
	for key, facts := range structs {
		_ = key
		inv := ensure(facts.pkg)
		methods, present := inv.Classes[facts.name]
		if !present || methods == nil {
			methods = []string{}
		}
		for m := range facts.methods {
			methods = append(methods, m)
		}
		sort.Strings(methods)
		inv.Classes[facts.name] = methods
		out.Modules[facts.pkg] = inv
	}
	// Every exported free function becomes a module-level function.
	for key := range funcs {
		parts := strings.SplitN(key, ".", 2)
		if len(parts) != 2 {
			continue
		}
		pkg, name := parts[0], parts[1]
		inv := ensure(pkg)
		// de-dup
		present := false
		for _, existing := range inv.Functions {
			if existing == name {
				present = true
				break
			}
		}
		if !present {
			inv.Functions = append(inv.Functions, name)
		}
		out.Modules[pkg] = inv
	}
	for mod, inv := range out.Modules {
		sort.Strings(inv.Functions)
		for cls, methods := range inv.Classes {
			sort.Strings(methods)
			inv.Classes[cls] = methods
		}
		out.Modules[mod] = inv
	}
	return out
}

// --- CLI --------------------------------------------------------------------

// goSHA returns the signalwire-go repo HEAD SHA (or "N/A").
func goSHA(repoRoot string) string {
	cmd := exec.Command("git", "-C", repoRoot, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "N/A"
	}
	return strings.TrimSpace(string(out))
}

// findRepoRoot walks up from cwd looking for go.mod.
func findRepoRoot(cwd string) (string, error) {
	for dir := cwd; dir != "/"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
	}
	return "", fmt.Errorf("no go.mod found above %s", cwd)
}

func run() error {
	var (
		outputPath      = flag.String("output", "port_surface.json", "Write JSON to this path")
		goOutputPath    = flag.String("go-output", "port_surface_go.json", "Write Go-native surface JSON to this path (used by audit_docs.py)")
		additionsOutput = flag.String("additions-output", "port_additions_actual.json", "Write the unmapped-symbol inventory to this path; consumed by diff_port_surface.py to enforce PORT_ADDITIONS.md")
		stdout          = flag.Bool("stdout", false, "Print Python-shape JSON to stdout instead of --output")
		check           = flag.Bool("check", false, "Compare against existing --output / --go-output / --additions-output files; exit 1 on drift")
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

	structs, funcs, err := walk(pkgRoot)
	if err != nil {
		return fmt.Errorf("walk: %w", err)
	}

	sha := goSHA(repoRoot)

	snapshot := build(structs, funcs)
	snapshot.GeneratedFrom = fmt.Sprintf("signalwire-go @ %s", sha)

	rendered, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	rendered = append(rendered, '\n')

	goSnapshot := buildGoSurface(structs, funcs)
	goSnapshot.GeneratedFrom = fmt.Sprintf("signalwire-go (go-native) @ %s", sha)
	goRendered, err := json.MarshalIndent(goSnapshot, "", "  ")
	if err != nil {
		return err
	}
	goRendered = append(goRendered, '\n')

	additions := computePortAdditions(structs, funcs, repoRoot)
	addRendered, err := json.MarshalIndent(additions, "", "  ")
	if err != nil {
		return err
	}
	addRendered = append(addRendered, '\n')

	if *check {
		existing, err := os.ReadFile(*outputPath)
		if err != nil {
			return fmt.Errorf("check: read existing %s: %w", *outputPath, err)
		}
		if stripGen(rendered) != stripGen(existing) {
			fmt.Fprintln(os.Stderr, "DRIFT: port_surface.json is stale; regenerate with go run ./cmd/enumerate-surface")
			return fmt.Errorf("drift detected")
		}
		existingGo, err := os.ReadFile(*goOutputPath)
		if err != nil {
			return fmt.Errorf("check: read existing %s: %w", *goOutputPath, err)
		}
		if stripGen(goRendered) != stripGen(existingGo) {
			fmt.Fprintln(os.Stderr, "DRIFT: port_surface_go.json is stale; regenerate with go run ./cmd/enumerate-surface")
			return fmt.Errorf("drift detected")
		}
		existingAdd, err := os.ReadFile(*additionsOutput)
		if err != nil {
			return fmt.Errorf("check: read existing %s: %w", *additionsOutput, err)
		}
		if stripGen(addRendered) != stripGen(existingAdd) {
			fmt.Fprintln(os.Stderr, "DRIFT: port_additions_actual.json is stale; regenerate with go run ./cmd/enumerate-surface")
			return fmt.Errorf("drift detected")
		}
		return nil
	}

	if *stdout {
		_, err := os.Stdout.Write(rendered)
		return err
	}
	if err := os.WriteFile(*outputPath, rendered, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(*goOutputPath, goRendered, 0o644); err != nil {
		return err
	}
	return os.WriteFile(*additionsOutput, addRendered, 0o644)
}

func stripGen(b []byte) string {
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return string(b)
	}
	delete(m, "generated_from")
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return string(b)
	}
	return string(out)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
