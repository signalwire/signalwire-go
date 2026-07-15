// Copyright (c) 2025 SignalWire
//
// Mock-relay-backed tests for the typed Call convenience methods
// (PlayTTS / PlayAudio / PlaySilence / PlayRingtone / DetectDigit /
// DetectAnsweringMachine / DetectFax / PromptTTS / PromptAudio /
// WaitForAnswered / WaitForRinging / WaitForEnding).
//
// Each test drives the REAL relay client over the shared mock_relay
// WebSocket (no transport mock) and asserts the journaled command frame
// carries the exact RELAY media/params shape the Python reference emits.
// The wait-for-state tests assert the already-reached-state short-circuit.

package relay_test

import (
	"context"
	"testing"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/relay"
	"github.com/signalwire/signalwire-go/v3/pkg/relay/internal/mocktest"
)

// dialAnsweredCall arms a dial that resolves to an answered winner call and
// returns it. Convenience-method tests reuse this to obtain a live Call
// they can issue commands on, then assert the resulting journal frame.
func dialAnsweredCall(t *testing.T, h *mocktest.Harness, client *relay.Client, tag, callID string) *relay.Call {
	t.Helper()
	h.ArmDial(t, mocktest.DialOpts{
		Tag:          tag,
		WinnerCallID: callID,
		States:       []string{"created", "answered"},
		NodeID:       "node-mock-1",
		Device:       phoneDevice("", ""),
	})
	call, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag(tag),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	if call == nil {
		t.Fatal("Dial returned nil call")
	}
	return call
}

// lastFrameParams polls the recv journal for `method` and returns the
// most-recent frame's params, failing if none lands within 2s.
func lastFrameParams(t *testing.T, h *mocktest.Harness, method string) map[string]any {
	t.Helper()
	var params map[string]any
	ok := waitFor(2*time.Second, func() bool {
		entries := h.JournalRecv(t, method)
		if len(entries) == 0 {
			return false
		}
		params, _ = entries[len(entries)-1].FrameParams()
		return params != nil
	})
	if !ok {
		t.Fatalf("no %s frame landed in journal within 2s", method)
	}
	return params
}

// firstMediaEntry pulls play[0] as a map from a play/play_and_collect frame.
func firstMediaEntry(t *testing.T, params map[string]any) map[string]any {
	t.Helper()
	playList, _ := params["play"].([]any)
	if len(playList) == 0 {
		t.Fatalf("play list missing or empty: %#v", params["play"])
	}
	entry, _ := playList[0].(map[string]any)
	if entry == nil {
		t.Fatalf("play[0] is not a map: %#v", playList[0])
	}
	return entry
}

// mediaParams returns the nested params map of a media entry.
func mediaParams(t *testing.T, entry map[string]any) map[string]any {
	t.Helper()
	p, _ := entry["params"].(map[string]any)
	if p == nil {
		t.Fatalf("media entry has no params: %#v", entry)
	}
	return p
}

// ---------------------------------------------------------------------------
// Play family
// ---------------------------------------------------------------------------

// TestRelay_PlayTTSEmitsTTSMedia — PlayTTS builds a tts media entry with the
// nested text/language/gender/voice params and a top-level volume.
func TestRelay_PlayTTSEmitsTTSMedia(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := dialAnsweredCall(t, h, client, "t-ptts", "WIN-PTTS")
	_ = call.PlayTTS("hello world",
		relay.WithTTSLanguage("en-US"),
		relay.WithTTSGender("female"),
		relay.WithTTSVoice("Polly.Joanna"),
		relay.WithTTSVolume(3.5),
	)
	params := lastFrameParams(t, h, "calling.play")
	if params["call_id"] != "WIN-PTTS" {
		t.Errorf("call_id = %v, want WIN-PTTS", params["call_id"])
	}
	if v, _ := params["volume"].(float64); v != 3.5 {
		t.Errorf("volume = %v, want 3.5", params["volume"])
	}
	entry := firstMediaEntry(t, params)
	if entry["type"] != "tts" {
		t.Errorf("media type = %v, want tts", entry["type"])
	}
	mp := mediaParams(t, entry)
	if mp["text"] != "hello world" {
		t.Errorf("text = %v, want 'hello world'", mp["text"])
	}
	if mp["language"] != "en-US" {
		t.Errorf("language = %v, want en-US", mp["language"])
	}
	if mp["gender"] != "female" {
		t.Errorf("gender = %v, want female", mp["gender"])
	}
	if mp["voice"] != "Polly.Joanna" {
		t.Errorf("voice = %v, want Polly.Joanna", mp["voice"])
	}
}

// TestRelay_TTSGenderEnumOrString proves the typed relay.TTSGender constant and
// the bare string literal produce the IDENTICAL gender on the wire. Real
// behavior — the frame is journaled by the live mock_relay, no transport mock.
func TestRelay_TTSGenderEnumOrString(t *testing.T) {
	// The defined-string constant's value is the canonical wire token.
	if string(relay.GenderFemale) != "female" || string(relay.GenderMale) != "male" {
		t.Fatalf("TTSGender consts = %q/%q, want male/female",
			string(relay.GenderMale), string(relay.GenderFemale))
	}

	// Typed constant path: GenderFemale lands as the bare "female" string.
	c1, h1 := mocktest.New(t)
	if c1 == nil {
		return
	}
	call1 := dialAnsweredCall(t, h1, c1, "t-genc", "WIN-GENC")
	_ = call1.PlayTTS("hi", relay.WithTTSGender(relay.GenderFemale))
	gConst := mediaParams(t, firstMediaEntry(t, lastFrameParams(t, h1, "calling.play")))["gender"]

	// Bare-string path: "female" produces the identical wire value (Python str).
	c2, h2 := mocktest.New(t)
	if c2 == nil {
		return
	}
	call2 := dialAnsweredCall(t, h2, c2, "t-gens", "WIN-GENS")
	_ = call2.PlayTTS("hi", relay.WithTTSGender("female"))
	gStr := mediaParams(t, firstMediaEntry(t, lastFrameParams(t, h2, "calling.play")))["gender"]

	if gConst != "female" {
		t.Errorf("gender via GenderFemale = %v, want female", gConst)
	}
	if gConst != gStr {
		t.Errorf("typed const (%v) and string (%v) produced different wire genders", gConst, gStr)
	}
}

// TestRelay_PlayTTSOmitsUnsetOptionalParams — without options, only text is
// present in the tts params (language/gender/voice omitted entirely).
func TestRelay_PlayTTSOmitsUnsetOptionalParams(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := dialAnsweredCall(t, h, client, "t-ptts2", "WIN-PTTS2")
	_ = call.PlayTTS("bare")
	params := lastFrameParams(t, h, "calling.play")
	if _, has := params["volume"]; has {
		t.Errorf("volume should be omitted when WithTTSVolume not given, got %v", params["volume"])
	}
	mp := mediaParams(t, firstMediaEntry(t, params))
	if mp["text"] != "bare" {
		t.Errorf("text = %v, want bare", mp["text"])
	}
	for _, k := range []string{"language", "gender", "voice"} {
		if _, has := mp[k]; has {
			t.Errorf("tts params should omit %q when unset, got %v", k, mp[k])
		}
	}
}

// TestRelay_PlayAudioEmitsAudioMedia — PlayAudio builds an audio media entry
// carrying the url, with optional top-level volume.
func TestRelay_PlayAudioEmitsAudioMedia(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := dialAnsweredCall(t, h, client, "t-paud", "WIN-PAUD")
	_ = call.PlayAudio("https://example.com/a.mp3", relay.WithAudioVolume(-2.0))
	params := lastFrameParams(t, h, "calling.play")
	if v, _ := params["volume"].(float64); v != -2.0 {
		t.Errorf("volume = %v, want -2.0", params["volume"])
	}
	entry := firstMediaEntry(t, params)
	if entry["type"] != "audio" {
		t.Errorf("media type = %v, want audio", entry["type"])
	}
	mp := mediaParams(t, entry)
	if mp["url"] != "https://example.com/a.mp3" {
		t.Errorf("url = %v, want https://example.com/a.mp3", mp["url"])
	}
}

// TestRelay_PlaySilenceEmitsSilenceMedia — PlaySilence builds a silence media
// entry carrying the duration.
func TestRelay_PlaySilenceEmitsSilenceMedia(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := dialAnsweredCall(t, h, client, "t-psil", "WIN-PSIL")
	_ = call.PlaySilence(2.5)
	params := lastFrameParams(t, h, "calling.play")
	entry := firstMediaEntry(t, params)
	if entry["type"] != "silence" {
		t.Errorf("media type = %v, want silence", entry["type"])
	}
	mp := mediaParams(t, entry)
	if d, _ := mp["duration"].(float64); d != 2.5 {
		t.Errorf("duration = %v, want 2.5", mp["duration"])
	}
}

// TestRelay_PlayRingtoneEmitsRingtoneMedia — PlayRingtone builds a ringtone
// media entry carrying name and (optional) duration, with optional volume.
func TestRelay_PlayRingtoneEmitsRingtoneMedia(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := dialAnsweredCall(t, h, client, "t-prng", "WIN-PRNG")
	_ = call.PlayRingtone("us",
		relay.WithRingtoneDuration(4.0),
		relay.WithRingtoneVolume(1.0),
	)
	params := lastFrameParams(t, h, "calling.play")
	if v, _ := params["volume"].(float64); v != 1.0 {
		t.Errorf("volume = %v, want 1.0", params["volume"])
	}
	entry := firstMediaEntry(t, params)
	if entry["type"] != "ringtone" {
		t.Errorf("media type = %v, want ringtone", entry["type"])
	}
	mp := mediaParams(t, entry)
	if mp["name"] != "us" {
		t.Errorf("name = %v, want us", mp["name"])
	}
	if d, _ := mp["duration"].(float64); d != 4.0 {
		t.Errorf("duration = %v, want 4.0", mp["duration"])
	}
}

// TestRelay_PlayRingtoneOmitsDurationWhenUnset — name is present but duration
// is omitted from the ringtone params when WithRingtoneDuration not given.
func TestRelay_PlayRingtoneOmitsDurationWhenUnset(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := dialAnsweredCall(t, h, client, "t-prng2", "WIN-PRNG2")
	_ = call.PlayRingtone("gb")
	params := lastFrameParams(t, h, "calling.play")
	mp := mediaParams(t, firstMediaEntry(t, params))
	if mp["name"] != "gb" {
		t.Errorf("name = %v, want gb", mp["name"])
	}
	if _, has := mp["duration"]; has {
		t.Errorf("duration should be omitted when unset, got %v", mp["duration"])
	}
}

// ---------------------------------------------------------------------------
// Detect family
// ---------------------------------------------------------------------------

// detectObj pulls the nested detect object from a calling.detect frame.
func detectObj(t *testing.T, params map[string]any) map[string]any {
	t.Helper()
	d, _ := params["detect"].(map[string]any)
	if d == nil {
		t.Fatalf("detect object missing: %#v", params["detect"])
	}
	return d
}

// TestRelay_DetectDigitEmitsDigitDetect — DetectDigit builds a digit detect
// object with the nested digits param and a top-level timeout.
func TestRelay_DetectDigitEmitsDigitDetect(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := dialAnsweredCall(t, h, client, "t-ddig", "WIN-DDIG")
	_ = call.DetectDigit(relay.WithDigitDigits("123"), relay.WithDigitTimeout(7.0))
	params := lastFrameParams(t, h, "calling.detect")
	if params["call_id"] != "WIN-DDIG" {
		t.Errorf("call_id = %v, want WIN-DDIG", params["call_id"])
	}
	if to, _ := params["timeout"].(float64); to != 7.0 {
		t.Errorf("timeout = %v, want 7.0", params["timeout"])
	}
	d := detectObj(t, params)
	if d["type"] != "digit" {
		t.Errorf("detect type = %v, want digit", d["type"])
	}
	dp := mediaParams(t, d)
	if dp["digits"] != "123" {
		t.Errorf("digits = %v, want 123", dp["digits"])
	}
}

// TestRelay_DetectAnsweringMachineEmitsMachineDetect — DetectAnsweringMachine
// builds a machine detect object carrying only the provided params, with a
// top-level timeout.
func TestRelay_DetectAnsweringMachineEmitsMachineDetect(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := dialAnsweredCall(t, h, client, "t-damd", "WIN-DAMD")
	_ = call.DetectAnsweringMachine(
		relay.WithAMDInitialTimeout(5.0),
		relay.WithAMDEndSilenceTimeout(1.0),
		relay.WithAMDDetectInterruptions(true),
		relay.WithAMDTimeout(30.0),
	)
	params := lastFrameParams(t, h, "calling.detect")
	if to, _ := params["timeout"].(float64); to != 30.0 {
		t.Errorf("timeout = %v, want 30.0", params["timeout"])
	}
	d := detectObj(t, params)
	if d["type"] != "machine" {
		t.Errorf("detect type = %v, want machine", d["type"])
	}
	dp := mediaParams(t, d)
	if it, _ := dp["initial_timeout"].(float64); it != 5.0 {
		t.Errorf("initial_timeout = %v, want 5.0", dp["initial_timeout"])
	}
	if est, _ := dp["end_silence_timeout"].(float64); est != 1.0 {
		t.Errorf("end_silence_timeout = %v, want 1.0", dp["end_silence_timeout"])
	}
	if di, _ := dp["detect_interruptions"].(bool); !di {
		t.Errorf("detect_interruptions = %v, want true", dp["detect_interruptions"])
	}
	// Params the caller did not set must be absent (only-provided-keys).
	for _, k := range []string{"machine_voice_threshold", "machine_words_threshold", "detect_message_end"} {
		if _, has := dp[k]; has {
			t.Errorf("machine params should omit %q when unset, got %v", k, dp[k])
		}
	}
}

// TestRelay_DetectFaxEmitsFaxDetect — DetectFax builds a fax detect object
// carrying the nested tone param.
func TestRelay_DetectFaxEmitsFaxDetect(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := dialAnsweredCall(t, h, client, "t-dfax", "WIN-DFAX")
	_ = call.DetectFax(relay.WithFaxTone("CED"))
	params := lastFrameParams(t, h, "calling.detect")
	d := detectObj(t, params)
	if d["type"] != "fax" {
		t.Errorf("detect type = %v, want fax", d["type"])
	}
	dp := mediaParams(t, d)
	if dp["tone"] != "CED" {
		t.Errorf("tone = %v, want CED", dp["tone"])
	}
}

// ---------------------------------------------------------------------------
// Prompt family
// ---------------------------------------------------------------------------

// TestRelay_PromptTTSEmitsTTSPlayAndCollect — PromptTTS builds a tts media
// entry plus the caller's collect object on a play_and_collect frame.
func TestRelay_PromptTTSEmitsTTSPlayAndCollect(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := dialAnsweredCall(t, h, client, "t-rtts", "WIN-RTTS")
	collect := map[string]any{"digits": map[string]any{"max": 3}}
	_ = call.PromptTTS("enter pin", collect,
		relay.WithTTSVoice("en-US-Neural"),
		relay.WithTTSVolume(2.0),
	)
	params := lastFrameParams(t, h, "calling.play_and_collect")
	if v, _ := params["volume"].(float64); v != 2.0 {
		t.Errorf("volume = %v, want 2.0", params["volume"])
	}
	entry := firstMediaEntry(t, params)
	if entry["type"] != "tts" {
		t.Errorf("media type = %v, want tts", entry["type"])
	}
	mp := mediaParams(t, entry)
	if mp["text"] != "enter pin" {
		t.Errorf("text = %v, want 'enter pin'", mp["text"])
	}
	if mp["voice"] != "en-US-Neural" {
		t.Errorf("voice = %v, want en-US-Neural", mp["voice"])
	}
	col, _ := params["collect"].(map[string]any)
	if col == nil {
		t.Fatalf("collect object missing: %#v", params["collect"])
	}
	dig, _ := col["digits"].(map[string]any)
	if dig == nil || dig["max"] == nil {
		t.Errorf("collect.digits.max missing: %#v", col)
	}
}

// TestRelay_PromptAudioEmitsAudioPlayAndCollect — PromptAudio builds an audio
// media entry plus the caller's collect object on a play_and_collect frame.
func TestRelay_PromptAudioEmitsAudioPlayAndCollect(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := dialAnsweredCall(t, h, client, "t-raud", "WIN-RAUD")
	collect := map[string]any{"speech": map[string]any{"end_silence_timeout": 1}}
	_ = call.PromptAudio("https://example.com/prompt.wav", collect)
	params := lastFrameParams(t, h, "calling.play_and_collect")
	entry := firstMediaEntry(t, params)
	if entry["type"] != "audio" {
		t.Errorf("media type = %v, want audio", entry["type"])
	}
	mp := mediaParams(t, entry)
	if mp["url"] != "https://example.com/prompt.wav" {
		t.Errorf("url = %v, want https://example.com/prompt.wav", mp["url"])
	}
	col, _ := params["collect"].(map[string]any)
	if col == nil {
		t.Fatalf("collect object missing: %#v", params["collect"])
	}
	if _, ok := col["speech"].(map[string]any); !ok {
		t.Errorf("collect.speech missing: %#v", col)
	}
}

// ---------------------------------------------------------------------------
// Wait-for-state family — already-reached-state short-circuit
// ---------------------------------------------------------------------------

// TestRelay_WaitForAnsweredShortCircuitsWhenAnswered — after a dial that
// reaches "answered", WaitForAnswered returns immediately with the current
// call_state and does not block on the context.
func TestRelay_WaitForAnsweredShortCircuitsWhenAnswered(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := dialAnsweredCall(t, h, client, "t-wfa", "WIN-WFA")
	if call.State() != "answered" {
		t.Fatalf("precondition: call state = %q, want answered", call.State())
	}
	// A context that is already cancelled would make a *blocking* wait fail
	// immediately. Because the call is already at the target, the method must
	// short-circuit and succeed regardless.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ev, err := call.WaitForAnswered(ctx)
	if err != nil {
		t.Fatalf("WaitForAnswered should short-circuit (already answered), got err: %v", err)
	}
	if ev == nil {
		t.Fatal("WaitForAnswered returned nil event on short-circuit")
	}
	if ev.GetString("call_state") != "answered" {
		t.Errorf("short-circuit event call_state = %q, want answered", ev.GetString("call_state"))
	}
}

// TestRelay_WaitForRingingShortCircuitsWhenAnswered — "answered" is past
// "ringing" in the state order, so WaitForRinging short-circuits too.
func TestRelay_WaitForRingingShortCircuitsWhenAnswered(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := dialAnsweredCall(t, h, client, "t-wfr", "WIN-WFR")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ev, err := call.WaitForRinging(ctx)
	if err != nil {
		t.Fatalf("WaitForRinging should short-circuit (answered > ringing), got err: %v", err)
	}
	if ev.GetString("call_state") != "answered" {
		t.Errorf("short-circuit event call_state = %q, want answered", ev.GetString("call_state"))
	}
}

// TestRelay_WaitForEndingBlocksThenResolvesOnStateEvent — "ending" is past
// "answered", so WaitForEnding does NOT short-circuit; it blocks until an
// ending state event is pushed.
func TestRelay_WaitForEndingBlocksThenResolvesOnStateEvent(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := dialAnsweredCall(t, h, client, "t-wfe", "WIN-WFE")

	// Push the ending state shortly after the wait begins.
	go func() {
		time.Sleep(100 * time.Millisecond)
		h.Push(t, statePushFrame("WIN-WFE", "ending", "t-wfe", "outbound"), "")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ev, err := call.WaitForEnding(ctx)
	if err != nil {
		t.Fatalf("WaitForEnding: %v", err)
	}
	if ev.GetString("call_state") != "ending" {
		t.Errorf("resolved event call_state = %q, want ending", ev.GetString("call_state"))
	}
}
