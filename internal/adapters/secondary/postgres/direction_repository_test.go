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
