# PORT_ADDITIONS.md
#
# Every symbol listed here is a public Go-port API that has no direct
# Python-reference counterpart.  The format is one
# `<fully.qualified.symbol>: <rationale>` per line, as expected by
# porting-sdk/scripts/diff_port_surface.py.  Section headers (lines
# beginning with `#`) are ignored by the parser.
#
# Today the enumerator does NOT emit these under Python-style module
# names — Go-only structs simply fall through the structTable without
# being projected.  This file exists so reviewers can explicitly track
# port-only surface area as it grows.  When the enumerator starts
# projecting Go-only additions it will read this file verbatim.

# --- Relay ---
# AIEvent is a Go-only convenience type for the AI action lifecycle; the
# Python port models the same stream of events through the generic
# RelayEvent dispatcher.
signalwire.relay.event.AIEvent: Go-only typed wrapper around AI action events; Python uses RelayEvent directly

# --- Livewire plugins ---
# Extra STT/TTS plugin stubs for parity with LiveKit integrations the
# Python livewire shim did not bother to surface.  They're struct stubs
# used by the WithSTT/WithTTS provider string match.
signalwire.livewire.plugins.GoogleSTT: Go-only plugin stub; matches WithSTT("google") at AgentSession construction
signalwire.livewire.plugins.OpenAITTS: Go-only plugin stub; matches WithTTS("openai") at AgentSession construction

# --- Go idioms that have no Python analogue ---
# The Go port expresses optional constructor arguments via the
# functional-options pattern (With*).  These are not listed as additions
# because the enumerator drops them (they're typed as function aliases,
# not methods on a struct).  If future reviewers want to pin them they
# should land here alongside their owning constructor.
