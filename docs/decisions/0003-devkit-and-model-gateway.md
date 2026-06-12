# ADR 0003 — Devkit (pi, Herdr) and ghostd as the system model gateway

Status: accepted (design) · Target: devkit in Phase 4/5 (image + OOBE),
gateway in Phase 7 (with Ghost)

## The question

Should GhOSt bake in support for existing terminal AI coding agents — pi
(pi.dev, MIT, npm, 15+ providers incl. Anthropic/Ollama) and Herdr
(herdr.dev, open source, the "tmux for coding agents") — for immediate
functionality on day one?

## Decision: bundle, don't bake

They already run on GhOSt with zero work — the Terminal app is a real pty on
real Debian. So the OS core gains nothing by hard-depending on them, and
fast-moving third-party agents churn faster than OS images. Instead, three
cheap integrations make them feel native:

**1. The devkit (optional, one checkbox).**
A `ghost-devkit` install script (and an OOBE/Settings toggle): installs
Node (already present once CryptPad ships), `@earendil-works/pi-coding-agent`,
and Herdr. Image builds can pre-bake it with a pi-gen flag; default image
stays lean.

**2. Launcher tiles for terminal apps.**
The Terminal app/daemon grow a `command` option (`POST /term {cols, rows,
command}`), so the launcher can offer "pi" and "Herdr" tiles that open a
terminal window already running the tool. Herdr is extra-valuable here:
agents keep running when the window closes — fits the Pi-as-appliance story
(`ssh` back in later, nothing died).

**3. ghostd as the model gateway (the real integration).**
Ghost's router config (`ai.toml`, ADR 0002) already names every provider the
user has. ghostd exposes a localhost **OpenAI-compatible proxy**
(`http://127.0.0.1:7700/v1/...`, token-gated like everything else) that
forwards through the router — so pi, Herdr-managed agents, or anything else
pointed at it inherits:

- the user's keys without scattering them across tool configs (enter once in
  Settings → Ghost, every tool works),
- routing (local/LAN/cloud) and offline mode,
- the provenance log — *all* AI traffic on the machine is auditable in one
  place, which is the GhOSt promise.

Tools that want a specific provider can still talk to it directly; the
gateway is a default, not a cage.

## Why not deeper?

pi's RPC/SDK modes would allow embedding it as Ghost's coding backend, and
that stays on the table for Phase 7+ — but Ghost's identity is the *OS*
assistant (tools = system API), while pi's is the *repo* assistant. Keeping
them separate-but-sharing-providers is honest; gluing them together before
Ghost v1 exists is speculation.
