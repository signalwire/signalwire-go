package builtin

import (
	"fmt"
	"regexp"

	"github.com/signalwire/signalwire-go/v3/pkg/skills"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
)

// validKeyPattern matches keys composed solely of alphanumeric characters, underscores, and hyphens.
var validKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// PlayBackgroundFileSkill plays audio files in the background.
type PlayBackgroundFileSkill struct {
	skills.BaseSkill
	toolName string
	files    []map[string]any
}

// NewPlayBackgroundFile creates a new PlayBackgroundFileSkill.
func NewPlayBackgroundFile(params map[string]any) skills.SkillBase {
	return &PlayBackgroundFileSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "play_background_file",
			SkillDesc: "Control background file playback",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

// SupportsMultipleInstances returns true: several playback sets can coexist on
// one agent, each under its own "tool_name".
func (s *PlayBackgroundFileSkill) SupportsMultipleInstances() bool { return true }

// GetInstanceKey returns "play_background_file_<tool_name>", defaulting
// tool_name to "play_background_file".
func (s *PlayBackgroundFileSkill) GetInstanceKey() string {
	name := s.GetParamString("tool_name", "play_background_file")
	return "play_background_file_" + name
}

// Setup reads the required "files" param, a non-empty []any of file
// configurations. Per entry, a non-map, or one missing "key" or "url", is
// skipped; but an entry that HAS both and then fails validation FAILS the whole
// load — that is, an empty "description", a "key" containing anything outside
// [A-Za-z0-9_-], or a "wait" field that is not a bool. The key charset is
// enforced because the key is concatenated into the tool's "start_<key>" enum
// values. Setup returns false when "files" is absent, not a []any, empty, or
// when every entry was skipped.
func (s *PlayBackgroundFileSkill) Setup() bool {
	s.toolName = s.GetParamString("tool_name", "play_background_file")

	filesRaw, ok := s.Params["files"]
	if !ok {
		return false
	}
	filesSlice, ok := filesRaw.([]any)
	if !ok || len(filesSlice) == 0 {
		return false
	}

	s.files = make([]map[string]any, 0, len(filesSlice))
	for _, f := range filesSlice {
		m, ok := f.(map[string]any)
		if !ok {
			continue
		}
		key, _ := m["key"].(string)
		url, _ := m["url"].(string)
		description, _ := m["description"].(string)
		if key == "" || url == "" {
			continue
		}
		// Validate description is a non-empty string (matches Python _validate_config).
		if description == "" {
			return false
		}
		// Validate key format: alphanumeric, underscores, hyphens only.
		if !validKeyPattern.MatchString(key) {
			return false
		}
		// Validate optional wait field is a bool when present.
		if waitRaw, exists := m["wait"]; exists {
			if _, isBool := waitRaw.(bool); !isBool {
				return false
			}
		}
		s.files = append(s.files, m)
	}

	return len(s.files) > 0
}

// RegisterTools returns one playback-control tool whose required "action"
// argument is enum-constrained to "start_<key>" for each configured file plus
// "stop"; each option's description is folded into the parameter description so
// the model can choose by content. The handler emits the SWAIG
// play_background_file action (with per-file "wait", and post-process set) or
// stop_background_file. wait_for_fillers and skip_fillers are set on the tool,
// matching the reference, so filler audio does not overlap the playback.
func (s *PlayBackgroundFileSkill) RegisterTools() []skills.ToolRegistration {
	// Build enum values and dynamic action description matching Python get_tools() behavior.
	enumVals := make([]string, 0, len(s.files)+1)
	descriptions := make([]string, 0, len(s.files)+1)
	for _, f := range s.files {
		key, _ := f["key"].(string)
		desc, _ := f["description"].(string)
		enumVals = append(enumVals, "start_"+key)
		descriptions = append(descriptions, "start_"+key+": "+desc)
	}
	enumVals = append(enumVals, "stop")
	descriptions = append(descriptions, "stop: Stop any currently playing background file")

	actionDesc := "Action to perform. Options: "
	for i, d := range descriptions {
		if i > 0 {
			actionDesc += "; "
		}
		actionDesc += d
	}

	return []skills.ToolRegistration{
		{
			Name:        s.toolName,
			Description: fmt.Sprintf("Control background file playback for %s", s.toolName),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]any{
						"type":        "string",
						"description": actionDesc,
						"enum":        enumVals,
					},
				},
				"required": []string{"action"},
			},
			Handler: s.handlePlayback,
			// Python sets wait_for_fillers=True and skip_fillers=True on every tool dict.
			SwaigFields: map[string]any{
				"wait_for_fillers": true,
				"skip_fillers":     true,
			},
		},
	}
}

func (s *PlayBackgroundFileSkill) handlePlayback(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	action, _ := args["action"].(string)

	if action == "stop" {
		return swaig.NewFunctionResult("Stopping background file playback.").
			StopBackgroundFile()
	}

	// Look for a matching file
	for _, f := range s.files {
		key, _ := f["key"].(string)
		if action == "start_"+key {
			fileURL, _ := f["url"].(string)
			description, _ := f["description"].(string)
			wait, _ := f["wait"].(bool)

			result := swaig.NewFunctionResult(
				fmt.Sprintf("Now playing %s for you.", description),
			)
			result.SetPostProcess(true)
			result.PlayBackgroundFile(fileURL, wait)
			return result
		}
	}

	return swaig.NewFunctionResult("Unknown action. Use start_<key> or stop.")
}

// GetParameterSchema extends the common skill parameters with the required
// "files" array, each item requiring key, description and url, with an optional
// boolean "wait" defaulting to false.
func (s *PlayBackgroundFileSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["files"] = map[string]any{
		"type":        "array",
		"description": "Array of file configurations to make available for playback",
		"required":    true,
		// items sub-schema matches Python get_parameter_schema() (skill.py:60-84).
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key": map[string]any{
					"type":        "string",
					"description": "Unique identifier for the file",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "Human-readable description of the file",
				},
				"url": map[string]any{
					"type":        "string",
					"description": "URL of the audio/video file to play",
				},
				"wait": map[string]any{
					"type":        "boolean",
					"description": "Whether to wait for file to finish playing",
					"default":     false,
				},
			},
			"required": []string{"key", "description", "url"},
		},
	}
	return schema
}

func init() {
	skills.RegisterSkill("play_background_file", NewPlayBackgroundFile)
}
