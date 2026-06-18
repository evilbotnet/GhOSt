# ADR 0006 — App-provided tools (apps expose tools to Ghost)

Status: accepted (built) · Builds on [ADR 0002](0002-ghost-ai-assistant.md),
[ADR 0005](0005-ghost-extensibility.md)

## The question

ADR 0005 gave Ghost four extension points (soul, skills, tools, MCP), but all
of them are *static*: a file or config line, read at the start of a run, that
runs out-of-process (an executable, an MCP server). None let a **running app**
— a live window with state on screen — offer Ghost an action against *that
state*. "Open the Monitor", "jump to line 200 in the file I have open",
"export the doc I'm editing" all need the app that owns the state to do the
work, while it's running.

So: how does a running app expose a tool Ghost can call, and get the result
back into Ghost's loop?

## Decision: register tools over the WebSocket; invoke is request/response

The shell already holds one authenticated WebSocket to ghostd, and Ghost
already does request/response over it — the confirmation gate emits a
`confirm_request` and blocks on the user's `confirm`. App tools reuse exactly
that shape on a new topic, **`ghosttools`**:

```
app  → daemon   register  { tools: [ {name, description, properties,
                                       required, mutating}, ... ] }
daemon → app    invoke    { callId, name, args }
app  → daemon   result    { callId, output, error }
```

- **register** — an app announces its tool *schemas* (JSON-Schema-shaped, same
  as every other Ghost tool). The daemon keeps the current set; re-registering
  replaces it. The shell re-sends on every WS (re)connect, so a daemon restart
  or dropped socket self-heals.
- **invoke** — when Ghost calls an app tool, the daemon publishes `invoke` to
  the WS and blocks the tool's `run` on a per-`callId` channel (30s timeout),
  the same machinery as `confirm`.
- **result** — the app runs the tool locally (it has the window, the state, the
  DOM) and sends `output` or `error` back; the daemon unblocks `run` and feeds
  the result into Ghost's normal tool loop.

Mutating app tools are still **Allow/Deny-gated** like any other mutating tool
— the gate runs in the daemon before `invoke` is ever published, so an app
cannot bypass confirmation by living client-side.

## Why over the WS, not HTTP callbacks

The app is a browser tab — it can't be dialed by the daemon over HTTP, only
pushed to over the socket it already owns. The WS is authenticated once at
connect, multiplexed, and reconnect-aware; piggybacking on it means no new
auth surface, no new port, and the dead-app case is just "the socket closed".

## Implementation

- Daemon: `daemon/internal/ai/clienttools.go` — `clientToolReg` registers a
  `ghosttools` prefix handler on the hub, holds the current tool defs, and
  exposes `tools()` (merged into Ghost's tool map per run, alongside built-ins,
  skills, ext-tools, and MCP) and `invoke()` (publish + await on a `callId`
  channel). Wired into `Ghost` in `ghost.go`.
- Shell: `apps/shell/src/lib/ghost/clienttools.ts` — the reference client.
  Registers on connect (via a new `onOpen` hook in `api/ws.ts`), answers
  `invoke` frames, and ships two real tools so Ghost can drive the desktop:
  - `list_apps` — enumerate the installable desktop apps + ids
  - `open_app {id}` — open one (Ghost: "open the system monitor" → window opens)
- Verified end-to-end over a live WebSocket in `clienttools_test.go`
  (register → invoke → result → output returned to the caller).

## Scope / future

v1 assumes a **single registering client** (the shell). The protocol carries no
notion of *which* app owns a tool, so two apps registering the same tool name
collide and `invoke` is broadcast to all `ghosttools` subscribers. That is the
correct shape for today (one shell, in-shell apps share its socket) and the
clean extension is per-client routing: tag each registration with the
connection id and address `invoke` to that client. An `.osapp`
([ADR 0001](0001-app-platform.md)) gets the same channel for free — this is how
a third-party app will teach Ghost to use it.
