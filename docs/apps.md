# Adding applications to GhOSt

Four routes, in order of effort. (Background: [ADR 0001](decisions/0001-app-platform.md).)

## 1. Install a web app (no code)

Launcher → **Install web app** → paste a URL. The daemon stores it in
`~/.config/ghost/webapps.json` and launches it as a chromeless
`chromium --app` window — its own taskbar entry, no tabs, feels native.
Anything with a good web client (Excalidraw, Proton Mail, a music player,
your home dashboard) is one paste away. Uninstall via
`DELETE /api/v1/apps/{id}` (Settings UI for this is on the list).

## 2. Write a built-in shell app (web tech, full citizen)

Built-in apps are Svelte components that run inside the shell's own window
manager and talk to the system through `ghostd`'s typed API client. Two steps:

**a. The component** — `apps/shell/src/lib/apps/hello/Hello.svelte`:

```svelte
<script lang="ts">
  import { api } from '../../api/client';
  import type { Win } from '../../wm/wm.svelte';

  let { win }: { win: Win } = $props();   // your window: set win.title, etc.
  let host = $state('…');
  api.get<{ hostname: string }>('/system/status').then((s) => (host = s.hostname));
</script>

<div style="padding:20px">Hello from {host}!</div>
```

**b. The registry entry** — add to `apps/shell/src/lib/apps/registry.ts`:

```ts
import Hello from './hello/Hello.svelte';
// …in the apps array:
{
  id: 'hello', name: 'Hello', icon: 'info',
  component: Hello,
  defaultSize: { w: 420, h: 300 },
  minSize: { w: 280, h: 180 },
},
```

That's it — launcher tile, window chrome, taskbar entry, drag/resize all come
from the shell. Available to your app:

- `api` (`lib/api/client.ts`) — typed REST: `/fs/*`, `/term`, `/system/*`,
  `/browser/open`, `/apps/*` … (see `packages/protocol/openapi.yaml`)
- `subscribe(topic, fn)` (`lib/api/ws.ts`) — live streams: `term.<id>`,
  `system`, `windows`, `ai.<session>`
- `Icon.svelte` glyphs, theme tokens (`lib/theme/tokens.css`) — use the CSS
  custom properties so themes apply to your app for free
- Needs a new system capability? Add an endpoint to
  `daemon/internal/httpapi/server.go` backed by a small package under
  `daemon/internal/` — that's how every built-in works. If it genuinely needs
  root, it goes through a new validated verb in `daemon/internal/admin`
  (never sudo from the daemon).

Iterate with `./scripts/dev.sh` (hot reload on :5173), ship with
`./scripts/deploy-vm.sh` or an image rebuild.

## 3. Install Linux software (it's Debian underneath)

Open the Terminal: `sudo apt install <anything>`. Native Wayland GUI apps
appear as compositor windows that the taskbar already tracks. CLI tools just
work in the Terminal. This is the escape hatch that makes GhOSt a real
computer and not an appliance — see also `sudo ghost-install-office`
(CryptPad) and, soon, the devkit
([ADR 0003](decisions/0003-devkit-and-model-gateway.md)).

## 4. `.osapp` packages (planned)

The third-party contract: a zip with a manifest declaring
`permissions: ["fs:home:rw", …]`, served by ghostd with a **scoped token** so
an app can only call the API surface the user granted at install. Design in
[ADR 0001](decisions/0001-app-platform.md); the auth layer it rides on
already exists.

## Extending Ghost (the AI layer)

Ghost grows the way Claude Code does — drop-in **skills** (expertise) and
**tools** (capabilities), no recompile. Full design: [ADR 0005](decisions/0005-ghost-extensibility.md).

### A skill — teach Ghost a workflow

A folder with a `SKILL.md` (YAML frontmatter + markdown instructions). Only
the description sits in the prompt; Ghost calls `load_skill` to read the body
when a task matches — progressive disclosure.

```
~/.config/ghost/skills/release-notes/SKILL.md
```
```markdown
---
name: release-notes
description: Draft release notes from recent git commits. Use when the user asks for a changelog or release notes.
---
1. Run the `git_log` tool (or read the repo) to get recent commits.
2. Group them into Features / Fixes / Docs.
3. Write tight, user-facing bullets — no commit hashes.
```

### A tool — give Ghost a new action

A JSON manifest + an executable. Args arrive as `$GHOST_ARG_<KEY>` (scalars),
or full JSON on stdin / in `$GHOST_TOOL_ARGS`. `mutating: true` ⇒ Allow/Deny
card before it runs.

```
~/.config/ghost/tools/weather.tool.json   +   weather.sh
```
```json
{
  "name": "weather",
  "description": "Current weather for a city. Call when the user asks about weather.",
  "mutating": false,
  "command": ["sh", "weather.sh"],
  "inputSchema": { "properties": { "city": {"type":"string"} }, "required": ["city"] }
}
```

Both appear in the Ghost panel's extensions footer and are offered to whatever
model the user configured. Working examples ship in the image — see
[os/overlay/usr/share/ghost/](../os/overlay/usr/share/ghost/).

### A built-in tool (compiled, for core OS capabilities)

Core tools live in `daemon/internal/ai/tools.go` — a name, JSON schema, a
`run` over the daemon's subsystems, and a `mutating` flag. Use this tier when
the capability is part of the OS itself (files, browser, system) rather than a
user add-on. Same confirmation gate, enforced by the OS, not the prompt.
