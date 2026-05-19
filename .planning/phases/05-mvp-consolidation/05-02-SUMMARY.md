---
phase: 05-mvp-consolidation
plan: 02
subsystem: database
tags: surrealdb, seed, demo, surql, time-entries, expenses
requires:
  - phase: 05-mvp-consolidation
    plan: 01
    provides: Foundation seed data (org, users, units, projects, subprojects, WGs, WG members)
provides:
  - "12 sample time entries in 'submitted' status across 3 employees and 6 projects"
  - "6 sample expenses with mileage, meal, and other categories"
  - "Complete MVP demo seed dataset ready for schema load"
affects:
  - "05-03 (manual verification of seed data in UI)"
tech-stack:
  added: []
  patterns:
    - "Time entries with all 6 mandatory record links (org_id, user_id, project_id, subproject_id, wg_id, unit_id)"
    - "Expense entries with valid category values and optional project_id"
    - "Past-week date spread for realistic demo data"
key-files:
  created: []
  modified:
    - schema/003_seed_demo.surql (appended 318 lines — 12 time entries + 6 expenses)
key-decisions:
  - "Used actual WG membership boundaries to assign time entries (Emma → Platform Eng + Cloud Migration WGs, James → Data Analytics + Finance Tools WGs, Lisa → DevOps + HR System WGs)"
  - "All entries in 'submitted' status to demonstrate the approval workflow"
  - "Past 5 business days (12-18 May 2026) for realistic-looking data"
patterns-established:
  - "String-type record IDs for time entries (te_001–te_012) and expenses (exp_001–exp_006)"
  - "Entry dates use SurQL <datetime>'...' syntax"
  - "Spent hours range 4.0–7.5 for realistic variation"
  - "Descriptions follow pattern: work product/feature + brief detail"
requirements-completed:
  - D-01
  - D-16
  - D-17
duration: 8 min
completed: 2026-05-19
---

# Phase 5: MVP Consolidation — Plan 02 Summary

**Appended 12 sample time entries (4 per employee) and 6 sample expenses (2 per employee) to the demo seed file, completing the full MVP seed dataset with realistic past-week data in 'submitted' status for approval workflow demonstration**

## Performance

- **Duration:** 8 min
- **Started:** 2026-05-19T11:14:00Z
- **Completed:** 2026-05-19T11:22:00Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- Appended 12 time entries (4 per employee: Emma Wilson, James Park, Lisa Torres) with all 6 mandatory record links pointing to existing seed entities
- Every time entry has valid `hours > 0`, non-empty `description`, `<datetime>` entry_date, and `status = "submitted"`
- Spread entries across past 5 business days (Tue 12 May – Mon 18 May 2026) on 6 different projects
- Appended 6 expenses (2 per employee) with valid categories (mileage, meal, other) and realistic EUR amounts
- All employee → project assignments follow actual WG membership boundaries from the seed
- Seed file completed at 1052 lines (exceeds 450 min_lines requirement)

## Task Commits

Each task was committed atomically:

1. **Task 1: Append time entries (3-5 per employee) and expenses (1-2 per employee) to seed file** - `14d0ccf` (feat)

## Files Created/Modified
- `schema/003_seed_demo.surql` (modified, +318 lines, now 1052 total) — Appended 12 time entries and 6 expenses completing the full MVP demo dataset

## Decisions Made
- Used actual WG membership data to assign projects to employees: Emma → Platform Engineering (4 entries) and Cloud Migration (1 entry), James → Data Analytics (3 entries) and Finance Tools (1 entry), Lisa → DevOps Setup (3 entries) and HR System (1 entry)
- All entries set to "submitted" status to demonstrate the approval workflow pipeline
- Entry dates span the past 5 business days with realistic work descriptions
- Expense categories reflect common business expenses: mileage (client visits, conference parking), meal (team lunch, business dinner), other (office supplies)

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered
None — all verifications passed on first attempt.

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- Complete MVP demo seed dataset is ready — all entities from org through expenses are populated
- 1052-line seed file with 73 total CREATE statements is complete and validated
- Ready for **Plan 03**: Manual verification of seed data in SurrealDB and UI

## Self-Check: PASSED

- [x] `schema/003_seed_demo.surql` exists (1052 lines, exceeds 450 min_lines)
- [x] 12 time entries (9-15 range) — verified via grep count
- [x] 6 expenses (3-6 range) — verified via grep count
- [x] All 12 time entries have all 6 required record links — verified via field extraction
- [x] All 6 expenses have valid category values (mileage, meal, other)
- [x] All 12 time entries have hours > 0, non-empty description
- [x] Time entry IDs are unique strings (te_001–te_012)
- [x] Expense IDs are unique strings (exp_001–exp_006)
- [x] All user_id references point to one of the 3 employee UUIDs
- [x] All project_id references point to existing seed project IDs
- [x] Every CREATE statement ends with a semicolon
- [x] Commit 14d0ccf exists

---
*Phase: 05-mvp-consolidation*
*Completed: 2026-05-19*
