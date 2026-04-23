package builtin

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// NativeVectorSearchSkill searches knowledge using a remote search server.
type NativeVectorSearchSkill struct {
	skills.BaseSkill
	toolName           string
	remoteURL          string
	remoteBaseURL      string
	remoteAuth         string // Base64-encoded "user:pass" for Authorization header, empty if no auth
	indexName          string
	count              int
	noResults          string
	similarityThreshold float64
	tags               []string
	responsePrefix     string
	responsePostfix    string
	maxContentLength   int
	logger             *slog.Logger
}

// NewNativeVectorSearch creates a new NativeVectorSearchSkill.
func NewNativeVectorSearch(params map[string]any) skills.SkillBase {
	return &NativeVectorSearchSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "native_vector_search",
			SkillDesc: "Search document indexes using vector similarity and keyword search (local or remote)",
			SkillVer:  "1.0.0",
			Params:    params,
		},
		logger: slog.Default(),
	}
}

func (s *NativeVectorSearchSkill) SupportsMultipleInstances() bool { return true }

func (s *NativeVectorSearchSkill) GetInstanceKey() string {
	toolName := s.GetParamString("tool_name", "search_knowledge")
	indexName := s.GetParamString("index_name", "default")
	return "native_vector_search_" + toolName + "_" + indexName
}

// validateRemoteURL performs SSRF protection on the remote URL. Mirrors
// Python's validate_url() from signalwire/utils/url_validator.py:
//   - Requires http/https scheme and a hostname.
//   - Resolves hostname to IPs; rejects on DNS failure unless
//     SWML_ALLOW_PRIVATE_URLS is set (matches Python's allow_private/env-var
//     escape hatch for test environments).
//   - Rejects IPs in private/loopback/link-local/IPv6-private ranges.
func validateRemoteURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got %q", parsed.Scheme)
	}
	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL has no hostname")
	}

	// Opt-in escape hatch for test environments. Matches Python's
	// SWML_ALLOW_PRIVATE_URLS check in url_validator.validate_url.
	if allowPrivateURLs() {
		return nil
	}

	// Resolve hostname to IPs. DNS failure rejects — matches Python's
	// socket.gaierror → return False.
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return fmt.Errorf("could not resolve hostname %q: %w", hostname, err)
	}

	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("URL resolves to a loopback/link-local address: %s", addr)
		}
		if ip.IsPrivate() {
			return fmt.Errorf("URL resolves to a private IP address: %s", addr)
		}
	}
	return nil
}

// allowPrivateURLs reports whether SWML_ALLOW_PRIVATE_URLS is set to a
// truthy value. Matches Python's env-var check.
func allowPrivateURLs() bool {
	switch strings.ToLower(os.Getenv("SWML_ALLOW_PRIVATE_URLS")) {
	case "1", "true", "yes":
		return true
	}
	return false
}

func (s *NativeVectorSearchSkill) Setup() bool {
	s.toolName = s.GetParamString("tool_name", "search_knowledge")
	s.remoteURL = s.GetParamString("remote_url", "")
	s.indexName = s.GetParamString("index_name", "default")
	s.count = s.GetParamInt("count", 5)
	s.noResults = s.GetParamString("no_results_message", "No information found for '{query}'")
	s.similarityThreshold = s.GetParamFloat("similarity_threshold", 0.0)
	s.responsePrefix = s.GetParamString("response_prefix", "")
	s.responsePostfix = s.GetParamString("response_postfix", "")
	s.maxContentLength = s.GetParamInt("max_content_length", 32768)

	// Parse tags param
	s.tags = nil
	if rawTags, ok := s.Params["tags"]; ok {
		if tagSlice, ok := rawTags.([]any); ok {
			for _, t := range tagSlice {
				if ts, ok := t.(string); ok {
					s.tags = append(s.tags, ts)
				}
			}
		}
	}

	if s.remoteURL == "" {
		s.logger.Error("native_vector_search: remote_url is required")
		return false
	}

	// SSRF guard: validate URL before connecting
	if err := validateRemoteURL(s.remoteURL); err != nil {
		s.logger.Error("native_vector_search: remote_url rejected by SSRF protection", "error", err)
		return false
	}

	// Parse auth from URL (user:pass@host pattern)
	// Mirrors Python: urlparse extracts username/password, reconstructs clean base URL
	s.remoteAuth = ""
	s.remoteBaseURL = s.remoteURL
	parsed, err := url.Parse(s.remoteURL)
	if err == nil && parsed.User != nil {
		user := parsed.User.Username()
		pass, _ := parsed.User.Password()
		if user != "" {
			credentials := user + ":" + pass
			s.remoteAuth = base64.StdEncoding.EncodeToString([]byte(credentials))
			// Reconstruct URL without embedded credentials
			parsed.User = nil
			s.remoteBaseURL = parsed.String()
		}
	}

	// Strip trailing slash from base URL
	s.remoteBaseURL = strings.TrimRight(s.remoteBaseURL, "/")

	// Test connection to health endpoint
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", s.remoteBaseURL+"/health", nil)
	if err != nil {
		s.logger.Error("native_vector_search: failed to build health request", "error", err)
		return false
	}
	if s.remoteAuth != "" {
		req.Header.Set("Authorization", "Basic "+s.remoteAuth)
	}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("native_vector_search: failed to connect to remote search server", "error", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		s.logger.Error("native_vector_search: authentication failed for remote search server")
		return false
	}
	if resp.StatusCode != http.StatusOK {
		s.logger.Error("native_vector_search: remote search server returned non-200 status", "status", resp.StatusCode)
		return false
	}

	s.logger.Info("native_vector_search: remote search server available", "url", s.remoteBaseURL)
	return true
}

func (s *NativeVectorSearchSkill) RegisterTools() []skills.ToolRegistration {
	desc := s.GetParamString("description", "Search the knowledge base for information")
	return []skills.ToolRegistration{
		{
			Name:        s.toolName,
			Description: desc,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query or question",
					},
					"count": map[string]any{
						"type":        "integer",
						"description": fmt.Sprintf("Number of results to return (default: %d)", s.count),
						"default":     s.count,
					},
				},
				"required": []string{"query"},
			},
			Handler: s.handleSearch,
		},
	}
}

func (s *NativeVectorSearchSkill) handleSearch(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	query, _ := args["query"].(string)
	if query == "" {
		return swaig.NewFunctionResult("Please provide a search query.")
	}

	// Allow per-call count override from AI (mirrors Python's count = args.get('count', self.count))
	count := s.count
	if argCount, ok := args["count"]; ok {
		switch n := argCount.(type) {
		case int:
			count = n
		case float64:
			count = int(n)
		}
	}

	searchReq := map[string]any{
		"query":                query,
		"index_name":           s.indexName,
		"count":                count,
		"similarity_threshold": s.similarityThreshold,
		"tags":                 s.tags,
	}

	bodyBytes, _ := json.Marshal(searchReq)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", s.remoteBaseURL+"/search", strings.NewReader(string(bodyBytes)))
	if err != nil {
		s.logger.Error("native_vector_search: failed to build search request", "error", err)
		return swaig.NewFunctionResult("Search service is temporarily unavailable. Please try again later.")
	}
	req.Header.Set("Content-Type", "application/json")
	if s.remoteAuth != "" {
		req.Header.Set("Authorization", "Basic "+s.remoteAuth)
	}

	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("native_vector_search: search request failed", "error", err)
		return swaig.NewFunctionResult("Search service is temporarily unavailable. Please try again later.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("native_vector_search: search server returned error", "status", resp.StatusCode)
		return swaig.NewFunctionResult("Search service returned an error. Please try again later.")
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		s.logger.Error("native_vector_search: failed to decode search response", "error", err)
		return swaig.NewFunctionResult("Error processing search results.")
	}

	results, _ := data["results"].([]any)
	if len(results) == 0 {
		msg := strings.ReplaceAll(s.noResults, "{query}", query)
		if s.responsePrefix != "" {
			msg = s.responsePrefix + " " + msg
		}
		if s.responsePostfix != "" {
			msg = msg + " " + s.responsePostfix
		}
		return swaig.NewFunctionResult(msg)
	}

	// Calculate per-result content budget (mirrors Python)
	estimatedOverheadPerResult := 300
	prefixPostfixOverhead := len(s.responsePrefix) + len(s.responsePostfix) + 100
	totalOverhead := (len(results) * estimatedOverheadPerResult) + prefixPostfixOverhead
	availableForContent := s.maxContentLength - totalOverhead
	perResultLimit := 1000
	if len(results) > 0 {
		candidate := availableForContent / len(results)
		if candidate > 500 {
			perResultLimit = candidate
		} else {
			perResultLimit = 500
		}
	}

	var sb strings.Builder
	if s.responsePrefix != "" {
		sb.WriteString(s.responsePrefix)
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("Found %d results for '%s':\n\n", len(results), query))

	for i, r := range results {
		m, _ := r.(map[string]any)
		if m == nil {
			continue
		}
		content, _ := m["content"].(string)
		score, _ := m["score"].(float64)
		metadata, _ := m["metadata"].(map[string]any)
		filename := ""
		section := ""
		if metadata != nil {
			filename, _ = metadata["filename"].(string)
			section, _ = metadata["section"].(string)
		}

		// Truncate content to per-result limit
		if len(content) > perResultLimit {
			content = content[:perResultLimit] + "..."
		}

		resultText := fmt.Sprintf("**Result %d** (from %s", i+1, filename)
		if section != "" {
			resultText += ", section: " + section
		}
		resultText += fmt.Sprintf(", relevance: %.2f)\n%s\n\n", score, content)
		sb.WriteString(resultText)
	}

	if s.responsePostfix != "" {
		sb.WriteString(s.responsePostfix)
	}

	return swaig.NewFunctionResult(sb.String())
}

func (s *NativeVectorSearchSkill) GetHints() []string {
	hints := []string{"search", "find", "look up", "documentation", "knowledge base"}
	if customHints, ok := s.Params["hints"].([]any); ok {
		for _, h := range customHints {
			if hs, ok := h.(string); ok {
				hints = append(hints, hs)
			}
		}
	}
	return hints
}

func (s *NativeVectorSearchSkill) GetPromptSections() []map[string]any {
	// Honor skip_prompt param (mirrors Python SkillBase.get_prompt_sections() check)
	if s.GetParamBool("skip_prompt", false) {
		return nil
	}
	return []map[string]any{
		{
			"title": "Knowledge Search",
			"body":  "You can search knowledge sources using " + s.toolName + ".",
			"bullets": []string{
				"Use " + s.toolName + " to search document indexes",
				"Search for relevant information using clear, specific queries",
				"If no results are found, suggest the user try rephrasing their question",
			},
		},
	}
}

func (s *NativeVectorSearchSkill) GetGlobalData() map[string]any {
	// Python returns {} for remote mode (no local search engine stats available).
	// Go returns an equivalent non-nil map with a mode sentinel.
	return map[string]any{"search_mode": "remote"}
}

func (s *NativeVectorSearchSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["remote_url"] = map[string]any{
		"type":        "string",
		"description": "URL of remote search server (e.g., http://localhost:8001 or http://user:pass@host:8001)",
		"required":    true,
	}
	schema["index_name"] = map[string]any{
		"type":        "string",
		"description": "Name of index on remote server",
		"default":     "default",
		"required":    false,
	}
	schema["count"] = map[string]any{
		"type":        "integer",
		"description": "Number of search results to return",
		"default":     5,
		"required":    false,
		"minimum":     1,
		"maximum":     20,
	}
	schema["similarity_threshold"] = map[string]any{
		"type":        "number",
		"description": "Minimum similarity score for results (0.0 = no limit, 1.0 = exact match)",
		"default":     0.0,
		"required":    false,
		"minimum":     0.0,
		"maximum":     1.0,
	}
	schema["tags"] = map[string]any{
		"type":        "array",
		"description": "Tags to filter search results",
		"default":     []any{},
		"required":    false,
		"items":       map[string]any{"type": "string"},
	}
	schema["response_prefix"] = map[string]any{
		"type":        "string",
		"description": "Prefix to add to search results",
		"default":     "",
		"required":    false,
	}
	schema["response_postfix"] = map[string]any{
		"type":        "string",
		"description": "Postfix to add to search results",
		"default":     "",
		"required":    false,
	}
	schema["max_content_length"] = map[string]any{
		"type":        "integer",
		"description": "Maximum total response size in characters (distributed across all results)",
		"default":     32768,
		"required":    false,
		"minimum":     1000,
	}
	schema["no_results_message"] = map[string]any{
		"type":        "string",
		"description": "Message when no results are found. Use {query} as placeholder.",
		"default":     "No information found for '{query}'",
		"required":    false,
	}
	schema["hints"] = map[string]any{
		"type":        "array",
		"description": "Speech recognition hints for this skill",
		"default":     []any{},
		"required":    false,
		"items":       map[string]any{"type": "string"},
	}
	schema["description"] = map[string]any{
		"type":        "string",
		"description": "Tool description shown to the AI",
		"default":     "Search the knowledge base for information",
		"required":    false,
	}
	return schema
}

func init() {
	skills.RegisterSkill("native_vector_search", NewNativeVectorSearch)
}
