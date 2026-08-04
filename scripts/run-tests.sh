#!/usr/bin/env bash
# run-tests.sh — canonical TEST entry point for signalwire-go (go test ./...).
#
# THE way to run this port's tests (porting-sdk/RUN_LINT_FORMAT_SPEC.md); run-ci,
# agents, humans call this — not `go test` directly. Self-bootstraps its tool
# environment and operates from the module root regardless of caller CWD.
#
# Runs the full suite (`go test ./...`) and exits non-zero on any failure.
#
# Optional filter passthrough: any args are forwarded to `go test`, so a caller
# can run a subset, e.g.
#     bash scripts/run-tests.sh -run TestFoo ./pkg/agent/...
#     bash scripts/run-tests.sh ./pkg/swml/...
# With no args the default target is the whole module (./...).
#
# ---------------------------------------------------------------------------
# PROOF OF EXECUTION (the reason this script is more than a one-line wrapper)
# ---------------------------------------------------------------------------
# `go test ./...` exits 0 for two very different outcomes, and only one of them
# ran a test:
#   * the package's test binary EXECUTED           → `ok  <pkg>  7.569s`
#   * the result was REPLAYED FROM THE TEST CACHE  → `ok  <pkg>  (cached)`
# Exit status cannot tell them apart. `go test` caches successful package results
# keyed on the test binary + inputs, so a TEST gate that only checks `$?` can
# report PASS having executed NOTHING.
#
# MEASURED in this repo (2026-08-03, 8-core Apple Silicon): a second consecutive
# `bash scripts/run-tests.sh` over an unchanged tree reported ALL 22 test-bearing
# packages `(cached)`, finished in 0.66s, and exited 0. Zero tests ran. A first
# run after touching one package still reported 19 of 22 `(cached)`.
#
# That matters most exactly when it is least obvious: a commit that changes
# GENERATED code (the SWAIG/relay regen waves) can be handed a replayed green, so
# the gate's PASS is precisely zero evidence that the regenerated code compiles
# and passes.
#
# THE MECHANISM, and why this one:
# We parse the per-package summary lines and require that at least one
# test-bearing package actually EXECUTED; if a real run was expected and every
# package came back `(cached)`, the gate FAILS instead of reporting a hollow PASS.
#
# `(cached)` is DOCUMENTED CONTRACT, not console decoration. From `go help test`:
#   "When the result of a test can be recovered from the cache, go test will
#    redisplay the previous output instead of running the test binary again. When
#    this happens, go test prints '(cached)' in place of the elapsed time in the
#    summary line."
# This is the opposite of the Gradle situation (where `> Task :test FROM-CACHE`
# is console formatting that varies with log level and --console mode, and was
# rightly rejected as a signal). Here the marker is a specified field of the
# summary line.
#
# Alternatives considered and REJECTED:
#   * `go test -json` — CANNOT distinguish cached from executed. Verified: a
#     cached run replays the identical start/run/output/pass event stream, with
#     no Action, no boolean, and no Elapsed difference. The ONLY cache signal
#     anywhere in the JSON stream is the `(cached)` TEXT summary line wrapped
#     inside an `output` event. So the structured format is strictly less
#     informative than the summary line, not more.
#   * An EXECUTION RECEIPT written from the test binary (the mechanism the java
#     port uses, where a Gradle doLast is a first-class execution signal). Go has
#     no task-action hook; the nearest equivalent is TestMain, which does not
#     exist anywhere in this repo (0 of 136 test files) and would have to be
#     added to all 22 test-bearing packages. And it would prove nothing extra: a
#     cached package never runs its binary, so no receipt appears — i.e. the
#     receipt would carry exactly the same per-package "did it run" bit that
#     `(cached)` already reports, bought with 22 new TestMain funcs and a
#     side-effecting file write from inside every test binary.
#   * Forcing `-count=1` on every invocation — the blunt instrument. It defeats
#     result caching outright, so the gate can never be fooled, but it also
#     removes the reuse unconditionally and, more importantly, it HIDES rather
#     than reports the condition: the gate would silently re-run instead of
#     telling anyone the previous result was a replay. It also spends the full
#     suite cost on every one of run-ci's invocations. Available on demand via
#     the passthrough (`bash scripts/run-tests.sh -count=1`) and already used
#     deliberately by the STRICT-MOCKS lane in run-ci.sh, but not imposed here.
#
# The Go BUILD cache is left entirely alone, and so is the TEST cache itself.
# Compile/result reuse is a legitimate and large speedup; the defect was never
# that the cache exists — it was a GATE treating a cache hit as evidence of
# EXECUTION. The ordinary path stays cached; it just has to say so instead of
# claiming a pass.
#
# COST, stated plainly (measured 2026-08-03, 8-core Apple Silicon, 4 sibling
# agents active):
#   * The assertion itself costs NOTHING measurable — it forces no flag and adds
#     no test work. It tees the output it was already producing and greps the
#     summary lines. The default invocation is byte-for-byte the `go test`
#     command it always was.
#   * A genuine full run is ~40s (39.6s with -count=1, 22 packages). A replayed
#     no-op "pass" was 0.66s. That ~39s is NOT a new cost this assertion adds —
#     it is the cost of actually testing, which the gate was previously skipping
#     while reporting success.
#   * The BEHAVIOUR CHANGE the owner should weigh: a SECOND consecutive run over
#     an UNCHANGED tree now EXITS 1 ("nothing executed") where it used to print a
#     hollow PASS. That is deliberate — the alternative is a gate that cannot
#     fail — but a local re-run of run-ci with no edits in between now reports
#     red on TEST. The remedy is in the failure message (`-count=1`), and any
#     real edit to a package's inputs makes it execute normally with no flag.
#   * A FILTERED/subset invocation is exempt from the all-cached assertion (see
#     below): callers legitimately re-run one package repeatedly, and failing
#     that would make the passthrough hostile to use.

# shellcheck source=scripts/_env.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_env.sh"

# _env.sh sets `set -euo pipefail`. We must NOT let a nonzero `go test` abort the
# script before the summary analysis runs, and we must not let a grep with no
# match kill it either — so the run's status is captured explicitly and every
# grep/wc is guarded.

TEST_ARGS=()
FILTERED=""
if [ "$#" -gt 0 ]; then
    # Caller supplied args (a -run pattern and/or package paths) — forward as-is.
    # If they passed only flags (no package path) go test defaults to the current
    # dir; we're cd'd to the module root so add ./... unless a package/path is set.
    FILTERED="1"
    has_pkg=""
    for a in "$@"; do
        case "$a" in
            ./*|/*|*/...) has_pkg="1" ;;
        esac
    done
    TEST_ARGS=("$@")
    if [ -z "$has_pkg" ]; then
        TEST_ARGS+=(./...)
    fi
else
    TEST_ARGS=(./...)
fi

# Scratch log for the summary-line analysis. Kept REPO-LOCAL (never a shared
# global temp) so concurrent runs in other checkouts cannot collide, and removed
# on exit.
mkdir -p "$REPO/.sw-tmp"
LOG="$(mktemp "$REPO/.sw-tmp/go-test.XXXXXX")"
trap 'rm -f "$LOG"' EXIT

echo "==> TEST (go test ${TEST_ARGS[*]}) — module: $REPO"

set +e
go test "${TEST_ARGS[@]}" 2>&1 | tee "$LOG"
GO_TEST_RC="${PIPESTATUS[0]}"
set -e

if [ "$GO_TEST_RC" -ne 0 ]; then
    echo "==> TEST FAILED (go test exit $GO_TEST_RC)" >&2
    exit "$GO_TEST_RC"
fi

# ---- proof-of-execution analysis --------------------------------------------
# Summary-line shapes (documented in `go help test`):
#   ok    <pkg>  1.234s     → executed
#   ok    <pkg>  (cached)   → replayed from the test cache, NOTHING ran
#   ?     <pkg>  [no test files]
CACHED_PKGS="$(grep -c '^ok .*(cached)' "$LOG" || true)"
EXECUTED_PKGS="$(grep '^ok ' "$LOG" | grep -vc '(cached)' || true)"
CACHED_PKGS="${CACHED_PKGS:-0}"
EXECUTED_PKGS="${EXECUTED_PKGS:-0}"
TESTED_PKGS=$((CACHED_PKGS + EXECUTED_PKGS))

echo "==> TEST packages: $EXECUTED_PKGS executed, $CACHED_PKGS replayed from cache (of $TESTED_PKGS test-bearing)"

if [ "$TESTED_PKGS" -eq 0 ]; then
    echo "" >&2
    echo "FATAL: go test reported success but NO test-bearing package appeared in its" >&2
    echo "       output at all. A suite that selects nothing cannot pass." >&2
    if [ -n "$FILTERED" ]; then
        echo "       Check the filter: ${TEST_ARGS[*]}" >&2
    fi
    exit 1
fi

if [ "$EXECUTED_PKGS" -eq 0 ]; then
    if [ -n "$FILTERED" ]; then
        # A subset invocation is a developer/agent tool: re-running one package
        # repeatedly is normal and legitimate, so this is a loud warning, not a
        # failure. The gate path (run-ci) always calls this script bare.
        echo "" >&2
        echo "WARNING: all $CACHED_PKGS package(s) were replayed from the test cache —" >&2
        echo "         no test actually ran. This filtered invocation is NOT failed for" >&2
        echo "         it, but this run is no evidence the code under test passes." >&2
        echo "         Force a real run with: -count=1" >&2
        echo "" >&2
        # Do NOT fall through to the "executed for real" line below: nothing ran.
        exit 0
    else
        echo "" >&2
        echo "FATAL: the test suite did NOT EXECUTE — go test reported success by" >&2
        echo "       replaying all $CACHED_PKGS package result(s) from its test cache" >&2
        echo "       ('(cached)' on every summary line). NO test ran, so this run is" >&2
        echo "       ZERO evidence that the suite passes." >&2
        echo "" >&2
        echo "       This is not a failure to work around: it means no package's inputs" >&2
        echo "       have changed since the last real run. If you need a genuine run" >&2
        echo "       anyway (verifying regenerated code, chasing a stale result):" >&2
        echo "         bash scripts/run-tests.sh -count=1" >&2
        echo "" >&2
        exit 1
    fi
fi

echo "==> TEST executed for real ($EXECUTED_PKGS package(s) ran)"
