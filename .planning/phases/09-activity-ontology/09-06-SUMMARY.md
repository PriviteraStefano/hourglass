---
phase: 09-activity-ontology
plan: 09-06
subsystem: database
tags: [postgres, migration, adr-be-004, activity-ontology, kind-label]

# Dependency graph
requires:
  - phase: 09-activity-ontology
    provides: migration 011 (activities schema, subproject rows kind='task') and the 011 up→down→up cycle test
provides:
  - Forward migration 013 relabeling subproject-derived activities kind='task' → 'phase' (SPEC acceptance #6)
  - Migration cycle test asserting the corrected kind distribution (engagement 6, phase 6, task 0, internal 1)
  - Variadic applyMigrations skip helper ((t, pool, withSeed bool, skip ...string))
affects: [09-activity-ontology verification (truth #10 → green), 09-07 wg test re-seed, Phase 10 WG renovation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Forward label-fix migrations for applied-history corrections (ADR-BE-004 append-only)"
    - "Variadic skip list in migration test apply helper (slices.Contains)"

key-files:
  created:
    - migrations/013_activity_kind_phase_fix.up.sql
    - migrations/013_activity_kind_phase_fix.down.sql
  modified:
    - internal/adapters/secondary/postgres/activity_ontology_migration_test.go
    - internal/adapters/secondary/postgres/staffing_schema_migration_test.go

key-decisions:
  - "013 relabels via predicate kind='task' AND parent_id IS NOT NULL — targets only subproject-derived rows; root rows and other kinds untouched"
  - "013 skipped in the 011-test pre-state apply (UPDATE on activities fails 42P01 before 011 creates it); applied in sorted order (no skip) in the staffing test where it is a no-op"

patterns-established:
  - "Gap closures land as forward migrations numbered max+1, never as edits to applied history (011/012 byte-identical verified)"

requirements-completed: [P-007-D2, P-007-D6]

# Metrics
duration: 3min
completed: 2026-07-31
---

# Phase 09 Plan 06: Gap fix — kind='phase' forward migration (GAP 1) Summary

**Forward migration 013 relabels subproject-derived activities from `kind='task'` to `kind='phase'` per SPEC acceptance #6, and the 011 migration cycle test now asserts the corrected distribution (engagement 6 / phase 6 / task 0 / internal 1), closing VERIFICATION gap 1 without violating the ADR-BE-004 append-only migration rule.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-07-31T18:15:56Z
- **Completed:** 2026-07-31T18:19:28Z
- **Tasks:** 2
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments

- Created `migrations/013_activity_kind_phase_fix.{up,down}.sql` — one-statement forward UPDATE (`kind='task' AND parent_id IS NOT NULL` → `'phase'`) plus its exact reverse; 011/012/000_full_schema remain byte-identical to HEAD (immutability verified via `git diff --stat`).
- Reworked the 011 migration cycle test to interleave 013: up011→up013 (asserts phase 6 / task 0), down013→down011 (asserts the reversal: task 6 / phase 0, proving down013 reverses exactly), up011→up013 (re-asserts phase 6 / task 0). 013 is skipped in pre-state to avoid SQLSTATE 42P01.
- Converted `applyMigrations` to a variadic skip signature `(t, pool, withSeed bool, skip ...string)` using `slices.Contains`, updating both call sites (activity ontology + staffing schema tests).
- VERIFICATION truth #10 (subproject-derived activities carry `kind='phase'`) is now green at the migration-test level; truth #8 (clean up→down→up cycle) stays green with 013 interleaved.

## Task Commits

Each task was committed atomically:

1. **Task 1: Write forward migration 013 (up + down)** - `4ebedcb` (feat)
2. **Task 2: Update migration tests for 013 (applyMigrations skip + cycle assertions)** - `52bd3ed` (test)

**Plan metadata:** pending (docs: complete plan — committed with SUMMARY)

## Files Created/Modified

- `migrations/013_activity_kind_phase_fix.up.sql` - Forward label fix: `UPDATE activities SET kind='phase' WHERE kind='task' AND parent_id IS NOT NULL`; header documents the immutability rationale and expected post-fix distribution
- `migrations/013_activity_kind_phase_fix.down.sql` - Exact reverse (`phase` → `task`, same row set); best-effort lossiness documented per ADR-BE-004
- `internal/adapters/secondary/postgres/activity_ontology_migration_test.go` - `applyMigrations` variadic signature; 013 reads; 013 interleaved into up/down/cycle passes; kind assertions flipped to phase=6/task=0 with the down-reversal proof
- `internal/adapters/secondary/postgres/staffing_schema_migration_test.go` - Call site updated to the new variadic signature (013 applies in sorted order after 011, a no-op for its assertions)

## Decisions Made

- 013's predicate (`kind='task' AND parent_id IS NOT NULL`) targets ONLY subproject-derived rows: all subproject children have a parent, and no user-created 'task'-kind rows exist in the pre-deploy MVP seed scope (D-6). Root 'engagement'/'internal' rows are untouched (threat T-09-06-01 mitigation).
- 013 is skipped in the 011-test pre-state apply because it UPDATEs the `activities` table that only exists after 011 — applied earlier it fails with SQLSTATE 42P01. In the staffing test it is not skipped (applies after 011 in sorted order; flip is a no-op for those assertions).
- Down013 runs before down011 in the cycle test, matching the plan's interleave requirement and proving the reversal on the same row set before the schema rewrite.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `go test ./... -count=1` shows 18/19 packages PASS; `internal/core/services/working_group` FAILS with `TestWorkingGroupIntegration` → `relation "projects" does not exist (SQLSTATE 42P01)` (working_group_integration_test.go:51). This is the **pre-existing documented deferral** from VERIFICATION truth #14 / deferred-items.md item 2 — the test seeds tables dropped by migration 011. Plan 09-07 (same wave) is the dedicated WG test re-seed gap fix; the plan's own verification section scopes the full-suite gate to "once plans 09-06+09-07+09-08 have all landed". Out of scope for 09-06 per the scope-boundary rule.
- Pre-existing dirty state left untouched: modified `migrate` binary (compiled artifact) and `.planning/` metadata files (orchestrator-owned).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- VERIFICATION truth #10 (kind='phase') is green at the migration-test level; truth #8 (cycle) stays green; truth #14 (postgres package) green.
- Ready for 09-07 (WG integration test re-seed — fixes the last red package) and 09-08 (cycle prevention on activities.parent_id) to complete the phase gate.
- 011/012/000_full_schema remain immutable (byte-identical to HEAD), satisfying threat T-09-06-02.

---
*Phase: 09-activity-ontology*
*Completed: 2026-07-31*

## Self-Check: PASSED

- All 4 task files + SUMMARY.md exist on disk (FOUND)
- Task commits present in git history: 4ebedcb (feat), 52bd3ed (test)
- Postgres package green: `go test ./internal/adapters/secondary/postgres/ -count=1` → ok
- 011/012/000 immutability: `git diff --stat` on those files is empty
