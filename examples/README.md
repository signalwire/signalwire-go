# Examples

This directory contains runnable examples demonstrating the SignalWire AI Agents Go SDK.

## Agent Examples

| Example | Description |
|---------|-------------|
| [simple_agent](simple_agent/) | Basic AI agent with prompt and tools. Creating an AgentBase, setting prompt text, adding hints and language, defining SWAIG tools, setting global data, and running the agent. |
| [simple_dynamic_agent](simple_dynamic_agent/) | Per-request agent customization using a dynamic config callback. Inspects query parameters to adjust the prompt, global data, and tools based on caller tier. |
| [multi_agent_server](multi_agent_server/) | Hosts multiple AI agents on a single HTTP server using AgentServer. Each agent gets its own route with unique prompts and tools. |
| [call_flow](call_flow/) | Call flow verbs and SWAIG actions. Demonstrates pre-answer verbs (ringback), answer configuration, post-answer verbs, post-AI verbs, debug events, and tools that return call control actions. |
| [contexts_demo](contexts_demo/) | Multi-step conversation workflows using contexts and steps. Creating multiple contexts with sequential steps, step criteria, navigation rules, and function restrictions. |
| [datamap_demo](datamap_demo/) | Server-side tools using DataMap that execute on SignalWire servers without requiring webhook endpoints. Both webhook-based API calls and expression-based pattern matching. |
| [session_state](session_state/) | Global data management and lifecycle callbacks. Setting initial global data, post-prompt for conversation summaries, OnSummary callback, and tools that read/write session state. |
| [skills_demo](skills_demo/) | Skills integration using the built-in skills registry. Listing available skills, instantiating via factory functions, loading through SkillManager, and registering tools with an agent. |

## Prefab Examples

| Example | Description |
|---------|-------------|
| [prefab_info_gatherer](prefab_info_gatherer/) | InfoGathererAgent pre-built pattern that collects answers to a series of questions sequentially with built-in tools and prompt sections. |
| [prefab_survey](prefab_survey/) | SurveyAgent conducts structured surveys with typed questions (rating, multiple choice, yes/no, open-ended) including response validation and summary generation. |

## Platform Integration Examples

| Example | Description |
|---------|-------------|
| [relay_demo](relay_demo/) | RELAY WebSocket call control. Creating a RELAY client, handling inbound calls, playing TTS audio, and hanging up. Requires SignalWire credentials. |
| [rest_demo](rest_demo/) | REST API usage with SignalWireClient. Creating a client, listing phone numbers, and other namespace usage patterns. Requires SignalWire credentials. |
| [livewire_agent](livewire_agent/) | LiveKit-style agent running on SignalWire's platform using familiar LiveKit API symbols. |

## Running Examples

Most examples can be run directly:

```bash
go run ./examples/simple_agent/
```

Examples that interact with SignalWire services (relay_demo, rest_demo) require environment variables:

```bash
export SIGNALWIRE_PROJECT_ID=your-project-id
export SIGNALWIRE_API_TOKEN=your-api-token
export SIGNALWIRE_SPACE_URL=your-space.signalwire.com
go run ./examples/relay_demo/
```
