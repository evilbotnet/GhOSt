#!/bin/sh
# Example mutating Ghost tool. Arg `text` arrives as $GHOST_ARG_TEXT.
notes="$HOME/Documents/notes.md"
mkdir -p "$(dirname "$notes")"
printf '\n- %s — %s\n' "$(date '+%Y-%m-%d %H:%M')" "$GHOST_ARG_TEXT" >> "$notes"
echo "Noted to $notes"
