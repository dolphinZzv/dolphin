#!/bin/bash
# docker-smoke.sh — Docker image smoke tests
# Usage: ./scripts/docker-smoke.sh [image_tag]
# Default tag: dolphin:smoke-test

set -euo pipefail

IMAGE="${1:-dolphin:smoke-test}"

fail() {
	echo "FAIL: $*" >&2
	exit 1
}

# ── 1. Build Docker image (skip if already present or SKIP_BUILD is set) ──
if [ -z "${SKIP_BUILD:-}" ] && ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
	echo "=== Building Docker image: $IMAGE ==="
	docker build -t "$IMAGE" .
else
	echo "=== Using existing image: $IMAGE ==="
fi

# ── 2. Test: version ────────────────────────────────────────
echo "=== Test: version ==="
output=$(docker run --rm "$IMAGE" version 2>/dev/null || true)
if echo "$output" | grep -q "dolphin"; then
	echo "  ✓ version contains 'dolphin'"
else
	fail "'dolphin version' output did not contain 'dolphin': $output"
fi

# ── 3. Test: help ───────────────────────────────────────────
echo "=== Test: help ==="
output=$(docker run --rm "$IMAGE" --help 2>/dev/null || true)
if echo "$output" | grep -qi "usage"; then
	echo "  ✓ help contains 'usage'"
else
	fail "'dolphin --help' output did not contain 'usage': $output"
fi

# ── 4. Test: exits gracefully without config ────────────────
echo "=== Test: run exits with expected error (no LLM) ==="
output=$(docker run --rm "$IMAGE" 2>&1 || true)
if echo "$output" | grep -iqE "(llm not configured|no api key)"; then
	echo "  ✓ exits with LLM config message"
else
	fail "expected LLM config warning, got: $output"
fi

# ── 5. Cleanup ──────────────────────────────────────────────
docker rmi "$IMAGE" >/dev/null 2>&1 || true

echo ""
echo "=== All Docker smoke tests passed ==="
