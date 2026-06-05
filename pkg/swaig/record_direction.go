package swaig

// RecordDirection is the closed set of audio directions accepted by
// FunctionResult.RecordCall's direction argument, as a defined string type with
// typed constants. It gives Go callers editor autocompletion plus call-site typo
// checking — a bare string like "lisetn" only fails downstream (the server
// rejects it), whereas a mistyped constant fails to compile.
//
// Because Go auto-converts untyped string-constant literals to a defined string
// type, every call site keeps working both ways:
//
//	fr.RecordCall("id", true, swaig.FormatWAV, swaig.RecordDirectionListen, nil) // typed const
//	fr.RecordCall("id", true, swaig.FormatWAV, "listen", nil)                    // bare string still compiles
//
// RecordDirection is a string subtype, so the value written into the SWML
// record_call params is byte-identical to the bare string the reference uses —
// parity with Python's record_call(direction=...) keyword (a plain str). The
// enumerator emits the direction param as union<RecordDirection,string>, so
// signature drift stays 0 against the reference's str (the string member
// absorbs).
//
// IMPORTANT: this set ({speak, listen, both}) is DISTINCT from TapDirection
// ({speak, hear, both}) — record_call uses "listen" where tap uses "hear". The
// Python reference validates the two with two different lists
// (function_result.py:917 vs :1212), so they are modelled as two separate types
// and must never be unified. (And both differ again from the RELAY play/record/
// tap direction vocabulary — three distinct vocabularies in all.)
type RecordDirection string

// Audio directions for RecordCall. These are exactly the strings the SWML
// record_call verb accepts for direction; the values are emitted verbatim into
// the record_call params (matching the Python reference's
// valid_directions = ["speak", "listen", "both"]).
const (
	RecordDirectionSpeak  RecordDirection = "speak"  // what the party says
	RecordDirectionListen RecordDirection = "listen" // what the party hears
	RecordDirectionBoth   RecordDirection = "both"   // what the party hears and says (the reference default)
)
