# Examples

This directory contains runnable examples demonstrating the SignalWire AI Agents Go SDK.

## Agent Examples

| Example | Description |
|---------|-------------|
| [simple_agent](simple_agent/) | Basic AI agent with prompt and tools. Creating an AgentBase, setting prompt text, adding hints and language, defining SWAIG tools, setting global data, and running the agent. |
| [simple_dynamic_agent](simple_dynamic_agent/) | Per-request agent customization using a dynamic config callback. Inspects query parameters to adjust the prompt, global data, and tools based on caller tier. |
| [simple_dynamic_enhanced](simple_dynamic_enhanced/) | Enhanced dynamic config adapting to VIP status, department, customer ID, and language parameters. |
| [simple_static](simple_static/) | Minimal static agent. All configuration set once at startup: voice, language, AI parameters, hints, global data, and POM prompt sections. |
| [declarative](declarative/) | Agent configured declaratively using struct-level POM sections, post-prompt summary, and tool definitions without subclassing. |
| [custom_path](custom_path/) | Agent with a custom HTTP path (`/chat`). Dynamic per-request personalisation based on query parameters (user name, topic, mood). |
| [multi_agent_server](multi_agent_server/) | Hosts multiple AI agents on a single HTTP server using AgentServer. Each agent gets its own route with unique prompts and tools. |
| [multi_endpoint](multi_endpoint/) | Single agent with multiple SWML routes using AgentServer alongside health, readiness, and index endpoints. |
| [comprehensive_dynamic](comprehensive_dynamic/) | Tier-based dynamic config (standard/premium/enterprise) with industry-specific prompts, voice selection, LLM parameter tuning, and A/B testing. |
| [call_flow](call_flow/) | Call flow verbs and SWAIG actions. Demonstrates pre-answer verbs (ringback), answer configuration, post-answer verbs, post-AI verbs, debug events, and tools that return call control actions. |
| [contexts_demo](contexts_demo/) | Multi-step conversation workflows using contexts and steps. Creating multiple contexts with sequential steps, step criteria, navigation rules, and function restrictions. |
| [gather_info](gather_info/) | GatherInfo with typed questions in context steps. Structured data collection using the contexts system's gather_info mode for patient intake. |
| [datamap_demo](datamap_demo/) | Server-side tools using DataMap that execute on SignalWire servers without requiring webhook endpoints. Both webhook-based API calls and expression-based pattern matching. |
| [advanced_datamap](advanced_datamap/) | Advanced DataMap features: regex expression patterns, webhooks with headers/body/form_param, foreach array processing, multi-webhook fallback chains, and global error keys. |
| [session_state](session_state/) | Global data management and lifecycle callbacks. Setting initial global data, post-prompt for conversation summaries, OnSummary callback, and tools that read/write session state. |
| [skills_demo](skills_demo/) | Skills integration using the built-in skills registry. Listing available skills, instantiating via factory functions, loading through SkillManager, and registering tools with an agent. |
| [llm_params](llm_params/) | LLM parameter tuning demo. Creating agents with different response profiles (precise vs. creative) using SetPromptLlmParams and SetPostPromptLlmParams. |

## SWAIG Features & Call Control Examples

| Example | Description |
|---------|-------------|
| [swaig_features](swaig_features/) | SwaigFunctionResult actions showcase: Say, Hangup, Hold, Connect, SendSms, UpdateGlobalData, SetMetadata, PlayBackgroundFile, ToggleFunctions, SwitchContext, and method chaining. |
| [record_call](record_call/) | Call recording configuration using RecordCall and StopRecordCall helpers. Basic, stereo, voicemail, and complete customer service workflows. |
| [room_and_sip](room_and_sip/) | Room and SIP configuration using JoinRoom, JoinConference, and SipRefer helpers for multi-party communication and SIP transfers. |
| [tap](tap/) | TAP configuration for media monitoring. WebSocket and RTP tap streaming with direction control, codec selection, and compliance workflows. |

## Prefab Examples

| Example | Description |
|---------|-------------|
| [dynamic_info_gatherer](dynamic_info_gatherer/) | Dynamic InfoGatherer with callback-based question selection (support, medical, onboarding). |
| [prefab_info_gatherer](prefab_info_gatherer/) | InfoGathererAgent pre-built pattern that collects answers to a series of questions sequentially with built-in tools and prompt sections. |
| [prefab_survey](prefab_survey/) | SurveyAgent conducts structured surveys with typed questions (rating, multiple choice, yes/no, open-ended) including response validation and summary generation. |
| [concierge](concierge/) | ConciergeAgent prefab. Virtual concierge for a luxury resort with amenities, services, availability checks, and directions. |
| [receptionist](receptionist/) | ReceptionistAgent prefab. Greets callers, collects information, and transfers to the appropriate department (sales, support, billing). |
| [faq_bot](faq_bot/) | FAQBotAgent prefab. Answers frequently asked questions from a provided FAQ database with search and category matching. |

## Skill Integration Examples

| Example | Description |
|---------|-------------|
| [joke_agent](joke_agent/) | Joke skill demo using the built-in skills system with API Ninjas. Requires `API_NINJAS_KEY` environment variable. |
| [joke_skill](joke_skill/) | Joke skill via the modular skills system with DataMap for serverless execution. |
| [web_search](web_search/) | Web search skill using Google Custom Search API. Requires `GOOGLE_SEARCH_API_KEY` and `GOOGLE_SEARCH_ENGINE_ID`. |
| [web_search_multi_instance](web_search_multi_instance/) | Multiple web search instances (general, news, quick) plus Wikipedia. |
| [wikipedia](wikipedia/) | Wikipedia search skill for factual information retrieval from Wikipedia articles. |
| [datasphere](datasphere/) | Datasphere skill integration for document search through SignalWire Datasphere. Requires SignalWire credentials and `DATASPHERE_DOCUMENT_ID`. |
| [datasphere_multi_instance](datasphere_multi_instance/) | Multiple DataSphere instances with custom tool names for separate knowledge bases. |
| [datasphere_serverless_env](datasphere_serverless_env/) | DataSphere serverless skill configured from environment variables. |
| [datasphere_webhook_env](datasphere_webhook_env/) | Webhook-based DataSphere skill configured from environment variables. |
| [mcp_gateway](mcp_gateway/) | MCP gateway skill integration. Bridges MCP (Model Context Protocol) server tools as SWAIG functions. Requires a running MCP gateway. |

## SWML Service Examples

| Example | Description |
|---------|-------------|
| [swml_service](swml_service/) | Basic SWMLService (non-AI SWML, IVR-style). Builds and serves SWML documents with answer, play, prompt, switch, connect, record, and hangup verbs. |
| [dynamic_swml_service](dynamic_swml_service/) | SWML service with dynamic routing. Generates different SWML documents based on incoming request data (caller type, VIP status, department). |
| [swml_service_routing](swml_service_routing/) | SWML service with routing callbacks. Multiple paths (`/main`, `/customer`, `/product`) served from a single SWMLService instance. |

## Deployment Examples

| Example | Description |
|---------|-------------|
| [kubernetes](kubernetes/) | Kubernetes-ready agent with health/readiness probes, environment variable configuration, and production deployment patterns. |
| [lambda](lambda/) | Serverless Lambda handler pattern. Agent created at package level with AsRouter() for wrapping with an API Gateway adapter. |

## Platform Integration Examples

| Example | Description |
|---------|-------------|
| [relay_demo](relay_demo/) | RELAY WebSocket call control. Creating a RELAY client, handling inbound calls, playing TTS audio, and hanging up. Requires SignalWire credentials. |
| [rest_demo](rest_demo/) | REST API usage with RestClient. Creating a client, listing phone numbers, and other namespace usage patterns. Requires SignalWire credentials. |
| [livewire_agent](livewire_agent/) | LiveKit-style agent running on SignalWire's platform using familiar LiveKit API symbols. |

## Running Examples

Most examples can be run directly:

```bash
go run ./examples/simple_agent/
```

Examples with `//go:build ignore` tags (most new examples) should be run by specifying the file:

```bash
go run ./examples/concierge/main.go
```

Examples that require environment variables:

```bash
# SignalWire credentials (relay_demo, rest_demo, datasphere)
export SIGNALWIRE_PROJECT_ID=your-project-id
export SIGNALWIRE_API_TOKEN=your-api-token
export SIGNALWIRE_SPACE_URL=your-space.signalwire.com

# Skill-specific credentials
export API_NINJAS_KEY=your-api-key              # joke_agent
export GOOGLE_SEARCH_API_KEY=your-api-key       # web_search
export GOOGLE_SEARCH_ENGINE_ID=your-engine-id   # web_search
export DATASPHERE_DOCUMENT_ID=your-doc-id       # datasphere
export MCP_GATEWAY_URL=http://localhost:8080     # mcp_gateway
```
