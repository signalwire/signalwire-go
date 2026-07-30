#!/usr/bin/env bash
# run-lint.sh — canonical LINT entry point for signalwire-go (go vet + golangci-lint).
#
# THE way to lint this port (porting-sdk/RUN_LINT_FORMAT_SPEC.md); run-ci, agents,
# humans call this — not `go vet` / `golangci-lint` directly. Self-bootstraps its
# tool environment (installs the pinned golangci-lint if absent locally; fails loud
# in CI) and operates from the module root regardless of caller CWD.
#
# Three layers, all blocking:
#   1. go vet ./...            — builtin static-analysis floor (always available)
#   2. golangci-lint run       — deep linter set governed by .golangci.yml
#   3. the EXAMPLE trees       — see "example coverage" below
#
# Exits non-zero on any finding.
#
# ---- example coverage (why layer 3 exists) ----------------------------------
# Every file under examples/, rest/examples/ and relay/examples/ is a standalone
# `package main` demo carrying a `//go:build swexample` tag, so `go build ./...`
# (and therefore `./...` in layers 1-2) skips them. They are shipping code and
# are linted at the SAME bar as the library — no relaxed ruleset, same
# .golangci.yml.
#
# Two mechanics are load-bearing here:
#
#   * The tag is `swexample`, NOT `ignore`. `ignore` cannot be passed to
#     -tags/--build-tags at all: the Go standard library uses `//go:build ignore`
#     on its own generator programs (math/rand/gen_cooked.go, strconv/pow10gen.go,
#     crypto/internal/fips140/aes/ctr_arm64_gen.go, ...), so `-tags ignore`
#     un-ignores those too and the stdlib package graph collapses with
#     "found packages X and main" / import-cycle typecheck errors — zero real
#     findings, non-zero exit. A project-specific tag has no such collision.
#
#   * rest/examples/ and relay/examples/ co-locate MANY `package main` files in
#     ONE directory, so they cannot be analysed in package mode (`main
#     redeclared in this block`). Those two trees are linted FILE BY FILE, the
#     same form scripts/compile_examples.sh uses — naming a single .go target
#     overrides the directory's build-tag filter and still type-checks fully
#     against the SDK. examples/ is one-main-per-dir, so it lints in package mode.
#
# Keeping the file paths stable matters: CHECKLIST.md and PORT_EXAMPLE_OMISSIONS.md
# pin these exact example paths as a cross-port contract.
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

# ---- layer 3: the example trees (see the header) ----------------------------
# Layer 3 ACCUMULATES its result rather than exiting on the first failure, so a
# burn-down sees every example finding in one run instead of one per invocation.
EXAMPLE_TAG="swexample"
example_rc=0

# examples/ — one main per directory, so package mode works.
go vet -tags "$EXAMPLE_TAG" ./examples/... || example_rc=1
golangci-lint run --config "$REPO/.golangci.yml" \
    --max-same-issues 0 --max-issues-per-linter 0 \
    --build-tags "$EXAMPLE_TAG" \
    ${FIX:+$FIX} "$@" ./examples/... || example_rc=1

# rest/examples + relay/examples — many mains per directory; lint file by file.
example_files=()
while IFS= read -r f; do
    example_files+=("$f")
done < <(find "$REPO/rest/examples" "$REPO/relay/examples" -name '*.go' -print | sort)

# Non-vacuity guard: a pass that matches no files exits 0 having analysed
# nothing, which is indistinguishable from "clean". Refuse that outcome.
if [ "${#example_files[@]}" -eq 0 ]; then
    echo "FATAL: no files found under rest/examples or relay/examples — the" >&2
    echo "       example lint pass would silently analyse nothing. Refusing to" >&2
    echo "       report a clean lint. Check the tree layout." >&2
    exit 1
fi

for f in "${example_files[@]}"; do
    go vet -tags "$EXAMPLE_TAG" "$f" || example_rc=1
    golangci-lint run --config "$REPO/.golangci.yml" \
        --max-same-issues 0 --max-issues-per-linter 0 \
        --build-tags "$EXAMPLE_TAG" \
        ${FIX:+$FIX} "$@" "$f" || example_rc=1
done
[ "$example_rc" -eq 0 ] || exit 1

echo "lint: go vet + golangci-lint clean (incl. ${#example_files[@]} single-file examples + ./examples/...)."
