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
	return o
}

// ---------------------------------------------------------------------------
// VAD stubs
// ---------------------------------------------------------------------------

// SileroVAD is a stub for the Silero VAD provider.
type SileroVAD struct{}

// NewSileroVAD creates a SileroVAD stub.
func NewSileroVAD() *SileroVAD {
	return &SileroVAD{}
}

// Load is a noop — Silero VAD model loading is not needed on SignalWire.
func (s *SileroVAD) Load() *SileroVAD {
	return s
}
