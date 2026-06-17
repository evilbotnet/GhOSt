---
name: draft-document
description: Write a well-structured document — letter, README, outline, plan, meeting notes, checklist, README, blog post — from the user's request and save it to ~/Documents. Use when the user asks to draft, write, or create a document, letter, plan, outline, or any piece of structured writing they want saved as a file.
---

# Draft a document and save it

When the user asks you to write or draft a document:

1. Pin down the essentials before writing: the document's purpose, audience, and
   roughly how long. If any of these is genuinely unclear, ask one brief
   question — otherwise make a sensible choice and proceed.
2. Write the full document in Markdown with real structure: a title (`#`),
   logical sections (`##`), and lists or tables where they help. Make it
   complete and ready to use, not a skeleton of placeholders.
3. Choose a filename from the topic in kebab-case, e.g.
   `project-proposal.md` or `cover-letter-acme.md`. Plain prose the user wants as
   `.txt` can use that extension instead.
4. Before saving, check the target: call `list_files` on `~/Documents`. If your
   chosen filename already exists, append today's date (e.g.
   `project-proposal-2026-06-16.md`) so you never overwrite existing work.
5. Propose a `write_file` to `~/Documents/<filename>` with the full content. If
   the user named a different folder, save there instead.
6. If the document is long, still write it in one `write_file` — don't dribble it
   out across multiple appends.

The save is its own confirmation card — propose it, don't ask in prose. After
it's accepted, confirm the file path in one line and offer to revise.
