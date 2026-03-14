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
	fr := NewFunctionResult("")
	m := fr.ToMap()
	if m["response"] != "" {
		t.Errorf("response = %v, want empty string", m["response"])
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
		SetPostProcess(true)

	m := fr.ToMap()
	pp, ok := m["post_process"]
	if !ok {
		t.Fatal("post_process should be present when true")
	}
	if pp != true {
		t.Errorf("post_process = %v, want true", pp)
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
	actions := m["action"].([]map[string]any)
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
	actions := m["action"].([]map[string]any)
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
		Connect("+15551234567", true, "+15559876543")

	m := fr.ToMap()
	actions := m["action"].([]map[string]any)
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
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	connectVerb := main[0].(map[string]any)
	connectParams := connectVerb["connect"].(map[string]any)

	if connectParams["to"] != "+15551234567" {
		t.Errorf("to = %v, want +15551234567", connectParams["to"])
	}
	if connectParams["from"] != "+15559876543" {
		t.Errorf("from = %v, want +15559876543", connectParams["from"])
	}
}

func TestConnectNoFrom(t *testing.T) {
	fr := NewFunctionResult("Transferring").
		Connect("+15551234567", false, "")

	actions := fr.ToMap()["action"].([]map[string]any)
	action := actions[0]
	if action["transfer"] != "false" {
		t.Errorf("transfer = %v, want %q", action["transfer"], "false")
	}

	swml := action["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	connectVerb := main[0].(map[string]any)
	connectParams := connectVerb["connect"].(map[string]any)

	if _, ok := connectParams["from"]; ok {
		t.Error("from should not be present when empty")
	}
}

func TestSwmlTransfer(t *testing.T) {
	fr := NewFunctionResult("Transferring").
		SwmlTransfer("https://example.com/swml", "Goodbye!", true)

	actions := fr.ToMap()["action"].([]map[string]any)
	action := actions[0]
	if action["transfer"] != "true" {
		t.Errorf("transfer = %v, want %q", action["transfer"], "true")
	}

	swml := action["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	if len(main) != 2 {
		t.Fatalf("main verbs length = %d, want 2", len(main))
	}

	setVerb := main[0].(map[string]any)
	setData := setVerb["set"].(map[string]any)
	if setData["ai_response"] != "Goodbye!" {
		t.Errorf("ai_response = %v, want %q", setData["ai_response"], "Goodbye!")
	}

	transferVerb := main[1].(map[string]any)
	transferData := transferVerb["transfer"].(map[string]any)
	if transferData["dest"] != "https://example.com/swml" {
		t.Errorf("dest = %v, want %q", transferData["dest"], "https://example.com/swml")
	}
}

func TestHangup(t *testing.T) {
	fr := NewFunctionResult("Goodbye").Hangup()

	actions := fr.ToMap()["action"].([]map[string]any)
	if len(actions) != 1 {
		t.Fatalf("actions length = %d, want 1", len(actions))
	}
	if actions[0]["hangup"] != true {
		t.Errorf("hangup = %v, want true", actions[0]["hangup"])
	}
}

func TestHold(t *testing.T) {
	fr := NewFunctionResult("Please hold").Hold(120)

	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["hold"] != 120 {
		t.Errorf("hold = %v, want 120", actions[0]["hold"])
	}
}

func TestHoldClampMin(t *testing.T) {
	fr := NewFunctionResult("hold").Hold(-10)
	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["hold"] != 0 {
		t.Errorf("hold = %v, want 0 (clamped)", actions[0]["hold"])
	}
}

func TestHoldClampMax(t *testing.T) {
	fr := NewFunctionResult("hold").Hold(9999)
	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["hold"] != 900 {
		t.Errorf("hold = %v, want 900 (clamped)", actions[0]["hold"])
	}
}

func TestWaitForUserAnswerFirst(t *testing.T) {
	fr := NewFunctionResult("wait").WaitForUser(nil, nil, true)
	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["wait_for_user"] != "answer_first" {
		t.Errorf("wait_for_user = %v, want %q", actions[0]["wait_for_user"], "answer_first")
	}
}

func TestWaitForUserTimeout(t *testing.T) {
	timeout := 30
	fr := NewFunctionResult("wait").WaitForUser(nil, &timeout, false)
	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["wait_for_user"] != 30 {
		t.Errorf("wait_for_user = %v, want 30", actions[0]["wait_for_user"])
	}
}

func TestWaitForUserEnabled(t *testing.T) {
	enabled := false
	fr := NewFunctionResult("wait").WaitForUser(&enabled, nil, false)
	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["wait_for_user"] != false {
		t.Errorf("wait_for_user = %v, want false", actions[0]["wait_for_user"])
	}
}

func TestWaitForUserDefault(t *testing.T) {
	fr := NewFunctionResult("wait").WaitForUser(nil, nil, false)
	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["wait_for_user"] != true {
		t.Errorf("wait_for_user = %v, want true", actions[0]["wait_for_user"])
	}
}

func TestStop(t *testing.T) {
	fr := NewFunctionResult("stopping").Stop()
	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["stop"] != true {
		t.Errorf("stop = %v, want true", actions[0]["stop"])
	}
}

// --- State & Data Management ---

func TestUpdateGlobalData(t *testing.T) {
	fr := NewFunctionResult("updated").
		UpdateGlobalData(map[string]any{"key": "value", "count": 42})

	actions := fr.ToMap()["action"].([]map[string]any)
	data := actions[0]["set_global_data"].(map[string]any)
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

	actions := fr.ToMap()["action"].([]map[string]any)
	keys := actions[0]["unset_global_data"].([]string)
	if len(keys) != 2 || keys[0] != "key1" || keys[1] != "key2" {
		t.Errorf("keys = %v, want [key1, key2]", keys)
	}
}

func TestSetMetadata(t *testing.T) {
	fr := NewFunctionResult("metadata").
		SetMetadata(map[string]any{"session": "abc123"})

	actions := fr.ToMap()["action"].([]map[string]any)
	data := actions[0]["set_meta_data"].(map[string]any)
	if data["session"] != "abc123" {
		t.Errorf("session = %v, want %q", data["session"], "abc123")
	}
}

func TestRemoveMetadata(t *testing.T) {
	fr := NewFunctionResult("remove").
		RemoveMetadata([]string{"old_key"})

	actions := fr.ToMap()["action"].([]map[string]any)
	keys := actions[0]["unset_meta_data"].([]string)
	if len(keys) != 1 || keys[0] != "old_key" {
		t.Errorf("keys = %v, want [old_key]", keys)
	}
}

func TestSwmlUserEvent(t *testing.T) {
	fr := NewFunctionResult("event sent").
		SwmlUserEvent(map[string]any{"type": "cards_dealt", "score": 21})

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	userEvent := verb["user_event"].(map[string]any)
	event := userEvent["event"].(map[string]any)

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

	actions := fr.ToMap()["action"].([]map[string]any)
	cs := actions[0]["context_switch"].(map[string]any)
	if cs["step"] != "betting" {
		t.Errorf("step = %v, want %q", cs["step"], "betting")
	}
}

func TestSwmlChangeContext(t *testing.T) {
	fr := NewFunctionResult("changing context").
		SwmlChangeContext("support")

	actions := fr.ToMap()["action"].([]map[string]any)
	cs := actions[0]["context_switch"].(map[string]any)
	if cs["context"] != "support" {
		t.Errorf("context = %v, want %q", cs["context"], "support")
	}
}

func TestSwitchContextSimple(t *testing.T) {
	fr := NewFunctionResult("switch").
		SwitchContext("You are a helper.", "", false, false, false)

	actions := fr.ToMap()["action"].([]map[string]any)
	cs := actions[0]["context_switch"]
	if cs != "You are a helper." {
		t.Errorf("context_switch = %v, want simple string", cs)
	}
}

func TestSwitchContextAdvanced(t *testing.T) {
	fr := NewFunctionResult("switch").
		SwitchContext("new prompt", "user msg", true, false, false)

	actions := fr.ToMap()["action"].([]map[string]any)
	cs := actions[0]["context_switch"].(map[string]any)
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

	actions := fr.ToMap()["action"].([]map[string]any)
	cs := actions[0]["context_switch"].(map[string]any)
	if cs["isolated"] != true {
		t.Error("isolated should be true")
	}
}

func TestReplaceInHistoryString(t *testing.T) {
	fr := NewFunctionResult("replace").
		ReplaceInHistory("summary text")

	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["replace_in_history"] != "summary text" {
		t.Errorf("replace_in_history = %v, want %q", actions[0]["replace_in_history"], "summary text")
	}
}

func TestReplaceInHistoryBool(t *testing.T) {
	fr := NewFunctionResult("replace").
		ReplaceInHistory(true)

	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["replace_in_history"] != true {
		t.Errorf("replace_in_history = %v, want true", actions[0]["replace_in_history"])
	}
}

// --- Media Control ---

func TestSay(t *testing.T) {
	fr := NewFunctionResult("speaking").Say("Hello there!")

	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["say"] != "Hello there!" {
		t.Errorf("say = %v, want %q", actions[0]["say"], "Hello there!")
	}
}

func TestPlayBackgroundFile(t *testing.T) {
	fr := NewFunctionResult("playing").
		PlayBackgroundFile("music.mp3", false)

	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["playback_bg"] != "music.mp3" {
		t.Errorf("playback_bg = %v, want %q", actions[0]["playback_bg"], "music.mp3")
	}
}

func TestPlayBackgroundFileWait(t *testing.T) {
	fr := NewFunctionResult("playing").
		PlayBackgroundFile("music.mp3", true)

	actions := fr.ToMap()["action"].([]map[string]any)
	bg := actions[0]["playback_bg"].(map[string]any)
	if bg["file"] != "music.mp3" {
		t.Errorf("file = %v, want %q", bg["file"], "music.mp3")
	}
	if bg["wait"] != true {
		t.Error("wait should be true")
	}
}

func TestStopBackgroundFile(t *testing.T) {
	fr := NewFunctionResult("stopping").StopBackgroundFile()

	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["stop_playback_bg"] != true {
		t.Errorf("stop_playback_bg = %v, want true", actions[0]["stop_playback_bg"])
	}
}

func TestRecordCall(t *testing.T) {
	fr := NewFunctionResult("recording").
		RecordCall("rec-123", true, "wav", "both")

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	params := verb["record_call"].(map[string]any)

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
}

func TestRecordCallNoControlID(t *testing.T) {
	fr := NewFunctionResult("recording").
		RecordCall("", false, "mp3", "speak")

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	params := verb["record_call"].(map[string]any)

	if _, ok := params["control_id"]; ok {
		t.Error("control_id should not be present when empty")
	}
}

func TestStopRecordCall(t *testing.T) {
	fr := NewFunctionResult("stop recording").
		StopRecordCall("rec-123")

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	params := verb["stop_record_call"].(map[string]any)

	if params["control_id"] != "rec-123" {
		t.Errorf("control_id = %v, want %q", params["control_id"], "rec-123")
	}
}

// --- Speech & AI Config ---

func TestAddDynamicHints(t *testing.T) {
	hints := []any{"Cabby", map[string]any{"pattern": "cab bee", "replace": "Cabby"}}
	fr := NewFunctionResult("hints").AddDynamicHints(hints)

	actions := fr.ToMap()["action"].([]map[string]any)
	h := actions[0]["add_dynamic_hints"].([]any)
	if len(h) != 2 {
		t.Fatalf("hints length = %d, want 2", len(h))
	}
	if h[0] != "Cabby" {
		t.Errorf("hint[0] = %v, want %q", h[0], "Cabby")
	}
}

func TestClearDynamicHints(t *testing.T) {
	fr := NewFunctionResult("clear").ClearDynamicHints()

	actions := fr.ToMap()["action"].([]map[string]any)
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

	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["end_of_speech_timeout"] != 500 {
		t.Errorf("end_of_speech_timeout = %v, want 500", actions[0]["end_of_speech_timeout"])
	}
}

func TestSetSpeechEventTimeout(t *testing.T) {
	fr := NewFunctionResult("timeout").SetSpeechEventTimeout(1000)

	actions := fr.ToMap()["action"].([]map[string]any)
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

	actions := fr.ToMap()["action"].([]map[string]any)
	tf := actions[0]["toggle_functions"].([]map[string]any)
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

	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["functions_on_speaker_timeout"] != true {
		t.Errorf("functions_on_speaker_timeout = %v, want true", actions[0]["functions_on_speaker_timeout"])
	}
}

func TestEnableExtensiveData(t *testing.T) {
	fr := NewFunctionResult("data").EnableExtensiveData(true)

	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["extensive_data"] != true {
		t.Errorf("extensive_data = %v, want true", actions[0]["extensive_data"])
	}
}

func TestUpdateSettings(t *testing.T) {
	fr := NewFunctionResult("settings").
		UpdateSettings(map[string]any{"temperature": 0.5, "top_p": 0.9})

	actions := fr.ToMap()["action"].([]map[string]any)
	s := actions[0]["settings"].(map[string]any)
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

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
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

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	if swml["transfer"] != "true" {
		t.Errorf("transfer = %v, want %q", swml["transfer"], "true")
	}
}

func TestExecuteSwmlString(t *testing.T) {
	fr := NewFunctionResult("exec").ExecuteSwml(`{"version":"1.0.0"}`, false)

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	if swml["raw_swml"] != `{"version":"1.0.0"}` {
		t.Errorf("raw_swml = %v", swml["raw_swml"])
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
		JoinConference("my-conf", true, "onEnter", "https://example.com/hold.mp3")

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	params := verb["join_conference"].(map[string]any)

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
	fr := NewFunctionResult("joining").
		JoinConference("simple-conf", false, "true", "")

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	params := verb["join_conference"].(map[string]any)

	if _, ok := params["muted"]; ok {
		t.Error("muted should not be present when false")
	}
	if _, ok := params["beep"]; ok {
		t.Error("beep should not be present when 'true' (default)")
	}
	if _, ok := params["wait_url"]; ok {
		t.Error("wait_url should not be present when empty")
	}
}

func TestJoinRoom(t *testing.T) {
	fr := NewFunctionResult("joining room").JoinRoom("my-room")

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	params := verb["join_room"].(map[string]any)

	if params["name"] != "my-room" {
		t.Errorf("name = %v, want %q", params["name"], "my-room")
	}
}

func TestSipRefer(t *testing.T) {
	fr := NewFunctionResult("referring").SipRefer("sip:agent@example.com")

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	params := verb["sip_refer"].(map[string]any)

	if params["to_uri"] != "sip:agent@example.com" {
		t.Errorf("to_uri = %v", params["to_uri"])
	}
}

func TestTap(t *testing.T) {
	fr := NewFunctionResult("tapping").
		Tap("rtp://192.168.1.1:5000", "tap-1", "speak", "PCMA")

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	params := verb["tap"].(map[string]any)

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
		Tap("ws://example.com", "", "both", "PCMU")

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	params := verb["tap"].(map[string]any)

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

func TestStopTap(t *testing.T) {
	fr := NewFunctionResult("stop tap").StopTap("tap-1")

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	params := verb["stop_tap"].(map[string]any)

	if params["control_id"] != "tap-1" {
		t.Errorf("control_id = %v", params["control_id"])
	}
}

func TestStopTapNoControlID(t *testing.T) {
	fr := NewFunctionResult("stop tap").StopTap("")

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	params := verb["stop_tap"].(map[string]any)

	if len(params) != 0 {
		t.Errorf("stop_tap should be empty map, got %v", params)
	}
}

func TestSendSms(t *testing.T) {
	fr := NewFunctionResult("sms sent").
		SendSms("+15551234567", "+15559876543", "Hello!", nil, nil)

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	params := verb["send_sms"].(map[string]any)

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
}

func TestSendSmsWithMediaAndTags(t *testing.T) {
	fr := NewFunctionResult("sms sent").
		SendSms("+15551234567", "+15559876543", "",
			[]string{"https://example.com/image.jpg"},
			[]string{"support", "urgent"})

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	params := verb["send_sms"].(map[string]any)

	if _, ok := params["body"]; ok {
		t.Error("body should not be present when empty")
	}
	media := params["media"].([]string)
	if len(media) != 1 || media[0] != "https://example.com/image.jpg" {
		t.Errorf("media = %v", media)
	}
	tags := params["tags"].([]string)
	if len(tags) != 2 {
		t.Errorf("tags = %v", tags)
	}
}

func TestPay(t *testing.T) {
	fr := NewFunctionResult("processing payment").
		Pay("https://example.com/pay", "dtmf", "", 5, 1)

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)

	// Without actionURL, no "set" verb should be prepended
	if len(main) != 1 {
		t.Fatalf("main verbs = %d, want 1", len(main))
	}
	verb := main[0].(map[string]any)
	params := verb["pay"].(map[string]any)
	if params["payment_connector_url"] != "https://example.com/pay" {
		t.Errorf("payment_connector_url = %v", params["payment_connector_url"])
	}
}

func TestPayWithActionURL(t *testing.T) {
	fr := NewFunctionResult("processing payment").
		Pay("https://example.com/pay", "dtmf", "Payment complete", 5, 1)

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)

	if len(main) != 2 {
		t.Fatalf("main verbs = %d, want 2 (set + pay)", len(main))
	}

	setVerb := main[0].(map[string]any)
	setData := setVerb["set"].(map[string]any)
	if setData["ai_response"] != "Payment complete" {
		t.Errorf("ai_response = %v", setData["ai_response"])
	}
}

// --- RPC Actions ---

func TestExecuteRpc(t *testing.T) {
	fr := NewFunctionResult("rpc").
		ExecuteRpc("custom.method", map[string]any{"key": "value"})

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	rpc := verb["execute_rpc"].(map[string]any)

	if rpc["method"] != "custom.method" {
		t.Errorf("method = %v, want %q", rpc["method"], "custom.method")
	}
	if rpc["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want %q", rpc["jsonrpc"], "2.0")
	}
	params := rpc["params"].(map[string]any)
	if params["key"] != "value" {
		t.Errorf("params.key = %v, want %q", params["key"], "value")
	}
}

func TestExecuteRpcNoParams(t *testing.T) {
	fr := NewFunctionResult("rpc").
		ExecuteRpc("simple.method", nil)

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	rpc := verb["execute_rpc"].(map[string]any)

	if _, ok := rpc["params"]; ok {
		t.Error("params should not be present when nil")
	}
}

func TestRpcDial(t *testing.T) {
	timeout := 30
	fr := NewFunctionResult("dialing").
		RpcDial("+15551234567", "+15559876543", "https://example.com/swml", &timeout, "us-east")

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	rpc := verb["execute_rpc"].(map[string]any)

	if rpc["method"] != "calling.dial" {
		t.Errorf("method = %v, want %q", rpc["method"], "calling.dial")
	}

	params := rpc["params"].(map[string]any)
	if params["dest_swml"] != "https://example.com/swml" {
		t.Errorf("dest_swml = %v", params["dest_swml"])
	}
	if params["timeout"] != 30 {
		t.Errorf("timeout = %v, want 30", params["timeout"])
	}
	if params["region"] != "us-east" {
		t.Errorf("region = %v, want %q", params["region"], "us-east")
	}

	devices := params["devices"].(map[string]any)
	if devices["type"] != "phone" {
		t.Errorf("type = %v, want %q", devices["type"], "phone")
	}
	deviceParams := devices["params"].(map[string]any)
	if deviceParams["to_number"] != "+15551234567" {
		t.Errorf("to_number = %v", deviceParams["to_number"])
	}
}

func TestRpcDialNoOptional(t *testing.T) {
	fr := NewFunctionResult("dialing").
		RpcDial("+15551234567", "+15559876543", "https://example.com/swml", nil, "")

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	rpc := verb["execute_rpc"].(map[string]any)
	params := rpc["params"].(map[string]any)

	if _, ok := params["timeout"]; ok {
		t.Error("timeout should not be present when nil")
	}
	if _, ok := params["region"]; ok {
		t.Error("region should not be present when empty")
	}
}

func TestRpcAiMessage(t *testing.T) {
	fr := NewFunctionResult("messaging").
		RpcAiMessage("call-abc-123", "Please take a message")

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	rpc := verb["execute_rpc"].(map[string]any)

	if rpc["method"] != "calling.ai_message" {
		t.Errorf("method = %v, want %q", rpc["method"], "calling.ai_message")
	}
	if rpc["call_id"] != "call-abc-123" {
		t.Errorf("call_id = %v, want %q", rpc["call_id"], "call-abc-123")
	}

	params := rpc["params"].(map[string]any)
	if params["role"] != "system" {
		t.Errorf("role = %v, want %q", params["role"], "system")
	}
	if params["message_text"] != "Please take a message" {
		t.Errorf("message_text = %v", params["message_text"])
	}
}

func TestRpcAiUnhold(t *testing.T) {
	fr := NewFunctionResult("unholding").
		RpcAiUnhold("call-abc-123")

	actions := fr.ToMap()["action"].([]map[string]any)
	swml := actions[0]["SWML"].(map[string]any)
	sections := swml["sections"].(map[string]any)
	main := sections["main"].([]any)
	verb := main[0].(map[string]any)
	rpc := verb["execute_rpc"].(map[string]any)

	if rpc["method"] != "calling.ai_unhold" {
		t.Errorf("method = %v, want %q", rpc["method"], "calling.ai_unhold")
	}
	if rpc["call_id"] != "call-abc-123" {
		t.Errorf("call_id = %v, want %q", rpc["call_id"], "call-abc-123")
	}
}

func TestSimulateUserInput(t *testing.T) {
	fr := NewFunctionResult("simulating").
		SimulateUserInput("I need help")

	actions := fr.ToMap()["action"].([]map[string]any)
	if actions[0]["simulate_user_input"] != "I need help" {
		t.Errorf("simulate_user_input = %v, want %q", actions[0]["simulate_user_input"], "I need help")
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
	actions := m["action"].([]map[string]any)
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
	prompt := CreatePaymentPrompt("credit-card-payment", actions)

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
	prompt := CreatePaymentPrompt("refund", []map[string]string{})

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
