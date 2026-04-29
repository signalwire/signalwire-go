package prefabs

import (
	"sort"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/logging"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// BedrockOptions configures a new BedrockAgent.
//
// BedrockAgent is the Go equivalent of the Python BedrockAgent class.  It
// extends the standard agent infrastructure but renders the SWML document
// with the "amazon_bedrock" verb instead of the default "ai" verb.  All
// standard AgentBase capabilities (prompt, SWAIG tools, skills, post-prompt,
// dynamic config, etc.) work unchanged.
type BedrockOptions struct {
	// Name is the agent name.  Defaults to "bedrock_agent".
	Name string

	// Route is the HTTP route for the agent.  Defaults to "/bedrock".
	Route string

	// SystemPrompt is an optional initial system prompt.  It can be
	// overridden later via SetPromptText.
	SystemPrompt string

	// VoiceID is the Bedrock voice identifier.  Defaults to "matthew".
	VoiceID string

	// Temperature is the generation temperature (0–1).  Defaults to 0.7.
	Temperature float64

	// TopP is the nucleus-sampling parameter (0–1).  Defaults to 0.9.
	TopP float64

	// MaxTokens is the maximum number of tokens to generate.
	// Defaults to 1024.
	MaxTokens int

	// AgentOptions are functional options forwarded to NewAgentBase.
	// Use them to set host, port, auth credentials, etc.
	AgentOptions []agent.AgentOption
}

// BedrockAgent wraps AgentBase and configures it to emit the
// "amazon_bedrock" SWML verb instead of the standard "ai" verb.
//
// The voice_id, temperature, and top_p values are injected into the
// rendered prompt config (matching Python's _add_voice_to_prompt).
// Keys that are text-model-specific (barge_confidence, presence_penalty,
// frequency_penalty) are removed from the prompt config because they do
// not apply to Bedrock's voice-to-voice model.
type BedrockAgent struct {
	*agent.AgentBase

	voiceID     string
	temperature float64
	topP        float64
	maxTokens   int
	logger      *logging.Logger
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// NewBedrockAgent creates a BedrockAgent with Bedrock-specific SWML rendering.
//
// Python equivalent: BedrockAgent.__init__
func NewBedrockAgent(opts BedrockOptions) *BedrockAgent {
	name := opts.Name
	if name == "" {
		name = "bedrock_agent"
	}
	route := opts.Route
	if route == "" {
		route = "/bedrock"
	}
	voiceID := opts.VoiceID
	if voiceID == "" {
		voiceID = "matthew"
	}
	temperature := opts.Temperature
	if temperature == 0 {
		temperature = 0.7
	}
	topP := opts.TopP
	if topP == 0 {
		topP = 0.9
	}
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	// Build base AgentOption list: name + route first, then caller overrides.
	baseOpts := []agent.AgentOption{
		agent.WithName(name),
		agent.WithRoute(route),
		agent.WithAIVerbName("amazon_bedrock"),
	}
	baseOpts = append(baseOpts, opts.AgentOptions...)

	base := agent.NewAgentBase(baseOpts...)

	ba := &BedrockAgent{
		AgentBase:   base,
		voiceID:     voiceID,
		temperature: temperature,
		topP:        topP,
		maxTokens:   maxTokens,
		logger:      logging.New(name),
	}

	// Install the prompt transformer that adds voice + inference params and
	// removes text-model-specific keys.  This matches Python's
	// _add_voice_to_prompt().
	base.SetPromptTransformer(ba.addVoiceToPrompt)

	// Apply initial system prompt if provided.
	if opts.SystemPrompt != "" {
		base.SetPromptText(opts.SystemPrompt)
	}

	ba.logger.Info("BedrockAgent initialized: %s on route %s", name, route)

	return ba
}

// ---------------------------------------------------------------------------
// Prompt transformer (Python: _add_voice_to_prompt)
// ---------------------------------------------------------------------------

// addVoiceToPrompt is the prompt transformer installed in AgentBase.
// It copies the assembled prompt config, strips text-model-specific keys,
// and adds voice_id, temperature, and top_p.
//
// Python equivalent: BedrockAgent._add_voice_to_prompt
func (ba *BedrockAgent) addVoiceToPrompt(promptCfg map[string]any) map[string]any {
	// Text-model-specific keys that do not apply to Bedrock voice-to-voice.
	skip := map[string]bool{
		"barge_confidence":  true,
		"presence_penalty":  true,
		"frequency_penalty": true,
	}

	filtered := make(map[string]any, len(promptCfg))
	for k, v := range promptCfg {
		if !skip[k] {
			filtered[k] = v
		}
	}

	// Add Bedrock voice + inference params.
	filtered["voice_id"] = ba.voiceID
	filtered["temperature"] = ba.temperature
	filtered["top_p"] = ba.topP

	return filtered
}

// ---------------------------------------------------------------------------
// Public mutators — Python equivalents
// ---------------------------------------------------------------------------

// SetVoice sets the Bedrock voice ID.
//
// Python equivalent: BedrockAgent.set_voice
func (ba *BedrockAgent) SetVoice(voiceID string) {
	ba.voiceID = voiceID
	ba.logger.Debug("Voice set to: %s", voiceID)
}

// SetInferenceParams updates one or more Bedrock inference parameters.
// Pass zero-value pointers to leave a parameter unchanged.
//
// Python equivalent: BedrockAgent.set_inference_params
func (ba *BedrockAgent) SetInferenceParams(temperature, topP float64, maxTokens int) {
	if temperature != 0 {
		ba.temperature = temperature
	}
	if topP != 0 {
		ba.topP = topP
	}
	if maxTokens != 0 {
		ba.maxTokens = maxTokens
	}
	ba.logger.Debug("Inference params updated: temp=%v, top_p=%v, max_tokens=%v",
		ba.temperature, ba.topP, ba.maxTokens)
}

// SetLLMModel logs a warning and does nothing.  Bedrock uses a fixed
// voice-to-voice model, so overriding the model name is not meaningful.
//
// Python equivalent: BedrockAgent.set_llm_model
func (ba *BedrockAgent) SetLLMModel(model string) {
	ba.logger.Warn("SetLLMModel(%q) called but Bedrock uses a fixed voice-to-voice model", model)
}

// SetLLMTemperature is a convenience wrapper that delegates to
// SetInferenceParams.
//
// Python equivalent: BedrockAgent.set_llm_temperature
func (ba *BedrockAgent) SetLLMTemperature(temperature float64) {
	ba.SetInferenceParams(temperature, 0, 0)
}

// SetPostPromptLLMParams logs a warning and ignores the parameters.
// Bedrock's post-prompt summarisation uses OpenAI configured at the
// platform level (in the C code), so SDK-level overrides have no effect.
// The keys of params are listed in the warning so the caller can see
// what was ignored.
//
// Python equivalent: BedrockAgent.set_post_prompt_llm_params
func (ba *BedrockAgent) SetPostPromptLLMParams(params map[string]any) {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ba.logger.Warn(
		"SetPostPromptLLMParams(%v) called but Bedrock post-prompt uses OpenAI configured in C code; ignoring keys",
		keys,
	)
}

// SetPromptLLMParams logs a warning directing the caller to
// SetInferenceParams instead. The keys of params are listed in the
// warning so the caller can see what was ignored.
//
// Python equivalent: BedrockAgent.set_prompt_llm_params
func (ba *BedrockAgent) SetPromptLLMParams(params map[string]any) {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ba.logger.Warn(
		"SetPromptLLMParams(%v) called — use SetInferenceParams() for Bedrock; ignoring keys",
		keys,
	)
}
