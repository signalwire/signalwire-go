# Plan: ST1003 initialism renames (idiomatic Go acronym capitalization)

## STATUS: DONE (2026-07-14)
- **23 public-symbol renames**: DONE (HTTPClient/SIPProfileNamespace/…, verified all-caps; DRIFT clean).
- **Generated REST field surface**: DONE — the generator's name-minting
  (`structFieldName`/`segCase`/`commonInitialisms` in `cmd/generate-rest/main.go`) now
  emits ST1003-idiomatic struct field names (`CallerId`→`CallerID`, `FallbackUrl`→
  `FallbackURL`, `url_method`→`URLMethod`). Regenerated the whole REST surface.
  Wire-safety proven: the multiset of all `body["…"]` keys + `json:"…"` tags is
  byte-identical before/after (696 tokens each side, zero net change) — the wire key
  derives from the SPEC property name, not the Go field name — and `port_signatures.json`
  is byte-identical (goNameToSnake round-trips `CallerID`→`caller_id`), so DRIFT stayed 0.
- **ST1003 ENABLED** in `.golangci.yml` (staticcheck `checks: +ST1003`). `run-lint.sh`
  clean. NOTE: staticcheck's ST1003 has an internal generated-file skip that
  golangci-lint's `generated: disable` does NOT override — so ST1003 lints only
  hand-written code; the generated names were made idiomatic anyway so they are correct
  regardless of linter visibility.
- **Type names LEFT as-is**: `Play_url` and `Types_StatusCodes_*` carry underscores but
  are PARITY-LOCKED — the Python oracle's own `ref_name` uses the identical underscore
  spellings (`python_surface.json`: `Types_StatusCodes_*`; `play_url`), so renaming the
  Go type would break DRIFT/SURFACE-DIFF. (They're generated types, which ST1003 skips
  anyway, so no lint conflict.)
- Full `bash scripts/run-ci.sh` = CI PASS (TEST/FMT/LINT-with-ST1003/DRIFT/SURFACE-DIFF/
  REST-COVERAGE/SEMVER-DIFF/GEN-FRESH/GEN-IDIOM all green). SEMVER-DIFF green with no
  baseline regen (struct field names aren't enumerated surface).

---

**Goal:** rename the 23 public Go symbols flagged by staticcheck ST1003 so embedded
acronyms are all-caps (Go convention: `URL`/`HTTP`/`SIP`/`API`/`RPC`, not `Url`/`Http`/`Sip`/…),
WITHOUT breaking Python-parity drift. Then ST1003 can stay ENABLED in the lint gate.

## Why this is safe (verified, not assumed)

Names cross the Go↔Python boundary through an explicit translation layer, so idiomatic Go
names map to canonical Python names and the drift gate compares the canonical side:

- **Methods**: `internal/surface/tables.go` has explicit `Methods map[string]string` =
  `goMethodName -> pythonName`. Renaming the Go method only changes the **key**; the
  **value** (Python canonical, e.g. `set_web_hook_url`) is unchanged → drift unchanged.
- **Struct fields / params**: auto-converted by `goNameToSnake` (cmd/enumerate-signatures).
  VERIFIED the existing converter already yields identical output for the all-caps forms:
  `SIPProfile→sip_profile`, `HTTPClient→http_client`, `SetWebHookURL→set_web_hook_url`,
  `ExecuteRPC→execute_rpc`, `CreateSimpleAPITool→create_simple_api_tool`,
  `RPCAiMessage→rpc_ai_message`. No converter change needed.
- **Types**: tables.go keys like `"rest.HttpClient"` are the Go side (rename to
  `"rest.HTTPClient"`); the `Class:` / `Module:` values are PYTHON's actual names
  (`HttpClient`, `SipProfileResource` — Python does NOT follow Go's rule) and MUST NOT
  change — they're the drift reference (confirmed in python_signatures.json).

## The 23 renames (Go symbol → idiomatic Go symbol; canonical/Python side unchanged)

Methods: ManualSetProxyUrl→ManualSetProxyURL, SetWebHookUrl→SetWebHookURL,
SetPostPromptUrl→SetPostPromptURL, EnableSipRouting→EnableSIPRouting,
RegisterSipRoutingCallback→RegisterSIPRoutingCallback,
AutoMapSipUsernames→AutoMapSIPUsernames, RegisterSipUsername→RegisterSIPUsername,
SetupSipRouting→SetupSIPRouting, RegisterGlobalSipRoutingCallback→RegisterGlobalSIPRoutingCallback,
SipRefer→SIPRefer, ExecuteRpc→ExecuteRPC, RpcDial→RPCDial, RpcAiMessage→RPCAiMessage,
RpcAiUnhold→RPCAiUnhold, HttpClient(method)→HTTPClient.
Types: HttpClient→HTTPClient, SipProfileNamespace→SIPProfileNamespace,
SipProfileResource→SIPProfileResource.
Funcs: CreateSimpleApiTool→CreateSimpleAPITool, NewHttpClient→NewHTTPClient,
NewSipProfileNamespace→NewSIPProfileNamespace, extractSipUsername→extractSIPUsername.
Field: SipProfile→SIPProfile.

NOTE: only the leading acronym changes; `Ai` is left as-is (RPCAiMessage, not RPCAIMessage) —
staticcheck does not flag `Ai`, and `AI`-vs-`Ai` is a separate question not in scope here.

## Execution

1. Per symbol: rename the declaration AND every call site (use gofmt-safe identifier
   replacement; verify with grep that no `\bOldName\b` remains except in tables.go values
   that are Python names — there are none, since Python uses different spellings).
2. Update tables.go: rename Go-side keys (`"rest.HttpClient"`→`"rest.HTTPClient"`, method-map
   keys); LEAVE `Class:`/`Module:`/value strings untouched.
3. Rebuild + regenerate signatures; run the full 10-gate run-ci.sh.
4. The proof: DRIFT + SURFACE-FRESH + SURFACE-DIFF gates stay PASS (canonical output
   unchanged), and ST1003 count → 0.
5. Enable ST1003 in .golangci.yml (remove from any exclusion / confirm it's on).

## Breaking-change note
These are public API renames. Acceptable because the SDK is pre-release/unversioned and the
idiomatic form is correct Go; documented in the PR. Examples + tests updated in the same PR.
