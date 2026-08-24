# Phase 14: Availability Backend — Absences + Capacity — Pattern Map

**Mapped:** 2026-08-08
**Files analyzed:** 16 new / 9 modified
**Analogs found:** 24 / 25 (ADR revision docs have no code analog)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `migrations/023_availability_status.{up,down}.sql` | migration | batch (DDL) | `migrations/012_staffing_schema.up.sql` (CHECK drop+recreate 48-50) + `021_direction_rows.up.sql` (2VL reason CHECK 57-61) | exact |
| `migrations/024_work_schedules.{up,down}.sql` | migration | batch (DDL) | `migrations/022_org_settings.up.sql` (new table + ALTER memberships) + `012_staffing_schema.up.sql:39-42` | exact |
| `migrations/025_certificate_attachments.{up,down}.sql` | migration | batch (DDL) | `migrations/017_audit_logs.up.sql` (new FK table) | role-match (no BYTEA precedent in repo) |
| `internal/core/domain/availability/availability.go` | domain | CRUD (state machine) | `internal/core/domain/direction/direction.go` (169 lines) | exact |
| `internal/core/domain/availability/errors.go` | domain | n/a | `internal/core/domain/ticket/ticket.go:65-83` (sentinels + JSONNames) | exact |
| `internal/core/ports/availability_repository.go` | port | CRUD + read-model | `internal/core/ports/direction_repository.go` (111 lines) | exact |
| `internal/core/services/availability/availability.go` | service | CRUD + read-model + event-driven (certificate attach) | `internal/core/services/direction/direction.go` (887 lines) + `services/orgsettings/orgsettings.go` (fallback chain 62-95) | exact |
| `internal/adapters/secondary/postgres/availability_repository.go` | repository | CRUD + read-model | `internal/adapters/secondary/postgres/direction_repository.go` (871 lines) | exact |
| `internal/adapters/primary/http/availability_handler.go` | controller | request-response | `internal/adapters/primary/http/direction_handler.go` (323 lines) | exact |
| `internal/core/services/testdata/mock_availability_repo.go` | test (mock) | n/a | `internal/core/services/testdata/mock_direction_repo.go` | exact |
| `internal/adapters/secondary/postgres/availability_ontology_migrations_test.go` | test | batch | `direction_ontology_migrations_test.go` (47-240) | exact |
| `internal/adapters/secondary/postgres/availability_repository_test.go` | test | CRUD (incl. concurrent) | `direction_repository_test.go` + `ticket_repository_test.go` concurrent battery (414-509) | exact |
| `internal/core/services/availability/availability_test.go` | test | CRUD | `internal/core/services/direction/direction_test.go` (mock-driven) | exact |
| `internal/adapters/primary/http/availability_handler_test.go` | test | request-response | `direction_handler_test.go` + `handler_test_helper.go` fixture (50-300) | exact |
| **MOD** `internal/models/models.go` (+ `models_test.go`) | model (extend) | n/a | itself — Role constants (lines 11-25), `IsValid()` (18-25) | exact (self) |
| **MOD** `internal/adapters/secondary/postgres/exported_test_helpers.go` | test helper (extend) | n/a | itself — `TeardownTestSchema` (79-129), `seedDirectionRow` (253-274) | exact (self) |
| **MOD** `internal/adapters/primary/http/handler_test_helper.go` | test helper (extend) | n/a | itself — `newHandlerFixture` (50-300: repos 60-75, services 78-109, routes 124+) | exact (self) |
| **MOD** `cmd/server/main.go` | config (wiring) | n/a | itself — orgsettings block (177-179), direction block (181-187), route block (284-290) | exact (self) |
| **MOD** `internal/adapters/secondary/postgres/direction_repository.go` | repository (D-13-29 closure) | CRUD + read-model | itself — Coverage subqueries (767, 775), AbsenceWindows (812-839) | exact (self) |
| **MOD** `internal/core/ports/direction_repository.go` + `internal/core/domain/direction/direction.go` | port/domain (doc updates) | n/a | itself — port doc (98-102), AbsenceWindow doc (142-153) | exact (self) |
| **MOD** Phase 13 test seeds (`direction_repository_test.go`, `direction_test.go`, `direction_handler_test.go`) | test (behavior change) | n/a | themselves — declared-window seeds flip to confirmed | exact (self) |
| `hourglass-vault/decisions/` ADR-P-008 revision + BE encoding ADR | doc | n/a | no code analog — `hourglass-vault/decisions/backend/ADR-BE-018` as doc template | n/a |

---

## Pattern Assignments

### `internal/core/domain/availability/availability.go` (domain, CRUD state machine)

**Analog:** `internal/core/domain/direction/direction.go` (169 lines — copy the skeleton)

**Entity shape** (direction.go lines 19-36): struct of `uuid.UUID`/`*uuid.UUID`/`time.Time`/`*float64` fields with json tags; `Hours *float64` for partial-day windows (NULL = full day). Window entity: `ID, OrgID, UserID, Kind, StartsOn, EndsOn, Hours *float64, CertificateRef *string, Note *string, Status, RejectionReason *string, CreatedBy, CreatedAt, UpdatedAt`.

**Closed status vocabulary + matrix** (direction.go lines 38-70 — the pinned D-14-08 matrix):
```go
const (
	StatusDeclared   = "declared"
	StatusConfirmed  = "confirmed"
	StatusRejected   = "rejected"   // D-14-08/09: terminal
	StatusWithdrawn  = "withdrawn"  // D-14-08/10: terminal
)
var transitionMatrix = map[string]map[string]bool{
	StatusDeclared: {
		StatusConfirmed: true,
		StatusRejected:  true,
		StatusWithdrawn: true,
	},
}
func CanTransition(from, to string) bool { return transitionMatrix[from][to] }
func IsTerminalStatus(s string) bool     { return s == StatusRejected || s == StatusWithdrawn }
```
Kind vocabulary mirrors the 012 CHECK (`holiday`, `permit`, `medical`, `unavailable`). Note (D-14-02): `medical` **skips the matrix** — auto-confirmed at declare; the matrix governs holiday/permit/unavailable only.

**Audit vocabulary pinned in the domain** (direction.go lines 155-169 — copy verbatim, swap constants):
```go
const (
	AuditEntityWindow = "availability_window"
	AuditActionDeclared           = "declared"
	AuditActionConfirmed          = "confirmed"
	AuditActionRejected           = "rejected"
	AuditActionWithdrawn          = "withdrawn"
	AuditActionEdited             = "edited"              // HR medical edit (D-14-11)
	AuditActionCertificateAttached = "certificate_attached" // D-14-12
)
```
Reject/withdraw audit payloads carry `{reason}` (mirror direction.go:439); HR edits carry `{before, after}` (D-14-12).

### `internal/core/domain/availability/errors.go` (domain, sentinels)

**Analog:** `internal/core/domain/ticket/ticket.go` lines 65-83 (or `direction/errors.go` — copy shape verbatim):
```go
var (
	ErrWindowNotFound       = errors.New("window not found")
	ErrInvalidRequest       = errors.New("invalid request")
	ErrForbidden            = errors.New("forbidden")
	ErrInvalidTransition    = errors.New("invalid window status transition")
	ErrOverlap              = errors.New("overlapping window")   // 409 (D-14-13..15)
	ErrRejectReasonRequired = errors.New("rejection reason required") // 400 (D-14-09)
	ErrInvalidHours         = errors.New("invalid window hours")  // 400 — DECIMAL(4,2) ceiling (Pitfall 6)
	ErrInvalidKind          = errors.New("invalid window kind")
	ErrCertificateRequired  = errors.New("medical window requires certificate ref") // D-14-05
	ErrNotMedical           = errors.New("only medical windows support this operation") // HR edit/attach (D-14-11)
)
var JSONNames = map[error]string{ ... } // house style (ticket.go:74-83)
```

### `internal/core/ports/availability_repository.go` (port)

**Analog:** `internal/core/ports/direction_repository.go` (111 lines) — doc-header style (lines 12-32: "Write shape: every mutator takes its audit row(s) and writes them IN THE SAME TRANSACTION") + mutator signatures taking `*audit.AuditLog` (line 48).

Signature shape — every mutator takes its audit row and writes it in-tx (direction_repository.go:48-78):
```go
type AvailabilityRepository interface {
	Declare(ctx context.Context, orgID uuid.UUID, w *availability.Window, audit *audit.AuditLog) (*availability.Window, error)
	// Declare tx: lock the user's ACTIVE overlapping windows FOR UPDATE → any
	// row → ErrOverlap (CR-01, D-14-15); INSERT; medical → status 'confirmed'
	// immediately (D-14-02) with the second audit row.
	Confirm(ctx context.Context, orgID, id uuid.UUID, audit *audit.AuditLog) (*availability.Window, error)
	Reject(ctx context.Context, orgID, id uuid.UUID, reason string, audit *audit.AuditLog) (*availability.Window, error)
	Withdraw(ctx context.Context, orgID, id uuid.UUID, audit *audit.AuditLog) (*availability.Window, error)
	UpdateMedical(ctx context.Context, orgID, id uuid.UUID, w *availability.Window, audit *audit.AuditLog) (*availability.Window, error) // HR edit
	AttachCertificate(ctx context.Context, orgID, id uuid.UUID, att *availability.Attachment, audit *audit.AuditLog) error
	Windows(ctx context.Context, orgID uuid.UUID, filter WindowsFilter) ([]availability.Window, error)     // org-wide read (D-14-24)
	Capacity(ctx context.Context, orgID uuid.UUID, employeeIDs []uuid.UUID, periodStart, periodEnd time.Time) ([]availability.CapacityRow, error)
	Schedule(ctx context.Context, orgID, employeeID uuid.UUID) (*availability.ScheduleResolution, error) // or resolved service-side from contractTypes + memberships
	// + contract-type CRUD methods (List/Create/Update/Delete — hr-gated service-side)
}
```
Port doc comments must document the D-13-29 closure for the **direction** port — `AbsenceWindows` doc (direction_repository.go:98-102) flips "BOTH statuses per D-13-29" → confirmed-only.

### `internal/core/services/availability/availability.go` (service — the phase's core)

**Analog:** `internal/core/services/direction/direction.go` (887 lines) for orchestration + `internal/core/services/orgsettings/orgsettings.go` (173 lines) for the fallback chain.

**Service struct + constructor** (direction.go lines 68-88): repo deps + shared services — `ports.AvailabilityRepository`, `ports.OrganizationRepository` (membership validity D-14-22 + membership contract_type_id), `ports.ContractTypeRepository` (or folded), `*routing.Service` (ResolveUnitManager for confirm/reject authority — D-14-01), `ports.ActivityRepository` (subtree for workload). NO new routing instance — reuse the one from main.go (D-G parity, main.go:136).

**Fallback chain resolution — the ResolvePlanningMode shape** (orgsettings.go lines 62-95, the capacity schedule resolver D-14-18):
```go
// effective schedule = membership contract_type_id + day-hours override
// → org default contract_type → 8h × Mon–Fri; resolution LEVEL returned
// in the response (D-14-18 "documented in the response").
```
The orgsettings precedent: membership override first (orgsettings.go:67-74), then org-level key (76-90), then hardcoded default (94). Mirror the invalid-stored-value → ErrInvalidValue surfacing (73, 89).

**Role gates** (orgsettings.go lines 126-130 — the manager-gate shape; swap `models.RoleHR`):
```go
if role != string(models.RoleHR) {
	return nil, availabilitydomain.ErrForbidden
}
```
Gates per D-14-26: declare → owner-self or hr; withdraw → owner (declared-only); confirm/reject → resolved unit manager via `s.routing.ResolveUnitManager(ctx, unitID)` (routing.go:109-134 — reuse verbatim, no re-implementation); HR edit + certificate → `models.RoleHR`; reads → any org member with D-14-24 field filtering service-side.

**Confirm authority** (routing.go lines 109-134 — copy the call shape):
```go
managerID, found, err := s.routing.ResolveUnitManager(ctx, unitID)
```
D-14-04 self-confirm: when the actor == resolved managerID, allow (deliberate deviation from entry-approval — document in the service comment). No manager found (`found=false`) → confirm/reject blocked (ErrForbidden).

**Membership validity split** (direction.go lines 602-660 + 773-791 — capacity's D-14-22): resolve scope → per-employee `orgRepo.GetMembership` → drop validity-outside employees from the repo call (membershipValid, lines 777-791 — reuse as-is, do not fork; boundaries inclusive, nil membership invalid).

**Scope → employee universe** (direction.go lines 666-726 — the D-14-20 capacity scope resolution): `resolveScopeEmployees` switch — employee | wg (`wgRepo.ListMembers` 710-716) | unit (ListMembers + GetDescendants 678-699) | org (ListMembers of org). Capacity adds the activity scope: employees with entries on the subtree (D-14-19/20 — planner's call: repo-side DISTINCT user_id or a repo helper).

**Cents arithmetic** (direction.go lines 113-124): `wholeCent` checks `math.Round(h*100) == h*100`; **Pitfall 6**: window hours must be validated against the **99.99** DECIMAL(4,2) ceiling (a `windowHoursValid` helper), not direction's `maxEstHours = 999999.99` (line 116).

**Audit DTO built service-side** (orgsettings.go lines 157-166 — before/after payload shape; ticket.go:237-245 pattern):
```go
actor := actorID
return s.repo.Reject(ctx, orgID, id, reason, &audit.AuditLog{
	OrgID: orgID, EntityType: availabilitydomain.AuditEntityWindow,
	EntityID: id, Action: availabilitydomain.AuditActionRejected,
	ActorID: &actor, Payload: map[string]any{"reason": reason},
	CreatedAt: now,
})
```

### `internal/adapters/secondary/postgres/availability_repository.go` (repo — the correctness layer)

**Analog:** `internal/adapters/secondary/postgres/direction_repository.go` (871 lines).

**Struct + compile-time assertion** (org_settings_repository.go lines 28-37 — copy verbatim):
```go
type AvailabilityRepository struct { pool *pgxpool.Pool }
var _ ports.AvailabilityRepository = (*AvailabilityRepository)(nil)
func NewAvailabilityRepository(pool *pgxpool.Pool) *AvailabilityRepository { return &AvailabilityRepository{pool: pool} }
```

**In-tx audit insert helper** (org_settings_repository.go lines 121-146 — copy as `insertAvailabilityAudit`; caller-controlled entity_type; payload JSONB marshaled; nil actor/comment → SQL NULL):
```go
func insertAvailabilityAudit(ctx context.Context, tx pgx.Tx, log *audit.AuditLog) error {
	id := uuid.New()
	var payload any
	if len(log.Payload) > 0 {
		payloadJSON, err := json.Marshal(log.Payload) // ...
		payload = payloadJSON
	}
	var comment any
	if log.Comment != "" { comment = log.Comment }
	_, err := tx.Exec(ctx,
		`INSERT INTO audit_logs (id, org_id, entity_type, entity_id, action, actor_id, comment, payload, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, log.OrgID, log.EntityType, log.EntityID, log.Action, log.ActorID, comment, payload, log.CreatedAt)
	if err != nil { return wrapPGError(err, "insert availability audit log") }
	return nil
}
```

**Declare tx with the overlap guard — CR-01** (the D-14-15 shape; skeleton from direction_repository.go:164-256):
```go
tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
defer func() { _ = tx.Rollback(ctx) }()
// Authoritative overlap check under the user's ACTIVE windows lock:
var overlapping uuid.UUID
err = tx.QueryRow(ctx,
	`SELECT aw.id FROM availability_windows aw
	  WHERE aw.org_id = $1 AND aw.user_id = $2
	    AND aw.status IN ('declared','confirmed')           -- D-14-13: active-only
	    AND aw.starts_on <= $4::date AND aw.ends_on >= $3::date  -- D-14-14: inclusive range intersection
	  ORDER BY aw.id LIMIT 1 FOR UPDATE`,                   -- serialize concurrent declares
	orgID, userID, startsOn, endsOn).Scan(&overlapping)
if err == nil { return nil, availabilitydomain.ErrOverlap }   // → 409
if !errors.Is(err, pgx.ErrNoRows) { ... }
// INSERT window (medical → status 'confirmed', D-14-02) + 1-2 audit rows
// in-tx + getByIDTx re-read + commit (direction_repository.go:248-255)
```

**In-tx state transition (Confirm/Reject/Withdraw)** — the Activate skeleton (direction_repository.go:265-313) + cancelWithGuard reason check (339-400):
```go
tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
defer func() { _ = tx.Rollback(ctx) }()
var currentStatus string
err = tx.QueryRow(ctx,
	`SELECT status FROM availability_windows WHERE id = $1 AND org_id = $2 FOR UPDATE`,
	id, orgID).Scan(&currentStatus)
if errors.Is(err, pgx.ErrNoRows) { return nil, availabilitydomain.ErrWindowNotFound }
if !availabilitydomain.CanTransition(currentStatus, availabilitydomain.StatusConfirmed) {
	return nil, availabilitydomain.ErrInvalidTransition   // authoritative in-tx matrix re-check
}
ct, err := tx.Exec(ctx,
	`UPDATE availability_windows SET status = 'confirmed', updated_at = $1
	 WHERE id = $2 AND org_id = $3 AND status = $4`,      // status-precondition backstop
	time.Now().UTC(), id, orgID, currentStatus)
if err != nil { return nil, wrapPGError(err, "confirm window") }
if ct.RowsAffected() == 0 { return nil, availabilitydomain.ErrWindowNotFound }
if auditLog != nil { if err := insertAvailabilityAudit(ctx, tx, auditLog); err != nil { return nil, err } }
// re-read in-tx + commit (direction_repository.go:305-312)
```
Reject: reason fast-fail at repo boundary first (`if reason == "" → ErrRejectReasonRequired`, cancelWithGuard lines 340-344) + `SET status='rejected', rejection_reason=$1`; audit payload `{reason}`.

**Coverage/Capacity read-model** (direction_repository.go lines 744-803 — the math shape; the ONLY status change for the D-13-29 closure is at lines 767/775):
```sql
SELECT e.employee_id, d.day,
       CASE WHEN full_abs.day IS NOT NULL THEN 0.0
            ELSE GREATEST(daily.daily_hours - COALESCE(partial_abs.hours, 0.0), 0.0)
       END AS capacity, ...
 FROM unnest($2::uuid[]) AS e(employee_id)
 CROSS JOIN generate_series($3::date, $4::date, '1 day') AS d(day)
 LEFT JOIN (
	SELECT aw.user_id AS employee_id, gs.day, SUM(aw.hours) AS hours
	FROM availability_windows aw
	CROSS JOIN LATERAL generate_series(GREATEST(aw.starts_on, $3::date), LEAST(aw.ends_on, $4::date), '1 day') AS gs(day)
	WHERE aw.org_id = $1 AND aw.user_id = ANY($2)
	  AND aw.status = 'confirmed' AND aw.hours IS NOT NULL     -- D-14-21: was IN ('declared','confirmed')
	GROUP BY aw.user_id, gs.day
 ) partial_abs ON partial_abs.employee_id = e.employee_id AND partial_abs.day = d.day
 -- full_abs twin with hours IS NULL; GREATEST floors at 0 (never negative)
```
Plus `normalizeDay` (723-725) + `roundCents` render (792-796). Capacity replaces `daily_hours` with the per-employee schedule resolution (service computes, or repo joins contract_types — planner discretion); the declared-advisory field (D-14-21) is a second query/subquery over `status = 'declared'`.

**Workload subtree CTE** (direction_repository.go lines 547-566 — re-anchor, status predicate per D-14-19):
```sql
WITH RECURSIVE subtree AS (
	SELECT id FROM activities WHERE id = $1
	UNION ALL
	SELECT a.id FROM activities a JOIN subtree s ON a.parent_id = s.id
)
SELECT te.user_id AS employee_id, SUM(te.hours) AS workload
FROM time_entries te
WHERE te.is_deleted = false
  AND te.status IN ('submitted','approved')        -- D-14-19 literal pin
  AND te.activity_id IN (SELECT id FROM subtree)
GROUP BY te.user_id;
```
**Pitfall 3:** one subtree CTE per scope activity, then join aggregates — never a correlated per-row CTE.

### `internal/adapters/primary/http/availability_handler.go` (controller)

**Analog:** `internal/adapters/primary/http/direction_handler.go` (323 lines — copy the skeleton).

**Claims + parse + writeError skeleton** (direction_handler.go lines 72-93):
```go
func (h *AvailabilityHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil { api.RespondWithError(w, http.StatusBadRequest, "invalid window id"); return }
	row, err := h.service.Confirm(ctx, orgID, userID, role, id)
	if err != nil { h.writeError(w, err); return }
	api.RespondWithJSON(w, http.StatusOK, row)
}
```

**writeError sentinel map** (direction_handler.go lines 304-323 — copy; swap domain package):
```go
func (h *AvailabilityHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, availabilitydomain.ErrWindowNotFound):
		api.RespondWithError(w, http.StatusNotFound, "window not found")
	case errors.Is(err, availabilitydomain.ErrInvalidRequest),
		errors.Is(err, availabilitydomain.ErrInvalidHours),
		errors.Is(err, availabilitydomain.ErrInvalidKind),
		errors.Is(err, availabilitydomain.ErrRejectReasonRequired),
		errors.Is(err, availabilitydomain.ErrCertificateRequired),
		errors.Is(err, availabilitydomain.ErrNotMedical):
		api.RespondWithError(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, availabilitydomain.ErrForbidden):
		api.RespondWithError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, availabilitydomain.ErrInvalidTransition),
		errors.Is(err, availabilitydomain.ErrOverlap):
		api.RespondWithError(w, http.StatusConflict, "conflict")
	default:
		api.RespondWithError(w, http.StatusInternalServerError, "internal server error")
	}
}
```

**parsePeriod** (direction_handler.go lines 283-298 — reuse verbatim for capacity `period_start`/`period_end`; `parseOptionalQueryUUID` 268-278 for optional filters).

**Certificate upload boundary** (expense.go lines 492-511 — the MIME/size gate, deliberately NOT the disk storage):
```go
r.Body = http.MaxBytesReader(w, r.Body, 5<<20)   // 5 MB cap (research OQ3)
if err := r.ParseMultipartForm(5 << 20); err != nil { /* 400 */ }
file, header, err := r.FormFile("file")
ext := strings.ToLower(filepath.Ext(header.Filename))
if ext != ".pdf" && ext != ".jpg" && ext != ".jpeg" && ext != ".png" { /* 400 allowlist */ }
```
**Anti-pattern (locked D-14-07):** do NOT follow expense.go:513-537 (`uploads/` disk dirs + `os.Create`) — certificate bytes go to the DB attachment table; only the MIME/size gate pattern transfers.

### Migrations

**`023_availability_status.{up,down}.sql`** — analog `012_staffing_schema.up.sql:48-50` (CHECK drop+recreate) + `021_direction_rows.up.sql:57-61` (never-NULL-satisfiable reason CHECK):
```sql
-- PostgreSQL cannot alter CHECK constraints in place — drop and recreate (012:46-50).
ALTER TABLE availability_windows DROP CONSTRAINT IF EXISTS availability_windows_status_check;
ALTER TABLE availability_windows ADD CONSTRAINT availability_windows_status_check
    CHECK (status IN ('declared','confirmed','rejected','withdrawn'));       -- D-14-08
ALTER TABLE availability_windows ADD COLUMN rejection_reason TEXT;           -- D-14-08/09
-- 2VL form: never NULL-satisfiable — a rejected row with NULL reason is FALSE OR FALSE (021:57-61).
ALTER TABLE availability_windows ADD CONSTRAINT availability_windows_reject_reason_check
    CHECK (status <> 'rejected' OR rejection_reason IS NOT NULL);
```
Down: drop the new constraint + column, restore the original CHECK with the same name. Header comment notes the migration continues from max (023+) per ADR-BE-004 (012:9-10 lesson) and distinguishes `contract_types` from the `contracts.contract_type` column (research A1).

**`024_work_schedules.{up,down}.sql`** — analog `022_org_settings.up.sql` (small new table + ALTER memberships):
- `contract_types(id, org_id REFERENCES organizations, name, cadence VARCHAR CHECK IN ('week','month'), hours_per_period DECIMAL(5,2) CHECK > 0, day_hours JSONB NOT NULL, is_default BOOLEAN NOT NULL DEFAULT false, created_at, updated_at)` — cadence CHECK + positive-hours CHECK forms from 021:49-53; JSONB matrix validated code-side (022:5-6 comment — "CHECK on JSONB is infeasible by design").
- `ALTER TABLE organization_memberships ADD COLUMN contract_type_id UUID REFERENCES contract_types(id);` (form from 012:39-42 / 022:24) + day-hours override (JSONB column or rows table — planner discretion, research OQ5).
- Down: drop column(s) first, then table (022 down precedent — "Drop column then table", 13-PATTERNS:436).

**`025_certificate_attachments.{up,down}.sql`** — analog `017_audit_logs.up.sql` (FK table):
- `certificate_attachments(id UUID PK, org_id REFERENCES organizations, entity_type VARCHAR CHECK ('availability_window'), entity_id UUID REFERENCES availability_windows(id), content_type VARCHAR(100), size_bytes BIGINT, storage BYTEA, created_by REFERENCES users, created_at)` + index on `(entity_id)` — D-14-07.

### MODIFIED files

**`internal/models/models.go`** (lines 11-25 — the `hr` gap, research Pitfall 8):
```go
const (
	RoleEmployee Role = "employee"
	RoleManager  Role = "manager"
	RoleFinance  Role = "finance"
	RoleCustomer Role = "customer"
	RoleHR       Role = "hr"        // ← ADD (DB CHECK already carries it, 012:50)
)
func (r Role) IsValid() bool {
	switch r {
	case RoleEmployee, RoleManager, RoleFinance, RoleCustomer, RoleHR:  // ← ADD case
		return true
	default: return false
	}
}
```
Plus `models_test.go` validCases. JWT claims need no change (role passes verbatim from membership, auth.go:503).

**`internal/adapters/secondary/postgres/exported_test_helpers.go`** — teardown list (lines 81-121): add `certificate_attachments` BEFORE `availability_windows` (FK), `contract_types` AFTER `organization_memberships` (membership FK → contract_types must drop after); add `seedAvailabilityWindowWithCert` and `seedContractType` helpers following `seedDirectionRow` (lines 253-274) shape. Name the window helper `seedAvailabilityWindowWithCert`, NOT `seedAvailabilityWindow` — the Phase 13 helper of that name already lives in `direction_repository_test.go:121` (package postgres, no certificate_ref param); a same-name addition is a same-package compile error (checker BLOCKER on 14-01 Task 3).

**`cmd/server/main.go`** — wire availability (orgsettings block style, lines 177-187) + register routes (direction block style, lines 284-290):
```go
availabilityRepo := postgres.NewAvailabilityRepository(pool)
availabilityService := availabilitysvc.NewService(availabilityRepo, orgRepo, activityRepo, routingSvc)
availabilityHandler := http.NewAvailabilityHandler(availabilityService)
mux.HandleFunc("POST /availability/windows", middleware.Auth(authService, availabilityHandler.Declare))
mux.HandleFunc("POST /availability/windows/{id}/withdraw", middleware.Auth(authService, availabilityHandler.Withdraw))
mux.HandleFunc("POST /availability/windows/{id}/confirm", middleware.Auth(authService, availabilityHandler.Confirm))
mux.HandleFunc("POST /availability/windows/{id}/reject", middleware.Auth(authService, availabilityHandler.Reject))
mux.HandleFunc("PUT /availability/windows/{id}", middleware.Auth(authService, availabilityHandler.UpdateMedical))
mux.HandleFunc("POST /availability/windows/{id}/certificate", middleware.Auth(authService, availabilityHandler.AttachCertificate))
mux.HandleFunc("GET /availability/windows", middleware.Auth(authService, availabilityHandler.ListWindows))
mux.HandleFunc("GET /availability/capacity", middleware.Auth(authService, availabilityHandler.Capacity))
// + contract-types CRUD routes (D-14-27)
```
`/availability` is a fresh literal prefix — no ServeMux wildcard coexistence issue (unlike /organizations/settings, 13-PATTERNS Pitfall 6).

**`internal/adapters/secondary/postgres/direction_repository.go`** — D-13-29 closure (research Pitfall 5, lock in as ONE explicit task):
- line 767 + 775: `AND aw.status IN ('declared','confirmed')` → `AND aw.status = 'confirmed'` (Coverage partial_abs/full_abs subqueries)
- line 816: same flip in `AbsenceWindows` (812-839)
- port doc (ports/direction_repository.go:98-102) + `AbsenceWindow` doc (direction.go:142-153) rewrite to confirmed-only
- Phase 13 test seeds flip declared → confirmed; add a declared-window subtest asserting NO warning (the behavioral proof of D-14-21)

---

## Shared Patterns

### In-tx re-validation under FOR UPDATE + audit in-tx (CR-01, BE-012/BE-016)
**Source:** `direction_repository.go` lines 164-256 (Create), 265-313 (Activate), 339-400 (cancelWithGuard); audit helper `org_settings_repository.go:121-146`
**Apply to:** Every availability mutator — declare (overlap guard under the user's active windows lock), confirm/reject/withdraw (matrix re-check under the row lock), HR medical edit, certificate attach. Sequence: `BeginTx` → `SELECT ... FOR UPDATE` (org-scoped) → re-check matrix/overlap against LOCKED values → `UPDATE` with the locked status as `WHERE` precondition backstop → audit row(s) in the SAME tx → in-tx re-read → commit. Pool-level service checks are fast-fail UX only.

### State matrix + terminal status (fast-fail + authoritative re-check)
**Source:** `domain/direction/direction.go:47-70`, `domain/ticket/ticket.go:89-119`
**Apply to:** confirm/reject/withdraw service fast-fail + repo in-tx re-check. `rejected`/`withdrawn` terminal; `medical` never enters the matrix (auto-confirm).

### Audit vocabulary pinned in domain
**Source:** `domain/direction/direction.go:155-169`
**Apply to:** `AuditEntityWindow = "availability_window"` + actions declared/confirmed/rejected/withdrawn/edited/certificate_attached. Exported so repo/service can never drift. **Never** reuse direction's `cancelled`/`superseded` vocabulary.

### Fallback-chain resolution
**Source:** `orgsettings.go:62-95` (membership override → org key → hardcoded default)
**Apply to:** capacity schedule resolution (override → contract_type → org default → 8h×Mon–Fri, D-14-18); resolution level returned in the response.

### Sentinel → HTTP map
**Source:** `direction_handler.go:304-323`
**Apply to:** availability handler; 404/400/403/409/500, never 500 for client input.

### Cents arithmetic + DECIMAL ceilings
**Source:** `direction.go:113-124` (wholeCent), `coverage_repository.go` roundCents, `direction_repository.go:792-796`
**Apply to:** window hours validated against the **99.99** ceiling (Pitfall 6); capacity subtraction/rendering via `roundCents`.

### Membership validity filter
**Source:** `direction.go:773-791` (`membershipValid`)
**Apply to:** capacity employee-universe (D-14-22) — reuse as-is, do not fork (D-G parity).

### Terminal-activity recursive CTE
**Source:** `direction_repository.go:547-566`
**Apply to:** workload aggregation (D-14-19) — re-anchor, status `IN ('submitted','approved')`, grouped per employee.

### Response envelope
**Source:** `pkg/api` — `api.RespondWithJSON` / `api.RespondWithError` (used throughout direction_handler.go)
**Apply to:** every availability response; capacity returns rows + totals + declared-advisory + schedule resolution level; windows list filters `certificate_ref` + docs server-side (D-14-24).

### Migration house rules (ADR-BE-004)
**Source:** `012:46-50` (CHECK drop+recreate), `021:57-61` (2VL reason), `022` (small additive + memberships ALTER), cycle test template `direction_ontology_migrations_test.go:47-240`
**Apply to:** 023/024/025 — append-only numbering from 023, up/down pairs, up/down/up cycle tests with 23514 + constraint-name assertions.

### Teardown list
**Source:** `exported_test_helpers.go:79-129`
**Apply to:** add `certificate_attachments` before `availability_windows`, `contract_types` after `organization_memberships` (FK dependency order).

---

## Anti-Patterns to Avoid

1. **NULL three-valued logic in the reject-reason CHECK** — a CHECK like `CHECK (status = 'rejected' AND rejection_reason IS NOT NULL)` silently ACCEPTS non-rejected rows with NULL reason when written as other NULL-bearing shapes. Use the never-NULL-satisfiable form `CHECK (status <> 'rejected' OR rejection_reason IS NOT NULL)` (021:57-61) and assert with 23514 + constraint name (direction_ontology_migrations_test.go:146-157).
2. **Vocabulary drift** — no `'cancelled'`/`'superseded'` anywhere in the availability package: windows *withdraw*, they don't cancel (research Pitfall 2). Warning sign: any literal `'cancelled'` or `'superseded'` in availability code.
3. **Constraint name / migration-number collisions** — `availability_windows_status_check` already exists (012:27) — 023 must DROP IF EXISTS before re-ADD (012:48). Migration files must start at **023** (latest is 022; the Phase 11 A8 lesson — 011 was taken, 012 was named 012).
4. **`hr` role as string literal** — `models.RoleHR` + `IsValid()` case + `models_test.go` validCases in the same plan as the first role gate (research Pitfall 8). Warning sign: `"hr"` raw literal in service code.
5. **Teardown ordering** — `contract_types` must drop AFTER `organization_memberships`; `certificate_attachments` BEFORE `availability_windows` (FKs). Skipping the teardown edit flakes testcontainers suites across packages (research Pitfall 7).
6. **D-13-29 closure without test updates** — flipping direction_repository.go:767/775/816 to confirmed-only changes Phase 13 behavior; the plan MUST own the Phase 13 test-seed flips (declared → confirmed) or the direction suites go red (research Pitfall 5).
7. **Correlated per-row CTE in the capacity query** — employees×days×subtree-depth blowup. One `WITH RECURSIVE` subtree per scope activity, grouped per employee, joined; bounded period (parsePeriod); reuse `idx_availability_windows_org_user_dates` (012:33-34) by filtering `org_id + user_id = ANY($2)` before the date predicate (research Pitfall 3).
8. **Copying direction's `wholeCent`/`maxEstHours` unmodified** — windows are DECIMAL(4,2) → max 99.99; hours = 100.00 must 400, never reach PG 22003 (research Pitfall 6).
9. **Overlap predicate/race mistakes** — inclusive dates (`starts_on <= new_end AND ends_on >= new_start` — a window ending the day before does NOT overlap), active-only statuses (`declared`+`confirmed`), and FOR UPDATE in-tx or two concurrent declares both commit (TOCTOU) (research Pitfall 4). Concurrency battery test required.
10. **Expense disk-storage copy** — expense.go:513-537 writes to `uploads/` on disk; D-14-07 locks DB-backed BYTEA storage. Only the MIME/size gate (expense.go:492-511) transfers.
11. **Re-implementing manager resolution** — confirm/reject authority is `routing.ResolveUnitManager` (routing.go:109-134), shared instance from main.go (D-G parity — no second instance, no fork).

---

## Test Patterns

### Migration cycle tests (023/024/025)
**Analog:** `direction_ontology_migrations_test.go:47-240` (TestMigration021/022 template):
```go
func TestMigration023_AvailabilityStatus_UpDownUpCycle(t *testing.T) {
	pool := TestPool(t)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })
	ctx := context.Background()
	now := time.Now()
	up := readMigration(t, "023_availability_status.up.sql")
	down := readMigration(t, "023_availability_status.down.sql")
	applyMigrations(t, pool, true, "023_availability_status.up.sql") // pre-state skipping the migration under test
	orgID := seedOrg(t, pool, now); userID := seedUser(t, pool, now)
	// UP: apply → assertConstraintExists for the NEW constraint names
	//   (availability_windows_reject_reason_check), functional probes:
	//   INSERT status='rejected' WITH NULL rejection_reason → 23514 +
	//   constraint name (lines 146-157 shape); status='withdrawn' passes.
	// DOWN: assertTableExists(..., false); original status CHECK restored
	//   (probe: status='rejected' rejected again by the OLD constraint).
	// UP again: green.
}
```
Helpers available: `TestPool`, `TeardownTestSchema`, `readMigration`, `applyMigrations`, `assertTableExists`, `assertConstraintExists`, `assertPrimaryKey` (lines 27-36), `seedOrg`, `seedUser`, `seedActivity`, `seedUnit`. 023's cycle additionally asserts the DOWN restores the original two-value CHECK. 024 asserts cadence/hours CHECKs + membership column nullability (022 test lines 204-208 shape). 025 asserts table + FK + index.

### Repo integration tests
**Analog:** `direction_repository_test.go` + the concurrent battery skeleton (`ticket_repository_test.go:414-509`): `SetupTestSchema(t, pool)` + `t.Cleanup(TeardownTestSchema)`; repo := `NewAvailabilityRepository(pool)`. Required batteries: declare happy-path (each kind, medical auto-confirm + certificate_ref requirement, D-14-05); overlap rejections (active-only D-14-13, inclusive edges D-14-14); **concurrent overlapping declares → exactly one succeeds** (start channel + buffered results channel + deterministic outcome set, ticket_repository_test.go shape); confirm/reject/withdraw matrix + terminal rejection + reason; HR edit + attach; audit rows written in-tx (failed audit rolls back the state write); capacity math (fallback chain levels, confirmed-only subtraction, declared advisory, validity exclusion, workload Σ). Seed via `seedAvailabilityWindowWithCert`/`seedContractType` (new helpers).

### Handler integration tests
**Analog:** `direction_handler_test.go` via `newHandlerFixture(t, pool)` (handler_test_helper.go:50-300 — repos block 60-75, services block 78-109, routes block 124+) + `registerAndLogin`. Permission matrix per D-14-26: non-owner withdraw 403, non-manager confirm 403, non-hr certificate 403, cross-org id → 404 (no existence oracle), reject-without-reason 400, overlap 409, self-confirm allowed (D-14-04). Windows read field filtering: `certificate_ref` present for hr + unit manager, absent for other members (D-14-24). Fixture wiring mirrors main.go — add availability repos/services/handlers/routes to `newHandlerFixture` in the same plan task as the main.go wiring (compile-forced).

### Unit tests
**Analog:** `services/direction/direction_test.go` with `testdata.MockAvailabilityRepo` (mock pattern: struct + mutex + maps + `GetFn` override + `Audits` capture + compile-time assertion `var _ ports.AvailabilityRepository = (*MockAvailabilityRepo)(nil)` — mock_direction_repo.go shape). Cover: matrix fast-fails, role gates (including hr), fallback-chain resolution levels (override → type → default → 8×5), membershipValid split, overlap fast-fail, declare validation (kind/date-range/hours ceiling/certificate_ref).

### Model test
`models_test.go` validCases: add `models.RoleHR` to the valid role list (research Wave 0 gap — the only auth change).

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `hourglass-vault/decisions/` ADR-P-008 revision + BE encoding ADR | doc | n/a | No code analog — follow `hourglass-vault/decisions/backend/ADR-BE-018` as the vault template; pin the window status vocab + matrix, rejection_reason 2VL CHECK, audit vocabulary (declared/confirmed/rejected/withdrawn/edited/certificate_attached), contract_types + override model, attachment table + GDPR special-category flag (D-14-06), D-1a simplification (D-14-01) |
| `migrations/025_certificate_attachments` | migration | batch (DDL) | No BYTEA column precedent in the repo — use `017_audit_logs.up.sql` FK-table form + standard PG BYTEA; size cap enforced at the handler (expense.go:492-511 gate), not in SQL |

---

## Metadata

**Analog search scope:** `internal/core/domain/`, `internal/core/ports/`, `internal/core/services/`, `internal/adapters/primary/http/`, `internal/adapters/secondary/postgres/`, `internal/models/`, `cmd/server/`, `migrations/`
**Files scanned:** ~20 (domains, ports, services, repos, handlers, migrations, test helpers, fixtures)
**Pattern extraction date:** 2026-08-08
