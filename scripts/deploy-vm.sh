#!/usr/bin/env bash
# Push a release build into the dev VM and re-provision.
#   OPENOS_VM=user@192.168.x.x ./scripts/deploy-vm.sh
set -euo pipefail
cd "$(dirname "$0")/.."

VM="${OPENOS_VM:?set OPENOS_VM=user@host (the Debian 13 arm64 VM)}"

./scripts/build-release.sh
rsync -az --delete dist/openos/ "$VM:/tmp/openos-dist/"
ssh -t "$VM" "sudo /tmp/openos-dist/provision.sh /tmp/openos-dist && sudo systemctl restart greetd"
