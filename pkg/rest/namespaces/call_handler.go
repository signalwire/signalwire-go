// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// PhoneCallHandler is the value of the ``call_handler`` field accepted by
// phone_numbers.Update.
//
// Named PhoneCallHandler (not CallHandler) to avoid colliding with the RELAY
// client's inbound-call-handler callback type already present in the SDK
// (pkg/relay OnCallHandler).
//
// Setting a phone number's call_handler + the handler-specific companion
// field routes inbound calls and, for most values, auto-materializes the
// matching Fabric resource on the server. See the high-level helpers on
// PhoneNumbersNamespace (SetSwmlWebhook, SetCxmlWebhook, SetCxmlApplication,
// SetAiAgent, SetCallFlow, SetRelayApplication, SetRelayTopic).
//
//	Enum member       Companion field (required)    Auto-creates resource
//	RelayScript       call_relay_script_url         swml_webhook
//	LamlWebhooks      call_request_url              cxml_webhook
//	LamlApplication   call_laml_application_id      cxml_application
//	AiAgent           call_ai_agent_id              ai_agent
//	CallFlow          call_flow_id                  call_flow
//	RelayApplication  call_relay_application        relay_application
//	RelayTopic        call_relay_topic              (routes via RELAY)
//	RelayContext      call_relay_context            (legacy, prefer topic)
//	RelayConnector    (connector config)            (internal)
//	VideoRoom         call_video_room_id            (routes to Video API)
//	Dialogflow        call_dialogflow_agent_id      (none)
//
// Note: LamlWebhooks (wire value "laml_webhooks") produces a cXML handler,
// not a generic webhook. For SWML, use RelayScript.
type PhoneCallHandler string

// PhoneCallHandler wire values accepted by phone_numbers.Update.
const (
	PhoneCallHandlerRelayScript      PhoneCallHandler = "relay_script"
	PhoneCallHandlerLamlWebhooks     PhoneCallHandler = "laml_webhooks"
	PhoneCallHandlerLamlApplication  PhoneCallHandler = "laml_application"
	PhoneCallHandlerAiAgent          PhoneCallHandler = "ai_agent"
	PhoneCallHandlerCallFlow         PhoneCallHandler = "call_flow"
	PhoneCallHandlerRelayApplication PhoneCallHandler = "relay_application"
	PhoneCallHandlerRelayTopic       PhoneCallHandler = "relay_topic"
	PhoneCallHandlerRelayContext     PhoneCallHandler = "relay_context"
	PhoneCallHandlerRelayConnector   PhoneCallHandler = "relay_connector"
	PhoneCallHandlerVideoRoom        PhoneCallHandler = "video_room"
	PhoneCallHandlerDialogflow       PhoneCallHandler = "dialogflow"
)

// AllPhoneCallHandlers returns every PhoneCallHandler value. Useful for
// enum-contract tests and for callers that need to validate or enumerate
// the set.
func AllPhoneCallHandlers() []PhoneCallHandler {
	return []PhoneCallHandler{
		PhoneCallHandlerRelayScript,
		PhoneCallHandlerLamlWebhooks,
		PhoneCallHandlerLamlApplication,
		PhoneCallHandlerAiAgent,
		PhoneCallHandlerCallFlow,
		PhoneCallHandlerRelayApplication,
		PhoneCallHandlerRelayTopic,
		PhoneCallHandlerRelayContext,
		PhoneCallHandlerRelayConnector,
		PhoneCallHandlerVideoRoom,
		PhoneCallHandlerDialogflow,
	}
}
