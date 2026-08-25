---
status: complete
quick_id: 260825-untrack-binaries
date: 2026-08-25
---

# Summary — 260825-untrack-binaries

**Concern:** CONCERNS.md #17 Build binaries committed to git.

**Change:** Added `/server` and `/migrate` to `.gitignore` and untracked the committed binaries
with `git rm --cached` (files remain on disk; CI rebuilds them). This stops bloating history/clones
with stale, secret-leaking artifacts and removes the risk of committing built binaries.

**Verification:** binaries no longer tracked; `.gitignore` updated.
