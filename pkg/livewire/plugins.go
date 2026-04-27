package livewire

// ---------------------------------------------------------------------------
// STT provider stubs
// ---------------------------------------------------------------------------

// DeepgramSTT is a stub for the Deepgram STT provider.
// On SignalWire, speech recognition is handled by the control plane.
type DeepgramSTT struct {
	Model string
}

// NewDeepgramSTT creates a DeepgramSTT stub.
func NewDeepgramSTT(opts ...func(*DeepgramSTT)) *DeepgramSTT {
	d := &DeepgramSTT{}
	for _, opt := range opts {
		opt(d)
	}
	getGlobalNoop().once("stt_node", "stt_node override: SignalWire's control plane handles the full media pipeline at scale")
	return d
}

// GoogleSTT is a stub for the Google STT provider.
type GoogleSTT struct {
	Model string
}

// NewGoogleSTT creates a GoogleSTT stub.
func NewGoogleSTT(opts ...func(*GoogleSTT)) *GoogleSTT {
	g := &GoogleSTT{}
	for _, opt := range opts {
		opt(g)
	}
	getGlobalNoop().once("stt_node", "stt_node override: SignalWire's control plane handles the full media pipeline at scale")
	return g
}

// InferenceSTT is a stub for the SignalWire-hosted inference STT provider.
// On SignalWire, speech recognition is handled by the control plane.
// Mirrors Python InferenceSTT(model="") (livewire/__init__.py:736).
type InferenceSTT struct {
	Model string
}

// NewInferenceSTT creates an InferenceSTT stub with the given options.
func NewInferenceSTT(opts ...func(*InferenceSTT)) *InferenceSTT {
	s := &InferenceSTT{}
	for _, opt := range opts {
		opt(s)
	}
	getGlobalNoop().once("stt_node", "stt_node override: SignalWire's control plane handles the full media pipeline at scale")
	return s
}

// WithInferenceSTTModel returns a functional option that sets the model string.
func WithInferenceSTTModel(model string) func(*InferenceSTT) {
	return func(s *InferenceSTT) { s.Model = model }
}

// ---------------------------------------------------------------------------
// TTS provider stubs
// ---------------------------------------------------------------------------

// ElevenLabsTTS is a stub for the ElevenLabs TTS provider.
type ElevenLabsTTS struct {
	Voice string
}

// NewElevenLabsTTS creates an ElevenLabsTTS stub.
func NewElevenLabsTTS(opts ...func(*ElevenLabsTTS)) *ElevenLabsTTS {
	e := &ElevenLabsTTS{}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// CartesiaTTS is a stub for the Cartesia TTS provider.
type CartesiaTTS struct {
	Voice string
}

// NewCartesiaTTS creates a CartesiaTTS stub.
func NewCartesiaTTS(opts ...func(*CartesiaTTS)) *CartesiaTTS {
	c := &CartesiaTTS{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// OpenAITTS is a stub for the OpenAI TTS provider.
type OpenAITTS struct {
	Voice string
}

// NewOpenAITTS creates an OpenAITTS stub.
func NewOpenAITTS(opts ...func(*OpenAITTS)) *OpenAITTS {
	o := &OpenAITTS{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// ---------------------------------------------------------------------------
// LLM provider stubs
// ---------------------------------------------------------------------------

// OpenAILLM is a stub for the OpenAI LLM provider.
type OpenAILLM struct {
	Model string
}

// NewOpenAILLM creates an OpenAILLM stub.
func NewOpenAILLM(opts ...func(*OpenAILLM)) *OpenAILLM {
	o := &OpenAILLM{}
	for _, opt := range opts {
		opt(o)
	}
	getGlobalNoop().once("llm_node", "OpenAILLM: model selection is mapped to SignalWire AI params — OpenAI plugin wrapper is a no-op")
	return o
}

// InferenceLLM is a stub for the SignalWire-hosted inference LLM.
// On SignalWire, the LLM pipeline is handled by the control plane;
// the Model field is forwarded to SignalWire AI parameters.
// Mirrors Python InferenceLLM(model="") (livewire/__init__.py:751).
type InferenceLLM struct {
	Model string
}

// NewInferenceLLM creates an InferenceLLM stub with the given options.
func NewInferenceLLM(opts ...func(*InferenceLLM)) *InferenceLLM {
	l := &InferenceLLM{}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// WithInferenceLLMModel returns a functional option that sets the model string.
func WithInferenceLLMModel(model string) func(*InferenceLLM) {
	return func(l *InferenceLLM) { l.Model = model }
}

// ---------------------------------------------------------------------------
// VAD stubs
// ---------------------------------------------------------------------------

// SileroVAD is a stub for the Silero VAD provider.
type SileroVAD struct{}

// NewSileroVAD creates a SileroVAD stub.
// Mirrors Python SileroVAD(**kwargs) — accepts functional options for
// LiveKit portability, matching the in-file convention for all other stubs.
func NewSileroVAD(opts ...func(*SileroVAD)) *SileroVAD {
	s := &SileroVAD{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Load is a noop — Silero VAD model loading is not needed on SignalWire.
func (s *SileroVAD) Load() *SileroVAD {
	return s
}
