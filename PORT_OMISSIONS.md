# PORT_OMISSIONS.md
#
# Every symbol listed here is a public Python-reference API member that the
# Go port deliberately does not implement.  The format is one
# `<fully.qualified.symbol>: <rationale>` per line, as expected by
# porting-sdk/scripts/diff_port_surface.py.  Section headers (lines that
# begin with `#`) are ignored by the parser.
#
# Rationale conventions:
#   * "not_yet_implemented: <reason>" = tracked gap, future PR will add it.
#   * anything else = deliberate omission (subsystem skipped, Python-only
#     implementation detail, or port-specific architectural difference).

# --- Search subsystem ---
signalwire.search.document_processor.DocumentProcessor: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.document_processor.DocumentProcessor.__init__: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.document_processor.DocumentProcessor.create_chunks: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.index_builder.IndexBuilder: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.index_builder.IndexBuilder.__init__: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.index_builder.IndexBuilder.build_index: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.index_builder.IndexBuilder.build_index_from_sources: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.index_builder.IndexBuilder.validate_index: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.migration.SearchIndexMigrator: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.migration.SearchIndexMigrator.__init__: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.migration.SearchIndexMigrator.get_index_info: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.migration.SearchIndexMigrator.migrate_pgvector_to_sqlite: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.migration.SearchIndexMigrator.migrate_sqlite_to_pgvector: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.models.resolve_model_alias: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.pgvector_backend.PgVectorBackend: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.pgvector_backend.PgVectorBackend.__init__: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.pgvector_backend.PgVectorBackend.close: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.pgvector_backend.PgVectorBackend.create_schema: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.pgvector_backend.PgVectorBackend.delete_collection: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.pgvector_backend.PgVectorBackend.get_stats: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.pgvector_backend.PgVectorBackend.list_collections: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.pgvector_backend.PgVectorBackend.store_chunks: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.pgvector_backend.PgVectorSearchBackend: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.pgvector_backend.PgVectorSearchBackend.__init__: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.pgvector_backend.PgVectorSearchBackend.close: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.pgvector_backend.PgVectorSearchBackend.fetch_candidates: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.pgvector_backend.PgVectorSearchBackend.get_stats: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.pgvector_backend.PgVectorSearchBackend.search: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.query_processor.detect_language: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.query_processor.ensure_nltk_resources: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.query_processor.get_synonyms: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.query_processor.get_wordnet_pos: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.query_processor.load_spacy_model: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.query_processor.preprocess_document_content: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.query_processor.preprocess_query: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.query_processor.remove_duplicate_words: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.query_processor.set_global_model: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.query_processor.vectorize_query: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.search_engine.SearchEngine: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.search_engine.SearchEngine.__init__: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.search_engine.SearchEngine.get_stats: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.search_engine.SearchEngine.search: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.search_service.SearchService: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.search_service.SearchService.__init__: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.search_service.SearchService.search_direct: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.search_service.SearchService.start: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only
signalwire.search.search_service.SearchService.stop: search subsystem (vector store, indexing, migrations) is Python-only; Go port uses native_vector_search skill in network mode only

# --- CLI tooling ---
signalwire.cli.build_search.console_entry_point: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.build_search.main: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.build_search.migrate_command: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.build_search.remote_command: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.build_search.search_command: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.build_search.validate_command: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.agent_loader.discover_agents_in_file: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.agent_loader.discover_services_in_file: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.agent_loader.load_agent_from_file: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.agent_loader.load_service_from_file: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.argparse_helpers.CustomArgumentParser: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.argparse_helpers.CustomArgumentParser.__init__: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.argparse_helpers.CustomArgumentParser.error: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.argparse_helpers.CustomArgumentParser.parse_args: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.argparse_helpers.CustomArgumentParser.print_usage: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.argparse_helpers.parse_function_arguments: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.dynamic_config.apply_dynamic_config: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.service_loader.ServiceCapture: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.service_loader.ServiceCapture.__init__: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.service_loader.ServiceCapture.capture: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.service_loader.discover_agents_in_file: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.service_loader.load_agent_from_file: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.service_loader.load_and_simulate_service: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.core.service_loader.simulate_request_to_service: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.Colors: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.DokkuProjectGenerator: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.DokkuProjectGenerator.__init__: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.DokkuProjectGenerator.generate: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.cmd_config: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.cmd_deploy: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.cmd_init: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.cmd_logs: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.cmd_scale: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.generate_password: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.main: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.print_error: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.print_header: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.print_step: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.print_success: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.print_warning: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.prompt: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.dokku.prompt_yes_no: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.execution.datamap_exec.execute_datamap_function: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.execution.datamap_exec.simple_template_expand: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.execution.webhook_exec.execute_external_webhook_function: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.Colors: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.ProjectGenerator: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.ProjectGenerator.__init__: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.ProjectGenerator.generate: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.generate_password: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.get_agent_template: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.get_app_template: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.get_env_credentials: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.get_readme_template: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.get_test_template: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.get_web_index_template: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.main: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.mask_token: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.print_error: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.print_step: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.print_success: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.print_warning: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.prompt: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.prompt_multiselect: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.prompt_select: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.prompt_yes_no: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.run_interactive: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.init_project.run_quick: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.output.output_formatter.display_agent_tools: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.output.output_formatter.format_result: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.output.swml_dump.handle_dump_swml: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.output.swml_dump.setup_output_suppression: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.data_generation.adapt_for_call_type: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.data_generation.generate_comprehensive_post_data: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.data_generation.generate_fake_node_id: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.data_generation.generate_fake_sip_from: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.data_generation.generate_fake_sip_to: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.data_generation.generate_fake_swml_post_data: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.data_generation.generate_fake_uuid: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.data_generation.generate_minimal_post_data: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.data_overrides.apply_convenience_mappings: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.data_overrides.apply_overrides: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.data_overrides.parse_value: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.data_overrides.set_nested_value: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockHeaders: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockHeaders.__contains__: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockHeaders.__getitem__: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockHeaders.__init__: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockHeaders.get: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockHeaders.items: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockHeaders.keys: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockHeaders.values: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockQueryParams: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockQueryParams.__contains__: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockQueryParams.__getitem__: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockQueryParams.__init__: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockQueryParams.get: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockQueryParams.items: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockQueryParams.keys: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockQueryParams.values: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockRequest: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockRequest.__init__: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockRequest.body: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockRequest.client: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockRequest.json: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockURL: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockURL.__init__: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.MockURL.__str__: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.ServerlessSimulator: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.ServerlessSimulator.__init__: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.ServerlessSimulator.activate: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.ServerlessSimulator.add_override: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.ServerlessSimulator.deactivate: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.ServerlessSimulator.get_current_env: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.create_mock_request: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.simulation.mock_env.load_env_file: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.swaig_test_wrapper.main: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.test_swaig.console_entry_point: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.test_swaig.main: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.test_swaig.print_help_examples: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.test_swaig.print_help_platforms: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.types.AgentInfo: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.types.CallData: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.types.DataMapConfig: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.types.FunctionInfo: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.types.PostData: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead
signalwire.cli.types.VarsData: Python-only CLI tooling; Go port ships cmd/swaig-test and cmd/enumerate-surface instead

# --- MCP gateway service ---
signalwire.mcp_gateway.gateway_service.MCPGateway: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.gateway_service.MCPGateway.__init__: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.gateway_service.MCPGateway.run: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.gateway_service.MCPGateway.shutdown: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.gateway_service.main: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPClient: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPClient.__init__: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPClient.call_method: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPClient.call_tool: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPClient.get_tools: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPClient.start: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPClient.stop: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPManager: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPManager.__init__: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPManager.create_client: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPManager.get_service: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPManager.get_service_tools: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPManager.list_services: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPManager.shutdown: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPManager.validate_services: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPService: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPService.__hash__: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.mcp_manager.MCPService.__post_init__: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.session_manager.Session: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.session_manager.Session.is_alive: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.session_manager.Session.is_expired: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.session_manager.Session.touch: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.session_manager.SessionManager: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.session_manager.SessionManager.__init__: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.session_manager.SessionManager.close_session: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.session_manager.SessionManager.create_session: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.session_manager.SessionManager.get_service_session_count: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.session_manager.SessionManager.get_session: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.session_manager.SessionManager.list_sessions: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer
signalwire.mcp_gateway.session_manager.SessionManager.shutdown: MCP gateway service daemon is Python-only; Go port exposes MCP via AgentBase.AddMcpServer / EnableMcpServer

# --- POM builder ---
signalwire.core.pom_builder.PomBuilder: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.core.pom_builder.PomBuilder.__init__: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.core.pom_builder.PomBuilder.add_section: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.core.pom_builder.PomBuilder.add_subsection: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.core.pom_builder.PomBuilder.add_to_section: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.core.pom_builder.PomBuilder.from_sections: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.core.pom_builder.PomBuilder.get_section: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.core.pom_builder.PomBuilder.has_section: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.core.pom_builder.PomBuilder.render_markdown: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.core.pom_builder.PomBuilder.render_xml: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.core.pom_builder.PomBuilder.to_dict: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.core.pom_builder.PomBuilder.to_json: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.PromptObjectModel: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.PromptObjectModel.__init__: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.PromptObjectModel.add_pom_as_subsection: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.PromptObjectModel.add_section: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.PromptObjectModel.find_section: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.PromptObjectModel.from_json: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.PromptObjectModel.from_yaml: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.PromptObjectModel.render_markdown: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.PromptObjectModel.render_xml: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.PromptObjectModel.to_dict: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.PromptObjectModel.to_json: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.PromptObjectModel.to_yaml: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.Section: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.Section.__init__: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.Section.add_body: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.Section.add_bullets: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.Section.add_subsection: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.Section.render_markdown: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.Section.render_xml: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom.Section.to_dict: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom_tool.detect_file_format: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom_tool.load_pom: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom_tool.main: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection
signalwire.pom.pom_tool.render_pom: POM (Prompt Object Model) builder class is Python-only; Go AgentBase accepts POM sections as maps via SetPromptPom / PromptAddSection

# --- Utils / web / auth helpers ---
signalwire.core.auth_handler.AuthHandler: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.auth_handler.AuthHandler.__init__: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.auth_handler.AuthHandler.flask_decorator: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.auth_handler.AuthHandler.get_auth_info: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.auth_handler.AuthHandler.get_fastapi_dependency: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.auth_handler.AuthHandler.verify_api_key: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.auth_handler.AuthHandler.verify_basic_auth: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.auth_handler.AuthHandler.verify_bearer_token: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.config_loader.ConfigLoader: Python config_loader drives the CLI tooling; Go port reads env vars directly in agent/server constructors
signalwire.core.config_loader.ConfigLoader.__init__: Python config_loader drives the CLI tooling; Go port reads env vars directly in agent/server constructors
signalwire.core.config_loader.ConfigLoader.find_config_file: Python config_loader drives the CLI tooling; Go port reads env vars directly in agent/server constructors
signalwire.core.config_loader.ConfigLoader.get: Python config_loader drives the CLI tooling; Go port reads env vars directly in agent/server constructors
signalwire.core.config_loader.ConfigLoader.get_config: Python config_loader drives the CLI tooling; Go port reads env vars directly in agent/server constructors
signalwire.core.config_loader.ConfigLoader.get_config_file: Python config_loader drives the CLI tooling; Go port reads env vars directly in agent/server constructors
signalwire.core.config_loader.ConfigLoader.get_section: Python config_loader drives the CLI tooling; Go port reads env vars directly in agent/server constructors
signalwire.core.config_loader.ConfigLoader.has_config: Python config_loader drives the CLI tooling; Go port reads env vars directly in agent/server constructors
signalwire.core.config_loader.ConfigLoader.merge_with_env: Python config_loader drives the CLI tooling; Go port reads env vars directly in agent/server constructors
signalwire.core.config_loader.ConfigLoader.substitute_vars: Python config_loader drives the CLI tooling; Go port reads env vars directly in agent/server constructors
signalwire.core.logging_config.configure_logging: Python logging_config wraps the Python logging lib; Go port uses pkg/logging with equivalent behaviour
signalwire.core.logging_config.get_execution_mode: Python logging_config wraps the Python logging lib; Go port uses pkg/logging with equivalent behaviour
signalwire.core.logging_config.get_logger: Python logging_config wraps the Python logging lib; Go port uses pkg/logging with equivalent behaviour
signalwire.core.logging_config.reset_logging_configuration: Python logging_config wraps the Python logging lib; Go port uses pkg/logging with equivalent behaviour
signalwire.core.logging_config.strip_control_chars: Python logging_config wraps the Python logging lib; Go port uses pkg/logging with equivalent behaviour
signalwire.core.security.session_manager.SessionManager.activate_session: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security.session_manager.SessionManager.create_session: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security.session_manager.SessionManager.debug_token: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security.session_manager.SessionManager.end_session: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security.session_manager.SessionManager.generate_token: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security.session_manager.SessionManager.get_session_metadata: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security.session_manager.SessionManager.set_session_metadata: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security.session_manager.SessionManager.validate_token: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security_config.SecurityConfig: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security_config.SecurityConfig.__init__: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security_config.SecurityConfig.get_basic_auth: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security_config.SecurityConfig.get_cors_config: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security_config.SecurityConfig.get_security_headers: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security_config.SecurityConfig.get_ssl_context_kwargs: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security_config.SecurityConfig.get_url_scheme: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security_config.SecurityConfig.load_from_env: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security_config.SecurityConfig.log_config: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security_config.SecurityConfig.should_allow_host: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.core.security_config.SecurityConfig.validate_ssl_config: Python security/auth helpers are embedded into Go AgentBase withAuth middleware + security.SessionManager; standalone classes not exposed
signalwire.utils.is_serverless_mode: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.utils.schema_utils.SchemaUtils: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.utils.schema_utils.SchemaUtils.__init__: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.utils.schema_utils.SchemaUtils.full_validation_available: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.utils.schema_utils.SchemaUtils.generate_method_body: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.utils.schema_utils.SchemaUtils.generate_method_signature: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.utils.schema_utils.SchemaUtils.get_all_verb_names: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.utils.schema_utils.SchemaUtils.get_verb_parameters: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.utils.schema_utils.SchemaUtils.get_verb_properties: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.utils.schema_utils.SchemaUtils.get_verb_required_properties: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.utils.schema_utils.SchemaUtils.load_schema: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.utils.schema_utils.SchemaUtils.validate_document: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.utils.schema_utils.SchemaUtils.validate_verb: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.utils.schema_utils.SchemaValidationError: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.utils.schema_utils.SchemaValidationError.__init__: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.utils.url_validator.validate_url: Python-specific utility helpers (schema_utils, url_validator); Go port uses net/url + encoding/json equivalents inline
signalwire.web.web_service.WebService: Python web_service.py wraps FastAPI; Go port uses net/http directly with no public wrapper to mirror
signalwire.web.web_service.WebService.__init__: Python web_service.py wraps FastAPI; Go port uses net/http directly with no public wrapper to mirror
signalwire.web.web_service.WebService.add_directory: Python web_service.py wraps FastAPI; Go port uses net/http directly with no public wrapper to mirror
signalwire.web.web_service.WebService.remove_directory: Python web_service.py wraps FastAPI; Go port uses net/http directly with no public wrapper to mirror
signalwire.web.web_service.WebService.start: Python web_service.py wraps FastAPI; Go port uses net/http directly with no public wrapper to mirror
signalwire.web.web_service.WebService.stop: Python web_service.py wraps FastAPI; Go port uses net/http directly with no public wrapper to mirror

# --- Bedrock prefab agent ---
signalwire.agents.bedrock.BedrockAgent: not_yet_implemented: Bedrock prefab agent planned but not shipped in Go port
signalwire.agents.bedrock.BedrockAgent.__init__: not_yet_implemented: Bedrock prefab agent planned but not shipped in Go port
signalwire.agents.bedrock.BedrockAgent.__repr__: not_yet_implemented: Bedrock prefab agent planned but not shipped in Go port
signalwire.agents.bedrock.BedrockAgent.set_inference_params: not_yet_implemented: Bedrock prefab agent planned but not shipped in Go port
signalwire.agents.bedrock.BedrockAgent.set_llm_model: not_yet_implemented: Bedrock prefab agent planned but not shipped in Go port
signalwire.agents.bedrock.BedrockAgent.set_llm_temperature: not_yet_implemented: Bedrock prefab agent planned but not shipped in Go port
signalwire.agents.bedrock.BedrockAgent.set_post_prompt_llm_params: not_yet_implemented: Bedrock prefab agent planned but not shipped in Go port
signalwire.agents.bedrock.BedrockAgent.set_prompt_llm_params: not_yet_implemented: Bedrock prefab agent planned but not shipped in Go port
signalwire.agents.bedrock.BedrockAgent.set_voice: not_yet_implemented: Bedrock prefab agent planned but not shipped in Go port

# --- Individual skill modules (per-skill classes/handlers) ---
signalwire.skills.api_ninjas_trivia.skill.ApiNinjasTriviaSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.api_ninjas_trivia.skill.ApiNinjasTriviaSkill.__init__: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.api_ninjas_trivia.skill.ApiNinjasTriviaSkill.get_instance_key: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.api_ninjas_trivia.skill.ApiNinjasTriviaSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.api_ninjas_trivia.skill.ApiNinjasTriviaSkill.get_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.api_ninjas_trivia.skill.ApiNinjasTriviaSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.api_ninjas_trivia.skill.ApiNinjasTriviaSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.claude_skills.skill.ClaudeSkillsSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.claude_skills.skill.ClaudeSkillsSkill.get_hints: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.claude_skills.skill.ClaudeSkillsSkill.get_instance_key: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.claude_skills.skill.ClaudeSkillsSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.claude_skills.skill.ClaudeSkillsSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.claude_skills.skill.ClaudeSkillsSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere.skill.DataSphereSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere.skill.DataSphereSkill.cleanup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere.skill.DataSphereSkill.get_global_data: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere.skill.DataSphereSkill.get_hints: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere.skill.DataSphereSkill.get_instance_key: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere.skill.DataSphereSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere.skill.DataSphereSkill.get_prompt_sections: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere.skill.DataSphereSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere.skill.DataSphereSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere_serverless.skill.DataSphereServerlessSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere_serverless.skill.DataSphereServerlessSkill.get_global_data: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere_serverless.skill.DataSphereServerlessSkill.get_hints: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere_serverless.skill.DataSphereServerlessSkill.get_instance_key: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere_serverless.skill.DataSphereServerlessSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere_serverless.skill.DataSphereServerlessSkill.get_prompt_sections: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere_serverless.skill.DataSphereServerlessSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datasphere_serverless.skill.DataSphereServerlessSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datetime.skill.DateTimeSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datetime.skill.DateTimeSkill.get_hints: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datetime.skill.DateTimeSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datetime.skill.DateTimeSkill.get_prompt_sections: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datetime.skill.DateTimeSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.datetime.skill.DateTimeSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.google_maps.skill.GoogleMapsClient: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.google_maps.skill.GoogleMapsClient.__init__: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.google_maps.skill.GoogleMapsClient.compute_route: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.google_maps.skill.GoogleMapsClient.validate_address: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.google_maps.skill.GoogleMapsSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.google_maps.skill.GoogleMapsSkill.get_hints: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.google_maps.skill.GoogleMapsSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.google_maps.skill.GoogleMapsSkill.get_prompt_sections: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.google_maps.skill.GoogleMapsSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.google_maps.skill.GoogleMapsSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.info_gatherer.skill.InfoGathererSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.info_gatherer.skill.InfoGathererSkill.get_global_data: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.info_gatherer.skill.InfoGathererSkill.get_instance_key: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.info_gatherer.skill.InfoGathererSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.info_gatherer.skill.InfoGathererSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.info_gatherer.skill.InfoGathererSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.joke.skill.JokeSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.joke.skill.JokeSkill.get_global_data: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.joke.skill.JokeSkill.get_hints: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.joke.skill.JokeSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.joke.skill.JokeSkill.get_prompt_sections: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.joke.skill.JokeSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.joke.skill.JokeSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.math.skill.MathSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.math.skill.MathSkill.get_hints: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.math.skill.MathSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.math.skill.MathSkill.get_prompt_sections: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.math.skill.MathSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.math.skill.MathSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.mcp_gateway.skill.MCPGatewaySkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.mcp_gateway.skill.MCPGatewaySkill.get_global_data: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.mcp_gateway.skill.MCPGatewaySkill.get_hints: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.mcp_gateway.skill.MCPGatewaySkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.mcp_gateway.skill.MCPGatewaySkill.get_prompt_sections: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.mcp_gateway.skill.MCPGatewaySkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.mcp_gateway.skill.MCPGatewaySkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.native_vector_search.skill.NativeVectorSearchSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.native_vector_search.skill.NativeVectorSearchSkill.cleanup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.native_vector_search.skill.NativeVectorSearchSkill.get_global_data: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.native_vector_search.skill.NativeVectorSearchSkill.get_hints: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.native_vector_search.skill.NativeVectorSearchSkill.get_instance_key: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.native_vector_search.skill.NativeVectorSearchSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.native_vector_search.skill.NativeVectorSearchSkill.get_prompt_sections: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.native_vector_search.skill.NativeVectorSearchSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.native_vector_search.skill.NativeVectorSearchSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.play_background_file.skill.PlayBackgroundFileSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.play_background_file.skill.PlayBackgroundFileSkill.__init__: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.play_background_file.skill.PlayBackgroundFileSkill.get_instance_key: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.play_background_file.skill.PlayBackgroundFileSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.play_background_file.skill.PlayBackgroundFileSkill.get_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.play_background_file.skill.PlayBackgroundFileSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.play_background_file.skill.PlayBackgroundFileSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.spider.skill.SpiderSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.spider.skill.SpiderSkill.__init__: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.spider.skill.SpiderSkill.cleanup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.spider.skill.SpiderSkill.get_hints: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.spider.skill.SpiderSkill.get_instance_key: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.spider.skill.SpiderSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.spider.skill.SpiderSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.spider.skill.SpiderSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.swml_transfer.skill.SWMLTransferSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.swml_transfer.skill.SWMLTransferSkill.get_hints: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.swml_transfer.skill.SWMLTransferSkill.get_instance_key: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.swml_transfer.skill.SWMLTransferSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.swml_transfer.skill.SWMLTransferSkill.get_prompt_sections: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.swml_transfer.skill.SWMLTransferSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.swml_transfer.skill.SWMLTransferSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.weather_api.skill.WeatherApiSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.weather_api.skill.WeatherApiSkill.__init__: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.weather_api.skill.WeatherApiSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.weather_api.skill.WeatherApiSkill.get_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.weather_api.skill.WeatherApiSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.weather_api.skill.WeatherApiSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.GoogleSearchScraper: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.GoogleSearchScraper.__init__: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.GoogleSearchScraper.extract_html_content: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.GoogleSearchScraper.extract_reddit_content: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.GoogleSearchScraper.extract_text_from_url: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.GoogleSearchScraper.is_reddit_url: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.GoogleSearchScraper.search_and_scrape: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.GoogleSearchScraper.search_and_scrape_best: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.GoogleSearchScraper.search_google: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.WebSearchSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.WebSearchSkill.get_global_data: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.WebSearchSkill.get_hints: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.WebSearchSkill.get_instance_key: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.WebSearchSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.WebSearchSkill.get_prompt_sections: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.WebSearchSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill.WebSearchSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_improved.GoogleSearchScraper: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_improved.GoogleSearchScraper.__init__: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_improved.GoogleSearchScraper.extract_text_from_url: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_improved.GoogleSearchScraper.search_and_scrape: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_improved.GoogleSearchScraper.search_and_scrape_best: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_improved.GoogleSearchScraper.search_google: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_improved.WebSearchSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_improved.WebSearchSkill.get_global_data: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_improved.WebSearchSkill.get_hints: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_improved.WebSearchSkill.get_instance_key: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_improved.WebSearchSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_improved.WebSearchSkill.get_prompt_sections: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_improved.WebSearchSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_improved.WebSearchSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_original.GoogleSearchScraper: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_original.GoogleSearchScraper.__init__: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_original.GoogleSearchScraper.extract_text_from_url: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_original.GoogleSearchScraper.search_and_scrape: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_original.GoogleSearchScraper.search_google: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_original.WebSearchSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_original.WebSearchSkill.get_global_data: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_original.WebSearchSkill.get_hints: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_original.WebSearchSkill.get_instance_key: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_original.WebSearchSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_original.WebSearchSkill.get_prompt_sections: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_original.WebSearchSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.web_search.skill_original.WebSearchSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.wikipedia_search.skill.WikipediaSearchSkill: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.wikipedia_search.skill.WikipediaSearchSkill.get_hints: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.wikipedia_search.skill.WikipediaSearchSkill.get_parameter_schema: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.wikipedia_search.skill.WikipediaSearchSkill.get_prompt_sections: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.wikipedia_search.skill.WikipediaSearchSkill.register_tools: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.wikipedia_search.skill.WikipediaSearchSkill.search_wiki: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API
signalwire.skills.wikipedia_search.skill.WikipediaSearchSkill.setup: Python skill module exposes internal handler helpers as public classes; Go port ships the same skills via pkg/skills/builtin/*Skill structs and the one-liner AddSkill API

# --- Skill registry plumbing ---
signalwire.skills.registry.SkillRegistry: Python SkillRegistry class maps onto Go free functions in pkg/skills/registry.go (RegisterSkill, ListSkills, GetSkillFactory, AddSkillDirectory)
signalwire.skills.registry.SkillRegistry.__init__: Python SkillRegistry class maps onto Go free functions in pkg/skills/registry.go (RegisterSkill, ListSkills, GetSkillFactory, AddSkillDirectory)
signalwire.skills.registry.SkillRegistry.add_skill_directory: Python SkillRegistry class maps onto Go free functions in pkg/skills/registry.go (RegisterSkill, ListSkills, GetSkillFactory, AddSkillDirectory)
signalwire.skills.registry.SkillRegistry.discover_skills: Python SkillRegistry class maps onto Go free functions in pkg/skills/registry.go (RegisterSkill, ListSkills, GetSkillFactory, AddSkillDirectory)
signalwire.skills.registry.SkillRegistry.get_all_skills_schema: Python SkillRegistry class maps onto Go free functions in pkg/skills/registry.go (RegisterSkill, ListSkills, GetSkillFactory, AddSkillDirectory)
signalwire.skills.registry.SkillRegistry.get_skill_class: Python SkillRegistry class maps onto Go free functions in pkg/skills/registry.go (RegisterSkill, ListSkills, GetSkillFactory, AddSkillDirectory)
signalwire.skills.registry.SkillRegistry.list_all_skill_sources: Python SkillRegistry class maps onto Go free functions in pkg/skills/registry.go (RegisterSkill, ListSkills, GetSkillFactory, AddSkillDirectory)
signalwire.skills.registry.SkillRegistry.list_skills: Python SkillRegistry class maps onto Go free functions in pkg/skills/registry.go (RegisterSkill, ListSkills, GetSkillFactory, AddSkillDirectory)
signalwire.skills.registry.SkillRegistry.register_skill: Python SkillRegistry class maps onto Go free functions in pkg/skills/registry.go (RegisterSkill, ListSkills, GetSkillFactory, AddSkillDirectory)

# --- Core mixins not split into Go ---
signalwire.core.mixins.auth_mixin.AuthMixin: Python AuthMixin helpers integrated into Go AgentBase withAuth middleware; no standalone class
signalwire.core.mixins.auth_mixin.AuthMixin.get_basic_auth_credentials: Python AuthMixin helpers integrated into Go AgentBase withAuth middleware; no standalone class
signalwire.core.mixins.auth_mixin.AuthMixin.validate_basic_auth: Python AuthMixin helpers integrated into Go AgentBase withAuth middleware; no standalone class
signalwire.core.mixins.mcp_server_mixin.MCPServerMixin: Python MCPServerMixin is a marker class with no public methods; Go inlines MCP into AgentBase
signalwire.core.mixins.prompt_mixin.PromptMixin.get_post_prompt: not_yet_implemented: AgentBase.GetPostPrompt accessor not yet exposed
signalwire.core.mixins.serverless_mixin.ServerlessMixin: Python ServerlessMixin wraps Lambda detection; Go port uses pkg/lambda Handler + agent.handleServerless
signalwire.core.mixins.serverless_mixin.ServerlessMixin.handle_serverless_request: Python ServerlessMixin wraps Lambda detection; Go port uses pkg/lambda Handler + agent.handleServerless
signalwire.core.mixins.state_mixin.StateMixin: Python StateMixin validates tool tokens; Go AgentBase delegates to security.SessionManager.ValidateToken directly
signalwire.core.mixins.state_mixin.StateMixin.validate_tool_token: Python StateMixin validates tool tokens; Go AgentBase delegates to security.SessionManager.ValidateToken directly
signalwire.core.mixins.tool_mixin.ToolMixin.tool: Python @tool decorator is Python-specific; Go port uses AgentBase.DefineTool(ToolDefinition{...}) struct-literal
signalwire.core.mixins.web_mixin.WebMixin.get_app: Python get_app returns the FastAPI app; Go equivalent is AgentBase.AsRouter which returns http.Handler
signalwire.core.mixins.web_mixin.WebMixin.on_request: Python on_request hook; Go port uses AgentBase.SetDynamicConfigCallback for the same role
signalwire.core.mixins.web_mixin.WebMixin.on_swml_request: Python on_swml_request hook; Go port uses AgentBase.SetDynamicConfigCallback for the same role
signalwire.core.mixins.web_mixin.WebMixin.register_routing_callback: Python WebMixin.register_routing_callback delegates to SWMLService; Go port exposes swml.Service.RegisterRoutingCallback directly
signalwire.core.mixins.web_mixin.WebMixin.setup_graceful_shutdown: Python setup_graceful_shutdown installs signal handlers; Go port relies on net/http.Server.Shutdown via context cancellation

# --- Core agent internal submodules ---
signalwire.agent_server.AgentServer.register_global_routing_callback: not_yet_implemented: global routing callback registration not yet wired in server.AgentServer
signalwire.core.agent.prompt.manager.PromptManager: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.prompt.manager.PromptManager.__init__: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.prompt.manager.PromptManager.define_contexts: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.prompt.manager.PromptManager.get_contexts: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.prompt.manager.PromptManager.get_post_prompt: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.prompt.manager.PromptManager.get_prompt: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.prompt.manager.PromptManager.get_raw_prompt: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.prompt.manager.PromptManager.prompt_add_section: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.prompt.manager.PromptManager.prompt_add_subsection: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.prompt.manager.PromptManager.prompt_add_to_section: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.prompt.manager.PromptManager.prompt_has_section: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.prompt.manager.PromptManager.set_post_prompt: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.prompt.manager.PromptManager.set_prompt_pom: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.prompt.manager.PromptManager.set_prompt_text: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.tools.decorator.ToolDecorator: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.tools.decorator.ToolDecorator.create_class_decorator: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.tools.decorator.ToolDecorator.create_instance_decorator: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.tools.registry.ToolRegistry: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.tools.registry.ToolRegistry.__init__: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.tools.registry.ToolRegistry.define_tool: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.tools.registry.ToolRegistry.get_all_functions: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.tools.registry.ToolRegistry.get_function: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.tools.registry.ToolRegistry.has_function: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.tools.registry.ToolRegistry.register_class_decorated_tools: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.tools.registry.ToolRegistry.register_swaig_function: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.tools.registry.ToolRegistry.remove_function: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.tools.type_inference.create_typed_handler_wrapper: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent.tools.type_inference.infer_schema: Python prompt/tool submodules are internal implementation detail; Go AgentBase consolidates the API onto one struct
signalwire.core.agent_base.AgentBase.auto_map_sip_usernames: Python auto_map_sip_usernames is covered by the autoMap boolean argument to AgentBase.EnableSipRouting in the Go port
signalwire.core.agent_base.AgentBase.get_full_url: not_yet_implemented: Go AgentBase.GetFullURL helper not yet exposed (swml.Service.GetFullURL serves equivalent role)
signalwire.core.skill_base.SkillBase.define_tool: Python skill tool registration uses a decorator; Go port uses BaseSkill.RegisterTools returning []ToolRegistration
signalwire.core.skill_base.SkillBase.get_skill_data: Python get_skill_data exposes the skill's parameter map; Go port uses BaseSkill.GetParam / GetParamString / GetParamInt etc.
signalwire.core.skill_base.SkillBase.register_tools: Python register_tools is an override hook; Go port uses the SkillBase interface RegisterTools method (capitalised, emitted under the skill struct)
signalwire.core.skill_base.SkillBase.setup: Python setup is an override hook; Go port uses SkillBase interface Setup method — emitted via builtin skill structs, not BaseSkill alone
signalwire.core.skill_base.SkillBase.update_skill_data: Python update_skill_data mutates skill params at runtime; Go port treats params as immutable after construction
signalwire.core.skill_base.SkillBase.validate_env_vars: not_yet_implemented: explicit env-var validation hook not exposed on BaseSkill (skills call os.Getenv in Setup)
signalwire.core.skill_base.SkillBase.validate_packages: Python validate_packages checks pip dependencies at runtime; Go has no equivalent (dependencies checked at build time)

# --- Core SWML / SWAIG / function_result internals ---
signalwire.core.function_result.FunctionResult.create_payment_action: not_yet_implemented: payment builder helpers (create_payment_prompt/action/parameter) planned but not yet shipped
signalwire.core.function_result.FunctionResult.create_payment_parameter: not_yet_implemented: payment builder helpers (create_payment_prompt/action/parameter) planned but not yet shipped
signalwire.core.function_result.FunctionResult.create_payment_prompt: not_yet_implemented: payment builder helpers (create_payment_prompt/action/parameter) planned but not yet shipped
signalwire.core.swaig_function.SWAIGFunction: Python SWAIGFunction wraps a tool registration; Go port uses ToolDefinition + AgentBase.DefineTool directly
signalwire.core.swaig_function.SWAIGFunction.__call__: Python SWAIGFunction wraps a tool registration; Go port uses ToolDefinition + AgentBase.DefineTool directly
signalwire.core.swaig_function.SWAIGFunction.__init__: Python SWAIGFunction wraps a tool registration; Go port uses ToolDefinition + AgentBase.DefineTool directly
signalwire.core.swaig_function.SWAIGFunction.execute: Python SWAIGFunction wraps a tool registration; Go port uses ToolDefinition + AgentBase.DefineTool directly
signalwire.core.swaig_function.SWAIGFunction.to_swaig: Python SWAIGFunction wraps a tool registration; Go port uses ToolDefinition + AgentBase.DefineTool directly
signalwire.core.swaig_function.SWAIGFunction.validate_args: Python SWAIGFunction wraps a tool registration; Go port uses ToolDefinition + AgentBase.DefineTool directly
signalwire.core.swml_builder.SWMLBuilder: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_builder.SWMLBuilder.__getattr__: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_builder.SWMLBuilder.__init__: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_builder.SWMLBuilder.add_section: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_builder.SWMLBuilder.ai: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_builder.SWMLBuilder.answer: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_builder.SWMLBuilder.build: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_builder.SWMLBuilder.hangup: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_builder.SWMLBuilder.play: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_builder.SWMLBuilder.render: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_builder.SWMLBuilder.reset: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_builder.SWMLBuilder.say: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_handler.AIVerbHandler: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_handler.AIVerbHandler.build_config: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_handler.AIVerbHandler.get_verb_name: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_handler.AIVerbHandler.validate_config: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_handler.SWMLVerbHandler: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_handler.SWMLVerbHandler.build_config: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_handler.SWMLVerbHandler.get_verb_name: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_handler.SWMLVerbHandler.validate_config: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_handler.VerbHandlerRegistry: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_handler.VerbHandlerRegistry.__init__: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_handler.VerbHandlerRegistry.get_handler: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_handler.VerbHandlerRegistry.has_handler: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_handler.VerbHandlerRegistry.register_handler: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_renderer.SwmlRenderer: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_renderer.SwmlRenderer.render_function_response_swml: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_renderer.SwmlRenderer.render_swml: Python SWML build/render internals; Go pkg/swml consolidates via Service/Document/Schema types
signalwire.core.swml_service.SWMLService.__getattr__: Python __getattr__ dispatches verbs dynamically; Go port exposes verbs as explicit methods on swml.Service (Answer, Play, etc.)
signalwire.core.swml_service.SWMLService.add_section: not_yet_implemented: swml.Service.AddSection helper not yet exposed (use Document.AddSection directly)
signalwire.core.swml_service.SWMLService.as_router: not_yet_implemented: swml.Service.AsRouter not yet exposed; use Serve()
signalwire.core.swml_service.SWMLService.extract_sip_username: Python classmethod; Go port exposes as package-level free function swml.ExtractSIPUsername
signalwire.core.swml_service.SWMLService.full_validation_enabled: Python property; Go port enforces full schema validation unconditionally against swml/schema.json
signalwire.core.swml_service.SWMLService.manual_set_proxy_url: not_yet_implemented: swml.Service.ManualSetProxyUrl not yet exposed (AgentBase.ManualSetProxyUrl covers the common case)
signalwire.core.swml_service.SWMLService.register_verb_handler: not_yet_implemented: dynamic verb-handler registration not yet exposed on swml.Service
signalwire.core.swml_service.SWMLService.stop: not_yet_implemented: swml.Service.Stop graceful-shutdown hook not yet exposed

# --- Relay Call / Client / Message ---
signalwire.relay.call.AIAction.stop: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.Call.live_transcribe: not_yet_implemented: RELAY live transcribe/translate planned but not yet exposed on Call
signalwire.relay.call.Call.live_translate: not_yet_implemented: RELAY live transcribe/translate planned but not yet exposed on Call
signalwire.relay.call.Call.refer: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.Call.wait_for_ended: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.CollectAction.start_input_timers: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.CollectAction.stop: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.CollectAction.volume: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.DetectAction.stop: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.FaxAction.stop: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.PayAction.stop: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.PlayAction.stop: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.RecordAction.pause: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.RecordAction.resume: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.RecordAction.stop: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.StandaloneCollectAction.start_input_timers: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.StandaloneCollectAction.stop: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.StreamAction.stop: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.TapAction.stop: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.call.TranscribeAction.stop: not_yet_implemented: relay.Call method not yet ported
signalwire.relay.client.RelayClient.__aenter__: Python async context-manager protocol; Go port uses explicit Run()/Stop() pair
signalwire.relay.client.RelayClient.__aexit__: Python async context-manager protocol; Go port uses explicit Run()/Stop() pair
signalwire.relay.client.RelayClient.__del__: Python __del__ finalizer; Go GC handles resource cleanup — Stop() releases the WebSocket
signalwire.relay.client.RelayClient.connect: Python RelayClient.connect is subsumed by Run() in the Go port (single blocking entry point)
signalwire.relay.client.RelayClient.execute: Python RelayClient.execute is an internal JSON-RPC helper; Go keeps Client.execute() unexported
signalwire.relay.client.RelayClient.receive: Python receive/unreceive manage context subscriptions; Go uses Client constructor option WithContexts
signalwire.relay.client.RelayClient.relay_protocol: Python relay_protocol property exposes the internal JSON-RPC session; Go keeps this unexported
signalwire.relay.client.RelayClient.unreceive: Python receive/unreceive manage context subscriptions; Go uses Client constructor option WithContexts
signalwire.relay.client.RelayError: not_yet_implemented: typed RelayError struct planned; Go currently returns standard error values
signalwire.relay.client.RelayError.__init__: not_yet_implemented: typed RelayError struct planned; Go currently returns standard error values
signalwire.relay.event.parse_event: Python parse_event module-level function is exposed as relay.ParseEvent in Go
signalwire.relay.message.Message.__repr__: Python Message __init__ dunder is covered by Go relay internal factory (newMessage)
signalwire.relay.message.Message.result: Python Message __init__ dunder is covered by Go relay internal factory (newMessage)

# --- REST namespace omissions ---
signalwire.rest._base.CrudWithAddresses: Python mixin shared between fabric resources; Go port inlines list_addresses onto the individual fabric resource structs
signalwire.rest._base.CrudWithAddresses.list_addresses: Python mixin method; Go port emits list_addresses directly on ConferenceRoomsResource/SubscribersResource/CallFlowsResource/GenericResources
signalwire.rest.call_handler.PhoneCallHandler: Python PhoneCallHandler is a typing helper alias; Go port uses pkg/rest/namespaces/call_handler.go (string type)
signalwire.rest.namespaces.compat.CompatTokens.delete: not_yet_implemented: compat namespace item pending
signalwire.rest.namespaces.compat.CompatTokens.update: not_yet_implemented: compat namespace item pending
signalwire.rest.namespaces.fabric.CxmlApplicationsResource: not_yet_implemented: CxmlApplicationsResource not yet wired in FabricNamespace
signalwire.rest.namespaces.fabric.CxmlApplicationsResource.create: not_yet_implemented: CxmlApplicationsResource not yet wired in FabricNamespace
signalwire.rest.namespaces.fabric.CxmlWebhooksResource: deprecated legacy resource; Go port omits per phone-binding.md (use phone_numbers.SetCxmlWebhook)
signalwire.rest.namespaces.fabric.FabricResource: internal base class for fabric resources; Go port inlines CRUD onto concrete resource structs
signalwire.rest.namespaces.fabric.FabricResourcePUT: internal base class variant; Go port inlines update handling onto concrete resource structs
signalwire.rest.namespaces.fabric.SwmlWebhooksResource: deprecated legacy resource; Go port omits per phone-binding.md (use phone_numbers.SetSwmlWebhook)

# --- Prefab internal handlers ---
signalwire.prefabs.concierge.ConciergeAgent.check_availability: Python prefab exposes internal tool handlers (on_summary, check_*, etc.) as public methods; Go port registers equivalent SWAIG tools in the constructor so the runtime behaviour is identical but the methods are not re-exported on the struct
signalwire.prefabs.concierge.ConciergeAgent.get_directions: Python prefab exposes internal tool handlers (on_summary, check_*, etc.) as public methods; Go port registers equivalent SWAIG tools in the constructor so the runtime behaviour is identical but the methods are not re-exported on the struct
signalwire.prefabs.concierge.ConciergeAgent.on_summary: Python prefab exposes internal tool handlers (on_summary, check_*, etc.) as public methods; Go port registers equivalent SWAIG tools in the constructor so the runtime behaviour is identical but the methods are not re-exported on the struct
signalwire.prefabs.faq_bot.FAQBotAgent.on_summary: Python prefab exposes internal tool handlers (on_summary, check_*, etc.) as public methods; Go port registers equivalent SWAIG tools in the constructor so the runtime behaviour is identical but the methods are not re-exported on the struct
signalwire.prefabs.faq_bot.FAQBotAgent.search_faqs: Python prefab exposes internal tool handlers (on_summary, check_*, etc.) as public methods; Go port registers equivalent SWAIG tools in the constructor so the runtime behaviour is identical but the methods are not re-exported on the struct
signalwire.prefabs.info_gatherer.InfoGathererAgent.on_swml_request: Python prefab exposes internal tool handlers (on_summary, check_*, etc.) as public methods; Go port registers equivalent SWAIG tools in the constructor so the runtime behaviour is identical but the methods are not re-exported on the struct
signalwire.prefabs.info_gatherer.InfoGathererAgent.set_question_callback: Python prefab exposes internal tool handlers (on_summary, check_*, etc.) as public methods; Go port registers equivalent SWAIG tools in the constructor so the runtime behaviour is identical but the methods are not re-exported on the struct
signalwire.prefabs.info_gatherer.InfoGathererAgent.start_questions: Python prefab exposes internal tool handlers (on_summary, check_*, etc.) as public methods; Go port registers equivalent SWAIG tools in the constructor so the runtime behaviour is identical but the methods are not re-exported on the struct
signalwire.prefabs.info_gatherer.InfoGathererAgent.submit_answer: Python prefab exposes internal tool handlers (on_summary, check_*, etc.) as public methods; Go port registers equivalent SWAIG tools in the constructor so the runtime behaviour is identical but the methods are not re-exported on the struct
signalwire.prefabs.receptionist.ReceptionistAgent.on_summary: Python prefab exposes internal tool handlers (on_summary, check_*, etc.) as public methods; Go port registers equivalent SWAIG tools in the constructor so the runtime behaviour is identical but the methods are not re-exported on the struct
signalwire.prefabs.survey.SurveyAgent.log_response: Python prefab exposes internal tool handlers (on_summary, check_*, etc.) as public methods; Go port registers equivalent SWAIG tools in the constructor so the runtime behaviour is identical but the methods are not re-exported on the struct
signalwire.prefabs.survey.SurveyAgent.on_summary: Python prefab exposes internal tool handlers (on_summary, check_*, etc.) as public methods; Go port registers equivalent SWAIG tools in the constructor so the runtime behaviour is identical but the methods are not re-exported on the struct
signalwire.prefabs.survey.SurveyAgent.validate_response: Python prefab exposes internal tool handlers (on_summary, check_*, etc.) as public methods; Go port registers equivalent SWAIG tools in the constructor so the runtime behaviour is identical but the methods are not re-exported on the struct

# --- Livewire shim gaps ---
signalwire.livewire.Agent.llm_node: not_yet_implemented: pipeline node override points (llm_node/stt_node/tts_node) are LiveKit-specific; Go shim delegates to SWML AI verb
signalwire.livewire.Agent.on_enter: not_yet_implemented: lifecycle hook not yet wired in Go livewire.Agent
signalwire.livewire.Agent.on_exit: not_yet_implemented: lifecycle hook not yet wired in Go livewire.Agent
signalwire.livewire.Agent.on_user_turn_completed: not_yet_implemented: turn lifecycle hook not yet wired in Go livewire.Agent
signalwire.livewire.Agent.session: not_yet_implemented: Agent.session helper accessor not yet exposed (sessions constructed via NewAgentSession)
signalwire.livewire.Agent.stt_node: not_yet_implemented: pipeline node override points are LiveKit-specific; Go shim delegates to SWML AI verb
signalwire.livewire.Agent.tts_node: not_yet_implemented: pipeline node override points are LiveKit-specific; Go shim delegates to SWML AI verb
signalwire.livewire.Agent.update_instructions: not_yet_implemented: AgentSession.UpdateInstructions exists; equivalent on Agent not yet exposed
signalwire.livewire.Agent.update_tools: not_yet_implemented: AgentSession.UpdateTools not yet exposed
signalwire.livewire.AgentSession.history: not_yet_implemented: AgentSession.History accessor not yet exposed
signalwire.livewire.AgentSession.update_agent: not_yet_implemented: AgentSession.UpdateAgent not yet exposed
signalwire.livewire.AgentSession.userdata: not_yet_implemented: AgentSession.Userdata property not yet exposed
signalwire.livewire.ChatContext: not_yet_implemented: ChatContext not ported; livewire shim is minimal and omits chat-history typing
signalwire.livewire.ChatContext.__init__: not_yet_implemented: ChatContext not ported
signalwire.livewire.ChatContext.append: not_yet_implemented: ChatContext not ported
signalwire.livewire.InferenceLLM: Python inference wrappers; Go port uses WithLLM/WithSTT/WithTTS AgentSession options
signalwire.livewire.InferenceLLM.__init__: Python inference wrappers; Go port uses WithLLM option on AgentSession
signalwire.livewire.InferenceSTT: Python inference wrappers; Go port uses WithSTT option on AgentSession
signalwire.livewire.InferenceSTT.__init__: Python inference wrappers; Go port uses WithSTT option on AgentSession
signalwire.livewire.InferenceTTS: Python inference wrappers; Go port uses WithTTS option on AgentSession
signalwire.livewire.InferenceTTS.__init__: Python inference wrappers; Go port uses WithTTS option on AgentSession
signalwire.livewire.JobContext.wait_for_participant: not_yet_implemented: JobContext.WaitForParticipant not yet implemented
signalwire.livewire.RunContext.userdata: not_yet_implemented: RunContext.Userdata accessor not yet exposed
signalwire.livewire.ToolError: not_yet_implemented: ToolError sentinel type not ported (Go returns errors via standard error interface)
signalwire.livewire.function_tool: not_yet_implemented: function_tool module-level helper not exposed as a Go free function (use Agent.FunctionTool method)

# --- Misc not-yet-implemented items ---
signalwire.add_skill_directory: not_yet_implemented: top-level helper not yet exposed as a free function in Go port
signalwire.list_skills_with_params: not_yet_implemented: top-level helper not yet exposed as a free function in Go port
signalwire.run_agent: not_yet_implemented: top-level helper not yet exposed as a free function in Go port
signalwire.start_agent: not_yet_implemented: top-level helper not yet exposed as a free function in Go port

# --- Idiom: Python class accessors that Go folds into private fields or package-level helpers ---
signalwire.agent_server.AgentServer.app: Python exposes the underlying FastAPI ``app`` object; Go uses net/http with no equivalent app handle
signalwire.agent_server.AgentServer.logger: Python instance ``logger`` property; Go's AgentServer uses the package-level ``logging`` helper rather than a per-instance accessor
signalwire.core.agent_base.AgentBase.skill_manager: Python exposes ``self.skill_manager`` for direct access; Go folds the SkillManager into a private ``skillManager`` field and surfaces user-facing methods (AddSkill, RemoveSkill, ListSkills, HasSkill) directly on AgentBase
signalwire.core.skill_manager.SkillManager.logger: Python instance ``logger`` property; Go's SkillManager uses the package-level ``logging`` helper and has no per-instance logger accessor
signalwire.core.swml_service.SWMLService.security: Python exposes a ``security`` property returning a SecurityConfig; Go folds auth state into private fields on Service (basicAuthUser, bearerToken, apiKey, ...) configured via WithSecurityConfig/WithBasicAuth/WithBearerToken/WithAPIKey options
signalwire.core.swml_service.SWMLService.verb_registry: Python uses a separate VerbRegistry helper class; Go uses a private ``verbHandlers`` map on Service and exposes RegisterVerbHandler directly
signalwire.rest._base.BaseResource.__init__: Go's namespaces.Resource is a tiny base struct initialized inline by namespace constructors (struct-literal); no public NewResource factory mirrors Python's BaseResource(http, base_path)

