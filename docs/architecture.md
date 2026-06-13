# GhOSt architecture

An open-source, web-native operating system in the spirit of ChromeOS,
with no Google dependence. Target hardware: Raspberry Pi 400/4 (ARM64, 4 GB).
Development host: any machine; a Debian 13 ARM64 VM stands in for the Pi.

## The stack

```
┌──────────────────────────────────────────────────────────┐
│  Shell (Svelte 5) — WM, Files/Terminal/Editor/Settings,  │   web tech
│  Ghost panel, setup wizard                               │
│  …rendered by Chromium  --app=http://127.0.0.1:7700      │
├──────────────────────────────────────────────────────────┤
│  ghostd (Go, :7700) — fs / pty / system / windows /      │   this repo
│  office / webapps / Ghost agent loop / REST+WS           │
│     └─ ghost-admin.service (root, 4 validated verbs)     │
├──────────────────────────────────────────────────────────┤
│  labwc (Wayland) · greetd autologin · PipeWire · NM      │   Debian pkgs
├──────────────────────────────────────────────────────────┤
│  Raspberry Pi OS Lite (Trixie) / Debian 13, arm64        │
└──────────────────────────────────────────────────────────┘
```

## Key decisions (short form — see git history for the full plan)

- **labwc over cage**: we need multiple native windows (browsing) plus
  foreign-toplevel management for the shell taskbar. labwc is Pi OS Trixie's
  default compositor, so it's the best-tested option on the target hardware.
- **One Chromium instance**: the shell runs as a chromeless `--app` window
  pinned as the desktop; "Browser" opens normal tabbed Chromium windows in the
  same instance (daemon: `chromium --new-window <url>` with the shared
  profile). Iframes can't host arbitrary sites; per-window instances cost
  ~300 MB each. Enterprise policies lock down devtools/extensions/chrome://.
- **Go daemon, not Node**: ~25 MB RSS vs ~80 MB, single static binary
  cross-compiled from the dev machine. CryptPad (Phase 3) is the only Node
  process, socket-activated so it costs nothing when Office is closed.
- **In-shell windows for our apps, compositor windows for browsing**: HTML
  windows are cheap, themeable, and pixel-consistent; real sites need real
  browser windows.
- **No backdrop-filter anywhere**: the Pi 400 GPU can't afford it. Depth comes
  from layered shadows and hairline borders (see `theme/tokens.css`).

## Security model

- ghostd binds `127.0.0.1` only and refuses non-loopback `--listen`.
- Per-session bearer token (0600 file). In production ghostd serves the built
  shell and injects the token into `index.html` for its own origin only.
- Origin allowlist on every API request (+ the Vite origin in `--dev`)
  blocks malicious-website `fetch()` to localhost.
- Filesystem ops are canonicalized (symlink-aware) and confined to
  `$HOME` + `/media`; deletes go to `~/.ghost-trash`.
- The daemon never runs as root. Wi-Fi via NetworkManager (nmcli now, D-Bus +
  polkit rules later), audio via PipeWire, brightness/power via logind.
- The four genuinely-root operations (set password, grant sudo, timezone,
  hostname) go through `ghostd helper` — a validated-verb root sidecar on
  `/run/ghost/admin.sock` (0660 root:ghost), GhOSt's systemd-timedated.
- The user's own sudo (password-gated, set in the first-boot wizard) is the
  deliberate power-user path: this is their machine, not a managed kiosk.

## Memory budget (Pi 400, 4 GB)

Base OS ~250 + labwc/greetd ~50 + Chromium core ~400 + shell renderer ~150 +
3-4 tabs ~500-800 + ghostd ~25 + CryptPad-when-open ~250 ≈ **1.6-2.0 GB under
load**. Levers: zram (ships in image), CryptPad on demand, single Chromium
instance, `--renderer-process-limit`, capped terminal scrollback.

## Repository map

| Path | What |
| --- | --- |
| `apps/shell` | the desktop (Svelte 5 + Vite) |
| `daemon` | ghostd (Go) — see `internal/*` per subsystem |
| `packages/protocol` | REST + WS contract |
| `os/overlay` | rootfs overlay shared by the VM and the Pi image |
| `os/vm` | provision a Debian 13 ARM64 VM into GhOSt (scripted QEMU VM: `scripts/vm-qemu.sh`) |
| `os/pi` | flashable Pi 400 image build (chroot-customized RPi OS Lite) |

## Phases

0. ✅ scaffold + shell skeleton in a dev browser
1. ✅ ghostd + Files/Terminal/Editor (macOS inner loop)
2. ✅ scripted QEMU VM boots into the kiosk shell (browser windows, crash
   recovery, window tracking)
3. ✅ CryptPad office (on-demand + sandbox-origin proxy) + real Settings
4. ✅ flashable Pi 400 image (chroot-customized RPi OS Lite) — boots on
   real hardware
5. ✅ first-boot wizard, ghost-admin root helper, sudo path, quiet boot
   — *open:* lock screen, updates panel, themes, notifications
6. ✅ app platform Layer 1: installable web apps
   — *open:* `.osapp` scoped packages ([ADR 0001](decisions/0001-app-platform.md))
7. ✅ **Ghost**: daemon-hosted agent loop, tools = the OS API,
   confirmation-gated, provider-agnostic (verified against a LAN vLLM)
   — *open:* ghostd OpenAI-compatible gateway for terminal AI tools
   ([ADR 0003](decisions/0003-devkit-and-model-gateway.md), as-built: [ADR 0004](decisions/0004-ghost-implementation.md))
