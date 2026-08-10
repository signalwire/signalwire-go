package relay

import (
	"os"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
)

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

// WithPlayControlID sets an explicit control_id for the play action.
// This is the play operation's control_id parameter. When omitted the SDK
// auto-generates a UUID. The same key is honored by play_and_collect.
func WithPlayControlID(id string) PlayOption {
	return func(m map[string]any) {
		m["_control_id"] = id
	}
}

// WithPlayDirection sets the play direction (e.g. "self" / "peer" / "both").
func WithPlayDirection(dir string) PlayOption {
	return func(m map[string]any) {
		m["direction"] = dir
	}
}

// WithPlayLoop sets the number of loop iterations for playback.
func WithPlayLoop(n int) PlayOption {
	return func(m map[string]any) {
		m["loop"] = n
	}
}

// WithPlayOnCompleted registers a callback fired when the play action
// reaches a terminal state. This is the play operation's on_completed parameter.
func WithPlayOnCompleted(cb func(*RelayEvent)) PlayOption {
	return func(m map[string]any) {
		m["_on_completed"] = cb
	}
}

// ---------------------------------------------------------------------------
// Functional options for the typed play/detect/prompt convenience methods
// ---------------------------------------------------------------------------
//
// These express the named parameters of play_tts / play_audio /
// play_ringtone / detect_digit / detect_answering_machine / detect_fax /
// prompt_tts / prompt_audio. Each builds the exact RELAY media/params shape the
// underlying generic (Play / PlayAndCollect / Detect) emits on the wire.
//
// The options write into a per-call scratch map keyed with underscore-
// prefixed sentinels so the convenience method can pull them off and fold
// them into the right media dict before delegating. Sentinels never go on
// the wire — the method deletes them.

// TTSOption configures a PlayTTS or PromptTTS call. It sets the language /
// gender / voice fields nested inside the “{"type":"tts","params":{...}}“
// media entry and the top-level volume on the play frame.
type TTSOption func(m map[string]any)

// WithTTSLanguage sets the TTS language (the play_tts/prompt_tts operation's language parameter).
func WithTTSLanguage(language string) TTSOption {
	return func(m map[string]any) { m["_tts_language"] = language }
}

// WithTTSGender sets the TTS voice gender (the play_tts/prompt_tts operation's gender parameter).
// The parameter is the defined string type TTSGender: the GenderMale /
// GenderFemale constants give autocomplete + a compile-time typo check, while
// Go's untyped-constant auto-conversion keeps a bare "female" literal
// compiling. The value is stored as a plain string so the wire shape is a
// plain string gender.
func WithTTSGender(gender TTSGender) TTSOption {
	return func(m map[string]any) { m["_tts_gender"] = string(gender) }
}

// WithTTSVoice sets the TTS voice (the play_tts/prompt_tts operation's voice parameter).
func WithTTSVoice(voice string) TTSOption {
	return func(m map[string]any) { m["_tts_voice"] = voice }
}

// WithTTSVolume sets the playback volume in dB (the play_tts/prompt_tts operation's volume parameter).
func WithTTSVolume(db float64) TTSOption {
	return func(m map[string]any) { m["volume"] = db }
}

// AudioOption configures a PlayAudio or PromptAudio call.
type AudioOption func(m map[string]any)

// WithAudioVolume sets the playback volume in dB (the play_audio/prompt_audio operation's volume parameter).
func WithAudioVolume(db float64) AudioOption {
	return func(m map[string]any) { m["volume"] = db }
}

// RingtoneOption configures a PlayRingtone call.
type RingtoneOption func(m map[string]any)

// WithRingtoneDuration sets how long the ringtone plays, in seconds
// (the play_ringtone operation's duration parameter). It is nested inside the ringtone media
// params, giving “{"type":"ringtone","params":{"duration":...}}“ on the wire.
func WithRingtoneDuration(seconds float64) RingtoneOption {
	return func(m map[string]any) { m["_ringtone_duration"] = seconds }
}

// WithRingtoneVolume sets the playback volume in dB (the play_ringtone operation's volume parameter).
func WithRingtoneVolume(db float64) RingtoneOption {
	return func(m map[string]any) { m["volume"] = db }
}

// DetectDigitOption configures a DetectDigit call.
type DetectDigitOption func(m map[string]any)

// WithDigitDigits restricts detection to the given DTMF digit set, nested
// inside the detect params (the detect_digit operation's digits parameter).
func WithDigitDigits(digits string) DetectDigitOption {
	return func(m map[string]any) { m["_digits"] = digits }
}

// WithDigitTimeout sets the detect timeout in seconds (the detect_digit operation's timeout parameter).
func WithDigitTimeout(seconds float64) DetectDigitOption {
	return func(m map[string]any) { m["_timeout"] = seconds }
}

// AMDOption configures a DetectAnsweringMachine call. Each option maps to
// one detect_answering_machine named parameter and is nested inside
// the “{"type":"machine","params":{...}}“ detect media entry — only keys the
// caller actually provided are emitted.
type AMDOption func(m map[string]any)

// WithAMDInitialTimeout sets initial_timeout (the detect_answering_machine operation).
func WithAMDInitialTimeout(seconds float64) AMDOption {
	return func(m map[string]any) { m["initial_timeout"] = seconds }
}

// WithAMDEndSilenceTimeout sets end_silence_timeout (the detect_answering_machine operation).
func WithAMDEndSilenceTimeout(seconds float64) AMDOption {
	return func(m map[string]any) { m["end_silence_timeout"] = seconds }
}

// WithAMDMachineVoiceThreshold sets machine_voice_threshold (the detect_answering_machine operation).
func WithAMDMachineVoiceThreshold(threshold float64) AMDOption {
	return func(m map[string]any) { m["machine_voice_threshold"] = threshold }
}

// WithAMDMachineWordsThreshold sets machine_words_threshold (the detect_answering_machine operation).
func WithAMDMachineWordsThreshold(threshold int) AMDOption {
	return func(m map[string]any) { m["machine_words_threshold"] = threshold }
}

// WithAMDDetectInterruptions sets detect_interruptions (the detect_answering_machine operation).
func WithAMDDetectInterruptions(enabled bool) AMDOption {
	return func(m map[string]any) { m["detect_interruptions"] = enabled }
}

// WithAMDDetectMessageEnd sets detect_message_end (the detect_answering_machine operation).
func WithAMDDetectMessageEnd(enabled bool) AMDOption {
	return func(m map[string]any) { m["detect_message_end"] = enabled }
}

// WithAMDTimeout sets the overall detect timeout in seconds (the detect_answering_machine operation's timeout parameter).
func WithAMDTimeout(seconds float64) AMDOption {
	return func(m map[string]any) { m["_timeout"] = seconds }
}

// DetectFaxOption configures a DetectFax call.
type DetectFaxOption func(m map[string]any)

// WithFaxTone restricts fax detection to a specific tone (CED/CNG), nested
// inside the detect params (the detect_fax operation's tone parameter).
func WithFaxTone(tone string) DetectFaxOption {
	return func(m map[string]any) { m["_tone"] = tone }
}

// WithFaxDetectTimeout sets the detect timeout in seconds (the detect_fax operation's timeout parameter).
func WithFaxDetectTimeout(seconds float64) DetectFaxOption {
	return func(m map[string]any) { m["_timeout"] = seconds }
}

// RecordOption configures a Record call.
type RecordOption func(m map[string]any)

// WithRecordBeep enables a beep before recording.
func WithRecordBeep(beep bool) RecordOption {
	return func(m map[string]any) {
		m["beep"] = beep
	}
}

// WithRecordFormat sets the recording format (e.g. "wav", "mp3"). The
// parameter is the defined string type swaig.RecordFormat: the Format*
// constants give autocomplete + a compile-time typo check, while Go's
// untyped-constant auto-conversion keeps a bare "wav" literal compiling. The
// value is stored as a plain string so the wire shape is unchanged.
func WithRecordFormat(format swaig.RecordFormat) RecordOption {
	return func(m map[string]any) {
		m["format"] = string(format)
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

// WithRecordControlID sets an explicit control_id for the record action.
// This is the record operation's control_id parameter.
func WithRecordControlID(id string) RecordOption {
	return func(m map[string]any) {
		m["_control_id"] = id
	}
}

// WithRecordAudio sets the audio config map for the record action's
// "record": {"audio": ...} payload. This is the record operation's audio parameter.
func WithRecordAudio(audio map[string]any) RecordOption {
	return func(m map[string]any) {
		m["_audio"] = audio
	}
}

// WithRecordOnCompleted registers a callback fired when the record
// action reaches a terminal state. This is the record operation's
// on_completed parameter.
func WithRecordOnCompleted(cb func(*RelayEvent)) RecordOption {
	return func(m map[string]any) {
		m["_on_completed"] = cb
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

// WithConnectTag sets an explicit tag for the connect operation
// (the connect operation's tag parameter).
func WithConnectTag(tag string) ConnectOption {
	return func(m map[string]any) {
		m["tag"] = tag
	}
}

// WithConnectMaxDuration sets the maximum connect duration in seconds
// (the connect operation's max_duration parameter).
func WithConnectMaxDuration(seconds int) ConnectOption {
	return func(m map[string]any) {
		m["max_duration"] = seconds
	}
}

// WithConnectMaxPricePerMinute sets the max price per minute
// (the connect operation's max_price_per_minute parameter).
func WithConnectMaxPricePerMinute(price float64) ConnectOption {
	return func(m map[string]any) {
		m["max_price_per_minute"] = price
	}
}

// WithConnectStatusURL sets the status callback URL for the connect operation
// (the connect operation's status_url parameter).
func WithConnectStatusURL(url string) ConnectOption {
	return func(m map[string]any) {
		m["status_url"] = url
	}
}

// StreamOption configures a Stream call.
type StreamOption func(m map[string]any)

// WithStreamControlID supplies an explicit control_id for the stream
// action, which rides as the stream operation's control_id parameter.
func WithStreamControlID(id string) StreamOption {
	return func(m map[string]any) { m["_control_id"] = id }
}

// WithStreamName sets the stream name (the stream operation's name parameter).
func WithStreamName(name string) StreamOption {
	return func(m map[string]any) { m["name"] = name }
}

// WithStreamCodec sets the stream audio codec.
func WithStreamCodec(codec string) StreamOption {
	return func(m map[string]any) {
		m["codec"] = codec
	}
}

// WithStreamTrack sets the stream track (the stream operation's track parameter).
func WithStreamTrack(track string) StreamOption {
	return func(m map[string]any) { m["track"] = track }
}

// WithStreamStatusURL sets the stream status callback URL
// (the stream operation's status_url parameter).
func WithStreamStatusURL(url string) StreamOption {
	return func(m map[string]any) { m["status_url"] = url }
}

// WithStreamStatusURLMethod sets the HTTP method for the status callback
// (the stream operation's status_url_method parameter).
func WithStreamStatusURLMethod(method string) StreamOption {
	return func(m map[string]any) { m["status_url_method"] = method }
}

// WithStreamAuthorizationBearerToken sets the bearer token sent with the
// stream (the stream operation's authorization_bearer_token parameter).
func WithStreamAuthorizationBearerToken(token string) StreamOption {
	return func(m map[string]any) { m["authorization_bearer_token"] = token }
}

// WithStreamCustomParameters sets custom parameters forwarded with the stream
// (the stream operation's custom_parameters parameter).
func WithStreamCustomParameters(params map[string]any) StreamOption {
	return func(m map[string]any) { m["custom_parameters"] = params }
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

// WithConferenceStartOnEnter sets start_on_enter (the join_conference operation).
func WithConferenceStartOnEnter(v bool) ConferenceOption {
	return func(m map[string]any) { m["start_on_enter"] = v }
}

// WithConferenceEndOnExit sets end_on_exit (the join_conference operation).
func WithConferenceEndOnExit(v bool) ConferenceOption {
	return func(m map[string]any) { m["end_on_exit"] = v }
}

// WithConferenceWaitURL sets wait_url (the join_conference operation).
func WithConferenceWaitURL(url string) ConferenceOption {
	return func(m map[string]any) { m["wait_url"] = url }
}

// WithConferenceMaxParticipants sets max_participants (the join_conference operation).
func WithConferenceMaxParticipants(n int) ConferenceOption {
	return func(m map[string]any) { m["max_participants"] = n }
}

// WithConferenceRecord sets record (the join_conference operation).
func WithConferenceRecord(v string) ConferenceOption {
	return func(m map[string]any) { m["record"] = v }
}

// WithConferenceRegion sets region (the join_conference operation).
func WithConferenceRegion(v string) ConferenceOption {
	return func(m map[string]any) { m["region"] = v }
}

// WithConferenceTrim sets trim (the join_conference operation).
func WithConferenceTrim(v string) ConferenceOption {
	return func(m map[string]any) { m["trim"] = v }
}

// WithConferenceCoach sets coach (the join_conference operation).
func WithConferenceCoach(v string) ConferenceOption {
	return func(m map[string]any) { m["coach"] = v }
}

// WithConferenceStatusCallback sets status_callback (the join_conference operation).
func WithConferenceStatusCallback(v string) ConferenceOption {
	return func(m map[string]any) { m["status_callback"] = v }
}

// WithConferenceStatusCallbackEvent sets status_callback_event (the join_conference operation).
func WithConferenceStatusCallbackEvent(v string) ConferenceOption {
	return func(m map[string]any) { m["status_callback_event"] = v }
}

// WithConferenceStatusCallbackEventType sets status_callback_event_type (the join_conference operation).
func WithConferenceStatusCallbackEventType(v string) ConferenceOption {
	return func(m map[string]any) { m["status_callback_event_type"] = v }
}

// WithConferenceStatusCallbackMethod sets status_callback_method (the join_conference operation).
func WithConferenceStatusCallbackMethod(v string) ConferenceOption {
	return func(m map[string]any) { m["status_callback_method"] = v }
}

// WithConferenceRecordingStatusCallback sets recording_status_callback (the join_conference operation).
func WithConferenceRecordingStatusCallback(v string) ConferenceOption {
	return func(m map[string]any) { m["recording_status_callback"] = v }
}

// WithConferenceRecordingStatusCallbackEvent sets recording_status_callback_event (the join_conference operation).
func WithConferenceRecordingStatusCallbackEvent(v string) ConferenceOption {
	return func(m map[string]any) { m["recording_status_callback_event"] = v }
}

// WithConferenceRecordingStatusCallbackEventType sets recording_status_callback_event_type (the join_conference operation).
func WithConferenceRecordingStatusCallbackEventType(v string) ConferenceOption {
	return func(m map[string]any) { m["recording_status_callback_event_type"] = v }
}

// WithConferenceRecordingStatusCallbackMethod sets recording_status_callback_method (the join_conference operation).
func WithConferenceRecordingStatusCallbackMethod(v string) ConferenceOption {
	return func(m map[string]any) { m["recording_status_callback_method"] = v }
}

// WithConferenceStream sets the stream object (the join_conference operation's
// stream_obj parameter, emitted under the "stream" wire key).
func WithConferenceStream(streamObj map[string]any) ConferenceOption {
	return func(m map[string]any) { m["stream"] = streamObj }
}

// FaxOption configures a SendFax call.
type FaxOption func(m map[string]any)

// WithFaxHeaderInfo sets the fax header info string (the header_info parameter).
func WithFaxHeaderInfo(headerInfo string) FaxOption {
	return func(m map[string]any) {
		if headerInfo != "" {
			m["header_info"] = headerInfo
		}
	}
}

// WithFaxControlID supplies an explicit control_id for the fax action,
// which rides as the send_fax / receive_fax control_id parameter.
func WithFaxControlID(id string) FaxOption {
	return func(m map[string]any) { m["_control_id"] = id }
}

// PayOption configures a Pay call.
type PayOption func(m map[string]any)

// WithPayControlID supplies an explicit control_id for the pay action,
// which rides as the pay operation's control_id parameter.
func WithPayControlID(id string) PayOption {
	return func(m map[string]any) { m["_control_id"] = id }
}

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
func WithPayMaxAttempts(maxAttempts string) PayOption {
	return func(m map[string]any) { m["max_attempts"] = maxAttempts }
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

// WithAIControlID supplies an explicit control_id for the AI action,
// which rides as the ai operation's control_id parameter.
func WithAIControlID(id string) AIOption {
	return func(m map[string]any) { m["_control_id"] = id }
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

// WithAIAgent sets the AI agent config (the ai operation's agent parameter).
func WithAIAgent(agent map[string]any) AIOption {
	return func(m map[string]any) { m["agent"] = agent }
}

// WithAIPostPromptURL sets the post-prompt URL (the ai operation's post_prompt_url parameter).
func WithAIPostPromptURL(url string) AIOption {
	return func(m map[string]any) { m["post_prompt_url"] = url }
}

// WithAIPostPromptAuthUser sets the post-prompt basic-auth user
// (the ai operation's post_prompt_auth_user parameter).
func WithAIPostPromptAuthUser(user string) AIOption {
	return func(m map[string]any) { m["post_prompt_auth_user"] = user }
}

// WithAIPostPromptAuthPassword sets the post-prompt basic-auth password
// (the ai operation's post_prompt_auth_password parameter).
func WithAIPostPromptAuthPassword(password string) AIOption {
	return func(m map[string]any) { m["post_prompt_auth_password"] = password }
}

// WithAIGlobalData sets the AI global data (the ai operation's global_data parameter).
func WithAIGlobalData(data map[string]any) AIOption {
	return func(m map[string]any) { m["global_data"] = data }
}

// WithAIPronounce sets the AI pronounce rules (the ai operation's pronounce parameter).
func WithAIPronounce(pronounce []map[string]any) AIOption {
	return func(m map[string]any) { m["pronounce"] = pronounce }
}

// WithAIHints sets the AI hints (the ai operation's hints parameter).
func WithAIHints(hints []string) AIOption {
	return func(m map[string]any) { m["hints"] = hints }
}

// WithAILanguages sets the AI languages (the ai operation's languages parameter).
func WithAILanguages(languages []map[string]any) AIOption {
	return func(m map[string]any) { m["languages"] = languages }
}

// WithAISWAIG sets the AI SWAIG config (the ai operation's SWAIG parameter).
func WithAISWAIG(swaig map[string]any) AIOption {
	return func(m map[string]any) { m["SWAIG"] = swaig }
}

// WithAIParams sets the AI parameters. This is the ai / amazon_bedrock
// operation's ai_params parameter: the map is emitted nested under the
// "params" wire key — NOT spread onto the top level of the frame.
func WithAIParams(params map[string]any) AIOption {
	return func(m map[string]any) {
		m["params"] = params
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

// WithPingWatchdog enables the client-side liveness watchdog: the client sends
// signalwire.ping every interval and, after maxFailures consecutive unanswered
// pings, declares the peer half-open and forces a reconnect (F2.1). interval<=0
// disables it (the default). maxFailures<1 is clamped to 1.
func WithPingWatchdog(interval time.Duration, maxFailures int) ClientOption {
	return func(c *Client) {
		c.pingInterval = interval
		if maxFailures < 1 {
			maxFailures = 1
		}
		c.maxPingFailures = maxFailures
	}
}

// WithExecuteTimeout bounds how long an execute() RPC waits for its response
// before returning ErrExecuteTimeout. Default 30s. A short value lets a liveness
// harness bound a silent/black-hole peer inside a tight window.
func WithExecuteTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		if d > 0 {
			c.executeTimeout = d
		}
	}
}

// WithReconnectBackoff sets the initial reconnect backoff (the delay before the
// first reconnect attempt; it grows exponentially up to maxBackoff). Default 1s.
func WithReconnectBackoff(d time.Duration) ClientOption {
	return func(c *Client) {
		if d > 0 {
			c.reconnectBackoff = d
		}
	}
}

// DefaultMaxActiveCalls is the inbound-call ceiling applied when neither
// WithMaxActiveCalls nor RELAY_MAX_ACTIVE_CALLS sets one: overload protection
// is always in effect, never "unlimited".
const DefaultMaxActiveCalls = 1000

// WithMaxActiveCalls limits the number of concurrent active calls. A value <= 0
// is clamped to 1 — max(1, n) — so the cap is always a positive ceiling.
func WithMaxActiveCalls(n int) ClientOption {
	return func(c *Client) {
		if n < 1 {
			n = 1
		}
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

// WithDialTimeout sets the legacy per-leg dial timeout in seconds.
// (Was the only Go option; retained for back-compat. To bound the
// overall Dial() call use WithDialClientTimeout.)
func WithDialTimeout(t int) DialOption {
	return func(m map[string]any) {
		m["timeout"] = t
	}
}

// WithDialTag sets an explicit caller-supplied dial tag. When omitted
// the SDK generates a UUID.
func WithDialTag(tag string) DialOption {
	return func(m map[string]any) {
		m["tag"] = tag
	}
}

// WithDialClientTimeout bounds how long Dial() will wait for the
// calling.call.dial event before raising a timeout error. This is the dial
// operation's dial_timeout parameter, in seconds. Default is 120s when omitted.
//
// The duration is consumed by the Go Dial() loop; it never goes on the
// wire — that's why it's stored under an underscore-prefixed key
// removed before transmit.
func WithDialClientTimeout(d time.Duration) DialOption {
	return func(m map[string]any) {
		m["_dial_timeout"] = d
	}
}

// WithDialMaxDuration sets the maximum call duration in minutes. This is the
// dial operation's max_duration parameter.
func WithDialMaxDuration(minutes int) DialOption {
	return func(m map[string]any) {
		m["max_duration"] = minutes
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

// WithMessageContext sets the routing context for the message. This is the
// send_message operation's context parameter — it defaults to the relay
// protocol when omitted.
func WithMessageContext(ctx string) MessageOption {
	return func(m map[string]any) {
		m["context"] = ctx
	}
}

// WithMessageOnCompleted registers a callback invoked when the message reaches
// a terminal state (delivered, undelivered, or failed). The callback receives
// both the message and the terminal RelayEvent. This is the send_message
// operation's on_completed parameter.
func WithMessageOnCompleted(cb func(*Message, *RelayEvent)) MessageOption {
	return func(m map[string]any) {
		m["_on_completed"] = cb
	}
}

// applyEnvDefaults fills any unset auth/space fields from SIGNALWIRE_*
// environment variables. Called automatically at the end of
// NewRelayClient. Idempotent — calling again after fields are populated is a
// no-op.
func (c *Client) applyEnvDefaults() {
	if c.projectID == "" {
		c.projectID = os.Getenv("SIGNALWIRE_PROJECT_ID")
	}
	if c.token == "" {
		c.token = os.Getenv("SIGNALWIRE_API_TOKEN")
	}
	if c.jwtToken == "" {
		c.jwtToken = os.Getenv("SIGNALWIRE_JWT_TOKEN")
	}
	if c.space == "" {
		c.space = os.Getenv("SIGNALWIRE_SPACE")
	}
	if c.maxActiveCalls == 0 {
		if v := os.Getenv("RELAY_MAX_ACTIVE_CALLS"); v != "" {
			n := 0
			for _, ch := range v {
				if ch < '0' || ch > '9' {
					n = 0
					break
				}
				n = n*10 + int(ch-'0')
			}
			if n > 0 {
				c.maxActiveCalls = n
			}
		}
	}
	// Still unset -> the default ceiling of 1000. An unset knob
	// still enforces a cap; it is never "unlimited".
	if c.maxActiveCalls == 0 {
		c.maxActiveCalls = DefaultMaxActiveCalls
	}
}

// WithEnvDefaults is now a no-op pass-through retained for backwards
// compatibility — env defaults are loaded automatically at the end of
// NewRelayClient. New code can rely on the auto-load behavior and omit this
// option entirely.
func WithEnvDefaults() ClientOption {
	return func(c *Client) {
		c.applyEnvDefaults()
	}
}
