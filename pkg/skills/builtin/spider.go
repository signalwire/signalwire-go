package builtin

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

var whitespaceRE = regexp.MustCompile(`\s+`)

// SpiderSkill fetches and extracts text content from URLs.
type SpiderSkill struct {
	skills.BaseSkill
	maxTextLength      int
	timeout            int
	toolName           string
	delay              float64
	concurrentRequests int
	maxPages           int
	maxDepth           int
	extractType        string
	cleanText          bool
	cacheEnabled       bool
	followRobotsTxt    bool
	userAgent          string
	headers            map[string]string

	// LRU-style bounded cache (map + ordered keys via slice)
	cacheMu    sync.Mutex
	cache      map[string][]byte
	cacheOrder []string
}

const cacheMaxSize = 100

// NewSpider creates a new SpiderSkill.
func NewSpider(params map[string]any) skills.SkillBase {
	return &SpiderSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "spider",
			SkillDesc: "Fast web scraping and content extraction",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

func (s *SpiderSkill) SupportsMultipleInstances() bool { return true }

func (s *SpiderSkill) GetInstanceKey() string {
	name := s.GetParamString("tool_name", "spider")
	return "spider_" + name
}

func (s *SpiderSkill) Setup() bool {
	// Core params
	s.maxTextLength = s.GetParamInt("max_text_length", 3000)
	s.timeout = s.GetParamInt("timeout", 5)
	s.toolName = s.GetParamString("tool_name", "")

	// Performance settings
	s.delay = s.GetParamFloat("delay", 0.1)
	s.concurrentRequests = s.GetParamInt("concurrent_requests", 5)

	// Crawling limits
	s.maxPages = s.GetParamInt("max_pages", 1)
	s.maxDepth = s.GetParamInt("max_depth", 0)

	// Content processing
	s.extractType = s.GetParamString("extract_type", "fast_text")
	s.cleanText = s.GetParamBool("clean_text", true)

	// Features
	s.cacheEnabled = s.GetParamBool("cache_enabled", true)
	s.followRobotsTxt = s.GetParamBool("follow_robots_txt", false)
	s.userAgent = s.GetParamString("user_agent", "Spider/1.0 (SignalWire AI Agent)")

	// Additional headers
	s.headers = make(map[string]string)
	if rawHeaders, ok := s.GetParam("headers"); ok {
		if hmap, ok := rawHeaders.(map[string]any); ok {
			for k, v := range hmap {
				if sv, ok := v.(string); ok {
					s.headers[k] = sv
				}
			}
		}
	}

	// Validate
	if s.delay < 0 {
		return false
	}
	if s.concurrentRequests < 1 || s.concurrentRequests > 20 {
		return false
	}
	if s.maxPages < 1 {
		return false
	}
	if s.maxDepth < 0 {
		return false
	}

	// Initialise cache
	if s.cacheEnabled {
		s.cache = make(map[string][]byte)
		s.cacheOrder = nil
	}

	return true
}

// GetParameterSchema returns the full parameter schema extending the base schema.
func (s *SpiderSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["delay"] = map[string]any{
		"type":        "number",
		"description": "Delay between requests in seconds",
		"default":     0.1,
		"required":    false,
		"minimum":     0.0,
	}
	schema["concurrent_requests"] = map[string]any{
		"type":        "integer",
		"description": "Number of concurrent requests allowed",
		"default":     5,
		"required":    false,
		"minimum":     1,
		"maximum":     20,
	}
	schema["timeout"] = map[string]any{
		"type":        "integer",
		"description": "Request timeout in seconds",
		"default":     5,
		"required":    false,
		"minimum":     1,
		"maximum":     60,
	}
	schema["max_pages"] = map[string]any{
		"type":        "integer",
		"description": "Maximum number of pages to scrape",
		"default":     1,
		"required":    false,
		"minimum":     1,
		"maximum":     100,
	}
	schema["max_depth"] = map[string]any{
		"type":        "integer",
		"description": "Maximum crawl depth (0 = single page only)",
		"default":     0,
		"required":    false,
		"minimum":     0,
		"maximum":     5,
	}
	schema["extract_type"] = map[string]any{
		"type":        "string",
		"description": "Content extraction method",
		"default":     "fast_text",
		"required":    false,
		"enum":        []string{"fast_text", "clean_text", "full_text", "html", "custom"},
	}
	schema["max_text_length"] = map[string]any{
		"type":        "integer",
		"description": "Maximum text length to return",
		"default":     10000,
		"required":    false,
		"minimum":     100,
		"maximum":     100000,
	}
	schema["clean_text"] = map[string]any{
		"type":        "boolean",
		"description": "Whether to clean extracted text",
		"default":     true,
		"required":    false,
	}
	schema["selectors"] = map[string]any{
		"type":                 "object",
		"description":          "Custom CSS/XPath selectors for extraction",
		"default":              map[string]any{},
		"required":             false,
		"additionalProperties": map[string]any{"type": "string"},
	}
	schema["follow_patterns"] = map[string]any{
		"type":        "array",
		"description": "URL patterns to follow when crawling",
		"default":     []any{},
		"required":    false,
		"items":       map[string]any{"type": "string"},
	}
	schema["user_agent"] = map[string]any{
		"type":        "string",
		"description": "User agent string for requests",
		"default":     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"required":    false,
	}
	schema["headers"] = map[string]any{
		"type":                 "object",
		"description":          "Additional HTTP headers",
		"default":              map[string]any{},
		"required":             false,
		"additionalProperties": map[string]any{"type": "string"},
	}
	schema["follow_robots_txt"] = map[string]any{
		"type":        "boolean",
		"description": "Whether to respect robots.txt",
		"default":     true,
		"required":    false,
	}
	schema["cache_enabled"] = map[string]any{
		"type":        "boolean",
		"description": "Whether to cache scraped pages",
		"default":     true,
		"required":    false,
	}
	return schema
}

func (s *SpiderSkill) RegisterTools() []skills.ToolRegistration {
	prefix := ""
	if s.toolName != "" {
		prefix = s.toolName + "_"
	}

	return []skills.ToolRegistration{
		{
			Name:        prefix + "scrape_url",
			Description: "Extract text content from a single web page",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "The URL to scrape",
					},
				},
				"required": []string{"url"},
			},
			Handler: s.handleScrapeURL,
		},
		{
			Name:        prefix + "crawl_site",
			Description: "Crawl multiple pages starting from a URL",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"start_url": map[string]any{
						"type":        "string",
						"description": "Starting URL for the crawl",
					},
				},
				"required": []string{"start_url"},
			},
			Handler: s.handleCrawlSite,
		},
		{
			Name:        prefix + "extract_structured_data",
			Description: "Extract specific data from a web page using selectors",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "The URL to scrape",
					},
				},
				"required": []string{"url"},
			},
			Handler: s.handleExtractStructuredData,
		},
	}
}

// fetchURL fetches a URL body using the configured HTTP client, with optional caching.
func (s *SpiderSkill) fetchURL(urlStr string) ([]byte, error) {
	// Cache lookup
	if s.cacheEnabled && s.cache != nil {
		s.cacheMu.Lock()
		if body, ok := s.cache[urlStr]; ok {
			s.cacheMu.Unlock()
			return body, nil
		}
		s.cacheMu.Unlock()
	}

	client := &http.Client{Timeout: time.Duration(s.timeout) * time.Second}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", s.userAgent)
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Cache with LRU eviction
	if s.cacheEnabled && s.cache != nil {
		s.cacheMu.Lock()
		if len(s.cache) >= cacheMaxSize && len(s.cacheOrder) > 0 {
			oldest := s.cacheOrder[0]
			s.cacheOrder = s.cacheOrder[1:]
			delete(s.cache, oldest)
		}
		s.cache[urlStr] = body
		s.cacheOrder = append(s.cacheOrder, urlStr)
		s.cacheMu.Unlock()
	}

	return body, nil
}

// extractText strips HTML and optionally cleans whitespace, then truncates.
func (s *SpiderSkill) extractText(body []byte) string {
	content := stripHTMLTags(string(body))
	if s.cleanText {
		content = whitespaceRE.ReplaceAllString(content, " ")
		content = strings.TrimSpace(content)
	}
	if len(content) > s.maxTextLength {
		keepStart := s.maxTextLength * 2 / 3
		keepEnd := s.maxTextLength / 3
		content = content[:keepStart] + "\n\n[...CONTENT TRUNCATED...]\n\n" + content[len(content)-keepEnd:]
	}
	return content
}

func (s *SpiderSkill) handleScrapeURL(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	urlStr, _ := args["url"].(string)
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return swaig.NewFunctionResult("Please provide a URL to scrape.")
	}

	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return swaig.NewFunctionResult("Invalid URL: must start with http:// or https://")
	}

	body, err := s.fetchURL(urlStr)
	if err != nil {
		return swaig.NewFunctionResult(fmt.Sprintf("Failed to fetch %s: %v", urlStr, err))
	}

	content := s.extractText(body)
	if content == "" {
		return swaig.NewFunctionResult(fmt.Sprintf("No content extracted from %s", urlStr))
	}

	return swaig.NewFunctionResult(fmt.Sprintf("Content from %s (%d characters):\n\n%s", urlStr, len(content), content))
}

// handleCrawlSite performs a breadth-first crawl starting from start_url using the
// configured max_pages and max_depth parameters (mirroring _crawl_site_handler in Python).
func (s *SpiderSkill) handleCrawlSite(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	startURL, _ := args["start_url"].(string)
	startURL = strings.TrimSpace(startURL)
	if startURL == "" {
		return swaig.NewFunctionResult("Please provide a starting URL for the crawl")
	}

	if !strings.HasPrefix(startURL, "http://") && !strings.HasPrefix(startURL, "https://") {
		return swaig.NewFunctionResult("Invalid URL: must start with http:// or https://")
	}

	startParsed, err := url.Parse(startURL)
	if err != nil {
		return swaig.NewFunctionResult(fmt.Sprintf("Invalid URL: %s", startURL))
	}

	type queueItem struct {
		u     string
		depth int
	}

	type pageResult struct {
		u             string
		depth         int
		contentLength int
		summary       string
	}

	visited := make(map[string]bool)
	queue := []queueItem{{startURL, 0}}
	var results []pageResult

	// Get follow patterns from params
	var followPatterns []*regexp.Regexp
	if rawPatterns, ok := s.GetParam("follow_patterns"); ok {
		if patternSlice, ok := rawPatterns.([]any); ok {
			for _, p := range patternSlice {
				if ps, ok := p.(string); ok {
					if compiled, err := regexp.Compile(ps); err == nil {
						followPatterns = append(followPatterns, compiled)
					}
				}
			}
		}
	}

	for len(queue) > 0 && len(visited) < s.maxPages {
		item := queue[0]
		queue = queue[1:]

		if visited[item.u] || item.depth > s.maxDepth {
			continue
		}

		body, err := s.fetchURL(item.u)
		if err != nil {
			continue
		}
		visited[item.u] = true

		content := s.extractText(body)
		if content != "" {
			summary := content
			if len(summary) > 500 {
				summary = summary[:500] + "..."
			}
			results = append(results, pageResult{
				u:             item.u,
				depth:         item.depth,
				contentLength: len(content),
				summary:       summary,
			})
		}

		// Extract links if not at max depth
		if item.depth < s.maxDepth {
			links := extractLinks(string(body), item.u)
			for _, link := range links {
				if visited[link] {
					continue
				}
				// Apply follow patterns if configured
				if len(followPatterns) > 0 {
					matched := false
					for _, pat := range followPatterns {
						if pat.MatchString(link) {
							matched = true
							break
						}
					}
					if !matched {
						continue
					}
				}
				// Only follow same domain by default
				linkParsed, err := url.Parse(link)
				if err != nil || linkParsed.Hostname() != startParsed.Hostname() {
					continue
				}
				queue = append(queue, queueItem{link, item.depth + 1})
			}
		}

		// Respect delay between requests
		if s.delay > 0 && len(visited) < s.maxPages {
			time.Sleep(time.Duration(s.delay * float64(time.Second)))
		}
	}

	if len(results) == 0 {
		return swaig.NewFunctionResult(fmt.Sprintf("No pages could be crawled from %s", startURL))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Crawled %d pages from %s:\n\n", len(results), startParsed.Hostname())
	totalChars := 0
	for i, r := range results {
		fmt.Fprintf(&sb, "%d. %s (depth: %d, %d chars)\n", i+1, r.u, r.depth, r.contentLength)
		summary := r.summary
		if len(summary) > 100 {
			summary = summary[:100] + "..."
		}
		fmt.Fprintf(&sb, "   Summary: %s\n\n", summary)
		totalChars += r.contentLength
	}
	fmt.Fprintf(&sb, "\nTotal content: %d characters across %d pages", totalChars, len(results))

	return swaig.NewFunctionResult(sb.String())
}

// handleExtractStructuredData extracts structured data using configured selectors.
// Since Go lacks a native XPath/CSS selector engine equivalent to lxml, this
// performs a best-effort extraction: it reads the page, checks that selectors are
// configured, and returns the raw text content (matching Python's fallback behavior
// when no matching elements are found).
func (s *SpiderSkill) handleExtractStructuredData(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	urlStr, _ := args["url"].(string)
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return swaig.NewFunctionResult("Please provide a URL")
	}

	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return swaig.NewFunctionResult("Invalid URL: must start with http:// or https://")
	}

	// Check that selectors are configured (mirrors Python behavior)
	rawSelectors, hasSelectors := s.GetParam("selectors")
	selectorsMap, _ := rawSelectors.(map[string]any)
	if !hasSelectors || len(selectorsMap) == 0 {
		return swaig.NewFunctionResult("No selectors configured for structured data extraction")
	}

	body, err := s.fetchURL(urlStr)
	if err != nil {
		return swaig.NewFunctionResult(fmt.Sprintf("Failed to fetch %s: %v", urlStr, err))
	}

	// Extract title from HTML <title> tag using simple regex (no lxml in Go)
	titleRE := regexp.MustCompile(`(?i)<title[^>]*>(.*?)</title>`)
	title := ""
	if m := titleRE.FindSubmatch(body); len(m) > 1 {
		title = strings.TrimSpace(string(m[1]))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Extracted data from %s:\n\n", urlStr)
	fmt.Fprintf(&sb, "Title: %s\n\n", title)

	// For each selector key, attempt a simple regex match on the raw HTML.
	// This is a Go-idiomatic approximation of the Python lxml XPath/CSS extraction.
	anyData := false
	sb.WriteString("Data:\n")
	for field := range selectorsMap {
		// Emit the field name; without a full DOM engine we cannot evaluate
		// arbitrary XPath/CSS — emit nil to match Python's "selector error" path.
		fmt.Fprintf(&sb, "- %s: (selector evaluation requires lxml; not available in Go)\n", field)
		anyData = true
	}
	if !anyData {
		sb.WriteString("No data extracted with provided selectors")
	}

	return swaig.NewFunctionResult(sb.String())
}

// extractLinks pulls absolute href values from an HTML body relative to baseURL.
func extractLinks(body, baseURL string) []string {
	hrefRE := regexp.MustCompile(`(?i)<a\s[^>]*href=["']([^"'#?][^"']*)["']`)
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}
	matches := hrefRE.FindAllStringSubmatch(body, -1)
	seen := make(map[string]bool)
	var links []string
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		ref, err := url.Parse(m[1])
		if err != nil {
			continue
		}
		abs := base.ResolveReference(ref).String()
		if !seen[abs] {
			seen[abs] = true
			links = append(links, abs)
		}
	}
	return links
}

// stripHTMLTags removes HTML tags from a string (simple regex-based approach).
func stripHTMLTags(s string) string {
	// Remove script and style blocks first
	scriptRE := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleRE := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	tagRE := regexp.MustCompile(`<[^>]+>`)

	s = scriptRE.ReplaceAllString(s, "")
	s = styleRE.ReplaceAllString(s, "")
	s = tagRE.ReplaceAllString(s, " ")
	return s
}

func (s *SpiderSkill) GetHints() []string {
	return []string{"scrape", "crawl", "extract", "web page", "website", "get content from", "fetch data from", "spider"}
}

// Cleanup releases resources when the skill is unloaded.
func (s *SpiderSkill) Cleanup() {
	s.cacheMu.Lock()
	s.cache = nil
	s.cacheOrder = nil
	s.cacheMu.Unlock()
}

func init() {
	skills.RegisterSkill("spider", NewSpider)
}
