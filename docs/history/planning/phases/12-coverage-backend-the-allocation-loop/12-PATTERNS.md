# Phase 12: Coverage Backend — The Allocation Loop — Pattern Map

**Mapped:** 2026-08-07
**Files analyzed:** 24 (14 new, 10 modified)
**Analogs found:** 22 / 24 (2 doc-only tasks have no code analog)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `migrations/018_activity_beneficiary_unit.{up,down}.sql` (NEW) | migration | schema change | `migrations/015_activity_origins.up.sql` (nullable col + index); `016` 3VL CHECK | exact |
| `migrations/019_coverage_allocations.{up,down}.sql` (NEW) | migration | schema change | `migrations/015_activity_origins.up.sql` (tagged union + refs-to-type CHECK + vocabulary CHECK) | exact |
| `migrations/020_coverage_snapshots.{up,down}.sql` (NEW) | migration | schema change | `migrations/017_audit_logs.up.sql` (append-only table + index) | role-match |
| `internal/core/domain/coverage/coverage.go` + `errors.go` (NEW) | model | CRUD | `internal/core/domain/ticket/ticket.go` (entity + closed vocab + sentinels + JSONNames) | exact |
| `internal/core/ports/coverage_repository.go` (NEW) | port | CRUD | `internal/core/ports/ticket_repository.go` | exact |
| `internal/core/services/coverage/coverage.go` (NEW) | service | request-response + CRUD | `internal/core/services/ticket/ticket.go` (gates + audit) + `time_entry.go:174-193` (manager gate) | exact |
| `internal/adapters/primary/http/coverage_handler.go` (NEW) | controller | request-response | `internal/adapters/primary/http/ticket_handler.go` | exact |
| `internal/adapters/secondary/postgres/coverage_repository.go` (NEW) | repository | CRUD (tx) | `ticket_repository.go` (BeginTx + FOR UPDATE + in-tx audit) + `activity_repository.go:236-264` (CTE resolvers) | exact |
| `internal/core/services/coverage/coverage_test.go` (NEW) | test | — | `internal/core/services/ticket/ticket_test.go` | exact |
| `internal/adapters/secondary/postgres/coverage_repository_test.go` (NEW) | test | — | `internal/adapters/secondary/postgres/ticket_repository_test.go` (incl. CR-01 concurrent test) | exact |
| `internal/adapters/secondary/postgres/coverage_ontology_migrations_test.go` (NEW) | test | — | `internal/adapters/secondary/postgres/ontology_extension_migrations_test.go` | exact |
| `internal/adapters/primary/http/coverage_handler_test.go` (NEW) | test | — | `internal/adapters/primary/http/ticket_handler_test.go` | exact |
| `internal/core/services/testdata/mock_coverage_repo.go` (NEW) | test | — | `internal/core/services/testdata/mock_ticket_repo.go` | exact |
| `hourglass-vault/decisions/backend/ADR-BE-0xx — Coverage encoding.md` (NEW) | doc | — | `hourglass-vault/decisions/backend/ADR-BE-016` (conventions; not code) | no analog |
| `internal/core/domain/activity/activity.go` (MODIFY) | model | CRUD | origin axis block in same file (lines 74-82, 120-163) | exact |
| `internal/core/ports/activity_repository.go` (MODIFY) | port | CRUD | `ResolveCommercialContext` decl (line 28) | exact |
| `internal/adapters/secondary/postgres/activity_repository.go` (MODIFY) | repository | CRUD | `ResolveCommercialContext` (236-264), `GetAncestry` (194-231), `scanActivity` (605-634), `Update` (428-503) | exact |
| `internal/core/services/activity/activity.go` (MODIFY) | service | CRUD | `validateOrigin` (130-194), `Update` (210-230) | exact |
| `internal/adapters/primary/http/activity_handler.go` (MODIFY) | controller | request-response | `UpdateActivityRequest` DTO (57-77), Create/Update parse (167-213, 306-374) | exact |
| `internal/adapters/secondary/postgres/exported_test_helpers.go` (MODIFY) | test | — | `TeardownTestSchema` (79-124) + seed helpers (138-196) | exact |
| `cmd/server/main.go` (MODIFY) | config | — | ticket wiring (145-153) + routes (220-228) | exact |
| `internal/core/services/testdata/mocks.go` (MODIFY) | test | — | `MockActivityRepo` (480+) | exact |
| `hourglass-vault/decisions/project/ADR-P-012` + `_index.md` (MODIFY) | doc | — | vault conventions | no analog |

---

## Pattern Assignments

### `migrations/018_activity_beneficiary_unit.{up,down}.sql` (migration)

**Analog:** `migrations/015_activity_origins.up.sql` (single nullable column + index) and `016_contract_sold_hours.down.sql` (drop pattern)

**Core pattern** (`015` lines 19-24 — nullable additive columns; no backfill):
```sql
-- 018 up: single nullable column, no 3VL CHECK needed (mirrors contract_id, per 011)
ALTER TABLE activities ADD COLUMN beneficiary_unit_id UUID REFERENCES units(id);
CREATE INDEX idx_activities_beneficiary_unit_id ON activities(beneficiary_unit_id);
```
**Down pattern** (`016_contract_sold_hours.down.sql` lines 2-6):
```sql
-- 018 down (ADR-BE-004 cycle):
ALTER TABLE activities DROP INDEX ...; -- drop index first
ALTER TABLE activities DROP COLUMN IF EXISTS beneficiary_unit_id;
```

### `migrations/019_coverage_allocations.{up,down}.sql` (migration)

**Analog:** `migrations/015_activity_origins.up.sql` — the discriminator + nullable refs + three-valued-logic CHECK house rule (ADR-BE-016)

**Vocabulary CHECK** (`015` lines 29-30):
```sql
ALTER TABLE activities ADD CONSTRAINT activities_origin_type_check
    CHECK (origin_type IN ('manager_assignment', 'employee_proposal', 'customer_ticket'));
```

**Refs-to-type CHECK — the 3VL guard D-01 copies verbatim** (`015` lines 36-47; same shape in `016` lines 25-29):
```sql
ALTER TABLE activities ADD CONSTRAINT activities_origin_refs_check
    CHECK (origin_type IS NULL OR
           (origin_type = 'manager_assignment' AND
            assigned_by IS NOT NULL AND assigned_to IS NOT NULL AND ...) OR
           (origin_type = 'customer_ticket' AND
            ticket_id IS NOT NULL AND ...));
```
→ For 019: `CHECK (source_type IS NULL OR (source_type='contract' AND contract_id IS NOT NULL AND unit_id IS NULL) OR (source_type='absorption' AND unit_id IS NOT NULL AND contract_id IS NULL) OR (source_type='transfer' AND contract_id IS NOT NULL AND unit_id IS NULL))` plus mandatory-field CHECKs `(source_type <> 'absorption' OR reason IS NOT NULL)` etc. (full recommended shape in RESEARCH.md Pattern 1, lines 206-237). `hours DECIMAL(8,2) NOT NULL CHECK (hours > 0)` matches `time_entries.hours` exactly (migration 000 line 278).

### `migrations/020_coverage_snapshots.{up,down}.sql` (migration)

**Analog:** `migrations/017_audit_logs.up.sql` — append-only table + entity index (no UPDATE/DELETE paths)

**Core pattern** (`017` lines 16-29):
```sql
CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id),
    ...
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id, created_at);
```
→ For 020: two tables `coverage_period_closes` (header) + `coverage_snapshot_rows` (entry-level, `ON DELETE CASCADE` from close) with `idx_coverage_snapshot_rows_close` / `idx_coverage_snapshot_rows_entry` (recommended shape in RESEARCH.md Pattern 5, lines 279-305). Down drops rows then header.

---

### `internal/core/domain/coverage/coverage.go` (model)

**Analog:** `internal/core/domain/ticket/ticket.go` — entity + closed vocabulary constants + sentinels + JSONNames map

**Imports + sentinel pattern** (`ticket.go` lines 3-8, 65-83):
```go
package ticket

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTicketNotFound = errors.New("ticket not found")
	ErrInvalidRequest = errors.New("invalid request")
	ErrForbidden      = errors.New("forbidden")
	...
)

// JSONNames maps sentinel errors to stable JSON-safe names (house style).
var JSONNames = map[error]string{ ... }
```

**Closed vocabulary constants** (`ticket.go` lines 45-63):
```go
const (
	KindQuestion = "question"
	...
	StatusOpen = "open"
)
```
→ For coverage: `SourceTypeContract/Absorption/Transfer` consts, `AbsorptionReasonWarrantyBug/UnderEstimate/Goodwill`, `EntryTypeTime`, `StatusApproved` reuse. New sentinels per RESEARCH.md: `ErrEntryNotCoverable`, `ErrForbidden`, `ErrInvalidRequest`, `ErrAllocationSumMismatch`, `ErrPeriodAlreadyClosed` (409).

**Entity struct with nullable refs** (`ticket.go` lines 15-33 — `*uuid.UUID` for nullable, `time.Time`):
```go
type Ticket struct {
	ID             uuid.UUID  `json:"id"`
	...
	AssigneeID     *uuid.UUID `json:"assignee_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
```
→ `CoverageAllocation{ID, OrgID, EntryType, EntryID, SourceType, ContractID *uuid.UUID, UnitID *uuid.UUID, Hours float64, Reason *string, Justification *string, CreatedAt, UpdatedAt}` + `CoverageProposal` / `ToCoverQueueRow` / `SnapshotRow` read models.

### `internal/core/ports/coverage_repository.go` (port)

**Analog:** `internal/core/ports/ticket_repository.go` — interface with doc comments stating in-tx audit contract

**Core pattern** (`ticket_repository.go` lines 12-41):
```go
// Every mutator (Create/UpdateDetails/...) writes its audit_logs row IN THE
// SAME TRANSACTION as the state write (Pitfall 2, ADR-BE-016): the caller
// passes the audit row(s) to write. The interface is append-only by
// construction — no update/delete paths ... (TICK-05).
type TicketRepository interface {
	Get(ctx context.Context, orgID, ticketID uuid.UUID) (*ticket.Ticket, error)
	Create(ctx context.Context, orgID uuid.UUID, t *ticket.Ticket, audit *audit.AuditLog) (*ticket.Ticket, error)
	...
	LoggedHours(ctx context.Context, ticketID uuid.UUID) (float64, error)
}
```
→ `CoverageRepository`: `ReplaceAllocations(ctx, orgID, entryID, allocs []*coverage.Allocation, audit *audit.AuditLog) ([]*coverage.Allocation, error)`, `ListByEntry`, `Proposal(ctx, orgID, entryID)`, `ToCoverQueue(ctx, orgID)`, `BucketBalance(ctx, orgID, contractID)`, `ClosePeriod(ctx, orgID, start, end time.Time, closeID, by uuid.UUID, audit *audit.AuditLog) (*coverage.PeriodClose, error)`, `GetSnapshot(ctx, orgID, closeID)`, `ListHistory` (mirror).

### `internal/core/services/coverage/coverage.go` (service)

**Analog:** `internal/core/services/ticket/ticket.go` (service owns gates + builds audit rows) + `time_entry.go:174-193` (D-08 manager gate)

**Service struct + ctor with port deps** (`ticket.go` lines 28-42):
```go
type Service struct {
	repo         ports.TicketRepository
	activityRepo ports.ActivityRepository
	contractRepo ports.ContractRepository
	orgRepo      ports.OrganizationRepository
}
func NewService(...) *Service { ... }
```
→ Coverage `Service` holds `repo ports.CoverageRepository`, `activityRepo`, `contractRepo`, `routing *routing.Service`, `entryRepo` (approved-entry reads) — wiring in `cmd/server/main.go`.

**D-08 manager gate — copy the Approve gate verbatim** (`time_entry.go` lines 171-193):
```go
// Structural self-approval barrier: the owner can never approve their own
// entry in any role (ADR-P-001 Q4 ...).
if e.UserID == userID {
	return nil, time_entry.ErrForbidden
}
res, err := s.routing.ResolveManagerStage(ctx, e.OrgID, e.ActivityID, e.UnitID, e.UserID)
if err != nil {
	if errors.Is(err, activity.ErrActivityNotLoggable) {
		return nil, time_entry.ErrForbidden
	}
	return nil, err
}
if !res.RoleGated && !contains(res.ApproverIDs, userID) {
	return nil, time_entry.ErrForbidden
}
```
→ Coverage write: resolve routing on the entry's `(OrgID, ActivityID, UnitID, UserID)`; finance/employee rejected for writes, reads open to manager+finance.

**Audit-row construction** (`ticket.go` lines 105-113):
```go
actor := actorID
return s.repo.Create(ctx, orgID, t, &audit.AuditLog{
	OrgID: orgID, EntityType: "ticket", EntityID: t.ID,
	Action: "created", ActorID: &actor,
	Payload: map[string]any{"kind": req.Kind},
	CreatedAt: now,
})
```
→ Coverage: `EntityType: "coverage_allocation"`, `Action: "allocations-set"` / `"coverage-closed"`, payload = full allocation set JSON (A7).

**Fast-fail vs in-tx authoritative pattern** (`ticket.go` lines 200-203 doc + `Dismiss` 269-272 doc): pool-level Σ/status checks are UX only; the repo re-validates under FOR UPDATE.

**Service sentinel returns:** mirror `ErrForbidden`/`ErrInvalidRequest` returns; `contains` helper copy from `time_entry.go:302-309`.

### `internal/adapters/primary/http/coverage_handler.go` (controller)

**Analog:** `internal/adapters/primary/http/ticket_handler.go` — thin handler, boundary DTOs, sentinel map, `writeError`

**Handler struct + imports** (`ticket_handler.go` lines 3-14, 35-41):
```go
import (
	"encoding/json"
	"errors"
	"net/http"
	"github.com/google/uuid"
	ticketdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/ticket"
	ticketsvc "github.com/stefanoprivitera/hourglass/internal/core/services/ticket"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)
type TicketHandler struct { service *ticketsvc.Service }
func NewTicketHandler(service *ticketsvc.Service) *TicketHandler { ... }
```

**Handler method shape** (`ticket_handler.go` lines 96-125):
```go
func (h *TicketHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	assigneeID, err := parseOptionalUUID(req.AssigneeID)
	if err != nil { api.RespondWithError(w, http.StatusBadRequest, "invalid assignee_id"); return }
	created, err := h.service.Create(ctx, orgID, userID, role, &ticketsvc.CreateTicketRequest{...})
	if err != nil { h.writeError(w, err); return }
	api.RespondWithJSON(w, http.StatusCreated, created)
}
```

**Boundary DTO with string IDs** (`ticket_handler.go` lines 44-49): string IDs parsed at the boundary via `parseOptionalUUID` (activity_handler.go:81-90).

**Sentinel map — copy the switch** (`ticket_handler.go` lines 347-364):
```go
func (h *TicketHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ticketdomain.ErrTicketNotFound):
		api.RespondWithError(w, http.StatusNotFound, "ticket not found")
	case errors.Is(err, ticketdomain.ErrInvalidRequest):
		api.RespondWithError(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, ticketdomain.ErrForbidden):
		api.RespondWithError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ticketdomain.ErrDismissalBlocked):
		api.RespondWithError(w, http.StatusConflict, "dismissal blocked: ...")
	default:
		api.RespondWithError(w, http.StatusInternalServerError, "internal server error")
	}
}
```
→ Coverage: `ErrEntryNotCoverable` → 404, `ErrAllocationSumMismatch`/`ErrInvalidRequest` → 400, `ErrForbidden` → 403, `ErrPeriodAlreadyClosed` → 409.

### `internal/adapters/secondary/postgres/coverage_repository.go` (repository)

**Analog:** `internal/adapters/secondary/postgres/ticket_repository.go` (773 lines) — the full Phase 11 repo; `activity_repository.go:236-264` for CTE resolvers

**Repo skeleton + compile-time assertion** (`ticket_repository.go` lines 22-36):
```go
type TicketRepository struct { pool *pgxpool.Pool }
var _ ports.TicketRepository = (*TicketRepository)(nil)
func NewTicketRepository(pool *pgxpool.Pool) *TicketRepository { return &TicketRepository{pool: pool} }
```

**In-tx audit helper — copy and rename** (`ticket_repository.go` lines 83-112):
```go
func insertTicketAudit(ctx context.Context, tx pgx.Tx, log *audit.AuditLog) error {
	id := uuid.New()
	var payload any
	if len(log.Payload) > 0 {
		payloadJSON, err := json.Marshal(log.Payload)
		if err != nil { return fmt.Errorf("marshal ticket audit payload: %w", err) }
		payload = payloadJSON
	}
	var comment any
	if log.Comment != "" { comment = log.Comment }
	_, err := tx.Exec(ctx,
		`INSERT INTO audit_logs (id, org_id, entity_type, entity_id, action, actor_id, comment, payload, created_at)
		 VALUES ($1, $2, 'ticket', $3, $4, $5, $6, $7, $8)`,
		id, log.OrgID, log.EntityID, log.Action, log.ActorID, comment, payload, log.CreatedAt)
	if err != nil { return wrapPGError(err, "insert ticket audit log") }
	return nil
}
```
→ `insertCoverageAudit` with `entity_type` param or hardcoded `'coverage_allocation'` — do NOT add a public `CreateTx` to the port (ticket precedent, RESEARCH Pattern 4).

**Replace-set tx: BeginTx + FOR UPDATE + DELETE/INSERT + audit + Commit** (mirror `Rotate` refresh_token_repo.go:88-137 + ticket `UpdateState` 239-298):
```go
tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
if err != nil { return nil, fmt.Errorf("begin replace allocations: %w", err) }
defer func() { _ = tx.Rollback(ctx) }() // safe even after Commit

// 1. Lock the entry row and re-validate (CR-01: in-tx check, never pool-only)
var hours float64
err = tx.QueryRow(ctx,
	`SELECT hours FROM time_entries
	  WHERE id = $1 AND org_id = $2 AND status = 'approved' AND is_deleted = false
	  FOR UPDATE`, entryID, orgID).Scan(&hours)
if errors.Is(err, pgx.ErrNoRows) { return nil, coverage.ErrEntryNotCoverable }
// 2. Σ validation (cents arithmetic avoids float64 artifacts)
// 3. DELETE FROM coverage_allocations WHERE entry_id = $1 AND entry_type = 'time'
// 4. INSERT each allocation row (source_type + refs + hours + reason/justification)
// 5. insertCoverageAudit(ctx, tx, &audit.AuditLog{ ... Action: "allocations-set", ... })
if err := tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit replace allocations: %w", err) }
```
(Full shape in RESEARCH.md Common Operation 1, lines 418-446.)

**FOR UPDATE precedent to copy** (`ticket_repository.go` lines 249-251):
```go
err = tx.QueryRow(ctx,
	`SELECT status FROM tickets WHERE id = $1 AND org_id = $2 FOR UPDATE`,
	ticketID, orgID).Scan(&currentStatus)
if errors.Is(err, pgx.ErrNoRows) { return nil, ticketdomain.ErrTicketNotFound }
```

**Pool + Tx pair for guards** (`ticket_repository.go` 688-717 — `LoggedHours` / `loggedHoursTx`): pool-level = fast-fail UX, `*Tx` variant = authoritative inside mutator. The close tx follows the same pair shape (overlap check pool + in-tx).

**Chain-walk CTE resolver — copy `ResolveCommercialContext`** (`activity_repository.go` lines 236-264):
```go
WITH RECURSIVE chain AS (
	SELECT id, parent_id, contract_id FROM activities WHERE id = $1
	UNION ALL
	SELECT a.id, a.parent_id, a.contract_id
	FROM activities a
	INNER JOIN chain c ON a.id = c.parent_id
	WHERE c.contract_id IS NULL
)
SELECT c.contract_id, ct.customer_id
FROM chain c
LEFT JOIN contracts ct ON ct.id = c.contract_id
WHERE c.contract_id IS NOT NULL
LIMIT 1
```
→ `ResolveFundingContext` (add `ct.contract_type, ct.sold_hours` via JOIN) and `ResolveBeneficiaryUnit` (same CTE with `beneficiary_unit_id`, `WHERE c.contract_id IS NULL OR c.beneficiary_unit_id IS NULL` walk — RESEARCH Common Operation 2, lines 448-466).

**ListHistory audit read** (`ticket_repository.go` 639-678): scan `(id, org_id, entity_type, entity_id, action, actor_id, comment, payload, created_at)`, unmarshal payload JSONB into `map[string]any` — reuse for `GET /coverage/allocations/{entry_id}/history` with `entity_type='coverage_allocation'`.

**wrapPGError** (`postgres.go:16-33`) for all pool/tx errors; sentinel mapping via `errors.Is(err, pgx.ErrNoRows)` handled explicitly in queries.

### `internal/core/services/testdata/mock_coverage_repo.go` (test)

**Analog:** `internal/core/services/testdata/mock_ticket_repo.go` — mutex-guarded map + call/audit capture + `Fn` override knobs

**Core pattern** (`mock_ticket_repo.go` lines 22-51):
```go
type MockTicketRepo struct {
	mu      sync.Mutex
	Tickets map[uuid.UUID]*ticketdomain.Ticket
	GetFn   func(ctx context.Context, orgID, ticketID uuid.UUID) (*ticketdomain.Ticket, error)
	Audits  []*audit.AuditLog          // every audit row handed to a mutator
	LoggedHoursResult float64          // behavior knobs
	LoggedHoursFn     func(ctx context.Context, ticketID uuid.UUID) (float64, error)
}
var _ ports.TicketRepository = (*MockTicketRepo)(nil)
```
→ `MockCoverageRepo` mirrors: `Allocations map[uuid.UUID][]*coverage.Allocation`, `Audits []*audit.AuditLog`, per-method `Fn` overrides, same-org semantics on Get. Register in `mocks.go`/`factories.go` pattern (mocks.go:480 `MockActivityRepo`).

### `internal/adapters/secondary/postgres/exported_test_helpers.go` (MODIFY)

**Analog:** same file — `TeardownTestSchema` + seed helpers

**Teardown list** (lines 79-124): add `coverage_snapshot_rows`, `coverage_period_closes`, `coverage_allocations` **before** `time_entries`/`activities` (dependency order, Pitfall 8):
```go
tables := []string{
	...
	"financial_cutoff_periods",
	"coverage_snapshot_rows",     // ← NEW (before time_entries)
	"coverage_period_closes",     // ← NEW
	"coverage_allocations",       // ← NEW
	...
	"time_entries",
	"activities",
	...
}
```
**Seed helpers** (138-196): may add `seedContract(t, pool, orgID, contractType, soldHours)` / `seedTimeEntry(t, pool, orgID, userID, activityID, unitID, hours, status)` following `seedActivity`/`seedUnit` INSERT shape.

### `cmd/server/main.go` (MODIFY)

**Analog:** same file — ticket wiring + route registration

**Service wiring** (lines 144-153):
```go
auditRepo := postgres.NewGeneralAuditLogRepository(pool)
ticketRepo := postgres.NewTicketRepository(pool)
...
ticketService := ticketsvc.NewService(ticketRepo, activityRepo, contractRepo, orgRepo)
ticketHandler := http.NewTicketHandler(ticketService)
```
→ Coverage: `coverageRepo := postgres.NewCoverageRepository(pool)`; `coverageService := coveragesvc.NewService(coverageRepo, activityRepo, contractRepo, timeEntryRepo, routingSvc)`; `coverageHandler := http.NewCoverageHandler(coverageService)`.

**Route registration** (lines 220-228 — all `middleware.Auth(authService, ...)`):
```go
mux.HandleFunc("POST /tickets", middleware.Auth(authService, ticketHandler.Create))
...
mux.HandleFunc("GET /tickets/{id}/history", middleware.Auth(authService, ticketHandler.History))
```
→ Coverage routes (RESEARCH.md lines 314-324): `PUT /time-entries/{id}/allocations`, `GET /coverage/to-cover`, `GET /coverage/proposals/{entry_id}`, `GET /coverage/buckets/{contract_id}/balance`, `POST /coverage/close`, `GET /coverage/snapshots/{close_id}`, `GET /coverage/allocations/{entry_id}/history`.

### `internal/core/domain/activity/activity.go` (MODIFY)

**Analog:** origin axis block in same file (Phase 11 pattern for new nullable columns)

**Field addition** (after `ContractID` line 64):
```go
ContractID *uuid.UUID `json:"contract_id,omitempty"` // D-3: nullable = internal work
// COV-05: beneficiary unit, nullable; inherited downward like contract_id (D-3)
BeneficiaryUnitID *uuid.UUID `json:"beneficiary_unit_id,omitempty"`
```
Also add to `CreateActivityRequest` (after line 126) and `UpdateActivityRequest` (after line 148) — pointer fields only, `json:"...,omitempty"`.

### `internal/core/ports/activity_repository.go` (MODIFY)

**Analog:** `ResolveCommercialContext` decl (line 28)

```go
// ResolveBeneficiaryUnit walks ancestry: nearest ancestor with
// beneficiary_unit_id set (COV-05 — inherited downward like contract_id).
ResolveBeneficiaryUnit(ctx context.Context, activityID uuid.UUID) (*uuid.UUID, error)
```

### `internal/adapters/secondary/postgres/activity_repository.go` (MODIFY)

**Analog:** same file — four touch points (Pitfall 5)

1. **`baseActivityQuery` SELECT** (lines 33-36): add `a.beneficiary_unit_id` (display joins untouched).
2. **`GetAncestry` explicit SELECT list** (lines 195-211 — the recursion lists columns explicitly; MUST gain `beneficiary_unit_id` in seed + recursive + final SELECT).
3. **`scanActivity`/`scanActivityResponse`** (lines 605-634): add `var beneficiaryUnitID *uuid.UUID` + scan slot + `a.BeneficiaryUnitID = beneficiaryUnitID` (exact origin-columns pattern).
4. **`Update` dynamic SET** (lines 428-503): add `if req.BeneficiaryUnitID != nil { sets = append(...); args = append(args, *req.BeneficiaryUnitID); argIdx++ }` (mirror `req.ContractID` branch lines 478-482).
5. **New `ResolveBeneficiaryUnit`** — CTE shape copy of `ResolveCommercialContext` (236-264) with `beneficiary_unit_id` and the dual-column walk guard from RESEARCH Common Operation 2.

### `internal/core/services/activity/activity.go` (MODIFY)

**Analog:** `validateOrigin` + `Update` in same file

- `Create` (87-117): add same-org existence check for `req.BeneficiaryUnitID` via `unitRepo` (mirror the `req.ContractID` check at 106-110).
- `Update` (210-230): add the same check before `s.activityRepo.Update` (beneficiary unit is editable, unlike origin refs — do NOT add to `hasOriginFields`).

### `internal/adapters/primary/http/activity_handler.go` (MODIFY)

**Analog:** same file — `UpdateActivityRequest` DTO + parse blocks

- Add `BeneficiaryUnitID *string \`json:"beneficiary_unit_id,omitempty"\`` to `UpdateActivityRequest` (lines 57-77) and `CreateActivityRequest` (36-55).
- Parse in Create (mirror `req.ContractID` block lines 175-182) and Update (mirror lines 315-322):
```go
if req.BeneficiaryUnitID != nil {
	bid, err := uuid.Parse(*req.BeneficiaryUnitID)
	if err != nil { api.RespondWithError(w, http.StatusBadRequest, "invalid beneficiary_unit_id"); return }
	svcReq.BeneficiaryUnitID = &bid
}
```

### `internal/adapters/secondary/postgres/coverage_ontology_migrations_test.go` (NEW — cycle tests)

**Analog:** `ontology_extension_migrations_test.go` — up → down → up self-seed cycle per migration

**Cycle skeleton** (lines 26-84, `TestMigration014` shape; 018 skeleton in RESEARCH Common Operation 4):
```go
func TestMigration014_TicketSchema_UpDownUpCycle(t *testing.T) {
	pool := TestPool(t)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })
	ctx := context.Background()
	now := time.Now()

	up014 := readMigration(t, "014_ticket_schema.up.sql")
	down014 := readMigration(t, "014_ticket_schema.down.sql")
	applyMigrations(t, pool, true, "014...", "015...", "016...", "017...") // pre-state, skip under test
	orgID := seedOrg(t, pool, now)
	...
	// UP: assert tables/constraints (assertTableExists / assertConstraintExists /
	// assertFkAction) + functional 3VL assertions (legacy NULL row passes, mixed
	// refs fail with pgErr.Code "23514" + constraint name — lines 142-150)
	// DOWN: columns/constraints gone (information_schema + pg_constraint counts)
	// UP again: green
}
```
For 019 add the Pitfall-2 guards: absorption-without-unit fails, transfer-without-justification fails, legacy all-NULL passes.

### `internal/adapters/secondary/postgres/coverage_repository_test.go` (NEW)

**Analog:** `ticket_repository_test.go` (844 lines) — repo integration tests incl. the CR-01 concurrent test (lines 418-506)

- Replace-set: Σ = hours commits, Σ ≠ hours rejected, partial state impossible (COV-01)
- **Concurrent replace-set test mirroring CR-01** (ticket_repository_test.go:418-506 pattern — two goroutines, assert no violating state commits)
- Audit assertions: count + payload (COV-03), close freezes rows + later edits don't change snapshot (COV-04), bucket balance negative allowed (COV-02)

### `internal/adapters/primary/http/coverage_handler_test.go` (NEW)

**Analog:** `ticket_handler_test.go` (280 lines) — handler integration tests with permission matrix

- Pattern: build service with real repo + testcontainers pool (`handler_integration_test.go` / `handler_test_helper.go`), seed org/user/roles, hit handler via `httptest`, assert envelope + status codes. Permission matrix: manager writes, employee/finance/customer rejected (D-08).

### `internal/core/services/coverage/coverage_test.go` (NEW)

**Analog:** `internal/core/services/ticket/ticket_test.go` (765 lines) — unit tests with `testdata.MockCoverageRepo`

- Pattern: table-driven service tests using the mock; assert audit payloads via `mock.Audits`; permission tests with fake routing resolution (mock `routing.ResolveManagerStage` path — the mock repos make the resolution deterministic).

---

## Shared Patterns

### In-transaction audit writes (BE-016)
**Source:** `ticket_repository.go:83-112` (`insertTicketAudit`)
**Apply to:** `coverage_repository.go` replace-set + close (copy helper as `insertCoverageAudit`, private, no public `CreateTx` on the port). Entity vocab (A7): `entity_type='coverage_allocation'` + actions `allocations-set`/`coverage-closed`, payload JSONB. Never fire-and-forget (CONTEXT discretion, RESEARCH Pattern 4).

### FOR UPDATE in-tx re-validation (CR-01)
**Source:** `ticket_repository.go:249-251` (lock) + `loggedHoursTx` pair pattern (706-717)
**Apply to:** `ReplaceAllocations` — `SELECT hours FROM time_entries WHERE id=$1 AND org_id=$2 AND status='approved' AND is_deleted=false FOR UPDATE`, re-check Σ in cents inside the tx; pool-level checks are fast-fail UX only. Close tx re-checks overlap in-tx.

### Manager gate (D-08)
**Source:** `time_entry.go:174-193` + `routing.go:57-104` (`ResolveManagerStage`)
**Apply to:** `coverage.Service` write path — structural self-barrier (`e.UserID == userID` → forbidden), `ErrActivityNotLoggable` → forbidden, `!res.RoleGated && !contains(res.ApproverIDs, userID)` → forbidden. Finance read-only. Reuse `routing.ResolveManagerStage`, never re-implement (RESEARCH Don't-Hand-Roll).

### 3VL CHECK guard house rule
**Source:** `migrations/015_activity_origins.up.sql:36-47`, `016_contract_sold_hours.up.sql:25-29`
**Apply to:** migration 019 `coverage_allocations_source_check` (all `source_type IS NULL OR (...)` with explicit `IS [NOT] NULL` pins) + mandatory-field CHECKs `(source_type <> 'absorption' OR reason IS NOT NULL)`.

### Sentinel error mapping
**Source:** `ticket_handler.go:347-364` (`writeError` switch), `postgres.go:16-33` (`wrapPGError`)
**Apply to:** `coverage_handler.go` (404/400/403/409/500 map) and all repo methods.

### Response envelope
**Source:** `pkg/api.RespondWithJSON` / `RespondWithError` (used everywhere; e.g. `ticket_handler.go:124,138`)
**Apply to:** every coverage handler response — `{ data | error }` shape, no ad-hoc marshaling.

### Migrations append-only + cycle tests (ADR-BE-004)
**Source:** `migrations/*.{up,down}.sql` numbering 018→019→020; `ontology_extension_migrations_test.go` self-seed cycle skeleton
**Apply to:** all three migrations with up/down pairs; `TeardownTestSchema` list updated in the same task (Pitfall 8).

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `hourglass-vault/decisions/backend/ADR-BE-0xx — Coverage encoding.md` | doc | — | Vault decision record; follow ADR-BE-016/014 file conventions (numbered, "Accepted"/"Draft" status, decision/cost/verification sections). Must cost the D-K polymorphic validation branch honestly. |
| `hourglass-vault/decisions/project/ADR-P-012` + `_index.md` | doc | — | Status flip Proposed → Accepted; index entry for the new BE ADR. No code analog. |

## Metadata

**Analog search scope:** `internal/core/domain/`, `internal/core/ports/`, `internal/core/services/`, `internal/adapters/primary/http/`, `internal/adapters/secondary/postgres/`, `migrations/`, `cmd/server/`, `internal/core/services/testdata/`
**Files scanned:** 24 (11 analogs read in full, 4 partially via targeted grep; all < 2,000 lines)
**Pattern extraction date:** 2026-08-07

**Key takeaway for the planner:** every new coverage artifact has an exact Phase 11 (ticket/origin) analog — copy the ticket stack for domain/port/service/handler/repo/mocks/tests, copy the 015/016 migrations for the tagged-union CHECK, and copy `time_entry.go` Approve + `routing.ResolveManagerStage` for the D-08 gate. The only genuinely novel mechanics are the replace-set DELETE+INSERT tx shape (compose `Rotate` + `insertTicketAudit` precedents) and the close snapshot write (append-only tables like 017).
