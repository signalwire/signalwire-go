package builtin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// Valid trivia categories from API Ninjas.
var validTriviaCategories = map[string]string{
	"artliterature":    "Art and Literature",
	"language":         "Language",
	"sciencenature":    "Science and Nature",
	"general":          "General Knowledge",
	"fooddrink":        "Food and Drink",
	"peopleplaces":     "People and Places",
	"geography":        "Geography",
	"historyholidays":  "History and Holidays",
	"entertainment":    "Entertainment",
	"toysgames":        "Toys and Games",
	"music":            "Music",
	"mathematics":      "Mathematics",
	"religionmythology": "Religion and Mythology",
	"sportsleisure":    "Sports and Leisure",
}

// APINinjasTriviaSkill gets trivia questions from API Ninjas.
type APINinjasTriviaSkill struct {
	skills.BaseSkill
	apiKey     string
	toolName   string
	categories []string
}

// NewAPINinjasTrivia creates a new APINinjasTriviaSkill.
func NewAPINinjasTrivia(params map[string]any) skills.SkillBase {
	return &APINinjasTriviaSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "api_ninjas_trivia",
			SkillDesc: "Get trivia questions from API Ninjas",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

func (s *APINinjasTriviaSkill) RequiredEnvVars() []string {
	if s.Params != nil {
		if _, ok := s.Params["api_key"]; ok {
			return nil
		}
	}
	return []string{"API_NINJAS_KEY"}
}

func (s *APINinjasTriviaSkill) SupportsMultipleInstances() bool { return true }

func (s *APINinjasTriviaSkill) GetInstanceKey() string {
	name := s.GetParamString("tool_name", "get_trivia")
	return "api_ninjas_trivia_" + name
}

func (s *APINinjasTriviaSkill) Setup() bool {
	s.apiKey = s.GetParamString("api_key", os.Getenv("API_NINJAS_KEY"))
	if s.apiKey == "" {
		return false
	}
	s.toolName = s.GetParamString("tool_name", "get_trivia")

	// Read categories param; default to all valid category keys
	if v, ok := s.Params["categories"]; ok {
		switch raw := v.(type) {
		case []string:
			s.categories = raw
		case []any:
			cats := make([]string, 0, len(raw))
			for _, item := range raw {
				str, ok := item.(string)
				if !ok {
					return false
				}
				cats = append(cats, str)
			}
			s.categories = cats
		default:
			return false
		}
	} else {
		// Default: all valid categories
		cats := make([]string, 0, len(validTriviaCategories))
		for k := range validTriviaCategories {
			cats = append(cats, k)
		}
		s.categories = cats
	}

	// Validate categories is non-empty and each entry is a valid category key
	if len(s.categories) == 0 {
		return false
	}
	for _, cat := range s.categories {
		if _, valid := validTriviaCategories[cat]; !valid {
			return false
		}
	}

	return true
}

func (s *APINinjasTriviaSkill) RegisterTools() []skills.ToolRegistration {
	// Build rich description matching Python: "Category for trivia question. Options: key: Name; ..."
	descriptions := make([]string, 0, len(s.categories))
	for _, cat := range s.categories {
		if name, ok := validTriviaCategories[cat]; ok {
			descriptions = append(descriptions, cat+": "+name)
		}
	}
	categoryDesc := "Category for trivia question. Options: " + strings.Join(descriptions, "; ")

	return []skills.ToolRegistration{
		{
			Name:        s.toolName,
			Description: "Get a trivia question from a category",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"category": map[string]any{
						"type":        "string",
						"description": categoryDesc,
						"enum":        s.categories,
					},
				},
				"required": []string{"category"},
			},
			Handler: s.handleGetTrivia,
		},
	}
}

func (s *APINinjasTriviaSkill) handleGetTrivia(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	category, _ := args["category"].(string)
	if category == "" {
		category = "general"
	}

	apiURL := fmt.Sprintf("https://api.api-ninjas.com/v1/trivia?category=%s", category)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return swaig.NewFunctionResult("Error creating trivia request.")
	}
	req.Header.Set("X-Api-Key", s.apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return swaig.NewFunctionResult("Sorry, I cannot get trivia questions right now.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return swaig.NewFunctionResult("Sorry, I cannot get trivia questions right now. Please try again later.")
	}

	var results []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil || len(results) == 0 {
		return swaig.NewFunctionResult("No trivia questions available right now.")
	}

	q := results[0]
	question, _ := q["question"].(string)
	answer, _ := q["answer"].(string)
	cat, _ := q["category"].(string)

	return swaig.NewFunctionResult(
		fmt.Sprintf("Category: %s\nQuestion: %s\nAnswer: %s\n\nBe sure to give the user time to answer before revealing the answer.", cat, question, answer),
	)
}

func (s *APINinjasTriviaSkill) GetHints() []string {
	return []string{"trivia", "quiz", "question", "game"}
}

func (s *APINinjasTriviaSkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Trivia Questions",
			"body":  "You can get trivia questions to play with users.",
			"bullets": []string{
				"Use " + s.toolName + " to get trivia questions from various categories",
				"Give the user time to answer before revealing the answer",
			},
		},
	}
}

func (s *APINinjasTriviaSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["api_key"] = map[string]any{
		"type":        "string",
		"description": "API Ninjas API key",
		"required":    true,
		"hidden":      true,
		"env_var":     "API_NINJAS_KEY",
	}

	allKeys := make([]string, 0, len(validTriviaCategories))
	for k := range validTriviaCategories {
		allKeys = append(allKeys, k)
	}
	schema["categories"] = map[string]any{
		"type":        "array",
		"description": "List of trivia categories to enable.",
		"required":    false,
		"items": map[string]any{
			"type": "string",
			"enum": allKeys,
		},
	}
	return schema
}

func init() {
	skills.RegisterSkill("api_ninjas_trivia", NewAPINinjasTrivia)
}
