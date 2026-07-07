# ROUTE_COLLISION_ALLOW.md — justified route-collision exceptions (Go)

Each entry is a proven, spec-documented exception, not a way to silence the gate.
Key format: `<Class>.<canonical_op>` (matching the gate's finding key).

## Entries

- `CallFlows.list_addresses` — The SignalWire fabric API serves Call Flow addresses
  (and versions) under the SINGULAR `call_flow` sub-path — `/api/fabric/resources/
  call_flow/{id}/addresses` — while the collection itself is the PLURAL
  `/api/fabric/resources/call_flows`. This is a real platform wire quirk, documented
  in the authoritative spec (porting-sdk `rest-apis/fabric/openapi.yaml`, the
  `/resources/call_flows` `x-sdk-resource` block: "Versions AND addresses live under
  the SINGULAR call_flow sub-path (a real platform quirk)"). The reference (Python)
  overrides `list_addresses` to this singular path and serves exactly ONE route for
  it. Go now matches: `CallFlowsResource` embeds the plain `*CrudResource` (NOT
  `*CrudWithAddresses`), so the inherited plural-path `ListAddresses` is unreachable
  and the singular override is the class's ONLY `list_addresses` route. The gate's
  plural-collection heuristic still flags the divergent segment, but there is a single
  canonical route and it is the correct (spec/wire) one. (burn-go, 2026-07-07)
- `ConferenceRooms.list_addresses` — Same fabric platform quirk: Conference Room
  addresses are served under the SINGULAR `conference_room` sub-path
  (`/api/fabric/resources/conference_room/{id}/addresses`) while the collection is the
  plural `/api/fabric/resources/conference_rooms`. `ConferenceRoomsResource` embeds
  `*CrudResource`, so the singular override is the only `list_addresses` route, matching
  the Python reference. (burn-go, 2026-07-07)
