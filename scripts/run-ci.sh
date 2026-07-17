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
# checks the journal for BOTH coverage AND wire-truth (STRICT-MOCKS §2.2a: any
# journaled wire_violation reds the gate — respelling-proof, since it reads the
# mock's own spec-vs-wire judgement).
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
    # -run Gen_: drive ONLY the generated *Gen_* wire-coverage suite (every generated
    # test func name contains "Gen_", e.g. TestRelayRestGen_Addresses_Create) against
    # the mock to populate the coverage journal. The hand-authored *_mock_test.go
    # files (small_namespaces_mock_test.go, registry_mock_test.go, fabric_mock_test.go,
    # pagination_mock_test.go, paginate_method_mock_test.go, …) still run under the
    # plain TEST gate for their own assertions — they're excluded here so the two
    # owner-parked spec gaps they legitimately exercise (recordings page_size, fabric
    # cursor pagination replay) don't land in the journal the STRICT-MOCKS post-pass
    # below checks. This mirrors python's REST-COVERAGE `-k "Wire and not
    # wire_regression_pins"` selector.
    MOCK_SIGNALWIRE_PORT="$port" go test "$PORT_ROOT/pkg/rest/..." -run 'Gen_' -p 1 -count=1 || return 1
    python3 -m mock_signalwire.rest_coverage \
        --mock-url "http://127.0.0.1:$port" \
        --spec-root "$PORTING_SDK_DIR/rest-apis" \
        --allowlist "$PORTING_SDK_DIR/REST_COVERAGE_BASELINE.md" \
        --allowlist "$PORT_ROOT/REST_COVERAGE_GAPS.md" \
        --gap-baseline "$PORTING_SDK_DIR/REST_COVERAGE_GAP_BASELINE.md" || return 1
    # STRICT-MOCKS §2.2a — fail the gate on ANY journaled wire_violation. The shared
    # helper reads the same live mock journal and exits non-zero on any offender (see
    # porting-sdk/scripts/assert_no_wire_violations.py). WIRE_VIOLATIONS_ALLOW.md is
    # currently empty (no signed exceptions) — the two owner-parked spec gaps
    # (recordings page_size, fabric cursor pagination replay) are excluded from this
    # journal at the source (the -run Gen_ selector above), not allowlisted.
    python3 "$PORTING_SDK_DIR/scripts/assert_no_wire_violations.py" \
        --rest-mock-url "http://127.0.0.1:$port" \
        --allowlist "$PORT_ROOT/WIRE_VIOLATIONS_ALLOW.md"
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

# STRICT-MOCKS §2.2b — run the full suite against a strict mock_relay
# (MOCK_RELAY_STRICT=1): any unknown RELAY frame field / duplicate command-id is
# rejected with an error frame instead of being tolerantly journaled, so a wrong
# RELAY wire shape fails the test rather than being silently accepted. go's RELAY
# suite is already wire-clean under strict (see the STRICT-MOCKS gate below, which
# re-runs pkg/relay/... in isolation under the same flag for a full-suite nightly
# regression floor).
sched_gate TEST defer=1 desc="go test ./... (scripts/run-tests.sh) (STRICT-MOCKS: MOCK_RELAY_STRICT=1)" \
    -- env MOCK_RELAY_STRICT=1 bash "$PORT_ROOT/scripts/run-tests.sh"

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

# DOC-WIRE (§2.1) — the wire SHAPE emitted by the doc examples must be spec-clean.
# doc_wire.py spawns the mock, points the runner at it via SIGNALWIRE_MOCK_URL, and
# reads the mock's wire_violations journal (over HTTP; the runner's stdout is
# irrelevant, only its exit code). The runner is a go test that replays the doc
# examples' untyped-map wire calls (map[string]any bodies + phone-search params —
# the shapes SNIPPET-COMPILE's type checker can't validate for key correctness)
# against the mock. Per-PR (cheap). go's red list is empty (its wire keys already
# match the spec: `areacode`, not `area_code`).
sched_gate DOC-WIRE desc="doc-example wire shapes emit no unknown-field/dup-id violations against the strict mock" \
    -- python3 "$PORTING_SDK_DIR/scripts/doc_wire.py" --port go --repo "$PORT_ROOT" \
        --runner "go test -count=1 -run TestDocWireFixtures ./pkg/rest/namespaces/"

# STATUS-CLAIM (§2.3) — doc status phrases ("not implemented", "no … adapter",
# "transport pending", …) must match shipped reality. Per-PR (cheap, deterministic
# doc/source scan) so a false status claim is caught at PR time. go's initial red
# list (the cloud_functions_guide GCF-denial, README Azure over-claim, simulate.go
# stale not-implemented list) was burned in this PR.
sched_gate STATUS-CLAIM desc="doc status claims (not-implemented/adapter/pending) match shipped reality" \
    -- python3 "$PORTING_SDK_DIR/scripts/status_claim.py" --port go --repo "$PORT_ROOT" \
        --surface "$PORT_ROOT/port_surface.json"

sched_gate EXAMPLES-RUN tier=nightly defer=1 desc="shipped examples load/compile (modulo EXAMPLES_RUN_ALLOW.md)" \
    -- python3 "$PORTING_SDK_DIR/scripts/examples_run.py" --port go --repo "$PORT_ROOT"

sched_gate SNIPPET-RUN tier=nightly defer=1 desc="dynamic-port doc snippets run to a zero exit (go: self-skips, SNIPPET-COMPILE covers it)" \
    -- python3 "$PORTING_SDK_DIR/scripts/snippet_run.py" --port go --repo "$PORT_ROOT" --report-only

# WAIT-LIVENESS (§2.4) — the RELAY Action.Wait() liveness contract: wait() BLOCKS
# until the deferred completing event arrives, then returns with the finished
# state (never a no-op that returns at t~=0, never a hang). cmd/wait-liveness-dump
# spawns a real mock_relay, arms a deferred completing event, drives Action.Wait,
# and emits the liveness classification; the differ compares it to the python
# golden. Real-time behavioral check → tier=nightly (deferred behind the cheap
# wave). Regression floor: no red list expected once wired.
sched_gate WAIT-LIVENESS tier=nightly defer=1 desc="RELAY Action.Wait() blocks-until-event liveness matches the python golden" \
    -- python3 "$PORTING_SDK_DIR/scripts/diff_port_wait_liveness.py" --port go \
        --python-sdk "$PYTHON_SDK_DIR" \
        --dump-cmd "go run ./cmd/wait-liveness-dump"

# STRICT-MOCKS (§2.2) — re-run the RELAY suite with the mock in STRICT mode
# (MOCK_RELAY_STRICT=1: the mock 400s an unknown field or a duplicate id instead
# of tolerantly journaling it), so a wire-shape regression the tolerant mock would
# swallow fails loud. go's RELAY tests pass clean under strict today (empty red
# list). tier=nightly (a full second suite pass is heavy) + defer.
sched_gate STRICT-MOCKS tier=nightly defer=1 desc="RELAY suite passes with the mock in 400-on-violation strict mode (MOCK_RELAY_STRICT=1)" \
    -- env MOCK_RELAY_STRICT=1 bash "$PORT_ROOT/scripts/run-tests.sh" -count=1 ./pkg/relay/...

# ---- §G anti-laundering ledger ----------------------------------------------
sched_gate SUPPRESSION-LEDGER res=dayone desc="no un-ledgered analyzer suppressions" \
    -- python3 "$PORTING_SDK_DIR/scripts/suppression_ledger.py" --port go --repo "$PORT_ROOT"

# ---- §D1 packaging ----------------------------------------------------------
sched_gate PACKAGE-SMOKE tier=nightly defer=1 desc="the real publishable module builds + imports from a clean env" \
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

sched_gate IGNORE-LEDGER-VERIFY res=dayone desc="no laundered false-absence entries in DOC_AUDIT_IGNORE.md (strict: reason/approver/date required)" \
    -- python3 "$PORTING_SDK_DIR/scripts/ignore_ledger_verify.py" --port go --repo "$PORT_ROOT" --require-fields

sched_gate META-CONSISTENT tier=nightly res=dayone desc="package metadata consistency" \
    -- python3 "$PORTING_SDK_DIR/scripts/meta_consistent.py" --port go --repo "$PORT_ROOT"

sched_gate ARTIFACT-DENY res=dayone desc="no porting artifacts in the published package (git ls-files proxy + ARTIFACT_DENY_ALLOW.md)" \
    -- python3 "$PORTING_SDK_DIR/scripts/artifact_deny.py" --port go --repo "$PORT_ROOT"

# ---- expansion gates (GATE_EXPANSION_PLAN Tiers 5+) --------------------------
# Blocking; backlog burned to zero and the GEN-TYPE-DEGENERACY / ROUTE-COLLISION
# allowlists are user-approved (stamped 9cd5624). ROUTE-COLLISION builds go's
# route-registry itself (`go run ./cmd/route-registry`, its built-in REGISTRY_CMD
# — the same source the SPEC-PARITY gate uses). RELEASE-FRESH is BLOCKING: go now
# ships a gated publish workflow (.github/workflows/publish.yml runs run-ci.sh
# before the release step), so the publish path is gated.

sched_gate GEN-TYPE-DEGENERACY res=dayone desc="generated types aren't degenerate (modulo GEN_TYPE_DEGENERACY_ALLOW.md)" \
    -- python3 "$PORTING_SDK_DIR/scripts/gen_type_degeneracy.py" --port go --repo "$PORT_ROOT"

sched_gate PUBLIC-JARGON res=dayone desc="no porting/internal jargon in the public API surface" \
    -- python3 "$PORTING_SDK_DIR/scripts/public_jargon.py" --port go --repo "$PORT_ROOT"

sched_gate ROUTE-COLLISION res=dayone desc="no route-split/crud-dup latent defects (modulo ROUTE_COLLISION_ALLOW.md)" \
    -- python3 "$PORTING_SDK_DIR/scripts/route_collision.py" --port go --repo "$PORT_ROOT"

sched_gate GEN-IDIOM res=dayone desc="generated code is not lint-excluded (idiomatic, gate-clean)" \
    -- python3 "$PORTING_SDK_DIR/scripts/gen_idiom.py" --port go --repo "$PORT_ROOT"

sched_gate RELEASE-FRESH res=dayone desc="release hygiene: publish path must run run-ci before publishing (blocking)" \
    -- python3 "$PORTING_SDK_DIR/scripts/release_fresh.py" --port go --repo "$PORT_ROOT"

# SEMVER-DIFF (§D3) — the public API surface change since the release FLOOR must
# match the version bump. go has no in-tree version file (the git tag is the
# version), so the floor is the committed port_signatures.baseline.json
# (baseline_version 3.0.2). For a tag-versioned port there is nothing in-tree to
# bump, so this gate is a RELEASER NOTE, not a per-PR block: it reports the bump
# the next tag needs. It runs --report-only for exactly this reason — and that is
# also why it does NOT participate in the PART-D wave-A blocking flip: the two
# wave-A findings on go's SEMVER path are (1) "baseline needs re-anchoring to a
# release tag" — un-fixable until a v3.x tag actually exists (go's only tag is
# v1.1.0; there is no v3.0.2 tag to anchor to yet), and (2) the additions-since-
# baseline bump note. Both are release-time decisions for a human, not a code
# defect a PR can fix, so blocking them would wedge every PR on an unreleasable
# baseline. CURRENT NOTE (surface today vs the 3.0.2 baseline): the Messages +
# Projects namespaces add 7 members → the next tag needs at least a MINOR bump
# (3.1.0). See the PR body's "/v3 module path" + version discussion.
sched_gate SEMVER-DIFF res=dayone deps=SIGNATURES desc="API surface change since the release floor matches the version bump (releaser note; report-only for a tag-versioned port)" \
    -- python3 "$PORTING_SDK_DIR/scripts/semver_diff.py" --port go --repo "$PORT_ROOT" --report-only

# ---- summary ----------------------------------------------------------------

sched_run
rc=$?
if [ "$rc" -eq 0 ]; then
    echo "==> CI PASS"
else
    echo "==> CI FAIL (gates:$FAILED_GATES )"
fi
exit "$rc"
