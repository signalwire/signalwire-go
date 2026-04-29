package prefabs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// SurveyQuestion describes a single question in a survey.
//
// Prefer NewSurveyQuestion for construction — it defaults Required:true to
// match Python SurveyAgent behavior (signalwire/prefabs/survey.py _validate_questions
// sets required=True when unspecified). Struct literals are still supported,
// but the Go zero value for Required is false, which diverges from Python.
type SurveyQuestion struct {
	ID       string   // unique question identifier
	Text     string   // the question to ask
	Type     string   // "rating", "multiple_choice", "yes_no", "open_ended"
	Scale    int      // 1..Scale for rating questions (default 5)
	Choices  []string // options for multiple_choice questions
	Required bool     // whether a non-empty answer is required — Python default true
}

// SurveyQuestionOption configures a question during construction.
type SurveyQuestionOption func(*SurveyQuestion)

// WithQuestionID sets the question ID.
func WithQuestionID(id string) SurveyQuestionOption {
	return func(q *SurveyQuestion) { q.ID = id }
}

// WithQuestionType sets the question type ("rating", "multiple_choice",
// "yes_no", "open_ended").
func WithQuestionType(t string) SurveyQuestionOption {
	return func(q *SurveyQuestion) { q.Type = t }
}

// WithQuestionScale sets the scale for rating questions (answers run 1..n).
func WithQuestionScale(n int) SurveyQuestionOption {
	return func(q *SurveyQuestion) { q.Scale = n }
}

// WithQuestionChoices sets the choice list for multiple_choice questions.
func WithQuestionChoices(choices ...string) SurveyQuestionOption {
	return func(q *SurveyQuestion) { q.Choices = choices }
}

// WithOptional marks a question as not required. Matches Python's
// required=False escape hatch on SurveyAgent questions.
func WithOptional() SurveyQuestionOption {
	return func(q *SurveyQuestion) { q.Required = false }
}

// NewSurveyQuestion constructs a SurveyQuestion with Required:true, matching
// Python SurveyAgent._validate_questions which defaults required=True when
// unspecified. Callers opt out with WithOptional().
func NewSurveyQuestion(text string, opts ...SurveyQuestionOption) SurveyQuestion {
	q := SurveyQuestion{
		Text:     text,
		Required: true,
	}
	for _, opt := range opts {
		opt(&q)
	}
	return q
}

// SurveyOptions configures a new SurveyAgent.
type SurveyOptions struct {
	Name       string
	Route      string
	SurveyName string
	BrandName  string
	Questions  []SurveyQuestion
	MaxRetries int
	Intro      string
	Conclusion string
}

// SurveyAgent conducts structured surveys with typed questions.
type SurveyAgent struct {
	*agent.AgentBase
	questions    []SurveyQuestion
	surveyName   string
	brandName    string
	maxRetries   int
	introduction string
	conclusion   string
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// NewSurveyAgent creates an agent that conducts a structured survey.
func NewSurveyAgent(opts SurveyOptions) *SurveyAgent {
	name := opts.Name
	if name == "" {
		name = "survey"
	}
	route := opts.Route
	if route == "" {
		route = "/survey"
	}
	brandName := opts.BrandName
	if brandName == "" {
		brandName = "Our Company"
	}
	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 2
	}
	intro := opts.Intro
	if intro == "" {
		intro = fmt.Sprintf("Welcome to our %s. We appreciate your participation.", opts.SurveyName)
	}
	conclusion := opts.Conclusion
	if conclusion == "" {
		conclusion = "Thank you for completing our survey. Your feedback is valuable to us."
	}

	// Normalise rating scales. Required default is applied at construction
	// time by NewSurveyQuestion (matches Python SurveyAgent._validate_questions
	// defaulting required=True); callers using struct literals pick up Go's
	// zero-value false instead.
	for i := range opts.Questions {
		if opts.Questions[i].Type == "rating" && opts.Questions[i].Scale <= 0 {
			opts.Questions[i].Scale = 5
		}
	}

	base := agent.NewAgentBase(
		agent.WithName(name),
		agent.WithRoute(route),
	)

	sa := &SurveyAgent{
		AgentBase:    base,
		questions:    opts.Questions,
		surveyName:   opts.SurveyName,
		brandName:    brandName,
		maxRetries:   maxRetries,
		introduction: intro,
		conclusion:   conclusion,
	}

	// ---- Prompt ----
	base.PromptAddSection("Personality",
		fmt.Sprintf("You are a friendly and professional survey agent representing %s.", brandName),
		nil,
	)
	base.PromptAddSection("Goal",
		fmt.Sprintf("Conduct the '%s' survey by asking questions and collecting responses.", opts.SurveyName),
		nil,
	)
	base.PromptAddSection("Instructions", "", []string{
		"Guide the user through each survey question in sequence.",
		"Ask only one question at a time and wait for a response.",
		"For rating questions, explain the scale (e.g., 1-5, where 5 is best).",
		"For multiple choice questions, list all the options.",
		fmt.Sprintf("If a response is invalid, explain and retry up to %d times.", maxRetries),
		"Be conversational but stay focused on collecting the survey data.",
		"After all questions are answered, thank the user for their participation.",
	})
	base.PromptAddSection("Introduction",
		fmt.Sprintf("Begin with this introduction: %s", intro),
		nil,
	)

	// ---- Survey Questions prompt section ----
	base.PromptAddSection("Survey Questions", "Ask these questions in order:", nil)
	for _, q := range opts.Questions {
		body := fmt.Sprintf("ID: %s\nType: %s\nRequired: %v", q.ID, q.Type, q.Required)
		if q.Type == "rating" {
			body += fmt.Sprintf("\nScale: 1-%d", q.Scale)
		}
		if q.Type == "multiple_choice" && len(q.Choices) > 0 {
			body += fmt.Sprintf("\nOptions: %s", strings.Join(q.Choices, ", "))
		}
		base.PromptAddSubsection("Survey Questions", q.Text, body, nil)
	}

	base.PromptAddSection("Conclusion",
		fmt.Sprintf("End with this conclusion: %s", conclusion),
		nil,
	)

	// ---- Post-prompt for JSON summary ----
	base.SetPostPrompt(`Return a JSON summary of the survey responses:
{
    "survey_name": "SURVEY_NAME",
    "responses": {
        "QUESTION_ID_1": "RESPONSE_1",
        "QUESTION_ID_2": "RESPONSE_2"
    },
    "completion_status": "complete/incomplete",
    "timestamp": "CURRENT_TIMESTAMP"
}`)

	// ---- Hints ----
	hints := []string{opts.SurveyName, brandName}
	for _, q := range opts.Questions {
		switch q.Type {
		case "rating":
			for i := 1; i <= q.Scale; i++ {
				hints = append(hints, strconv.Itoa(i))
			}
		case "multiple_choice":
			hints = append(hints, q.Choices...)
		case "yes_no":
			hints = append(hints, "yes", "no")
		}
	}
	base.AddHints(hints)

	// ---- AI behavior parameters ----
	base.SetParams(map[string]any{
		"wait_for_user":              false,
		"end_of_speech_timeout":      1500,
		"ai_volume":                  5,
		"static_greeting":            intro,
		"static_greeting_no_barge":   true,
	})

	// ---- Global data ----
	questionMaps := make([]map[string]any, len(opts.Questions))
	for i, q := range opts.Questions {
		m := map[string]any{
			"id":   q.ID,
			"text": q.Text,
			"type": q.Type,
		}
		if q.Type == "rating" {
			m["scale"] = q.Scale
		}
		if q.Type == "multiple_choice" && len(q.Choices) > 0 {
			m["choices"] = q.Choices
		}
		questionMaps[i] = m
	}
	base.SetGlobalData(map[string]any{
		"survey_name": opts.SurveyName,
		"brand_name":  brandName,
		"questions":   questionMaps,
		"max_retries": maxRetries,
	})

	// ---- Native functions ----
	base.SetNativeFunctions([]string{"check_time"})

	// ---- Tools ----
	sa.registerTools()

	// ---- Summary callback ----
	base.OnSummary(func(summary map[string]any, rawData map[string]any) {
		if summary != nil {
			if data, err := json.Marshal(summary); err == nil {
				base.Logger.Info("Survey completed: %s", string(data))
			} else {
				base.Logger.Info("Survey summary (unstructured): %v", summary)
			}
		}
	})

	return sa
}

// ---------------------------------------------------------------------------
// Tool registration
// ---------------------------------------------------------------------------

func (sa *SurveyAgent) registerTools() {
	// validate_response ------------------------------------------------
	sa.DefineTool(agent.ToolDefinition{
		Name:        "validate_response",
		Description: "Validate if a response meets the requirements for a specific question",
		Parameters: map[string]any{
			"question_id": map[string]any{
				"type":        "string",
				"description": "The ID of the question",
			},
			"response": map[string]any{
				"type":        "string",
				"description": "The user's response to validate",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			qID, _ := args["question_id"].(string)
			response, _ := args["response"].(string)

			var question *SurveyQuestion
			for i := range sa.questions {
				if sa.questions[i].ID == qID {
					question = &sa.questions[i]
					break
				}
			}
			if question == nil {
				return swaig.NewFunctionResult(fmt.Sprintf("Error: Question with ID '%s' not found.", qID))
			}

			switch question.Type {
			case "rating":
				rating, err := strconv.Atoi(strings.TrimSpace(response))
				if err != nil || rating < 1 || rating > question.Scale {
					return swaig.NewFunctionResult(
						fmt.Sprintf("Invalid rating. Please provide a number between 1 and %d.", question.Scale),
					)
				}
			case "multiple_choice":
				found := false
				respLower := strings.TrimSpace(strings.ToLower(response))
				for _, c := range question.Choices {
					if strings.ToLower(c) == respLower {
						found = true
						break
					}
				}
				if !found {
					return swaig.NewFunctionResult(
						fmt.Sprintf("Invalid choice. Please select one of: %s.", strings.Join(question.Choices, ", ")),
					)
				}
			case "yes_no":
				r := strings.TrimSpace(strings.ToLower(response))
				if r != "yes" && r != "no" && r != "y" && r != "n" {
					return swaig.NewFunctionResult("Please answer with 'yes' or 'no'.")
				}
			case "open_ended":
				if strings.TrimSpace(response) == "" && question.Required {
					return swaig.NewFunctionResult("A response is required for this question.")
				}
			}

			return swaig.NewFunctionResult(fmt.Sprintf("Response to '%s' is valid.", qID))
		},
	})

	// log_response -----------------------------------------------------
	sa.DefineTool(agent.ToolDefinition{
		Name:        "log_response",
		Description: "Log a validated response to a survey question",
		Parameters: map[string]any{
			"question_id": map[string]any{
				"type":        "string",
				"description": "The ID of the question",
			},
			"response": map[string]any{
				"type":        "string",
				"description": "The user's validated response",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			qID, _ := args["question_id"].(string)

			// Find the question text for a more informative message
			var questionText string
			for _, q := range sa.questions {
				if q.ID == qID {
					questionText = q.Text
					break
				}
			}
			if questionText == "" {
				questionText = qID
			}

			return swaig.NewFunctionResult(fmt.Sprintf("Response to '%s' has been recorded.", questionText))
		},
	})
}
