#!/usr/bin/env bash
# compile_examples.sh -- compile-check every example file, one by one.
#
# Go's ``package main`` means that two examples living side-by-side in the
# same directory (as ``rest/examples/*.go`` and ``relay/examples/*.go`` do)
# cannot be built by ``go build ./rest/examples/...``.  Each file also
# carries a ``//go:build ignore`` tag so ``go build ./...`` skips them.
#
# This script compiles each file independently via
# ``go build -o /dev/null FILE.go``; that form honours even
# ``//go:build ignore``-tagged files because a single .go target overrides
# the directory's build-tag filter.
#
# Fails (exit 1) if any example doesn't compile.  Keep this green --
# it's wired into the Layer C CI workflow at
# ``.github/workflows/doc-audit.yml``.

set -euo pipefail

cd "$(dirname "$0")/.."

total=0
failed=0
failed_files=()

for f in $(find examples relay/examples rest/examples -name "*.go" -print 2>/dev/null | sort); do
    total=$((total + 1))
    if ! go build -o /dev/null "$f" 2> /tmp/compile_examples_err; then
        failed=$((failed + 1))
        failed_files+=("$f")
        echo "FAIL: $f"
        sed 's/^/    /' /tmp/compile_examples_err
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
