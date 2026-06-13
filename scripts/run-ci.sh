#!/usr/bin/env bash
# run-ci.sh — canonical local-and-CI gate runner for signalwire-go.
#
# Same script is invoked locally (`bash scripts/run-ci.sh`) AND by the
# GitHub Actions workflow. No drift between local and CI behavior.
#
# Gates (in order, fail-fast):
#   1. go test ./...                      — language test runner
#   2. signature regen                    — go run ./cmd/enumerate-signatures
#   3. drift gate                         — porting-sdk diff_port_signatures.py
#   4. surface-fresh gate                 — porting-sdk check_surface_freshness.py
#   5. no-cheat gate                      — porting-sdk audit_no_cheat_tests.py
#   6. emission gate                      — porting-sdk diff_port_emission.py
#   7. fmt gate                           — gofmt (local: auto-fix; CI: -l check)
#   8. lint gate                          — go vet ./...
#   9. doc-audit gate                     — porting-sdk audit_docs.py
#  10. surface-diff gate                  — porting-sdk diff_port_surface.py
#
# Each gate prints `[GATE-NAME] ... PASS` or `[GATE-NAME] ... FAIL: <reason>`
# Final line: `==> CI PASS` or `==> CI FAIL (gates: <list>)`.
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

# ---- gate plumbing ----------------------------------------------------------

FAILED_GATES=""

run_gate() {
    # run_gate <gate-name> <description> <command...>
    local name="$1"
    shift
    local description="$1"
    shift
    local logfile
    logfile="$(mktemp)"
    "$@" >"$logfile" 2>&1
    local rc=$?
    if [ "$rc" -eq 0 ]; then
        echo "[$name] $description ... PASS"
        rm -f "$logfile"
        return 0
    fi
    echo "[$name] $description ... FAIL: exit $rc"
    sed 's/^/    /' "$logfile" | tail -40
    rm -f "$logfile"
    FAILED_GATES="$FAILED_GATES $name"
    return $rc
}

# ---- per-port commands ------------------------------------------------------

cd "$PORT_ROOT"

echo "==> running CI gates for $PORT_NAME (porting-sdk at $PORTING_SDK_DIR)"

# Gate 1: language test runner
run_gate "TEST" "go test ./..." \
    go test ./...

# Gate 2: signature regen
run_gate "SIGNATURES" "regenerate port_signatures.json" \
    bash -c 'go run ./cmd/enumerate-signatures > port_signatures.json'

# Gate 3: drift gate (filtered drift must be 0)
run_gate "DRIFT" "diff_port_signatures vs python reference" \
    python3 "$PORTING_SDK_DIR/scripts/diff_port_signatures.py" \
        --reference "$PORTING_SDK_DIR/python_signatures.json" \
        --port-signatures "$PORT_ROOT/port_signatures.json" \
        --surface-omissions "$PORT_ROOT/PORT_OMISSIONS.md" \
        --surface-additions "$PORT_ROOT/PORT_ADDITIONS.md" \
        --omissions "$PORT_ROOT/PORT_SIGNATURE_OMISSIONS.md"

# Gate 4: surface-fresh gate — the committed cross-port port_surface.json must
# match a fresh regen (modulo the volatile generated_from git-sha). Closes the
# Layer-B-not-gated hole: DRIFT above only gates Layer A (signatures), so the
# surface could silently rot. `go run ./cmd/enumerate-surface` rewrites
# port_surface.json IN PLACE (default; not stdout) and also touches
# port_surface_go.json + port_additions_actual.json — we only *gate* the
# cross-port port_surface.json (the file diff_port_signatures.py's --surface*
# flags consume), but we restore all three the regen wrote so the gate is
# side-effect-free whether it passes or fails.
run_gate "SURFACE-FRESH" "check_surface_freshness vs committed port_surface.json" \
    bash -c '
        # Restore every file the regen rewrites, on ANY exit path (pass, fail,
        # or a broken enumerator), so the gate leaves no working-tree changes.
        trap "git checkout -- port_surface.json port_surface_go.json port_additions_actual.json 2>/dev/null" EXIT
        git show HEAD:port_surface.json > /tmp/committed_surface.json 2>/dev/null \
            || cp port_surface.json /tmp/committed_surface.json
        go run ./cmd/enumerate-surface || exit $?
        python3 "'"$PORTING_SDK_DIR"'/scripts/check_surface_freshness.py" \
            --committed /tmp/committed_surface.json \
            --fresh port_surface.json
    '

# Gate 5: no-cheat gate
run_gate "NO-CHEAT" "audit_no_cheat_tests" \
    python3 "$PORTING_SDK_DIR/scripts/audit_no_cheat_tests.py" --root "$PORT_ROOT"

# Gate 6: emission — byte-compare the SWAIG FunctionResult serialisation against
# Python's to_dict() over the shared 81-entry corpus. The drift gate (Gate 3)
# polices the SURFACE; this one polices the EMISSION (action shape/keys/values +
# the to_dict() envelope), the bug class the §6 sweep proved is otherwise drift-0
# and invisible to CI. Pure serialisation — no mock servers, no network; needs
# only signalwire-python adjacent (already required) + the emit-corpus program.
# The dump program is cmd/emit-corpus (go run ./cmd/emit-corpus). go was the
# emission PoC, so it carried the dump but was skipped in the 8-port gate rollout
# — this closes that gap so go's emission can't silently drift either.
run_gate "EMISSION" "diff_port_emission vs python to_dict() oracle" \
    python3 "$PORTING_SDK_DIR/scripts/diff_port_emission.py" \
        --port go \
        --port-repo "$PORT_ROOT"

# Gate 7: FMT — the language format gate. Canonical gate name is language-neutral
# (FMT); each port runs its own formatter under it. Here that is gofmt (Go's
# builtin, canonical formatter — no tool to install, no config to bikeshed,
# matches Rust's "formatter ships with the toolchain" shape). Source-style only
# — proven surface/emission-neutral (a gofmt reformat leaves port_signatures.json
# byte-identical modulo the git-sha provenance), so it can never move the audit.
#   * LOCAL ($CI unset)  → `gofmt -w .`: silently reformats your working tree, so
#     you never have to run gofmt by hand; surfaces a note if it changed files.
#   * CI ($CI=true)      → `gofmt -l .` must list nothing: read-only safety net
#     that FAILS if unformatted code reached CI (a committer who skipped run-ci).
# (goimports/golangci-lint are the deferred ADVISORY tier — they need a tool
# install + carry a backlog; gofmt + go vet are the zero-backlog day-1 floor.)
fmt_gate() {
    if [ -n "${CI:-}" ]; then
        local unformatted
        unformatted="$(gofmt -l .)"
        if [ -n "$unformatted" ]; then
            echo "unformatted files (run \`gofmt -w .\`):"
            echo "$unformatted"
            return 1
        fi
        return 0
    else
        gofmt -w .
        if ! git diff --quiet 2>/dev/null; then
            echo "    (FMT auto-applied formatting to your working tree — review & stage)"
        fi
        return 0
    fi
}
run_gate "FMT" "gofmt (local: auto-fix; CI: -l check)" fmt_gate

# Gate 8: LINT — the language lint gate (go: go vet). `go vet ./...` is the
# builtin static-analysis floor and exits 0 on this codebase today, so it gates
# blocking from day one with zero backlog. The TEST gate's `go test ./...` only
# vets test-bearing packages, so vet is NOT already guaranteed — making it
# explicit is a free, high-signal gate. (golangci-lint, which bundles
# staticcheck/errcheck/unused, is the deferred advisory tier: it needs a
# .golangci.yml excluding examples/ and will surface a backlog — burn down
# advisory, then promote, exactly as Rust did with clippy.)
run_gate "LINT" "go vet ./... (lint gate)" \
    go vet ./...

# Gate 9: DOC-AUDIT — every method/class referenced in docs/ + examples/ fenced
# code blocks must resolve to a real symbol in the port surface (catches
# phantom-API doc promises). Mirrors .github/workflows/doc-audit.yml exactly so
# there's no local/CI drift — previously this ran ONLY in that workflow, never
# under run-ci.sh, so a developer's local run was blind to doc drift. Uses the
# committed port_surface_go.json (the Go-shaped surface audit_docs consumes);
# the SURFACE-FRESH gate above already proved the surface is fresh.
run_gate "DOC-AUDIT" "audit_docs vs port_surface_go.json" \
    python3 "$PORTING_SDK_DIR/scripts/audit_docs.py" \
        --root "$PORT_ROOT" \
        --surface "$PORT_ROOT/port_surface_go.json" \
        --ignore "$PORT_ROOT/DOC_AUDIT_IGNORE.md"

# Gate 10: SURFACE-DIFF — diff the port surface against the Python reference
# (omissions/additions accounted for in PORT_OMISSIONS.md / PORT_ADDITIONS.md).
# SURFACE-FRESH (gate 4) only checks the committed surface MATCHES A REGEN; this
# checks it MATCHES PYTHON. Mirrors .github/workflows/surface-audit.yml — same
# no-drift reason as gate 9: it ran only in that workflow, not under run-ci.sh.
run_gate "SURFACE-DIFF" "diff_port_surface vs python_surface.json" \
    python3 "$PORTING_SDK_DIR/scripts/diff_port_surface.py" \
        --reference "$PORTING_SDK_DIR/python_surface.json" \
        --port-surface "$PORT_ROOT/port_surface.json" \
        --omissions "$PORT_ROOT/PORT_OMISSIONS.md" \
        --additions "$PORT_ROOT/PORT_ADDITIONS.md"

# ---- summary ----------------------------------------------------------------

if [ -z "$FAILED_GATES" ]; then
    echo "==> CI PASS"
    exit 0
else
    echo "==> CI FAIL (gates:$FAILED_GATES )"
    exit 1
fi
