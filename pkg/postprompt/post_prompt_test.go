package postprompt

import (
	"reflect"
	"testing"
)

const fenced = "```json\n{\"summary\": \"s\", \"already_answered\": [\"pricing\"]}\n```"

func TestParseShapes(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want map[string]any
	}{
		{"flat voice keys", map[string]any{"summary": "s", "user_goal": "g"}, map[string]any{"summary": "s", "user_goal": "g"}},
		{"fenced chat raw", map[string]any{"raw": fenced}, map[string]any{"summary": "s", "already_answered": []any{"pricing"}}},
		{"list under parsed", map[string]any{"parsed": []any{map[string]any{"summary": "s3"}}, "raw": "..."}, map[string]any{"summary": "s3"}},
		{"parsed bare dict", map[string]any{"parsed": map[string]any{"summary": "s"}}, map[string]any{"summary": "s"}},
		{"prose kept", map[string]any{"raw": "They asked about pricing."}, map[string]any{"summary": "They asked about pricing."}},
		{"json not an object", map[string]any{"raw": `"just a string"`}, map[string]any{"summary": "just a string"}},
	}
	for _, c := range cases {
		if got := ParsePostPromptData(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %#v, want %#v", c.name, got, c.want)
		}
	}
	if got := ParsePostPromptData(map[string]any{"parsed": []any{map[string]any{"summary": "s"}}}); got["parsed"] != nil {
		t.Errorf("parsed wrapper must be unwrapped, got %#v", got)
	}
	for _, junk := range []any{nil, map[string]any{}, "text", 42, []any{}, map[string]any{"raw": ""}, map[string]any{"raw": "   "}, map[string]any{"raw": nil}} {
		if got := ParsePostPromptData(junk); len(got) != 0 {
			t.Errorf("junk %#v: got %#v, want empty", junk, got)
		}
	}
}

func TestStripJSONFence(t *testing.T) {
	for raw, want := range map[string]string{
		"```json\n{\"a\":1}\n```": `{"a":1}`,
		"```\nplain\n```":         "plain",
		"no fence at all":         "no fence at all",
		"":                        "",
	} {
		if got := StripJSONFence(raw); got != want {
			t.Errorf("StripJSONFence(%q) = %q, want %q", raw, got, want)
		}
	}
}

func dialogueLog() []any {
	return []any{
		map[string]any{"role": "user", "content": "hi"},
		map[string]any{"role": "assistant", "content": "hello"},
		map[string]any{"role": "system", "content": "the prompt"},
		map[string]any{"role": "system-log", "content": "step trace"},
		map[string]any{"role": "tool", "content": "tool output"},
		map[string]any{"role": "assistant", "content": "", "tool_calls": []any{map[string]any{"id": 1}}},
		map[string]any{"role": "assistant-manual", "content": "let me look that up"},
		map[string]any{"role": "assistant", "content": "   "},
		"not even a dict",
	}
}

func TestDialogueTurns(t *testing.T) {
	want := []map[string]string{{"role": "user", "content": "hi"}, {"role": "assistant", "content": "hello"}}
	if got := DialogueTurns(dialogueLog(), DialogueOptions{}); !reflect.DeepEqual(got, want) {
		t.Errorf("DialogueTurns = %#v", got)
	}
	withEcho := append(dialogueLog(), map[string]any{"role": "assistant", "content": fenced})
	echo := fenced
	for _, turn := range DialogueTurns(withEcho, DialogueOptions{DropEcho: &echo}) {
		if turn["content"] == fenced {
			t.Error("summary echo not dropped")
		}
	}
	if got := DialogueTurns(withEcho, DialogueOptions{}); len(got) != 3 {
		t.Errorf("echo kept when not asked to drop: %d turns, want 3", len(got))
	}
	for _, junk := range []any{nil, []any{}, "nonsense", 42} {
		if got := DialogueTurns(junk, DialogueOptions{}); len(got) != 0 {
			t.Errorf("junk %#v yielded %#v", junk, got)
		}
	}
}

func TestNormalizeVoiceAndChat(t *testing.T) {
	voice := NormalizePostPrompt(map[string]any{
		"conversation_type": "voice",
		"call_id":           "c-1",
		"post_prompt_data":  map[string]any{"parsed": []any{map[string]any{"summary": "v"}}},
		"raw_call_log":      []any{map[string]any{"role": "user", "content": "hi"}},
	})
	if voice.Medium != "voice" || voice.ConversationID != nil || voice.CallID == nil || *voice.CallID != "c-1" ||
		!reflect.DeepEqual(voice.Summary, map[string]any{"summary": "v"}) || len(voice.Dialogue) != 1 {
		t.Errorf("voice = %#v", voice)
	}
	chat := NormalizePostPrompt(map[string]any{
		"conversation_type": "chat",
		"conversation_id":   "conv-9",
		"post_prompt_data":  map[string]any{"raw": fenced},
		"raw_messages": []any{
			map[string]any{"role": "user", "content": "hi"},
			map[string]any{"role": "assistant", "content": fenced},
		},
	})
	if chat.Medium != "chat" || chat.ConversationID == nil || *chat.ConversationID != "conv-9" ||
		!reflect.DeepEqual(chat.Summary["already_answered"], []any{"pricing"}) ||
		!reflect.DeepEqual(chat.Dialogue, []map[string]string{{"role": "user", "content": "hi"}}) {
		t.Errorf("chat = %#v", chat)
	}
	if got := NormalizePostPrompt(map[string]any{"call_log": []any{map[string]any{"role": "user", "content": "hi"}}}); len(got.Dialogue) != 1 {
		t.Errorf("call_log key: %#v", got)
	}
	for _, junk := range []any{nil, "text", 42, []any{}} {
		got := NormalizePostPrompt(junk)
		if got.Medium != "" || len(got.Summary) != 0 || len(got.Dialogue) != 0 {
			t.Errorf("junk %#v: %#v", junk, got)
		}
	}
	body := map[string]any{"conversation_type": "voice", "extra": "kept"}
	if got := NormalizePostPrompt(body); got.Raw["extra"] != "kept" {
		t.Errorf("raw not preserved: %#v", got.Raw)
	}
}
