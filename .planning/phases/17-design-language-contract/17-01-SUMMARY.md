---
phase: 17-design-language-contract
plan: 01
subsystem: ui
tags: [design-language, docs, tokens, shadcn]

requires:
  - phase: 15-ux-foundation-design-tokens-shared-components
    provides: historical token dump and frozen-component intent (inputs, not authority)
provides:
  - docs/design/LANGUAGE.md — source-of-truth design-language contract (type, color, density, motion, status)
affects: [18-chrome, 19-workflows, 20-composition, job-cluster-implementation]

actuals:
  tokens: 1800
  tasks: 2
  commits: 1

tech-stack:
  added: []
  patterns: [design-language contract under docs/design/, repo-root CSS pointers without value duplication]
key-files:
  created: [docs/design/LANGUAGE.md]
  modified: []
key-decisions:
  - "LANGUAGE.md wins on vocabulary/meaning; CSS wins on values; later docs may only add surface/layout/copy/composition (D-17-09)"
  - "Nine role Gaps recorded (ui, density, duration, easing, neutral, info, success, warning, danger) without inventing tokens (D-17-08)"
requirements-completed: [DL-01]

coverage:
  - id: D1
    description: "docs/design/LANGUAGE.md authored as the DL-01 design-language contract following D-17-20 section order, citing web/src/index.css and web/components.json by repo-root path only"
    requirement: DL-01
    verification:
      - kind: other
        ref: "grep -c 'oklch(' docs/design/LANGUAGE.md => 0; grep -c 'no .* token in index.css.' => 9; grep -q 'LANGUAGE.md wins' => true"
        status: pass
    human_judgment: true
    rationale: "Design-language contract is advisory product language; human review of vocabulary scope and authority framing is appropriate before downstream chrome/workflow/composition docs build on it."
---

# Phase 17 Plan 01: Design-language contract Summary

**`docs/design/LANGUAGE.md` authored as the source-of-truth design-language contract covering type, color, density, motion, and status vocabulary (DL-01)**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-08-28T10:49:00Z
- **Completed:** 2026-08-28T11:05:00Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- Authored `docs/design/LANGUAGE.md` with the exact D-17-20 section order (Purpose → Changelog → Foundations → Overlay → Light/dark → Do/don't → Pointers → Not-in-this-file → 15-UI-SPEC note → Gaps).
- Documented five vocabularies (Type, Color, Density, Motion, Status) as English roles + MUST/SHOULD rules + repo-root CSS pointers; no oklch/hex/px value tables.
- Recorded nine role Gaps without inventing tokens; cited `web/src/index.css` and `web/components.json` by path only.
- Listed Phase 15 frozen components as inputs with live gaps (StatusBadge present, PageHeader/FilterBar/DataTable/EmptyState/ConfirmDialog absent); did not restore or rewrite them.

## Task Commits

Each task was committed atomically:

1. **Task 1: Author docs/design/LANGUAGE.md full contract** - `87121d9` (docs)
2. **Task 2: Conformance audit of LANGUAGE.md against locked decisions** - verified in place (no further file change required beyond the inline authority-phrase fix folded into the Task 1 commit)

**Plan metadata:** `12746e7` (docs: complete plan)

## Files Created/Modified
- `docs/design/LANGUAGE.md` - the design-language contract (new file)

## Decisions Made
- None beyond the locked decisions D-17-01..D-17-48; followed plan as specified.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `docs/design/LANGUAGE.md` is ready for Plan 17-02 (INDEX.md + AGENTS.md gate) and for Phase 18 chrome work to cite.
- No blockers.

---
*Phase: 17-design-language-contract*
*Completed: 2026-08-28*
