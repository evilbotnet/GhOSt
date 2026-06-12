#!/usr/bin/env bash
# Turn a fresh Debian 13 (trixie) arm64 VM into GhOSt.
# Idempotent: safe to re-run after changes to the overlay or binaries.
#
# Usage (inside the VM, as root, from a checkout or rsync'd dist):
#   sudo ./provision.sh /path/to/ghost-dist
# where ghost-dist contains: ghostd (arm64 binary), shell/ (built frontend),
# overlay/ (this repo's os/overlay).
set -euo pipefail

DIST="${1:?usage: provision.sh <dist-dir with ghostd, shell/, overlay/>}"

echo "==> packages"
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y --no-install-recommends \
  labwc greetd seatd chromium wlrctl grim curl \
  pipewire pipewire-pulse wireplumber \
  network-manager polkitd \
  fonts-noto-core fonts-noto-color-emoji

echo "==> ghost user"
if ! id ghost >/dev/null 2>&1; then
  useradd -m -s /usr/sbin/nologin ghost
fi
usermod -aG video,render,audio,netdev ghost
loginctl enable-linger ghost

echo "==> overlay + binaries"
cp -rv "$DIST/overlay/." /
install -m 0755 "$DIST/ghostd" /usr/local/bin/ghostd
rm -rf /usr/share/ghost/shell
cp -r "$DIST/shell" /usr/share/ghost/shell
chmod +x /usr/share/ghost/session
install -d -o ghost -g ghost -m 0700 /home/ghost/.config /home/ghost/.config/ghost

echo "==> services"
systemctl daemon-reload
machinectl shell ghost@ /usr/bin/systemctl --user daemon-reload 2>/dev/null || true
machinectl shell ghost@ /usr/bin/systemctl --user enable ghost-daemon.service 2>/dev/null || \
  su -s /bin/sh ghost -c 'XDG_RUNTIME_DIR=/run/user/$(id -u) systemctl --user enable ghost-daemon.service' || true
systemctl enable greetd seatd NetworkManager
systemctl set-default graphical.target

echo "==> done — reboot to enter GhOSt"
