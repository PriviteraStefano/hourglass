package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	directiondomain "github.com/stefanoprivitera/hourglass/internal/core/domain/direction"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Shared helpers (direction plane integration tests)
// ---------------------------------------------------------------------------

// directionAudit builds the audit row the service would pass to a mutator:
// entity_type pinned to the direction vocabulary (ADR-BE-018 §3), entity_id =
// the direction row id the event addresses.
func directionAudit(orgID, entityID, actorID uuid.UUID, action string, payload map[string]any, now time.Time) *audit.AuditLog {
	return &audit.AuditLog{
		OrgID:      orgID,
		EntityType: directiondomain.AuditEntityDirection,
		EntityID:   entityID,
		Action:     action,
		ActorID:    &actorID,
		Payload:    payload,
		CreatedAt:  now,
	}
}

// seedWgMember adds a member to a working group (wg_members — migration 000
// table: wg_id + user_id UNIQUE, unit_id required).
func seedWgMember(t *testing.T, pool *pgxpool.Pool, wgID, userID, unitID uuid.UUID) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO wg_members (id, wg_id, user_id, unit_id) VALUES ($1, $2, $3, $4)`,
		uuid.New(), wgID, userID, unitID)
	require.NoError(t, err)
}

// seedDirectionRowCancelled seeds a cancelled direction row with its mandatory
// reason (direction_cancel_reason_check) — seedDirectionRow cannot produce one
// (no reason column).
func seedDirectionRowCancelled(t *testing.T, pool *pgxpool.Pool, orgID, directedBy uuid.UUID, directedTo *uuid.UUID, activityID uuid.UUID, reason string, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO direction (id, org_id, directed_by, directed_to, wg_id, activity_id,
			planned_date, est_hours, priority, due_date, status, supersedes_id, origin_direction_id, reason, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NULL, $5, NULL, NULL, NULL, NULL, 'cancelled', NULL, NULL, $6, $7, $7)`,
		id, orgID, directedBy, directedTo, activityID, reason, now)
	require.NoError(t, err)
	return id
}

// seedClaimRow seeds a claim row directly (origin_direction_id set,
// user-targeted, queued, draft) — the shape Claim produces (D-13-11), used
// by the supersede-chain tests before the Claim method exists.
func seedClaimRow(t *testing.T, pool *pgxpool.Pool, orgID, directedBy, claimantID uuid.UUID, wgRowID, activityID uuid.UUID, estHours float64, priority *int, dueDate *time.Time, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO direction (id, org_id, directed_by, directed_to, wg_id, activity_id,
			planned_date, est_hours, priority, due_date, status, supersedes_id, origin_direction_id, reason, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NULL, $5, NULL, $6, $7, $8, 'draft', NULL, $9, NULL, $10, $10)`,
		id, orgID, directedBy, claimantID, activityID, estHours, priority, dueDate, wgRowID, now)
	require.NoError(t, err)
	return id
}

// claimSum returns the Σ of claim rows for a WG row under the pinned budget
// predicate: origin_direction_id = wgRowID AND status IN ('draft','active')
// (ADR-BE-018 §5 — superseded/cancelled claim rows never consume budget).
func claimSum(t *testing.T, pool *pgxpool.Pool, wgRowID uuid.UUID) float64 {
	t.Helper()
	var sum float64
	err := pool.QueryRow(context.Background(),
		`SELECT COALESCE(SUM(est_hours), 0) FROM direction
		 WHERE origin_direction_id = $1 AND status IN ('draft','active')`,
		wgRowID).Scan(&sum)
	require.NoError(t, err)
	return sum
}

// countDirectionAudits counts audit rows for (entity_id, action) with the
// direction entity_type (ADR-BE-018 §3).
func countDirectionAudits(t *testing.T, pool *pgxpool.Pool, entityID uuid.UUID, action string) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'direction' AND entity_id = $1 AND action = $2`,
		entityID, action).Scan(&n)
	require.NoError(t, err)
	return n
}

// cancelReasonPayload reads the latest 'cancelled' audit payload for a row
// (the reason payload, D-13-10).
func cancelReasonPayload(t *testing.T, pool *pgxpool.Pool, entityID uuid.UUID) map[string]any {
	t.Helper()
	var payloadJSON []byte
	err := pool.QueryRow(context.Background(),
		`SELECT payload FROM audit_logs
		 WHERE entity_type = 'direction' AND entity_id = $1 AND action = 'cancelled'
		 ORDER BY created_at DESC LIMIT 1`,
		entityID).Scan(&payloadJSON)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(payloadJSON, &payload))
	return payload
}

// seedAvailabilityWindow inserts an availability_windows row (migration 012
// schema: the employee column is user_id; hours NULL = full absence; status
// declared|confirmed).
func seedAvailabilityWindow(t *testing.T, pool *pgxpool.Pool, orgID, userID uuid.UUID, kind string, startsOn, endsOn time.Time, hours *float64, status string, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO availability_windows (id, org_id, user_id, kind, starts_on, ends_on, hours, status, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $3, $9)`,
		id, orgID, userID, kind, startsOn, endsOn, hours, status, now)
	require.NoError(t, err)
	return id
}

// seedOrgSettings upserts an org_settings row (migration 022) — e.g.
// planning_daily_hours for the coverage capacity denominator (D-13-24).
func seedOrgSettings(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, key string, value float64, now time.Time) {
	t.Helper()
	v, err := json.Marshal(value)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(),
		`INSERT INTO org_settings (org_id, key, value, updated_at) VALUES ($1, $2, $3::jsonb, $4)`,
		orgID, key, string(v), now)
	require.NoError(t, err)
}

// planRowIDs extracts the row ids in ListPlan order — the compact assertion
// shape for ordering/selection tests.
func planRowIDs(rows []directiondomain.PlanRow) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids
}

// planRowByID indexes ListPlan rows for per-row derived-state assertions.
func planRowByID(rows []directiondomain.PlanRow) map[uuid.UUID]directiondomain.PlanRow {
	m := make(map[uuid.UUID]directiondomain.PlanRow, len(rows))
	for _, r := range rows {
		m[r.ID] = r
	}
	return m
}

// ---------------------------------------------------------------------------
// Task 1 — Create (plain + supersede-on-create tx)
// ---------------------------------------------------------------------------

// TestDirectionRepository_Create proves the plain create path: the row is
// stored and returned with the passed status (default draft), and the
// 'created' audit row lands in the SAME tx (BE-012).
func TestDirectionRepository_Create(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	planned := now.AddDate(0, 0, 1)
	d := &directiondomain.Direction{
		ID:          uuid.New(),
		DirectedBy:  managerID,
		DirectedTo:  &employeeID,
		ActivityID:  activityID,
		PlannedDate: &planned,
		EstHours:    ptr(8.0),
	}

	created, err := repo.Create(ctx, orgID, d, nil,
		[]*audit.AuditLog{directionAudit(orgID, d.ID, managerID, directiondomain.AuditActionCreated, nil, now)})
	require.NoError(t, err)
	require.Equal(t, d.ID, created.ID)
	require.Equal(t, directiondomain.StatusDraft, created.Status)
	require.Equal(t, employeeID, *created.DirectedTo)
	require.Nil(t, created.WgID)
	require.Nil(t, created.SupersedesID)
	require.Nil(t, created.OriginDirectionID)
	require.Equal(t, 8.0, *created.EstHours)

	require.Equal(t, 1, countDirectionAudits(t, pool, d.ID, directiondomain.AuditActionCreated),
		"the created audit row must land in the same tx")
}

// TestDirectionRepository_Create_Supersede proves supersede-on-create in one
// tx: the target flips to superseded, the new row's supersedes_id points at
// the target, and BOTH audit rows (created + superseded) exist (D-13-08,
// BE-012).
func TestDirectionRepository_Create_Supersede(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	targetID := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "draft", now)

	d := &directiondomain.Direction{
		ID:         uuid.New(),
		DirectedBy: managerID,
		DirectedTo: &employeeID,
		ActivityID: activityID,
		EstHours:   ptr(6.0),
	}
	created, err := repo.Create(ctx, orgID, d, &targetID, []*audit.AuditLog{
		directionAudit(orgID, d.ID, managerID, directiondomain.AuditActionCreated, nil, now),
		directionAudit(orgID, targetID, managerID, directiondomain.AuditActionSuperseded, nil, now),
	})
	require.NoError(t, err)

	// The new row's supersedes_id points at the target.
	require.NotNil(t, created.SupersedesID)
	require.Equal(t, targetID, *created.SupersedesID)

	// The target flipped to superseded in the same tx.
	var targetStatus string
	err = pool.QueryRow(ctx, `SELECT status FROM direction WHERE id = $1`, targetID).Scan(&targetStatus)
	require.NoError(t, err)
	require.Equal(t, directiondomain.StatusSuperseded, targetStatus)

	// Both audit rows exist.
	require.Equal(t, 1, countDirectionAudits(t, pool, d.ID, directiondomain.AuditActionCreated))
	require.Equal(t, 1, countDirectionAudits(t, pool, targetID, directiondomain.AuditActionSuperseded))
}

// TestDirectionRepository_Create_SupersedeChainRewrite proves the idempotency
// edge (probe DIR-01, Pitfall 4): a second create-with-supersedes targeting an
// ALREADY superseded row fails in-tx with ErrInvalidTransition — the chain is
// append-only, never rewritten.
func TestDirectionRepository_Create_SupersedeChainRewrite(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	targetID := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "draft", now)

	// First supersede lands.
	_, err := repo.Create(ctx, orgID, &directiondomain.Direction{
		ID: uuid.New(), DirectedBy: managerID, DirectedTo: &employeeID,
		ActivityID: activityID, EstHours: ptr(6.0),
	}, &targetID, []*audit.AuditLog{
		directionAudit(orgID, uuid.New(), managerID, directiondomain.AuditActionCreated, nil, now),
		directionAudit(orgID, targetID, managerID, directiondomain.AuditActionSuperseded, nil, now),
	})
	require.NoError(t, err)

	// Rewriting the chain (superseding the superseded row) is rejected in-tx.
	_, err = repo.Create(ctx, orgID, &directiondomain.Direction{
		ID: uuid.New(), DirectedBy: managerID, DirectedTo: &employeeID,
		ActivityID: activityID, EstHours: ptr(6.0),
	}, &targetID, []*audit.AuditLog{
		directionAudit(orgID, uuid.New(), managerID, directiondomain.AuditActionCreated, nil, now),
		directionAudit(orgID, targetID, managerID, directiondomain.AuditActionSuperseded, nil, now),
	})
	require.ErrorIs(t, err, directiondomain.ErrInvalidTransition,
		"a superseded row is terminal — the chain must never be rewritten")
}

// TestDirectionRepository_Create_SupersedeCancelled proves a cancelled target
// is not supersedable (terminal, Pitfall 4).
func TestDirectionRepository_Create_SupersedeCancelled(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	targetID := seedDirectionRowCancelled(t, pool, orgID, managerID, &employeeID, activityID, "no longer needed", now)

	_, err := repo.Create(ctx, orgID, &directiondomain.Direction{
		ID: uuid.New(), DirectedBy: managerID, DirectedTo: &employeeID,
		ActivityID: activityID, EstHours: ptr(6.0),
	}, &targetID, []*audit.AuditLog{
		directionAudit(orgID, uuid.New(), managerID, directiondomain.AuditActionCreated, nil, now),
		directionAudit(orgID, targetID, managerID, directiondomain.AuditActionSuperseded, nil, now),
	})
	require.ErrorIs(t, err, directiondomain.ErrInvalidTransition,
		"a cancelled row is terminal — never supersedable")
}

// TestDirectionRepository_Create_SupersedeCrossOrg proves the org-scoped lock
// (T-13-18): a supersedes target in ANOTHER org reads as ErrDirectionNotFound
// — no existence oracle.
func TestDirectionRepository_Create_SupersedeCrossOrg(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	otherOrgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	// The target lives in the OTHER org.
	targetID := seedDirectionRow(t, pool, otherOrgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "draft", now)

	_, err := repo.Create(ctx, orgID, &directiondomain.Direction{
		ID: uuid.New(), DirectedBy: managerID, DirectedTo: &employeeID,
		ActivityID: activityID, EstHours: ptr(6.0),
	}, &targetID, []*audit.AuditLog{
		directionAudit(orgID, uuid.New(), managerID, directiondomain.AuditActionCreated, nil, now),
		directionAudit(orgID, targetID, managerID, directiondomain.AuditActionSuperseded, nil, now),
	})
	require.ErrorIs(t, err, directiondomain.ErrDirectionNotFound,
		"cross-org supersede targets must read as not-found (org-scoped lock)")
}

// TestDirectionRepository_Create_SupersedeClaimRow proves the claim-row
// supersede pin (ADR-BE-018 §5): the new row inherits origin_direction_id
// from the claim target, and the WG-budget Σ over draft|active claim rows is
// UNCHANGED by the supersede — the superseded target drops out, the new row
// counts in its place (no strand, no double count).
func TestDirectionRepository_Create_SupersedeClaimRow(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	// WG row with an 8.00 budget + one full claim row against it.
	wgID := seedWorkingGroup(t, pool, orgID, activityID, managerID, now)
	wgRowID := seedDirectionRow(t, pool, orgID, managerID, nil, ptr(wgID), activityID, nil, ptr(8.0), "active", now)
	claimID := seedClaimRow(t, pool, orgID, managerID, employeeID, wgRowID, activityID, 8.0, nil, nil, now)
	require.Equal(t, 8.0, claimSum(t, pool, wgRowID), "pre-state: one draft claim consumes the full budget")

	// Supersede the claim row: the new row inherits the origin.
	d := &directiondomain.Direction{
		ID:         uuid.New(),
		DirectedBy: managerID,
		DirectedTo: &employeeID,
		ActivityID: activityID,
		EstHours:   ptr(8.0),
	}
	created, err := repo.Create(ctx, orgID, d, &claimID, []*audit.AuditLog{
		directionAudit(orgID, d.ID, managerID, directiondomain.AuditActionCreated, nil, now),
		directionAudit(orgID, claimID, managerID, directiondomain.AuditActionSuperseded, nil, now),
	})
	require.NoError(t, err)

	require.NotNil(t, created.OriginDirectionID, "the superseding row must inherit the claim's origin")
	require.Equal(t, wgRowID, *created.OriginDirectionID)

	// The Σ is unchanged: the superseded target dropped out, the new draft
	// row counts in its place (ADR-BE-018 §5).
	require.Equal(t, 8.0, claimSum(t, pool, wgRowID),
		"the supersede must not strand or double-count budget (Σ unchanged)")

	var targetStatus string
	err = pool.QueryRow(ctx, `SELECT status FROM direction WHERE id = $1`, claimID).Scan(&targetStatus)
	require.NoError(t, err)
	require.Equal(t, directiondomain.StatusSuperseded, targetStatus)
}

// TestDirectionRepository_Create_SupersedeClaimRowToWgShape proves the
// shape guard (ADR-BE-018 §5): superseding a claim row into a WG-shaped new
// row (wg_id set) is ErrInvalidTarget — a WG row cannot carry a claim's
// origin.
func TestDirectionRepository_Create_SupersedeClaimRowToWgShape(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	wgID := seedWorkingGroup(t, pool, orgID, activityID, managerID, now)

	wgRowID := seedDirectionRow(t, pool, orgID, managerID, nil, ptr(wgID), activityID, nil, ptr(8.0), "active", now)
	claimID := seedClaimRow(t, pool, orgID, managerID, employeeID, wgRowID, activityID, 8.0, nil, nil, now)

	// A WG-shaped superseding row (wg_id set, directed_to nil).
	_, err := repo.Create(ctx, orgID, &directiondomain.Direction{
		ID: uuid.New(), DirectedBy: managerID, WgID: ptr(wgID),
		ActivityID: activityID, EstHours: ptr(8.0),
	}, &claimID, []*audit.AuditLog{
		directionAudit(orgID, uuid.New(), managerID, directiondomain.AuditActionCreated, nil, now),
		directionAudit(orgID, claimID, managerID, directiondomain.AuditActionSuperseded, nil, now),
	})
	require.ErrorIs(t, err, directiondomain.ErrInvalidTarget,
		"a WG-shaped superseding row cannot carry a claim's origin")
}

// ---------------------------------------------------------------------------
// Task 2 — Activate + Cancel txs (matrix re-validation under lock)
// ---------------------------------------------------------------------------

// TestDirectionRepository_Activate proves draft → active with the 'activated'
// audit row written in the same tx (BE-012).
func TestDirectionRepository_Activate(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	rowID := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "draft", now)

	activated, err := repo.Activate(ctx, orgID, rowID,
		directionAudit(orgID, rowID, managerID, directiondomain.AuditActionActivated, nil, now))
	require.NoError(t, err)
	require.Equal(t, directiondomain.StatusActive, activated.Status)
	require.Equal(t, 1, countDirectionAudits(t, pool, rowID, directiondomain.AuditActionActivated))
}

// TestDirectionRepository_Activate_Twice proves the idempotency edge: a second
// activate on an active row fails in-tx with ErrInvalidTransition (probe
// DIR-01) — no silent double-transition.
func TestDirectionRepository_Activate_Twice(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	rowID := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "active", now)

	_, err := repo.Activate(ctx, orgID, rowID,
		directionAudit(orgID, rowID, managerID, directiondomain.AuditActionActivated, nil, now))
	require.ErrorIs(t, err, directiondomain.ErrInvalidTransition,
		"active → active is not in the matrix")

	var status string
	err = pool.QueryRow(ctx, `SELECT status FROM direction WHERE id = $1`, rowID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, directiondomain.StatusActive, status, "a failed transition must not change the row")
}

// TestDirectionRepository_Activate_Cancelled proves a terminal row rejects the
// activate transition (Pitfall 4).
func TestDirectionRepository_Activate_Cancelled(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	rowID := seedDirectionRowCancelled(t, pool, orgID, managerID, &employeeID, activityID, "no longer needed", now)

	_, err := repo.Activate(ctx, orgID, rowID,
		directionAudit(orgID, rowID, managerID, directiondomain.AuditActionActivated, nil, now))
	require.ErrorIs(t, err, directiondomain.ErrInvalidTransition,
		"a cancelled row is terminal — never activatable")
}

// TestDirectionRepository_Activate_CrossOrg proves the org-scoped lock: an
// activate on another org's row reads as ErrDirectionNotFound (T-13-18).
func TestDirectionRepository_Activate_CrossOrg(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	otherOrgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	rowID := seedDirectionRow(t, pool, otherOrgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "draft", now)

	_, err := repo.Activate(ctx, orgID, rowID,
		directionAudit(orgID, rowID, managerID, directiondomain.AuditActionActivated, nil, now))
	require.ErrorIs(t, err, directiondomain.ErrDirectionNotFound)
}

// TestDirectionRepository_Cancel proves draft → cancelled with a reason: the
// audit row carries the reason payload (D-13-10), the row's reason column is
// persisted, and the return reflects the cancelled state.
func TestDirectionRepository_Cancel(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	rowID := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "draft", now)

	cancelled, err := repo.Cancel(ctx, orgID, rowID, "scope changed",
		directionAudit(orgID, rowID, managerID, directiondomain.AuditActionCancelled, map[string]any{"reason": "scope changed"}, now))
	require.NoError(t, err)
	require.Equal(t, directiondomain.StatusCancelled, cancelled.Status)
	require.NotNil(t, cancelled.Reason)
	require.Equal(t, "scope changed", *cancelled.Reason)

	require.Equal(t, 1, countDirectionAudits(t, pool, rowID, directiondomain.AuditActionCancelled))
	require.Equal(t, "scope changed", cancelReasonPayload(t, pool, rowID)["reason"])
}

// TestDirectionRepository_Cancel_EmptyReason proves the reason requirement
// (D-13-10): an empty reason is rejected at the repo boundary BEFORE any lock
// or write (ErrCancelReasonRequired).
func TestDirectionRepository_Cancel_EmptyReason(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	rowID := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "draft", now)

	_, err := repo.Cancel(ctx, orgID, rowID, "",
		directionAudit(orgID, rowID, managerID, directiondomain.AuditActionCancelled, nil, now))
	require.ErrorIs(t, err, directiondomain.ErrCancelReasonRequired)

	var status string
	err = pool.QueryRow(ctx, `SELECT status FROM direction WHERE id = $1`, rowID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, directiondomain.StatusDraft, status, "a reason-less cancel must not change the row")
	require.Zero(t, countDirectionAudits(t, pool, rowID, directiondomain.AuditActionCancelled))
}

// TestDirectionRepository_Cancel_Twice proves a cancelled row is terminal: a
// second cancel fails in-tx with ErrInvalidTransition (no silent
// double-transition).
func TestDirectionRepository_Cancel_Twice(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	rowID := seedDirectionRowCancelled(t, pool, orgID, managerID, &employeeID, activityID, "first reason", now)

	_, err := repo.Cancel(ctx, orgID, rowID, "second reason",
		directionAudit(orgID, rowID, managerID, directiondomain.AuditActionCancelled, nil, now))
	require.ErrorIs(t, err, directiondomain.ErrInvalidTransition,
		"cancelled → cancelled is not in the matrix")
}

// TestDirectionRepository_Cancel_CrossOrg proves the org-scoped lock: a cancel
// on another org's row reads as ErrDirectionNotFound (T-13-18).
func TestDirectionRepository_Cancel_CrossOrg(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	otherOrgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	rowID := seedDirectionRow(t, pool, otherOrgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "draft", now)

	_, err := repo.Cancel(ctx, orgID, rowID, "scope changed",
		directionAudit(orgID, rowID, managerID, directiondomain.AuditActionCancelled, nil, now))
	require.ErrorIs(t, err, directiondomain.ErrDirectionNotFound)
}

// ---------------------------------------------------------------------------
// Task 3 — Claim tx (WG-row lock + Σ cents guard + membership re-check),
// Unclaim, concurrent battery, supersede-chain hours-return
// ---------------------------------------------------------------------------

// claimAudit builds the 'claimed' audit row the service would pass (payload
// carries the WG row id + claimed hours, ADR-BE-018 §3).
func claimAudit(orgID, claimRowID, actorID uuid.UUID, wgRowID uuid.UUID, estHours float64, now time.Time) *audit.AuditLog {
	return &audit.AuditLog{
		OrgID:      orgID,
		EntityType: directiondomain.AuditEntityDirection,
		EntityID:   claimRowID,
		Action:     directiondomain.AuditActionClaimed,
		ActorID:    &actorID,
		Payload:    map[string]any{"wg_row_id": wgRowID.String(), "est_hours": estHours},
		CreatedAt:  now,
	}
}

// seedWgClaimSetup seeds the WG + active WG row (budget) + a member — the
// claim test pre-state.
func seedWgClaimSetup(t *testing.T, pool *pgxpool.Pool, orgID, managerID, memberID uuid.UUID, activityID uuid.UUID, budget *float64, now time.Time) (wgRowID uuid.UUID) {
	t.Helper()
	wgID := seedWorkingGroup(t, pool, orgID, activityID, managerID, now)
	seedWgMember(t, pool, wgID, memberID, seedUnit(t, pool, orgID, now))
	return seedDirectionRow(t, pool, orgID, managerID, nil, ptr(wgID), activityID, nil, budget, "active", now)
}

// TestDirectionRepository_Claim proves the claim row shape (D-13-11, A8):
// user-targeted, queued, draft, attribution preserved, priority/due_date
// copied from the WG row, origin_direction_id set, 'claimed' audit in-tx,
// and the Σ reflects the claimed hours.
func TestDirectionRepository_Claim(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	memberID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	wgRowID := seedWgClaimSetup(t, pool, orgID, managerID, memberID, activityID, ptr(8.0), now)
	// Give the WG row a priority/due_date to prove the copy (A8).
	_, err := pool.Exec(ctx, `UPDATE direction SET priority = 3, due_date = DATE '2026-08-20' WHERE id = $1`, wgRowID)
	require.NoError(t, err)

	claim, err := repo.Claim(ctx, orgID, wgRowID, memberID, 2.0,
		claimAudit(orgID, uuid.Nil, memberID, wgRowID, 2.0, now))
	require.NoError(t, err)

	// Claim row shape.
	require.Equal(t, directiondomain.StatusDraft, claim.Status)
	require.Equal(t, memberID, *claim.DirectedTo, "directed_to = claimant")
	require.Equal(t, managerID, claim.DirectedBy, "directed_by = the WG row's creator (attribution, D-13-11)")
	require.Nil(t, claim.WgID, "claim rows are user-targeted")
	require.Equal(t, activityID, claim.ActivityID)
	require.Nil(t, claim.PlannedDate, "claim rows land queued (planned_date NULL, A8)")
	require.Equal(t, 2.0, *claim.EstHours)
	require.Equal(t, 3, *claim.Priority, "priority copied from the WG row")
	require.NotNil(t, claim.DueDate, "due_date copied from the WG row")
	require.Equal(t, wgRowID, *claim.OriginDirectionID, "origin_direction_id = the WG row")

	require.Equal(t, 2.0, claimSum(t, pool, wgRowID))
	require.Equal(t, 1, countDirectionAudits(t, pool, claim.ID, directiondomain.AuditActionClaimed))
}

// TestDirectionRepository_Claim_NotWgMember proves the in-tx membership
// re-check (D-13-12): a non-member claimant is rejected authoritatively.
func TestDirectionRepository_Claim_NotWgMember(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	memberID := seedUser(t, pool, now)
	outsiderID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	wgRowID := seedWgClaimSetup(t, pool, orgID, managerID, memberID, activityID, ptr(8.0), now)

	_, err := repo.Claim(ctx, orgID, wgRowID, outsiderID, 2.0,
		claimAudit(orgID, uuid.Nil, outsiderID, wgRowID, 2.0, now))
	require.ErrorIs(t, err, directiondomain.ErrNotWgMember)
	require.Zero(t, claimSum(t, pool, wgRowID), "a rejected claim must not consume budget")
}

// TestDirectionRepository_Claim_WgRowDraft proves the active-only pin
// (ADR-BE-018 §5, D-13-16): a draft WG row is not claimable.
func TestDirectionRepository_Claim_WgRowDraft(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	memberID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	wgID := seedWorkingGroup(t, pool, orgID, activityID, managerID, now)
	seedWgMember(t, pool, wgID, memberID, seedUnit(t, pool, orgID, now))
	wgRowID := seedDirectionRow(t, pool, orgID, managerID, nil, ptr(wgID), activityID, nil, ptr(8.0), "draft", now)

	_, err := repo.Claim(ctx, orgID, wgRowID, memberID, 2.0,
		claimAudit(orgID, uuid.Nil, memberID, wgRowID, 2.0, now))
	require.ErrorIs(t, err, directiondomain.ErrWgRowNotActive)
}

// TestDirectionRepository_Claim_OverBudget proves the single-threaded 409:
// a claim that would push the Σ over the WG budget fails with
// ErrClaimOverBudget and nothing commits.
func TestDirectionRepository_Claim_OverBudget(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	memberID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	wgRowID := seedWgClaimSetup(t, pool, orgID, managerID, memberID, activityID, ptr(8.0), now)

	_, err := repo.Claim(ctx, orgID, wgRowID, memberID, 6.0,
		claimAudit(orgID, uuid.Nil, memberID, wgRowID, 6.0, now))
	require.NoError(t, err)

	_, err = repo.Claim(ctx, orgID, wgRowID, memberID, 3.0,
		claimAudit(orgID, uuid.Nil, memberID, wgRowID, 3.0, now))
	require.ErrorIs(t, err, directiondomain.ErrClaimOverBudget,
		"Σ 6.0 + 3.0 > budget 8.0 — the in-tx Σ guard must reject (409)")
	require.Equal(t, 6.0, claimSum(t, pool, wgRowID), "an over-budget claim must not consume budget")
}

// TestDirectionRepository_Claim_Uncapped proves the uncapped path (D-13-14):
// a WG row with NULL budget accepts claims without a Σ gate — two 10h claims
// both succeed.
func TestDirectionRepository_Claim_Uncapped(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	memberA := seedUser(t, pool, now)
	memberB := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	wgID := seedWorkingGroup(t, pool, orgID, activityID, managerID, now)
	unitID := seedUnit(t, pool, orgID, now)
	seedWgMember(t, pool, wgID, memberA, unitID)
	seedWgMember(t, pool, wgID, memberB, unitID)
	// Budget NULL = uncapped.
	wgRowID := seedDirectionRow(t, pool, orgID, managerID, nil, ptr(wgID), activityID, nil, nil, "active", now)

	_, err := repo.Claim(ctx, orgID, wgRowID, memberA, 10.0,
		claimAudit(orgID, uuid.Nil, memberA, wgRowID, 10.0, now))
	require.NoError(t, err)
	_, err = repo.Claim(ctx, orgID, wgRowID, memberB, 10.0,
		claimAudit(orgID, uuid.Nil, memberB, wgRowID, 10.0, now))
	require.NoError(t, err, "an uncapped WG row accepts any positive claim (D-13-14)")
	require.Equal(t, 20.0, claimSum(t, pool, wgRowID))
}

// TestDirectionRepository_Unclaim proves unclaim = cancel of a claim row
// (D-13-16): the reason requirement holds, hours return to the WG budget
// (Σ-derived), and a re-claim succeeds.
func TestDirectionRepository_Unclaim(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	memberID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	wgRowID := seedWgClaimSetup(t, pool, orgID, managerID, memberID, activityID, ptr(8.0), now)

	claim, err := repo.Claim(ctx, orgID, wgRowID, memberID, 8.0,
		claimAudit(orgID, uuid.Nil, memberID, wgRowID, 8.0, now))
	require.NoError(t, err)
	require.Equal(t, 8.0, claimSum(t, pool, wgRowID))

	// Unclaim with a reason — the hours return (Σ-derived, D-13-16).
	unclaimed, err := repo.Unclaim(ctx, orgID, claim.ID, "no longer wanted",
		directionAudit(orgID, claim.ID, memberID, directiondomain.AuditActionCancelled,
			map[string]any{"reason": "no longer wanted"}, now))
	require.NoError(t, err)
	require.Equal(t, directiondomain.StatusCancelled, unclaimed.Status)
	require.Equal(t, 0.0, claimSum(t, pool, wgRowID), "unclaim frees the claimed hours (Σ-derived)")
	require.Equal(t, 1, countDirectionAudits(t, pool, claim.ID, directiondomain.AuditActionCancelled))

	// Re-claim the freed hours.
	reclaim, err := repo.Claim(ctx, orgID, wgRowID, memberID, 8.0,
		claimAudit(orgID, uuid.Nil, memberID, wgRowID, 8.0, now))
	require.NoError(t, err)
	require.Equal(t, 8.0, claimSum(t, pool, wgRowID), "re-claim succeeds after unclaim")
	require.Equal(t, directiondomain.StatusDraft, reclaim.Status)
}

// TestDirectionRepository_Unclaim_NotClaimRow proves the claim-row guard: the
// unclaim path rejects rows without origin_direction_id (ErrInvalidRequest).
func TestDirectionRepository_Unclaim_NotClaimRow(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	rowID := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "draft", now)

	_, err := repo.Unclaim(ctx, orgID, rowID, "nope",
		directionAudit(orgID, rowID, managerID, directiondomain.AuditActionCancelled, nil, now))
	require.ErrorIs(t, err, directiondomain.ErrInvalidRequest,
		"a non-claim row is not unclaimable through the unclaim path")
}

// TestDirectionClaim_Concurrent is the CR-01 closure battery (mirrors
// ticket_repository_test.go:418-506 + TestCoverageReplace_Concurrent): N=5
// members each claim 2.00h against an 8.00 WG budget concurrently. The
// WG-row FOR UPDATE lock serializes the claims — exactly the budget-bounded
// set commits (4 succeed + 1 ErrClaimOverBudget), Σ == 8.00, and
// over-subscription never commits.
func TestDirectionClaim_Concurrent(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	wgID := seedWorkingGroup(t, pool, orgID, activityID, managerID, now)
	unitID := seedUnit(t, pool, orgID, now)

	const n = 5
	members := make([]uuid.UUID, 0, n)
	for i := 0; i < n; i++ {
		m := seedUser(t, pool, now)
		seedWgMember(t, pool, wgID, m, unitID)
		members = append(members, m)
	}
	wgRowID := seedDirectionRow(t, pool, orgID, managerID, nil, ptr(wgID), activityID, nil, ptr(8.0), "active", now)

	start := make(chan struct{})
	results := make(chan error, n)
	for _, m := range members {
		m := m
		go func() {
			<-start
			_, err := repo.Claim(ctx, orgID, wgRowID, m, 2.0,
				claimAudit(orgID, uuid.Nil, m, wgRowID, 2.0, now))
			results <- err
		}()
	}
	close(start)

	successes, overBudget := 0, 0
	for i := 0; i < n; i++ {
		err := <-results
		switch {
		case err == nil:
			successes++
		case errors.Is(err, directiondomain.ErrClaimOverBudget):
			overBudget++
		default:
			t.Fatalf("unexpected claim race outcome: %v", err)
		}
	}
	require.Equal(t, 4, successes, "exactly the budget-bounded set commits (4 x 2.00 = 8.00)")
	require.Equal(t, 1, overBudget, "the 5th claim must fail the in-tx Σ re-check")

	var sum float64
	var claimRows int
	err := pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(est_hours), 0), COUNT(*) FROM direction
		 WHERE origin_direction_id = $1 AND status IN ('draft','active')`,
		wgRowID).Scan(&sum, &claimRows)
	require.NoError(t, err)
	require.Equal(t, 8.0, sum, "Σ claimed == the WG budget — over-subscription never commits")
	require.Equal(t, 4, claimRows, "exactly 4 claim rows exist in any committed state")
}

// TestDirectionClaim_SupersedeCancelReclaim is the D-13-15/D-13-16 contract
// through the chain (ADR-BE-018 §5): claim 8.00 against the 8.00 budget →
// supersede the claim row (the superseded row drops out of the Σ, the new
// row — origin carried, draft — counts the same 8.00: no strand, no double
// count) → cancel the new row (hours return, Σ == 0 — a re-claim succeeds) →
// re-claim 8.00 (Σ == 8.00 again).
func TestDirectionClaim_SupersedeCancelReclaim(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	memberID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	wgRowID := seedWgClaimSetup(t, pool, orgID, managerID, memberID, activityID, ptr(8.0), now)

	// 1. Claim the full budget.
	claim, err := repo.Claim(ctx, orgID, wgRowID, memberID, 8.0,
		claimAudit(orgID, uuid.Nil, memberID, wgRowID, 8.0, now))
	require.NoError(t, err)
	require.Equal(t, 8.0, claimSum(t, pool, wgRowID))

	// 2. Supersede the claim row — the chain carries the hours.
	superseding, err := repo.Create(ctx, orgID, &directiondomain.Direction{
		ID: uuid.New(), DirectedBy: managerID, DirectedTo: &memberID,
		ActivityID: activityID, EstHours: ptr(8.0),
	}, &claim.ID, []*audit.AuditLog{
		directionAudit(orgID, uuid.New(), managerID, directiondomain.AuditActionCreated, nil, now),
		directionAudit(orgID, claim.ID, managerID, directiondomain.AuditActionSuperseded, nil, now),
	})
	require.NoError(t, err)
	require.Equal(t, wgRowID, *superseding.OriginDirectionID, "the superseding row inherits the claim's origin")
	require.Equal(t, 8.0, claimSum(t, pool, wgRowID),
		"the supersede must keep the Σ at 8.00 — no double count, no strand")

	// 3. Cancel the superseding row — hours return to the WG budget.
	_, err = repo.Cancel(ctx, orgID, superseding.ID, "dropping the claim",
		directionAudit(orgID, superseding.ID, memberID, directiondomain.AuditActionCancelled,
			map[string]any{"reason": "dropping the claim"}, now))
	require.NoError(t, err)
	require.Equal(t, 0.0, claimSum(t, pool, wgRowID), "cancelling through the chain releases the hours")

	// 4. Re-claim the released budget.
	reclaim, err := repo.Claim(ctx, orgID, wgRowID, memberID, 8.0,
		claimAudit(orgID, uuid.Nil, memberID, wgRowID, 8.0, now))
	require.NoError(t, err)
	require.Equal(t, 8.0, claimSum(t, pool, wgRowID), "re-claim succeeds — Σ back to fully_claimed")
	require.Equal(t, directiondomain.StatusDraft, reclaim.Status)
}

// ---------------------------------------------------------------------------
// Task 1 (13-06) — ListPlan read-model: row selection + derived states
// (done/lapsed/claim spectrum) + queued ordering
// ---------------------------------------------------------------------------

// TestDirectionRepository_ListPlan_PeriodAndScope proves the ListPlan row
// selection (D-13-27): only draft|active rows; scheduled rows with
// planned_date inside [periodStart, periodEnd]; queued rows (planned_date
// NULL) with due_date inside the period OR no due_date at all (the queue is
// part of the plan view); superseded/cancelled rows never appear (history —
// D-13-08); employeeID filters to one employee, nil = org-wide.
func TestDirectionRepository_ListPlan_PeriodAndScope(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	otherID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	periodStart, periodEnd := today.AddDate(0, 0, -3), today.AddDate(0, 0, 3)

	// Scheduled rows.
	schedIn := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, ptr(today.AddDate(0, 0, 1)), ptr(4.0), "active", now)
	_ = seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, ptr(today.AddDate(0, 0, -5)), ptr(4.0), "active", now) // out of period
	otherEmp := seedDirectionRow(t, pool, orgID, managerID, &otherID, nil, activityID, ptr(today.AddDate(0, 0, 1)), ptr(4.0), "active", now.Add(-1*time.Hour))

	// Queued rows (planned_date NULL).
	queuedNoDue := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "draft", now)
	queuedDueIn := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "draft", now)
	queuedDueOut := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "draft", now)
	_, err := pool.Exec(ctx, `UPDATE direction SET due_date = $1 WHERE id = $2`, today.AddDate(0, 0, 2), queuedDueIn)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE direction SET due_date = $1 WHERE id = $2`, today.AddDate(0, 0, 10), queuedDueOut)
	require.NoError(t, err)

	// History rows — never appear (D-13-08).
	_ = seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, ptr(today.AddDate(0, 0, 1)), ptr(4.0), "superseded", now)
	_ = seedDirectionRowCancelled(t, pool, orgID, managerID, &employeeID, activityID, "no longer needed", now)

	// Employee-scoped view: scheduled in-period + queued rows (due in range or
	// no due); no history rows, no other-employee rows.
	rows, err := repo.ListPlan(ctx, orgID, &employeeID, periodStart, periodEnd)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{schedIn, queuedDueIn, queuedNoDue}, planRowIDs(rows),
		"scheduled in-period first (planned_date set), then queued: due_date set before due_date NULL; history/other-employee rows excluded")

	// Org-wide view includes the other employee's row.
	rowsAll, err := repo.ListPlan(ctx, orgID, nil, periodStart, periodEnd)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{otherEmp, schedIn, queuedDueIn, queuedNoDue}, planRowIDs(rowsAll))
}

// TestDirectionRepository_ListPlan_Ordering proves the pinned queued ordering
// (D-13-06, probe DIR-01 — stable order for equal keys): priority ASC (lower
// = higher) NULLS LAST → due_date ASC NULLS LAST → created_at ASC.
func TestDirectionRepository_ListPlan_Ordering(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	// All queued (planned_date NULL). created_at differs per seed.
	a := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "active", now.Add(-5*time.Hour))
	b := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "active", now.Add(-3*time.Hour))
	c := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "active", now.Add(-4*time.Hour))
	d := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "active", now.Add(-2*time.Hour))
	e := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, activityID, nil, ptr(4.0), "active", now.Add(-1*time.Hour))

	setQueued := func(id uuid.UUID, priority *int, due *time.Time) {
		t.Helper()
		_, err := pool.Exec(ctx, `UPDATE direction SET priority = $1, due_date = $2 WHERE id = $3`, priority, due, id)
		require.NoError(t, err)
	}
	setQueued(a, ptr(1), ptr(today.AddDate(0, 0, 1)))
	setQueued(b, ptr(1), ptr(today.AddDate(0, 0, 2)))
	setQueued(c, ptr(1), ptr(today.AddDate(0, 0, 2))) // same priority+due as b → created_at tiebreak
	setQueued(d, ptr(2), ptr(today.AddDate(0, 0, 3)))
	setQueued(e, nil, ptr(today.AddDate(0, 0, 3))) // priority NULL → last

	periodStart, periodEnd := today.AddDate(0, 0, -3), today.AddDate(0, 0, 3)
	rows, err := repo.ListPlan(ctx, orgID, nil, periodStart, periodEnd)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{a, c, b, d, e}, planRowIDs(rows),
		"priority 1 first (lower = higher), then 2, NULL last; equal priority → due_date ASC; equal both → created_at ASC")
}

// TestDirectionRepository_DerivedStates_DoneLapsed proves the derived
// done/lapsed states (D-13-09, ADR-BE-018 §2): done = the Phase 11
// terminal-activity CTE re-anchored at activities.id with the semantic
// inversion (NOT EXISTS of non-terminal entries on the subtree); lapsed =
// the row's date (planned_date, or due_date for queued) in the past AND no
// non-deleted entries of ANY status on the subtree (OQ2/A3 — a draft entry
// kills lapsed: "work started").
func TestDirectionRepository_DerivedStates_DoneLapsed(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)

	// One activity per scenario so subtree entries stay independent.
	actFuture := seedActivity(t, pool, orgID, "engagement", nil, now)
	actLapsed := seedActivity(t, pool, orgID, "engagement", nil, now)
	actDraft := seedActivity(t, pool, orgID, "engagement", nil, now)
	actApproved := seedActivity(t, pool, orgID, "engagement", nil, now)
	actQueuedLapsed := seedActivity(t, pool, orgID, "engagement", nil, now)

	// (a) future planned_date, empty subtree → done, not lapsed.
	futureID := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, actFuture, ptr(today.AddDate(0, 0, 1)), ptr(4.0), "active", now)
	// (b) past planned_date, empty subtree → done + lapsed.
	lapsedID := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, actLapsed, ptr(yesterday), ptr(4.0), "active", now)
	// (c) past planned_date + a DRAFT entry on a child (subtree) → not done + not lapsed (A3).
	draftChildID := seedActivity(t, pool, orgID, "engagement", &actDraft, now)
	draftID := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, actDraft, ptr(yesterday), ptr(4.0), "active", now)
	seedTimeEntry(t, pool, orgID, employeeID, draftChildID, unitID, 2.0, yesterday, "draft", now)
	// (d) past planned_date + only an APPROVED (terminal) entry → done + not lapsed (entries exist).
	approvedID := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, actApproved, ptr(yesterday), ptr(4.0), "active", now)
	seedTimeEntry(t, pool, orgID, employeeID, actApproved, unitID, 3.0, yesterday, "approved", now)
	// (e) queued row with a past due_date, empty subtree → lapsed (due_date governs).
	queuedLapsedID := seedDirectionRow(t, pool, orgID, managerID, &employeeID, nil, actQueuedLapsed, nil, ptr(4.0), "active", now)
	_, err := pool.Exec(ctx, `UPDATE direction SET due_date = $1 WHERE id = $2`, yesterday, queuedLapsedID)
	require.NoError(t, err)

	periodStart, periodEnd := today.AddDate(0, 0, -3), today.AddDate(0, 0, 3)
	rows, err := repo.ListPlan(ctx, orgID, nil, periodStart, periodEnd)
	require.NoError(t, err)
	byID := planRowByID(rows)

	require.True(t, byID[futureID].Done, "empty subtree → done")
	require.False(t, byID[futureID].Lapsed, "future date → not lapsed")
	require.True(t, byID[lapsedID].Done, "empty subtree → done")
	require.True(t, byID[lapsedID].Lapsed, "past planned_date + no entries → lapsed")
	require.False(t, byID[draftID].Done, "a draft entry on the subtree is non-terminal → not done")
	require.False(t, byID[draftID].Lapsed, "A3: a draft entry on the subtree kills lapsed (work started)")
	require.True(t, byID[approvedID].Done, "only terminal entries → done")
	require.False(t, byID[approvedID].Lapsed, "entries exist (even approved) → not lapsed")
	require.True(t, byID[queuedLapsedID].Done, "empty subtree → done")
	require.True(t, byID[queuedLapsedID].Lapsed, "queued row: past due_date governs lapsed")
}

// TestDirectionRepository_ListPlan_ClaimSpectrum proves the D-13-15 claim
// spectrum on WG rows: budget NULL → not_claimed until claims exist, then
// partially_claimed (never fully — D-13-14); budget set → not_claimed →
// partially_claimed → fully_claimed (Σ == budget); cancelled claims release
// their hours; a superseded claim row drops out of the Σ while its
// superseding row (origin_direction_id carried) counts the same hours — the
// spectrum stays fully_claimed through the chain (checker fix, ADR-BE-018
// §5).
func TestDirectionRepository_ListPlan_ClaimSpectrum(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	memberID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	wgID := seedWorkingGroup(t, pool, orgID, activityID, managerID, now)
	unitID := seedUnit(t, pool, orgID, now)
	seedWgMember(t, pool, wgID, memberID, unitID)
	wgRowID := seedDirectionRow(t, pool, orgID, managerID, nil, ptr(wgID), activityID, nil, ptr(8.0), "active", now)

	uncappedWgID := seedWorkingGroup(t, pool, orgID, activityID, managerID, now)
	seedWgMember(t, pool, uncappedWgID, memberID, unitID)
	uncappedRowID := seedDirectionRow(t, pool, orgID, managerID, nil, ptr(uncappedWgID), activityID, nil, nil, "active", now)

	periodStart, periodEnd := today.AddDate(0, 0, -3), today.AddDate(0, 0, 3)
	spectrum := func(rowID uuid.UUID) string {
		t.Helper()
		rows, err := repo.ListPlan(ctx, orgID, nil, periodStart, periodEnd)
		require.NoError(t, err)
		byID := planRowByID(rows)
		row, ok := byID[rowID]
		require.True(t, ok, "WG row must appear in the plan view")
		return row.ClaimState
	}

	// 1. No claims → not_claimed.
	require.Equal(t, directiondomain.ClaimStateNotClaimed, spectrum(wgRowID))

	// 2. Claim 3 of 8 → partially_claimed.
	claim1 := seedClaimRow(t, pool, orgID, managerID, memberID, wgRowID, activityID, 3.0, nil, nil, now)
	require.Equal(t, directiondomain.ClaimStatePartiallyClaimed, spectrum(wgRowID))

	// 3. Claim 5 more → fully_claimed.
	claim2 := seedClaimRow(t, pool, orgID, managerID, memberID, wgRowID, activityID, 5.0, nil, nil, now)
	require.Equal(t, directiondomain.ClaimStateFullyClaimed, spectrum(wgRowID))

	// 4. Cancel one claim → Σ 5 → partially_claimed (hours released, D-13-16).
	_, err := repo.Cancel(ctx, orgID, claim2, "dropping part of the claim",
		directionAudit(orgID, claim2, memberID, directiondomain.AuditActionCancelled, map[string]any{"reason": "dropping part of the claim"}, now))
	require.NoError(t, err)
	require.Equal(t, directiondomain.ClaimStatePartiallyClaimed, spectrum(wgRowID))

	// 5. Re-claim the freed hours → fully_claimed again.
	_ = seedClaimRow(t, pool, orgID, managerID, memberID, wgRowID, activityID, 5.0, nil, nil, now)
	require.Equal(t, directiondomain.ClaimStateFullyClaimed, spectrum(wgRowID))

	// 6. Supersede the 3h claim row: the superseded row drops out, the
	//    superseding row (origin carried) counts the same 3h — Σ stays 8,
	//    spectrum stays fully_claimed (chain consistency, checker fix).
	superseding, err := repo.Create(ctx, orgID, &directiondomain.Direction{
		ID: uuid.New(), DirectedBy: managerID, DirectedTo: &memberID,
		ActivityID: activityID, EstHours: ptr(3.0),
	}, &claim1, []*audit.AuditLog{
		directionAudit(orgID, uuid.New(), managerID, directiondomain.AuditActionCreated, nil, now),
		directionAudit(orgID, claim1, managerID, directiondomain.AuditActionSuperseded, nil, now),
	})
	require.NoError(t, err)
	require.Equal(t, wgRowID, *superseding.OriginDirectionID)
	require.Equal(t, 8.0, claimSum(t, pool, wgRowID), "supersede must not strand or double-count budget")
	require.Equal(t, directiondomain.ClaimStateFullyClaimed, spectrum(wgRowID))

	// 7. Uncapped WG row (budget NULL): claims exist → partially_claimed,
	//    never fully_claimed (D-13-14).
	require.Equal(t, directiondomain.ClaimStateNotClaimed, spectrum(uncappedRowID))
	_ = seedClaimRow(t, pool, orgID, managerID, memberID, uncappedRowID, activityID, 10.0, nil, nil, now)
	require.Equal(t, directiondomain.ClaimStatePartiallyClaimed, spectrum(uncappedRowID))
}

// ---------------------------------------------------------------------------
// Task 2 (13-06) — Coverage read-model (capacity math, planned Σ, gap) +
// AbsenceWindows
// ---------------------------------------------------------------------------

// coverageByCell indexes CoverageRow by (employee, day) for per-cell
// assertions.
func coverageByCell(rows []directiondomain.CoverageRow) map[[2]any]directiondomain.CoverageRow {
	m := make(map[[2]any]directiondomain.CoverageRow, len(rows))
	for _, r := range rows {
		m[[2]any{r.EmployeeID, r.Date}] = r
	}
	return m
}

// TestDirectionRepository_Coverage_CapacityMath proves the per-day capacity
// math (D-13-24, Pitfall 10): default 8.0 when no org_settings row; a full
// window (hours NULL) zeroes the day (away); a partial window reduces by its
// hours; a multi-day window subtracts once per covered day; overlapping
// windows both subtract; capacity floors at 0 (never negative — a zeroed day
// is away, never fake over-capacity).
func TestDirectionRepository_Coverage_CapacityMath(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	employeeID := seedUser(t, pool, now)

	d0, d1, d2, d3, d4, d5 := today, today.AddDate(0, 0, 1), today.AddDate(0, 0, 2),
		today.AddDate(0, 0, 3), today.AddDate(0, 0, 4), today.AddDate(0, 0, 5)

	// d0: two OVERLAPPING partial permits (3.5 + 2.0) → 8 − 5.5 = 2.5.
	seedAvailabilityWindow(t, pool, orgID, employeeID, "permit", d0, d0, ptr(3.5), "confirmed", now)
	seedAvailabilityWindow(t, pool, orgID, employeeID, "permit", d0, d0, ptr(2.0), "confirmed", now)
	// d1: single full holiday → 0.
	seedAvailabilityWindow(t, pool, orgID, employeeID, "holiday", d1, d1, nil, "confirmed", now)
	// d2..d3: multi-day FULL window (declared) → 0 on both days (once each).
	seedAvailabilityWindow(t, pool, orgID, employeeID, "holiday", d2, d3, nil, "declared", now)
	// d4: partial multi-day permit spanning beyond the period end → the
	// in-period day subtracts its hours once (6.5), the out-of-period part is
	// clamped away.
	seedAvailabilityWindow(t, pool, orgID, employeeID, "permit", d4, d5, ptr(1.5), "declared", now)
	// d5: a partial permit larger than the daily hours → capacity floors at 0.
	seedAvailabilityWindow(t, pool, orgID, employeeID, "permit", d5, d5, ptr(9.0), "declared", now)

	periodStart, periodEnd := today, d5
	rows, err := repo.Coverage(ctx, orgID, []uuid.UUID{employeeID}, periodStart, periodEnd)
	require.NoError(t, err)
	byDay := coverageByCell(rows)

	require.Equal(t, 6, len(rows), "one row per (employee, day) for every day in the period")
	require.Equal(t, 2.5, byDay[[2]any{employeeID, d0}].Capacity, "overlapping partial permits both subtract (8 − 3.5 − 2)")
	require.Equal(t, 0.0, byDay[[2]any{employeeID, d1}].Capacity, "full absence zeroes the day")
	require.Equal(t, 0.0, byDay[[2]any{employeeID, d2}].Capacity, "multi-day full window covers d2")
	require.Equal(t, 0.0, byDay[[2]any{employeeID, d3}].Capacity, "multi-day full window covers d3 — once per day")
	require.Equal(t, 6.5, byDay[[2]any{employeeID, d4}].Capacity, "partial multi-day window subtracts once on the in-period day")
	require.Equal(t, 0.0, byDay[[2]any{employeeID, d5}].Capacity, "capacity floors at 0 — never negative (Pitfall 10)")
}

// TestDirectionRepository_Coverage_CustomDailyHours proves the org_settings
// planning_daily_hours override (D-13-24, ADR-BE-018 consequence): a 7.5 row
// changes the capacity denominator; the COALESCE default (8.0) applies when
// the key is absent.
func TestDirectionRepository_Coverage_CustomDailyHours(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	employeeID := seedUser(t, pool, now)

	seedOrgSettings(t, pool, orgID, "planning_daily_hours", 7.5, now)

	d0, d1, d2 := today, today.AddDate(0, 0, 1), today.AddDate(0, 0, 2)
	seedAvailabilityWindow(t, pool, orgID, employeeID, "permit", d1, d1, ptr(3.5), "confirmed", now)
	seedAvailabilityWindow(t, pool, orgID, employeeID, "holiday", d2, d2, nil, "confirmed", now)

	rows, err := repo.Coverage(ctx, orgID, []uuid.UUID{employeeID}, today, d2)
	require.NoError(t, err)
	byDay := coverageByCell(rows)

	require.Equal(t, 7.5, byDay[[2]any{employeeID, d0}].Capacity, "custom daily hours become the denominator")
	require.Equal(t, 4.0, byDay[[2]any{employeeID, d1}].Capacity, "7.5 − 3.5 partial permit")
	require.Equal(t, 0.0, byDay[[2]any{employeeID, d2}].Capacity, "full absence zeroes regardless of the denominator")
}

// TestDirectionRepository_Coverage_PlannedAndGap proves the planned Σ and gap
// per (employee, day) (D-13-25/26, Pitfall 7): planned counts ONLY
// draft|active rows for that employee on that exact day (superseded/cancelled
// history rows excluded — T-13-19); multiple rows sharing the day sum
// (D-W/D-AA multiplicity); an employee with no rows surfaces planned 0 and
// gap == capacity (uncovered day — D-13-26); over-capacity yields a negative
// gap; fully-absent days still return a row (the repo owns the math, the
// service owns the surfacing policy).
func TestDirectionRepository_Coverage_PlannedAndGap(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	emp1 := seedUser(t, pool, now)
	emp2 := seedUser(t, pool, now)
	emp3 := seedUser(t, pool, now)
	emp4 := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	// emp1: two active rows sharing the day (Σ 5.5) + superseded + cancelled
	// history rows that must NOT count.
	_ = seedDirectionRow(t, pool, orgID, managerID, &emp1, nil, activityID, ptr(today), ptr(3.5), "active", now)
	_ = seedDirectionRow(t, pool, orgID, managerID, &emp1, nil, activityID, ptr(today), ptr(2.0), "active", now)
	_ = seedDirectionRow(t, pool, orgID, managerID, &emp1, nil, activityID, ptr(today), ptr(4.0), "superseded", now)
	cancID := seedDirectionRow(t, pool, orgID, managerID, &emp1, nil, activityID, ptr(today), ptr(1.0), "active", now)
	_, err := pool.Exec(ctx, `UPDATE direction SET status = 'cancelled', reason = 'history' WHERE id = $1`, cancID)
	require.NoError(t, err)
	// emp2: over-capacity (9.0 planned vs 8.0 capacity → gap −1).
	_ = seedDirectionRow(t, pool, orgID, managerID, &emp2, nil, activityID, ptr(today), ptr(9.0), "active", now)
	// emp3: no direction rows + a full absence → away day surfaced with 0s.
	seedAvailabilityWindow(t, pool, orgID, emp3, "holiday", today, today, nil, "confirmed", now)
	// emp4: no rows today (only a draft row on a different day) → uncovered
	// day (planned 0, gap == capacity).
	_ = seedDirectionRow(t, pool, orgID, managerID, &emp4, nil, activityID, ptr(today.AddDate(0, 0, 1)), ptr(4.0), "draft", now)

	rows, err := repo.Coverage(ctx, orgID, []uuid.UUID{emp1, emp2, emp3, emp4}, today, today)
	require.NoError(t, err)
	byCell := coverageByCell(rows)

	require.Equal(t, 4, len(rows), "one row per employee per day in the period")
	require.Equal(t, 5.5, byCell[[2]any{emp1, today}].Planned, "Σ of the two active rows sharing the day; superseded/cancelled rows excluded (T-13-19)")
	require.Equal(t, 2.5, byCell[[2]any{emp1, today}].Gap, "8.0 − 5.5")
	require.Equal(t, 9.0, byCell[[2]any{emp2, today}].Planned)
	require.Equal(t, -1.0, byCell[[2]any{emp2, today}].Gap, "over-capacity: gap is negative")
	require.Equal(t, 0.0, byCell[[2]any{emp3, today}].Capacity, "fully-absent day still returns a row (service owns surfacing)")
	require.Equal(t, 0.0, byCell[[2]any{emp3, today}].Gap)
	require.Equal(t, 0.0, byCell[[2]any{emp4, today}].Planned, "no rows today → planned 0")
	require.Equal(t, 8.0, byCell[[2]any{emp4, today}].Gap, "uncovered day surfaced: gap == capacity (D-13-26)")
}

// TestDirectionRepository_AbsenceWindows proves AbsenceWindows returns
// availability_windows rows with status IN ('declared','confirmed') — BOTH
// statuses per D-13-29 — overlapping [start, end], org-scoped, with kind +
// hours + starts_on + ends_on for the service-side warning formatting.
func TestDirectionRepository_AbsenceWindows(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	otherOrgID := seedOrg(t, pool, now)
	emp1 := seedUser(t, pool, now)
	emp2 := seedUser(t, pool, now)

	_ = seedAvailabilityWindow(t, pool, orgID, emp1, "holiday", today, today.AddDate(0, 0, 1), nil, "declared", now)
	_ = seedAvailabilityWindow(t, pool, orgID, emp1, "permit", today.AddDate(0, 0, 2), today.AddDate(0, 0, 2), ptr(3.5), "confirmed", now)
	// Out of range (ends before the period starts) — excluded.
	_ = seedAvailabilityWindow(t, pool, orgID, emp1, "unavailable", today.AddDate(0, 0, -5), today.AddDate(0, 0, -1), nil, "declared", now)
	// Another org's window for the same employee — excluded (org-scoped).
	_ = seedAvailabilityWindow(t, pool, otherOrgID, emp1, "medical", today, today, nil, "confirmed", now)
	// emp2 has no windows in the period.
	_ = seedAvailabilityWindow(t, pool, orgID, emp2, "holiday", today.AddDate(0, 0, -5), today.AddDate(0, 0, -1), nil, "declared", now)

	windows, err := repo.AbsenceWindows(ctx, orgID, []uuid.UUID{emp1, emp2}, today, today.AddDate(0, 0, 3))
	require.NoError(t, err)
	require.Equal(t, 2, len(windows), "declared + confirmed rows in the overlap, org-scoped")

	// The 'declared' window is the full holiday (hours NULL); the 'confirmed'
	// window is the partial permit (hours 3.5) — the struct carries kind +
	// hours, the status vocabulary is exercised by which rows come back.
	var full, partial *directiondomain.AbsenceWindow
	for i := range windows {
		switch windows[i].Kind {
		case "holiday":
			full = &windows[i]
		case "permit":
			partial = &windows[i]
		}
	}
	require.NotNil(t, full, "the declared-status full window must be returned")
	require.Equal(t, emp1, full.EmployeeID)
	require.Equal(t, "holiday", full.Kind)
	require.Nil(t, full.Hours, "full absence: hours NULL")
	require.Equal(t, today, full.StartsOn)
	require.Equal(t, today.AddDate(0, 0, 1), full.EndsOn)

	require.NotNil(t, partial, "the confirmed-status partial window must be returned")
	require.Equal(t, emp1, partial.EmployeeID)
	require.Equal(t, "permit", partial.Kind)
	require.NotNil(t, partial.Hours)
	require.Equal(t, 3.5, *partial.Hours)
}

// ---------------------------------------------------------------------------
// Task 3 (13-06) — FirstDirectionRefs (origin fallback lookup, D-13-32..34)
// ---------------------------------------------------------------------------

// TestDirectionRepository_FirstDirectionRefs proves the origin-fallback
// lookup (D-13-32..34, Pattern 8): the manager-assignment-shaped refs
// (assigned_by = directed_by, assigned_to = directed_to) of the EARLIEST
// created_at NON-CANCELLED direction row for the activity, org-scoped,
// read-only; nil when no such row (refs stay empty — D-13-33/34). Never the
// other origin shapes.
func TestDirectionRepository_FirstDirectionRefs(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewDirectionRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	otherOrgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	empA := seedUser(t, pool, now)
	empB := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	otherActivityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	// (a) Two rows on activityID: the EARLIEST created_at non-cancelled row
	// wins (assigned_to = empA).
	_ = seedDirectionRow(t, pool, orgID, managerID, &empB, nil, activityID, nil, ptr(4.0), "draft", now.Add(-2*time.Hour))
	_ = seedDirectionRow(t, pool, orgID, managerID, &empA, nil, activityID, nil, ptr(4.0), "draft", now.Add(-1*time.Hour))
	refs, err := repo.FirstDirectionRefs(ctx, orgID, activityID)
	require.NoError(t, err)
	require.NotNil(t, refs, "a non-cancelled row exists → refs are not empty")
	require.Equal(t, managerID, *refs.AssignedBy)
	require.Equal(t, empB, *refs.AssignedTo, "the earliest created_at row supplies the pair")

	// (b) A CANCELLED earliest row is skipped in favor of a later
	// non-cancelled row (on otherActivityID).
	_ = seedDirectionRowCancelled(t, pool, orgID, managerID, &empA, otherActivityID, "cancelled first", now.Add(-3*time.Hour))
	_ = seedDirectionRow(t, pool, orgID, managerID, &empB, nil, otherActivityID, nil, ptr(4.0), "draft", now.Add(-1*time.Hour))
	refs, err = repo.FirstDirectionRefs(ctx, orgID, otherActivityID)
	require.NoError(t, err)
	require.NotNil(t, refs)
	require.Equal(t, empB, *refs.AssignedTo, "the cancelled earliest row is skipped")

	// (c) No rows at all → (nil, nil).
	noRowsActivityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	refs, err = repo.FirstDirectionRefs(ctx, orgID, noRowsActivityID)
	require.NoError(t, err)
	require.Nil(t, refs, "no rows → refs stay empty (D-13-33)")

	// (d) Rows exist but ALL cancelled → (nil, nil).
	allCancelledActivityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	_ = seedDirectionRowCancelled(t, pool, orgID, managerID, &empA, allCancelledActivityID, "first", now.Add(-2*time.Hour))
	_ = seedDirectionRowCancelled(t, pool, orgID, managerID, &empB, allCancelledActivityID, "second", now.Add(-1*time.Hour))
	refs, err = repo.FirstDirectionRefs(ctx, orgID, allCancelledActivityID)
	require.NoError(t, err)
	require.Nil(t, refs, "all-cancelled rows are not a fallback source")

	// (e) Rows for ANOTHER org → (nil, nil) (org-scoped, T-13-21 — no cross-
	// org leak into the activity read path).
	_ = seedDirectionRow(t, pool, otherOrgID, managerID, &empA, nil, activityID, nil, ptr(4.0), "draft", now.Add(-1*time.Hour))
	refs, err = repo.FirstDirectionRefs(ctx, orgID, activityID)
	require.NoError(t, err)
	require.NotNil(t, refs, "org-scoping: the other-org row must not shadow this org's rows")

	// (f) A WG row (wg_id set, directed_to NULL) → AssignedBy set,
	// AssignedTo nil (the fallback carries what exists).
	wgActivityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	wgID := seedWorkingGroup(t, pool, orgID, wgActivityID, managerID, now)
	_ = seedDirectionRow(t, pool, orgID, managerID, nil, ptr(wgID), wgActivityID, nil, ptr(8.0), "active", now.Add(-1*time.Hour))
	refs, err = repo.FirstDirectionRefs(ctx, orgID, wgActivityID)
	require.NoError(t, err)
	require.NotNil(t, refs)
	require.Equal(t, managerID, *refs.AssignedBy)
	require.Nil(t, refs.AssignedTo, "a WG row carries no directed_to — AssignedTo stays nil")
}
