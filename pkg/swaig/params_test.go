package swaig

import (
	"encoding/json"
	"reflect"
	"testing"
)

// jsonEqual reports whether two values serialize to identical JSON. JSON object
// key order is insignificant, so this is the strongest "same wire shape" check:
// it normalizes map ordering while still comparing every key, value, and the
// enum/required array element ORDER (JSON arrays are ordered).
func jsonEqual(t *testing.T, got, want any) {
	t.Helper()
	gb, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal got: %v", err)
	}
	wb, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal want: %v", err)
	}
	// Round-trip both through interface{} so map key order is normalized
	// before comparison (json.Marshal of a map sorts keys, but arrays stay
	// ordered — exactly the semantics we want).
	var gv, wv any
	if err := json.Unmarshal(gb, &gv); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	if err := json.Unmarshal(wb, &wv); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if !reflect.DeepEqual(gv, wv) {
		t.Errorf("JSON mismatch:\n got = %s\nwant = %s", gb, wb)
	}
}

// TestParamsByteIdenticalAllKinds is the headline proof: the builder's
// Properties()/RequiredNames() output is byte-identical (reflect.DeepEqual AND
// JSON-equal) to the equivalent hand-written map[string]any + []string, across
// EVERY property kind — String, Number, Integer, Boolean, Enum (driven by the
// Tier-1 RecordFormatValues), Array (of a kind), and a nested Object — plus the
// default/format/inline-required options.
func TestParamsByteIdenticalAllKinds(t *testing.T) {
	// ---- builder form -------------------------------------------------
	built := NewParams().
		String("service", "The service to check").
		Number("budget", "Max budget in dollars", Default(100.0)).
		Integer("party_size", "Number of guests").
		Boolean("vip", "Whether the guest is a VIP", Default(false)).
		Enum("fmt", RecordFormatValues(), "Recording format").
		Array("tags", PropString("a single tag"), "Free-form tags").
		Object("address", NewParams().
			String("street", "Street line").
			String("zip", "Postal code").
			Required("zip"),
			"Mailing address").
		String("date", "The date (YYYY-MM-DD)", Format("date"), Required())

	// "service" + "date" required (date inline via Required()); "service"
	// added through the variadic setter to exercise both code paths.
	built.Required("service")

	gotProps, gotReq := built.Build()

	// ---- hand-written form (the EXACT current wire shape) -------------
	wantProps := map[string]any{
		"service": map[string]any{
			"type":        "string",
			"description": "The service to check",
		},
		"budget": map[string]any{
			"type":        "number",
			"description": "Max budget in dollars",
			"default":     100.0,
		},
		"party_size": map[string]any{
			"type":        "integer",
			"description": "Number of guests",
		},
		"vip": map[string]any{
			"type":        "boolean",
			"description": "Whether the guest is a VIP",
			"default":     false,
		},
		"fmt": map[string]any{
			"type":        "string",
			"description": "Recording format",
			"enum":        []any{"mp3", "wav", "mp4"},
		},
		"tags": map[string]any{
			"type":        "array",
			"description": "Free-form tags",
			"items": map[string]any{
				"type":        "string",
				"description": "a single tag",
			},
		},
		"address": map[string]any{
			"type":        "object",
			"description": "Mailing address",
			"properties": map[string]any{
				"street": map[string]any{
					"type":        "string",
					"description": "Street line",
				},
				"zip": map[string]any{
					"type":        "string",
					"description": "Postal code",
				},
			},
			"required": []string{"zip"},
		},
		"date": map[string]any{
			"type":        "string",
			"description": "The date (YYYY-MM-DD)",
			"format":      "date",
		},
	}
	wantReq := []string{"date", "service"} // first-seen order: date (inline) then service

	// ---- byte-identical proofs ----------------------------------------
	if !reflect.DeepEqual(gotProps, wantProps) {
		t.Errorf("Properties() not DeepEqual to hand-written map.\n got = %#v\nwant = %#v", gotProps, wantProps)
	}
	jsonEqual(t, gotProps, wantProps)

	if !reflect.DeepEqual(gotReq, wantReq) {
		t.Errorf("RequiredNames() = %#v, want %#v", gotReq, wantReq)
	}
	jsonEqual(t, gotReq, wantReq)
}

// TestParamsEnumByteIdenticalAcrossAllTier1Enums proves the Enum kind threads
// each Tier-1 typed-enum value set straight into the schema "enum" array,
// byte-identically to the hand-written closed-set literal.
func TestParamsEnumByteIdenticalAcrossAllTier1Enums(t *testing.T) {
	cases := []struct {
		name   string
		values []string
		want   []any
	}{
		{"record_format", RecordFormatValues(), []any{"mp3", "wav", "mp4"}},
		{"record_direction", RecordDirectionValues(), []any{"speak", "listen", "both"}},
		{"tap_direction", TapDirectionValues(), []any{"speak", "hear", "both"}},
		{"codec", CodecValues(), []any{"PCMU", "PCMA"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NewParams().Enum("x", tc.values, "an enum field").Properties()
			want := map[string]any{
				"x": map[string]any{
					"type":        "string",
					"description": "an enum field",
					"enum":        tc.want,
				},
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("enum %s not DeepEqual.\n got = %#v\nwant = %#v", tc.name, got, want)
			}
			jsonEqual(t, got, want)
		})
	}
}

// TestParamsByteIdenticalAgainstConciergeTool proves the builder reproduces a
// REAL hand-written tool schema from the repo verbatim — the concierge
// check_availability tool (pkg/prefabs/concierge.go), which is three plain
// string properties. This guards against the builder drifting from the actual
// shapes shipped in the prefabs.
func TestParamsByteIdenticalAgainstConciergeTool(t *testing.T) {
	got := NewParams().
		String("service", "The service to check (e.g., spa, restaurant)").
		String("date", "The date to check (YYYY-MM-DD format)").
		String("time", "The time to check (HH:MM format, 24-hour)").
		Properties()

	// Copied verbatim from concierge.go:183-196.
	want := map[string]any{
		"service": map[string]any{
			"type":        "string",
			"description": "The service to check (e.g., spa, restaurant)",
		},
		"date": map[string]any{
			"type":        "string",
			"description": "The date to check (YYYY-MM-DD format)",
		},
		"time": map[string]any{
			"type":        "string",
			"description": "The time to check (HH:MM format, 24-hour)",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("concierge schema not reproduced.\n got = %#v\nwant = %#v", got, want)
	}
	jsonEqual(t, got, want)
}

// TestParamsEmptyAndNoRequired pins the no-property / no-required edges so the
// builder matches a hand-written `Parameters: nil`-style omission: empty (but
// non-nil) properties map, and a nil required slice.
func TestParamsEmptyAndNoRequired(t *testing.T) {
	p := NewParams()

	props := p.Properties()
	if props == nil {
		t.Fatal("Properties() should be non-nil empty map, got nil")
	}
	if len(props) != 0 {
		t.Errorf("expected 0 properties, got %d: %#v", len(props), props)
	}

	if req := p.RequiredNames(); req != nil {
		t.Errorf("RequiredNames() with nothing required should be nil, got %#v", req)
	}

	// A property declared without Required() must NOT appear in required.
	p.String("only", "the only field")
	if req := p.RequiredNames(); req != nil {
		t.Errorf("declaring a field must not make it required, got %#v", req)
	}
}

// TestParamsRequiredDedupAndOrder proves required-name accumulation
// deduplicates and preserves first-seen order across both the inline Required()
// option and the variadic Required(...) setter.
func TestParamsRequiredDedupAndOrder(t *testing.T) {
	p := NewParams().
		String("a", "field a", Required()).
		String("b", "field b").
		String("c", "field c", Required()).
		Required("b", "a", "c", "b") // duplicates of a,b,c + repeat b

	got := p.RequiredNames()
	want := []string{"a", "c", "b"} // a (inline), c (inline), then b (first new from setter)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("required order/dedup wrong: got %#v, want %#v", got, want)
	}
}

// TestParamsOverwritePreservesOrder proves redeclaring a property updates it in
// place without duplicating its key, so Properties() stays a clean map.
func TestParamsOverwritePreservesOrder(t *testing.T) {
	p := NewParams().
		String("x", "first version").
		Integer("x", "second version") // overwrite x with a different kind

	props := p.Properties()
	if len(props) != 1 {
		t.Fatalf("expected 1 property after overwrite, got %d: %#v", len(props), props)
	}
	xm, _ := props["x"].(map[string]any)
	if xm["type"] != "integer" || xm["description"] != "second version" {
		t.Errorf("overwrite did not take effect: %#v", xm)
	}
}

// TestParamsFreshCopiesEachCall guards that Properties()/RequiredNames() return
// independent values per call, so a caller mutating one doesn't corrupt the
// builder's internal state or a previously-returned result.
func TestParamsFreshCopiesEachCall(t *testing.T) {
	p := NewParams().String("a", "field a").Required("a")

	r1 := p.RequiredNames()
	r2 := p.RequiredNames()
	r1[0] = "mutated"
	if r2[0] != "a" {
		t.Errorf("RequiredNames() must return independent slices; r2 corrupted to %q", r2[0])
	}

	m1 := p.Properties()
	m2 := p.Properties()
	delete(m1, "a")
	if _, ok := m2["a"]; !ok {
		t.Error("Properties() must return independent maps; deleting from one affected the other")
	}
}

// TestEnumValueHelpersDeriveFromConstants proves the *Values() helpers return
// the exact wire strings of the typed constants (not a hand-maintained list
// that could drift from the constant definitions).
func TestEnumValueHelpersDeriveFromConstants(t *testing.T) {
	if got, want := RecordFormatValues(), []string{string(FormatMP3), string(FormatWAV), string(FormatMP4)}; !reflect.DeepEqual(got, want) {
		t.Errorf("RecordFormatValues() = %v, want %v", got, want)
	}
	if got, want := RecordDirectionValues(), []string{string(RecordDirectionSpeak), string(RecordDirectionListen), string(RecordDirectionBoth)}; !reflect.DeepEqual(got, want) {
		t.Errorf("RecordDirectionValues() = %v, want %v", got, want)
	}
	if got, want := TapDirectionValues(), []string{string(TapDirectionSpeak), string(TapDirectionHear), string(TapDirectionBoth)}; !reflect.DeepEqual(got, want) {
		t.Errorf("TapDirectionValues() = %v, want %v", got, want)
	}
	if got, want := CodecValues(), []string{string(CodecPCMU), string(CodecPCMA)}; !reflect.DeepEqual(got, want) {
		t.Errorf("CodecValues() = %v, want %v", got, want)
	}
}
