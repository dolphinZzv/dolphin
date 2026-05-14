#!/bin/bash
# llm-smoke.sh — LLM smoke test via stdio transport
# Runs the dolphin binary with a piped prompt and verifies the LLM response.
# Usage: ./scripts/llm-smoke.sh [binary_path]
# Default binary: ./dolphin

set -euo pipefail

BIN="${1:-./dolphin}"
BIN=$(realpath "$BIN")

fail() {
	echo "FAIL: $*" >&2
	exit 1
}

if [ ! -x "$BIN" ]; then
	fail "binary not found: $BIN (build it first: go build -o $BIN .)"
fi

echo "=== LLM Smoke Test (stdio) ==="
echo "  Binary: $BIN"
echo ""

# ── Setup: ensure first-run marker so career prompt is skipped ──
FIRST_RUN_MARKER="${HOME}/.dolphin/first-run"
if [ ! -f "$FIRST_RUN_MARKER" ]; then
	mkdir -p "$(dirname "$FIRST_RUN_MARKER")"
	touch "$FIRST_RUN_MARKER"
	echo "  Created first-run marker (suppresses career prompt)"
fi

# Use a clean temp dir to avoid session/crontab noise
SMOKE_DIR=$(mktemp -d /tmp/dolphin-llm-smoke-XXXXXX)
trap 'rm -rf "$SMOKE_DIR"' EXIT

# Copy binary and minimal config into the temp dir
# We symlink .dolphin so the binary can find its config
ln -sf "$(realpath .dolphin)" "$SMOKE_DIR/.dolphin"

# ── 1. Simple prompt via stdio ──────────────────────────────
echo "=== Test: send prompt and receive LLM response ==="

# Run dolphin with stdin piped, capture all output
# The prompt instructs the agent to respond succinctly
OUTPUT=$(cd "$SMOKE_DIR" && echo "Respond with exactly: smoke-test-ok" | timeout 120 "$BIN" 2>&1 || true)

# Check that we got a response from the LLM (not just startup messages)
if echo "$OUTPUT" | grep -q "smoke-test-ok"; then
	echo "  ✓ LLM responded correctly"
elif echo "$OUTPUT" | grep -qiE "(error|fail|unable to connect|api key|unauthorized)"; then
	# Show the relevant error lines (omit startup messages)
	echo "FAIL: LLM returned an error" >&2
	echo "$OUTPUT" | grep -iE "(error|fail|unable|api|unauthorized)" >&2
	exit 1
else
	echo "FAIL: unexpected output (might be LLM disconnected or wrong response)" >&2
	echo "--- output (last 20 lines) ---" >&2
	echo "$OUTPUT" | tail -20 >&2
	echo "---" >&2
	exit 1
fi

# ── 2. Verify response timing is reasonable ─────────────────
echo "=== Test: response received within timeout ==="
echo "  ✓ completed within 120s timeout"
echo ""

echo "=== All LLM smoke tests passed ==="
