#!/usr/bin/env bash
# GhOSt dev VM via plain QEMU (Apple Silicon host, HVF acceleration).
# A scripted alternative to UTM: a Debian 13 arm64 cloud image + cloud-init,
# matching the Pi 400's 4 GB RAM. The display is exposed over local VNC
# (open "Screen Sharing" → vnc://127.0.0.1:5907) and SSH on 127.0.0.1:2222.
#
#   ./scripts/vm-qemu.sh create   # one-time: seed ISO, ssh key, disk resize
#   ./scripts/vm-qemu.sh start    # boot in the background
#   ./scripts/vm-qemu.sh ssh ...  # run a command (or get a shell)
#   ./scripts/vm-qemu.sh stop
set -euo pipefail

VMDIR="${GHOST_VMDIR:-$HOME/ghost-vm}"
DISK="$VMDIR/debian-13-arm64.qcow2"
SEED="$VMDIR/seed.iso"
KEY="$VMDIR/id_ed25519"
VARS="$VMDIR/efi-vars.fd"
PIDFILE="$VMDIR/qemu.pid"
FW="$(dirname "$(dirname "$(command -v qemu-system-aarch64)")")/share/qemu/edk2-aarch64-code.fd"
SSH_PORT=2222
VNC_DISPLAY=7   # VNC on 5900+7 = 5907

cmd_create() {
  mkdir -p "$VMDIR"
  [[ -f "$DISK" ]] || {
    echo "missing $DISK — download with:"
    echo "  curl -fLo $DISK https://cloud.debian.org/images/cloud/trixie/latest/debian-13-genericcloud-arm64.qcow2"
    exit 1
  }
  [[ -f "$KEY" ]] || ssh-keygen -t ed25519 -N "" -f "$KEY" -C ghost-vm >/dev/null

  local seeddir
  seeddir=$(mktemp -d)
  cat > "$seeddir/user-data" <<EOF
#cloud-config
hostname: ghost
preserve_hostname: false
users:
  - name: admin
    sudo: ALL=(ALL) NOPASSWD:ALL
    groups: users
    shell: /bin/bash
    ssh_authorized_keys:
      - $(cat "$KEY.pub")
ssh_pwauth: false
EOF
  echo "instance-id: ghost-vm-1" > "$seeddir/meta-data"
  rm -f "$SEED"
  hdiutil makehybrid -iso -joliet -default-volume-name cidata -o "$SEED" "$seeddir" >/dev/null
  rm -rf "$seeddir"

  qemu-img resize "$DISK" 14G
  [[ -f "$VARS" ]] || dd if=/dev/zero of="$VARS" bs=1m count=64 2>/dev/null
  echo "VM created in $VMDIR"
}

cmd_start() {
  [[ -f "$PIDFILE" ]] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null && {
    echo "already running (pid $(cat "$PIDFILE"))"
    return
  }
  qemu-system-aarch64 \
    -machine virt,highmem=on -accel hvf -cpu host -smp 4 -m 4096 \
    -drive if=pflash,format=raw,readonly=on,file="$FW" \
    -drive if=pflash,format=raw,file="$VARS" \
    -drive if=virtio,format=qcow2,file="$DISK" \
    -drive if=virtio,format=raw,readonly=on,file="$SEED" \
    -device virtio-gpu-pci,xres=1366,yres=768 \
    -device qemu-xhci -device usb-kbd -device usb-tablet \
    -netdev user,id=n0,hostfwd=tcp:127.0.0.1:$SSH_PORT-:22 \
    -device virtio-net-pci,netdev=n0 \
    -display vnc=127.0.0.1:$VNC_DISPLAY \
    -daemonize -pidfile "$PIDFILE"
  echo "VM booting — ssh: port $SSH_PORT · display: vnc://127.0.0.1:$((5900 + VNC_DISPLAY))"
}

cmd_ssh() {
  ssh -p $SSH_PORT -i "$KEY" \
    -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR \
    admin@127.0.0.1 "$@"
}

cmd_stop() {
  [[ -f "$PIDFILE" ]] && kill "$(cat "$PIDFILE")" 2>/dev/null || true
  rm -f "$PIDFILE"
  echo "stopped"
}

cmd_status() {
  if [[ -f "$PIDFILE" ]] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
    echo "running (pid $(cat "$PIDFILE"))"
  else
    echo "stopped"
  fi
}

case "${1:-}" in
  create) cmd_create ;;
  start) cmd_start ;;
  ssh) shift; cmd_ssh "$@" ;;
  stop) cmd_stop ;;
  status) cmd_status ;;
  *) echo "usage: $0 {create|start|ssh|stop|status}"; exit 1 ;;
esac
