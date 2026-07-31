---
phase: 09-activity-ontology
plan: 09-07
subsystem: testing
tags: [postgres, activities, working_group, migration-011, integration-test, testcontainers]

# Dependency graph
requires:
  - phase: 09-activity-ontology
    provides: migration 011 (projects/subprojects dropped, activities + activity_kinds tables)
provides:
  - Green working_group integration test seeding via activity_kinds + activities (GAP 2 closure)
  - VERIFICATION truth #14 contribution: full suite green (19/19 packages)
affects: [09-08 (last wave-3 plan), phase 10 (SubprojectID field rename), verification gate]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Test seeding follows the in-repo pattern: seed an activity_kinds catalog row before inserting an activities row (FK on (org_id, name))"

key-files:
  created: []
  modified:
    - internal/core/services/working_group/working_group_integration_test.go

key-decisions:
  - "seedWGData seeds one root 'engagement' activity (parent_id NULL, contract_id NULL) instead of a project+subproject pair; the WG's legacy SubprojectID field still maps to activities.activity_id at the repo layer, so service-side calls are unchanged"

patterns-established:
  - "Seed helper convention for the activity schema: kind catalog row via ON CONFLICT DO NOTHING, then activities row with only the NOT NULL columns (defaults for governance_model/is_shared/is_active/timestamps)"

requirements-completed: [P-007-D5, BE-014-R5]

# Metrics
duration: 3min
completed: 2026-07-31
---

# Phase 09 Plan 07: Gap fix — working_group integration test re-seed (GAP 2) Summary

**Re-seeded `seedWGData` in the working_group integration test onto the activity ontology schema (activity_kinds + activities), closing VERIFICATION GAP 2 — the package no longer touches the migration-011-dropped projects/subprojects tables and all 4 subtests pass against the real activity-backed schema**

## Performance

- **Duration:** 3 min
- **Started:** 2026-07-31T18:19:00Z
- **Completed:** 2026-07-31T18:22:00Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- `seedWGData` now seeds an `activity_kinds('engagement')` catalog row plus one root `activities` row (parent_id NULL, contract_id NULL, defaults for governance_model/is_shared/is_active/timestamps) — zero references to the dropped `projects`/`subprojects` tables
- Renamed `projectID`/`subprojectID` locals to `activityID` in the helper and all three consuming subtests; the legacy `SubprojectID` request field still carries the activity id through `WorkingGroupRepository.Create` into the `activity_id` column (ADR-P-007 D-5), so the service-side call is byte-for-byte unchanged
- All 4 integration subtests (CreateAndGetByID, ListByOrg, GetNotFound, Delete) pass; the full `./internal/core/services/...` suite is green (12/12 service packages)

## Task Commits

Each task was committed atomically:

1. **Task 1: Re-seed seedWGData with activity_kinds + activities** - `d94c8ed` (fix)

**Plan metadata:** pending orchestrator final-commit (executor does not own STATE/ROADMAP writes in this wave)

## Files Created/Modified
- `internal/core/services/working_group/working_group_integration_test.go` - seedWGData rewritten: org + manager user + activity_kinds('engagement') row (ON CONFLICT DO NOTHING) + one root activities row; returns (orgID, managerID, activityID); subtest destructuring updated with assertions semantically identical

## Decisions Made
- Re-seed uses a root 'engagement' activity matching the migration-011 project→activity mapping (projects became kind 'engagement', parent NULL) so the fixture mirrors production data shape exactly. contract_id stays NULL (internal activity per D-3).
- The `SubprojectID` field name is intentionally untouched — its rename to `ActivityID` is Phase 10 scope (per plan instructions).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- GAP 2 closed: `go test ./internal/core/services/working_group/ -run TestWorkingGroupIntegration` is green, removing the last red package that blocked VERIFICATION truth #14 (19/19)
- Ready for 09-08 (the final wave-3 plan); after it lands, the phase gate `go test ./... -count=1` should report 19/19 packages PASS
- Phase 10 can proceed with the `SubprojectID` → `ActivityID` field rename now that the test fixture no longer depends on the legacy naming

---
*Phase: 09-activity-ontology*
*Completed: 2026-07-31*
