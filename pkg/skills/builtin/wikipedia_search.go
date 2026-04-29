package builtin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// WikipediaSearchSkill searches Wikipedia for information.
type WikipediaSearchSkill struct {
	skills.BaseSkill
	numResults      int
	noResultsMessage string
}

// NewWikipediaSearch creates a new WikipediaSearchSkill.
func NewWikipediaSearch(params map[string]any) skills.SkillBase {
	return &WikipediaSearchSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "wikipedia_search",
			SkillDesc: "Search Wikipedia for information about a topic and get article summaries",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

func (s *WikipediaSearchSkill) Setup() bool {
	s.numResults = s.GetParamInt("num_results", 1)
	if s.numResults < 1 {
		s.numResults = 1
	}
	// noResultsMessage is configurable; matches Python setup() skill.py:71-74.
	s.noResultsMessage = s.GetParamString("no_results_message",
		"I couldn't find any Wikipedia articles for '%s'. Try rephrasing your search or using different keywords.")
	return true
}

func (s *WikipediaSearchSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["num_results"] = map[string]any{
		"type":        "integer",
		"description": "Maximum number of Wikipedia articles to return",
		"default":     1,
		"required":    false,
		"minimum":     1,
		"maximum":     5,
	}
	schema["no_results_message"] = map[string]any{
		"type":        "string",
		"description": "Custom message when no Wikipedia articles are found",
		"default":     "I couldn't find any Wikipedia articles for '%s'. Try rephrasing your search or using different keywords.",
		"required":    false,
	}
	return schema
}

func (s *WikipediaSearchSkill) RegisterTools() []skills.ToolRegistration {
	return []skills.ToolRegistration{
		{
			Name:        "search_wiki",
			Description: "Search Wikipedia for information about a topic and get article summaries",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search term or topic to look up on Wikipedia",
					},
				},
				"required": []string{"query"},
			},
			Handler: s.handleSearch,
		},
	}
}

func (s *WikipediaSearchSkill) handleSearch(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	query, _ := args["query"].(string)
	if query == "" {
		return swaig.NewFunctionResult("Please provide a search query for Wikipedia.")
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// Step 1: Search for articles
	searchURL := fmt.Sprintf(
		"https://en.wikipedia.org/w/api.php?action=query&list=search&format=json&srsearch=%s&srlimit=%d",
		url.QueryEscape(query),
		s.numResults,
	)

	resp, err := client.Get(searchURL)
	if err != nil {
		return swaig.NewFunctionResult(fmt.Sprintf("Error accessing Wikipedia: %v", err))
	}
	defer resp.Body.Close()

	var searchData map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&searchData); err != nil {
		return swaig.NewFunctionResult("Error processing Wikipedia search results.")
	}

	queryData, _ := searchData["query"].(map[string]any)
	searchResults, _ := queryData["search"].([]any)
	if len(searchResults) == 0 {
		return swaig.NewFunctionResult(fmt.Sprintf(s.noResultsMessage, query))
	}

	// Step 2: Get extracts for each result
	var articles []string
	for i, result := range searchResults {
		if i >= s.numResults {
			break
		}
		m, _ := result.(map[string]any)
		if m == nil {
			continue
		}
		title, _ := m["title"].(string)
		if title == "" {
			continue
		}

		extractURL := fmt.Sprintf(
			"https://en.wikipedia.org/w/api.php?action=query&prop=extracts&exintro&explaintext&format=json&titles=%s",
			url.QueryEscape(title),
		)

		extractResp, err := client.Get(extractURL)
		if err != nil {
			continue
		}

		var extractData map[string]any
		if err := json.NewDecoder(extractResp.Body).Decode(&extractData); err != nil {
			extractResp.Body.Close()
			continue
		}
		extractResp.Body.Close()

		pages, _ := extractData["query"].(map[string]any)
		pagesMap, _ := pages["pages"].(map[string]any)
		for _, page := range pagesMap {
			pageMap, _ := page.(map[string]any)
			if pageMap == nil {
				continue
			}
			extract, _ := pageMap["extract"].(string)
			if extract != "" {
				articles = append(articles, fmt.Sprintf("**%s**\n\n%s", title, strings.TrimSpace(extract)))
			}
			break // Only the first page
		}
	}

	if len(articles) == 0 {
		return swaig.NewFunctionResult(fmt.Sprintf(s.noResultsMessage, query))
	}

	return swaig.NewFunctionResult(strings.Join(articles, "\n\n"+strings.Repeat("=", 50)+"\n\n"))
}

func (s *WikipediaSearchSkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Wikipedia Search",
			// Body matches Python get_prompt_sections() (skill.py:190): tool name + num_results interpolated.
			"body": fmt.Sprintf("You can search Wikipedia for factual information using search_wiki. This will return up to %d Wikipedia article summaries.", s.numResults),
			"bullets": []string{
				"Use search_wiki for factual, encyclopedic information",
				"Great for answering questions about people, places, concepts, and history",
				"Returns reliable, well-sourced information from Wikipedia articles",
			},
		},
	}
}

func init() {
	skills.RegisterSkill("wikipedia_search", NewWikipediaSearch)
}
