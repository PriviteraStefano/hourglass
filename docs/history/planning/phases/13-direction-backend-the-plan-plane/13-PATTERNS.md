# Phase 13: Direction Backend — The Plan Plane — Pattern Map

**Mapped:** 2026-08-08
**Files analyzed:** 25 (19 new / 6 modified)
**Analogs found:** 24 / 25 (1 ADR doc has no code analog)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `migrations/021_direction_rows.{up,down}.sql` | migration | batch (DDL) | `migrations/014_ticket_schema.up.sql` + `015_activity_origins.up.sql` | exact |
| `migrations/022_org_settings.{up,down}.sql` | migration | batch (DDL) | `migrations/012_staffing_schema.up.sql` | exact |
| `internal/core/domain/direction/direction.go` | domain | CRUD (state machine) | `internal/core/domain/ticket/ticket.go` | exact |
| `internal/core/domain/direction/errors.go` | domain | n/a | `internal/core/domain/coverage/errors.go` | exact |
| `internal/core/domain/orgsettings/orgsettings.go` | domain | CRUD (key/value) | `internal/core/domain/coverage/coverage.go` (vocab consts) + `coverage/errors.go` | role-match |
| `internal/core/ports/direction_repository.go` | port | CRUD + read-model | `internal/core/ports/ticket_repository.go` + `ports/coverage_repository.go` | exact |
| `internal/core/ports/org_settings_repository.go` | port | CRUD (key/value) | `internal/core/ports/audit_log_repository.go` | role-match |
| `internal/core/services/direction/direction.go` | service | CRUD + read-model + event-driven (supersede/claim tx) | `internal/core/services/ticket/ticket.go` (matrix) + `services/coverage/coverage.go` (Σ cents, gates) | exact |
| `internal/core/services/orgsettings/orgsettings.go` | service | CRUD (key/value) | `internal/core/services/organization/organization.go` (settings) + `coverage.go` (manager gate) | role-match |
| `internal/adapters/primary/http/direction_handler.go` | controller | request-response | `internal/adapters/primary/http/coverage_handler.go` | exact |
| `internal/adapters/primary/http/org_settings_handler.go` | controller | request-response | `internal/adapters/primary/http/organization.go` (GetSettings/UpdateSettings) + `coverage_handler.go` (writeError) | role-match |
| `internal/adapters/secondary/postgres/direction_repository.go` | repository | CRUD + read-model | `internal/adapters/secondary/postgres/ticket_repository.go` + `coverage_repository.go` | exact |
| `internal/adapters/secondary/postgres/org_settings_repository.go` | repository | CRUD (key/value) | `internal/adapters/secondary/postgres/coverage_repository.go` (small repo + in-tx audit) | role-match |
| `internal/adapters/secondary/postgres/direction_ontology_migrations_test.go` | test | batch | `internal/adapters/secondary/postgres/coverage_ontology_migrations_test.go` | exact |
| `internal/adapters/secondary/postgres/direction_repository_test.go` | test | CRUD (incl. concurrent) | `internal/adapters/secondary/postgres/ticket_repository_test.go` (lines 414-509 concurrent battery) | exact |
| `internal/adapters/secondary/postgres/org_settings_repository_test.go` | test | CRUD | `coverage_ontology_migrations_test.go` (seed + assert helpers) | role-match |
| `internal/core/services/direction/direction_test.go` | test | CRUD | `internal/core/services/ticket/ticket_test.go` (mock-driven unit) | role-match |
| `internal/core/services/orgsettings/orgsettings_test.go` | test | CRUD | `internal/core/services/coverage/coverage_test.go` | role-match |
| `internal/core/services/testdata/mock_direction_repo.go` | test (mock) | n/a | `internal/core/services/testdata/mock_ticket_repo.go` | exact |
| `internal/core/services/testdata/mock_org_settings_repo.go` | test (mock) | n/a | `internal/core/services/testdata/mock_ticket_repo.go` | exact |
| `internal/adapters/primary/http/direction_handler_test.go` | test | request-response | `internal/adapters/primary/http/coverage_handler_test.go` + `handler_test_helper.go` fixture | exact |
| `internal/adapters/primary/http/org_settings_handler_test.go` | test | request-response | `internal/adapters/primary/http/organization_test.go` (literal-route coexistence) | role-match |
| **MOD** `internal/core/services/activity/activity.go` | service (extend) | CRUD | itself — `GetByID` (lines 73-75), `List` (lines 57-59) | exact (self) |
| **MOD** `internal/core/services/testdata/mocks.go` | test (extend) | n/a | `testdata/mocks.go` MockActivityRepo constructor | exact (self) |
| **MOD** `internal/adapters/secondary/postgres/exported_test_helpers.go` | test helper (extend) | n/a | itself — `TeardownTestSchema` (lines 79-127) | exact (self) |
| **MOD** `cmd/server/main.go` | config (wiring) | n/a | itself — coverage wiring block (lines 160-162, 242-249) | exact (self) |
| **MOD** `internal/adapters/primary/http/handler_test_helper.go` | test helper (extend) | n/a | itself — `newHandlerFixture` (lines 48-239) | exact (self) |
| `hourglass-vault/decisions/` ADR-P-015 + BE ADR | doc | n/a | no code analog — use `hourglass-vault/decisions/backend/ADR-BE-017` as doc template | n/a |

---

## Pattern Assignments

### `internal/core/domain/direction/direction.go` (domain, CRUD state machine)

**Analog:** `internal/core/domain/ticket/ticket.go` (129 lines — read whole file)

**Entity + status vocabulary** (ticket.go lines 15-33, 55-63):
```go
type Ticket struct {
	ID             uuid.UUID  `json:"id"`
	OrgID          uuid.UUID  `json:"org_id"`
	...
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
```
Direction row replaces with: `DirectedBy uuid.UUID`, `DirectedTo *uuid.UUID`, `WgID *uuid.UUID`, `ActivityID uuid.UUID`, `PlannedDate *time.Time`, `EstHours *float64`, `Priority *int`, `DueDate *time.Time`, `SupersedesID *uuid.UUID`, `OriginDirectionID *uuid.UUID`, `Reason *string` (D-13-01). Status constants mirror D-13-07: `StatusDraft/StatusActive/StatusSuperseded/StatusCancelled`.

**Transition matrix** (ticket.go lines 85-113) — the pinned direction matrix (D-13-07/08; supersede NOT an endpoint):
```go
var transitionMatrix = map[string]map[string]bool{
	StatusOpen: {
		StatusTriage:    true,
		StatusDismissed: true,
	},
	...
}
// CanTransition reports whether the locked matrix allows from→to.
func CanTransition(from, to string) bool {
	return transitionMatrix[from][to]
}
// IsTerminalStatus reports whether the status is a terminal state
func IsTerminalStatus(s string) bool {
	return s == StatusClosed || s == StatusDismissed
}
```
Direction matrix: `draft → {active, cancelled}`; `active → {cancelled}`; superseded reachable only via Create-with-supersedes_id (D-13-08). Add `IsTerminalStatus` covering superseded+cancelled.

**Derived-on-read precedent** (ticket.go lines 24-30): `DismissedNote` is computed at scan time, never stored — same philosophy as derived `done`/`lapsed`/`claimed` spectrum (D-13-09/15). Add a `DirectionRefs` struct for the origin-fallback shape (D-13-32: `AssignedBy`, `AssignedTo`).

**Audit vocabulary constants** (from `internal/core/domain/coverage/coverage.go` lines 145-153):
```go
const (
	AuditActionAllocationsSet     = "allocations-set"
	AuditActionCoverageClosed     = "coverage-closed"
	AuditEntityCoverageAllocation = "coverage_allocation"
)
```
Direction pins (A1): `AuditEntityDirection = "direction"` with actions `created`/`activated`/`cancelled`/`superseded`/`claimed`/`unclaimed`; `AuditEntityOrgSettings = "org_settings"` with `settings-updated`. **Exported** so repo/service/handler can never drift.

### `internal/core/domain/direction/errors.go` (domain, sentinels)

**Analog:** `internal/core/domain/coverage/errors.go` (38 lines — copy shape verbatim)

```go
var (
	ErrEntryNotCoverable = errors.New("entry not coverable")
	...
	ErrForbidden = errors.New("forbidden")
	ErrInvalidRequest = errors.New("invalid request")
	ErrNotFound = errors.New("not found")
)
// JSONNames maps sentinel errors to stable JSON-safe names (house style...)
var JSONNames = map[error]string{ ... }
```
Direction needs: `ErrDirectionNotFound`, `ErrInvalidRequest`, `ErrForbidden`, `ErrInvalidTransition`, `ErrClaimOverBudget` (409 — A1/D-13-13), `ErrInvalidMode`/`ErrInvalidTarget` (XOR fast-fail), `ErrNotWgMember` (D-13-12), `ErrWgRowNotActive`, `ErrCancelReasonRequired` (D-13-10/16), `ErrInvalidHours` (D-13-03). Plus `JSONNames`.

### `internal/core/domain/orgsettings/orgsettings.go` (domain, small)

**Analog:** `internal/core/domain/coverage/coverage.go` vocabulary-const block (lines 119-153) + `coverage/errors.go` sentinel block

Key vocabulary constants (A7, D-13-18/24): `KeyPlanningDailyHours = "planning_daily_hours"`, `KeyPlanningDeadline = "planning_deadline"`, `KeyPlanningHorizon = "planning_horizon"`, `KeyPlanningMode = "planning_mode"`; mode vocabulary `manager_planned`/`self_planned` (D-13-19); horizon vocabulary `day|week|month` (D-13-21); `DefaultDailyHours = 8.0` (D-13-24). Export a `knownKeys map[string]validatorFn` — validation per key lives in code (D-13-18: CHECK on JSONB impossible). Sentinels: `ErrUnknownKey`, `ErrInvalidValue`, `ErrForbidden`, `ErrNotFound` + `JSONNames`.

### `internal/core/ports/direction_repository.go` (port)

**Analog:** `internal/core/ports/ticket_repository.go` (41 lines) — doc-comment style + mutator signature shape; `internal/core/ports/coverage_repository.go` (69 lines) — Σ-invariant contract + read-model methods

Signature shape — every mutator takes its audit row(s) and writes them IN THE SAME TRANSACTION (ticket_repository.go lines 28-40):
```go
type TicketRepository interface {
	Get(ctx context.Context, orgID, ticketID uuid.UUID) (*ticket.Ticket, error)
	Create(ctx context.Context, orgID uuid.UUID, t *ticket.Ticket, audit *audit.AuditLog) (*ticket.Ticket, error)
	UpdateState(ctx context.Context, orgID, ticketID uuid.UUID, to string, note *string, audit *audit.AuditLog) (*ticket.Ticket, error)
	...
}
```
Direction port (planner confirms exact methods; recommended):
- `Create(ctx, orgID, d *direction.Direction, supersedesID *uuid.UUID, audits []*audit.AuditLog) (*direction.Direction, error)` — supersede-on-create in one tx (D-13-08)
- `Activate(ctx, orgID, id uuid.UUID, audit *audit.AuditLog) (*direction.Direction, error)`
- `Cancel(ctx, orgID, id uuid.UUID, reason string, audit *audit.AuditLog) (*direction.Direction, error)`
- `Claim(ctx, orgID, wgRowID, claimantID uuid.UUID, estHours float64, audit *audit.AuditLog) (*direction.Direction, error)` — the Σ guard under FOR UPDATE (D-13-13)
- `ListPlan(ctx, orgID, employeeID *uuid.UUID, periodStart, periodEnd time.Time) ([]direction.PlanRow, error)` — read-model + derived states (D-13-27)
- `Coverage(ctx, orgID, scope string, scopeID *uuid.UUID, periodStart, periodEnd time.Time) ([]direction.CoverageRow, error)` — DIR-06
- `Warnings(ctx, orgID, employeeID uuid.UUID, periodStart, periodEnd time.Time) ([]direction.Warning, error)` — DIR-05 (or folded into ListPlan/Coverage)
- `FirstDirectionRefs(ctx, orgID, activityID uuid.UUID) (*direction.DirectionRefs, error)` — the origin-fallback port the activity service consumes (D-13-32/33)

### `internal/core/ports/org_settings_repository.go` (port)

**Analog:** `internal/core/ports/audit_log_repository.go` (19 lines — tiny interface with doc comment) + `organization_repository.go` settings methods

```go
type OrgSettingsRepository interface {
	Get(ctx context.Context, orgID uuid.UUID, key string) (json.RawMessage, error) // nil when absent
	List(ctx context.Context, orgID uuid.UUID) (map[string]json.RawMessage, error)
	Upsert(ctx context.Context, orgID uuid.UUID, key string, value json.RawMessage, audit *audit.AuditLog) error // in-tx audit (D-13-22)
}
```

### `internal/core/services/direction/direction.go` (service — the phase's core)

**Analog:** `internal/core/services/ticket/ticket.go` (478 lines) for the lifecycle + audit-DTO pattern; `internal/core/services/coverage/coverage.go` (505 lines) for gates, Σ cents, routing, read-models.

**Service struct + constructor** (ticket.go lines 28-42; coverage.go lines 46-64):
```go
type Service struct {
	repo         ports.TicketRepository
	activityRepo ports.ActivityRepository
	...
}
func NewService(...) *Service { return &Service{...} }
```
Direction service deps (research DIR-01/03/06): `ports.DirectionRepository`, `ports.ActivityRepository` (same-org, WG scope, terminal CTE reads), `ports.WorkingGroupRepository` (ListMembers — D-13-12), `ports.UnitRepository` (scope resolution + ResolveUnitManager), `ports.OrganizationRepository` (membership validity), `*routing.Service` (BE-014 manager gate).

**Create gate — the D-08 gate shape** (coverage.go lines 353-369 — copy verbatim):
```go
res, err := s.routing.ResolveManagerStage(ctx, e.OrgID, e.ActivityID, e.UnitID, e.UserID)
...
if !res.RoleGated && !contains(res.ApproverIDs, actor) {
	return nil, coverage.ErrForbidden
}
if res.RoleGated && role != string(models.RoleManager) {
	return nil, coverage.ErrForbidden
}
```
For direction creation: self-direction (`directed_to == actor`) always allowed (D-S); manager-directed rows resolve the manager stage on the **WG's anchored activity** for WG rows (OQ8), or the directed activity for user rows.

**Σ fast-fail in cents** (coverage.go lines 340-346 — the claim/coverage arithmetic):
```go
var sum int64
for _, a := range req {
	sum += int64(math.Round(a.Hours * 100))
}
if len(req) == 0 || sum != int64(math.Round(e.Hours*100)) {
	return nil, coverage.ErrAllocationSumMismatch
}
```
Direction uses it for: Σ est_hours > capacity → **soft warning, not rejection** (D-13-03); claim Σ guard pool-level check = fast-fail UX only (CR-01, the repo re-checks under lock).

**Whole-cent rejection** (coverage.go lines 379-383 — est_hours validation, D-13-03):
```go
if a.Hours <= 0 || math.Round(a.Hours*100) != a.Hours*100 {
	return nil, coverage.ErrInvalidRequest
}
```

**Audit DTO built service-side, handed to repo** (ticket.go lines 105-113, 237-245):
```go
actor := actorID
return s.repo.Create(ctx, orgID, t, &audit.AuditLog{
	OrgID:      orgID,
	EntityType: "ticket",
	EntityID:   t.ID,
	Action:     "created",
	ActorID:    &actor,
	Payload:    map[string]any{"kind": req.Kind},
	CreatedAt:  now,
})
```
Direction events: `created` (payload: mode, planned_date, est_hours, supersedes_id), `activated`, `cancelled` (reason), `superseded`, `claimed`, `unclaimed`; settings: `settings-updated` with `{key, before, after}` (D-13-22).

**Permission gate + same-org ref validation** (ticket.go lines 78-89; coverage.go lines 311-325): fetch entry, org compare, role checks, all before repo call. Direction adds the XOR fast-fail (D-13-05), mode gate (D-13-20), WG-membership check (D-13-12), WG-scope check (D-13-17, A5 predicate: activity same-org AND (activity == `wg.SubprojectID` OR anchor in `GetAncestry(activityID)`)), est_hours > 0 (D-13-03).

**Warning computation (DIR-05)** — new pure function in the service; contract from 13-UI-SPEC: `{type, message}` with types `away|partial|over-capacity|invalid`, message pre-rendered server-side (research Common Operation 4). Reads `availability_windows` (declared+confirmed, D-13-29) + membership `valid_from/valid_until` (D-13-31) via the repo.

### `internal/core/services/orgsettings/orgsettings.go` (service, small)

**Analog:** `internal/core/services/organization/organization.go` (112 lines) settings methods + `coverage.go` ClosePeriod manager gate (lines 480-483)

Manager+ gate (D-13-23, from coverage.go lines 481-483):
```go
if role != string(models.RoleManager) {
	return nil, coverage.ErrForbidden
}
```
Get/Put with per-key validation → unknown key 400 (D-13-18); PUT builds the audit row with before/after and hands it to the repo for the in-tx write (D-13-22); membership `planning_mode` override resolution falls back to org default (D-13-19) — a `ResolvePlanningMode(ctx, orgID, employeeID)` helper used by the direction service mode gate.

### `internal/adapters/primary/http/direction_handler.go` (controller)

**Analog:** `internal/adapters/primary/http/coverage_handler.go` (333 lines — copy the skeleton)

**Claims + parse + writeError skeleton** (coverage_handler.go lines 81-97, 306-323):
```go
func (h *CoverageHandler) PutAllocations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)
	entryID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}
	var req ReplaceAllocationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	...
	stored, err := h.service.ReplaceAllocations(ctx, orgID, entryID, allocs, userID.String(), role)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, stored)
}
```
**writeError sentinel map** (coverage_handler.go lines 306-323 — copy; swap domain package):
```go
func (h *CoverageHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, coveragedomain.ErrEntryNotCoverable):
		api.RespondWithError(w, http.StatusNotFound, "entry not coverable")
	case errors.Is(err, coveragedomain.ErrNotFound):
		api.RespondWithError(w, http.StatusNotFound, "not found")
	case errors.Is(err, coveragedomain.ErrInvalidRequest):
		api.RespondWithError(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, coveragedomain.ErrForbidden):
		api.RespondWithError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, coveragedomain.ErrPeriodAlreadyClosed):
		api.RespondWithError(w, http.StatusConflict, "period already closed")
	default:
		api.RespondWithError(w, http.StatusInternalServerError, "internal server error")
	}
}
```
Direction map: `ErrDirectionNotFound`→404, `ErrInvalidRequest`/`ErrInvalidHours`/`ErrUnknownKey`→400, `ErrForbidden`/`ErrNotWgMember`→403, `ErrInvalidTransition`→409, `ErrClaimOverBudget`→409, `ErrCancelReasonRequired`→400, default→500. Optional uuid helper: `derefStr`/`parseOptionalUUID` (coverage_handler.go lines 325-333).

Routes (agent discretion — recommended per research architecture diagram): `POST /direction`, `POST /direction/{id}/activate`, `POST /direction/{id}/cancel`, `POST /direction/claims`, `POST /direction/claims/{id}/cancel`, `GET /direction`, `GET /direction/coverage`. Create response returns row + warnings (D-13-03).

### `internal/adapters/primary/http/org_settings_handler.go` (controller)

**Analog:** `internal/adapters/primary/http/organization.go` lines 100-149 (GetSettings/UpdateSettings) for route shape; `coverage_handler.go` writeError for sentinel mapping

Key difference (D-13-23): **no path param** — org from `middleware.GetOrganizationID(ctx)`:
```go
func (h *OrganizationHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	orgUUID, err := uuid.Parse(r.PathValue("id"))  // ← REMOVE for literal route
	...
}
```
Handler shape: `GET /organizations/settings` returns `map[string]any` of key→value; `PUT /organizations/settings` body `{"planning_daily_hours": 7.5}` — decode into `map[string]json.RawMessage`, one-or-many keys, service validates each (unknown → 400).

### `internal/adapters/secondary/postgres/direction_repository.go` (postgres repo — the correctness layer)

**Analog:** `internal/adapters/secondary/postgres/ticket_repository.go` (861 lines) + `coverage_repository.go` (552 lines)

**Struct + compile-time assertion** (ticket_repository.go lines 27-36):
```go
type TicketRepository struct {
	pool *pgxpool.Pool
}
var _ ports.TicketRepository = (*TicketRepository)(nil)
func NewTicketRepository(pool *pgxpool.Pool) *TicketRepository {
	return &TicketRepository{pool: pool}
}
```

**In-tx audit insert helper** — use the **coverage variant** (caller-controlled entity_type, coverage_repository.go lines 45-70):
```go
func insertCoverageAudit(ctx context.Context, tx pgx.Tx, log *audit.AuditLog) error {
	id := uuid.New()
	var payload any
	if len(log.Payload) > 0 {
		payloadJSON, err := json.Marshal(log.Payload)
		...
		payload = payloadJSON
	}
	var comment any
	if log.Comment != "" { comment = log.Comment }
	_, err := tx.Exec(ctx,
		`INSERT INTO audit_logs (id, org_id, entity_type, entity_id, action, actor_id, comment, payload, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, log.OrgID, log.EntityType, log.EntityID, log.Action, log.ActorID, comment, payload, log.CreatedAt)
	if err != nil { return wrapPGError(err, "insert coverage audit log") }
	return nil
}
```
Copy as `insertDirectionAudit` (entity_type from log.EntityType) — used by create/activate/cancel/claim/supersede.

**Supersede-on-create tx — the CR-01 lock + re-check** (ticket_repository.go lines 239-305 — the Dismiss/UpdateState skeleton; research Common Operation 1):
```go
tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
if err != nil { return nil, fmt.Errorf("begin ticket state update: %w", err) }
defer func() { _ = tx.Rollback(ctx) }()

var currentStatus string
err = tx.QueryRow(ctx,
	`SELECT status FROM tickets WHERE id = $1 AND org_id = $2 FOR UPDATE`,
	ticketID, orgID).Scan(&currentStatus)
if errors.Is(err, pgx.ErrNoRows) { return nil, ticketdomain.ErrTicketNotFound }
...
if !ticketdomain.CanTransition(currentStatus, to) {
	return nil, ticketdomain.ErrInvalidTransition
}
ct, err := tx.Exec(ctx,
	`UPDATE tickets SET status = $1, updated_at = $2 WHERE id = $3 AND org_id = $4 AND status = $5`,
	to, time.Now().UTC(), ticketID, orgID, currentStatus)
if err != nil { return nil, wrapPGError(err, "update ticket status") }
if ct.RowsAffected() == 0 { return nil, ticketdomain.ErrTicketNotFound }
if auditLog != nil {
	if err := insertTicketAudit(ctx, tx, auditLog); err != nil { return nil, err }
}
if err := tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit ticket state update: %w", err) }
return r.Get(ctx, orgID, ticketID)
```
Direction create: lock supersedes target `FOR UPDATE` → re-check status draft|active (D-13-08) → INSERT new row → UPDATE target to `superseded` (status-precondition backstop) → 2 audit rows (`created` + `superseded`) → commit.

**Claim tx with the Σ guard** (coverage_repository.go ReplaceAllocations lines 105-178 — lock + re-check + insert + audit; research Common Operation 2):
```go
var entryHours float64
err = tx.QueryRow(ctx,
	`SELECT hours FROM time_entries
	 WHERE id = $1 AND org_id = $2 AND status = 'approved' AND is_deleted = false
	 FOR UPDATE`, entryID, orgID).Scan(&entryHours)
if errors.Is(err, pgx.ErrNoRows) { return nil, coverage.ErrEntryNotCoverable }
...
var sumCents int64
for _, a := range allocs {
	sumCents += int64(math.Round(a.Hours * 100))
}
if sumCents != int64(math.Round(entryHours*100)) {
	return nil, coverage.ErrAllocationSumMismatch
}
```
Claim: `SELECT est_hours, status FROM direction WHERE id = $1 AND org_id = $2 AND wg_id IS NOT NULL FOR UPDATE` → re-check WG row active + membership + Σ claims (cents) ≤ est_hours → INSERT claim row (directed_by = WG creator, directed_to = claimant, origin_direction_id, est_hours = claimed) → `claimed` audit → commit. Uncapped when est_hours NULL (D-13-14).

**Derived `done` — re-anchored terminal CTE** (ticket_repository.go lines 816-835 — the pool-level + Tx pair pattern; research Common Operation 3):
```go
func (r *TicketRepository) HasNonTerminalActivities(ctx context.Context, ticketID uuid.UUID) (bool, error) {
	var has bool
	err := r.pool.QueryRow(ctx,
		`WITH RECURSIVE subtree AS (
			SELECT id FROM activities WHERE ticket_id = $1 AND origin_type = 'customer_ticket'
			UNION ALL
			SELECT a.id FROM activities a JOIN subtree s ON a.parent_id = s.id
		 )
		 SELECT EXISTS (
			SELECT 1 FROM time_entries te
			WHERE te.is_deleted = false
			  AND te.status IN ('draft','submitted','pending_manager','pending_finance')
			  AND te.activity_id IN (SELECT id FROM subtree)
		 )`,
		ticketID).Scan(&has)
	...
}
```
Direction: anchor `activities.id = $1`; `done` = NOT exists (semantic inversion, D-13-09); `lapsed` = past planned_date/due_date AND no non-deleted entries on subtree (A3 — any status). Keep the `Tx` variant pair (`hasNonTerminalActivitiesTx` pattern, lines 842-861) for in-tx re-checks.

**Read-model queries** — `ToCoverQueue` (coverage_repository.go lines 215-252) shows the read-model shape: LEFT JOIN + COALESCE(SUM) + GROUP BY + `roundCents` (lines 297-299). Coverage read-model: per-day Σ planned (`status IN ('draft','active')` AND `planned_date = day` AND `directed_to = employee`) vs capacity (D-13-24: `planning_daily_hours` − absence hours; full absence `hours IS NULL` → 0 → `away`, excluded from uncovered; partial → capacity −= hours).

**Row scan helper** (ticket_repository.go lines 48-65): `scanTicketRow` normalizes nullable columns — direction needs the same for `planned_date/est_hours/priority/due_date/supersedes_id/origin_direction_id/reason` (5+ nullable columns; scan into locals then assign).

### `internal/adapters/secondary/postgres/org_settings_repository.go` (postgres repo, small)

**Analog:** `coverage_repository.go` structure (struct + assertion + in-tx audit helper + single-tx methods). Upsert shape:
```go
tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
defer func() { _ = tx.Rollback(ctx) }()
_, err = tx.Exec(ctx,
	`INSERT INTO org_settings (org_id, key, value, updated_at)
	 VALUES ($1, $2, $3, $4)
	 ON CONFLICT (org_id, key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at`,
	orgID, key, valueJSON, time.Now().UTC())
// + insertOrgSettingsAudit(ctx, tx, auditLog) in the SAME tx (D-13-22)
if err := tx.Commit(ctx); err != nil { ... }
```
Wrap with `wrapPGError(err, "upsert org setting")`.

### Migrations

**`021_direction_rows.up.sql`** — analog `014_ticket_schema.up.sql` (table + vocab CHECK + index block) and `015_activity_origins.up.sql` (XOR-style CHECK with explicit `IS [NOT] NULL`). Research Pattern 1 has the full recommended DDL (lines 246-284) — CHECKs to copy the *form* of:
- status vocab: `CHECK (status IN ('draft','active','superseded','cancelled'))` — form from 014 line 26-28
- XOR target: `CHECK ((directed_to IS NOT NULL AND wg_id IS NULL) OR (directed_to IS NULL AND wg_id IS NOT NULL))` — form from 015 lines 36-47 (**no 3VL guard needed**: new table, no legacy rows — research Pitfall 2)
- queued-only WG: `CHECK (wg_id IS NULL OR planned_date IS NULL)` (D-13-17)
- est_hours: `CHECK (est_hours IS NULL OR est_hours > 0)` + scheduled-required `CHECK (planned_date IS NULL OR est_hours IS NOT NULL)` (D-13-02)
- cancel reason: `CHECK (status <> 'cancelled' OR reason IS NOT NULL)` (D-13-10)
- `hours DECIMAL(8,2)` mirrors `time_entries.hours` (000:278)
- indexes: `(org_id, directed_to, planned_date)`, `(org_id, wg_id)`, `(activity_id, created_at)` (origin fallback, D-13-33), `(supersedes_id)`
- `.down.sql`: `DROP TABLE IF EXISTS direction CASCADE;` (form from `019_coverage_allocations.down.sql`)

**`022_org_settings.up.sql`** — analog `012_staffing_schema.up.sql` (CREATE TABLE + ALTER TABLE ADD COLUMN block):
- `org_settings(org_id UUID REFERENCES organizations(id), key VARCHAR(50), value JSONB NOT NULL, updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), PRIMARY KEY (org_id, key))` (D-13-18)
- `ALTER TABLE organization_memberships ADD COLUMN planning_mode VARCHAR(20);` (D-13-19, nullable override — form from 012 lines 39-42; vocabulary CHECK optional per research "planner call")
- `.down.sql`: drop column first, then table (research: "Drop column then table")

### Test files

**Migration cycle tests** — `internal/adapters/secondary/postgres/direction_ontology_migrations_test.go`. Analog: `coverage_ontology_migrations_test.go` (287 lines). Skeleton per test:
```go
func TestMigration021_DirectionRows_UpDownUpCycle(t *testing.T) {
	pool := TestPool(t)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })
	ctx := context.Background()
	now := time.Now()
	up021 := readMigration(t, "021_direction_rows.up.sql")
	down021 := readMigration(t, "021_direction_rows.down.sql")
	applyMigrations(t, pool, true, "021_direction_rows.up.sql", "022_org_settings.up.sql") // pre-state skipping the migration under test
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	// --- UP: apply + CHECK assertions (XOR, queued-only, est_hours, cancel reason — 
	//   insert violating rows and assert 23514 + constraint name, form at lines 161-171) ---
	// --- DOWN: assertTableExists(..., false) ---
	// --- UP again: green ---
}
```
Helpers available: `readMigration`, `applyMigrations`, `assertTableExists`, `assertConstraintExists`, `seedOrg`, `seedUser`, `seedActivity`, `seedUnit` (all in exported_test_helpers.go / package test files). Self-seed pre-state inline — no demo data (Phase 11 rule).

**Repo integration tests** — `direction_repository_test.go`. Analog: `ticket_repository_test.go`. The concurrent-claim test mirrors the CR-01 battery (ticket_repository_test.go lines 414-509): start channel + buffered results channel + deterministic outcome set; seeding helpers like `insertTicketWithStatus` (lines 428-437). `SetupTestSchema(t, pool)` + `t.Cleanup(TeardownTestSchema)`; repo := `NewDirectionRepository(pool)`.

**Handler tests** — `direction_handler_test.go` / `org_settings_handler_test.go`. Analog: `coverage_handler_test.go` using `newHandlerFixture(t, pool)` (handler_test_helper.go lines 48-239) + `registerAndLogin`. The literal-route test asserts `GET /organizations/settings` hits the new handler while `GET /organizations/{uuid}/settings` still returns typed settings (Pitfall 6).

**Unit tests** — `services/direction/direction_test.go`, `services/orgsettings/orgsettings_test.go` with `testdata.MockDirectionRepo`/`MockOrgSettingsRepo`. Mock pattern (mock_ticket_repo.go lines 14-51): struct + mutex + maps + `GetFn` override + `Audits` capture + compile-time assertion `var _ ports.X = (*MockX)(nil)`.

### MODIFIED files

**`internal/core/services/activity/activity.go`** — origin fallback (D-13-32..34). Service gains a direction-ref dependency; `GetByID` (lines 73-75) and `List` (lines 57-59) currently delegate straight to the repo — wrap them:
```go
func (s *Service) GetByID(ctx context.Context, orgID, activityID uuid.UUID) (*activitydomain.ActivityResponse, error) {
	a, err := s.activityRepo.Get(ctx, orgID, activityID)
	if err != nil { return nil, err }
	if a.OriginType == nil {  // D-13-32 trigger: stored origin absent (A4)
		refs, err := s.directionRepo.FirstDirectionRefs(ctx, orgID, activityID)
		if err != nil { return nil, err }
		if refs != nil { a.AssignedBy, a.AssignedTo = refs.AssignedBy, refs.AssignedTo }  // derived, never written back (D-13-34)
	}
	return a, nil
}
```
`NewService` signature changes (line 43) → `cmd/server/main.go` line 146 + `handler_test_helper.go` line 85 + `testdata/mocks.go` MockActivityRepo constructor (Pitfall 5 — plan as one task).

**`internal/adapters/secondary/postgres/exported_test_helpers.go`** — add to `TeardownTestSchema` tables list (lines 81-119): `direction` and `org_settings` before `working_groups`/`activities`/`organization_memberships` (dependency order, Pitfall 8). Also add a `seedDirectionRow` helper following `seedTimeEntry` (lines 235-244) shape.

**`cmd/server/main.go`** — wire direction + orgsettings services (block after coverage, lines 160-162):
```go
directionRepo := postgres.NewDirectionRepository(pool)
directionService := directionsvc.NewService(directionRepo, activityRepo, wgRepo, unitRepo, orgRepo, routingSvc)
directionHandler := http.NewDirectionHandler(directionService)
orgSettingsRepo := postgres.NewOrgSettingsRepository(pool)
orgSettingsService := orgsettingssvc.NewService(orgSettingsRepo)
orgSettingsHandler := http.NewOrgSettingsHandler(orgSettingsService)
```
Update `activityService := activitysvc.NewService(...)` (line 146) with the direction repo arg. Register routes (lines 242-249 style):
```go
mux.HandleFunc("POST /direction", middleware.Auth(authService, directionHandler.Create))
...
mux.HandleFunc("GET /organizations/settings", middleware.Auth(authService, orgSettingsHandler.Get))   // literal — coexists with /organizations/{id}/settings (line 211) per ServeMux most-specific-wins; proven by POST /organizations/invite (line 209)
mux.HandleFunc("PUT /organizations/settings", middleware.Auth(authService, orgSettingsHandler.Put))
```
**Do NOT remove** the `GET/PUT /organizations/{id}/settings` registrations (lines 211-212) — Pitfall 6.

**`internal/adapters/primary/http/handler_test_helper.go`** — mirror the main.go wiring in `newHandlerFixture` (repos block lines 57-73, services block lines 75-92, routes block after line 222): add direction/orgsettings repos+services+handlers+literal routes; update the activityService constructor (line 85).

---

## Shared Patterns

### Authentication / claims
**Source:** `internal/adapters/primary/http/coverage_handler.go` lines 81-85 + `internal/middleware/middleware.go` lines 67-85
**Apply to:** Both new handlers
```go
ctx := r.Context()
orgID := middleware.GetOrganizationID(ctx)   // JWT-resolved org — settings (D-13-23) and direction both use it; NO org path param
userID := middleware.GetUserID(ctx)
role := middleware.GetRole(ctx)
```

### Permission gating (mode gate + manager reach)
**Source:** `internal/core/services/coverage/coverage.go` lines 350-369 (D-08 gate) + `internal/core/services/routing/routing.go` lines 57-104
**Apply to:** direction create/claim; settings PUT (role-only variant at coverage.go 481-483)
```go
if e.UserID == actor { return nil, coverage.ErrForbidden }                    // structural self-barrier
res, err := s.routing.ResolveManagerStage(ctx, e.OrgID, e.ActivityID, e.UnitID, e.UserID)
if err != nil { ... }
if !res.RoleGated && !contains(res.ApproverIDs, actor) { return nil, coverage.ErrForbidden }
if res.RoleGated && role != string(models.RoleManager) { return nil, coverage.ErrForbidden }
```
Reuse `routing.ResolveManagerStage` — never re-implement (D-G parity).

### State-machine CR-01 closure (fast-fail + in-tx FOR UPDATE re-validation)
**Source:** service `ticket.go` lines 200-246 (pool-level fast-fail comment at 200-203) + repo `ticket_repository.go` lines 239-305
**Apply to:** direction activate/cancel/supersede/claim — every mutator
The pattern is: service checks are UX only; the repo re-locks `FOR UPDATE`, re-checks the matrix/Σ against the locked row, and uses a status-precondition UPDATE backstop. Claim Σ re-runs in-tx under the WG-row lock (CR-01 — Pitfall 1).

### In-tx audit writes (BE-012)
**Source:** `insertCoverageAudit` — `internal/adapters/secondary/postgres/coverage_repository.go` lines 45-70 (caller-controlled entity_type — use this one, not the ticket variant)
**Apply to:** direction mutators (entity_type='direction'), settings upsert (entity_type='org_settings', before/after payload), supersede (2 rows). Never fire-and-forget; rollback on audit failure.

### Sentinel errors + JSONNames
**Source:** `internal/core/domain/coverage/errors.go` (whole file, 38 lines)
**Apply to:** direction + orgsettings domains; handlers map via `errors.Is` switch (coverage_handler.go 306-323).

### Cents arithmetic for Σ comparisons
**Source:** `internal/core/services/coverage/coverage.go` lines 340-346 + `roundCents` (`coverage_repository.go` lines 297-299)
**Apply to:** claim Σ guard, coverage read-model gap math, est_hours validation. DECIMAL in SQL is exact; float64 only renders.

### Terminal-activity CTE (derived done)
**Source:** `ticket_repository.go` lines 816-835 (pool) + 842-861 (Tx pair)
**Apply to:** derived `done` (re-anchor at `activities.id`, invert the EXISTS), `lapsed` (add no-entries predicate, A3).

### Response envelope
**Source:** `pkg/api` — `api.RespondWithJSON(w, code, body)` / `api.RespondWithError(w, code, msg)` (used everywhere, e.g. coverage_handler.go)
**Apply to:** every handler response; create returns row + warnings (D-13-03); coverage returns grouped rows + totals + warnings.

### Migration house rules (ADR-BE-004)
**Source:** `migrations/014`, `015`, `017`, `019` + `coverage_ontology_migrations_test.go`
**Apply to:** 021/022 — append-only numbering, up/down pairs, cycle tests (up→down→up), CHECK vocabularies, explicit `IS [NOT] NULL` on new-table XORs (no 3VL guard needed), `DECIMAL(8,2)` est_hours mirroring time_entries.hours.

### Teardown list
**Source:** `exported_test_helpers.go` lines 79-127
**Apply to:** add `direction`, `org_settings` in dependency order before `working_groups`/`activities`/`organization_memberships`.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `hourglass-vault/decisions/ADR-P-015` + BE encoding ADR | doc | n/a | No code analog — follow the vault template of `hourglass-vault/decisions/backend/ADR-BE-017` (coverage encoding ADR) and the milestone convention; pin status/derived vocab, claim spectrum, audit vocabulary, settings keys, `planning_daily_hours` default, claim lock |

---

## Metadata

**Analog search scope:** `internal/core/domain/`, `internal/core/services/`, `internal/core/ports/`, `internal/adapters/primary/http/`, `internal/adapters/secondary/postgres/`, `internal/middleware/`, `cmd/server/`, `migrations/`, `internal/core/services/testdata/`
**Files scanned:** ~40 (domains, services, repos, handlers, ports, migrations, test helpers, mocks)
**Pattern extraction date:** 2026-08-08
