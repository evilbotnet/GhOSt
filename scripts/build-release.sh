#!/usr/bin/env bash
# Build everything for a Linux arm64 target into dist/ghost/.
set -euo pipefail
cd "$(dirname "$0")/.."

echo "==> shell"
pnpm --filter @ghostos/shell build

echo "==> ghostd (linux/arm64)"
(cd daemon && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ../dist/ghost/ghostd ./cmd/ghostd)

echo "==> staging"
mkdir -p dist/ghost
rm -rf dist/ghost/shell dist/ghost/overlay
cp -r apps/shell/dist dist/ghost/shell
cp -r os/overlay dist/ghost/overlay
cp os/vm/provision.sh dist/ghost/

echo "==> dist/ghost ready:"
du -sh dist/ghost/* | sed 's/^/    /'
cp os/vm/install-cryptpad.sh dist/ghost/
cp os/pi/build-image.sh dist/ghost/
