# ADR 0002 — "Ghost": the AI layer that makes GhOSt unique

Status: accepted (design) · Target: Phase 7 (after the app platform)

## The thesis

Every OS is bolting AI on as a chat sidebar. GhOSt can do something none of
them can do honestly: **the assistant's tool surface IS the operating
system's API.** Our daemon already exposes files, terminal, settings, windows,
Wi-Fi, and app launch as a clean, localhost, permission-gated API — which is
*exactly* the shape an agentic LLM needs. We don't integrate AI into the OS;
the OS was accidentally designed as an agent harness.

Naming: the OS is **GhOSt** (**G**o · **h**tml · **O**perating **S**ystem ·
**t**ypescript); the assistant is **Ghost** — the ghost in the shell that the
name promises. The daemon `ghostd` hosts it.

## Architecture

```
Shell: Ghost panel (Super+Space command palette / docked sidebar)
  │  WS topic ai.<session> (stream tokens, tool-call cards, confirmations)
ghostd: internal/ai — agent loop in Go
  │  tools = the daemon's own subsystems:
  │    fs.list/read/write/trash · term.exec (gated) · system.wifi/volume
  │    browser.open · apps.launch · screenshot
  └─ provider interface:
       1. Anthropic API (Go SDK anthropic-sdk-go, tool use + streaming)
       2. Any OpenAI-compatible endpoint (Ollama on the LAN, llama.cpp local)
```

- **The agent loop runs in the daemon** (manual tool-use loop, not a runner):
  every mutating tool call emits a confirmation card to the shell over the WS
  and blocks until the user approves — the same UX pattern as app permission
  prompts. Read-only tools (list files, get status) auto-allow.
- **Provider is a setting, not a dependency.** BYO Anthropic key (stored 0600
  in `~/.config/ghost/`, never leaves the device except to the API you chose)
  or point it at a local/LAN model. Ghost is **off until configured** — an
  open-source OS must not phone home by default.
- Model defaults when using the Anthropic provider: `claude-opus-4-8` for
  quality, `claude-haiku-4-5` as the budget/latency option — user-selectable
  in Settings. Adaptive thinking on; streaming always (tokens render live in
  the panel).

## What it can do on day one (all existing daemon capabilities)

"Organize my Downloads into folders by type" · "connect to the café wifi" ·
"find the doc where I wrote about the Pi build and open it" · "clone this
repo and run its tests, tell me what fails" (term.exec, gated) · "dim it,
play time is over" (volume) · "open three browser windows: news, mail, music".

## Reality check: local models on a Pi 400

4 GB shared with Chromium leaves room for ~1B-class quantized models via
llama.cpp — fine for command-palette intent parsing ("open settings → wifi"),
not for real agentic work. So the honest tiering is:

| Tier | Backend | What it powers |
| --- | --- | --- |
| 0 (always, free) | no LLM | fuzzy launcher/search, deterministic commands |
| 1 (optional, local) | ~1B model on-device | natural-language command parsing, offline |
| 2 (optional, BYO) | LAN Ollama / Anthropic API | full Ghost: multi-step agent with tools |

## Why this is the differentiator

- Open source end-to-end: the harness, the tool definitions, the prompts —
  all inspectable. The counter-positioning to Copilot+ / Gemini-in-ChromeOS
  is *auditable AI*: you can read exactly what the assistant can touch.
- The permission system (ADR 0001) and Ghost share one mechanism: scoped
  tokens + user-visible grants. Ghost is just another principal.
- Apps (Layer 2) can *expose* tools to Ghost via their manifest later —
  an agentic app ecosystem nobody else has on a $70 computer.
