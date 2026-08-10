package swaig

// RecordFormat is the closed set of call-recording container formats as a
// defined string type with typed constants. FunctionResult.RecordCall takes it
// for the format argument, giving Go callers editor autocompletion plus
// call-site typo checking — a bare string like "wvv" only fails downstream,
// whereas a mistyped constant fails to compile.
//
// Because Go auto-converts untyped string-constant literals to a defined string
// type, every call site keeps working both ways:
//
//	fr.RecordCall("id", true, swaig.FormatWAV, "both", nil)  // typed const
//	fr.RecordCall("id", true, "wav", "both", nil)            // bare string still compiles
//
// RecordFormat is a string subtype, so the value written into the SWML record
// params is byte-identical to the bare string it wraps. The enumerator emits
// it as union<RecordFormat,string>, so a bare string is
// equally accepted.
type RecordFormat string

// Recording container formats. These are the canonical format tokens the SWML
// record verb accepts; the values are the exact strings emitted in the record
// params.
const (
	FormatMP3 RecordFormat = "mp3"
	FormatWAV RecordFormat = "wav"
	FormatMP4 RecordFormat = "mp4"
)
