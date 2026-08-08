package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The three cycle tests below mirror the 014-017 shape
// (ontology_extension_migrations_test.go): apply pre-state migrations (skipping
// the migration under test) → apply up → assert schema shape + functional
// behavior → apply down → assert reversed → apply up again → assert still
// green (ADR-BE-004 up/down/up cycle).
//
// The functional assertions are the Pitfall-2 regression guards: every new
// CHECK must accept legacy-shaped (all-NULL discriminator) rows via the
// three-valued-logic guard, and reject rows that violate the per-type rules
// or miss a mandatory field (absorption reason / transfer justification).

// TestMigration018_ActivityBeneficiaryUnit_UpDownUpCycle verifies migration
// 018 (activities.beneficiary_unit_id): nullable column + index, set/read-back
// round-trip, NULL default valid for legacy rows (COV-05).
func TestMigration018_ActivityBeneficiaryUnit_UpDownUpCycle(t *testing.T) {
	pool := TestPool(t)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })
	ctx := context.Background()
	now := time.Now()

	up018 := readMigration(t, "018_activity_beneficiary_unit.up.sql")
	down018 := readMigration(t, "018_activity_beneficiary_unit.down.sql")

	// --- Pre-state: schema 000-017 + 019/020 (018 skipped so UP can apply) ---
	applyMigrations(t, pool, true, "018_activity_beneficiary_unit.up.sql")
	orgID := seedOrg(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	// --- UP ------------------------------------------------------------------
	_, err := pool.Exec(ctx, up018)
	require.NoError(t, err, "018 up should apply cleanly")

	// Column exists and is nullable (single nullable column — no 3VL CHECK,
	// mirrors contract_id per 011).
	var isNullable string
	require.NoError(t, pool.QueryRow(ctx, `SELECT is_nullable FROM information_schema.columns
		WHERE table_name = 'activities' AND column_name = 'beneficiary_unit_id'`).Scan(&isNullable))
	assert.Equal(t, "YES", isNullable, "beneficiary_unit_id must be nullable")

	var idxCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pg_indexes WHERE indexname = 'idx_activities_beneficiary_unit_id'`).Scan(&idxCount))
	assert.Equal(t, 1, idxCount, "idx_activities_beneficiary_unit_id must exist")

	// Functional: set + read back the beneficiary unit on the seeded activity.
	_, err = pool.Exec(ctx, `UPDATE activities SET beneficiary_unit_id = $1 WHERE id = $2`, unitID, activityID)
	require.NoError(t, err, "setting beneficiary_unit_id should succeed")
	var gotUnit uuid.UUID
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT beneficiary_unit_id FROM activities WHERE id = $1`, activityID).Scan(&gotUnit))
	assert.Equal(t, unitID, gotUnit, "beneficiary_unit_id must read back")

	// Legacy valid: a fresh activity defaults to NULL — passes untouched.
	legacyID := seedActivity(t, pool, orgID, "engagement", nil, now)
	var gotLegacy *uuid.UUID
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT beneficiary_unit_id FROM activities WHERE id = $1`, legacyID).Scan(&gotLegacy))
	assert.Nil(t, gotLegacy, "new activity must default to NULL beneficiary_unit_id")

	// --- DOWN ----------------------------------------------------------------
	_, err = pool.Exec(ctx, down018)
	require.NoError(t, err, "018 down should apply cleanly")

	var n int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'activities' AND column_name = 'beneficiary_unit_id'`).Scan(&n))
	assert.Zero(t, n, "beneficiary_unit_id must be dropped by 018 down")
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM pg_indexes
		WHERE indexname = 'idx_activities_beneficiary_unit_id'`).Scan(&n))
	assert.Zero(t, n, "idx_activities_beneficiary_unit_id must be dropped by 018 down")

	// --- UP again (cycle) ----------------------------------------------------
	_, err = pool.Exec(ctx, up018)
	require.NoError(t, err, "018 up should re-apply cleanly after down")
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'activities' AND column_name = 'beneficiary_unit_id'`).Scan(&n))
	assert.Equal(t, 1, n, "beneficiary_unit_id must exist after re-apply")
}

// TestMigration019_CoverageAllocations_UpDownUpCycle verifies migration 019
// (coverage_allocations): the six constraints exist by name; the 3VL guard is
// proven functionally — a legacy all-NULL source_type row passes, absorption
// without a unit fails on coverage_allocations_source_check (23514), transfer
// without justification fails on coverage_allocations_justification_check,
// a contract draw carrying both refs fails on source_check, and hours=0 fails
// the hours CHECK (Pitfall 2 warning signs, T-12-01/T-12-02).
func TestMigration019_CoverageAllocations_UpDownUpCycle(t *testing.T) {
	pool := TestPool(t)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })
	ctx := context.Background()
	now := time.Now()

	up019 := readMigration(t, "019_coverage_allocations.up.sql")
	down019 := readMigration(t, "019_coverage_allocations.down.sql")

	// --- Pre-state: schema 000-018 (019/020 skipped) -------------------------
	applyMigrations(t, pool, true,
		"019_coverage_allocations.up.sql", "020_coverage_snapshots.up.sql")
	orgID := seedOrg(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)

	// FK pre-state: a contracts row must exist for contract-scoped inserts.
	contractID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO contracts (id, name, km_rate, currency, governance_model, created_by_org_id, is_active, created_at, updated_at)
		VALUES ($1, 'Seed contract', 0, 'EUR', 'creator_controlled', $2, TRUE, $3, $3)`,
		contractID, orgID, now)
	require.NoError(t, err)

	// --- UP ------------------------------------------------------------------
	_, err = pool.Exec(ctx, up019)
	require.NoError(t, err, "019 up should apply cleanly")

	assertTableExists(t, pool, ctx, "coverage_allocations", true)
	for _, con := range []string{
		"coverage_allocations_source_check",
		"coverage_allocations_source_type_check",
		"coverage_allocations_reason_check",
		"coverage_allocations_justification_check",
		"coverage_allocations_reason_vocab_check",
		"coverage_allocations_entry_type_check",
	} {
		assertConstraintExists(t, pool, ctx, con)
	}

	// (a) Legacy-shaped row: source_type NULL, all refs NULL — must pass
	//     (three-valued-logic guard, Pitfall 2 regression guard).
	_, err = pool.Exec(ctx, `INSERT INTO coverage_allocations
		(org_id, entry_type, entry_id, source_type, contract_id, unit_id, hours, reason, justification, created_at, updated_at)
		VALUES ($1, 'time', $2, NULL, NULL, NULL, 8.00, NULL, NULL, $3, $3)`,
		orgID, uuid.New(), now)
	require.NoError(t, err, "legacy-shaped all-NULL source_type row must pass")

	// (b) Valid contract draw: source_type='contract' + contract_id, unit NULL.
	_, err = pool.Exec(ctx, `INSERT INTO coverage_allocations
		(org_id, entry_type, entry_id, source_type, contract_id, unit_id, hours, reason, justification, created_at, updated_at)
		VALUES ($1, 'time', $2, 'contract', $3, NULL, 8.00, NULL, NULL, $4, $4)`,
		orgID, uuid.New(), contractID, now)
	require.NoError(t, err, "valid contract draw must pass")

	// (c) Valid absorption: source_type='absorption' + unit_id + reason.
	_, err = pool.Exec(ctx, `INSERT INTO coverage_allocations
		(org_id, entry_type, entry_id, source_type, contract_id, unit_id, hours, reason, justification, created_at, updated_at)
		VALUES ($1, 'time', $2, 'absorption', NULL, $3, 2.50, 'Goodwill', NULL, $4, $4)`,
		orgID, uuid.New(), unitID, now)
	require.NoError(t, err, "valid absorption with unit and reason must pass")

	// (d) Absorption WITHOUT unit — must fail on coverage_allocations_source_check.
	_, err = pool.Exec(ctx, `INSERT INTO coverage_allocations
		(org_id, entry_type, entry_id, source_type, contract_id, unit_id, hours, reason, justification, created_at, updated_at)
		VALUES ($1, 'time', $2, 'absorption', NULL, NULL, 2.50, 'Goodwill', NULL, $3, $3)`,
		orgID, uuid.New(), now)
	require.Error(t, err, "absorption without unit must fail")
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23514", pgErr.Code, "absorption-without-unit must be a CHECK violation")
	assert.Equal(t, "coverage_allocations_source_check", pgErr.ConstraintName,
		"absorption-without-unit must trip the source check")

	// (e) Transfer WITHOUT justification — must fail on
	//     coverage_allocations_justification_check.
	_, err = pool.Exec(ctx, `INSERT INTO coverage_allocations
		(org_id, entry_type, entry_id, source_type, contract_id, unit_id, hours, reason, justification, created_at, updated_at)
		VALUES ($1, 'time', $2, 'transfer', $3, NULL, 4.00, NULL, NULL, $4, $4)`,
		orgID, uuid.New(), contractID, now)
	require.Error(t, err, "transfer without justification must fail")
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23514", pgErr.Code, "transfer-without-justification must be a CHECK violation")
	assert.Equal(t, "coverage_allocations_justification_check", pgErr.ConstraintName,
		"transfer-without-justification must trip the justification check")

	// (f) source_type='contract' with BOTH contract_id and unit_id — must fail
	//     on coverage_allocations_source_check (mixed refs, Pitfall 2).
	_, err = pool.Exec(ctx, `INSERT INTO coverage_allocations
		(org_id, entry_type, entry_id, source_type, contract_id, unit_id, hours, reason, justification, created_at, updated_at)
		VALUES ($1, 'time', $2, 'contract', $3, $4, 4.00, NULL, NULL, $5, $5)`,
		orgID, uuid.New(), contractID, unitID, now)
	require.Error(t, err, "contract draw with both refs must fail")
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23514", pgErr.Code, "mixed-refs insert must be a CHECK violation")
	assert.Equal(t, "coverage_allocations_source_check", pgErr.ConstraintName,
		"mixed-refs insert must trip the source check")

	// (g) hours = 0 — must fail the hours CHECK (000 time_entries parity).
	_, err = pool.Exec(ctx, `INSERT INTO coverage_allocations
		(org_id, entry_type, entry_id, source_type, contract_id, unit_id, hours, reason, justification, created_at, updated_at)
		VALUES ($1, 'time', $2, 'contract', $3, NULL, 0.00, NULL, NULL, $4, $4)`,
		orgID, uuid.New(), contractID, now)
	require.Error(t, err, "hours=0 must fail the hours CHECK")
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23514", pgErr.Code, "hours=0 insert must be a CHECK violation")

	// --- DOWN ----------------------------------------------------------------
	_, err = pool.Exec(ctx, down019)
	require.NoError(t, err, "019 down should apply cleanly")
	assertTableExists(t, pool, ctx, "coverage_allocations", false)

	// --- UP again (cycle) ----------------------------------------------------
	_, err = pool.Exec(ctx, up019)
	require.NoError(t, err, "019 up should re-apply cleanly after down")
	assertTableExists(t, pool, ctx, "coverage_allocations", true)
	assertConstraintExists(t, pool, ctx, "coverage_allocations_source_check")
}

// TestMigration020_CoverageSnapshots_UpDownUpCycle verifies migration 020
// (coverage_period_closes + coverage_snapshot_rows): both tables, both
// indexes, and the ON DELETE CASCADE from the close header to its rows —
// proven functionally by deleting the header and asserting the rows cascade
// away (append-only: the header delete is the only delete path, COV-04).
func TestMigration020_CoverageSnapshots_UpDownUpCycle(t *testing.T) {
	pool := TestPool(t)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })
	ctx := context.Background()
	now := time.Now()

	up020 := readMigration(t, "020_coverage_snapshots.up.sql")
	down020 := readMigration(t, "020_coverage_snapshots.down.sql")

	// --- Pre-state: schema 000-019 (020 skipped) -----------------------------
	applyMigrations(t, pool, true, "020_coverage_snapshots.up.sql")
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	// --- UP ------------------------------------------------------------------
	_, err := pool.Exec(ctx, up020)
	require.NoError(t, err, "020 up should apply cleanly")

	assertTableExists(t, pool, ctx, "coverage_period_closes", true)
	assertTableExists(t, pool, ctx, "coverage_snapshot_rows", true)

	for _, idx := range []string{"idx_coverage_snapshot_rows_close", "idx_coverage_snapshot_rows_entry"} {
		var idxCount int
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM pg_indexes WHERE indexname = $1`, idx).Scan(&idxCount))
		assert.Equal(t, 1, idxCount, "%s must exist", idx)
	}

	// Functional: insert a close header + one snapshot row, delete the header,
	// and assert the row cascades away.
	closeID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO coverage_period_closes (id, org_id, period_start, period_end, closed_by, closed_at)
		VALUES ($1, $2, DATE '2026-07-01', DATE '2026-07-31', $3, $4)`,
		closeID, orgID, userID, now)
	require.NoError(t, err, "close header insert should succeed")

	_, err = pool.Exec(ctx, `INSERT INTO coverage_snapshot_rows
		(close_id, entry_id, employee_id, entry_date, activity_id, source_type, contract_id, unit_id, hours, reason, justification)
		VALUES ($1, $2, $3, DATE '2026-07-15', $4, 'contract', NULL, NULL, 8.00, NULL, NULL)`,
		closeID, uuid.New(), userID, uuid.New())
	require.NoError(t, err, "snapshot row insert should succeed")

	var rowCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM coverage_snapshot_rows WHERE close_id = $1`, closeID).Scan(&rowCount))
	assert.Equal(t, 1, rowCount, "snapshot row must be present before close delete")

	_, err = pool.Exec(ctx, `DELETE FROM coverage_period_closes WHERE id = $1`, closeID)
	require.NoError(t, err, "close header delete should succeed")
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM coverage_snapshot_rows WHERE close_id = $1`, closeID).Scan(&rowCount))
	assert.Zero(t, rowCount, "snapshot rows must cascade with the close header")

	// --- DOWN ----------------------------------------------------------------
	_, err = pool.Exec(ctx, down020)
	require.NoError(t, err, "020 down should apply cleanly")
	assertTableExists(t, pool, ctx, "coverage_snapshot_rows", false)
	assertTableExists(t, pool, ctx, "coverage_period_closes", false)

	// --- UP again (cycle) ----------------------------------------------------
	_, err = pool.Exec(ctx, up020)
	require.NoError(t, err, "020 up should re-apply cleanly after down")
	assertTableExists(t, pool, ctx, "coverage_period_closes", true)
	assertTableExists(t, pool, ctx, "coverage_snapshot_rows", true)
}
