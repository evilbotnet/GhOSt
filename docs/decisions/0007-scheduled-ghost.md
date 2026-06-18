# ADR 0007 — Scheduled Ghost (proactive runs)

Status: accepted (built) · Builds on [ADR 0002](0002-ghost-ai-assistant.md),
[ADR 0004](0004-ghost-implementation.md)

## The question

Ghost is reactive — it acts when the user types. An AI-native OS should also be
*proactive*: "every morning, tell me what changed in Downloads"; "every 30
minutes, warn me if the disk is over 90%"; "at 6pm, summarise today's edits".
How do scheduled, headless Ghost runs work, and — critically — what may an
unattended run *do*?

## Decision: a cron-lite scheduler firing read-only Ghost runs into notifications

A `Scheduler` in the daemon holds named **schedules** — a prompt plus a cadence
— persisted to `~/.config/ghost/schedules.json`. A one-minute tick fires
anything due; each fire is a fresh, ephemeral Ghost run whose final message is
delivered as a desktop notification (the existing `notify` WS topic) and stored
as the schedule's `lastResult`.

**Cadence** is deliberately minimal — one of:
- `every` — a Go duration (`"30m"`, `"6h"`); sub-minute is rejected so a typo
  can't busy-loop.
- `at` — a daily local time (`"08:00"`).

Cron expressions were considered and rejected for v1: a duration + a daily time
cover the real "appliance" cases, parse trivially, and are explainable in the
UI without a cron cheatsheet. A full cron field is a clean later addition.

## The safety decision: scheduled runs are read-only

No user is at the keyboard, so the confirmation gate ([ADR 0004](0004-ghost-implementation.md))
has no one to ask. Rather than auto-approving mutations (a standing licence for
an LLM to change the system unattended — unacceptable) or silently dropping
them, a headless run **declines every mutating tool** and tells the model so:

> "this is an unattended scheduled run — it is read-only, so this action was
> not performed. Report what you found and what you would do; don't retry."

So proactive Ghost *observes and reports*; if it finds something that needs
action, it says so in the notification and the user opens Ghost to do it with
the gate intact. This is implemented as the `headless` flag threaded through
`runCore` — the same loop, one branch before the gate. Each scheduled run also
gets a fresh session (no shared history with the interactive chat).

## Implementation

- `daemon/internal/ai/scheduler.go` — `Schedule` model, JSON persistence
  (atomic temp+rename), `schedule()` next-run math, the tick loop, and CRUD
  (`List`/`Save`/`Remove`/`RunNow`). On fire it reschedules *before* running so
  a slow run can't double-fire.
- `daemon/internal/ai/ghost.go` — `runCore(id, prompt, headless)` returns the
  final assistant text; `RunScheduled` wraps it for a fresh read-only session.
- HTTP: `GET/POST /ai/schedules`, `DELETE /ai/schedules/{id}`,
  `POST /ai/schedules/{id}/run` (fire now). Managed from the Hub.
- Tests: `scheduler_test.go` (interval + daily + sub-minute-rejection math),
  `clienttools_test.go` shares the harness.

## Scope / future

Cron expressions; per-schedule "autonomy" levels (e.g. a whitelist of tools a
trusted schedule may run unattended); catch-up policy for runs missed while the
machine slept (today: skipped — next tick reschedules forward). The read-only
default stands regardless; any future autonomy is strictly opt-in per schedule.
