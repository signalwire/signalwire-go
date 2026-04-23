package builtin

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// InfoGathererSkill guides an AI agent through a series of questions,
// collecting and storing answers in global_data.
type InfoGathererSkill struct {
	skills.BaseSkill
	questions         []map[string]any
	prefix            string
	startToolName     string
	submitToolName    string
	completionMessage string
}

// NewInfoGatherer creates a new InfoGathererSkill.
func NewInfoGatherer(params map[string]any) skills.SkillBase {
	return &InfoGathererSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "info_gatherer",
			SkillDesc: "Gather answers to a configurable list of questions",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

func (s *InfoGathererSkill) SupportsMultipleInstances() bool { return true }

func (s *InfoGathererSkill) GetInstanceKey() string {
	if s.prefix != "" {
		return "info_gatherer_" + s.prefix
	}
	return "info_gatherer"
}

func (s *InfoGathererSkill) Setup() bool {
	questionsRaw, ok := s.Params["questions"]
	if !ok {
		return false
	}
	questionsSlice, ok := questionsRaw.([]any)
	if !ok || len(questionsSlice) == 0 {
		return false
	}

	s.questions = make([]map[string]any, 0, len(questionsSlice))
	for _, q := range questionsSlice {
		m, ok := q.(map[string]any)
		if !ok {
			return false
		}
		if _, ok := m["key_name"].(string); !ok {
			return false
		}
		if _, ok := m["question_text"].(string); !ok {
			return false
		}
		s.questions = append(s.questions, m)
	}

	s.prefix = s.GetParamString("prefix", "")
	if s.prefix != "" {
		s.startToolName = s.prefix + "_start_questions"
		s.submitToolName = s.prefix + "_submit_answer"
	} else {
		s.startToolName = "start_questions"
		s.submitToolName = "submit_answer"
	}

	s.completionMessage = s.GetParamString("completion_message",
		"Thank you! All questions have been answered. You can now summarize "+
			"the information collected or ask if there's anything else the user "+
			"would like to discuss.")

	return true
}

func (s *InfoGathererSkill) RegisterTools() []skills.ToolRegistration {
	return []skills.ToolRegistration{
		{
			Name:        s.startToolName,
			Description: "Start the question sequence with the first question",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
			Handler: s.handleStartQuestions,
		},
		{
			Name:        s.submitToolName,
			Description: "Submit an answer to the current question and move to the next one",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"answer": map[string]any{
						"type":        "string",
						"description": "The user's answer to the current question",
					},
					"confirmed_by_user": map[string]any{
						"type":        "boolean",
						"description": "Only set to true when the user has explicitly confirmed the answer is correct",
					},
				},
				"required": []string{"answer"},
			},
			Handler: s.handleSubmitAnswer,
		},
	}
}

func (s *InfoGathererSkill) handleStartQuestions(_ map[string]any, rawData map[string]any) *swaig.FunctionResult {
	if len(s.questions) == 0 {
		return swaig.NewFunctionResult("I don't have any questions to ask.")
	}

	// Read question_index from state to support resuming mid-sequence
	globalData, _ := rawData["global_data"].(map[string]any)
	namespace := s.getNamespace()
	state, _ := globalData[namespace].(map[string]any)

	questionIndex := 0
	if idx, ok := state["question_index"].(float64); ok {
		questionIndex = int(idx)
	} else if idx, ok := state["question_index"].(int); ok {
		questionIndex = idx
	}
	if questionIndex >= len(s.questions) {
		questionIndex = 0
	}

	q := s.questions[questionIndex]
	questionText, _ := q["question_text"].(string)
	total := len(s.questions)

	instruction := fmt.Sprintf(
		"Ask each question one at a time, wait for the user's answer, "+
			"then call %s with their answer. Do not reuse previous answers.\n\n"+
			"[Question %d of %d]: \"%s\"",
		s.submitToolName, questionIndex+1, total, questionText,
	)

	if confirm, ok := q["confirm"].(bool); ok && confirm {
		instruction += fmt.Sprintf(
			"\nThis question requires confirmation. Read the answer back to the user "+
				"and ask them to confirm it is correct before calling %s.", s.submitToolName,
		)
	}

	if promptAdd, ok := q["prompt_add"].(string); ok && promptAdd != "" {
		instruction += "\nNote: " + promptAdd
	}

	return swaig.NewFunctionResult(instruction)
}

func (s *InfoGathererSkill) handleSubmitAnswer(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
	answer, _ := args["answer"].(string)
	confirmed, _ := args["confirmed_by_user"].(bool)

	// Get current state from global_data
	globalData, _ := rawData["global_data"].(map[string]any)
	namespace := s.getNamespace()
	state, _ := globalData[namespace].(map[string]any)

	questionIndex := 0
	if idx, ok := state["question_index"].(float64); ok {
		questionIndex = int(idx)
	} else if idx, ok := state["question_index"].(int); ok {
		questionIndex = idx
	}

	if questionIndex >= len(s.questions) {
		return swaig.NewFunctionResult("All questions have already been answered.")
	}

	current := s.questions[questionIndex]
	if confirm, ok := current["confirm"].(bool); ok && confirm && !confirmed {
		return swaig.NewFunctionResult(fmt.Sprintf(
			"Before submitting, you must read the answer \"%s\" back to the user "+
				"and ask them to confirm it is correct. Then call this function again with "+
				"confirmed set to true. If the user says it is wrong, ask the question again.",
			answer,
		))
	}

	keyName, _ := current["key_name"].(string)
	answers, _ := state["answers"].([]any)
	answers = append(answers, map[string]any{"key_name": keyName, "answer": answer})
	newIndex := questionIndex + 1

	if newIndex < len(s.questions) {
		nextQ := s.questions[newIndex]
		questionText, _ := nextQ["question_text"].(string)
		total := len(s.questions)

		instruction := fmt.Sprintf("Previous answer saved. [Question %d of %d]: \"%s\"",
			newIndex+1, total, questionText)

		if confirm, ok := nextQ["confirm"].(bool); ok && confirm {
			instruction += fmt.Sprintf(
				"\nThis question requires confirmation. Read the answer back to the user "+
					"and ask them to confirm before calling %s.", s.submitToolName)
		}

		if promptAdd, ok := nextQ["prompt_add"].(string); ok && promptAdd != "" {
			instruction += "\nNote: " + promptAdd
		}

		result := swaig.NewFunctionResult(instruction)
		result.UpdateGlobalData(map[string]any{
			namespace: map[string]any{
				"questions":      s.questions,
				"question_index": newIndex,
				"answers":        answers,
			},
		})
		return result
	}

	// All questions answered
	result := swaig.NewFunctionResult(s.completionMessage)
	result.UpdateGlobalData(map[string]any{
		namespace: map[string]any{
			"questions":      s.questions,
			"question_index": newIndex,
			"answers":        answers,
		},
	})
	result.ToggleFunctions([]map[string]any{
		{"function": s.startToolName, "active": false},
		{"function": s.submitToolName, "active": false},
	})
	return result
}

func (s *InfoGathererSkill) getNamespace() string {
	if s.prefix != "" {
		return "skill:" + s.prefix
	}
	return "skill:" + s.GetInstanceKey()
}

func (s *InfoGathererSkill) GetGlobalData() map[string]any {
	return map[string]any{
		s.getNamespace(): map[string]any{
			"questions":      s.questions,
			"question_index": 0,
			"answers":        []any{},
		},
	}
}

func (s *InfoGathererSkill) GetPromptSections() []map[string]any {
	var questionBullets []string
	for i, q := range s.questions {
		text, _ := q["question_text"].(string)
		questionBullets = append(questionBullets, fmt.Sprintf("Question %d: %s", i+1, text))
	}

	body := fmt.Sprintf(
		"You need to gather answers to a series of questions from the user. "+
			"Start by asking if they are ready. Once confirmed, call %s to get the "+
			"first question. After each answer, call %s with the answer.",
		s.startToolName, s.submitToolName,
	)

	return []map[string]any{
		{
			"title":   "Info Gatherer (" + s.GetInstanceKey() + ")",
			"body":    body,
			"bullets": append(questionBullets, "Ask questions one at a time and wait for answers"),
		},
	}
}

func (s *InfoGathererSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["questions"] = map[string]any{
		"type":        "array",
		"description": "List of question objects with key_name, question_text, and optional confirm",
		"required":    true,
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key_name":      map[string]any{"type": "string"},
				"question_text": map[string]any{"type": "string"},
				"confirm":       map[string]any{"type": "boolean"},
				"prompt_add":    map[string]any{"type": "string"},
			},
		},
	}
	schema["prefix"] = map[string]any{
		"type":        "string",
		"description": "Optional prefix for tool names and namespace",
		"required":    false,
	}
	schema["completion_message"] = map[string]any{
		"type":        "string",
		"description": "Message returned after all questions are answered",
		"required":    false,
	}
	return schema
}

func init() {
	skills.RegisterSkill("info_gatherer", NewInfoGatherer)
}
