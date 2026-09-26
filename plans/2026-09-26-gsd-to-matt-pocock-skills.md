# GSD → Matt Pocock skills migration

**Date:** 2026-09-26
**Status:** complete (see Outcome)
**Scope:** retire GSD everywhere (repo + user machine), install and configure the
[Matt Pocock skill set](https://github.com/mattpocock/skills) as the delivery workflow for Hourglass.

## Decisions (locked 2026-09-26, with the repo owner)

| Question | Answer |
|---|---|
| GSD removal scope | Global + repo. No GSD artifacts survive. |
| `.planning/` fate | Archived to `docs/history/planning/` (frozen record), live pointers rewritten. |
| Issue tracker | GitHub Issues on `PriviteraStefano/hourglass` via the `gh` CLI. |
| Skill install | Stable set (25 skills), project-committed: canonical `.agents/skills/` + `.claude/skills/` symlinks + `skills-lock.json`. Stale global copies refreshed. |
| ADR home (agent call) | Existing convention kept: `hourglass-vault/decisions/`. `CONTEXT.md` at repo root. |

## Ownership map (what replaces what)

| GSD artifact / agent | Replacement |
|---|---|
| `.planning/ROADMAP.md` (phase list) | GitHub milestone `v0.2.1` + issues (`/to-tickets`, `/wayfinder`) |
| `.planning/REQUIREMENTS.md` | Spec issues on the tracker (`/to-spec`) |
| `.planning/phases/NN-*-CONTEXT.md`, `-DISCUSSION-LOG.md` | `/grill-with-docs` → root `CONTEXT.md` + ADRs |
| `.planning/phases/NN-RESEARCH.md`, `.planning/research/*` | `/research` (cited Markdown in-repo) |
| `.planning/codebase/*` | `docs/codebase/*` — still agent-facing, no longer GSD-owned |
| `NN-*-PLAN.md` | Tickets with blocking edges (`/to-tickets`) |
| gsd-executor / gsd-verifier / gsd-code-reviewer | `/implement` → `/tdd` → `/code-review` |
| `.planning/STATE.md` (session continuity) | `/handoff` docs + tracker state |
| `.planning/WINDOWS.md` (broken-windows ledger) | Tracker issues with labels |
| gsd-sketch / `NN-UI-SPEC.md` | `/prototype` |
| `gsd-graphify-update.sh` hook | AGENTS.md manual rule + `.opencode/plugins/graphify.js` (repo-scoped hook optional, see Follow-ups) |

## GSD inventory being removed (evidence)

**Global**
- npm global packages `@opengsd/gsd-pi@1.11.0`, `@opengsd/gsd-browser@0.2.2`; bins `gsd`, `gsd-cli`, `gsd-pi`, `gsd-browser` in `~/.local/bin`.
- `~/.claude/gsd-core/` (331 files), `~/.claude/get-shit-done/`, `~/.claude/gsd-migration-journal/`, `gsd-install-state.json`, `gsd-file-manifest.json` (450 entries), `.gsd-profile`.
- `~/.claude/agents/gsd-*.md` (33), `~/.claude/skills/gsd-*/` (67), `~/.claude/hooks/gsd-*` (17) + `hooks/managed-hooks-registry.cjs` + `hooks/lib/gsd-graphify-rebuild.sh`.
- `~/.claude/settings.json`: 7 hook events carrying `gsd-*` commands + GSD `statusLine`. **Orca hook groups are preserved.**

**Repo**
- `.planning/` (215 tracked files) → archived.
- `.gitignore` auto-generated "GSD baseline" block (`.gsd`, `.gsd-worktrees/`, `.gsd-backups/`, `.gsd-id`, `.bg-shell/`).
- `.claude/settings.local.json` MCP servers `gsd-workflow`, `gsd-browser`.
- Branches `gsd/phase-17-*`, `gsd/phase-18-*`, `milestone/M001` + locked worktree `.gsd-worktrees/M001` (main checkout parked on `gsd/phase-18-chrome-contract`).
- Pointers to `.planning/` in `docs/design/*` (8 files), `hourglass-vault/decisions/ADR-BE-002`, `ADR-BE-009`.

## Sequence

**S1 — Plan committed** (`plans/2026-09-26-gsd-to-matt-pocock-skills.md`).

**S2 — Install the skill set (project scope)**

```bash
npx -y skills@latest add mattpocock/skills -y \
  -a claude-code -a universal \
  -s ask-matt -s code-review -s codebase-design -s diagnosing-bugs -s domain-modeling \
  -s grill-with-docs -s implement -s improve-codebase-architecture -s prototype -s research \
  -s resolving-merge-conflicts -s setup-matt-pocock-skills -s tdd -s to-spec -s to-tickets \
  -s triage -s wayfinder -s wizard \
  -s grill-me -s grilling -s handoff -s teach -s to-questionnaire -s wait-what -s writing-for-agents
```

Canonical `.agents/skills/<name>/`, symlinks `.claude/skills/<name>`, root `skills-lock.json` (v3, merged with the existing wshobson/anthropics entries).
Verify: `npx skills ls`; symlink targets resolve; skill discoverable.

**S3 — Configure the repo for the skill set** (outputs of `/setup-matt-pocock-skills`, written directly)
- `docs/agents/issue-tracker.md` — GitHub mode, `gh` CLI, PRs-as-request-surface off.
- `docs/agents/domain.md` — single-context; root `CONTEXT.md`; ADRs in `hourglass-vault/decisions/`.
- `docs/agents/triage-labels.md` — default five canonical labels.
- `AGENTS.md` — add `## Agent skills` block (issue tracker / triage labels / domain docs).
- `CONTEXT.md` — seeded from `.planning/PROJECT.md` (what/cvalue/constraints/key decisions) + design-language pointers.
- Create the five triage labels on GitHub.

**S4 — Relocate load-bearing GSD docs**
- `git mv .planning/codebase docs/codebase`.
- Update operative pointers: `AGENTS.md` (CONCERNS #11/#12/#15), `hourglass-vault/decisions/ADR-BE-002`, `ADR-BE-009`, cross-refs inside the moved files.

**S5 — Archive the planning tree**
- `git mv .planning docs/history/planning`; add `docs/history/README.md` + `docs/history/planning/README.md` (frozen, internal paths predate the move).
- Rewrite live-doc pointers in `docs/design/LANGUAGE.md`, `CHROME.md`, `workflows/*.md` (8 files) to `docs/history/planning/...`.
- Leave dated snapshots untouched (`hourglass-vault/research/2026-07-28 … audit`, archived phase files).

**S6 — Migrate live work state to the tracker**
- Milestone `v0.2.1` (+ labels `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`, `deferred`, `tech-debt`).
- Issues: Phase 20 (COMP-01 + SKETCH-01); JOB-01 job-cluster implementation (blocked by Phase 20); four deferred v0.1/v0.2 items (12-UAT gap, 12-VERIFICATION gap, `260801-got-investigate-sidebar-collapsed-mode-hover`, `260801-o06-postgresql-pool`).
- Triage stale issue #49 (Phase 8 hardening, complete since 2026-07-31) → comment + close.

**S7 — Remove GSD (repo)**
- `.gitignore`: drop the GSD-specific entries; keep the generic OS/editor junk lines.
- `.claude/settings.local.json`: drop `gsd-workflow`/`gsd-browser`.
- `git worktree remove --force .gsd-worktrees/M001` (unlock first), delete branches `milestone/M001`, `gsd/phase-17-*`, `gsd/phase-18-*`; move the main checkout back to `main`.

**S8 — Remove GSD (global)**
- Backup `~/.claude/settings.json`.
- `npm uninstall -g @opengsd/gsd-pi @opengsd/gsd-browser`; remove leftover bins.
- Delete `gsd-core/`, `get-shit-done/`, `gsd-migration-journal/`, `gsd-install-state.json`, `gsd-file-manifest.json`, `.gsd-profile`, `agents/gsd-*`, `skills/gsd-*`, `hooks/gsd-*`, `hooks/managed-hooks-registry.cjs`, `hooks/lib/gsd-graphify-rebuild.sh`.
- Strip GSD hook entries + GSD `statusLine` from `~/.claude/settings.json`, preserving every non-GSD (Orca) group.

**S9 — Refresh the global skill set**
- `npx skills update -g` for the surviving Matt Pocock skills; remove superseded `write-a-prd`, `prd-to-plan`, `prd-to-issues`, `request-refactor-plan`; add the stable set globally so non-repo sessions get current skills.

**S10 — Verification sweep**
- `grep -rn "gsd\|GSD"` over the repo → matches only inside `docs/history/planning/**` and git history.
- `grep -rn "\.planning/"` over live docs → zero.
- `npx skills ls` shows the 25 project skills; one skill invocation smoke-tested.
- `gh issue list --milestone v0.2.1`; `gh label list`.
- `which gsd` empty; `python3 -c` manifest check: none of the 450 manifest paths exist.
- `~/.claude/settings.json` valid JSON, Orca hook groups intact (count per event unchanged minus GSD entries).
- Repo checks that touch docs: `scripts/docs-check.sh`, `scripts/validate-mermaid.sh`, `scripts/verify-readme.mjs`, `scripts/verify-wiki.mjs` (as applicable to moved content).

**S11 — Commit** in logical commits (skills+setup, docs relocation/archive, GSD removal); report the branch state so the owner can merge to `main`.

## Rollback

- Global: `~/.claude/settings.json` backup; npm reinstall via `npx @opengsd/gsd-pi`; `gsd-core` restorable from the npm package.
- Repo: single `git revert` per commit; `.planning/` structure preserved verbatim at `docs/history/planning/`.
- Tracker: issues/labels created are additive and removable.

## Out of scope (explicitly untouched)

- `plans/` (architecture docs, e.g. `hexagonal-migration.md`), `openwiki/`, `wiki/`, `docs/superpowers/specs/`, `.opencode/plans/` (pre-GSD-era planning homes; separate cleanup if wanted).
- `hourglass-vault/` structure and its ADR/vision content (only two path citations updated).
- Non-GSD skills (anthropics, wshobson, vercel-labs, obra/superpowers, orca-cli, orchestration, obsidian-vault).

## Follow-ups (not part of this cutover)

1. ~~Repo-scoped graphify hook~~ — dropped: graphify already owns repo git hooks (`post-commit`, `post-checkout`; "Installed by: graphify hook install"), so the graph rebuilds after commits without GSD.
2. First run of the new flow for Phase 20 (issue #50): `/grill-with-docs` → `/to-spec` → `/to-tickets` → `/implement`; `/wayfinder` for the job-cluster chunk (#51).
3. Consolidate the remaining doc homes (`docs/superpowers/specs/`, `.opencode/plans/`) if desired.

## Outcome (2026-09-26) — complete

Executed in the Orca `phase-19` worktree on branch `PriviteraStefano/phase-19` (the branch tip equalled `main` when work started). Commits: plan → skills + repo config → archive/relocation → repo GSD removal.

### Evidence

| Check | Result |
|---|---|
| GSD files in `~/.claude` | manifest 450/450 present before, 0 after |
| GSD files in `~/.codex` | manifest 531/531 before, 0 after |
| GSD files in `~/.config/opencode` | manifest 619/619 before, 0 after |
| `~/.claude/settings.json` | 15 `gsd-*` hook commands + GSD statusLine removed; 12 Orca hook groups preserved; JSON valid |
| npm globals | `@opengsd/gsd-pi`, `@opengsd/gsd-browser` uninstalled; `gsd*` bins gone |
| repo sweep | no live `gsd` / `.planning` references outside `docs/history/planning/` (frozen) and this plan |
| project skills | 42 in `.agents/skills` (25 new), 42 resolving `.claude/skills` symlinks, `skills-lock.json` merged |
| global skills | 32 in `~/.agents/skills`, 30 in `~/.claude/skills`, no dangling symlinks, superseded names gone |
| tracker | milestone `v0.2.1` + issues #50–#55; #51 blocked-by #50 (native dependency); stale #49 closed |
| docs checks | `docs-check.sh`, `validate-mermaid.sh`, `verify-readme.mjs`, `verify-wiki.mjs` all pass |
| frontend | `bun run build` green (only after the generated route tree exists — see issue #54) |

### Discovered during execution (beyond the plan)

- GSD also lived in `~/.codex` (33 `[agents.gsd-*]` sections in `config.toml`, `hooks.json` + hooks, 67 skills, 33 agent files) and in `~/.config/opencode` (619 files: `gsd-core`, 71 commands, hooks, scripts, plugin, plus a `gsd` MCP server and permission block in `opencode.jsonc`), mirrored into Orca's `codex-runtime-home`. All removed; Orca's own hooks and runtime config preserved.
- The `graphify` OpenCode skill had been displaced into GSD's `gsd-user-files-backup`; restored to `~/.config/opencode/skills/graphify`.
- `260801-o06`'s "migration chain not re-runnable" debt is fixed by the 2026-08-25 `schema_migrations` ledger → not re-filed. The surviving defects became issues #54 (fresh-clone build) and #55 (`-all` flag).
- Backups kept: `~/.claude/settings.json.pre-gsd-removal-20260926`, `~/.codex/config.toml.pre-gsd-removal-20260926`, `~/.config/opencode/opencode.jsonc.pre-gsd-removal-20260926`, plus the Orca runtime-home `config.toml.pre-gsd-removal-20260926`.

### Not done here

- The commits sit on `PriviteraStefano/phase-19` in this Orca worktree; `origin/main` does not have them yet (merge/push is the owner's call).
- First run of the new flow for Phase 20 — follow-up 2.
