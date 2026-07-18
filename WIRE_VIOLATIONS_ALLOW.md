# WIRE_VIOLATIONS_ALLOW.md — signed exceptions to the STRICT-MOCKS wire-truth gate

The STRICT-MOCKS consumer (`porting-sdk/scripts/assert_no_wire_violations.py`, wired
into REST-COVERAGE / EXAMPLES-RUN / SNIPPET-RUN) reads the mock journal after a run
and fails on ANY `wire_violation` — a request/frame that put a shape on the wire the
OpenAPI/RELAY spec does not declare (an undeclared query param, an unknown body key,
an unknown frame field). A wire violation is a spec bug or a real defect; the fix is
to make the wire match the spec, NOT to allowlist it.

This file exists for the rare, genuinely-justified exception, and each entry needs a
human-signed reason. Format (one per line):

    - <kind>:<name> — reason (approver, date)

where `<kind>` is the violation kind (`unknown_query_param`, `unknown_body_key`,
`unknown_frame_field`, `duplicate_command_id`) and `<name>` is the offending
key/param name. A bare `kind:name` with no ` — reason` is NOT matched, so it cannot
silently widen the allowlist.

## Currently empty

No entries. The wired gates (REST-COVERAGE / EXAMPLES-RUN / SNIPPET-RUN) run wire-clean
against the reference.

Two known spec gaps were surfaced during the STRICT-MOCKS bring-up but are NOT
allowlisted here, because REST-COVERAGE's coverage-driving run selects only the
generated wire suite (`go test ./pkg/rest/... -run 'Gen_'`) — the hand-authored
`pkg/rest/namespaces/*_mock_test.go` files that legitimately exercise these two
spec gaps do NOT run under that selector (they still run under the plain TEST gate,
which is not journal-checked). A name-only token like `unknown_query_param:cursor`
would also over-broadly mask any future real violation on endpoints that legitimately
declare `cursor`, so keeping them out of the checked journal (rather than
allowlisting) is the tighter fix:

  * `page_size` on `relay-rest.list_recordings` — the spec's `list_recordings` op has
    `parameters: []` while every sibling `list_*` op declares `page_size`.
    Owner-approved to FIX THE SPEC (add `page_size`), pending prime-rails confirmation
    that the server accepts it. Tracked separately (porting-sdk fix/recordings-pagination-spec);
    do NOT strip the test (`pkg/rest/namespaces/small_namespaces_mock_test.go`,
    `TestRecordings_*`).
  * `cursor` on `fabric.list_fabric_addresses` and `fabric.list_ai_agents` — same
    class: the fabric list ops have `parameters: []`, but the server returns a
    `links.next` cursor URL that the SDK's generic paginator (`Paginate()` /
    `PaginatedIterator`) replays as a `?cursor=` param. Same owner+prime-rails
    adjudication as recordings. go additionally exercises this on `list_ai_agents`
    (`pkg/rest/namespaces/paginate_method_mock_test.go`,
    `TestCrudResourcePaginate_WalksAllPages` — the base `CrudResource.Paginate()`
    path) beyond the `list_fabric_addresses` case
    (`pkg/rest/namespaces/pagination_mock_test.go`) python covers — identical gap,
    different endpoint.
