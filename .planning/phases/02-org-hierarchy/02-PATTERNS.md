# Phase 2: Org Hierarchy - Pattern Map

**Mapped:** 2026-06-10
**Files analyzed:** 12 (6 modify, 0 create, 0 delete)
**Analogs found:** 12 / 12

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/core/domain/unit/unit.go` | model | N/A | Same file (self) | exact |
| `internal/core/ports/unit_repository.go` | port | CRUD | Same file (self) | exact |
| `internal/core/services/unit/unit.go` | service | CRUD | Same file (self) | exact |
| `internal/adapters/secondary/postgres/unit_repository.go` | repository | CRUD | Same file (self) — `HasMembers` | exact |
| `internal/adapters/secondary/postgres/unit_member_repository.go` | repository | CRUD | Same file (self) — `ListByUnit`, `Add` | exact |
| `internal/adapters/primary/http/unit.go` | controller | request-response | Same file (self) — `AddMember`, `RemoveMember` | exact |
| `cmd/server/main.go` | config | N/A | Same file (self) — existing unit routes | exact |
| `web/src/api/units.ts` | api | CRUD | Same file (self) — existing mutation patterns | exact |
| `web/src/routes/_authenticated/org-hierarchy/-context/org-hierarchy-context.tsx` | store | state management | Same file (self) | exact |
| `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/unit-detail-panel.tsx` | component | request-response | Same file (self) — `MemberRow`, `Sheet` pattern | exact |
| `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/reparent-confirm-dialog.tsx` | component | request-response | Same file (self) | exact |
| `web/src/routes/_authenticated/org-hierarchy/-components/org-hierarchy-page.tsx` | component/page | request-response | Same file (self) | exact |

## Pattern Assignments

---

### `internal/core/domain/unit/unit.go` (model, N/A)

**Modification:** Add two new sentinel errors for delete protection.

**Analog:** Same file, lines 10-16

**Existing sentinel errors pattern** (lines 10-16):
```go
var (
	ErrUnitNotFound            = errors.New("unit not found")
	ErrInvalidParentUnit       = errors.New("invalid parent unit")
	ErrCircularParent          = errors.New("cannot make unit a descendant of itself")
	ErrCannotDeleteWithMembers = errors.New("cannot delete unit with members")
	ErrMemberNotFound          = errors.New("unit member not found")
)
```

**New errors to add after line 15:**
```go
	ErrCannotDeleteRootUnit    = errors.New("cannot delete root unit")
	ErrCannotDeleteWithChildren = errors.New("cannot delete unit with child units")
```

---

### `internal/core/ports/unit_repository.go` (port, CRUD)

**Modification:** Add `UpdateMember`, `HasChildren`, `ListMembersByUnitIDs`, `ListMembershipsForUser` methods.

**Analog:** Same file (self), lines 10-22.

**Existing interface pattern** (lines 10-22):
```go
type UnitRepository interface {
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]unit.Unit, error)
	GetByID(ctx context.Context, id string) (*unit.Unit, error)
	Create(ctx context.Context, u *unit.Unit) (*unit.Unit, error)
	Update(ctx context.Context, u *unit.Unit) (*unit.Unit, error)
	Delete(ctx context.Context, id string) error
	GetDescendants(ctx context.Context, id string) ([]unit.Unit, error)
	HasMembers(ctx context.Context, id string) (bool, error)
	ListMembers(ctx context.Context, unitID string) ([]unit.UnitMember, error)
	AddMember(ctx context.Context, m *unit.UnitMember) (*unit.UnitMember, error)
	RemoveMember(ctx context.Context, id string) error
	GetMemberCountsByOrg(ctx context.Context, orgID uuid.UUID) (map[string]int, error)
}
```

**New methods to add after line 21:**
```go
	// UpdateMember updates a unit membership (is_primary, end_date).
	UpdateMember(ctx context.Context, m *unit.UnitMember) (*unit.UnitMember, error)

	// HasChildren returns true if the unit has at least one child unit.
	HasChildren(ctx context.Context, id string) (bool, error)

	// ListMembersByUnitIDs returns members for multiple unit IDs at once.
	ListMembersByUnitIDs(ctx context.Context, orgID uuid.UUID, unitIDs []string) ([]unit.UnitMember, error)

	// ListMembershipsForUser returns all unit memberships for a user across all units.
	ListMembershipsForUser(ctx context.Context, userID uuid.UUID) ([]unit.UnitMember, error)
```

---

### `internal/core/services/unit/unit.go` (service, CRUD)

**Modification:** Add `UpdateMember` method with "one primary per user" enforcement + update `Delete` for root/children checks.

**Analog:** Same file (self).

**Existing `Delete` method** (lines 124-133) — currently only checks members:
```go
func (s *Service) Delete(ctx context.Context, id string) error {
	hasMembers, err := s.repo.HasMembers(ctx, id)
	if err != nil {
		return err
	}
	if hasMembers {
		return unit.ErrCannotDeleteWithMembers
	}
	return s.repo.Delete(ctx, id)
}
```

**Existing `AddMember` method** (lines 169-181) — pattern for member mutations:
```go
func (s *Service) AddMember(ctx context.Context, unitID string, orgID uuid.UUID, req *unit.AddUnitMemberRequest) (*unit.UnitMember, error) {
	m := &unit.UnitMember{
		ID:        uuid.New().String(),
		OrgID:     orgID,
		UserID:    req.UserID,
		UnitID:    unitID,
		Role:      req.Role,
		IsPrimary: req.IsPrimary,
		StartDate: time.Now(),
		CreatedAt: time.Now(),
	}
	return s.repo.AddMember(ctx, m)
}
```

**Existing `RemoveMember` method** (lines 183-185) — simple delegation:
```go
func (s *Service) RemoveMember(ctx context.Context, id string) error {
	return s.repo.RemoveMember(ctx, id)
}
```

**New `UpdateMember` service method pattern** (D-03, D-02):
```go
func (s *Service) UpdateMember(ctx context.Context, unitID, membershipID string, isPrimary bool, endDate *time.Time) (*unit.UnitMember, error) {
	members, err := s.repo.ListMembers(ctx, unitID)
	if err != nil {
		return nil, err
	}

	var targetMember *unit.UnitMember
	for _, m := range members {
		if m.ID == membershipID {
			targetMember = &m
			break
		}
	}
	if targetMember == nil {
		return nil, unit.ErrMemberNotFound
	}

	// One primary per user (D-03)
	if isPrimary {
		allMemberships, err := s.repo.ListMembershipsForUser(ctx, targetMember.UserID)
		if err != nil {
			return nil, err
		}
		for _, m := range allMemberships {
			if m.IsPrimary && m.ID != membershipID {
				m.IsPrimary = false
				if _, err := s.repo.UpdateMember(ctx, &m); err != nil {
					return nil, err
				}
			}
		}
	}

	targetMember.IsPrimary = isPrimary
	targetMember.EndDate = endDate
	return s.repo.UpdateMember(ctx, targetMember)
}
```

**Updated `Delete` method** (D-05, D-06):
```go
func (s *Service) Delete(ctx context.Context, id string) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	// Root unit check (D-05)
	if u.HierarchyLevel == 0 {
		return unit.ErrCannotDeleteRootUnit
	}
	// Children check (D-05)
	hasChildren, err := s.repo.HasChildren(ctx, id)
	if err != nil {
		return err
	}
	if hasChildren {
		return unit.ErrCannotDeleteWithChildren
	}
	// Members check (already exists)
	hasMembers, err := s.repo.HasMembers(ctx, id)
	if err != nil {
		return err
	}
	if hasMembers {
		return unit.ErrCannotDeleteWithMembers
	}
	return s.repo.Delete(ctx, id)
}
```

---

### `internal/adapters/secondary/postgres/unit_repository.go` (repository, CRUD)

**Modification:** Add `HasChildren` implementation.

**Analog:** Same file — `HasMembers` method (lines 179-191).

**Existing `HasMembers` method** (lines 179-191) — `EXISTS` query pattern:
```go
func (r *UnitRepository) HasMembers(ctx context.Context, id string) (bool, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return false, fmt.Errorf("parse unit id: %w", err)
	}
	query := `SELECT EXISTS(SELECT 1 FROM unit_memberships WHERE unit_id = $1)`
	var exists bool
	err = r.pool.QueryRow(ctx, query, uid).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has members: %w", err)
	}
	return exists, nil
}
```

**New `HasChildren` method** (follows same `EXISTS` pattern):
```go
func (r *UnitRepository) HasChildren(ctx context.Context, id string) (bool, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return false, fmt.Errorf("parse unit id: %w", err)
	}
	query := `SELECT EXISTS(SELECT 1 FROM units WHERE parent_unit_id = $1)`
	var exists bool
	err = r.pool.QueryRow(ctx, query, uid).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has children: %w", err)
	}
	return exists, nil
}
```

**Delegation pattern** — new methods route through `UnitRepository` to `UnitMemberRepository` (follow lines 193-206):
```go
// UpdateMember delegates to UnitMemberRepository.
func (r *UnitRepository) UpdateMember(ctx context.Context, m *unit.UnitMember) (*unit.UnitMember, error) {
	return r.members.Update(ctx, m)
}

// ListMembersByUnitIDs delegates to UnitMemberRepository.
func (r *UnitRepository) ListMembersByUnitIDs(ctx context.Context, orgID uuid.UUID, unitIDs []string) ([]unit.UnitMember, error) {
	return r.members.ListByUnitIDs(ctx, orgID, unitIDs)
}
```

---

### `internal/adapters/secondary/postgres/unit_member_repository.go` (repository, CRUD)

**Modification:** Add `Update` and `ListByUnitIDs` implementations.

**Analog:** Same file.

**Existing `Add` method** (lines 58-86) — `RETURNING` scan with UUID parsing:
```go
func (r *UnitMemberRepository) Add(ctx context.Context, m *unit.UnitMember) (*unit.UnitMember, error) {
	id := uuid.New()
	m.ID = id.String()

	unitID, err := uuid.Parse(m.UnitID)
	if err != nil {
		return nil, fmt.Errorf("parse unit id: %w", err)
	}

	query := `INSERT INTO unit_memberships (id, org_id, user_id, unit_id, is_primary, role, start_date, end_date, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING id, org_id, user_id, unit_id, is_primary, role, start_date, end_date, created_at`

	var sqlID uuid.UUID
	var sqlUnitID uuid.UUID
	var um unit.UnitMember
	err = r.pool.QueryRow(ctx, query,
		id, m.OrgID, m.UserID, unitID, m.IsPrimary, m.Role, m.StartDate, m.EndDate,
	).Scan(
		&sqlID, &um.OrgID, &um.UserID, &sqlUnitID, &um.IsPrimary, &um.Role,
		&um.StartDate, &um.EndDate, &um.CreatedAt,
	)
	if err != nil {
		return nil, wrapPGError(err, "add unit member")
	}
	um.ID = sqlID.String()
	um.UnitID = sqlUnitID.String()
	return &um, nil
}
```

**New `Update` method** (follows same `RETURNING` + scan pattern):
```go
func (r *UnitMemberRepository) Update(ctx context.Context, m *unit.UnitMember) (*unit.UnitMember, error) {
	id, err := uuid.Parse(m.ID)
	if err != nil {
		return nil, fmt.Errorf("parse member id: %w", err)
	}

	query := `UPDATE unit_memberships
		SET is_primary = $2, end_date = $3
		WHERE id = $1
		RETURNING id, org_id, user_id, unit_id, is_primary, role, start_date, end_date, created_at`

	var sqlID uuid.UUID
	var sqlUnitID uuid.UUID
	var um unit.UnitMember
	err = r.pool.QueryRow(ctx, query, id, m.IsPrimary, m.EndDate).Scan(
		&sqlID, &um.OrgID, &um.UserID, &sqlUnitID, &um.IsPrimary, &um.Role,
		&um.StartDate, &um.EndDate, &um.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, unit.ErrMemberNotFound
		}
		return nil, wrapPGError(err, "update unit member")
	}
	um.ID = sqlID.String()
	um.UnitID = sqlUnitID.String()
	return &um, nil
}
```

**New `ListByUnitIDs` method** (follows `ListByUnit` rows-iteration pattern from lines 24-55):
```go
func (r *UnitMemberRepository) ListByUnitIDs(ctx context.Context, orgID uuid.UUID, unitIDs []string) ([]unit.UnitMember, error) {
	ids := make([]uuid.UUID, 0, len(unitIDs))
	for _, id := range unitIDs {
		uid, err := uuid.Parse(strings.TrimSpace(id))
		if err != nil {
			continue
		}
		ids = append(ids, uid)
	}
	if len(ids) == 0 {
		return []unit.UnitMember{}, nil
	}

	query := `SELECT um.id, um.org_id, um.user_id, um.unit_id, um.is_primary, um.role,
		um.start_date, um.end_date, um.created_at,
		COALESCE(u.firstname || ' ' || u.lastname, '') AS user_name,
		COALESCE(u.email, '') AS user_email
		FROM unit_memberships um
		LEFT JOIN users u ON um.user_id = u.id
		WHERE um.unit_id = ANY($1) AND um.org_id = $2
		ORDER BY um.unit_id, um.created_at DESC`

	rows, err := r.pool.Query(ctx, query, ids, orgID)
	if err != nil {
		return nil, fmt.Errorf("list members by unit ids: %w", err)
	}
	defer rows.Close()

	var members []unit.UnitMember
	for rows.Next() {
		m, err := scanUnitMember(rows)
		if err != nil {
			return nil, fmt.Errorf("scan unit member: %w", err)
		}
		members = append(members, *m)
	}
	if members == nil {
		members = []unit.UnitMember{}
	}
	return members, rows.Err()
}
```

---

### `internal/adapters/primary/http/unit.go` (controller, request-response)

**Modification:** Add `UpdateMember` and `ListMembersBatch` handlers.

**Analog:** Same file.

**Existing `AddMember` handler** (lines 232-271) — pattern for member mutation with request body decoding:
```go
func (h *UnitHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitID := r.PathValue("id")

	if unitID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "invalid unit id")
		return
	}

	orgID := middleware.GetOrganizationID(ctx)

	var req AddUnitMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	m, err := h.service.AddMember(ctx, unitID, orgID, &unit.AddUnitMemberRequest{
		UserID:    userID,
		Role:      req.Role,
		IsPrimary: req.IsPrimary,
	})
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to add member")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, m)
}
```

**Existing `RemoveMember` handler** (lines 273-289) — pattern for member operations with `PathValue`:
```go
func (h *UnitHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	membershipID := r.PathValue("membership_id")

	if membershipID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "invalid membership id")
		return
	}

	err := h.service.RemoveMember(ctx, membershipID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to remove member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

**Existing sentinel error mapping in `Delete`** (lines 146-170) — pattern for domain error → HTTP status:
```go
func (h *UnitHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitID := r.PathValue("id")

	if unitID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "invalid unit id")
		return
	}

	err := h.service.Delete(ctx, unitID)
	if err != nil {
		if err == unit.ErrCannotDeleteWithMembers {
			api.RespondWithError(w, http.StatusBadRequest, "cannot delete unit with members")
			return
		}
		if err == unit.ErrUnitNotFound {
			api.RespondWithError(w, http.StatusNotFound, "unit not found")
			return
		}
		if err == unit.ErrCannotDeleteRootUnit {
			api.RespondWithError(w, http.StatusBadRequest, "cannot delete root unit")
			return
		}
		if err == unit.ErrCannotDeleteWithChildren {
			api.RespondWithError(w, http.StatusBadRequest, "cannot delete unit with child units")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete unit")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

**New request type** (follow `AddUnitMemberRequest` pattern at lines 203-207):
```go
type UpdateUnitMemberRequest struct {
	IsPrimary bool       `json:"is_primary"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}
```

**New `UpdateMember` handler** (D-02, follows `AddMember` + `RemoveMember` patterns):
```go
func (h *UnitHandler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitID := r.PathValue("id")
	membershipID := r.PathValue("membership_id")

	if unitID == "" || membershipID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "invalid unit id or membership id")
		return
	}

	var req UpdateUnitMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	m, err := h.service.UpdateMember(ctx, unitID, membershipID, req.IsPrimary, req.EndDate)
	if err != nil {
		if err == unit.ErrMemberNotFound {
			api.RespondWithError(w, http.StatusNotFound, "member not found")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update member")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, m)
}
```

**New `ListMembersBatch` handler** (D-13, follows `ListMembers` pattern at lines 209-230):
```go
func (h *UnitHandler) ListMembersBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitIDsParam := r.URL.Query().Get("unit_ids")
	if unitIDsParam == "" {
		api.RespondWithError(w, http.StatusBadRequest, "unit_ids query parameter is required")
		return
	}

	unitIDs := strings.Split(unitIDsParam, ",")
	orgID := middleware.GetOrganizationID(ctx)

	members, err := h.service.ListMembersByUnitIDs(ctx, orgID, unitIDs)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch members")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, members)
}
```

**Updated `Delete` handler error mapping** (add two new sentinel checks):
```go
	// Add after existing ErrCannotDeleteWithMembers check:
	if err == unit.ErrCannotDeleteRootUnit {
		api.RespondWithError(w, http.StatusBadRequest, "cannot delete root unit")
		return
	}
	if err == unit.ErrCannotDeleteWithChildren {
		api.RespondWithError(w, http.StatusBadRequest, "cannot delete unit with child units")
		return
	}
```

---

### `cmd/server/main.go` (config, N/A)

**Modification:** Add three new route registrations.

**Analog:** Existing unit routes (lines 140-149) and member routes.

**Existing unit route registration pattern** (lines 140-149):
```go
	mux.HandleFunc("GET /units", middleware.Auth(authService, unitHandler.List))
	mux.HandleFunc("POST /units", middleware.Auth(authService, unitHandler.Create))
	mux.HandleFunc("GET /units/{id}", middleware.Auth(authService, unitHandler.Get))
	mux.HandleFunc("PUT /units/{id}", middleware.Auth(authService, unitHandler.Update))
	mux.HandleFunc("DELETE /units/{id}", middleware.Auth(authService, unitHandler.Delete))
	mux.HandleFunc("GET /units/tree", middleware.Auth(authService, unitHandler.GetTree))
	mux.HandleFunc("GET /units/{id}/descendants", middleware.Auth(authService, unitHandler.GetDescendants))
	mux.HandleFunc("GET /units/{id}/members", middleware.Auth(authService, unitHandler.ListMembers))
	mux.HandleFunc("POST /units/{id}/members", middleware.Auth(authService, unitHandler.AddMember))
	mux.HandleFunc("DELETE /units/{id}/members/{membership_id}", middleware.Auth(authService, unitHandler.RemoveMember))
```

**New routes to add after line 149:**
```go
	mux.HandleFunc("PUT /units/{id}/members/{membership_id}", middleware.Auth(authService, unitHandler.UpdateMember))
	mux.HandleFunc("GET /units/members/batch", middleware.Auth(authService, unitHandler.ListMembersBatch))
```

---

### `web/src/api/units.ts` (api, CRUD)

**Modification:** Add `updateUnitMemberMutationOpts`.

**Analog:** Same file.

**Existing mutation pattern** (lines 100-119) — `mutationOptions` with `onSuccess` + client invalidation:
```typescript
export const addUnitMemberMutationOpts = mutationOptions({
  mutationFn: ({unitId, body}: { unitId: string; body: AddUnitMemberRequest }) =>
    api<import('@/types/unit.ts').UnitMember>(`/units/${unitId}/members`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  onSuccess: (_, {unitId}, {client}) => {
    client.invalidateQueries({queryKey: ['units', 'members', unitId]})
    client.invalidateQueries({queryKey: ['units', 'tree']})
  },
})

export const removeUnitMemberMutationOpts = mutationOptions({
  mutationFn: ({unitId, membershipId}: { unitId: string; membershipId: string }) =>
    api<void>(`/units/${unitId}/members/${membershipId}`, {method: 'DELETE'}),
  onSuccess: (_, {unitId}, {client}) => {
    client.invalidateQueries({queryKey: ['units', 'members', unitId]})
    client.invalidateQueries({queryKey: ['units', 'tree']})
  },
})
```

**Existing `reparentUnitMutationOpts`** (lines 79-89) — already exists, needs wiring in dialog:
```typescript
export const reparentUnitMutationOpts = mutationOptions({
  mutationFn: ({id, parent_unit_id}: { id: string; parent_unit_id: string | null }) =>
    api<Unit>(`/units/${id}`, {
      method: 'PUT',
      body: JSON.stringify({parent_unit_id}),
    }),
  onSuccess: (_, __, {client}) => {
    client.invalidateQueries({queryKey: ['units', 'tree']})
    client.invalidateQueries({queryKey: ['units', 'detail']})
  },
})
```

**New mutation** (D-02 for `PUT /units/{id}/members/{membershipId}`):
```typescript
export const updateUnitMemberMutationOpts = mutationOptions({
  mutationFn: ({unitId, membershipId, is_primary}: {
    unitId: string
    membershipId: string
    is_primary: boolean
  }) =>
    api<import('@/types/unit.ts').UnitMember>(`/units/${unitId}/members/${membershipId}`, {
      method: 'PUT',
      body: JSON.stringify({is_primary}),
    }),
  onSuccess: (_, {unitId}, {client}) => {
    client.invalidateQueries({queryKey: ['units', 'members', unitId]})
    client.invalidateQueries({queryKey: ['units', 'tree']})
  },
})
```

**New query for batch members** (D-13):
```typescript
export const unitMembersBatchQueryOpts = (unitIds: string[]) => queryOptions({
  queryKey: ['units', 'members', 'batch', ...unitIds.sort()],
  queryFn: async () => {
    const params = unitIds.map(encodeURIComponent).join(',')
    const data = await api<import('@/types/unit.ts').UnitMember[]>(`/units/members/batch?unit_ids=${params}`)
    return UnitMemberSchema.array().parse(data)
  },
  enabled: unitIds.length > 0,
})
```

---

### `web/src/routes/_authenticated/org-hierarchy/-context/org-hierarchy-context.tsx` (store, state management)

**Modification:** Remove `pendingEdgeConnect` and `setPendingEdgeConnect` from state and actions (D-09).

**Analog:** Same file.

**Existing state interface** (lines 7-19) — `pendingEdgeConnect` to remove:
```typescript
export interface OrgHierarchyState {
  viewMode: 'tree' | 'members'
  collapsedIds: Set<string>
  searchQuery: string
  selectedUnit: Unit | null
  formOpen: boolean
  formMode: 'create' | 'edit'
  editingUnit: Unit | null
  deleteOpen: boolean
  reparentTarget: Unit | null
  draggingUnit: Unit | null
  pendingEdgeConnect: { source: string; target: string } | null   // ← REMOVE
}
```

**Existing actions interface** (lines 21-38) — `setPendingEdgeConnect` to remove:
```typescript
export interface OrgHierarchyActions {
  // ... other methods ...
  setPendingEdgeConnect: (connect: { source: string; target: string } | null) => void  // ← REMOVE
  // ...
}
```

**Existing initial state** (lines 43-54) — `pendingEdgeConnect` initial value to remove:
```typescript
    pendingEdgeConnect: null,   // ← REMOVE
```

**Existing setter** (lines 71) — to remove:
```typescript
    setPendingEdgeConnect: (connect) => set({pendingEdgeConnect: connect}),   // ← REMOVE
```

---

### `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/unit-detail-panel.tsx` (component, request-response)

**Modification:** Add "Make Primary" button to `MemberRow` (D-01) + add subtree members section (D-15/D-16).

**Analog:** Same file.

**Existing `MemberRow`** (lines 50-82) — add `onMakePrimary` prop and button:
```tsx
function MemberRow({member, onRemove}: { member: UnitMember; onRemove?: () => void }) {
  const initials = member.user_name
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)

  return (
    <div className="flex items-center gap-2 py-1.5 group">
      <Avatar data-size="sm">
        <AvatarFallback className="text-[10px]">{initials}</AvatarFallback>
      </Avatar>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium truncate">{member.user_name}</p>
        <p className="text-xs text-muted-foreground truncate">{member.role}</p>
      </div>
      {member.is_primary && (
        <Badge variant="outline" className="text-[10px] px-1 py-0 shrink-0">Primary</Badge>
      )}
      {onRemove && (
        <Button
          variant="ghost"
          size="sm"
          className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
          onClick={onRemove}
        >
          <XIcon className="h-3 w-3 text-muted-foreground"/>
        </Button>
      )}
    </div>
  )
}
```

**Updated `MemberRow` with `onMakePrimary`** (D-01):
```tsx
function MemberRow({member, onRemove, onMakePrimary}: {
  member: UnitMember
  onRemove?: () => void
  onMakePrimary?: () => void
}) {
  // ... same initials, avatar, name, role ...
  return (
    <div className="flex items-center gap-2 py-1.5 group">
      {/* ... same avatar, name, role ... */}
      <div className="flex items-center gap-1">
        {member.is_primary ? (
          <Badge variant="outline" className="text-[10px] px-1 py-0 shrink-0">Primary</Badge>
        ) : onMakePrimary && (
          <Button
            variant="ghost"
            size="sm"
            className="h-6 text-[10px] px-1 opacity-0 group-hover:opacity-100 transition-opacity"
            onClick={onMakePrimary}
          >
            Make Primary
          </Button>
        )}
        {onRemove && (
          <XButton onClick={onRemove} />
        )}
      </div>
    </div>
  )
}
```

**Existing Sheet pattern** (lines 186-309) — `selectedUnit` drives open state:
```tsx
export function UnitDetailPanel() {
  const selectedUnit = useOrgHierarchyStore(s => s.selectedUnit)
  const setSelectedUnit = useOrgHierarchyStore(s => s.setSelectedUnit)
  // ...
  return (
    <Sheet open={selectedUnit !== null} onOpenChange={(o) => !o && setSelectedUnit(null)}>
      <SheetContent className="w-[400px] sm:max-w-[400px] overflow-y-auto">
        {selectedUnit && (
          <>
            {/* ... header, breadcrumbs, members section, actions ... */}
            <div className="mt-6 space-y-6 px-4">
              {/* Members section with MemberRow */}
              <div>
                <h4 className="text-sm font-medium mb-2 flex items-center gap-1.5">
                  <Users className="h-3.5 w-3.5"/>
                  Members
                </h4>
                {unitMembers.map((m) => (
                  <MemberRow
                    key={m.id}
                    member={m}
                    onRemove={() => handleRemoveMember(m)}
                  />
                ))}
              </div>
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  )
}
```

**New subtree members section** (D-15/D-16) — added after the existing members section:
```tsx
// Add import for ChevronDown, ChevronRight
import {ChevronDown, ChevronRight} from 'lucide-react'

// New component, placed after AddMemberPopover
function SubtreeMembersSection({unitId}: { unitId: string }) {
  const {data: tree} = useSuspenseQuery(unitTreeQueryOpts)
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set())

  const findNode = (nodes: UnitTreeNode[], id: string): UnitTreeNode | undefined => {
    for (const n of nodes) {
      if (n.unit.id === id) return n
      const found = n.children ? findNode(n.children, id) : undefined
      if (found) return found
    }
    return undefined
  }

  const node = findNode(tree, unitId)
  if (!node?.children?.length) return null

  return (
    <div className="mt-4">
      <h4 className="text-sm font-medium mb-2 flex items-center gap-1.5">
        <Users className="h-3.5 w-3.5"/>
        Sub-unit Members
      </h4>
      {node.children.map((child) => (
        <SubtreeGroup
          key={child.unit.id}
          node={child}
          expanded={expandedGroups.has(child.unit.id)}
          onToggle={() => {
            const next = new Set(expandedGroups)
            if (next.has(child.unit.id)) next.delete(child.unit.id)
            else next.add(child.unit.id)
            setExpandedGroups(next)
          }}
          depth={1}
        />
      ))}
    </div>
  )
}
```

---

### `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/reparent-confirm-dialog.tsx` (component, request-response)

**Modification:** Switch from `updateUnitMutationOpts` to `reparentUnitMutationOpts` (D-08) + remove `pendingEdgeConnect` usage (D-09).

**Analog:** Same file.

**Current implementation** (lines 1-78) — uses `updateUnitMutationOpts`:
```tsx
import {useMutation} from "@tanstack/react-query";
import {updateUnitMutationOpts} from "@/api/units.ts";

export function ReparentConfirmDialog() {
  const draggingUnit = useOrgHierarchyStore(s => s.draggingUnit)
  const reparentTarget = useOrgHierarchyStore(s => s.reparentTarget)
  const pendingEdgeConnect = useOrgHierarchyStore(s => s.pendingEdgeConnect)  // ← REMOVE
  const setDraggingUnit = useOrgHierarchyStore(s => s.setDraggingUnit)
  const setReparentTarget = useOrgHierarchyStore(s => s.setReparentTarget)
  const setPendingEdgeConnect = useOrgHierarchyStore(s => s.setPendingEdgeConnect)  // ← REMOVE

  const reparentOpen = draggingUnit !== null && reparentTarget !== null

  const {mutateAsync: updateUnit} = useMutation(updateUnitMutationOpts)  // ← SWAP

  if (!draggingUnit || !reparentTarget) return null

  const onOpenChange = (open: boolean) => {
    if (!open) {
      setDraggingUnit(null)
      setReparentTarget(null)
      setPendingEdgeConnect(null)  // ← REMOVE
    }
  }

  const handleConfirm = async () => {
    try {
      const newParentId = reparentTarget.id
      await updateUnit({   // ← CHANGE to reparentUnit
        id: draggingUnit.id,
        body: {
          name: draggingUnit.name,
          code: draggingUnit.code,
          description: draggingUnit.description,
          parent_unit_id: newParentId   // ← CHANGE to only send parent_unit_id
        }
      })
      toast.success("Unit moved successfully")
      onOpenChange(false)
    } catch {
      toast.error("Failed to move unit")
    }
  }
```

**Updated implementation** (D-08, D-09):
```tsx
import {useMutation} from "@tanstack/react-query";
import {reparentUnitMutationOpts} from "@/api/units.ts";

export function ReparentConfirmDialog() {
  const draggingUnit = useOrgHierarchyStore(s => s.draggingUnit)
  const reparentTarget = useOrgHierarchyStore(s => s.reparentTarget)
  const setDraggingUnit = useOrgHierarchyStore(s => s.setDraggingUnit)
  const setReparentTarget = useOrgHierarchyStore(s => s.setReparentTarget)

  const reparentOpen = draggingUnit !== null && reparentTarget !== null

  const {mutateAsync: reparentUnit} = useMutation(reparentUnitMutationOpts)

  if (!draggingUnit || !reparentTarget) return null

  const onOpenChange = (open: boolean) => {
    if (!open) {
      setDraggingUnit(null)
      setReparentTarget(null)
    }
  }

  const handleConfirm = async () => {
    try {
      await reparentUnit({
        id: draggingUnit.id,
        parent_unit_id: reparentTarget.id,
      })
      toast.success("Unit moved successfully")
      onOpenChange(false)
    } catch {
      toast.error("Failed to move unit")
    }
  }
```

---

### `web/src/routes/_authenticated/org-hierarchy/-components/org-hierarchy-page.tsx` (component/page, request-response)

**Modification:** Remove `pendingEdgeConnect` reference in `onConnect` (D-09).

**Analog:** Same file.

**Current `onConnect`** (lines 232-242) — uses `setPendingEdgeConnect`:
```typescript
  const onConnect = useCallback(
    (params: { source: string; target: string }) => {
      const sourceUnit = allUnitsMap.get(params.source)
      const targetUnit = allUnitsMap.get(params.target)
      if (sourceUnit && targetUnit) {
        setPendingEdgeConnect({ source: params.source, target: params.target })
        reparentUnit(sourceUnit, targetUnit)
      }
    },
    [allUnitsMap, setPendingEdgeConnect, reparentUnit]
  )
```

**Updated `onConnect`** (D-09) — remove `setPendingEdgeConnect` call:
```typescript
  const onConnect = useCallback(
    (params: { source: string; target: string }) => {
      const sourceUnit = allUnitsMap.get(params.source)
      const targetUnit = allUnitsMap.get(params.target)
      if (sourceUnit && targetUnit) {
        reparentUnit(sourceUnit, targetUnit)
      }
    },
    [allUnitsMap, reparentUnit]
  )
```

Also remove `setPendingEdgeConnect` from destructuring at line 132:
```typescript
  // const setPendingEdgeConnect = useOrgHierarchyStore(s => s.setPendingEdgeConnect)  // ← REMOVE
```

---

## Shared Patterns

### Authentication
**Source:** `internal/middleware/middleware.go` lines 40-61
**Apply to:** All new route registrations in `cmd/server/main.go`
```go
func Auth(authService *auth.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			api.RespondWithError(w, http.StatusUnauthorized, "missing access token")
			return
		}
		claims, err := authService.ValidateToken(cookie.Value)
		if err != nil {
			api.RespondWithError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, OrganizationIDKey, claims.OrganizationID)
		ctx = context.WithValue(ctx, RoleKey, claims.Role)
		ctx = context.WithValue(ctx, EmailKey, claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
```
**Usage in route registration:**
```go
mux.HandleFunc("PUT /units/{id}/members/{membership_id}", middleware.Auth(authService, unitHandler.UpdateMember))
mux.HandleFunc("GET /units/members/batch", middleware.Auth(authService, unitHandler.ListMembersBatch))
```

### API Response Envelope
**Source:** `pkg/api/response.go` lines 1-34
**Apply to:** All new handler methods in `unit.go`
```go
type Response struct {
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
	Status int         `json:"-"`
}

func RespondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	response := Response{Data: payload, Status: status}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

func RespondWithError(w http.ResponseWriter, status int, message string) {
	response := Response{Error: message, Status: status}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}
```

### Organization ID Extraction
**Source:** `internal/middleware/middleware.go` lines 90-95
**Apply to:** All handler methods requiring org scoping
```go
func GetOrganizationID(ctx context.Context) uuid.UUID {
	if orgID, ok := ctx.Value(OrganizationIDKey).(uuid.UUID); ok {
		return orgID
	}
	return uuid.UUID{}
}
```

### Frontend API Client (cookie auth + 401 refresh)
**Source:** `web/src/lib/api.ts` lines 8-56
**Apply to:** All new API calls
```typescript
export async function api<T>(path: string, options?: RequestInit): Promise<T> {
  let res = await fetch(`${API_BASE}${path}`, {
    ...options,
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...options?.headers },
  })
  if (res.status === 401) {
    if (!refreshPromise) {
      refreshPromise = fetch(`${API_BASE}/auth/refresh`, {
        method: 'POST', credentials: 'include'
      }).then(async (refresh) => {
        if (!refresh.ok) { window.location.href = '/login'; throw new Error('Unauthorized') }
      }).finally(() => { refreshPromise = null })
    }
    await refreshPromise
    res = await fetch(`${API_BASE}${path}`, { ...options, credentials: 'include', headers: { 'Content-Type': 'application/json', ...options?.headers } })
  }
  if (!res.ok) {
    const error = await res.json().catch(() => ({message: 'Request failed'})) as ApiError
    throw new Error(error.message || error.error || 'Request failed')
  }
  return (await res.json() as ApiResponse<T>).data
}
```

### Frontend Mutation Invalidation Pattern
**Source:** `web/src/api/units.ts` lines 100-119
**Apply to:** All new mutation options
```typescript
onSuccess: (_, {unitId}, {client}) => {
  client.invalidateQueries({queryKey: ['units', 'members', unitId]})
  client.invalidateQueries({queryKey: ['units', 'tree']})
},
```

### Frontend toast.promise Pattern
**Source:** `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/unit-detail-panel.tsx` lines 101-117
**Apply to:** "Make Primary" mutation success/error feedback
```typescript
toast.promise(
  addMember({unitId, body: {user_id: userId, role: 'member', is_primary: false}}),
  {
    loading: 'Adding member...',
    success: `Added "${userName}" to unit`,
    error: 'Failed to add member',
  }
)
```

### PG Repository Error Wrapping
**Source:** `internal/adapters/secondary/postgres/postgres.go` lines 16-33
**Apply to:** New repository methods
```go
func wrapPGError(err error, op string) error {
	if err == nil { return nil }
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", op, ports.ErrNotFound)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": return fmt.Errorf("%s: %w", op, ports.ErrConflict)
		case "23503": return fmt.Errorf("%s: %w", op, ports.ErrForeignKey)
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
```

### Scan Pattern for Unit Members
**Source:** `internal/adapters/secondary/postgres/unit_member_repository.go` lines 104-123
**Apply to:** New `ListByUnitIDs` method
```go
func scanUnitMember(s memberScanner) (*unit.UnitMember, error) {
	var m unit.UnitMember
	var sqlID uuid.UUID
	var sqlUnitID uuid.UUID
	err := s.Scan(
		&sqlID, &m.OrgID, &m.UserID, &sqlUnitID, &m.IsPrimary, &m.Role,
		&m.StartDate, &m.EndDate, &m.CreatedAt,
		&m.UserName, &m.UserEmail,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, unit.ErrMemberNotFound
		}
		return nil, fmt.Errorf("scan unit member: %w", err)
	}
	m.ID = sqlID.String()
	m.UnitID = sqlUnitID.String()
	return &m, nil
}
```

## No Analog Found

All 12 modified files have exact analogs (themselves). No new files being created.

## Metadata

**Analog search scope:** 
- Backend: `internal/core/domain/unit/`, `internal/core/ports/`, `internal/core/services/unit/`, `internal/adapters/primary/http/`, `internal/adapters/secondary/postgres/`, `internal/middleware/`, `pkg/api/`, `cmd/server/`
- Frontend: `web/src/api/`, `web/src/types/`, `web/src/routes/_authenticated/org-hierarchy/`, `web/src/lib/`

**Files scanned:** ~20 (all directly relevant files)
**Pattern extraction date:** 2026-06-10
