# Root-hygiene allowlist

Files tracked at the repo root that the porting-sdk audit pipeline and this
repo's `scripts/run-ci.sh` read by relative path from the root. Moving them
would break the shared pipeline (which cannot be edited from here). Each is
load-bearing, not clutter.

- CHECKLIST.md — required audit-contract file read by porting-sdk audit scripts (orchestrator, 2026-07-06)
- DOC_AUDIT_IGNORE.md — required audit-contract file read by porting-sdk audit scripts (orchestrator, 2026-07-06)
- PORT_ADDITIONS.md — required audit-contract file read by porting-sdk audit scripts (orchestrator, 2026-07-06)
- PORT_OMISSIONS.md — required audit-contract file read by porting-sdk audit scripts (orchestrator, 2026-07-06)
- PORT_SIGNATURE_OMISSIONS.md — required audit-contract file read by porting-sdk audit scripts (orchestrator, 2026-07-06)
- PORT_TEST_OMISSIONS.md — required audit-contract file read by porting-sdk audit scripts (orchestrator, 2026-07-06)
- PORT_EXAMPLE_OMISSIONS.md — required audit-contract file read by porting-sdk audit_example_parity.py (orchestrator, 2026-07-06)
- REST_COVERAGE_GAPS.md — required audit-contract file read by porting-sdk audit scripts (orchestrator, 2026-07-06)
- audit_coverage.json — required audit-contract file read by porting-sdk audit_coverage_map.py (orchestrator, 2026-07-06)
- audit_coverage_baseline.json — required audit-contract file read by porting-sdk audit_coverage_map.py (orchestrator, 2026-07-06)
- port_additions_actual.json — regenerated + read at root by scripts/run-ci.sh SURFACE-FRESH and porting-sdk diff_port_surface.py (orchestrator, 2026-07-06)
- port_signatures.json — regenerated + read at root by scripts/run-ci.sh and porting-sdk diff_port_signatures.py (orchestrator, 2026-07-06)
- port_surface.json — regenerated + read at root by scripts/run-ci.sh and porting-sdk audit_docs.py/ignore_ledger_verify.py (orchestrator, 2026-07-06)
- port_surface_go.json — regenerated + read at root by scripts/run-ci.sh DOC-AUDIT (orchestrator, 2026-07-06)
- ROOT_HYGIENE_ALLOW.md — this allowlist itself, required at root by porting-sdk root_hygiene.py (orchestrator, 2026-07-06)
- ARTIFACT_DENY_ALLOW.md — required at root by porting-sdk artifact_deny.py (orchestrator, 2026-07-06)
- GEN_TYPE_DEGENERACY_ALLOW.md — required at root by porting-sdk gen_type_degeneracy.py (user-approved 2026-07-07)
- ROUTE_COLLISION_ALLOW.md — required at root by porting-sdk route_collision.py (user-approved 2026-07-07)
