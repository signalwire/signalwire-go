package builtin

import (
	"math/rand"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
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

func (s *JokeSkill) Setup() bool {
	s.toolName = s.GetParamString("tool_name", "tell_joke")
	return true
}

func (s *JokeSkill) RegisterTools() []skills.ToolRegistration {
	return []skills.ToolRegistration{
		{
			Name:        s.toolName,
			Description: "Tell a random joke",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
			Handler: s.handleTellJoke,
		},
	}
}

func (s *JokeSkill) handleTellJoke(_ map[string]any, _ map[string]any) *swaig.FunctionResult {
	joke := builtinJokes[rand.Intn(len(builtinJokes))]
	return swaig.NewFunctionResult("Here's a joke: " + joke)
}

func (s *JokeSkill) GetHints() []string {
	return []string{"joke", "funny", "humor", "laugh"}
}

func (s *JokeSkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Joke Telling",
			"body":  "You can tell jokes to entertain users.",
			"bullets": []string{
				"Use " + s.toolName + " to tell jokes when users ask for humor",
				"Be enthusiastic and fun when sharing jokes",
			},
		},
	}
}

func init() {
	skills.RegisterSkill("joke", NewJoke)
}
