---
phase: 05-projects
plan: 01
subsystem: api
tags: [go, project, repository, service, postgres]
requires:
  - phase: 03-customers
    provides: customer CRUD patterns reused
provides:
  - UpdateProjectRequest domain struct
  - ErrHasActiveTimeEntries / ErrHasActiveSubprojectEntries sentinel errors
  - ProjectRepository port: Update, Delete, HasActiveTimeEntries
  - PG repository: Update (dynamic SET), Delete (adoption cascade tx), HasActiveTimeEntries (combined query)
  - Service layer: Update (finance gate) + Delete (finance + owner + entries check)
  - MockProjectRepo stubs for Update, Delete, HasActiveTimeEntries
affects:
  - handlers (will wire service methods)
  - frontend project CRUD pages (will call new endpoints)
tech-stack:
  added: []
  patterns:
    - Dynamic SET clause for UPDATE (matching contract_repository pattern)
    - Transaction for delete + adoption cleanup cascade
    - Combined boolean subquery for active time entries across project + subprojects
key-files:
  created: []
  modified:
    - internal/core/domain/project/project.go
    - internal/core/ports/project_repository.go
    - internal/core/services/testdata/mocks.go
    - internal/adapters/secondary/postgres/project_repository.go
    - internal/core/services/project/project.go
key-decisions:
  - "UpdateProjectRequest uses non-pointer bool for IsShared (always sent, matching CreateProjectRequest pattern rather than UpdateContractRequest's *bool)"
  - "HasActiveTimeEntries returns distinct bools for project entries and subproject entries, enabling D-06 distinct 409 error messages"
  - "Delete checks subproject entries BEFORE direct entries per D-06 ordering requirement, returning ErrHasActiveSubprojectEntries before ErrHasActiveTimeEntries"
requirements-completed: [PROJ-03, PROJ-04]
duration: 1 min
completed: 2026-06-11
---

# Phase 5 Plan 1: Project Update/Delete Backend Foundation Summary

**Domain types, port interface, PG repository methods (Update/Delete/HasActiveTimeEntries), and service-layer role-gated Update/Delete — backend foundation for project CRUD management**

## Performance

- **Duration:** 1 min
- **Started:** 2026-06-11T09:06:51Z
- **Completed:** 2026-06-11T09:07:39Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- Added `ErrHasActiveTimeEntries` and `ErrHasActiveSubprojectEntries` sentinel errors in project domain
- Added `UpdateProjectRequest` struct with name, type, contract_id, governance_model, is_shared fields
- Added `Update`, `Delete`, `HasActiveTimeEntries` to `ProjectRepository` port interface
- Added `HasActiveTimeEntriesFn` field + stub methods to `MockProjectRepo`
- Implemented PG repository `Update` with dynamic SET clause (excluding zero-value fields)
- Implemented PG repository `Delete` with adoption cleanup + project delete in single transaction
- Implemented PG repository `HasActiveTimeEntries` with combined subquery for project + subproject entries
- Implemented service `Update` with finance role gate
- Implemented service `Delete` with finance role gate + owner check + active entries check (subprojects first)
- All sentinel errors are distinct strings (no collision with contract domain errors)

## Task Commits

Each task was committed atomically:

1. **Task 1: Domain sentinel errors + UpdateProjectRequest + Port interface + Mock methods** - `05c7434` (feat)
2. **Task 2: Repository Update (dynamic SET) + Delete (adoption cascade) + HasActiveTimeEntries (combined query)** - `103605f` (feat)
3. **Task 3: Service Update and Delete methods with role gating and protection checks** - `6c37af0` (feat)

## Files Created/Modified

- `internal/core/domain/project/project.go` — Added `ErrHasActiveTimeEntries`, `ErrHasActiveSubprojectEntries`, `UpdateProjectRequest` struct
- `internal/core/ports/project_repository.go` — Added `Update`, `Delete`, `HasActiveTimeEntries` to interface
- `internal/core/services/testdata/mocks.go` — Added `HasActiveTimeEntriesFn` field, `Update`, `Delete`, `HasActiveTimeEntries` mock methods
- `internal/adapters/secondary/postgres/project_repository.go` — Added `Update` (dynamic SET), `Delete` (tx + adoption cascade), `HasActiveTimeEntries` (combined subquery)
- `internal/core/services/project/project.go` — Added `Update` (finance gate), `Delete` (finance + owner + entries check)

## Decisions Made

- **Non-pointer IsShared:** `UpdateProjectRequest` uses `bool` (not `*bool`), matching `CreateProjectRequest` rather than `UpdateContractRequest`. The frontend always provides `is_shared`, so optionality is unnecessary.
- **Distinct bool returns for HasActiveTimeEntries:** Returns `(hasEntries, hasSubprojectEntries, error)` enabling the service layer to return distinct error messages (D-06).
- **Subproject check first in Delete service:** Subproject entries are checked before direct entries, so `ErrHasActiveSubprojectEntries` takes priority over `ErrHasActiveTimeEntries` per D-06.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all tasks compiled and committed without issues.

## Next Phase Readiness

- Backend foundation for project Update/Delete is complete
- Ready for handler wiring (Plan 05-02) and frontend CRUD pages (Plan 05-03/05-04)

## Self-Check: PASSED

- `go build ./...` — passes (no output, exit 0) ✓
- `UpdateProjectRequest` struct defined with all 5 fields ✓
- 2 new sentinel errors added ✓
- `ProjectRepository` interface has 3 new methods ✓
- `MockProjectRepo` has stub implementations + `HasActiveTimeEntriesFn` ✓
- PG repo has Update (dynamic SET), Delete (tx), HasActiveTimeEntries (combined query) ✓
- Service has Update (finance gate) and Delete (finance + owner + entries check) ✓
- All sentinel errors distinct from contract domain ✓
- 3 commits with proper `feat(05-01):` format ✓

---

*Phase: 05-projects*
*Completed: 2026-06-11*
