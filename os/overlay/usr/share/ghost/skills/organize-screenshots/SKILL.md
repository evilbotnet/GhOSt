---
name: organize-screenshots
description: Sort screenshots in ~/Pictures into per-month subfolders so they stop cluttering the top level. Use when the user asks to organize, sort, file, or tidy their screenshots, or mentions a pile of screen-*.png files building up in Pictures.
---

# Sort screenshots into month folders

GhOSt's screenshot tool saves files to `~/Pictures` named like
`screen-2026-06-16-142530.png` (the date is embedded). When the user asks to
organize their screenshots:

1. Call `list_files` on `~/Pictures` (or the folder the user names).
2. Select the screenshot files: names starting with `screen-` and ending in
   `.png`, plus any other obvious screenshots the user points at. Ignore other
   images and all folders. If none are found, say so and stop.
3. For each screenshot, derive its month from the `YYYY-MM` in the filename and
   target a subfolder named that way, e.g. `~/Pictures/2026-06`. If a file's
   name has no parseable date, group it under `~/Pictures/Unsorted` instead.
4. For each distinct target month that has at least one file, `make_dir` that
   subfolder. If it already exists the call may error — that's fine, carry on.
5. `move_file` each screenshot into its month folder. Skip any file already
   sitting inside a `YYYY-MM` (or `Unsorted`) folder — those are done.
6. If a name collision would occur in the target, append `-2`, `-3`, etc. to the
   moved file's name so nothing is overwritten.

Each move is its own confirmation card — propose the moves, don't ask in prose.
Finish with a one-line summary: how many screenshots moved into how many month
folders.
