package swaig

import (
	"encoding/json"
	"testing"
)

// --- Constructor & ToMap ---

func TestNewFunctionResult(t *testing.T) {
	fr := NewFunctionResult("Hello")
	if fr.response != "Hello" {
		t.Errorf("response = %q, want %q", fr.response, "Hello")
	}
	if fr.postProcess {
		t.Error("postProcess should default to false")
	}
	if len(fr.actions) != 0 {
		t.Errorf("actions length = %d, want 0", len(fr.actions))
	}
}

func TestNewFunctionResultEmptyResponse(t *testing.T) {
	// An empty-response result with no actions defaults to "Action completed."
	// (Python to_dict parity).
	fr := NewFunctionResult("")
	m := fr.ToMap()
	if m["response"] != "Action completed." {
		t.Errorf("response = %v, want %q", m["response"], "Action completed.")
	}
}

// --- Getters ---

func TestResponseGetter(t *testing.T) {
	fr := NewFunctionResult("Hello")
	if fr.Response() != "Hello" {
		t.Errorf("Response() = %q, want %q", fr.Response(), "Hello")
	}
}

func TestActionsGetter(t *testing.T) {
	fr := NewFunctionResult("test").AddAction("say", "hi")
	acts := fr.Actions()
	if len(acts) != 1 {
		t.Fatalf("Actions() length = %d, want 1", len(acts))
	}
	if acts[0]["say"] != "hi" {
		t.Errorf("Actions()[0][say] = %v, want %q", acts[0]["say"], "hi")
	}
}

func TestPostProcessGetter(t *testing.T) {
	fr := NewFunctionResult("test").SetPostProcess(true)
	if !fr.PostProcess() {
		t.Error("PostProcess() should return true after SetPostProcess(true)")
	}
}

func TestPostProcessGetterDefault(t *testing.T) {
	fr := NewFunctionResult("test")
	if fr.PostProcess() {
		t.Error("PostProcess() should return false by default")
	}
}

func TestToMapBasic(t *testing.T) {
	fr := NewFunctionResult("test response")
	m := fr.ToMap()

	if m["response"] != "test response" {
		t.Errorf("response = %v, want %q", m["response"], "test response")
	}
	if _, ok := m["action"]; ok {
		t.Error("action should not be present when empty")
	}
	if _, ok := m["post_process"]; ok {
		t.Error("post_process should not be present when false")
	}
}

func TestToMapWithActions(t *testing.T) {
	fr := NewFunctionResult("response").
		AddAction("say", "hello")

	m := fr.ToMap()
	actions, ok := m["action"].([]map[string]any)
	if !ok {
		t.Fatal("action should be []map[string]any")
	}
	if len(actions) != 1 {
		t.Fatalf("actions length = %d, want 1", len(actions))
	}
	if actions[0]["say"] != "hello" {
		t.Errorf("action say = %v, want %q", actions[0]["say"], "hello")
	}
}

func TestToMapWithPostProcess(t *testing.T) {
	fr := NewFunctionResult("response").
		AddAction("say", "hello").
		SetPostProcess(true)

	m := fr.ToMap()
	pp, ok := m["post_process"]
	if !ok {
		t.Fatal("post_process should be present when true and an action exists")
	}
	if pp != true {
		t.Errorf("post_process = %v, want true", pp)
	}
}

func TestToMapPostProcessOmittedWithoutAction(t *testing.T) {
	// post_process is dropped when there are no actions, even if set true
	// (Python to_dict: `if self.post_process and self.action`).
	fr := NewFunctionResult("response").SetPostProcess(true)
	if _, ok := fr.ToMap()["post_process"]; ok {
		t.Error("post_process should be omitted when there are no actions")
	}
}

func TestToMapEmptyResponseOmittedWithAction(t *testing.T) {
	// response is omitted when empty but an action is present (Python parity).
	fr := NewFunctionResult("").AddAction("say", "hello")
	m := fr.ToMap()
	if _, ok := m["response"]; ok {
		t.Error("empty response should be omitted when an action is present")
	}
	if _, ok := m["action"]; !ok {
		t.Error("action should be present")
	}
}

func TestToMapEmptyDefaultsToActionCompleted(t *testing.T) {
	// An otherwise-empty result defaults to {"response": "Action completed."}.
	m := NewFunctionResult("").ToMap()
	if m["response"] != "Action completed." {
		t.Errorf("empty result response = %v, want %q", m["response"], "Action completed.")
	}
	if _, ok := m["action"]; ok {
		t.Error("action should not be present")
	}
}

func TestToMapPostProcessFalseOmitted(t *testing.T) {
	fr := NewFunctionResult("response").
		SetPostProcess(false)

	m := fr.ToMap()
	if _, ok := m["post_process"]; ok {
		t.Error("post_process should not be present when false")
	}
}

func TestToMapJSONSerialization(t *testing.T) {
	fr := NewFunctionResult("test").
		AddAction("hangup", true).
		SetPostProcess(true)

	m := fr.ToMap()
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
	if parsed["response"] != "test" {
		t.Errorf("response = %v, want %q", parsed["response"], "test")
	}
	if parsed["post_process"] != true {
		t.Errorf("post_process = %v, want true", parsed["post_process"])
	}
}

// --- Method Chaining ---

func TestMethodChaining(t *testing.T) {
	fr := NewFunctionResult("start").
		SetResponse("updated").
		SetPostProcess(true).
		AddAction("say", "hello").
		AddAction("hangup", true)

	m := fr.ToMap()
	if m["response"] != "updated" {
		t.Errorf("response = %v, want %q", m["response"], "updated")
	}
	if m["post_process"] != true {
		t.Error("post_process should be true")
	}
	actions, ok := m["action"].([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", m["action"])
	}
	if len(actions) != 2 {
		t.Fatalf("actions length = %d, want 2", len(actions))
	}
}

func TestAddActions(t *testing.T) {
	fr := NewFunctionResult("test").
		AddActions([]map[string]any{
			{"say": "hello"},
			{"hangup": true},
		})

	m := fr.ToMap()
	actions, ok := m["action"].([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", m["action"])
	}
	if len(actions) != 2 {
		t.Fatalf("actions length = %d, want 2", len(actions))
	}
	if actions[0]["say"] != "hello" {
		t.Errorf("first action = %v, want say:hello", actions[0])
	}
}

// --- Call Control Actions ---

func TestConnect(t *testing.T) {
	fr := NewFunctionResult("Transferring").
		Connect(ConnectOptions{Destination: "+15551234567", Final: true, From: "+15559876543"})

	m := fr.ToMap()
	actions, ok := m["action"].([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", m["action"])
	}
	if len(actions) != 1 {
		t.Fatalf("actions length = %d, want 1", len(actions))
	}

	action := actions[0]
	if action["transfer"] != "true" {
		t.Errorf("transfer = %v, want %q", action["transfer"], "true")
	}

	swml, ok := action["SWML"].(map[string]any)
	if !ok {
		t.Fatal("SWML should be a map")
	}
	sections, ok := swml["sections"].(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", swml["sections"])
	}
	main, ok := sections["main"].([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", sections["main"])
	}
	connectVerb, ok := main[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", main[0])
	}
	connectParams, ok := connectVerb["connect"].(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", connectVerb["connect"])
	}

	if connectParams["to"] != "+15551234567" {
		t.Errorf("to = %v, want +15551234567", connectParams["to"])
	}
	if connectParams["from"] != "+15559876543" {
		t.Errorf("from = %v, want +15559876543", connectParams["from"])
	}
}

func TestConnectNoFrom(t *testing.T) {
	fr := NewFunctionResult("Transferring").
		Connect(ConnectOptions{Destination: "+15551234567", Final: false})

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	action := actions[0]
	if action["transfer"] != "false" {
		t.Errorf("transfer = %v, want %q", action["transfer"], "false")
	}

	swml := as[map[string]any](t, action["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	connectVerb := as[map[string]any](t, main[0])
	connectParams := as[map[string]any](t, connectVerb["connect"])

	if _, ok := connectParams["from"]; ok {
		t.Error("from should not be present when empty")
	}
}

func TestSwmlTransfer(t *testing.T) {
	fr := NewFunctionResult("Transferring").
		SwmlTransfer("https://example.com/swml", "Goodbye!", true)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	action := actions[0]
	if action["transfer"] != "true" {
		t.Errorf("transfer = %v, want %q", action["transfer"], "true")
	}

	swml := as[map[string]any](t, action["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	if len(main) != 2 {
		t.Fatalf("main verbs length = %d, want 2", len(main))
	}

	setVerb := as[map[string]any](t, main[0])
	setData := as[map[string]any](t, setVerb["set"])
	if setData["ai_response"] != "Goodbye!" {
		t.Errorf("ai_response = %v, want %q", setData["ai_response"], "Goodbye!")
	}

	transferVerb := as[map[string]any](t, main[1])
	transferData := as[map[string]any](t, transferVerb["transfer"])
	if transferData["dest"] != "https://example.com/swml" {
		t.Errorf("dest = %v, want %q", transferData["dest"], "https://example.com/swml")
	}
}

func TestHangup(t *testing.T) {
	fr := NewFunctionResult("Goodbye").Hangup()

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if len(actions) != 1 {
		t.Fatalf("actions length = %d, want 1", len(actions))
	}
	if actions[0]["hangup"] != true {
		t.Errorf("hangup = %v, want true", actions[0]["hangup"])
	}
}

func TestHold(t *testing.T) {
	fr := NewFunctionResult("Please hold").Hold(120)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["hold"] != 120 {
		t.Errorf("hold = %v, want 120", actions[0]["hold"])
	}
}

func TestHoldClampMin(t *testing.T) {
	fr := NewFunctionResult("hold").Hold(-10)
	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["hold"] != 0 {
		t.Errorf("hold = %v, want 0 (clamped)", actions[0]["hold"])
	}
}

func TestHoldClampMax(t *testing.T) {
	fr := NewFunctionResult("hold").Hold(9999)
	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["hold"] != 900 {
		t.Errorf("hold = %v, want 900 (clamped)", actions[0]["hold"])
	}
}

func TestWaitForUserAnswerFirst(t *testing.T) {
	fr := NewFunctionResult("wait").WaitForUser(WaitForUserOptions{AnswerFirst: true})
	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["wait_for_user"] != "answer_first" {
		t.Errorf("wait_for_user = %v, want %q", actions[0]["wait_for_user"], "answer_first")
	}
}

func TestWaitForUserTimeout(t *testing.T) {
	timeout := 30
	fr := NewFunctionResult("wait").WaitForUser(WaitForUserOptions{Timeout: &timeout})
	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["wait_for_user"] != 30 {
		t.Errorf("wait_for_user = %v, want 30", actions[0]["wait_for_user"])
	}
}

func TestWaitForUserEnabled(t *testing.T) {
	enabled := false
	fr := NewFunctionResult("wait").WaitForUser(WaitForUserOptions{Enabled: &enabled})
	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["wait_for_user"] != false {
		t.Errorf("wait_for_user = %v, want false", actions[0]["wait_for_user"])
	}
}

func TestWaitForUserDefault(t *testing.T) {
	fr := NewFunctionResult("wait").WaitForUser(WaitForUserOptions{})
	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["wait_for_user"] != true {
		t.Errorf("wait_for_user = %v, want true", actions[0]["wait_for_user"])
	}
}

func TestStop(t *testing.T) {
	fr := NewFunctionResult("stopping").Stop()
	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["stop"] != true {
		t.Errorf("stop = %v, want true", actions[0]["stop"])
	}
}

// --- State & Data Management ---

func TestUpdateGlobalData(t *testing.T) {
	fr := NewFunctionResult("updated").
		UpdateGlobalData(map[string]any{"key": "value", "count": 42})

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	data := as[map[string]any](t, actions[0]["set_global_data"])
	if data["key"] != "value" {
		t.Errorf("key = %v, want %q", data["key"], "value")
	}
	if data["count"] != 42 {
		t.Errorf("count = %v, want 42", data["count"])
	}
}

func TestRemoveGlobalData(t *testing.T) {
	fr := NewFunctionResult("removed").
		RemoveGlobalData([]string{"key1", "key2"})

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	keys := as[[]string](t, actions[0]["unset_global_data"])
	if len(keys) != 2 || keys[0] != "key1" || keys[1] != "key2" {
		t.Errorf("keys = %v, want [key1, key2]", keys)
	}
}

func TestRemoveGlobalDataKey(t *testing.T) {
	fr := NewFunctionResult("removed").
		RemoveGlobalDataKey("single-key")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	// Single key variant emits bare string, not array — matches Python Union[str, List[str]] behavior
	key := as[string](t, actions[0]["unset_global_data"])
	if key != "single-key" {
		t.Errorf("unset_global_data = %v, want %q", key, "single-key")
	}
}

func TestSetMetadata(t *testing.T) {
	fr := NewFunctionResult("metadata").
		SetMetadata(map[string]any{"session": "abc123"})

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	data := as[map[string]any](t, actions[0]["set_meta_data"])
	if data["session"] != "abc123" {
		t.Errorf("session = %v, want %q", data["session"], "abc123")
	}
}

func TestRemoveMetadata(t *testing.T) {
	fr := NewFunctionResult("remove").
		RemoveMetadata([]string{"old_key"})

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	keys := as[[]string](t, actions[0]["unset_meta_data"])
	if len(keys) != 1 || keys[0] != "old_key" {
		t.Errorf("keys = %v, want [old_key]", keys)
	}
}

func TestRemoveMetadataKey(t *testing.T) {
	fr := NewFunctionResult("remove").
		RemoveMetadataKey("single-meta-key")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	// Single key variant emits bare string, not array — matches Python Union[str, List[str]] behavior
	key := as[string](t, actions[0]["unset_meta_data"])
	if key != "single-meta-key" {
		t.Errorf("unset_meta_data = %v, want %q", key, "single-meta-key")
	}
}

func TestSwmlUserEvent(t *testing.T) {
	fr := NewFunctionResult("event sent").
		SwmlUserEvent(map[string]any{"type": "cards_dealt", "score": 21})

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	userEvent := as[map[string]any](t, verb["user_event"])
	event := as[map[string]any](t, userEvent["event"])

	if event["type"] != "cards_dealt" {
		t.Errorf("type = %v, want %q", event["type"], "cards_dealt")
	}
	if event["score"] != 21 {
		t.Errorf("score = %v, want 21", event["score"])
	}
}

func TestSwmlChangeStep(t *testing.T) {
	fr := NewFunctionResult("changing step").
		SwmlChangeStep("betting")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	// Python emits {"change_step": "betting"} — plain string, not a struct
	if actions[0]["change_step"] != "betting" {
		t.Errorf("change_step = %v, want %q", actions[0]["change_step"], "betting")
	}
}

func TestSwmlChangeContext(t *testing.T) {
	fr := NewFunctionResult("changing context").
		SwmlChangeContext("support")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	// Python emits {"change_context": "support"} — plain string, not a struct
	if actions[0]["change_context"] != "support" {
		t.Errorf("change_context = %v, want %q", actions[0]["change_context"], "support")
	}
}

func TestSwitchContextSimple(t *testing.T) {
	fr := NewFunctionResult("switch").
		SwitchContext("You are a helper.", "", false, false, false)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	cs := actions[0]["context_switch"]
	if cs != "You are a helper." {
		t.Errorf("context_switch = %v, want simple string", cs)
	}
}

func TestSwitchContextAdvanced(t *testing.T) {
	fr := NewFunctionResult("switch").
		SwitchContext("new prompt", "user msg", true, false, false)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	cs := as[map[string]any](t, actions[0]["context_switch"])
	if cs["system_prompt"] != "new prompt" {
		t.Errorf("system_prompt = %v", cs["system_prompt"])
	}
	if cs["user_prompt"] != "user msg" {
		t.Errorf("user_prompt = %v", cs["user_prompt"])
	}
	if cs["consolidate"] != true {
		t.Error("consolidate should be true")
	}
	if _, ok := cs["full_reset"]; ok {
		t.Error("full_reset should not be present when false")
	}
}

func TestSwitchContextIsolated(t *testing.T) {
	fr := NewFunctionResult("switch").
		SwitchContext("prompt", "", false, false, true)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	cs := as[map[string]any](t, actions[0]["context_switch"])
	if cs["isolated"] != true {
		t.Error("isolated should be true")
	}
}

func TestReplaceInHistoryString(t *testing.T) {
	fr := NewFunctionResult("replace").
		ReplaceInHistory("summary text")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["replace_in_history"] != "summary text" {
		t.Errorf("replace_in_history = %v, want %q", actions[0]["replace_in_history"], "summary text")
	}
}

func TestReplaceInHistoryBool(t *testing.T) {
	fr := NewFunctionResult("replace").
		ReplaceInHistory(true)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["replace_in_history"] != true {
		t.Errorf("replace_in_history = %v, want true", actions[0]["replace_in_history"])
	}
}

// --- Media Control ---

func TestSay(t *testing.T) {
	fr := NewFunctionResult("speaking").Say("Hello there!")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["say"] != "Hello there!" {
		t.Errorf("say = %v, want %q", actions[0]["say"], "Hello there!")
	}
}

func TestPlayBackgroundFile(t *testing.T) {
	fr := NewFunctionResult("playing").
		PlayBackgroundFile("music.mp3", false)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["playback_bg"] != "music.mp3" {
		t.Errorf("playback_bg = %v, want %q", actions[0]["playback_bg"], "music.mp3")
	}
}

func TestPlayBackgroundFileWait(t *testing.T) {
	fr := NewFunctionResult("playing").
		PlayBackgroundFile("music.mp3", true)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	bg := as[map[string]any](t, actions[0]["playback_bg"])
	if bg["file"] != "music.mp3" {
		t.Errorf("file = %v, want %q", bg["file"], "music.mp3")
	}
	if bg["wait"] != true {
		t.Error("wait should be true")
	}
}

func TestStopBackgroundFile(t *testing.T) {
	fr := NewFunctionResult("stopping").StopBackgroundFile()

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["stop_playback_bg"] != true {
		t.Errorf("stop_playback_bg = %v, want true", actions[0]["stop_playback_bg"])
	}
}

func TestRecordCall(t *testing.T) {
	fr := NewFunctionResult("recording").
		RecordCall("rec-123", true, "wav", "both", nil)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["record_call"])

	if params["control_id"] != "rec-123" {
		t.Errorf("control_id = %v, want %q", params["control_id"], "rec-123")
	}
	if params["stereo"] != true {
		t.Errorf("stereo = %v, want true", params["stereo"])
	}
	if params["format"] != "wav" {
		t.Errorf("format = %v, want %q", params["format"], "wav")
	}
	if params["direction"] != "both" {
		t.Errorf("direction = %v, want %q", params["direction"], "both")
	}
	// Python emits beep and input_sensitivity UNCONDITIONALLY on every call
	// (function_result.py:921-928 — beep:false, input_sensitivity:44.0 defaults).
	// With nil opts, both defaults must still be present.
	beep, ok := params["beep"]
	if !ok {
		t.Error("beep must always be present (Python emits it unconditionally)")
	} else if beep != false {
		t.Errorf("beep = %v, want false (default)", beep)
	}
	is, ok := params["input_sensitivity"]
	if !ok {
		t.Error("input_sensitivity must always be present (Python emits it unconditionally)")
	} else if is != 44.0 {
		t.Errorf("input_sensitivity = %v, want 44.0 (default)", is)
	}
}

// TestRecordCall_FormatEnumOrString proves the typed RecordFormat constant and
// the bare string literal produce the IDENTICAL format token in the emitted
// SWML record_call params. Real behavior — the verb is built and inspected, no
// mock.
func TestRecordCall_FormatEnumOrString(t *testing.T) {
	// The defined-string constant's value is the canonical wire token.
	if string(FormatWAV) != "wav" || string(FormatMP3) != "mp3" || string(FormatMP4) != "mp4" {
		t.Fatalf("RecordFormat consts = %q/%q/%q, want wav/mp3/mp4",
			string(FormatWAV), string(FormatMP3), string(FormatMP4))
	}

	extractFormat := func(fr *FunctionResult) any {
		actions := as[[]map[string]any](t, fr.ToMap()["action"])
		swml := as[map[string]any](t, actions[0]["SWML"])
		main := as[[]any](t, as[map[string]any](t, swml["sections"])["main"])
		return as[map[string]any](t, as[map[string]any](t, main[0])["record_call"])["format"]
	}

	// Typed constant path.
	fConst := extractFormat(NewFunctionResult("rec").RecordCall("id", true, FormatWAV, "both", nil))
	// Bare-string path (Python uses str).
	fStr := extractFormat(NewFunctionResult("rec").RecordCall("id", true, "wav", "both", nil))

	if fConst != "wav" {
		t.Errorf("format via FormatWAV = %v, want wav", fConst)
	}
	if fConst != fStr {
		t.Errorf("typed const (%v) and string (%v) produced different formats", fConst, fStr)
	}
}

func TestRecordCallNoControlID(t *testing.T) {
	fr := NewFunctionResult("recording").
		RecordCall("", false, "mp3", "speak", nil)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["record_call"])

	if _, ok := params["control_id"]; ok {
		t.Error("control_id should not be present when empty")
	}
	// Even on this nil-opts path, beep/input_sensitivity defaults are always emitted.
	if params["beep"] != false {
		t.Errorf("beep = %v, want false (always emitted, default)", params["beep"])
	}
	if params["input_sensitivity"] != 44.0 {
		t.Errorf("input_sensitivity = %v, want 44.0 (always emitted, default)", params["input_sensitivity"])
	}
}

func TestRecordCallWithOptions(t *testing.T) {
	fr := NewFunctionResult("recording").
		RecordCall("rec-456", false, "mp3", "speak", &RecordCallOptions{
			Terminators:       "#",
			Beep:              true,
			InputSensitivity:  40.0,
			InitialTimeout:    5.0,
			EndSilenceTimeout: 3.0,
			MaxLength:         120.0,
			StatusURL:         "https://example.com/recording-status",
		})

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["record_call"])

	if params["terminators"] != "#" {
		t.Errorf("terminators = %v, want %q", params["terminators"], "#")
	}
	if params["beep"] != true {
		t.Errorf("beep = %v, want true", params["beep"])
	}
	if params["input_sensitivity"] != 40.0 {
		t.Errorf("input_sensitivity = %v, want 40.0", params["input_sensitivity"])
	}
	if params["initial_timeout"] != 5.0 {
		t.Errorf("initial_timeout = %v, want 5.0", params["initial_timeout"])
	}
	if params["end_silence_timeout"] != 3.0 {
		t.Errorf("end_silence_timeout = %v, want 3.0", params["end_silence_timeout"])
	}
	if params["max_length"] != 120.0 {
		t.Errorf("max_length = %v, want 120.0", params["max_length"])
	}
	if params["status_url"] != "https://example.com/recording-status" {
		t.Errorf("status_url = %v", params["status_url"])
	}
}

// recordCallParams drives the real RecordCall verb and returns the serialized
// record_call params map for inspection (no mock — the SWML document is built
// and read back through ToMap).
func recordCallParams(t *testing.T, fr *FunctionResult) map[string]any {
	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	main := as[[]any](t, as[map[string]any](t, swml["sections"])["main"])
	return as[map[string]any](t, as[map[string]any](t, main[0])["record_call"])
}

// TestRecordCall_DirectionConstantsAreWireStrings proves (a) each RecordDirection
// constant equals its exact wire token. These tokens are emitted verbatim into
// the SWML record_call direction param (Python's
// valid_directions = ["speak", "listen", "both"]).
func TestRecordCall_DirectionConstantsAreWireStrings(t *testing.T) {
	cases := map[RecordDirection]string{
		RecordDirectionSpeak:  "speak",
		RecordDirectionListen: "listen",
		RecordDirectionBoth:   "both",
	}
	for c, want := range cases {
		if string(c) != want {
			t.Errorf("RecordDirection const %q != wire token %q", string(c), want)
		}
	}
	// Guard the 3-vocabulary trap: record_call uses "listen", NOT tap's "hear".
	if string(RecordDirectionListen) == string(TapDirectionHear) {
		t.Error("RecordDirection must use 'listen', distinct from TapDirection's 'hear'")
	}
}

// TestRecordCall_DirectionEnumOrStringByteIdentical proves (b) the typed
// RecordDirection constant and the equivalent bare string produce the
// BYTE-IDENTICAL record_call action. Real behavior: both drive the actual
// method, are serialized to JSON, and compared byte-for-byte.
func TestRecordCall_DirectionEnumOrStringByteIdentical(t *testing.T) {
	for _, d := range []struct {
		typed RecordDirection
		str   string
	}{
		{RecordDirectionSpeak, "speak"},
		{RecordDirectionListen, "listen"},
		{RecordDirectionBoth, "both"},
	} {
		typedJSON, err := json.Marshal(NewFunctionResult("rec").
			RecordCall("id", true, FormatWAV, d.typed, nil).ToMap())
		if err != nil {
			t.Fatalf("marshal typed: %v", err)
		}
		strJSON, err := json.Marshal(NewFunctionResult("rec").
			RecordCall("id", true, FormatWAV, RecordDirection(d.str), nil).ToMap())
		if err != nil {
			t.Fatalf("marshal string: %v", err)
		}
		if string(typedJSON) != string(strJSON) {
			t.Errorf("direction %q: typed and bare-string actions differ:\n typed=%s\n  str=%s",
				d.str, typedJSON, strJSON)
		}
		// And the emitted token must be exactly the wire string.
		if got := recordCallParams(t, NewFunctionResult("rec").
			RecordCall("id", true, FormatWAV, d.typed, nil))["direction"]; got != d.str {
			t.Errorf("emitted direction = %v, want %q", got, d.str)
		}
	}
}

// TestRecordCall_DirectionAllRoundTrip proves (c) every advertised
// RecordDirection round-trips onto the wire — each constant, when passed to the
// real method, appears unchanged in the serialized direction param.
func TestRecordCall_DirectionAllRoundTrip(t *testing.T) {
	for _, d := range []RecordDirection{RecordDirectionSpeak, RecordDirectionListen, RecordDirectionBoth} {
		got := recordCallParams(t, NewFunctionResult("rec").
			RecordCall("id", false, FormatMP3, d, nil))["direction"]
		if got != string(d) {
			t.Errorf("RecordDirection %q did not round-trip onto the wire (got %v)", string(d), got)
		}
	}
	// (d) Out-of-set rejection: the Python reference raises ValueError on a bad
	// direction, but the Go builder returns *FunctionResult for fluent chaining
	// and therefore performs NO client-side validation (a documented Go-idiom
	// divergence — the server validates; see PORT_OMISSIONS / journal §6). There
	// is consequently no method-side set to assert the type against here; the
	// type's set is instead pinned to the reference's valid_directions by the
	// constants above. (Matches the existing RecordFormat enum test, which is
	// likewise rejection-free for the same reason.)
}

func TestStopRecordCall(t *testing.T) {
	fr := NewFunctionResult("stop recording").
		StopRecordCall("rec-123")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["stop_record_call"])

	if params["control_id"] != "rec-123" {
		t.Errorf("control_id = %v, want %q", params["control_id"], "rec-123")
	}
}

// --- Speech & AI Config ---

func TestAddDynamicHints(t *testing.T) {
	hints := []any{"Cabby", map[string]any{"pattern": "cab bee", "replace": "Cabby"}}
	fr := NewFunctionResult("hints").AddDynamicHints(hints)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	h := as[[]any](t, actions[0]["add_dynamic_hints"])
	if len(h) != 2 {
		t.Fatalf("hints length = %d, want 2", len(h))
	}
	if h[0] != "Cabby" {
		t.Errorf("hint[0] = %v, want %q", h[0], "Cabby")
	}
}

func TestClearDynamicHints(t *testing.T) {
	fr := NewFunctionResult("clear").ClearDynamicHints()

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	ch, ok := actions[0]["clear_dynamic_hints"].(map[string]any)
	if !ok {
		t.Fatal("clear_dynamic_hints should be a map")
	}
	if len(ch) != 0 {
		t.Errorf("clear_dynamic_hints should be empty map, got %v", ch)
	}
}

func TestSetEndOfSpeechTimeout(t *testing.T) {
	fr := NewFunctionResult("timeout").SetEndOfSpeechTimeout(500)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["end_of_speech_timeout"] != 500 {
		t.Errorf("end_of_speech_timeout = %v, want 500", actions[0]["end_of_speech_timeout"])
	}
}

func TestSetSpeechEventTimeout(t *testing.T) {
	fr := NewFunctionResult("timeout").SetSpeechEventTimeout(1000)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["speech_event_timeout"] != 1000 {
		t.Errorf("speech_event_timeout = %v, want 1000", actions[0]["speech_event_timeout"])
	}
}

func TestToggleFunctions(t *testing.T) {
	toggles := []map[string]any{
		{"function": "search", "active": true},
		{"function": "transfer", "active": false},
	}
	fr := NewFunctionResult("toggle").ToggleFunctions(toggles)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	tf := as[[]map[string]any](t, actions[0]["toggle_functions"])
	if len(tf) != 2 {
		t.Fatalf("toggles length = %d, want 2", len(tf))
	}
	if tf[0]["function"] != "search" {
		t.Errorf("toggle[0].function = %v, want %q", tf[0]["function"], "search")
	}
	if tf[1]["active"] != false {
		t.Errorf("toggle[1].active = %v, want false", tf[1]["active"])
	}
}

func TestEnableFunctionsOnTimeout(t *testing.T) {
	fr := NewFunctionResult("timeout").EnableFunctionsOnTimeout(true)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["functions_on_speaker_timeout"] != true {
		t.Errorf("functions_on_speaker_timeout = %v, want true", actions[0]["functions_on_speaker_timeout"])
	}
}

func TestEnableExtensiveData(t *testing.T) {
	fr := NewFunctionResult("data").EnableExtensiveData(true)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	if actions[0]["extensive_data"] != true {
		t.Errorf("extensive_data = %v, want true", actions[0]["extensive_data"])
	}
}

func TestUpdateSettings(t *testing.T) {
	fr := NewFunctionResult("settings").
		UpdateSettings(map[string]any{"temperature": 0.5, "top_p": 0.9})

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	s := as[map[string]any](t, actions[0]["settings"])
	if s["temperature"] != 0.5 {
		t.Errorf("temperature = %v, want 0.5", s["temperature"])
	}
}

// --- Advanced Features ---

func TestExecuteSwmlMap(t *testing.T) {
	content := map[string]any{
		"version": "1.0.0",
		"sections": map[string]any{
			"main": []any{map[string]any{"answer": map[string]any{}}},
		},
	}
	fr := NewFunctionResult("exec").ExecuteSwml(content, false)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	if swml["version"] != "1.0.0" {
		t.Errorf("version = %v, want %q", swml["version"], "1.0.0")
	}
	if _, ok := swml["transfer"]; ok {
		t.Error("transfer should not be present when false")
	}
}

func TestExecuteSwmlMapWithTransfer(t *testing.T) {
	content := map[string]any{"version": "1.0.0"}
	fr := NewFunctionResult("exec").ExecuteSwml(content, true)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	if swml["transfer"] != "true" {
		t.Errorf("transfer = %v, want %q", swml["transfer"], "true")
	}
}

func TestExecuteSwmlStringParsesJSON(t *testing.T) {
	// Python json.loads() a valid JSON-object string and spreads the parsed
	// document at top level (function_result.py:411-417). The version key lands
	// at the top of the SWML action, and there is NO raw_swml wrapper.
	fr := NewFunctionResult("exec").ExecuteSwml(`{"version":"1.0.0","sections":{"main":[]}}`, false)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	if _, ok := swml["raw_swml"]; ok {
		t.Errorf("a valid JSON-object string should be parsed and spread, not wrapped in raw_swml; got %v", swml["raw_swml"])
	}
	if swml["version"] != "1.0.0" {
		t.Errorf("version = %v, want %q (spread from parsed JSON)", swml["version"], "1.0.0")
	}
	if _, ok := swml["sections"]; !ok {
		t.Error("sections key should be spread from the parsed JSON document")
	}
}

func TestExecuteSwmlStringParsedWithTransfer(t *testing.T) {
	// A parsed JSON-object string still gets the transfer key added on top.
	fr := NewFunctionResult("exec").ExecuteSwml(`{"version":"1.0.0"}`, true)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	if swml["version"] != "1.0.0" {
		t.Errorf("version = %v, want %q", swml["version"], "1.0.0")
	}
	if swml["transfer"] != "true" {
		t.Errorf("transfer = %v, want %q", swml["transfer"], "true")
	}
	if _, ok := swml["raw_swml"]; ok {
		t.Error("raw_swml should not be present for a valid JSON-object string")
	}
}

func TestExecuteSwmlStringInvalidJSONFallsBackToRawSwml(t *testing.T) {
	// Python falls back to {"raw_swml": v} only on a JSONDecodeError. A
	// non-JSON string is preserved verbatim under raw_swml.
	raw := "this is not json"
	fr := NewFunctionResult("exec").ExecuteSwml(raw, false)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	if swml["raw_swml"] != raw {
		t.Errorf("raw_swml = %v, want %q", swml["raw_swml"], raw)
	}
}

func TestExecuteSwmlDoesNotMutateInput(t *testing.T) {
	content := map[string]any{"version": "1.0.0"}
	NewFunctionResult("exec").ExecuteSwml(content, true)

	if _, ok := content["transfer"]; ok {
		t.Error("original content should not be mutated")
	}
}

func TestJoinConference(t *testing.T) {
	fr := NewFunctionResult("joining").
		JoinConference("my-conf", &JoinConferenceOptions{
			Muted:   true,
			Beep:    "onEnter",
			WaitURL: "https://example.com/hold.mp3",
		})

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["join_conference"])

	if params["name"] != "my-conf" {
		t.Errorf("name = %v, want %q", params["name"], "my-conf")
	}
	if params["muted"] != true {
		t.Errorf("muted = %v, want true", params["muted"])
	}
	if params["beep"] != "onEnter" {
		t.Errorf("beep = %v, want %q", params["beep"], "onEnter")
	}
	if params["wait_url"] != "https://example.com/hold.mp3" {
		t.Errorf("wait_url = %v", params["wait_url"])
	}
}

func TestJoinConferenceDefaults(t *testing.T) {
	// All-default case: Python emits the SIMPLE form — the join_conference value
	// is the bare conference-name STRING, not an object (function_result.py:1124).
	// Matching that exactly is what keeps the wire byte-identical to the
	// reference (and to the other ports). Earlier this asserted an object form,
	// which was a drift-0 emission bug the cross-port emission differ caught.
	fr := NewFunctionResult("joining").
		JoinConference("simple-conf", nil)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])

	name, ok := verb["join_conference"].(string)
	if !ok {
		t.Fatalf("join_conference should be a bare string in the simple form, got %T (%v)",
			verb["join_conference"], verb["join_conference"])
	}
	if name != "simple-conf" {
		t.Errorf("join_conference = %q, want %q", name, "simple-conf")
	}
}

func TestJoinConferenceSimpleVsFull(t *testing.T) {
	// A single non-default option flips the emission from the bare-string simple
	// form to the full object form (parity with Python's branch at
	// function_result.py:1124 vs :1134).
	full := NewFunctionResult("joining").
		JoinConference("conf", &JoinConferenceOptions{Muted: true})
	actions := as[[]map[string]any](t, full.ToMap()["action"])
	verb := as[map[string]any](t, as[[]any](t, as[map[string]any](t, as[map[string]any](t, actions[0]["SWML"])["sections"])["main"])[0])
	params, ok := verb["join_conference"].(map[string]any)
	if !ok {
		t.Fatalf("a non-default option must produce the object form, got %T", verb["join_conference"])
	}
	if params["name"] != "conf" {
		t.Errorf("name = %v, want %q", params["name"], "conf")
	}
	if params["muted"] != true {
		t.Errorf("muted = %v, want true", params["muted"])
	}
}

func TestJoinConferenceFullOptions(t *testing.T) {
	fr := NewFunctionResult("joining").
		JoinConference("full-conf", &JoinConferenceOptions{
			Record:                        "record-from-start",
			Region:                        "us1",
			Trim:                          "do-not-trim",
			Coach:                         "call-sid-123",
			StatusCallbackEvent:           "start end join",
			StatusCallback:                "https://example.com/status",
			StatusCallbackMethod:          "GET",
			RecordingStatusCallback:       "https://example.com/rec-status",
			RecordingStatusCallbackMethod: "GET",
			RecordingStatusCallbackEvent:  "in-progress",
			MaxParticipants:               50,
		})

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["join_conference"])

	if params["record"] != "record-from-start" {
		t.Errorf("record = %v", params["record"])
	}
	if params["region"] != "us1" {
		t.Errorf("region = %v", params["region"])
	}
	if params["trim"] != "do-not-trim" {
		t.Errorf("trim = %v", params["trim"])
	}
	if params["coach"] != "call-sid-123" {
		t.Errorf("coach = %v", params["coach"])
	}
	if params["max_participants"] != 50 {
		t.Errorf("max_participants = %v, want 50", params["max_participants"])
	}
}

func TestJoinRoom(t *testing.T) {
	fr := NewFunctionResult("joining room").JoinRoom("my-room")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["join_room"])

	if params["name"] != "my-room" {
		t.Errorf("name = %v, want %q", params["name"], "my-room")
	}
}

func TestSipRefer(t *testing.T) {
	fr := NewFunctionResult("referring").SIPRefer("sip:agent@example.com")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["sip_refer"])

	if params["to_uri"] != "sip:agent@example.com" {
		t.Errorf("to_uri = %v", params["to_uri"])
	}
}

func TestTap(t *testing.T) {
	fr := NewFunctionResult("tapping").
		Tap("rtp://192.168.1.1:5000", "tap-1", "speak", "PCMA", 0, "")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["tap"])

	if params["uri"] != "rtp://192.168.1.1:5000" {
		t.Errorf("uri = %v", params["uri"])
	}
	if params["control_id"] != "tap-1" {
		t.Errorf("control_id = %v", params["control_id"])
	}
	if params["direction"] != "speak" {
		t.Errorf("direction = %v, want %q", params["direction"], "speak")
	}
	if params["codec"] != "PCMA" {
		t.Errorf("codec = %v, want %q", params["codec"], "PCMA")
	}
}

func TestTapDefaults(t *testing.T) {
	fr := NewFunctionResult("tapping").
		Tap("ws://example.com", "", "both", "PCMU", 0, "")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["tap"])

	if _, ok := params["control_id"]; ok {
		t.Error("control_id should not be present when empty")
	}
	if _, ok := params["direction"]; ok {
		t.Error("direction should not be present when 'both' (default)")
	}
	if _, ok := params["codec"]; ok {
		t.Error("codec should not be present when 'PCMU' (default)")
	}
}

func TestTapWithRtpPtimeAndStatusURL(t *testing.T) {
	fr := NewFunctionResult("tapping").
		Tap("rtp://192.168.1.1:5000", "", "both", "PCMU", 30, "https://example.com/tap-status")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["tap"])

	if params["rtp_ptime"] != 30 {
		t.Errorf("rtp_ptime = %v, want 30", params["rtp_ptime"])
	}
	if params["status_url"] != "https://example.com/tap-status" {
		t.Errorf("status_url = %v", params["status_url"])
	}
}

// tapParamsOf drives the real Tap verb and returns the serialized tap params map
// for inspection (no mock — the SWML document is built and read back via ToMap).
func tapParamsOf(t *testing.T, fr *FunctionResult) map[string]any {
	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	main := as[[]any](t, as[map[string]any](t, swml["sections"])["main"])
	return as[map[string]any](t, as[map[string]any](t, main[0])["tap"])
}

// TestTap_DirectionConstantsAreWireStrings proves (a) each TapDirection constant
// equals its exact wire token (Python's valid_directions = ["speak","hear","both"]).
func TestTap_DirectionConstantsAreWireStrings(t *testing.T) {
	cases := map[TapDirection]string{
		TapDirectionSpeak: "speak",
		TapDirectionHear:  "hear",
		TapDirectionBoth:  "both",
	}
	for c, want := range cases {
		if string(c) != want {
			t.Errorf("TapDirection const %q != wire token %q", string(c), want)
		}
	}
	// Guard the 3-vocabulary trap: tap uses "hear", NOT record_call's "listen".
	if string(TapDirectionHear) == string(RecordDirectionListen) {
		t.Error("TapDirection must use 'hear', distinct from RecordDirection's 'listen'")
	}
}

// TestTap_CodecConstantsAreWireStrings proves (a) each Codec constant equals its
// exact wire token (Python's valid_codecs = ["PCMU","PCMA"]).
func TestTap_CodecConstantsAreWireStrings(t *testing.T) {
	cases := map[Codec]string{
		CodecPCMU: "PCMU",
		CodecPCMA: "PCMA",
	}
	for c, want := range cases {
		if string(c) != want {
			t.Errorf("Codec const %q != wire token %q", string(c), want)
		}
	}
}

// TestTap_DirectionAndCodecEnumOrStringByteIdentical proves (b) the typed
// TapDirection/Codec constants and their equivalent bare strings produce the
// BYTE-IDENTICAL tap action. Real behavior: both drive the actual method, are
// serialized to JSON, and compared byte-for-byte. Uses "speak"/"PCMA" (the
// non-default arms) so the params actually emit (Tap omits the default
// both/PCMU), exercising the string(direction)/string(codec) wire conversion.
func TestTap_DirectionAndCodecEnumOrStringByteIdentical(t *testing.T) {
	type pair struct {
		dirTyped   TapDirection
		dirStr     string
		codecTyped Codec
		codecStr   string
	}
	for _, p := range []pair{
		{TapDirectionSpeak, "speak", CodecPCMA, "PCMA"},
		{TapDirectionHear, "hear", CodecPCMA, "PCMA"},
		{TapDirectionBoth, "both", CodecPCMU, "PCMU"},
	} {
		typedJSON, err := json.Marshal(NewFunctionResult("tap").
			Tap("rtp://h:1", "id", p.dirTyped, p.codecTyped, 0, "").ToMap())
		if err != nil {
			t.Fatalf("marshal typed: %v", err)
		}
		strJSON, err := json.Marshal(NewFunctionResult("tap").
			Tap("rtp://h:1", "id", TapDirection(p.dirStr), Codec(p.codecStr), 0, "").ToMap())
		if err != nil {
			t.Fatalf("marshal string: %v", err)
		}
		if string(typedJSON) != string(strJSON) {
			t.Errorf("dir=%q codec=%q: typed and bare-string tap actions differ:\n typed=%s\n  str=%s",
				p.dirStr, p.codecStr, typedJSON, strJSON)
		}
	}
}

// TestTap_DirectionAndCodecAllRoundTrip proves (c) every advertised TapDirection
// and Codec round-trips onto the wire. The default arms (both/PCMU) are
// deliberately OMITTED by Tap (compatibility with Python's "differ from defaults"
// emission), so they are asserted ABSENT; the non-default arms must appear
// verbatim.
func TestTap_DirectionAndCodecAllRoundTrip(t *testing.T) {
	// Non-default directions are emitted verbatim.
	for _, d := range []TapDirection{TapDirectionSpeak, TapDirectionHear} {
		got := tapParamsOf(t, NewFunctionResult("tap").Tap("ws://x", "", d, CodecPCMU, 0, ""))["direction"]
		if got != string(d) {
			t.Errorf("TapDirection %q did not round-trip (got %v)", string(d), got)
		}
	}
	// The default direction (both) is suppressed.
	if _, ok := tapParamsOf(t, NewFunctionResult("tap").
		Tap("ws://x", "", TapDirectionBoth, CodecPCMU, 0, ""))["direction"]; ok {
		t.Error("TapDirectionBoth (default) must be omitted from tap params")
	}
	// The non-default codec is emitted verbatim.
	if got := tapParamsOf(t, NewFunctionResult("tap").
		Tap("ws://x", "", TapDirectionBoth, CodecPCMA, 0, ""))["codec"]; got != "PCMA" {
		t.Errorf("CodecPCMA did not round-trip (got %v)", got)
	}
	// The default codec (PCMU) is suppressed.
	if _, ok := tapParamsOf(t, NewFunctionResult("tap").
		Tap("ws://x", "", TapDirectionBoth, CodecPCMU, 0, ""))["codec"]; ok {
		t.Error("CodecPCMU (default) must be omitted from tap params")
	}
	// (d) Out-of-set rejection: as with RecordCall/RecordFormat, the Go Tap
	// builder returns *FunctionResult for chaining and does NO client-side
	// validation (documented Go-idiom divergence; the server validates). The
	// type's set is pinned to the reference's valid_directions/valid_codecs by
	// the constant assertions above rather than by a method-side rejection.
}

func TestStopTap(t *testing.T) {
	fr := NewFunctionResult("stop tap").StopTap("tap-1")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["stop_tap"])

	if params["control_id"] != "tap-1" {
		t.Errorf("control_id = %v", params["control_id"])
	}
}

func TestStopTapNoControlID(t *testing.T) {
	fr := NewFunctionResult("stop tap").StopTap("")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["stop_tap"])

	if len(params) != 0 {
		t.Errorf("stop_tap should be empty map, got %v", params)
	}
}

func TestSendSms(t *testing.T) {
	fr := NewFunctionResult("sms sent").
		SendSms("+15551234567", "+15559876543", "Hello!", nil, nil, "")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["send_sms"])

	if params["to_number"] != "+15551234567" {
		t.Errorf("to_number = %v", params["to_number"])
	}
	if params["from_number"] != "+15559876543" {
		t.Errorf("from_number = %v", params["from_number"])
	}
	if params["body"] != "Hello!" {
		t.Errorf("body = %v, want %q", params["body"], "Hello!")
	}
	if _, ok := params["media"]; ok {
		t.Error("media should not be present when nil")
	}
	if _, ok := params["tags"]; ok {
		t.Error("tags should not be present when nil")
	}
	if _, ok := params["region"]; ok {
		t.Error("region should not be present when empty")
	}
}

func TestSendSmsWithRegion(t *testing.T) {
	fr := NewFunctionResult("sms sent").
		SendSms("+15551234567", "+15559876543", "Hello!", nil, nil, "us-east")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["send_sms"])

	if params["region"] != "us-east" {
		t.Errorf("region = %v, want %q", params["region"], "us-east")
	}
}

func TestSendSmsWithMediaAndTags(t *testing.T) {
	fr := NewFunctionResult("sms sent").
		SendSms("+15551234567", "+15559876543", "",
			[]string{"https://example.com/image.jpg"},
			[]string{"support", "urgent"}, "")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	params := as[map[string]any](t, verb["send_sms"])

	if _, ok := params["body"]; ok {
		t.Error("body should not be present when empty")
	}
	media := as[[]string](t, params["media"])
	if len(media) != 1 || media[0] != "https://example.com/image.jpg" {
		t.Errorf("media = %v", media)
	}
	tags := as[[]string](t, params["tags"])
	if len(tags) != 2 {
		t.Errorf("tags = %v", tags)
	}
}

func TestPay(t *testing.T) {
	// With nil opts, Python defaults are applied and the default ai_response set verb is included
	fr := NewFunctionResult("processing payment").
		Pay("https://example.com/pay", nil)

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])

	// Default behavior includes the "set" verb with ai_response
	if len(main) != 2 {
		t.Fatalf("main verbs = %d, want 2 (set + pay)", len(main))
	}
	verb := as[map[string]any](t, main[1])
	params := as[map[string]any](t, verb["pay"])
	if params["payment_connector_url"] != "https://example.com/pay" {
		t.Errorf("payment_connector_url = %v", params["payment_connector_url"])
	}
	if params["input"] != "dtmf" {
		t.Errorf("input = %v, want %q", params["input"], "dtmf")
	}
}

func TestPayAlwaysEmitsSetVerb(t *testing.T) {
	// Python ALWAYS emits both the set{ai_response} verb and the pay verb
	// (function_result.py:870-878). A single-verb pay main is a shape Python can
	// never produce — there is no suppression path. Any AIResponse value (even a
	// bare "-") is emitted verbatim as the set verb's ai_response, never dropped.
	fr := NewFunctionResult("processing payment").
		Pay("https://example.com/pay", &PayOptions{AIResponse: "-"})

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])

	// Both verbs are always present: set first, then pay.
	if len(main) != 2 {
		t.Fatalf("main verbs = %d, want 2 (set + pay)", len(main))
	}

	setVerb := as[map[string]any](t, main[0])
	setData := as[map[string]any](t, setVerb["set"])
	if setData["ai_response"] != "-" {
		t.Errorf("ai_response = %v, want %q (emitted verbatim, not suppressed)", setData["ai_response"], "-")
	}

	verb := as[map[string]any](t, main[1])
	params := as[map[string]any](t, verb["pay"])
	if params["payment_connector_url"] != "https://example.com/pay" {
		t.Errorf("payment_connector_url = %v", params["payment_connector_url"])
	}
}

func TestPayWithCustomAiResponse(t *testing.T) {
	fr := NewFunctionResult("processing payment").
		Pay("https://example.com/pay", &PayOptions{AIResponse: "Payment complete"})

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])

	if len(main) != 2 {
		t.Fatalf("main verbs = %d, want 2 (set + pay)", len(main))
	}

	setVerb := as[map[string]any](t, main[0])
	setData := as[map[string]any](t, setVerb["set"])
	if setData["ai_response"] != "Payment complete" {
		t.Errorf("ai_response = %v", setData["ai_response"])
	}
}

func TestPayWithFullOptions(t *testing.T) {
	fr := NewFunctionResult("processing payment").
		Pay("https://example.com/pay", &PayOptions{
			InputMethod:     "voice",
			PaymentMethod:   "credit-card",
			Timeout:         10,
			MaxAttempts:     3,
			SecurityCode:    true,
			SecurityCodeSet: true,
			PostalCode:      false,
			TokenType:       "one-time",
			ChargeAmount:    "9.99",
			Currency:        "eur",
			Language:        "fr-FR",
			Voice:           "man",
			ValidCardTypes:  "visa",
			StatusURL:       "https://example.com/status",
			AIResponse:      "Payment processed",
		})

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])

	// set verb is always first, pay verb second (Python emits both unconditionally).
	if len(main) != 2 {
		t.Fatalf("main verbs = %d, want 2 (set + pay)", len(main))
	}
	setData := as[map[string]any](t, as[map[string]any](t, main[0])["set"])
	if setData["ai_response"] != "Payment processed" {
		t.Errorf("ai_response = %v, want %q", setData["ai_response"], "Payment processed")
	}
	verb := as[map[string]any](t, main[1])
	params := as[map[string]any](t, verb["pay"])

	if params["input"] != "voice" {
		t.Errorf("input = %v, want %q", params["input"], "voice")
	}
	if params["token_type"] != "one-time" {
		t.Errorf("token_type = %v, want %q", params["token_type"], "one-time")
	}
	if params["charge_amount"] != "9.99" {
		t.Errorf("charge_amount = %v, want %q", params["charge_amount"], "9.99")
	}
	if params["currency"] != "eur" {
		t.Errorf("currency = %v, want %q", params["currency"], "eur")
	}
	if params["status_url"] != "https://example.com/status" {
		t.Errorf("status_url = %v", params["status_url"])
	}
	if params["postal_code"] != "false" {
		t.Errorf("postal_code = %v, want %q", params["postal_code"], "false")
	}
}

// --- RPC Actions ---

func TestExecuteRpc(t *testing.T) {
	fr := NewFunctionResult("rpc").
		ExecuteRPC("custom.method", map[string]any{"key": "value"}, "", "")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	rpc := as[map[string]any](t, verb["execute_rpc"])

	if rpc["method"] != "custom.method" {
		t.Errorf("method = %v, want %q", rpc["method"], "custom.method")
	}
	// Python's execute_rpc verb never carries a jsonrpc key — that envelope
	// belongs to the RELAY/MCP transport layer, not the SWML verb.
	if _, ok := rpc["jsonrpc"]; ok {
		t.Errorf("jsonrpc should NOT be present in the execute_rpc verb, got %v", rpc["jsonrpc"])
	}
	params := as[map[string]any](t, rpc["params"])
	if params["key"] != "value" {
		t.Errorf("params.key = %v, want %q", params["key"], "value")
	}
}

func TestExecuteRpcNoParams(t *testing.T) {
	fr := NewFunctionResult("rpc").
		ExecuteRPC("simple.method", nil, "", "")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	rpc := as[map[string]any](t, verb["execute_rpc"])

	if _, ok := rpc["params"]; ok {
		t.Error("params should not be present when nil")
	}
}

func TestExecuteRpcWithCallIDAndNodeID(t *testing.T) {
	fr := NewFunctionResult("rpc").
		ExecuteRPC("my.method", map[string]any{"key": "val"}, "call-123", "node-456")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	rpc := as[map[string]any](t, verb["execute_rpc"])

	if rpc["call_id"] != "call-123" {
		t.Errorf("call_id = %v, want %q", rpc["call_id"], "call-123")
	}
	if rpc["node_id"] != "node-456" {
		t.Errorf("node_id = %v, want %q", rpc["node_id"], "node-456")
	}
}

func TestRpcDial(t *testing.T) {
	fr := NewFunctionResult("dialing").
		RPCDial("+15551234567", "+15559876543", "https://example.com/swml", "phone")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	rpc := as[map[string]any](t, verb["execute_rpc"])

	// Python uses bare "dial", not "calling.dial"
	if rpc["method"] != "dial" {
		t.Errorf("method = %v, want %q", rpc["method"], "dial")
	}
	// The jsonrpc-removal fix ripples through RPCDial (built on ExecuteRPC).
	if _, ok := rpc["jsonrpc"]; ok {
		t.Errorf("jsonrpc should NOT be present, got %v", rpc["jsonrpc"])
	}

	params := as[map[string]any](t, rpc["params"])
	if params["dest_swml"] != "https://example.com/swml" {
		t.Errorf("dest_swml = %v", params["dest_swml"])
	}

	devices := as[map[string]any](t, params["devices"])
	if devices["type"] != "phone" {
		t.Errorf("type = %v, want %q", devices["type"], "phone")
	}
	deviceParams := as[map[string]any](t, devices["params"])
	if deviceParams["to_number"] != "+15551234567" {
		t.Errorf("to_number = %v", deviceParams["to_number"])
	}
}

func TestRpcDialDefaultDeviceType(t *testing.T) {
	fr := NewFunctionResult("dialing").
		RPCDial("+15551234567", "+15559876543", "https://example.com/swml", "")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	rpc := as[map[string]any](t, verb["execute_rpc"])
	params := as[map[string]any](t, rpc["params"])

	devices := as[map[string]any](t, params["devices"])
	// Empty deviceType should default to "phone"
	if devices["type"] != "phone" {
		t.Errorf("type = %v, want %q (default)", devices["type"], "phone")
	}
}

func TestRpcAiMessage(t *testing.T) {
	fr := NewFunctionResult("messaging").
		RPCAiMessage("call-abc-123", "Please take a message", "")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	rpc := as[map[string]any](t, verb["execute_rpc"])

	// Python uses bare "ai_message", not "calling.ai_message"
	if rpc["method"] != "ai_message" {
		t.Errorf("method = %v, want %q", rpc["method"], "ai_message")
	}
	if rpc["call_id"] != "call-abc-123" {
		t.Errorf("call_id = %v, want %q", rpc["call_id"], "call-abc-123")
	}
	// The jsonrpc-removal fix ripples through RPCAiMessage (built on ExecuteRPC).
	if _, ok := rpc["jsonrpc"]; ok {
		t.Errorf("jsonrpc should NOT be present, got %v", rpc["jsonrpc"])
	}

	params := as[map[string]any](t, rpc["params"])
	if params["role"] != "system" {
		t.Errorf("role = %v, want %q", params["role"], "system")
	}
	if params["message_text"] != "Please take a message" {
		t.Errorf("message_text = %v", params["message_text"])
	}
}

func TestRpcAiMessageCustomRole(t *testing.T) {
	fr := NewFunctionResult("messaging").
		RPCAiMessage("call-abc-123", "Hello", "user")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	rpc := as[map[string]any](t, verb["execute_rpc"])
	params := as[map[string]any](t, rpc["params"])

	if params["role"] != "user" {
		t.Errorf("role = %v, want %q", params["role"], "user")
	}
}

func TestRpcAiUnhold(t *testing.T) {
	fr := NewFunctionResult("unholding").
		RPCAiUnhold("call-abc-123")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	swml := as[map[string]any](t, actions[0]["SWML"])
	sections := as[map[string]any](t, swml["sections"])
	main := as[[]any](t, sections["main"])
	verb := as[map[string]any](t, main[0])
	rpc := as[map[string]any](t, verb["execute_rpc"])

	// Python uses bare "ai_unhold", not "calling.ai_unhold"
	if rpc["method"] != "ai_unhold" {
		t.Errorf("method = %v, want %q", rpc["method"], "ai_unhold")
	}
	if rpc["call_id"] != "call-abc-123" {
		t.Errorf("call_id = %v, want %q", rpc["call_id"], "call-abc-123")
	}
	// The jsonrpc-removal fix ripples through RPCAiUnhold (built on ExecuteRPC).
	if _, ok := rpc["jsonrpc"]; ok {
		t.Errorf("jsonrpc should NOT be present, got %v", rpc["jsonrpc"])
	}
}

func TestSimulateUserInput(t *testing.T) {
	fr := NewFunctionResult("simulating").
		SimulateUserInput("I need help")

	actions := as[[]map[string]any](t, fr.ToMap()["action"])
	// Python emits {"user_input": "..."} — not "simulate_user_input"
	if actions[0]["user_input"] != "I need help" {
		t.Errorf("user_input = %v, want %q", actions[0]["user_input"], "I need help")
	}
}

// --- Edge Cases ---

func TestEmptyActionsSliceNotNil(t *testing.T) {
	fr := NewFunctionResult("test")
	if fr.actions == nil {
		t.Error("actions should be initialized as empty slice, not nil")
	}
}

func TestMultipleActionsChained(t *testing.T) {
	fr := NewFunctionResult("complex").
		Say("Please hold").
		Hold(60).
		UpdateGlobalData(map[string]any{"status": "on_hold"}).
		SetPostProcess(true)

	m := fr.ToMap()
	actions, ok := m["action"].([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", m["action"])
	}
	if len(actions) != 3 {
		t.Fatalf("actions length = %d, want 3", len(actions))
	}
	if m["post_process"] != true {
		t.Error("post_process should be true")
	}
}

func TestSetResponseOverwrites(t *testing.T) {
	fr := NewFunctionResult("initial").
		SetResponse("overwritten")

	m := fr.ToMap()
	if m["response"] != "overwritten" {
		t.Errorf("response = %v, want %q", m["response"], "overwritten")
	}
}

func TestStringMethod(t *testing.T) {
	fr := NewFunctionResult("Hello World").
		AddAction("say", "hi").
		SetPostProcess(true)

	s := fr.String()
	if s == "" {
		t.Error("String() should return non-empty representation")
	}
	// Just verify it doesn't panic and contains key info
	if !contains(s, "Hello World") {
		t.Errorf("String() = %q, should contain response", s)
	}
	if !contains(s, "actions=1") {
		t.Errorf("String() = %q, should contain action count", s)
	}
}

func TestStringMethodLongResponse(t *testing.T) {
	long := "This is a very long response that exceeds fifty characters in length easily"
	fr := NewFunctionResult(long)
	s := fr.String()
	if !contains(s, "...") {
		t.Errorf("String() should truncate long responses, got %q", s)
	}
}

func TestNilParamsAddActions(t *testing.T) {
	fr := NewFunctionResult("test")
	// Passing nil should not panic
	fr.AddActions(nil)
	m := fr.ToMap()
	if _, ok := m["action"]; ok {
		t.Error("action should not be present when no actions added")
	}
}

func TestAddActionsEmpty(t *testing.T) {
	fr := NewFunctionResult("test")
	fr.AddActions([]map[string]any{})
	m := fr.ToMap()
	if _, ok := m["action"]; ok {
		t.Error("action should not be present for empty actions slice")
	}
}

// --- Payment Helpers ---

func TestCreatePaymentPrompt(t *testing.T) {
	actions := []map[string]string{
		CreatePaymentAction("payment-card-number", "Please enter your card number"),
		CreatePaymentAction("payment-expiration-date", "Enter expiration date"),
	}
	prompt := CreatePaymentPrompt("credit-card-payment", actions, "", "")

	if prompt["for"] != "credit-card-payment" {
		t.Errorf("for = %v, want %q", prompt["for"], "credit-card-payment")
	}
	acts, ok := prompt["actions"].([]map[string]string)
	if !ok {
		t.Fatal("actions should be []map[string]string")
	}
	if len(acts) != 2 {
		t.Fatalf("actions length = %d, want 2", len(acts))
	}
	if acts[0]["type"] != "payment-card-number" {
		t.Errorf("action[0].type = %v, want %q", acts[0]["type"], "payment-card-number")
	}
	if acts[0]["phrase"] != "Please enter your card number" {
		t.Errorf("action[0].phrase = %v, want %q", acts[0]["phrase"], "Please enter your card number")
	}
	if acts[1]["type"] != "payment-expiration-date" {
		t.Errorf("action[1].type = %v, want %q", acts[1]["type"], "payment-expiration-date")
	}
	// No card_type or error_type when empty
	if _, ok := prompt["card_type"]; ok {
		t.Error("card_type should not be present when empty")
	}
	if _, ok := prompt["error_type"]; ok {
		t.Error("error_type should not be present when empty")
	}
}

func TestCreatePaymentPromptWithCardAndErrorType(t *testing.T) {
	actions := []map[string]string{
		CreatePaymentAction("Say", "Enter your card"),
	}
	prompt := CreatePaymentPrompt("card-entry", actions, "visa mastercard", "invalid-card-number")

	if prompt["card_type"] != "visa mastercard" {
		t.Errorf("card_type = %v, want %q", prompt["card_type"], "visa mastercard")
	}
	if prompt["error_type"] != "invalid-card-number" {
		t.Errorf("error_type = %v, want %q", prompt["error_type"], "invalid-card-number")
	}
}

func TestCreatePaymentAction(t *testing.T) {
	action := CreatePaymentAction("payment-card-number", "Enter your card number")

	if action["type"] != "payment-card-number" {
		t.Errorf("type = %v, want %q", action["type"], "payment-card-number")
	}
	if action["phrase"] != "Enter your card number" {
		t.Errorf("phrase = %v, want %q", action["phrase"], "Enter your card number")
	}
	if len(action) != 2 {
		t.Errorf("action should have exactly 2 keys, got %d", len(action))
	}
}

func TestCreatePaymentParameter(t *testing.T) {
	param := CreatePaymentParameter("min-postal-code-length", "5")

	if param["name"] != "min-postal-code-length" {
		t.Errorf("name = %v, want %q", param["name"], "min-postal-code-length")
	}
	if param["value"] != "5" {
		t.Errorf("value = %v, want %q", param["value"], "5")
	}
	if len(param) != 2 {
		t.Errorf("param should have exactly 2 keys, got %d", len(param))
	}
}

func TestCreatePaymentPrompt_EmptyActions(t *testing.T) {
	prompt := CreatePaymentPrompt("refund", []map[string]string{}, "", "")

	if prompt["for"] != "refund" {
		t.Errorf("for = %v, want %q", prompt["for"], "refund")
	}
	acts, ok := prompt["actions"].([]map[string]string)
	if !ok {
		t.Fatal("actions should be []map[string]string")
	}
	if len(acts) != 0 {
		t.Errorf("actions length = %d, want 0", len(acts))
	}
}

// helper for string containment check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
