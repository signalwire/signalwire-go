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
// Lambda + proxy route preservation (regression)
// ---------------------------------------------------------------------------
//
// These tests pin down the exact bug called out in the Lambda port brief:
// when SWML_PROXY_URL_BASE is set AND the agent is running inside Lambda,
// the webhook URL MUST include both the agent's route and the /swaig
// suffix. Python and TypeScript originally dropped the route in this
// combination; the Go SDK had the correct behaviour from the start but
// we now guard it with these explicit checks.

// clearAgentAWSEnv zeroes the env vars inspected by swml.GetExecutionMode
// and the proxy override, so that tests running inside a real cloud
// runtime (GKE, Lambda, etc.) start from a clean slate.
func clearAgentAWSEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"GATEWAY_INTERFACE",
		"AWS_LAMBDA_FUNCTION_NAME",
		"LAMBDA_TASK_ROOT",
		"AWS_LAMBDA_FUNCTION_URL",
		"AWS_REGION",
		"FUNCTION_TARGET",
		"K_SERVICE",
		"GOOGLE_CLOUD_PROJECT",
		"AZURE_FUNCTIONS_ENVIRONMENT",
		"FUNCTIONS_WORKER_RUNTIME",
		"AzureWebJobsStorage",
		"SWML_PROXY_URL_BASE",
	} {
		t.Setenv(k, "")
	}
}

func TestBuildWebhookURL_LambdaNonRootRouteAppendsRoute(t *testing.T) {
	clearAgentAWSEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "demo-func")
	t.Setenv("AWS_LAMBDA_FUNCTION_URL", "https://demo-func.lambda-url.us-east-1.on.aws")

	a := NewAgentBase(
		WithRoute("/my-agent"),
		WithBasicAuth("u", "p"),
	)
	url := a.buildWebhookURL()
	const want = "/my-agent/swaig"
	if !strings.Contains(url, want) {
		t.Fatalf("buildWebhookURL() = %q, want substring %q", url, want)
	}
	if !strings.Contains(url, "demo-func.lambda-url.us-east-1.on.aws") {
		t.Fatalf("buildWebhookURL() = %q, want Lambda host", url)
	}
}

// REGRESSION GUARD: non-root route + Lambda env + proxy base must all
// produce a webhook URL that still contains the agent's route + /swaig.
// This is the exact combo that originally hid the bug in Python and TS.
func TestBuildWebhookURL_Regression_LambdaProxyRouteCombo(t *testing.T) {
	clearAgentAWSEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "demo-func")
	t.Setenv("AWS_LAMBDA_FUNCTION_URL", "https://should-be-ignored.lambda-url.us-east-1.on.aws")
	t.Setenv("SWML_PROXY_URL_BASE", "https://xyz.lambda-url.us-east-1.on.aws")

	a := NewAgentBase(
		WithRoute("/my-agent"),
		WithBasicAuth("u", "p"),
	)
	url := a.buildWebhookURL()
	if !strings.Contains(url, "/my-agent/swaig") {
		t.Fatalf(
			"route-preservation regression: buildWebhookURL() = %q, "+
				"want substring %q. The proxy base MUST have the "+
				"agent's route and /swaig appended.",
			url, "/my-agent/swaig",
		)
	}
	if !strings.Contains(url, "xyz.lambda-url.us-east-1.on.aws") {
		t.Fatalf("buildWebhookURL() = %q, expected proxy host", url)
	}
	// Make absolutely sure the buggy shape is NOT present.
	if strings.HasSuffix(url, "xyz.lambda-url.us-east-1.on.aws/swaig") {
		t.Fatalf("buildWebhookURL() = %q matches the forbidden shape", url)
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
