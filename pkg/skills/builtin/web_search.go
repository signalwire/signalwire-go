package builtin

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// WebSearchSkill searches the web using Google Custom Search API.
type WebSearchSkill struct {
	skills.BaseSkill
	apiKey           string
	searchEngineID   string
	numResults       int
	toolName         string
	defaultDelay     float64
	maxContentLength int
	oversampleFactor float64
	minQualityScore  float64
	noResultsMessage string
}

// NewWebSearch creates a new WebSearchSkill.
func NewWebSearch(params map[string]any) skills.SkillBase {
	return &WebSearchSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "web_search",
			SkillDesc: "Search the web for information using Google Custom Search API",
			SkillVer:  "2.0.0",
			Params:    params,
		},
	}
}

func (s *WebSearchSkill) RequiredEnvVars() []string {
	if s.Params != nil {
		_, hasKey := s.Params["api_key"]
		_, hasEngine := s.Params["search_engine_id"]
		if hasKey && hasEngine {
			return nil
		}
	}
	return []string{"GOOGLE_SEARCH_API_KEY", "GOOGLE_SEARCH_ENGINE_ID"}
}

func (s *WebSearchSkill) SupportsMultipleInstances() bool { return true }

// GetInstanceKey returns a unique key incorporating both searchEngineID and toolName,
// matching Python's f"{SKILL_NAME}_{search_engine_id}_{tool_name}" pattern.
func (s *WebSearchSkill) GetInstanceKey() string {
	return "web_search_" + s.searchEngineID + "_" + s.toolName
}

func (s *WebSearchSkill) Setup() bool {
	s.apiKey = s.GetParamString("api_key", os.Getenv("GOOGLE_SEARCH_API_KEY"))
	s.searchEngineID = s.GetParamString("search_engine_id", os.Getenv("GOOGLE_SEARCH_ENGINE_ID"))
	if s.apiKey == "" || s.searchEngineID == "" {
		return false
	}
	s.numResults = s.GetParamInt("num_results", 3)
	s.toolName = s.GetParamString("tool_name", "web_search")
	s.defaultDelay = s.GetParamFloat("delay", 0.5)
	s.maxContentLength = s.GetParamInt("max_content_length", 32768)
	s.oversampleFactor = s.GetParamFloat("oversample_factor", 2.5)
	s.minQualityScore = s.GetParamFloat("min_quality_score", 0.3)
	s.noResultsMessage = s.GetParamString("no_results_message",
		"I couldn't find quality results for '{query}'. "+
			"The search returned only low-quality or inaccessible pages. "+
			"Try rephrasing your search or asking about a different topic.")
	return true
}

func (s *WebSearchSkill) RegisterTools() []skills.ToolRegistration {
	return []skills.ToolRegistration{
		{
			Name:        s.toolName,
			Description: "Search the web for high-quality information, automatically filtering low-quality results",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search query - what you want to find information about",
					},
				},
				"required": []string{"query"},
			},
			Handler: s.handleWebSearch,
		},
	}
}

// searchResult holds a single Google Custom Search result.
type searchResult struct {
	title   string
	link    string
	snippet string
}

// processedResult holds a scraped and scored result.
type processedResult struct {
	title        string
	link         string
	snippet      string
	content      string
	domain       string
	qualityScore float64
	metrics      map[string]any
}

// searchGoogle calls the Google Custom Search API and returns raw results.
//
// Base URL is normally googleapis.com; the porting-sdk's
// audit_skills_dispatch.py overrides via WEB_SEARCH_BASE_URL so a
// loopback fixture can stand in for Google CSE.
func (s *WebSearchSkill) searchGoogle(query string, numResults int) ([]searchResult, error) {
	if numResults > 10 {
		numResults = 10
	}
	base := os.Getenv("WEB_SEARCH_BASE_URL")
	if base == "" {
		base = "https://www.googleapis.com"
	}
	base = strings.TrimRight(base, "/")
	apiURL := fmt.Sprintf(
		"%s/customsearch/v1?key=%s&cx=%s&q=%s&num=%d",
		base,
		s.apiKey,
		s.searchEngineID,
		url.QueryEscape(query),
		numResults,
	)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search API returned %d", resp.StatusCode)
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	items, _ := data["items"].([]any)
	if len(items) == 0 {
		return nil, nil
	}

	var results []searchResult
	for _, item := range items {
		m, _ := item.(map[string]any)
		if m == nil {
			continue
		}
		results = append(results, searchResult{
			title:   stringVal(m["title"]),
			link:    stringVal(m["link"]),
			snippet: stringVal(m["snippet"]),
		})
	}
	return results, nil
}

// isRedditURL checks whether a URL is from Reddit.
func isRedditURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return strings.Contains(host, "reddit.com") || strings.Contains(host, "redd.it")
}

// extractTextFromURL routes to the appropriate extractor.
func (s *WebSearchSkill) extractTextFromURL(rawURL string) (string, map[string]any) {
	if isRedditURL(rawURL) {
		return s.extractRedditContent(rawURL)
	}
	return s.extractHTMLContent(rawURL)
}

// extractRedditContent fetches the Reddit JSON API for a post URL, extracts post
// title, body, and top-scored comments, then calculates quality metrics.
func (s *WebSearchSkill) extractRedditContent(rawURL string) (string, map[string]any) {
	jsonURL := strings.TrimRight(rawURL, "/") + ".json"
	if strings.HasSuffix(rawURL, ".json") {
		jsonURL = rawURL
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", jsonURL, nil)
	if err != nil {
		return s.extractHTMLContent(rawURL)
	}
	req.Header.Set("User-Agent", "SignalWire-WebSearch/2.0")

	resp, err := client.Do(req)
	if err != nil {
		return s.extractHTMLContent(rawURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return s.extractHTMLContent(rawURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return s.extractHTMLContent(rawURL)
	}

	var data []any
	if err := json.Unmarshal(body, &data); err != nil || len(data) < 1 {
		return s.extractHTMLContent(rawURL)
	}

	// data[0] contains the post listing
	listing, ok := data[0].(map[string]any)
	if !ok {
		return s.extractHTMLContent(rawURL)
	}
	listingData, _ := listing["data"].(map[string]any)
	children, _ := listingData["children"].([]any)
	if len(children) == 0 {
		return s.extractHTMLContent(rawURL)
	}
	child0, _ := children[0].(map[string]any)
	postData, _ := child0["data"].(map[string]any)
	if postData == nil {
		return s.extractHTMLContent(rawURL)
	}

	title := stringVal(postData["title"])
	author := stringVal(postData["author"])
	score := intVal(postData["score"])
	numComments := intVal(postData["num_comments"])
	subreddit := stringVal(postData["subreddit"])

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Reddit r/%s Discussion\n", subreddit))
	sb.WriteString(fmt.Sprintf("Post: %s\n", title))
	sb.WriteString(fmt.Sprintf("Author: %s | Score: %d | Comments: %d\n", author, score, numComments))

	selftext := strings.TrimSpace(stringVal(postData["selftext"]))
	if selftext != "" && selftext != "[removed]" && selftext != "[deleted]" {
		if len(selftext) > 1000 {
			selftext = selftext[:1000]
		}
		sb.WriteString(fmt.Sprintf("\nOriginal Post:\n%s\n", selftext))
	}

	// Extract comments from data[1] if present
	type comment struct {
		body   string
		author string
		score  int
	}
	var validComments []comment

	if len(data) > 1 {
		if commentsListing, ok := data[1].(map[string]any); ok {
			if cd, ok := commentsListing["data"].(map[string]any); ok {
				if commentChildren, ok := cd["children"].([]any); ok {
					for i, c := range commentChildren {
						if i >= 20 {
							break
						}
						cm, _ := c.(map[string]any)
						if cm == nil {
							continue
						}
						if stringVal(cm["kind"]) != "t1" {
							continue
						}
						cd2, _ := cm["data"].(map[string]any)
						if cd2 == nil {
							continue
						}
						body := strings.TrimSpace(stringVal(cd2["body"]))
						if body == "" || body == "[removed]" || body == "[deleted]" || len(body) <= 50 {
							continue
						}
						validComments = append(validComments, comment{
							body:   body,
							author: stringVal(cd2["author"]),
							score:  intVal(cd2["score"]),
						})
					}
				}
			}
		}
	}

	// Sort by score descending, take top 5
	sort.Slice(validComments, func(i, j int) bool {
		return validComments[i].score > validComments[j].score
	})
	if len(validComments) > 5 {
		validComments = validComments[:5]
	}

	if len(validComments) > 0 {
		sb.WriteString("\n--- Top Discussion ---")
		for i, c := range validComments {
			text := c.body
			if len(text) > 500 {
				text = text[:500] + "..."
			}
			sb.WriteString(fmt.Sprintf("\nComment %d (Score: %d, Author: %s):\n%s\n", i+1, c.score, c.author, text))
		}
	}

	text := sb.String()

	// Quality metrics (Reddit-specific, mirrors Python)
	lengthScore := math.Min(1.0, float64(len(text))/2000.0)
	engagementScore := math.Min(1.0, float64(score+numComments)/100.0)
	hasCommentScore := 1.0
	if len(validComments) == 0 {
		hasCommentScore = 0.3
	}
	qualityScore := lengthScore*0.4 + engagementScore*0.3 + hasCommentScore*0.3

	u, _ := url.Parse(rawURL)
	metrics := map[string]any{
		"text_length":   len(text),
		"score":         score,
		"num_comments":  numComments,
		"domain":        strings.ToLower(u.Hostname()),
		"is_reddit":     true,
		"quality_score": math.Round(qualityScore*1000) / 1000,
	}

	if len(text) > s.maxContentLength {
		text = text[:s.maxContentLength]
	}
	return text, metrics
}

// extractHTMLContent fetches a web page and extracts meaningful text content,
// removing navigation, ads, scripts, and other boilerplate.
func (s *WebSearchSkill) extractHTMLContent(rawURL string) (string, map[string]any) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", map[string]any{"error": err.Error(), "quality_score": 0}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", map[string]any{"error": err.Error(), "quality_score": 0}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", map[string]any{"error": fmt.Sprintf("HTTP %d", resp.StatusCode), "quality_score": 0}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024)) // max 2MB read
	if err != nil {
		return "", map[string]any{"error": err.Error(), "quality_score": 0}
	}

	text := extractTextFromHTML(string(body))
	metrics := calculateContentQuality(text, rawURL, "")

	if len(text) > s.maxContentLength {
		text = text[:s.maxContentLength]
	}
	return text, metrics
}

// extractTextFromHTML extracts readable text from raw HTML by stripping tags.
// It removes scripts, styles, and common navigation/boilerplate elements.
func extractTextFromHTML(html string) string {
	// Remove script and style blocks
	scriptRe := regexp.MustCompile(`(?is)<(script|style|nav|footer|header|aside|noscript|iframe)[^>]*>.*?</(script|style|nav|footer|header|aside|noscript|iframe)>`)
	html = scriptRe.ReplaceAllString(html, " ")

	// Remove all remaining tags
	tagRe := regexp.MustCompile(`<[^>]+>`)
	html = tagRe.ReplaceAllString(html, " ")

	// Decode common HTML entities
	html = strings.ReplaceAll(html, "&amp;", "&")
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&quot;", `"`)
	html = strings.ReplaceAll(html, "&#39;", "'")
	html = strings.ReplaceAll(html, "&nbsp;", " ")

	// Collapse whitespace
	wsRe := regexp.MustCompile(`\s+`)
	html = wsRe.ReplaceAllString(html, " ")

	return strings.TrimSpace(html)
}

// calculateContentQuality scores extracted text content on a 0-1 scale.
// Mirrors the Python GoogleSearchScraper._calculate_content_quality method.
func calculateContentQuality(text, rawURL, query string) map[string]any {
	if text == "" {
		return map[string]any{"quality_score": float64(0), "text_length": 0}
	}

	metrics := map[string]any{}
	textLength := len(text)
	metrics["text_length"] = textLength

	// Length score (mirrors Python thresholds)
	var lengthScore float64
	switch {
	case textLength < 500:
		lengthScore = 0
	case textLength < 2000:
		lengthScore = float64(textLength-500) / 1500.0 * 0.5
	case textLength <= 10000:
		lengthScore = 1.0
	default:
		lengthScore = math.Max(0.8, 1.0-float64(textLength-10000)/20000.0)
	}

	// Word diversity
	words := strings.Fields(strings.ToLower(text))
	var diversityScore float64
	if len(words) > 0 {
		uniqueWords := make(map[string]struct{}, len(words))
		for _, w := range words {
			uniqueWords[w] = struct{}{}
		}
		ratio := float64(len(uniqueWords)) / float64(len(words))
		diversityScore = math.Min(1.0, float64(len(uniqueWords))/(float64(len(words))*0.3))
		metrics["word_diversity"] = ratio
	}

	// Boilerplate penalty
	boilerplatePhrases := []string{
		"cookie", "privacy policy", "terms of service", "subscribe",
		"sign up", "log in", "advertisement", "sponsored", "copyright",
		"all rights reserved", "skip to", "navigation", "breadcrumb",
		"reddit inc", "google llc", "expand navigation", "members •",
		"archived post", "votes cannot be cast", "r/", "subreddit",
		"youtube", "facebook", "twitter", "instagram", "pinterest",
	}
	lowerText := strings.ToLower(text)
	boilerplateCount := 0
	for _, phrase := range boilerplatePhrases {
		if strings.Contains(lowerText, phrase) {
			boilerplateCount++
		}
	}
	boilerplatePenalty := math.Max(0, 1.0-float64(boilerplateCount)*0.15)
	metrics["boilerplate_count"] = boilerplateCount

	// Sentence score
	sentenceRe := regexp.MustCompile(`[.!?]+`)
	parts := sentenceRe.Split(text, -1)
	sentenceCount := 0
	for _, p := range parts {
		if len(strings.TrimSpace(p)) > 30 {
			sentenceCount++
		}
	}
	sentenceScore := math.Min(1.0, float64(sentenceCount)/10.0)
	metrics["sentence_count"] = sentenceCount

	// Domain score
	u, _ := url.Parse(rawURL)
	domain := strings.ToLower(u.Hostname())
	qualityDomains := []string{
		"wikipedia.org", "starwars.fandom.com", "imdb.com",
		"screenrant.com", "denofgeek.com", "ign.com",
		"hollywoodreporter.com", "variety.com", "ew.com",
		"stackexchange.com", "stackoverflow.com",
		"github.com", "medium.com", "dev.to", "arxiv.org",
		"nature.com", "sciencedirect.com", "ieee.org",
	}
	lowQualityDomains := []string{
		"reddit.com", "youtube.com", "facebook.com", "twitter.com",
		"instagram.com", "pinterest.com", "tiktok.com", "x.com",
	}
	domainScore := 1.0
	for _, d := range qualityDomains {
		if strings.Contains(domain, d) {
			domainScore = 1.5
			break
		}
	}
	if domainScore == 1.0 {
		for _, d := range lowQualityDomains {
			if strings.Contains(domain, d) {
				domainScore = 0.1
				break
			}
		}
	}
	metrics["domain"] = domain

	// Query relevance
	relevanceScore := 0.5
	if query != "" {
		stopWords := map[string]bool{
			"the": true, "a": true, "an": true, "and": true, "or": true,
			"but": true, "in": true, "on": true, "at": true, "to": true,
			"for": true, "of": true, "with": true, "by": true, "from": true,
			"as": true, "is": true, "was": true, "are": true, "were": true,
		}
		var queryWords []string
		for _, w := range strings.Fields(query) {
			lw := strings.ToLower(w)
			if !stopWords[lw] && len(w) > 2 {
				queryWords = append(queryWords, lw)
			}
		}
		if len(queryWords) > 0 {
			lowerContent := strings.ToLower(text)
			wordsFound := 0
			for _, w := range queryWords {
				if strings.Contains(lowerContent, w) {
					wordsFound++
				}
			}
			exactBonus := 0.0
			for i := 0; i < len(queryWords)-1; i++ {
				phrase := queryWords[i] + " " + queryWords[i+1]
				if strings.Contains(lowerContent, phrase) {
					exactBonus += 0.2
				}
			}
			relevanceScore = math.Min(1.0, float64(wordsFound)/float64(len(queryWords))+exactBonus)
			metrics["query_relevance"] = math.Round(relevanceScore*1000) / 1000
			metrics["query_words_found"] = fmt.Sprintf("%d/%d", wordsFound, len(queryWords))
		}
	} else {
		metrics["query_relevance"] = 0.5
	}

	// Final score (same weights as Python)
	qualityScore := lengthScore*0.25 +
		diversityScore*0.10 +
		boilerplatePenalty*0.10 +
		sentenceScore*0.15 +
		domainScore*0.15 +
		relevanceScore*0.25

	metrics["quality_score"] = math.Round(qualityScore*1000) / 1000
	metrics["length_score"] = math.Round(lengthScore*1000) / 1000
	metrics["diversity_score"] = math.Round(diversityScore*1000) / 1000
	metrics["boilerplate_penalty"] = math.Round(boilerplatePenalty*1000) / 1000
	metrics["sentence_score"] = math.Round(sentenceScore*1000) / 1000
	metrics["domain_score"] = math.Round(domainScore*1000) / 1000

	return metrics
}

func (s *WebSearchSkill) handleWebSearch(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	query, _ := args["query"].(string)
	query = strings.TrimFunc(query, unicode.IsSpace)
	if query == "" {
		return swaig.NewFunctionResult("Please provide a search query. What would you like me to search for?")
	}

	// Fetch oversample_factor × num_results, capped at 10 (mirrors Python)
	fetchCount := int(math.Min(10, float64(s.numResults)*s.oversampleFactor))
	if fetchCount < 1 {
		fetchCount = 1
	}

	searchResults, err := s.searchGoogle(query, fetchCount)
	if err != nil {
		return swaig.NewFunctionResult("Sorry, I encountered an error while searching. Please try again later.")
	}
	if len(searchResults) == 0 {
		return swaig.NewFunctionResult(s.formatNoResults(query))
	}

	// Scrape and score each result
	var processed []processedResult
	for _, r := range searchResults {
		text, metrics := s.extractTextFromURL(r.link)
		if text != "" {
			// Recalculate with query for relevance
			metrics = calculateContentQuality(text, r.link, query)
		}

		qs, _ := metrics["quality_score"].(float64)
		if qs >= s.minQualityScore && text != "" {
			dom, _ := metrics["domain"].(string)
			if dom == "" {
				u, _ := url.Parse(r.link)
				dom = strings.ToLower(u.Hostname())
			}
			processed = append(processed, processedResult{
				title:        r.title,
				link:         r.link,
				snippet:      r.snippet,
				content:      text,
				domain:       dom,
				qualityScore: qs,
				metrics:      metrics,
			})
		}

		if s.defaultDelay > 0 {
			time.Sleep(time.Duration(s.defaultDelay * float64(time.Second)))
		}
	}

	if len(processed) == 0 {
		return swaig.NewFunctionResult(s.formatNoResults(query))
	}

	// Sort by quality score descending
	sort.Slice(processed, func(i, j int) bool {
		return processed[i].qualityScore > processed[j].qualityScore
	})

	// Select diverse results (prefer different domains), then fill to numResults
	var best []processedResult
	seenDomains := map[string]bool{}
	for _, r := range processed {
		if !seenDomains[r.domain] && len(best) < s.numResults {
			best = append(best, r)
			seenDomains[r.domain] = true
		}
	}
	for _, r := range processed {
		if len(best) >= s.numResults {
			break
		}
		alreadyIn := false
		for _, b := range best {
			if b.link == r.link {
				alreadyIn = true
				break
			}
		}
		if !alreadyIn {
			best = append(best, r)
		}
	}

	if len(best) == 0 {
		return swaig.NewFunctionResult(s.formatNoResults(query))
	}

	// Calculate per-result content budget (mirrors Python)
	estimatedOverheadPerResult := 400
	totalOverhead := len(best) * estimatedOverheadPerResult
	availableForContent := s.maxContentLength - totalOverhead
	perResultLimit := availableForContent / len(best)
	if perResultLimit < 2000 {
		perResultLimit = 2000
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d results meeting quality threshold from %d searched.\n", len(processed), len(searchResults)))
	sb.WriteString(fmt.Sprintf("Showing top %d from diverse sources:\n\n", len(best)))

	for i, r := range best {
		sb.WriteString(fmt.Sprintf("=== RESULT %d (Quality: %.2f) ===\n", i+1, r.qualityScore))
		sb.WriteString(fmt.Sprintf("Title: %s\n", r.title))
		sb.WriteString(fmt.Sprintf("URL: %s\n", r.link))
		sb.WriteString(fmt.Sprintf("Source: %s\n", r.domain))
		sb.WriteString(fmt.Sprintf("Snippet: %s\n", r.snippet))

		textLen, _ := r.metrics["text_length"].(int)
		sentCount, _ := r.metrics["sentence_count"].(int)
		qRel, _ := r.metrics["query_relevance"].(float64)
		qWords, _ := r.metrics["query_words_found"].(string)
		if qWords == "" {
			qWords = "N/A"
		}
		sb.WriteString(fmt.Sprintf("Content Stats: %d chars, %d sentences\n", textLen, sentCount))
		sb.WriteString(fmt.Sprintf("Query Relevance: %.2f (keywords: %s)\n", qRel, qWords))
		sb.WriteString("Content:\n")

		content := r.content
		if len(content) > perResultLimit {
			content = content[:perResultLimit] + "..."
		}
		sb.WriteString(content)
		sb.WriteString("\n" + strings.Repeat("=", 50) + "\n\n")
	}

	return swaig.NewFunctionResult(fmt.Sprintf("Quality web search results for '%s':\n\n%s", query, sb.String()))
}

// formatNoResults returns the configured no-results message with query substituted.
func (s *WebSearchSkill) formatNoResults(query string) string {
	return strings.ReplaceAll(s.noResultsMessage, "{query}", query)
}

// GetGlobalData returns global context data signalling that quality-filtered web
// search is available. Mirrors Python's get_global_data return value.
func (s *WebSearchSkill) GetGlobalData() map[string]any {
	return map[string]any{
		"web_search_enabled": true,
		"search_provider":    "Google Custom Search",
		"quality_filtering":  true,
	}
}

func (s *WebSearchSkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Web Search Capability",
			"body":  "You can search the internet for high-quality information using the " + s.toolName + " tool.",
			"bullets": []string{
				"Use " + s.toolName + " when users ask for information you need to look up",
				"The search automatically filters out low-quality results like empty pages",
				"Results are ranked by content quality, relevance, and domain reputation",
				"Summarize the high-quality results in a clear, helpful way",
			},
		},
	}
}

func (s *WebSearchSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["api_key"] = map[string]any{
		"type":        "string",
		"description": "Google Custom Search API key",
		"required":    true,
		"hidden":      true,
		"env_var":     "GOOGLE_SEARCH_API_KEY",
	}
	schema["search_engine_id"] = map[string]any{
		"type":        "string",
		"description": "Google Custom Search Engine ID",
		"required":    true,
		"hidden":      true,
		"env_var":     "GOOGLE_SEARCH_ENGINE_ID",
	}
	schema["num_results"] = map[string]any{
		"type":        "integer",
		"description": "Number of high-quality results to return",
		"default":     3,
		"required":    false,
		"min":         1,
		"max":         10,
	}
	schema["delay"] = map[string]any{
		"type":        "number",
		"description": "Delay between scraping pages in seconds",
		"default":     0.5,
		"required":    false,
		"min":         0,
	}
	schema["max_content_length"] = map[string]any{
		"type":        "integer",
		"description": "Maximum total response size in characters",
		"default":     32768,
		"required":    false,
		"min":         1000,
	}
	schema["oversample_factor"] = map[string]any{
		"type":        "number",
		"description": "How many extra results to fetch for quality filtering (e.g., 2.5 = fetch 2.5x requested)",
		"default":     2.5,
		"required":    false,
		"min":         1.0,
		"max":         3.5,
	}
	schema["min_quality_score"] = map[string]any{
		"type":        "number",
		"description": "Minimum quality score (0-1) for including a result",
		"default":     0.3,
		"required":    false,
		"min":         0.0,
		"max":         1.0,
	}
	schema["no_results_message"] = map[string]any{
		"type":        "string",
		"description": "Message to show when no quality results are found. Use {query} as placeholder.",
		"default":     "I couldn't find quality results for '{query}'. The search returned only low-quality or inaccessible pages. Try rephrasing your search or asking about a different topic.",
		"required":    false,
	}
	return schema
}

// stringVal safely extracts a string from an any value.
func stringVal(v any) string {
	s, _ := v.(string)
	return s
}

// intVal safely extracts an int from an any value (handles float64 from JSON).
func intVal(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	}
	return 0
}

func init() {
	skills.RegisterSkill("web_search", NewWebSearch)
}
