# Phase 5: Projects - Context

**Gathered:** 2026-06-11
**Status:** Ready for planning

<domain>
## Phase Boundary

Project CRUD (create, read, update, delete) with subproject display on detail page, org-scope filtering (owned/adopted/all), governance model selection, and delete protection against active time entries on the project and its subprojects.

Builds on existing backend (ProjectHandler, ProjectService, ProjectRepository) and frontend (project-list, project-detail, create-project-dialog) that are already implemented. The existing frontend detail page has disabled Edit/Delete buttons with "Coming soon" tooltips that need to be wired.

Not in scope: Subproject CRUD management (create/edit/delete subprojects) — display only. Project manager management already exists and needs no changes.

</domain>

<decisions>
## Implementation Decisions

### Edit Project UI (PROJ-03)
- **D-01:** **Dialog-based edit** — Reuse the CreateProjectDialog pattern as an edit dialog. Not inline page edit mode. Quick to build, consistent with existing patterns.
- **D-02:** **All fields editable** — Name, type (billable/internal), contract, governance model, and shared toggle. No restrictions on which fields can change after creation.

### Delete Protection (PROJ-04)
- **D-03:** **Block on active time entries only** — Check time entry status. If all time entries for the project AND its subprojects are in approved/rejected status, deletion is allowed. If any time entry is draft/submitted/pending, block deletion.
- **D-04:** **Check time entries for subprojects too** — When evaluating delete protection, also check if any subproject of the project has active time entries. Block if any exist.
- **D-05:** **Cascade-clean adoptions** — When the creator org deletes a shared project, delete all adoption records too. No orphaned adoptions.
- **D-06:** **Return specific 409 errors** — Distinct error messages for "has active time entries" vs "has active subproject entries" vs other constraints.

### Subproject Display (PROJ-05)
- **D-07:** **Expandable section on detail page** — Collapsible accordion/expandable section below project details. Shows subproject names, descriptions, and status.
- **D-08:** **One level of nesting only** — Parent → children. No grandchild subprojects. Simple and clear for MVP.
- **D-09:** **Dedicated backend endpoint** — `GET /projects/{id}/subprojects` returns subprojects for a project. Not embedded in ProjectResponse. Clean separation, reuses existing SubprojectRepository.

### Contract ID
- **D-10:** **Contract is required** — All projects must reference a contract. No nullable contract_id. Internal projects use an "Internal Operations" contract (seed data should include this).
- **D-11:** **Seed an internal contract** — The existing seed migration (`003_seed.up.sql`) should include at least one internal/non-billable contract for internal projects.

### Existing API Integration
- **D-12:** **Org-scope filtering already works** — The existing `GET /projects?scope=owned|adopted|all&contract_id=` endpoint is sufficient. No changes needed.
- **D-13:** **Project adoption (POST /projects/{id}/adopt) already works** — No changes needed.
- **D-14:** **Project manager endpoints already work** — GET/POST/DELETE /projects/{id}/managers. No changes needed for MVP.

### the agent's Discretion
- Exact update/delete handler response format (API envelope consistent with existing patterns)
- Subproject display component design within the expandable section (accordion vs simple expand/collapse)
- Delete confirmation dialog wording and UX flow
- Test file locations and specific test cases within existing patterns
- Frontend mutation `onSuccess` behavior (navigation vs inline feedback)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project context
- `.planning/PROJECT.md` — Project overview, key decisions, constraints
- `.planning/REQUIREMENTS.md` — Requirements (PROJ-01 through PROJ-06)
- `.planning/ROADMAP.md` — Phase definitions, dependency graph, key behaviors and edge cases
- `.planning/STATE.md` — Current phase state and session info
- `docs/superpowers/specs/2026-06-08-mvp-consolidation-design.md` §Feature 5 — Projects design spec

### Backend: Project
- `internal/core/domain/project/project.go` — Domain model (needs UpdateProjectRequest, delete-related errors)
- `internal/core/ports/project_repository.go` — Repository interface (needs Update, Delete, HasActiveTimeEntries)
- `internal/core/services/project/project.go` — Service layer (needs Update, Delete with protection checks)
- `internal/core/services/project/project_test.go` — Existing unit tests (needs update/delete tests)
- `internal/adapters/primary/http/project.go` — HTTP handler (needs Update, Delete, ListSubprojects)
- `internal/adapters/primary/http/project_test.go` — Handler tests (needs update/delete tests)
- `internal/adapters/secondary/postgres/project_repository.go` — PG repo (needs Update, Delete, HasActiveTimeEntries)

### Backend: Subproject
- `internal/core/ports/subproject_repository.go` — Subproject port interface (ListByProject, etc.)
- `internal/adapters/secondary/postgres/subproject_repository.go` — PG subproject repo (full CRUD exists)
- `internal/models/surreal_models.go` — Subproject struct (line 47-56)

### Backend: Related
- `internal/core/services/contract/contract.go` — Reference pattern for delete protection with HasProjects check (Phase 4)
- `internal/core/services/unit/unit.go` — Reference pattern for delete protection with children/members checks (Phase 2)
- `internal/models/models.go` — Role, Status, GovernanceModel constants
- `cmd/server/main.go` — Route registration (needs PUT/DELETE /projects/{id} and GET /projects/{id}/subprojects)

### Frontend: Projects
- `web/src/routes/_authenticated/projects/index.tsx` — List page route
- `web/src/routes/_authenticated/projects/-components/project-list.tsx` — List page component (tabs, search, adopt, create)
- `web/src/routes/_authenticated/projects/-components/create-project-dialog.tsx` — Create dialog (reference for edit dialog pattern)
- `web/src/routes/_authenticated/projects/$id.tsx` — Detail page route
- `web/src/routes/_authenticated/projects/-components/project-detail.tsx` — Detail page component (needs edit/delete wiring, subproject section)
- `web/src/api/projects.ts` — API module (needs update/delete mutations)
- `web/src/types/api.ts` — CreateProjectRequest type (needs UpdateProjectRequest)
- `web/src/types/models.ts` — Project type (line 56-70)

### Prior Phase Context
- `.planning/phases/04-contracts/04-CONTEXT.md` — Contract CRUD patterns, delete protection, test coverage approach
- `.planning/phases/03-customers/03-CONTEXT.md` — Customer CRUD patterns, dialog-based forms
- `.planning/phases/02-org-hierarchy/02-CONTEXT.md` — Delete protection enforcement patterns

</canonical_refs>

<code_context>
## Existing Code Insights

### Already Built (no changes needed)
- Project list page with tabs (owned/adopted/all), search, adopt dialog
- Create project dialog with name, type, contract, governance, shared toggle
- Project detail page (read-only — Edit/Delete exist but are disabled with "Coming soon")
- API module with `projectsQueryOpts`, `projectQueryOpts`, `createProjectMutationOpts`, `adoptProjectMutationOpts`
- Full backend: handler (List, Create, Get, Adopt, ListManagers, AddManager, RemoveManager), service, repository
- Subproject repository with full CRUD (ListByProject, GetByID, Create, Update, Delete)
- Route wiring for GET/POST /projects, GET /projects/{id}, POST /projects/{id}/adopt, manager endpoints

### Reusable Assets
- `create-project-dialog.tsx` — Reference pattern for edit dialog (copy and adapt, same form fields)
- `internal/core/services/contract/contract.go` — Reference pattern for HasProjects-style delete protection checks
- `internal/core/errors/sentinel.go` or domain errors — Existing sentinel error pattern for delete constraints

### Established Patterns
- Dialog-based CRUD (all existing project operations use dialogs)
- `useMutation` with `onSuccess: (_, __, {client})` pattern for cache invalidation
- `useSuspenseQuery` / `useQuery` for data fetching
- Mutation `onSuccess` navigates to detail page (existing create flow)
- Tab-scoped query keys: `['projects', scope, contractId]`
- Detail `getByID` query key: `['projects', id]`
- Backend: handler delegates to service → service validates → repo executes SQL → handler formats response
- Delete protection: sentinel errors returned from service → handler maps to HTTP 409
- Subproject ID used by time entries and working groups — subproject FK constraint ensures integrity

### Integration Points
- Edit dialog: import `ContractsApis` for contract selector, `ProjectsApis.createProjectMutationOpts` as template (or new update mutation)
- Delete button: delete confirmation dialog, calls new `deleteProjectMutationOpts`, handles 409 errors
- Subproject section: needs backend endpoint `GET /projects/{id}/subprojects` → frontend fetches and displays in expandable section
- Route registration in `cmd/server/main.go`: add PUT/DELETE /projects/{id} and GET /projects/{id}/subprojects
- Sidebar projects nav link already exists (no change needed)

</code_context>

<specifics>
## Specific Ideas

- **Edit dialog**: Model after `CreateProjectDialog` — same fields but pre-populated with existing project data. Can reuse the same form component with edit mode flag.
- **Delete protection UX**: When delete is blocked by active time entries, show the error message from the backend 409 response inline in the dialog. No pre-fetching of protection status.
- **Subproject expandable section**: Simple collapsible list on the detail page. Each item shows: name, description snippet, active/inactive badge.
- **Seed data**: Include an "Internal Operations" contract in the seed for internal projects.
- **Update on mutation success**: Navigate to detail page or show the updated data inline (the agent's discretion — match existing create pattern).

</specifics>

<deferred>
## Deferred Ideas

### Subproject CRUD Management
Creating, editing, deleting subprojects from the frontend. Backend CRUD already exists in SubprojectRepository but no frontend or handler endpoints. Deferred because the MVP scope is display-only. Belongs in a future phase after time entries are built.

### Multi-level Subproject Nesting
Recursive subproject hierarchy (grandchildren, etc.). Deferred — one level is sufficient for MVP.

</deferred>

---

*Phase: 05-Projects*
*Context gathered: 2026-06-11*
