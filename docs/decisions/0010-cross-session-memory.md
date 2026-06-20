# ADR 0010 — Cross-session memory

Status: accepted (built) · Builds on [ADR 0002](0002-ghost-ai-assistant.md),
[ADR 0005](0005-ghost-extensibility.md)

## The question

Ghost's conversation history is per-session and in-memory — it forgets
everything on restart. An assistant that drives your OS should remember the
durable things: your name, that you prefer metric units, the project you're
working on. How should Ghost persist and use that, without becoming a privacy
hole or a context-bloat problem?

## Decision: short markdown facts, injected — not recalled

Memories live as markdown files in `~/.config/ghost/memory/<slug>.md`
(frontmatter `description`; body = the fact), exactly mirroring the skills
format ([ADR 0005](0005-ghost-extensibility.md)) so the same `parseFrontmatter`
loader serves both.

The load-bearing choice is **inject, don't recall.** Skills are progressively
disclosed (an index in the prompt; a `load_skill` tool pulls the body on
demand) because skill bodies are long and only sometimes relevant. Memories are
the opposite: short and *always* relevant. So the full memory set is written
straight into the system prompt under a "What you remember" section. A `recall`
tool the model might forget to call would defeat the purpose — a preference is
useless if the assistant doesn't reliably see it. Injection guarantees it does.

This works because memories are kept short by design (the `remember` tool stores
concise facts). If the set ever grows large enough to pressure the context
window, progressive disclosure becomes the fallback — but that's a scaling
problem we don't have at one-user-on-a-Pi scale.

## Control stays with the user

- **`remember` and `forget` are mutating tools** → they go through the
  confirmation gate ([ADR 0004](0004-ghost-implementation.md)). Ghost proposes
  "remember: <fact>"; nothing is persisted without an Allow. Proactive/headless
  runs ([ADR 0007](0007-scheduled-ghost.md)) can't write memory at all (mutating
  tools are declined there).
- **Everything is visible and editable** in **Hub → Memory** — list, add, and
  delete, backed by `GET/POST/DELETE /ai/memory`. No hidden state; the files are
  plain markdown the user can read or edit directly.
- Memory is local, like the rest of GhOSt — it never leaves the device.

## Implementation

- `daemon/internal/ai/memory.go` — `Memory` model, `LoadMemories`,
  `SaveMemory`/`DeleteMemory` (slugified filenames, atomic write),
  `memoryPromptSection` (full-body injection), and the `remember`/`forget`
  tools. Wired into `buildSystemPrompt` and `buildTools`.
- HTTP: `GET/POST/DELETE /ai/memory`; Hub Memory tab.
- `memory_test.go`: save/load/update-in-place/forget/slug + prompt injection.

## Scope / future

Auto-suggested memories (Ghost noticing "want me to remember that?"); a relevance
cap or progressive disclosure if the set grows; categories/tags; export with the
backup feature. None needed for v1 — the simple injected-facts model is the
honest fit for an OS assistant's durable preferences.
