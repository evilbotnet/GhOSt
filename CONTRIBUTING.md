# Contributing to GhOSt

GhOSt is an open-source, web-native, AI-native OS for the Raspberry Pi 400.
The north star: **an operating system you own, that an AI can drive on your
behalf — locally, auditably, and without overlords** (see
[docs/roadmap.md](docs/roadmap.md)).

There's a place to help at every level — from a five-line skill that needs no
code, to the Go daemon and Svelte shell. This guide gets you set up and points
you at the right entry point.

## Quick start (macOS or Linux)

Prerequisites: **Go 1.24+**, **Node 22+**, **pnpm 10** (`corepack enable`).

```bash
git clone https://github.com/evilbotnet/GhOSt && cd GhOSt
pnpm install
make dev          # Vite shell on :5173 + ghostd daemon on :7700
# open http://localhost:5173
```

- `make dev` — the inner loop: hot-reloading shell + the daemon with a dev token.
- `make run-dist` — production-like: ghostd serves the *built* shell on one
  origin (`http://127.0.0.1:7700`), exactly as it runs on the Pi. Use this as a
  final smoke test before a release.

On macOS the daemon uses a mock driver for hardware bits (Wi-Fi, volume, lock,
GPIO); files, terminal, metrics, Ghost, the app platform, and the store are all
real. The full hardware surface lights up on the Debian VM and the Pi.

## Repository layout

```
apps/shell/      Svelte 5 + TypeScript desktop shell
daemon/          ghostd — the Go daemon (the system API + Ghost)
  internal/ai/         Ghost: agent loop, router, tools, skills, MCP, scheduler
  internal/osapp/      .osapp packages + permission scopes
  internal/store/      signed git-index store client
  internal/httpapi/    REST + WebSocket + auth/scope enforcement
os/overlay/      rootfs overlay (systemd units, shipped skills/tools, policies)
os/pi/           build-image.sh — the flashable Pi image
docs/decisions/  ADRs — the "why" behind every subsystem (read these)
```

## Ways to contribute, easiest first

1. **Skills** (no code) — teach Ghost to do a multi-step task well. A folder + a
   `SKILL.md`. See the [gallery](docs/gallery.md#add-a-skill).
2. **Tools** (a small script) — give Ghost a new local action. A JSON manifest +
   an executable. See the [gallery](docs/gallery.md#add-a-tool).
3. **MCP servers** — wire in a whole toolset over the Model Context Protocol
   (Hub → MCP, or `ai.toml`). Today's transport is stdio (run a server locally,
   e.g. via `npx` or Docker).
4. **`.osapp` packages** — full third-party apps with scoped permissions, sand-
   boxed and installable from the store. See
   [ADR 0009](docs/decisions/0009-osapp-packaging-store.md).
5. **Core** — the daemon (Go) and shell (Svelte). Start with the
   [architecture](docs/architecture.md) and the relevant ADR.

Skills and tools are the best on-ramp: they ship in
[docs/gallery.md](docs/gallery.md) and need no Go or Svelte.

## Before you open a PR

CI ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)) runs on every PR and
must pass. Run the same checks locally:

```bash
# daemon
cd daemon && go vet ./... && go test ./...
# shell
pnpm --filter @ghostos/shell check && pnpm --filter @ghostos/shell build
```

- Keep PRs focused; one concern per PR.
- Match the surrounding code — its naming, comment density, and idioms.
- Add or update tests for daemon behavior; the daemon has good coverage and
  it's expected to stay that way.
- Update the relevant ADR (or add one under `docs/decisions/`) when you change
  an architectural decision. Touch `docs/roadmap.md` if you move an item.
- **No secrets, keys, or personal data** in commits. AI co-authorship trailers
  (`Co-Authored-By:`) are welcome.

## Releases

Maintainers tag `vX.Y.Z`; [`release.yml`](.github/workflows/release.yml) builds
the flashable arm64 Pi image on a native arm64 runner and attaches it (with a
sha256) to the GitHub Release. No local image build required.

## License

GhOSt is MIT-licensed ([LICENSE](LICENSE)). By contributing you agree your
contributions are licensed under the same terms.
