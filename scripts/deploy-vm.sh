#!/usr/bin/env bash
# Push a release build into the dev VM and re-provision.
#   GHOST_VM=user@192.168.x.x ./scripts/deploy-vm.sh          (UTM / real host)
#   GHOST_VM=admin@127.0.0.1 GHOST_SSH_PORT=2222 \
#     GHOST_SSH_KEY=~/ghost-vm/id_ed25519 ./scripts/deploy-vm.sh   (qemu VM)
set -euo pipefail
cd "$(dirname "$0")/.."

VM="${GHOST_VM:?set GHOST_VM=user@host (the Debian 13 arm64 VM)}"
PORT="${GHOST_SSH_PORT:-22}"
SSH_OPTS=(-p "$PORT" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR)
[[ -n "${GHOST_SSH_KEY:-}" ]] && SSH_OPTS+=(-i "$GHOST_SSH_KEY")

./scripts/build-release.sh
rsync -az --delete -e "ssh ${SSH_OPTS[*]}" dist/ghost/ "$VM:/tmp/ghost-dist/"
ssh "${SSH_OPTS[@]}" "$VM" "sudo /tmp/ghost-dist/provision.sh /tmp/ghost-dist && sudo systemctl restart greetd"
