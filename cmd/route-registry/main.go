// Command route-registry enumerates the REST routes the Go SDK ACTUALLY
// IMPLEMENTS — "Set B" for the cross-port SPEC-PARITY gate.
//
// This is the routes the live RestClient actually dispatches, captured from the
// REAL code path — not parsed from source (an AST scraper would have to
// re-implement the CrudResource / base-path machinery and would drift) and not
// read from the test journal (which only sees routes that happen to be tested,
// the exact blind spot the gate closes).
//
// How it works: construct the real RestClient and point it (via SetBaseURL) at
// an in-process httptest server — a RECORDING TRANSPORT — that captures
// (req.Method, req.URL.Path) for every request and returns a stub 200 `{}`,
// doing no network I/O. Every route — CRUD-base, custom CreateSubscriberToken,
// deprecation-wrapped Create, the Set* phone-number helpers, etc. — funnels
// through HTTPClient.doRequest → http.Client.Do → that server. We then use
// reflection to walk every namespace on the client, every public method on
// every sub-resource, and invoke each with sentinel arguments synthesised by
// parameter type (string path params become the literal sentinel, normalised
// back to {id}; map bodies/params become empty maps; *Options become nil;
// context.Context becomes Background). The captured path is thus a template
// comparable to the spec's path_template.
//
// A method that cannot be invoked is NOT silently skipped — a dropped method is
// a route missing from Set B, which would turn a real divergence into a false
// "Go matches the spec" pass. Methods that genuinely do not map to a single
// canonical route (client-side path helpers, multi-route convenience wrappers)
// must be listed explicitly in registrySkip with a reason; everything else that
// fails to invoke, or invokes but issues no HTTP request, is a hard ERROR
// (non-zero exit + recorded in "errors"), mirroring python_route_registry.py
// and route-registry.ts.
//
// Output: JSON {"routes":[{"method","path_template","via"}],"skipped":[...],
// "errors":[...]} on stdout. Exit 1 if any uninvokable, un-skip-listed method
// (Set B incomplete). ONLY the JSON is written to stdout; diagnostics go to
// stderr so the shared diff can consume stdout directly.
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/route-registry
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/signalwire/signalwire-go/pkg/rest"
)

// sentinel stands in for any path parameter (resource id, sid, etc.). It is one
// path segment with no '/', and is normalised back to "{id}" so Set B path
// templates line up with the spec's. The project id passed to the client also
// becomes a path segment (compat's {AccountSid}); we pass the same sentinel so
// it too normalises to {id} and the spec matcher resolves it from config.
const sentinel = "__ID__"

// registrySkip lists methods that do NOT map to a single canonical REST route,
// keyed by "<Namespace>.<Resource>.<Method>" or a "<Namespace>.<Resource>.*"
// wildcard. EVERY entry needs a reason; a method that merely fails to invoke or
// issues no HTTP request is an ERROR, not an implicit skip — add it here (with
// justification) or fix the harness so it invokes.
var registrySkip = map[string]string{
	// cXML applications expose the CrudResource surface for symmetry but Create
	// is intentionally unsupported (returns an error by design) — there is no
	// POST /cxml_applications canonical route, so it is not in Set B. Mirrors
	// python's fabric.cxml_applications.create skip.
	"Fabric.CXMLApplications.Create": "no create route — returns an error by design (cXML apps cannot be created via API)",

	// Path is the CrudResource/Resource path-builder helper, not a route — it
	// returns a string and issues no HTTP request. It is exported because Go has
	// no package-private-but-cross-file visibility; every resource inherits it.
	"*.Path": "path-builder helper, not a route (issues no HTTP request)",
}

func skipReason(key string) (string, bool) {
	if r, ok := registrySkip[key]; ok {
		return r, true
	}
	// "<Ns>.<Res>.*" wildcard.
	if i := strings.LastIndex(key, "."); i >= 0 {
		if r, ok := registrySkip[key[:i]+".*"]; ok {
			return r, true
		}
	}
	// "*.<Method>" wildcard (a helper inherited by every resource).
	if i := strings.LastIndex(key, "."); i >= 0 {
		if r, ok := registrySkip["*."+key[i+1:]]; ok {
			return r, true
		}
	}
	return "", false
}

type routeRec struct {
	Method       string   `json:"method"`
	PathTemplate string   `json:"path_template"`
	Via          []string `json:"via"`
}

type skipRec struct {
	Key    string `json:"key"`
	Reason string `json:"reason"`
}

type errRec struct {
	Key   string `json:"key"`
	Error string `json:"error"`
}

// recorder captures (method, path) for each HTTP request the SDK dispatches.
type recorder struct {
	mu    sync.Mutex
	calls []struct{ method, path string }
}

func (r *recorder) reset() {
	r.mu.Lock()
	r.calls = r.calls[:0]
	r.mu.Unlock()
}

func (r *recorder) snapshot() []struct{ method, path string } {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]struct{ method, path string }, len(r.calls))
	copy(out, r.calls)
	return out
}

func main() {
	// Body lives in run() so deferred cleanup (srv.Close) runs on every exit
	// path — os.Exit in main would skip defers (gocritic exitAfterDefer).
	os.Exit(run())
}

func run() int {
	rec := &recorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rec.mu.Lock()
		rec.calls = append(rec.calls, struct{ method, path string }{req.Method, req.URL.Path})
		rec.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	// Build the real client with throwaway creds; the project id ("__ID__")
	// becomes the compat {AccountSid} path segment and normalises to {id}.
	client, err := rest.NewRestClient(sentinel, "t", "example.signalwire.com")
	if err != nil {
		fmt.Fprintf(os.Stderr, "route-registry: NewRestClient failed: %v\n", err)
		return 2
	}
	client.SetBaseURL(srv.URL)

	routes := map[string]*routeRec{} // keyed by "METHOD PATH"
	var skipped []skipRec
	var errs []errRec

	handleResource := func(nsName, resName string, resVal reflect.Value) {
		methods := publicMethods(resVal)
		for _, m := range methods {
			key := nsName + "." + resName + "." + m.name
			if reason, ok := skipReason(key); ok {
				skipped = append(skipped, skipRec{Key: key, Reason: reason})
				continue
			}
			rec.reset()
			if invErr := invoke(m.fn); invErr != "" {
				errs = append(errs, errRec{Key: key, Error: invErr})
				continue
			}
			calls := rec.snapshot()
			if len(calls) == 0 {
				errs = append(errs, errRec{
					Key: key,
					Error: "invoked but issued no HTTP request (client-side helper? " +
						"add to registrySkip with a reason)",
				})
				continue
			}
			for _, c := range calls {
				path := strings.ReplaceAll(c.path, sentinel, "{id}")
				rk := c.method + " " + path
				if ex, ok := routes[rk]; ok {
					ex.Via = append(ex.Via, key)
				} else {
					routes[rk] = &routeRec{Method: c.method, PathTemplate: path, Via: []string{key}}
				}
			}
		}
	}

	// Walk the client's namespace fields. The generated resource tree lives in
	// an unexported anonymous embed (_GeneratedResourceTree, rest_tree_generated.go)
	// whose fields promote onto RestClient; descend into that embed so its
	// namespace fields are enumerated. A plain exported namespace field on the
	// client itself (hand-wired, if any) is walked directly.
	handleNamespaceField := func(f reflect.StructField, nsVal reflect.Value) {
		if !isResourceLike(nsVal) {
			return
		}
		nsName := f.Name
		// A namespace may itself be a flat resource with route methods
		// (e.g. Calling, PhoneNumbers).
		if len(publicMethods(nsVal)) > 0 {
			handleResource(nsName, nsName, nsVal)
		}
		// …and/or a container of sub-resource fields.
		for _, sub := range subResources(nsVal) {
			handleResource(nsName, sub.name, sub.val)
		}
	}
	var walkNamespaceFields func(sv reflect.Value)
	walkNamespaceFields = func(sv reflect.Value) {
		st := sv.Type()
		for i := range st.NumField() {
			f := st.Field(i)
			// Descend into the embedded resource-tree struct (anonymous, may be
			// unexported) so its promoted namespace fields are reached.
			if f.Anonymous {
				fv := sv.Field(i)
				if fv.Kind() == reflect.Struct {
					walkNamespaceFields(fv)
				} else if fv.Kind() == reflect.Pointer && !fv.IsNil() && fv.Elem().Kind() == reflect.Struct {
					walkNamespaceFields(fv.Elem())
				}
				continue
			}
			if !f.IsExported() {
				continue
			}
			handleNamespaceField(f, sv.Field(i))
		}
	}
	walkNamespaceFields(reflect.ValueOf(client).Elem())

	// Sort routes deterministically by (path, method).
	out := make([]*routeRec, 0, len(routes))
	for _, r := range routes {
		sort.Strings(r.Via)
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PathTemplate != out[j].PathTemplate {
			return out[i].PathTemplate < out[j].PathTemplate
		}
		return out[i].Method < out[j].Method
	})
	sort.Slice(skipped, func(i, j int) bool { return skipped[i].Key < skipped[j].Key })
	sort.Slice(errs, func(i, j int) bool { return errs[i].Key < errs[j].Key })

	payload := map[string]any{
		"routes":  out,
		"skipped": skipped,
		"errors":  errs,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		fmt.Fprintf(os.Stderr, "route-registry: encode failed: %v\n", err)
		return 2
	}

	if len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "route-registry: %d uninvokable/no-request method(s) (Set B incomplete)\n", len(errs))
		return 1
	}
	return 0
}

type namedMethod struct {
	name string
	fn   reflect.Value
}

type namedValue struct {
	name string
	val  reflect.Value
}

// isResourceLike reports whether v is a non-nil pointer to a struct (a
// namespace or resource instance). Plain data fields (strings, the unexported
// http handle) are not walked.
func isResourceLike(v reflect.Value) bool {
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return false
	}
	return v.Elem().Kind() == reflect.Struct
}

// publicMethods returns the exported methods on v's type (pointer receiver
// methods included, since v is a pointer). Methods are sorted by name.
func publicMethods(v reflect.Value) []namedMethod {
	t := v.Type()
	out := make([]namedMethod, 0, t.NumMethod())
	for i := range t.NumMethod() {
		m := t.Method(i)
		if !m.IsExported() {
			continue
		}
		out = append(out, namedMethod{name: m.Name, fn: v.Method(i)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// subResources returns the exported pointer-to-struct fields of a namespace
// (its sub-resources). Anonymous/embedded fields are skipped — their methods
// are already promoted onto the namespace itself and walked via publicMethods.
func subResources(ns reflect.Value) []namedValue {
	e := ns.Elem()
	t := e.Type()
	var out []namedValue
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() || f.Anonymous {
			continue
		}
		fv := e.Field(i)
		if isResourceLike(fv) {
			out = append(out, namedValue{name: f.Name, val: fv})
		}
	}
	return out
}

// invoke calls fn with sentinel arguments synthesised by parameter type. It
// returns "" on success or a description of why the method could not be
// invoked. A panic during the call (e.g. a nil-deref the harness can't avoid)
// is recovered and reported rather than crashing the whole enumeration.
func invoke(fn reflect.Value) (errMsg string) {
	t := fn.Type()
	defer func() {
		if r := recover(); r != nil {
			errMsg = fmt.Sprintf("panic: %v", r)
		}
	}()

	args := make([]reflect.Value, 0, t.NumIn())
	for i := range t.NumIn() {
		// For a variadic final parameter, supply zero extra args.
		if t.IsVariadic() && i == t.NumIn()-1 {
			break
		}
		av, ok := sentinelFor(t.In(i))
		if !ok {
			return fmt.Sprintf("unhandled parameter type %s at position %d "+
				"(extend sentinelFor or add to registrySkip with a reason)", t.In(i), i)
		}
		args = append(args, av)
	}
	fn.Call(args)
	return ""
}

var ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()

// sentinelFor returns a sentinel reflect.Value for a parameter type. string ->
// the path sentinel; map -> an empty map; pointer (e.g. *Options) -> a typed
// nil; context.Context -> Background; slice -> nil slice. Anything else is
// unhandled (reported by invoke so it can't be silently dropped).
func sentinelFor(t reflect.Type) (reflect.Value, bool) {
	switch t.Kind() {
	case reflect.String:
		return reflect.ValueOf(sentinel), true
	case reflect.Map:
		// Empty, non-nil map of the right type (map[string]string / map[string]any).
		return reflect.MakeMap(t), true
	case reflect.Pointer:
		// Typed nil pointer — the Set* helpers accept nil *Options.
		return reflect.Zero(t), true
	case reflect.Slice:
		return reflect.Zero(t), true
	case reflect.Interface:
		if t == ctxType {
			return reflect.ValueOf(context.Background()), true
		}
		// Other interfaces: a typed nil interface value.
		return reflect.Zero(t), true
	case reflect.Struct:
		// §5/§4a: a generated-REST operation/command method takes its wire body as
		// a named params struct (`<Recv><Method>Params`). A zero-valued struct is a
		// fine sentinel — every field is nil/empty, so the method POSTs an empty
		// body (only the discriminator for command-dispatch), which is all the route
		// capture needs (it records method + path, not the body shape).
		return reflect.Zero(t), true
	case reflect.Int, reflect.Int64, reflect.Int32:
		return reflect.Zero(t), true
	case reflect.Bool:
		return reflect.Zero(t), true
	default:
		return reflect.Value{}, false
	}
}
