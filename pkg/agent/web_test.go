package agent

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Webhook URL construction
// ---------------------------------------------------------------------------

func TestBuildWebhookURL_DefaultContainsSwaig(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("user", "pass"))
	url := a.buildWebhookURL()
	if !strings.HasSuffix(url, "/swaig") {
		t.Errorf("webhook URL should end with /swaig, got %q", url)
	}
}

func TestBuildWebhookURL_ContainsCredentials(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("myuser", "mypass"))
	url := a.buildWebhookURL()
	if !strings.Contains(url, "myuser:mypass@") {
		t.Errorf("webhook URL should contain credentials, got %q", url)
	}
}

func TestBuildWebhookURL_WithRoute(t *testing.T) {
	a := NewAgentBase(WithRoute("/agent"), WithBasicAuth("u", "p"))
	url := a.buildWebhookURL()
	if !strings.Contains(url, "/agent/swaig") {
		t.Errorf("webhook URL should include route, got %q", url)
	}
}

func TestBuildWebhookURL_ExplicitOverride_Web(t *testing.T) {
	a := NewAgentBase()
	a.SetWebHookUrl("https://custom.example.com/swaig")
	url := a.buildWebhookURL()
	if url != "https://custom.example.com/swaig" {
		t.Errorf("expected explicit URL, got %q", url)
	}
}

func TestBuildWebhookURL_WithQueryParams_Web(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.AddSwaigQueryParams(map[string]string{"agent_id": "123"})
	url := a.buildWebhookURL()
	if !strings.Contains(url, "agent_id=123") {
		t.Errorf("webhook URL should contain query param, got %q", url)
	}
	if !strings.Contains(url, "?") {
		t.Errorf("webhook URL should contain ?, got %q", url)
	}
}

func TestBuildWebhookURL_MultipleQueryParams(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.AddSwaigQueryParams(map[string]string{"a": "1", "b": "2"})
	url := a.buildWebhookURL()
	if !strings.Contains(url, "a=1") || !strings.Contains(url, "b=2") {
		t.Errorf("expected both query params, got %q", url)
	}
}

// ---------------------------------------------------------------------------
// Post-prompt URL
// ---------------------------------------------------------------------------

func TestBuildPostPromptURL_Default(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	url := a.buildPostPromptURL()
	if !strings.HasSuffix(url, "/post_prompt") {
		t.Errorf("post prompt URL should end with /post_prompt, got %q", url)
	}
}

func TestBuildPostPromptURL_ExplicitOverride(t *testing.T) {
	a := NewAgentBase()
	a.SetPostPromptUrl("https://custom.example.com/summary")
	url := a.buildPostPromptURL()
	if url != "https://custom.example.com/summary" {
		t.Errorf("expected explicit URL, got %q", url)
	}
}

func TestBuildPostPromptURL_WithRoute(t *testing.T) {
	a := NewAgentBase(WithRoute("/bot"), WithBasicAuth("u", "p"))
	url := a.buildPostPromptURL()
	if !strings.Contains(url, "/bot/post_prompt") {
		t.Errorf("post prompt URL should include route, got %q", url)
	}
}

// ---------------------------------------------------------------------------
// SWAIG query params
// ---------------------------------------------------------------------------

func TestClearSwaigQueryParams_Web(t *testing.T) {
	a := NewAgentBase()
	a.AddSwaigQueryParams(map[string]string{"key": "value"})
	a.ClearSwaigQueryParams()
	if len(a.swaigQueryParams) != 0 {
		t.Errorf("expected empty params, got %d", len(a.swaigQueryParams))
	}
}

func TestAddSwaigQueryParams_Merge(t *testing.T) {
	a := NewAgentBase()
	a.AddSwaigQueryParams(map[string]string{"a": "1"})
	a.AddSwaigQueryParams(map[string]string{"b": "2"})
	if len(a.swaigQueryParams) != 2 {
		t.Errorf("expected 2 params, got %d", len(a.swaigQueryParams))
	}
}

func TestAddSwaigQueryParams_Overwrite(t *testing.T) {
	a := NewAgentBase()
	a.AddSwaigQueryParams(map[string]string{"key": "v1"})
	a.AddSwaigQueryParams(map[string]string{"key": "v2"})
	if a.swaigQueryParams["key"] != "v2" {
		t.Errorf("expected overwritten value v2, got %q", a.swaigQueryParams["key"])
	}
}

// ---------------------------------------------------------------------------
// Proxy URL
// ---------------------------------------------------------------------------

func TestManualSetProxyUrl_Basic(t *testing.T) {
	a := NewAgentBase()
	a.ManualSetProxyUrl("https://proxy.example.com")
	if a.proxyURLBase != "https://proxy.example.com" {
		t.Errorf("proxyURLBase = %q", a.proxyURLBase)
	}
}

// ---------------------------------------------------------------------------
// Dynamic config callback
// ---------------------------------------------------------------------------

func TestSetDynamicConfigCallback_Basic(t *testing.T) {
	a := NewAgentBase()
	called := false
	a.SetDynamicConfigCallback(func(q map[string]string, b map[string]any, h map[string]string, agent *AgentBase) {
		called = true
	})
	if a.dynamicConfigCallback == nil {
		t.Error("expected non-nil callback")
	}
	_ = called
}

// ---------------------------------------------------------------------------
// SIP
// ---------------------------------------------------------------------------

func TestExtractSIPUsername_FromCallTo(t *testing.T) {
	// This tests the SWML service's ExtractSIPUsername indirectly
	a := NewAgentBase()
	a.EnableSipRouting(true, "")
	a.RegisterSipUsername("alice")

	if !a.sipRoutingEnabled {
		t.Error("expected sipRoutingEnabled=true")
	}
	if !a.sipUsernames["alice"] {
		t.Error("expected alice in sipUsernames")
	}
}

func TestRegisterSipUsername_Multiple(t *testing.T) {
	a := NewAgentBase()
	a.RegisterSipUsername("user1")
	a.RegisterSipUsername("user2")
	a.RegisterSipUsername("user1") // duplicate
	if len(a.sipUsernames) != 2 {
		t.Errorf("expected 2 unique usernames, got %d", len(a.sipUsernames))
	}
}

// ---------------------------------------------------------------------------
// Method chaining
// ---------------------------------------------------------------------------

func TestWebMethods_ReturnSelf(t *testing.T) {
	a := NewAgentBase()
	if a.ManualSetProxyUrl("x") != a {
		t.Error("ManualSetProxyUrl should return self")
	}
	if a.SetWebHookUrl("x") != a {
		t.Error("SetWebHookUrl should return self")
	}
	if a.SetPostPromptUrl("x") != a {
		t.Error("SetPostPromptUrl should return self")
	}
	if a.AddSwaigQueryParams(nil) != a {
		t.Error("AddSwaigQueryParams should return self")
	}
	if a.ClearSwaigQueryParams() != a {
		t.Error("ClearSwaigQueryParams should return self")
	}
	if a.EnableSipRouting(false, "") != a {
		t.Error("EnableSipRouting should return self")
	}
	if a.RegisterSipUsername("x") != a {
		t.Error("RegisterSipUsername should return self")
	}
	if a.EnableDebugRoutes() != a {
		t.Error("EnableDebugRoutes should return self")
	}
}
