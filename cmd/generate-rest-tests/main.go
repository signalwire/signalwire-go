// Command generate-rest-tests emits the full-mock REST wire-test suite for the
// Go SDK — a SUCCESS + an ERROR test for every REST route the SDK actually
// implements, run against the shared porting-sdk mock_signalwire server.
//
// It is the Go realization of REST_TEST_GENERATOR_RULES.md (the portable REST
// *test* generator; reference: generate_python_rest_types.py::generate_rest_tests,
// mirror: signalwire-typescript/scripts/generate-rest-tests.ts). For each route:
//
//   - a SUCCESS test: call the real client method against the mock, assert the
//     mock journaled the expected (method, matched_route);
//   - an ERROR test: arm a 500 for that route, assert the SDK returns
//     *rest.SignalWireRestError with StatusCode 500.
//
// The assertion oracle is INDEPENDENT of the resource generator (RULES §1):
//
//   - the (method, path) to call comes from the route registry (cmd/route-registry,
//     captured from the REAL client) — reused via os/exec, NOT re-walked here;
//   - the matched_route to assert comes from the OpenAPI operationId
//     (<spec_dir>.<operationId>), the same value the mock derives its route table
//     from — so a generated test catches SDK-vs-contract drift, not a generator
//     self-snapshot.
//
// Inputs joined by (METHOD, normalized-path) (RULES §2): the registry's routes
// (path params already normalized to {id}) × the spec operationIds (the spec path
// normalized the SAME way before the join). Routing collisions are resolved
// longest-template-wins (RULES §7) so the asserted route is the one the mock
// ACTUALLY journals (e.g. GET /rooms/{id} vs GET /rooms/{name}).
//
// Call args are type-correct BY CONSTRUCTION (RULES §4/§6): each via method's
// real parameter types are read by reflecting the live client, and a Go literal
// of the right kind is synthesized (path-id string → "x-1"; map[string]string
// query → nil; map[string]any body → a filled literal of the spec op's required
// fields; context.Context → context.Background(); trailing variadic → omitted).
// The generated files compile under go build / go vet with no edits.
//
// GEN-FRESH: `--check` reproduces the committed *_generated_test.go and exits
// non-zero if any file differs. Resolves porting-sdk via $PORTING_SDK or sibling.
//
// Usage:
//
//	go run ./cmd/generate-rest-tests          # (re)write the generated test files
//	go run ./cmd/generate-rest-tests --check  # GEN-FRESH: fail if any is stale
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/signalwire/signalwire-go/pkg/rest"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// 1. The route registry — captured from the real client (RULES §3).
//
// We reuse the EXISTING cmd/route-registry binary via os/exec (it already
// constructs the live client with a recording transport and walks every
// namespace/resource/method). Reading its JSON keeps a SINGLE capture
// implementation (no second walk that could drift). It emits ONLY JSON to
// stdout; the SDK logger writes to stderr.
// ---------------------------------------------------------------------------

type routeRec struct {
	Method       string   `json:"method"`
	PathTemplate string   `json:"path_template"`
	Via          []string `json:"via"`
}

type registryOut struct {
	Routes  []routeRec        `json:"routes"`
	Skipped []json.RawMessage `json:"skipped"`
	Errors  []json.RawMessage `json:"errors"`
}

func loadRegistry(repoRoot string) ([]routeRec, error) {
	cmd := exec.Command("go", "run", "./cmd/route-registry")
	cmd.Dir = repoRoot
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("run cmd/route-registry: %w", err)
	}
	// route-registry may emit a log line before the JSON; slice from the first '{'.
	if i := bytes.IndexByte(out, '{'); i > 0 {
		out = out[i:]
	}
	var reg registryOut
	if err := json.Unmarshal(out, &reg); err != nil {
		return nil, fmt.Errorf("decode route-registry JSON: %w", err)
	}
	if len(reg.Errors) != 0 {
		return nil, fmt.Errorf("route-registry reported %d capture error(s) — Set B incomplete (fix cmd/route-registry first)", len(reg.Errors))
	}
	return reg.Routes, nil
}

// ---------------------------------------------------------------------------
// 2. The join — registry routes × spec operationIds by (method, normalized-path).
// ---------------------------------------------------------------------------

type joinedRow struct {
	method string // HTTP verb, uppercased
	path   string // registry path_template (already normalized to {id})
	opID   string // <spec_dir>.<operationId> (the wire-collision winner)
	via    string // <ns>.<resource>.<method>
	spec   string // spec dir name (= matched_route prefix; the test-file group key)
}

// normParams normalizes every {param} to {id} (the registry already does this for
// its captured params; do it to the spec path too so renamed params — {token_id},
// {name} — line up).
func normParams(p string) string {
	return replaceBraces(p, "{id}")
}

// wireKey turns every {param} into a bare X: the wire-identical key used for
// collision ranking (longest original template wins).
func wireKey(p string) string {
	return replaceBraces(p, "X")
}

// replaceBraces replaces every {…} run in p with repl.
func replaceBraces(p, repl string) string {
	var b strings.Builder
	for {
		i := strings.IndexByte(p, '{')
		if i < 0 {
			b.WriteString(p)
			break
		}
		j := strings.IndexByte(p[i:], '}')
		if j < 0 {
			b.WriteString(p)
			break
		}
		b.WriteString(p[:i])
		b.WriteString(repl)
		p = p[i+j+1:]
	}
	return b.String()
}

// specPrefix returns the servers[0].url path portion after "signalwire.com".
func specPrefix(root *yaml.Node) string {
	servers := mapChild(root, "servers")
	if servers == nil || servers.Kind != yaml.SequenceNode || len(servers.Content) == 0 {
		return ""
	}
	url := scalarChild(servers.Content[0], "url")
	if i := strings.Index(url, "signalwire.com"); i >= 0 {
		return url[i+len("signalwire.com"):]
	}
	return ""
}

// wireWinner tracks the longest original spec template mapping to a wireKey.
type wireWinner struct {
	length int
	route  string // <spec>.<operationId>
}

// join builds the (method,normalized-path)→matched_route join and returns the
// rows for every registry route that has a spec op AND a non-empty via.
func join(routes []routeRec, psdk string, specDirs []string) ([]joinedRow, error) {
	opBy := map[string]string{}     // "METHOD normPath" -> <spec>.<operationId>
	wire := map[string]wireWinner{} // "METHOD wireKey"  -> longest-template winner
	verbs := []string{"get", "post", "put", "patch", "delete"}

	for _, spec := range specDirs {
		raw, err := os.ReadFile(filepath.Join(psdk, "rest-apis", spec, "openapi.yaml"))
		if err != nil {
			return nil, err
		}
		var doc yaml.Node
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			return nil, err
		}
		root := rootOf(&doc)
		prefix := specPrefix(root)
		paths := mapChild(root, "paths")
		if paths == nil || paths.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(paths.Content); i += 2 {
			pathKey := paths.Content[i].Value
			body := paths.Content[i+1]
			orig := prefix + pathKey // original spec path with real param names
			full := replaceBraces(orig, "{id}")
			wk := replaceBraces(orig, "X")
			for _, verb := range verbs {
				op := mapChild(body, verb)
				if op == nil {
					continue
				}
				opID := scalarChild(op, "operationId")
				if opID == "" {
					continue
				}
				route := spec + "." + opID
				vu := strings.ToUpper(verb)
				opBy[vu+" "+full] = route
				wkey := vu + " " + wk
				if cur, ok := wire[wkey]; !ok || len(orig) > cur.length {
					wire[wkey] = wireWinner{length: len(orig), route: route}
				}
			}
		}
	}

	var rows []joinedRow
	for _, r := range routes {
		if len(r.Via) == 0 {
			continue // helper route with no via — skip
		}
		np := normParams(r.PathTemplate)
		if _, ok := opBy[r.Method+" "+np]; !ok {
			continue // no spec op for this route — skip (coverage finding, not a bug)
		}
		w, ok := wire[r.Method+" "+wireKey(r.PathTemplate)]
		if !ok {
			continue
		}
		opID := w.route
		spec := opID[:strings.IndexByte(opID, '.')]
		rows = append(rows, joinedRow{
			method: r.Method,
			path:   np,
			opID:   opID,
			via:    r.Via[0],
			spec:   spec,
		})
	}
	return rows, nil
}

// ---------------------------------------------------------------------------
// 3. Type-correct call synthesis (RULES §4/§6) via live-client reflection.
//
// We reflect the live RestClient exactly as cmd/route-registry does, indexing
// each via path (<ns>.<resource>.<method>) to the ordered reflect input types of
// its method. The emitted call uses the flat-namespace-collapsed attribute path
// and a type-correct literal per required parameter.
// ---------------------------------------------------------------------------

// callInfo holds the reflected input types of a via method (receiver excluded)
// and whether its final parameter is variadic.
type callInfo struct {
	in       []reflect.Type
	variadic bool
}

var ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()

// buildCallIndex reflects the live client and maps each via path to its method's
// parameter types. Mirrors cmd/route-registry's walk (descend into the embedded
// _GeneratedResourceTree; a namespace may itself be a flat resource AND a
// container of sub-resources).
func buildCallIndex() (map[string]callInfo, error) {
	client, err := rest.NewRestClient("p", "t", "example.signalwire.com")
	if err != nil {
		return nil, err
	}
	index := map[string]callInfo{}

	record := func(nsName, resName string, resVal reflect.Value) {
		t := resVal.Type()
		for i := range t.NumMethod() {
			m := t.Method(i)
			if !m.IsExported() {
				continue
			}
			ft := m.Func.Type()
			ci := callInfo{variadic: ft.IsVariadic()}
			for j := 1; j < ft.NumIn(); j++ { // skip receiver at 0
				ci.in = append(ci.in, ft.In(j))
			}
			index[nsName+"."+resName+"."+m.Name] = ci
		}
	}

	isResourceLike := func(v reflect.Value) bool {
		return v.Kind() == reflect.Pointer && !v.IsNil() && v.Elem().Kind() == reflect.Struct
	}
	hasMethods := func(v reflect.Value) bool { return v.Type().NumMethod() > 0 }

	var walk func(sv reflect.Value)
	walk = func(sv reflect.Value) {
		st := sv.Type()
		for i := range st.NumField() {
			f := st.Field(i)
			if f.Anonymous {
				fv := sv.Field(i)
				if fv.Kind() == reflect.Struct {
					walk(fv)
				} else if fv.Kind() == reflect.Pointer && !fv.IsNil() && fv.Elem().Kind() == reflect.Struct {
					walk(fv.Elem())
				}
				continue
			}
			if !f.IsExported() {
				continue
			}
			nsVal := sv.Field(i)
			if !isResourceLike(nsVal) {
				continue
			}
			nsName := f.Name
			if hasMethods(nsVal) {
				record(nsName, nsName, nsVal)
			}
			// sub-resources
			e := nsVal.Elem()
			et := e.Type()
			for j := range et.NumField() {
				sf := et.Field(j)
				if !sf.IsExported() || sf.Anonymous {
					continue
				}
				sv2 := e.Field(j)
				if isResourceLike(sv2) {
					record(nsName, sf.Name, sv2)
				}
			}
		}
	}
	walk(reflect.ValueOf(client).Elem())
	return index, nil
}

// attrPath collapses the flat-namespace leading duplicate for the emitted
// client.<...> expression (Calling.Calling.Dial → Calling.Dial); container
// namespaces (Video.Rooms.Get) keep all segments.
func attrPath(via string) string {
	segs := strings.Split(via, ".")
	if len(segs) >= 2 && segs[0] == segs[1] {
		return strings.Join(segs[1:], ".")
	}
	return via
}

// synthCall returns the literal `client.<attr>(args…)` for a via, or ("", false)
// if the method is not reflected. Body params (map[string]any) are filled from
// the joined route's spec required fields when available.
func synthCall(row joinedRow, index map[string]callInfo, bodyFill string) (string, bool) {
	ci, ok := index[row.via]
	if !ok {
		return "", false
	}
	var args []string
	n := len(ci.in)
	for i, pt := range ci.in {
		if ci.variadic && i == n-1 {
			break // trailing variadic (…extras) omitted
		}
		args = append(args, sentinelFor(pt, bodyFill))
	}
	return "client." + attrPath(row.via) + "(" + strings.Join(args, ", ") + ")", true
}

// sentinelFor emits a type-correct Go literal for a parameter type.
func sentinelFor(t reflect.Type, bodyFill string) string {
	switch t.Kind() {
	case reflect.String:
		return `"x-1"`
	case reflect.Map:
		// map[string]any body -> filled literal; map[string]string query -> nil.
		if t.Key().Kind() == reflect.String && t.Elem().Kind() == reflect.Interface {
			if bodyFill != "" {
				return bodyFill
			}
			return "map[string]any{}"
		}
		return "nil"
	case reflect.Pointer:
		return "nil" // *Options etc.
	case reflect.Slice:
		return "nil"
	case reflect.Interface:
		if t == ctxType {
			return "context.Background()"
		}
		return "nil"
	case reflect.Struct:
		// §5/§4a: a generated-REST operation/command method takes its wire body as
		// a named params struct (`<Recv><Method>Params`) with an `Extras` door. Emit
		// a package-qualified composite literal, funneling the required-body fill
		// through Extras (same wire body the old flat `extras map[string]any` param
		// produced — the mock success path stays realistic).
		lit := t.Name()
		if t.PkgPath() != "" {
			lit = t.PkgPath()[strings.LastIndexByte(t.PkgPath(), '/')+1:] + "." + t.Name()
		}
		if bodyFill != "" {
			return lit + "{Extras: " + bodyFill + "}"
		}
		return lit + "{}"
	case reflect.Int, reflect.Int64, reflect.Int32:
		return "0"
	case reflect.Bool:
		return "false"
	default:
		return "nil"
	}
}

// ---------------------------------------------------------------------------
// 4. Spec body-field fill (RULES §4): a filled map[string]any literal of the
// spec operation's REQUIRED request-body fields. Reuses the same yaml.Node
// schema helpers cmd/generate-rest carries (requiredBodyFields here). Since the
// Go body is map[string]any, an empty map compiles fine; still, we fill required
// fields so the mock's success path is realistic.
// ---------------------------------------------------------------------------

// bodyFillFor returns a filled map[string]any literal for a joined route's spec
// operation required body fields, or "" if the op declares no body (or no
// required fields). Keyed off the winner opID's (spec, operationId).
func bodyFillFor(row joinedRow, specDocs map[string]*yaml.Node, psdk string) string {
	spec := row.spec
	opID := row.opID[strings.IndexByte(row.opID, '.')+1:]
	root := specDocs[spec]
	if root == nil {
		return ""
	}
	// Find the op node by operationId across the spec's paths.
	paths := mapChild(root, "paths")
	if paths == nil {
		return ""
	}
	var opNode *yaml.Node
	for i := 0; i+1 < len(paths.Content); i += 2 {
		body := paths.Content[i+1]
		for _, verb := range []string{"get", "post", "put", "patch", "delete"} {
			op := mapChild(body, verb)
			if op != nil && scalarChild(op, "operationId") == opID {
				opNode = op
			}
		}
	}
	if opNode == nil {
		return ""
	}
	reqBody := mapChild(opNode, "requestBody")
	if reqBody == nil {
		return ""
	}
	content := mapChild(reqBody, "content")
	if content == nil || content.Kind != yaml.MappingNode || len(content.Content) < 2 {
		return ""
	}
	sch := mapChild(content.Content[1], "schema")
	if sch == nil {
		return ""
	}
	schemas := componentsSchemas(root)
	req := requiredFields(schemas, sch)
	if len(req) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("map[string]any{")
	for i, f := range req {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%q: %q", f, "x-1")
	}
	b.WriteString("}")
	return b.String()
}

// componentsSchemas returns the components.schemas node of a spec root.
func componentsSchemas(root *yaml.Node) *yaml.Node {
	return mapChild(mapChild(root, "components"), "schemas")
}

// requiredFields returns the ordered required property names of an object schema,
// flattening allOf/anyOf/oneOf and following $refs. A top-level oneOf/anyOf union
// of distinct request types yields no fill (the reference passes a whole body;
// the mock does not require specific fields), so we only descend allOf here for
// required semantics and return the union of declared required across branches.
func requiredFields(schemas, node *yaml.Node) []string {
	seen := map[string]bool{}
	var order []string
	var walk func(n *yaml.Node)
	walk = func(n *yaml.Node) {
		n = resolveRef(schemas, n)
		if n == nil {
			return
		}
		for _, comb := range []string{"allOf"} {
			if lst := mapChild(n, comb); lst != nil && lst.Kind == yaml.SequenceNode {
				for _, br := range lst.Content {
					walk(br)
				}
			}
		}
		if req := mapChild(n, "required"); req != nil && req.Kind == yaml.SequenceNode {
			for _, r := range req.Content {
				if !seen[r.Value] {
					seen[r.Value] = true
					order = append(order, r.Value)
				}
			}
		}
	}
	walk(node)
	return order
}

// resolveRef follows a $ref one level within components.schemas.
func resolveRef(schemas, node *yaml.Node) *yaml.Node {
	visited := map[string]bool{}
	for node != nil {
		ref := scalarChild(node, "$ref")
		if ref == "" {
			return node
		}
		leaf := ref
		if i := strings.LastIndexByte(ref, '/'); i >= 0 {
			leaf = ref[i+1:]
		}
		if visited[leaf] || schemas == nil {
			return node
		}
		visited[leaf] = true
		node = mapChild(schemas, leaf)
	}
	return node
}

// ---------------------------------------------------------------------------
// YAML helpers (ordered node walk; mirror cmd/generate-rest).
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
// 5. Emit — one pkg/rest/namespaces/<spec>_generated_test.go per spec namespace.
// ---------------------------------------------------------------------------

const fileHeader = `// Code generated by cmd/generate-rest-tests; DO NOT EDIT.
//
// AUTO-GENERATED full-mock REST wire tests for the %q namespace — regenerate:
//   go run ./cmd/generate-rest-tests
//
// Each route the SDK implements (captured from the real client by
// cmd/route-registry, joined to the spec operationId) gets a SUCCESS test (call
// it, assert method + matched_route on the mock journal) and an ERROR test (arm a
// 500, assert *rest.SignalWireRestError with StatusCode 500). The assertion
// oracle is the spec operationId — independent of the resource generator — so
// these catch SDK-vs-contract drift, not a generator self-snapshot.

package namespaces_test

import (
	"context"
	"errors"
	"testing"
%s
	"github.com/signalwire/signalwire-go/pkg/rest"
	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)
`

// namespacesImport is the extra import line emitted when a spec's generated tests
// reference a params-struct composite literal (`namespaces.<...>Params{…}`).
const namespacesImport = "\n\t\"github.com/signalwire/signalwire-go/pkg/rest/namespaces\""

// slug is the resource.method tail of the via, non-alnum→"_", trailing "_"
// trimmed — for deterministic, stable test names.
func slug(via string) string {
	tail := via
	if i := strings.IndexByte(via, '.'); i >= 0 {
		tail = via[i+1:]
	}
	var b strings.Builder
	for _, r := range tail {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

// pascalSpec turns a spec dir name into a PascalCase test-name fragment
// (relay-rest → RelayRest).
func pascalSpec(spec string) string {
	parts := strings.FieldsFunc(spec, func(r rune) bool { return r == '-' || r == '_' })
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return b.String()
}

type rowWithCall struct {
	joinedRow
	call     string
	testName string // Test<Spec>Gen_<slug>
}

// emitSpecFile builds the generated test file source for one spec's rows.
func emitSpecFile(spec string, rows []rowWithCall) string {
	// Emit the test bodies first so we can tell whether any call references a
	// params-struct literal (`namespaces.<...>Params{…}`) and add the import iff so.
	var body strings.Builder
	for _, r := range rows {
		// SUCCESS
		fmt.Fprintf(&body, "func %s(t *testing.T) {\n", r.testName)
		body.WriteString("\tt.Parallel()\n")
		body.WriteString("\tclient, mock := mocktest.New(t)\n")
		body.WriteString("\tif client == nil {\n\t\treturn\n\t}\n")
		body.WriteString("\tmock.Reset(t)\n")
		fmt.Fprintf(&body, "\t_, err := %s\n", r.call)
		body.WriteString("\tif err != nil {\n\t\tt.Fatalf(\"call: %v\", err)\n\t}\n")
		body.WriteString("\tj := mock.Last(t)\n")
		fmt.Fprintf(&body, "\tif j.Method != %q {\n\t\tt.Errorf(\"method = %%q want %s\", j.Method)\n\t}\n", r.method, r.method)
		fmt.Fprintf(&body, "\tif j.MatchedRoute == nil || *j.MatchedRoute != %q {\n\t\tt.Errorf(\"matched_route = %%v want %s\", j.MatchedRoute)\n\t}\n", r.opID, r.opID)
		body.WriteString("}\n\n")

		// ERROR
		fmt.Fprintf(&body, "func %s_Error(t *testing.T) {\n", r.testName)
		body.WriteString("\tt.Parallel()\n")
		body.WriteString("\tclient, mock := mocktest.New(t)\n")
		body.WriteString("\tif client == nil {\n\t\treturn\n\t}\n")
		body.WriteString("\tmock.Reset(t)\n")
		fmt.Fprintf(&body, "\tmock.PushScenario(t, %q, 500, map[string]any{\"error\": \"x\"})\n", r.opID)
		fmt.Fprintf(&body, "\t_, err := %s\n", r.call)
		body.WriteString("\tvar restErr *rest.SignalWireRestError\n")
		body.WriteString("\tif !errors.As(err, &restErr) {\n\t\tt.Fatalf(\"want *SignalWireRestError, got %v\", err)\n\t}\n")
		body.WriteString("\tif restErr.StatusCode != 500 {\n\t\tt.Errorf(\"status = %d want 500\", restErr.StatusCode)\n\t}\n")
		body.WriteString("}\n\n")
	}
	nsImport := ""
	if strings.Contains(body.String(), "namespaces.") {
		nsImport = namespacesImport
	}
	var b strings.Builder
	fmt.Fprintf(&b, fileHeader, spec, nsImport)
	b.WriteString("\n")
	b.WriteString(body.String())
	return b.String()
}

// ---------------------------------------------------------------------------
// Driver.
// ---------------------------------------------------------------------------

// specDirsWithOpenAPI returns the sorted spec dirs under rest-apis that carry an
// openapi.yaml.
func specDirsWithOpenAPI(psdk string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(psdk, "rest-apis"))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(psdk, "rest-apis", e.Name(), "openapi.yaml")); err == nil {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

func findRepoRoot(start string) (string, error) {
	cur := start
	for {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("no go.mod above %s", start)
		}
		cur = parent
	}
}

func resolvePortingSDK(repoRoot string) (string, error) {
	if p := os.Getenv("PORTING_SDK"); p != "" {
		if _, err := os.Stat(filepath.Join(p, "rest-apis")); err == nil {
			return p, nil
		}
	}
	cand := filepath.Join(repoRoot, "..", "porting-sdk")
	if _, err := os.Stat(filepath.Join(cand, "rest-apis")); err == nil {
		return filepath.Abs(cand)
	}
	return "", fmt.Errorf("porting-sdk not found (set $PORTING_SDK or clone adjacent)")
}

func gofmtSrc(src string) ([]byte, error) {
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return nil, fmt.Errorf("gofmt: %w\n---\n%s", err, src)
	}
	return formatted, nil
}

func run() error {
	check := flag.Bool("check", false, "GEN-FRESH: exit non-zero if any generated test file is stale")
	flag.Parse()

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	repoRoot, err := findRepoRoot(cwd)
	if err != nil {
		return err
	}
	psdk, err := resolvePortingSDK(repoRoot)
	if err != nil {
		if *check {
			return fmt.Errorf("generate-rest-tests --check: %w", err)
		}
		fmt.Fprintf(os.Stderr, "generate-rest-tests: %v — skipping (committed files kept)\n", err)
		return nil
	}

	specDirs, err := specDirsWithOpenAPI(psdk)
	if err != nil {
		return err
	}

	// Load each spec doc once (for body-field fill).
	specDocs := map[string]*yaml.Node{}
	for _, spec := range specDirs {
		raw, err := os.ReadFile(filepath.Join(psdk, "rest-apis", spec, "openapi.yaml"))
		if err != nil {
			return err
		}
		var doc yaml.Node
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			return err
		}
		specDocs[spec] = rootOf(&doc)
	}

	routes, err := loadRegistry(repoRoot)
	if err != nil {
		return err
	}
	rows, err := join(routes, psdk, specDirs)
	if err != nil {
		return err
	}
	index, err := buildCallIndex()
	if err != nil {
		return err
	}

	// Synthesize calls; collect uncovered (joined but not invokable) as findings.
	// Group by spec. Dedup test names (rare via collisions across wire-collapse).
	bySpec := map[string][]rowWithCall{}
	var uncovered []string
	seenName := map[string]bool{}
	nRoutesCovered := map[string]bool{} // via -> covered (for the 209 count)
	for _, row := range rows {
		fill := bodyFillFor(row, specDocs, psdk)
		call, ok := synthCall(row, index, fill)
		if !ok {
			uncovered = append(uncovered, fmt.Sprintf("%s (%s %s)", row.via, row.method, row.path))
			continue
		}
		name := "Test" + pascalSpec(row.spec) + "Gen_" + slug(row.via)
		// Ensure uniqueness (append a disambiguator on collision).
		base := name
		k := 2
		for seenName[name] {
			name = fmt.Sprintf("%s_%d", base, k)
			k++
		}
		seenName[name] = true
		bySpec[row.spec] = append(bySpec[row.spec], rowWithCall{joinedRow: row, call: call, testName: name})
		nRoutesCovered[row.via] = true
	}

	// Deterministic: sort specs; sort rows within a spec by (via+method).
	var specs []string
	for s := range bySpec {
		specs = append(specs, s)
	}
	sort.Strings(specs)

	nsDir := filepath.Join(repoRoot, "pkg", "rest", "namespaces")
	var stale []string
	totalTests := 0
	nFiles := 0
	for _, spec := range specs {
		rws := bySpec[spec]
		sort.SliceStable(rws, func(i, j int) bool {
			return rws[i].via+rws[i].method < rws[j].via+rws[j].method
		})
		src := emitSpecFile(spec, rws)
		formatted, err := gofmtSrc(src)
		if err != nil {
			return fmt.Errorf("%s: %w", spec, err)
		}
		outPath := filepath.Join(nsDir, strings.ReplaceAll(spec, "-", "_")+"_generated_test.go")
		if *check {
			existing, err := os.ReadFile(outPath)
			if err != nil || !bytes.Equal(existing, formatted) {
				stale = append(stale, outPath)
			}
		} else {
			if err := os.WriteFile(outPath, formatted, 0o644); err != nil {
				return err
			}
			fmt.Printf("generated %s (%d routes, %d tests)\n", outPath, len(rws), len(rws)*2)
		}
		nFiles++
		totalTests += len(rws) * 2
	}

	if len(uncovered) > 0 {
		fmt.Fprintf(os.Stderr, "\nUNCOVERED (%d joined route(s) with no reflectable via method):\n", len(uncovered))
		for _, u := range uncovered {
			fmt.Fprintf(os.Stderr, "  - %s\n", u)
		}
	}

	if *check {
		if len(stale) > 0 {
			fmt.Fprintf(os.Stderr, "\nGEN-FRESH FAIL: %d generated test file(s) stale — run `go run ./cmd/generate-rest-tests` and commit:\n", len(stale))
			for _, f := range stale {
				fmt.Fprintf(os.Stderr, "  - %s\n", f)
			}
			return fmt.Errorf("stale generated test files")
		}
		fmt.Printf("GEN-FRESH: %d generated REST test file(s) up to date (%d tests).\n", nFiles, totalTests)
		return nil
	}
	fmt.Printf("total: %d files, %d generated wire tests across %d namespaces, %d/%d registry routes covered\n",
		nFiles, totalTests, len(specs), len(nRoutesCovered), len(routes))
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
