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
}

type goFunc = goSignature // free function

type goStructFacts struct {
	pkg     string
	name    string
	methods map[string]*goSignature
}

func walk(root string) (map[string]*goStructFacts, map[string]*goFunc, error) {
	structs := map[string]*goStructFacts{}
	funcs := map[string]*goFunc{}

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
		return parseFile(path, structs, funcs)
	})
	return structs, funcs, err
}

func parseFile(path string, structs map[string]*goStructFacts, funcs map[string]*goFunc) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	pkgName := file.Name.Name
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
				if _, isStruct := ts.Type.(*ast.StructType); !isStruct {
					continue
				}
				key := pkgName + "." + ts.Name.Name
				if _, present := structs[key]; !present {
					structs[key] = &goStructFacts{
						pkg:     pkgName,
						name:    ts.Name.Name,
						methods: map[string]*goSignature{},
					}
				}
			}
		case *ast.FuncDecl:
			if !ast.IsExported(d.Name.Name) {
				continue
			}
			sig := buildSignature(pkgName, d)
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
		// Take the first result. Multi-return Go funcs collapse to the
		// first result for canonical shape; the second result is usually
		// an `error`, which is mapped to `any` and not part of the
		// Python signature. Methods that legitimately return tuples will
		// surface as drift and need PORT_SIGNATURE_OMISSIONS.md entries.
		first := fd.Type.Results.List[0]
		sig.returns = exprString(first.Type)
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
			for i := 0; i < n; i++ {
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
			for i := 0; i < n; i++ {
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

// translateType maps a source-level Go type expression to the canonical
// vocabulary. Returns ("", failure) when the type can't be translated;
// the caller decides whether to fail loudly or skip.
func translateType(t string, aliases map[string]string, ctx string) (string, *translationFailure) {
	t = strings.TrimSpace(t)
	if t == "" {
		return "void", nil
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
	// Function type → callable<list<args>,ret>
	if strings.HasPrefix(t, "func(") {
		return translateFunc(t, aliases, ctx)
	}
	// Direct alias hit
	if v, ok := aliases[t]; ok {
		return v, nil
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
	// Bare identifier — could be a struct in the same package
	if len(t) > 0 && t[0] >= 'A' && t[0] <= 'Z' {
		if classRef := lookupClassRefByShort(t); classRef != "" {
			return classRef, nil
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
	for i := 0; i < len(rest); i++ {
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
// ``relay.Call`` or ``agent.AgentBase`` to the canonical
// ``class:signalwire.<...>.<Class>`` form using StructTable.
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
	Version       string                       `json:"version"`
	GeneratedFrom string                       `json:"generated_from"`
	Modules       map[string]sigModuleInventory `json:"modules"`
}

type sigModuleInventory struct {
	Classes   map[string]sigClassEntry  `json:"classes,omitempty"`
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

func toCanonicalSignature(sig *goSignature, aliases map[string]string, isMethod bool, isCtor bool, ctx string) (canonicalSignature, []translationFailure) {
	var failures []translationFailure
	params := []canonicalParam{}
	if isMethod {
		params = append(params, canonicalParam{Name: "self", Kind: "self"})
	}
	for _, p := range sig.params {
		canon, fail := translateType(p.typeStr, aliases, ctx+"["+p.name+"]")
		if fail != nil {
			failures = append(failures, *fail)
			continue
		}
		params = append(params, canonicalParam{
			Name:     goNameToSnake(p.name),
			Type:     canon,
			Required: boolPtr(true), // Go has no defaults; every param is required
		})
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

func build(structs map[string]*goStructFacts, funcs map[string]*goFunc, aliases map[string]string) (sigDoc, []translationFailure) {
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
				if mSig, present := facts.methods[goMethod]; present {
					sig, fails := toCanonicalSignature(mSig, aliases, true, false, fmt.Sprintf("%s.%s.%s", target.Module, target.Class, pyMethod))
					failures = append(failures, fails...)
					addClassMethod(target.Module, target.Class, pyMethod, sig)
				}
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
		outputPath   = flag.String("output", "port_signatures.json", "Write JSON to this path")
		aliasesPath  = flag.String("aliases", "", "Path to porting-sdk/type_aliases.yaml (autodetected if empty)")
		strict       = flag.Bool("strict", false, "Exit non-zero on any translation failure")
		stdoutFlag   = flag.Bool("stdout", false, "Print to stdout")
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
			"/usr/local/home/devuser/src/porting-sdk/type_aliases.yaml",
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

	structs, funcs, err := walk(pkgRoot)
	if err != nil {
		return fmt.Errorf("walk: %w", err)
	}

	doc, failures := build(structs, funcs, aliases)
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
