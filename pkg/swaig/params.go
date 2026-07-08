package swaig

// params.go — a typed, fluent builder for SWAIG tool parameters.
//
// Background. agent.ToolDefinition carries the tool's argument schema as two
// loosely-typed fields:
//
//	Parameters map[string]any // a JSON-Schema *properties* map
//	Required   []string       // the top-level *required* list
//
// Hand-writing that properties map means nesting map[string]any literals by
// hand, e.g.
//
//	Parameters: map[string]any{
//	    "service": map[string]any{"type": "string", "description": "The service"},
//	    "date":    map[string]any{"type": "string", "description": "YYYY-MM-DD"},
//	},
//	Required: []string{"service", "date"},
//
// which is easy to mistype ("tpye"), hard to read, and gives the Go compiler no
// help. Params is a fluent builder that produces the *byte-identical* same two
// values type-safely:
//
//	p := swaig.NewParams().
//	    String("service", "The service to check").
//	    String("date", "The date (YYYY-MM-DD)").
//	    Enum("fmt", swaig.RecordFormatValues(), "Recording format").
//	    Required("service", "date")
//
//	agent.ToolDefinition{
//	    Name:       "check",
//	    Parameters: p.Properties(), // map[string]any  — the properties map
//	    Required:   p.Required(),   // []string        — the required list
//	}
//
// This is a typed convenience over the SAME wire shape, NOT a new format: the
// untyped Parameters/Required path stays fully working and unchanged. The
// builder is purely additive (a Go-only PORT_ADDITION); it has no Python-
// reference counterpart, so it carries no signature/surface drift (the
// enumerators only project structs/funcs listed in the adapter rename tables,
// and Params is in neither — documented in PORT_ADDITIONS.md).
//
// Supported property kinds: String, Number, Integer, Boolean, Enum (a closed
// set, which integrates the Tier-1 typed enums via RecordFormatValues /
// RecordDirectionValues / TapDirectionValues / CodecValues), Array (of a kind)
// and Object (nested properties). Every kind accepts trailing PropOption values
// for description-adjacent attributes (default, format, an inline enum, and an
// inline Required marker).

// Params accumulates SWAIG tool-parameter properties and the top-level required
// list, then renders them into the exact map[string]any / []string pair that
// agent.ToolDefinition.Parameters and .Required expect.
//
// The zero value is not usable; construct with NewParams. Methods return the
// receiver so calls chain fluently. Params is not safe for concurrent mutation
// (build it on one goroutine, then read it).
type Params struct {
	// order preserves property insertion order so Properties() is
	// deterministic for callers that range it (JSON object key order is
	// insignificant on the wire, but determinism keeps tests and diffs
	// stable).
	order []string
	props map[string]*Prop
	// required holds the top-level required names in first-seen order,
	// deduplicated, accumulated from both Required(...) and any inline
	// PropRequired() options.
	required []string
	reqSeen  map[string]bool
	// description is the tool-level description this typed declaration carries,
	// surfaced by InferSchema (the Go analog of the docstring summary Python's
	// infer_schema derives from the handler). Empty means "no description".
	description string
	// hasRawData records whether the handler this declaration describes also
	// receives the SWAIG raw payload, surfaced by InferSchema's has_raw_data.
	hasRawData bool
}

// Describe sets the tool-level description carried by this typed declaration,
// surfaced by InferSchema as the schema's description. Returns the receiver for
// chaining.
func (b *Params) Describe(description string) *Params {
	b.description = description
	return b
}

// WithRawData marks that the handler this declaration describes also receives
// the SWAIG raw payload (the analog of Python's `raw_data` handler parameter),
// surfaced by InferSchema's has_raw_data return. Returns the receiver.
func (b *Params) WithRawData() *Params {
	b.hasRawData = true
	return b
}

// NewParams returns an empty parameter builder ready to accept property
// declarations.
func NewParams() *Params {
	return &Params{
		props:   map[string]*Prop{},
		reqSeen: map[string]bool{},
	}
}

// Prop is a single JSON-Schema property under construction. It is produced by
// the kind constructors (PropString, PropArray, PropObject, …) for use as the
// item schema of Array, and is otherwise an internal detail of Params. Build it
// via the Prop* constructors; the zero value is not usable.
type Prop struct {
	// schema is the property's JSON-Schema object, mutated in place by the
	// PropOption helpers. It always carries at least "type"; Enum/array/object
	// kinds add "enum"/"items"/"properties" as appropriate.
	schema map[string]any
	// markRequired is set by the Required() PropOption. Params.add / PropObject
	// read it after applying options to decide whether to add the property name
	// to a required list. It lives here (not in the schema map) so it never
	// reaches the wire.
	markRequired bool
}

// PropOption mutates a property's JSON-Schema object. Options are applied left
// to right after the kind and description are set, so a later option overrides
// an earlier one writing the same key.
type PropOption func(p *Prop)

// Default sets the property's JSON-Schema "default" keyword to v (emitted
// verbatim — pass the wire value).
func Default(v any) PropOption {
	return func(p *Prop) { p.schema["default"] = v }
}

// Format sets the property's JSON-Schema "format" keyword (e.g. "date",
// "date-time", "email").
func Format(format string) PropOption {
	return func(p *Prop) { p.schema["format"] = format }
}

// WithEnum constrains the property to the given closed set, setting the
// JSON-Schema "enum" keyword. Use it to attach an enum to a String/Integer/…
// property; the Enum kind constructor is the shorthand for a string enum.
func WithEnum(values ...string) PropOption {
	return func(p *Prop) { p.schema["enum"] = toAnySlice(values) }
}

// Required marks the enclosing property as required. On a top-level Params
// property it adds the name to the schema's top-level required array; inside an
// Object it adds to that object's required array. This is the per-property
// alternative to Params.Required(names...).
func Required() PropOption {
	return func(p *Prop) { p.markRequired = true }
}

// ---------------------------------------------------------------------------
// Kind constructors (standalone Prop values, e.g. for Array items)
// ---------------------------------------------------------------------------

// PropString returns a string-typed property with the given description.
func PropString(description string, opts ...PropOption) *Prop {
	return newProp("string", description, opts)
}

// PropNumber returns a number-typed (floating-point) property.
func PropNumber(description string, opts ...PropOption) *Prop {
	return newProp("number", description, opts)
}

// PropInteger returns an integer-typed property.
func PropInteger(description string, opts ...PropOption) *Prop {
	return newProp("integer", description, opts)
}

// PropBoolean returns a boolean-typed property.
func PropBoolean(description string, opts ...PropOption) *Prop {
	return newProp("boolean", description, opts)
}

// PropEnum returns a string-typed property constrained to the given closed set
// (JSON-Schema "enum"). values is typically one of RecordFormatValues(),
// RecordDirectionValues(), TapDirectionValues(), CodecValues(), or any caller
// list.
func PropEnum(values []string, description string, opts ...PropOption) *Prop {
	p := newProp("string", description, nil)
	p.schema["enum"] = toAnySlice(values)
	applyOpts(p, opts)
	return p
}

// PropArray returns an array-typed property whose elements match items.
func PropArray(items *Prop, description string, opts ...PropOption) *Prop {
	p := newProp("array", description, nil)
	if items != nil {
		p.schema["items"] = items.schema
	}
	applyOpts(p, opts)
	return p
}

// PropObject returns an object-typed property whose nested properties (and
// nested required list) come from nested.
func PropObject(nested *Params, description string, opts ...PropOption) *Prop {
	p := newProp("object", description, nil)
	if nested != nil {
		p.schema["properties"] = nested.Properties()
		if req := nested.requiredNames(); len(req) > 0 {
			p.schema["required"] = req
		}
	}
	applyOpts(p, opts)
	return p
}

// ---------------------------------------------------------------------------
// Fluent property declarations on Params
// ---------------------------------------------------------------------------

// String adds a string-typed property named name.
func (b *Params) String(name, description string, opts ...PropOption) *Params {
	return b.add(name, PropString(description, opts...))
}

// Number adds a number-typed (floating-point) property named name.
func (b *Params) Number(name, description string, opts ...PropOption) *Params {
	return b.add(name, PropNumber(description, opts...))
}

// Integer adds an integer-typed property named name.
func (b *Params) Integer(name, description string, opts ...PropOption) *Params {
	return b.add(name, PropInteger(description, opts...))
}

// Boolean adds a boolean-typed property named name.
func (b *Params) Boolean(name, description string, opts ...PropOption) *Params {
	return b.add(name, PropBoolean(description, opts...))
}

// Enum adds a string-typed property named name constrained to the closed set
// values (JSON-Schema "enum"). Pass RecordFormatValues() / RecordDirectionValues()
// / TapDirectionValues() / CodecValues() to wire one of the Tier-1 typed enums,
// or any caller list.
func (b *Params) Enum(name string, values []string, description string, opts ...PropOption) *Params {
	return b.add(name, PropEnum(values, description, opts...))
}

// Array adds an array-typed property named name whose elements match items.
func (b *Params) Array(name string, items *Prop, description string, opts ...PropOption) *Params {
	return b.add(name, PropArray(items, description, opts...))
}

// Object adds an object-typed property named name whose nested properties come
// from nested.
func (b *Params) Object(name string, nested *Params, description string, opts ...PropOption) *Params {
	return b.add(name, PropObject(nested, description, opts...))
}

// Required marks one or more already-declared (or yet-to-be-declared) property
// names as required at the top level. Duplicate names are ignored; first-seen
// order is preserved. Returns the receiver for chaining.
func (b *Params) Required(names ...string) *Params {
	for _, n := range names {
		b.markRequired(n)
	}
	return b
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

// Properties renders the accumulated properties as a JSON-Schema *properties*
// map, assignable directly to agent.ToolDefinition.Parameters. The result is a
// fresh map on each call (callers may mutate it freely); it is byte-identical to
// the equivalent hand-written map[string]any literal.
//
// A builder with no properties returns an empty (non-nil) map, so the caller can
// pass it straight through; assign nil explicitly if you want the "no schema"
// path (ToolDefinition omits the parameters block when Parameters is nil/empty).
func (b *Params) Properties() map[string]any {
	out := make(map[string]any, len(b.props))
	for name, p := range b.props {
		out[name] = p.schema
	}
	return out
}

// requiredNames is the shared getter behind RequiredNames and Build; it returns
// a fresh copy of the top-level required names in first-seen order, or nil when
// nothing is required.
func (b *Params) requiredNames() []string {
	if len(b.required) == 0 {
		return nil
	}
	out := make([]string, len(b.required))
	copy(out, b.required)
	return out
}

// RequiredNames returns the top-level required property names in first-seen
// order, assignable directly to agent.ToolDefinition.Required. It returns nil
// when nothing is required (matching a hand-written `Required: nil`); a fresh
// slice is returned on each call.
//
// (The fluent setter is Required(names ...string) *Params; this getter is named
// RequiredNames so the two don't collide on the method set.)
func (b *Params) RequiredNames() []string {
	return b.requiredNames()
}

// Build renders both halves at once: the properties map (for
// ToolDefinition.Parameters) and the required list (for ToolDefinition.Required).
// It is sugar for (b.Properties(), b.RequiredNames()):
//
//	params, required := swaig.NewParams().
//	    String("service", "The service").Required("service").Build()
//	td := agent.ToolDefinition{Name: "x", Parameters: params, Required: required}
func (b *Params) Build() (map[string]any, []string) {
	return b.Properties(), b.requiredNames()
}

// ---------------------------------------------------------------------------
// Enum value helpers — bridge the Tier-1 typed enums into the schema "enum"
// ---------------------------------------------------------------------------

// RecordFormatValues returns the RecordFormat closed set as wire strings
// (mp3, wav, mp4), suitable for Params.Enum / PropEnum / WithEnum. The values
// are derived from the typed constants, so adding a constant updates the list.
func RecordFormatValues() []string {
	return enumStrings(FormatMP3, FormatWAV, FormatMP4)
}

// RecordDirectionValues returns the RecordDirection closed set as wire strings
// (speak, listen, both), suitable for Params.Enum / PropEnum / WithEnum.
func RecordDirectionValues() []string {
	return enumStrings(RecordDirectionSpeak, RecordDirectionListen, RecordDirectionBoth)
}

// TapDirectionValues returns the TapDirection closed set as wire strings
// (speak, hear, both), suitable for Params.Enum / PropEnum / WithEnum.
func TapDirectionValues() []string {
	return enumStrings(TapDirectionSpeak, TapDirectionHear, TapDirectionBoth)
}

// CodecValues returns the Codec closed set as wire strings (PCMU, PCMA),
// suitable for Params.Enum / PropEnum / WithEnum.
func CodecValues() []string {
	return enumStrings(CodecPCMU, CodecPCMA)
}

// ---------------------------------------------------------------------------
// internals
// ---------------------------------------------------------------------------

// newProp builds a Prop with the given JSON-Schema type and description, then
// applies opts. A non-empty description sets the "description" keyword; an empty
// description omits it (matching hand-written maps that leave it out).
func newProp(kind, description string, opts []PropOption) *Prop {
	p := &Prop{schema: map[string]any{"type": kind}}
	if description != "" {
		p.schema["description"] = description
	}
	applyOpts(p, opts)
	return p
}

func applyOpts(p *Prop, opts []PropOption) {
	for _, o := range opts {
		if o != nil {
			o(p)
		}
	}
}

// add records a property under name (de-duplicating the order slice on
// overwrite) and honors an inline Required() marker.
func (b *Params) add(name string, p *Prop) *Params {
	if _, exists := b.props[name]; !exists {
		b.order = append(b.order, name)
	}
	b.props[name] = p
	if p.markRequired {
		b.markRequired(name)
	}
	return b
}

func (b *Params) markRequired(name string) {
	if b.reqSeen[name] {
		return
	}
	b.reqSeen[name] = true
	b.required = append(b.required, name)
}

// toAnySlice copies a []string into a []any so it serializes as a JSON array of
// strings — byte-identical to a hand-written []any{"a","b"} enum literal.
func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// enumStrings converts typed enum constants to their underlying wire strings.
func enumStrings[T ~string](vals ...T) []string {
	out := make([]string, len(vals))
	for i, v := range vals {
		out[i] = string(v)
	}
	return out
}
