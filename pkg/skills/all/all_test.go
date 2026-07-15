package all

import (
	"testing"

	"github.com/signalwire/signalwire-go/v3/pkg/skills"
)

// allSkillNames is the full built-in set: the 17 light skills (registered by
// pkg/skills/builtin) plus spider (registered by pkg/skills/builtin/spider).
// Importing this umbrella package must register all 18 — that one-step
// "everything works" guarantee is the reason the package exists.
var allSkillNames = []string{
	"datetime",
	"math",
	"joke",
	"weather_api",
	"web_search",
	"wikipedia_search",
	"google_maps",
	"spider",
	"datasphere",
	"datasphere_serverless",
	"swml_transfer",
	"play_background_file",
	"api_ninjas_trivia",
	"native_vector_search",
	"info_gatherer",
	"claude_skills",
	"mcp_gateway",
	"custom_skills",
}

// TestUmbrellaRegistersAllSkills verifies that blank-importing pkg/skills/all
// (this package's import block does so transitively) registers every built-in
// skill, including the dependency-carrying spider skill.
func TestUmbrellaRegistersAllSkills(t *testing.T) {
	registered := make(map[string]bool)
	for _, name := range skills.ListSkills() {
		registered[name] = true
	}

	for _, name := range allSkillNames {
		if !registered[name] {
			t.Errorf("skill %q not registered via the all umbrella", name)
		}
	}

	if len(skills.ListSkills()) < len(allSkillNames) {
		t.Errorf("expected at least %d skills via all, got %d",
			len(allSkillNames), len(skills.ListSkills()))
	}
}
