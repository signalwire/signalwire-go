// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// PhoneNumbersNamespace provides phone number management with search and
// typed helpers for binding an inbound call to a handler (SWML webhook, cXML
// webhook, AI agent, call flow, RELAY application/topic).
//
// Binding model: set ``call_handler`` + the handler-specific companion field
// on the phone number; the server auto-materializes the matching Fabric
// resource. The helpers below (Set*) are one-line wrappers around Update
// with the right call_handler + field combination baked in. See
// PhoneCallHandler for the enum.
type PhoneNumbersNamespace struct {
	*CrudResource
}

// NewPhoneNumbersNamespace creates a new PhoneNumbersNamespace.
func NewPhoneNumbersNamespace(client HTTPClient) *PhoneNumbersNamespace {
	return &PhoneNumbersNamespace{
		CrudResource: NewCrudResourcePUT(client, "/api/relay/rest/phone_numbers"),
	}
}

// Search searches for available phone numbers with optional filter parameters
// such as area_code, contains, starts_with, etc.
func (r *PhoneNumbersNamespace) Search(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path("search"), params)
}

// ---------- Typed binding helpers ----------
//
// Each helper is a one-line wrapper over Update with the right call_handler
// value and companion field already set. Options structs are used for
// helpers that accept additional optional fields; the Extra map on each
// options struct passes through any additional wire-level fields the helper
// doesn't name explicitly.

// SetSwmlWebhook routes inbound calls to an SWML webhook URL.
//
// Your backend returns an SWML document per call. The server auto-creates a
// swml_webhook Fabric resource keyed off this URL.
func (r *PhoneNumbersNamespace) SetSwmlWebhook(sid, url string, extra ...map[string]any) (map[string]any, error) {
	body := map[string]any{
		"call_handler":          string(PhoneCallHandlerRelayScript),
		"call_relay_script_url": url,
	}
	mergeExtra(body, extra)
	return r.Update(sid, body)
}

// CxmlWebhookOptions holds optional fields for SetCxmlWebhook.
type CxmlWebhookOptions struct {
	// FallbackURL is used when the primary URL fails.
	FallbackURL string
	// StatusCallbackURL receives call status updates.
	StatusCallbackURL string
	// Extra passes through additional wire-level fields.
	Extra map[string]any
}

// SetCxmlWebhook routes inbound calls to a cXML (Twilio-compat / LAML) webhook.
//
// Despite the wire value "laml_webhooks" being plural, this creates a single
// cxml_webhook Fabric resource. Pass opts to set FallbackURL and
// StatusCallbackURL; pass nil for the minimal form.
func (r *PhoneNumbersNamespace) SetCxmlWebhook(sid, url string, opts *CxmlWebhookOptions) (map[string]any, error) {
	body := map[string]any{
		"call_handler":     string(PhoneCallHandlerLamlWebhooks),
		"call_request_url": url,
	}
	if opts != nil {
		if opts.FallbackURL != "" {
			body["call_fallback_url"] = opts.FallbackURL
		}
		if opts.StatusCallbackURL != "" {
			body["call_status_callback_url"] = opts.StatusCallbackURL
		}
		for k, v := range opts.Extra {
			body[k] = v
		}
	}
	return r.Update(sid, body)
}

// SetCxmlApplication routes inbound calls to an existing cXML application by ID.
func (r *PhoneNumbersNamespace) SetCxmlApplication(sid, applicationID string, extra ...map[string]any) (map[string]any, error) {
	body := map[string]any{
		"call_handler":             string(PhoneCallHandlerLamlApplication),
		"call_laml_application_id": applicationID,
	}
	mergeExtra(body, extra)
	return r.Update(sid, body)
}

// SetAiAgent routes inbound calls to an AI Agent Fabric resource by ID.
func (r *PhoneNumbersNamespace) SetAiAgent(sid, agentID string, extra ...map[string]any) (map[string]any, error) {
	body := map[string]any{
		"call_handler":      string(PhoneCallHandlerAiAgent),
		"call_ai_agent_id":  agentID,
	}
	mergeExtra(body, extra)
	return r.Update(sid, body)
}

// CallFlowOptions holds optional fields for SetCallFlow.
type CallFlowOptions struct {
	// Version accepts "working_copy" or "current_deployed" (server default
	// when omitted).
	Version string
	// Extra passes through additional wire-level fields.
	Extra map[string]any
}

// SetCallFlow routes inbound calls to a Call Flow by ID. Pass nil opts for
// the minimal form; pass opts.Version to pin a specific version.
func (r *PhoneNumbersNamespace) SetCallFlow(sid, flowID string, opts *CallFlowOptions) (map[string]any, error) {
	body := map[string]any{
		"call_handler": string(PhoneCallHandlerCallFlow),
		"call_flow_id": flowID,
	}
	if opts != nil {
		if opts.Version != "" {
			body["call_flow_version"] = opts.Version
		}
		for k, v := range opts.Extra {
			body[k] = v
		}
	}
	return r.Update(sid, body)
}

// SetRelayApplication routes inbound calls to a named RELAY application.
func (r *PhoneNumbersNamespace) SetRelayApplication(sid, name string, extra ...map[string]any) (map[string]any, error) {
	body := map[string]any{
		"call_handler":            string(PhoneCallHandlerRelayApplication),
		"call_relay_application":  name,
	}
	mergeExtra(body, extra)
	return r.Update(sid, body)
}

// RelayTopicOptions holds optional fields for SetRelayTopic.
type RelayTopicOptions struct {
	// StatusCallbackURL receives topic status updates.
	StatusCallbackURL string
	// Extra passes through additional wire-level fields.
	Extra map[string]any
}

// SetRelayTopic routes inbound calls to a RELAY topic (client subscription).
// Pass nil opts for the minimal form.
func (r *PhoneNumbersNamespace) SetRelayTopic(sid, topic string, opts *RelayTopicOptions) (map[string]any, error) {
	body := map[string]any{
		"call_handler":     string(PhoneCallHandlerRelayTopic),
		"call_relay_topic": topic,
	}
	if opts != nil {
		if opts.StatusCallbackURL != "" {
			body["call_relay_topic_status_callback_url"] = opts.StatusCallbackURL
		}
		for k, v := range opts.Extra {
			body[k] = v
		}
	}
	return r.Update(sid, body)
}

// mergeExtra merges a single optional extra-fields map into body.
func mergeExtra(body map[string]any, extra []map[string]any) {
	if len(extra) == 0 {
		return
	}
	for _, m := range extra {
		for k, v := range m {
			body[k] = v
		}
	}
}
