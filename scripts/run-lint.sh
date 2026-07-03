#!/usr/bin/env bash
# run-lint.sh — canonical LINT entry point for signalwire-go (go vet + golangci-lint).
#
# THE way to lint this port (porting-sdk/RUN_LINT_FORMAT_SPEC.md); run-ci, agents,
# humans call this — not `go vet` / `golangci-lint` directly. Self-bootstraps its
# tool environment (installs the pinned golangci-lint if absent locally; fails loud
# in CI) and operates from the module root regardless of caller CWD.
#
# Two layers, both blocking:
#   1. go vet ./...            — builtin static-analysis floor (always available)
#   2. golangci-lint run       — deep linter set governed by .golangci.yml
#
# Exits non-zero on any finding.
#
# Optional: --fix passes golangci-lint's autofix (go vet has no autofix, so it
# still runs report-only first). Anything after --fix (or the whole argv when the
# first arg isn't --fix) is forwarded to golangci-lint.

# shellcheck source=scripts/_env.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_env.sh"

FIX=""
if [ "${1:-}" = "--fix" ]; then
    FIX="--fix"
    shift
fi

go vet ./... || exit 1

ensure_golangci || exit 1

golangci-lint run --config "$REPO/.golangci.yml" \
    --max-same-issues 0 --max-issues-per-linter 0 \
    ${FIX:+$FIX} "$@" ./... || exit 1

echo "lint: go vet + golangci-lint clean."
