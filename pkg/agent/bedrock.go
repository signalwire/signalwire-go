package agent

import "fmt"

// BedrockAgent is an AgentBase specialisation for Amazon Bedrock's voice-to-voice
// model. It renders the SWML "amazon_bedrock" verb (instead of "ai") and exposes
// Bedrock inference controls (voice, temperature, top_p, max_tokens, model).
// Mirrors signalwire.agents.bedrock.BedrockAgent.
type BedrockAgent struct {
	*AgentBase

	voiceID     string
	temperature float64
	topP        float64
	maxTokens   int
	model       string
}

// BedrockOptions configures a BedrockAgent at construction.
type BedrockOptions struct {
	Name         string
	Route        string
	SystemPrompt string
	VoiceID      string
	Temperature  float64
	TopP         float64
	MaxTokens    int
}

// NewBedrockAgent creates a BedrockAgent. Unset numeric options fall back to the
// Bedrock defaults (temperature 0.7, top_p 0.9, max_tokens 1024) and voice
// "matthew", matching the Python reference.
func NewBedrockAgent(opts BedrockOptions) *BedrockAgent {
	name := opts.Name
	if name == "" {
		name = "bedrock_agent"
	}
	route := opts.Route
	if route == "" {
		route = "/"
	}
	voice := opts.VoiceID
	if voice == "" {
		voice = "matthew"
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

	base := NewAgentBase(
		WithName(name),
		WithRoute(route),
		// Bedrock renders the amazon_bedrock verb in place of ai.
		WithAIVerbName("amazon_bedrock"),
	)
	if opts.SystemPrompt != "" {
		base.SetPromptText(opts.SystemPrompt)
	}

	b := &BedrockAgent{
		AgentBase:   base,
		voiceID:     voice,
		temperature: temperature,
		topP:        topP,
		maxTokens:   maxTokens,
	}
	b.applyInferenceParams()
	return b
}

// applyInferenceParams pushes the current Bedrock inference settings onto the
// base agent's param map so they render inside the amazon_bedrock verb.
func (b *BedrockAgent) applyInferenceParams() {
	b.AgentBase.SetParams(map[string]any{
		"voice_id":    b.voiceID,
		"temperature": b.temperature,
		"top_p":       b.topP,
		"max_tokens":  b.maxTokens,
	})
}

// SetVoice sets the Bedrock voice id (e.g. "matthew", "joanna").
func (b *BedrockAgent) SetVoice(voiceID string) *BedrockAgent {
	b.voiceID = voiceID
	b.applyInferenceParams()
	return b
}

// SetInferenceParams updates the Bedrock inference parameters. A zero value
// leaves the corresponding parameter unchanged.
func (b *BedrockAgent) SetInferenceParams(temperature, topP float64, maxTokens int) *BedrockAgent {
	if temperature != 0 {
		b.temperature = temperature
	}
	if topP != 0 {
		b.topP = topP
	}
	if maxTokens != 0 {
		b.maxTokens = maxTokens
	}
	b.applyInferenceParams()
	return b
}

// SetLLMModel sets the Bedrock model identifier.
func (b *BedrockAgent) SetLLMModel(model string) *BedrockAgent {
	b.model = model
	b.AgentBase.SetParam("model", model)
	return b
}

// SetLLMTemperature sets the generation temperature.
func (b *BedrockAgent) SetLLMTemperature(temperature float64) *BedrockAgent {
	b.temperature = temperature
	b.applyInferenceParams()
	return b
}

// SetPromptLLMParams applies the Bedrock inference params as the prompt LLM
// params (temperature/top_p/max_tokens).
func (b *BedrockAgent) SetPromptLLMParams() *BedrockAgent {
	b.AgentBase.SetPromptLlmParams(b.bedrockParams())
	return b
}

// SetPostPromptLLMParams applies the Bedrock inference params as the post-prompt
// LLM params.
func (b *BedrockAgent) SetPostPromptLLMParams() *BedrockAgent {
	b.AgentBase.SetPostPromptLlmParams(b.bedrockParams())
	return b
}

func (b *BedrockAgent) bedrockParams() map[string]any {
	return map[string]any{
		"temperature": b.temperature,
		"top_p":       b.topP,
		"max_tokens":  b.maxTokens,
	}
}

// String renders a debug representation (Python __repr__).
func (b *BedrockAgent) String() string {
	return fmt.Sprintf("BedrockAgent(name=%q, voice=%q, model=%q)", b.GetName(), b.voiceID, b.model)
}
