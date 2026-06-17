---
name: capture-idea
description: Quickly capture a thought, idea, todo, or reminder into the user's notes file, lightly cleaned up and timestamped. Use when the user says "note this", "jot down", "remember that", "add to my notes", "capture this idea", or rattles off something they want saved for later.
---

# Capture a quick note or idea

Notes live in `~/Documents/notes.md`; the `append_note` tool appends a line to
that file. When the user wants to capture something:

1. Take what the user said and turn it into a clean, self-contained note. Fix
   obvious typos and dictation artifacts, but keep their wording and intent —
   don't editorialize or pad it.
2. If the thought is a single line, keep it as one. If it's several related
   points, structure it as a short bulleted block so it's readable later.
3. Prefix the note with today's date in `YYYY-MM-DD` form so notes stay sortable
   and the user can find when they captured it.
4. Propose an `append_note` with the finished text. One capture = one
   `append_note` call; don't split a single thought across several.
5. If the user is clearly capturing a todo or reminder, prefix the content with
   `TODO:` so it stands out among plain notes.
6. Don't read or rewrite the rest of the notes file — only append. If the user
   asks to review existing notes, `read_file` on `~/Documents/notes.md` instead.

The append is its own confirmation card — propose it, don't ask in prose. After
it's accepted, confirm in one line what was saved.
