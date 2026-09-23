#!/usr/bin/env bash
# run-actionlint.sh — canonical ACTIONLINT entry point for signalwire-go.
#
# THE way to run the ACTIONLINT gate for this port; run-ci, agents and humans call
# this rather than invoking porting-sdk's actionlint_gate.py (or the raw
# `actionlint` binary) directly. Self-bootstraps its tool environment exactly like
# run-lint.sh does for golangci-lint: installs the PINNED actionlint if absent
# locally, fails loud in CI, and refuses to run against a different version.
#
# WHY THE VERSION IS ENFORCED HERE
# --------------------------------
# actionlint adds checks regularly, so an unpinned version is a
# green-locally/red-in-CI generator: the gate's verdict changes with the calendar
# rather than with the workflow files. The workflows used to install
# `actionlint@latest`, meaning CI got the newest release at run time while a
# contributor ran whatever they had — a new release then failed this gate on an
# UNCHANGED workflow file. Both halves are now pinned to ACTIONLINT_VERSION
# (scripts/_env.sh), in lockstep with .github/workflows/{test,nightly}.yml, the
# same contract GOLANGCI_VERSION already had.
#
# Bumping actionlint means editing scripts/_env.sh AND both workflows together,
# with whatever findings the new version reports fixed in the same commit.
# SW_ALLOW_TOOL_VERSION_DRIFT=1 downgrades the version check to a warning, for a
# deliberate bump run only.
#
# Exits non-zero on any finding (or on a missing/mismatched binary).
set -euo pipefail

_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/_env.sh
source "$_SCRIPT_DIR/_env.sh"

ensure_actionlint || exit 1

# Resolve porting-sdk the same way run-ci.sh does: an explicit $PORTING_SDK wins,
# else the sibling checkout. run-ci exports PORTING_SDK_DIR, so a run-ci-driven
# invocation reuses its already-resolved path instead of re-deriving it.
if [ -z "${PORTING_SDK_DIR:-}" ]; then
    if [ -n "${PORTING_SDK:-}" ] && [ -d "$PORTING_SDK/scripts" ]; then
        PORTING_SDK_DIR="$PORTING_SDK"
    elif [ -d "$REPO/../porting-sdk/scripts" ]; then
        PORTING_SDK_DIR="$(cd "$REPO/../porting-sdk" && pwd)"
    else
        echo "FATAL: porting-sdk not found (set PORTING_SDK or clone it beside this repo)" >&2
        exit 2
    fi
fi

echo "==> ACTIONLINT (actionlint $ACTIONLINT_VERSION) — repo: $REPO"
exec python3 "$PORTING_SDK_DIR/scripts/actionlint_gate.py" --repo "$REPO"
