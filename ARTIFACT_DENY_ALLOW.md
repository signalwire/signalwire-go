# Artifact-deny allowlist (go)

Go has no package-manifest file-exclusion mechanism. Unlike Cargo (`include`/
`exclude`), npm (`.npmignore` / `files`), or a gemspec's `s.files`, a published
Go module is simply its committed tree at the tagged version — there is no
manifest that selects which files enter the "package." A consumer that
`go get`s `github.com/signalwire/signalwire-go` compiles only the packages it
imports (the library lives under `pkg/...`); the files below impose no build or
dependency cost on that consumer.

Every entry below must stay tracked in-repo: the shared porting-sdk audit
pipeline reads or runs each one in place (e.g. `diff_port_emission.py` runs
`go run ./cmd/emit-corpus`; `diff_skill_contracts.py` runs
`go run ./cmd/emit-skills`; the surface/signature/coverage gates read the JSON
and PORT_*.md contract files at the repo root). Deleting or relocating them, or
gating the cmd/ tools behind a build tag, would break that pipeline (which
cannot be edited from this repo).

Audit-contract data files (read in-place by porting-sdk audit scripts):

- CHECKLIST.md — porting audit-contract file read by porting-sdk audit scripts (orchestrator, 2026-07-06)
- DOC_AUDIT_IGNORE.md — porting audit-contract file read by porting-sdk audit scripts (orchestrator, 2026-07-06)
- PORT_ADDITIONS.md — porting audit-contract file read by porting-sdk audit scripts (orchestrator, 2026-07-06)
- PORT_EXAMPLE_OMISSIONS.md — porting audit-contract file read by porting-sdk audit_example_parity.py (orchestrator, 2026-07-06)
- PORT_OMISSIONS.md — porting audit-contract file read by porting-sdk audit scripts (orchestrator, 2026-07-06)
- PORT_SIGNATURE_OMISSIONS.md — porting audit-contract file read by porting-sdk audit scripts (orchestrator, 2026-07-06)
- PORT_TEST_OMISSIONS.md — porting audit-contract file read by porting-sdk audit scripts (orchestrator, 2026-07-06)
- REST_COVERAGE_GAPS.md — porting audit-contract file read by porting-sdk audit scripts (orchestrator, 2026-07-06)
- audit_coverage.json — porting audit-contract file read by porting-sdk audit_coverage_map.py (orchestrator, 2026-07-06)
- audit_coverage_baseline.json — porting audit-contract file read by porting-sdk audit_coverage_map.py (orchestrator, 2026-07-06)
- port_signatures.json — porting audit-contract file regenerated + read by scripts/run-ci.sh and porting-sdk diff scripts (orchestrator, 2026-07-06)
- port_surface.json — porting audit-contract file regenerated + read by scripts/run-ci.sh and porting-sdk audit scripts (orchestrator, 2026-07-06)
- port_surface_go.json — porting audit-contract file regenerated + read by scripts/run-ci.sh DOC-AUDIT (orchestrator, 2026-07-06)

Internal porting cmd tools (run in-place via `go run ./cmd/<tool>` by porting-sdk; not consumer commands):

- cmd/emit-corpus/main.go — internal EMISSION-DUMP tool run by porting-sdk diff_port_emission.py; go has no way to exclude it from the module tree (orchestrator, 2026-07-06)
- cmd/emit-skills/main.go — internal SKILL-DUMP tool run by porting-sdk diff_skill_contracts.py; go has no way to exclude it from the module tree (orchestrator, 2026-07-06)
