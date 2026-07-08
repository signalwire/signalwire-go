#!/usr/bin/env bash
# run-format.sh — canonical FMT entry point for signalwire-go (gofmt).
#
# This is THE way to format this port. run-ci, agents, humans all call this — not
# `gofmt` directly (porting-sdk/RUN_LINT_FORMAT_SPEC.md). Self-bootstraps its tool
# environment and operates from the module root regardless of caller CWD.
#
# Modes:
#   (default, local)  APPLY   — `gofmt -w .` reformats the tree in place; exit 0
#                               even if it changed files.
#   --check   (CI)    VERIFY  — `gofmt -l .` lists nothing; exit non-zero if any
#                               file is unformatted. Modifies nothing.
#
# Formats BOTH hand-written and generated Go (the generated tree is gofmt-clean by
# construction, so --check stays green). Idempotent: a 2nd apply run is a no-op.
#
# Scope: the port's OWN tracked Go tree (`git ls-files '*.go'`), NOT a bare
# `gofmt .`. A bare `.` walks gitignored scratch — notably `.sw-tmp/`, where a
# concurrent run-ci gate (snippet-compile / DOC-AUDIT) drops deliberately-partial
# doc-snippet .go fragments that are not valid standalone Go. gofmt errors on
# those, which under the concurrent scheduler intermittently reddens FMT. Scoping
# to tracked files makes FMT deterministic and gitignore-respecting.

# shellcheck source=scripts/_env.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_env.sh"

MODE="apply"
if [ "${1:-}" = "--check" ]; then
    MODE="check"
fi

# Tracked .go files, NUL-delimited (paths are safe but keep the idiom robust).
# Fall back to `.` only if git is unavailable (never in run-ci / CI).
go_files() {
    if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
        git ls-files -z '*.go'
    else
        find . -path ./.sw-tmp -prune -o -name '*.go' -print0
    fi
}

if [ "$MODE" = "check" ]; then
    unformatted="$(go_files | xargs -0 gofmt -l)"
    if [ -n "$unformatted" ]; then
        echo "unformatted files (run \`bash scripts/run-format.sh\`):" >&2
        echo "$unformatted" >&2
        exit 1
    fi
    echo "gofmt --check: all files formatted."
    exit 0
else
    go_files | xargs -0 gofmt -w
    if command -v git >/dev/null 2>&1 && ! git diff --quiet 2>/dev/null; then
        echo "    (gofmt applied formatting to your working tree — review & stage)"
    fi
    echo "gofmt: applied."
    exit 0
fi
