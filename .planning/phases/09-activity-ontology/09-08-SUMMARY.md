---
phase: 09-activity-ontology
plan: 09-08
subsystem: api
tags: [activities, cycle-prevention, sentinel-errors, parent-validation, get-ancestry]

# Dependency graph
requires:
  - phase: 09-activity-ontology
    provides: activity domain/service/port/postgres-repository/mock scaffolding from 09-01..09-05 (recursive activities table, GetAncestry CTE, ADR-BE-001 sentinel convention)
provides:
  - ErrActivityCycle sentinel (ADR-BE-001) for cycle-prevention rejections on activities.parent_id
  - Shared validateParent service helper: parent exists + same-org + GetAncestry path check on BOTH Create and Update
  - MockActivityRepo.GetAncestry map-walk derivation with GetAncestryFn override
  - TestService_UpdateParentValidation (6 subtests) + create-with-parent subtest
  - ErrActivityCycle → HTTP 400 mapping in the Update handler (was 500 fallthrough)
affects: [09-activity-ontology verification, future activity-reparent frontend work, entry services that validate activity chains]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Sentinel error per ADR-BE-001 declared in the domain var block, matched with errors.Is at the handler boundary"
    - "Shared service-layer validation helper (validateParent) invoked uniformly by Create (uuid.Nil own-id) and Update (real id) — one enforcement point above a permissive repository"

key-files:
  created: []
  modified:
    - internal/core/domain/activity/activity.go
    - internal/core/services/activity/activity.go
    - internal/core/services/testdata/mocks.go
    - internal/core/services/activity/activity_test.go
    - internal/adapters/primary/http/activity_handler.go

key-decisions:
  - "Service is the single enforcement point for parent validity; the repository stays permissive by design (documented in validateParent's comment) — all API traffic passes through the service"
  - "Create passes uuid.Nil as own-activity-id so the GetAncestry path check runs uniformly on insert (a fresh id can never appear in existing ancestry) instead of a separate code path"
  - "GetAncestry mock chain includes the starting node itself so self-parent reparenting is detectable in unit tests"

patterns-established:
  - "validateParent(ctx, orgID, ownActivityID, parentID) → nil | repo error | ErrInvalidRequest | ErrActivityCycle"

requirements-completed: [P-007-D2, P-007-D6]

# Metrics
duration: 4min
completed: 2026-07-31
---

# Phase 09 Plan 08: Gap fix — cycle prevention on activities.parent_id Summary

**ErrActivityCycle sentinel + shared service-layer validateParent (Get + same-org + GetAncestry path check) enforced on both Create and Update, closing VERIFICATION GAP 3 and the latent Update same-org gap, with 7 new unit tests and a 400 handler mapping**

## Performance

- **Duration:** 4 min
- **Started:** 2026-07-31T18:23:33Z
- **Completed:** 2026-07-31T18:27:53Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- `ErrActivityCycle` sentinel declared in the activity domain var block per ADR-BE-001 (SPEC in-scope item: cycle prevention on `activities.parent_id`, path check on insert/update)
- `validateParent` service helper — parent exists (ErrActivityNotFound), same-org (ErrInvalidRequest), and GetAncestry walk rejecting when the chain contains the activity's own id (ErrActivityCycle); Create routes through it with `uuid.Nil` own-id, Update gains parent validation it previously skipped entirely (the latent gap the verifier flagged)
- `MockActivityRepo.GetAncestry` replaced its `nil, nil` stub with a map-walk derivation (chain includes the starting node, `seen` loop guard, `GetAncestryFn` override honored)
- `TestService_UpdateParentValidation` — 6 subtests over a 3-level tree (2 descendant-reparent rejections, self-parent rejection, valid same-org reparent, cross-org parent → ErrInvalidRequest, missing parent → ErrActivityNotFound) + new `TestService_Create` subtest proving the insert path check passes for a valid same-org parent
- Handler Update error switch maps `ErrActivityCycle` to 400 "activity parent would create a cycle" — no longer falls through to 500

## Task Commits

Each task was committed atomically:

1. **Task 1: Domain sentinel + service enforcement + mock ancestry** - `c65725f` (feat)
2. **Task 2: Unit tests for cycle rejection + handler 400 mapping** - `9ce170b` (test)

**Plan metadata:** pending final docs commit (09-08-SUMMARY.md)

## Files Created/Modified

- `internal/core/domain/activity/activity.go` - Added `ErrActivityCycle` sentinel with SPEC-citing comment
- `internal/core/services/activity/activity.go` - Added `validateParent` helper; Create's inline parent block replaced with shared call; Update calls it before delegating to repo
- `internal/core/services/testdata/mocks.go` - `GetAncestryFn` override field; `GetAncestry` map-walk implementation replacing the stub
- `internal/core/services/activity/activity_test.go` - `TestService_UpdateParentValidation` + create-with-parent subtest
- `internal/adapters/primary/http/activity_handler.go` - `ErrActivityCycle` case in Update error switch → 400

## Decisions Made

- Service is the single enforcement point for parent validity; the repository stays permissive by design (T-09-08-04 accepted disposition — the service is the only call path the API exposes)
- Create passes `uuid.Nil` as the own-activity-id so the path check runs uniformly per the SPEC without a separate insert-only branch
- Mock ancestry chain includes the starting node, enabling self-parent detection in unit tests (subtest 3)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- VERIFICATION GAP 3 closed: no `parent_id` cycle can be created through the API on insert or update; reparent validation is test-covered (7 new unit tests)
- Latent gap closed: Update now validates parent same-org
- Full backend suite green for the packages touched by this plan (`go test ./internal/core/services/activity/ ./internal/core/services/working_group/ ./internal/adapters/primary/http/ ./internal/adapters/secondary/postgres/ -count=1`)
- Ready for the remaining wave-3 plans (09-06/09-07 already landed) and the phase gate `go test ./... -count=1`
- Outstanding human item (from VERIFICATION, not this plan): optional live-server smoke test exercising a cycle-creating PUT expecting 400 — requires running server + DB + authenticated session

## Self-Check: PASSED

- `ErrActivityCycle` present in domain var block (grep: 2 matches)
- `validateParent` present in service (grep: 4 matches)
- `GetAncestryFn` present in mocks (grep: 4 matches)
- `TestService_UpdateParentValidation` present in test file (grep: 3 matches)
- Handler cycle case present (grep: 1 match)
- Commits `c65725f` + `9ce170b` exist in git log

---
*Phase: 09-activity-ontology*
*Completed: 2026-07-31*
