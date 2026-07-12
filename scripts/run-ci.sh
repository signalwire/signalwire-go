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

echo "==> running CI gates for $PORT_NAME (porting-sdk at $PORTING_SDK_DIR)"

pick_free_port() {
    python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()'
}

# SURFACE-FRESH — the committed cross-port port_surface.json must match a fresh
# regen (modulo the volatile generated_from git-sha). The enumerator rewrites
# port_surface.json + port_surface_go.json + port_additions_actual.json in place;
# restore all three on any exit path so the gate is side-effect-free.
surface_fresh_gate() {
    trap "git checkout -- port_surface.json port_surface_go.json port_additions_actual.json 2>/dev/null" RETURN
    # Scratch under the repo-local, gitignored .sw-tmp/ (never machine-wide /tmp).
    mkdir -p "$PORT_ROOT/.sw-tmp"
    local committed="$PORT_ROOT/.sw-tmp/committed_surface.json"
    git show HEAD:port_surface.json > "$committed" 2>/dev/null \
        || cp port_surface.json "$committed"
    go run ./cmd/enumerate-surface || return $?
    python3 "$PORTING_SDK_DIR/scripts/check_surface_freshness.py" \
        --committed "$committed" \
        --fresh port_surface.json
}

# REST-COVERAGE — every implemented REST route covered success+error. Self-
# contained: spins its own mock on a free port, runs the rest suite serially, then
# checks the journal.
rest_coverage_gate() {
    local port
    port="$(pick_free_port)" || { echo "could not allocate a free port" >&2; return 1; }
    local mock_pkg_parent="$PORTING_SDK_DIR/test_harness/mock_signalwire"
    export PYTHONPATH="$mock_pkg_parent${PYTHONPATH:+:$PYTHONPATH}"
    # Mock log under the repo-local, gitignored .sw-tmp/ (never machine-wide /tmp).
    mkdir -p "$PORT_ROOT/.sw-tmp"
    local mock_log="$PORT_ROOT/.sw-tmp/rest_cov_mock_go.$$.log"
    python3 -m mock_signalwire --host 127.0.0.1 --port "$port" --log-level error \
        >"$mock_log" 2>&1 &
    local mock_pid=$!
    # shellcheck disable=SC2064
    trap "kill $mock_pid 2>/dev/null" RETURN
    # Fail LOUD if the mock dies mid-startup or never becomes healthy — never hang.
    local i ready=0
    for i in $(seq 1 60); do
        if ! kill -0 "$mock_pid" 2>/dev/null; then
            echo "mock_signalwire died on port $port — log:" >&2
            cat "$mock_log" >&2
            return 1
        fi
        if python3 -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:$port/__mock__/health',timeout=1)" 2>/dev/null; then
            ready=1
            break
        fi
        sleep 0.5
    done
    if [ "$ready" -ne 1 ]; then
        echo "mock_signalwire on port $port not healthy within 30s" >&2
        return 1
    fi
    python3 -c "import urllib.request; urllib.request.urlopen(urllib.request.Request('http://127.0.0.1:$port/__mock__/journal/reset',method='POST'),timeout=5).read()"
    MOCK_SIGNALWIRE_PORT="$port" go test "$PORT_ROOT/pkg/rest/..." -p 1 -count=1 || return 1
    python3 -m mock_signalwire.rest_coverage \
        --mock-url "http://127.0.0.1:$port" \
        --spec-root "$PORTING_SDK_DIR/rest-apis" \
        --allowlist "$PORTING_SDK_DIR/REST_COVERAGE_BASELINE.md" \
        --allowlist "$PORT_ROOT/REST_COVERAGE_GAPS.md" \
        --gap-baseline "$PORTING_SDK_DIR/REST_COVERAGE_GAP_BASELINE.md"
}

# SPEC-PARITY — implemented routes == canonical spec. cmd/route-registry drives the
# live RestClient through a recording transport and captures every dispatched route.
spec_parity_gate() {
    local mock_pkg_parent="$PORTING_SDK_DIR/test_harness/mock_signalwire"
    export PYTHONPATH="$mock_pkg_parent${PYTHONPATH:+:$PYTHONPATH}"
    local registry
    registry="$(mktemp)"
    # 2>/dev/null so the SDK's deprecation-warning logger (stderr) can't pollute
    # the JSON; the registry exits non-zero if Set B is incomplete.
    if ! go run "$PORT_ROOT/cmd/route-registry" >"$registry" 2>/dev/null; then
        rm -f "$registry"
        return 1
    fi
    python3 "$PORTING_SDK_DIR/scripts/diff_spec_implementation.py" \
        --registry-json "$registry" \
        --gaps "$PORTING_SDK_DIR/SPEC_IMPLEMENTATION_GAPS.md"
    local rc=$?
    rm -f "$registry"
    return $rc
}

# ---- register gates ----------------------------------------------------------
sched_init "$@"

sched_gate TEST defer=1 desc="go test ./... (scripts/run-tests.sh)" \
    -- bash "$PORT_ROOT/scripts/run-tests.sh"

sched_gate SIGNATURES desc="regenerate port_signatures.json" \
    -- bash -c 'go run ./cmd/enumerate-signatures > port_signatures.json'

sched_gate DRIFT deps=SIGNATURES desc="diff_port_signatures vs python reference" \
    -- python3 "$PORTING_SDK_DIR/scripts/diff_port_signatures.py" \
        --reference "$PORTING_SDK_DIR/python_signatures.json" \
        --port-signatures "$PORT_ROOT/port_signatures.json" \
        --surface-omissions "$PORT_ROOT/PORT_OMISSIONS.md" \
        --surface-additions "$PORT_ROOT/PORT_ADDITIONS.md" \
        --omissions "$PORT_ROOT/PORT_SIGNATURE_OMISSIONS.md"

sched_gate SURFACE-FRESH res=surface desc="check_surface_freshness vs committed port_surface.json" \
    --fn surface_fresh_gate

sched_gate NO-CHEAT desc="audit_no_cheat_tests" \
    -- python3 "$PORTING_SDK_DIR/scripts/audit_no_cheat_tests.py" --root "$PORT_ROOT"

sched_gate REST-COVERAGE defer=1 desc="every implemented REST route covered success+error (parity + allowlist)" \
    --fn rest_coverage_gate

sched_gate SPEC-PARITY defer=1 desc="implemented routes == canonical spec (modulo SPEC_IMPLEMENTATION_GAPS.md)" \
    --fn spec_parity_gate

sched_gate GEN-FRESH desc="generated REST layer matches the canonical specs" \
    -- go run ./cmd/generate-rest --check

sched_gate GEN-FRESH-TESTS desc="generated REST wire tests match the canonical specs" \
    -- go run ./cmd/generate-rest-tests --check

sched_gate GEN-FRESH-RELAY desc="generated RELAY protocol types match the canonical specs" \
    -- go run ./cmd/generate-relay-protocol --check

sched_gate GEN-FRESH-SWAIG desc="generated SWAIG read-side payloads match the canonical specs" \
    -- go run ./cmd/generate-swaig-payloads --check

sched_gate GEN-FRESH-SWML desc="generated SWML verb config types match the canonical specs" \
    -- go run ./cmd/generate-swml-verbs --check

sched_gate EMISSION desc="diff_port_emission vs python to_dict() oracle" \
    -- python3 "$PORTING_SDK_DIR/scripts/diff_port_emission.py" \
        --port go \
        --port-repo "$PORT_ROOT"

# ---- Layer-D behavioral coverage (per-surface differs vs python oracle) ------
sched_gate BEHAVIORAL-WIRE desc="diff_port_wire vs python oracle (Layer D)" \
    -- python3 "$PORTING_SDK_DIR/scripts/diff_port_wire.py" \
        --port go --python-sdk "$PYTHON_SDK_DIR" \
        --dump-cmd "go run ./cmd/wire-dump"

sched_gate BEHAVIORAL-SWML desc="diff_port_swml vs python oracle (Layer D)" \
    -- python3 "$PORTING_SDK_DIR/scripts/diff_port_swml.py" \
        --port go --python-sdk "$PYTHON_SDK_DIR" \
        --dump-cmd "go run ./cmd/swml-dump"

sched_gate BEHAVIORAL-STATE desc="diff_port_state vs python oracle (Layer D)" \
    -- python3 "$PORTING_SDK_DIR/scripts/diff_port_state.py" \
        --port go --python-sdk "$PYTHON_SDK_DIR" \
        --dump-cmd "go run ./cmd/state-dump"

sched_gate BEHAVIORAL-HTTP desc="diff_port_http vs python oracle (Layer D)" \
    -- python3 "$PORTING_SDK_DIR/scripts/diff_port_http.py" \
        --port go --python-sdk "$PYTHON_SDK_DIR" \
        --dump-cmd "go run ./cmd/http-dump"

sched_gate BEHAVIORAL-WIRE_RELAY desc="diff_port_wire_relay vs python oracle (Layer D)" \
    -- python3 "$PORTING_SDK_DIR/scripts/diff_port_wire_relay.py" \
        --port go --python-sdk "$PYTHON_SDK_DIR" \
        --dump-cmd "go run ./cmd/wire-relay-dump"

sched_gate SKILL-CONTRACT desc="diff_skill_contracts vs python reference" \
    -- python3 "$PORTING_SDK_DIR/scripts/diff_skill_contracts.py" \
        --dump-cmd "go run ./cmd/emit-skills" \
        --port-repo "$PORT_ROOT"

sched_gate SWAIG-COVERAGE desc="every engine SWAIG action emittable (modulo allowlist)" \
    -- python3 "$PORTING_SDK_DIR/scripts/swaig_coverage.py" --check \
        --emission "$PORT_ROOT/pkg/swaig/function_result.go"

sched_gate FMT defer=1 desc="gofmt via scripts/run-format.sh (local: auto-fix; CI: --check)" \
    -- bash "$PORT_ROOT/scripts/run-format.sh" ${CI:+--check}

sched_gate LINT defer=1 desc="go vet + golangci-lint via scripts/run-lint.sh" \
    -- bash "$PORT_ROOT/scripts/run-lint.sh"

sched_gate DOC-AUDIT res=surface desc="audit_docs vs port_surface_go.json" \
    -- python3 "$PORTING_SDK_DIR/scripts/audit_docs.py" \
        --root "$PORT_ROOT" \
        --surface "$PORT_ROOT/port_surface_go.json" \
        --ignore "$PORT_ROOT/DOC_AUDIT_IGNORE.md"

sched_gate SURFACE-DIFF res=surface desc="diff_port_surface vs python_surface.json" \
    -- python3 "$PORTING_SDK_DIR/scripts/diff_port_surface.py" \
        --reference "$PORTING_SDK_DIR/python_surface.json" \
        --port-surface "$PORT_ROOT/port_surface.json" \
        --omissions "$PORT_ROOT/PORT_OMISSIONS.md" \
        --additions "$PORT_ROOT/PORT_ADDITIONS.md"

sched_gate SWAIG-CLI desc="swaig-test shared mini-contract (verbs/serverless-reject/default-action)" \
    -- python3 "$PORTING_SDK_DIR/scripts/audit_swaig_cli_contract.py" \
        --port go \
        --cmd "go run ./cmd/swaig-test" \
        --require-url-model \
        --default-action-argv='--url|http://user:pass@127.0.0.1:1/' \
        --has-serverless \
        --serverless-argv='--url|http://user:pass@127.0.0.1:1/|--simulate-serverless|bogus-platform-xyz|--dump-swml'

# ---- §C1 doc/example/CLI execution gates ------------------------------------
# SNIPPET-COMPILE (typecheck doc code fences with the real SDK linked) + DOC-CLI
# (probe documented swaig-test invocations against the real CLI's parser) are
# cheap → cheap wave, blocking. EXAMPLES-RUN executes/loads the shipped examples
# (defer, blocking). SNIPPET-RUN is dynamic-ports-only; for go it self-skips
# (SNIPPET-COMPILE covers the compiled port) — wired report-only so the self-skip
# never fails the run.
sched_gate SNIPPET-COMPILE tier=nightly desc="documented code snippets compile against the real SDK" \
    -- python3 "$PORTING_SDK_DIR/scripts/snippet_compile.py" --port go --repo "$PORT_ROOT"

sched_gate DOC-CLI desc="documented swaig-test invocations parse against the real CLI" \
    -- python3 "$PORTING_SDK_DIR/scripts/doc_cli.py" --port go --repo "$PORT_ROOT"

# Wave-3 doc/API-truth gates — deterministic source/doc analysis (no build, no
# mock, ~1.3s for all six). Per-PR tier: cheap enough to catch doc/API drift at
# PR time rather than a day later in nightly.
sched_gate ERROR-ENVELOPE desc="REST error carries the full (status,body,url,method) envelope + raised on >=400" \
    -- python3 "$PORTING_SDK_DIR/scripts/error_envelope.py" --port go --repo "$PORT_ROOT"
sched_gate DEAD-PUBLIC-ERROR desc="exported error types are raised/caught/user-signalled (no dead error surface)" \
    -- python3 "$PORTING_SDK_DIR/scripts/dead_public_error.py" --port go --repo "$PORT_ROOT"
sched_gate PAGINATION-WIRED desc="shipped iterator-protocol paginator is wired into list()" \
    -- python3 "$PORTING_SDK_DIR/scripts/pagination_wired.py" --port go --repo "$PORT_ROOT"
sched_gate DOC-ENV desc="documented SIGNALWIRE_*/SWML_* env vars <=> code-read vars agree" \
    -- python3 "$PORTING_SDK_DIR/scripts/doc_env.py" --port go --repo "$PORT_ROOT"
sched_gate COUNT-CLAIM desc="numeric doc claims (skills/namespaces) match reality" \
    -- python3 "$PORTING_SDK_DIR/scripts/count_claim.py" --port go --repo "$PORT_ROOT"
sched_gate ACCESSOR-TRUTH desc="documented backtick method() refs exist in source" \
    -- python3 "$PORTING_SDK_DIR/scripts/accessor_truth.py" --port go --repo "$PORT_ROOT"

sched_gate EXAMPLES-RUN tier=nightly defer=1 desc="shipped examples load/compile (modulo EXAMPLES_RUN_ALLOW.md)" \
    -- python3 "$PORTING_SDK_DIR/scripts/examples_run.py" --port go --repo "$PORT_ROOT"

sched_gate SNIPPET-RUN tier=nightly defer=1 desc="dynamic-port doc snippets run to a zero exit (go: self-skips, SNIPPET-COMPILE covers it)" \
    -- python3 "$PORTING_SDK_DIR/scripts/snippet_run.py" --port go --repo "$PORT_ROOT" --report-only

# ---- §G anti-laundering ledger ----------------------------------------------
sched_gate SUPPRESSION-LEDGER res=dayone desc="no un-ledgered analyzer suppressions" \
    -- python3 "$PORTING_SDK_DIR/scripts/suppression_ledger.py" --port go --repo "$PORT_ROOT"

# ---- §D1 packaging ----------------------------------------------------------
sched_gate PACKAGE-SMOKE defer=1 desc="the real publishable module builds + imports from a clean env" \
    -- python3 "$PORTING_SDK_DIR/scripts/package_smoke.py" --port go --repo "$PORT_ROOT"

# ---- Day-one deterministic gates --------------------------------------------
# ARTIFACT-DENY uses the git ls-files PROXY (not --listing): go publishes no
# package artifact with an include/exclude manifest, so there is no authoritative
# package listing to feed. The proxy + ARTIFACT_DENY_ALLOW.md is the check.

sched_gate DOC-LANG-PURITY res=dayone desc="no python-verbatim docs in a non-python port" \
    -- python3 "$PORTING_SDK_DIR/scripts/doc_lang_purity.py" --port go --repo "$PORT_ROOT"

sched_gate DOC-LINKS res=dayone desc="every relative markdown link resolves to a tracked file" \
    -- python3 "$PORTING_SDK_DIR/scripts/doc_links.py" --port go --repo "$PORT_ROOT"

sched_gate README-INCLUDE res=dayone desc="doc code blocks are byte-identical to their gate-compiled fixture regions" \
    -- python3 "$PORTING_SDK_DIR/scripts/readme_include.py" --port go --repo "$PORT_ROOT"

sched_gate ROOT-HYGIENE res=dayone desc="no audit/scratch clutter tracked at repo root (allowlist ROOT_HYGIENE_ALLOW.md)" \
    -- python3 "$PORTING_SDK_DIR/scripts/root_hygiene.py" --port go --repo "$PORT_ROOT"

sched_gate IGNORE-LEDGER-VERIFY res=dayone desc="no laundered false-absence entries in DOC_AUDIT_IGNORE.md" \
    -- python3 "$PORTING_SDK_DIR/scripts/ignore_ledger_verify.py" --port go --repo "$PORT_ROOT"

sched_gate META-CONSISTENT res=dayone desc="package metadata consistency" \
    -- python3 "$PORTING_SDK_DIR/scripts/meta_consistent.py" --port go --repo "$PORT_ROOT"

sched_gate ARTIFACT-DENY res=dayone desc="no porting artifacts in the published package (git ls-files proxy + ARTIFACT_DENY_ALLOW.md)" \
    -- python3 "$PORTING_SDK_DIR/scripts/artifact_deny.py" --port go --repo "$PORT_ROOT"

# ---- expansion gates (GATE_EXPANSION_PLAN Tiers 5+) --------------------------
# Blocking; backlog burned to zero and the GEN-TYPE-DEGENERACY / ROUTE-COLLISION
# allowlists are user-approved (stamped 9cd5624). ROUTE-COLLISION builds go's
# route-registry itself (`go run ./cmd/route-registry`, its built-in REGISTRY_CMD
# — the same source the SPEC-PARITY gate uses). RELEASE-FRESH is report-only: go
# has no publish/release workflow, so there is no publish path to gate (a gap to
# flag, not a RED).

sched_gate GEN-TYPE-DEGENERACY res=dayone desc="generated types aren't degenerate (modulo GEN_TYPE_DEGENERACY_ALLOW.md)" \
    -- python3 "$PORTING_SDK_DIR/scripts/gen_type_degeneracy.py" --port go --repo "$PORT_ROOT"

sched_gate PUBLIC-JARGON res=dayone desc="no porting/internal jargon in the public API surface" \
    -- python3 "$PORTING_SDK_DIR/scripts/public_jargon.py" --port go --repo "$PORT_ROOT"

sched_gate ROUTE-COLLISION res=dayone desc="no route-split/crud-dup latent defects (modulo ROUTE_COLLISION_ALLOW.md)" \
    -- python3 "$PORTING_SDK_DIR/scripts/route_collision.py" --port go --repo "$PORT_ROOT"

sched_gate GEN-IDIOM res=dayone desc="generated code is not lint-excluded (idiomatic, gate-clean)" \
    -- python3 "$PORTING_SDK_DIR/scripts/gen_idiom.py" --port go --repo "$PORT_ROOT"

sched_gate RELEASE-FRESH res=dayone desc="release hygiene (report-only: go has no publish workflow to gate)" \
    -- python3 "$PORTING_SDK_DIR/scripts/release_fresh.py" --port go --repo "$PORT_ROOT" --report-only

# ---- summary ----------------------------------------------------------------

sched_run
rc=$?
if [ "$rc" -eq 0 ]; then
    echo "==> CI PASS"
else
    echo "==> CI FAIL (gates:$FAILED_GATES )"
fi
exit "$rc"
