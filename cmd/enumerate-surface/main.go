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

// funcReturns records, for each exported free function ("<pkg>.<Name>"), the SHORT
// name of its single named return TYPE (e.g. "AgentOption") when the function returns
// exactly one bare/pointer identifier type. Used by the functional-options fold to
// recognise `WithX() <T>Option` constructors generically (the generalisation of the
// hardcoded aiChatOptionFuncs allowlist). Empty when the return isn't a single simple
// identifier (multi-return, tuple, map, etc.).
var funcReturns = map[string]string{}

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
	// fields holds the SHORT (exported, PascalCase) names of the struct's own
	// declared data fields (non-embedded). Recorded so the oracle-gated relay
	// Event / AI-Chat DTO / RequestOptions field emission (emitDataclassFields)
	// can surface each public field the @dataclass reference now enumerates —
	// the deserialized event payload the Go event struct carries. The base-class
	// (*RelayEvent) fields promoted through the embed are NOT here (per-subclass
	// declared fields only), which matches the reference: the oracle records a
	// subclass's own dataclass fields on the subclass and the base's on the base.
	fields map[string]struct{}
	// readers holds the subset of `methods` that are ZERO-ARG, SINGLE-RETURN
	// exported methods — i.e. Go's idiomatic READ ACCESSOR over an unexported
	// field (`func (c *Call) CallID() string { return c.callID }`). Recorded so
	// the ORACLE-GATED accessor fold can surface them under the reference's plain
	// attribute spelling (ALLOWLIST_DISCIPLINE §7 row 1: `getX()` folds to the
	// public attribute `self.x`). Gating on the oracle is what makes this safe and
	// self-retiring: an accessor is folded ONLY when the reference records a member
	// of that snake_case name on that same class, so it can never invent surface
	// and it needs no hand-maintained per-symbol list.
	readers map[string]struct{}
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
						fields:  map[string]struct{}{},
						readers: map[string]struct{}{},
					}
				}
				if structs[key].fields == nil {
					structs[key].fields = map[string]struct{}{}
				}
				if structs[key].readers == nil {
					structs[key].readers = map[string]struct{}{}
				}
				// §5/§4a: mark generated-REST params-struct plumbing so it is
				// excluded from the SURFACE-DIFF additions inventory.
				if isRestResource && strings.HasSuffix(ts.Name.Name, "Params") {
					structs[key].paramsPlumbing = true
				}
				// Record anonymous (embedded) fields so promoted methods can be
				// resolved through the embed chain during projection.
				structs[key].embeds = append(structs[key].embeds, embeddedTypeNames(st)...)
				// Record own (non-embedded) exported data-field names for the
				// oracle-gated @dataclass field emission (emitDataclassFields).
				if st.Fields != nil {
					for _, f := range st.Fields.List {
						if len(f.Names) == 0 {
							continue // embedded field, not a named data field
						}
						for _, n := range f.Names {
							if ast.IsExported(n.Name) {
								structs[key].fields[n.Name] = struct{}{}
							}
						}
					}
				}
			}
		case *ast.FuncDecl:
			if !ast.IsExported(d.Name.Name) {
				continue
			}
			if d.Recv == nil || len(d.Recv.List) == 0 {
				key := pkgName + "." + d.Name.Name
				funcs[key] = struct{}{}
				funcReturns[key] = singleReturnTypeName(d.Type)
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
					fields:  map[string]struct{}{},
					readers: map[string]struct{}{},
				}
			}
			if structs[key].readers == nil {
				structs[key].readers = map[string]struct{}{}
			}
			structs[key].methods[d.Name.Name] = struct{}{}
			// A zero-arg, single-return exported method is Go's read accessor
			// over an unexported field; record it for the oracle-gated accessor
			// fold (see goStructFacts.readers).
			if isZeroArgReader(d.Type) {
				structs[key].readers[d.Name.Name] = struct{}{}
			}
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

// singleReturnTypeName returns the SHORT type name of a function's return value when
// it returns EXACTLY ONE result whose type is a bare or pointer identifier
// (`AgentOption`, `*Foo`); otherwise "". Used to recognise `WithX() <T>Option`
// functional-option constructors.
func singleReturnTypeName(ft *ast.FuncType) string {
	if ft == nil || ft.Results == nil || len(ft.Results.List) != 1 {
		return ""
	}
	r := ft.Results.List[0]
	if len(r.Names) > 1 {
		return ""
	}
	return recvTypeName(r.Type)
}

// isZeroArgReader reports whether a method signature is a READ ACCESSOR: no
// parameters and exactly one result. That is Go's idiomatic expression of a
// public attribute over an unexported field, and it is the shape the
// oracle-gated accessor fold surfaces under the reference's plain attribute
// name. A method taking arguments is a verb, not a reader; a method returning
// nothing or a (value, error) pair is not an attribute read either.
func isZeroArgReader(ft *ast.FuncType) bool {
	if ft == nil || ft.Results == nil || len(ft.Results.List) != 1 {
		return false
	}
	if len(ft.Results.List[0].Names) > 1 {
		return false
	}
	if ft.Params != nil && len(ft.Params.List) > 0 {
		return false
	}
	return true
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

// --- @dataclass field emission (oracle-gated) -------------------------------

// goNameToPython delegates to internal/surface, the SINGLE home of the
// Go-identifier -> Python-canonical fold. See internal/surface/names.go for why
// the correction table is shared with cmd/enumerate-signatures rather than
// duplicated (the two copies had already diverged on `FAQs`).
func goNameToPython(s string) string { return surfacepkg.GoNameToPython(s) }

// oracleModuleMembers is the parse of python_surface.json restricted to the
// per-class member sets emitDataclassFields gates on. Shape:
// module -> class -> set(reference member names).
type oracleModuleMembers map[string]map[string]map[string]bool

// RETIRED: dataclassFieldModules.
//
// Field emission used to be scoped to a CLOSED set of three modules
// (signalwire.relay.event, signalwire.ai_chat.client,
// signalwire.rest._request_options) — the modules whose reference classes carried
// public @dataclass fields when the table was written.
//
// The oracle has since grown well past those three: class B2 made every public
// `__init__` attribute that is ALSO a constructor param part of the recorded
// surface, across the whole SDK. The hardcoded module list then became a stale
// exclusion that silently threw away capability the port already had — 30 of go's
// 110 missing symbols were fields or accessors the Go structs carried, dropped
// only because their module was not one of the three.
//
// loadOracleMembers now returns EVERY module the oracle records, so the ORACLE
// alone gates emission. That is strictly safer than the module list it replaces
// (a field is emitted only when the reference records a member of that name on
// that same class, so it can never over-emit) and it self-retires: the gate tracks
// the oracle instead of needing a hand edit each time the oracle grows.

// resolvePortingSDK locates the adjacent porting-sdk checkout that carries the
// reference oracle, trying in order:
//
//  1. $PORTING_SDK — the explicit override every CI workflow in the matrix sets
//     (and the only reliable answer when the layout is not sibling-adjacent).
//  2. <repoRoot>/../porting-sdk — the local development layout, where porting-sdk
//     is a SIBLING of the port repo.
//  3. <repoRoot>/porting-sdk — the CI layout, where actions/checkout places
//     porting-sdk INSIDE the port repo (`path: porting-sdk`).
//
// It returns an error rather than "" when none resolves, and every caller must
// FAIL on that error rather than degrade.
//
// That last point is the whole reason this function exists. The oracle loaders used
// to hardcode layout (2) and return nil on any failure — so under the CI layout the
// walk missed, the oracle loaded EMPTY, and every oracle-gated field simply did not
// emit. The result was a valid-LOOKING port_surface.json that was missing ~200
// members, a surface-audit red that could not be reproduced locally (where the
// sibling DOES resolve), and a long hunt for a cause that was never in the port's
// source. dotnet's lane hit the identical trap. A gate that cannot resolve its
// oracle must say so, not quietly emit less.
func resolvePortingSDK(repoRoot string) (string, error) {
	candidates := []string{}
	if env := os.Getenv("PORTING_SDK"); env != "" {
		candidates = append(candidates, env)
	}
	candidates = append(candidates,
		filepath.Join(repoRoot, "..", "porting-sdk"),
		filepath.Join(repoRoot, "porting-sdk"),
	)
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "python_surface.json")); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf(
		"porting-sdk not found (looked for python_surface.json under %v); "+
			"set PORTING_SDK or clone porting-sdk adjacent to this repo", candidates)
}

// loadOracleMembers reads python_surface.json and returns the per-class member
// sets. FAILS LOUD: an unresolvable or unparseable oracle is an error, never a
// silent empty result (see resolvePortingSDK).
func loadOracleMembers(repoRoot string) (oracleModuleMembers, error) {
	psdk, err := resolvePortingSDK(repoRoot)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(psdk, "python_surface.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read oracle %s: %w", path, err)
	}
	var parsed struct {
		Modules map[string]struct {
			Classes map[string][]string `json:"classes"`
		} `json:"modules"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parse oracle %s: %w", path, err)
	}
	out := oracleModuleMembers{}
	for mod, mi := range parsed.Modules {
		cm := map[string]map[string]bool{}
		for cls, members := range mi.Classes {
			set := map[string]bool{}
			for _, m := range members {
				set[m] = true
			}
			cm[cls] = set
		}
		out[mod] = cm
	}
	return out, nil
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
func build(structs map[string]*goStructFacts, funcs map[string]struct{}, oracle oracleModuleMembers) surface {
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
			// @dataclass field emission (oracle-gated). For the closed set of
			// modules whose reference classes carry public @dataclass fields
			// (relay Event structs, AI-Chat DTOs, RequestOptions), surface each
			// exported Go struct field whose snake_case name is in the oracle's
			// member set for this (module, class). This makes the deserialized
			// event-payload fields the Go struct carries PRESENT (folded), so the
			// 106 reference fields compare EQUAL instead of showing as omissions.
			// Gating on the oracle guarantees we emit exactly the reference set
			// and never a port-internal helper field.
			if clsMembers, ok := oracle[target.Module][target.Class]; ok {
				for goField := range facts.fields {
					snake := goNameToPython(goField)
					if clsMembers[snake] {
						addMethod(target.Module, target.Class, snake)
					}
				}
				// Accessor fold (ALLOWLIST_DISCIPLINE §7 row 1), same oracle
				// gate. Go's idiomatic expression of a public attribute over an
				// unexported field is a zero-arg reader method
				// (`func (c *Call) CallID() string { return c.callID }`). Fold
				// each such reader onto the reference's plain attribute spelling
				// when the oracle records a member of that snake_case name on
				// this same class.
				//
				// Why this is not an ad-hoc rename table: the ORACLE decides.
				// The fold can only ever emit a name the reference already
				// records on this class, so it cannot invent surface, and it
				// retires itself as the oracle changes — there is no
				// hand-maintained per-symbol list to go stale (the failure mode
				// java and dotnet both hit with their strip tables).
				//
				// EXCLUDED: a reader whose snake name the StructTable already
				// maps some Go method to. That mapping is the deliberate,
				// reviewed projection; keeping it authoritative is what stops
				// the fold from quietly re-pointing a real reference METHOD at
				// a same-named accessor.
				mapped := map[string]bool{}
				for _, py := range target.Methods {
					mapped[py] = true
				}
				for goReader := range facts.readers {
					snake := goNameToPython(goReader)
					if clsMembers[snake] && !mapped[snake] {
						addMethod(target.Module, target.Class, snake)
					}
				}
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

// isMockTestSymbol reports whether a surface key belongs to the `mocktest`
// shared test-harness package (keys are "<pkg>.<Symbol>"). Those symbols are
// test infrastructure, not shipped SDK surface, so they are excluded from the
// port-additions inventory (they must never register as SURFACE-DIFF additions).
func isMockTestSymbol(key string) bool {
	return strings.HasPrefix(key, "mocktest.")
}

// optionsPlumbingStructs is the set of per-call OPTIONS STRUCTS whose fields
// enumerate-signatures UNFOLDS back into the configured method's signature (the
// ai_chat per-call options via aiChatMethodSigs, and the swml/swaig options via
// optionsStructUnfoldMethods — both in cmd/enumerate-signatures/main.go). They are
// call-shape plumbing (the Go named-options idiom for the Python kwargs), not oracle
// surface, so they are excluded from the SURFACE-DIFF additions inventory — the same
// treatment the generated-REST *Params plumbing structs get. This generalises the
// former hardcoded ai_chat-only allowlist to every unfolded options struct.
// (If you add a struct to optionsStructUnfoldMethods, add it here too.)
var optionsPlumbingStructs = map[string]bool{
	// ai_chat per-call options (unfolded via aiChatMethodSigs)
	"aichat.CreateOptions":    true,
	"aichat.ChatOptions":      true,
	"aichat.SummarizeOptions": true,
	// swml/swaig per-call options (unfolded via optionsStructUnfoldMethods)
	"swml.PlayOptions":         true,
	"swml.AIOptions":           true,
	"swaig.ConnectOptions":     true,
	"swaig.WaitForUserOptions": true,
}

// isFunctionalOptionCtor reports whether an exported free function key
// ("<pkg>.<Name>") is a functional-options constructor: its name starts with `With`
// and its sole return type is a `*Option`-suffixed identifier (e.g. AgentOption,
// ConferenceOption, aichat.ClientOption). These encode a single Python keyword
// argument of the constructor/method they configure; enumerate-signatures unfolds the
// variadic `...Option` back into that expanded param list, so they carry no standalone
// oracle surface and must not register as SURFACE-DIFF additions. This is the general
// form of the retired hardcoded aiChatOptionFuncs allowlist.
func isFunctionalOptionCtor(key string) bool {
	dot := strings.Index(key, ".")
	if dot < 0 {
		return false
	}
	name := key[dot+1:]
	if !strings.HasPrefix(name, "With") {
		return false
	}
	// The return type is an option type when it is named `Option` or `<T>Option`
	// (ai_chat's + security's option type is the bare `Option`; agent/relay/swml use
	// `<T>Option`). A `With*` free function returning such a type is a functional-option
	// constructor. Empty return (multi-return / non-identifier) does NOT qualify.
	ret := funcReturns[key]
	return ret == "Option" || strings.HasSuffix(ret, "Option")
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
		// Per-call options structs whose fields enumerate-signatures unfolds back into
		// the configured method signature (ai_chat + swml/swaig — see
		// optionsPlumbingStructs) are call-shape plumbing, not oracle surface, exactly
		// like the REST *Params plumbing. Never list them as SURFACE-DIFF additions.
		if optionsPlumbingStructs[key] {
			continue
		}
		// The mocktest package is the shared test harness (mock server + journal),
		// not shipped SDK surface — exclude it the same way as params plumbing so
		// its exports never register as SURFACE-DIFF port-additions.
		if isMockTestSymbol(key) {
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
		// ai_chat functional-options constructors (aichat.With*) are the Go spelling
		// of the AIChatClient.__init__ kwargs (project/token/space/url/session);
		// enumerate-signatures splices them into __init__, so they are call-shape
		// plumbing, not standalone oracle surface. Exclude from SURFACE-DIFF additions.
		if isFunctionalOptionCtor(key) {
			continue
		}
		// mocktest is the shared test harness, not shipped SDK surface (see above).
		if isMockTestSymbol(key) {
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

// --- Composition-attribute enrich -------------------------------------------

// sigSnapshot is the minimal shape of port_signatures.json needed to import
// composition attributes into the surface.
type sigSnapshot struct {
	Modules map[string]struct {
		Classes map[string]struct {
			Methods map[string]struct {
				Params []struct {
					Kind string `json:"kind"`
				} `json:"params"`
				Returns string `json:"returns"`
			} `json:"methods"`
		} `json:"classes"`
	} `json:"modules"`
}

// isCompositionReturn mirrors the Python surface enumerator's
// `_enrich_composition_attributes._is_composition_return` (porting-sdk
// scripts/enumerate_python.py): a composition attribute RETURNS an SDK class —
// bare (`class:signalwire.…`) or wrapped in `optional<…>` / `list<…>` — but NOT a
// `union<…>` (those are the auto-vivified SWML verb SETTERS, a distinct idiom class
// folded elsewhere). Scalar state (`string`, `int`, …) is not class-typed and is
// not imported, keeping go's two oracles (surface + signatures) consistent BY
// CONSTRUCTION exactly as the Python pair is.
// pythonReservedWords are identifiers the Python reference generator cannot surface
// as a member name — it drops them to a comment (the same set the SIGNATURE diff
// tolerates as reserved-word leaves, e.g. `else`/`from`). Go legitimately emits such a
// wire-field accessor, but importing it as a composition attribute would surface a
// member the reference can never have → a phantom addition. Exclude them here; the
// signature side already carries the reserved-word leaf, so the two oracles stay
// consistent (the member is present in signatures, absent from surface — matching the
// reference on BOTH sides).
var pythonReservedWords = map[string]bool{
	"else": true, "from": true, "import": true, "class": true, "def": true,
	"return": true, "global": true, "lambda": true, "pass": true, "raise": true,
	"yield": true, "async": true, "await": true, "with": true, "as": true,
	"not": true, "and": true, "or": true, "is": true, "in": true, "if": true,
	"elif": true, "while": true, "for": true, "try": true, "except": true,
	"finally": true, "del": true, "assert": true, "break": true, "continue": true,
	"nonlocal": true, "None": true, "True": true, "False": true,
}

func isCompositionReturn(ret string) bool {
	if strings.HasPrefix(ret, "union<") {
		return false
	}
	return strings.Contains(ret, "class:signalwire.")
}

// enrichCompositionAttributes adds COMPOSITION-ATTRIBUTE members from go's own
// signature oracle (port_signatures.json) into the surface snapshot in place, the
// exact mirror of the Python reference's `_enrich_composition_attributes`. The
// surface `build` step projects only StructTable-mapped ergonomic methods and drops
// self-only class-returning accessors (REST namespace accessors like
// `FabricNamespace.addresses`, generated SWML-model getters like `AIObject.hints`,
// composition handles like `AgentServer.logger`); the SIGNATURE oracle already
// records them as griffe-typed self-only members. Importing them here makes the two
// go oracles consistent by construction — the same guarantee the Python pair has —
// so the composition surface the reference now enumerates is PRESENT (folded), not a
// phantom omission. Idempotent; skips silently if port_signatures.json is absent
// (first-generation / degraded env), so surface never HARD-depends on it.
//
// A member is a composition attribute iff its signature is self-only (no params
// other than the receiver) AND returns an SDK class per isCompositionReturn.
func enrichCompositionAttributes(snapshot *surface, repoRoot string) error {
	sigPath := filepath.Join(repoRoot, "port_signatures.json")
	raw, err := os.ReadFile(sigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var sig sigSnapshot
	if err := json.Unmarshal(raw, &sig); err != nil {
		return fmt.Errorf("enrich: parse %s: %w", sigPath, err)
	}
	for mod, sinv := range sig.Modules {
		for cls, sce := range sinv.Classes {
			var comp []string
			for m, msig := range sce.Methods {
				// A reserved-word leaf (`else`, `from`, …) cannot be a reference
				// surface member — the Python generator drops it. Emitting it here
				// would be a phantom addition; skip it (it stays on the signature side).
				if pythonReservedWords[m] {
					continue
				}
				nonSelf := false
				for _, p := range msig.Params {
					if p.Kind != "self" {
						nonSelf = true
						break
					}
				}
				if nonSelf {
					continue
				}
				if isCompositionReturn(msig.Returns) {
					comp = append(comp, m)
				}
			}
			if len(comp) == 0 {
				continue
			}
			inv, ok := snapshot.Modules[mod]
			if !ok {
				inv = moduleInventory{Classes: map[string][]string{}, Functions: []string{}}
			}
			if inv.Classes == nil {
				inv.Classes = map[string][]string{}
			}
			existing := inv.Classes[cls]
			seen := map[string]bool{}
			for _, e := range existing {
				seen[e] = true
			}
			for _, m := range comp {
				if !seen[m] {
					existing = append(existing, m)
					seen[m] = true
				}
			}
			sort.Strings(existing)
			inv.Classes[cls] = existing
			snapshot.Modules[mod] = inv
		}
	}
	return nil
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

// assertOracleGatedEmission verifies the oracle-gated field/accessor fold actually
// emitted. ChatLog.messages is the canary: it is a plain public @dataclass field on
// a class the walker always finds, so it is present whenever the fold ran and
// absent whenever the oracle failed to load — exactly the signal that was silently
// missing before.
func assertOracleGatedEmission(snapshot *surface) error {
	const (
		mod    = "signalwire.ai_chat.client"
		cls    = "ChatLog"
		member = "messages"
	)
	for _, m := range snapshot.Modules[mod].Classes[cls] {
		if m == member {
			return nil
		}
	}
	return fmt.Errorf(
		"oracle-gated emission produced nothing: %s.%s.%s is missing from the "+
			"snapshot. The reference oracle was resolved but its member sets did not "+
			"reach the fold — do NOT commit this snapshot, it is missing every "+
			"oracle-gated field and accessor", mod, cls, member)
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

	oracle, err := loadOracleMembers(repoRoot)
	if err != nil {
		return fmt.Errorf("load reference oracle: %w", err)
	}
	snapshot := build(structs, funcs, oracle)
	if err := enrichCompositionAttributes(&snapshot, repoRoot); err != nil {
		return fmt.Errorf("enrich composition attributes: %w", err)
	}
	snapshot.GeneratedFrom = fmt.Sprintf("signalwire-go @ %s", sha)

	// Self-check: assert an ORACLE-GATED member actually made it into the snapshot.
	//
	// resolvePortingSDK now fails loud, so an unreachable oracle can no longer
	// produce a quietly-degraded snapshot. This is the belt to that braces: it
	// catches any FUTURE way the gating could silently no-op (a renamed oracle key,
	// a reshaped surface JSON, a fold that stops firing) at the moment of emission
	// rather than as an unreproducible parity red days later.
	if err := assertOracleGatedEmission(&snapshot); err != nil {
		return err
	}

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
