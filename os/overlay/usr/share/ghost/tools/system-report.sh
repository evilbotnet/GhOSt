#!/bin/sh
# Example read-only Ghost tool. No args.
echo "== uptime / load =="; uptime
echo; echo "== memory =="; free -h 2>/dev/null || vm_stat 2>/dev/null | head -6
echo; echo "== disk (root) =="; df -h / 2>/dev/null
