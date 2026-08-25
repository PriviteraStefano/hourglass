---
quick_id: 260825-untrack-binaries
description: Untrack compiled server/migrate binaries from git (CONCERNS #17)
date: 2026-08-25
status: complete
---

# Quick Task 260825-untrack-binaries

## Plan

Address CONCERNS.md #17 "Build binaries committed to git".

Task: add `/server` and `/migrate` to `.gitignore` and `git rm --cached` the tracked binaries so
they are no longer part of the repository. CI builds them; they must not be committed.

## Verify

- `.gitignore` ignores `/server` and `/migrate`
- `git ls-files | grep -E '^(server|migrate)$'` returns nothing
