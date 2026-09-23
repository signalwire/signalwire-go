# Changelog

All notable changes to the SignalWire AI Agents Go SDK are documented here.

This project adheres to [Semantic Versioning](https://semver.org/). Versions are
published as git tags (`v<MAJOR>.<MINOR>.<PATCH>`) resolved by the Go module proxy.

## 3.0.0

First release of the generated-REST surface. Unreleased — no `v3.x` tag has been
pushed; the version below is intent. `port_signatures.baseline.json` carries the
separate SemVer *floor* (`baseline_version`) that the SEMVER-DIFF gate diffs against;
that floor is a snapshot identifier and is deliberately not renumbered with this entry.

- REST client with generated, typed namespaced resources across the REST API
  namespaces (calling, chat, datasphere, fabric, fax, logs, message, project,
  pubsub, relay-rest, video, voice).
- RELAY WebSocket client (Blade/JSON-RPC 2.0) for real-time call control, with
  the four correlation mechanisms (JSON-RPC id, call_id, control_id, tag).
- AgentBase, SWML document model/builder, SWAIG function-result action layer,
  DataMap server-side tools, contexts/steps workflows, skills, and prefabs.
- `swaig-test` CLI for local agent testing.
- Full cross-port CI gate set wired via `scripts/run-ci.sh`, including the
  Wave-3 release-readiness gates (SEMVER-DIFF, RELEASE-FRESH, META-CONSISTENT,
  strict IGNORE-LEDGER-VERIFY) and a gated publish workflow.
