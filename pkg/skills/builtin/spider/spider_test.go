package spider

import (
	"testing"

	"github.com/signalwire/signalwire-go/v3/pkg/skills"
)

// TestSpiderSetup tests SpiderSkill. The skill registers itself via this
// package's init(); importing the package (as this test file does, being in
// the package) runs that init(), so the registry lookup resolves.
func TestSpiderSetup(t *testing.T) {
	factory := skills.GetSkillFactory("spider")
	if factory == nil {
		t.Fatal("spider factory not found")
	}
	s := factory(nil)
	if !s.Setup() {
		t.Error("spider Setup() returned false")
	}
	if !s.SupportsMultipleInstances() {
		t.Error("spider should support multiple instances")
	}
	tools := s.RegisterTools()
	if len(tools) == 0 {
		t.Error("spider RegisterTools() returned empty")
	}
}

func TestSpider_CustomToolName(t *testing.T) {
	factory := skills.GetSkillFactory("spider")
	s := factory(map[string]any{"tool_name": "my_spider"})
	s.Setup()
	tools := s.RegisterTools()
	if tools[0].Name != "my_spider_scrape_url" {
		t.Errorf("tool name = %q, want my_spider_scrape_url", tools[0].Name)
	}
}

func TestSpider_DefaultToolName(t *testing.T) {
	factory := skills.GetSkillFactory("spider")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	if tools[0].Name != "scrape_url" {
		t.Errorf("tool name = %q, want scrape_url", tools[0].Name)
	}
}

func TestSpider_HasHints(t *testing.T) {
	factory := skills.GetSkillFactory("spider")
	s := factory(nil)
	s.Setup()
	hints := s.GetHints()
	if len(hints) == 0 {
		t.Error("expected hints")
	}
}

func TestSpider_InstanceKey(t *testing.T) {
	factory := skills.GetSkillFactory("spider")
	s := factory(map[string]any{"tool_name": "custom"})
	key := s.GetInstanceKey()
	if key != "spider_custom" {
		t.Errorf("instance key = %q, want spider_custom", key)
	}
}
