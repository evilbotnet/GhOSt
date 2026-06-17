---
name: disk-checkup
description: Check how much disk space is free and find what's consuming it in the home folder, then advise on what could be cleared — read-only, the user decides. Use when the user asks why their disk is full, where their space went, to free up space, or for a storage/disk checkup.
---

# Disk space checkup

When the user asks about disk space or what's filling up their drive:

1. Call `system_report` to get current disk usage (and uptime/memory). Lead with
   the headline: how much is used vs. free, as a percentage and in plain terms.
2. Walk the usual heavy folders with `list_files`: `~/Downloads`, `~/Pictures`,
   `~/Desktop`, `~/Documents`. For each, note the largest files and any obvious
   bulk (big archives, installers, videos, many screenshots).
3. Build a short report of likely space hogs, biggest first, with each item's
   folder and reported size so the user can judge.
4. Advise, don't act. Suggest candidates for cleanup in priority order — used
   installers and archives in Downloads, old screenshots, large one-off
   downloads. Make clear these are suggestions.
5. This skill is read-only: do NOT propose `trash_file`, `move_file`, or any
   mutation yourself. If the user then says "yes, clear those", hand off to the
   `clean-downloads` skill (via `load_skill`) or propose the specific
   `trash_file` actions they confirmed.
6. If disk usage is healthy (plenty free), say so plainly rather than inventing
   problems.

Finish with a one-line summary: percent free, and the single biggest thing the
user could clear if they want space back.
