# Phase 11: Foundations — Schema + Origins + Tickets Backend - Pattern Map

**Mapped:** 2026-08-03
**Files analyzed:** 24 (12 new, 12 modified/extends)
**Analogs found:** 20 / 20 with concrete analog (4 new files are self-extension of existing files)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/core/domain/ticket/ticket.go` (NEW) | model | state machine | `internal/core/domain/time_entry/time_entry.go` | exact (status consts + sentinels + state predicates) |
| `internal/core/domain/audit/audit.go` (NEW) | model | event-driven | `time_entry.AuditLog` struct in `domain/time_entry/time_entry.go:77-88` | exact |
| `internal/core/ports/ticket_repository.go` (NEW) | port | CRUD | `internal/core/ports/activity_repository.go` | exact |
| `internal/core/ports/audit_log_repository.go` (NEW) | port | event-driven | `internal/core/ports/time_entry_repository.go:37-38` (`AuditLogRepository`) | exact |
| `internal/core/services/ticket/` (NEW) | service | state machine + event-driven | `internal/core/services/time_entry/time_entry.go` | exact |
| `internal/core/services/routing/` (NEW) | service | request-response (resolution) | `time_entry.go:127-226` (`resolveManagerStage`/`resolveUnitManager`) — verbatim extraction | exact |
| `internal/adapters/primary/http/ticket_handler.go` (NEW) | controller | request-response | `internal/adapters/primary/http/activity_handler.go` (CRUD + sentinel switch) + `time_entry.go` (action endpoints) | exact |
| `internal/adapters/secondary/postgres/ticket_repository.go` (NEW) | repository | CRUD + transaction | `activity_repository.go` (CRUD/scan) + `refresh_token_repo.go:88-137` (BeginTx) | exact |
| `internal/adapters/secondary/postgres/audit_log_repository.go` (NEW) | repository | event-driven | `time_entry_repository.go:303-343` (`AuditLogRepository`) | exact (must add tx-aware variant — see note) |
| `migrations/014_activity_origins.{up,down}.sql` (NEW) | migration | DDL | `migrations/011_activity_ontology.up.sql:46-64` (activities base) + `012_staffing_schema.up.sql` (ALTER pattern) | exact |
| `migrations/015_contract_sold_hours.{up,down}.sql` (NEW) | migration | DDL | `migrations/012_staffing_schema.up.sql:39-50` (nullable ALTER + CHECK) | exact |
| `migrations/016_ticket_schema.{up,down}.sql` (NEW) | migration | DDL | `migrations/012_staffing_schema.up.sql:15-34` (CREATE TABLE + CHECK vocab + index) | exact |
| `migrations/017_audit_logs.{up,down}.sql` (NEW) | migration | DDL | `migrations/012_staffing_schema.up.sql` (append-only table + index) | exact |
| `internal/core/domain/activity/activity.go` (EXTEND) | model | CRUD | itself — add origin fields to struct (L41-57) + `CreateActivityRequest` (L96-106) | self |
| `internal/core/domain/contract/contract.go` (EXTEND) | model | CRUD | itself — add `contract_type`/`sold_hours`/`sold_period` to struct (L20-31) + requests | self |
| `internal/core/services/activity/activity.go` (EXTEND) | service | CRUD | itself — origin validation in `Create` (L64-90) + immutability in `Update` (L94-111) | self |
| `internal/core/services/contract/contract.go` (EXTEND) | service | CRUD | itself — sold_hours validation in `Update` (L42-47) | self |
| `internal/adapters/primary/http/activity_handler.go` (EXTEND) | controller | request-response | itself — origin payload in `CreateActivityRequest` (L36-46) + Create (L98-164) | self |
| `internal/adapters/primary/http/contract.go` (EXTEND) | controller | request-response | itself — `KmRate *float64` nullable-field pattern (L129-137, L139-183) for sold_hours | self |
| `internal/adapters/secondary/postgres/activity_repository.go` (EXTEND) | repository | CRUD | itself — `Create` INSERT (L121-135), `scanActivity` (L594-614), `baseActivityQuery` (L32-45) | self |
| `internal/adapters/secondary/postgres/contract_repository.go` (EXTEND) | repository | CRUD | itself — dynamic SET `Update` (L140-218), `scanContractResponse` (L315-327) | self |
| `cmd/server/main.go` (EXTEND) | config/wiring | request-response | itself — route registration (L191-205), service wiring (L120-130) | self |
| `internal/core/services/time_entry/time_entry.go` (EXTEND: extract) | service | — | itself — move L127-226 to `services/routing` | self |
| `internal/adapters/secondary/postgres/exported_test_helpers.go` (EXTEND) | test util | — | itself — teardown list (L79-121) | self |
| `activity_ontology_migration_test.go` + `staffing_schema_migration_test.go` (FIX + EXTEND) | test | — | `staffing_schema_migration_test.go:22-144` (cycle-test skeleton) — also fix seed wiring per Pitfall 3 | exact |
| Migration cycle tests `TestMigration014..017` (NEW) | test | — | `staffing_schema_migration_test.go:22-144` | exact |

## Pattern Assignments

### `internal/core/domain/ticket/ticket.go` (model, state machine)

**Analog:** `internal/core/domain/time_entry/time_entry.go` (100 lines — copy shape wholesale)

**Imports + sentinel errors** (time_entry.go L1-18):
```go
package time_entry

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTimeEntryNotFound    = errors.New("time entry not found")
	ErrEntryNotDraft        = errors.New("entry is not in draft status")
	ErrForbidden            = errors.New("forbidden")
)
```

**Status vocabulary constants** (time_entry.go L20-27) — tickets need the same for `status` (`open`,`triage`,`planned`,`in_progress`,`resolved`,`closed`,`dismissed`) and `kind` (`question`,`bug`,`change`,`evolution`). DB CHECK is the source of truth; constants mirror it:
```go
const (
	StatusDraft          = "draft"
	StatusSubmitted      = "submitted"
	StatusPendingManager = "pending_manager"
	...
)
```

**Entity + state predicates** (time_entry.go L29-47 + L90-100) — ticket entity carries state fields (org_id, title, description, kind, status, requester_id, assignee_id, dismissal note per A4); transition edges are service-side but the domain carries the vocabulary and predicates:
```go
func (e *TimeEntry) IsOwner(userID uuid.UUID) bool {
	return e.UserID == userID
}

func (e *TimeEntry) CanEdit() bool {
	return e.Status == StatusDraft || e.Status == StatusSubmitted || e.Status == StatusRejected
}
```

**New sentinels needed** (no analog — define per D-14/D-03): `ErrInvalidTransition`, `ErrTicketNotFound`, `ErrOriginImmutable`, `ErrDismissalBlocked`, `ErrResolvedBlocked`, `ErrForbidden` — same `errors.New` style.

---

### `internal/core/domain/audit/audit.go` (model, event-driven)

**Analog:** `time_entry.AuditLog` struct (time_entry.go L77-88) + port (ports/time_entry_repository.go L37-38)

**Entity to copy** (time_entry.go L77-88) — generalize for `entity_type`/`entity_id` instead of entry-specific fields; `payload` maps to the existing `Changes map[string]any`:
```go
type AuditLog struct {
	ID        uuid.UUID      `json:"id"`
	OrgID     uuid.UUID      `json:"org_id"`
	EntryID   string         `json:"entry_id"`
	EntryType string         `json:"entry_type"`
	Action    string         `json:"action"`
	ActorRole string         `json:"actor_role"`
	ActorID   uuid.UUID      `json:"actor_id"`
	Reason    string         `json:"reason"`
	Changes   map[string]any `json:"changes"`
	Timestamp time.Time      `json:"timestamp"`
}
```

**Port to copy** (ports/time_entry_repository.go L37-38) — keep `Create` for fire-and-forget entry parity, **add** a tx-aware method for tickets (triage/transitions write audit rows in the same transaction, D-10/Pitfall 2):
```go
type AuditLogRepository interface {
	Create(ctx context.Context, log *time_entry.AuditLog) error
}
```
Ticket port addition (discretion area, but must exist for atomic triage):
```go
CreateTx(ctx context.Context, tx pgx.Tx, log *audit.AuditLog) error
```

---

### `internal/core/services/ticket/` (service, state machine + event-driven)

**Analog:** `internal/core/services/time_entry/time_entry.go` (412 lines — the codebase's only state-machine service)

**Service struct + DI constructor** (time_entry.go L15-31) — ticket service takes ticketRepo + auditRepo + activityRepo (+ wgRepo/unitRepo for proposal routing if it composes triage):
```go
type Service struct {
	repo         ports.TimeEntryRepository
	approvalRepo ports.TimeEntryApprovalRepository
	wgRepo       ports.WorkingGroupRepository
	activityRepo ports.ActivityRepository
	unitRepo     ports.UnitRepository
}

func NewService(repo ports.TimeEntryRepository, approvalRepo ports.TimeEntryApprovalRepository, wgRepo ports.WorkingGroupRepository, activityRepo ports.ActivityRepository, unitRepo ports.UnitRepository) *Service {
	return &Service{repo: repo, approvalRepo: approvalRepo, wgRepo: wgRepo, activityRepo: activityRepo, unitRepo: unitRepo}
}
```

**State-transition method shape** (time_entry.go L228-262 `Submit`) — every ticket transition follows this exact skeleton: load → permission check → state check → resolve → mutate → repo.Update. Copy for `Transition`/`Reopen`:
```go
func (s *Service) Submit(ctx context.Context, id, userID uuid.UUID) (*time_entry.TimeEntry, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !e.CanSubmit() {
		return nil, time_entry.ErrEntryNotDraft
	}
	if !e.IsOwner(userID) {
		return nil, time_entry.ErrNotOwner
	}

	// ... resolve stage ...
	now := time.Now()
	e.SubmittedAt = &now
	e.UpdatedAt = now
	// ... mutate status ...

	return s.repo.Update(ctx, e)
}
```

**Role-gated transition with audit row write** (time_entry.go L264-320 `Approve`) — the audit-row-after-update pattern; for tickets the audit row must be written **in the same tx** (Pitfall 2) instead of this pool-level `CreateApproval`:
```go
	if err := s.approvalRepo.CreateApproval(ctx, &time_entry.Approval{
		ID:          uuid.New(),
		EntryID:     id,
		Action:      "approve",
		ActorUserID: userID,
		ActorRole:   role,
		CreatedAt:   time.Now(),
	}); err != nil {
		return nil, err
	}
```

**Dismissal guard** (no analog — new method): service computes Σ logged hours via a stable port method `LoggedHours(ctx, ticketID) (float64, error)` (D-13/Pitfall 6), signature kept for the Phase 12 computation swap. SQL lives repo-side:
```sql
SELECT COALESCE(SUM(hours),0) FROM time_entries
WHERE is_deleted = false AND status IN ('submitted','approved')
AND activity_id IN (SELECT id FROM activities WHERE ticket_id = $1 AND origin_type = 'customer_ticket')
```

**Permission gates** (D-15) — read claims via `middleware.GetRole/GetUserID/GetOrganizationID` (middleware.go L67-80); compare against `models.RoleEmployee/RoleManager/RoleFinance` (same pattern as activity.go L95: `if role != string(models.RoleFinance)`).

---

### `internal/core/services/routing/` (service, extraction — verbatim move)

**Analog:** `internal/core/services/time_entry/time_entry.go` L127-226 — the ENTIRE `managerResolution` struct, `resolveManagerStage`, and `resolveUnitManager` move verbatim into the new package (Pattern 5, D-G parity). No re-implementation — copy:

```go
// time_entry.go L133-137
type managerResolution struct {
	approverIDs   []uuid.UUID
	roleGated     bool
	skipToFinance bool // D-11: the entry owner IS in the approver set
}
```
`resolveManagerStage` (L149-196) and `resolveUnitManager` (L201-226) move as-is. Dependencies become constructor params: `routing.NewService(wgRepo, activityRepo, unitRepo)`. The time_entry service then calls `routing.ResolveManagerStage(...)` instead of its own method (L243, L282). Proposal approval (FND-02) calls the same function with `unitID` = proposer's primary unit. Preserve `ErrActivityNotLoggable` propagation (activity.go L25).

---

### `internal/adapters/primary/http/ticket_handler.go` (controller, request-response)

**Analog:** `internal/adapters/primary/http/activity_handler.go` (357 lines) — same structure; action endpoints (triage/transition/comments) follow `time_entry.go` handler's `POST /time-entries/{id}/submit` shape (main.go L219).

**Handler struct + constructor** (activity_handler.go L27-34):
```go
type ActivityHandler struct {
	service *activitysvc.Service
	repo    ports.ActivityRepository
}

func NewActivityHandler(service *activitysvc.Service, repo ports.ActivityRepository) *ActivityHandler {
	return &ActivityHandler{service: service, repo: repo}
}
```

**DTO with string IDs parsed at the boundary** (activity_handler.go L36-46) — same for ticket DTOs (kind/status as strings; parent/contract/ticket IDs as strings):
```go
type CreateActivityRequest struct {
	ParentID        string                 `json:"parent_id"`
	Name            string                 `json:"name"`
	Kind            string                 `json:"kind"`
	ContractID      string                 `json:"contract_id"`
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	...
}
```

**Create handler skeleton** (activity_handler.go L98-164) — decode → `validateStringLengths` → `uuid.Parse` → service call → sentinel switch → envelope:
```go
func (h *ActivityHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	var req CreateActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// ... boundary validation ...
	activity, err := h.service.Create(r.Context(), orgID, svcReq)
	if err != nil {
		switch {
		case errors.Is(err, activitydomain.ErrInvalidRequest):
			api.RespondWithError(w, http.StatusBadRequest, "invalid activity payload")
		case errors.Is(err, activitydomain.ErrActivityNotFound):
			api.RespondWithError(w, http.StatusBadRequest, "parent activity not found")
		default:
			api.RespondWithError(w, http.StatusInternalServerError, "failed to create activity")
		}
		return
	}
	api.RespondWithJSON(w, http.StatusCreated, activity)
}
```

**Guard sentinels → 409** (activity_handler.go L303-330) — dismissal-block and resolved-block map to `http.StatusConflict` exactly like has-children:
```go
	case errors.Is(err, activitydomain.ErrHasChildren):
		api.RespondWithError(w, http.StatusConflict, "activity has children and cannot be deleted")
```

---

### `internal/adapters/secondary/postgres/ticket_repository.go` (repository, CRUD + transaction)

**Analog:** `internal/adapters/secondary/postgres/activity_repository.go` (CRUD) + `refresh_token_repo.go` (BeginTx)

**Struct + compile-time port assertion** (activity_repository.go L19-28):
```go
type ActivityRepository struct {
	pool *pgxpool.Pool
}

// Compile-time assertion: ActivityRepository implements the full port.
var _ ports.ActivityRepository = (*ActivityRepository)(nil)

func NewActivityRepository(pool *pgxpool.Pool) *ActivityRepository {
	return &ActivityRepository{pool: pool}
}
```

**Atomic triage — the codebase's only BeginTx precedent** (refresh_token_repo.go L88-136; the research's Common Operation 1 shape). This is the exact pattern for `Triage` (ticket→planned + 1..N activities + 2 audit rows in one tx):
```go
tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
if err != nil {
	return nil, fmt.Errorf("begin refresh token rotation: %w", err)
}
defer func() { _ = tx.Rollback(ctx) }()
// ... tx.QueryRow / tx.Exec writes ...
if err := tx.Commit(ctx); err != nil {
	return nil, fmt.Errorf("commit refresh token rotation: %w", err)
}
```
Simpler variant (activity_repository.go L498-521 `Delete`) if TxOptions are unneeded: `r.pool.Begin(ctx)` + `defer tx.Rollback(ctx)` + `tx.Commit(ctx)`. **Note:** activity inserts inside the triage tx must run the same validation as `POST /activities` (Pitfall 7) — reuse activity-repo query logic against `tx` (pass a `pgx.Tx`-compatible interface or run the validations in the service before calling the tx).

**INSERT + RETURNING scan** (activity_repository.go L121-135) — copy for ticket create:
```go
	_, err := r.pool.Exec(ctx, `INSERT INTO activities (id, org_id, parent_id, name, description, kind,
		contract_id, governance_model, created_by_org_id, is_shared, billable, budget_amount,
		is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, true, NOW(), NOW())`,
		id, orgID, req.ParentID, req.Name, req.Description, req.Kind,
		req.ContractID, req.GovernanceModel, orgID, req.IsShared, req.Billable, req.BudgetAmount)
	if err != nil {
		return nil, wrapPGError(err, "create activity")
	}
	return r.Get(ctx, orgID, id)
```

**Scan helper with nullable columns** (activity_repository.go L594-614) — nullable `*uuid.UUID` locals assigned after Scan; same for `assignee_id`, `dismissal_note`, origin refs:
```go
	var parentID *uuid.UUID
	var contractID *uuid.UUID
	...
	a.ParentID = parentID
	a.ContractID = contractID
```

**Error mapping:** `pgx.ErrNoRows` → domain sentinel (activity_repository.go L112-114), `wrapPGError` for constraint errors (postgres.go L16-33).

---

### `internal/adapters/secondary/postgres/audit_log_repository.go` (repository, event-driven)

**Analog:** `internal/adapters/secondary/postgres/time_entry_repository.go` L303-343

**Struct + constructor + JSON marshal** (time_entry_repository.go L303-343):
```go
// AuditLogRepository implements ports.AuditLogRepository using a pgxpool.
type AuditLogRepository struct {
	pool *pgxpool.Pool
}

func NewAuditLogRepository(pool *pgxpool.Pool) *AuditLogRepository {
	return &AuditLogRepository{pool: pool}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *time_entry.AuditLog) error {
	id := uuid.New()
	changesJSON, err := json.Marshal(log.Changes)
	if err != nil {
		return fmt.Errorf("marshal audit log changes: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO time_entry_approvals (id, time_entry_id, user_id, action, comment, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, entryID, log.ActorID, log.Action, string(changesJSON), log.Timestamp)
	return wrapPGError(err, "create audit log")
}
```
**Deviation (required):** the `CREATE TABLE`-only precedent writes via `r.pool.Exec` — the ticket path must additionally expose `CreateTx(ctx, tx pgx.Tx, log)` so triage/transition audit rows commit atomically with the state change (D-10, Pitfall 2). Insert shape: `INSERT INTO audit_logs (id, org_id, entity_type, entity_id, action, actor_id, comment, payload, created_at) VALUES (...)`, `payload` = marshaled `Changes` map.

---

### `migrations/014_activity_origins.{up,down}.sql` (migration, DDL)

**Analog:** `migrations/011_activity_ontology.up.sql:46-64` (activities table being extended) + `012_staffing_schema.up.sql:39-50` (nullable ALTER + CHECK drop/recreate)

**ALTER + nullable columns** (012 L39-42 pattern):
```sql
ALTER TABLE organization_memberships
    ADD COLUMN valid_from DATE,
    ADD COLUMN valid_until DATE,
    ADD COLUMN work_permit_expires_at DATE;
```

**Discriminator CHECK with three-valued-logic guard** (Pattern 1 — RESEARCH.md L204-224; the `origin_type IS NULL OR (...)` guard is the critical house-style detail, per Pitfall 1):
```sql
ALTER TABLE activities ADD CONSTRAINT activities_origin_refs_check
  CHECK (
    origin_type IS NULL
    OR (origin_type = 'manager_assignment' AND assigned_by IS NOT NULL AND assigned_to IS NOT NULL
        AND proposed_by IS NULL AND reviewed_by IS NULL AND ticket_id IS NULL)
    OR (origin_type = 'employee_proposal' AND proposed_by IS NOT NULL
        AND assigned_by IS NULL AND assigned_to IS NULL AND ticket_id IS NULL)
    OR (origin_type = 'customer_ticket' AND ticket_id IS NOT NULL
        AND assigned_by IS NULL AND assigned_to IS NULL
        AND proposed_by IS NULL AND reviewed_by IS NULL)
  );
CREATE INDEX idx_activities_ticket_id ON activities(ticket_id);
```
**Planner note (A8):** 016 (tickets) must precede 014 so `ticket_id REFERENCES tickets(id)` resolves — order migrations accordingly.

---

### `migrations/015_contract_sold_hours.{up,down}.sql` (migration, DDL)

**Analog:** `012_staffing_schema.up.sql` L39-50 — nullable ALTER + CHECK. D-08/D-09 shape (Pattern 2 in RESEARCH L349-362):
```sql
ALTER TABLE contracts ADD COLUMN contract_type VARCHAR(50);
ALTER TABLE contracts ADD COLUMN sold_hours DECIMAL(10,2);
ALTER TABLE contracts ADD COLUMN sold_period VARCHAR(10);
ALTER TABLE contracts ADD CONSTRAINT contracts_sold_check
  CHECK (
    contract_type IS NULL
    OR (contract_type = 'support' AND sold_hours IS NOT NULL AND sold_period IS NOT NULL)
    OR (contract_type = 'project'  AND sold_period IS NULL)
  );
```
(Align `sold_hours` scale with `time_entries.hours DECIMAL(8,2)` per FND-03 — planner confirms final precision.)

---

### `migrations/016_ticket_schema.{up,down}.sql` (migration, DDL)

**Analog:** `012_staffing_schema.up.sql` L15-34 (CREATE TABLE + inline CHECK vocab + index) and 011 L46-64 (FK style). RESEARCH Common Operation 4 (L377-397) gives the skeleton; house-style CHECK vocabulary:
```sql
CREATE TABLE tickets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(id),
    title         VARCHAR(255) NOT NULL,
    description   TEXT,
    kind          VARCHAR(50) NOT NULL CHECK (kind IN ('question','bug','change','evolution')),
    status        VARCHAR(50) NOT NULL DEFAULT 'open'
                  CHECK (status IN ('open','triage','planned','in_progress','resolved','closed','dismissed')),
    requester_id  UUID NOT NULL REFERENCES users(id),
    assignee_id   UUID REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_tickets_org_id ON tickets(org_id);
CREATE INDEX idx_tickets_status ON tickets(status);
```
Plus `ticket_comments` (D-06 first-class comments: id, ticket_id FK, author_id, body, created_at — no update/delete paths, TICK-05). `down.sql`: `DROP TABLE IF EXISTS ... CASCADE` (012 down style).

---

### `migrations/017_audit_logs.{up,down}.sql` (migration, DDL)

**Analog:** `012_staffing_schema.up.sql` — simple append-only table. D-05 shape:
`id UUID PK, org_id FK, entity_type VARCHAR(50) NOT NULL, entity_id UUID NOT NULL, action VARCHAR(50) NOT NULL, actor_id UUID REFERENCES users(id), comment TEXT, payload JSONB, created_at TIMESTAMPTZ DEFAULT NOW()` + `CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id, created_at)`.

---

### Modified files (self-analogs — extend in place)

| File | What to extend | Analog line refs |
|------|---------------|------------------|
| `domain/activity/activity.go` | Add `OriginType *string` + 5 nullable ref fields (`AssignedBy/AssignedTo/ProposedBy/ReviewedBy/TicketID *uuid.UUID`) to `Activity` (L41-57), `CreateActivityRequest` (L96-106) | self |
| `domain/contract/contract.go` | Add `ContractType *string`, `SoldHours *float64`, `SoldPeriod *string` to `Contract` (L20-31) + both requests | self |
| `services/activity/activity.go` | Origin same-org validation + role gates in `Create` (L64-90 pattern); **immutability check in `Update`** (L94-111: compare existing vs req, reject origin changes with new sentinel `ErrOriginImmutable` — D-03); proposal approve endpoint (is_active flip + audit row, D-12) | self |
| `services/contract/contract.go` | sold_hours semantic validation in `Update` (L42-47): support requires sold_period, project forbids it (D-08) | self |
| `http/activity_handler.go` | Add origin fields to `CreateActivityRequest` (L36-46) + parse in `Create` (L134-149 uuid.Parse pattern) | self |
| `http/contract.go` | Add `ContractType *string / SoldHours *float64 / SoldPeriod *string` to DTOs (L129-137 `*float64` nullable pattern — same as KmRate) + pass through in `Update` (L159-167) | self |
| `postgres/activity_repository.go` | Extend `baseActivityQuery` (L32-45), `Create` INSERT (L121-135), `scanActivity`/`scanActivityResponse` (L594-638) with origin columns as nullable `*uuid.UUID` locals | self |
| `postgres/contract_repository.go` | Extend `baseContractQuery` (L29-47), `Update` dynamic SET (L140-218, `*float64` block L150-154 is the sold_hours template; L175-188 nullable-clear pattern), `scanContractResponse` (L315-327) | self |
| `cmd/server/main.go` | Construct ticket/audit repos + services (L120-130 pattern); register routes (L191-205 pattern): `mux.HandleFunc("POST /tickets", middleware.Auth(authService, ticketHandler.Create))`, `POST /tickets/{id}/triage`, `POST /tickets/{id}/comments`, `POST /tickets/{id}/transition`, `GET /tickets/{id}/history` | self |
| `time_entry.go` | Delete L127-226 after extraction; call `routing.ResolveManagerStage` at L243 + L282 | self |
| `exported_test_helpers.go` | Teardown list (L79-121): add `audit_logs`, `ticket_comments`, `tickets` **before** `activities` (Pitfall 8 — tickets FK from activities, so drop order matters) | self |

---

## Shared Patterns

### Authentication + permission gates
**Source:** `internal/middleware/middleware.go` L67-80 + `cmd/server/main.go` L191-205
**Apply to:** every new handler method
```go
func GetUserID(ctx context.Context) uuid.UUID { ... }          // L67
func GetOrganizationID(ctx context.Context) uuid.UUID { ... }  // L73
func GetRole(ctx context.Context) string { ... }               // L80
```
Routes registered with `middleware.Auth(authService, handler.Method)` (main.go L191-205); role gates enforced **in the service** (activity.go L95: `if role != string(models.RoleFinance) { return nil, activitydomain.ErrForbidden }`), D-04/D-11/D-15 gates compare `models.RoleManager` / `models.RoleFinance` / `models.RoleEmployee`.

### Error handling (3 layers)
**Source:** `postgres.go` L16-33 (repo), domain sentinels (time_entry.go L10-18), handler switch (activity_handler.go L152-162)
**Apply to:** all repos + services + handlers
```go
// repo layer: wrapPGError translates pgx errors → ports.ErrNotFound/ErrConflict/ErrForeignKey
func wrapPGError(err error, op string) error { ... }
```
Service returns domain sentinels; handler maps with `errors.Is`/`switch err` → `api.RespondWithError(w, status, msg)`; guard violations → 409 (activity_handler.go L318-323).

### API response envelope
**Source:** `pkg/api/response.go` (via AGENTS.md L120-128)
**Apply to:** all handlers — `api.RespondWithJSON(w, http.StatusCreated, activity)` on success, `api.RespondWithError` on failure.

### Atomic multi-write transactions
**Source:** `refresh_token_repo.go` L88-136 (`pool.BeginTx` + `defer Rollback` + `Commit`)
**Apply to:** ticket triage (D-10), ticket transitions (audit rows in-tx, Pitfall 2)
```go
tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
if err != nil { return nil, fmt.Errorf("begin ...: %w", err) }
defer func() { _ = tx.Rollback(ctx) }()
// ... tx.Exec writes ...
if err := tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit ...: %w", err) }
```

### DB CHECK vocabularies + three-valued-logic guard
**Source:** `migrations/012_staffing_schema.up.sql` L15-30, RESEARCH Pattern 1
**Apply to:** all 4 new migrations — every multi-column CHECK written as `discriminator IS NULL OR (<per-type rules with explicit IS [NOT] NULL>)` so legacy rows pass (Pitfall 1).

### Service unit tests (mock-repo fixture)
**Source:** `internal/core/services/time_entry/time_entry_test.go` L19-46
**Apply to:** `services/ticket/` and `services/routing/` unit tests
```go
type serviceFixture struct {
	svc          *Service
	repo         *testdata.MockTimeEntryRepo
	...
}
func setupService(t *testing.T) *serviceFixture { ... }
```
Mocks live in `internal/core/services/testdata/` (MockActivityRepo, MockTimeEntryRepo, etc.) — ticket tests add `MockTicketRepo`/`MockAuditLogRepo` there.

### Integration tests + migration cycle tests
**Source:** `staffing_schema_migration_test.go` L22-144 + `exported_test_helpers.go` L20-75 (TestPool/SetupPackageContainer/applyMigrations/readMigration/assertTableExists/assertConstraintExists/assertColumnNotNull/assertFkAction)
**Apply to:** `TestMigration014..017`, ticket repo tests, and the **pre-existing red-test fix** (Pitfall 3 — both 011 and 012 cycle tests assert MVP seed counts that `applyMigrations` no longer loads; fix the seed wiring first, then write 014-017 tests against the fixed helper).

## No Analog Found

| File | Role | Data Flow | Reason / Fallback |
|------|------|-----------|--------------------|
| `internal/core/services/routing/` (package) | service | request-response | No standalone package analog — but its two functions are a **verbatim extraction** from `time_entry.go` L127-226, so pattern fidelity is exact; unit-test shape follows the fixture pattern above |
| Ticket transition matrix + dismissal guard | service logic | state machine | No existing transition-matrix or hours-Σ logic in the codebase; build on `Submit`/`Approve` skeleton (time_entry.go L228-320) + repo `COALESCE(SUM(hours),0)` query (Pitfall 6); edges per A7 |

## Metadata

**Analog search scope:** `internal/core/domain/*`, `internal/core/ports/`, `internal/core/services/*`, `internal/adapters/primary/http/`, `internal/adapters/secondary/postgres/`, `cmd/server/main.go`, `migrations/`
**Files scanned:** 31 (24 read in full for pattern extraction)
**Pattern extraction date:** 2026-08-03
