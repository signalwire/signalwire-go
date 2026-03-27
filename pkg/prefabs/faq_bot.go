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

// FAQ represents a single frequently asked question and its answer.
type FAQ struct {
	Question   string
	Answer     string
	Categories []string
}

// FAQBotOptions configures a new FAQBotAgent.
type FAQBotOptions struct {
	Name           string
	Route          string
	FAQs           []FAQ
	SuggestRelated bool
	Persona        string
}

// FAQBotAgent answers frequently asked questions by matching user queries
// against a provided FAQ database.
type FAQBotAgent struct {
	*agent.AgentBase
	faqs           []FAQ
	suggestRelated bool
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// NewFAQBotAgent creates an agent that answers frequently asked questions.
func NewFAQBotAgent(opts FAQBotOptions) *FAQBotAgent {
	name := opts.Name
	if name == "" {
		name = "faq_bot"
	}
	route := opts.Route
	if route == "" {
		route = "/faq"
	}
	persona := opts.Persona
	if persona == "" {
		persona = "You are a helpful FAQ bot that provides accurate answers to common questions."
	}

	base := agent.NewAgentBase(
		agent.WithName(name),
		agent.WithRoute(route),
	)

	fb := &FAQBotAgent{
		AgentBase:      base,
		faqs:           opts.FAQs,
		suggestRelated: opts.SuggestRelated,
	}

	// ---- Prompt ----
	base.PromptAddSection("Personality", persona, nil)
	base.PromptAddSection("Goal",
		"Answer user questions by matching them to the most similar FAQ in your database.",
		nil,
	)

	instructions := []string{
		"Compare user questions to your FAQ database and find the best match.",
		"Provide the answer from the FAQ database for the matching question.",
		"If no close match exists, politely say you don't have that information.",
		"Be concise and factual in your responses.",
	}
	if opts.SuggestRelated {
		instructions = append(instructions,
			"When appropriate, suggest other related questions from the FAQ database that might be helpful.",
		)
	}
	base.PromptAddSection("Instructions", "", instructions)

	// Build FAQ database section: each FAQ as a subsection
	base.PromptAddSection("FAQ Database",
		"Here is your database of frequently asked questions and answers:",
		nil,
	)
	for _, faq := range opts.FAQs {
		if faq.Question == "" || faq.Answer == "" {
			continue
		}
		body := faq.Answer
		if len(faq.Categories) > 0 {
			body += "\n\nCategories: " + strings.Join(faq.Categories, ", ")
		}
		base.PromptAddSubsection("FAQ Database", faq.Question, body, nil)
	}

	if opts.SuggestRelated {
		base.PromptAddSection("Related Questions",
			"When appropriate, suggest other related questions from the FAQ database that might be helpful.",
			nil,
		)
	}

	// ---- Post-prompt ----
	base.SetPostPrompt(`Return a JSON summary of this interaction:
{
    "question": "MAIN_QUESTION_ASKED",
    "matched_faq": "MATCHED_FAQ_QUESTION_OR_null",
    "answered_successfully": true/false,
    "suggested_related": []
}`)

	// ---- Global data ----
	categories := map[string]bool{}
	for _, faq := range opts.FAQs {
		for _, cat := range faq.Categories {
			categories[cat] = true
		}
	}
	catList := make([]string, 0, len(categories))
	for cat := range categories {
		catList = append(catList, cat)
	}
	base.SetGlobalData(map[string]any{
		"faq_count":  len(opts.FAQs),
		"categories": catList,
	})

	// ---- Hints ----
	hints := make([]string, 0)
	for _, faq := range opts.FAQs {
		words := strings.Fields(faq.Question)
		for _, w := range words {
			cleaned := strings.Trim(w, ".,?!")
			if len(cleaned) >= 4 {
				hints = append(hints, cleaned)
			}
		}
		hints = append(hints, faq.Categories...)
	}
	if len(hints) > 0 {
		// Deduplicate
		seen := map[string]bool{}
		unique := make([]string, 0, len(hints))
		for _, h := range hints {
			lower := strings.ToLower(h)
			if !seen[lower] {
				seen[lower] = true
				unique = append(unique, h)
			}
		}
		base.AddHints(unique)
	}

	// ---- Tools ----
	fb.registerTools()

	return fb
}

// ---------------------------------------------------------------------------
// Tool registration
// ---------------------------------------------------------------------------

func (fb *FAQBotAgent) registerTools() {
	// search_faqs -------------------------------------------------------
	fb.DefineTool(agent.ToolDefinition{
		Name:        "search_faqs",
		Description: "Search for FAQs matching a specific query or category",
		Parameters: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			query := strings.ToLower(strings.TrimSpace(args["query"].(string)))

			type scored struct {
				question string
				score    int
			}
			var results []scored

			for _, faq := range fb.faqs {
				q := strings.ToLower(faq.Question)
				score := 0

				if query != "" && strings.Contains(q, query) {
					if q == query {
						score += 100
					} else {
						score += 50
					}
					if strings.HasPrefix(q, query) {
						score += 25
					}
				}

				// Also check individual words for partial matching
				if score == 0 && query != "" {
					queryWords := strings.Fields(query)
					for _, qw := range queryWords {
						if len(qw) >= 3 && strings.Contains(q, qw) {
							score += 10
						}
					}
				}

				if score > 0 {
					results = append(results, scored{question: faq.Question, score: score})
				}
			}

			// Sort descending by score (simple insertion sort for small lists)
			for i := 1; i < len(results); i++ {
				for j := i; j > 0 && results[j].score > results[j-1].score; j-- {
					results[j], results[j-1] = results[j-1], results[j]
				}
			}

			// Limit to top 3
			if len(results) > 3 {
				results = results[:3]
			}

			if len(results) > 0 {
				var sb strings.Builder
				sb.WriteString("Here are the most relevant FAQs:\n\n")
				for i, r := range results {
					fmt.Fprintf(&sb, "%d. %s\n", i+1, r.question)
				}
				return swaig.NewFunctionResult(sb.String())
			}

			return swaig.NewFunctionResult("No matching FAQs found.")
		},
	})
}
