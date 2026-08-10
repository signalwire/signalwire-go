#!/usr/bin/env bash
# _env.sh — shared, CWD-independent tool-environment bootstrap for signalwire-go.
#
# Sourced by scripts/run-format.sh, scripts/run-lint.sh, scripts/run-tests.sh
# (and available to run-ci.sh) so the FMT/LINT/TEST tool environment lives in ONE
# place and works no matter the caller's CWD or shell setup. This is the fix for
# the "works in run-ci, `command not found` everywhere else" class of false
# failures (see porting-sdk/RUN_LINT_FORMAT_SPEC.md).
#
# Contract for callers: `source` this file (do not exec it). After sourcing:
#   * $REPO           — absolute module root (dir containing go.mod)
#   * you are cd'd into $REPO (operate from the module root regardless of caller CWD)
#   * `go` is resolvable, or we've already failed loud with an install hint
#   * ensure_golangci — function that guarantees golangci-lint is on PATH (installs
#                       the pinned version locally if absent; fails loud in CI)
#
# Idempotent: sourcing twice is a no-op beyond re-cd'ing to $REPO.

set -euo pipefail

# ---- resolve the module root from THIS script's own path (CWD-independent) ---
# $REPO is the dir that contains go.mod; scripts/ live directly under it.
_ENV_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(dirname "$_ENV_DIR")"
export REPO

if [ ! -f "$REPO/go.mod" ]; then
    echo "FATAL: $REPO has no go.mod — _env.sh could not resolve the Go module root" >&2
    exit 2
fi

# Operate from the module root no matter where the caller invoked us.
cd "$REPO"

# ---- ensure the Go toolchain is resolvable ----------------------------------
# Put the standard Go install locations + GOPATH/bin on PATH so `go` and
# go-installed tools (golangci-lint) resolve from a bare login shell too.
for _gobin in /opt/homebrew/bin /usr/local/go/bin /usr/local/bin "$HOME/go/bin"; do
    case ":$PATH:" in
        *":$_gobin:"*) ;;                       # already present
        *) [ -d "$_gobin" ] && PATH="$_gobin:$PATH" ;;
    esac
done
export PATH

if ! command -v go >/dev/null 2>&1; then
    echo "FATAL: 'go' not found on PATH." >&2
    echo "       Install the Go toolchain (https://go.dev/dl/) and re-run." >&2
    echo "       On macOS: brew install go" >&2
    exit 127
fi

# GOPATH/bin is where `go install` drops tool binaries (golangci-lint); ensure
# it's on PATH for the rest of this process.
_GOPATH_BIN="$(go env GOPATH)/bin"
case ":$PATH:" in
    *":$_GOPATH_BIN:"*) ;;
    *) PATH="$_GOPATH_BIN:$PATH"; export PATH ;;
esac

# ---- golangci-lint self-bootstrap -------------------------------------------
# Pinned dev dependency — keep in lockstep with run-ci.sh's GOLANGCI_VERSION and
# .github/workflows/test.yml's golangci-lint-action version. Locally we install
# the pinned version if absent; in CI the workflow installs it and we fail loud
# rather than silently install a possibly-mismatched build.
GOLANGCI_VERSION="v2.12.2"
export GOLANGCI_VERSION

ensure_golangci() {
    command -v golangci-lint >/dev/null 2>&1 && return 0
    if [ -n "${CI:-}" ]; then
        echo "golangci-lint not found in CI — the workflow must install it (golangci-lint-action)" >&2
        return 1
    fi
    echo "    (golangci-lint $GOLANGCI_VERSION missing — installing the pinned dev dependency...)" >&2
    go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$GOLANGCI_VERSION" || {
        echo "    golangci-lint install failed — install it manually:" >&2
        echo "      go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$GOLANGCI_VERSION" >&2
        return 1
    }
    command -v golangci-lint >/dev/null 2>&1
}

# ---- actionlint self-bootstrap ----------------------------------------------
# Pinned dev dependency, same contract as golangci-lint above — keep in lockstep
# with .github/workflows/{test,nightly}.yml's `actionlint/cmd/actionlint@v…`.
#
# The workflows previously installed `actionlint@latest`, which is a
# green-locally/red-in-CI generator: CI resolves the newest release at run time
# while local dev runs whatever it has, so a new actionlint (they add checks
# regularly) fails the ACTIONLINT gate on an UNCHANGED workflow file. Pinning both
# halves means the gate's verdict changes only when this version does.
ACTIONLINT_VERSION="v1.7.12"
export ACTIONLINT_VERSION

ensure_actionlint() {
    local have
    if command -v actionlint >/dev/null 2>&1; then
        # `actionlint --version` prints the bare number when installed from a
        # release archive/brew ("1.7.12") but a v-prefixed tag when built from
        # source via `go install` ("v1.7.12"). Compare on the extracted x.y.z so
        # both spellings of the SAME version match — a naive prefix compare made
        # the go-installed binary look mismatched and reinstalled it every run.
        have="$(actionlint --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)"
        if [ "$have" = "${ACTIONLINT_VERSION#v}" ]; then
            return 0
        fi
        if [ "${SW_ALLOW_TOOL_VERSION_DRIFT:-0}" = "1" ]; then
            echo "    (actionlint is '${have:-unknown}', not the pinned $ACTIONLINT_VERSION — drift allowed)" >&2
            return 0
        fi
        if [ -n "${CI:-}" ]; then
            echo "actionlint in CI is '${have:-unknown}', not the pinned $ACTIONLINT_VERSION" >&2
            echo "  the workflow must install the pin; a different version changes the gate verdict" >&2
            return 1
        fi
        echo "    (actionlint is '${have:-unknown}', not the pinned $ACTIONLINT_VERSION — installing the pin...)" >&2
    elif [ -n "${CI:-}" ]; then
        echo "actionlint not found in CI — the workflow must install it" >&2
        return 1
    else
        echo "    (actionlint $ACTIONLINT_VERSION missing — installing the pinned dev dependency...)" >&2
    fi
    go install "github.com/rhysd/actionlint/cmd/actionlint@$ACTIONLINT_VERSION" || {
        echo "    actionlint install failed — install it manually:" >&2
        echo "      go install github.com/rhysd/actionlint/cmd/actionlint@$ACTIONLINT_VERSION" >&2
        return 1
    }
    command -v actionlint >/dev/null 2>&1
}
