---
name: clean-downloads
description: Inspect ~/Downloads and report what's there — flagging stale, large, and duplicate-looking files — then optionally tidy it. Use when the user asks to clean, review, audit, or free up space in their Downloads folder, or wonders what's piling up in Downloads.
---

# Review and clean the Downloads folder

When the user asks to clean up, review, or audit `~/Downloads`:

1. Call `list_files` on `~/Downloads`. If it's empty, say so and stop — nothing
   to do.
2. Read-only analysis first. From the listing, group what you see and report:
   - **Installers / archives**: dmg, pkg, exe, zip, tar, gz, 7z, rar, iso —
     these are usually safe to remove once used.
   - **Likely stale**: anything whose name suggests a finished one-off
     (invoices, tickets, "final", screenshots saved here by accident).
   - **Large items**: call out the biggest files by their reported size so the
     user knows what's eating space.
   - **Possible duplicates**: names like `report.pdf` and `report (1).pdf`, or
     `file.zip` alongside an extracted `file/` folder.
3. Present this as a short written summary BEFORE touching anything. Let the
   user decide what to act on.
4. If the user wants to tidy rather than delete, propose `move_file` actions
   into a `~/Downloads/Archive` folder (create it first with `make_dir`).
5. If the user explicitly wants files gone, propose `trash_file` for each — never
   delete silently, and never trash without the user asking. Trash, never a
   permanent delete (there is no permanent delete anyway).
6. Skip folders unless the user names one specifically.

Each move or trash is its own confirmation card — propose the actions, don't ask
permission in prose. Finish with a one-line summary of what was flagged and what
you proposed.
