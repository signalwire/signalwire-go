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
//	fr.Tap("rtp://h:1", "id", swaig.TapDirectionListen, swaig.CodecPCMU, 0, "") // typed const
//	fr.Tap("rtp://h:1", "id", "listen", "PCMU", 0, "")                        // bare string still compiles
//
// TapDirection is a string subtype, so the value written into the SWML tap
// params is byte-identical to the bare string it wraps. The enumerator emits the
// direction param as union<TapDirection,string>, so a bare string is equally accepted.
//
// The set is the SWML tap verb's direction enum, {speak, listen, both} — the same
// wire values as RecordDirection (record_call), kept as a separate type per verb.
// ("hear" is NOT a tap direction: the verb rejects it.)
type TapDirection string

// Audio directions for Tap. These are exactly the strings the SWML tap verb
// accepts for direction; the values are emitted verbatim into the tap params
// .
const (
	TapDirectionSpeak  TapDirection = "speak"  // what the party says
	TapDirectionListen TapDirection = "listen" // what the party hears
	TapDirectionBoth   TapDirection = "both"   // what the party hears and says (the reference default)
)
