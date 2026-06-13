# ADR 0004 — Ghost & app platform: what shipped (Phases 6–7)

Status: implemented · Supersedes the design sketches in ADR 0001/0002 with
the as-built shape.

## Ghost (Phase 7)

The agent loop lives in `daemon/internal/ai`, hosted by ghostd:

- **Provider-agnostic core** (`provider.go`): one `LLM` interface, two
  implementations — `anthropic.go` (official anthropic-sdk-go, tool use) and
  `openai.go` (chat/completions dialect: Ollama, vLLM, llama.cpp). The wizard/
  Settings write `ai.toml`; `config.go` resolves `routing.agent` to a provider.
- **Tools = the OS API** (`tools.go`): list/read/write/move/trash/mkdir files,
  open browser, set volume, system status. Read-only tools auto-run; every
  `mutating` tool is gated.
- **Confirmation gate** (`ghost.go`): the loop emits `confirm_request` over WS
  topic `ai.<session>` and blocks until the shell sends `confirm {allow}`
  (2-min default-deny). Same trust mechanism as app permissions.
- **Shell**: `GhostPanel.svelte` (Super+Space / taskbar), streams the trace,
  renders Allow/Deny cards, shows a provenance badge (which model answered).
  Settings → Ghost AI reconfigures routing live.

Verified against a real LAN vLLM (Qwen3.6-35B): read-only ask, mutating ask
with allow (folder created), mutating ask with deny (folder NOT created, model
adapts). Provenance badge shows `lan / Qwen/...`.

## App platform (Phase 6 — Layer 1 only)

`daemon/internal/webapps`: install any URL as a first-class app (stored in
`~/.config/ghost/webapps.json`), launched as a chromeless `chromium --app`
window with its own taskbar entry. Launcher gains "Install web app" + the
installed apps. Layers 2 (.osapp packages + scoped tokens) and 3 (apt/native)
remain future work per ADR 0001 — Layer 1 already makes the whole web the
app catalog.
