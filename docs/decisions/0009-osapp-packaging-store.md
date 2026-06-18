# ADR 0009 — `.osapp` packaging and the store

Status: accepted (built) · Deepens [ADR 0001](0001-app-platform.md) Layer 2 ·
relates to [ADR 0006](0006-app-ghost-tools.md)

## The question

[ADR 0001](0001-app-platform.md) chose the three-layer app platform and named
`.osapp` (Layer 2) as "the real platform": a zip + manifest + scoped token,
distributed via a git-index store. This ADR fixes the *details* that make it
buildable and safe — package format, install/verify, the permission grant, the
sandbox, and the store index + trust model — without which "Layer 2" is a
slogan.

## Package format

`name-version.osapp` is a zip rooted at the app id:

```
tone.studio/
  manifest.json
  index.html          # entry, loaded in the app's own origin-path
  icon.svg
  assets/…
```

`manifest.json` (extends the ADR 0001 sketch):

```jsonc
{
  "id": "tone.studio",            // reverse-DNS, immutable identity
  "name": "Tone Studio",
  "version": "0.3.1",             // semver
  "entry": "index.html",
  "icon": "icon.svg",
  "window": { "w": 900, "h": 620, "min": { "w": 480, "h": 360 } },
  "permissions": ["fs:home:rw", "system:audio"],
  "ghostTools": [                  // optional — see ADR 0006
    { "name": "export_wav", "description": "Export the current track to a WAV",
      "mutating": true }
  ],
  "author": "…", "license": "MIT", "source": "https://…"
}
```

`id` is identity (collisions are rejected at install); `version` drives
upgrades. Two new fields beyond ADR 0001: `window.min`, and `ghostTools` —
tools the app will register over the [ADR 0006](0006-app-ghost-tools.md)
channel, *declared up front* so they appear in the permission prompt (an app
teaching Ghost a `mutating` action is exactly as sensitive as a permission).

## Install, verify, run

1. **Acquire** — from the store index or a local file. ghostd downloads to a
   temp dir.
2. **Verify** — SHA-256 must match the index entry (or, for sideloads, the user
   is shown the hash and an "untrusted source" warning). Optional publisher
   signature (see trust model).
3. **Unpack** — to `~/.local/share/ghost/apps/<id>/`, atomically (unpack to
   `<id>.incoming`, then rename). Reject zip-slip paths and absolute/`..`
   members.
4. **Grant** — the shell renders a permission prompt from `permissions` +
   `ghostTools`; the daemon stores the grant keyed by `id`. No grant → no
   install.
5. **Serve & run** — ghostd serves the app at `/apps/<id>/` and mints it a
   **scoped token** carrying only its granted scopes. The app is an in-shell
   window (iframe to its origin-path) using the same REST/WS API as built-ins —
   built-ins are just apps with all scopes (the symmetry from ADR 0001).

## Permission enforcement (the load-bearing part)

Scopes ride the auth layer GhOSt already has. The per-app token encodes its
scopes; every privileged handler checks the scope, not just token validity:

- `fs:home:ro|rw` — the fs API confines to `$HOME` *and* checks read vs write.
- `system:audio|wifi|power` — narrows the `system` API per capability.
- `term:exec` — gates the pty API (most sensitive; off by default, prominent in
  the prompt).
- `net:fetch` — proxied outbound HTTP (so an app's network use is auditable),
  vs. raw access.
- `gpio:rw` — the [ADR 0008](0008-gpio.md) tools.
- `shell:theme|tray` — Layer-2 extensions (`type: "extension"`).

An app's `ghostTools` are registered under its scoped token, so a tool an app
exposes to Ghost can do no more than the app itself — and `mutating` ones still
hit the confirmation gate. This closes the obvious hole (an app escalating via
a tool it hands to Ghost).

## The store

**Index, not a server.** A signed git repo (`ghost-store`) holds
`index.json`: a list of `{id, name, version, description, icon, download,
sha256, permissions, author, signature?}`. ghostd fetches it, the Hub renders a
catalog, install runs the flow above. No GhOSt-operated backend, no account —
the index is just data, mirrorable and forkable (the "no Google overlords"
thesis applies to our own store too).

**Trust model, staged:**
- v1 — **hash-pinned**: the index entry's `sha256` is authoritative; the index
  repo's integrity is the git history + whoever you chose to trust as its
  remote. Sideloads are allowed with a clear "untrusted" warning.
- v2 — **publisher signatures**: an optional `minisign`/`cosign` signature over
  the package, with the publisher key in the manifest and pinned on first
  install (TOFU); upgrades must match the pinned key. Defends against a
  compromised index swapping a hash.
- Curated vs. community indexes are just different remotes; an org can pin its
  own.

**Updates.** `version` per app; ghostd compares against the index on a schedule
(natural fit for [ADR 0007](0007-scheduled-ghost.md)) and offers upgrades.
Upgrade re-verifies hash/signature and *re-prompts only for newly requested
permissions* (a diff of the grant), so a silent permission grab is impossible.

## Why this shape

- **No new infra** — packaging rides the existing static-serve + token-scope +
  confirmation-gate machinery; the store is a git repo. The whole platform is
  data + the daemon we already have.
- **Honest symmetry** — third-party apps use the identical API surface as
  built-ins, so the platform can't rot into a second-class citizen.
- **The AI-native twist** — `ghostTools` make an installed app a first-class
  extension of *Ghost*, not just a window. Installing Tone Studio teaches Ghost
  "export the track" — bounded by the app's own scopes.

## As built

- **`daemon/internal/osapp`** — manifest + `Install` (SHA-256 verify, `..`/
  absolute/backslash zip-slip rejection, atomic `<id>.incoming`→rename), the
  grants store (`~/.local/share/ghost/apps/grants.json`), and the scope
  vocabulary + `Allows` (with `rw⊇ro`, `control⊇read` implication).
- **Scoped tokens** (`daemon/internal/httpapi`) — `auth` now resolves a token
  to a scope set (session token → `*`; per-app token → its grant) and enforces
  the scope each path requires via a central `scopeFor` table that **fails
  closed** (any unlisted path is shell-only). Each app is served its own token,
  injected into its entry HTML, minted on first serve and revoked on uninstall.
- **`daemon/internal/store`** — fetches `index.json` + `index.json.sig`,
  verifies the **Ed25519** signature against a pinned public key *before*
  trusting any entry, then installs by type (app → download+hash+`osapp`,
  skill/tool → safe-unzip into the config dir, mcp → `ai.AddMCPServer`).
  Publisher tooling: `ghostd store-keygen` / `ghostd store-sign`.
- **Hub → Store tab** — browse the verified catalog, one-click install (apps
  show a permission prompt before the grant), plus an Installed-packages list
  with uninstall.
- Tests: `osapp_test.go` (install/hash/zip-slip/scopes), `httpapi/scopes_test.go`
  (live auth enforcement: granted vs. denied vs. revoked), `store_test.go`
  (signed install + wrong-key + tampered-index rejection). Verified end-to-end
  through a running daemon (configure → verify → install → serve with scoped
  token → confined API access).

## App isolation: sandboxed opaque origin (resolved)

The scoped token confines a *cooperating* app, but path is not an origin
boundary — so a *malicious* app could otherwise reach the shell's window and
read its full-privilege session token. Closed by giving every app a real origin
boundary via a **CSP `sandbox` directive**, enforced by the daemon:

```
Content-Security-Policy: sandbox allow-scripts allow-forms allow-modals; frame-ancestors 'self'
```

served on `/apps/<id>/`. `sandbox` without `allow-same-origin` forces the
browser to give the app document a **unique opaque origin** — cross-origin to
*everything* — so it cannot reach `window.parent` (the shell) and the shell
cannot read into it. The shell also renders the app in an `<iframe sandbox>`
(belt and suspenders), but the header makes the isolation the *daemon's*
guarantee, holding even top-level or if the shell forgot the attribute.
`frame-ancestors 'self'` stops a hostile page from framing the app.

This was chosen over per-app loopback ports / `*.localhost` subdomains: the
opaque origin is the strongest isolation (cross-origin in both directions),
needs no Host routing and no dependency on `*.localhost` resolution (which
varies by browser/OS), and behaves identically in dev and on the Pi.

**Origin policy, split by principal** (`httpapi.auth`): the superuser session
token is honoured *only* from the shell's own origin (CSRF / DNS-rebinding
defense), while a per-app token — itself an unguessable, scope-bound capability,
arriving from the app's opaque origin as `Origin: null` — is not origin-gated;
the token + scope check is its boundary. An app token can never escalate beyond
its grant regardless of the Origin it claims.

**Tradeoff:** an opaque-origin document has no cookies / `localStorage` /
IndexedDB. This is acceptable — and arguably better: GhOSt apps persist through
the scoped `fs` API (auditable, permission-gated) rather than opaque browser
storage. Apps needing rich client storage are a future consideration (e.g. an
opt-in `storage` scope backed by a daemon-mediated per-app store).

Verified end-to-end in the browser: a test app reports `origin: null`, its
`window.parent` token read throws `SecurityError`, the shell's read of the
app's `contentDocument` is `null`, its granted `notify` call succeeds while its
ungranted `term` call is refused 403 — and `httpapi/scopes_test.go` covers the
origin rules (session origin-locked; app token exempt but unescalatable).

## Scope / open questions

Per-app storage quotas and a data-reset/uninstall-cleanup contract; whether
`net:fetch` proxying is worth the complexity in v1; multi-window apps; an
app-level "background" lifecycle (today an app is its window); publisher
signatures over individual packages (v2 — today the signed index pins each
package's hash). None block the v1 hash-pinned, signed-index store.
