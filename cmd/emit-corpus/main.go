// Command emit-corpus is the Go port's EMISSION-DUMP program for the cross-port
// emission differ (porting-sdk/scripts/diff_port_emission.py).
//
// It builds the shared FunctionResult corpus
// (porting-sdk/scripts/emission_corpus.py — the single source of truth) using
// the Go SDK's native swaig.FunctionResult API, serialises each entry the same
// way the SDK serialises on the wire (ToMap), and prints ONE JSON object mapping
//
//	corpus-id -> emission
//
// to stdout. The differ runs this program, parses that object, and byte-compares
// each entry against Python's to_dict(). See the "per-port dump contract" in the
// differ's --help and IDIOM_PASS_JOURNAL.md §4 Tier-0.
//
// CONTRACT (why this file looks the way it does):
//   - Every corpus id in emission_corpus.corpus_ids() MUST appear here exactly
//     once (the differ rejects an id-set mismatch as a setup error — a skewed set
//     would mask real diffs). When the shared corpus grows, add the new id here.
//   - The argument VALUES are the WIRE values (plain strings/numbers/bools/maps).
//     Where the Go API types a closed set (RecordFormat, RecordDirection,
//     TapDirection, Codec) we pass the typed constant whose string value is the
//     wire value, proving the typed path emits byte-identically to the string.
//   - Only stdout carries the JSON object; nothing else is printed there.
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/emit-corpus
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
)

// entry pairs a stable corpus id with the FunctionResult it produces. Building
// the result lazily (a func) keeps each line a single, readable native call.
type entry struct {
	id    string
	build func() *swaig.FunctionResult
}

// fr is a tiny constructor helper: swaig.NewFunctionResult(response).
func fr(response string) *swaig.FunctionResult { return swaig.NewFunctionResult(response) }

// ptrBool / ptrInt produce pointers for WaitForUser's optional enabled/timeout.
func ptrBool(b bool) *bool { return &b }
func ptrInt(i int) *int    { return &i }

// corpus is the Go-native mirror of porting-sdk/scripts/emission_corpus.py.
// The ids and the resulting emission must match the Python oracle exactly
// (modulo the whole-float artifact the differ normalises: Python 44.0 == Go 44).
var corpus = []entry{
	// ---- envelope edge cases (ToMap shape) ----------------------------------
	{"envelope.empty", func() *swaig.FunctionResult { return fr("") }},
	{"envelope.response_only", func() *swaig.FunctionResult { return fr("Hello, world!") }},
	{"envelope.post_process_no_action", func() *swaig.FunctionResult {
		return fr("hi").SetPostProcess(true)
	}},
	{"envelope.action_only", func() *swaig.FunctionResult { return fr("").Hangup() }},
	{"envelope.post_process_with_action", func() *swaig.FunctionResult {
		return fr("Transferring").SetPostProcess(true).Hangup()
	}},
	{"envelope.response_and_action", func() *swaig.FunctionResult {
		return fr("Goodbye").Hangup()
	}},

	// ---- connect ------------------------------------------------------------
	{"connect.final_true", func() *swaig.FunctionResult {
		return fr("").Connect(swaig.ConnectOptions{Destination: "+15551234567", Final: true})
	}},
	{"connect.final_false", func() *swaig.FunctionResult {
		return fr("").Connect(swaig.ConnectOptions{Destination: "+15551234567", Final: false})
	}},
	{"connect.from_addr", func() *swaig.FunctionResult {
		return fr("").Connect(swaig.ConnectOptions{Destination: "support@example.com", Final: false, From: "+15559876543"})
	}},

	// ---- swml_transfer ------------------------------------------------------
	{"swml_transfer.default", func() *swaig.FunctionResult {
		return fr("").SwmlTransfer("https://dest.example.com/swml", "Goodbye!", true)
	}},
	{"swml_transfer.final_false", func() *swaig.FunctionResult {
		return fr("").SwmlTransfer("https://dest.example.com/swml", "Welcome back. How else can I help?", false)
	}},

	// ---- simple call-control actions ---------------------------------------
	{"hangup", func() *swaig.FunctionResult { return fr("").Hangup() }},
	{"hold.default", func() *swaig.FunctionResult { return fr("").Hold(300) }},
	{"hold.value", func() *swaig.FunctionResult { return fr("").Hold(120) }},
	{"hold.clamp_high", func() *swaig.FunctionResult { return fr("").Hold(5000) }},
	{"hold.clamp_low", func() *swaig.FunctionResult { return fr("").Hold(-5) }},
	{"stop", func() *swaig.FunctionResult { return fr("").Stop() }},
	{"say", func() *swaig.FunctionResult { return fr("").Say("Please hold while I connect you.") }},

	// ---- wait_for_user (each branch) ---------------------------------------
	{"wait_for_user.default", func() *swaig.FunctionResult { return fr("").WaitForUser(swaig.WaitForUserOptions{}) }},
	{"wait_for_user.answer_first", func() *swaig.FunctionResult { return fr("").WaitForUser(swaig.WaitForUserOptions{AnswerFirst: true}) }},
	{"wait_for_user.timeout", func() *swaig.FunctionResult { return fr("").WaitForUser(swaig.WaitForUserOptions{Timeout: ptrInt(30)}) }},
	{"wait_for_user.enabled_true", func() *swaig.FunctionResult {
		return fr("").WaitForUser(swaig.WaitForUserOptions{Enabled: ptrBool(true)})
	}},
	{"wait_for_user.enabled_false", func() *swaig.FunctionResult {
		return fr("").WaitForUser(swaig.WaitForUserOptions{Enabled: ptrBool(false)})
	}},

	// ---- global data / metadata --------------------------------------------
	{"set_global_data", func() *swaig.FunctionResult {
		return fr("").UpdateGlobalData(map[string]any{"plan": "premium", "chips": 1000})
	}},
	{"unset_global_data.list", func() *swaig.FunctionResult {
		return fr("").RemoveGlobalData([]string{"plan", "chips"})
	}},
	{"unset_global_data.str", func() *swaig.FunctionResult { return fr("").RemoveGlobalDataKey("plan") }},
	{"set_metadata", func() *swaig.FunctionResult {
		return fr("").SetMetadata(map[string]any{"token": "abc", "count": 3})
	}},
	{"unset_metadata.list", func() *swaig.FunctionResult {
		return fr("").RemoveMetadata([]string{"token", "count"})
	}},
	{"unset_metadata.str", func() *swaig.FunctionResult { return fr("").RemoveMetadataKey("token") }},

	// ---- swml_user_event ----------------------------------------------------
	{"swml_user_event", func() *swaig.FunctionResult {
		return fr("").SwmlUserEvent(map[string]any{
			"type": "cards_dealt", "player_hand": []any{"AS", "KH"}, "player_score": 21,
		})
	}},

	// ---- step / context changes --------------------------------------------
	{"change_step", func() *swaig.FunctionResult { return fr("").SwmlChangeStep("collect_payment") }},
	{"change_context", func() *swaig.FunctionResult { return fr("").SwmlChangeContext("billing") }},

	// ---- switch_context (simple vs object) ---------------------------------
	// Go's SwitchContext has a trailing isolated param (a documented PORT_ADDITION);
	// the corpus exercises the Python-equivalent paths with isolated=false.
	{"switch_context.simple", func() *swaig.FunctionResult {
		return fr("").SwitchContext("You are now a billing agent.", "", false, false, false)
	}},
	{"switch_context.object", func() *swaig.FunctionResult {
		return fr("").SwitchContext("New system prompt", "User said something", true, false, false)
	}},
	{"switch_context.full_reset", func() *swaig.FunctionResult {
		return fr("").SwitchContext("Reset prompt", "", false, true, false)
	}},

	// ---- background file play/stop -----------------------------------------
	{"playback_bg.simple", func() *swaig.FunctionResult { return fr("").PlayBackgroundFile("music.mp3", false) }},
	{"playback_bg.wait", func() *swaig.FunctionResult { return fr("").PlayBackgroundFile("music.mp3", true) }},
	{"stop_playback_bg", func() *swaig.FunctionResult { return fr("").StopBackgroundFile() }},

	// ---- join_room / sip_refer ---------------------------------------------
	{"join_room", func() *swaig.FunctionResult { return fr("").JoinRoom("team-standup") }},
	{"sip_refer", func() *swaig.FunctionResult { return fr("").SIPRefer("sip:agent@example.com") }},

	// ---- send_sms -----------------------------------------------------------
	{"send_sms.body", func() *swaig.FunctionResult {
		return fr("").SendSms("+15551112222", "+15553334444", "Your appointment is confirmed.", nil, nil, "")
	}},
	{"send_sms.full", func() *swaig.FunctionResult {
		return fr("").SendSms("+15551112222", "+15553334444", "See attached.",
			[]string{"https://ex.com/a.jpg"}, []string{"receipt", "vip"}, "us")
	}},

	// ---- pay ----------------------------------------------------------------
	{"pay.minimal", func() *swaig.FunctionResult {
		return fr("").Pay("https://pay.example.com/connector", nil)
	}},
	{"pay.full", func() *swaig.FunctionResult {
		return fr("").Pay("https://pay.example.com/connector", &swaig.PayOptions{
			InputMethod:         "dtmf",
			StatusURL:           "https://ex.com/status",
			PaymentMethod:       "credit-card",
			Timeout:             7,
			MaxAttempts:         2,
			SecurityCode:        false,
			SecurityCodeSet:     true, // explicitly set false (default is true)
			PostalCode:          "90210",
			MinPostalCodeLength: 5,
			TokenType:           "one-time",
			ChargeAmount:        "9.99",
			Currency:            "usd",
			Language:            "en-US",
			Voice:               "woman",
			Description:         "Order 42",
			ValidCardTypes:      "visa amex",
			Parameters:          []map[string]string{{"name": "order_id", "value": "42"}},
			Prompts: []map[string]any{{
				"for":       "payment-card-number",
				"actions":   []any{map[string]any{"type": "Say", "phrase": "Enter your card number"}},
				"card_type": "visa amex",
			}},
		})
	}},
	{"pay.postal_bool", func() *swaig.FunctionResult {
		return fr("").Pay("https://pay.example.com/connector", &swaig.PayOptions{PostalCode: true})
	}},

	// ---- record_call (incl. mp4 + each direction) --------------------------
	{"record_call.defaults", func() *swaig.FunctionResult {
		return fr("").RecordCall("", false, swaig.FormatWAV, swaig.RecordDirectionBoth, nil)
	}},
	{"record_call.wav_speak", func() *swaig.FunctionResult {
		return fr("").RecordCall("", false, swaig.FormatWAV, swaig.RecordDirectionSpeak, nil)
	}},
	{"record_call.mp3_listen", func() *swaig.FunctionResult {
		return fr("").RecordCall("", false, swaig.FormatMP3, swaig.RecordDirectionListen, nil)
	}},
	{"record_call.mp4_both", func() *swaig.FunctionResult {
		return fr("").RecordCall("", false, swaig.FormatMP4, swaig.RecordDirectionBoth, nil)
	}},
	{"record_call.full", func() *swaig.FunctionResult {
		return fr("").RecordCall("rec1", true, swaig.FormatMP3, swaig.RecordDirectionBoth, &swaig.RecordCallOptions{
			Terminators:          "#",
			Beep:                 true,
			InputSensitivity:     30.0,
			InitialTimeout:       5.0,
			InitialTimeoutSet:    true,
			EndSilenceTimeout:    3.0,
			EndSilenceTimeoutSet: true,
			MaxLength:            120.0,
			MaxLengthSet:         true,
			StatusURL:            "https://ex.com/rec",
		})
	}},
	{"stop_record_call.bare", func() *swaig.FunctionResult { return fr("").StopRecordCall("") }},
	{"stop_record_call.id", func() *swaig.FunctionResult { return fr("").StopRecordCall("rec1") }},

	// ---- tap (each direction / codec) --------------------------------------
	{"tap.defaults", func() *swaig.FunctionResult {
		return fr("").Tap("rtp://10.0.0.1:5004", "", swaig.TapDirectionBoth, swaig.CodecPCMU, 20, "")
	}},
	{"tap.speak_pcma", func() *swaig.FunctionResult {
		return fr("").Tap("ws://ex.com/tap", "", swaig.TapDirectionSpeak, swaig.CodecPCMA, 20, "")
	}},
	{"tap.hear_pcmu", func() *swaig.FunctionResult {
		return fr("").Tap("wss://ex.com/tap", "", swaig.TapDirectionHear, swaig.CodecPCMU, 20, "")
	}},
	{"tap.both_full", func() *swaig.FunctionResult {
		return fr("").Tap("rtp://10.0.0.1:5004", "tap1", swaig.TapDirectionBoth, swaig.CodecPCMA, 40, "https://ex.com/tapstatus")
	}},
	{"stop_tap.bare", func() *swaig.FunctionResult { return fr("").StopTap("") }},
	{"stop_tap.id", func() *swaig.FunctionResult { return fr("").StopTap("tap1") }},

	// ---- join_conference (simple + full) -----------------------------------
	{"join_conference.simple", func() *swaig.FunctionResult { return fr("").JoinConference("sales-floor", nil) }},
	{"join_conference.full", func() *swaig.FunctionResult {
		startOnEnter := false
		return fr("").JoinConference("sales-floor", &swaig.JoinConferenceOptions{
			Muted:                         true,
			Beep:                          "onEnter",
			StartOnEnter:                  &startOnEnter,
			EndOnExit:                     true,
			WaitURL:                       "https://ex.com/hold",
			MaxParticipants:               50,
			Record:                        "record-from-start",
			Region:                        "us-east",
			Trim:                          "do-not-trim",
			Coach:                         "call-123",
			StatusCallbackEvent:           "start end join leave",
			StatusCallback:                "https://ex.com/cb",
			StatusCallbackMethod:          "GET",
			RecordingStatusCallback:       "https://ex.com/rcb",
			RecordingStatusCallbackMethod: "GET",
			RecordingStatusCallbackEvent:  "in-progress completed",
		})
	}},

	// ---- execute_rpc + the three rpc helpers -------------------------------
	{"execute_rpc.minimal", func() *swaig.FunctionResult { return fr("").ExecuteRPC("ai_unhold", nil, "", "") }},
	{"execute_rpc.full", func() *swaig.FunctionResult {
		return fr("").ExecuteRPC("ai_message",
			map[string]any{"role": "system", "message_text": "Hello"}, "call-abc", "node-1")
	}},
	{"rpc_dial", func() *swaig.FunctionResult {
		return fr("").RPCDial("+15551234567", "+15559876543", "https://ex.com/call-agent", "phone")
	}},
	{"rpc_ai_message", func() *swaig.FunctionResult {
		return fr("").RPCAiMessage("call-abc", "Please take a message.", "system")
	}},
	{"rpc_ai_unhold", func() *swaig.FunctionResult { return fr("").RPCAiUnhold("call-abc") }},

	// ---- simulate_user_input -----------------------------------------------
	{"simulate_user_input", func() *swaig.FunctionResult {
		return fr("").SimulateUserInput("I'd like to pay my bill.")
	}},

	// ---- dynamic hints ------------------------------------------------------
	{"add_dynamic_hints", func() *swaig.FunctionResult {
		return fr("").AddDynamicHints([]any{
			"Cabby",
			map[string]any{"pattern": "cab bee", "replace": "Cabby", "ignore_case": true},
		})
	}},
	{"clear_dynamic_hints", func() *swaig.FunctionResult { return fr("").ClearDynamicHints() }},

	// ---- toggle_functions / functions-on-timeout ---------------------------
	{"toggle_functions", func() *swaig.FunctionResult {
		return fr("").ToggleFunctions([]map[string]any{
			{"function": "transfer", "active": false},
			{"function": "lookup", "active": true},
		})
	}},
	{"functions_on_speaker_timeout.true", func() *swaig.FunctionResult {
		return fr("").EnableFunctionsOnTimeout(true)
	}},
	{"functions_on_speaker_timeout.false", func() *swaig.FunctionResult {
		return fr("").EnableFunctionsOnTimeout(false)
	}},

	// ---- extensive_data -----------------------------------------------------
	{"extensive_data.true", func() *swaig.FunctionResult { return fr("").EnableExtensiveData(true) }},
	{"extensive_data.false", func() *swaig.FunctionResult { return fr("").EnableExtensiveData(false) }},

	// ---- replace_in_history (str + bool) -----------------------------------
	{"replace_in_history.bool", func() *swaig.FunctionResult { return fr("").ReplaceInHistory(true) }},
	{"replace_in_history.str", func() *swaig.FunctionResult {
		return fr("").ReplaceInHistory("Summarized the order.")
	}},

	// ---- settings -----------------------------------------------------------
	{"settings", func() *swaig.FunctionResult {
		return fr("").UpdateSettings(map[string]any{"temperature": 0.7, "max-tokens": 256, "top-p": 0.9})
	}},

	// ---- speech timeouts ----------------------------------------------------
	{"end_of_speech_timeout", func() *swaig.FunctionResult { return fr("").SetEndOfSpeechTimeout(800) }},
	{"speech_event_timeout", func() *swaig.FunctionResult { return fr("").SetSpeechEventTimeout(1200) }},

	// ---- execute_swml (dict + JSON-string + transfer) ----------------------
	{"execute_swml.dict", func() *swaig.FunctionResult {
		return fr("").ExecuteSwml(map[string]any{
			"version": "1.0.0", "sections": map[string]any{"main": []any{map[string]any{"answer": map[string]any{}}}},
		}, false)
	}},
	{"execute_swml.dict_transfer", func() *swaig.FunctionResult {
		return fr("").ExecuteSwml(map[string]any{
			"version": "1.0.0", "sections": map[string]any{"main": []any{map[string]any{"answer": map[string]any{}}}},
		}, true)
	}},
	{"execute_swml.json_string", func() *swaig.FunctionResult {
		return fr("").ExecuteSwml(`{"version": "1.0.0", "sections": {"main": [{"hangup": {}}]}}`, false)
	}},
}

func main() {
	out := make(map[string]any, len(corpus))
	seen := make(map[string]bool, len(corpus))
	for _, e := range corpus {
		if seen[e.id] {
			fmt.Fprintf(os.Stderr, "emit-corpus: duplicate corpus id %q\n", e.id)
			os.Exit(1)
		}
		seen[e.id] = true
		out[e.id] = e.build().ToMap()
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false) // keep '+'/'&' etc. literal; matches Python json output
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "emit-corpus: encode failed: %v\n", err)
		os.Exit(1)
	}
}
