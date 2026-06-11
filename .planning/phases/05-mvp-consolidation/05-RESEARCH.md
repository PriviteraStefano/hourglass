# Phase 5: Projects - Research

**Researched:** 2026-06-11
**Domain:** Project CRUD with subproject display, governance models, org-scope filtering, delete protection
**Confidence:** HIGH

## Summary

This phase wires up the existing project list/create/detail skeleton (frontend and backend) with Edit, Delete, and subproject display. The backend already has `ProjectHandler` (List, Create, Get, Adopt, ListManagers, AddManager, RemoveManager), `ProjectService`, and `ProjectRepository` — but lacks `Update`, `Delete`, and `HasActiveTimeEntries`. The subproject repository has full CRUD but no handler endpoint. The frontend already has a project detail page with disabled Edit/Delete buttons ("Coming soon") that need to be wired to real dialogs.

**Primary recommendation:** Follow the contract CRUD pattern established in Phase 4 — `UpdateProjectRequest` in domain, `Update`/`Delete`/`HasActiveTimeEntries` on the repository interface, `Update`/`Delete` in service with role gating (finance-only), and corresponding handler methods. Wire the subproject listing as `GET /projects/{id}/subprojects` using the existing `SubprojectRepository.ListByProject`. Frontend: adapt `CreateProjectDialog` into an edit dialog, add delete confirmation with 409 error handling, and add an expandable subproject section on the detail page.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Project Update validation | Backend / Service | — | Business logic (role check, field validation) belongs in service |
| Project Delete protection | Backend / Service | Backend / Repository | Service checks role and calls repo for time entry existence check |
| Subproject listing | Backend / Repository | Backend / Handler | Pure data read, SubprojectRepository.ListByProject already exists |
| Edit project UI | Browser / Client | — | Dialog-based form, consistent with existing create pattern |
| Delete confirmation UX | Browser / Client | — | Client-side confirmation dialog, backend returns 409 on violations |
| Subproject display | Browser / Client | — | Expandable section on detail page, fetches from dedicated endpoint |

## Standard Stack

### Core — No new libraries needed

This phase introduces zero new dependencies. All implementation uses the existing stack:
- **Backend:** Go 1.26.1, standard library `net/http`, `pgx/v5` for PostgreSQL, `github.com/google/uuid`
- **Frontend:** React 19, TanStack Router v1, TanStack React Query v5, shadcn/ui components (Dialog, Select, Button, Badge, Accordion/Collapsible), lucide-react icons, Tailwind CSS

### Verified Versions
```bash
# Go — confirmed via go.mod
# All backend deps already present — no new npm installs needed
```

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Dedicated `GET /projects/{id}/subprojects` | Embed subprojects in `ProjectResponse` | D-09 explicitly chose dedicated endpoint. Cleaner separation, avoids bloating ProjectResponse. |
| Dialog-based edit | Inline page edit | D-01 chose dialog-based. Faster to build, consistent with CreateProjectDialog pattern. |
| Accordion for subproject display | Simple expand/collapse div | The agent's discretion. shadcn Accordion is already available, but a simple collapsible div also works. |

## Package Legitimacy Audit

> **Not applicable.** This phase adds zero external packages. All code uses existing backend (Go stdlib + pgx) and frontend (React, TanStack, shadcn/ui) dependencies already installed.

## Architecture Patterns

### System Architecture Diagram

```
Frontend (Browser)                    Backend (Go)                     PostgreSQL
==========================           ======================           =============

[Project Detail Page]               [ProjectHandler]
  │                                    │
  │  PUT /projects/{id}                │
  │  {name, type, contract_id, ...}   ├──► ProjectService.Update()
  │◄──── {data: project}              │      ├── check finance role
  │                                    │      └──► ProjectRepository.Update()
  │                                    │             └── dynamic SET SQL
  │  DELETE /projects/{id}             │
  │◄──── {data: message}              ├──► ProjectService.Delete()
  │       or {error: "..."} (409)     │      ├── check finance role
  │                                    │      ├── check project ownership
  │                                    │      ├──► HasActiveTimeEntries(projectID)
  │                                    │      │     └── check subprojects too
  │                                    │      ├── delete project_adoptions
  │                                    │      └──► ProjectRepository.Delete()
  │                                    │
  │  GET /projects/{id}/subprojects    │
  │◄──── {data: subprojects[]}        ├──► SubprojectRepository.ListByProject()
  │                                    │
  [Edit Dialog]                        │
  │  — opens pre-populated form        │
  │  — all fields editable             │
  │  — on submit → PUT /projects/{id}  │
  │                                    │
  [Delete Confirmation Dialog]         │
  │  — shows warning text              │
  │  — on confirm → DELETE /projects/{id}
  │  — on 409 → show error message     │
  │                                    │
  [Subproject Expandable Section]      │
  │  — accordion/collapsible below     │
  │    project details                 │
  │  — fetches on mount if expanded    │
```

### Recommended Project Structure

No structural changes needed. All changes are additions to existing files:

```
Backend additions:
  internal/core/domain/project/project.go
    → Add UpdateProjectRequest struct
    → Add ErrHasActiveTimeEntries, ErrHasActiveSubprojectEntries sentinel errors
  internal/core/ports/project_repository.go
    → Add Update, Delete, HasActiveTimeEntries method signatures
  internal/core/services/project/project.go
    → Add Update, Delete methods with role check + delete protection
  internal/adapters/primary/http/project.go
    → Add Update, Delete, ListSubprojects handler methods
    → Add UpdateProjectRequest HTTP DTO
  internal/adapters/secondary/postgres/project_repository.go
    → Add Update (dynamic SET), Delete (with cascade adoptions), HasActiveTimeEntries
  internal/core/services/testdata/mocks.go
    → Add Update, Delete, HasActiveTimeEntries to MockProjectRepo
  cmd/server/main.go
    → Add routes: PUT /projects/{id}, DELETE /projects/{id}, GET /projects/{id}/subprojects
    → Instantiate SubprojectRepository, inject into ProjectHandler

Frontend additions:
  web/src/api/projects.ts
    → Add updateProjectMutationOpts, deleteProjectMutationOpts, subprojectsQueryOpts
  web/src/routes/_authenticated/projects/-components/project-detail.tsx
    → Wire Edit button → edit dialog
    → Wire Delete button → delete confirmation dialog
    → Add expandable subproject section
  web/src/routes/_authenticated/projects/-components/edit-project-dialog.tsx (new)
    → Reuse CreateProjectDialog pattern with pre-populated fields
  web/src/types/api.ts
    → Add UpdateProjectRequest type
```

### Pattern 1: Dynamic SET Update (Backend)
**What:** Build UPDATE SQL dynamically from non-zero request fields, following the contract repo pattern.
**When to use:** For the `ProjectRepository.Update` method — only update fields that are present in the request.
**Example:**
```go
// Source: [CITED: internal/adapters/secondary/postgres/contract_repository.go:128-206]
func (r *ProjectRepository) Update(ctx context.Context, orgID, projectID uuid.UUID, req *projectdomain.UpdateProjectRequest) (*projectdomain.ProjectResponse, error) {
    var sets []string
    var args []interface{}
    argIdx := 1

    if req.Name != "" {
        sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
        args = append(args, req.Name)
        argIdx++
    }
    // ... more fields (type, governance_model, is_shared)

    allSets := append(sets, "updated_at = NOW()")
    whereIdx := argIdx
    args = append(args, projectID, orgID)

    query := fmt.Sprintf(`UPDATE projects SET %s WHERE id = $%d AND created_by_org_id = $%d`,
        strings.Join(allSets, ", "), whereIdx, whereIdx+1)

    cmd, err := r.pool.Exec(ctx, query, args...)
    if err != nil {
        return nil, wrapPGError(err, "update project")
    }
    if cmd.RowsAffected() == 0 {
        return nil, projectdomain.ErrProjectNotFound
    }
    return r.Get(ctx, orgID, projectID)
}
```

### Pattern 2: Delete Protection (Backend)
**What:** Check for active time entries before deletion. If any time entry on the project or its subprojects is in draft/submitted/pending status, block deletion with a descriptive 409 error.
**When to use:** For the `ProjectService.Delete` method.
**Example:**
```go
// Source: [CITED: internal/core/services/contract/contract.go:59-86]
func (s *Service) Delete(ctx context.Context, role string, orgID, projectID uuid.UUID) error {
    if role != string(models.RoleFinance) {
        return projectdomain.ErrForbidden
    }
    existing, err := s.repo.Get(ctx, orgID, projectID)
    if err != nil {
        return err
    }
    if existing.CreatedByOrgID != orgID {
        return projectdomain.ErrForbidden
    }

    hasEntries, hasSubprojectEntries, err := s.repo.HasActiveTimeEntries(ctx, projectID)
    if err != nil {
        return err
    }
    if hasSubprojectEntries {
        return projectdomain.ErrHasActiveSubprojectEntries
    }
    if hasEntries {
        return projectdomain.ErrHasActiveTimeEntries
    }
    return s.repo.Delete(ctx, orgID, projectID)
}
```

### Pattern 3: Edit Dialog (Frontend)
**What:** Copy and adapt the existing `CreateProjectDialog` — same fields but pre-populated with existing project data.
**When to use:** For the edit project functionality (PROJ-03).
**Key differences from CreateProjectDialog:**
- Title: "Edit Project" instead of "Create Project"
- Submit button: "Save Changes" instead of "Create"
- Form fields pre-populated with existing project data
- Calls `updateProjectMutationOpts` instead of `createProjectMutationOpts`
- On success: invalidate `['projects', id]` query key and update displayed data

### Anti-Patterns to Avoid
- **Hard-deleting without adoption cleanup:** If a shared project is deleted, all `project_adoptions` records must be cascade-deleted first. D-05 requires this. The repository `Delete` method should run `DELETE FROM project_adoptions WHERE project_id = $1` before the project `DELETE`.
- **Checking only direct project time entries:** D-04 requires checking subproject time entries too. The `HasActiveTimeEntries` query must account for entries where `project_id = $1 OR subproject_id IN (SELECT id FROM subprojects WHERE project_id = $1)`.
- **Returning generic 409:** D-06 requires distinct error messages for "has active time entries" vs "has active subproject entries" vs other constraints.
- **Fetching subproject data inline in project response:** D-09 chose a dedicated endpoint. Do not join subprojects into the project query — use a separate fetch.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Update SQL building | Hand-coded field-by-field | Dynamic SET with `fmt.Sprintf` | Already established in contract repo pattern. Consistent approach. |
| Delete protection | Pre-fetch status in frontend | Backend 409 errors | D-06 says return specific 409s from backend. No pre-fetching. |
| Form component for edit | Separate component from scratch | Fork `CreateProjectDialog` | Same fields, same layout. Change mode via prop or create sibling file. |

**Key insight:** This phase follows patterns already established by Phase 4 (Contracts) and Phase 3 (Customers). The contract handler's Update/Delete pattern with role-gating and sentinel errors is the canonical reference.

## Common Pitfalls

### Pitfall 1: Not Checking Subproject Time Entries on Delete
**What goes wrong:** Project delete is blocked for time entries on the project itself, but a user can delete a project that has active time entries on its subprojects — causing FK constraint violations or orphaned data.
**Why it happens:** The `HasActiveTimeEntries` SQL only checks `time_entries.project_id`, missing `time_entries.subproject_id`.
**How to avoid:** Use a combined query:
```sql
SELECT COUNT(*) FROM time_entries
WHERE (project_id = $1 OR subproject_id IN (SELECT id FROM subprojects WHERE project_id = $1))
  AND status NOT IN ('approved', 'rejected')
  AND is_deleted = false
```
**Warning signs:** Time entries with approved/rejected status should not block deletion. Only draft/submitted/pending.

### Pitfall 2: Not Cascade-Deleting Adoptions
**What goes wrong:** Deleting a shared project leaves orphaned `project_adoptions` records referencing a nonexistent project.
**Why it happens:** The `projects` table likely has no `ON DELETE CASCADE` on the `project_adoptions` FK, and the repo Delete method doesn't clean them up manually.
**How to avoid:** Run `DELETE FROM project_adoptions WHERE project_id = $1` in a transaction before `DELETE FROM projects WHERE id = $1 AND created_by_org_id = $2`.
**Warning signs:** After deletion, the project_adoptions table still has rows with the deleted project ID.

### Pitfall 3: Forgetting to Extract Subproject Response Fields
**What goes wrong:** `SubprojectRepository.ListByProject` returns `[]models.Subproject` where `ID` and `ProjectID` are `string` types (UUID as string). The handler returns this raw, but the frontend expects proper JSON field names.
**Why it happens:** `models.Subproject` has `ID` and `ProjectID` as `string` with `json:"id"` / `json:"project_id"` tags — but these are stored as UUID in PG and converted to string in the scan helper.
**How to avoid:** The existing `scanSubproject` function handles this correctly. Ensure the handler returns the response directly via `api.RespondWithJSON`.
**Warning signs:** The frontend receives UUID string representations instead of proper JSON.

## Code Examples

### Backend: Repository Update Method
```go
// Source: [CITED: internal/adapters/secondary/postgres/contract_repository.go:128-206]
// Follow this pattern for ProjectRepository.Update
```

### Backend: Repository HasActiveTimeEntries Query
```go
func (r *ProjectRepository) HasActiveTimeEntries(ctx context.Context, projectID uuid.UUID) (directCount int, subprojectCount int, err error) {
    query := `SELECT
        (SELECT COUNT(*) FROM time_entries
         WHERE project_id = $1
           AND status NOT IN ('approved', 'rejected')
           AND is_deleted = false) AS direct_count,
        (SELECT COUNT(*) FROM time_entries te
         WHERE te.subproject_id IN (SELECT id FROM subprojects WHERE project_id = $1)
           AND te.status NOT IN ('approved', 'rejected')
           AND te.is_deleted = false) AS subproject_count`
    err = r.pool.QueryRow(ctx, query, projectID).Scan(&directCount, &subprojectCount)
    return
}
```

### Backend: Repository Delete with Adoption Cleanup
```go
func (r *ProjectRepository) Delete(ctx context.Context, orgID, projectID uuid.UUID) error {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback(ctx)

    // Cascade-clean adoptions (D-05)
    _, err = tx.Exec(ctx, `DELETE FROM project_adoptions WHERE project_id = $1`, projectID)
    if err != nil {
        return wrapPGError(err, "delete project adoptions")
    }

    cmd, err := tx.Exec(ctx,
        `DELETE FROM projects WHERE id = $1 AND created_by_org_id = $2`,
        projectID, orgID)
    if err != nil {
        return wrapPGError(err, "delete project")
    }
    if cmd.RowsAffected() == 0 {
        return projectdomain.ErrProjectNotFound
    }

    return tx.Commit(ctx)
}
```

### Frontend: Update/Delete Mutation Options (web/src/api/projects.ts)
```typescript
const updateProjectMutationOpts = mutationOptions({
  mutationFn: ({id, data}: {id: string; data: UpdateProjectRequest}) =>
    api<Project>(`/projects/${id}`, {method: 'PUT', body: JSON.stringify(data)}),
  onSuccess: (_, {id}, {client}) => {
    client.invalidateQueries({queryKey: ['projects']})
    client.invalidateQueries({queryKey: ['projects', id]})
    toast.success('Project updated')
  },
})

const deleteProjectMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<void>(`/projects/${id}`, {method: 'DELETE'}),
  onSuccess: (_, __, {client}) => {
    client.invalidateQueries({queryKey: ['projects']})
    toast.success('Project deleted')
  },
  onError: (error: ApiError) => {
    // Error contains message from backend 409
    toast.error(error.message || 'Failed to delete project')
  },
})

const subprojectsQueryOpts = (id: string) => queryOptions({
  queryKey: ['projects', id, 'subprojects'],
  queryFn: () => api<Subproject[]>(`/projects/${id}/subprojects`),
  enabled: !!id,
})
```

### Frontend: Edit Dialog Pattern (web/src/.../edit-project-dialog.tsx)
```typescript
// Copy create-project-dialog.tsx structure
// Add project prop for pre-population
interface EditProjectDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  project: Project  // pre-populate from current data
}

// Pre-populate state from project prop
const [name, setName] = useState(project.name)
const [type, setType] = useState(project.type)
// ... same for other fields

// On submit:
const handleSubmit = () => {
  updateProject.mutate(
    { id: project.id, data: { name, type, contract_id, governance_model, is_shared } },
    { onSuccess: () => { onOpenChange(false); /* navigation or inline update */ } }
  )
}
```

## State of the Art

No significant state changes — this is additive CRUD functionality built on existing architecture. The contract handler's Update/Delete pattern (Phase 4) is the canonical pattern to follow.

**Deprecated/outdated:**
- The existing "Coming soon" tooltip on Edit/Delete buttons — to be removed once wired.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Update/Delete operations require finance role | Architecture Patterns | If user needs different role gating, change the role check in the service |

## Open Questions (RESOLVED)

1. **Should project Update/Delete require finance role?**
   - What we know: The contract Update/Delete requires `models.RoleFinance`. The AddManager/RemoveManager (project) also requires finance. The existing project handlers don't have an Update/Delete that we can reference.
   - What's unclear: D-02 says "all fields editable" but doesn't specify role gating. It's likely that update and delete should follow the same pattern as contracts (finance-only), but this should be confirmed.
   - Recommendation: **Assume finance role** to match contract pattern. The service layer will be `if role != string(models.RoleFinance) { return ErrForbidden }`. If user wants a different gate, it's a one-line change in the service.
   - **RESOLVED:** Plans consistently apply finance role gating for Update/Delete, matching the contract pattern.

2. **Should Edit be owned-only or for adopted orgs too?**
   - What we know: Delete is owned-only (the service checks `existing.CreatedByOrgID != orgID` and returns forbidden).
   - What's unclear: Should an adopted org be able to edit project fields? The governance model suggests shared projects need consensus, but MVP may allow creator edits only.
   - Recommendation: **Assume owned-only** to match the delete pattern. Only the creating organization can edit/delete their projects.
   - **RESOLVED:** Plans apply owner-only edit/delete, matching the contract delete pattern.

## Validation Architecture

> `workflow.nyquist_validation` is not explicitly `false` in `.planning/config.json` — including this section.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` + `stretchr/testify` (backend), Vitest + Playwright (frontend) |
| Config file | `go.mod` for Go; `web/package.json` for frontend |
| Quick run command | `go test -count=1 -timeout 120s ./internal/core/services/project/...` |
| Full suite command | `go test -count=1 -timeout 300s ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PROJ-01 | Project list filters correctly | unit (mock) | `go test ./internal/core/services/project/... -run TestService_List` | ✅ `project_test.go` |
| PROJ-02 | Create validates required fields | unit (mock) | `go test ./internal/core/services/project/... -run TestService_Create` | ✅ `project_test.go` |
| PROJ-03 | Update project (service) | unit (mock) | `go test ./internal/core/services/project/... -run TestService_Update` | ❌ Needs new test |
| PROJ-03 | Update project (handler) | unit (mock) | `go test ./internal/adapters/primary/http/... -run TestProjectHandler_Update` | ❌ Needs new test |
| PROJ-04 | Delete blocked on active entries | unit (mock) | `go test ./internal/core/services/project/... -run TestService_Delete` | ❌ Needs new test |
| PROJ-04 | Delete returns 409 (handler) | unit (mock) | `go test ./internal/adapters/primary/http/... -run TestProjectHandler_Delete` | ❌ Needs new test |
| PROJ-04 | Delete cascade-cleans adoptions (repo) | integration | `go test ./internal/adapters/secondary/postgres/... -run TestProjectRepository_Delete` | ❌ Needs new test |
| PROJ-05 | List subprojects endpoint | unit (mock) | `go test ./internal/adapters/primary/http/... -run TestProjectHandler_ListSubprojects` | ❌ Needs new test |
| PROJ-06 | Adopted projects show creation org | integration | Already covered by existing `Get` response (`created_by_org_name`) | ✅ |

### Sampling Rate
- **Per task commit:** `go test -count=1 -timeout 120s ./internal/core/services/project/...` (service tests)
- **Per wave merge:** `go test -count=1 -timeout 300s ./internal/...` (full backend)
- **Phase gate:** Full suite green + `cd web && bun run build` (type-check)

### Wave 0 Gaps
- [ ] `internal/core/services/project/project_test.go` — add `TestService_Update`, `TestService_Delete` subtests
- [ ] `internal/adapters/primary/http/project_test.go` — add `TestProjectHandler_Update`, `TestProjectHandler_Delete`, `TestProjectHandler_ListSubprojects`
- [ ] `internal/adapters/secondary/postgres/project_repository_test.go` — add `TestProjectRepository_Update`, `TestProjectRepository_Delete`, `TestProjectRepository_HasActiveTimeEntries`
- [ ] `internal/core/services/testdata/mocks.go` — add `Update`, `Delete`, `HasActiveTimeEntries` to `MockProjectRepo`

## Security Domain

> `security_enforcement` key absent from config.json — treated as enabled.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V4 Access Control | yes | Role gating (finance-only) on Update/Delete operations |
| V5 Input Validation | yes | Service-layer validation of `UpdateProjectRequest` fields |
| V8 Data Protection | yes | 409 errors leak no internal state beyond necessary error messages |

### Known Threat Patterns for {Go + React}
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Unauthorized update/delete | Elevation of Privilege | Finance role check in service layer (matching contract pattern) |
| Data exposure via error messages | Information Disclosure | Return generic error messages on 500, specific but not revealing on 409 |

## Sources

### Primary (HIGH confidence)
- [CITED: internal/adapters/primary/http/contract.go] — Contract Update/Delete handler pattern (the canonical reference for this phase)
- [CITED: internal/core/services/contract/contract.go] — Contract Delete protection with HasTimeEntries/HasProjects checks
- [CITED: internal/adapters/secondary/postgres/contract_repository.go] — Dynamic SET update SQL and Delete with RowsAffected check
- [CITED: internal/core/domain/contract/contract.go] — Sentinel error pattern (ErrHasActiveProjects, etc.)
- [CITED: internal/adapters/secondary/postgres/project_repository.go] — Existing project query patterns
- [CITED: web/src/routes/_authenticated/projects/-components/create-project-dialog.tsx] — Create dialog reference for edit dialog

### Secondary (MEDIUM confidence)
- [CITED: web/src/types/api.ts] — CreateProjectRequest type, template for UpdateProjectRequest

### Tertiary (LOW confidence)
- No tertiary sources — all findings verified against existing codebase.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Verified against existing project files
- Architecture: HIGH - Patterns established by contracts/customers phases
- Pitfalls: HIGH - Derived from known PG migration patterns and the exact existing codebase

**Research date:** 2026-06-11
**Valid until:** 2026-07-11 (stable Go/postgres/stdlib stack, no fast-moving deps)
