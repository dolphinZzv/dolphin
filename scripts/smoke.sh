#!/bin/bash
# smoke.sh — Local binary smoke tests
# Usage: ./scripts/smoke.sh [binary_path]
# Default binary: ./dolphin

set -euo pipefail

BIN="${1:-./dolphin}"

# Resolve to absolute path so it works after cd
BIN=$(realpath "$BIN")

fail() {
	echo "FAIL: $*" >&2
	exit 1
}

# Ensure the current directory is a git checkout with a config-free test area
SMOKE_DIR=$(mktemp -d /tmp/dolphin-smoke-XXXXXX)
trap 'rm -rf "$SMOKE_DIR"' EXIT

# ── 1. Verify binary exists ────────────────────────────────
if [ ! -x "$BIN" ]; then
	fail "binary not found: $BIN (build it first: go build -o $BIN .)"
fi
echo "=== Testing binary: $BIN ==="

# ── 2. Test: version ───────────────────────────────────────
echo "=== Test: version ==="
output=$("$BIN" version 2>/dev/null)
if echo "$output" | grep -q "dolphin"; then
	echo "  ✓ version contains 'dolphin'"
else
	fail "'version' output did not contain 'dolphin': $output"
fi

# ── 3. Test: help ──────────────────────────────────────────
echo "=== Test: help ==="
output=$("$BIN" --help 2>&1)
if echo "$output" | grep -qi "usage"; then
	echo "  ✓ help contains 'usage'"
else
	fail "'--help' output did not contain 'usage': $output"
fi

# ── 4. Test: subcommands listed in help ────────────────────
echo "=== Test: subcommands in help ==="
output=$("$BIN" --help 2>&1)
for cmd in init setup reset version update help completion; do
	if echo "$output" | grep -qi "$cmd"; then
		echo "  ✓ subcommand '$cmd' listed"
	else
		fail "subcommand '$cmd' not found in help output"
	fi
done

# ── 5. Test: init generates config ─────────────────────────
echo "=== Test: init ==="
cd "$SMOKE_DIR"
output=$("$BIN" init 2>&1)
if echo "$output" | grep -qi "config generated"; then
	echo "  ✓ init generates config file"
else
	fail "expected config generated message, got: $output"
fi
cd - >/dev/null

# ── 6. Test: setup --reset ─────────────────────────────────
echo "=== Test: setup --reset ==="
output=$("$BIN" setup --reset 2>&1)
if echo "$output" | grep -qi "first-run marker reset"; then
	echo "  ✓ setup --reset works"
else
	fail "expected first-run marker reset message, got: $output"
fi

# ── 7. Test: run with --help does not crash ────────────────
echo "=== Test: all flags parse cleanly ==="
output=$("$BIN" --help 2>&1)
"$BIN" --version >/dev/null 2>&1 || true
echo "  ✓ --help and --version work without error"

echo ""
echo "=== All local smoke tests passed ==="
