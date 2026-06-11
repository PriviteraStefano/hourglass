# Phase 5: Projects - Pattern Map

**Mapped:** 2026-06-11
**Files analyzed:** 14 new/modified
**Analogs found:** 14 / 14

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/core/domain/project/project.go` | model | CRUD | `internal/core/domain/contract/contract.go` | exact (same role+flow) |
| `internal/core/ports/project_repository.go` | model | CRUD | `internal/core/ports/contract_repository.go` | exact (same role+flow) |
| `internal/core/services/project/project.go` | service | CRUD | `internal/core/services/contract/contract.go` | exact (same role+flow) |
| `internal/adapters/primary/http/project.go` | controller | request-response | `internal/adapters/primary/http/contract.go` | exact (same role+flow) |
| `internal/adapters/secondary/postgres/project_repository.go` | model | CRUD | `internal/adapters/secondary/postgres/contract_repository.go` | exact (same role+flow) |
| `cmd/server/main.go` | config | — | existing routes in same file | exact (same file) |
| `internal/core/services/project/project_test.go` | test | CRUD | `internal/core/services/contract/contract_test.go` | exact (same role+flow) |
| `internal/adapters/primary/http/project_test.go` | test | request-response | `internal/adapters/primary/http/project.go` patterns | role-match |
| `internal/core/services/testdata/mocks.go` | utility | CRUD | existing `MockContractRepo` in same file | exact (same file) |
| `web/src/api/projects.ts` | service | CRUD | `web/src/api/contracts.ts` | exact (same role+flow) |
| `web/src/types/api.ts` | model | — | existing `CreateProjectRequest` in same file | exact (same file) |
| `web/src/routes/_authenticated/projects/-components/project-detail.tsx` | component | request-response | existing same file + `web/src/api/contracts.ts` mutation pattern | exact (same file) |
| `web/src/routes/_authenticated/projects/-components/edit-project-dialog.tsx` | component (NEW) | request-response | `web/src/routes/_authenticated/projects/-components/create-project-dialog.tsx` | exact (same role+flow) |

## Pattern Assignments

### `internal/core/domain/project/project.go` (model, CRUD)

**Analog:** `internal/core/domain/contract/contract.go`

**Imports pattern** (lines 1-9):
```go
package contract

import (
    "errors"
    "time"

    "github.com/google/uuid"
    "github.com/stefanoprivitera/hourglass/internal/models"
)
```

**Sentinel error pattern** (lines 11-18):
```go
var (
    ErrProjectNotFound      = errors.New("project not found")
    ErrForbidden            = errors.New("forbidden")
    ErrInvalidRequest       = errors.New("invalid request")
    ErrAlreadyAdopted       = errors.New("already adopted")
    ErrUserNotFound         = errors.New("user not found")
)
```
→ Add: `ErrHasActiveTimeEntries`, `ErrHasActiveSubprojectEntries`

**Update request pattern** (lines 58-66):
```go
type UpdateContractRequest struct {
    Name            string              `json:"name"`
    KmRate          *float64            `json:"km_rate,omitempty"`
    Currency        string              `json:"currency"`
    GovernanceModel models.GovernanceModel `json:"governance_model"`
    IsShared        *bool               `json:"is_shared,omitempty"`
    IsActive        *bool               `json:"is_active,omitempty"`
    CustomerID      *string             `json:"customer_id,omitempty"`
}
```
→ Create `UpdateProjectRequest` with Name, Type, ContractID, GovernanceModel, IsShared fields

---

### `internal/core/ports/project_repository.go` (model, CRUD)

**Analog:** `internal/core/ports/contract_repository.go`

**Interface pattern** (lines 10-19):
```go
type ContractRepository interface {
    List(ctx context.Context, orgID uuid.UUID, scope string, isActive *bool) ([]contractdomain.ContractResponse, error)
    Create(ctx context.Context, orgID uuid.UUID, req *contractdomain.CreateContractRequest) (*contractdomain.ContractResponse, error)
    Get(ctx context.Context, orgID, contractID uuid.UUID) (*contractdomain.ContractResponse, error)
    Adopt(ctx context.Context, orgID, contractID uuid.UUID) (*contractdomain.ContractAdoption, error)
    Update(ctx context.Context, orgID, contractID uuid.UUID, req *contractdomain.UpdateContractRequest) (*contractdomain.ContractResponse, int, error)
    Delete(ctx context.Context, orgID, contractID uuid.UUID) error
    HasTimeEntries(ctx context.Context, contractID uuid.UUID) (int, error)
    HasProjects(ctx context.Context, contractID uuid.UUID) (int, error)
}
```
→ Add to `ProjectRepository`: `Update`, `Delete`, `HasActiveTimeEntries`

**HasActiveTimeEntries signature** — returns `(hasEntries bool, hasSubprojectEntries bool, err error)`:
```go
HasActiveTimeEntries(ctx context.Context, projectID uuid.UUID) (bool, bool, error)
```

---

### `internal/core/services/project/project.go` (service, CRUD)

**Analog:** `internal/core/services/contract/contract.go`

**Service struct pattern** (lines 12-18):
```go
type Service struct {
    repo ports.ContractRepository
}

func NewService(repo ports.ContractRepository) *Service {
    return &Service{repo: repo}
}
```

**Role-gated update pattern** (lines 42-47):
```go
func (s *Service) Update(ctx context.Context, role string, orgID, contractID uuid.UUID, req *contractdomain.UpdateContractRequest) (*contractdomain.ContractResponse, int, error) {
    if role != string(models.RoleFinance) {
        return nil, 0, contractdomain.ErrForbidden
    }
    return s.repo.Update(ctx, orgID, contractID, req)
}
```
→ For ProjectService.Update, omit the `int` return (no mileage recalculation):
```go
func (s *Service) Update(ctx context.Context, role string, orgID, projectID uuid.UUID, req *projectdomain.UpdateProjectRequest) (*projectdomain.ProjectResponse, error) {
    if role != string(models.RoleFinance) {
        return nil, projectdomain.ErrForbidden
    }
    return s.repo.Update(ctx, orgID, projectID, req)
}
```

**Delete protection pattern** (lines 59-87):
```go
func (s *Service) Delete(ctx context.Context, role string, orgID, contractID uuid.UUID) error {
    if role != string(models.RoleFinance) {
        return contractdomain.ErrForbidden
    }
    existing, err := s.repo.Get(ctx, orgID, contractID)
    if err != nil {
        return err
    }
    if existing.CreatedByOrgID != orgID {
        return contractdomain.ErrForbidden
    }
    count, err := s.repo.HasTimeEntries(ctx, contractID)
    if err != nil {
        return err
    }
    if count > 0 {
        return contractdomain.ErrHasTimeEntries
    }
    projectCount, err := s.repo.HasProjects(ctx, contractID)
    if err != nil {
        return err
    }
    if projectCount > 0 {
        return contractdomain.ErrHasActiveProjects
    }
    return s.repo.Delete(ctx, orgID, contractID)
}
```
→ For ProjectService.Delete (distinct errors per D-06):
```go
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

---

### `internal/adapters/primary/http/project.go` (controller, request-response)

**Analog:** `internal/adapters/primary/http/contract.go`

**Handler struct pattern** (lines 15-21):
```go
type ContractHandler struct {
    service *contractsvc.Service
}

func NewContractHandler(service *contractsvc.Service) *ContractHandler {
    return &ContractHandler{service: service}
}
```

**Update handler with error switching** (lines 131-167):
```go
func (h *ContractHandler) Update(w http.ResponseWriter, r *http.Request) {
    orgID := middleware.GetOrganizationID(r.Context())
    contractID, err := uuid.Parse(r.PathValue("id"))
    if err != nil {
        api.RespondWithError(w, http.StatusBadRequest, "invalid contract id")
        return
    }
    var req UpdateContractRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    updated, affectedMileageCount, err := h.service.Update(r.Context(), middleware.GetRole(r.Context()), orgID, contractID, &contractdomain.UpdateContractRequest{
        Name:            req.Name,
        // ... mapped fields
    })
    if err != nil {
        switch err {
        case contractdomain.ErrForbidden:
            api.RespondWithError(w, http.StatusForbidden, "only finance users can update contracts")
        case contractdomain.ErrContractNotFound:
            api.RespondWithError(w, http.StatusNotFound, "contract not found")
        default:
            api.RespondWithError(w, http.StatusInternalServerError, "failed to update contract")
        }
        return
    }
    api.RespondWithJSON(w, http.StatusOK, updated)
}
```

**Delete handler with 409 conflict** (lines 200-223):
```go
func (h *ContractHandler) Delete(w http.ResponseWriter, r *http.Request) {
    // ... parse ID ...
    err = h.service.Delete(r.Context(), middleware.GetRole(r.Context()), orgID, contractID)
    if err != nil {
        switch err {
        case contractdomain.ErrForbidden:
            api.RespondWithError(w, http.StatusForbidden, "only finance users can delete contracts")
        case contractdomain.ErrContractNotFound:
            api.RespondWithError(w, http.StatusNotFound, "contract not found")
        case contractdomain.ErrHasTimeEntries:
            api.RespondWithError(w, http.StatusConflict, "contract has time entries and cannot be deleted")
        case contractdomain.ErrHasActiveProjects:
            api.RespondWithError(w, http.StatusConflict, "contract has projects and cannot be deleted")
        default:
            api.RespondWithError(w, http.StatusInternalServerError, "failed to delete contract")
        }
        return
    }
    api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "contract deleted"})
}
```
→ Project Delete maps: `ErrHasActiveTimeEntries` → 409, `ErrHasActiveSubprojectEntries` → 409

**ListSubprojects handler** — new handler, pure read, uses response envelope:
```go
func (h *ProjectHandler) ListSubprojects(w http.ResponseWriter, r *http.Request) {
    projectID, err := uuid.Parse(r.PathValue("id"))
    if err != nil {
        api.RespondWithError(w, http.StatusBadRequest, "invalid project id")
        return
    }
    subprojects, err := h.service.ListSubprojects(r.Context(), projectID)
    if err != nil {
        api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch subprojects")
        return
    }
    api.RespondWithJSON(w, http.StatusOK, subprojects)
}
```

**Update/Delete HTTP DTO pattern** (lines 121-129):
```go
type UpdateContractRequest struct {
    Name            string                 `json:"name,omitempty"`
    KmRate          *float64               `json:"km_rate,omitempty"`
    Currency        string                 `json:"currency,omitempty"`
    GovernanceModel models.GovernanceModel `json:"governance_model,omitempty"`
    IsShared        *bool                  `json:"is_shared,omitempty"`
    IsActive        *bool                  `json:"is_active,omitempty"`
    CustomerID      *string                `json:"customer_id,omitempty"`
}
```
→ Create `UpdateProjectRequest` DTO in `http/project.go`

---

### `internal/adapters/secondary/postgres/project_repository.go` (model, CRUD)

**Analog:** `internal/adapters/secondary/postgres/contract_repository.go`

**Dynamic SET update pattern** (lines 129-207):
```go
func (r *ContractRepository) Update(ctx context.Context, orgID, contractID uuid.UUID, req *contractdomain.UpdateContractRequest) (*contractdomain.ContractResponse, int, error) {
    var sets []string
    var args []interface{}
    argIdx := 1

    if req.Name != "" {
        sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
        args = append(args, req.Name)
        argIdx++
    }
    if req.GovernanceModel != "" {
        sets = append(sets, fmt.Sprintf("governance_model = $%d", argIdx))
        args = append(args, req.GovernanceModel)
        argIdx++
    }
    // ... more fields ...

    allSets := append(sets, "updated_at = NOW()")
    whereIdx := argIdx
    args = append(args, contractID, orgID)

    query := fmt.Sprintf(`UPDATE contracts SET %s WHERE id = $%d AND created_by_org_id = $%d`,
        strings.Join(allSets, ", "), whereIdx, whereIdx+1)

    cmd, err := r.pool.Exec(ctx, query, args...)
    if err != nil {
        return nil, 0, wrapPGError(err, "update contract")
    }
    if cmd.RowsAffected() == 0 {
        return nil, 0, contractdomain.ErrContractNotFound
    }
    return r.Get(ctx, orgID, contractID)
}
```

**Delete with adoption cleanup** (RESEARCH.md lines 276-301):
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

**HasActiveTimeEntries query** (RESEARCH.md lines 259-272):
```go
func (r *ProjectRepository) HasActiveTimeEntries(ctx context.Context, projectID uuid.UUID) (bool, bool, error) {
    query := `SELECT
        (SELECT COUNT(*) FROM time_entries
         WHERE project_id = $1
           AND status NOT IN ('approved', 'rejected')
           AND is_deleted = false) > 0 AS has_entries,
        (SELECT COUNT(*) FROM time_entries te
         WHERE te.subproject_id IN (SELECT id FROM subprojects WHERE project_id = $1)
           AND te.status NOT IN ('approved', 'rejected')
           AND te.is_deleted = false) > 0 AS has_subproject_entries`
    var hasEntries, hasSubprojectEntries bool
    err := r.pool.QueryRow(ctx, query, projectID).Scan(&hasEntries, &hasSubprojectEntries)
    if err != nil {
        return false, false, fmt.Errorf("has active time entries: %w", err)
    }
    return hasEntries, hasSubprojectEntries, nil
}
```

---

### `cmd/server/main.go` (config)

**Analog:** Existing route registration in same file (lines 178-184)

**Route registration pattern:**
```go
mux.HandleFunc("PUT /projects/{id}", middleware.Auth(authService, projectHandler.Update))
mux.HandleFunc("DELETE /projects/{id}", middleware.Auth(authService, projectHandler.Delete))
mux.HandleFunc("GET /projects/{id}/subprojects", middleware.Auth(authService, projectHandler.ListSubprojects))
```

**Repository + handler injection pattern** (lines 120-122):
```go
projectRepo := postgres.NewProjectRepository(pool)
projectService := projectsvc.NewService(projectRepo)
projectHandler := http.NewProjectHandler(projectService)
```
→ Add: `subprojectRepo := postgres.NewSubprojectRepository(pool)` and pass to `NewProjectHandler`

**Handler constructor update** — `ProjectHandler` needs `SubprojectRepository` or add `ListSubprojects` to `ProjectRepository` interface. Option A (pass SubprojectRepository to handler) is simpler:
```go
projectHandler := http.NewProjectHandler(projectService, subprojectRepo)
```

---

### `internal/core/services/project/project_test.go` (test, CRUD)

**Analog:** `internal/core/services/contract/contract_test.go`

**Setup pattern** (lines 16-21):
```go
func setupService(t *testing.T) (*Service, *testdata.MockContractRepo) {
    t.Helper()
    repo := &testdata.MockContractRepo{}
    svc := NewService(repo)
    return svc, repo
}
```

**Seed pattern** (lines 23-44):
```go
func seedContract(repo *testdata.MockContractRepo, overrides ...func(*contractdomain.ContractResponse)) *contractdomain.ContractResponse {
    c := &contractdomain.ContractResponse{
        Contract: contractdomain.Contract{
            ID:              uuid.New(),
            Name:            "Test Contract",
            // ...
        },
    }
    for _, o := range overrides { o(c) }
    if repo.Contracts == nil {
        repo.Contracts = make(map[uuid.UUID]*contractdomain.ContractResponse)
    }
    repo.Contracts[c.ID] = c
    return c
}
```

**TestService_Update test pattern** (lines 142-179):
```go
func TestService_Update(t *testing.T) {
    t.Parallel()
    tests := []struct {
        name    string
        role    string
        wantErr error
    }{
        {name: "finance role updates",  role: string(models.RoleFinance), wantErr: nil},
        {name: "non-finance role forbidden", role: string(models.RoleEmployee), wantErr: contractdomain.ErrForbidden},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc, repo := setupService(t)
            orgID := uuid.New()
            seeded := seedContract(repo, func(c *contractdomain.ContractResponse) { c.CreatedByOrgID = orgID })
            result, _, err := svc.Update(context.Background(), tt.role, orgID, seeded.ID, &contractdomain.UpdateContractRequest{Name: "Updated"})
            if tt.wantErr != nil {
                assert.ErrorIs(t, err, tt.wantErr)
                assert.Nil(t, result)
                return
            }
            assert.NoError(t, err)
            assert.NotNil(t, result)
        })
    }
}
```

**TestService_Delete test pattern** (lines 181-215):
```go
func TestService_Delete(t *testing.T) {
    t.Parallel()
    t.Run("finance role deletes", func(t *testing.T) { ... })
    t.Run("not found", func(t *testing.T) { ... })
    t.Run("unauthorized role", func(t *testing.T) { ... })
    t.Run("blocked by projects", func(t *testing.T) {
        svc, repo := setupService(t)
        orgID := uuid.New()
        seeded := seedContract(repo, func(c *contractdomain.ContractResponse) { c.CreatedByOrgID = orgID })
        repo.HasProjectsFn = func(ctx context.Context, contractID uuid.UUID) (int, error) {
            return 1, nil
        }
        err := svc.Delete(context.Background(), string(models.RoleFinance), orgID, seeded.ID)
        assert.ErrorIs(t, err, contractdomain.ErrHasActiveProjects)
    })
}
```

---

### `internal/adapters/primary/http/project_test.go` (test, request-response)

**Analog:** Follow the pattern established in existing handler code. No existing handler test file found that directly matches — use the existing handler's error-switch pattern to test each error case.

**Test strategy** (from CONTEXT.md / RESEARCH.md):
- `TestProjectHandler_Update` — test successful update, forbidden, not found
- `TestProjectHandler_Delete` — test successful delete, forbidden, not found, 409 for HasActiveTimeEntries, 409 for HasActiveSubprojectEntries
- `TestProjectHandler_ListSubprojects` — test returns subprojects, empty subprojects

**Use `httptest.NewRecorder()` pattern** (standard Go):
```go
func TestProjectHandler_Update(t *testing.T) {
    req := httptest.NewRequest("PUT", "/projects/"+id.String(), body)
    req = req.WithContext(middleware.WithOrgID(context.Background(), orgID))
    w := httptest.NewRecorder()
    handler.Update(w, req)
    assert.Equal(t, http.StatusOK, w.Code)
}
```

---

### `internal/core/services/testdata/mocks.go` (utility)

**Analog:** Existing `MockContractRepo` in same file (lines 336-391)

**Pattern for MockProjectRepo additions** — add these methods after `RemoveManager`:
```go
func (m *MockProjectRepo) Update(ctx context.Context, orgID, projectID uuid.UUID, req *projectdomain.UpdateProjectRequest) (*projectdomain.ProjectResponse, error) {
    return &projectdomain.ProjectResponse{}, nil
}

func (m *MockProjectRepo) Delete(ctx context.Context, orgID, projectID uuid.UUID) error {
    return nil
}

func (m *MockProjectRepo) HasActiveTimeEntries(ctx context.Context, projectID uuid.UUID) (bool, bool, error) {
    return false, false, nil
}
```

**Follow the HasProjectsFn pattern for configurable mock behavior** (lines 339-340):
```go
type MockContractRepo struct {
    mu            sync.Mutex
    Contracts     map[uuid.UUID]*contractdomain.ContractResponse
    HasProjectsFn func(ctx context.Context, contractID uuid.UUID) (int, error)
}
```
→ Add `HasActiveTimeEntriesFn` to `MockProjectRepo` for testing delete-blocked scenarios.

---

### `web/src/api/projects.ts` (service, CRUD)

**Analog:** `web/src/api/contracts.ts`

**Import pattern** (lines 1-5):
```typescript
import {mutationOptions, queryOptions} from '@tanstack/react-query'
import {toast} from 'sonner'
import {api} from '@/lib/api.ts'
import type {Project} from '@/types/models'
import type {CreateProjectRequest} from "@/types";
```

**Query key and options pattern** (lines 7-35):
```typescript
function projectsQueryKey(scope: 'owned' | 'adopted' | 'all', contractId?: string) {
  return ['projects', scope, contractId] as const
}

function projectQueryKey(id: string) {
  return ['projects', id] as const
}

function projectsQueryOpts(scope: 'owned' | 'adopted' | 'all' = 'owned', contractId?: string) {
  // ... URL construction + queryOptions with queryKey, queryFn, staleTime
}
```

**Update mutation pattern** (lines 65-75 from contracts.ts):
```typescript
const updateProjectMutationOpts = mutationOptions({
  mutationFn: ({id, data}: { id: string; data: UpdateProjectRequest }) =>
    api<Project>(`/projects/${id}`, {method: 'PUT', body: JSON.stringify(data)}),
  onSuccess: (_, {id}, {client}) => {
    client.invalidateQueries({queryKey: ['projects']})
    client.invalidateQueries({queryKey: ['projects', id]})
    toast.success('Project updated')
  },
})
```

**Delete mutation pattern** (lines 77-84 from contracts.ts):
```typescript
const deleteProjectMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<void>(`/projects/${id}`, {method: 'DELETE'}),
  onSuccess: (_, __, {client}) => {
    client.invalidateQueries({queryKey: ['projects']})
    toast.success('Project deleted')
  },
  onError: (error: Error) => {
    toast.error(error.message || 'Failed to delete project')
  },
})
```

**Subprojects query options** (RESEARCH.md lines 328-333):
```typescript
const subprojectsQueryOpts = (id: string) => queryOptions({
  queryKey: ['projects', id, 'subprojects'],
  queryFn: () => api<Subproject[]>(`/projects/${id}/subprojects`),
  enabled: !!id,
})
```

**Export pattern** (lines 94-102 from contracts.ts):
```typescript
export const ProjectsApis = {
  projectsQueryOpts,
  projectQueryOpts,
  createProjectMutationOpts,
  adoptProjectMutationOpts,
  updateProjectMutationOpts,  // ADD
  deleteProjectMutationOpts,  // ADD
  subprojectsQueryOpts,       // ADD
}
```

---

### `web/src/types/api.ts` (model)

**Analog:** Existing `CreateProjectRequest` in same file (lines 91-97)

**Type pattern:**
```typescript
export interface UpdateProjectRequest {
  name: string
  type: 'billable' | 'internal'
  contract_id: string
  governance_model: 'creator_controlled' | 'unanimous' | 'majority'
  is_shared: boolean
}
```

---

### `web/src/routes/_authenticated/projects/-components/project-detail.tsx` (component, request-response)

**Analog:** Existing same file + `web/src/api/contracts.ts` mutation pattern

**Current disabled buttons** (lines 63-82) — replace with working buttons:

**Edit button pattern** — opens EditProjectDialog:
```tsx
const [editOpen, setEditOpen] = useState(false)

// In JSX, replace disabled Edit button:
<Button variant="outline" onClick={() => setEditOpen(true)}>
  Edit
</Button>

// Add dialog at bottom:
<EditProjectDialog
  open={editOpen}
  onOpenChange={setEditOpen}
  project={p}
  onSuccess={() => { /* invalidate + show updated data */ }}
/>
```

**Delete button pattern** — confirmation dialog calling delete mutation:
```tsx
const deleteProject = useMutation(ProjectsApis.deleteProjectMutationOpts)
const [deleteOpen, setDeleteOpen] = useState(false)
const navigate = useNavigate()

// Delete confirmation dialog:
<Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
  <DialogContent>
    <DialogHeader>
      <DialogTitle>Delete Project</DialogTitle>
      <DialogDescription>
        Are you sure you want to delete "{p.name}"? This action cannot be undone.
      </DialogDescription>
    </DialogHeader>
    <DialogFooter>
      <Button variant="outline" onClick={() => setDeleteOpen(false)}>Cancel</Button>
      <Button variant="destructive" onClick={() => {
        deleteProject.mutate(p.id, {
          onSuccess: () => { navigate({to: '/projects'}); setDeleteOpen(false) },
          onError: (err) => { toast.error(err.message); setDeleteOpen(false) },
        })
      }}>
        Delete
      </Button>
    </DialogFooter>
  </DialogContent>
</Dialog>
```

**Subproject expandable section** — simple collapsible below the Details card:
```tsx
// Fetch subprojects
const {data: subprojects} = useQuery(ProjectsApis.subprojectsQueryOpts(id))

// Expandable section
<Accordion type="single" collapsible>
  <AccordionItem value="subprojects">
    <AccordionTrigger>
      Subprojects ({subprojects?.data?.length ?? 0})
    </AccordionTrigger>
    <AccordionContent>
      {subprojects?.data?.map((sp) => (
        <div key={sp.id} className="flex items-center justify-between py-2">
          <div>
            <span className="font-medium">{sp.name}</span>
            {sp.description && <p className="text-sm text-muted-foreground">{sp.description}</p>}
          </div>
          <Badge variant={sp.is_active ? 'default' : 'secondary'}>
            {sp.is_active ? 'Active' : 'Inactive'}
          </Badge>
        </div>
      ))}
    </AccordionContent>
  </AccordionItem>
</Accordion>
```

---

### `web/src/routes/_authenticated/projects/-components/edit-project-dialog.tsx` (component, NEW, request-response)

**Analog:** `web/src/routes/_authenticated/projects/-components/create-project-dialog.tsx`

**Copy the full structure** of `create-project-dialog.tsx` (lines 1-186) and adapt:

**Props difference** — add `project: Project` for pre-population:
```tsx
interface EditProjectDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: (project: Project) => void
  project: Project  // pre-populated data
}
```

**State pre-population from project prop**:
```tsx
const [name, setName] = useState(project.name)
const [type, setType] = useState<'billable' | 'internal'>(project.type)
const [contractId, setContractId] = useState(project.contract_id)
const [governanceModel, setGovernanceModel] = useState(project.governance_model)
const [isShared, setIsShared] = useState(project.is_shared)
```

**Submit handler** — calls update mutation instead of create:
```tsx
const updateProject = useMutation(ProjectsApis.updateProjectMutationOpts)

const handleSubmit = () => {
  if (!name.trim() || !contractId) return
  updateProject.mutate(
    { id: project.id, data: { name: name.trim(), type, contract_id: contractId, governance_model: governanceModel, is_shared: isShared } },
    {
      onSuccess: () => {
        onOpenChange(false)
        if (onSuccess) onSuccess(project)
      },
    }
  )
}
```

**Title and button changes**:
- DialogTitle: "Edit Project" (not "Create Project")
- Submit button text: "Save Changes" (not "Create")
- No `resetForm()` needed (values come from project prop)

## Shared Patterns

### Authentication
**Source:** `internal/middleware/auth.go` / `cmd/server/main.go`
**Apply to:** All new handler routes in `cmd/server/main.go`

All new project routes use the existing auth middleware:
```go
mux.HandleFunc("PUT /projects/{id}", middleware.Auth(authService, projectHandler.Update))
mux.HandleFunc("DELETE /projects/{id}", middleware.Auth(authService, projectHandler.Delete))
mux.HandleFunc("GET /projects/{id}/subprojects", middleware.Auth(authService, projectHandler.ListSubprojects))
```

### API Response Envelope
**Source:** `pkg/api/response.go`
**Apply to:** All new handler methods

All responses use the shared envelope:
```go
api.RespondWithJSON(w, http.StatusOK, payload)     // → {"data": payload}
api.RespondWithError(w, status, message)             // → {"error": message}
```

### Error Handling (Backend)
**Source:** `internal/adapters/primary/http/contract.go` lines 152-167, 209-222
**Apply to:** `http/project.go` — Update and Delete handlers

Use `switch err` pattern with domain sentinel errors:
- `ErrForbidden` → 403
- `ErrProjectNotFound` → 404
- `ErrHasActiveTimeEntries` → 409 with distinct message
- `ErrHasActiveSubprojectEntries` → 409 with distinct message
- Default → 500

### Role Gating
**Source:** `internal/core/services/contract/contract.go` lines 42-44, 59-61
**Apply to:** `project.go` service — Update and Delete methods

Finance role check pattern:
```go
if role != string(models.RoleFinance) {
    return ..., projectdomain.ErrForbidden
}
```

### Ownership Check on Delete
**Source:** `internal/core/services/contract/contract.go` lines 67-69
**Apply to:** `project.go` service — Delete method

```go
existing, err := s.repo.Get(ctx, orgID, projectID)
if err != nil { return err }
if existing.CreatedByOrgID != orgID {
    return projectdomain.ErrForbidden
}
```

### Error Handling (Frontend)
**Source:** `web/src/lib/api.ts` lines 51-55
**Apply to:** All frontend mutation callers

API errors are thrown as `Error` objects with the `message` field from the response:
```typescript
const error = await res.json().catch(() => ({message: 'Request failed'})) as ApiError
throw new Error(error.message || error.error || 'Request failed')
```

### Mutation Error Pattern
**Apply to:** Delete confirmation in `project-detail.tsx`

Use `onError` on the mutation call or `mutationOptions`:
```typescript
onError: (error: Error) => {
  toast.error(error.message || 'Failed to delete project')
}
```

## No Analog Found

All 14 files have direct analogs in the existing codebase. No files require RESEARCH.md patterns as primary source.

## Metadata

**Analog search scope:** `internal/` (Go backend), `web/src/` (React frontend)
**Files scanned:** 20+ (domain, ports, services, handlers, repositories, tests, mocks, frontend api, components, types)
**Pattern extraction date:** 2026-06-11
