# GhOSt image for Raspberry Pi 400/4

`build-image.sh` produces a flashable SD-card image by customizing the
official **Raspberry Pi OS Lite (arm64)** image in a native arm64 chroot —
the `sdm` approach rather than a pi-gen stage build. Why: it runs anywhere
arm64 Debian runs (our dev VM, no Docker/x86 build box), takes minutes
instead of hours, and the Pi kernel + firmware + V3D graphics stack come
straight from the official image instead of being our problem.

## Build (in the dev VM or any arm64 Debian)

```sh
./scripts/build-release.sh                       # on the Mac: dist/ghost/
rsync dist/ghost/ into the builder, then there:
sudo ./build-image.sh /path/to/ghost-dist ghost-pi.img
xz -T0 -3 ghost-pi.img                           # ~4x smaller for transport
```

## Flash + first boot (Pi 400)

1. Flash `ghost-pi.img(.xz)` with Raspberry Pi Imager (no customization
   needed — ignore its OS-settings prompts) or `dd`.
2. Boot the Pi 400 (quiet boot — no kernel text wall). First boot
   auto-expands the filesystem and opens the **GhOSt setup wizard**:
   password, timezone, Wi-Fi, and Ghost AI routing (local/LAN/cloud).
3. After setup the desktop is yours. Browser/Files/Terminal/Editor work
   immediately.
4. Office: open a Terminal and run `sudo ghost-install-office` once
   (CryptPad ~500 MB; kept out of the image to keep it lean).

## Passwords and root

- **There is no root password** — root is locked, like Ubuntu/RPi OS.
- The wizard's password is for the `ghost` user and gates `sudo` in the
  Terminal: `sudo raspi-config`, `sudo apt …` all work after setup.
- Privileged OS actions (password, timezone, hostname) go through
  `ghost-admin.service`, a four-verb root helper on a unix socket only the
  ghost user can reach — the daemon itself never runs as root.

## What the image contains

- Raspberry Pi OS Lite base: Pi kernel, firmware, `vc4-kms-v3d`,
  NetworkManager, Pi-tuned Chromium from the RPi repo
- greetd autologin (`ghost` user) → labwc → Chromium `--app` shell
- ghostd + shell + overlay (same units as the VM: Restart=always)
- zram (1.5 GB zstd, swappiness 100) per the 4 GB memory plan
- first-boot user wizard disabled (greetd owns tty1)

## Debugging on the device

SSH is off by default. The in-shell Terminal is a full shell:
`sudo systemctl enable --now ssh` if you want in from the LAN.
`journalctl --user -u ghost-daemon` / `-u ghost-chromium` are the first
places to look; `sudo journalctl -u greetd` if the session won't start.
