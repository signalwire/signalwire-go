// Package contexts provides the Contexts & Steps workflow system for
// SignalWire AI agents.
//
// Instead of a single flat prompt, agents can define structured Contexts
// (conversation flows) containing ordered Steps (sequential stages). Each
// step carries its own prompt, completion criteria, function restrictions,
// and navigation rules. The builder serialises the whole tree into the
// map[string]any format expected by the SWML AI verb.
package contexts

import (
	"errors"
	"fmt"
	"strings"
)

// Limits guard against unreasonable configurations.
const (
	MaxContexts       = 50
	MaxStepsPerContext = 100
)

// ReservedNativeToolNames is the set of tool names the runtime auto-injects
// when contexts/steps are present. User-defined SWAIG tools must not
// collide with these names.
//
//   - next_step / change_context are injected when valid_steps or
//     valid_contexts is set so the model can navigate the flow.
//   - gather_submit is injected while a step's gather_info is collecting
//     answers.
//
// ContextBuilder.Validate() rejects any agent that registers a user tool
// sharing one of these names — the runtime would never call the user tool
// because the native one wins.
var ReservedNativeToolNames = map[string]struct{}{
	"next_step":      {},
	"change_context": {},
	"gather_submit":  {},
}

// ---------------------------------------------------------------------------
// GatherQuestion
// ---------------------------------------------------------------------------

// GatherQuestionOption is a functional option applied to a GatherQuestion.
type GatherQuestionOption func(*GatherQuestion)

// WithType sets the JSON-schema type for the answer (default "string").
func WithType(t string) GatherQuestionOption {
	return func(q *GatherQuestion) { q.Type = t }
}

// WithConfirm sets whether the model must confirm the answer with the user.
func WithConfirm(c bool) GatherQuestionOption {
	return func(q *GatherQuestion) { q.Confirm = c }
}

// WithPrompt sets extra instruction text appended for this question.
func WithPrompt(p string) GatherQuestionOption {
	return func(q *GatherQuestion) { q.Prompt = p }
}

// WithFunctions sets additional function names visible for this question.
func WithFunctions(f []string) GatherQuestionOption {
	return func(q *GatherQuestion) { q.Functions = f }
}

// GatherQuestion represents a single question in a gather_info configuration.
type GatherQuestion struct {
	Key       string
	Question  string
	Type      string   // default "string"
	Confirm   bool
	Prompt    string   // optional
	Functions []string // optional
}

// ToMap serialises the question to the SWML map format.
func (q *GatherQuestion) ToMap() map[string]any {
	m := map[string]any{
		"key":      q.Key,
		"question": q.Question,
	}
	if q.Type != "" && q.Type != "string" {
		m["type"] = q.Type
	}
	if q.Confirm {
		m["confirm"] = true
	}
	if q.Prompt != "" {
		m["prompt"] = q.Prompt
	}
	if len(q.Functions) > 0 {
		m["functions"] = q.Functions
	}
	return m
}

// ---------------------------------------------------------------------------
// GatherInfo
// ---------------------------------------------------------------------------

// GatherInfo configures information gathering for a step.
type GatherInfo struct {
	OutputKey        string
	CompletionAction string
	Prompt           string
	Questions        []GatherQuestion
}

// AddQuestion appends a question and returns the GatherInfo for chaining.
func (g *GatherInfo) AddQuestion(key, question string, opts ...GatherQuestionOption) *GatherInfo {
	q := GatherQuestion{
		Key:      key,
		Question: question,
		Type:     "string",
	}
	for _, o := range opts {
		o(&q)
	}
	g.Questions = append(g.Questions, q)
	return g
}

// Validate returns an error if the GatherInfo is not ready for serialisation.
// Specifically, it rejects a GatherInfo with no questions, which would produce
// invalid SWML. This matches the Python SDK's ValueError raised by to_dict()
// when _questions is empty.
func (g *GatherInfo) Validate() error {
	if len(g.Questions) == 0 {
		return errors.New("gather_info must have at least one question")
	}
	return nil
}

// ToMap serialises to the SWML map format.
// Callers that construct GatherInfo directly should call Validate() first to
// ensure the result is valid SWML. Step.ToMap() enforces this automatically
// by only calling ToMap() when len(Questions) > 0.
func (g *GatherInfo) ToMap() map[string]any {
	qs := make([]map[string]any, len(g.Questions))
	for i := range g.Questions {
		qs[i] = g.Questions[i].ToMap()
	}
	m := map[string]any{
		"questions": qs,
	}
	if g.Prompt != "" {
		m["prompt"] = g.Prompt
	}
	if g.OutputKey != "" {
		m["output_key"] = g.OutputKey
	}
	if g.CompletionAction != "" {
		m["completion_action"] = g.CompletionAction
	}
	return m
}

// ---------------------------------------------------------------------------
// Step
// ---------------------------------------------------------------------------

// Step represents a single step within a context. All setter methods return
// *Step so they can be chained.
type Step struct {
	name               string
	text               string
	sections           []map[string]any
	stepCriteria       string
	functions          any // string "none" or []string
	validSteps         []string
	validContexts      []string
	isEnd              bool
	skipUserTurn       bool
	skipToNextStep     bool
	gatherInfo         *GatherInfo
	resetSystemPrompt  string
	resetUserPrompt    string
	resetConsolidate   *bool
	resetFullReset     *bool
}

// Name returns the step's name.
func (s *Step) Name() string { return s.name }

// SetText sets the step's prompt text directly.
func (s *Step) SetText(text string) *Step {
	s.text = text
	return s
}

// AddSection adds a POM section to the step.
func (s *Step) AddSection(title, body string) *Step {
	s.sections = append(s.sections, map[string]any{"title": title, "body": body})
	return s
}

// AddBullets adds a POM section with bullet points.
func (s *Step) AddBullets(title string, bullets []string) *Step {
	s.sections = append(s.sections, map[string]any{"title": title, "bullets": bullets})
	return s
}

// SetStepCriteria sets the criteria for determining when this step is complete.
func (s *Step) SetStepCriteria(criteria string) *Step {
	s.stepCriteria = criteria
	return s
}

// SetFunctions sets which non-internal functions are callable while this
// step is active.
//
// IMPORTANT — inheritance behavior:
// If you do NOT call this method, the step inherits whichever function set
// was active on the previous step (or the previous context's last step).
// The server-side runtime only resets the active set when a step explicitly
// declares its `functions` field. This is the most common source of bugs
// in multi-step agents: forgetting SetFunctions on a later step lets the
// previous step's tools leak through. Best practice is to call
// SetFunctions explicitly on every step that should differ from the
// previous one.
//
// Keep the per-step active set small: LLM tool selection accuracy
// degrades noticeably past ~7-8 simultaneously-active tools per call.
// Use per-step whitelisting to partition large tool collections.
//
// Accepts:
//
//   - []string{"a", "b"}  — whitelist of function names allowed in this step
//   - []string{}          — explicit disable-all
//   - "none"              — synonym for the empty slice
//
// Internal functions (e.g. gather_submit, hangup_hook) are ALWAYS protected
// and cannot be deactivated by this whitelist. The native navigation tools
// next_step and change_context are injected automatically when
// SetValidSteps / SetValidContexts is used; they are not affected by this
// list and do not need to appear in it.
func (s *Step) SetFunctions(functions any) *Step {
	s.functions = functions
	return s
}

// SetValidSteps sets which steps can be navigated to from this step.
func (s *Step) SetValidSteps(steps []string) *Step {
	s.validSteps = steps
	return s
}

// SetValidContexts sets which contexts can be navigated to from this step.
func (s *Step) SetValidContexts(contexts []string) *Step {
	s.validContexts = contexts
	return s
}

// SetEnd marks this step as terminal for the step flow.
//
// IMPORTANT: end=true does NOT end the conversation or hang up the call.
// It exits step mode entirely after this step executes — clearing the
// steps list, current step index, valid_steps, and valid_contexts. The
// agent keeps running, but operates only under the base system prompt and
// the context-level prompt; no more step instructions are injected and no
// more next_step tool is offered.
//
// To actually end the call, call a hangup tool or define a hangup_hook.
func (s *Step) SetEnd(end bool) *Step {
	s.isEnd = end
	return s
}

// SetSkipUserTurn sets whether to skip waiting for user input after this step.
func (s *Step) SetSkipUserTurn(skip bool) *Step {
	s.skipUserTurn = skip
	return s
}

// SetSkipToNextStep sets whether to automatically advance to the next step.
func (s *Step) SetSkipToNextStep(skip bool) *Step {
	s.skipToNextStep = skip
	return s
}

// SetGatherInfo enables info gathering for this step and returns the Step
// for fluent chaining. This matches the Python SDK's set_gather_info, which
// returns self so that step-level setters (SetFunctions, SetValidSteps, etc.)
// can be chained after configuring gather info.
//
// To add questions to the gather info, use AddGatherQuestion on the same
// *Step receiver.
func (s *Step) SetGatherInfo(outputKey, completionAction, prompt string) *Step {
	s.gatherInfo = &GatherInfo{
		OutputKey:        outputKey,
		CompletionAction: completionAction,
		Prompt:           prompt,
	}
	return s
}

// AddGatherQuestion adds a question to this step's gather_info. SetGatherInfo
// should be called first (this method silently initialises the struct if not,
// to keep callers from having to worry about ordering). Returns the Step for
// chaining.
//
// IMPORTANT — gather mode locks function access:
// While the model is asking gather questions, the runtime forcibly
// deactivates ALL of the step's other functions. The only callable tools
// during a gather question are:
//
//   - gather_submit (the native answer-submission tool)
//   - Whatever names you list with WithFunctions in this question's opts
//
// next_step and change_context are also filtered out — the model cannot
// navigate away until the gather completes. This is by design: it forces
// a tight ask → submit → next-question loop.
//
// If a question needs to call out to a tool (e.g. validate an email,
// geocode a ZIP), pass that tool name via WithFunctions on this question.
// Functions listed here are active ONLY for this question.
func (s *Step) AddGatherQuestion(key, question string, opts ...GatherQuestionOption) *Step {
	if s.gatherInfo == nil {
		// Silently initialise so callers are not forced into ordering.
		s.gatherInfo = &GatherInfo{}
	}
	s.gatherInfo.AddQuestion(key, question, opts...)
	return s
}

// ClearSections removes all POM sections and direct text from this step.
func (s *Step) ClearSections() *Step {
	s.sections = nil
	s.text = ""
	return s
}

// SetResetSystemPrompt sets the system prompt for context switching.
func (s *Step) SetResetSystemPrompt(prompt string) *Step {
	s.resetSystemPrompt = prompt
	return s
}

// SetResetUserPrompt sets the user prompt for context switching.
func (s *Step) SetResetUserPrompt(prompt string) *Step {
	s.resetUserPrompt = prompt
	return s
}

// SetResetConsolidate sets whether to consolidate conversation on context switch.
func (s *Step) SetResetConsolidate(consolidate bool) *Step {
	s.resetConsolidate = &consolidate
	return s
}

// SetResetFullReset sets whether to do a full reset on context switch.
func (s *Step) SetResetFullReset(fullReset bool) *Step {
	s.resetFullReset = &fullReset
	return s
}

// renderText produces the prompt string for the step.
func (s *Step) renderText() string {
	if s.text != "" {
		return s.text
	}
	if len(s.sections) == 0 {
		return ""
	}
	var parts []string
	for _, sec := range s.sections {
		title, _ := sec["title"].(string)
		if bullets, ok := sec["bullets"].([]string); ok {
			parts = append(parts, fmt.Sprintf("## %s", title))
			for _, b := range bullets {
				parts = append(parts, fmt.Sprintf("- %s", b))
			}
		} else {
			body, _ := sec["body"].(string)
			parts = append(parts, fmt.Sprintf("## %s", title))
			parts = append(parts, body)
		}
		parts = append(parts, "") // blank line for spacing
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// ToMap serialises the step to the SWML map format.
func (s *Step) ToMap() map[string]any {
	m := map[string]any{
		"name": s.name,
		"text": s.renderText(),
	}

	if s.stepCriteria != "" {
		m["step_criteria"] = s.stepCriteria
	}
	if s.functions != nil {
		m["functions"] = s.functions
	}
	if s.validSteps != nil {
		m["valid_steps"] = s.validSteps
	}
	if s.validContexts != nil {
		m["valid_contexts"] = s.validContexts
	}
	if s.isEnd {
		m["end"] = true
	}
	if s.skipUserTurn {
		m["skip_user_turn"] = true
	}
	if s.skipToNextStep {
		m["skip_to_next_step"] = true
	}

	// Build reset object if any reset field is set.
	reset := map[string]any{}
	if s.resetSystemPrompt != "" {
		reset["system_prompt"] = s.resetSystemPrompt
	}
	if s.resetUserPrompt != "" {
		reset["user_prompt"] = s.resetUserPrompt
	}
	if s.resetConsolidate != nil && *s.resetConsolidate {
		reset["consolidate"] = true
	}
	if s.resetFullReset != nil && *s.resetFullReset {
		reset["full_reset"] = true
	}
	if len(reset) > 0 {
		m["reset"] = reset
	}

	if s.gatherInfo != nil && len(s.gatherInfo.Questions) > 0 {
		m["gather_info"] = s.gatherInfo.ToMap()
	}

	return m
}

// ---------------------------------------------------------------------------
// Context
// ---------------------------------------------------------------------------

// Context represents a single context containing ordered steps.
// All setter methods return *Context for chaining.
type Context struct {
	name           string
	steps          []*Step // ordered
	stepMap        map[string]*Step
	initialStep    string
	validContexts  []string
	validSteps     []string
	postPrompt     string
	systemPrompt   string
	prompt         string
	consolidate    *bool
	fullReset      *bool
	userPrompt     string
	isolated       bool
	sections       []map[string]any
	systemSections []map[string]any
	enterFillers   map[string][]string
	exitFillers    map[string][]string
}

// newContext creates a Context with the given name.
func newContext(name string) *Context {
	return &Context{
		name:    name,
		stepMap: make(map[string]*Step),
	}
}

// Name returns the context's name.
func (c *Context) Name() string { return c.name }

// AddStep creates a new step, appends it to the ordered list, stores it in
// the lookup map, and returns the Step for further configuration.
func (c *Context) AddStep(name string) *Step {
	s := &Step{name: name}
	c.steps = append(c.steps, s)
	c.stepMap[name] = s
	return s
}

// GetStep returns the step with the given name, or nil if not found.
func (c *Context) GetStep(name string) *Step {
	return c.stepMap[name]
}

// RemoveStep removes a step by name. Returns the receiver for method chaining.
func (c *Context) RemoveStep(name string) *Context {
	if _, ok := c.stepMap[name]; !ok {
		return c
	}
	delete(c.stepMap, name)
	for i, s := range c.steps {
		if s.name == name {
			c.steps = append(c.steps[:i], c.steps[i+1:]...)
			break
		}
	}
	return c
}

// MoveStep moves an existing step to the given position (0-based index).
// Returns the receiver for method chaining.
func (c *Context) MoveStep(name string, position int) *Context {
	idx := -1
	for i, s := range c.steps {
		if s.name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return c
	}
	step := c.steps[idx]
	// Remove from current position.
	c.steps = append(c.steps[:idx], c.steps[idx+1:]...)
	// Clamp position.
	if position < 0 {
		position = 0
	}
	if position > len(c.steps) {
		position = len(c.steps)
	}
	// Insert at new position.
	c.steps = append(c.steps[:position], append([]*Step{step}, c.steps[position:]...)...)
	return c
}

// SetInitialStep sets which step the context starts on when entered.
//
// By default, a context starts on its first step (index 0). Use this to
// skip a preamble step on re-entry via change_context.
func (c *Context) SetInitialStep(stepName string) *Context {
	c.initialStep = stepName
	return c
}

// SetValidContexts sets which contexts can be navigated to from this context.
func (c *Context) SetValidContexts(ctxs []string) *Context {
	c.validContexts = ctxs
	return c
}

// SetValidSteps sets which steps can be navigated to from any step in this context.
func (c *Context) SetValidSteps(steps []string) *Context {
	c.validSteps = steps
	return c
}

// SetPostPrompt sets the post-prompt override for this context.
func (c *Context) SetPostPrompt(prompt string) *Context {
	c.postPrompt = prompt
	return c
}

// SetSystemPrompt sets the system prompt for context switching.
func (c *Context) SetSystemPrompt(prompt string) *Context {
	c.systemPrompt = prompt
	return c
}

// SetPrompt sets the context's prompt text directly.
func (c *Context) SetPrompt(prompt string) *Context {
	c.prompt = prompt
	return c
}

// SetConsolidate sets whether to consolidate conversation history on entry.
func (c *Context) SetConsolidate(consolidate bool) *Context {
	c.consolidate = &consolidate
	return c
}

// SetFullReset sets whether to do a full reset when entering this context.
func (c *Context) SetFullReset(fullReset bool) *Context {
	c.fullReset = &fullReset
	return c
}

// SetUserPrompt sets the user prompt to inject when entering this context.
func (c *Context) SetUserPrompt(prompt string) *Context {
	c.userPrompt = prompt
	return c
}

// SetIsolated marks this context as isolated — entering it wipes
// conversation history.
//
// When isolated=true and the context is entered via change_context, the
// runtime wipes the conversation array. The model starts fresh with only
// the new context's system_prompt + step instructions, with no memory of
// prior turns.
//
// EXCEPTION — reset overrides the wipe:
// If the context also has a reset configuration (via SetConsolidate or
// SetFullReset), the wipe is skipped in favor of the reset behavior. Use
// reset with consolidate=true to summarize prior history into a single
// message instead of dropping it entirely.
//
// Use cases:
//
//   - Switching to a sensitive billing flow that should not see prior
//     small-talk
//   - Handing off to a different agent persona
//   - Resetting after a long off-topic detour
func (c *Context) SetIsolated(isolated bool) *Context {
	c.isolated = isolated
	return c
}

// AddSection adds a POM section to the context prompt.
func (c *Context) AddSection(title, body string) *Context {
	c.sections = append(c.sections, map[string]any{"title": title, "body": body})
	return c
}

// AddBullets adds a POM section with bullet points to the context prompt.
func (c *Context) AddBullets(title string, bullets []string) *Context {
	c.sections = append(c.sections, map[string]any{"title": title, "bullets": bullets})
	return c
}

// AddSystemSection adds a POM section to the system prompt.
func (c *Context) AddSystemSection(title, body string) *Context {
	c.systemSections = append(c.systemSections, map[string]any{"title": title, "body": body})
	return c
}

// AddSystemBullets adds a POM section with bullet points to the system prompt.
func (c *Context) AddSystemBullets(title string, bullets []string) *Context {
	c.systemSections = append(c.systemSections, map[string]any{"title": title, "bullets": bullets})
	return c
}

// SetEnterFillers sets all enter fillers at once.
func (c *Context) SetEnterFillers(fillers map[string][]string) *Context {
	c.enterFillers = fillers
	return c
}

// SetExitFillers sets all exit fillers at once.
func (c *Context) SetExitFillers(fillers map[string][]string) *Context {
	c.exitFillers = fillers
	return c
}

// AddEnterFiller adds enter fillers for a specific language code.
func (c *Context) AddEnterFiller(langCode string, fillers []string) *Context {
	if c.enterFillers == nil {
		c.enterFillers = make(map[string][]string)
	}
	c.enterFillers[langCode] = fillers
	return c
}

// AddExitFiller adds exit fillers for a specific language code.
func (c *Context) AddExitFiller(langCode string, fillers []string) *Context {
	if c.exitFillers == nil {
		c.exitFillers = make(map[string][]string)
	}
	c.exitFillers[langCode] = fillers
	return c
}

// renderSections converts a slice of POM section maps into a markdown string.
func renderSections(sections []map[string]any) string {
	if len(sections) == 0 {
		return ""
	}
	var parts []string
	for _, sec := range sections {
		title, _ := sec["title"].(string)
		if bullets, ok := sec["bullets"].([]string); ok {
			parts = append(parts, fmt.Sprintf("## %s", title))
			for _, b := range bullets {
				parts = append(parts, fmt.Sprintf("- %s", b))
			}
		} else {
			body, _ := sec["body"].(string)
			parts = append(parts, fmt.Sprintf("## %s", title))
			parts = append(parts, body)
		}
		parts = append(parts, "")
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// ToMap serialises the context to the SWML map format.
func (c *Context) ToMap() map[string]any {
	m := map[string]any{}

	// Steps (ordered).
	stepList := make([]map[string]any, len(c.steps))
	for i, s := range c.steps {
		stepList[i] = s.ToMap()
	}
	m["steps"] = stepList

	if c.validContexts != nil {
		m["valid_contexts"] = c.validContexts
	}
	if c.validSteps != nil {
		m["valid_steps"] = c.validSteps
	}
	if c.initialStep != "" {
		m["initial_step"] = c.initialStep
	}
	if c.postPrompt != "" {
		m["post_prompt"] = c.postPrompt
	}

	// System prompt: POM sections take precedence over raw string.
	if len(c.systemSections) > 0 {
		m["system_prompt"] = renderSections(c.systemSections)
	} else if c.systemPrompt != "" {
		m["system_prompt"] = c.systemPrompt
	}

	if c.consolidate != nil && *c.consolidate {
		m["consolidate"] = true
	}
	if c.fullReset != nil && *c.fullReset {
		m["full_reset"] = true
	}
	if c.userPrompt != "" {
		m["user_prompt"] = c.userPrompt
	}
	if c.isolated {
		m["isolated"] = true
	}

	// Context prompt: POM sections produce "pom" key, raw string uses "prompt".
	if len(c.sections) > 0 {
		m["pom"] = c.sections
	} else if c.prompt != "" {
		m["prompt"] = c.prompt
	}

	if c.enterFillers != nil {
		m["enter_fillers"] = c.enterFillers
	}
	if c.exitFillers != nil {
		m["exit_fillers"] = c.exitFillers
	}

	return m
}

// ---------------------------------------------------------------------------
// ContextBuilder
// ---------------------------------------------------------------------------

// ToolLister is implemented by an agent so ContextBuilder.Validate() can
// check registered SWAIG tool names against ReservedNativeToolNames.
// AgentBase implements this by returning the insertion-ordered list of
// registered tool names.
type ToolLister interface {
	// ListToolNames returns the names of every registered SWAIG tool.
	ListToolNames() []string
}

// ContextBuilder is the top-level builder for multi-step, multi-context
// AI agent workflows.
//
// A ContextBuilder owns one or more Contexts; each Context owns an ordered
// list of Steps. Only one context and one step is active at a time. Per
// chat turn, the runtime injects the current step's instructions as a
// system message, then asks the LLM for a response.
//
// # Native tools auto-injected by the runtime
//
// When a step (or its enclosing context) declares valid_steps or
// valid_contexts, the runtime auto-injects two native tools so the model
// can navigate the flow:
//
//   - next_step(step: enum)         — present when valid_steps is set
//   - change_context(context: enum) — present when valid_contexts is set
//
// Their enum schemas are rewritten on every turn to match whatever
// valid_steps / valid_contexts apply to the current step. You do NOT
// need to define these tools yourself; they appear automatically.
//
// A third native tool — gather_submit — is injected during gather_info
// questioning (see Step.SetGatherInfo / Step.AddGatherQuestion).
//
// These three names — next_step, change_context, gather_submit — are
// reserved. ContextBuilder.Validate() rejects any agent that defines a
// SWAIG tool with one of these names. See ReservedNativeToolNames.
//
// # Function whitelisting (Step.SetFunctions)
//
// Each step may declare a functions whitelist. The whitelist is applied
// in-memory at the start of each LLM turn. CRITICALLY: if a step does NOT
// declare a functions field, it INHERITS the previous step's active set.
// See Step.SetFunctions for details and examples.
type ContextBuilder struct {
	contexts   []*Context // ordered
	contextMap map[string]*Context
	agent      ToolLister // optional; set via AttachAgent
}

// AttachAgent wires an agent into the builder so Validate() can check
// user-defined tool names against ReservedNativeToolNames. AgentBase
// calls this internally when you invoke DefineContexts().
func (cb *ContextBuilder) AttachAgent(a ToolLister) *ContextBuilder {
	cb.agent = a
	return cb
}

// NewContextBuilder creates a new empty ContextBuilder.
func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{
		contextMap: make(map[string]*Context),
	}
}

// Reset removes all contexts, returning the builder to its initial state.
// Use this in a dynamic config callback when you need to rebuild contexts
// from scratch for a specific request.
func (cb *ContextBuilder) Reset() *ContextBuilder {
	cb.contexts = nil
	cb.contextMap = make(map[string]*Context)
	return cb
}

// AddContext creates a new context with the given name and returns it.
func (cb *ContextBuilder) AddContext(name string) *Context {
	ctx := newContext(name)
	cb.contexts = append(cb.contexts, ctx)
	cb.contextMap[name] = ctx
	return ctx
}

// GetContext returns the context with the given name, or nil if not found.
func (cb *ContextBuilder) GetContext(name string) *Context {
	return cb.contextMap[name]
}

// Validate checks the builder configuration for common errors:
//   - At least one context must be defined.
//   - A single context must be named "default".
//   - Every context must contain at least one step.
//   - Every step must have a name.
//   - valid_steps entries (except "next") must name existing steps in the same context.
//   - valid_contexts entries (context-level) must name existing contexts.
//   - valid_contexts entries (step-level) must name existing contexts.
//   - gather_info questions must be non-empty and have unique keys.
//   - gather_info completion_action (if set) targets an existing step.
//   - No user-defined SWAIG tool collides with a reserved native name.
func (cb *ContextBuilder) Validate() error {
	if len(cb.contexts) == 0 {
		return errors.New("at least one context must be defined")
	}
	if len(cb.contexts) == 1 && cb.contexts[0].name != "default" {
		return fmt.Errorf("when using a single context, it must be named 'default' (got %q)", cb.contexts[0].name)
	}
	for _, ctx := range cb.contexts {
		if len(ctx.steps) == 0 {
			return fmt.Errorf("context %q must have at least one step", ctx.name)
		}
		for _, s := range ctx.steps {
			if s.name == "" {
				return fmt.Errorf("all steps in context %q must have a name", ctx.name)
			}
		}
	}

	// Validate initial_step references a real step in the context.
	for _, ctx := range cb.contexts {
		if ctx.initialStep != "" {
			if _, ok := ctx.stepMap[ctx.initialStep]; !ok {
				available := make([]string, 0, len(ctx.stepMap))
				for n := range ctx.stepMap {
					available = append(available, n)
				}
				sortedAvailable := append([]string(nil), available...)
				for i := 1; i < len(sortedAvailable); i++ {
					for j := i; j > 0 && sortedAvailable[j-1] > sortedAvailable[j]; j-- {
						sortedAvailable[j-1], sortedAvailable[j] = sortedAvailable[j], sortedAvailable[j-1]
					}
				}
				return fmt.Errorf(
					"context %q has initial_step=%q but that step does not exist. Available steps: %v",
					ctx.name, ctx.initialStep, sortedAvailable,
				)
			}
		}
	}

	// Validate step references in valid_steps. Each entry must be either the
	// special keyword "next" (advance to the next sequential step) or the
	// name of an existing step in the same context.
	for _, ctx := range cb.contexts {
		for _, step := range ctx.steps {
			for _, ref := range step.validSteps {
				if ref == "next" {
					continue
				}
				if _, ok := ctx.stepMap[ref]; !ok {
					return fmt.Errorf(
						"step %q in context %q references unknown step %q",
						step.name, ctx.name, ref,
					)
				}
			}
		}
	}

	// Validate context references in valid_contexts (context-level). Each
	// entry must name an existing context.
	for _, ctx := range cb.contexts {
		for _, ref := range ctx.validContexts {
			if _, ok := cb.contextMap[ref]; !ok {
				return fmt.Errorf(
					"context %q references unknown context %q",
					ctx.name, ref,
				)
			}
		}
	}

	// Validate context references in valid_contexts (step-level). Each
	// entry must name an existing context.
	for _, ctx := range cb.contexts {
		for _, step := range ctx.steps {
			for _, ref := range step.validContexts {
				if _, ok := cb.contextMap[ref]; !ok {
					return fmt.Errorf(
						"step %q in context %q references unknown context %q",
						step.name, ctx.name, ref,
					)
				}
			}
		}
	}

	// Validate gather_info question lists: must be non-empty and must not
	// contain duplicate keys within the same step.
	for _, ctx := range cb.contexts {
		for _, step := range ctx.steps {
			if step.gatherInfo == nil {
				continue
			}
			if len(step.gatherInfo.Questions) == 0 {
				return fmt.Errorf(
					"step %q in context %q has gather_info with no questions",
					step.name, ctx.name,
				)
			}
			seen := make(map[string]struct{}, len(step.gatherInfo.Questions))
			for _, q := range step.gatherInfo.Questions {
				if _, dup := seen[q.Key]; dup {
					return fmt.Errorf(
						"step %q in context %q has duplicate gather_info question key %q",
						step.name, ctx.name, q.Key,
					)
				}
				seen[q.Key] = struct{}{}
			}
		}
	}

	// Validate gather_info completion_action references in gather_info. completion_action
	// is either "next_step" (advance to the following step in order), the
	// name of an existing step in the same context, or empty (stay in the
	// current step).
	for _, ctx := range cb.contexts {
		for i, step := range ctx.steps {
			if step.gatherInfo == nil || step.gatherInfo.CompletionAction == "" {
				continue
			}
			action := step.gatherInfo.CompletionAction
			if action == "next_step" {
				if i == len(ctx.steps)-1 {
					return fmt.Errorf(
						"step %q in context %q has gather_info completion_action='next_step' "+
							"but it is the last step in the context. Either "+
							"(1) add another step after %q, "+
							"(2) set completion_action to the name of an existing step in "+
							"this context to jump to it, or "+
							"(3) leave completion_action empty (default) to stay in %q "+
							"after gathering completes",
						step.name, ctx.name, step.name, step.name,
					)
				}
			} else if _, ok := ctx.stepMap[action]; !ok {
				available := make([]string, 0, len(ctx.stepMap))
				for n := range ctx.stepMap {
					available = append(available, n)
				}
				// Sort for deterministic error messages.
				sortedAvailable := append([]string(nil), available...)
				for i := 1; i < len(sortedAvailable); i++ {
					for j := i; j > 0 && sortedAvailable[j-1] > sortedAvailable[j]; j-- {
						sortedAvailable[j-1], sortedAvailable[j] = sortedAvailable[j], sortedAvailable[j-1]
					}
				}
				return fmt.Errorf(
					"step %q in context %q has gather_info completion_action=%q "+
						"but %q is not a step in this context. Valid options: "+
						"'next_step' (advance to the next sequential step), '' "+
						"(stay in the current step), or one of %v",
					step.name, ctx.name, action, action, sortedAvailable,
				)
			}
		}
	}

	// Validate that user-defined tools do not collide with reserved native
	// tool names. The runtime auto-injects next_step / change_context /
	// gather_submit when contexts/steps are present, so user tools sharing
	// those names would never be called.
	if cb.agent != nil {
		names := cb.agent.ListToolNames()
		colliding := make([]string, 0)
		for _, name := range names {
			if _, ok := ReservedNativeToolNames[name]; ok {
				colliding = append(colliding, name)
			}
		}
		if len(colliding) > 0 {
			// Stable-sort the colliding and reserved lists.
			for i := 1; i < len(colliding); i++ {
				for j := i; j > 0 && colliding[j-1] > colliding[j]; j-- {
					colliding[j-1], colliding[j] = colliding[j], colliding[j-1]
				}
			}
			reserved := make([]string, 0, len(ReservedNativeToolNames))
			for n := range ReservedNativeToolNames {
				reserved = append(reserved, n)
			}
			for i := 1; i < len(reserved); i++ {
				for j := i; j > 0 && reserved[j-1] > reserved[j]; j-- {
					reserved[j-1], reserved[j] = reserved[j], reserved[j-1]
				}
			}
			return fmt.Errorf(
				"tool name(s) %v collide with reserved native tools "+
					"auto-injected by contexts/steps. The names %v are reserved "+
					"and cannot be used for user-defined SWAIG tools when "+
					"contexts/steps are in use. Rename your tool(s) to avoid "+
					"the collision",
				colliding, reserved,
			)
		}
	}

	return nil
}

// ToMap serialises all contexts to the SWML map format. It calls Validate
// first and returns an error if validation fails.
func (cb *ContextBuilder) ToMap() (map[string]any, error) {
	if err := cb.Validate(); err != nil {
		return nil, err
	}
	m := make(map[string]any, len(cb.contexts))
	for _, ctx := range cb.contexts {
		m[ctx.name] = ctx.ToMap()
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// CreateSimpleContext creates a standalone Context. If name is empty it
// defaults to "default".
func CreateSimpleContext(name string) *Context {
	if name == "" {
		name = "default"
	}
	return newContext(name)
}
