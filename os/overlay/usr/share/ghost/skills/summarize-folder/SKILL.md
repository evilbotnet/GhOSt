---
name: summarize-folder
description: Read the text-based files in a folder and produce a concise written summary of what the folder contains, optionally saving it as a notes/index file. Use when the user asks what's in a folder, to summarize a folder's contents, to recap notes/documents in a directory, or to make an index of a project folder.
---

# Summarize the contents of a folder

When the user asks what's in a folder or to summarize a folder's documents:

1. Call `list_files` on the target folder. If the user didn't name one, ask
   which — accept shorthand like "documents" → `~/Documents`. If it's empty,
   say so and stop.
2. Identify the readable text files: txt, md, csv, json, yaml, toml, and source
   code. Skip binaries (images, archives, pdf, docx, zip, etc.) — you can only
   read text. Note any skipped files briefly so the user knows they exist.
3. For each readable file (cap at the ~15 most relevant if there are many — pick
   by name and recency), call `read_file` and extract the gist: what it is, its
   key points or purpose. Don't quote walls of text.
4. Write a concise summary: one short paragraph or a bulleted line per file,
   plus a one-sentence overview of what the folder as a whole seems to be for.
5. Offer to save the summary. If the user agrees (or asked for an index up
   front), propose a `write_file` to `<folder>/SUMMARY.md`. If that file already
   exists, use `SUMMARY-<today's date>.md` instead so nothing is overwritten.
6. Never modify the source files — this skill only reads them.

The save step is its own confirmation card — propose it, don't ask in prose.
Finish with a one-line summary: how many files you read and where (if anywhere)
you saved the result.
