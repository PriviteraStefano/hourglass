---
phase: 05-mvp-consolidation
plan: 02
subsystem: api
tags: [go, project, handler, routing, subproject]
requires:
  - phase: 05-mvp-consolidation
    plan: 01
    provides: Project service Update/Delete + ProjectRepository Update/Delete/HasActiveTimeEntries
provides:
  - HTTP endpoints PUT/DELETE /projects/{id} with role gating and delete protection
  - GET /projects/{id}/subprojects endpoint for subproject listing
  - SubprojectRepository injection into ProjectHandler
affects: [frontend integration in 05-03]
tech-stack:
  added: []
  patterns: [error-switch pattern mapping sentinel errors to HTTP status codes]
key-files:
  created: []
  modified:
    - internal/adapters/primary/http/project.go
    - internal/adapters/primary/http/project_test.go
    - internal/adapters/primary/http/handler_test_helper.go
    - cmd/server/main.go
key-decisions:
  - "ListSubprojects uses injected SubprojectRepository directly (no service delegation needed for read-only query)"
  - "Delete protection returns distinct 409 messages for project-level vs subproject-level active entries"
requirements-completed:
  - PROJ-03
  - PROJ-04
  - PROJ-05
duration: 2 min
completed: 2026-06-11
---

# Phase 5: MVP Consolidation Summary - Plan 02

**Wire HTTP Update/Delete/ListSubprojects handler methods + subproject repository injection + route registration in main.go**

## Performance

- **Duration:** 2 min
- **Started:** 2026-06-11T09:29:50Z
- **Completed:** 2026-06-11T09:31:21Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Added `UpdateProjectRequest` DTO and `Update`/`Delete`/`ListSubprojects` handler methods to `ProjectHandler`
- Extended `ProjectHandler` with `subprojectRepo ports.SubprojectRepository` field and constructor accepts it
- Mapped sentinel errors to HTTP statuses: Forbidden→403, NotFound→404, `ErrHasActiveTimeEntries`→409, `ErrHasActiveSubprojectEntries`→409
- Registered `PUT /projects/{id}`, `DELETE /projects/{id}`, `GET /projects/{id}/subprojects` with auth middleware
- Updated `project_test.go` (8 call sites) and `handler_test_helper.go` for new two-arg constructor
- Full `go build ./...` passes cleanly

## Task Commits

Each task was committed atomically:

1. **Task 1: Add Update/Delete/ListSubprojects handler methods + DTO** - `7a9b71f` (feat)
2. **Task 2: Route registration + SubprojectRepository injection** - `e719044` (feat)

**Plan metadata:** (committed below)

## Files Created/Modified

- `internal/adapters/primary/http/project.go` - Added `subprojectRepo` field, `UpdateProjectRequest` DTO, `Update`/`Delete`/`ListSubprojects` handlers
- `internal/adapters/primary/http/project_test.go` - Updated 8 `NewProjectHandler(nil)` calls to `NewProjectHandler(nil, nil)`
- `internal/adapters/primary/http/handler_test_helper.go` - Added `subprojectRepo := postgres.NewSubprojectRepository(pool)` and passed to constructor
- `cmd/server/main.go` - Added `subprojectRepo` injection, updated constructor, registered 3 new routes

## Decisions Made

- `ListSubprojects` uses the injected `SubprojectRepository` directly from the handler — no service delegation needed for this simple read-only query
- Error mapping follows the contract.go pattern: `switch err` with sentinel errors, distinct 409 messages for project vs subproject active entries (per D-06)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all vet/compilation checks passed first time.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Backend endpoints for Update/Delete/ListSubprojects are wired and compiled
- Ready for frontend integration (05-03) to wire the Edit/Delete buttons and subproject section

## Self-Check: PASSED

- ✅ Files verified: SUMMARY.md exists
- ✅ Commits verified: 7a9b71f, e719044, 4d9b3ef all present
- ✅ `go build ./...` succeeds

---

*Phase: 05-mvp-consolidation*
*Completed: 2026-06-11*
