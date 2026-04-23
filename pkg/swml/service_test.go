package swml

import (
	"encoding/json"
	"net/http"
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
	err = svc.Play(&playURL, nil, nil, nil, nil, nil, nil)
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
		{"Play", func() error { u := "say:hello"; return svc.Play(&u, nil, nil, nil, nil, nil, nil) }},
		{"Record", func() error { return svc.Record(map[string]any{}) }},
		{"RecordCall", func() error { return svc.RecordCall(map[string]any{}) }},
		{"StopRecordCall", func() error { return svc.StopRecordCall(map[string]any{}) }},
		{"Sleep", func() error { return svc.Sleep(100) }},
		{"Connect", func() error { return svc.Connect(map[string]any{}) }},
		{"SendDigits", func() error { return svc.SendDigits(map[string]any{}) }},
		{"SendSMS", func() error { return svc.SendSMS(map[string]any{}) }},
		{"SendFax", func() error { return svc.SendFax(map[string]any{}) }},
		{"ReceiveFax", func() error { return svc.ReceiveFax(map[string]any{}) }},
		{"SIPRefer", func() error { return svc.SIPRefer(map[string]any{}) }},
		{"AI", func() error { return svc.AI(nil, nil, nil, nil, nil, nil) }},
		{"AmazonBedrock", func() error { return svc.AmazonBedrock(map[string]any{}) }},
		{"Cond", func() error { return svc.Cond(map[string]any{}) }},
		{"Switch", func() error { return svc.Switch(map[string]any{}) }},
		{"Execute", func() error { return svc.Execute(map[string]any{}) }},
		{"Return", func() error { return svc.Return(map[string]any{}) }},
		{"Goto", func() error { return svc.Goto(map[string]any{}) }},
		{"Label", func() error { return svc.Label(map[string]any{}) }},
		{"Set", func() error { return svc.Set(map[string]any{}) }},
		{"Unset", func() error { return svc.Unset(map[string]any{}) }},
		{"Transfer", func() error { return svc.Transfer(map[string]any{}) }},
		{"Tap", func() error { return svc.Tap(map[string]any{}) }},
		{"StopTap", func() error { return svc.StopTap(map[string]any{}) }},
		{"Denoise", func() error { return svc.Denoise(map[string]any{}) }},
		{"StopDenoise", func() error { return svc.StopDenoise(map[string]any{}) }},
		{"JoinRoom", func() error { return svc.JoinRoom(map[string]any{}) }},
		{"JoinConference", func() error { return svc.JoinConference(map[string]any{}) }},
		{"Prompt", func() error { return svc.Prompt(map[string]any{}) }},
		{"EnterQueue", func() error { return svc.EnterQueue(map[string]any{}) }},
		{"Request", func() error { return svc.Request(map[string]any{}) }},
		{"Pay", func() error { return svc.Pay(map[string]any{}) }},
		{"DetectMachine", func() error { return svc.DetectMachine(map[string]any{}) }},
		{"LiveTranscribe", func() error { return svc.LiveTranscribe(map[string]any{}) }},
		{"LiveTranslate", func() error { return svc.LiveTranslate(map[string]any{}) }},
		{"UserEvent", func() error { return svc.UserEvent(map[string]any{}) }},
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
	svc.Answer(&maxDur2, nil)
	playURL2 := "https://example.com/audio.mp3"
	svc.Play(&playURL2, nil, nil, nil, nil, nil, nil)
	svc.Hangup(nil)

	rendered, err := svc.Render()
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(rendered), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)
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
	svc.Answer(nil, nil)

	result := svc.OnRequest(nil, "")
	if result["version"] != "1.0.0" {
		t.Error("OnRequest should return SWML document")
	}
}

func TestServiceRoutingCallback(t *testing.T) {
	svc := NewService(WithName("test"))
	svc.Answer(nil, nil)

	customDoc := map[string]any{"version": "custom", "sections": map[string]any{"main": []any{}}}
	svc.RegisterRoutingCallback("/custom", func(r *http.Request, body map[string]any) map[string]any {
		return customDoc
	})

	// Default path returns normal document
	result := svc.OnRequest(nil, "/other")
	if result["version"] != "1.0.0" {
		t.Error("non-matching path should return default document")
	}

	// Matching path returns custom document
	result = svc.OnRequest(nil, "/custom")
	if result["version"] != "custom" {
		t.Errorf("matching path should return custom document, got version=%v", result["version"])
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
			"no sip prefix",
			map[string]any{"call": map[string]any{"to": "bob@example.com"}},
			"bob",
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
