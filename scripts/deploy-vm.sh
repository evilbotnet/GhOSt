#!/usr/bin/env bash
# Push a release build into the dev VM and re-provision.
#   GHOST_VM=user@192.168.x.x ./scripts/deploy-vm.sh
set -euo pipefail
cd "$(dirname "$0")/.."

VM="${GHOST_VM:?set GHOST_VM=user@host (the Debian 13 arm64 VM)}"

./scripts/build-release.sh
rsync -az --delete dist/ghost/ "$VM:/tmp/ghost-dist/"
ssh -t "$VM" "sudo /tmp/ghost-dist/provision.sh /tmp/ghost-dist && sudo systemctl restart greetd"
