#!/usr/bin/env bash
# Turn a fresh Debian 13 (trixie) arm64 VM into OpenOS.
# Idempotent: safe to re-run after changes to the overlay or binaries.
#
# Usage (inside the VM, as root, from a checkout or rsync'd dist):
#   sudo ./provision.sh /path/to/openos-dist
# where openos-dist contains: osd (arm64 binary), shell/ (built frontend),
# overlay/ (this repo's os/overlay).
set -euo pipefail

DIST="${1:?usage: provision.sh <dist-dir with osd, shell/, overlay/>}"

echo "==> packages"
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y --no-install-recommends \
  labwc greetd seatd chromium wlrctl grim curl \
  pipewire pipewire-pulse wireplumber \
  network-manager polkitd \
  fonts-noto-core fonts-noto-color-emoji

echo "==> openos user"
if ! id openos >/dev/null 2>&1; then
  useradd -m -s /usr/sbin/nologin openos
fi
usermod -aG video,render,audio,netdev openos
loginctl enable-linger openos

echo "==> overlay + binaries"
cp -rv "$DIST/overlay/." /
install -m 0755 "$DIST/osd" /usr/local/bin/osd
rm -rf /usr/share/openos/shell
cp -r "$DIST/shell" /usr/share/openos/shell
chmod +x /usr/share/openos/session
install -d -o openos -g openos -m 0700 /home/openos/.config /home/openos/.config/openos

echo "==> services"
systemctl daemon-reload
machinectl shell openos@ /usr/bin/systemctl --user daemon-reload 2>/dev/null || true
machinectl shell openos@ /usr/bin/systemctl --user enable openos-daemon.service 2>/dev/null || \
  su -s /bin/sh openos -c 'XDG_RUNTIME_DIR=/run/user/$(id -u) systemctl --user enable openos-daemon.service' || true
systemctl enable greetd seatd NetworkManager
systemctl set-default graphical.target

echo "==> done — reboot to enter OpenOS"
