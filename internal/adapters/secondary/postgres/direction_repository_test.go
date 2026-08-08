package postgres

import (
	"context"
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
		ID:         uuid.New(),
		DirectedBy: managerID,
		DirectedTo: &employeeID,
		ActivityID: activityID,
		PlannedDate: &planned,
		EstHours:   ptr(8.0),
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
