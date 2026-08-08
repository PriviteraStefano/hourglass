package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration012_StaffingSchema_UpDownUpCycle verifies migration 012
// (staffing schema, ADR-P-008) against the MVP seed data:
//   - up applies cleanly and creates availability_windows per the ADR sketch
//     (columns, nullability, CHECKs, index), adds the three nullable validity
//     DATE columns to organization_memberships, and extends the role CHECK
//     with 'hr'
//   - behavior: valid window insert succeeds; invalid kind and inverted dates
//     fail on CHECK; role='hr' update succeeds; role='bogus' still fails
//   - down restores the pre-012 state exactly (table gone, index gone,
//     columns gone, role CHECK without 'hr' — hr update fails again)
//   - up → down → up cycle completes without error
func TestMigration012_StaffingSchema_UpDownUpCycle(t *testing.T) {
	pool := TestPool(t)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })
	ctx := context.Background()

	up012 := readMigration(t, "012_staffing_schema.up.sql")
	down012 := readMigration(t, "012_staffing_schema.down.sql")

	// --- Pre-state: schema 000-011 + MVP seed -------------------------------
	// 013 is not skipped here: it applies in sorted order AFTER 011 (the
	// activities table exists), and its kind relabel is a no-op for this
	// test's assertions. 014-017 are skipped so the pre-state stays exactly
	// at 000-013 (this test predates those migrations). 018-020 are skipped
	// too (coverage migrations, applied by later cycle tests).
	applyMigrations(t, pool, true, "012_staffing_schema.up.sql",
		"014_ticket_schema.up.sql", "015_activity_origins.up.sql",
		"016_contract_sold_hours.up.sql", "017_audit_logs.up.sql",
		"018_activity_beneficiary_unit.up.sql", "019_coverage_allocations.up.sql",
		"020_coverage_snapshots.up.sql")
	// The historical MVP seed (003_seed.up.sql) is no longer a migration
	// fixture — seed data lives in scripts/seed_demo.sql which applyMigrations
	// never loads. Self-seed the membership pre-state: 6 memberships across
	// one org (helpers don't cover organization_memberships, so direct SQL).
	now := time.Now()
	orgID := seedOrg(t, pool, now)
	for i := 0; i < 6; i++ {
		userID := seedUser(t, pool, now)
		_, err := pool.Exec(ctx,
			`INSERT INTO organization_memberships (user_id, organization_id, role, is_active, created_at, updated_at)
			 VALUES ($1, $2, 'employee', TRUE, $3, $3)`, userID, orgID, now)
		require.NoError(t, err)
	}
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM organization_memberships", 6)

	// --- UP ------------------------------------------------------------------
	_, err := pool.Exec(ctx, up012)
	require.NoError(t, err, "012 up should apply cleanly")

	// availability_windows exists with the ADR-P-008 column set.
	assertTableExists(t, pool, ctx, "availability_windows", true)
	assertColumnNotNull(t, pool, ctx, "availability_windows", "org_id")
	assertColumnNotNull(t, pool, ctx, "availability_windows", "user_id")
	assertColumnNotNull(t, pool, ctx, "availability_windows", "kind")
	assertColumnNotNull(t, pool, ctx, "availability_windows", "starts_on")
	assertColumnNotNull(t, pool, ctx, "availability_windows", "ends_on")
	assertColumnNotNull(t, pool, ctx, "availability_windows", "status")
	assertColumnNotNull(t, pool, ctx, "availability_windows", "created_by")

	// hours / certificate_ref / note are nullable (partial-day permits, D-1).
	var nullable string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT is_nullable FROM information_schema.columns
		 WHERE table_name = 'availability_windows' AND column_name = 'hours'`).Scan(&nullable))
	assert.Equal(t, "YES", nullable, "availability_windows.hours must be nullable")
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT is_nullable FROM information_schema.columns
		 WHERE table_name = 'availability_windows' AND column_name = 'certificate_ref'`).Scan(&nullable))
	assert.Equal(t, "YES", nullable, "availability_windows.certificate_ref must be nullable")

	// CHECK constraints per the ADR sketch.
	assertConstraintExists(t, pool, ctx, "availability_windows_kind_check")
	assertConstraintExists(t, pool, ctx, "availability_windows_status_check")
	assertConstraintExists(t, pool, ctx, "availability_windows_check") // ends_on >= starts_on

	// Per-person date-range index.
	var idxCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pg_indexes WHERE indexname = 'idx_availability_windows_org_user_dates'`).Scan(&idxCount))
	assert.Equal(t, 1, idxCount, "idx_availability_windows_org_user_dates must exist")

	// organization_memberships gained three nullable DATE columns (D-2).
	for _, col := range []string{"valid_from", "valid_until", "work_permit_expires_at"} {
		var isNull string
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT is_nullable FROM information_schema.columns
			 WHERE table_name = 'organization_memberships' AND column_name = $1`, col).Scan(&isNull))
		assert.Equal(t, "YES", isNull, "organization_memberships.%s must be nullable", col)
	}

	// role CHECK now accepts 'hr' (D-4) and still rejects bogus values.
	_, err = pool.Exec(ctx, `UPDATE organization_memberships
		SET role = 'hr', updated_at = NOW() WHERE id = (SELECT id FROM organization_memberships LIMIT 1)`)
	require.NoError(t, err, "role 'hr' must be accepted after 012 up")
	_, err = pool.Exec(ctx, `UPDATE organization_memberships
		SET role = 'bogus' WHERE id = (SELECT id FROM organization_memberships LIMIT 1)`)
	require.Error(t, err, "role 'bogus' must still fail after 012 up")

	// Valid availability window insert succeeds.
	_, err = pool.Exec(ctx, `INSERT INTO availability_windows
		(org_id, user_id, kind, starts_on, ends_on, hours, note, status, created_by)
		VALUES ((SELECT id FROM organizations LIMIT 1), (SELECT id FROM users LIMIT 1),
		        'holiday', '2026-08-10', '2026-08-21', 4.50, 'summer break', 'declared',
		        (SELECT id FROM users LIMIT 1))`)
	require.NoError(t, err, "valid availability_windows insert should succeed")

	// Invalid kind fails on CHECK.
	_, err = pool.Exec(ctx, `INSERT INTO availability_windows
		(org_id, user_id, kind, starts_on, ends_on, created_by)
		VALUES ((SELECT id FROM organizations LIMIT 1), (SELECT id FROM users LIMIT 1),
		        'sick', '2026-08-10', '2026-08-21', (SELECT id FROM users LIMIT 1))`)
	require.Error(t, err, "invalid kind must fail on CHECK")

	// Inverted dates fail on CHECK.
	_, err = pool.Exec(ctx, `INSERT INTO availability_windows
		(org_id, user_id, kind, starts_on, ends_on, created_by)
		VALUES ((SELECT id FROM organizations LIMIT 1), (SELECT id FROM users LIMIT 1),
		        'holiday', '2026-08-21', '2026-08-10', (SELECT id FROM users LIMIT 1))`)
	require.Error(t, err, "inverted dates must fail on CHECK")

	// --- DOWN ----------------------------------------------------------------
	_, err = pool.Exec(ctx, down012)
	require.NoError(t, err, "012 down should apply cleanly")

	// availability_windows and its index are gone.
	assertTableExists(t, pool, ctx, "availability_windows", false)
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pg_indexes WHERE indexname = 'idx_availability_windows_org_user_dates'`).Scan(&idxCount))
	assert.Zero(t, idxCount, "idx_availability_windows_org_user_dates must be dropped")

	// The three membership validity columns are gone.
	var colCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'organization_memberships'
		   AND column_name IN ('valid_from', 'valid_until', 'work_permit_expires_at')`).Scan(&colCount))
	assert.Zero(t, colCount, "validity columns must be dropped by 012 down")

	// role CHECK restored without 'hr': hr update now fails, bogus still fails.
	_, err = pool.Exec(ctx, `UPDATE organization_memberships
		SET role = 'hr' WHERE id = (SELECT id FROM organization_memberships LIMIT 1)`)
	require.Error(t, err, "role 'hr' must be rejected after 012 down")
	_, err = pool.Exec(ctx, `UPDATE organization_memberships
		SET role = 'bogus' WHERE id = (SELECT id FROM organization_memberships LIMIT 1)`)
	require.Error(t, err, "role 'bogus' must still fail after 012 down")

	// --- UP again (cycle) ----------------------------------------------------
	_, err = pool.Exec(ctx, up012)
	require.NoError(t, err, "012 up should re-apply cleanly after down")
	assertTableExists(t, pool, ctx, "availability_windows", true)
	_, err = pool.Exec(ctx, `UPDATE organization_memberships
		SET role = 'hr', updated_at = NOW() WHERE id = (SELECT id FROM organization_memberships LIMIT 1)`)
	require.NoError(t, err, "role 'hr' accepted again after re-apply")
}
