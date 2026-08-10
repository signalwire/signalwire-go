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
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	surfacepkg "github.com/signalwire/signalwire-go/v3/internal/surface"
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
	"signalwire.rest._base.ReadResource.paginate":                                     true, // def paginate(self, **params: Any) -> PaginatedIterator
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
//
// FunctionResult.__init__ is the same shape on the CONSTRUCTION path: the
// reference is `FunctionResult(response: str | None = None, post_process: bool =
// False)` and coerces `None` to `""`
// (`self.response = response if response is not None else ""`), which is exactly
// the zero value Go produces for an omitted `response ...string`. Go's
// NewFunctionResult reads only `response[0]`, i.e. "zero or one", never a list.
var optionalTailVariadicMethods = map[string]bool{
	"signalwire.relay.call.PlayAction.pause":                  true,
	"signalwire.relay.call.RecordAction.pause":                true,
	"signalwire.relay.call.CollectAction.pause":               true,
	"signalwire.core.function_result.FunctionResult.__init__": true,
}

// optionalTailVariadicComposite is the COMPOSITE-element analog of the table
// above: reference methods whose final parameter is a single optional
// non-scalar (`headers: dict[str, str] | None = None`) that Go spells as a
// trailing variadic of that composite (call with 0 or 1 argument). The
// optionalScalarVariadicElemTypes fold deliberately excludes composites,
// because for most of them a variadic genuinely means "zero or more" — so the
// composite cases are opted in ONE AT A TIME here, each verified against both
// the reference signature and the Go body.
//
// The raw translation records `list<optional<dict<string,string>>>`
// required:false, which mismatches the reference's `optional<dict<string,string>>`;
// this reclassifies it to the ELEMENT type so the port compares EQUAL.
//
// Verified: signalwire/rest/_base.py SignalWireRestError.__init__(..., headers:
// dict[str, str] | None = None); Go NewSignalWireRestError reads only
// `headers[0]` and treats an empty variadic as nil (client.go), i.e. exactly
// "zero or one", never a list.
var optionalTailVariadicComposite = map[string]bool{
	"signalwire.rest._base.SignalWireRestError.__init__": true,
}

// optionalScalarVariadicElemTypes are the element types for which a TRAILING
// variadic is the Go idiom for "an optional scalar whose reference default is
// non-zero" rather than a genuine multi-argument list. Bools and numerics only:
// their Go zero value is itself a meaningful argument (false / 0), so a plain
// parameter cannot express absence, which is exactly the condition that forces
// the variadic. Strings are excluded — the pause-control table above handles the
// one string case, whose reference type is `str | None` rather than a bare `str`
// — as are composites, where a variadic really does mean "zero or more".
var optionalScalarVariadicElemTypes = map[string]bool{
	"bool":    true,
	"int":     true,
	"int8":    true,
	"int16":   true,
	"int32":   true,
	"int64":   true,
	"uint":    true,
	"uint8":   true,
	"uint16":  true,
	"uint32":  true,
	"uint64":  true,
	"float32": true,
	"float64": true,
}

// optionalScalarVariadicElem returns the element type of a `...T` whose T is one
// of the optional-scalar element types above, and whether it qualified.
func optionalScalarVariadicElem(typeStr string) (string, bool) {
	elem := strings.TrimPrefix(typeStr, "...")
	return elem, optionalScalarVariadicElemTypes[elem]
}

// optionalRequestOptionsTailMethods lists the fully-qualified Python reference
// methods whose LAST param is the optional `request_options: RequestOptions |
// None = None`, which the Go constructors spell as a trailing variadic
// `opts ...*RequestOptions` (so an existing 3-arg call compiles unchanged while
// a caller can pass a client-default envelope). Reclassify that `...*RequestOptions`
// tail to the reference's optional RequestOptions param so the ctor compares
// EQUAL (idiom reconciled in the enumerator, not an omission). Scoped to these
// two ctors + a `...*RequestOptions` tail.
var optionalRequestOptionsTailMethods = map[string]bool{
	"signalwire.rest._base.HttpClient.__init__":  true,
	"signalwire.rest.client.RestClient.__init__": true,
	// The Paginator ctor (NewPaginator → PaginatedIterator.__init__): the
	// reference records request_options as a plain POSITIONAL __init__ param
	// (`__init__(self, http, path, params, data_key, request_options=None)`), not
	// a keyword-only one — so its trailing `opts ...*RequestOptions` reconciles to
	// the positional class param via the ctor rule, NOT the keyword-only rule the
	// generated verbs use.
	"signalwire.rest._pagination.PaginatedIterator.__init__": true,
}

// optionsStructUnfoldMethods maps a fully-qualified Python reference method whose
// Go spelling takes a single named options STRUCT (`opts <Struct>`) to the SHORT
// name of that struct. The idiom-convergence pass (plan 6.2-go) collapsed the
// flat 7-positional-pointer SWML Service.Play / 6-positional Service.AI signatures
// into named options structs (swml.PlayOptions / swml.AIOptions) so a caller reads
// `svc.Play(swml.PlayOptions{URL: &u})` instead of `svc.Play(&u, nil, nil, nil,
// nil, nil, nil)`. That is a pure call-site reshape: the enumerator UNFOLDS the
// struct's fields back into the flat keyword param set the Python oracle records,
// so port_signatures.json stays byte-identical (drift 0). The struct fields must
// be in field order and 1:1 with the reference keyword params (an `Extra`/`Extras`
// field folds to the reference's `**kwargs` var_keyword tail, matching the old
// flat `extra map[string]any`). Recorded from the swml package like the REST
// params structs, then unfolded in toCanonicalSignature.
var optionsStructUnfoldMethods = map[string]string{
	"signalwire.core.swml_builder.SWMLBuilder.play":                "PlayOptions",
	"signalwire.core.swml_builder.SWMLBuilder.ai":                  "AIOptions",
	"signalwire.core.function_result.FunctionResult.connect":       "ConnectOptions",
	"signalwire.core.function_result.FunctionResult.wait_for_user": "WaitForUserOptions",
}

// aiChatMethodSigs SPLICES the canonical signature for the AIChatClient turn
// methods whose Go idiom (a leading ctx context.Context + a per-call options
// struct / functional-options constructor) collapses several Python keyword
// arguments into one Go param. Rather than teach the generic unfold every field's
// optionality + oracle order, these four methods carry their exact reference
// param list here (the "splice the canonical oracle signature" fold dotnet used):
// the wire behaviour is identical (the Go client sends the same JSON-RPC params),
// only the CALL SHAPE differs, so recording the reference-shaped signature keeps
// port_signatures.json byte-identical to the oracle (drift 0). Keyed by the fully
// qualified Python method name. When present, it REPLACES the AST-derived signature.
//
// chat/create_conversation carry the full reference param set (role, config_url,
// user_metadata, timeout, reinit / config_url, user_message, timeout, user_metadata,
// reinit) — the Go client's ChatOptions/CreateOptions fields — in reference order.
// summarize's reference tail is `**sampling: Any` (a var_keyword bag the oracle
// strips per porting-sdk #58); the Go SummarizeOptions models the same open sampling
// set as typed fields, so the splice records only summary_prompt plus the stripped
// **kwargs tail, matching what the oracle records.
var aiChatMethodSigs = map[string]canonicalSignature{
	"signalwire.ai_chat.client.AIChatClient.__init__": {
		Params: []canonicalParam{
			{Name: "self", Kind: "self"},
			{Name: "project", Type: "optional<string>", Required: boolPtr(false), Default: json.RawMessage("null")},
			{Name: "token", Type: "optional<string>", Required: boolPtr(false), Default: json.RawMessage("null")},
			{Name: "space", Type: "optional<string>", Required: boolPtr(false), Default: json.RawMessage("null")},
			{Name: "url", Type: "optional<string>", Required: boolPtr(false), Default: json.RawMessage("null")},
			// The oracle dropped Python's `session: aiohttp.ClientSession | None`
			// (a Python-only DI seam) as of porting-sdk ai-chat-client @ f6efa9b, so
			// __init__ folds naturally to (project, token, space, url). Go's
			// WithHTTPClient(*http.Client) functional option remains as an idiomatic
			// transport-injection extra, invisible to this reference-shaped signature.
		},
		Returns: "void",
	},
	"signalwire.ai_chat.client.AIChatClient.create_conversation": {
		Params: []canonicalParam{
			{Name: "self", Kind: "self"},
			{Name: "conversation_id", Type: "string", Required: boolPtr(true)},
			{Name: "config_url", Type: "string", Required: boolPtr(true)},
			{Name: "user_message", Type: "optional<string>", Required: boolPtr(false), Default: json.RawMessage("null")},
			{Name: "timeout", Type: "optional<int>", Required: boolPtr(false), Default: json.RawMessage("null")},
			{Name: "user_metadata", Type: "optional<dict<string,any>>", Required: boolPtr(false), Default: json.RawMessage("null")},
			{Name: "reinit", Type: "bool", Required: boolPtr(false), Default: json.RawMessage("false")},
		},
		Returns: "class:signalwire.ai_chat.client.ConversationInfo",
	},
	"signalwire.ai_chat.client.AIChatClient.chat": {
		Params: []canonicalParam{
			{Name: "self", Kind: "self"},
			{Name: "conversation_id", Type: "string", Required: boolPtr(true)},
			{Name: "message", Type: "string", Required: boolPtr(true)},
			{Name: "role", Type: "string", Required: boolPtr(false), Default: json.RawMessage("\"user\"")},
			{Name: "config_url", Type: "optional<string>", Required: boolPtr(false), Default: json.RawMessage("null")},
			{Name: "user_metadata", Type: "optional<dict<string,any>>", Required: boolPtr(false), Default: json.RawMessage("null")},
			// timeout/reinit use the oracle's canonical int/bool spelling (matching
			// create_conversation); porting-sdk @ 3e24867 fixed the earlier
			// integer/boolean typo so chat and create_conversation now agree.
			{Name: "timeout", Type: "optional<int>", Required: boolPtr(false), Default: json.RawMessage("null")},
			{Name: "reinit", Type: "bool", Required: boolPtr(false), Default: json.RawMessage("false")},
		},
		Returns: "class:signalwire.ai_chat.client.ChatResponse",
	},
	"signalwire.ai_chat.client.AIChatClient.summarize": {
		Params: []canonicalParam{
			{Name: "self", Kind: "self"},
			{Name: "conversation_id", Type: "string", Required: boolPtr(true)},
			{Name: "summary_prompt", Type: "optional<string>", Required: boolPtr(false), Default: json.RawMessage("null")},
			{Name: "kwargs", Kind: "var_keyword", Type: "any", Required: boolPtr(false), Default: json.RawMessage("{}")},
		},
		Returns: "string",
	},
}

// aiChatCtorSigs SYNTHESIZES the __init__ signature for the AI-Chat data/error
// structs, which the reference records as generated-dataclass / exception
// auto-constructors. Go builds each via a composite struct literal (no exported
// NewX factory), so the reference's __init__ has no directly-corresponding Go
// method — it is projected here from the struct's public fields, in declaration
// order, matching the reference dataclass field order. Keyed by the fully qualified
// Python class name; emitted as that class's __init__.
var aiChatCtorSigs = map[string]canonicalSignature{
	"signalwire.ai_chat.client.AIChatError": {
		Params: []canonicalParam{
			{Name: "self", Kind: "self"},
			{Name: "code", Type: "optional<int>", Required: boolPtr(true)},
			{Name: "message", Type: "string", Required: boolPtr(true)},
		},
		Returns: "void",
	},
	"signalwire.ai_chat.client.ConversationInfo": {
		Params: []canonicalParam{
			{Name: "self", Kind: "self"},
			{Name: "id", Type: "string", Required: boolPtr(true)},
			{Name: "status", Type: "string", Required: boolPtr(true)},
			{Name: "initial_message", Type: "optional<string>", Required: boolPtr(false), Default: json.RawMessage("null")},
		},
		Returns: "void",
	},
	"signalwire.ai_chat.client.ChatResponse": {
		Params: []canonicalParam{
			{Name: "self", Kind: "self"},
			{Name: "text", Type: "string", Required: boolPtr(true)},
			{Name: "conversation_id", Type: "string", Required: boolPtr(true)},
			{Name: "user_event", Type: "optional<dict<string,any>>", Required: boolPtr(false), Default: json.RawMessage("null")},
		},
		Returns: "void",
	},
	"signalwire.ai_chat.client.ChatLog": {
		Params: []canonicalParam{
			{Name: "self", Kind: "self"},
			{Name: "messages", Type: "list<dict<string,any>>", Required: boolPtr(false), Default: json.RawMessage("[]")},
			{Name: "call_timeline", Type: "list<dict<string,any>>", Required: boolPtr(false), Default: json.RawMessage("[]")},
		},
		Returns: "void",
	},
}

// handOptionsStructs is the allowlist of SHORT struct names (in hand-written,
// non-generated files) whose fields the enumerator records into
// paramsStructFields so optionsStructUnfoldMethods can unfold them. Keeping it an
// explicit allowlist avoids capturing every exported struct's fields.
var handOptionsStructs = map[string]bool{
	"PlayOptions":        true,
	"AIOptions":          true,
	"ConnectOptions":     true,
	"WaitForUserOptions": true,
}

// paramsStructField is one field of a generated-REST params struct (§5/§4a).
type paramsStructField struct {
	name    string // exported Go field name (e.g. "QueryString", "Extras")
	typeStr string // source-level type expression (e.g. "any", "map[string]any")
	// required carries the SPEC's required flag, read from the generated
	// `sw:"required"` / `sw:"optional"` struct tag. nil when the struct has no
	// tag (a hand-written options struct), in which case the caller falls back
	// to pointer-ness. See swRequiredTag + the REST-unfold in
	// toCanonicalSignature for why the tag is load-bearing: a COMPOSITE field
	// (`[]T` / `map[K]V`) is spelled identically whether the spec requires it or
	// not, so pointer-ness alone cannot recover its contract.
	required *bool
}

// swRequiredTag reads the generated `sw:"required"` / `sw:"optional"` tag off a
// params-struct field, returning nil when the field carries no such tag.
func swRequiredTag(tag *ast.BasicLit) *bool {
	if tag == nil {
		return nil
	}
	lit, err := strconv.Unquote(tag.Value)
	if err != nil {
		return nil
	}
	switch reflect.StructTag(lit).Get("sw") {
	case "required":
		return boolPtr(true)
	case "optional":
		return boolPtr(false)
	}
	return nil
}

// paramsStructFields maps a generated-REST `<...>Params` struct's SHORT type name
// to its ordered fields. Populated while parsing pkg/rest/namespaces/
// *_resources_generated.go. The signature enumerator UNFOLDS these fields back
// into the flat keyword param set the Python oracle records, so collapsing the old
// flat-positional operation/command params into a named Go options struct is a pure
// call-site reshape and keeps port_signatures.json byte-identical (drift 0).
var paramsStructFields = map[string][]paramsStructField{}

// ctorOptionsStructFields maps a CONSTRUCTOR options-struct's `<pkg>.<Name>` key
// to its exported fields. Populated for every exported `<X>Options` struct while
// walking. The CONSTRUCTION CONTRACT (§10) unfolds these back into the named
// configurable set: `prefabs.NewSurveyAgent(SurveyOptions{Name: …, MaxRetries: …})`
// is Go's spelling of the reference's `SurveyAgent(name=…, max_retries=…)`, so
// the struct's fields ARE its construction params — one of the mechanisms §10
// explicitly names ("rust options-struct fields … ts/dotnet options objects").
var ctorOptionsStructFields = map[string][]paramsStructField{}

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
	// defaultJSON is the JSON encoding of the value a caller ACTUALLY GETS when
	// they do not supply this argument, or "" when the parameter has no default
	// (the caller must always pass it). See extractParamDefaults for the three
	// Go mechanisms this is recovered from and the ones it deliberately refuses.
	defaultJSON string
	// optional records that the port DOES model this argument's ABSENCE — a
	// caller can decline to supply a value and the method still behaves. See
	// extractParamOptionality for the three Go mechanisms that constitute
	// absence-modelling and the ones deliberately refused. Distinct from
	// defaultJSON: a parameter can be omittable without this extractor being able
	// to name the resulting VALUE (a guard falling back to a method call), and
	// `required` is the flag the drift gate compares.
	optional bool
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
// FIELDS as zero-arg members — EVERY exported wire field, whatever its type.
//
// This used to filter to CLASS-typed fields only (a `$ref` to another generated
// payload struct, a list of one, or a union carrying one), on the premise that a
// primitive-typed field was Python-internal scaffolding the reference skipped.
// That premise was true only of a DEFECTIVE reference: porting-sdk's oracle
// enumerated each payload TypedDict's class-typed fields and dropped the
// primitive ones, so `ai_name: str` never reached it. porting-sdk e432177
// ("record EVERY TypedDict field, not only class-typed ones") corrected that,
// and the corrected oracle carries the primitives.
//
// Measured across the two oracle revisions, for `AIParams`:
//
//	reference members   pre-e432177: 60    post-e432177: 87
//	go source json keys (swml_verbs_generated.go)          87
//	port_signatures.json under the old filter              60
//
// The 60 was not a coincidence — the port was reproducing the reference's bug
// exactly. Every one of these fields IS declared by the port and IS a wire key;
// filtering them lost real surface. A generated-payload struct declares no
// scaffolding: each field is a wire key, so there is nothing here to skip.
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
// exported field as a member (keyed by wire name), whatever its type.
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

// guardedFields indexes, per FILE, the struct fields whose every READ in that
// file sits behind a zero/nil guard on the field itself. Keyed
// "<Struct>.<Field>".
//
// Why this exists: the port frequently models "the caller may decline this
// argument" ONE HOP away from the signature. `Step.SetGatherInfo(outputKey,
// completionAction, prompt string, isolated bool)` stores its four params
// verbatim into a `GatherInfo` literal and never guards them; the substitution
// happens in `GatherInfo.ToMap`, which emits each wire key only when the field
// is non-zero:
//
//	if g.Prompt != "" { m["prompt"] = g.Prompt }
//
// So `SetGatherInfo("", "", "", false)` IS a supported call producing exactly
// the reference's `output_key=None, completion_action=None, prompt=None,
// isolated=False` document — but a body-local scan sees only an unconditional
// store and calls all four required. Same shape in `DataMap.Webhook` /
// `DataMap.Parameter` (stored into webhookDef / paramDef, guarded at
// serialize time).
//
// The index is deliberately CONSERVATIVE, in the direction that keeps a param
// REQUIRED:
//   - a field with ZERO reads is not guarded (nothing proves the zero value is
//     handled);
//   - a single read outside a guard on that same field disqualifies it;
//   - the analysis is FILE-scoped, so a field read in another file of the same
//     package is invisible — and, being invisible, cannot vouch for the field.
//     Only a struct whose declaration and whose every use live in one file can
//     qualify.
func guardedFieldIndex(file *ast.File) map[string]bool {
	reads := map[string]int{}   // "Struct.Field" -> total reads
	guarded := map[string]int{} // "Struct.Field" -> reads inside a guard on it
	// fieldStruct resolves "<Struct>.<field>" -> the named struct type that
	// field holds, so a TWO-HOP read (`dm.webhookConfig.headers`) resolves to
	// "webhookDef.headers". DataMap.Webhook / .Parameter store their params
	// into exactly such a nested struct, and the guards live on the nested
	// fields at serialize time.
	fieldStruct := map[string]string{}
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
			st, ok := ts.Type.(*ast.StructType)
			if !ok || st.Fields == nil {
				continue
			}
			for _, f := range st.Fields.List {
				ft := exprString(f.Type)
				// A `[]T` field binds its ELEMENT type: a `for _, p := range
				// dm.parameters` loop reads `p.<Field>` on the element struct,
				// and the emptiness guards that vouch for an optional param
				// live exactly there (DataMap.Parameter stores into a
				// []paramDef whose enum/required are guarded in ToSwaigFunction).
				ft = strings.TrimPrefix(ft, "[]")
				ft = strings.TrimPrefix(ft, "*")
				if ft == "" || strings.ContainsAny(ft, "[]{}( )") {
					continue
				}
				if i := strings.LastIndex(ft, "."); i >= 0 {
					ft = ft[i+1:]
				}
				for _, n := range f.Names {
					fieldStruct[ts.Name.Name+"."+n.Name] = ft
				}
			}
		}
	}
	// resolveSel maps a selector expression to its "<Struct>.<Field>" key given
	// the receiver variable name + its struct type. Handles the direct
	// `<rv>.<Field>` and the one-level-nested `<rv>.<container>.<Field>`.
	resolveSel := func(sel *ast.SelectorExpr, rv, st string) string {
		switch x := sel.X.(type) {
		case *ast.Ident:
			if x.Name == rv {
				return st + "." + sel.Sel.Name
			}
		case *ast.SelectorExpr:
			inner, ok := x.X.(*ast.Ident)
			if ok && inner.Name == rv {
				if nested, ok := fieldStruct[st+"."+x.Sel.Name]; ok {
					return nested + "." + sel.Sel.Name
				}
			}
		}
		return ""
	}
	// recvStruct maps a method's receiver VARIABLE name to its struct type, so
	// `g.Prompt` inside a `func (g *GatherInfo)` method resolves to
	// "GatherInfo.Prompt".
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil || fd.Recv == nil || len(fd.Recv.List) == 0 {
			continue
		}
		st := recvTypeName(fd.Recv.List[0].Type)
		if st == "" || len(fd.Recv.List[0].Names) == 0 {
			continue
		}
		rv := fd.Recv.List[0].Names[0].Name
		if rv == "" || rv == "_" {
			continue
		}
		// Walk, tracking which field names are currently "in scope" as guarded
		// by an enclosing `if <rv>.<Field> <zero-test>` condition.
		// rangeVar binds a `for _, v := range <rv>.<field>` loop variable to the
		// element struct type, so `v.<Field>` resolves like `<rv>.<field>.<Field>`.
		rangeVar := map[string]string{}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			rs, ok := n.(*ast.RangeStmt)
			if !ok || rs.Value == nil {
				return true
			}
			v, ok := rs.Value.(*ast.Ident)
			if !ok || v.Name == "_" {
				return true
			}
			sel, ok := rs.X.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			id, ok := sel.X.(*ast.Ident)
			if !ok || id.Name != rv {
				return true
			}
			if elem, ok := fieldStruct[st+"."+sel.Sel.Name]; ok {
				rangeVar[v.Name] = elem
			}
			return true
		})
		var walk func(n ast.Node, inGuard map[string]bool)
		resolve := func(sel *ast.SelectorExpr) string {
			if id, ok := sel.X.(*ast.Ident); ok {
				if elem, ok := rangeVar[id.Name]; ok {
					return elem + "." + sel.Sel.Name
				}
			}
			return resolveSel(sel, rv, st)
		}
		countRead := func(sel *ast.SelectorExpr, inGuard map[string]bool) {
			key := resolve(sel)
			if key == "" {
				return
			}
			reads[key]++
			if inGuard[key] {
				guarded[key]++
			}
		}
		walk = func(n ast.Node, inGuard map[string]bool) {
			switch t := n.(type) {
			case *ast.IfStmt:
				// Fields this condition zero-tests become guarded inside the body.
				inner := map[string]bool{}
				for k, v := range inGuard {
					inner[k] = v
				}
				for _, f := range condGuardedFields(t.Cond, rv, st, func(sel *ast.SelectorExpr, _, _ string) string {
					return resolve(sel)
				}) {
					inner[f] = true
				}
				if t.Init != nil {
					walk(t.Init, inGuard)
				}
				// The CONDITION's own reads are the test itself — count them as
				// guarded, since evaluating `g.Prompt != ""` never misuses a zero.
				// The condition's own reads count as GUARDED only for the
				// fields this condition actually zero-tests. A field merely
				// MENTIONED in the condition (`if w.eventType ==
				// event.EventType` — a match against another value) is a real
				// read of a value the caller must have supplied, so it counts
				// as UNGUARDED and disqualifies the field.
				condZero := map[string]bool{}
				for _, f := range condGuardedFields(t.Cond, rv, st, func(sel *ast.SelectorExpr, _, _ string) string {
					return resolve(sel)
				}) {
					condZero[f] = true
				}
				ast.Inspect(t.Cond, func(c ast.Node) bool {
					if sel, ok := c.(*ast.SelectorExpr); ok {
						if key := resolve(sel); key != "" {
							reads[key]++
							if condZero[key] {
								guarded[key]++
							}
						}
						return false
					}
					return true
				})
				if t.Body != nil {
					walk(t.Body, inner)
				}
				if t.Else != nil {
					walk(t.Else, inGuard)
				}
				return
			case *ast.SelectorExpr:
				countRead(t, inGuard)
				return
			}
			// Default: recurse into children with the same guard set.
			var kids []ast.Node
			ast.Inspect(n, func(c ast.Node) bool {
				if c == nil || c == n {
					return c == n
				}
				kids = append(kids, c)
				return false
			})
			for _, k := range kids {
				walk(k, inGuard)
			}
		}
		walk(fd.Body, map[string]bool{})
	}
	out := map[string]bool{}
	for k, n := range reads {
		if n > 0 && guarded[k] == n {
			out[k] = true
		}
	}
	return out
}

// exprIsZeroLiteral reports whether expr is a zero-value LITERAL — `""`, `0`,
// `false`, or the untyped `nil`. Type-agnostic on purpose: the caller has the
// field name but not its declared type, and any of these literals appearing as
// the comparand makes the test an absence test.
func exprIsZeroLiteral(e ast.Expr) bool {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name == "nil" || t.Name == "false"
	case *ast.BasicLit:
		switch t.Kind {
		case token.STRING:
			return t.Value == `""` || t.Value == "``"
		case token.INT, token.FLOAT:
			return t.Value == "0" || t.Value == "0.0"
		default:
			// CHAR / IMAG and every non-literal token: not a zero value we vouch for.
			return false
		}
	}
	return false
}

// condGuardedFields returns the receiver-field names an `if` condition
// zero/nil-tests, through `&&` / `||` nesting. `if g.Prompt != ""` guards
// "Prompt"; `if g.Isolated` guards "Isolated" (a bare bool IS its own test).
func condGuardedFields(expr ast.Expr, rv, st string,
	resolveSel func(*ast.SelectorExpr, string, string) string) []string {
	var out []string
	var rec func(e ast.Expr)
	fieldOf := func(e ast.Expr) string {
		sel, ok := e.(*ast.SelectorExpr)
		if !ok {
			return ""
		}
		return resolveSel(sel, rv, st)
	}
	rec = func(e ast.Expr) {
		switch t := e.(type) {
		case *ast.ParenExpr:
			rec(t.X)
		case *ast.UnaryExpr:
			if t.Op == token.NOT {
				rec(t.X)
			}
		case *ast.SelectorExpr:
			if f := fieldOf(t); f != "" {
				out = append(out, f)
			}
		case *ast.BinaryExpr:
			if t.Op == token.LAND || t.Op == token.LOR {
				rec(t.X)
				rec(t.Y)
				return
			}
			if t.Op != token.EQL && t.Op != token.NEQ &&
				t.Op != token.LSS && t.Op != token.GTR &&
				t.Op != token.LEQ && t.Op != token.GEQ {
				return
			}
			// The comparand must be the ZERO value. `if w.eventType ==
			// event.EventType` is a MATCH against another value, not an
			// absence test — treating it as a guard made Call.wait_for's
			// `event_type` read optional against a reference that requires it
			// (measured: exactly one manufactured reverse flip). Only a literal
			// zero (`""`, `0`, `false`, `nil`) vouches for the field.
			if f := fieldOf(t.X); f != "" && exprIsZeroLiteral(t.Y) {
				out = append(out, f)
			}
			if f := fieldOf(t.Y); f != "" && exprIsZeroLiteral(t.X) {
				out = append(out, f)
			}
			// `len(g.Questions) > 0`
			if call, ok := t.X.(*ast.CallExpr); ok && len(call.Args) == 1 {
				if fn, ok := call.Fun.(*ast.Ident); ok && fn.Name == "len" {
					if f := fieldOf(call.Args[0]); f != "" {
						out = append(out, f)
					}
				}
			}
		}
	}
	rec(expr)
	return out
}

// paramStoredIntoGuardedField reports whether param is stored VERBATIM into a
// struct-literal field that guardedFieldIndex vouched for — i.e. the port
// accepts the param's zero value and drops it at serialize time, so the caller
// may decline the argument.
func paramStoredIntoGuardedField(fd *ast.FuncDecl, param string, guarded map[string]bool) bool {
	if fd == nil || fd.Body == nil || len(guarded) == 0 {
		return false
	}
	found := false
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		st := ""
		switch t := cl.Type.(type) {
		case *ast.Ident:
			st = t.Name
		case *ast.SelectorExpr:
			st = t.Sel.Name
		}
		if st == "" {
			return true
		}
		for _, el := range cl.Elts {
			kv, ok := el.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			// The value must be the BARE param ident — a wrapped/derived value
			// (`strings.ToUpper(method)`) is a different quantity whose zero the
			// field guard does not speak for.
			val, ok := kv.Value.(*ast.Ident)
			if !ok || val.Name != param {
				continue
			}
			if guarded[st+"."+key.Name] {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// nilPassedSiblingArgs indexes, per FILE, the (callee, argument-position) pairs
// where SOME call in that file passes a literal `nil`. Keyed "<callee>#<idx>".
//
// A sibling passing nil is a PROOF, written in the port's own code, that the
// callee accepts nil in that slot. `HTTPClient.Post(path, body, params, opts)`
// forwards body/params straight into `doRequestContextOpts(ctx, "POST", path,
// body, params, opts)`, and the sibling `Delete` calls the same function as
// `doRequestContextOpts(ctx, "DELETE", path, nil, nil, opts)` — so `Post(p,
// nil, nil, nil)` is a supported call, exactly the reference's `body: Any =
// None, params: dict | None = None`.
//
// Conservative by construction: it only records positions a real call already
// nils, and it is FILE-scoped, so it can never vouch for a callee it has not
// seen nilled.
func nilPassedSiblingArgs(file *ast.File) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		callee := calleeName(call.Fun)
		if callee == "" {
			return true
		}
		for i, a := range call.Args {
			if id, ok := a.(*ast.Ident); ok && id.Name == "nil" {
				out[calleeSlotKey(callee, len(call.Args), i)] = true
			}
		}
		return true
	})
	return out
}

// calleeSlotKey keys a callee argument slot by NAME, ARITY and INDEX. Arity is
// part of the key because a bare method name is ambiguous across overloaded-ish
// shapes: the generated REST tree calls both `HTTP.Get(ctx, path, nil, opts...)`
// and `HTTP.Post(ctx, path, data, nil, opts...)`. Keying on name+index alone
// made Get's nilled slot-2 vouch for Post's slot-2 `data`, which the reference
// records REQUIRED — a manufactured reverse flip, caught by the drift gate.
func calleeSlotKey(callee string, arity, idx int) string {
	return callee + "/" + strconv.Itoa(arity) + "#" + strconv.Itoa(idx)
}

// calleeName renders a call target as a stable key: the bare function name, or
// `<sel>` for a method call (the receiver EXPRESSION is dropped so
// `c.doRequestContextOpts` and `a.doRequestContextOpts` share a key — they are
// the same method).
func calleeName(fun ast.Expr) string {
	switch t := fun.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return ""
}

// paramForwardedToNilledSlot reports whether param is passed VERBATIM into a
// callee argument position that some sibling call in the same file nils.
func paramForwardedToNilledSlot(fd *ast.FuncDecl, param string, nilled map[string]bool) bool {
	if fd == nil || fd.Body == nil || len(nilled) == 0 {
		return false
	}
	found := false
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		callee := calleeName(call.Fun)
		if callee == "" {
			return true
		}
		for i, a := range call.Args {
			id, ok := a.(*ast.Ident)
			if !ok || id.Name != param {
				continue
			}
			if nilled[calleeSlotKey(callee, len(call.Args), i)] {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func parseFile(path string, structs map[string]*goStructFacts, funcs map[string]*goFunc, payloads *genPayloadFacts) error {
	fset := token.NewFileSet()
	// ParseComments is required for the `//sw:param` directives (see
	// applyParamDirectives) — without it every FuncDecl.Doc is nil and the
	// directives silently do nothing.
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution|parser.ParseComments)
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
	// FILE-scoped index of struct fields whose every read is zero-guarded; feeds
	// extractParamOptionality mechanism (4).
	guardedFields := guardedFieldIndex(file)
	// FILE-scoped index of callee arg positions a sibling call already nils;
	// feeds extractParamOptionality mechanism (5).
	// NOT applied to generated REST resources: there the param's requiredness
	// comes from the SPEC (carried by the `sw:` params-struct tag), and a
	// bodyless operation legitimately calls the SAME verb with a nil body
	// (`RegistryBrands.RequestVerification` posts nil) — which would falsely
	// vouch for a sibling's spec-REQUIRED body. Measured: enabling it there
	// manufactured exactly one reverse flip, RegistryBrands.create_campaign.
	var nilledArgs map[string]bool
	if !isRestResource {
		nilledArgs = nilPassedSiblingArgs(file)
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !(ast.IsExported(ts.Name.Name) || isPromotedFieldCarrier(ts.Name.Name)) {
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
						req := swRequiredTag(f.Tag)
						for _, n := range f.Names {
							fields = append(fields, paramsStructField{name: n.Name, typeStr: typeStr, required: req})
						}
					}
					paramsStructFields[ts.Name.Name] = fields
				}
				// Hand-written SWML options structs (PlayOptions/AIOptions): record
				// their fields too, so optionsStructUnfoldMethods can unfold a
				// `opts <Struct>` method param back to the flat oracle keyword set
				// (plan 6.2-go idiom convergence, drift-neutral).
				if handOptionsStructs[ts.Name.Name] && st.Fields != nil {
					var fields []paramsStructField
					for _, f := range st.Fields.List {
						typeStr := exprString(f.Type)
						req := swRequiredTag(f.Tag)
						for _, n := range f.Names {
							fields = append(fields, paramsStructField{name: n.Name, typeStr: typeStr, required: req})
						}
					}
					paramsStructFields[ts.Name.Name] = fields
				}
				// CONSTRUCTION CONTRACT (§10): record every exported `<X>Options`
				// struct's exported fields so a `NewX(XOptions{...})` factory can
				// be unfolded into its named construction params.
				if strings.HasSuffix(ts.Name.Name, "Options") && st.Fields != nil {
					var fields []paramsStructField
					for _, f := range st.Fields.List {
						typeStr := exprString(f.Type)
						for _, n := range f.Names {
							if !ast.IsExported(n.Name) {
								continue
							}
							fields = append(fields, paramsStructField{name: n.Name, typeStr: typeStr})
						}
					}
					if len(fields) > 0 {
						ctorOptionsStructFields[pkgName+"."+ts.Name.Name] = fields
					}
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
						if isLoggerHandleType(typeStr) {
							// Owner ruling 2026-07-24 (ALLOWLIST_DISCIPLINE.md §8):
							// logging is a MODULE-LEVEL capability a port may reach
							// however its language does. The per-instance logger
							// handle is Python's structlog idiom leaking into the
							// enumerated surface, and is NOT contract — the reference
							// suppresses it at porting-sdk/scripts/enumerate_python.py
							// (_LOGGER_FACTORY_RETURN). Go embeds a `Logger
							// *logging.Logger` in agent/skills/swml, so field
							// projection promoted it onto 14 classes and each one
							// needed a dead PORT_SIGNATURE_OMISSIONS entry. Fold it
							// here instead, keyed on the logging-handle TYPE (not a
							// member-name list) so it cannot drift as structs are
							// added or renamed — mirroring the reference's return-type
							// rule. The capability stays signalled by the 5
							// module-level free functions the oracle records
							// (get_logger / configure_logging / get_execution_mode /
							// reset_logging_configuration / strip_control_chars).
							continue
						}
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
			sig := buildSignature(pkgName, d, guardedFields, nilledArgs)
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

// ---------------------------------------------------------------------------
// PARAMETER DEFAULT VALUES
//
// Go HAS NO DEFAULT PARAMETER VALUES. There is no `func f(x int = 5)`, so the
// question "what is this parameter's default" has no syntactic answer the way it
// does in Ruby (`def m(b = 42)`) or C# (`int b = 42`). What the reference oracle
// records as a default is, operationally, "the value a caller gets when they do
// not supply that argument" — so THAT is what is recovered here, from the two
// mechanisms by which this SDK actually gives a Go caller an omittable argument.
// Anything else emits NO default, because for a plain Go param there IS none: a
// caller MUST pass it, and `required: true` with no default is the honest record.
//
// RECOVERED (a caller can genuinely omit the argument):
//
//  1. SENTINEL GUARD — a leading `if <param> <op> <zero> { <param> = <literal> }`
//     at the top of the body, where <zero> is the param type's zero value. The
//     SDK's convention for "pass the zero value to mean 'give me the default'":
//
//         func NewSessionManager(tokenExpirySecs int, ...) *SessionManager {
//             if tokenExpirySecs <= 0 { tokenExpirySecs = 900 }
//
//     Passing 0 yields 900, so 900 IS the default — and it is exactly what the
//     reference records (`token_expiry_secs=900`, pinned by
//     pkg/security/session_manager_test.go).
//
//  2. TRAILING VARIADIC with a zero-length fallback — `ignoreCase ...bool` read
//     as `len(ignoreCase) > 0 && ignoreCase[0]`. Omitting the argument entirely
//     is legal and yields the fallback, so the fallback IS the default:
//
//         func (a *AgentBase) AddPatternHint(..., ignoreCase ...bool) *AgentBase {
//             ... "ignore_case": len(ignoreCase) > 0 && ignoreCase[0],
//
//     Omit it → false, matching the reference's `ignore_case=False`.
//
// DELIBERATELY REFUSED (would be a confident WRONG value):
//
//   - A CLAMP is not a default. `FunctionResult.Hold(timeout int)` does
//     `if timeout < 0 { timeout = 0 }; if timeout > 900 { timeout = 900 }` —
//     that bounds the range, it does not supply an omitted value. Passing the
//     zero value yields 0, NOT the reference's 300. Recording 300 (or 0) would
//     be a fabricated default. Refused by requiring the assigned literal to
//     differ from the compared-against zero value, and by only accepting the
//     `== zero` / `<= 0` / `== ""` guard forms — never a `>` upper bound.
//   - A guard in a HELPER the body delegates to (agent.RegisterRoutingCallback
//     passes `path` through `normalizeCallbackPath`, which maps "" -> "/sip").
//     The value is real but recovering it needs interprocedural analysis that
//     would just as happily follow a helper that ISN'T defaulting. Left absent.
//   - Everything else: a plain Go param with no guard at all (GetFullURL's
//     `includeAuth bool`, GetSecurityHeaders' `isHTTPS bool`). The caller must
//     pass a value; there is no default to record.
//
// This is ADDITIVE — it only ever populates goParam.defaultJSON. It never
// changes which params are enumerated, their order, their types, or their
// `required` flags.
// ---------------------------------------------------------------------------

// zeroLiteralFor returns the JSON encoding of the zero value of a Go type, and
// whether the type is one whose zero value this extractor understands. Only
// scalar types participate — a sentinel guard on a slice/map/pointer is testing
// nil-ness, which carries no defaultable literal.
func zeroLiteralFor(typeStr string) (string, bool) {
	switch strings.TrimSpace(typeStr) {
	case "string":
		return `""`, true
	case "bool":
		return "false", true
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return "0", true
	case "float32", "float64":
		return "0", true
	}
	// A Tier-1 closed-set DEFINED STRING type (`type TapDirection string`, the
	// closedSetUnions table) has the same zero value as its underlying string,
	// so `if direction != "" { … }` is the identical absence test the plain
	// `string` form gets. Without this, typing a param as its closed set (a
	// pure DX improvement) silently flipped it from optional to required —
	// FunctionResult.Tap's `direction`/`codec` and RecordCall's `format`/
	// `direction` all read as required despite bodies that omit the wire key
	// on the zero value.
	if _, ok := closedSetUnions[strings.TrimSpace(typeStr)]; ok {
		return `""`, true
	}
	return "", false
}

// basicLitJSON renders a Go literal expression as its JSON encoding, or ("",
// false) when it is not a plain literal this extractor will vouch for. Handles
// the string/int/float/bool literal forms plus a unary minus, and a conversion
// wrapping a literal (`RecordFormat("wav")`) so a defined-string-type constant
// still yields its underlying value. A named constant, a function call, or any
// computed expression is REFUSED — resolving it needs go/types, and a guessed
// value is worse than none.
func basicLitJSON(e ast.Expr) (string, bool) {
	switch t := e.(type) {
	case *ast.BasicLit:
		switch t.Kind {
		case token.STRING:
			s, err := strconv.Unquote(t.Value)
			if err != nil {
				return "", false
			}
			b, err := json.Marshal(s)
			if err != nil {
				return "", false
			}
			return string(b), true
		case token.INT, token.FLOAT:
			return t.Value, true
		default:
			// CHAR / IMAG and every non-literal token: no JSON-comparable value.
			return "", false
		}
	case *ast.Ident:
		// Only the predeclared booleans; any other identifier is a named
		// constant this extractor will not resolve.
		if t.Name == "true" || t.Name == "false" {
			return t.Name, true
		}
		return "", false
	case *ast.UnaryExpr:
		if t.Op == token.SUB {
			if inner, ok := basicLitJSON(t.X); ok {
				return "-" + inner, true
			}
		}
		return "", false
	case *ast.CallExpr:
		// A single-argument CONVERSION of a literal — `RecordFormat("wav")`,
		// `float64(44)` — carries the literal's value through. A FUNCTION CALL
		// does not, and the two are syntactically identical without go/types.
		//
		// Getting this wrong produces a confidently WRONG default: an earlier
		// revision accepted any single-arg call, so `NewRestClient`'s
		// `project = os.Getenv("SIGNALWIRE_PROJECT_ID")` unwrapped to the
		// literal and recorded the ENV VAR NAME as the default value of
		// `project`. Only a callee that is a bare identifier naming a
		// PREDECLARED numeric/string/bool type is accepted; a package-qualified
		// callee (`os.Getenv`, `strconv.Itoa`) is always a function call and is
		// refused outright, as is any user-defined type name (whose underlying
		// type cannot be confirmed here). Refusing a real conversion costs a
		// missing default; accepting a call invents a wrong one.
		if len(t.Args) != 1 {
			return "", false
		}
		fn, ok := t.Fun.(*ast.Ident)
		if !ok || !isPredeclaredConversionType(fn.Name) {
			return "", false
		}
		return basicLitJSON(t.Args[0])
	}
	return "", false
}

// isPredeclaredConversionType reports whether name is a Go predeclared scalar
// type, i.e. a `name(literal)` expression is certainly a value-preserving
// CONVERSION and not a function call. See the CallExpr case of basicLitJSON.
func isPredeclaredConversionType(name string) bool {
	switch name {
	case "string", "bool",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "byte", "rune":
		return true
	}
	return false
}

// sentinelGuardDefault inspects ONE statement for the sentinel-guard shape
//
//	if <param> <op> <zero> { <param> = <literal> }
//
// and returns the literal's JSON encoding when it matches. op must be one of
// `==` / `<=` / `<` — the "unset or below the floor" forms. A `>` / `>=` guard
// is an UPPER bound (a clamp) and is refused, as is an assignment whose literal
// equals the zero value being tested (also a clamp, e.g. `if t < 0 { t = 0 }`).
// The body must be exactly the one assignment, so a guard with side effects is
// not mistaken for a default.
func sentinelGuardDefault(stmt ast.Stmt, param, typeStr string) (string, bool) {
	ifStmt, ok := stmt.(*ast.IfStmt)
	if !ok || ifStmt.Init != nil || ifStmt.Else != nil || ifStmt.Body == nil {
		return "", false
	}
	if len(ifStmt.Body.List) != 1 {
		return "", false
	}
	zero, known := zeroLiteralFor(typeStr)
	if !known {
		return "", false
	}
	// Condition: `<param> <op> <zero-ish literal>`.
	cond, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok {
		return "", false
	}
	switch cond.Op {
	case token.EQL, token.LEQ, token.LSS:
	default:
		return "", false // `>`/`>=`/`!=` — an upper bound or a negation, not an unset test
	}
	lhs, ok := cond.X.(*ast.Ident)
	if !ok || lhs.Name != param {
		return "", false
	}
	rhsJSON, ok := basicLitJSON(cond.Y)
	if !ok || rhsJSON != zero {
		return "", false // compared against something other than the zero value
	}
	// Body: `<param> = <literal>`.
	assign, ok := ifStmt.Body.List[0].(*ast.AssignStmt)
	if !ok || assign.Tok != token.ASSIGN ||
		len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return "", false
	}
	target, ok := assign.Lhs[0].(*ast.Ident)
	if !ok || target.Name != param {
		return "", false
	}
	valJSON, ok := basicLitJSON(assign.Rhs[0])
	if !ok {
		return "", false
	}
	if valJSON == zero {
		// `if t < 0 { t = 0 }` — a floor clamp, not a default.
		return "", false
	}
	return valJSON, true
}

// variadicFallbackDefault recovers the default of a TRAILING variadic scalar
// param whose body reads it with a zero-length fallback. Two shapes are
// recognized, both of which mean "omit the argument and you get <default>":
//
//	len(p) > 0 && p[0]                     -> false   (bool)
//	if len(p) > 0 { x = p[0] }             -> the value x held before the if
//
// Only the FIRST is claimed here — it is unambiguous and self-contained. The
// second requires tracking the prior assignment and is left to the sentinel
// path when it happens to be expressible. A variadic used any other way (a
// genuine multi-value list) yields no default.
func variadicFallbackDefault(fd *ast.FuncDecl, param, elemType string) (string, bool) {
	if fd.Body == nil {
		return "", false
	}
	if elemType != "bool" {
		return "", false
	}
	found := false
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		be, ok := n.(*ast.BinaryExpr)
		if !ok || be.Op != token.LAND {
			return true
		}
		// LHS: len(<param>) > 0
		lb, ok := be.X.(*ast.BinaryExpr)
		if !ok || lb.Op != token.GTR {
			return true
		}
		call, ok := lb.X.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}
		fn, ok := call.Fun.(*ast.Ident)
		if !ok || fn.Name != "len" {
			return true
		}
		arg, ok := call.Args[0].(*ast.Ident)
		if !ok || arg.Name != param {
			return true
		}
		if lit, ok := lb.Y.(*ast.BasicLit); !ok || lit.Value != "0" {
			return true
		}
		// RHS: <param>[0]
		ix, ok := be.Y.(*ast.IndexExpr)
		if !ok {
			return true
		}
		base, ok := ix.X.(*ast.Ident)
		if !ok || base.Name != param {
			return true
		}
		found = true
		return false
	})
	if !found {
		return "", false
	}
	// `len(p) > 0 && p[0]` evaluates to FALSE when the argument is omitted.
	return "false", true
}

// guardTargetsParam reports whether stmt is an `if` whose condition tests param
// AND whose body assigns to param — i.e. it is a guard ABOUT this parameter,
// regardless of whether its assigned value is a literal this extractor accepts.
// extractParamDefaults uses it to find the FIRST arm of a fallback chain, so an
// unresolvable first arm suppresses a later arm's literal instead of letting it
// masquerade as the default.
func guardTargetsParam(stmt ast.Stmt, param string) bool {
	ifStmt, ok := stmt.(*ast.IfStmt)
	if !ok || ifStmt.Body == nil || len(ifStmt.Body.List) != 1 {
		return false
	}
	cond, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	lhs, ok := cond.X.(*ast.Ident)
	if !ok || lhs.Name != param {
		return false
	}
	assign, ok := ifStmt.Body.List[0].(*ast.AssignStmt)
	if !ok || len(assign.Lhs) != 1 {
		return false
	}
	target, ok := assign.Lhs[0].(*ast.Ident)
	return ok && target.Name == param
}

// extractParamDefaults populates each param's defaultJSON from the function
// body, per the mechanisms documented above. Params with no recoverable default
// are left untouched (defaultJSON stays "").
func extractParamDefaults(sig *goSignature, fd *ast.FuncDecl) {
	if fd.Body == nil {
		return
	}
	for i := range sig.params {
		p := &sig.params[i]
		// (2) trailing variadic scalar with a zero-length fallback.
		if strings.HasPrefix(p.typeStr, "...") {
			if def, ok := variadicFallbackDefault(fd, p.name, p.typeStr[3:]); ok {
				p.defaultJSON = def
			}
			continue
		}
		// (1) sentinel guard. Scoped to the LEADING statements of the body —
		// a guard buried after real work is not the "unset argument" idiom, and
		// scanning the whole body would pick up an unrelated reassignment.
		//
		// Only the FIRST guard on this param counts, and if it does not yield a
		// literal the param gets NO default. A CHAINED fallback must not leak
		// its terminal literal: AgentServer.Register does
		//
		//     if route == "" { route = a.GetRoute() }   // <- the real default
		//     if route == "" { route = "/" }            // <- only if THAT is empty
		//
		// so the value an omitting caller gets is `a.GetRoute()`, not "/". The
		// reference agrees, recording `route`'s default as None (resolved
		// dynamically). Taking "/" would assert a static default the port does
		// not have. Stopping at the first guard on the param makes the
		// unresolvable first arm suppress the whole chain.
		for _, stmt := range fd.Body.List {
			if _, isIf := stmt.(*ast.IfStmt); !isIf {
				if _, isAssign := stmt.(*ast.AssignStmt); isAssign {
					continue // a local setup assignment; keep scanning
				}
				if _, isDecl := stmt.(*ast.DeclStmt); isDecl {
					continue
				}
				break
			}
			if !guardTargetsParam(stmt, p.name) {
				continue // an unrelated guard; keep scanning for this param's
			}
			if def, ok := sentinelGuardDefault(stmt, p.name, p.typeStr); ok {
				p.defaultJSON = def
			}
			break // first guard on this param decides — chained arms are refused
		}
	}
}

// ---------------------------------------------------------------------------
// PARAMETER OPTIONALITY
//
// `required` is a CONTRACT the caller observes: must I supply a value for this
// argument? Go has no default-argument syntax, so the enumerator used to answer
// "required" for EVERY parameter — a CONFIDENT WRONG value everywhere the SDK
// does model absence, and the reason the unified drift checker reported 285
// `required-flip` findings the moment it began comparing `required` outside
// `__init__`.
//
// A Go parameter is OPTIONAL when the port gives the caller a way to say
// "nothing here" and the body HONOURS it — supplies its own value, or skips the
// work the argument would have driven. Three mechanisms:
//
//  1. VARIADIC — `...T`. The argument list can end before it, so omitting is
//     legal by construction. (`opts ...*RequestOptions`, `ignoreCase ...bool`.)
//
//  2. DEFAULTING ZERO-VALUE GUARD — the body tests the parameter against its
//     zero value and, in that branch, either substitutes a value or scopes the
//     optional work:
//
//     if route == "" { route = a.GetRoute() }        // AgentServer.Register
//     if temperature != 0 { b.temp = temperature }   // BedrockAgent.SetInferenceParams
//     if autoMap { … }                               // AgentServer.SetupSIPRouting
//
//     Passing the zero value is a SUPPORTED call meaning "leave it alone", so
//     the caller can decline. Broader than sentinelGuardDefault, which
//     additionally demands a resolvable literal so it can name the resulting
//     VALUE; optionality only needs the guard to exist and to be a DEFAULTING
//     one. A guard whose fallback is a method call (`a.GetRoute()`) makes the
//     param omittable even though no static default is recordable — precisely
//     the reference's `route: Optional[str] = None` shape.
//
//  3. SAFE-NIL POINTER — a `*T` the body never reads THROUGH outside a nil
//     guard. `Say(text string, voice, language, gender *string, volume
//     *float64)` only forwards its pointers into a nil-guarded options struct,
//     so `Say("hi", nil, nil, nil, nil)` is a supported call — exactly the
//     reference's five `= None` defaults. See pointerDerefUnguarded.
//
// DELIBERATELY REFUSED — each of these matched an earlier, looser revision of
// this rule and produced a WRONG `required: false` that the drift checker caught
// as a flip in the opposite direction:
//
//   - A REJECTION guard. `IsValidHostname(host)` does `if host == "" { return
//     false }`; `AddSkillDirectory(path)` does `if path == "" { return
//     errors.New(…) }`; `AddPatternHint` does `if hint == "" || … { return a }`.
//     The zero value is REFUSED, not defaulted — the caller must supply a real
//     one, and the reference agrees (`required: true`). Detected by the guard
//     body terminating the function (`return` / `panic` / `os.Exit`), possibly
//     after logging.
//
//   - A CLAMP. `Context.MoveStep(name, position int)` does `if position < 0 {
//     position = 0 }` — that bounds a SUPPLIED value into range; passing 0
//     still means "index 0", not "unspecified". Refused by requiring an
//     assignment in the guard body to assign something OTHER than the zero
//     value being tested (the same discrimination sentinelGuardDefault makes).
//
//   - A DEREFERENCED POINTER. `*T` is also how Go passes a struct at all —
//     `AgentServer.Register(a *agent.AgentBase, …)` calls `a.GetRoute()` and the
//     reference records `agent` required. Pointer-ness alone proves nothing;
//     see mechanism (3), which requires the body to leave the pointee untouched
//     outside a nil guard.
//
//   - A nil-able map/slice with no guard. `AddAnswerVerb(config map[string]any)`
//     happens to no-op on nil because `range nil` iterates zero times, but that
//     is an emergent property of one body shape, not a declared contract; a
//     sibling that indexes the map panics on nil. Only an explicit guard counts.
//
// This changes ONLY the `required` flag. Parameter identity, order, type and
// kind are untouched, and it never invents a `default`.
// ---------------------------------------------------------------------------

// defaultingZeroGuard reports whether stmt is an `if` that (a) tests param
// against its zero value and (b) treats that case as "not supplied" rather than
// rejecting it or clamping it. See the refusal list above.
func defaultingZeroGuard(stmt ast.Stmt, param, typeStr string) bool {
	ifStmt, ok := stmt.(*ast.IfStmt)
	if !ok || ifStmt.Cond == nil || ifStmt.Body == nil {
		return false
	}
	if !condTestsZeroValue(ifStmt.Cond, param, typeStr) {
		return false
	}
	// (a) REJECTION — the zero-value branch leaves the function. The argument is
	// refused, not defaulted.
	//
	// EXCEPT a productive DISPATCH: `if wait { return fr.AddAction("playback_bg",
	// map[...]{…, "wait": true}) }` followed by `return fr.AddAction("playback_bg",
	// filename)` does not refuse the zero — it picks the OTHER shape for it, which
	// is exactly what an omitted argument means (the reference's `wait: bool =
	// False`). A rejection returns nothing useful (`return`, `return false`,
	// `return nil, err`) or panics; a dispatch returns a CALL result and the
	// function keeps going past the guard with its own productive return.
	if blockTerminates(ifStmt.Body) && !isProductiveDispatch(ifStmt.Body) {
		return false
	}
	// An `else` that terminates is the same rejection written the other way
	// round (`if p != "" { … } else { return }`).
	if ifStmt.Else != nil {
		if eb, ok := ifStmt.Else.(*ast.BlockStmt); ok && blockTerminates(eb) {
			return false
		}
	}
	// (b) CLAMP — the branch assigns the param the very zero value it just
	// tested for (`if position < 0 { position = 0 }`). That bounds a supplied
	// value; it does not accept an absent one.
	if len(ifStmt.Body.List) == 1 {
		if assign, ok := ifStmt.Body.List[0].(*ast.AssignStmt); ok &&
			len(assign.Lhs) == 1 && len(assign.Rhs) == 1 {
			if target, ok := assign.Lhs[0].(*ast.Ident); ok && target.Name == param {
				if zero, known := zeroLiteralFor(typeStr); known {
					if got, ok := basicLitJSON(assign.Rhs[0]); ok && got == zero {
						return false
					}
				}
			}
		}
	}
	return true
}

// paramPassthroughGuard reports whether stmt is `if <param> != <zero> { return
// <param> }` — the "use what was supplied, otherwise fall through and
// substitute" idiom. The zero value is neither refused nor clamped: control
// continues past the guard to build the default, which is precisely what the
// reference's `= None` slot means.
func paramPassthroughGuard(stmt ast.Stmt, param, typeStr string) bool {
	ifStmt, ok := stmt.(*ast.IfStmt)
	if !ok || ifStmt.Cond == nil || ifStmt.Body == nil || ifStmt.Else != nil {
		return false
	}
	// The condition must be the NON-zero test (`p != ""`), not the zero test.
	bin, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return false
	}
	if !condTestsZeroValue(ifStmt.Cond, param, typeStr) {
		return false
	}
	if len(ifStmt.Body.List) != 1 {
		return false
	}
	ret, ok := ifStmt.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}
	id, ok := ret.Results[0].(*ast.Ident)
	return ok && id.Name == param
}

// isProductiveDispatch reports whether a terminating guard body returns a
// COMPUTED value — a function/method CALL result — rather than refusing the
// input. `return fr.AddAction(…)` builds and returns the alternate shape for the
// zero value; `return`, `return false`, `return nil, err` and `panic(…)` refuse
// it. Only a single-statement return of one call expression counts, which keeps
// the rule narrow: a guard that logs then bails, or returns a zero literal, is
// still read as a rejection.
func isProductiveDispatch(b *ast.BlockStmt) bool {
	if b == nil || len(b.List) != 1 {
		return false
	}
	ret, ok := b.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}
	call, isCall := ret.Results[0].(*ast.CallExpr)
	if !isCall {
		return false
	}
	// An ERROR CONSTRUCTION is a rejection wearing a call's clothes:
	// `AddSkillDirectory(path)` does `if path == "" { return errors.New("… must
	// be non-empty") }`, and the reference records `path` REQUIRED. Refuse the
	// known error constructors so a returned error never reads as a default.
	// (Measured: without this the rule manufactured exactly this reverse flip.)
	if isErrorConstruction(call) {
		return false
	}
	return true
}

// isErrorConstruction reports whether a call builds an error value —
// `errors.New(…)`, `fmt.Errorf(…)`, or any `New*Error(…)` / `*Error(…)`
// constructor.
func isErrorConstruction(call *ast.CallExpr) bool {
	name := ""
	pkg := ""
	switch t := call.Fun.(type) {
	case *ast.Ident:
		name = t.Name
	case *ast.SelectorExpr:
		name = t.Sel.Name
		if id, ok := t.X.(*ast.Ident); ok {
			pkg = id.Name
		}
	}
	if pkg == "errors" && name == "New" {
		return true
	}
	if pkg == "fmt" && name == "Errorf" {
		return true
	}
	return strings.HasSuffix(name, "Error") || strings.HasSuffix(name, "Errorf")
}

// blockTerminates reports whether every path through the block leaves the
// enclosing function — a `return`, a `panic(…)`, or an `os.Exit(…)` as the
// FINAL statement. Preceding statements (a log line, an error wrap) do not
// change that, so only the last one is inspected.
func blockTerminates(b *ast.BlockStmt) bool {
	if b == nil || len(b.List) == 0 {
		return false
	}
	switch last := b.List[len(b.List)-1].(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.BranchStmt:
		// `continue` / `break` / `goto` inside a loop body skips the iteration —
		// the same "refuse this input" meaning.
		return true
	case *ast.ExprStmt:
		call, ok := last.X.(*ast.CallExpr)
		if !ok {
			return false
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			return fn.Name == "panic"
		case *ast.SelectorExpr:
			pkg, ok := fn.X.(*ast.Ident)
			return ok && pkg.Name == "os" && fn.Sel.Name == "Exit"
		}
	}
	return false
}

// condTestsZeroValue reports whether expr tests param against the zero value of
// typeStr. Recurses through `&&` / `||` so a compound guard
// (`if path == "" || path == "/"`) still counts.
func condTestsZeroValue(expr ast.Expr, param, typeStr string) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		// `if autoMap { … }` — a bare bool param IS the test; false is the zero.
		return typeStr == "bool" && t.Name == param
	case *ast.UnaryExpr:
		if t.Op == token.NOT {
			return condTestsZeroValue(t.X, param, typeStr)
		}
		return false
	case *ast.ParenExpr:
		return condTestsZeroValue(t.X, param, typeStr)
	case *ast.BinaryExpr:
		switch t.Op {
		case token.LAND, token.LOR:
			return condTestsZeroValue(t.X, param, typeStr) ||
				condTestsZeroValue(t.Y, param, typeStr)
		case token.EQL, token.NEQ, token.LEQ, token.LSS, token.GTR, token.GEQ:
			// `p <op> <zero>` (or the mirrored `<zero> <op> p`). Both directions
			// are absence tests when the comparand is the ZERO value: `if realm
			// == "" { … }` normalises the unset case, `if maxTriggers > 0 {
			// p["max_triggers"] = maxTriggers }` includes the key only when it
			// WAS supplied. An upper-bound clamp (`if timeout > 900 { timeout =
			// 900 }`) compares against a NON-zero literal and so fails
			// exprIsZeroValue below; a floor clamp (`if position < 0 { position
			// = 0 }`) does compare against zero and is caught by
			// defaultingZeroGuard's clamp test instead.
		default:
			return false
		}
		lhs, lok := t.X.(*ast.Ident)
		rhs, rok := t.Y.(*ast.Ident)
		switch {
		case lok && lhs.Name == param:
			return exprIsZeroValue(t.Y, typeStr)
		case rok && rhs.Name == param:
			return exprIsZeroValue(t.X, typeStr)
		}
		// `len(p) == 0` on a nil-able slice/map is an explicit absence test.
		if call, ok := t.X.(*ast.CallExpr); ok && len(call.Args) == 1 {
			if fn, ok := call.Fun.(*ast.Ident); ok && fn.Name == "len" {
				if arg, ok := call.Args[0].(*ast.Ident); ok && arg.Name == param {
					if lit, ok := t.Y.(*ast.BasicLit); ok && lit.Value == "0" {
						return true
					}
				}
			}
		}
		return false
	}
	return false
}

// exprIsZeroValue reports whether expr is the zero-value literal for typeStr —
// `""`, `0`, `false`, or the untyped `nil` for a nil-able type.
func exprIsZeroValue(expr ast.Expr, typeStr string) bool {
	if id, ok := expr.(*ast.Ident); ok && id.Name == "nil" {
		return isNilableType(typeStr)
	}
	zero, known := zeroLiteralFor(typeStr)
	if !known {
		return false
	}
	got, ok := basicLitJSON(expr)
	return ok && got == zero
}

// isNilableParamType reports whether a param's declared type can hold nil, so
// that a nilled callee slot is evidence about THIS param.
func isNilableParamType(typeStr string) bool {
	return isNilableType(typeStr)
}

// isNilableType reports whether typeStr's zero value is `nil` — i.e. comparing
// the param to nil is a well-formed absence test.
func isNilableType(typeStr string) bool {
	s := strings.TrimSpace(typeStr)
	return strings.HasPrefix(s, "*") || strings.HasPrefix(s, "[]") ||
		strings.HasPrefix(s, "map[") || strings.HasPrefix(s, "chan ") ||
		strings.HasPrefix(s, "func(") || s == "error" || s == "any" ||
		s == "interface{}"
}

// extractParamOptionality populates each param's `optional` flag per the
// mechanisms documented above. Params with no absence-modelling are left
// required — the honest record for a bare Go parameter.
func extractParamOptionality(sig *goSignature, fd *ast.FuncDecl, guardedFields, nilledArgs map[string]bool) {
	for i := range sig.params {
		p := &sig.params[i]
		// (1) variadic is visible in the TYPE alone and holds whether or not the
		// function has a body (interface methods included).
		if strings.HasPrefix(p.typeStr, "...") {
			p.optional = true
			continue
		}
		if fd == nil || fd.Body == nil {
			continue
		}
		// (2) defaulting zero-value guard, scanned over the WHOLE body rather
		// than just the leading statements the way extractParamDefaults scopes
		// its search. A default VALUE must come from the "normalise the argument
		// up front" idiom to be trustworthy, but OPTIONALITY only asks whether
		// the zero value is a supported input, and `if autoMap { … }`
		// legitimately sits at the point of use rather than the top.
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			if p.optional {
				return false
			}
			// Do NOT descend into a nested function literal. A guard inside a
			// returned CLOSURE tests a CAPTURED variable at call time, not the
			// caller's argument: `CreateTypedHandlerWrapper(fn, hasRawData)`
			// returns `func(args, rawData) { if hasRawData { … } … }`, whose
			// dispatch says nothing about whether the CALLER may omit
			// hasRawData — and the reference records it REQUIRED. (Measured: a
			// version that descended manufactured exactly this reverse flip.)
			if _, isFuncLit := n.(*ast.FuncLit); isFuncLit {
				return false
			}
			if stmt, ok := n.(ast.Stmt); ok && defaultingZeroGuard(stmt, p.name, p.typeStr) {
				p.optional = true
				return false
			}
			return true
		})
		// (3) SAFE-NIL POINTER. `*T` alone proves nothing — it is also how Go
		// passes a struct at all (`Register(a *agent.AgentBase)`, which the
		// reference records REQUIRED). What proves optionality is the body never
		// TOUCHING the pointee without a guard: `Say(text string, voice,
		// language, gender *string, volume *float64)` only forwards its pointers
		// into a nil-guarded options struct, so `Say("hi", nil, nil, nil, nil)`
		// is a supported call — exactly the reference's five `= None` defaults.
		// A body that dereferences (`*p`) or selects through the pointer
		// (`p.Field`, `p.Method()`) outside a nil guard would panic on nil, so
		// the caller MUST supply a value and the param stays required.
		if !p.optional && strings.HasPrefix(p.typeStr, "*") &&
			!pointerDerefUnguarded(fd, p.name) {
			p.optional = true
		}
		// (4) STORED INTO A GUARDED FIELD. The port models the declined argument
		// one hop from the signature: the param goes verbatim into a struct
		// literal, and the struct's serializer emits the wire key only when the
		// field is non-zero. See guardedFieldIndex.
		if !p.optional && paramStoredIntoGuardedField(fd, p.name, guardedFields) {
			p.optional = true
		}
		// (6) PARAM-PASSTHROUGH GUARD. `CreateSession(callID string)` does
		// `if callID != "" { return callID }` and otherwise GENERATES one. The
		// guard returns the PARAM ITSELF — it neither rejects the zero nor
		// clamps it; the fall-through IS the substituted default, exactly the
		// reference's `call_id: str | None = None`. Narrow on purpose: the
		// returned expression must be the bare param ident, which a rejection
		// (`return false`, `return errors.New(…)`) never is.
		if !p.optional && fd != nil && fd.Body != nil {
			for _, stmt := range fd.Body.List {
				if paramPassthroughGuard(stmt, p.name, p.typeStr) {
					p.optional = true
					break
				}
			}
		}
		// (5) FORWARDED TO A SLOT A SIBLING NILS. Scoped to nil-able types: a
		// value-typed param has no nil to pass, so a nilled slot says nothing
		// about it. See nilPassedSiblingArgs.
		if !p.optional && isNilableParamType(p.typeStr) &&
			paramForwardedToNilledSlot(fd, p.name, nilledArgs) {
			p.optional = true
		}
	}
}

// pointerDerefUnguarded reports whether the body reads THROUGH the pointer
// param — `*p`, `p.Field`, `p.Method()`, `p[i]` — anywhere outside a statement
// guarded by a nil test on that same param. Such a body panics when handed nil,
// so the caller cannot decline the argument.
//
// Conservative in the direction that keeps a param REQUIRED: any dereference it
// cannot prove is nil-guarded counts as unguarded.
func pointerDerefUnguarded(fd *ast.FuncDecl, param string) bool {
	if fd == nil || fd.Body == nil {
		// No body to inspect (an interface method): cannot prove safety.
		return true
	}
	unguarded := false
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if unguarded {
			return false
		}
		switch t := n.(type) {
		case *ast.IfStmt:
			if t.Cond != nil && condTestsNilOfParam(t.Cond, param) {
				// A nil test whose branch LEAVES THE FUNCTION is a REJECTION —
				// `if output == nil { panic("… must not be nil") }`. Nil is
				// refused, so the caller must supply a value: same verdict as an
				// unguarded dereference. (DataMap.Expression does exactly this
				// and the reference records `output` required.)
				if blockTerminates(t.Body) {
					unguarded = true
					return false
				}
				// Otherwise `if p != nil { … }` — the arms are reached only with
				// the nil-ness known, so dereferences inside are safe. Skip the
				// whole statement.
				return false
			}
		case *ast.StarExpr:
			if id, ok := t.X.(*ast.Ident); ok && id.Name == param {
				unguarded = true
				return false
			}
		case *ast.SelectorExpr:
			if id, ok := t.X.(*ast.Ident); ok && id.Name == param {
				unguarded = true
				return false
			}
		case *ast.IndexExpr:
			if id, ok := t.X.(*ast.Ident); ok && id.Name == param {
				unguarded = true
				return false
			}
		}
		return true
	})
	return unguarded
}

// condTestsNilOfParam reports whether expr compares param against nil, through
// any `&&` / `||` nesting.
func condTestsNilOfParam(expr ast.Expr, param string) bool {
	switch t := expr.(type) {
	case *ast.ParenExpr:
		return condTestsNilOfParam(t.X, param)
	case *ast.BinaryExpr:
		if t.Op == token.LAND || t.Op == token.LOR {
			return condTestsNilOfParam(t.X, param) || condTestsNilOfParam(t.Y, param)
		}
		if t.Op != token.EQL && t.Op != token.NEQ {
			return false
		}
		lhs, lok := t.X.(*ast.Ident)
		rhs, rok := t.Y.(*ast.Ident)
		if lok && lhs.Name == param && rok && rhs.Name == "nil" {
			return true
		}
		if rok && rhs.Name == param && lok && lhs.Name == "nil" {
			return true
		}
	}
	return false
}

func buildSignature(pkg string, fd *ast.FuncDecl, guardedFields, nilledArgs map[string]bool) *goSignature {
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
	extractParamDefaults(sig, fd)
	extractParamOptionality(sig, fd, guardedFields, nilledArgs)
	applyParamDirectives(sig, fd)
	return sig
}

// ---------------------------------------------------------------------------
// `//sw:param <name> required|optional` — the DECLARED-CONTRACT escape hatch
//
// The syntactic mechanisms above recover optionality from what the body DOES.
// They are complete for every shape where Go can express the difference, but a
// residue of parameters exists where the reference contract and the Go source
// are genuinely indistinguishable by syntax, in BOTH directions:
//
//   * `set_multilingual(config)` REQUIRES config and `add_answer_verb(config=None)`
//     does not, yet both are spelled `config map[string]any` and both bodies
//     write `if len(config) > 0 { … }` / `range config`. The `len(p) == 0`
//     absence test cannot tell them apart — a blanket rule in either direction
//     manufactures flips in the other (measured: a `len()`-is-optional rule took
//     the count from 60 to 67).
//
//   * `DataMap.Output(result *swaig.FunctionResult)` stores its pointer without
//     dereferencing, which mechanism (3) reads as "nil is supported" — but the
//     reference declares `result` REQUIRED, because a DataMap with a nil output
//     is not a usable tool. The absence of a dereference is a property of a
//     one-line setter, not a declared contract.
//
// For those, the fact lives ONLY in the reference contract, so the honest fix is
// to CARRY it into the Go source rather than to guess it. This directive is that
// carrier: it is source, it is reviewable next to the function it describes, and
// it is inert at runtime (a comment).
//
// Discipline — this is NOT an allow-list and must not become one:
//   * It states the CONTRACT ("must the caller supply this?"), which is exactly
//     what `required` means; it never excuses a difference or hides a finding.
//     A directive that disagrees with the reference is a BUG, and the drift gate
//     still reports the flip, so a wrong one cannot go green.
//   * Reach for it only after the syntactic mechanisms have been checked and the
//     shape is provably ambiguous. If the body genuinely models absence, fix the
//     mechanism instead — that generalises, a directive does not.
//   * Every use names the reference declaration it mirrors.
// ---------------------------------------------------------------------------

// paramDirectiveRe matches `//sw:param <name> required` / `//sw:param <name> optional`
// in a function's doc comment. Trailing prose after the verb is allowed so the
// directive can carry its own justification on the same line.
var paramDirectiveRe = regexp.MustCompile(`^//sw:param\s+(\S+)\s+(required|optional)\b`)

// applyParamDirectives overrides the inferred optionality of any param named by a
// `//sw:param` directive in fd's doc comment. Applied AFTER the syntactic
// mechanisms so it is the final word; a directive naming an unknown param is a
// fail-loud error rather than a silent no-op, so a rename cannot strand it.
func applyParamDirectives(sig *goSignature, fd *ast.FuncDecl) {
	if fd == nil || fd.Doc == nil {
		return
	}
	for _, c := range fd.Doc.List {
		m := paramDirectiveRe.FindStringSubmatch(strings.TrimSpace(c.Text))
		if m == nil {
			continue
		}
		name, verb := m[1], m[2]
		found := false
		for i := range sig.params {
			if sig.params[i].name == name {
				sig.params[i].optional = verb == "optional"
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr,
				"enumerate-signatures: //sw:param names %q, which %s has no parameter for\n",
				name, sig.name)
		}
	}
}

// loggerHandleTypes are the Go type spellings of a per-instance logging handle.
// Keyed on the TYPE, not the field name, so the exclusion cannot drift as structs
// are added or renamed (a field named `Logger` of some OTHER type is still real
// surface, and a logging handle stored under a different field name is still
// excluded). See isLoggerHandleType.
var loggerHandleTypes = map[string]bool{
	"*logging.Logger": true,
	"logging.Logger":  true,
}

// isLoggerHandleType reports whether a struct field's type is the SDK's logging
// handle. Per the owner ruling of 2026-07-24 (ALLOWLIST_DISCIPLINE.md §8) logging
// is a module-level capability, so the per-instance handle is idiom rather than
// contract and is folded out of the enumerated surface on both sides.
func isLoggerHandleType(typeStr string) bool {
	return loggerHandleTypes[strings.TrimSpace(typeStr)]
}

// transportHandleTypes are the Go type spellings of the REST TRANSPORT handle a
// resource holds so it can issue requests. Keyed on the TYPE (like
// loggerHandleTypes) so the exclusion cannot drift as resources are added.
//
// Every generated REST resource embeds `Resource{HTTP HTTPClient; Base string}`,
// and Go promotes `HTTP` onto all ~90 of them. It is NOT caller surface: the
// reference stores the same handle as the PRIVATE `_client` attribute, which the
// oracle does not record. Go has to export it only because the resources and the
// `rest` package that builds the adapter live in different packages — a
// visibility constraint, not an API decision. Projecting it would manufacture a
// bogus `http` accessor on every resource class. Same rule as the existing
// context.Context / http.Header carve-outs in the accessor projection: a
// plumbing-typed field is not a sub-resource accessor.
var transportHandleTypes = map[string]bool{
	"HTTPClient":            true,
	"*HTTPClient":           true,
	"namespaces.HTTPClient": true,
}

// isTransportHandleType reports whether a struct field's type is the REST
// transport handle. See transportHandleTypes.
func isTransportHandleType(typeStr string) bool {
	return transportHandleTypes[strings.TrimSpace(typeStr)]
}

// isPromotedFieldCarrier delegates to the SHARED table in internal/surface, so
// the signature and surface enumerators cannot disagree about which unexported
// carriers must be walked. See surfacepkg.IsPromotedFieldCarrier.
func isPromotedFieldCarrier(name string) bool { return surfacepkg.IsPromotedFieldCarrier(name) }

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

// promotedFieldSet returns every exported FIELD reachable on facts — its own
// plus the ones Go promotes through the anonymous-embed chain — keyed by Go
// field name. An own field always wins over a promoted one of the same name
// (that is Go's own shallower-depth selector rule).
//
// Field promotion is why `RestClient` needed this: the hand client declares NO
// exported fields of its own. All 22 namespace accessors (`Fabric`, `Calling`,
// `Video`, …) live on the generated `_GeneratedResourceTree` it embeds, and Go
// promotes them so `client.Fabric.AIAgents.List(...)` resolves on the client
// exactly as the reference's `client.fabric.ai_agents.list()` does. A walker
// that reads only own fields sees zero of them and reports all 22 as
// "missing-port" — a blind spot in the enumerator, not an absent capability
// (the accessors are proven live by the mock-backed
// TestResourceTreeAccessors_* tests). This mirrors the promoted-field walk the
// construction collector already does (collectStructLiteralFields) and the
// reference enumerator's own `_wired_base_attributes` lift.
func promotedFieldSet(structs map[string]*goStructFacts, facts *goStructFacts) map[string]*goSignature {
	out := map[string]*goSignature{}
	seen := map[string]bool{}
	var walk func(f *goStructFacts, depth int)
	walk = func(f *goStructFacts, depth int) {
		if f == nil || depth > 4 || seen[f.pkg+"."+f.name] {
			return
		}
		seen[f.pkg+"."+f.name] = true
		for name, sig := range f.methods {
			if !sig.isField {
				continue
			}
			// Shallower depth wins: a field already recorded came from this
			// struct or a nearer embed and shadows the deeper one.
			if _, already := out[name]; already {
				continue
			}
			out[name] = sig
		}
		for _, embed := range f.embeds {
			if base, ok := structs[f.pkg+"."+embed]; ok {
				walk(base, depth+1)
			}
		}
	}
	walk(facts, 0)
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
	raw, err := os.ReadFile(path) //nolint:gosec // G304: developer-run codegen reading a spec/source path derived from the repo root or $PORTING_SDK, not from untrusted input.
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
	// namespaces.Paginator is the value CrudResource.Paginate returns — the
	// Go-idiom equivalent of Python ReadResource.paginate()'s PaginatedIterator,
	// and (since plan 6.2-go retired the orphan rest.PaginatedIterator) the port's
	// sole representative of that class. It lives in the namespaces package to
	// avoid the rest->namespaces import cycle. Its return type folds to that class
	// ref for the signature comparison (idiom reconciled in the alias table, not an
	// omission); the adapter StructTable also maps its methods onto the class.
	"Paginator":            "class:signalwire.rest._pagination.PaginatedIterator",
	"namespaces.Paginator": "class:signalwire.rest._pagination.PaginatedIterator",
	// rest.EffectiveOptions is the Go public spelling of the reference's
	// private _EffectiveOptions (the fully-resolved options returned by
	// Resolve and consumed by StatusIsRetryable). Fold it to that class ref so
	// the two functions' signatures compare EQUAL (idiom in the alias table).
	"EffectiveOptions":      "class:signalwire.rest._request_options._EffectiveOptions",
	"rest.EffectiveOptions": "class:signalwire.rest._request_options._EffectiveOptions",
}

// ctxTailMethods names the methods whose Python reference declares a TRAILING
// optional `timeout` — the slot the Go leading `ctx context.Context` stands in
// for. On these the enumerator moves the recorded ctx param to the tail so it
// aligns with the reference's timeout instead of shifting every other param by
// one. Keyed by the reference QN (module.Class.method). Methods whose reference
// has NO timeout are absent: their extra leading ctx is absorbed by the diff.
var ctxTailMethods = map[string]bool{
	"signalwire.relay.call.Call.wait_for": true,
}

// collectedRegistrationVoid names the methods whose reference returns `None`
// because Python registers by MUTATION (the body calls `self.agent.define_tool(...)`
// for each tool and returns nothing), while Go registers by COLLECTION — the
// `SkillBase` interface declares `RegisterTools() []ToolRegistration`
// (pkg/skills/skill_base.go:52) and the SkillManager consumes the returned slice.
// Both register exactly the same tools; the difference is WHO performs the
// mutation, which is a registration-idiom difference, not a capability one.
//
// Go cannot express the Python form without giving up its own contract: a
// `RegisterTools()` with no return value would have to reach into the agent and
// mutate it, which is precisely the interface Go's composition-over-inheritance
// skill model exists to avoid. So the collected-slice return folds to the
// reference's `void` HERE, at the enumerator, where the params keep comparing —
// rather than being excused in PORT_SIGNATURE_OMISSIONS.md, which would stop
// comparing the symbol altogether.
//
// Keyed by the reference QN (module.Class.method) so the fold is scoped to the
// skill-registration contract and no other method returning a slice is affected.
//
// DERIVED, not hand-listed. This used to be a size-1 map holding only
// MCPGatewaySkill — correct while the signature oracle recorded 7 of 18 skill
// modules and MCPGatewaySkill was the only concrete skill whose register_tools
// the signature axis projected. porting-sdk 8496c77 (2026-07-30) made the oracle
// record all 18, and every one of them declares the SAME
// `RegisterTools() []ToolRegistration` against the same reference `-> None`. A
// hand list would have to be re-edited on every skill added; deriving it from
// SkillContractTable means the fold covers exactly the skills the projection
// emits, automatically.
var collectedRegistrationVoid = func() map[string]bool {
	m := map[string]bool{}
	for _, sc := range surfacepkg.SkillContractTable {
		for _, leaf := range sc.Methods {
			if leaf == "register_tools" {
				m[sc.Module+"."+sc.ClassName+".register_tools"] = true
			}
		}
	}
	return m
}()

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
	// http.Header is Go's (nil-able) response-header map. The reference spells the
	// 6.6 `headers` ctor param `optional<dict<string,string>>`; fold the stdlib type
	// to that canonical spelling (a pure type-fold reconciled at the adapter, per
	// RULES — never an omission).
	if t == "http.Header" || t == "net/http.Header" {
		return "optional<dict<string,string>>", nil
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
	// Construction is THE CONSTRUCTION CONTRACT (porting-sdk
	// ALLOWLIST_DISCIPLINE.md §10): a NAME-KEYED, unordered set of configurable
	// construction params per class. See buildConstruction.
	Construction map[string]constructionEntry `json:"construction,omitempty"`
}

type constructionEntry struct {
	Params map[string]constructionParam `json:"params"`
}

type constructionParam struct {
	Type     string `json:"type"`
	Required bool   `json:"required"`
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

// indexOfParam returns the index of the first canonicalParam with the given
// name, or -1. Used by the request_options ordering reconciliation.
func indexOfParam(params []canonicalParam, name string) int {
	for i, p := range params {
		if p.Name == name {
			return i
		}
	}
	return -1
}

func goNameToSnake(s string) string { return surfacepkg.GoNameToSnake(s) }

// isSignatureDivergentDataclassField reports whether an oracle-gated @dataclass
// field's raw Go type is a genuine idiom divergence with no faithful reference-
// shaped signature, so it must not be emitted on the signature side (the member
// stays surface-only + an annotated PORT_SIGNATURE_OMISSIONS entry). Currently the
// two RequestOptions cases: context.Context (Go's cancellation primitive vs the
// reference _AbortSignal object) and map[int]bool (Go's SET idiom vs the
// reference list[int] for retry-on-status).
func isSignatureDivergentDataclassField(goType string) bool {
	switch goType {
	case "context.Context", "map[int]bool":
		return true
	}
	return false
}

// sigOracleMembers is python_signatures.json restricted to the per-class member
// sets the @dataclass field emission gates on: module -> class -> set(member).
type sigOracleMembers map[string]map[string]map[string]bool

// RETIRED: dataclassFieldModules — same reasoning as in cmd/enumerate-surface.
// The closed three-module scope became a stale exclusion once class B2 widened the
// oracle to record ctor-param `__init__` attributes SDK-wide; loadSigOracle now
// returns every module and the oracle alone gates emission, so the gate follows
// the oracle instead of a hand-maintained list.

// loadSigOracle reads python_signatures.json (adjacent porting-sdk) and returns
// the per-class member sets for dataclassFieldModules only. GATES field emission
// so the port surfaces exactly the reference field set per class. Returns nil
// silently if the oracle can't be located/parsed (degraded-env tolerance —
// emission then no-ops, like the composition enrich's adjacency tolerance).
// resolvePortingSDK is the fail-loud oracle-checkout resolver, matching
// cmd/enumerate-surface's: $PORTING_SDK, then the sibling layout, then the CI
// in-repo layout, and an ERROR rather than a silent miss. See that copy for why
// silence here produced a valid-looking snapshot missing ~200 members and a
// surface-audit red that could not be reproduced locally.
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
		if _, err := os.Stat(filepath.Join(c, "python_signatures.json")); err == nil { //nolint:gosec // G703: path is composed from the repo root / $PORTING_SDK in a developer-run tool, not from untrusted input.
			return c, nil
		}
	}
	return "", fmt.Errorf(
		"porting-sdk not found (looked for python_signatures.json under %v); "+
			"set PORTING_SDK or clone porting-sdk adjacent to this repo", candidates)
}

// loadSigOracle reads python_signatures.json and returns the per-class member
// sets. FAILS LOUD (see resolvePortingSDK).
func loadSigOracle(repoRoot string) (sigOracleMembers, error) {
	psdk, err := resolvePortingSDK(repoRoot)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(psdk, "python_signatures.json")
	raw, err := os.ReadFile(path) //nolint:gosec // G304: developer-run codegen reading a spec/source path derived from the repo root or $PORTING_SDK, not from untrusted input.
	if err != nil {
		return nil, fmt.Errorf("read oracle %s: %w", path, err)
	}
	var parsed struct {
		Modules map[string]struct {
			Classes map[string]struct {
				Methods map[string]json.RawMessage `json:"methods"`
			} `json:"classes"`
		} `json:"modules"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parse oracle %s: %w", path, err)
	}
	out := sigOracleMembers{}
	for mod, mi := range parsed.Modules {
		cm := map[string]map[string]bool{}
		for cls, ci := range mi.Classes {
			set := map[string]bool{}
			for m := range ci.Methods {
				set[m] = true
			}
			cm[cls] = set
		}
		out[mod] = cm
	}
	return out, nil
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

func goFieldToPython(s string) string { return surfacepkg.GoNameToPython(s) }

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
		// Idiom-convergence (plan 6.2-go): a SWML Service method spelled as a single
		// named options struct (`opts PlayOptions` / `opts AIOptions`) UNFOLDS back to
		// the flat keyword param set the Python oracle records. Each field → the same
		// param the old flat-positional signature produced (translateType of the field
		// type, required:true — Go has no defaults), so port_signatures.json is
		// byte-identical to the pre-reshape form (drift 0). A trailing `Extra` map
		// field folds to the reference's `**kwargs` var_keyword tail (the old
		// `extra map[string]any` last-param behavior), gated on the method being in
		// kwargsTailMethods just like the flat form was.
		if structName, ok := optionsStructUnfoldMethods[ctx]; ok && p.typeStr == structName {
			if fields, ok := paramsStructFields[structName]; ok {
				for fi, f := range fields {
					isLast := fi == len(fields)-1
					if isLast && (f.name == "Extra" || f.name == "Extras") &&
						strings.HasPrefix(f.typeStr, "map[string]") && kwargsTailMethods[ctx] {
						params = append(params, canonicalParam{
							Name: "kwargs", Kind: "var_keyword", Type: "any",
							Required: boolPtr(false), Default: json.RawMessage("{}"),
						})
						continue
					}
					fCanon, fail := translateType(f.typeStr, aliases, ctx+"["+f.name+"]")
					if fail != nil {
						failures = append(failures, *fail)
						continue
					}
					// Requiredness comes from the field's `sw:"required"` /
					// `sw:"optional"` tag when the struct declares one — the same
					// mechanism the generated-REST unfold uses, and for the same
					// reason. Pointer-ness answers this correctly only for a SCALAR
					// field: Go spells an omittable `*bool` / `*int` but has no
					// `*[]T` / `*map[K]V` form, so a COMPOSITE field is `[]T` /
					// `map[K]V` whether the reference requires it or not
					// (AIOptions.PromptPOM is optional, Play's `urls` optional,
					// while a required composite would look identical). Nor does
					// pointer-ness reach a non-pointer field whose reference default
					// is a NON-ZERO scalar (`connect(final: bool = True)`,
					// `wait_for_user(answer_first: bool = False)`), where the Go
					// zero value is a supported "leave it" input the verb builder
					// honours. The tag carries that fact from the reference contract
					// into the source; it is inert at runtime (these structs are
					// read field-by-field into the verb config and never
					// JSON-marshaled). Untagged fields fall back to pointer-ness.
					fieldRequired := !strings.HasPrefix(f.typeStr, "*")
					if f.required != nil {
						fieldRequired = *f.required
					}
					params = append(params, canonicalParam{
						Name: goFieldToPython(f.name), Type: fCanon,
						Required: boolPtr(fieldRequired),
					})
				}
				continue
			}
		}
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
		// A trailing variadic of a BOOL or NUMERIC scalar is the Go idiom for a
		// reference parameter whose default is a NON-ZERO scalar —
		// `hold(timeout=300)`, `enable_debug_events(level=1)`,
		// `swml_transfer(final=True)`. Go's zero value is itself a meaningful
		// argument for those (0 seconds, level off, temporary transfer), so a plain
		// `int`/`bool` parameter has no spelling for "not supplied"; `...int` /
		// `...bool` (call with 0 or 1 argument) is the idiom that does, and the body
		// substitutes the reference default for the empty case.
		//
		// The raw translation would record `list<int>` / `list<bool>` required:true,
		// which mismatches the reference's scalar. Reclassify to the SCALAR the
		// variadic carries — NOT `optional<T>`: the reference types these as plain
		// `bool` / `int` (a non-None default does not widen the annotation), so
		// `optional<>` would trade a param-mismatch for a different one.
		//
		// Deliberately NOT applied to `...string` (handled above, and where the
		// reference DOES use `str | None`), nor to composite element types, where a
		// trailing variadic is a genuine multi-argument list rather than an
		// optional-scalar idiom.
		if pi == len(sig.params)-1 && strings.HasPrefix(p.typeStr, "...") {
			if elem, ok := optionalScalarVariadicElem(p.typeStr); ok {
				elemCanon, fail := translateType(elem, aliases, ctx)
				if fail != nil {
					failures = append(failures, *fail)
				} else {
					params = append(params, canonicalParam{
						Name: goNameToSnake(p.name), Type: elemCanon,
						Required: boolPtr(false),
					})
					continue
				}
			}
		}
		// The COMPOSITE-element analog, opted in per method by
		// optionalTailVariadicComposite: a trailing variadic of a non-scalar that
		// the body reads as "zero or one" rather than as a list. Reclassify to the
		// ELEMENT type, preserving whatever optional<> the element already carries.
		if pi == len(sig.params)-1 && strings.HasPrefix(p.typeStr, "...") &&
			optionalTailVariadicComposite[ctx] {
			elemCanon, fail := translateType(strings.TrimPrefix(p.typeStr, "..."), aliases, ctx)
			if fail != nil {
				failures = append(failures, *fail)
			} else {
				params = append(params, canonicalParam{
					Name: goNameToSnake(p.name), Type: elemCanon,
					Required: boolPtr(false),
				})
				continue
			}
		}
		// A ctor's trailing variadic `...*RequestOptions` is the Go idiom for the
		// reference's optional `request_options: RequestOptions | None = None`.
		// Reclassify it to the single optional class param so it compares EQUAL
		// (see optionalRequestOptionsTailMethods). Scoped to the LAST param and to
		// a `...*RequestOptions`.
		if pi == len(sig.params)-1 && optionalRequestOptionsTailMethods[ctx] &&
			p.typeStr == "...*RequestOptions" {
			params = append(params, canonicalParam{
				Name:     goNameToSnake(p.name),
				Type:     "optional<class:signalwire.rest._request_options.RequestOptions>",
				Required: boolPtr(false),
			})
			continue
		}
		// request_options (PY-7 / GO-1): every generated-REST verb, its
		// base-embedded CRUD/read counterparts, and the set_method wrappers carry a
		// trailing per-request `opts ...*RequestOptions` (a variadic) — or, for the
		// set_method form whose `extra ...map[string]any` already claims the
		// variadic slot, a non-variadic `requestOptions *RequestOptions`. Both are
		// the Go spelling of the reference's optional keyword-only
		// `request_options: RequestOptions | None = None`. Reclassify EITHER to that
		// single optional keyword class param so the port compares EQUAL (idiom
		// reconciled in the enumerator, not an omission). The type marker
		// `*RequestOptions` / `...*RequestOptions` is unambiguous — it appears
		// nowhere else — so this is not table-gated. It is emitted here (before the
		// var_keyword/struct handling) so the extra map[string]any tail still folds
		// to **kwargs after it.
		// The GENERATED verbs spell it as the variadic `opts ...*RequestOptions`;
		// the set_method wrappers (whose `extra ...` already claims the variadic
		// slot) spell it as a non-variadic param named exactly `requestOptions`.
		// Both map to the reference's optional keyword-only request_options. A
		// non-variadic `*RequestOptions` under ANY OTHER name (the hand HTTPClient
		// verbs' `opts *RequestOptions`, where the reference records a plain
		// POSITIONAL request_options) is NOT reclassified here — it falls through to
		// the normal positional class translation below.
		// EXCLUDE the ctor/HttpClient methods in optionalRequestOptionsTailMethods:
		// their reference records request_options as a plain POSITIONAL param (a
		// normal `__init__(..., request_options=None)` / hand-verb signature, not a
		// keyword-only `*, request_options`). Those are handled by the ctor-scoped
		// rule below (positional). Only the generated verbs + set_methods get the
		// keyword-only classification here.
		if !optionalRequestOptionsTailMethods[ctx] &&
			(p.typeStr == "...*RequestOptions" ||
				(p.typeStr == "*RequestOptions" && p.name == "requestOptions")) {
			params = append(params, canonicalParam{
				Name:     "request_options",
				Kind:     "keyword",
				Type:     "optional<class:signalwire.rest._request_options.RequestOptions>",
				Required: boolPtr(false),
				Default:  json.RawMessage("null"),
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
					// The REST generator ALREADY encodes each SCALAR field's
					// optionality in its Go type, and the emitted method body
					// proves it: a POINTER field is nil-guarded before it reaches
					// the wire
					//
					//     if params.AddressType != nil { body["address_type"] = … }
					//
					// so omitting it is a supported call, while a VALUE field is
					// serialized unconditionally
					//
					//     body["label"] = params.Label
					//
					// so the caller must supply it. Addresses.create's 9 value
					// fields are exactly the oracle's 9 `required: true` params and
					// its 2 pointer fields the oracle's 2 `required: false` ones.
					// Recording every unfolded field `required: true` threw that
					// away and was the single largest source of go's
					// `required-flip` findings (148 of 285).
					//
					// For a COMPOSITE field that reasoning runs out: `[]T` /
					// `map[K]V` is spelled identically whether the spec requires it
					// or not (the generator has no `*[]T` form), so it is
					// nil-guarded in BOTH cases and the guard proves nothing.
					// `Calling.play`'s `Play []map[string]any` is nil-guarded and
					// the oracle records it REQUIRED; `Calling.collect`'s
					// `Digits map[string]any` is nil-guarded and OPTIONAL.
					// Inferring optionality from nil-ability was measured: it
					// resolved 14 findings and manufactured 10 in the opposite
					// direction (a port claiming optional where the spec requires),
					// the worse error. The fix is upstream: `generate-rest` now
					// emits the spec's flag as a `sw:"required"`/`sw:"optional"`
					// struct tag (paramsStructFieldDef), and this unfold reads it.
					// Pointer-ness remains only as the fallback for a struct with
					// no tag.
					if f.name == "Extras" {
						// `Extras` is the open-ended overflow bag, merged via the
						// nil-safe mergeExtra; the oracle records it optional.
						params = append(params, canonicalParam{
							Name: "extras", Kind: "keyword", Type: fCanon,
							Required: boolPtr(false),
						})
						params = append(params, canonicalParam{
							Name: "kwargs", Kind: "var_keyword", Type: "any",
							Required: boolPtr(false), Default: json.RawMessage("{}"),
						})
						continue
					}
					// The generated `sw:"..."` tag carries the SPEC's flag
					// verbatim; use it whenever present. Only a params struct
					// emitted before the tag existed falls back to pointer-ness.
					req := f.required
					if req == nil {
						req = boolPtr(!strings.HasPrefix(f.typeStr, "*"))
					}
					params = append(params, canonicalParam{
						Name: goNameToSnake(f.name), Kind: "keyword", Type: fCanon,
						Required: req,
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
			Name: goNameToSnake(p.name),
			Type: canon,
			// Go has no LANGUAGE-LEVEL defaults, but `required` is not about
			// default SYNTAX — it is the caller-observable contract "must I
			// supply this?". Required unless the port models the argument's
			// absence (pointer / variadic / zero-value guard); see
			// extractParamOptionality.
			Required: boolPtr(!p.optional),
		}
		// Where the port DOES give a caller an omittable argument — a sentinel
		// guard or a zero-length variadic fallback (see extractParamDefaults) —
		// record the value the caller actually gets. This is ADDITIVE: it sets
		// only Default and deliberately leaves Required alone, so the param set,
		// order, types and required flags are untouched.
		if p.defaultJSON != "" {
			cp.Default = json.RawMessage(p.defaultJSON)
		}
		// A leading `ctx context.Context` (translated to optional<float>, the
		// reference's timeout=None slot) is OPTIONAL, not required: a caller can pass
		// context.Background() for the no-deadline case, exactly like the Python
		// reference omits its optional `timeout`. Recording it required:false lets
		// diff_port_signatures absorb the single leading ctx before the prefix compare
		// (the Go ctx-first idiom reconciliation), so a ctx-first generated method
		// compares EQUAL to the ctx-free Python reference. The ctx is never serialized,
		// so this is a pure signature-shape reconciliation, not a wire change.
		if p.typeStr == "context.Context" {
			cp.Required = boolPtr(false)
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
		// The hand base resource methods (namespaces common.go: List / Get /
		// ListAddresses / Paginate on CrudResource / CrudWithAddresses / the
		// ReadResource embed) are NOT in a *_resources_generated.go file, so
		// sig.restResource is false — but they carry the SAME `params
		// map[string]string` query bag that the reference records as a stripped
		// `**params` var_keyword. Recognize it by shape: a `params`/`extra` map bag
		// IMMEDIATELY FOLLOWED by the `opts ...*RequestOptions` tail (the base-verb
		// signature) → var_keyword, so it absorbs against the reference's stripped
		// **params exactly like the generated form.
		if (p.name == "params" || p.name == "extra") &&
			strings.HasPrefix(p.typeStr, "map[string]") &&
			pi+1 < len(sig.params) && sig.params[pi+1].typeStr == "...*RequestOptions" {
			params = append(params, canonicalParam{
				Name: p.name, Kind: "var_keyword", Type: "any",
				Required: boolPtr(false), Default: json.RawMessage("{}"),
			})
			continue
		}
		params = append(params, cp)
	}
	// request_options ordering (PY-7 / GO-1): Python declares
	// `*, request_options=None, **params` — the keyword-only request_options comes
	// BEFORE the `**params` var_keyword tail; the oracle strips the tail, leaving
	// request_options LAST. The Go verb spells the query/extra bag first
	// (`params map[string]string` / `extra ...`) and the request_options tail
	// after it, so the enumerated order is `..., <var_keyword>, request_options`.
	// Reorder so request_options precedes any trailing var_keyword param(s),
	// matching Python's declaration order — then the port's stripped-equivalent
	// (a trailing optional var_keyword the diff absorbs) aligns with the
	// reference's request_options positionally. Pure signature-shape
	// reconciliation; the wire is unchanged.
	if roIdx := indexOfParam(params, "request_options"); roIdx >= 0 {
		ro := params[roIdx]
		// Find the first var_keyword tail param (the reclassified GET query /
		// set_method extra bag = the reference's stripped `**params`).
		// request_options must sit BEFORE it to match Python's
		// `*, request_options=None, **params` order.
		firstVK := -1
		for i, cp := range params {
			if cp.Kind == "var_keyword" {
				firstVK = i
				break
			}
		}
		if firstVK >= 0 && firstVK < roIdx {
			// Remove request_options from its slot and re-insert before the var_keyword.
			params = append(params[:roIdx], params[roIdx+1:]...)
			out := make([]canonicalParam, 0, len(params)+1)
			out = append(out, params[:firstVK]...)
			out = append(out, ro)
			out = append(out, params[firstVK:]...)
			params = out
		}
	}
	// ctx ORDERING (the timeout-slot analog of the request_options reorder above).
	// A leading `ctx context.Context` records as the reference's `optional<float>`
	// timeout slot. On a method whose reference declares NO timeout, the diff
	// absorbs the extra leading ctx and the prefix aligns. But when the reference
	// DOES declare a trailing `timeout: float | None = None` (Call.wait_for), the
	// two sides have the SAME arity, absorption does not fire, and the port's ctx
	// sits at slot 0 against the reference's slot-2 timeout — misaligning every
	// param between them. Move the ctx to the TAIL for exactly those methods so
	// the slots line up. Table-gated on the reference QN, so a ctx on any other
	// method keeps its leading position and the absorption path.
	if ctxTailMethods[ctx] {
		if ci := indexOfParam(params, "ctx"); ci >= 0 {
			cparam := params[ci]
			params = append(params[:ci], params[ci+1:]...)
			params = append(params, cparam)
		}
	}
	returns := "void"
	switch {
	case isCtor:
		returns = "void"
	case collectedRegistrationVoid[ctx]:
		// Go's collect-and-return registration idiom folds to the reference's
		// mutate-and-return-None. See collectedRegistrationVoid.
		returns = "void"
	default:
		canon, fail := translateType(sig.returns, aliases, ctx+"[->]")
		if fail != nil {
			failures = append(failures, *fail)
		} else {
			returns = canon
		}
	}
	return canonicalSignature{Params: params, Returns: returns}, failures
}

func build(structs map[string]*goStructFacts, funcs map[string]*goFunc, payloads *genPayloadFacts, aliases map[string]string, sigOracle sigOracleMembers) (sigDoc, []translationFailure) {
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
		// ai_chat idiom fold: the AIChatClient turn methods + constructor collapse
		// several Python kwargs into a leading ctx + a Go options struct/functional
		// options. Splice the reference-shaped signature (see aiChatMethodSigs) so
		// the recorded signature matches the oracle exactly (drift 0), keeping the
		// Go call shape idiomatic.
		if spliced, ok := aiChatMethodSigs[mod+"."+cls+"."+method]; ok {
			sig = spliced
		}
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
	// aiChatCtorSynthesized guards the one-time emission of each ai_chat data/error
	// struct's synthesized __init__ (aiChatCtorSigs), so a struct that IS listed in
	// StructTable (its methods projected above) still gets its field-derived ctor.
	aiChatCtorSynthesized := map[string]bool{}
	emitAIChatCtor := func(mod, cls string) {
		qn := mod + "." + cls
		if sig, ok := aiChatCtorSigs[qn]; ok && !aiChatCtorSynthesized[qn] {
			aiChatCtorSynthesized[qn] = true
			addClassMethod(mod, cls, "__init__", sig)
		}
	}
	addFunction := func(mod, name string, sig canonicalSignature) {
		inv := ensureModule(mod)
		if inv.Functions == nil {
			inv.Functions = map[string]canonicalSignature{}
		}
		inv.Functions[name] = sig
		out.Modules[mod] = inv
	}

	// structLiteralFields collects, per canonical class, the exported struct
	// FIELDS (own + promoted through the embed chain) of the Go struct behind it.
	// For a plain data struct with no NewX factory, the composite struct literal
	// IS Go's construction mechanism and those exported fields ARE its named
	// configurable set — the third construction source (see buildConstruction).
	// Collected here because the embed chain lives in `structs`, which is not
	// reachable from the emitted JSON.
	structLiteralFields := map[string]map[string]constructionParam{}
	collectStructLiteralFields := func(target surfacepkg.ClassTarget, facts *goStructFacts) {
		qn := target.Module + "." + target.Class
		dst, ok := structLiteralFields[qn]
		if !ok {
			dst = map[string]constructionParam{}
			structLiteralFields[qn] = dst
		}
		// factoryRequired: the names a `New<Struct>(...)` factory MAKES required.
		// The relay events are the canonical case — every one is built by
		// `New<X>Event(params map[string]any)` (event.go), so `params` must be
		// supplied and `event_type` is supplied BY the factory (it passes the
		// EventCalling* constant). Both are required in Go exactly as they are in
		// the reference ctor; without this the fallback would mark every field
		// optional and manufacture ~50 bogus required-flips.
		factoryRequired := map[string]bool{}
		if fn, ok := funcs[facts.pkg+".New"+facts.name]; ok {
			for _, p := range fn.params {
				factoryRequired[goFieldToPython(p.name)] = true
			}
			// A factory that takes the raw payload map builds the typed value from
			// it; the discriminator it hard-codes is not caller-optional either.
			if len(fn.params) == 1 && strings.HasPrefix(fn.params[0].typeStr, "map[string]") {
				factoryRequired["event_type"] = true
			}
		}
		var walkFields func(f *goStructFacts, depth int)
		seen := map[string]bool{}
		walkFields = func(f *goStructFacts, depth int) {
			if f == nil || depth > 4 || seen[f.pkg+"."+f.name] {
				return
			}
			seen[f.pkg+"."+f.name] = true
			for goField, fSig := range f.methods {
				if !fSig.isField {
					continue
				}
				name := applyConstructionGlobalRename(qn, goFieldToPython(goField))
				if _, already := dst[name]; already {
					continue
				}
				t, fail := translateType(fSig.returns, aliases, qn+"."+name)
				if fail != nil || t == "" {
					t = "any"
				}
				// A struct-literal field is optional by construction — Go
				// zero-values every field the literal omits, exactly as a Python
				// defaulted kwarg does — UNLESS the type's factory demands it.
				dst[name] = constructionParam{Type: t, Required: factoryRequired[name]}
			}
			// Promoted fields: an embedded struct's exported fields are reachable
			// (and settable, via the embedded literal) on the outer struct.
			for _, emb := range f.embeds {
				short := strings.TrimPrefix(emb, "*")
				if i := strings.LastIndex(short, "."); i >= 0 {
					short = short[i+1:]
				}
				if next, ok := structs[f.pkg+"."+short]; ok {
					walkFields(next, depth+1)
				}
			}
		}
		walkFields(facts, 0)
		if len(dst) == 0 {
			delete(structLiteralFields, qn)
		}
	}

	// --- 1. Project struct methods onto Python classes ---
	for key, facts := range structs {
		targets, ok := structTable[key]
		if !ok {
			continue
		}
		for _, target := range targets {
			collectStructLiteralFields(target, facts)
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

			// ai_chat data/error struct __init__: Go builds each via a composite
			// struct literal (no exported NewX factory), so the reference's
			// generated-dataclass / exception auto-ctor is projected from the
			// struct fields (aiChatCtorSigs) rather than an AST method.
			emitAIChatCtor(target.Module, target.Class)

			// @dataclass field emission (oracle-gated). For the closed set of
			// modules whose reference classes carry public @dataclass fields
			// (relay Event structs, AI-Chat DTOs, RequestOptions), project each
			// exported Go struct field whose snake_case name is in the SIGNATURE
			// oracle's member set for this (module, class) as a self-only accessor
			// returning the field's translated type — mirroring the reference's
			// dataclass-field-as-attribute projection. These are PRIMITIVE-typed
			// (string/int/float/dict/optional<bool>), so the composition loop below
			// (SDK-class-only) skips them; this pass emits them so the DRIFT gate
			// sees the same 106 members the surface audit now folds. Gating on the
			// oracle guarantees exactly the reference set — never a helper field.
			if clsMembers, ok := sigOracle[target.Module][target.Class]; ok {
				for goField, fSig := range facts.methods {
					// A public data FIELD, or a zero-arg READ ACCESSOR over an
					// unexported one — both are Go's expression of a public
					// attribute, and both have the same shape here (no params,
					// one return). The ACCESSOR arm is the signature-side twin of
					// enumerate-surface's accessor fold; without it a member folds
					// in SURFACE-DIFF and reds in SIGNATURES-DIFF.
					if !fSig.isField && len(fSig.params) != 0 {
						continue
					}
					if !fSig.isField && fSig.returns == "" {
						continue // void method, not a reader
					}
					snake := goFieldToPython(goField)
					if _, already := target.Methods[goField]; already {
						continue
					}
					if !clsMembers[snake] {
						continue
					}
					// A field whose Go type is a GENUINE idiom divergence from the
					// reference dataclass field type is NOT emitted here — the port
					// carries the MEMBER (surface folds it) but its declared type has
					// no faithful reference-shaped signature, so it stays an annotated
					// PORT_SIGNATURE_OMISSIONS.md entry (surface-live / signature-
					// divergent, per the DUAL-GATE rule). Two such fields:
					//   - RequestOptions.AbortSignal (context.Context): Go's
					//     cancellation primitive, not the reference's _AbortSignal
					//     object — already a documented signature omission.
					//   - RequestOptions.RetryOnStatus (map[int]bool): Go's SET idiom
					//     for the retry statuses; the reference types it list[int].
					//     A set-as-map vs list is a shape divergence, not a matching
					//     signature.
					if isSignatureDivergentDataclassField(fSig.returns) {
						continue
					}
					sig, fails := toCanonicalSignature(fSig, aliases, true, false, fmt.Sprintf("%s.%s.%s", target.Module, target.Class, snake))
					failures = append(failures, fails...)
					addClassMethod(target.Module, target.Class, snake, sig)
				}
			}

			// Auto-emit exported fields whose type is an SDK class
			// (``*namespaces.FabricNamespace``, ``*FooClient``, etc.)
			// as zero-arg accessor methods. Mirrors the Python reference
			// adapter's instance-attribute projection for the same
			// composition pattern (RestClient.fabric, RestClient.calling).
			//
			// The field set is the PROMOTED one (own + embed chain), because Go
			// expresses "this client exposes these namespaces" by embedding the
			// generated resource tree — see promotedFieldSet.
			for goField, fSig := range promotedFieldSet(structs, facts) {
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
				// context.Context is not an SDK class — it is the abort-signal
				// primitive (RequestOptions.AbortSignal, plan 4.2). A ctx-typed
				// field must not be projected as an accessor (the ctx->timeout
				// type fold would misreport its return); the reference abort_signal
				// accessor is a signature-only idiom divergence
				// (PORT_SIGNATURE_OMISSIONS.md).
				// The REST transport handle (`Resource.HTTP`) is plumbing, not a
				// sub-resource accessor — the reference keeps the same handle
				// private as `_client`. Go promotes it onto every generated
				// resource through the embedded `Resource` base, so it must be
				// filtered here or it manufactures a bogus `http` accessor on
				// every resource class. See isTransportHandleType.
				if isTransportHandleType(fSig.returns) {
					continue
				}
				if ret == "context.Context" {
					continue
				}
				// http.Header is a stdlib type (the captured response headers on
				// SignalWireRestError, plan 6.6), NOT an SDK class — it is a data
				// field like the URL/Body/StatusCode primitives, not a sub-resource
				// accessor. Python models it as the instance attribute `.headers`
				// (not a method), which the surface oracle does not record, so the Go
				// field must not project as an accessor either (same reasoning as the
				// context.Context exclusion above).
				if ret == "http.Header" {
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

	// --- 1b. Built-in skill contract methods (SkillContractTable) ---
	//
	// Every Go built-in *Skill struct (pkg/skills/builtin/*.go, plus spider's own
	// sub-package) embeds `skills.BaseSkill` and OVERRIDES a subset of the
	// SkillBase contract; the rest is PROMOTED from BaseSkill. So the concrete
	// struct genuinely PROVIDES every contract method the reference records for
	// it — Go's embedding is the same inheritance the reference expresses with a
	// Python base class.
	//
	// This projection was previously SURFACE-ONLY (cmd/enumerate-surface consumes
	// SkillContractTable; the signature side had hand-written StructTable entries
	// for exactly TWO skills, MCPGatewaySkill and SpiderSkill). The premise, stated
	// in tables.go, was that "the signature enumerator projects a concrete builtin
	// skill package ONLY where the reference signature oracle records members for
	// it" — true when the oracle recorded 7 of 18 skill modules, FALSE since
	// porting-sdk 8496c77 (2026-07-30) fixed the oracle bug where a class whose
	// every method was a base-identical override vanished entirely. The oracle now
	// records 18 of 18, and the two axes must agree: SURFACE and SIGNATURE read the
	// SAME SkillContractTable, so a skill can never be present on one axis and
	// absent on the other again.
	//
	// The signature comes from the struct's own override when it declares one, else
	// from the embedded `skills.BaseSkill` default (the mirror of what Go's method
	// set actually resolves at a call site). BaseSkill is a QUALIFIED cross-package
	// embed, which resolvePromotedMethod's same-package embed walk cannot follow, so
	// it is resolved explicitly here from the `skills` package structs.
	//
	// ORACLE-GATED, and that gate is load-bearing. The SURFACE oracle records all
	// 18 skills' full contract sets; the SIGNATURE oracle does NOT — 8496c77 fixed
	// only the case where a base-identical-override skip erased the WHOLE class, and
	// left the M2 case (a base-identical override on a class that retains OTHER
	// members) still dropped, explicitly flagged in that commit as needing its own
	// ruling. Five skills (api_ninjas_trivia, play_background_file, spider,
	// weather_api, wikipedia_search) survived the old bug only because each happens
	// to declare one signature-DIFFERING member (`get_tools` / `remove_xpaths` /
	// `search_wiki`), so their `register_tools`/`setup`/… are still absent from the
	// signature oracle while present in the surface oracle. Emitting the full
	// SkillContractTable set unconditionally would manufacture port-only signature
	// symbols the reference does not record. So the surface axis emits the full set
	// (its oracle has it) and the signature axis emits the intersection with what the
	// signature oracle records — the two axes stay driven by ONE table, each gated by
	// its own oracle.
	//
	// Fail-loud: a contract leaf that is neither declared on the struct nor provided
	// by BaseSkill is a renamed/removed skill member, not something to skip — it
	// panics, exactly as the surface side does.
	{
		baseSkill := structs["skills.BaseSkill"]
		if baseSkill == nil {
			panic("enumerate-signatures: skills.BaseSkill not found in walk (skill contract projection needs its promoted defaults)")
		}
		for _, sc := range surfacepkg.SkillContractTable {
			facts, ok := structs[sc.GoStruct]
			if !ok {
				panic(fmt.Sprintf("enumerate-signatures: skill struct %q in SkillContractTable not found in walk", sc.GoStruct))
			}
			oracleMembers := sigOracle[sc.Module][sc.ClassName]
			for _, leaf := range sc.Methods {
				if !oracleMembers[leaf] {
					continue
				}
				goMethod := surfacepkg.SkillLeafToGoMethod(leaf)
				mSig, declared := facts.methods[goMethod]
				if !declared {
					if !surfacepkg.BaseSkillProvides[goMethod] {
						panic(fmt.Sprintf("enumerate-signatures: skill %s expects Go method %q (for %q) but it is neither declared nor a BaseSkill default", sc.GoStruct, goMethod, leaf))
					}
					mSig = baseSkill.methods[goMethod]
				}
				if mSig == nil {
					panic(fmt.Sprintf("enumerate-signatures: skill %s expects Go method %q (for %q) but it is neither declared nor a BaseSkill default", sc.GoStruct, goMethod, leaf))
				}
				sig, fails := toCanonicalSignature(mSig, aliases, true, false, fmt.Sprintf("%s.%s.%s", sc.Module, sc.ClassName, leaf))
				failures = append(failures, fails...)
				addClassMethod(sc.Module, sc.ClassName, leaf, sig)
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
	out.Construction = buildConstruction(out.Modules, funcs, structLiteralFields, aliases)
	return out, failures
}

// ---------------------------------------------------------------------------
// THE CONSTRUCTION CONTRACT (porting-sdk ALLOWLIST_DISCIPLINE.md §10)
// ---------------------------------------------------------------------------
//
// Go expresses a wide many-optional-kwarg Python constructor as a NewX FACTORY
// plus a family of WithX FUNCTIONAL OPTIONS:
//
//     agent.NewAgentBase(agent.WithName("x"), agent.WithRecordStereo(true), ...)
//
// Both halves are the construction parameter set — the factory's own params are
// the required ones, each WithX names one optional configurable. That is the
// same capability as Python's `AgentBase(name="x", record_stereo=True, ...)`,
// spelled the way Go spells it, so it SATISFIES the contract rather than being
// N port-only additions plus one blanket `__init__` signature omission.
//
// Why a name-keyed node instead of comparing `__init__` as a method: the
// signature diff matches params BY POSITION and deliberately ignores names, and
// position-matching a 22-param kwargs ctor against `NewAgentBase(opts ...Option)`
// is meaningless — so the only available move was one omission per class, which
// stops comparing all 22 params at once and hides a dropped `record_stereo`
// forever. Order/arity/mechanism are exactly what idiom is entitled to vary; the
// NAMED SET is the capability.

// optionConstructs binds a functional-option TYPE (declared as
// `type <T> func(*<Struct>)` in package <pkg>) to the canonical class it
// constructs. Only CONSTRUCTION options belong here: the relay per-verb option
// types (PlayOption/TTSOption/RecordOption/...) configure a METHOD CALL, not an
// object's construction, and are deliberately absent.
//
// Key is `<pkg>.<OptionType>` — the same `pkgName + "." + name` key space the
// AST walker uses for free functions — because two packages both spell their
// option type `Option` (aichat, security).
var optionConstructs = map[string]string{
	"agent.AgentOption":             "signalwire.core.agent_base.AgentBase",
	"swml.ServiceOption":            "signalwire.core.swml_service.SWMLService",
	"relay.ClientOption":            "signalwire.relay.client.RelayClient",
	"server.ServerOption":           "signalwire.agent_server.AgentServer",
	"aichat.Option":                 "signalwire.ai_chat.client.AIChatClient",
	"security.Option":               "signalwire.core.security.session_manager.SessionManager",
	"security.ConfigOption":         "signalwire.core.security_config.SecurityConfig",
	"swml.SchemaUtilsOption":        "signalwire.utils.schema_utils.SchemaUtils",
	"pom.SectionOption":             "signalwire.pom.pom.Section",
	"contexts.GatherQuestionOption": "signalwire.core.contexts.GatherQuestion",
	"prefabs.SurveyQuestionOption":  "signalwire.prefabs.survey.SurveyQuestion",
}

// optionParamRenames canonicalizes a WithX option's derived param name onto the
// reference spelling (ADAPTER_CONTRACT rule 3 — "translate names to
// Python-canonical form at adapter time"). Name-keyed matching gives names
// weight they did not carry under positional matching, so a genuine
// port-vs-reference spelling difference is a RENAME here, never an omission
// (§7). Each entry is verified against the Go option's BODY — it must write the
// same state the reference param configures, not merely look similar.
//
// Key is `<canonical class>.<derived name>` so a rename is scoped to the class
// whose contract it belongs to.
var optionParamRenames = map[string]string{ //nolint:gosec // G101: these are PARAMETER NAMES in a rename table (…AgentBase.token_expiry), not credentials — gosec matches on "token" in the key.
	// WithTokenExpiry(secs int) -> a.tokenExpirySecs (agent.go:192).
	"signalwire.core.agent_base.AgentBase.token_expiry": "token_expiry_secs",
	// WithSigningKeyTrustProxy(trust bool) -> a.signingKeyTrustProxy, the
	// X-Forwarded-Proto/Host honoring used during signature URL reconstruction
	// (agent.go:309-320) == the reference's trust_proxy_for_signature.
	"signalwire.core.agent_base.AgentBase.signing_key_trust_proxy": "trust_proxy_for_signature",
	// WithJWT(jwt string) -> the RELAY JWT credential (options.go).
	"signalwire.relay.client.RelayClient.jwt": "jwt_token",
	// server.WithServerHost/WithServerPort are prefixed only to disambiguate
	// them from the RunOption pair in the same package; they configure the
	// AgentServer's listen address == the reference's host/port.
	"signalwire.agent_server.AgentServer.server_host": "host",
	"signalwire.agent_server.AgentServer.server_port": "port",
	// WithSecret(key []byte) -> the SessionManager HMAC secret (session_manager.go:40).
	"signalwire.core.security.session_manager.SessionManager.secret": "secret_key",
	// The reference Section ctor spells this one in camelCase (it is a POM
	// wire key, not a Python identifier convention).
	"signalwire.pom.pom.Section.numbered_bullets": "numberedBullets",
	// swml.WithSchemaUtilsPath / WithSchemaUtilsValidation carry the
	// `SchemaUtils` infix only to disambiguate them from the ServiceOption pair
	// of the same plain name in the same package (service.go WithSchemaPath /
	// WithSchemaValidation); they configure SchemaUtils' schema_path /
	// schema_validation exactly as the reference names them.
	"signalwire.utils.schema_utils.SchemaUtils.schema_utils_path":       "schema_path",
	"signalwire.utils.schema_utils.SchemaUtils.schema_utils_validation": "schema_validation",
}

// ctorOptionsStructConstructs binds a CONSTRUCTOR options struct (`<pkg>.<Name>`
// in ctorOptionsStructFields) to the canonical class its factory builds. Only
// the ctor-shaped ones belong here: `rest.RequestOptions` is a real REFERENCE
// parameter (`request_options`), not an unfold, and the SWML per-verb
// PlayOptions/AIOptions configure a METHOD (they are already unfolded by
// optionsStructUnfoldMethods), so neither appears.
var ctorOptionsStructConstructs = map[string]string{
	"prefabs.BedrockOptions":      "signalwire.agents.bedrock.BedrockAgent",
	"prefabs.ConciergeOptions":    "signalwire.prefabs.concierge.ConciergeAgent",
	"prefabs.FAQBotOptions":       "signalwire.prefabs.faq_bot.FAQBotAgent",
	"prefabs.InfoGathererOptions": "signalwire.prefabs.info_gatherer.InfoGathererAgent",
	"prefabs.ReceptionistOptions": "signalwire.prefabs.receptionist.ReceptionistAgent",
	"prefabs.SurveyOptions":       "signalwire.prefabs.survey.SurveyAgent",
	"web.Options":                 "signalwire.web.web_service.WebService",
}

// ctorOptionsFieldRenames canonicalizes an options-struct FIELD name onto the
// reference spelling (ADAPTER_CONTRACT rule 3). Same discipline as
// optionParamRenames: verified against the field's use, and a rename rather than
// an omission (§7). Keyed `<canonical class>.<derived name>`.
var ctorOptionsFieldRenames = map[string]string{
	// SurveyOptions.Intro is the survey's opening script — the reference spells
	// the same slot `introduction` (survey.go:121,151 `intro := opts.Intro` ->
	// `introduction: intro`).
	"signalwire.prefabs.survey.SurveyAgent.intro": "introduction",
	// ConciergeOptions.Hours is documented in-source as "general hours of
	// operation" == the reference's hours_of_operation.
	"signalwire.prefabs.concierge.ConciergeAgent.hours": "hours_of_operation",
}

// ctorOptionsFieldPairs folds a pair of Go options-struct fields into ONE
// reference param whose type is a TUPLE. Go has no tuple literal, so a
// reference `basic_auth: optional<tuple<string,string>>` is idiomatically two
// string fields; that is the same capability spelled the way Go spells it, so
// it FOLDS (§0) rather than reading as a dropped param plus two extras. Keyed
// `<canonical class>.<first derived name>` -> the second field's derived name
// and the folded param.
var ctorOptionsFieldPairs = map[string]struct {
	With  string // the derived name of the second field
	As    string // the canonical reference param they fold into
	Type  string // the canonical (tuple) type
	Order int    // 0 = this key is the first tuple member
}{
	"signalwire.web.web_service.WebService.basic_auth_user": {
		With: "basic_auth_password", As: "basic_auth",
		Type: "optional<tuple<string,string>>",
	},
}

// ctorOptionsFieldMechanism are options-struct fields that are the Go MECHANISM
// of construction, not a construction PARAMETER. `AgentOptions []agent.AgentOption`
// on a prefab is the pass-through of the PARENT AgentBase option family (already
// enumerated on AgentBase itself), not a configurable of its own.
var ctorOptionsFieldMechanism = map[string]bool{
	"agent_options": true,
}

// constructionGlobalRenames canonicalizes a construction param name onto the
// reference spelling REGARDLESS of class — for a spelling difference that is
// systematic across a whole generated family, where per-class entries would be
// the same fact restated 56 times. ADAPTER_CONTRACT rule 3; a rename, never an
// omission (§7). Each is scoped by constructionGlobalRenameClasses so it cannot
// bleed onto an unrelated class that happens to use the same word.
var constructionGlobalRenames = map[string]string{
	// Every generated REST namespace/resource takes the shared HTTP transport as
	// `client HTTPClient` (calling_resources_generated.go:18
	// `func NewCallingNamespace(client HTTPClient)`); the reference names the same
	// slot `http`. Same object, same position, different spelling — 56 classes.
	"client": "http",
	// rest._base.BaseResource's `Base` field holds the resource's URL path
	// prefix == the reference's `base_path`.
	"base": "base_path",
	// The REST constructors take the per-request transport envelope as a
	// variadic `opts ...*RequestOptions` (client.go:179 NewHTTPClient); the
	// reference names the same object `request_options`. It is a real reference
	// PARAMETER, not an option bag — hence a rename here rather than the
	// mechanism skip.
	"opts": "request_options",
	// NewHTTPClient(projectID, ...) — the SignalWire project UUID, spelled
	// `project` by the reference (and by RestClient in this same port).
	"project_id": "project",
}

var constructionGlobalRenameClasses = map[string][]string{
	"client":     {"signalwire.rest."},
	"base":       {"signalwire.rest._base.BaseResource"},
	"opts":       {"signalwire.rest."},
	"project_id": {"signalwire.rest."},
}

// applyConstructionGlobalRename returns the canonical spelling of a derived
// param name for a class, or the name unchanged.
func applyConstructionGlobalRename(class, name string) string {
	renamed, ok := constructionGlobalRenames[name]
	if !ok {
		return name
	}
	for _, prefix := range constructionGlobalRenameClasses[name] {
		if strings.HasPrefix(class, prefix) {
			return renamed
		}
	}
	return name
}

// isConstructionMechanismParam reports whether an `__init__` param is the Go
// MECHANISM of construction rather than a construction PARAMETER: the receiver,
// the variadic `opts ...XOption` bag (every option in it is emitted by name from
// source (2)), and a bound `opts XOptions` struct (unfolded to its fields by
// source (1b)). A `class:...RequestOptions` param is NEITHER — it is the
// reference's own `request_options` param — so the suffix test is deliberately
// `Option>` / an explicitly-bound options struct, not any "Options" type.
func isConstructionMechanismParam(p canonicalParam) bool {
	switch p.Kind {
	case "self", "cls", "var_keyword", "var_positional":
		return true
	}
	if p.Name == "ctx" {
		// Go's leading `ctx context.Context` is the cancellation/deadline
		// MECHANISM, not a configurable of the object being built. The
		// signature differ already absorbs it wholesale (porting-sdk 21d67ad,
		// "Leading ctx reconciliation (Go idiom)"); the construction node holds
		// the same line.
		return true
	}
	if p.Name != "opts" && p.Name != "options" {
		return false
	}
	if strings.HasPrefix(p.Type, "list<class:") && strings.HasSuffix(p.Type, "Option>") {
		return true
	}
	leaf := strings.TrimSuffix(strings.TrimPrefix(p.Type, "class:"), ">")
	if i := strings.LastIndex(leaf, "."); i >= 0 {
		leaf = leaf[i+1:]
	}
	for k := range ctorOptionsStructConstructs {
		if k[strings.LastIndex(k, ".")+1:] == leaf {
			return true
		}
	}
	return false
}

// buildConstruction returns {"module.Class": {"params": {name: {type, required}}}}.
// Four sources, merged (factory params win over option funcs, because a factory
// param is genuinely required while an option is optional by construction):
//
//  1. the class's already-emitted `__init__` (the NewX factory), minus the
//     receiver and the `opts ...Option` variadic tail;
//  2. every `WithX` free function whose RETURN type is an option type bound in
//     optionConstructs — `WithRecordStereo(stereo bool) AgentOption` names the
//     `record_stereo` construction param of AgentBase;
//  3. for a plain data struct with NO factory and NO options (the relay event
//     structs), the exported struct FIELDS — own and promoted through the embed
//     chain — because the composite struct literal is Go's construction
//     mechanism for such a type and its exported fields are its named
//     configurable set. Scoped to classes sources (1)/(2) did not supply, so a
//     class with a real factory is never diluted by its internal state fields.
//
// `required` is deliberately compared by the diff and must not vary between
// ports (owner ruling 2026-07-24). A Go functional option is optional by
// construction, so where the reference marks a param required the diff raises a
// construction-required-flip rather than silently accepting an under-specified
// construction. That is a finding to report, not something to paper over here.
func buildConstruction(modules map[string]sigModuleInventory, funcs map[string]*goFunc, structLiteralFields map[string]map[string]constructionParam, aliases map[string]string) map[string]constructionEntry {
	out := map[string]constructionEntry{}

	ensure := func(qn string) map[string]constructionParam {
		e, ok := out[qn]
		if !ok {
			e = constructionEntry{Params: map[string]constructionParam{}}
			out[qn] = e
		}
		return e.Params
	}

	// (1) NewX factory params, via the already-canonical __init__ signature.
	for mod, inv := range modules {
		for cls, ce := range inv.Classes {
			init, ok := ce.Methods["__init__"]
			if !ok {
				continue
			}
			var params map[string]constructionParam
			for _, p := range init.Params {
				if isConstructionMechanismParam(p) || p.Name == "" ||
					strings.HasPrefix(p.Name, "_") {
					continue
				}
				if params == nil {
					params = ensure(mod + "." + cls)
				}
				required := true
				if p.Required != nil {
					required = *p.Required
				}
				t := p.Type
				if t == "" {
					t = "any"
				}
				name := applyConstructionGlobalRename(mod+"."+cls, p.Name)
				params[name] = constructionParam{Type: t, Required: required}
			}
		}
	}

	// (1b) CONSTRUCTOR OPTIONS-STRUCT unfold. `NewSurveyAgent(SurveyOptions{...})`
	// passes ONE struct where the reference takes named kwargs; its exported
	// fields ARE the construction params (§10 names this mechanism explicitly).
	// Unfolded here rather than in `__init__` because the signature side keeps
	// the honest Go call shape.
	structKeys := make([]string, 0, len(ctorOptionsStructConstructs))
	for k := range ctorOptionsStructConstructs {
		structKeys = append(structKeys, k)
	}
	sort.Strings(structKeys)
	for _, sk := range structKeys {
		target := ctorOptionsStructConstructs[sk]
		fields, ok := ctorOptionsStructFields[sk]
		if !ok {
			continue
		}
		params := ensure(target)
		derived := map[string]paramsStructField{}
		for _, f := range fields {
			derived[goFieldToPython(f.name)] = f
		}
		for _, f := range fields {
			name := goFieldToPython(f.name)
			if ctorOptionsFieldMechanism[name] {
				continue
			}
			if pair, ok := ctorOptionsFieldPairs[target+"."+name]; ok {
				if _, has := derived[pair.With]; has {
					params[pair.As] = constructionParam{Type: pair.Type, Required: false}
					continue
				}
			}
			// The SECOND member of a folded pair is consumed by the fold above.
			folded := false
			for k, pair := range ctorOptionsFieldPairs {
				if strings.HasPrefix(k, target+".") && pair.With == name {
					folded = true
					break
				}
			}
			if folded {
				continue
			}
			if renamed, ok := ctorOptionsFieldRenames[target+"."+name]; ok {
				name = renamed
			}
			t, fail := translateType(f.typeStr, aliases, target+"."+name)
			if fail != nil || t == "" {
				t = "any"
			}
			// A zero-valued options-struct field falls back to the reference
			// default (see each NewX), so every field is optional by construction.
			params[name] = constructionParam{Type: t, Required: false}
		}
	}

	// (2) WithX functional options, bound to their target class by option type.
	names := make([]string, 0, len(funcs))
	for k := range funcs {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, key := range names {
		fn := funcs[key]
		if !strings.HasPrefix(fn.name, "With") || len(fn.name) <= len("With") {
			continue
		}
		target, ok := optionConstructs[fn.pkg+"."+fn.returns]
		if !ok {
			continue
		}
		pname := goFieldToPython(fn.name[len("With"):])
		if renamed, ok := optionParamRenames[target+"."+pname]; ok {
			pname = renamed
		}
		params := ensure(target)
		// A factory param already recorded is the authoritative one (it carries
		// the real required flag); an option func only fills a gap.
		if _, exists := params[pname]; exists {
			continue
		}
		params[pname] = constructionParam{
			Type:     optionParamType(fn, aliases, target+"."+pname),
			Required: false,
		}
	}

	// (3) Struct-literal construction, for classes neither a factory nor an
	// option family supplied. A struct with a NewX factory already has its
	// authoritative param set from (1); adding its exported state fields there
	// would report internal state as configurable, so this is a FALLBACK only.
	for qn, fields := range structLiteralFields {
		if _, already := out[qn]; already {
			continue
		}
		params := ensure(qn)
		for name, spec := range fields {
			params[name] = spec
		}
	}

	for qn, e := range out {
		if len(e.Params) == 0 {
			delete(out, qn)
		}
	}
	return out
}

// optionParamType renders the canonical type a WithX option configures.
//
//   - zero args (a flag-style option such as WithOptional() / WithEnvDefaults())
//     configures a boolean;
//   - one arg is that arg's translated type;
//   - two-or-more args are the Go spelling of a reference TUPLE — Go has no
//     tuple literal, so `WithBasicAuth(user, password string)` is the idiomatic
//     spelling of the reference's `basic_auth: optional<tuple<string,string>>`.
//     Emitting it as a tuple is what makes the two compare EQUAL instead of
//     reading as a dropped capability.
func optionParamType(fn *goFunc, aliases map[string]string, ctx string) string {
	parts := make([]string, 0, len(fn.params))
	for _, p := range fn.params {
		canon, fail := translateType(p.typeStr, aliases, ctx)
		if fail != nil || canon == "" {
			canon = "any"
		}
		parts = append(parts, canon)
	}
	switch len(parts) {
	case 0:
		return "bool"
	case 1:
		return parts[0]
	default:
		return "tuple<" + strings.Join(parts, ",") + ">"
	}
}

// ---------------------------------------------------------------------------
// Repo helpers
// ---------------------------------------------------------------------------

func goSHA(repo string) string {
	cmd := exec.Command("git", "-C", repo, "rev-parse", "HEAD") //nolint:gosec // G204: fixed program "git" with a repo path the developer already controls.
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
		// Fail-loud is the DEFAULT, not an opt-in. As `false` this flag was dead
		// code: the usage header advertised it, but no gate ever passed it, so a
		// type that failed to translate silently DROPPED THE WHOLE SYMBOL and the
		// artifact was written anyway at exit 0 -- the port then got blamed for an
		// omission it never had. Use -strict=false to opt out explicitly.
		strict     = flag.Bool("strict", true, "Exit non-zero on any translation failure (default true)")
		stdoutFlag = flag.Bool("stdout", false, "Print to stdout")
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
		// Same three-layout resolution as resolvePortingSDK: $PORTING_SDK, the
		// sibling dev layout, then the CI in-repo layout. This one already failed
		// loud below, which is why it never produced a degraded snapshot the way the
		// oracle loaders did — but it must know about all three layouts too.
		candidates := []string{}
		if env := os.Getenv("PORTING_SDK"); env != "" {
			candidates = append(candidates, filepath.Join(env, "type_aliases.yaml"))
		}
		candidates = append(candidates,
			filepath.Join(repoRoot, "..", "porting-sdk", "type_aliases.yaml"),
			filepath.Join(repoRoot, "porting-sdk", "type_aliases.yaml"),
		)
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil { //nolint:gosec // G703: path is composed from the repo root / $PORTING_SDK in a developer-run tool, not from untrusted input.
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

	sigOracle, err := loadSigOracle(repoRoot)
	if err != nil {
		return fmt.Errorf("load reference signature oracle: %w", err)
	}
	doc, failures := build(structs, funcs, payloads, aliases, sigOracle)
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
	return os.WriteFile(*outputPath, rendered, 0o644) //nolint:gosec // G306: generated SOURCE CODE is committed to the repo and must be world-readable; 0600 would break every consumer.
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
