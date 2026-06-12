#!/usr/bin/env bash
# Build everything for a Linux arm64 target into dist/openos/.
set -euo pipefail
cd "$(dirname "$0")/.."

echo "==> shell"
pnpm --filter @openos/shell build

echo "==> osd (linux/arm64)"
(cd daemon && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ../dist/openos/osd ./cmd/osd)

echo "==> staging"
mkdir -p dist/openos
rm -rf dist/openos/shell dist/openos/overlay
cp -r apps/shell/dist dist/openos/shell
cp -r os/overlay dist/openos/overlay
cp os/vm/provision.sh dist/openos/

echo "==> dist/openos ready:"
du -sh dist/openos/* | sed 's/^/    /'
