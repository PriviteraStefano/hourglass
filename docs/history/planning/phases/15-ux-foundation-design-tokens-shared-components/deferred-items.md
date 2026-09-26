# Deferred / Out-of-Scope Items — Phase 15 Plan 01

Logged during execution (scope boundary: only auto-fix issues directly caused by
the current task's changes; pre-existing issues go here).

## Pre-existing fmt:check drift (oxfmt)

- **Discovered:** 2026-08-17, wave-gate verification (`bun run fmt:check`)
- **Issue:** 31 files fail `oxfmt --check` repo-wide, most unrelated to this plan
  (sidebar.tsx, working-groups dialogs, org-hierarchy pages, contracts list,
  approvals-page, time-entries components, `web/src/index.css`, vault docs).
- **Proof of pre-existing:** `git show 4189c00:web/src/index.css` and the
  sidebar/working-groups files at the base commit also fail `bunx oxfmt --check`.
- **Action:** All 8 plan-owned files were formatted and committed in
  `72f2e85` (`style(15-01): format plan files with oxfmt`). The remaining 31
  files are pre-existing repo-wide drift — NOT fixed here.
- **Recommendation:** A dedicated formatting sweep plan (e.g. `bunx oxfmt` on
  the whole `web/src` tree in one doc/style commit) before Phase 16 starts. The
  phase gate `bun run fmt:check` cannot pass until this is addressed.

## Ambient vault modifications (not plan work)

- `hourglass-vault/02-System-Guide/SG-08-Setup-and-Usage.md` — table
  reformatted by the running Obsidian app during execution; left unstaged.
- `hourglass-vault/.obsidian/workspace.json` — Obsidian session state; left
  unstaged.
- `.planning/STATE.md` — pre-existing dirty state from orchestrator, handled via
  `state.*` queries at close-out.