package swaig

// Codec is the closed set of audio codecs accepted by FunctionResult.Tap's codec
// argument (the SWAIG tap media stream), as a defined string type with typed
// constants. It gives Go callers editor autocompletion plus call-site typo
// checking — a bare string like "PCUM" only fails downstream (the server rejects
// it), whereas a mistyped constant fails to compile.
//
// Because Go auto-converts untyped string-constant literals to a defined string
// type, every call site keeps working both ways:
//
//	fr.Tap("rtp://h:1", "id", swaig.TapDirectionBoth, swaig.CodecPCMA, 0, "") // typed const
//	fr.Tap("rtp://h:1", "id", "both", "PCMA", 0, "")                          // bare string still compiles
//
// Codec is a string subtype, so the value written into the SWML tap params is
// byte-identical to the bare string the reference uses — compatibility with Python's
// tap(codec=...) keyword (a plain str). The enumerator emits the codec param as
// union<Codec,string>, so signature drift stays 0 against the reference's str
// (the string member absorbs).
//
// IMPORTANT: this 2-value SWAIG-tap set ({PCMU, PCMA}, validated at
// function_result.py:1217) is DISTINCT from the larger RELAY connect/stream
// device codec superset ({PCMU, PCMA, OPUS, G729, G722, VP8, H264, ...},
// comma-joinable). The relay codec is genuinely open/multi-value and is
// deliberately left a bare string (see PORT_ADDITIONS / the journal §3) — this
// type must never be reused there.
type Codec string

// Audio codecs for Tap. These are exactly the strings the SWML tap verb accepts
// for codec; the values are emitted verbatim into the tap params (matching the
// Python reference's valid_codecs = ["PCMU", "PCMA"], default "PCMU").
const (
	CodecPCMU Codec = "PCMU" // G.711 µ-law (the reference default)
	CodecPCMA Codec = "PCMA" // G.711 A-law
)
