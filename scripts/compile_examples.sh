#!/usr/bin/env bash
# compile_examples.sh -- compile-check every example file, one by one.
#
# Go's ``package main`` means that two examples living side-by-side in the
# same directory (as ``rest/examples/*.go`` and ``relay/examples/*.go`` do)
# cannot be built by ``go build ./rest/examples/...``.  EVERY example file
# across all four trees (``examples/``, ``rest/examples/``,
# ``relay/examples/``, ``livewire/examples/``) carries a ``//go:build swexample``
# tag so ``go build ./...`` skips the demos (they are standalone ``main``
# programs, not library code).  This script is what actually compile-checks
# them.
#
# This script compiles each file independently via
# ``go build -o /dev/null FILE.go``; that form honours even
# build-tagged files because a single .go target overrides the directory's
# build-tag filter.
#
# The examples are also LINTED, at the same bar as the library — see the
# "example coverage" header in scripts/run-lint.sh, which uses this same
# single-file form for the multi-main directories.
#
# Fails (exit 1) if any example doesn't compile.  Keep this green --
# it's wired into the Layer C CI workflow at
# ``.github/workflows/doc-audit.yml``.

set -euo pipefail

cd "$(dirname "$0")/.."

# Scratch stays repo-local (.sw-tmp is gitignored): a fixed /tmp path is shared
# machine-wide and collides when several checkouts compile concurrently.
SCRATCH_DIR=".sw-tmp"
mkdir -p "$SCRATCH_DIR"
ERR_FILE="$SCRATCH_DIR/compile_examples_err.$$"
trap 'rm -f "$ERR_FILE"' EXIT

total=0
failed=0
failed_files=()

for f in $(find examples relay/examples rest/examples -name "*.go" -print 2>/dev/null | sort); do
    total=$((total + 1))
    if ! go build -o /dev/null "$f" 2> "$ERR_FILE"; then
        failed=$((failed + 1))
        failed_files+=("$f")
        echo "FAIL: $f"
        sed 's/^/    /' "$ERR_FILE"
        echo
    fi
done

echo
if [ "$failed" -gt 0 ]; then
    echo "FAILED: $failed of $total example(s) did not compile:"
    for f in "${failed_files[@]}"; do
        echo "    $f"
    done
    exit 1
fi
echo "OK: all $total examples compile"
