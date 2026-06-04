package relay

// TTSGender is the closed set of text-to-speech voice genders as a defined
// string type with typed constants. WithTTSGender (the play_tts / prompt_tts
// gender option) takes it, giving Go callers editor autocompletion plus
// call-site typo checking — a bare string like "femail" only fails downstream,
// whereas a mistyped constant fails to compile.
//
// Because Go auto-converts untyped string-constant literals to a defined string
// type, every call site keeps working both ways:
//
//	relay.WithTTSGender(relay.GenderFemale)  // typed const — autocompleted
//	relay.WithTTSGender("female")            // bare string literal still compiles
//
// TTSGender is a string subtype, so the value written into the
// {"type":"tts","params":{"gender":...}} media entry is byte-identical to the
// bare string the reference uses — parity with Python's play_tts/prompt_tts
// gender keyword (a plain str).
type TTSGender string

// TTS voice genders. These are the canonical gender strings the RELAY wire
// accepts in the tts params; the values are the exact tokens emitted on the
// wire.
const (
	GenderMale   TTSGender = "male"
	GenderFemale TTSGender = "female"
)
