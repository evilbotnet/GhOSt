# GhOSt roadmap

The north star: **an operating system you own, that an AI can drive on your
behalf — locally, auditably, and without overlords.** Everything below is
weighed against that. GhOSt should feel like a real computer, lean hard into
being AI-native, and never make the user trade away control or privacy to get
there.

Status legend: ✅ done · 🔜 next · 🟡 planned · 💭 exploring.
Phases 0–7 (boot → VM → image → wizard → Ghost → extensibility → apps/polish)
are ✅ — see [architecture.md](architecture.md) and the ADRs.

---

## Now → Next (the immediate edge)

**Ghost's local router & command tier** ✅ (core) — [ADR 0002](decisions/0002-ghost-ai-assistant.md).
Built: a deterministic capability-tier router — tier-0 command rules turn one
utterance into one tool call with **no model, fully offline** ("volume 40",
"lock", "open <url>"), tier-2 escalates multi-step work to the agent provider
with `routing.fallback`, tier-1 uses a local intent model when it's the only
brain, plus `ask <provider>` overrides. Provenance shows the tier per reply.
The unified `Super+Space` **command palette** (apps + commands + Ghost in one
box) is shipped — typeahead opens any installed app (built-in, web, `.osapp`)
instantly, and free text hands off to Ghost's router. Remaining: the on-device
llama.cpp lifecycle (socket-activated, stop-when-idle — today the intent tier
is just a provider at a local URL; deferred to keep the image lean).

**`.osapp` packages + store** ✅ — [ADR 0001](decisions/0001-app-platform.md)
Layer 2 / [ADR 0009](decisions/0009-osapp-packaging-store.md). Shipped: a zip +
manifest installed under `~/.local/share/ghost/apps/<id>/` with **scoped-
permission tokens** the auth layer now enforces per request (fail-closed), and a
**store/registry** in the Hub that browses and one-click-installs apps, skills,
tools, and MCP servers from an **Ed25519-signed git index** (`ghostd
store-keygen`/`store-sign` for publishers). Apps are isolated in a daemon-
enforced **opaque-origin sandbox** (CSP `sandbox`), so a malicious app can't
read the shell's token — verified both directions (see ADR 0009).

**MCP Streamable-HTTP transport** 🔜 — the MCP client is stdio-only; add HTTP
so hosted MCP servers (not just `npx` ones) work. Move keys into a small
**credential vault** (currently key files).

**README/imagery & CI** 🔜 — automate image builds on tag (go vet +
svelte-check + build + release), add a contributor guide and a skills/tools
gallery so others can pile on.

---

## Planned (clear value, slated)

**AI**
- 🟡 **Voice** — "Hey Ghost": wake word + on-device STT (whisper.cpp) + TTS.
  An appliance you talk to is the killer Pi-400 demo.
- ✅ **Proactive Ghost** — scheduled runs on an interval or daily time, read-only,
  surfaced through notifications ([ADR 0007](decisions/0007-scheduled-ghost.md)).
- 🟡 **Cross-session memory** — a memory store Ghost reads/writes, so it
  remembers preferences and context between sessions.
- ✅ **Apps expose tools to Ghost** — a running app registers tools over the WS;
  Ghost invokes them, mutations gated ([ADR 0006](decisions/0006-app-ghost-tools.md)).
- ✅ **Model gateway** — [ADR 0003](decisions/0003-devkit-and-model-gateway.md):
  Ghost's configured provider exposed as a localhost OpenAI-compatible `/v1`
  endpoint so `pi`, Herdr, and other tools inherit your keys + routing + audit.
- 🟡 **Devkit** — optional one-click `pi` / Herdr install.

**System & OS**
- 🟡 **A/B image updates + verified boot** — atomic, rollback-safe updates
  beyond `apt` (the v2 update story).
- 🟡 **More Settings** — Bluetooth pairing, display scaling, keyboard layout,
  power/battery, saved Wi-Fi management, printing.
- 🟡 **Backup & restore** — snapshot/export `~` and all GhOSt config.

**Desktop**
- 🟡 **Window tiling / snapping + virtual desktops**, wallpaper picker, fonts.
- 🟡 **Offline-first** caching for installed web apps.
- 🟡 **Accessibility** — text scaling, high contrast, screen-reader passes.

---

## Exploring (bets worth a prototype)

- 💭 **GPIO & physical computing** — it's a Pi: a daemon API + Ghost tool for
  pins/sensors. "Ghost, blink the LED when the build finishes." Few OSes treat
  the GPIO header as a first-class, AI-addressable surface.
- 💭 **Touchscreen / tablet mode** — the official Pi display; a kiosk-tablet UX.
- 💭 **Camera app + Ghost vision** — Pi Camera in, multimodal model out.
- 💭 **Pi 5 / Pi 500 / other SBC images**, and an x86 mini-PC target.
- 💭 **Multi-user** — per-user encrypted homes; GhOSt as a shared family device.
- 💭 **Reproducible builds + SBOM** — `rpi-image-gen` path for auditable images.
- 💭 **Federation/sync** — optional, self-hosted sync of config/files/notes
  across your own GhOSt devices (no cloud account).

---

## Non-goals (what keeps GhOSt itself)

- No mandatory cloud account, no telemetry, no phone-home. AI is **off until
  configured** and always points at an endpoint the user chose.
- No locked bootloader or app gatekeeping — the Terminal is a full shell by
  design; the user owns the machine.
- The OS core stays small and auditable. Big third-party things live as
  `.osapp` packages, MCP servers, or `apt` — not baked into the core.
