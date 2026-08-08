package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The two cycle tests below mirror the TestMigration018..020 shape
// (coverage_ontology_migrations_test.go): apply pre-state migrations (skipping
// the migration under test) → apply up → assert schema shape + functional
// behavior → apply down → assert reversed → apply up again → assert still
// green (ADR-BE-004 up/down/up cycle).
//
// The functional assertions are the Pitfall-2 regression guards (T-13-01):
// every direction CHECK must reject its violating insert with 23514 + the
// named constraint, and the identity model must hold — multiple rows sharing
// (directed_to, activity_id, planned_date) insert successfully (D-W, D-AA).

// assertPrimaryKey asserts a PRIMARY KEY constraint exists by name on a table
// (pg_constraint, contype 'p').
func assertPrimaryKey(t *testing.T, pool *pgxpool.Pool, ctx context.Context, table, conname string) {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pg_constraint c
		 JOIN pg_class t ON t.oid = c.conrelid
		 WHERE c.conname = $1 AND c.contype = 'p' AND t.relname = $2`,
		conname, table).Scan(&n))
	assert.Equal(t, 1, n, "primary key %s on %s should exist", conname, table)
}

// TestMigration021_DirectionRows_UpDownUpCycle verifies migration 021
// (direction): table shape, all six constraints by name, and functionally —
// a valid scheduled row inserts, a second row sharing the same
// employee/activity/day inserts (identity = row id, D-W), the targetless row
// trips direction_target_check, a WG row with planned_date trips
// direction_wg_queued_check, a scheduled row without est_hours trips
// direction_scheduled_hours_check, est_hours=0 trips
// direction_est_hours_check, a cancelled row without reason trips
// direction_cancel_reason_check, and a queued row passes (Pitfall 2, T-13-01).
func TestMigration021_DirectionRows_UpDownUpCycle(t *testing.T) {
	pool := TestPool(t)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })
	ctx := context.Background()
	now := time.Now()

	up021 := readMigration(t, "021_direction_rows.up.sql")
	down021 := readMigration(t, "021_direction_rows.down.sql")

	// --- Pre-state: schema 000-020 + 022 (021 skipped so UP can apply) ---
	applyMigrations(t, pool, true, "021_direction_rows.up.sql")
	orgID := seedOrg(t, pool, now)
	directedByID := seedUser(t, pool, now)
	directedToID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	wgID := seedWorkingGroup(t, pool, orgID, activityID, directedByID, now)

	// --- UP ------------------------------------------------------------------
	_, err := pool.Exec(ctx, up021)
	require.NoError(t, err, "021 up should apply cleanly")

	assertTableExists(t, pool, ctx, "direction", true)
	for _, con := range []string{
		"direction_target_check",
		"direction_wg_queued_check",
		"direction_est_hours_check",
		"direction_scheduled_hours_check",
		"direction_status_check",
		"direction_cancel_reason_check",
	} {
		assertConstraintExists(t, pool, ctx, con)
	}

	// (a) Valid scheduled row: directed_to set, wg_id NULL, planned_date +
	//     est_hours — must pass.
	_, err = pool.Exec(ctx, `INSERT INTO direction
		(org_id, directed_by, directed_to, wg_id, activity_id, planned_date, est_hours, status, created_at, updated_at)
		VALUES ($1, $2, $3, NULL, $4, DATE '2026-08-10', 8.00, 'draft', $5, $5)`,
		orgID, directedByID, directedToID, activityID, now)
	require.NoError(t, err, "valid scheduled row must pass")

	// (b) Second row with the SAME directed_to + activity_id + planned_date
	//     — must pass: identity is the row id, per-day multiplicity is legal
	//     (D-W, D-AA).
	_, err = pool.Exec(ctx, `INSERT INTO direction
		(org_id, directed_by, directed_to, wg_id, activity_id, planned_date, est_hours, status, created_at, updated_at)
		VALUES ($1, $2, $3, NULL, $4, DATE '2026-08-10', 6.00, 'draft', $5, $5)`,
		orgID, directedByID, directedToID, activityID, now)
	require.NoError(t, err, "second row sharing employee/activity/day must pass (identity = row id)")

	// (c) Targetless row (both directed_to and wg_id NULL) — must fail on
	//     direction_target_check (explicit IS [NOT] NULL pins both sides).
	_, err = pool.Exec(ctx, `INSERT INTO direction
		(org_id, directed_by, directed_to, wg_id, activity_id, planned_date, est_hours, status, created_at, updated_at)
		VALUES ($1, $2, NULL, NULL, $3, DATE '2026-08-10', 8.00, 'draft', $4, $4)`,
		orgID, directedByID, activityID, now)
	require.Error(t, err, "targetless row must fail")
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23514", pgErr.Code, "targetless row must be a CHECK violation")
	assert.Equal(t, "direction_target_check", pgErr.ConstraintName,
		"targetless row must trip the target check")

	// (d) WG row with planned_date set — must fail on
	//     direction_wg_queued_check (D-13-17).
	_, err = pool.Exec(ctx, `INSERT INTO direction
		(org_id, directed_by, directed_to, wg_id, activity_id, planned_date, est_hours, status, created_at, updated_at)
		VALUES ($1, $2, NULL, $3, $4, DATE '2026-08-10', 8.00, 'draft', $5, $5)`,
		orgID, directedByID, wgID, activityID, now)
	require.Error(t, err, "WG row with planned_date must fail")
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23514", pgErr.Code, "WG row with planned_date must be a CHECK violation")
	assert.Equal(t, "direction_wg_queued_check", pgErr.ConstraintName,
		"WG row with planned_date must trip the queued-only check")

	// (e) Scheduled row without est_hours — must fail on
	//     direction_scheduled_hours_check (D-13-02).
	_, err = pool.Exec(ctx, `INSERT INTO direction
		(org_id, directed_by, directed_to, wg_id, activity_id, planned_date, est_hours, status, created_at, updated_at)
		VALUES ($1, $2, $3, NULL, $4, DATE '2026-08-10', NULL, 'draft', $5, $5)`,
		orgID, directedByID, directedToID, activityID, now)
	require.Error(t, err, "scheduled row without est_hours must fail")
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23514", pgErr.Code, "scheduled row without est_hours must be a CHECK violation")
	assert.Equal(t, "direction_scheduled_hours_check", pgErr.ConstraintName,
		"scheduled row without est_hours must trip the scheduled-hours check")

	// (f) est_hours = 0 — must fail on direction_est_hours_check
	//     (mirrors time_entries.hours CHECK).
	_, err = pool.Exec(ctx, `INSERT INTO direction
		(org_id, directed_by, directed_to, wg_id, activity_id, planned_date, est_hours, status, created_at, updated_at)
		VALUES ($1, $2, $3, NULL, $4, NULL, 0.00, 'draft', $5, $5)`,
		orgID, directedByID, directedToID, activityID, now)
	require.Error(t, err, "est_hours = 0 must fail")
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23514", pgErr.Code, "est_hours = 0 must be a CHECK violation")
	assert.Equal(t, "direction_est_hours_check", pgErr.ConstraintName,
		"est_hours = 0 must trip the est_hours check")

	// (g) Cancelled row without reason — must fail on
	//     direction_cancel_reason_check (D-13-10; the IS NOT NULL form is
	//     never NULL-satisfiable — Pitfall 2).
	_, err = pool.Exec(ctx, `INSERT INTO direction
		(org_id, directed_by, directed_to, wg_id, activity_id, planned_date, est_hours, status, reason, created_at, updated_at)
		VALUES ($1, $2, $3, NULL, $4, NULL, NULL, 'cancelled', NULL, $5, $5)`,
		orgID, directedByID, directedToID, activityID, now)
	require.Error(t, err, "cancelled row without reason must fail")
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23514", pgErr.Code, "cancelled row without reason must be a CHECK violation")
	assert.Equal(t, "direction_cancel_reason_check", pgErr.ConstraintName,
		"cancelled row without reason must trip the cancel-reason check")

	// (h) Queued row (planned_date NULL, est_hours NULL) — must pass (D-R).
	_, err = pool.Exec(ctx, `INSERT INTO direction
		(org_id, directed_by, directed_to, wg_id, activity_id, planned_date, est_hours, status, created_at, updated_at)
		VALUES ($1, $2, $3, NULL, $4, NULL, NULL, 'draft', $5, $5)`,
		orgID, directedByID, directedToID, activityID, now)
	require.NoError(t, err, "queued row must pass")

	// --- DOWN ----------------------------------------------------------------
	_, err = pool.Exec(ctx, down021)
	require.NoError(t, err, "021 down should apply cleanly")
	assertTableExists(t, pool, ctx, "direction", false)

	// --- UP again (cycle) ----------------------------------------------------
	_, err = pool.Exec(ctx, up021)
	require.NoError(t, err, "021 up should re-apply cleanly after down")
	assertTableExists(t, pool, ctx, "direction", true)
	assertConstraintExists(t, pool, ctx, "direction_target_check")
}

// TestMigration022_OrgSettings_UpDownUpCycle verifies migration 022
// (org_settings + organization_memberships.planning_mode): the table exists
// with PK(org_id, key) (D-13-18 — JSONB value has no CHECK by design, the
// vocabulary gate is code-level), an upsert replaces the value for the same
// key, planning_mode is a nullable column, and down drops the column before
// the table (Pitfall 8, T-13-03).
func TestMigration022_OrgSettings_UpDownUpCycle(t *testing.T) {
	pool := TestPool(t)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })
	ctx := context.Background()
	now := time.Now()

	up022 := readMigration(t, "022_org_settings.up.sql")
	down022 := readMigration(t, "022_org_settings.down.sql")

	// --- Pre-state: schema 000-021 (022 skipped so UP can apply) ---
	applyMigrations(t, pool, true, "022_org_settings.up.sql")
	orgID := seedOrg(t, pool, now)

	// --- UP ------------------------------------------------------------------
	_, err := pool.Exec(ctx, up022)
	require.NoError(t, err, "022 up should apply cleanly")

	assertTableExists(t, pool, ctx, "org_settings", true)
	assertPrimaryKey(t, pool, ctx, "org_settings", "org_settings_pkey")

	// planning_mode is a nullable override (no backfill — D-13-19).
	var isNullable string
	require.NoError(t, pool.QueryRow(ctx, `SELECT is_nullable FROM information_schema.columns
		WHERE table_name = 'organization_memberships' AND column_name = 'planning_mode'`).Scan(&isNullable))
	assert.Equal(t, "YES", isNullable, "planning_mode must be nullable")

	// Functional: INSERT a key, then upsert the same key — the value must be
	// replaced (PK(org_id, key) identity, D-13-18).
	_, err = pool.Exec(ctx, `INSERT INTO org_settings (org_id, key, value, updated_at)
		VALUES ($1, 'planning_daily_hours', '8', $2)`, orgID, now)
	require.NoError(t, err, "org_settings insert should succeed")
	_, err = pool.Exec(ctx, `INSERT INTO org_settings (org_id, key, value, updated_at)
		VALUES ($1, 'planning_daily_hours', '7.5', $2)
		ON CONFLICT (org_id, key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at`,
		orgID, now)
	require.NoError(t, err, "org_settings upsert should succeed")
	var got string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT value::text FROM org_settings WHERE org_id = $1 AND key = 'planning_daily_hours'`, orgID).Scan(&got))
	assert.Equal(t, "7.5", got, "value must be replaced by the ON CONFLICT upsert")

	// --- DOWN ----------------------------------------------------------------
	_, err = pool.Exec(ctx, down022)
	require.NoError(t, err, "022 down should apply cleanly")

	var n int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'organization_memberships' AND column_name = 'planning_mode'`).Scan(&n))
	assert.Zero(t, n, "planning_mode must be dropped by 022 down")
	assertTableExists(t, pool, ctx, "org_settings", false)

	// --- UP again (cycle) ----------------------------------------------------
	_, err = pool.Exec(ctx, up022)
	require.NoError(t, err, "022 up should re-apply cleanly after down")
	assertTableExists(t, pool, ctx, "org_settings", true)
	assertPrimaryKey(t, pool, ctx, "org_settings", "org_settings_pkey")
}
