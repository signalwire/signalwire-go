package agent

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// TestOnCallEndRegistersHookAndEnablesTranscript ports the reference's
// on_call_end contract: the reserved hangup_hook is registered once, the
// swaig_post_conversation param is turned on, handlers run in order with the
// resolved call log, and a failing handler does not stop the others.
func TestOnCallEndRegistersHookAndEnablesTranscript(t *testing.T) {
	a := NewAgentBase(WithName("ce"))
	var order []string
	var gotLog []map[string]any
	a.OnCallEnd(func(callLog []map[string]any, raw map[string]any) {
		order = append(order, "first")
		gotLog = callLog
	})
	a.OnCallEnd(func([]map[string]any, map[string]any) { panic("boom") })
	a.OnCallEnd(func([]map[string]any, map[string]any) { order = append(order, "third") })

	if a.params["swaig_post_conversation"] != true {
		t.Errorf("swaig_post_conversation = %v, want true", a.params["swaig_post_conversation"])
	}
	n := 0
	for _, name := range a.ListToolNames() {
		if name == "hangup_hook" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("hangup_hook registered %d times, want 1", n)
	}
	log := []any{map[string]any{"role": "user", "content": "hi"}}
	if _, err := a.OnFunctionCall("hangup_hook", map[string]any{}, map[string]any{"raw_call_log": log}); err != nil {
		t.Fatalf("hangup_hook: %v", err)
	}
	if !reflect.DeepEqual(order, []string{"first", "third"}) {
		t.Errorf("handler order = %v", order)
	}
	if !reflect.DeepEqual(gotLog, []map[string]any{{"role": "user", "content": "hi"}}) {
		t.Errorf("call log = %#v", gotLog)
	}
}

func TestOnCallEndRespectsExplicitFalse(t *testing.T) {
	a := NewAgentBase(WithName("ce2"))
	a.SetParam("swaig_post_conversation", false)
	a.OnCallEnd(func([]map[string]any, map[string]any) {})
	if a.params["swaig_post_conversation"] != false {
		t.Errorf("explicit false overridden: %v", a.params["swaig_post_conversation"])
	}
}

// TestAddPerCallConfigComposes: callbacks accumulate and run in order on the
// ephemeral agent; SetDynamicConfigCallback replaces the chain.
func TestAddPerCallConfigComposes(t *testing.T) {
	a := NewAgentBase(WithName("pc"), WithBasicAuth("u", "p"))
	var ran []string
	a.AddPerCallConfig(func(_ map[string]string, _ map[string]any, _ map[string]string, eph *AgentBase) {
		ran = append(ran, "a")
		eph.SetParam("temperature", 0.1)
	})
	a.AddPerCallConfig(func(_ map[string]string, _ map[string]any, _ map[string]string, eph *AgentBase) {
		ran = append(ran, "b")
	})
	req := httptest.NewRequest("POST", "/", nil)
	a.handleDynamicConfig(map[string]any{}, req)
	if !reflect.DeepEqual(ran, []string{"a", "b"}) {
		t.Errorf("ran = %v, want [a b]", ran)
	}
	if _, set := a.params["temperature"]; set {
		t.Error("per-call config leaked into the master agent")
	}
	ran = nil
	a.SetDynamicConfigCallback(func(map[string]string, map[string]any, map[string]string, *AgentBase) { ran = append(ran, "only") })
	a.handleDynamicConfig(map[string]any{}, req)
	if !reflect.DeepEqual(ran, []string{"only"}) {
		t.Errorf("after replace ran = %v", ran)
	}
}

// TestMountServesExtraHandlerUnderPrefix: Mount serves a handler at its prefix
// (prefix stripped) and never shadows the agent's own SWML route.
func TestMountServesExtraHandlerUnderPrefix(t *testing.T) {
	a := NewAgentBase(WithName("m"), WithRoute("/agent"), WithBasicAuth("u", "p"))
	a.Mount(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("chat:" + r.URL.Path))
	}), MountOptions{Prefix: "/chat/"})
	srv := httptest.NewServer(a.AsRouter())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/chat/hello")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if string(body) != "chat:/hello" {
		t.Errorf("mounted handler body = %q", body)
	}
	// The agent's own route still answers (401 without credentials, not the mount).
	resp2, err := http.Get(srv.URL + "/agent")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("agent route status = %d, want 401 (auth-protected agent route)", resp2.StatusCode)
	}
}

// TestMountSamePrefixFallsThrough: two handlers mounted at one prefix (a chat
// gateway and its handoff routes share a URL) are tried in order, a 404 from the
// first falling through to the second without leaking its headers.
func TestMountSamePrefixFallsThrough(t *testing.T) {
	a := NewAgentBase(WithName("m"), WithRoute("/agent"))
	first := http.NewServeMux()
	first.HandleFunc("POST /{$}", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("gateway")) })
	second := http.NewServeMux()
	second.HandleFunc("POST /say", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("say")) })
	a.Mount(first, MountOptions{Prefix: "/chat"})
	a.Mount(second, MountOptions{Prefix: "/chat"})
	srv := httptest.NewServer(a.AsRouter())
	defer srv.Close()

	for path, want := range map[string]string{"/chat/": "gateway", "/chat/say": "say"} {
		resp, err := http.Post(srv.URL+path, "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK || string(body) != want {
			t.Errorf("POST %s = %d %q, want %q", path, resp.StatusCode, body, want)
		}
		if resp.Header.Get("X-Content-Type-Options") != "" {
			t.Errorf("POST %s leaked the first handler's 404 headers: %v", path, resp.Header)
		}
	}
	resp, err := http.Post(srv.URL+"/chat/nothing", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unmatched path = %d, want 404", resp.StatusCode)
	}
}
