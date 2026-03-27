package skills

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// testSkill is a minimal SkillBase implementation for testing.
type testSkill struct {
	BaseSkill
}

func (s *testSkill) Setup() bool {
	return true
}

func (s *testSkill) RegisterTools() []ToolRegistration {
	return []ToolRegistration{
		{
			Name:        "test_tool",
			Description: "A test tool",
			Handler: func(args map[string]any, raw map[string]any) *swaig.FunctionResult {
				return swaig.NewFunctionResult("test ok")
			},
		},
	}
}

func newTestSkill(name string) SkillBase {
	return &testSkill{
		BaseSkill: BaseSkill{SkillName: name, SkillDesc: "test skill " + name},
	}
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

func TestRegisterSkill_AndGetFactory(t *testing.T) {
	RegisterSkill("test_registry", func(params map[string]any) SkillBase {
		return newTestSkill("test_registry")
	})
	defer func() {
		registryMu.Lock()
		delete(registry, "test_registry")
		registryMu.Unlock()
	}()

	factory := GetSkillFactory("test_registry")
	if factory == nil {
		t.Fatal("expected factory to be registered")
	}
	s := factory(nil)
	if s.Name() != "test_registry" {
		t.Errorf("name = %q", s.Name())
	}
}

func TestGetSkillFactory_NotFound(t *testing.T) {
	factory := GetSkillFactory("nonexistent_skill_xyz")
	if factory != nil {
		t.Error("expected nil factory for nonexistent skill")
	}
}

func TestListSkills_NotEmpty(t *testing.T) {
	RegisterSkill("test_list_skill", func(params map[string]any) SkillBase {
		return newTestSkill("test_list_skill")
	})
	defer func() {
		registryMu.Lock()
		delete(registry, "test_list_skill")
		registryMu.Unlock()
	}()

	listed := ListSkills()
	found := false
	for _, name := range listed {
		if name == "test_list_skill" {
			found = true
			break
		}
	}
	if !found {
		t.Error("test_list_skill not in ListSkills()")
	}
}

// ---------------------------------------------------------------------------
// SkillManager
// ---------------------------------------------------------------------------

func TestSkillManager_LoadAndUnload(t *testing.T) {
	sm := NewSkillManager()
	skill := newTestSkill("test_sm")

	ok, errMsg := sm.LoadSkill(skill)
	if !ok {
		t.Fatalf("LoadSkill failed: %s", errMsg)
	}

	if !sm.HasSkill("test_sm") {
		t.Error("HasSkill should return true")
	}

	loaded := sm.ListLoadedSkills()
	if len(loaded) != 1 || loaded[0] != "test_sm" {
		t.Errorf("ListLoadedSkills = %v", loaded)
	}

	got := sm.GetSkill("test_sm")
	if got == nil {
		t.Error("GetSkill returned nil")
	}

	ok2 := sm.UnloadSkill("test_sm")
	if !ok2 {
		t.Error("UnloadSkill returned false")
	}

	if sm.HasSkill("test_sm") {
		t.Error("HasSkill should return false after unload")
	}
}

func TestSkillManager_LoadDuplicate(t *testing.T) {
	sm := NewSkillManager()
	skill := newTestSkill("dup")

	ok, _ := sm.LoadSkill(skill)
	if !ok {
		t.Fatal("first LoadSkill failed")
	}

	// Create a second instance to try loading
	skill2 := newTestSkill("dup")
	ok, errMsg := sm.LoadSkill(skill2)
	if ok {
		t.Error("duplicate LoadSkill should fail")
	}
	if errMsg == "" {
		t.Error("expected error message for duplicate")
	}
}

func TestSkillManager_UnloadNonExistent(t *testing.T) {
	sm := NewSkillManager()
	if sm.UnloadSkill("nope") {
		t.Error("UnloadSkill should return false for non-existent")
	}
}

func TestSkillManager_GetNonExistent(t *testing.T) {
	sm := NewSkillManager()
	if sm.GetSkill("nope") != nil {
		t.Error("GetSkill should return nil for non-existent")
	}
}

// ---------------------------------------------------------------------------
// BaseSkill defaults and parameter helpers
// ---------------------------------------------------------------------------

func TestBaseSkill_Defaults(t *testing.T) {
	b := &BaseSkill{SkillName: "test", SkillDesc: "desc"}

	if b.Version() != "1.0.0" {
		t.Errorf("Version = %q", b.Version())
	}
	if b.RequiredEnvVars() != nil {
		t.Error("expected nil RequiredEnvVars")
	}
	if b.SupportsMultipleInstances() {
		t.Error("expected false")
	}
	if b.GetHints() != nil {
		t.Error("expected nil hints")
	}
	if b.GetGlobalData() != nil {
		t.Error("expected nil global data")
	}
	if b.GetPromptSections() != nil {
		t.Error("expected nil prompt sections")
	}
	if b.GetInstanceKey() != "test" {
		t.Errorf("instance key = %q", b.GetInstanceKey())
	}
}

func TestBaseSkill_CustomVersion(t *testing.T) {
	b := &BaseSkill{SkillName: "test", SkillDesc: "desc", SkillVer: "2.0.0"}
	if b.Version() != "2.0.0" {
		t.Errorf("Version = %q", b.Version())
	}
}

func TestBaseSkill_GetParam(t *testing.T) {
	b := &BaseSkill{
		SkillName: "test",
		Params:    map[string]any{"key": "value"},
	}
	v, ok := b.GetParam("key")
	if !ok || v != "value" {
		t.Errorf("GetParam(key) = %v, %v", v, ok)
	}
	_, ok = b.GetParam("missing")
	if ok {
		t.Error("GetParam(missing) should return false")
	}
}

func TestBaseSkill_GetParamNilParams(t *testing.T) {
	b := &BaseSkill{SkillName: "test"}
	_, ok := b.GetParam("key")
	if ok {
		t.Error("GetParam with nil Params should return false")
	}
}

func TestBaseSkill_GetParamString_TypeMismatch(t *testing.T) {
	b := &BaseSkill{
		SkillName: "test",
		Params:    map[string]any{"num": 42},
	}
	result := b.GetParamString("num", "default")
	if result != "default" {
		t.Errorf("expected default for non-string, got %q", result)
	}
}

func TestBaseSkill_GetParamInt_FloatConversion(t *testing.T) {
	b := &BaseSkill{
		SkillName: "test",
		Params:    map[string]any{"val": float64(42)},
	}
	result := b.GetParamInt("val", 0)
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

func TestBaseSkill_GetParamInt_Int64(t *testing.T) {
	b := &BaseSkill{
		SkillName: "test",
		Params:    map[string]any{"val": int64(100)},
	}
	result := b.GetParamInt("val", 0)
	if result != 100 {
		t.Errorf("expected 100, got %d", result)
	}
}

func TestBaseSkill_GetParamFloat_IntConversion(t *testing.T) {
	b := &BaseSkill{
		SkillName: "test",
		Params:    map[string]any{"val": 42},
	}
	result := b.GetParamFloat("val", 0)
	if result != 42.0 {
		t.Errorf("expected 42.0, got %v", result)
	}
}

func TestBaseSkill_GetParameterSchema_HasCommonFields(t *testing.T) {
	b := &BaseSkill{SkillName: "test", SkillDesc: "desc"}
	schema := b.GetParameterSchema()
	if schema["swaig_fields"] == nil {
		t.Error("expected swaig_fields")
	}
	if schema["skip_prompt"] == nil {
		t.Error("expected skip_prompt")
	}
	if schema["tool_name"] == nil {
		t.Error("expected tool_name")
	}
}

func TestBaseSkill_Cleanup_NoPanic(t *testing.T) {
	b := &BaseSkill{SkillName: "test"}
	b.Cleanup()
}

// ---------------------------------------------------------------------------
// ToolRegistration structure
// ---------------------------------------------------------------------------

func TestToolRegistration_Fields(t *testing.T) {
	tr := ToolRegistration{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Handler: func(args map[string]any, raw map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("ok")
		},
		Secure: true,
		Fillers: map[string][]string{
			"en-US": {"Please wait..."},
		},
		SwaigFields: map[string]any{"extra": "field"},
	}

	if tr.Name != "test_tool" {
		t.Errorf("Name = %q", tr.Name)
	}
	if !tr.Secure {
		t.Error("expected Secure=true")
	}
	if tr.Handler == nil {
		t.Error("expected non-nil Handler")
	}
}
