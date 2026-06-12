#!/usr/bin/env bash
# OpenOS dev inner loop (macOS or Linux):
#   - builds and runs the osd daemon on :7700 with a dev token
#   - runs the Vite dev server on :5173, proxying /api -> :7700
set -euo pipefail
cd "$(dirname "$0")/.."

TOKEN_FILE=".openos-dev-token"
if [[ ! -f "$TOKEN_FILE" ]]; then
  head -c 32 /dev/urandom | xxd -p -c 64 > "$TOKEN_FILE"
fi

cleanup() { kill 0 2>/dev/null || true; }
trap cleanup EXIT

(
  cd daemon
  go build -o osd ./cmd/osd
  exec ./osd --listen 127.0.0.1:7700 --token-file "../$TOKEN_FILE" --dev
) &

pnpm --filter @openos/shell dev &

wait
