---
phase: 05-projects
plan: 01
subsystem: backend
tags: [projects, crud, update, delete, time-entry-protection, pg-repository]
requires: []
provides: [project-update-delete-api, project-delete-protection, subproject-time-entry-check]
affects: [time-entries, contracts]
tech-stack:
  added: []
  patterns: [dynamic-SET-update, transactional-delete-with-cascade, combined-subquery-active-entries-check]
decisions:
  - "D-01: Dialog-based edit (not inline edit mode) — already decided in context"
  - "D-03: Delete blocked only on active time entries (draft/submitted/pending status)"
  - "D-04: Check time entries for subprojects too"
  - "D-05: Cascade-clean adoptions on delete"
  - "D-06: Return specific 409 errors (distinct project vs subproject messages)"
metrics:
  duration: null
  completed_date: "2026-06-11"
key-files:
  created: []
  modified:
    - internal/core/domain/project/project.go
    - internal/core/ports/project_repository.go
    - internal/core/services/testdata/mocks.go
    - internal/adapters/secondary/postgres/project_repository.go
    - internal/core/services/project/project.go
---

# Phase 05 Plan 01: Backend domain/ports/mocks/repo/service for Project Update + Delete

Implement backend foundation for project Update, Delete (with active time entry protection), and HasActiveTimeEntries checks — domain types, repository interface, mock, PostgreSQL repository, and service-layer logic.

## Verification Results

- `go build ./...` — **PASSED** (no output = no errors)
- `go vet ./internal/...` — **PASSED**
- Sentinel errors are distinct across domains:
  - `contract.ErrHasActiveProjects = "contract has active projects"`
  - `project.ErrHasActiveTimeEntries = "project has active time entries"`
  - `project.ErrHasActiveSubprojectEntries = "subproject has active time entries"` (no collision)

## Code Delivered

### Task 1 — Domain + Ports + Mocks (commit `05c7434`)

**Domain (`internal/core/domain/project/project.go`):**
- `ErrHasActiveTimeEntries` and `ErrHasActiveSubprojectEntries` sentinel errors
- `UpdateProjectRequest` struct (name, type, contract_id, governance_model, is_shared)

**Port (`internal/core/ports/project_repository.go`):**
- `Update(ctx, orgID, projectID, *UpdateProjectRequest) (*ProjectResponse, error)`
- `Delete(ctx, orgID, projectID) error`
- `HasActiveTimeEntries(ctx, projectID) (bool, bool, error)` — returns both project + subproject flags

**Mocks (`internal/core/services/testdata/mocks.go`):**
- `HasActiveTimeEntriesFn` field on `MockProjectRepo`
- `Update`, `Delete`, `HasActiveTimeEntries` stub methods

### Task 2 — PG Repository (commit `103605f`)

**`internal/adapters/secondary/postgres/project_repository.go`:**
- `Update` — dynamic SET clause (skips zero-value fields; is_shared always sent), `created_by_org_id` WHERE, returns `ErrProjectNotFound` on 0 rows affected
- `Delete` — transaction deleting `project_adoptions` first, then `projects` with `created_by_org_id` WHERE, returns `ErrProjectNotFound` on 0 rows
- `HasActiveTimeEntries` — combined subquery: checks `time_entries` directly for the project AND via `subprojects` JOIN for subproject entries (status NOT IN approved/rejected, is_deleted = false)

### Task 3 — Service (commit `6c37af0`)

**`internal/core/services/project/project.go`:**
- `Update` — finance role gate (`role != RoleFinance` → `ErrForbidden`), delegates to repo
- `Delete` — finance role gate + owner check (`CreatedByOrgID != orgID` → `ErrForbidden`) + active entries check (subproject entries checked FIRST per D-06 → `ErrHasActiveSubprojectEntries`, then project entries → `ErrHasActiveTimeEntries`), delegates to repo

## Deviations from Plan

**None.** Plan was already implemented and committed in prior execution. All code matches plan specifications exactly.

## Auth Gates

None encountered.

## Known Stubs

None.

## Threat Flags

None — all threat mitigations from plan's threat model are implemented:
- T-05-01: Finance role gate on Update (ASVS V4)
- T-05-02: Finance role + owner check on Delete (ASVS V4)
- T-05-03: `created_by_org_id` WHERE in dynamic SET update (ASVS V4)
- T-05-04: Distinct error messages for project vs subproject entries (ASVS V8)
- T-05-05: `created_by_org_id` WHERE + adoption cascade in delete transaction (ASVS V4)

## Self-Check: PASSED

| Check | Status |
|-------|--------|
| `go build ./...` passes | ✅ |
| `go vet ./internal/...` passes | ✅ |
| `UpdateProjectRequest` struct exists with 5 fields | ✅ (project.go:65-71) |
| 2 new sentinel errors exist | ✅ (project.go:17-18) |
| `ProjectRepository` has 3 new methods | ✅ (project_repository.go:20-22) |
| `MockProjectRepo` has stubs + `HasActiveTimeEntriesFn` | ✅ (mocks.go:476, 519-532) |
| PG repo has Update (dynamic SET) | ✅ (project_repository.go:213-266) |
| PG repo has Delete (tx + adoption cascade) | ✅ (project_repository.go:269-292) |
| PG repo has HasActiveTimeEntries (combined query) | ✅ (project_repository.go:295-311) |
| Service Update (finance gate) | ✅ (project.go:57-62) |
| Service Delete (finance + owner + entries check) | ✅ (project.go:64-86) |
| Sentinel errors distinct from contract domain | ✅ (no collisions) |
| Commits exist in git history | ✅ (05c7434, 103605f, 6c37af0) |
