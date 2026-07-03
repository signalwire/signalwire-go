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

# shellcheck source=scripts/_env.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_env.sh"

if [ "$#" -gt 0 ]; then
    # Caller supplied args (a -run pattern and/or package paths) — forward as-is.
    # If they passed only flags (no package path) go test defaults to the current
    # dir; we're cd'd to the module root so add ./... unless a package/path is set.
    has_pkg=""
    for a in "$@"; do
        case "$a" in
            ./*|/*|*/...) has_pkg="1" ;;
        esac
    done
    if [ -n "$has_pkg" ]; then
        exec go test "$@"
    else
        exec go test "$@" ./...
    fi
else
    exec go test ./...
fi
