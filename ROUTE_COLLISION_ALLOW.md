# ROUTE_COLLISION_ALLOW.md — justified route-collision exceptions (Go)

Each entry is a proven, spec-documented exception, not a way to silence the gate.
Key format: `<Class>.<canonical_op>` (matching the gate's finding key).

## Entries

(none — the ROUTE-COLLISION check is now spec-aware: it recognizes the fabric
`call_flow`/`conference_room` SINGULAR address sub-paths as spec-faithful platform
routing directly from `rest-apis/fabric/openapi.yaml`, so the former
`CallFlows.list_addresses` / `ConferenceRooms.list_addresses` entries — which
existed only to excuse the plural-collection heuristic — are obsolete and were
retired 2026-07-19 per the no-launder rule: a stale allowlist entry is retired once
the check handles the case, not left to rot.)
