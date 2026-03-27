// Example: skills_demo
//
// Skills integration using the built-in skills registry. Demonstrates
// listing available skills, instantiating them via factory functions,
// loading them through the SkillManager, and registering their tools
// with an agent.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"

	// Import builtin skills so their init() functions register them
	_ "github.com/signalwire/signalwire-go/pkg/skills/builtin"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("SkillsDemo"),
		agent.WithRoute("/skills"),
		agent.WithPort(3006),
	)

	a.SetPromptText(
		"You are a versatile assistant with multiple skills including " +
			"date/time information, math calculations, and joke telling.",
	)

	// List all registered skills from the global registry
	fmt.Println("Available skills:")
	for _, name := range skills.ListSkills() {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Println()

	// Create a skill manager to handle lifecycle
	manager := skills.NewSkillManager()

	// Load skills by creating instances through the registry
	skillConfigs := map[string]map[string]any{
		"datetime": {"timezone": "America/New_York"},
		"math":     {},
		"joke":     {},
	}

	for skillName, params := range skillConfigs {
		factory := skills.GetSkillFactory(skillName)
		if factory == nil {
			fmt.Printf("Skill %q not found in registry, skipping\n", skillName)
			continue
		}

		// Create the skill instance
		skill := factory(params)

		// Load it via the manager (validates env vars, calls Setup)
		ok, errMsg := manager.LoadSkill(skill)
		if !ok {
			fmt.Printf("Failed to load skill %q: %s\n", skillName, errMsg)
			continue
		}

		fmt.Printf("Loaded skill: %s v%s - %s\n", skill.Name(), skill.Version(), skill.Description())

		// Register the skill's tools with the agent
		for _, toolReg := range skill.RegisterTools() {
			// Wrap the swaig.ToolHandler as an agent.ToolHandler (same signature,
			// different named types).
			handler := toolReg.Handler
			a.DefineTool(agent.ToolDefinition{
				Name:        toolReg.Name,
				Description: toolReg.Description,
				Parameters:  toolReg.Parameters,
				Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
					return handler(args, rawData)
				},
				Secure:      toolReg.Secure,
				Fillers:     toolReg.Fillers,
				SwaigFields: toolReg.SwaigFields,
			})
			fmt.Printf("  Registered tool: %s\n", toolReg.Name)
		}

		// Add any speech hints from the skill
		hints := skill.GetHints()
		if len(hints) > 0 {
			a.AddHints(hints)
		}

		// Add any prompt sections from the skill
		for _, section := range skill.GetPromptSections() {
			title, _ := section["title"].(string)
			body, _ := section["body"].(string)
			var bullets []string
			if b, ok := section["bullets"].([]string); ok {
				bullets = b
			}
			a.PromptAddSection(title, body, bullets)
		}

		// Merge any global data from the skill
		if gd := skill.GetGlobalData(); gd != nil {
			a.UpdateGlobalData(gd)
		}
	}

	fmt.Printf("\nLoaded skills: %v\n", manager.ListLoadedSkills())
	fmt.Println("\nStarting SkillsDemo on :3006/skills ...")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
