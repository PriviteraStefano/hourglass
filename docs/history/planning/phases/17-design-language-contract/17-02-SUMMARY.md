---
phase: 17-design-language-contract
plan: 02
subsystem: ui
tags: [design-language, docs, agents-gate, index]

requires:
  - phase: 17-design-language-contract/01
    provides: docs/design/LANGUAGE.md (the contract this map references)
provides:
  - docs/design/INDEX.md — design-documentation map
  - AGENTS.md design gate — enforces INDEX.md-first before web/src UI changes
affects: [18-chrome, 19-workflows, 20-composition, all future web/src UI work]

actuals:
  tokens: 400
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns: [AGENTS.md design gate enforcing docs/design/INDEX.md before UI changes]
key-files:
  created: [docs/design/INDEX.md]
  modified: [AGENTS.md]
key-decisions:
  - "INDEX.md references the live LANGUAGE.md and reserves CHROME/workflows/COMPOSITION for Phases 18/19/20 (D-17-14/15)"
  - "AGENTS.md gate is the single design sentence inserted right after the title; no other design text added (D-17-17)"
requirements-completed: [DL-01]

coverage:
  - id: D1
    description: "docs/design/INDEX.md authored as the design-documentation map listing LANGUAGE.md (link) plus three reserved paths (CHROME Phase 18, workflows Phase 19, COMPOSITION Phase 20)"
    requirement: DL-01
    verification:
      - kind: other
        ref: "test -f docs/design/INDEX.md && grep -q 'design documentation map' docs/design/INDEX.md => true; 4 path entries present"
        status: pass
    human_judgment: true
    rationale: "Documentation map is advisory; human confirmation of reserved-path phrasing and scope is appropriate before Phases 18/19/20 build on it."
  - id: D2
    description: "AGENTS.md design gate inserted as line 2 immediately after the title, exactly once, with no other design text added"
    requirement: DL-01
    verification:
      - kind: other
        ref: "sed -n '1,3p' AGENTS.md | grep -q \"open \`docs/design/INDEX.md\` first\" => true; gate count == 1"
        status: pass
    human_judgment: true
    rationale: "Single-sentence gate is a pinned copy (D-17-17); human confirmation that no extra design text leaked into AGENTS.md is appropriate."
---

# Phase 17 Plan 02: Design-documentation map and AGENTS gate Summary

**`docs/design/INDEX.md` created and the single design gate inserted into `AGENTS.md`, closing the DL-01 discoverability/enforcement loop**

## Performance

- **Duration:** ~6 min
- **Started:** 2026-08-28T11:06:00Z
- **Completed:** 2026-08-28T11:14:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Authored `docs/design/INDEX.md` as a simple path map: `LANGUAGE.md` (markdown link) plus reserved `docs/design/CHROME.md` (Phase 18), `docs/design/workflows/` (Phase 19), `docs/design/COMPOSITION.md` (Phase 20), each with a one-line purpose.
- Inserted the exact D-17-17 design gate sentence into `AGENTS.md` as line 2 (immediately after the title), with the existing `## OpenWiki` content unchanged and no other design text added.

## Task Commits

Each task was committed atomically:

1. **Task 1: Author docs/design/INDEX.md path map** - `INDEX commit` (docs)
2. **Task 2: Insert the AGENTS.md design gate after the title** - `33f63dc` (docs)

**Plan metadata:** `12746e7` (docs: complete plan)

## Files Created/Modified
- `docs/design/INDEX.md` - the design-documentation map (new file)
- `AGENTS.md` - one design gate sentence inserted after the title

## Decisions Made
- None beyond the locked decisions D-17-13..D-17-17; followed plan as specified.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- DL-01 loop closed: contract (LANGUAGE.md) is discoverable via INDEX.md and enforced via the AGENTS.md gate.
- Ready for Phase 18 (chrome), Phase 19 (workflows), Phase 20 (composition) to build on the contract.
- No blockers.

---
*Phase: 17-design-language-contract*
*Completed: 2026-08-28*
