package builtin

import (
	"math/rand"

	"github.com/signalwire/signalwire-go/v3/pkg/skills"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
)

var builtinJokes = []string{
	"Why do programmers prefer dark mode? Because light attracts bugs!",
	"Why was the JavaScript developer sad? Because he didn't Node how to Express himself.",
	"What do you call a fake noodle? An impasta!",
	"Why don't scientists trust atoms? Because they make up everything!",
	"What did the ocean say to the beach? Nothing, it just waved.",
	"Why did the scarecrow win an award? Because he was outstanding in his field!",
	"What do you call a bear with no teeth? A gummy bear!",
	"Why did the bicycle fall over? Because it was two-tired!",
	"What do you call a fish without eyes? A fsh!",
	"Why don't eggs tell jokes? They'd crack each other up!",
	"What do you call a sleeping dinosaur? A dino-snore!",
	"Why did the math book look so sad? Because it had too many problems.",
	"What do you call a dog that does magic tricks? A Labracadabrador!",
	"Why can't you give Elsa a balloon? Because she will let it go!",
	"What did one wall say to the other? I'll meet you at the corner!",
}

// JokeSkill tells random jokes from a built-in list.
type JokeSkill struct {
	skills.BaseSkill
	apiKey   string
	toolName string
}

// NewJoke creates a new JokeSkill.
func NewJoke(params map[string]any) skills.SkillBase {
	return &JokeSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "joke",
			SkillDesc: "Tell jokes from a built-in collection",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

// Setup records the optional "api_key" and the tool name (default "get_joke").
// It always succeeds — jokes come from the in-process builtinJokes list, so no
// credential is needed and api_key is currently unused by the handler.
func (s *JokeSkill) Setup() bool {
	s.apiKey = s.GetParamString("api_key", "")
	s.toolName = s.GetParamString("tool_name", "get_joke")
	return true
}

// RegisterTools returns the single joke tool. Its required "type" argument is
// enum-constrained to "jokes" or "dadjokes", but the local handler ignores it
// and picks uniformly at random from the built-in list either way.
func (s *JokeSkill) RegisterTools() []skills.ToolRegistration {
	return []skills.ToolRegistration{
		{
			Name:        s.toolName,
			Description: "Tell a random joke",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type": map[string]any{
						"type":        "string",
						"description": "Type of joke to get",
						"enum":        []string{"jokes", "dadjokes"},
					},
				},
				"required": []string{"type"},
			},
			Handler: s.handleTellJoke,
		},
	}
}

func (s *JokeSkill) handleTellJoke(_ map[string]any, _ map[string]any) *swaig.FunctionResult {
	//nolint:gosec // G404: math/rand is fine here — picking a random joke to
	// tell, no security/crypto context.
	joke := builtinJokes[rand.Intn(len(builtinJokes))]
	return swaig.NewFunctionResult("Here's a joke: " + joke)
}

// GetGlobalData publishes joke_skill_enabled into the agent's global data so
// the prompt can tell the capability is present.
func (s *JokeSkill) GetGlobalData() map[string]any {
	return map[string]any{
		"joke_skill_enabled": true,
	}
}

// GetHints returns speech-recognition hints for humor vocabulary.
func (s *JokeSkill) GetHints() []string {
	return []string{"joke", "funny", "humor", "laugh"}
}

// GetPromptSections returns one POM section, naming the configured tool, that
// tells the agent to use it when a user asks for humor.
func (s *JokeSkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Joke Telling",
			"body":  "You can tell jokes to entertain users.",
			"bullets": []string{
				"Use " + s.toolName + " to tell jokes when users ask for humor",
				"You can tell regular jokes or dad jokes",
				"Be enthusiastic and fun when sharing jokes",
			},
		},
	}
}

func init() {
	skills.RegisterSkill("joke", NewJoke)
}
