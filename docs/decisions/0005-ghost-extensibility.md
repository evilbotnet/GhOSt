# ADR 0005 — Ghost extensibility: soul, skills, tools, MCP

Status: accepted (built) · Builds on [ADR 0002](0002-ghost-ai-assistant.md)

## The goal

GhOSt is AI-native: Ghost should grow new abilities — and a personality —
without recompiling the daemon, the way Claude Code grows via skills and MCP.
Four extension points:

- **Soul** sets *personality* — who the assistant is (name + persona).
- **Skills** add *expertise* — how to do a multi-step task well.
- **Tools** add *capabilities* — new local actions Ghost can take.
- **MCP servers** add *ecosystems* — whole toolsets over the Model Context Protocol.

All are drop-in: a file in a directory (or a config line), picked up on the
next Ghost run. Nothing is compiled; nothing restarts.

## Soul (personality, à la Hermes / OpenClaw)

`~/.config/ghost/SOUL.md` — frontmatter `name` + a markdown persona body,
written by the user when they "hatch" Ghost during onboarding (pick an
archetype — Ghost / Hermes / Sentinel / Sage — name it, add traits), or any
time in Settings → Ghost AI. The soul leads the system prompt (identity +
persona), then the operating rules, then the skills list. The hatched name
replaces "Ghost" throughout the shell. An unhatched install is just "Ghost"
with a neutral voice.

## Skills (expertise, progressively disclosed)

A skill is a directory with a `SKILL.md`: YAML frontmatter (`name`,
`description`) and a markdown body of instructions.

```
~/.config/ghost/skills/tidy-files/SKILL.md     # user skills
/usr/share/ghost/skills/<name>/SKILL.md        # shipped defaults
```

Only each skill's **name + description** sit in Ghost's system prompt (cheap).
A built-in read-only `load_skill` tool returns the full body, which Ghost
calls when a task matches a skill's description — progressive disclosure,
exactly like Anthropic Skills. The body can reference bundled files in the
skill dir; Ghost reads/runs them with its other tools. User dir wins on a
name collision with a shipped skill.

## Tools (capabilities, manifest + executable)

A tool is a JSON manifest naming an executable:

```json
{
  "name": "append_note",
  "description": "Append a timestamped note… Call when the user asks to jot down a note.",
  "mutating": true,
  "command": ["sh", "append-note.sh"],
  "timeoutSec": 10,
  "inputSchema": { "properties": { "text": {"type":"string"} }, "required": ["text"] }
}
```

The daemon registers it alongside the built-in OS tools. On a call it runs
`command` (cwd = the manifest's dir) and returns stdout. Args reach the
program three ways so any style is trivial: full JSON on **stdin**, the same
in **`GHOST_TOOL_ARGS`**, and each scalar as **`GHOST_ARG_<KEY>`** (so a shell
script just reads `$GHOST_ARG_TEXT`). Output is captured, size-capped, and
timed out. `mutating: true` ⇒ the same Allow/Deny confirmation card as any
built-in mutating action.

Searched in `/usr/share/ghost/tools` (defaults) then `~/.config/ghost/tools`
(user, wins on name).

## Security

External tools run as the `ghost` user — the same privilege as the Terminal
app, which is already a full shell by design. The boundary that matters is
unchanged: **mutating actions are gated by the OS, not the prompt**, and the
confirmation card shows the exact tool + arguments before anything runs. A
skill is only instructions; it can't do anything its reader's tools can't
already do (and those mutations are still gated). The lock screen remains the
real perimeter.

## MCP servers

Users add MCP servers in `ai.toml`:

```toml
[[ai.mcp_servers]]
name = "filesystem"
transport = "stdio"
command = ["npx","-y","@modelcontextprotocol/server-filesystem","/home/ghost"]
enabled = true
```

ghostd is itself a minimal MCP client (stdlib JSON-RPC 2.0 over stdio — no SDK
dependency; the wire protocol is small and stable). On a Ghost run it connects
each enabled server (cached, reconnected on death — not respawned per call),
lists its tools, and surfaces them as `mcp__<server>__<tool>`, mutating unless
the tool's `readOnlyHint` says otherwise — so they ride the **same Allow/Deny
gate** as everything else. A dead or misconfigured server is surfaced via
`/ai/mcp` and never breaks the loop. Streamable-HTTP transport is a documented
TODO; stdio covers the npx-server case that matters today. The model gateway
in [ADR 0003](0003-devkit-and-model-gateway.md) is the complementary half —
exposing *Ghost's* providers to other tools.

## Shipped defaults

A fresh GhOSt boots AI-native: seven skills (tidy-files, clean-downloads,
summarize-folder, organize-screenshots, capture-idea, disk-checkup,
draft-document) and two tools (`system_report`, `append_note`) ship in the
image, so Ghost is demonstrably extensible out of the box and every format has
a working example to copy. MCP servers ship none by default (the user adds
them) — an open-source OS shouldn't spawn subprocesses nobody asked for.

## Verification

All four verified against a self-hosted Qwen3-35B (LAN vLLM): the model loaded
and followed a skill (8 files → 5 buckets, each move gated); ran a read-only
external tool and a mutating one (gated, wrote notes.md); called an MCP tool
(`mcp__filesystem__list_allowed_directories`) on a live npx-spawned server; and
the hatched persona (Hermes) carried into the prompt and the shell.
