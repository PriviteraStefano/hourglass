---
phase: 18-chrome-contract
plan: 01
subsystem: ui
tags: [docs, chrome, design-contract, navigation, role-visibility]

# Dependency graph
requires:
  - phase: 17-design-language-contract
    provides: docs/design/LANGUAGE.md (vocabulary authority that CHROME.md must not override)
provides:
  - docs/design/CHROME.md — source-of-truth chrome contract (frame, navigation, role-scoped chrome, page anatomy)
affects: [phase 19 role contracts, phase 20 composition map, post-phase-20 job-cluster implementation]

# Actuals (#2632) — chars/4 over the realized diff.
actuals:
  tokens: 2119
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns: [chrome-contract doc pattern mirroring LANGUAGE.md (dual-audience, RFC 2119, changelog, authority stack, Not-in-this-file fence, Don't-only do/don't, Pointers, input note, Gaps)]

key-files:
  created: [docs/design/CHROME.md]
  modified: []

key-decisions:
  - "D-17-09: CHROME.md adds surface/layout/copy/composition only; LANGUAGE.md wins on type/color/density/motion/status vocabulary"
  - "D-18-01: ADR-P-011 D-1 is revised (not adopted); D-2/D-3/D-5/D-6 kept, Admin group dropped"
  - "D-18-02: four lifecycle nav groups Record/Organize/Approve/Report seed the future ABAC resource taxonomy"
  - "D-18-03: Today is the landing route /, outside the four groups"
  - "D-18-04: enumeration depth = groups + principle + clear-end features; middle Organize mapping deferred to Phase 20"
  - "D-18-05/08/09: single predicate-driven role-visibility mechanism (nav + page-action); customer deferred; per-role matrix carried as input"
  - "D-18-06/07: frame = Sidebar + per-page PageHeader (two-row) + Body; no global identity bar"
  - "D-18-10/11: page anatomy shell rules only; frozen components are absent inputs, never claimed to exist"
  - "D-17-21: changelog first entry exact; D-17-22: do/don't Don't-only; D-17-04: no oklch/hex token tables"
  - "SC2: Admin/Settings chrome excluded; SC4: docs-only, no UI/sketch/route work"

patterns-established:
  - "Chrome source-of-truth contract: defines shell, nav, role-scoped chrome, page anatomy; fences out vocabulary, workflow, composition, Admin/Settings"
  - "Frozen-component input table: present/absent status, no existence claim"

requirements-completed: [CHR-01]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "docs/design/CHROME.md authored as the chrome contract — frame, navigation (four lifecycle groups + Today landing), role-scoped chrome (single predicate mechanism), page anatomy (shell rules), with authority stack, Not-in-this-file fence, Don't-only do/don't, frozen-component input table, and four Gaps"
    requirement: CHR-01
    verification:
      - kind: other
        ref: "test -f docs/design/CHROME.md && grep -q '2026-08-29 · Phase 18 · Added chrome contract' && grep -q 'LANGUAGE.md wins on type, color, density, motion, and status vocabulary' && echo OK"
        status: pass
      - kind: other
        ref: "grep -c 'oklch(' docs/design/CHROME.md | grep -q '^0$' && grep -q 'Hiding a group here must never be mistaken for authorization' && echo OK"
        status: pass
    human_judgment: false

# Metrics
duration: 12min
completed: 2026-08-29
status: complete
---

# Phase 18 Plan 01: Chrome Contract Summary

**Authored `docs/design/CHROME.md` as the source-of-truth app-shell contract — frame, four-lifecycle navigation, predicate-driven role-scoped chrome, and page anatomy — revising (not adopting) ADR-P-011 and deferring Admin/Settings, the Organize mapping, the page-action predicate, the per-role matrix, and the customer surface.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-08-29T00:00:00Z
- **Completed:** 2026-08-29T00:12:00Z
- **Tasks:** 2
- **Files modified:** 1 (created)

## Accomplishments
- Authored `docs/design/CHROME.md` in the exact D-17-20 section order (Purpose → Changelog → Frame → Navigation → Role-scoped chrome → Page anatomy → Do/don't → Pointers → Not in this file → 15-UI-SPEC note → Gaps).
- Encoded every locked decision: D-17-09 (LANGUAGE.md wins), D-18-01 (revise ADR-P-011), D-18-02 (four lifecycle groups seed ABAC), D-18-03 (Today at `/`), D-18-04 (enumeration depth), D-18-05/08/09 (role-scoped chrome single predicate, matrix deferred, customer deferred), D-18-06/07 (frame + two-row PageHeader target spec), D-18-10/11 (page anatomy shell rules; frozen components absent inputs), D-17-21 (changelog), D-17-22 (Don't-only), D-17-04 (no value tables), SC2 (Admin/Settings excluded), SC4 (docs-only).
- Carried the verbatim `role-visibility.ts` authority warning and the frozen-component input table (StatusBadge Present; PageHeader/FilterBar/DataTable/EmptyState/ConfirmDialog Absent).

## Task Commits

Each task was committed atomically:

1. **Task 1: Author docs/design/CHROME.md full contract** - `4d37152` (docs)
2. **Task 2: Conformance audit of CHROME.md** - folded into `4d37152` (no deviation found; verified in place)

**Plan metadata:** `15a3fb3` (docs(18): create phase plan)

## Files Created/Modified
- `docs/design/CHROME.md` - the chrome contract (new file, 98 lines)

## Decisions Made
- Followed the plan exactly; no planner discretion deviated. ADR-P-011 is stated as revised input (D-1 replaced, D-2/D-3/D-5/D-6 retained, Admin dropped), not adopted.
- Frozen components are referenced only as absent inputs / intended implementers; no existence claim appears anywhere in the file (verified by grep).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- CHROME.md is the shell source of truth for Phase 19 (role/workflow contracts), Phase 20 (composition map), and post-Phase-20 job-cluster implementation.
- Deferred items recorded as Gaps: Organize middle mapping (Phase 20), page-action predicate (later job cluster, GAP A), per-role visibility matrix (Phase 19/20), customer surface (CUST-01).
- `docs/design/INDEX.md` was intentionally NOT modified (reserved CHROME.md line already correct; delegated to plan 18-02).

---
*Phase: 18-chrome-contract*
*Completed: 2026-08-29*
