package relay

// ---------------------------------------------------------------------------
// Functional options for Call methods
// ---------------------------------------------------------------------------

// PlayOption configures a Play call.
type PlayOption func(m map[string]any)

// WithPlayVolume sets the volume for playback in dB.
func WithPlayVolume(db float64) PlayOption {
	return func(m map[string]any) {
		m["volume"] = db
	}
}

// RecordOption configures a Record call.
type RecordOption func(m map[string]any)

// WithRecordBeep enables a beep before recording.
func WithRecordBeep(beep bool) RecordOption {
	return func(m map[string]any) {
		m["beep"] = beep
	}
}

// WithRecordFormat sets the recording format (e.g. "wav", "mp3").
func WithRecordFormat(format string) RecordOption {
	return func(m map[string]any) {
		m["format"] = format
	}
}

// WithRecordStereo enables stereo recording.
func WithRecordStereo(stereo bool) RecordOption {
	return func(m map[string]any) {
		m["stereo"] = stereo
	}
}

// WithRecordDirection sets the recording direction ("listen", "speak", "both").
func WithRecordDirection(dir string) RecordOption {
	return func(m map[string]any) {
		m["direction"] = dir
	}
}

// WithRecordTerminators sets DTMF terminators to stop recording.
func WithRecordTerminators(terminators string) RecordOption {
	return func(m map[string]any) {
		m["terminators"] = terminators
	}
}

// WithRecordInitialTimeout sets the initial timeout in seconds.
func WithRecordInitialTimeout(t float64) RecordOption {
	return func(m map[string]any) {
		m["initial_timeout"] = t
	}
}

// WithRecordEndSilenceTimeout sets the end-of-speech silence timeout in seconds.
func WithRecordEndSilenceTimeout(t float64) RecordOption {
	return func(m map[string]any) {
		m["end_silence_timeout"] = t
	}
}

// ConnectOption configures a Connect call.
type ConnectOption func(m map[string]any)

// WithConnectRingback sets ringback media for the connect operation.
func WithConnectRingback(media []map[string]any) ConnectOption {
	return func(m map[string]any) {
		m["ringback"] = media
	}
}

// StreamOption configures a Stream call.
type StreamOption func(m map[string]any)

// WithStreamDirection sets the stream direction.
func WithStreamDirection(dir string) StreamOption {
	return func(m map[string]any) {
		m["direction"] = dir
	}
}

// WithStreamCodec sets the stream audio codec.
func WithStreamCodec(codec string) StreamOption {
	return func(m map[string]any) {
		m["codec"] = codec
	}
}

// ConferenceOption configures a JoinConference call.
type ConferenceOption func(m map[string]any)

// WithConferenceBeep enables beep on join/leave.
func WithConferenceBeep(beep string) ConferenceOption {
	return func(m map[string]any) {
		m["beep"] = beep
	}
}

// WithConferenceMuted joins muted.
func WithConferenceMuted(muted bool) ConferenceOption {
	return func(m map[string]any) {
		m["muted"] = muted
	}
}

// WithConferenceDeaf joins deaf (cannot hear others).
func WithConferenceDeaf(deaf bool) ConferenceOption {
	return func(m map[string]any) {
		m["deaf"] = deaf
	}
}

// FaxOption configures a SendFax call.
type FaxOption func(m map[string]any)

// WithFaxHeaderInfo sets the fax header info string (matches Python's header_info param).
func WithFaxHeaderInfo(headerInfo string) FaxOption {
	return func(m map[string]any) {
		if headerInfo != "" {
			m["header_info"] = headerInfo
		}
	}
}

// PayOption configures a Pay call.
type PayOption func(m map[string]any)

// WithPayInputMethod sets the payment input method.
func WithPayInputMethod(method string) PayOption {
	return func(m map[string]any) { m["input"] = method }
}

// WithPayStatusURL sets the payment status callback URL.
func WithPayStatusURL(url string) PayOption {
	return func(m map[string]any) { m["status_url"] = url }
}

// WithPayPaymentMethod sets the payment method (e.g. "credit-card").
func WithPayPaymentMethod(method string) PayOption {
	return func(m map[string]any) { m["payment_method"] = method }
}

// WithPayTimeout sets the timeout string for the payment session.
func WithPayTimeout(timeout string) PayOption {
	return func(m map[string]any) { m["timeout"] = timeout }
}

// WithPayMaxAttempts sets the maximum number of payment attempts.
func WithPayMaxAttempts(max string) PayOption {
	return func(m map[string]any) { m["max_attempts"] = max }
}

// WithPaySecurityCode sets whether to collect security code.
func WithPaySecurityCode(code string) PayOption {
	return func(m map[string]any) { m["security_code"] = code }
}

// WithPayPostalCode sets whether to collect postal code.
func WithPayPostalCode(code string) PayOption {
	return func(m map[string]any) { m["postal_code"] = code }
}

// WithPayMinPostalCodeLength sets the minimum postal code length.
func WithPayMinPostalCodeLength(length string) PayOption {
	return func(m map[string]any) { m["min_postal_code_length"] = length }
}

// WithPayTokenType sets the payment token type.
func WithPayTokenType(tokenType string) PayOption {
	return func(m map[string]any) { m["token_type"] = tokenType }
}

// WithPayChargeAmount sets the charge amount.
func WithPayChargeAmount(amount string) PayOption {
	return func(m map[string]any) { m["charge_amount"] = amount }
}

// WithPayCurrency sets the payment currency.
func WithPayCurrency(currency string) PayOption {
	return func(m map[string]any) { m["currency"] = currency }
}

// WithPayLanguage sets the language for payment prompts.
func WithPayLanguage(language string) PayOption {
	return func(m map[string]any) { m["language"] = language }
}

// WithPayVoice sets the voice for payment prompts.
func WithPayVoice(voice string) PayOption {
	return func(m map[string]any) { m["voice"] = voice }
}

// WithPayDescription sets a description for the payment.
func WithPayDescription(desc string) PayOption {
	return func(m map[string]any) { m["description"] = desc }
}

// WithPayValidCardTypes sets the valid card types string.
func WithPayValidCardTypes(types string) PayOption {
	return func(m map[string]any) { m["valid_card_types"] = types }
}

// WithPayParameters sets additional payment parameters.
func WithPayParameters(parameters []map[string]any) PayOption {
	return func(m map[string]any) { m["parameters"] = parameters }
}

// WithPayPrompts sets custom payment prompts.
func WithPayPrompts(prompts []map[string]any) PayOption {
	return func(m map[string]any) { m["prompts"] = prompts }
}

// AIOption configures an AI operation on a call.
type AIOption func(m map[string]any)

// WithAIEngine sets the AI engine to use.
func WithAIEngine(engine string) AIOption {
	return func(m map[string]any) {
		m["engine"] = engine
	}
}

// WithAIPrompt sets the AI prompt text.
func WithAIPrompt(prompt map[string]any) AIOption {
	return func(m map[string]any) {
		m["prompt"] = prompt
	}
}

// WithAIPostPrompt sets the AI post-prompt configuration.
func WithAIPostPrompt(pp map[string]any) AIOption {
	return func(m map[string]any) {
		m["post_prompt"] = pp
	}
}

// WithAIParams sets arbitrary AI parameters.
func WithAIParams(params map[string]any) AIOption {
	return func(m map[string]any) {
		for k, v := range params {
			m[k] = v
		}
	}
}

// ---------------------------------------------------------------------------
// Functional options for Client methods
// ---------------------------------------------------------------------------

// ClientOption configures the RELAY Client.
type ClientOption func(c *Client)

// WithProject sets the project ID for authentication.
func WithProject(id string) ClientOption {
	return func(c *Client) {
		c.projectID = id
	}
}

// WithToken sets the API token for authentication.
func WithToken(token string) ClientOption {
	return func(c *Client) {
		c.token = token
	}
}

// WithJWT sets a pre-existing JWT for authentication.
func WithJWT(jwt string) ClientOption {
	return func(c *Client) {
		c.jwtToken = jwt
	}
}

// WithSpace sets the SignalWire space (e.g. "example.signalwire.com").
func WithSpace(space string) ClientOption {
	return func(c *Client) {
		c.space = space
	}
}

// WithContexts sets the contexts to subscribe to for inbound events.
func WithContexts(contexts ...string) ClientOption {
	return func(c *Client) {
		c.contexts = contexts
	}
}

// WithMaxActiveCalls limits the number of concurrent active calls.
func WithMaxActiveCalls(n int) ClientOption {
	return func(c *Client) {
		c.maxActiveCalls = n
	}
}

// DialOption configures a Dial (outbound call) operation.
type DialOption func(m map[string]any)

// WithDialFromNumber sets the caller ID for the outbound call.
func WithDialFromNumber(from string) DialOption {
	return func(m map[string]any) {
		m["from_number"] = from
	}
}

// WithDialTimeout sets the dial timeout in seconds.
func WithDialTimeout(t int) DialOption {
	return func(m map[string]any) {
		m["timeout"] = t
	}
}

// MessageOption configures a SendMessage operation.
type MessageOption func(m map[string]any)

// WithMessageMedia adds media URLs to the message (MMS).
func WithMessageMedia(urls []string) MessageOption {
	return func(m map[string]any) {
		m["media"] = urls
	}
}

// WithMessageRegion sets the region for message delivery.
func WithMessageRegion(region string) MessageOption {
	return func(m map[string]any) {
		m["region"] = region
	}
}

// WithMessageTags sets tags on the message for tracking.
func WithMessageTags(tags []string) MessageOption {
	return func(m map[string]any) {
		m["tags"] = tags
	}
}
