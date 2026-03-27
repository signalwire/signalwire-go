package builtin

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

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

func (s *PlayBackgroundFileSkill) SupportsMultipleInstances() bool { return true }

func (s *PlayBackgroundFileSkill) GetInstanceKey() string {
	name := s.GetParamString("tool_name", "play_background_file")
	return "play_background_file_" + name
}

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
		if key == "" || url == "" {
			continue
		}
		s.files = append(s.files, m)
	}

	return len(s.files) > 0
}

func (s *PlayBackgroundFileSkill) RegisterTools() []skills.ToolRegistration {
	// Build enum values
	enumVals := make([]string, 0, len(s.files)+1)
	for _, f := range s.files {
		key, _ := f["key"].(string)
		enumVals = append(enumVals, "start_"+key)
	}
	enumVals = append(enumVals, "stop")

	return []skills.ToolRegistration{
		{
			Name:        s.toolName,
			Description: fmt.Sprintf("Control background file playback for %s", s.toolName),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]any{
						"type":        "string",
						"description": "Action to perform (start_<key> or stop)",
						"enum":        enumVals,
					},
				},
				"required": []string{"action"},
			},
			Handler: s.handlePlayback,
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

func (s *PlayBackgroundFileSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["files"] = map[string]any{
		"type":        "array",
		"description": "Array of file configurations for playback",
		"required":    true,
	}
	return schema
}

func init() {
	skills.RegisterSkill("play_background_file", NewPlayBackgroundFile)
}
