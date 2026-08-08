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

// The four cycle tests below mirror the 012 skeleton (staffing_schema_migration_test.go):
// apply pre-state migrations (skipping the migration under test) → apply up →
// assert schema shape + functional behavior → apply down → assert reversed →
// apply up again → assert still green (ADR-BE-004 up/down/up cycle).
//
// The functional assertions are the Pitfall-1 regression guards: every new
// CHECK must accept legacy-shaped (all-NULL discriminator) rows via the
// three-valued-logic guard, and reject rows that violate the per-type rules.

// TestMigration014_TicketSchema_UpDownUpCycle verifies migration 014
// (tickets + ticket_comments): tables, kind/status CHECK vocabulary,
// CASCADE delete action, and round-trip insert/read.
func TestMigration014_TicketSchema_UpDownUpCycle(t *testing.T) {
	pool := TestPool(t)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })
	ctx := context.Background()
	now := time.Now()

	up014 := readMigration(t, "014_ticket_schema.up.sql")
	down014 := readMigration(t, "014_ticket_schema.down.sql")

	// --- Pre-state: schema 000-013 (014-017 skipped) -------------------------
	applyMigrations(t, pool, true,
		"014_ticket_schema.up.sql", "015_activity_origins.up.sql",
		"016_contract_sold_hours.up.sql", "017_audit_logs.up.sql",
		"018_activity_beneficiary_unit.up.sql", "019_coverage_allocations.up.sql",
		"020_coverage_snapshots.up.sql")
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	// --- UP ------------------------------------------------------------------
	_, err := pool.Exec(ctx, up014)
	require.NoError(t, err, "014 up should apply cleanly")

	assertTableExists(t, pool, ctx, "tickets", true)
	assertTableExists(t, pool, ctx, "ticket_comments", true)
	assertConstraintExists(t, pool, ctx, "tickets_kind_check")
	assertConstraintExists(t, pool, ctx, "tickets_status_check")
	// Comments cascade when their ticket is deleted (014 spec).
	assertFkAction(t, pool, ctx, "ticket_comments_ticket_id_fkey", "c")

	// Functional: insert a ticket + comment, both read back.
	ticketID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO tickets (id, org_id, title, kind, status, requester_id, created_at, updated_at)
		VALUES ($1, $2, 'Seed ticket', 'bug', 'open', $3, $4, $4)`,
		ticketID, orgID, userID, now)
	require.NoError(t, err, "valid ticket insert should succeed")
	_, err = pool.Exec(ctx, `INSERT INTO ticket_comments (ticket_id, author_id, body, created_at)
		VALUES ($1, $2, 'Seed comment', $3)`, ticketID, userID, now)
	require.NoError(t, err, "valid comment insert should succeed")

	var title, body string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT title FROM tickets WHERE id = $1`, ticketID).Scan(&title))
	assert.Equal(t, "Seed ticket", title)
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT body FROM ticket_comments WHERE ticket_id = $1`, ticketID).Scan(&body))
	assert.Equal(t, "Seed comment", body)

	// --- DOWN ----------------------------------------------------------------
	_, err = pool.Exec(ctx, down014)
	require.NoError(t, err, "014 down should apply cleanly")
	assertTableExists(t, pool, ctx, "tickets", false)
	assertTableExists(t, pool, ctx, "ticket_comments", false)

	// --- UP again (cycle) ----------------------------------------------------
	_, err = pool.Exec(ctx, up014)
	require.NoError(t, err, "014 up should re-apply cleanly after down")
	assertTableExists(t, pool, ctx, "tickets", true)
	assertTableExists(t, pool, ctx, "ticket_comments", true)
	assertConstraintExists(t, pool, ctx, "tickets_kind_check")
	assertConstraintExists(t, pool, ctx, "tickets_status_check")
}

// TestMigration015_ActivityOrigins_UpDownUpCycle verifies migration 015
// (origins on activities): the discriminator + per-type ref pins, with the
// three-valued-logic guard proven functionally — a legacy all-NULL row
// passes, a valid customer_ticket row passes, a mixed-refs row fails on the
// refs CHECK (Pitfall 1 warning sign).
func TestMigration015_ActivityOrigins_UpDownUpCycle(t *testing.T) {
	pool := TestPool(t)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })
	ctx := context.Background()
	now := time.Now()

	up015 := readMigration(t, "015_activity_origins.up.sql")
	down015 := readMigration(t, "015_activity_origins.down.sql")

	// --- Pre-state: schema 000-014 (014 applies so activities.ticket_id
	// resolves against tickets; 016/017 skipped) ------------------------------
	applyMigrations(t, pool, true,
		"015_activity_origins.up.sql", "016_contract_sold_hours.up.sql",
		"017_audit_logs.up.sql", "018_activity_beneficiary_unit.up.sql",
		"019_coverage_allocations.up.sql", "020_coverage_snapshots.up.sql")
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	seedActivityKind(t, pool, orgID, "engagement")

	// FK pre-state: a tickets row must exist before the valid customer_ticket
	// activity insert (mirrors the 014 test's ticket insert).
	ticketID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO tickets (id, org_id, title, kind, status, requester_id, created_at, updated_at)
		VALUES ($1, $2, 'Seed ticket', 'bug', 'open', $3, $4, $4)`,
		ticketID, orgID, userID, now)
	require.NoError(t, err)

	// --- UP ------------------------------------------------------------------
	_, err = pool.Exec(ctx, up015)
	require.NoError(t, err, "015 up should apply cleanly")

	assertConstraintExists(t, pool, ctx, "activities_origin_type_check")
	assertConstraintExists(t, pool, ctx, "activities_origin_refs_check")
	assertFkAction(t, pool, ctx, "activities_ticket_id_fkey", "a")

	// (a) Legacy-shaped row: ALL origin columns NULL — must succeed
	//     (three-valued-logic guard, Pitfall 1 regression guard).
	legacyID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO activities (id, org_id, name, kind, governance_model, created_by_org_id, is_active, created_at, updated_at)
		VALUES ($1, $2, 'Legacy activity', 'engagement', 'creator_controlled', $2, TRUE, $3, $3)`,
		legacyID, orgID, now)
	require.NoError(t, err, "legacy-shaped all-NULL origin row must pass the refs check")

	// (b) Valid customer_ticket row: origin_type + ticket_id, others NULL.
	validID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO activities (id, org_id, name, kind, governance_model, created_by_org_id, origin_type, ticket_id, is_active, created_at, updated_at)
		VALUES ($1, $2, 'Customer ticket activity', 'engagement', 'creator_controlled', $2, 'customer_ticket', $3, TRUE, $4, $4)`,
		validID, orgID, ticketID, now)
	require.NoError(t, err, "valid customer_ticket origin row must pass the refs check")

	// (c) Mixed-refs row: customer_ticket WITH assigned_by set — must fail
	//     on activities_origin_refs_check (Pitfall 1 warning sign).
	_, err = pool.Exec(ctx, `INSERT INTO activities (id, org_id, name, kind, governance_model, created_by_org_id, origin_type, ticket_id, assigned_by, is_active, created_at, updated_at)
		VALUES ($1, $2, 'Mixed refs activity', 'engagement', 'creator_controlled', $2, 'customer_ticket', $3, $4, TRUE, $5, $5)`,
		uuid.New(), orgID, ticketID, userID, now)
	require.Error(t, err, "mixed-refs row must fail the refs check")
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23514", pgErr.Code, "mixed-refs insert must be a CHECK violation")
	assert.Equal(t, "activities_origin_refs_check", pgErr.ConstraintName,
		"mixed-refs insert must trip the origin refs check")

	// --- DOWN ----------------------------------------------------------------
	_, err = pool.Exec(ctx, down015)
	require.NoError(t, err, "015 down should apply cleanly")

	var n int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'activities'
		  AND column_name IN ('origin_type','assigned_by','assigned_to','proposed_by','reviewed_by','ticket_id')`).Scan(&n))
	assert.Zero(t, n, "origin columns must be dropped by 015 down")
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM pg_constraint
		WHERE conname IN ('activities_origin_refs_check','activities_origin_type_check')`).Scan(&n))
	assert.Zero(t, n, "origin constraints must be dropped by 015 down")

	// --- UP again (cycle) ----------------------------------------------------
	_, err = pool.Exec(ctx, up015)
	require.NoError(t, err, "015 up should re-apply cleanly after down")
	assertConstraintExists(t, pool, ctx, "activities_origin_type_check")
	assertConstraintExists(t, pool, ctx, "activities_origin_refs_check")
}

// TestMigration016_ContractSoldHours_UpDownUpCycle verifies migration 016
// (contract_type/sold_hours/sold_period): legacy NULL rows pass, support
// contracts require both sold_hours and sold_period, project contracts must
// not carry sold_period.
func TestMigration016_ContractSoldHours_UpDownUpCycle(t *testing.T) {
	pool := TestPool(t)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })
	ctx := context.Background()
	now := time.Now()

	up016 := readMigration(t, "016_contract_sold_hours.up.sql")
	down016 := readMigration(t, "016_contract_sold_hours.down.sql")

	// --- Pre-state: schema 000-015 (016/017 skipped) -------------------------
	applyMigrations(t, pool, true, "016_contract_sold_hours.up.sql", "017_audit_logs.up.sql",
		"018_activity_beneficiary_unit.up.sql", "019_coverage_allocations.up.sql",
		"020_coverage_snapshots.up.sql")
	orgID := seedOrg(t, pool, now)

	// --- UP ------------------------------------------------------------------
	_, err := pool.Exec(ctx, up016)
	require.NoError(t, err, "016 up should apply cleanly")

	assertConstraintExists(t, pool, ctx, "contracts_sold_check")

	// (a) Legacy row: contract_type NULL — must succeed.
	_, err = pool.Exec(ctx, `INSERT INTO contracts (name, km_rate, currency, governance_model, created_by_org_id, is_active, created_at, updated_at)
		VALUES ('Legacy contract', 0, 'EUR', 'creator_controlled', $1, TRUE, $2, $2)`,
		orgID, now)
	require.NoError(t, err, "legacy contract_type NULL row must pass the sold check")

	// (b) 'support' with sold_hours + sold_period — must succeed.
	_, err = pool.Exec(ctx, `INSERT INTO contracts (name, km_rate, currency, governance_model, created_by_org_id, contract_type, sold_hours, sold_period, is_active, created_at, updated_at)
		VALUES ('Support contract', 0, 'EUR', 'creator_controlled', $1, 'support', 100.00, '2026-08', TRUE, $2, $2)`,
		orgID, now)
	require.NoError(t, err, "support contract with sold_hours and sold_period must pass")

	// (c) 'support' without sold_period — must fail on contracts_sold_check.
	_, err = pool.Exec(ctx, `INSERT INTO contracts (name, km_rate, currency, governance_model, created_by_org_id, contract_type, sold_hours, sold_period, is_active, created_at, updated_at)
		VALUES ('Broken support contract', 0, 'EUR', 'creator_controlled', $1, 'support', 100.00, NULL, TRUE, $2, $2)`,
		orgID, now)
	require.Error(t, err, "support contract without sold_period must fail")
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23514", pgErr.Code, "support-without-period insert must be a CHECK violation")
	assert.Equal(t, "contracts_sold_check", pgErr.ConstraintName)

	// (d) 'project' with sold_period — must fail on contracts_sold_check.
	_, err = pool.Exec(ctx, `INSERT INTO contracts (name, km_rate, currency, governance_model, created_by_org_id, contract_type, sold_hours, sold_period, is_active, created_at, updated_at)
		VALUES ('Broken project contract', 0, 'EUR', 'creator_controlled', $1, 'project', NULL, '2026-08', TRUE, $2, $2)`,
		orgID, now)
	require.Error(t, err, "project contract with sold_period must fail")
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "contracts_sold_check", pgErr.ConstraintName)

	// --- DOWN ----------------------------------------------------------------
	_, err = pool.Exec(ctx, down016)
	require.NoError(t, err, "016 down should apply cleanly")
	var n int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM pg_constraint
		WHERE conname = 'contracts_sold_check'`).Scan(&n))
	assert.Zero(t, n, "contracts_sold_check must be dropped by 016 down")

	// --- UP again (cycle) ----------------------------------------------------
	_, err = pool.Exec(ctx, up016)
	require.NoError(t, err, "016 up should re-apply cleanly after down")
	assertConstraintExists(t, pool, ctx, "contracts_sold_check")
}

// TestMigration017_AuditLogs_UpDownUpCycle verifies migration 017
// (audit_logs): table, entity index, and JSONB payload round-trip.
func TestMigration017_AuditLogs_UpDownUpCycle(t *testing.T) {
	pool := TestPool(t)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })
	ctx := context.Background()
	now := time.Now()

	up017 := readMigration(t, "017_audit_logs.up.sql")
	down017 := readMigration(t, "017_audit_logs.down.sql")

	// --- Pre-state: schema 000-016 (017 skipped) -----------------------------
	applyMigrations(t, pool, true, "017_audit_logs.up.sql",
		"018_activity_beneficiary_unit.up.sql", "019_coverage_allocations.up.sql",
		"020_coverage_snapshots.up.sql")
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	// --- UP ------------------------------------------------------------------
	_, err := pool.Exec(ctx, up017)
	require.NoError(t, err, "017 up should apply cleanly")

	assertTableExists(t, pool, ctx, "audit_logs", true)
	var idxCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pg_indexes WHERE indexname = 'idx_audit_logs_entity'`).Scan(&idxCount))
	assert.Equal(t, 1, idxCount, "idx_audit_logs_entity must exist")

	// Functional: insert an audit row (ticket dismissal, TICK-04 shape) and
	// assert the JSONB payload reads back intact.
	entityID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO audit_logs (org_id, entity_type, entity_id, action, actor_id, comment, payload, created_at)
		VALUES ($1, 'ticket', $2, 'dismissed', $3, 'Dismissed with hours logged', '{"dismissed_hours": 12.5}'::jsonb, $4)`,
		orgID, entityID, userID, now)
	require.NoError(t, err, "audit_logs insert should succeed")

	var payloadEq bool
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT payload = '{"dismissed_hours": 12.5}'::jsonb FROM audit_logs WHERE entity_id = $1`, entityID).Scan(&payloadEq))
	assert.True(t, payloadEq, "payload JSONB must read back intact")

	// --- DOWN ----------------------------------------------------------------
	_, err = pool.Exec(ctx, down017)
	require.NoError(t, err, "017 down should apply cleanly")
	assertTableExists(t, pool, ctx, "audit_logs", false)

	// --- UP again (cycle) ----------------------------------------------------
	_, err = pool.Exec(ctx, up017)
	require.NoError(t, err, "017 up should re-apply cleanly after down")
	assertTableExists(t, pool, ctx, "audit_logs", true)
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pg_indexes WHERE indexname = 'idx_audit_logs_entity'`).Scan(&idxCount))
	assert.Equal(t, 1, idxCount, "idx_audit_logs_entity must exist after re-apply")
}
