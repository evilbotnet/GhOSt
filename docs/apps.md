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

## Giving Ghost new abilities

Ghost's tools live in `daemon/internal/ai/tools.go` — each is a name, a JSON
schema, a `run` function over the daemon's subsystems, and a `mutating` flag
(mutating ⇒ Allow/Deny card). Adding a tool there makes it available to every
configured model, with the confirmation gate enforced by the OS, not the
prompt.
