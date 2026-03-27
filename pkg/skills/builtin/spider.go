package builtin

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

var whitespaceRE = regexp.MustCompile(`\s+`)

// SpiderSkill fetches and extracts text content from URLs.
type SpiderSkill struct {
	skills.BaseSkill
	maxTextLength int
	timeout       int
	toolName      string
}

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
	s.maxTextLength = s.GetParamInt("max_text_length", 10000)
	s.timeout = s.GetParamInt("timeout", 10)
	s.toolName = s.GetParamString("tool_name", "")
	return true
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
	}
}

func (s *SpiderSkill) handleScrapeURL(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	urlStr, _ := args["url"].(string)
	if urlStr == "" {
		return swaig.NewFunctionResult("Please provide a URL to scrape.")
	}

	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return swaig.NewFunctionResult("Invalid URL: must start with http:// or https://")
	}

	client := &http.Client{Timeout: time.Duration(s.timeout) * time.Second}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return swaig.NewFunctionResult(fmt.Sprintf("Invalid URL: %s", urlStr))
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return swaig.NewFunctionResult(fmt.Sprintf("Failed to fetch %s: %v", urlStr, err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return swaig.NewFunctionResult(fmt.Sprintf("Failed to fetch %s: HTTP %d", urlStr, resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return swaig.NewFunctionResult(fmt.Sprintf("Error reading response from %s", urlStr))
	}

	// Simple text extraction: strip HTML tags
	content := stripHTMLTags(string(body))
	content = whitespaceRE.ReplaceAllString(content, " ")
	content = strings.TrimSpace(content)

	if content == "" {
		return swaig.NewFunctionResult(fmt.Sprintf("No content extracted from %s", urlStr))
	}

	if len(content) > s.maxTextLength {
		keepStart := s.maxTextLength * 2 / 3
		keepEnd := s.maxTextLength / 3
		content = content[:keepStart] + "\n\n[...CONTENT TRUNCATED...]\n\n" + content[len(content)-keepEnd:]
	}

	return swaig.NewFunctionResult(fmt.Sprintf("Content from %s (%d characters):\n\n%s", urlStr, len(content), content))
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
	return []string{"scrape", "crawl", "extract", "web page", "website"}
}

func init() {
	skills.RegisterSkill("spider", NewSpider)
}
