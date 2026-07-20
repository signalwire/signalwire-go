#!/usr/bin/env bash
# run-ci.sh — canonical local-and-CI gate runner for signalwire-go.
#
# Same script is invoked locally (`bash scripts/run-ci.sh`) AND by the
# GitHub Actions workflow. No drift between local and CI behavior.
#
# FMT / LINT / TEST are the CANONICAL scripts (self-bootstrapping, CWD-independent):
#   scripts/run-format.sh · scripts/run-lint.sh · scripts/run-tests.sh
# (shared env in scripts/_env.sh). Do not invoke gofmt / go vet / golangci-lint /
# go test directly — go through those three scripts (porting-sdk/RUN_LINT_FORMAT_SPEC.md).
#
# GATE SCHEDULING (porting-sdk/scripts/gate_scheduler.sh — CI_PERF S1 + S2):
#   Gates run CONCURRENTLY up to a cap (SW_CI_JOBS, default nproc), scheduled by
#   their DATA dependencies:
#     * S2 concurrent wave: the pure-Python side-effect-free gates (DRIFT, NO-CHEAT,
#       EMISSION, SKILL-CONTRACT, SWAIG-COVERAGE, SURFACE-DIFF, DOC-AUDIT, SWAIG-CLI,
#       GEN-FRESH-TESTS) overlap — they share no mutable state.
#     * S1 fail-fast: heavy gates (TEST, LINT, FMT, REST-COVERAGE, SPEC-PARITY) are
#       deferred behind the cheap wave, so a trivial cheap-gate failure surfaces in
#       seconds; --fail-fast aborts the run before TEST starts.
#   HARD ordering is data-dependency ONLY:
#     * DRIFT reads port_signatures.json that SIGNATURES writes → deps=SIGNATURES.
#     * SURFACE-FRESH regenerates port_surface.json + port_surface_go.json in place
#       (and restores them); SURFACE-DIFF reads port_surface.json, DOC-AUDIT reads
#       port_surface_go.json → all three share res=surface (mutually exclusive).
#   Per-gate PASS/FAIL + the FAILED_GATES tally preserved exactly; each gate's output
#   captured + replayed atomically.
#
# Flags:
#   --fail-fast   stop launching new gates at the first failure (local dev loop).
#
# Exit codes:
#   0  all gates passed
#   1  one or more gates failed
#   2  porting-sdk not found (configuration error, distinct from gate failure)
#
# Resolves porting-sdk via $PORTING_SDK or sibling ../porting-sdk/.

set -u
set -o pipefail

PORT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PORT_NAME="signalwire-go"

# ---- locate porting-sdk -----------------------------------------------------

resolve_porting_sdk() {
    if [ -n "${PORTING_SDK:-}" ] && [ -d "$PORTING_SDK/scripts" ]; then
        echo "$PORTING_SDK"
        return 0
    fi
    if [ -d "$PORT_ROOT/../porting-sdk/scripts" ]; then
        (cd "$PORT_ROOT/../porting-sdk" && pwd)
        return 0
    fi
    return 1
}

PORTING_SDK_DIR="$(resolve_porting_sdk)" || {
    echo "FATAL: porting-sdk not found, clone it adjacent to this repo" >&2
    echo "       (expected $PORT_ROOT/../porting-sdk or \$PORTING_SDK env var)" >&2
    exit 2
}

# ---- locate signalwire-python (Layer-D behavioral oracle) -------------------
# The Layer-D differs (diff_port_<surface>.py) build their golden oracle by
# importing signalwire-python; they put "<dir>/signalwire" on sys.path, and the
# importable package lives at signalwire-python/signalwire/signalwire/, so the
# path we hand to --python-sdk is <workspace>/signalwire-python/signalwire.
# Mirror diff_port_emission.py's adjacency resolution (CI checks it out as a
# sibling of porting-sdk; do NOT hardcode ~/src — CI clones elsewhere).
resolve_python_sdk() {
    local c
    for c in \
        "${PYTHON_SDK:-}" \
        "$PORTING_SDK_DIR/../signalwire-python/signalwire" \
        "$PORT_ROOT/../signalwire-python/signalwire" \
        "$HOME/src/signalwire-python/signalwire"; do
        if [ -n "$c" ] && [ -d "$c/signalwire" ]; then
            (cd "$c" && pwd)
            return 0
        fi
    done
    return 1
}

PYTHON_SDK_DIR="$(resolve_python_sdk)" || {
    echo "FATAL: signalwire-python not found for Layer-D behavioral gates" >&2
    echo "       (expected signalwire-python adjacent to porting-sdk or \$PYTHON_SDK env var)" >&2
    exit 2
}

# ---- gate plumbing (shared scheduler) ---------------------------------------

# shellcheck source=/dev/null
source "$PORTING_SDK_DIR/scripts/gate_scheduler.sh"

# ---- per-port commands ------------------------------------------------------

cd "$PORT_ROOT"

# Gate-enforcement plan PART D — go's wave-A doc gates are BLOCKING. The widened
# audit_docs / count_claim / status_claim / semver_diff findings are no longer
# report-only for go: this port's wave-A red list has been burned to zero, so any
# regression fails the run. (Unset/1 would keep them report-only; 0 makes them
# count toward the exit code.) Local and CI stay in lockstep because the flag lives
# here in run-ci.sh, not only in a workflow env.
export SW_WAVE_A_REPORT_ONLY=0

# Gate-enforcement plan D3 — REST 400-strict default fleet-wide. The shared
# mock_signalwire honors MOCK_SIGNALWIRE_STRICT=1 (test_harness/mock_signalwire/
# strict.py): a wire-shape violation (unknown key / wrong type) returns a 400
# instead of being tolerantly journaled, so the REST-COVERAGE + TEST lanes catch a
# regression the tolerant mock would swallow. Exported here (not only in a
# workflow) so local and CI stay in lockstep; the mock the per-test harness spawns
# inherits it. Declared load-bearing in WIRED_MODES.md (the WIRED-MODES guard
# reds if a merge drops this line).
export MOCK_SIGNALWIRE_STRICT=1

echo "==> running CI gates for $PORT_NAME (porting-sdk at $PORTING_SDK_DIR)"

# ---- Part 5: the per-gate --fn helpers are now DEAD — reproduced in the suites -
# surface_fresh_gate (SURFACE-FRESH), rest_coverage_gate (REST-COVERAGE), and
# spec_parity_gate (SPEC-PARITY) used to be defined here as `--fn` gate bodies.
# Those exact bodies are now reproduced INSIDE the Part-5 suites
# (scripts/suites/_surface_fresh.py, _rest_coverage.py, _spec_parity.py), so they
# are no longer defined here. pick_free_port() likewise moved into the suites.
# (Byte-identity vs the old per-gate path is proven by porting-sdk's
# tests/test_suite_parity*.py.)

# ---- register gates ----------------------------------------------------------
sched_init "$@"

# HEAVY (deferred behind the cheap wave for S1 fail-fast).
sched_gate TEST defer=1 desc="go test ./... (scripts/run-tests.sh)" \
    -- bash "$PORT_ROOT/scripts/run-tests.sh"

# ---- Part 5 gate SUITES ------------------------------------------------------
# The former per-gate SIGNATURES/DRIFT/SURFACE-*/SEMVER-DIFF/GEN-TYPE-DEGENERACY/
# GEN-IDIOM/ROUTE-COLLISION/GEN-FRESH*/BEHAVIORAL-*/EMISSION/ERROR-ENVELOPE/
# PAGINATION-WIRED/DOC-WIRE/REST-COVERAGE/SPEC-PARITY/SKILL-CONTRACT/SWAIG-*/
# WAIT-LIVENESS/DOC-*/COUNT-CLAIM/ACCESSOR-TRUTH/STATUS-CLAIM/README-INCLUDE/
# *-LEDGER/PACKAGE-SMOKE/META-CONSISTENT/ARTIFACT-DENY/RELEASE-FRESH gates now run
# under 6 SUITE engines. Each suite emits every original gate NAME as a
# `[SUITE:RULE] ... PASS/FAIL` rule ID (failure identity + allowlists + finding
# output unchanged). A suite exits nonzero iff any of its rules fails. Byte-identity
# vs the old per-gate path is proven by porting-sdk/tests/test_suite_parity*.py.
#
# The `--fn` helpers the old gates used (surface_fresh_gate, rest_coverage_gate,
# spec_parity_gate, pick_free_port) are reproduced INSIDE the suites, so they are
# no longer defined here.
#
# Former single-gate scheduler features preserved by the suites internally:
#   * SIGNATURES→DRIFT ordering, the SEMVER-DIFF-reads-SIGNATURES data dep, and the
#     SURFACE-FRESH regenerate-then-restore all live inside the SURFACE suite.
#   * mixed tiers are split with --rules: PACKAGE + BEHAVIORAL each schedule a
#     per-PR line and a nightly line (nightly members broken out below).
# GO-SPECIFIC vs the TS reference: go's SURFACE suite ALSO carries ROUTE-COLLISION
# (ts does not schedule it), and go's behavioral RELAY rule keeps go's exact
# spelling BEHAVIORAL-WIRE_RELAY (underscore). DOC-AUDIT reads go's on-disk
# port_surface_go.json, which SURFACE-FRESH regenerates+restores — so SURFACE and
# DOC-TRUTH share res=surface (mutually exclusive, exactly as the old per-gate
# SURFACE-FRESH/SURFACE-DIFF/DOC-AUDIT surface mutex did).

# SURFACE (parity spine): SIGNATURES→DRIFT ordered, SURFACE-FRESH regen/restore,
# SURFACE-DIFF, SEMVER-DIFF, GEN-TYPE-DEGENERACY, GEN-IDIOM, ROUTE-COLLISION — all
# read the one enumeration. res=surface: SURFACE-FRESH regenerates port_surface.json
# + port_surface_go.json + port_additions_actual.json in place (and restores them),
# so it must not overlap DOC-TRUTH's DOC-AUDIT read of port_surface_go.json.
sched_gate SURFACE res=surface desc="surface parity suite (SIGNATURES/DRIFT/SURFACE-FRESH/SURFACE-DIFF/SEMVER-DIFF/GEN-TYPE-DEGENERACY/GEN-IDIOM/ROUTE-COLLISION)" \
    -- python3 "$PORTING_SDK_DIR/scripts/suites/surface.py" --port go --repo "$PORT_ROOT"

# GEN (regen-from-specs family): the 5 GEN-FRESH rules.
sched_gate GEN defer=1 desc="generated-code freshness suite (GEN-FRESH/-TESTS/-RELAY/-SWAIG/-SWML)" \
    -- python3 "$PORTING_SDK_DIR/scripts/suites/gen.py" --port go --repo "$PORT_ROOT"

# BEHAVIORAL (one Layer-D pass per rule): the per-PR rules. WAIT-LIVENESS (nightly)
# is the separate line below. NOTE go's underscore spelling BEHAVIORAL-WIRE_RELAY.
sched_gate BEHAVIORAL defer=1 desc="behavioral suite (BEHAVIORAL-*/EMISSION/ERROR-ENVELOPE/PAGINATION-WIRED/DOC-WIRE/REST-COVERAGE/SPEC-PARITY/SKILL-CONTRACT/SWAIG-COVERAGE/SWAIG-CLI)" \
    -- python3 "$PORTING_SDK_DIR/scripts/suites/behavioral.py" --port go --repo "$PORT_ROOT" \
        --rules BEHAVIORAL-WIRE,BEHAVIORAL-SWML,BEHAVIORAL-STATE,BEHAVIORAL-HTTP,BEHAVIORAL-WIRE_RELAY,ENVELOPE,EMISSION,ERROR-ENVELOPE,PAGINATION-WIRED,DOC-WIRE,REST-COVERAGE,SPEC-PARITY,SKILL-CONTRACT,SWAIG-COVERAGE,SWAIG-CLI

sched_gate BEHAVIORAL-NIGHTLY tier=nightly defer=1 desc="behavioral suite, nightly rules (WAIT-LIVENESS)" \
    -- python3 "$PORTING_SDK_DIR/scripts/suites/behavioral.py" --port go --repo "$PORT_ROOT" \
        --rules WAIT-LIVENESS

# DOC-TRUTH (one markdown walk): DOC-AUDIT/DOC-LINKS/DOC-LANG-PURITY/DOC-ENV/
# COUNT-CLAIM/ACCESSOR-TRUTH/STATUS-CLAIM/README-INCLUDE. res=surface: DOC-AUDIT
# reads go's on-disk port_surface_go.json, which the SURFACE suite regenerates.
sched_gate DOC-TRUTH res=surface desc="doc-truth suite (DOC-AUDIT/DOC-LINKS/DOC-LANG-PURITY/DOC-ENV/COUNT-CLAIM/ACCESSOR-TRUTH/STATUS-CLAIM/README-INCLUDE)" \
    -- python3 "$PORTING_SDK_DIR/scripts/suites/doc_truth.py" --port go --repo "$PORT_ROOT"

# LEDGER: SUPPRESSION-LEDGER + IGNORE-LEDGER-VERIFY.
sched_gate LEDGER res=dayone desc="ledger governance suite (SUPPRESSION-LEDGER/IGNORE-LEDGER-VERIFY)" \
    -- python3 "$PORTING_SDK_DIR/scripts/suites/ledger.py" --port go --repo "$PORT_ROOT"

# PACKAGE: per-PR rules (ARTIFACT-DENY/RELEASE-FRESH); nightly rules (PACKAGE-SMOKE/
# META-CONSISTENT) on the separate line below.
sched_gate PACKAGE res=dayone desc="package suite, per-PR rules (ARTIFACT-DENY/RELEASE-FRESH)" \
    -- python3 "$PORTING_SDK_DIR/scripts/suites/package.py" --port go --repo "$PORT_ROOT" \
        --rules ARTIFACT-DENY,RELEASE-FRESH

sched_gate PACKAGE-NIGHTLY tier=nightly defer=1 res=dayone desc="package suite, nightly rules (PACKAGE-SMOKE/META-CONSISTENT)" \
    -- python3 "$PORTING_SDK_DIR/scripts/suites/package.py" --port go --repo "$PORT_ROOT" \
        --rules PACKAGE-SMOKE,META-CONSISTENT

# ---- gates that stay standalone (native toolchains + singletons) -------------
sched_gate NO-CHEAT desc="audit_no_cheat_tests" \
    -- python3 "$PORTING_SDK_DIR/scripts/audit_no_cheat_tests.py" --root "$PORT_ROOT"

sched_gate FMT defer=1 desc="gofmt via scripts/run-format.sh (local: auto-fix; CI: --check)" \
    -- bash "$PORT_ROOT/scripts/run-format.sh" ${CI:+--check}

sched_gate LINT defer=1 desc="go vet + golangci-lint via scripts/run-lint.sh" \
    -- bash "$PORT_ROOT/scripts/run-lint.sh"

# ---- §C1 doc/example/CLI execution gates ------------------------------------
# SNIPPET-COMPILE (typecheck doc code fences with the real SDK linked) + DOC-CLI
# (probe documented swaig-test invocations against the real CLI's parser).
# SNIPPET-COMPILE is HEAVY → tier=nightly; DOC-CLI stays per-PR (cheap CLI-parse).
# EXAMPLES-RUN executes/loads the shipped examples (nightly, defer, blocking).
# SNIPPET-RUN is dynamic-ports-only; for go it self-skips (SNIPPET-COMPILE covers
# the compiled port) — wired report-only so the self-skip never fails the run.
sched_gate SNIPPET-COMPILE tier=nightly desc="documented code snippets compile against the real SDK" \
    -- python3 "$PORTING_SDK_DIR/scripts/snippet_compile.py" --port go --repo "$PORT_ROOT"

sched_gate DOC-CLI desc="documented swaig-test invocations parse against the real CLI" \
    -- python3 "$PORTING_SDK_DIR/scripts/doc_cli.py" --port go --repo "$PORT_ROOT"

# DEAD-PUBLIC-ERROR stays standalone (source analysis of exported error types — not
# a doc-truth/behavioral rule). ERROR-ENVELOPE/PAGINATION-WIRED/DOC-WIRE run under
# the BEHAVIORAL suite; DOC-ENV/COUNT-CLAIM/ACCESSOR-TRUTH/STATUS-CLAIM under
# DOC-TRUTH.
sched_gate DEAD-PUBLIC-ERROR desc="exported error types are raised/caught/user-signalled (no dead error surface)" \
    -- python3 "$PORTING_SDK_DIR/scripts/dead_public_error.py" --port go --repo "$PORT_ROOT"

sched_gate EXAMPLES-RUN tier=nightly defer=1 desc="shipped examples load/compile (modulo EXAMPLES_RUN_ALLOW.md)" \
    -- python3 "$PORTING_SDK_DIR/scripts/examples_run.py" --port go --repo "$PORT_ROOT"

sched_gate SNIPPET-RUN tier=nightly defer=1 desc="dynamic-port doc snippets run to a zero exit (go: self-skips, SNIPPET-COMPILE covers it)" \
    -- python3 "$PORTING_SDK_DIR/scripts/snippet_run.py" --port go --repo "$PORT_ROOT" --report-only

# STRICT-MOCKS (§2.2) — re-run the RELAY suite with the mock in STRICT mode
# (MOCK_RELAY_STRICT=1: the mock 400s an unknown field or a duplicate id instead
# of tolerantly journaling it), so a wire-shape regression the tolerant mock would
# swallow fails loud. go's RELAY tests pass clean under strict today (empty red
# list). tier=nightly (a full second suite pass is heavy) + defer.
#
# -race (plan §2.16): the RELAY package is go's concurrency core — a
# context-cancelled WS read loop, sync.RWMutex-guarded connection, and goroutine
# event dispatch (pkg/relay/client.go). The Go race detector instruments this
# strict RELAY pass so a data race in that machinery reds the gate instead of
# flaking intermittently. Kept on the strict RELAY line (not the whole ./...) to
# bound the -race slowdown to the package that actually needs it. Declared
# load-bearing in WIRED_MODES.md.
sched_gate STRICT-MOCKS tier=nightly defer=1 desc="RELAY suite passes with the mock in 400-on-violation strict mode (MOCK_RELAY_STRICT=1), race detector on" \
    -- env MOCK_RELAY_STRICT=1 bash "$PORT_ROOT/scripts/run-tests.sh" -race -count=1 ./pkg/relay/...

# ROOT-HYGIENE + PUBLIC-JARGON stay standalone (source/root analysis, not a suite
# family).
sched_gate ROOT-HYGIENE res=dayone desc="no audit/scratch clutter tracked at repo root (allowlist ROOT_HYGIENE_ALLOW.md)" \
    -- python3 "$PORTING_SDK_DIR/scripts/root_hygiene.py" --port go --repo "$PORT_ROOT"

# DUP-TREE (plan 3.3): go keeps a top-level rest/ docs tree alongside pkg/rest/. The
# duplicate-basename README pairs declared in DUP_TREE_PAIRS.md must stay in sync —
# byte-identical or a pointer stub — so the two trees can't silently re-diverge.
sched_gate DUP-TREE res=dayone desc="parallel doc trees stay in sync (rest/ vs pkg/rest/ README pairs — identical or pointer)" \
    -- python3 "$PORTING_SDK_DIR/scripts/dup_tree.py" --port go --repo "$PORT_ROOT"

# WIRED-MODES (plan 1.6 / D7): the merge-coherence guard. WIRED_MODES.md lists the
# load-bearing env/mode lines this run-ci MUST carry (MOCK_RELAY_STRICT=1,
# MOCK_SIGNALWIRE_STRICT export, -race). If a future merge silently drops one, this
# gate reds instead of shipping a green-but-vacuous strict/race lane.
# GUARDED: check_wired_modes.py ships on the porting-sdk plan branch; until that
# merges to porting-sdk main (which CI clones), skip-with-pass rather than red on a
# not-yet-landed sibling script. Remove the guard once it's on porting-sdk main.
sched_gate WIRED-MODES res=dayone desc="load-bearing run-ci modes present (WIRED_MODES.md merge-coherence guard)" \
    -- bash -c 'if [ -f "$1/scripts/check_wired_modes.py" ]; then python3 "$1/scripts/check_wired_modes.py" --port go --repo "$2"; else echo "[wired-modes] check_wired_modes.py not on porting-sdk main yet — skip-pass (plan-branch dep)"; fi' _ "$PORTING_SDK_DIR" "$PORT_ROOT"

sched_gate PUBLIC-JARGON res=dayone desc="no porting/internal jargon in the public API surface" \
    -- python3 "$PORTING_SDK_DIR/scripts/public_jargon.py" --port go --repo "$PORT_ROOT"

# DOC-SURFACE (plan §6.3): godoc coverage floor on the public surface. The floor
# is pinned in .doc_surface_floor (92.0% today) and ratchets up via --write-floor;
# report-only at graduation, so a doc regression is visible without failing the run
# yet (never-regress is enforced once the floor flips blocking).
# GUARDED: doc_surface.py ships on the porting-sdk plan branch; until it merges to
# porting-sdk main (which CI clones), skip-with-pass rather than red on a not-yet-
# landed sibling script. Remove the guard once it's on porting-sdk main.
sched_gate DOC-SURFACE res=dayone desc="godoc coverage floor on the public API surface (report-only, ratchets via .doc_surface_floor)" \
    -- bash -c 'if [ -f "$1/scripts/doc_surface.py" ]; then python3 "$1/scripts/doc_surface.py" --port go --repo "$2" --report-only; else echo "[doc-surface] doc_surface.py not on porting-sdk main yet — skip-pass (plan-branch dep)"; fi' _ "$PORTING_SDK_DIR" "$PORT_ROOT"

# GATE-INVENTORY NOTE (plan §2.16): porting-sdk/GATE_INVENTORY.md is generated by
# gen_gate_inventory.py from the REFERENCE port's run-ci.sh (typescript — the
# canonical copy every port mirrors), so the gates below that are GO-SPECIFIC do
# NOT appear in that generated inventory and that is intentional, not drift:
#   * DUP-TREE / WIRED-MODES — go keeps parallel rest/+relay/ doc trees and
#     load-bearing strict/-race modes that the ts reference does not have.
#   * ROUTE-COLLISION on the SURFACE suite — go schedules it; ts does not.
#   * BEHAVIORAL-WIRE_RELAY — go's underscore spelling of the RELAY behavioral rule.
#   * the STRICT-MOCKS RELAY line carries -race (go's concurrency core; §2.16) and
#     MOCK_SIGNALWIRE_STRICT=1 is exported for the REST lanes (D3) — both declared
#     in WIRED_MODES.md so a merge can't silently drop them.
# A reader diffing this file against GATE_INVENTORY.md should treat these as the
# port's own additions, governed here.

# ---- summary ----------------------------------------------------------------

sched_run
rc=$?
if [ "$rc" -eq 0 ]; then
    echo "==> CI PASS"
else
    echo "==> CI FAIL (gates:$FAILED_GATES )"
fi
exit "$rc"
