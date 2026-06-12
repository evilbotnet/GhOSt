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
ghostd: internal/ai — agent loop + ROUTER in Go
  │  tools = the daemon's own subsystems:
  │    fs.list/read/write/trash · term.exec (gated) · system.wifi/volume
  │    browser.open · apps.launch · screenshot
  ├─ router: picks a capability tier per request (rules, configurable)
  └─ provider interface (N named providers, any mix):
       · "local"  — llama.cpp server on-device (OpenAI-compatible)
       · "lan"    — vLLM / Ollama on another box (OpenAI-compatible)
       · "cloud"  — Anthropic API (Go SDK anthropic-sdk-go) or any
                    OpenAI-compatible cloud endpoint
```

- **The agent loop runs in the daemon** (manual tool-use loop, not a runner):
  every mutating tool call emits a confirmation card to the shell over the WS
  and blocks until the user approves — the same UX pattern as app permission
  prompts. Read-only tools (list files, get status) auto-allow.
- **Everything is configuration** (`~/.config/ghost/ai.toml`, editable in
  Settings → Ghost): providers are named entries with a type, URL, model, and
  optional key (0600, never leaves the device except to the endpoint *you*
  configured). Ghost is **off until configured** — an open-source OS must not
  phone home by default.

```toml
[ai]
enabled = true

[ai.providers.local]
type  = "openai-compatible"          # llama.cpp --server on-device
url   = "http://127.0.0.1:8090/v1"
model = "qwen3-1.7b-q4"

[ai.providers.lan]
type  = "openai-compatible"          # vLLM or Ollama on a beefier box
url   = "http://192.168.64.10:8000/v1"
model = "qwen3-32b"

[ai.providers.cloud]
type  = "anthropic"
model = "claude-opus-4-8"            # or claude-haiku-4-5 for budget
key_file = "~/.config/ghost/anthropic.key"

[ai.routing]
intent   = "local"   # who parses single-shot commands ("" = rules only)
agent    = "lan"     # who runs the multi-step tool loop ("" = disabled)
fallback = "cloud"   # when `agent` is unreachable ("" = fail closed)
```

## The router: route by capability tier, not by model

The honest constraint: ~1B-class local models are decent at *intent parsing*
but unreliable at *multi-step tool use*. So the router's job is not "pick a
model" — it's "decide which kind of work this is", with deterministic rules
(cheap, predictable, auditable) rather than model-judged routing in v1:

1. **Command tier** (always available, no escalation): the request maps to a
   single known OS action. The `intent` provider does constrained JSON-schema
   decoding (llama.cpp grammar) into one tool call — "volume 40", "open wifi
   settings", "launch the editor". No conversation, instant, fully offline.
   With no local model configured, a rules/fuzzy matcher covers the basics.
2. **Agent tier** (escalates to `routing.agent`): anything multi-step, needing
   more than one tool, needing reasoning over file contents, or longer than a
   command ("organize my Downloads", "clone and test this repo"). The full
   confirmation-gated tool loop runs against the configured LAN/cloud
   provider. If none is configured, Ghost says exactly that instead of
   pretending.
3. **Explicit override**: prefixing with "ask <provider>" pins the request;
   Settings can also force "never leave this device" (offline mode = command
   tier only).

Every Ghost reply carries a provenance badge in the panel — *answered by
local/qwen3-1.7b* vs *answered by cloud/claude* — so routing is never
invisible. Router decisions append to a local log the user can read.

## What it can do on day one (all existing daemon capabilities)

"Organize my Downloads into folders by type" · "connect to the café wifi" ·
"find the doc where I wrote about the Pi build and open it" · "clone this
repo and run its tests, tell me what fails" (term.exec, gated) · "dim it,
play time is over" (volume) · "open three browser windows: news, mail, music".

## Reality check: local models on a Pi 400

4 GB shared with Chromium leaves room for ~1B-class quantized models via
llama.cpp — fine for command-tier intent parsing ("open settings → wifi"),
not for real agentic work. Hence the router above; in tier terms:

| Tier | Backend | What it powers |
| --- | --- | --- |
| 0 (always, free) | no LLM | fuzzy launcher/search, deterministic commands |
| 1 (optional, local) | ~1B model on-device, llama.cpp | command tier: NL → one tool call, offline |
| 2 (optional, BYO) | LAN vLLM/Ollama or Anthropic API | agent tier: the multi-step tool loop |

The on-device llama.cpp server is itself optional and **stopped when idle**
(same socket-activation trick as CryptPad) so tier 1 costs zero RAM until the
first Ghost invocation.

## Why this is the differentiator

- Open source end-to-end: the harness, the tool definitions, the prompts —
  all inspectable. The counter-positioning to Copilot+ / Gemini-in-ChromeOS
  is *auditable AI*: you can read exactly what the assistant can touch.
- The permission system (ADR 0001) and Ghost share one mechanism: scoped
  tokens + user-visible grants. Ghost is just another principal.
- Apps (Layer 2) can *expose* tools to Ghost via their manifest later —
  an agentic app ecosystem nobody else has on a $70 computer.
