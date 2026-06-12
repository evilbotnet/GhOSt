# ADR 0001 — App platform: how users install and extend OpenOS

Status: accepted (design) · Target: Phase 6

## The question

A browser with system access becomes an *operating system* when it offers a
contract to third parties: identity, lifecycle, and permissions for software
the OS author never saw. ChromeOS has PWAs/Android/Linux-VMs; we need our own
answer or OpenOS stays an appliance.

## Decision: three layers, cheapest first

**Layer 1 — Web apps as first-class apps (almost free).**
Any URL can be "installed": the shell stores `{name, url, icon}` and launches
it via the daemon as `chromium --app=<url>` — its own compositor window, its
own taskbar entry, no tabs. Launcher gains "Install web app…". This instantly
makes the entire web our app catalog, exactly like ChromeOS's PWA story.

**Layer 2 — OpenOS packages (`.osapp`): the real platform.**
A zip with a manifest, installed to `~/.local/share/openos/apps/<id>/`:

```json
{
  "id": "tone.studio",
  "name": "Tone Studio",
  "version": "0.3.1",
  "icon": "icon.svg",
  "entry": "index.html",
  "window": { "w": 900, "h": 620 },
  "permissions": ["fs:home:rw", "term:exec", "system:audio"]
}
```

- osd serves installed apps at `http://127.0.0.1:7700/apps/<id>/` and issues
  each app a **scoped token** carrying only its granted permissions — the
  permission model rides on the auth layer we already built (the fs/term/
  system APIs check scopes, not just token validity).
- First launch shows a permission prompt rendered by the shell (the data is in
  the manifest; the grant is stored by the daemon).
- Apps run as in-shell windows (iframe to their own origin-path) and talk to
  the same REST/WS API the built-in apps use — built-ins are just apps with
  all scopes. That symmetry keeps the platform honest.
- Distribution v1: a git-repo index (name → download URL + hash), `osd`
  verifies the hash. A "store" UI is sugar on top, later.

**Layer 3 — It's Debian underneath (already true).**
`apt` + the terminal give power users the entire Linux software universe.
Native Wayland GUI apps already appear as compositor windows that our taskbar
tracks via foreign-toplevel-management — so `apt install gimp` just works™
(modulo memory). This is our equivalent of ChromeOS's Crostini, except it's
not a VM — it's the actual system, because the user owns the machine.

Shell extensions (themes, tray widgets) reuse the Layer-2 package format with
`type: "extension"` and a narrow `shell:*` permission namespace.

## Why not Chromium extensions?

They extend the *browser*, not the OS, are blocked by our kiosk policy for
good reasons (kiosk-escape surface), and would couple the platform to
Chromium internals. Our API boundary is the daemon, so our extension point is
the daemon's API.
