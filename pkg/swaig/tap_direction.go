package swaig

// TapDirection is the closed set of audio directions accepted by
// FunctionResult.Tap's direction argument, as a defined string type with typed
// constants. It gives Go callers editor autocompletion plus call-site typo
// checking — a bare string like "haer" only fails downstream (the server
// rejects it), whereas a mistyped constant fails to compile.
//
// Because Go auto-converts untyped string-constant literals to a defined string
// type, every call site keeps working both ways:
//
//	fr.Tap("rtp://h:1", "id", swaig.TapDirectionHear, swaig.CodecPCMU, 0, "") // typed const
//	fr.Tap("rtp://h:1", "id", "hear", "PCMU", 0, "")                          // bare string still compiles
//
// TapDirection is a string subtype, so the value written into the SWML tap
// params is byte-identical to the bare string the reference uses — parity with
// Python's tap(direction=...) keyword (a plain str). The enumerator emits the
// direction param as union<TapDirection,string>, so signature drift stays 0
// against the reference's str (the string member absorbs).
//
// IMPORTANT: this set ({speak, hear, both}) is DISTINCT from RecordDirection
// ({speak, listen, both}) — tap uses "hear" where record_call uses "listen". The
// Python reference validates the two with two different lists
// (function_result.py:1212 vs :917), so they are modelled as two separate types
// and must never be unified. (And both differ again from the RELAY play/record/
// tap direction vocabulary — three distinct vocabularies in all.)
type TapDirection string

// Audio directions for Tap. These are exactly the strings the SWML tap verb
// accepts for direction; the values are emitted verbatim into the tap params
// (matching the Python reference's valid_directions = ["speak", "hear", "both"]).
const (
	TapDirectionSpeak TapDirection = "speak" // what the party says
	TapDirectionHear  TapDirection = "hear"  // what the party hears
	TapDirectionBoth  TapDirection = "both"  // what the party hears and says (the reference default)
)
