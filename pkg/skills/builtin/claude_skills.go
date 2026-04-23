package builtin

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
	"gopkg.in/yaml.v3"
)

// _unsupportedFields maps SKILL.md frontmatter field names to warning message templates.
var _unsupportedFields = map[string]string{
	"context":      "context: fork is not supported in SignalWire agents — skill '%s' will run inline, not in a subagent",
	"agent":        "agent field is not supported in SignalWire agents — skill '%s' cannot select a subagent type",
	"allowed-tools": "allowed-tools is not supported in SignalWire agents — skill '%s' tool restrictions will not be enforced",
	"model":        "model field is not supported in SignalWire agents — skill '%s' model selection is controlled at the agent level",
	"hooks":        "hooks field is not supported in SignalWire agents — skill '%s' lifecycle hooks will not fire",
}

// _shellInjectionRE matches !`command` patterns in skill bodies.
var _shellInjectionRE = regexp.MustCompile("!`([^`]+)`")

// _sanitizeRE matches characters invalid in SWAIG tool names.
var _sanitizeRE = regexp.MustCompile(`[^a-z0-9_]`)

// _hyphenSpaceRE matches hyphens and spaces for sanitization.
var _hyphenSpaceRE = regexp.MustCompile(`[-\s]+`)

// _digitStartRE checks if a string starts with a digit.
var _digitStartRE = regexp.MustCompile(`^[0-9]`)

// skillEntry holds a parsed SKILL.md file and its metadata.
type skillEntry struct {
	name        string
	description string
	body        string

	// Invocation control
	disableModelInvocation bool
	userInvocable          bool
	argumentHint           string

	// Informational
	license       string
	compatibility string

	// Unsupported parsed fields (logged as warnings)
	context      interface{}
	agent        interface{}
	allowedTools interface{}
	model        interface{}
	hooks        interface{}

	// Discovered supporting files
	sections map[string]string // section name → absolute file path

	// Non-.md files (when allow_script_execution is true)
	files map[string][]string // "scripts"|"assets"|"other" → []relative path

	// Directory containing this skill
	skillDir string

	// Invocation control decisions
	skipTool   bool
	skipPrompt bool
}

// ClaudeSkillsSkill loads Claude-style SKILL.md files as SignalWire agent tools.
//
// Each directory under skills_path that contains a SKILL.md becomes a SWAIG tool.
// The SKILL.md frontmatter provides the tool name and description; the body
// becomes the tool's response content when invoked.
type ClaudeSkillsSkill struct {
	skills.BaseSkill

	// Configuration
	skillsPath            string
	includePatterns       []string
	excludePatterns       []string
	toolPrefix            string
	responsePrefix        string
	responsePostfix       string
	skillDescriptions     map[string]string
	allowShellInjection   bool
	allowScriptExecution  bool
	ignoreInvocationControl bool
	shellTimeout          int

	// Discovered skills
	loadedSkills []skillEntry
}

// NewClaudeSkills creates a new ClaudeSkillsSkill.
func NewClaudeSkills(params map[string]any) skills.SkillBase {
	return &ClaudeSkillsSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "claude_skills",
			SkillDesc: "Load Claude SKILL.md files as agent tools",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

// SupportsMultipleInstances returns true — different skills_path values create distinct instances.
func (s *ClaudeSkillsSkill) SupportsMultipleInstances() bool { return true }

// RequiredEnvVars returns nil — no env vars required (skills_path is a param).
func (s *ClaudeSkillsSkill) RequiredEnvVars() []string { return nil }

// Setup validates configuration and discovers SKILL.md files.
func (s *ClaudeSkillsSkill) Setup() bool {
	skillsPath := s.GetParamString("skills_path", "")
	if skillsPath == "" {
		slog.Error("claude_skills: skills_path parameter is required")
		return false
	}

	// Expand home directory and resolve to absolute path.
	if strings.HasPrefix(skillsPath, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			skillsPath = filepath.Join(home, skillsPath[2:])
		}
	}
	abs, err := filepath.Abs(skillsPath)
	if err != nil {
		slog.Error("claude_skills: failed to resolve skills_path", "path", skillsPath, "error", err)
		return false
	}
	s.skillsPath = abs

	info, err := os.Stat(s.skillsPath)
	if err != nil {
		slog.Error("claude_skills: skills_path does not exist", "path", s.skillsPath)
		return false
	}
	if !info.IsDir() {
		slog.Error("claude_skills: skills_path is not a directory", "path", s.skillsPath)
		return false
	}

	// Include/exclude glob patterns.
	s.includePatterns = getStringSliceParam(s.Params, "include", []string{"*"})
	s.excludePatterns = getStringSliceParam(s.Params, "exclude", []string{})

	// Safety flags.
	s.allowShellInjection = s.GetParamBool("allow_shell_injection", false)
	s.allowScriptExecution = s.GetParamBool("allow_script_execution", false)
	s.ignoreInvocationControl = s.GetParamBool("ignore_invocation_control", false)
	s.shellTimeout = s.GetParamInt("shell_timeout", 30)

	// Display params.
	s.toolPrefix = s.GetParamString("tool_prefix", "claude_")
	s.responsePrefix = s.GetParamString("response_prefix", "")
	s.responsePostfix = s.GetParamString("response_postfix", "")
	s.skillDescriptions = getStringMapParam(s.Params, "skill_descriptions")

	if s.allowShellInjection {
		slog.Warn("claude_skills: allow_shell_injection is enabled — skill bodies may execute arbitrary shell commands")
	}

	// Discover and parse all skills.
	s.loadedSkills = s.discoverSkills()

	if len(s.loadedSkills) == 0 {
		slog.Warn("claude_skills: no skills found", "path", s.skillsPath)
		// Return true anyway — empty skill set is valid.
	}

	slog.Info("claude_skills: loaded skills", "count", len(s.loadedSkills), "path", s.skillsPath)
	return true
}

// discoverSkills walks skills_path and parses each subdirectory containing SKILL.md.
func (s *ClaudeSkillsSkill) discoverSkills() []skillEntry {
	var result []skillEntry

	entries, err := os.ReadDir(s.skillsPath)
	if err != nil {
		slog.Error("claude_skills: failed to read skills_path", "error", err)
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(s.skillsPath, entry.Name())
		skillFile := filepath.Join(skillDir, "SKILL.md")

		if _, err := os.Stat(skillFile); err != nil {
			continue
		}

		// Apply include/exclude patterns against directory name.
		if !s.matchesPatterns(entry.Name()) {
			slog.Debug("claude_skills: skipping skill (excluded by patterns)", "name", entry.Name())
			continue
		}

		parsed, err := s.parseSkillMD(skillFile)
		if err != nil {
			slog.Error("claude_skills: failed to parse skill", "file", skillFile, "error", err)
			continue
		}

		// Use directory name as fallback skill name.
		if parsed.name == "" {
			parsed.name = entry.Name()
		}
		parsed.skillDir = skillDir

		// Discover supporting .md sections.
		parsed.sections = s.discoverSections(skillDir)

		// Discover non-.md files if script execution is enabled.
		if s.allowScriptExecution {
			parsed.files = s.discoverAllFiles(skillDir)
		} else {
			parsed.files = map[string][]string{}
		}

		// Warn about unsupported frontmatter fields.
		s.warnUnsupportedFields(parsed)

		// Warn about shell injection patterns when disabled.
		if !s.allowShellInjection {
			s.warnShellPatterns(parsed)
		}

		// Determine invocation control flags.
		s.applyInvocationControl(parsed)

		sectionCount := len(parsed.sections)
		slog.Debug("claude_skills: loaded skill", "name", parsed.name, "file", skillFile, "sections", sectionCount)
		result = append(result, *parsed)
	}

	return result
}

// discoverSections finds all .md files under skillDir (excluding SKILL.md) and
// maps a relative key (e.g. "intro" or "references/api") to the absolute path.
func (s *ClaudeSkillsSkill) discoverSections(skillDir string) map[string]string {
	sections := make(map[string]string)

	err := filepath.Walk(skillDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}
		if strings.ToUpper(info.Name()) == "SKILL.MD" {
			return nil
		}

		// Relative path from skillDir.
		rel, err := filepath.Rel(skillDir, path)
		if err != nil {
			return nil
		}

		// Build key: parent/stem for nested, just stem for top-level.
		dir := filepath.Dir(rel)
		stem := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
		var key string
		if dir == "." {
			key = stem
		} else {
			key = filepath.ToSlash(filepath.Join(dir, stem))
		}

		sections[key] = path
		return nil
	})
	if err != nil {
		slog.Error("claude_skills: error walking skill directory", "dir", skillDir, "error", err)
	}
	return sections
}

// discoverAllFiles categorizes non-.md files under skillDir into scripts/assets/other.
func (s *ClaudeSkillsSkill) discoverAllFiles(skillDir string) map[string][]string {
	files := map[string][]string{
		"scripts": {},
		"assets":  {},
		"other":   {},
	}

	err := filepath.Walk(skillDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}

		rel, err := filepath.Rel(skillDir, path)
		if err != nil {
			return nil
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")

		// Skip hidden files and __pycache__.
		for _, p := range parts {
			if strings.HasPrefix(p, ".") || p == "__pycache__" {
				return nil
			}
		}

		relStr := filepath.ToSlash(rel)
		switch parts[0] {
		case "scripts":
			files["scripts"] = append(files["scripts"], relStr)
		case "assets":
			files["assets"] = append(files["assets"], relStr)
		default:
			files["other"] = append(files["other"], relStr)
		}
		return nil
	})
	if err != nil {
		slog.Error("claude_skills: error walking for files", "dir", skillDir, "error", err)
	}

	for k := range files {
		sort.Strings(files[k])
	}
	return files
}

// matchesPatterns returns true if name matches any include pattern and no exclude pattern.
func (s *ClaudeSkillsSkill) matchesPatterns(name string) bool {
	for _, pattern := range s.excludePatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return false
		}
	}
	for _, pattern := range s.includePatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

// parseSkillMD reads and parses a SKILL.md file with optional YAML frontmatter.
func (s *ClaudeSkillsSkill) parseSkillMD(path string) (*skillEntry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	text := string(content)
	entry := &skillEntry{
		userInvocable: true,
	}

	if !strings.HasPrefix(text, "---") {
		// No frontmatter — treat entire content as body.
		entry.body = strings.TrimSpace(text)
		return entry, nil
	}

	// Split on "---" delimiter: ["", frontmatter, body...]
	parts := strings.SplitN(text, "---", 3)
	if len(parts) < 3 {
		slog.Warn("claude_skills: malformed frontmatter", "file", path)
		entry.body = strings.TrimSpace(text)
		return entry, nil
	}

	frontmatterStr := strings.TrimSpace(parts[1])
	body := strings.TrimSpace(parts[2])

	// Parse YAML frontmatter.
	var fm map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatterStr), &fm); err != nil {
		slog.Warn("claude_skills: failed to parse YAML frontmatter", "file", path, "error", err)
		fm = make(map[string]interface{})
	}
	if fm == nil {
		fm = make(map[string]interface{})
	}

	entry.body = body
	entry.name = getString(fm, "name")
	entry.description = getString(fm, "description")
	entry.disableModelInvocation = getBool(fm, "disable-model-invocation", false)
	entry.userInvocable = getBool(fm, "user-invocable", true)
	entry.argumentHint = getString(fm, "argument-hint")
	entry.license = getString(fm, "license")
	entry.compatibility = getString(fm, "compatibility")

	// Unsupported fields — stored for warning emission.
	entry.context = fm["context"]
	entry.agent = fm["agent"]
	entry.allowedTools = fm["allowed-tools"]
	entry.model = fm["model"]
	entry.hooks = fm["hooks"]

	return entry, nil
}

// warnUnsupportedFields logs warnings for set frontmatter fields not supported in SignalWire agents.
func (s *ClaudeSkillsSkill) warnUnsupportedFields(parsed *skillEntry) {
	name := parsed.name
	if name == "" {
		name = "unknown"
	}

	type check struct {
		key   string
		value interface{}
	}
	checks := []check{
		{"context", parsed.context},
		{"agent", parsed.agent},
		{"allowed-tools", parsed.allowedTools},
		{"model", parsed.model},
		{"hooks", parsed.hooks},
	}

	for _, c := range checks {
		if c.value != nil {
			msg := fmt.Sprintf(_unsupportedFields[c.key], name)
			slog.Warn("claude_skills: " + msg)
		}
	}

	if parsed.license != "" {
		slog.Debug("claude_skills: skill has license", "name", name, "license", parsed.license)
	}
	if parsed.compatibility != "" {
		slog.Debug("claude_skills: skill has compatibility", "name", name, "compatibility", parsed.compatibility)
	}
}

// warnShellPatterns warns about !`command` patterns when shell injection is disabled.
func (s *ClaudeSkillsSkill) warnShellPatterns(parsed *skillEntry) {
	name := parsed.name
	if name == "" {
		name = "unknown"
	}
	matches := _shellInjectionRE.FindAllStringSubmatch(parsed.body, -1)
	for _, m := range matches {
		slog.Warn("claude_skills: shell injection pattern found but allow_shell_injection is disabled — pattern will be passed through as-is",
			"command", m[1], "skill", name)
	}
}

// applyInvocationControl sets skipTool and skipPrompt on the skill entry
// based on disable-model-invocation and user-invocable frontmatter flags.
func (s *ClaudeSkillsSkill) applyInvocationControl(parsed *skillEntry) {
	if s.ignoreInvocationControl {
		parsed.skipTool = false
		parsed.skipPrompt = false
		return
	}

	if parsed.disableModelInvocation {
		// disable-model-invocation: true → no tool, no prompt.
		parsed.skipTool = true
		parsed.skipPrompt = true
		slog.Debug("claude_skills: skill has disable-model-invocation=true — skipping tool and prompt", "name", parsed.name)
	} else if !parsed.userInvocable {
		// user-invocable: false → no tool, yes prompt (knowledge-only).
		parsed.skipTool = true
		parsed.skipPrompt = false
		slog.Debug("claude_skills: skill has user-invocable=false — skipping tool, keeping prompt", "name", parsed.name)
	} else {
		parsed.skipTool = false
		parsed.skipPrompt = false
	}
}

// sanitizeToolName converts a skill name to a valid SWAIG tool name.
func (s *ClaudeSkillsSkill) sanitizeToolName(name string) string {
	lower := strings.ToLower(name)
	sanitized := _hyphenSpaceRE.ReplaceAllString(lower, "_")
	sanitized = _sanitizeRE.ReplaceAllString(sanitized, "")
	if _digitStartRE.MatchString(sanitized) {
		sanitized = "_" + sanitized
	}
	if sanitized == "" {
		return "unnamed"
	}
	return sanitized
}

// executeShellInjection replaces !`command` patterns with the command's stdout.
func (s *ClaudeSkillsSkill) executeShellInjection(content, skillDir string, timeout int) string {
	return _shellInjectionRE.ReplaceAllStringFunc(content, func(match string) string {
		sub := _shellInjectionRE.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		command := sub[1]
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Dir = skillDir
		out, err := cmd.Output()
		if ctx.Err() == context.DeadlineExceeded {
			slog.Error("claude_skills: shell command timed out", "command", command, "timeout", timeout)
			return fmt.Sprintf("[command timed out: %s]", command)
		}
		if err != nil {
			slog.Error("claude_skills: shell command failed", "command", command, "error", err)
			return fmt.Sprintf("[command error: %s]", command)
		}
		return strings.TrimRight(string(out), "\n")
	})
}

// substituteVariables replaces ${CLAUDE_SKILL_DIR} and ${CLAUDE_SESSION_ID} placeholders.
func (s *ClaudeSkillsSkill) substituteVariables(content, skillDir string, rawData map[string]any) string {
	content = strings.ReplaceAll(content, "${CLAUDE_SKILL_DIR}", skillDir)

	sessionID := ""
	if rawData != nil {
		if id, ok := rawData["call_id"].(string); ok {
			sessionID = id
		}
	}
	content = strings.ReplaceAll(content, "${CLAUDE_SESSION_ID}", sessionID)
	return content
}

// substituteArguments replaces $ARGUMENTS, $ARGUMENTS[N], and $N placeholders in body.
func (s *ClaudeSkillsSkill) substituteArguments(body, arguments string) string {
	if arguments == "" {
		arguments = ""
	}

	// Track whether body had a bare $ARGUMENTS (not indexed) before substitution.
	bareRE := regexp.MustCompile(`\$ARGUMENTS(?!\[)`)
	hasBareArguments := bareRE.MatchString(body)

	// Split into positional args.
	var positional []string
	if arguments != "" {
		positional = strings.Fields(arguments)
	}

	// Replace $ARGUMENTS[N] with positional args.
	indexedRE := regexp.MustCompile(`\$ARGUMENTS\[(\d+)\]`)
	result := indexedRE.ReplaceAllStringFunc(body, func(match string) string {
		sub := indexedRE.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		var idx int
		fmt.Sscanf(sub[1], "%d", &idx)
		if idx < len(positional) {
			return positional[idx]
		}
		return ""
	})

	// Replace $N shorthand (must come after $ARGUMENTS to avoid conflicts).
	shorthandRE := regexp.MustCompile(`\$(\d+)(?:\D|$)`)
	result = shorthandRE.ReplaceAllStringFunc(result, func(match string) string {
		sub := shorthandRE.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		var idx int
		fmt.Sscanf(sub[1], "%d", &idx)
		// Preserve trailing non-digit character.
		suffix := ""
		if len(match) > len(sub[1])+1 {
			suffix = string(match[len(match)-1])
		}
		if idx < len(positional) {
			return positional[idx] + suffix
		}
		return suffix
	})

	// Replace bare $ARGUMENTS with full string.
	result = strings.ReplaceAll(result, "$ARGUMENTS", arguments)

	// Fallback: append arguments if body had no bare $ARGUMENTS placeholder.
	if !hasBareArguments && arguments != "" {
		result += "\n\nARGUMENTS: " + arguments
	}

	return result
}

// RegisterTools returns one ToolRegistration per discovered SKILL.md that is not flagged to skip.
func (s *ClaudeSkillsSkill) RegisterTools() []skills.ToolRegistration {
	var registrations []skills.ToolRegistration

	for i := range s.loadedSkills {
		skill := &s.loadedSkills[i]

		if skill.skipTool {
			slog.Debug("claude_skills: skipping tool registration (invocation control)", "name", skill.name)
			continue
		}

		toolName := s.toolPrefix + s.sanitizeToolName(skill.name)

		// Description with override support.
		description := s.skillDescriptions[skill.name]
		if description == "" {
			description = skill.description
		}
		if description == "" {
			description = "Use the " + skill.name + " skill"
		}

		// Parameters: always include "arguments", optionally "section" if sections exist.
		argumentHint := skill.argumentHint
		if argumentHint == "" {
			argumentHint = "Arguments or context to pass to the skill"
		}
		properties := map[string]any{
			"arguments": map[string]any{
				"type":        "string",
				"description": argumentHint,
			},
		}

		sectionNames := sortedKeys(skill.sections)
		if len(sectionNames) > 0 {
			properties["section"] = map[string]any{
				"type":        "string",
				"description": "Which reference section to load",
				"enum":        sectionNames,
			}
		}

		parameters := map[string]any{
			"type":       "object",
			"properties": properties,
		}

		// Capture loop variables for handler closure.
		capturedSkill := skill
		capturedPrefix := s.responsePrefix
		capturedPostfix := s.responsePostfix

		handler := func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			section, _ := args["section"].(string)
			arguments, _ := args["arguments"].(string)

			var content string
			if section != "" {
				if sectionPath, ok := capturedSkill.sections[section]; ok {
					data, err := os.ReadFile(sectionPath)
					if err != nil {
						slog.Error("claude_skills: failed to read section", "section", section, "error", err)
						content = fmt.Sprintf("Error loading section '%s'", section)
					} else {
						content = string(data)
					}
				} else {
					content = capturedSkill.body
				}
			} else {
				content = capturedSkill.body
			}

			skillDir := capturedSkill.skillDir

			// 1. Shell injection (if enabled).
			if s.allowShellInjection {
				content = s.executeShellInjection(content, skillDir, s.shellTimeout)
			}

			// 2. Variable substitution.
			content = s.substituteVariables(content, skillDir, rawData)

			// 3. Argument substitution (with fallback append).
			content = s.substituteArguments(content, arguments)

			// 4. Prefix/postfix wrapping.
			if capturedPrefix != "" || capturedPostfix != "" {
				var parts []string
				if capturedPrefix != "" {
					parts = append(parts, capturedPrefix)
				}
				parts = append(parts, content)
				if capturedPostfix != "" {
					parts = append(parts, capturedPostfix)
				}
				content = strings.Join(parts, "\n\n")
			}

			return swaig.NewFunctionResult(content)
		}

		slog.Debug("claude_skills: registered tool", "tool", toolName, "sections", sectionNames)
		registrations = append(registrations, skills.ToolRegistration{
			Name:        toolName,
			Description: description,
			Parameters:  parameters,
			Handler:     handler,
		})
	}

	return registrations
}

// GetHints returns speech recognition hints derived from loaded skill names.
func (s *ClaudeSkillsSkill) GetHints() []string {
	seen := make(map[string]bool)
	var hints []string
	for _, skill := range s.loadedSkills {
		words := strings.Fields(strings.NewReplacer("-", " ", "_", " ").Replace(skill.name))
		for _, w := range words {
			if !seen[w] {
				seen[w] = true
				hints = append(hints, w)
			}
		}
	}
	return hints
}

// GetPromptSections returns one prompt section per loaded skill (excluding skipped ones).
func (s *ClaudeSkillsSkill) GetPromptSections() []map[string]any {
	if len(s.loadedSkills) == 0 {
		return nil
	}

	var sections []map[string]any
	for i := range s.loadedSkills {
		skill := &s.loadedSkills[i]

		if skill.skipPrompt {
			continue
		}

		toolName := s.toolPrefix + s.sanitizeToolName(skill.name)
		hasTool := !skill.skipTool

		body := skill.body

		// Append available sections if any.
		skillSections := sortedKeys(skill.sections)
		if len(skillSections) > 0 && hasTool {
			sectionList := strings.Join(skillSections, ", ")
			body += "\n\nAvailable reference sections: " + sectionList
			body += "\nCall " + toolName + "(section=\"<name>\") to load a section."
		}

		// Append discovered files if script execution is enabled.
		if s.allowScriptExecution {
			for _, category := range []string{"scripts", "assets", "other"} {
				fileList := skill.files[category]
				if len(fileList) > 0 {
					// ASCII title-case (strings.Title is deprecated).
					// category is always lowercase ASCII (scripts/assets/other),
					// so upper-first-byte is sufficient.
					label := strings.ToUpper(category[:1]) + category[1:]
					if category == "other" {
						label = "Other files"
					}
					body += "\n\n" + label + ": " + strings.Join(fileList, ", ")
				}
			}
		}

		sections = append(sections, map[string]any{
			"title": skill.name,
			"body":  body,
		})
	}

	return sections
}

// GetInstanceKey returns a unique key based on skills_path for multi-instance support.
func (s *ClaudeSkillsSkill) GetInstanceKey() string {
	skillsPath := s.GetParamString("skills_path", "default")
	h := simpleHash(skillsPath)
	return fmt.Sprintf("claude_skills_%d", h%10000)
}

// GetParameterSchema returns the full parameter schema for the SKILL.md loader.
func (s *ClaudeSkillsSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()

	schema["skills_path"] = map[string]any{
		"type":        "string",
		"description": "Path to directory containing Claude skill folders (each with SKILL.md)",
		"required":    true,
	}
	schema["include"] = map[string]any{
		"type":        "array",
		"description": "Glob patterns for skills to include (default: ['*'])",
		"default":     []string{"*"},
		"required":    false,
	}
	schema["exclude"] = map[string]any{
		"type":        "array",
		"description": "Glob patterns for skills to exclude",
		"default":     []string{},
		"required":    false,
	}
	schema["prompt_title"] = map[string]any{
		"type":        "string",
		"description": "Title for the prompt section listing skills",
		"default":     "Claude Skills",
		"required":    false,
	}
	schema["prompt_intro"] = map[string]any{
		"type":        "string",
		"description": "Introductory text for the prompt section",
		"default":     "You have access to specialized skills. Call the appropriate tool when the user's question matches:",
		"required":    false,
	}
	schema["skill_descriptions"] = map[string]any{
		"type":        "object",
		"description": "Override descriptions for specific skills (skill_name -> description)",
		"default":     map[string]string{},
		"required":    false,
	}
	schema["tool_prefix"] = map[string]any{
		"type":        "string",
		"description": "Prefix for generated tool names (default: 'claude_'). Use empty string for no prefix.",
		"default":     "claude_",
		"required":    false,
	}
	schema["response_prefix"] = map[string]any{
		"type":        "string",
		"description": "Text to prepend to skill results",
		"default":     "",
		"required":    false,
	}
	schema["response_postfix"] = map[string]any{
		"type":        "string",
		"description": "Text to append to skill results",
		"default":     "",
		"required":    false,
	}
	schema["allow_shell_injection"] = map[string]any{
		"type":        "boolean",
		"description": "Enable !`command` preprocessing in skill bodies. DANGEROUS: allows arbitrary shell execution.",
		"default":     false,
		"required":    false,
	}
	schema["allow_script_execution"] = map[string]any{
		"type":        "boolean",
		"description": "Discover and list scripts/, assets/ files in prompt sections",
		"default":     false,
		"required":    false,
	}
	schema["ignore_invocation_control"] = map[string]any{
		"type":        "boolean",
		"description": "Override disable-model-invocation and user-invocable flags, register everything",
		"default":     false,
		"required":    false,
	}
	schema["shell_timeout"] = map[string]any{
		"type":        "integer",
		"description": "Timeout in seconds for shell injection commands",
		"default":     30,
		"required":    false,
	}

	return schema
}

// ---- helper functions -------------------------------------------------------

// getStringSliceParam extracts a []string param from a map, with a default fallback.
func getStringSliceParam(params map[string]any, key string, defaultVal []string) []string {
	if params == nil {
		return defaultVal
	}
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	switch tv := v.(type) {
	case []string:
		return tv
	case []any:
		result := make([]string, 0, len(tv))
		for _, item := range tv {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return defaultVal
}

// getStringMapParam extracts a map[string]string param from a map.
func getStringMapParam(params map[string]any, key string) map[string]string {
	result := make(map[string]string)
	if params == nil {
		return result
	}
	v, ok := params[key]
	if !ok {
		return result
	}
	switch tv := v.(type) {
	case map[string]string:
		return tv
	case map[string]any:
		for k, val := range tv {
			if s, ok := val.(string); ok {
				result[k] = s
			}
		}
	}
	return result
}

// getString extracts a string value from a map[string]interface{}.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getBool extracts a bool value from a map[string]interface{}, with a default.
func getBool(m map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}

// sortedKeys returns sorted keys from a map[string]string.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// simpleHash returns a simple hash of a string (equivalent to Python's hash() % 10000 idiom).
func simpleHash(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

func init() {
	skills.RegisterSkill("claude_skills", NewClaudeSkills)
}
