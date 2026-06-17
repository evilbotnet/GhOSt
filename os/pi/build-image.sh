#!/usr/bin/env bash
# Build the flashable GhOSt image for Raspberry Pi 400/4.
#
# Runs on any arm64 Debian host as root (our dev VM is perfect): takes the
# official Raspberry Pi OS Lite arm64 image and customizes it in a native
# chroot — the sdm approach. No Docker, no emulation, no pi-gen stage builds;
# the Pi kernel/firmware/V3D stack comes straight from the official image.
#
#   sudo ./build-image.sh /path/to/ghost-dist [output.img]
# where ghost-dist contains: ghostd (arm64), shell/, overlay/, install-cryptpad.sh
set -euo pipefail

DIST="${1:?usage: build-image.sh <dist-dir> [output.img]}"
OUT="${2:-ghost-pi.img}"
BASE_URL="https://downloads.raspberrypi.com/raspios_lite_arm64_latest"
WORK="$(mktemp -d /var/tmp/ghost-img.XXXX)"
MNT="$WORK/mnt"
LOOP=""

cleanup() {
  set +e
  umount -R "$MNT" 2>/dev/null
  [[ -n "$LOOP" ]] && losetup -d "$LOOP" 2>/dev/null
  rm -rf "$WORK"
}
trap cleanup EXIT

echo "==> host build deps"
export DEBIAN_FRONTEND=noninteractive
apt-get install -y -qq --no-install-recommends parted xz-utils e2fsprogs curl >/dev/null

echo "==> base image"
curl -fL "$BASE_URL" -o "$WORK/base.img.xz"
xz -d "$WORK/base.img.xz"
mv "$WORK/base.img" "$OUT"

echo "==> grow image (+2.5G for chromium & friends)"
truncate -s +2560M "$OUT"
parted -s "$OUT" resizepart 2 100%
LOOP=$(losetup -fP --show "$OUT")
e2fsck -pf "${LOOP}p2" >/dev/null || [[ $? -le 2 ]] # 1-2 = fixed, fine
resize2fs "${LOOP}p2" 2>/dev/null

echo "==> mount"
mkdir -p "$MNT"
mount "${LOOP}p2" "$MNT"
mount "${LOOP}p1" "$MNT/boot/firmware"
for d in proc sys dev dev/pts; do mount --bind "/$d" "$MNT/$d"; done
cp /etc/resolv.conf "$MNT/etc/resolv.conf.ghost-build"
mv "$MNT/etc/resolv.conf" "$MNT/etc/resolv.conf.orig" 2>/dev/null || true
mv "$MNT/etc/resolv.conf.ghost-build" "$MNT/etc/resolv.conf"

echo "==> packages (native arm64 chroot)"
chroot "$MNT" /usr/bin/env DEBIAN_FRONTEND=noninteractive bash -c "
  apt-get update -qq
  apt-get install -y --no-install-recommends \
    labwc greetd chromium wlrctl grim swaylock swayidle curl \
    pipewire pipewire-pulse wireplumber \
    polkitd dbus-user-session zram-tools fontconfig \
    fonts-noto-core fonts-noto-color-emoji
  apt-get clean
  rm -rf /var/lib/apt/lists/*
"

echo "==> ghost user"
chroot "$MNT" bash -c "
  id ghost >/dev/null 2>&1 || useradd -m -s /bin/bash ghost
  usermod -aG video,render,audio,netdev ghost
  mkdir -p /var/lib/systemd/linger && touch /var/lib/systemd/linger/ghost
  install -d -o ghost -g ghost -m 0700 /home/ghost/.config /home/ghost/.config/ghost
"

echo "==> GhOSt payload"
cp -r "$DIST/overlay/." "$MNT/"
install -m 0755 "$DIST/ghostd" "$MNT/usr/local/bin/ghostd"
rm -rf "$MNT/usr/share/ghost/shell"
cp -r "$DIST/shell" "$MNT/usr/share/ghost/shell"
chmod +x "$MNT/usr/share/ghost/session"
install -m 0755 "$DIST/install-cryptpad.sh" "$MNT/usr/local/sbin/ghost-install-office"

echo "==> services"
systemctl --root="$MNT" enable ghost-admin.service >/dev/null 2>&1
systemctl --root="$MNT" enable greetd NetworkManager >/dev/null 2>&1 || true
systemctl --root="$MNT" --global enable ghost-daemon.service
systemctl --root="$MNT" set-default graphical.target
# The Pi first-boot user wizard owns tty1 — greetd does now.
systemctl --root="$MNT" disable userconfig.service >/dev/null 2>&1 || true
systemctl --root="$MNT" mask userconfig.service >/dev/null 2>&1 || true

echo "==> quiet boot (power on -> GhOSt, not a kernel text wall)"
sed -i '1 s/$/ quiet loglevel=3 vt.global_cursor_default=0 logo.nologo/' \
  "$MNT/boot/firmware/cmdline.txt"

echo "==> identity + memory tuning"
echo ghost > "$MNT/etc/hostname"
sed -i 's/raspberrypi/ghost/g' "$MNT/etc/hosts"
cat > "$MNT/etc/default/zramswap" <<'EOF'
ALGO=zstd
SIZE=1536
PRIORITY=100
EOF
echo "vm.swappiness=100" > "$MNT/etc/sysctl.d/90-ghost-zram.conf"

echo "==> teardown"
mv "$MNT/etc/resolv.conf.orig" "$MNT/etc/resolv.conf" 2>/dev/null || rm -f "$MNT/etc/resolv.conf"
umount -R "$MNT"
losetup -d "$LOOP"; LOOP=""

echo "==> done: $OUT ($(du -h "$OUT" | cut -f1))"
echo "    flash with Raspberry Pi Imager or: xz -T0 -3 $OUT && dd to SD"
