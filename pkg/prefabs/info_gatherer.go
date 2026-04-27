// Package prefabs provides pre-built agent patterns that extend AgentBase with
// common conversational workflows such as information gathering, surveys,
// reception/routing, FAQ answering, and virtual concierge services.
package prefabs

import (
	"fmt"
	"strings"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// Question describes a single question in an InfoGatherer sequence.
type Question struct {
	KeyName      string // identifier used to store the answer
	QuestionText string // the question to ask the user
	Confirm      bool   // if true the agent will confirm before accepting
}

// InfoGathererOptions configures a new InfoGathererAgent.
// Set Questions to nil to enable dynamic callback mode via SetQuestionCallback.
type InfoGathererOptions struct {
	Name      string
	Route     string
	Questions *[]Question // nil enables dynamic callback mode; non-nil is static mode
}

// InfoGathererAgent collects answers to a series of questions sequentially.
// Supports both static (questions provided at construction) and dynamic
// (questions determined per-request via SetQuestionCallback) modes.
type InfoGathererAgent struct {
	*agent.AgentBase
	staticQuestions *[]Question
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// NewInfoGathererAgent creates an agent that asks a series of questions and
// stores the answers in global data. Pass nil Questions to enable dynamic
// mode; call SetQuestionCallback on the returned agent to supply per-request
// questions.
func NewInfoGathererAgent(opts InfoGathererOptions) *InfoGathererAgent {
	name := opts.Name
	if name == "" {
		name = "info_gatherer"
	}
	route := opts.Route
	if route == "" {
		route = "/info_gatherer"
	}

	base := agent.NewAgentBase(
		agent.WithName(name),
		agent.WithRoute(route),
	)

	ig := &InfoGathererAgent{
		AgentBase:       base,
		staticQuestions: opts.Questions,
	}

	// ---- Prompt ----
	base.PromptAddSection(
		"Personality",
		"You are a helpful assistant whose job is to collect information by asking questions.",
		nil,
	)
	base.PromptAddSection(
		"Instructions",
		"",
		[]string{
			"Ask the user if they are ready to answer some questions.",
			"When they confirm, call the start_questions function.",
			"Ask only one question at a time.",
			"Wait for the user's answer before moving on.",
			"If confirmation is required for a question, repeat the answer back and ask the user to confirm before submitting.",
			"Use submit_answer to record each answer and receive the next question.",
		},
	)

	// ---- Global data (static mode only) ----
	if opts.Questions != nil {
		questions := *opts.Questions
		questionMaps := make([]map[string]any, len(questions))
		for i, q := range questions {
			questionMaps[i] = map[string]any{
				"key_name":      q.KeyName,
				"question_text": q.QuestionText,
				"confirm":       q.Confirm,
			}
		}
		base.SetGlobalData(map[string]any{
			"questions":      questionMaps,
			"question_index": 0,
			"answers":        []any{},
		})
	}
	// Dynamic mode: global_data will be set per-request by the dynamic config
	// callback registered via SetQuestionCallback.

	// ---- Tools ----
	ig.registerTools()

	return ig
}

// ---------------------------------------------------------------------------
// Dynamic question callback
// ---------------------------------------------------------------------------

// SetQuestionCallback registers a per-request callback that returns the list
// of questions to ask. Calling this method enables dynamic mode: on each
// SWML request the callback is invoked with the request's query parameters,
// body parameters, and headers; the returned []Question becomes the session's
// question list. This mirrors Python's InfoGathererAgent.set_question_callback.
//
// If Questions was set to nil in InfoGathererOptions (dynamic mode), a
// fallback question set is used when no callback is registered.
//
// Example:
//
//	ig.SetQuestionCallback(func(query, body map[string]any, headers map[string]string) []Question {
//	    if body["department"] == "support" {
//	        return []Question{{KeyName: "issue", QuestionText: "What is the issue?"}}
//	    }
//	    return []Question{{KeyName: "name", QuestionText: "What is your name?"}}
//	})
func (ig *InfoGathererAgent) SetQuestionCallback(
	cb func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string) []Question,
) {
	ig.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, a *agent.AgentBase) {
		var questions []Question

		func() {
			defer func() {
				if r := recover(); r != nil {
					// Callback panicked — fall back to safe defaults
					questions = fallbackQuestions()
				}
			}()
			questions = cb(queryParams, bodyParams, headers)
			if len(questions) == 0 {
				questions = fallbackQuestions()
			}
		}()

		questionMaps := make([]map[string]any, len(questions))
		for i, q := range questions {
			questionMaps[i] = map[string]any{
				"key_name":      q.KeyName,
				"question_text": q.QuestionText,
				"confirm":       q.Confirm,
			}
		}
		a.SetGlobalData(map[string]any{
			"questions":      questionMaps,
			"question_index": 0,
			"answers":        []any{},
		})
	})
}

// fallbackQuestions returns a minimal question set used when dynamic mode has
// no callback or the callback fails — mirrors Python's on_swml_request fallback.
func fallbackQuestions() []Question {
	return []Question{
		{KeyName: "name", QuestionText: "What is your name?"},
		{KeyName: "message", QuestionText: "How can I help you today?"},
	}
}

// ---------------------------------------------------------------------------
// Tool registration
// ---------------------------------------------------------------------------

func (ig *InfoGathererAgent) registerTools() {
	// start_questions --------------------------------------------------
	ig.DefineTool(agent.ToolDefinition{
		Name:        "start_questions",
		Description: "Start the question sequence with the first question",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			globalData, _ := rawData["global_data"].(map[string]any)
			questions, _ := globalData["questions"].([]any)
			if len(questions) == 0 {
				return swaig.NewFunctionResult("I don't have any questions to ask.")
			}

			first, _ := questions[0].(map[string]any)
			text, _ := first["question_text"].(string)
			confirm, _ := first["confirm"].(bool)

			instruction := ig.buildQuestionInstruction(text, confirm, true)
			result := swaig.NewFunctionResult(instruction)
			result.ReplaceInHistory("Welcome! Let me ask you a few questions.")
			return result
		},
	})

	// submit_answer ----------------------------------------------------
	// key_name is NOT a model-supplied parameter; it is derived server-side
	// from global_data["questions"][question_index]["key_name"], matching
	// Python's submit_answer behavior.
	ig.DefineTool(agent.ToolDefinition{
		Name:        "submit_answer",
		Description: "Submit an answer to the current question and move to the next one",
		Parameters: map[string]any{
			"answer": map[string]any{
				"type":        "string",
				"description": "The user's answer to the current question",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			answer, _ := args["answer"].(string)

			globalData, _ := rawData["global_data"].(map[string]any)
			questions, _ := globalData["questions"].([]any)
			idxFloat, _ := globalData["question_index"].(float64)
			idx := int(idxFloat)
			answers, _ := globalData["answers"].([]any)

			if idx >= len(questions) {
				return swaig.NewFunctionResult("All questions have already been answered.")
			}

			// Derive key_name from global_data, not from model-supplied args.
			current, _ := questions[idx].(map[string]any)
			keyName, _ := current["key_name"].(string)

			// Record the answer
			newAnswer := map[string]any{"key_name": keyName, "answer": answer}
			newAnswers := append(answers, newAnswer)
			newIdx := idx + 1

			if newIdx < len(questions) {
				next, _ := questions[newIdx].(map[string]any)
				nextText, _ := next["question_text"].(string)
				nextConfirm, _ := next["confirm"].(bool)

				instruction := ig.buildQuestionInstruction(nextText, nextConfirm, false)
				result := swaig.NewFunctionResult(instruction)
				result.ReplaceInHistory(true)
				result.UpdateGlobalData(map[string]any{
					"answers":        newAnswers,
					"question_index": newIdx,
				})
				return result
			}

			// All questions answered
			result := swaig.NewFunctionResult(
				"Thank you! All questions have been answered. You can now summarize the information collected or ask if there's anything else the user would like to discuss.",
			)
			result.ReplaceInHistory(true)
			result.UpdateGlobalData(map[string]any{
				"answers":        newAnswers,
				"question_index": newIdx,
			})
			return result
		},
	})
}

// buildQuestionInstruction generates the prompt text for a question.
func (ig *InfoGathererAgent) buildQuestionInstruction(questionText string, needsConfirm bool, isFirst bool) string {
	var sb strings.Builder
	if isFirst {
		fmt.Fprintf(&sb, "Ask the user to answer the following question: %s\n\n", questionText)
	} else {
		fmt.Fprintf(&sb, "Previous answer recorded. Now ask the user to answer the following question: %s\n\n", questionText)
	}
	sb.WriteString("Make sure the answer fits the scope and context of the question before submitting it. ")
	if needsConfirm {
		sb.WriteString("Insist that the user confirms the answer as many times as needed until they say it is correct.")
	} else {
		sb.WriteString("You don't need the user to confirm the answer to this question.")
	}
	return sb.String()
}
