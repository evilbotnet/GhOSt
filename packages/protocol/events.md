# WebSocket protocol

One multiplexed socket: `GET /api/v1/ws?token=<session-token>`.

Envelope (both directions):

```json
{ "topic": "term.04ac…", "event": "data", "payload": "…" }
```

Client → server events: `subscribe` / `unsubscribe` (any topic; a topic ending
in `.` subscribes to the prefix), plus topic-specific events below.

| Topic | Direction | Events |
| --- | --- | --- |
| `term.<id>` | s→c | `data` (string), `exit` |
| `term.<id>` | c→s | `input` (string), `resize` `{cols, rows}` |
| `system` | s→c | `status` (SystemStatus, every 5 s) |
| `windows` | s→c | `list` (native toplevels — Phase 2) |
| `fs.watch.<id>` | s→c | `change` (directory watch — planned) |
| `updates` | s→c | `progress` (apt/PackageKit — Phase 5) |

Slow consumers have frames dropped rather than blocking publishers; state
topics (`system`, `windows`) re-send full snapshots so drops self-heal.
