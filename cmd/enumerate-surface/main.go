// Command enumerate-surface emits a JSON snapshot of the Go SDK's public API
// translated into Python-reference symbol names.
//
// The output (``port_surface.json``) is compared against
// ``porting-sdk/python_surface.json`` by ``diff_port_surface.py`` to detect
// unexcused drift.  Each Go struct is mapped onto a (python_module,
// python_class) pair and each Go method onto a python method name — so that
// ``AgentBase.SetPromptText`` is emitted as
// ``signalwire.core.mixins.prompt_mixin.PromptMixin.set_prompt_text``.  The
// same Go struct may contribute to multiple Python classes (``AgentBase`` is
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

	surfacepkg "github.com/signalwire/signalwire-go/internal/surface"
)

// Re-export internal/surface package symbols under the previous local
// names so the existing main.go body keeps working without churn.
type classTarget = surfacepkg.ClassTarget

var (
	structTable = surfacepkg.StructTable
	freeFnTable = surfacepkg.FreeFnTable
	factoryInit = surfacepkg.FactoryInit
)

// --- AST walker -------------------------------------------------------------

// goStructFacts is the raw Go inventory for a single struct.
type goStructFacts struct {
	pkg     string
	name    string
	methods map[string]struct{}
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
						methods: map[string]struct{}{},
					}
				}
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

// surface is the final JSON shape.  Matches ``python_surface.json``.
type surface struct {
	Version       string                     `json:"version"`
	GeneratedFrom string                     `json:"generated_from"`
	Modules       map[string]moduleInventory `json:"modules"`
}

type moduleInventory struct {
	Classes   map[string][]string `json:"classes"`
	Functions []string            `json:"functions"`
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
// implicitly covered by listing the struct itself. Factory ``New<Struct>``
// constructors paired with a mapped struct are already projected as
// __init__ and not listed here.
func computePortAdditions(structs map[string]*goStructFacts, funcs map[string]struct{}, repo string) PortAdditions {
	var addStructs []string
	for key := range structs {
		if _, ok := structTable[key]; !ok {
			addStructs = append(addStructs, key)
		}
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
// **native** Go struct + method names.  Unlike ``build`` — which translates
// everything onto the Python reference's dotted path — this captures the
// exact identifiers a Go doc or example would use (``AgentBase.DefineTool``,
// ``RestClient``, ``RunAgent``).  Used by ``audit_docs.py`` on the Go port
// so that method-call references resolve against the actual surface.
//
// Shape matches ``port_surface.json`` but the module name is the short Go
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
		outputPath        = flag.String("output", "port_surface.json", "Write JSON to this path")
		goOutputPath      = flag.String("go-output", "port_surface_go.json", "Write Go-native surface JSON to this path (used by audit_docs.py)")
		additionsOutput   = flag.String("additions-output", "port_additions_actual.json", "Write the unmapped-symbol inventory to this path; consumed by diff_port_surface.py to enforce PORT_ADDITIONS.md")
		stdout            = flag.Bool("stdout", false, "Print Python-shape JSON to stdout instead of --output")
		check             = flag.Bool("check", false, "Compare against existing --output / --go-output / --additions-output files; exit 1 on drift")
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
	out, _ := json.MarshalIndent(m, "", "  ")
	return string(out)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
