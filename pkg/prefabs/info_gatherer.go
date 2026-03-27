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
type InfoGathererOptions struct {
	Name      string
	Route     string
	Questions []Question
}

// InfoGathererAgent collects answers to a series of questions sequentially.
type InfoGathererAgent struct {
	*agent.AgentBase
	questions []Question
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// NewInfoGathererAgent creates an agent that asks a series of questions and
// stores the answers in global data.
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
		AgentBase: base,
		questions: opts.Questions,
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

	// ---- Global data ----
	questionMaps := make([]map[string]any, len(opts.Questions))
	for i, q := range opts.Questions {
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

	// ---- Tools ----
	ig.registerTools()

	return ig
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
	ig.DefineTool(agent.ToolDefinition{
		Name:        "submit_answer",
		Description: "Submit an answer to the current question and move to the next one",
		Parameters: map[string]any{
			"key_name": map[string]any{
				"type":        "string",
				"description": "The key name of the question being answered",
			},
			"answer": map[string]any{
				"type":        "string",
				"description": "The user's answer to the current question",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			answer, _ := args["answer"].(string)
			keyName, _ := args["key_name"].(string)

			globalData, _ := rawData["global_data"].(map[string]any)
			questions, _ := globalData["questions"].([]any)
			idxFloat, _ := globalData["question_index"].(float64)
			idx := int(idxFloat)
			answers, _ := globalData["answers"].([]any)

			if idx >= len(questions) {
				return swaig.NewFunctionResult("All questions have already been answered.")
			}

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
