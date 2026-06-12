# GhOSt

**G**o · **h**tml · **O**perating **S**ystem · **t**ypescript — with a ghost
in the shell.

An open-source, web-native operating system in the spirit of ChromeOS —
without the Google. A minimal Linux base boots straight into a
hardware-accelerated Chromium running a desktop shell built entirely in web
tech (the *html* and *typescript*), with a small Go daemon (`ghostd`, the
*Go*) providing the system underneath. The resident AI assistant — the
*ghost* — arrives in Phase 7 ([ADR 0002](docs/decisions/0002-ghost-ai-assistant.md)).

Target: Raspberry Pi 400/4 (ARM64, 4 GB). Dev stand-in: Debian 13 ARM64 VM.

![stack](docs/architecture.md)

## Quick start (dev, any OS)

```sh
pnpm install
./scripts/dev.sh        # ghostd daemon :7700 + Vite :5173
open http://localhost:5173
```

You get the full desktop in a browser tab: window manager, Files (your real
home dir), Terminal (real pty), Editor, Settings, launcher (also on the
Meta key), status tray.

## Deploy to the VM / device

```sh
GHOST_VM=admin@<vm-ip> ./scripts/deploy-vm.sh   # see os/vm/README.md
```

## Layout

- `apps/shell` — the desktop (Svelte 5 + Vite)
- `daemon` — `ghostd`, the Go system daemon (fs, pty, system, browser, ws)
- `packages/protocol` — REST + WebSocket contract
- `os/` — overlay, VM provisioning, Pi image build
- `docs/architecture.md` — decisions, security model, memory budget, phases
