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
# signalwire.pom.pom.PromptObjectModel and signalwire.pom.pom.Section are
# now implemented natively in Go as pom.PromptObjectModel and pom.Section
# (pkg/pom/pom.go).  Tests in pkg/pom/pom_test.go assert exact-string
# parity with the Python renderer; signalwire-python parity tests live in
# tests/unit/pom/test_pom_render_parity.py.
#
# pom_tool is a Python-only CLI that wraps the POM module — kept omitted
# because Go ships a library, not a CLI.
signalwire.pom.pom_tool.detect_file_format: pom_tool is a Python-only CLI wrapper around pom.pom; Go ships the library only
signalwire.pom.pom_tool.load_pom: pom_tool is a Python-only CLI wrapper around pom.pom; Go ships the library only
signalwire.pom.pom_tool.main: pom_tool is a Python-only CLI wrapper around pom.pom; Go ships the library only
signalwire.pom.pom_tool.render_pom: pom_tool is a Python-only CLI wrapper around pom.pom; Go ships the library only

# --- Utils / web / auth helpers ---
signalwire.core.auth_handler.AuthHandler: impossible: Python standalone auth-helper class; Go folds auth into AgentBase withAuth middleware + security.SessionManager — no standalone class
signalwire.core.auth_handler.AuthHandler.__init__: impossible: constructor of the standalone auth-helper class Go does not model
signalwire.core.auth_handler.AuthHandler.flask_decorator: impossible: Flask-specific decorator; no Go equivalent (Go uses net/http middleware)
signalwire.core.auth_handler.AuthHandler.get_auth_info: impossible: Python auth-helper accessor; Go folds auth state into middleware, no standalone class
signalwire.core.auth_handler.AuthHandler.get_fastapi_dependency: impossible: FastAPI-specific dependency factory; no Go equivalent (Go uses net/http middleware)
signalwire.core.auth_handler.AuthHandler.verify_api_key: impossible: Python auth-helper method; Go verifies API keys inside withAuth middleware, no standalone class
signalwire.core.auth_handler.AuthHandler.verify_basic_auth: impossible: Python auth-helper method; Go verifies basic auth inside withAuth middleware, no standalone class
signalwire.core.auth_handler.AuthHandler.verify_bearer_token: impossible: Python auth-helper method; Go verifies bearer tokens inside withAuth middleware, no standalone class
signalwire.core.logging_config.configure_logging: impossible: wraps the Python logging library; Go uses pkg/logging (structured) with equivalent behaviour — no logging-lib configuration surface
signalwire.core.logging_config.get_logger: impossible: returns a Python logging.Logger; Go uses pkg/logging.New — no stdlib-logger accessor
signalwire.core.logging_config.reset_logging_configuration: impossible: resets Python logging-library global state; Go pkg/logging has no equivalent global reset
signalwire.core.logging_config.strip_control_chars: impossible: Python logging-formatter helper; Go pkg/logging sanitises inline with no exported free function
signalwire.core.security_config.SecurityConfig.get_ssl_context_kwargs: impossible: returns Python ssl.SSLContext kwargs; Go configures TLS via crypto/tls.Config, no kwargs-dict equivalent

# --- Bedrock prefab agent ---


# --- Skill registry plumbing ---
signalwire.skills.registry.SkillRegistry.discover_skills: approved: 2026-07 user sign-off — Go registers skills at compile time via package-level RegisterSkill/ListSkills/GetSkillFactory; no runtime instance-method discovery
signalwire.skills.registry.SkillRegistry.get_all_skills_schema: approved: 2026-07 user sign-off — Go uses package-level skill registration; no instance schema-aggregation method
signalwire.skills.registry.SkillRegistry.get_skill_class: approved: 2026-07 user sign-off — Go uses package-level GetSkillFactory; no instance class lookup
signalwire.skills.registry.SkillRegistry.list_all_skill_sources: approved: 2026-07 user sign-off — Go registers skills at compile time; no runtime source enumeration
signalwire.skills.registry.SkillRegistry.list_skills: approved: 2026-07 user sign-off — Go uses the package-level ListSkills free function
signalwire.skills.registry.SkillRegistry.register_skill: approved: 2026-07 user sign-off — Go uses the package-level RegisterSkill free function

# --- Core mixins not split into Go ---
signalwire.core.mixins.mcp_server_mixin.MCPServerMixin: approved: 2026-07 user sign-off — MCP-server mixin is a Python marker class (no public methods); Go inlines MCP into AgentBase (AddMcpServer/EnableMcpServer)
signalwire.core.mixins.serverless_mixin.ServerlessMixin: approved: 2026-07 user sign-off — Python serverless mixin (Lambda detection + request handling); Go delegates serverless to platform adapters (pkg/lambda Handler), not an in-process AgentBase mixin
signalwire.core.mixins.serverless_mixin.ServerlessMixin.handle_serverless_request: impossible: Python couples serverless request handling into the mixin; Go delegates to platform adapters (pkg/lambda) — no in-process AgentBase equivalent
signalwire.core.mixins.tool_mixin.ToolMixin.tool: impossible: Python @tool decorator relies on the decorator protocol; Go uses AgentBase.DefineTool(ToolDefinition{...})
signalwire.core.mixins.web_mixin.WebMixin.get_app: impossible: returns the FastAPI app object; Go has no framework app handle (AsRouter returns http.Handler)

# --- Core agent internal submodules ---
signalwire.core.agent.prompt.manager.PromptManager.__init__: impossible: Python internal submodule constructor; Go consolidates PromptManager into AgentBase (no separately-constructed manager)
signalwire.core.agent.tools.decorator.ToolDecorator: impossible: Python decorator-factory class relies on the decorator protocol; Go uses DefineTool struct-literals
signalwire.core.agent.tools.decorator.ToolDecorator.create_class_decorator: impossible: Python decorator-factory relies on the decorator protocol; Go uses DefineTool struct-literals
signalwire.core.agent.tools.decorator.ToolDecorator.create_instance_decorator: impossible: Python decorator-factory relies on the decorator protocol; Go uses DefineTool struct-literals
signalwire.core.agent.tools.registry.ToolRegistry.__init__: impossible: Python internal submodule constructor; Go consolidates the tool registry into AgentBase (no separately-constructed registry)
signalwire.core.agent.tools.registry.ToolRegistry.register_class_decorated_tools: impossible: registers @tool-decorated class methods discovered via the decorator protocol; Go has no decorator-discovery equivalent
signalwire.core.agent.tools.type_inference.create_typed_handler_wrapper: impossible: Python runtime type-inference wraps handlers via signature introspection; Go has no runtime reflection-based schema inference in this path
signalwire.core.agent.tools.type_inference.infer_schema: impossible: Python runtime type-inference derives a JSON schema from a function signature at runtime; Go tool schemas are declared explicitly
signalwire.core.skill_base.SkillBase.define_tool: impossible: Python skill tool registration uses a decorator; Go uses BaseSkill.RegisterTools returning []ToolRegistration
signalwire.core.skill_base.SkillBase.validate_env_vars: impossible: Python validates required env vars via runtime introspection of a declared list; Go skills read os.Getenv directly in Setup (RequiredEnvVars declares the list)
signalwire.core.skill_base.SkillBase.validate_packages: impossible: Python validate_packages checks pip dependencies at runtime; Go dependencies are resolved at build time — no runtime package check

# --- Core SWML / SWAIG / function_result internals ---
signalwire.core.swaig_function.SWAIGFunction: impossible: Python SWAIGFunction is a callable wrapper around a tool registration; Go models tools as ToolDefinition + AgentBase.DefineTool (no callable class)
signalwire.core.swaig_function.SWAIGFunction.__call__: impossible: Python callable protocol (__call__) has no Go equivalent
signalwire.core.swaig_function.SWAIGFunction.__init__: impossible: constructor of the callable wrapper class Go does not model
signalwire.core.swaig_function.SWAIGFunction.execute: impossible: Python SWAIGFunction.execute invokes the wrapped callable; Go invokes via swaig.ToolHandler func values
signalwire.core.swaig_function.SWAIGFunction.to_swaig: impossible: serialises the callable-wrapper to a SWAIG entry; Go builds SWAIG entries from ToolDefinition directly
signalwire.core.swaig_function.SWAIGFunction.validate_args: impossible: validates against the wrapper class; Go validates via ToolDefinition.ValidateArgs
signalwire.core.swml_builder.SWMLBuilder.__getattr__: impossible: Python dynamic attribute dispatch (__getattr__) has no Go equivalent; verbs are explicit methods on swml.Service
signalwire.core.swml_renderer.SwmlRenderer: impossible: Python SwmlRenderer is a stateless render helper; Go folds rendering into swml.Service.Render / swaig.FunctionResult — no separate renderer type
signalwire.core.swml_renderer.SwmlRenderer.render_function_response_swml: impossible: Go builds function-response SWML via swaig.FunctionResult — no separate renderer
signalwire.core.swml_renderer.SwmlRenderer.render_swml: impossible: Go folds SWML rendering into swml.Service.Render — no separate renderer
signalwire.core.swml_service.SWMLService.__getattr__: impossible: Python dynamic attribute dispatch (__getattr__) has no Go equivalent; verbs are explicit methods on swml.Service

# --- RELAY abstract action mixin bases (Go flattens the hierarchy) ---
# Python factors the call-action controls into an abstract mixin chain
# (StoppableAction -> PausableAction -> VolumeAction -> concrete PlayAction/...),
# so the control methods live on the abstract bases. Go gives each concrete action its
# own stop/pause/resume/volume method (pkg/relay/action.go) and does NOT model the
# abstract bases as Go types — but the CONTRACT (each control method on each concrete
# action) IS present and surfaced (recorded in PORT_ADDITIONS.md as changeset item H).
# The signature gate excuses the abstract-base methods structurally
# (_is_abstract_action_base_method); the surface gate excuses the bare base classes +
# their methods here. Mirrors the TS port precedent.
signalwire.relay.call.StoppableAction: impossible: Go has no abstract-base type; the stop contract is flattened onto each concrete *Action (pkg/relay/action.go) — see PORT_ADDITIONS
signalwire.relay.call.StoppableAction.stop: impossible: Go has no abstract-base type; stop is present on each concrete *Action (see PORT_ADDITIONS)
signalwire.relay.call.PausableAction: impossible: Go has no abstract-base type; the pause/resume contract is flattened onto each concrete *Action (see PORT_ADDITIONS)
signalwire.relay.call.PausableAction.pause: impossible: Go has no abstract-base type; pause is present on each concrete *Action (see PORT_ADDITIONS)
signalwire.relay.call.PausableAction.resume: impossible: Go has no abstract-base type; resume is present on each concrete *Action (see PORT_ADDITIONS)
signalwire.relay.call.VolumeAction: impossible: Go has no abstract-base type; the volume contract is flattened onto each concrete *Action (see PORT_ADDITIONS)
signalwire.relay.call.VolumeAction.volume: impossible: Go has no abstract-base type; volume is present on each concrete *Action (see PORT_ADDITIONS)

# --- Relay Call / Client / Message ---
signalwire.relay.client.RelayClient.__aenter__: impossible: Python async context-manager protocol (__aenter__) has no Go equivalent; Go uses explicit Connect()/Stop()
signalwire.relay.client.RelayClient.__aexit__: impossible: Python async context-manager protocol (__aexit__) has no Go equivalent; Go uses explicit Connect()/Stop()
signalwire.relay.client.RelayClient.__del__: impossible: Python __del__ finalizer has no Go equivalent; Go GC + Stop() release the WebSocket
signalwire.relay.message.Message.__repr__: impossible: Python __repr__ object-protocol method has no Go analog (Stringer not surfaced as a reference method)

# --- REST namespace omissions ---
signalwire.rest.call_handler.PhoneCallHandler: Python PhoneCallHandler is a typing helper alias; Go port uses pkg/rest/namespaces/call_handler.go (string type)
signalwire.rest.namespaces.fabric.CxmlApplicationsResource: not_yet_implemented: CxmlApplicationsResource not yet wired in FabricNamespace
signalwire.rest.namespaces.fabric.CxmlApplicationsResource.create: not_yet_implemented: CxmlApplicationsResource not yet wired in FabricNamespace
signalwire.rest.namespaces.fabric.CxmlWebhooksResource: deprecated legacy resource; Go port omits per phone-binding.md (use phone_numbers.SetCxmlWebhook)
signalwire.rest.namespaces.fabric.FabricResource: internal base class for fabric resources; Go port aliases it to namespaces.CrudWithAddresses (List/Create/Get/Update/Delete + ListAddresses)
signalwire.rest.namespaces.fabric.FabricResourcePUT: internal base-class variant (PUT updates); Go port aliases it to namespaces.CrudWithAddresses constructed via NewCrudWithAddressesPUT, so it exposes CRUD + ListAddresses
signalwire.rest.namespaces.fabric.SwmlWebhooksResource: deprecated legacy resource; Go port omits per phone-binding.md (use phone_numbers.SetSwmlWebhook)

# --- Prefab internal handlers ---

# --- Livewire shim gaps ---
signalwire.livewire.Agent.llm_node: approved: 2026-07 user sign-off — LiveKit-specific pipeline node override; Go livewire shim delegates to the SWML AI verb
signalwire.livewire.Agent.stt_node: approved: 2026-07 user sign-off — LiveKit-specific pipeline node override; Go livewire shim delegates to the SWML AI verb
signalwire.livewire.Agent.tts_node: approved: 2026-07 user sign-off — LiveKit-specific pipeline node override; Go livewire shim delegates to the SWML AI verb
signalwire.livewire.Agent.update_instructions: approved: 2026-07 user sign-off — LiveKit runtime instruction mutation; Go exposes AgentSession.UpdateInstructions instead of on Agent
signalwire.livewire.RunContext.userdata: impossible: Go exposes RunContext.Userdata as an exported struct field (idiomatic direct access); a same-named accessor method cannot coexist with the field

# --- Misc not-yet-implemented items ---

# --- Idiom: Python class accessors that Go folds into private fields or package-level helpers ---
signalwire.agent_server.AgentServer.app: Python exposes the underlying FastAPI ``app`` object; Go uses net/http with no equivalent app handle
signalwire.agent_server.AgentServer.agents: Python exposes ``agents`` as a public dict attribute; Go keeps the map private (``agents map[string]*agent.AgentBase``) and exposes it via the ``GetAgents()`` accessor (idiomatic Go private-field + accessor)
signalwire.agent_server.AgentServer.logger: Python instance ``logger`` property; Go's AgentServer uses the package-level ``logging`` helper rather than a per-instance accessor
signalwire.core.agent_base.AgentBase.skill_manager: Python exposes ``self.skill_manager`` for direct access; Go folds the SkillManager into a private ``skillManager`` field and surfaces user-facing methods (AddSkill, RemoveSkill, ListSkills, HasSkill) directly on AgentBase
signalwire.core.skill_manager.SkillManager.loaded_skills: Python exposes ``loaded_skills`` as a public dict attribute; Go keeps the map private (``loadedSkills map[string]SkillBase``) and exposes it via the ``ListLoadedSkills()`` accessor (idiomatic Go private-field + accessor)
signalwire.core.skill_manager.SkillManager.logger: Python instance ``logger`` property; Go's SkillManager uses the package-level ``logging`` helper and has no per-instance logger accessor
signalwire.core.swml_service.SWMLService.security: Python exposes a ``security`` property returning a SecurityConfig; Go folds auth state into private fields on Service (basicAuthUser, bearerToken, apiKey, ...) configured via WithSecurityConfig/WithBasicAuth/WithBearerToken/WithAPIKey options
signalwire.core.swml_service.SWMLService.verb_registry: Python uses a separate VerbRegistry helper class; Go uses a private ``verbHandlers`` map on Service and exposes RegisterVerbHandler directly
signalwire.pom.pom.PromptObjectModel.sections: go-bean-accessor — Python exposes a ``sections`` list property; Go promotes it to an exported struct field ``Sections []*Section`` on PromptObjectModel (no method, direct field access is idiomatic)
signalwire.pom.pom.Section.subsections: go-bean-accessor — Python exposes a ``subsections`` list property; Go promotes it to an exported struct field ``Subsections []*Section`` on Section (no method, direct field access is idiomatic)

signalwire.core.security.webhook_middleware.make_webhook_validation_dependency: impossible: FastAPI dependency factory; Go exposes equivalent as security.WebhookMiddleware (http.Handler middleware) — see PORT_ADDITIONS.md

# --- REST _base empty alias classes ---
signalwire.rest._base.FabricResource: impossible: Python empty base class (FabricResource(CrudResource)); Go aliases it to namespaces.CrudWithAddresses — no distinct surface to emit
signalwire.rest._base.FabricResourcePUT: impossible: Python empty base-class variant; Go aliases it to namespaces.CrudWithAddresses via NewCrudWithAddressesPUT — no distinct surface

# =====================================================================
# BACKLOG: generated typed-payload modules (SWML/SWAIG types-generation pass) —
# NOT YET ADOPTED. The REST *_resources_generated METHOD surface AND the REST
# field-level *_types_generated wire types are adopted (one Go type per
# components/schemas entry, surfaced as <ns>_types_generated classes). The RELAY
# WS protocol types (relay.protocol_types_generated) and the read-side SWAIG+SWML
# webhook types (swml_webhooks_types_generated) are ALSO now adopted — generated by
# cmd/generate-rest into pkg/relay/protocol_types_generated.go (123 structs) and
# pkg/rest/namespaces/swml_webhooks_types_generated.go (9 structs) — so their
# omissions have been removed (they are IMPLEMENTED, not deferred).
#
# What remains deferred is the CORE SWML/SWAIG generated-type surface: the SWML
# verb config types (core.swml_verbs_generated) and the read-side SWAIG payload
# types (core.post_prompt_generated / core.swaig_request_generated /
# core.swaig_actions_generated) — the D-workstream. Grouped here so the gate can
# distinguish "known-deferred" from "unexpected". These are REAL reference symbols
# the port has not generated yet — nothing invented.
#
# Note: the PostPromptData / SwaigArgument gen-type folds are now SATISFIED by the
# swml_webhooks_types_generated module above (each leaf duplicates into that module
# AND a still-deferred core.* module in the reference; the surface diff folds the
# duplicated leaf to gen-type.<Leaf>, which the port's swml_webhooks copy matches),
# so their omission entries have been removed.
# ---------------------------------------------------------------------
# (d) Deferred NON-REST generated types (SWML verbs / SWAIG read-side payloads):
# SWML/SWAIG core *_generated typed-payload modules (D workstream):
