// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package skills

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// Tests for the SkillBase accessor surface added in this PR
// (GetInstanceKey single-instance path, GetSkillNamespace, GetSkillData,
// UpdateSkillData). Equivalent Python coverage lives in
// tests/unit/skills/test_skill_base.py.
//
// Note on multi-instance dispatch: Go doesn't have virtual method dispatch
// from a base type. Real multi-instance skills (DataSphereSkill, etc.) must
// override BOTH SupportsMultipleInstances AND GetInstanceKey themselves —
// the base-class GetInstanceKey multi-instance branch is never actually
// reached by such skills because the *BaseSkill method receiver reads the
// base SupportsMultipleInstances. These tests therefore exercise only the
// single-instance path on the base type.

func TestGetInstanceKeySingleInstance(t *testing.T) {
	b := &BaseSkill{SkillName: "web_search"}
	if got := b.GetInstanceKey(); got != "web_search" {
		t.Fatalf("GetInstanceKey = %q, want %q", got, "web_search")
	}
}

// --- GetSkillNamespace ---

func TestGetSkillNamespaceUsesInstanceKeyByDefault(t *testing.T) {
	b := &BaseSkill{SkillName: "web_search"}
	if got := b.GetSkillNamespace(); got != "skill:web_search" {
		t.Fatalf("GetSkillNamespace = %q, want %q", got, "skill:web_search")
	}
}

func TestGetSkillNamespacePrefixOverridesInstanceKey(t *testing.T) {
	b := &BaseSkill{SkillName: "datasphere", Params: map[string]any{"prefix": "kb"}}
	if got := b.GetSkillNamespace(); got != "skill:kb" {
		t.Fatalf("GetSkillNamespace = %q, want %q", got, "skill:kb")
	}
}

// --- GetSkillData ---

func TestGetSkillDataReturnsEmptyWhenNoGlobalData(t *testing.T) {
	b := &BaseSkill{SkillName: "x"}
	got := b.GetSkillData(map[string]any{})
	if len(got) != 0 {
		t.Fatalf("GetSkillData without global_data should return empty map, got %v", got)
	}
}

func TestGetSkillDataReturnsEmptyWhenNamespaceAbsent(t *testing.T) {
	b := &BaseSkill{SkillName: "x"}
	got := b.GetSkillData(map[string]any{
		"global_data": map[string]any{"other_ns": map[string]any{"k": "v"}},
	})
	if len(got) != 0 {
		t.Fatalf("GetSkillData with mismatched namespace should return empty, got %v", got)
	}
}

func TestGetSkillDataReadsNamespacedEntry(t *testing.T) {
	b := &BaseSkill{SkillName: "web_search"}
	want := map[string]any{"cached_query": "hello"}
	got := b.GetSkillData(map[string]any{
		"global_data": map[string]any{"skill:web_search": want},
	})
	if got["cached_query"] != "hello" {
		t.Fatalf("GetSkillData = %v, want %v", got, want)
	}
}

// --- UpdateSkillData ---

func TestUpdateSkillDataWritesNamespacedEntry(t *testing.T) {
	b := &BaseSkill{SkillName: "web_search"}
	result := swaig.NewFunctionResult("ok")
	ret := b.UpdateSkillData(result, map[string]any{"cached_query": "hello"})
	if ret != result {
		t.Fatalf("UpdateSkillData should return the same FunctionResult for chaining")
	}

	// UpdateGlobalData pushes a set_global_data action carrying the
	// namespaced state; inspect via ToMap() since FunctionResult's fields
	// are unexported.
	payload := result.ToMap()
	actions, ok := payload["action"].([]map[string]any)
	if !ok || len(actions) != 1 {
		t.Fatalf("expected exactly one action, got %#v", payload["action"])
	}
	sgd, ok := actions[0]["set_global_data"].(map[string]any)
	if !ok {
		t.Fatalf("expected set_global_data action, got %#v", actions[0])
	}
	nsPayload, ok := sgd["skill:web_search"].(map[string]any)
	if !ok {
		t.Fatalf(`expected "skill:web_search" key in set_global_data, got %#v`, sgd)
	}
	if nsPayload["cached_query"] != "hello" {
		t.Fatalf(`namespaced payload = %#v, expected cached_query="hello"`, nsPayload)
	}
}

func TestUpdateSkillDataUsesPrefixWhenSet(t *testing.T) {
	b := &BaseSkill{SkillName: "web_search", Params: map[string]any{"prefix": "web"}}
	result := swaig.NewFunctionResult("ok")
	b.UpdateSkillData(result, map[string]any{"k": "v"})

	payload := result.ToMap()
	actions, _ := payload["action"].([]map[string]any)
	sgd, _ := actions[0]["set_global_data"].(map[string]any)
	if _, present := sgd["skill:web"]; !present {
		t.Fatalf(`expected prefix-based namespace "skill:web" in set_global_data, got %#v`, sgd)
	}
	if _, present := sgd["skill:web_search"]; present {
		t.Fatalf(`instance-key namespace "skill:web_search" should not appear when prefix is set, got %#v`, sgd)
	}
}
