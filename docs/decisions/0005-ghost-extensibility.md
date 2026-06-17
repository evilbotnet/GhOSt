# ADR 0005 — Ghost extensibility: skills and tools

Status: accepted (built) · Builds on [ADR 0002](0002-ghost-ai-assistant.md)

## The goal

GhOSt is AI-native: Ghost should grow new abilities without recompiling the
daemon, the way Claude Code grows via skills and MCP tools. Two orthogonal
extension points, mirroring the Anthropic model:

- **Skills** add *expertise* — how to do a multi-step task well.
- **Tools** add *capabilities* — new actions Ghost can take.

Both are drop-in: a file in a directory, picked up on the next Ghost run.
Nothing is compiled; nothing restarts.

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

## Why not MCP (yet)

MCP is the obvious future for *networked* tool servers and would let GhOSt
borrow the whole MCP ecosystem. But the manifest+exec model is simpler, has
zero runtime dependency, and covers the local-capability case that matters on
a Pi. MCP-server support is a clean future addition: another provider of tool
defs into the same agent loop. The model gateway in
[ADR 0003](0003-devkit-and-model-gateway.md) is the complementary half —
exposing *Ghost's* providers to other tools.

## Shipped defaults

A fresh GhOSt boots AI-native: one skill (`tidy-files`) and two tools
(`system_report`, `append_note`) ship in the image so Ghost is demonstrably
extensible out of the box and the formats have working examples to copy.
