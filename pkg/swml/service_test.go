package swml

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewService(t *testing.T) {
	svc := NewService(WithName("test-svc"), WithPort(8080))
	if svc.Name != "test-svc" {
		t.Errorf("Name = %q, want %q", svc.Name, "test-svc")
	}
	if svc.Port != 8080 {
		t.Errorf("Port = %d, want 8080", svc.Port)
	}
}

func TestServiceDefaultAuth(t *testing.T) {
	svc := NewService(WithName("test"))
	user, pass := svc.GetBasicAuthCredentials()
	if user == "" {
		t.Error("default user should not be empty")
	}
	if pass == "" {
		t.Error("default password should not be empty")
	}
}

func TestServiceExplicitAuth(t *testing.T) {
	svc := NewService(WithBasicAuth("myuser", "mypass"))
	user, pass := svc.GetBasicAuthCredentials()
	if user != "myuser" {
		t.Errorf("user = %q, want %q", user, "myuser")
	}
	if pass != "mypass" {
		t.Errorf("pass = %q, want %q", pass, "mypass")
	}
}

func TestServiceVerbMethods(t *testing.T) {
	svc := NewService(WithName("test"))

	// Test Answer verb with typed params
	maxDur := 300
	err := svc.Answer(&maxDur, nil)
	if err != nil {
		t.Fatalf("Answer failed: %v", err)
	}

	// Test Play verb with typed params
	playURL := "https://example.com/audio.mp3"
	err = svc.Play(PlayOptions{URL: &playURL})
	if err != nil {
		t.Fatalf("Play failed: %v", err)
	}

	// Test Sleep verb (special - takes integer)
	err = svc.Sleep(1000)
	if err != nil {
		t.Fatalf("Sleep failed: %v", err)
	}

	// Verify document has all verbs
	verbs := svc.GetDocument().GetVerbs("main")
	if len(verbs) != 3 {
		t.Fatalf("expected 3 verbs, got %d", len(verbs))
	}
}

func TestServiceAllVerbMethods(t *testing.T) {
	svc := NewService(WithName("test"))

	// Test every verb method exists and doesn't error
	tests := []struct {
		name string
		fn   func() error
	}{
		{"Answer", func() error { return svc.Answer(nil, nil) }},
		{"Hangup", func() error { return svc.Hangup(nil) }},
		{"Play", func() error { u := "say:hello"; return svc.Play(PlayOptions{URL: &u}) }},
		{"Record", func() error { return svc.Record(map[string]any{}) }},
		{"RecordCall", func() error { return svc.RecordCall(map[string]any{}) }},
		{"StopRecordCall", func() error { return svc.StopRecordCall(map[string]any{}) }},
		{"Sleep", func() error { return svc.Sleep(100) }},
		{"Connect", func() error {
			return svc.Connect(map[string]any{"to": "sip:x@y"})
		}},
		{"SendDigits", func() error { return svc.SendDigits(map[string]any{"digits": "123"}) }},
		{"SendSMS", func() error {
			return svc.SendSMS(map[string]any{"to_number": "+15551112222", "from_number": "+15553334444", "body": "hi"})
		}},
		{"SendFax", func() error { return svc.SendFax(map[string]any{"document": "http://x/f.pdf"}) }},
		{"ReceiveFax", func() error { return svc.ReceiveFax(map[string]any{}) }},
		{"SIPRefer", func() error { return svc.SIPRefer(map[string]any{"to_uri": "sip:x@y"}) }},
		{"AI", func() error {
			// After the verb-handler registry (PR #86) landed, AIVerbHandler
			// rejects a blank prompt. Provide the minimum valid shape.
			pt := "hello"
			return svc.AI(AIOptions{PromptText: &pt})
		}},
		{"AmazonBedrock", func() error {
			return svc.AmazonBedrock(map[string]any{"prompt": map[string]any{"text": "hi"}})
		}},
		// cond takes an ARRAY of condition objects per the schema, not a map;
		// drive ExecuteVerb with the correctly-typed value.
		{"Cond", func() error {
			return svc.ExecuteVerb("cond", []any{map[string]any{"when": "x", "then": []any{}}})
		}},
		{"Switch", func() error {
			return svc.Switch(map[string]any{"variable": "x", "case": map[string]any{}})
		}},
		{"Execute", func() error { return svc.Execute(map[string]any{"dest": "main"}) }},
		{"Return", func() error { return svc.Return(map[string]any{}) }},
		{"Goto", func() error { return svc.Goto(map[string]any{"label": "top"}) }},
		// label takes a STRING per the schema, not a map.
		{"Label", func() error { return svc.ExecuteVerb("label", "greeting") }},
		{"Set", func() error { return svc.Set(map[string]any{}) }},
		// unset takes a string or an array of strings per the schema, not a map.
		{"Unset", func() error { return svc.ExecuteVerb("unset", []any{"temp_data"}) }},
		{"Transfer", func() error { return svc.Transfer(map[string]any{"dest": "main"}) }},
		{"Tap", func() error { return svc.Tap(map[string]any{"uri": "rtp://x"}) }},
		{"StopTap", func() error { return svc.StopTap(map[string]any{}) }},
		{"Denoise", func() error { return svc.Denoise(map[string]any{}) }},
		{"StopDenoise", func() error { return svc.StopDenoise(map[string]any{}) }},
		{"JoinRoom", func() error { return svc.JoinRoom(map[string]any{"name": "room1"}) }},
		{"JoinConference", func() error { return svc.JoinConference(map[string]any{"name": "conf1"}) }},
		{"Prompt", func() error { return svc.Prompt(map[string]any{"play": "say:hi"}) }},
		{"EnterQueue", func() error {
			return svc.EnterQueue(map[string]any{"queue_name": "q", "transfer_after_bridge": "main"})
		}},
		{"Request", func() error {
			return svc.Request(map[string]any{"url": "http://x", "method": "GET"})
		}},
		{"Pay", func() error { return svc.Pay(map[string]any{"payment_connector_url": "http://x"}) }},
		{"DetectMachine", func() error { return svc.DetectMachine(map[string]any{}) }},
		// live_transcribe / live_translate take a typed action; "stop" (a string
		// const) is the minimal valid action per the schema. The wrappers accept
		// a map, so drive ExecuteVerb with the valid {action:"stop"} shape.
		{"LiveTranscribe", func() error {
			return svc.LiveTranscribe(map[string]any{"action": "stop"})
		}},
		{"LiveTranslate", func() error {
			return svc.LiveTranslate(map[string]any{"action": "stop"})
		}},
		{"UserEvent", func() error { return svc.UserEvent(map[string]any{"event": map[string]any{}}) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err != nil {
				t.Errorf("%s failed: %v", tt.name, err)
			}
		})
	}
}

func TestServiceExecuteVerbInvalid(t *testing.T) {
	svc := NewService(WithName("test"))
	err := svc.ExecuteVerb("totally_fake_verb", map[string]any{})
	if err == nil {
		t.Error("expected error for invalid verb")
	}
}

func TestServiceRender(t *testing.T) {
	svc := NewService(WithName("test"))
	maxDur2 := 300
	if err := svc.Answer(&maxDur2, nil); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	playURL2 := "https://example.com/audio.mp3"
	if err := svc.Play(PlayOptions{URL: &playURL2}); err != nil {
		t.Fatalf("Play: %v", err)
	}
	if err := svc.Hangup(nil); err != nil {
		t.Fatalf("Hangup: %v", err)
	}

	rendered, err := svc.Render()
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(rendered), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	sections, ok := doc["sections"].(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", doc["sections"])
	}
	main, ok := sections["main"].([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", sections["main"])
	}
	if len(main) != 3 {
		t.Errorf("expected 3 verbs, got %d", len(main))
	}
}

func TestServiceGetFullURL(t *testing.T) {
	svc := NewService(WithName("test"), WithPort(3000), WithRoute("/agent"))
	url := svc.GetFullURL(false)
	if url != "http://localhost:3000/agent" {
		t.Errorf("URL = %q, want %q", url, "http://localhost:3000/agent")
	}

	urlWithAuth := svc.GetFullURL(true)
	if urlWithAuth == url {
		t.Error("URL with auth should differ from URL without auth")
	}
}

func TestServiceOnRequest(t *testing.T) {
	svc := NewService(WithName("test"))
	if err := svc.Answer(nil, nil); err != nil {
		t.Fatalf("Answer: %v", err)
	}

	result := svc.OnRequest(nil, "")
	if result["version"] != "1.0.0" {
		t.Error("OnRequest should return SWML document")
	}
}

func TestServiceRoutingCallback(t *testing.T) {
	svc := NewService(WithName("test"), WithBasicAuth("u", "p"))
	if err := svc.Answer(nil, nil); err != nil {
		t.Fatalf("Answer: %v", err)
	}

	// A routing callback returns a route string to redirect (307), or nil to
	// continue to the default document, per (body, headers) -> *string.
	svc.RegisterRoutingCallback("/custom", func(body map[string]any, headers map[string]any) *string {
		if dept, _ := body["department"].(string); dept == "sales" {
			route := "/sales"
			return &route
		}
		return nil
	})

	authHdr := map[string]string{"Authorization": "Basic dTpw"} // base64("u:p")

	// Non-matching path: HandleRequest serves the default document (200).
	status, _, bodyStr := svc.HandleRequest("POST", "/other", authHdr, map[string]any{"x": 1})
	if status != 200 {
		t.Errorf("non-matching path: want 200, got %d", status)
	}
	if !strings.Contains(bodyStr, "\"version\":\"1.0.0\"") {
		t.Errorf("non-matching path should return default document, got %s", bodyStr)
	}

	// Matching path with a route-triggering body: 307 redirect to the route.
	status, hdrs, _ := svc.HandleRequest("POST", "/custom", authHdr, map[string]any{"department": "sales"})
	if status != 307 {
		t.Errorf("matching path with route body: want 307, got %d", status)
	}
	if hdrs["Location"] != "/sales" {
		t.Errorf("want Location=/sales, got %q", hdrs["Location"])
	}

	// Matching path but callback returns nil: fall through to default document.
	status, _, _ = svc.HandleRequest("POST", "/custom", authHdr, map[string]any{"department": "other"})
	if status != 200 {
		t.Errorf("matching path, nil route: want 200, got %d", status)
	}
}

func TestExtractSIPUsername(t *testing.T) {
	tests := []struct {
		name     string
		body     map[string]any
		expected string
	}{
		{
			"sip URI",
			map[string]any{"call": map[string]any{"to": "sip:alice@example.com"}},
			"alice",
		},
		{
			// A non-sip/non-tel 'to' is returned WHOLE (Python parity: only the
			// "sip:" branch splits on "@"; plain fields are passed through).
			"no sip prefix",
			map[string]any{"call": map[string]any{"to": "bob@example.com"}},
			"bob@example.com",
		},
		{
			"tel URI strips tel: prefix",
			map[string]any{"call": map[string]any{"to": "tel:+15551234567"}},
			"+15551234567",
		},
		{
			"plain username returned whole",
			map[string]any{"call": map[string]any{"to": "support"}},
			"support",
		},
		{
			"no call data",
			map[string]any{},
			"",
		},
		{
			"no to field",
			map[string]any{"call": map[string]any{"from": "+15551234567"}},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractSIPUsername(tt.body)
			if got != tt.expected {
				t.Errorf("ExtractSIPUsername = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFilterNilValues(t *testing.T) {
	input := map[string]any{
		"a": "value",
		"b": nil,
		"c": 42,
		"d": nil,
	}
	result := filterNilValues(input)
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
	if result["a"] != "value" {
		t.Error("should keep non-nil string")
	}
	if result["c"] != 42 {
		t.Error("should keep non-nil int")
	}
}

func TestFilterNilValuesNilInput(t *testing.T) {
	result := filterNilValues(nil)
	if result == nil {
		t.Error("should return empty map, not nil")
	}
	if len(result) != 0 {
		t.Error("should return empty map")
	}
}
