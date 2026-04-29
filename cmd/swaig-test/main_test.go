package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------------------------------------------------------------------------
// URL auth extraction
// ---------------------------------------------------------------------------

func TestParseAuthURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantURL  string
		wantUser string
		wantPass string
		wantErr  bool
	}{
		{
			name:     "url with auth",
			input:    "http://admin:secret@localhost:3000/",
			wantURL:  "http://localhost:3000/",
			wantUser: "admin",
			wantPass: "secret",
		},
		{
			name:     "url without auth",
			input:    "http://localhost:3000/",
			wantURL:  "http://localhost:3000/",
			wantUser: "",
			wantPass: "",
		},
		{
			name:     "url with auth and path",
			input:    "http://user:pass@localhost:3000/my-agent",
			wantURL:  "http://localhost:3000/my-agent",
			wantUser: "user",
			wantPass: "pass",
		},
		{
			name:     "url with user only",
			input:    "http://user@localhost:3000/",
			wantURL:  "http://localhost:3000/",
			wantUser: "user",
			wantPass: "",
		},
		{
			name:     "https url with auth",
			input:    "https://admin:secret@example.com/agent",
			wantURL:  "https://example.com/agent",
			wantUser: "admin",
			wantPass: "secret",
		},
		{
			name:     "url with special chars in password",
			input:    "http://user:p%40ss@localhost:3000/",
			wantURL:  "http://localhost:3000/",
			wantUser: "user",
			wantPass: "p@ss",
		},
		{
			name:    "invalid url",
			input:   "://bad",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotUser, gotPass, err := parseAuthURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotURL != tt.wantURL {
				t.Errorf("URL = %q, want %q", gotURL, tt.wantURL)
			}
			if gotUser != tt.wantUser {
				t.Errorf("user = %q, want %q", gotUser, tt.wantUser)
			}
			if gotPass != tt.wantPass {
				t.Errorf("pass = %q, want %q", gotPass, tt.wantPass)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Parameter parsing
// ---------------------------------------------------------------------------

func TestParseParams(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name:  "single string param",
			input: []string{"location=London"},
			want:  map[string]interface{}{"location": "London"},
		},
		{
			name:  "multiple params",
			input: []string{"city=Paris", "units=metric"},
			want:  map[string]interface{}{"city": "Paris", "units": "metric"},
		},
		{
			name:  "numeric param",
			input: []string{"count=42"},
			want:  map[string]interface{}{"count": float64(42)},
		},
		{
			name:  "boolean param",
			input: []string{"verbose=true"},
			want:  map[string]interface{}{"verbose": true},
		},
		{
			name:  "value with equals sign",
			input: []string{"query=a=b"},
			want:  map[string]interface{}{"query": "a=b"},
		},
		{
			name:  "empty value",
			input: []string{"key="},
			want:  map[string]interface{}{"key": ""},
		},
		{
			name:  "no params",
			input: []string{},
			want:  map[string]interface{}{},
		},
		{
			name:    "missing equals sign",
			input:   []string{"badparam"},
			wantErr: true,
		},
		{
			name:    "empty key",
			input:   []string{"=value"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseParams(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d params, want %d", len(got), len(tt.want))
			}
			for k, wantV := range tt.want {
				gotV, ok := got[k]
				if !ok {
					t.Errorf("missing key %q", k)
					continue
				}
				if fmt.Sprintf("%v", gotV) != fmt.Sprintf("%v", wantV) {
					t.Errorf("param[%q] = %v (%T), want %v (%T)", k, gotV, gotV, wantV, wantV)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SWML function extraction
// ---------------------------------------------------------------------------

func TestExtractFunctions(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNames []string
		wantErr   bool
	}{
		{
			name: "standard SWML with functions",
			input: `{
				"version": "1.0.0",
				"sections": {
					"main": [
						{"answer": {"max_duration": 14400}},
						{"ai": {
							"prompt": {"text": "You are a helpful agent."},
							"SWAIG": {
								"functions": [
									{"function": "get_weather", "purpose": "Get weather info"},
									{"function": "get_time", "purpose": "Get current time"}
								]
							}
						}}
					]
				}
			}`,
			wantNames: []string{"get_weather", "get_time"},
		},
		{
			name: "no SWAIG section",
			input: `{
				"version": "1.0.0",
				"sections": {
					"main": [
						{"answer": {"max_duration": 14400}},
						{"ai": {"prompt": {"text": "Hello"}}}
					]
				}
			}`,
			wantNames: nil,
		},
		{
			name: "empty functions",
			input: `{
				"version": "1.0.0",
				"sections": {
					"main": [
						{"ai": {
							"SWAIG": {"functions": []}
						}}
					]
				}
			}`,
			wantNames: nil,
		},
		{
			name:    "invalid JSON",
			input:   "not json",
			wantErr: true,
		},
		{
			name:    "missing sections",
			input:   `{"version": "1.0.0"}`,
			wantErr: true,
		},
		{
			name: "function with parameters",
			input: `{
				"version": "1.0.0",
				"sections": {
					"main": [
						{"ai": {
							"SWAIG": {
								"functions": [
									{
										"function": "search",
										"purpose": "Search for things",
										"argument": {
											"type": "object",
											"properties": {
												"query": {"type": "string", "description": "Search query"}
											}
										}
									}
								]
							}
						}}
					]
				}
			}`,
			wantNames: []string{"search"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			functions, err := extractFunctions([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNames == nil {
				if len(functions) != 0 {
					t.Errorf("expected no functions, got %d", len(functions))
				}
				return
			}

			if len(functions) != len(tt.wantNames) {
				t.Fatalf("got %d functions, want %d", len(functions), len(tt.wantNames))
			}

			for i, want := range tt.wantNames {
				name, _ := functions[i]["function"].(string)
				if name != want {
					t.Errorf("function[%d] = %q, want %q", i, name, want)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Flag parsing
// ---------------------------------------------------------------------------

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantURL    string
		wantDump   bool
		wantList   bool
		wantExec   string
		wantRaw    bool
		wantVerbose bool
		wantParams int
	}{
		{
			name:     "dump-swml",
			args:     []string{"--url", "http://localhost:3000/", "--dump-swml"},
			wantURL:  "http://localhost:3000/",
			wantDump: true,
		},
		{
			name:     "list-tools",
			args:     []string{"--url", "http://user:pass@localhost:3000/", "--list-tools"},
			wantURL:  "http://user:pass@localhost:3000/",
			wantList: true,
		},
		{
			name:       "exec with params",
			args:       []string{"--url", "http://localhost:3000/", "--exec", "get_weather", "--param", "city=London", "--param", "units=metric"},
			wantURL:    "http://localhost:3000/",
			wantExec:   "get_weather",
			wantParams: 2,
		},
		{
			name:    "raw flag",
			args:    []string{"--url", "http://localhost:3000/", "--dump-swml", "--raw"},
			wantURL: "http://localhost:3000/",
			wantDump: true,
			wantRaw: true,
		},
		{
			name:        "verbose flag",
			args:        []string{"--url", "http://localhost:3000/", "--list-tools", "--verbose"},
			wantURL:     "http://localhost:3000/",
			wantList:    true,
			wantVerbose: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := parseFlags(tt.args)
			if cfg.url != tt.wantURL {
				t.Errorf("url = %q, want %q", cfg.url, tt.wantURL)
			}
			if cfg.dumpSWML != tt.wantDump {
				t.Errorf("dumpSWML = %v, want %v", cfg.dumpSWML, tt.wantDump)
			}
			if cfg.listTools != tt.wantList {
				t.Errorf("listTools = %v, want %v", cfg.listTools, tt.wantList)
			}
			if cfg.exec != tt.wantExec {
				t.Errorf("exec = %q, want %q", cfg.exec, tt.wantExec)
			}
			if cfg.raw != tt.wantRaw {
				t.Errorf("raw = %v, want %v", cfg.raw, tt.wantRaw)
			}
			if cfg.verbose != tt.wantVerbose {
				t.Errorf("verbose = %v, want %v", cfg.verbose, tt.wantVerbose)
			}
			if len(cfg.params) != tt.wantParams {
				t.Errorf("params count = %d, want %d", len(cfg.params), tt.wantParams)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// JSON formatting
// ---------------------------------------------------------------------------

func TestFormatJSON(t *testing.T) {
	input := []byte(`{"name":"test","value":42}`)

	t.Run("pretty", func(t *testing.T) {
		got, err := formatJSON(input, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !containsAll(got, "\"name\"", "\"test\"", "\"value\"", "42") {
			t.Errorf("pretty output missing expected content: %s", got)
		}
		// Should contain newlines when pretty-printed
		if len(got) <= len(string(input)) {
			t.Errorf("pretty output should be longer than compact")
		}
	})

	t.Run("raw", func(t *testing.T) {
		got, err := formatJSON(input, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != `{"name":"test","value":42}` {
			t.Errorf("raw output = %q, want compact JSON", got)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := formatJSON([]byte("not json"), false)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

// ---------------------------------------------------------------------------
// Integration: run() validation
// ---------------------------------------------------------------------------

func TestRunValidation(t *testing.T) {
	t.Run("missing url", func(t *testing.T) {
		cfg := config{dumpSWML: true}
		err := run(cfg)
		if err == nil || err.Error() != "--url is required" {
			t.Errorf("expected '--url is required' error, got: %v", err)
		}
	})

	t.Run("no mode selected", func(t *testing.T) {
		cfg := config{url: "http://localhost:3000/"}
		err := run(cfg)
		if err == nil {
			t.Error("expected error when no mode selected")
		}
	})
}

// ---------------------------------------------------------------------------
// Integration: dump-swml against httptest server
// ---------------------------------------------------------------------------

func TestDoDumpSWML(t *testing.T) {
	swmlDoc := map[string]interface{}{
		"version": "1.0.0",
		"sections": map[string]interface{}{
			"main": []interface{}{
				map[string]interface{}{
					"ai": map[string]interface{}{
						"prompt": map[string]interface{}{"text": "Hello"},
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(swmlDoc)
	}))
	defer srv.Close()

	cfg := config{raw: false}
	err := doDumpSWML(srv.URL, "", "", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Integration: list-tools against httptest server
// ---------------------------------------------------------------------------

func TestDoListTools(t *testing.T) {
	swmlDoc := map[string]interface{}{
		"version": "1.0.0",
		"sections": map[string]interface{}{
			"main": []interface{}{
				map[string]interface{}{
					"ai": map[string]interface{}{
						"SWAIG": map[string]interface{}{
							"functions": []interface{}{
								map[string]interface{}{
									"function": "get_weather",
									"purpose":  "Get weather info",
								},
							},
						},
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(swmlDoc)
	}))
	defer srv.Close()

	cfg := config{}
	err := doListTools(srv.URL, "", "", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Integration: exec against httptest server
// ---------------------------------------------------------------------------

func TestDoExec(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/swaig" {
			t.Errorf("expected path /swaig, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}

		funcName, _ := body["function"].(string)
		if funcName != "get_weather" {
			t.Errorf("expected function 'get_weather', got %q", funcName)
		}

		callID, _ := body["call_id"].(string)
		if callID != "test-call-id" {
			t.Errorf("expected call_id 'test-call-id', got %q", callID)
		}

		// Verify argument structure
		arg, _ := body["argument"].(map[string]interface{})
		if arg == nil {
			t.Fatal("expected argument map")
		}
		parsed, _ := arg["parsed"].([]interface{})
		if len(parsed) != 1 {
			t.Fatalf("expected 1 parsed arg, got %d", len(parsed))
		}
		params, _ := parsed[0].(map[string]interface{})
		if params["city"] != "London" {
			t.Errorf("expected city=London, got %v", params["city"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"response": "Weather in London: Sunny, 20C",
		})
	}))
	defer srv.Close()

	cfg := config{
		exec:   "get_weather",
		params: paramList{"city=London"},
	}
	err := doExec(srv.URL, "", "", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Integration: exec with basic auth
// ---------------------------------------------------------------------------

func TestDoExecWithAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	t.Run("correct auth", func(t *testing.T) {
		cfg := config{exec: "test_func"}
		err := doExec(srv.URL, "admin", "secret", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wrong auth", func(t *testing.T) {
		cfg := config{exec: "test_func"}
		err := doExec(srv.URL, "wrong", "creds", cfg)
		if err == nil {
			t.Error("expected auth error")
		}
	})
}

// ---------------------------------------------------------------------------
// Integration: HTTP error handling
// ---------------------------------------------------------------------------

func TestHTTPErrors(t *testing.T) {
	t.Run("connection refused", func(t *testing.T) {
		cfg := config{}
		err := doDumpSWML("http://127.0.0.1:1", "", "", cfg)
		if err == nil {
			t.Error("expected connection error")
		}
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}))
		defer srv.Close()

		cfg := config{}
		err := doDumpSWML(srv.URL, "", "", cfg)
		if err == nil {
			t.Error("expected error for 500 response")
		}
	})
}

// ---------------------------------------------------------------------------
// --example mode: sentinel extractor
// ---------------------------------------------------------------------------

func TestExtractSentinelPayload(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string // substring expected in err.Error(); "" means no error
	}{
		{
			name: "happy path with surrounding chatter",
			input: "starting service…\n" +
				"__SWAIG_TOOLS_BEGIN__\n" +
				`{"tools":[{"function":"foo","description":"bar"}]}` + "\n" +
				"__SWAIG_TOOLS_END__\n" +
				"some trailing log line that we ignore\n",
			want: `{"tools":[{"function":"foo","description":"bar"}]}`,
		},
		{
			name:  "minimal payload",
			input: "__SWAIG_TOOLS_BEGIN__\n{}\n__SWAIG_TOOLS_END__\n",
			want:  "{}",
		},
		{
			name:    "missing begin sentinel",
			input:   "{}\n__SWAIG_TOOLS_END__\n",
			wantErr: "missing __SWAIG_TOOLS_BEGIN__",
		},
		{
			name:    "missing end sentinel",
			input:   "__SWAIG_TOOLS_BEGIN__\n{}\n",
			wantErr: "missing __SWAIG_TOOLS_END__",
		},
		{
			name:    "partial begin marker",
			input:   "__SWAIG_TOOLS_BEG\n{}\n__SWAIG_TOOLS_END__\n",
			wantErr: "missing __SWAIG_TOOLS_BEGIN__",
		},
		{
			name:    "partial end marker",
			input:   "__SWAIG_TOOLS_BEGIN__\n{}\n__SWAIG_TOOLS_EN\n",
			wantErr: "missing __SWAIG_TOOLS_END__",
		},
		{
			name:    "end before begin",
			input:   "__SWAIG_TOOLS_END__\n{}\n__SWAIG_TOOLS_BEGIN__\n",
			wantErr: "missing __SWAIG_TOOLS_END__",
		},
		{
			name:  "whitespace around payload is trimmed",
			input: "__SWAIG_TOOLS_BEGIN__\n   {\"k\":1}   \n__SWAIG_TOOLS_END__\n",
			want:  `{"k":1}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractSentinelPayload(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil; payload=%q", tt.wantErr, got)
				}
				if !contains(err.Error(), tt.wantErr) {
					t.Errorf("err = %q, want substring %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("payload = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// --example mode: target resolution
// ---------------------------------------------------------------------------

func TestResolveExampleTarget(t *testing.T) {
	t.Run("rejects empty name", func(t *testing.T) {
		_, err := resolveExampleTarget("")
		if err == nil {
			t.Error("expected error for empty name")
		}
	})
	t.Run("rejects path traversal", func(t *testing.T) {
		for _, n := range []string{"..", ".", "foo/bar", `foo\bar`} {
			if _, err := resolveExampleTarget(n); err == nil {
				t.Errorf("expected error for %q", n)
			}
		}
	})
	t.Run("rejects nonexistent example", func(t *testing.T) {
		_, err := resolveExampleTarget("definitely_does_not_exist_xyz")
		if err == nil {
			t.Error("expected error for missing example")
		}
	})
}

// ---------------------------------------------------------------------------
// --example mode: run() argument validation
// ---------------------------------------------------------------------------

func TestRunValidationExampleMode(t *testing.T) {
	t.Run("example and url are mutually exclusive", func(t *testing.T) {
		cfg := config{example: "foo", url: "http://localhost:3000/", listTools: true}
		err := run(cfg)
		if err == nil || !contains(err.Error(), "mutually exclusive") {
			t.Errorf("expected mutual-exclusion error, got: %v", err)
		}
	})
	t.Run("example with --dump-swml is rejected", func(t *testing.T) {
		cfg := config{example: "foo", dumpSWML: true}
		err := run(cfg)
		if err == nil || !contains(err.Error(), "only supports --list-tools") {
			t.Errorf("expected only-list-tools error, got: %v", err)
		}
	})
}

// contains is a minimal substring check that doesn't depend on
// strings.Contains so we keep the test file's import surface narrow.
func contains(s, sub string) bool {
	if sub == "" {
		return true
	}
	return len(s) >= len(sub) && containsAll(s, sub)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
