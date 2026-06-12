# OpenOS dev VM (UTM on Apple Silicon)

Pi OS Trixie ≈ Debian 13, so a Debian 13 ARM64 VM is the Pi 400 stand-in.
Pi OS images don't boot in UTM (they expect Pi firmware, not UEFI).

## One-time setup

1. Install [UTM](https://mac.getutm.app) (`brew install --cask utm`).
2. Create a VM from the UTM gallery's **Debian 13 (ARM64)** image, or install
   from the Debian netinst ISO: Virtualize → Linux, 4 GB RAM (match the Pi),
   2 cores, 16 GB disk. A standard install with an `admin` user is fine —
   provisioning creates the separate `openos` kiosk user.
3. In the VM: `sudo apt install openssh-server rsync` and note its IP
   (`ip a`, usually 192.168.64.x with UTM's shared network).
4. Optional: `ssh-copy-id admin@<vm-ip>`.

## Deploy loop (from the repo root on the Mac)

```sh
OPENOS_VM=admin@192.168.64.X ./scripts/deploy-vm.sh
```

This cross-compiles osd, builds the shell, rsyncs everything in, runs
`provision.sh`, and restarts greetd. The VM console should land in the OpenOS
desktop with no login prompt.

## Phase 2 verification checklist

- [ ] boots to the shell fullscreen, no greeter
- [ ] **risk check first**: `wlrctl toplevel list` over SSH — confirm the
      shell window's app_id matches the `chrome-127.0.0.1*` rule in
      `os/overlay/usr/share/openos/labwc/rc.xml` (fallback documented in plan)
- [ ] Files/Terminal work against the VM's real filesystem and shell
- [ ] taskbar Browser button opens a tabbed Chromium window above the desktop
- [ ] `chrome://settings` blocked by policy; F12 does nothing
- [ ] `pkill chromium` → session recovers to the desktop
- [ ] `systemctl --user -M openos@ status openos-daemon` is active
