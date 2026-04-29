# PORT_EXAMPLE_OMISSIONS.md

Examples in the Python reference SDK that the Go port does not ship under the
exact same normalized name. Each entry below is a Python example whose
canonical functionality is covered by a differently-named Go example, OR is
intentionally not ported (with rationale).

Format (matches `audit_example_parity.py` parser):

    - <python_example_stem>: <one-sentence rationale>

The `audit_example_parity.py` audit normalizes names by lowercasing and
stripping non-alphanumerics, so `advanced_datamap_demo.py` (Python) and
`advanced_datamap` (Go directory) normalize differently — even though the
Go example covers the exact same behaviour. Renaming a stable Go example
to track Python's stem would break existing references (commit history,
issue links, doc cross-refs); so we record the equivalence here instead.

## Naming-only divergences (Go example ships under shorter name)

- advanced_datamap_demo: Go example lives at `examples/advanced_datamap/main.go`
- auto_vivified_example: Go example lives at `examples/swmlservice_swaig_standalone/main.go` (auto-vivified verb registration is exercised inside the SWML standalone example)
- basic_swml_service: Go example lives at `examples/swml_service/main.go`
- call_flow_and_actions_demo: Go example lives at `examples/call_flow/main.go`
- comprehensive_dynamic_agent: Go example lives at `examples/comprehensive_dynamic/main.go`
- concierge_agent_example: Go example lives at `examples/concierge/main.go`
- custom_path_agent: Go example lives at `examples/custom_path/main.go`
- datasphere_multi_instance_demo: Go example lives at `examples/datasphere_multi_instance/main.go`
- datasphere_webhook_env_demo: Go example lives at `examples/datasphere_webhook_env/main.go`
- declarative_agent: Go example lives at `examples/declarative/main.go`
- dynamic_info_gatherer_example: Go example lives at `examples/dynamic_info_gatherer/main.go`
- faq_bot_agent: Go example lives at `examples/faq_bot/main.go`
- gather_info_demo: Go example lives at `examples/gather_info/main.go`
- info_gatherer_example: Go example lives at `examples/prefab_info_gatherer/main.go`
- joke_skill_demo: Go example lives at `examples/joke_skill/main.go`
- kubernetes_ready_agent: Go example lives at `examples/kubernetes/main.go`
- lambda_agent: Go example lives at `examples/lambda/main.go`
- llm_params_demo: Go example lives at `examples/llm_params/main.go`
- mcp_gateway_demo: Go example lives at `examples/mcp_gateway/main.go`
- multi_endpoint_agent: Go example lives at `examples/multi_endpoint/main.go`
- receptionist_agent_example: Go example lives at `examples/receptionist/main.go`
- record_call_example: Go example lives at `examples/record_call/main.go`
- relay_answer_and_welcome: Go example lives at `relay/examples/relay_answer_and_welcome.go` (the relay/ top-level dir is the canonical location for relay examples; per-port harness contract documented in PORTING_GUIDE.md)
- room_and_sip_example: Go example lives at `examples/room_and_sip/main.go`
- session_and_state_demo: Go example lives at `examples/session_state/main.go`
- simple_static_agent: Go example lives at `examples/simple_static/main.go`
- survey_agent_example: Go example lives at `examples/prefab_survey/main.go`
- swaig_features_agent: Go example lives at `examples/swaig_features/main.go`
- swml_service_example: Go example lives at `examples/swml_service/main.go` (same file as basic_swml_service — Python ships two examples that demonstrate slightly different patterns of the same SWMLService surface; Go consolidates)
- swml_service_routing_example: Go example lives at `examples/swml_service_routing/main.go`
- tap_example: Go example lives at `examples/tap/main.go`
- web_search_agent: Go example lives at `examples/web_search/main.go`
- web_search_multi_instance_demo: Go example lives at `examples/web_search_multi_instance/main.go`
- wikipedia_demo: Go example lives at `examples/wikipedia/main.go`

## Intentional non-port

- local_search_agent: Search functionality (vector / RAG search) is in Python's per-port skip list (`examples/search_*.py`) and only ships in Python. Go does not implement the search subsystem; this example would have nothing to demonstrate.
