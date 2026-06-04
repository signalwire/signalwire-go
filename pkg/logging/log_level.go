package logging

// LogLevel is the closed set of log-level *names* accepted at the string
// boundary (e.g. server.WithLogLevel, the SIGNALWIRE_LOG_LEVEL env var) as a
// defined string type with typed constants. It is distinct from Level: Level
// is the internal integer severity used by the loggers, whereas LogLevel is the
// user-facing textual name that ParseLevel resolves into a Level.
//
// Adding the typed constants gives Go callers editor autocompletion plus
// call-site typo checking — a bare string like "debgu" only diverges at
// runtime (ParseLevel silently falls back to LevelInfo), whereas a mistyped
// constant fails to compile. Because Go auto-converts untyped string-constant
// literals to a defined string type, every call site keeps working both ways:
//
//	server.WithLogLevel(logging.LevelNameDebug)  // typed const — autocompleted
//	server.WithLogLevel("debug")                 // bare string literal still compiles
//
// LogLevel is a string subtype, so its value is byte-identical to the bare
// string the reference uses — ParseLevel("debug") and ParseLevel(LevelNameDebug)
// resolve to the same Level, preserving parity with the Python reference whose
// log_level is a plain str (signalwire/core/logging_config.py).
type LogLevel string

// Canonical log-level names. These are exactly the strings ParseLevel maps to a
// Level; the values must stay in lockstep with the switch in ParseLevel. The
// set mirrors the Python reference's documented SIGNALWIRE_LOG_LEVEL vocabulary
// (debug, info, warning, error) plus the Go-side aliases ParseLevel also honors
// (warn, off).
const (
	LevelNameDebug   LogLevel = "debug"
	LevelNameInfo    LogLevel = "info"
	LevelNameWarn    LogLevel = "warn"
	LevelNameWarning LogLevel = "warning"
	LevelNameError   LogLevel = "error"
	LevelNameOff     LogLevel = "off"
)
