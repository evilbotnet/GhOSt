# Skills & tools gallery

Ghost grows new abilities without recompiling the daemon (see
[ADR 0005](decisions/0005-ghost-extensibility.md)):

- **Skills** add *expertise* — how to do a multi-step task well. A folder + a
  `SKILL.md`; no code. Ghost loads the right one on demand (progressive
  disclosure via the `load_skill` tool).
- **Tools** add *capabilities* — a new local action. A JSON manifest + an
  executable in any language. Mutating tools are confirmation-gated like
  everything else.

Both are drop-in. Shipped ones live in `/usr/share/ghost/{skills,tools}`; your
own go in `~/.config/ghost/{skills,tools}` and are picked up on the next Ghost
run. Manage and browse them in the **Hub**.

---

## Shipped skills

| Skill | What it does |
| --- | --- |
| **tidy-files** | Sort a messy folder into subfolders by type (Images, Documents, Archives, Code, Other). |
| **clean-downloads** | Audit `~/Downloads` — flag stale, large, and duplicate-looking files — then optionally tidy. |
| **organize-screenshots** | File screenshots in `~/Pictures` into per-month subfolders. |
| **summarize-folder** | Read a folder's text files and write a concise summary, optionally saved as an index. |
| **disk-checkup** | Find what's eating disk space in your home folder and advise — read-only, you decide. |
| **capture-idea** | Jot a thought/todo into your notes, lightly cleaned up and timestamped. |
| **draft-document** | Draft a letter, README, plan, or outline from your request and save it to `~/Documents`. |

## Shipped tools

| Tool | Mutating | What it does |
| --- | --- | --- |
| **system_report** | no | Quick health report: uptime, load, memory, disk. |
| **append_note** | yes | Append a timestamped note to `~/Documents/notes.md`. |

---

## Add a skill

A skill is a directory with a `SKILL.md`. The frontmatter `name` + `description`
is what Ghost sees when deciding whether to load it, so make the description say
*when* to use the skill. The body is the instructions Ghost follows.

```
~/.config/ghost/skills/my-skill/
└── SKILL.md
```

```markdown
---
name: my-skill
description: One sentence on what this does AND when to use it — the triggers
  the user might say ("when the user asks to …").
---

# Human-readable title

Step-by-step instructions for Ghost, in plain language. Reference the OS tools
by name (list_files, read_file, make_dir, move_file, write_file, open_browser,
set_volume, system_status, …). Mutating steps each show the user a confirmation
card — propose them; don't ask permission in prose.
```

Tips: keep it to the task; name the exact tools to call; spell out edge cases
("never move a folder"); end with a one-line summary step. See
`os/overlay/usr/share/ghost/skills/tidy-files/SKILL.md` for a complete example.

## Add a tool

A tool is a manifest (`<name>.tool.json`) plus an executable in the same folder.
The manifest's `inputSchema` advertises the arguments; `mutating: true` routes
it through the confirmation gate.

```
~/.config/ghost/tools/
├── greet.tool.json
└── greet.sh        (chmod +x)
```

```json
{
  "name": "greet",
  "description": "Say hello to someone. Use when the user asks to greet a name.",
  "mutating": false,
  "command": ["sh", "greet.sh"],
  "timeoutSec": 10,
  "inputSchema": {
    "properties": { "name": { "type": "string", "description": "who to greet" } },
    "required": ["name"]
  }
}
```

Arguments arrive **three ways** so any language is easy — pick one:

- full JSON on **stdin**
- full JSON in **`$GHOST_TOOL_ARGS`**
- each scalar as **`$GHOST_ARG_<NAME>`** (uppercased), e.g. `$GHOST_ARG_NAME`

```sh
#!/bin/sh
echo "Hello, ${GHOST_ARG_NAME}!"
```

The command's working directory is the manifest's folder, and stdout is returned
to Ghost. See `os/overlay/usr/share/ghost/tools/` for the shipped examples.

## Share it

- **Contribute it to GhOSt** — open a PR adding your skill/tool under
  `os/overlay/usr/share/ghost/...` so it ships in the image. Add a row to the
  tables above.
- **Publish to a store** — list it in a signed git-index store and anyone can
  one-click-install it from the Hub
  ([ADR 0009](decisions/0009-osapp-packaging-store.md)). Publishers sign the
  index with `ghostd store-keygen` / `ghostd store-sign`.
