# DOC_AUDIT_IGNORE

Every identifier listed here is intentionally skipped by
`porting-sdk/scripts/audit_docs.py`. Each line follows
`<name>: <rationale>` — the rationale explains *why* the identifier is
legitimately referenced in docs or examples without appearing in the Go
port surface.

Grouped by category. Keep the rationale concise.

## Go standard library — `fmt`

Printf: fmt.Printf formatted print to stdout
Println: fmt.Println line print to stdout
Sprintf: fmt.Sprintf formatted string
Fprintf: fmt.Fprintf formatted writer output
Fprintln: fmt.Fprintln writer line output

## Go standard library — `log`

Fatal: log.Fatal terminating error
Fatalf: log.Fatalf formatted terminating error

## Go standard library — `os` / `os/signal`

Getenv: os.Getenv environment variable lookup
Exit: os.Exit process termination

## Go standard library — `strconv`

Atoi: strconv.Atoi string-to-int conversion
ParseFloat: strconv.ParseFloat string-to-float conversion

## Go standard library — `strings`

Split: strings.Split token splitter
ToLower: strings.ToLower case conversion
ToUpper: strings.ToUpper case conversion

## Go standard library — `encoding/json`

MarshalIndent: json.MarshalIndent pretty JSON serialisation

## Go standard library — `time`

Now: time.Now current-time getter
Duration: time.Duration constructor (e.g. time.Duration(n)*time.Second)

## Go standard library — `context` / `os/signal`

Background: context.Background root context
WithTimeout: context.WithTimeout cancellable context
NotifyContext: signal.NotifyContext signal-aware context

## Go standard library — `math/rand`

Intn: rand.Intn random integer in range

## Go standard library — `sync`

Add: sync.WaitGroup.Add counter increment
Done: sync.WaitGroup.Done counter decrement

## Go standard library — `errors`

As: errors.As typed unwrap
Is: errors.Is sentinel comparison (docs/client-reference.md: errors.Is(err, relay.ErrDialTimeout)) — Go stdlib, sibling of errors.As

## Comment text — illustrative references inside `//` comments

Names that appear only inside a `//` code comment (not executable API surface a
reader is told to call). audit_docs flags the bare identifier; the comment text
merely mentions it.

Publish: illustrative PubSub.Publish reference inside a comment in examples/rest_demo/main.go

## Go type-alias idiom — scalar/func aliases the surface enumerator does not record

The surface enumerator records structs / marked-enums / methods / functions
only, NOT scalar or func named-type aliases. The names below are REAL exported
Go types (verified in source) that are consequently invisible to audit_docs —
this is an enumerator limitation, not an absent symbol or a doc bug.

ToolHandler: real exported func type `type ToolHandler func(...) *FunctionResult` (pkg/swaig/handler.go, pkg/agent/agent.go); referenced in a comment in examples/skills_demo/main.go
Uuid: real exported scalar type `type Uuid string` (pkg/rest/namespaces/relay_rest_types_generated.go); it is the declared type of id fields such as CallingNamespaceUpdateParams.Id, so docs write namespaces.Uuid(callID) to convert a string variable

## Anonymous-struct field name

fn: anonymous-struct field name (op.fn()) used to iterate a table-driven operation list in rest/examples/rest_calling_play_and_record.go

## Python standard library referenced from legacy Python code blocks

The top-level `docs/*.md` files carry over Python code blocks from the
upstream Python SDK while the Go-native rewrite is in progress. These
references are Python stdlib methods that appear inside those blocks.

## Comment text — Python-name references inside `//` comments

The Go port implements these under Go-idiomatic CamelCase identifiers (which
the audit resolves against `port_surface_go.json`). Each snake_case name below
appears only inside a `//` comment that documents the Python-reference
equivalent of the Go method invoked on the very next line — not an API surface
claim. audit_docs flags the bare snake_case identifier in the comment text.

register_routing_callback: Python SWMLService.register_routing_callback name in a `//` comment above the real Service.RegisterRoutingCallback call (examples/dynamic_swml_service/main.go, examples/swml_service_routing/main.go) — the Go method is in port_surface_go.json

## Go stdlib referenced by harness/example code

The audit harnesses (relay_audit_harness, skills_audit_harness,
rest_audit_harness) and other examples make heavy use of Go stdlib
methods that aren't part of the SignalWire surface. Each line below
documents the stdlib origin so a reviewer can spot a real symbol typo.

After: time.After channel-based timeout
Close: io.Closer.Close (used on http.Response.Body, ws.Conn, etc.)
Do: http.Client.Do
Encode: encoding/json.Encoder.Encode
Grow: strings.Builder.Grow capacity hint
HasPrefix: strings.HasPrefix
Index: strings.Index
IndexByte: strings.IndexByte
Load: sync/atomic.Bool.Load / atomic.Value.Load
NewEncoder: encoding/json.NewEncoder
NewRequest: net/http.NewRequest
ReadAll: io.ReadAll
SetEscapeHTML: encoding/json.Encoder.SetEscapeHTML
Sprint: fmt.Sprint
Store: sync/atomic.Bool.Store / atomic.Value.Store
TrimPrefix: strings.TrimPrefix
TrimRight: strings.TrimRight
TrimSpace: strings.TrimSpace
Unmarshal: encoding/json.Unmarshal
WriteByte: strings.Builder.WriteByte / bytes.Buffer.WriteByte
WriteString: strings.Builder.WriteString / bytes.Buffer.WriteString

## Go SDK constructors / methods (Python-name-mapped surface gap)

The Go port's `port_surface.json` is Python-shaped (Go struct methods are
mapped onto Python class methods so diff_port_surface can compare).
Go-idiomatic constructors (`NewXxx`) map to Python's `__init__`, which
the audit's CamelCase translation doesn't cover. The names below are
real Go SDK exports, listed to acknowledge the audit's translation
limitation (NOT to hide undefined symbols).


## Go With* options (Python uses keyword args; audit can't map kwargs)

The Python SDK uses `__init__(name=..., route=...)` keyword args. Go
uses `New(WithName(...), WithRoute(...))` functional options. The
audit can't bridge that idiom — every functional option below is a
real Go SDK export (one per Python kwarg) but doesn't appear under a
Python class name.


## Audit harness / example helper methods

These are Go-SDK methods added during this port's audit work. They
exist in code but the surface enumerator chose not to map them to
Python names (no Python equivalent) — the audit treats them as
unresolved. Each is a legitimate Go SDK export documented in code.


## Other Go-idiomatic surface


## Go standard library / doc-example references (2026-07-06 doc conversion)

Abs: math.Abs — Go stdlib
Contains: strings.Contains — Go stdlib
Decode: json.Decoder.Decode — Go stdlib
NewDecoder: json.NewDecoder — Go stdlib
NewReader: strings/bytes.NewReader — Go stdlib
Errorf: fmt.Errorf — Go stdlib
Marshal: json.Marshal — Go stdlib
Join: strings.Join — Go stdlib
Parse: url.Parse / template.Parse — Go stdlib
Print: fmt.Print — Go stdlib
Handle: http.ServeMux.Handle — Go stdlib
ListenAndServe: http.ListenAndServe — Go stdlib
NewServeMux: http.NewServeMux — Go stdlib
StripPrefix: http.StripPrefix — Go stdlib
SetBasicAuth: http.Request.SetBasicAuth — Go stdlib
Lock: sync.Mutex.Lock — Go stdlib
Unlock: sync.Mutex.Unlock — Go stdlib
RLock: sync.RWMutex.RLock — Go stdlib
RUnlock: sync.RWMutex.RUnlock — Go stdlib
Seconds: time.Duration.Seconds — Go stdlib
Since: time.Since — Go stdlib
Setenv: os.Setenv — Go stdlib
Stat: os.Stat — Go stdlib
go: Go `go` keyword highlighted in a code block — false-positive, not an identifier
init: Go language built-in package-init function referenced in a comment ("// Each imported package registers its skill via init().") in docs/third_party_skills.md — not an SDK symbol
SkillName: skills.SkillName typed-string conversion (e.g. skills.SkillName("weather")) — real port type used in AddSkill examples
NewCustomerSupportAgent: user-defined example agent constructor in docs/agent_guide.md
newMyAgent: doc-local example constructor defined and called within the same snippet (func newMyAgent() ... / func main() { _, _ = newMyAgent() }) in docs/skills_parameter_schema.md — not an SDK symbol
registerTools: user-defined method on an illustrative custom-prefab example in docs/architecture.md
testAPIConnection: user-defined helper method on an illustrative skill example in docs/third_party_skills.md
