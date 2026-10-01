package prefabs

import (
	"strings"
	"testing"
)

// A Bedrock agent must not lose debug-event wiring in its render.
//
// Cross-port context: typescript's BedrockAgent.renderSwml rebuilt the rendered
// document against a fixed six-key ALLOW-list, so any key the allow-list did not
// name vanished silently — debug events were unreachable on every Bedrock agent
// there, with no error and no test. The python reference has the same shape
// (agents/bedrock.py:113-130 rebuilds the amazon_bedrock verb from exactly
// prompt/SWAIG/params/global_data/post_prompt/post_prompt_url, dropping hints,
// languages, pronounce and multilingual).
//
// Go does NOT have that path: it renders the verb once under an overridden name
// (aiVerbName = "amazon_bedrock") and never rebuilds the document afterwards. The
// only post-assembly transform is the prompt transformer, which is a DENY-list
// (addVoiceToPrompt skips three text-model keys and copies everything else), so
// unknown keys pass through instead of disappearing. This test pins that
// property against the allow-list regression.
func TestBedrockAgent_DebugEventsSurviveTheRender(t *testing.T) {
	ba := NewBedrockAgent(BedrockOptions{Name: "bedrock", Route: "/bedrock"})
	ba.SetPromptText("You are a bot")
	ba.EnableDebugEvents(1)

	doc := ba.RenderSWML(nil, nil)
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	for _, v := range main {
		vm, _ := v.(map[string]any)
		cfg, ok := vm["amazon_bedrock"].(map[string]any)
		if !ok {
			continue
		}
		params, ok := cfg["params"].(map[string]any)
		if !ok {
			t.Fatal("expected a params object on the amazon_bedrock verb")
		}
		if params["debug_webhook_level"] != 1 {
			t.Errorf("debug_webhook_level = %v, want 1", params["debug_webhook_level"])
		}
		u, _ := params["debug_webhook_url"].(string)
		if !strings.Contains(u, "/debug_events") {
			t.Errorf("debug_webhook_url = %q, want it to contain /debug_events", u)
		}
		return
	}
	t.Fatal("amazon_bedrock verb not found")
}
