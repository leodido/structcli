#!/usr/bin/env bash
# Build all example WASM binaries for the demo.
# Run from the repo root: bash examples/wasm-demo/build-wasm.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
OUT_DIR="${REPO_ROOT}/examples/wasm-demo/public/wasm"
mkdir -p "$OUT_DIR"

examples=(minimal simple collections customtypes full loginsvc mcp-command-factory structerr)

for ex in "${examples[@]}"; do
  echo "Building ${ex}.wasm..."
  GOOS=wasip1 GOARCH=wasm go build -o "${OUT_DIR}/${ex}.wasm" "${REPO_ROOT}/examples/${ex}/"
done

echo "Done. WASM binaries in ${OUT_DIR}/"
