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
	numResults int
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
	return true
}

func (s *WikipediaSearchSkill) RegisterTools() []skills.ToolRegistration {
	return []skills.ToolRegistration{
		{
			Name:        "search_wikipedia",
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
		return swaig.NewFunctionResult(fmt.Sprintf("I couldn't find any Wikipedia articles for '%s'. Try rephrasing your search.", query))
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
		return swaig.NewFunctionResult(fmt.Sprintf("I couldn't find any Wikipedia articles for '%s'.", query))
	}

	return swaig.NewFunctionResult(strings.Join(articles, "\n\n"+strings.Repeat("=", 50)+"\n\n"))
}

func (s *WikipediaSearchSkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Wikipedia Search",
			"body":  "You can search Wikipedia for factual information using search_wikipedia.",
			"bullets": []string{
				"Use search_wikipedia for factual, encyclopedic information",
				"Great for answering questions about people, places, concepts, and history",
				"Returns reliable, well-sourced information from Wikipedia articles",
			},
		},
	}
}

func init() {
	skills.RegisterSkill("wikipedia_search", NewWikipediaSearch)
}
