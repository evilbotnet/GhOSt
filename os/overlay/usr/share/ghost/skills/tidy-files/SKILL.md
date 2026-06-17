---
name: tidy-files
description: Organize a messy folder by sorting files into subfolders by type (Images, Documents, Archives, Code, Other). Use when the user asks to tidy, organize, sort, or clean up a folder.
---

# Tidy a folder by file type

When the user asks to organize, tidy, sort, or clean up a folder:

1. Call `list_files` on the target folder. If the user didn't name one, ask
   which — but if they said "downloads", "desktop", etc., use `~/Downloads`,
   `~/Desktop`, and so on.
2. Group the **files** (never folders) by extension into these buckets:
   - **Images**: jpg, jpeg, png, gif, webp, heic, svg
   - **Documents**: pdf, doc, docx, txt, md, odt, rtf, csv, xlsx, pptx
   - **Archives**: zip, tar, gz, tgz, 7z, rar
   - **Code**: go, js, ts, py, sh, rs, c, h, json, yaml, toml
   - **Other**: anything else
3. For each non-empty bucket, `make_dir` a subfolder of that name inside the
   target folder. If it already exists the call may error — that's fine,
   carry on.
4. `move_file` each file into its bucket subfolder.
5. Never move a file that is already inside one of these bucket folders, and
   never move a folder.
6. Finish with a one-line summary: how many files landed in each bucket.

Each move is its own confirmation card for the user — that is expected and
good. Propose the moves; don't ask permission in prose.
