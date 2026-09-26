# Planning archive — GSD era (retired 2026-09-26)

Frozen record of the GSD workflow (`@opengsd/gsd-pi`) that drove this repo from 2026-06 to 2026-09-26. **Read-only history**: do not append, edit, or treat anything here as current state.

## Where the same information lives now

| Here | Now |
|---|---|
| `ROADMAP.md`, `MILESTONES.md` | GitHub milestone + issues on `PriviteraStefano/hourglass` |
| `REQUIREMENTS.md` | spec issues (`/to-spec`) |
| `STATE.md` | `/handoff` documents + tracker state |
| `phases/NN-*-CONTEXT.md`, `-DISCUSSION-LOG.md` | `/grill-with-docs` → `CONTEXT.md` + ADRs |
| `phases/NN-RESEARCH.md`, `research/` | `/research` |
| `phases/NN-*-PLAN.md` | tickets with blocking edges (`/to-tickets`) |
| `codebase/` | moved to `docs/codebase/` |
| `WINDOWS.md` (broken windows) | tracker issues, `deferred` label |
| `sketches/SKETCH-LOOP-CONTRACT.md` | input to Phase 20 (COMP-01 / SKETCH-01); `/prototype` afterwards |
| `quick/` (quick-task records) | git history; no successor artifact |

The live process is the Matt Pocock skill set committed under `.agents/skills/`; its repo configuration is `docs/agents/`.

## Reading notes

- Paths written **inside** these documents refer to the pre-2026-09-26 layout (`.planning/…`, `.gsd-worktrees/…`, `.planning/codebase/…`). They are intentionally not rewritten so the record stays verbatim. Translate: `.planning/X` → `docs/history/planning/X`, `.planning/codebase/X` → `docs/codebase/X`.
- Dated research notes elsewhere in the repo (`hourglass-vault/research/`) keep their original citations for the same reason.
- GSD's own agents, skills, hooks, and CLI have been uninstalled; nothing in this tree can be executed or resumed.
