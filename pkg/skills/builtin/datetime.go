package builtin

import (
	"fmt"
	"time"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// DateTimeSkill provides current date and time information.
type DateTimeSkill struct {
	skills.BaseSkill
	timezone string
}

// NewDateTime creates a new DateTimeSkill.
func NewDateTime(params map[string]any) skills.SkillBase {
	s := &DateTimeSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "datetime",
			SkillDesc: "Get current date, time, and timezone information",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
	return s
}

func (s *DateTimeSkill) Setup() bool {
	s.timezone = s.GetParamString("timezone", "UTC")
	return true
}

func (s *DateTimeSkill) RegisterTools() []skills.ToolRegistration {
	return []skills.ToolRegistration{
		{
			Name:        "get_datetime",
			Description: "Get the current date and time, optionally in a specific timezone",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"timezone": map[string]any{
						"type":        "string",
						"description": "Timezone name (e.g., 'America/New_York', 'Europe/London'). Defaults to UTC.",
					},
				},
			},
			Handler: s.handleGetDateTime,
		},
	}
}

func (s *DateTimeSkill) handleGetDateTime(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	tzName := "UTC"
	if v, ok := args["timezone"].(string); ok && v != "" {
		tzName = v
	} else if s.timezone != "" {
		tzName = s.timezone
	}

	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return swaig.NewFunctionResult(fmt.Sprintf("Error: invalid timezone '%s'", tzName))
	}

	now := time.Now().In(loc)
	dateStr := now.Format("Monday, January 02, 2006")
	timeStr := now.Format("03:04:05 PM MST")

	return swaig.NewFunctionResult(fmt.Sprintf("The current date is %s and the time is %s", dateStr, timeStr))
}

func (s *DateTimeSkill) GetHints() []string {
	return []string{"time", "date", "today", "now", "current", "timezone"}
}

func (s *DateTimeSkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Date and Time Information",
			"body":  "You can provide current date and time information.",
			"bullets": []string{
				"Use get_datetime to tell users the current date and time",
				"Supports different timezones",
			},
		},
	}
}

func init() {
	skills.RegisterSkill("datetime", NewDateTime)
}
