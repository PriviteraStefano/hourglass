package postgres

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readMigration(t *testing.T, name string) string {
	t.Helper()
	root := findProjectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "migrations", name))
	require.NoError(t, err)
	return string(content)
}

// applyMigrations applies every *.up.sql migration in sorted order (the same
// order SetupTestSchema and cmd/migrate use), optionally including seed files,
// skipping the named files.
func applyMigrations(t *testing.T, pool *pgxpool.Pool, withSeed bool, skip ...string) {
	t.Helper()
	root := findProjectRoot(t)
	files, err := filepath.Glob(filepath.Join(root, "migrations", "*.up.sql"))
	require.NoError(t, err)
	sort.Strings(files)
	for _, f := range files {
		base := filepath.Base(f)
		if slices.Contains(skip, base) {
			continue
		}
		if strings.Contains(base, "seed") && !withSeed {
			continue
		}
		content, err := os.ReadFile(f)
		require.NoError(t, err)
		_, err = pool.Exec(context.Background(), string(content))
		require.NoError(t, err, "migration %s failed: %v", base, err)
	}
}

func assertCount(t *testing.T, pool *pgxpool.Pool, ctx context.Context, q string, want int) {
	t.Helper()
	var got int
	require.NoError(t, pool.QueryRow(ctx, q).Scan(&got), "query failed: %s", q)
	assert.Equal(t, want, got, "count mismatch for: %s", q)
}

func assertTableExists(t *testing.T, pool *pgxpool.Pool, ctx context.Context, table string, want bool) {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = $1`, table).Scan(&n))
	assert.Equal(t, want, n == 1, "table %s existence mismatch (want %v, got %d)", table, want, n)
}

func assertColumnNotNull(t *testing.T, pool *pgxpool.Pool, ctx context.Context, table, column string) {
	t.Helper()
	var nullable string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT is_nullable FROM information_schema.columns
		 WHERE table_name = $1 AND column_name = $2`, table, column).Scan(&nullable))
	assert.Equal(t, "NO", nullable, "%s.%s must be NOT NULL", table, column)
}

// assertFkAction checks the referential action of a FK constraint
// (confdeltype: 'r' = RESTRICT, 'c' = CASCADE, 'a' = NO ACTION).
func assertFkAction(t *testing.T, pool *pgxpool.Pool, ctx context.Context, conname string, want string) {
	t.Helper()
	var got string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT confdeltype FROM pg_constraint WHERE conname = $1 AND contype = 'f'`, conname).Scan(&got))
	assert.Equal(t, want, got, "FK %s delete action mismatch", conname)
}

func assertConstraintExists(t *testing.T, pool *pgxpool.Pool, ctx context.Context, conname string) {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pg_constraint WHERE conname = $1`, conname).Scan(&n))
	assert.Equal(t, 1, n, "constraint %s should exist", conname)
}

// seedOrgID is the historical MVP seed organization UUID (003_seed.up.sql).
// Migration 011 seeds the four canonical activity_kinds ONLY for this org
// (011_activity_ontology.up.sql line 40), and the activities INSERT inherits
// the (org_id, kind) catalog FK — the pre-state org MUST use this exact id.
var seedOrgID = uuid.MustParse("019df8b0-0001-7000-8000-000000000001")

// seedLegacyMVPPrestate seeds the two-level model rows the 011 data migration
// rewrites, mirroring the historical MVP seed shapes the test asserts
// (6 projects / 6 subprojects / 6 working groups / 12 time entries / 6
// expenses with 4 project-linked and 2 NULL-project). The historical
// 003_seed.up.sql is no longer a migration (seed data moved to
// scripts/seed_demo.sql, which tests must not load), so the pre-state is
// seeded inline with helpers where they cover the table and direct SQL for
// the rows helpers don't cover. Assertions keep their original expected
// values — only the seeding strategy changes (ADR-BE-004: migrations
// untouched).
func seedLegacyMVPPrestate(t *testing.T, pool *pgxpool.Pool, ctx context.Context, now time.Time) {
	t.Helper()

	// Organization — fixed UUID so 011's kind-catalog seed row targets it.
	_, err := pool.Exec(ctx,
		`INSERT INTO organizations (id, name, slug, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		seedOrgID, "Tech Consulting Group", "tc-group-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)

	// Users (manager + entry author) and one unit (time_entries/expenses FKs).
	userID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, seedOrgID, now)

	// 6 projects (4 billable + 2 internal).
	projectIDs := make([]uuid.UUID, 6)
	for i := range projectIDs {
		id := uuid.New()
		projectIDs[i] = id
		projectType := "billable"
		if i >= 4 {
			projectType = "internal"
		}
		_, err = pool.Exec(ctx,
			`INSERT INTO projects (id, org_id, name, description, project_type, type,
			                      governance_model, created_by_org_id, is_shared, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $5, 'creator_controlled', $2, FALSE, TRUE, $6, $6)`,
			id, seedOrgID, "Project "+id.String()[:8], "Seed project", projectType, now)
		require.NoError(t, err)
	}

	// 6 subprojects (one per project).
	subprojectIDs := make([]uuid.UUID, 6)
	for i := range subprojectIDs {
		id := uuid.New()
		subprojectIDs[i] = id
		_, err = pool.Exec(ctx,
			`INSERT INTO subprojects (id, project_id, name, description, sequence_order, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, TRUE, $6, $6)`,
			id, projectIDs[i], "Subproject "+id.String()[:8], "Seed subproject", i+1, now)
		require.NoError(t, err)
	}

	// 6 working groups (one per subproject, same manager).
	wgIDs := make([]uuid.UUID, 6)
	for i := range wgIDs {
		id := uuid.New()
		wgIDs[i] = id
		_, err = pool.Exec(ctx,
			`INSERT INTO working_groups (id, org_id, subproject_id, name, description, unit_ids,
			                             enforce_unit_tuple, manager_id, delegate_ids, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, ARRAY[$6]::UUID[], TRUE, $7, ARRAY[]::UUID[], TRUE, $8, $8)`,
			id, seedOrgID, subprojectIDs[i], "WG "+id.String()[:8], "Seed working group", unitID, userID, now)
		require.NoError(t, err)
	}

	// 12 time entries (2 per subproject).
	for i := range subprojectIDs {
		for j := 0; j < 2; j++ {
			_, err = pool.Exec(ctx,
				`INSERT INTO time_entries (org_id, user_id, project_id, subproject_id, wg_id, unit_id,
				                           hours, description, entry_date, status, is_deleted, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $6, 7.5, $7, $8, 'submitted', FALSE, $9, $9)`,
				seedOrgID, userID, projectIDs[i], subprojectIDs[i], wgIDs[i], unitID,
				"Seed time entry", now, now)
			require.NoError(t, err)
		}
	}

	// 6 expenses: 4 project-linked, 2 with project_id NULL (internal spend).
	for i := 0; i < 6; i++ {
		var projectID any // nil for the last two
		if i < 4 {
			projectID = projectIDs[i]
		}
		_, err = pool.Exec(ctx,
			`INSERT INTO expenses (org_id, user_id, project_id, unit_id, category, amount,
			                       currency, description, expense_date, status, is_deleted, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, 'mileage', 15.00, 'EUR', $5, $6, 'submitted', FALSE, $7, $7)`,
			seedOrgID, userID, projectID, unitID, "Seed expense", now, now)
		require.NoError(t, err)
	}
}

// TestMigration011_ActivityOntology_UpDownUpCycle verifies migration 011
// (activity ontology) against the MVP seed data:
//   - up applies cleanly, rewrites the schema exactly per ADR-P-007, and
//     migrates every project/subproject/time entry/expense with zero orphans
//   - down applies cleanly and restores the two-level model 1:1
//   - up → down → up cycle completes without error
func TestMigration011_ActivityOntology_UpDownUpCycle(t *testing.T) {
	pool := TestPool(t)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })
	ctx := context.Background()

	up011 := readMigration(t, "011_activity_ontology.up.sql")
	down011 := readMigration(t, "011_activity_ontology.down.sql")
	up013 := readMigration(t, "013_activity_kind_phase_fix.up.sql")
	down013 := readMigration(t, "013_activity_kind_phase_fix.down.sql")

	// --- Pre-state: schema 000-010 + MVP seed -------------------------------
	// 013 is skipped in pre-state: it UPDATEs the activities table that only
	// exists after 011 — applied before 011 it would fail with SQLSTATE 42P01.
	// 014-017 are skipped too: 015 ALTERs activities (which only exists after
	// 011) — the pre-state must stay exactly at 000-010 per ADR-BE-004.
	applyMigrations(t, pool, true, "011_activity_ontology.up.sql", "013_activity_kind_phase_fix.up.sql",
		"014_ticket_schema.up.sql", "015_activity_origins.up.sql",
		"016_contract_sold_hours.up.sql", "017_audit_logs.up.sql")
	// The historical MVP seed (003_seed.up.sql) is no longer a migration
	// fixture — seed data lives in scripts/seed_demo.sql which applyMigrations
	// never loads. Self-seed the two-level pre-state the 011 data migration
	// asserts against (6 projects / 6 subprojects / 12 time entries / 6
	// expenses / 6 working groups).
	seedLegacyMVPPrestate(t, pool, ctx, time.Now())
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM projects", 6)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM subprojects", 6)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM time_entries", 12)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM expenses", 6)

	// --- UP ------------------------------------------------------------------
	_, err := pool.Exec(ctx, up011)
	require.NoError(t, err, "011 up should apply cleanly")

	// Forward label fix (SPEC acceptance #6, ADR-BE-004 append-only): the
	// subproject-derived rows hardcoded kind='task' by 011 line 115 are
	// relabeled 'phase' by 013.
	_, err = pool.Exec(ctx, up013)
	require.NoError(t, err, "013 up should apply cleanly")

	// Old two-level tables are gone.
	assertTableExists(t, pool, ctx, "projects", false)
	assertTableExists(t, pool, ctx, "subprojects", false)
	assertTableExists(t, pool, ctx, "project_adoptions", false)
	assertTableExists(t, pool, ctx, "project_managers", false)

	// New ontology tables exist.
	assertTableExists(t, pool, ctx, "activity_kinds", true)
	assertTableExists(t, pool, ctx, "activities", true)
	assertTableExists(t, pool, ctx, "activity_adoptions", true)
	assertTableExists(t, pool, ctx, "activity_managers", true)

	// enforce_unit_tuple is dropped (ADR-P-001 Q3).
	var n int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'working_groups' AND column_name = 'enforce_unit_tuple'`).Scan(&n))
	assert.Zero(t, n, "enforce_unit_tuple must be dropped")

	// Both entry types pin activity_id NOT NULL (ADR-P-007 D-4).
	assertColumnNotNull(t, pool, ctx, "time_entries", "activity_id")
	assertColumnNotNull(t, pool, ctx, "expenses", "activity_id")

	// FK actions match the ADR-P-007 sketch (parent_id/contract_id RESTRICT).
	assertFkAction(t, pool, ctx, "activities_parent_id_fkey", "r")
	assertFkAction(t, pool, ctx, "activities_contract_id_fkey", "r")
	assertConstraintExists(t, pool, ctx, "activities_governance_model_check")

	// Seed data: 6 projects (engagement) + 6 subprojects (phase after 013) +
	// 1 internal fallback activity (for the 2 NULL-project expenses) = 13.
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM activities", 13)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM activities WHERE kind = 'engagement'", 6)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM activities WHERE kind = 'phase'", 6)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM activities WHERE kind = 'task'", 0)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM activities WHERE kind = 'internal'", 1)

	// Zero orphaned entries after migration.
	assertCount(t, pool, ctx, `SELECT COUNT(*) FROM time_entries te
		LEFT JOIN activities a ON a.id = te.activity_id WHERE a.id IS NULL`, 0)
	assertCount(t, pool, ctx, `SELECT COUNT(*) FROM expenses e
		LEFT JOIN activities a ON a.id = e.activity_id WHERE a.id IS NULL`, 0)

	// Working groups re-anchored to activities (1:1 with subprojects).
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM working_groups", 6)
	assertCount(t, pool, ctx, `SELECT COUNT(*) FROM working_groups wg
		JOIN activities a ON a.id = wg.activity_id`, 6)

	// The internal fallback absorbed the two non-project expenses.
	assertCount(t, pool, ctx, `SELECT COUNT(*) FROM expenses e
		JOIN activities a ON a.id = e.activity_id WHERE a.kind = 'internal'`, 2)

	// Seed kinds exist for the MVP org.
	assertCount(t, pool, ctx, `SELECT COUNT(*) FROM activity_kinds WHERE is_seed = TRUE`, 4)

	// --- DOWN ----------------------------------------------------------------
	// 013 down first: the label fix must reverse before 011 down rewrites the
	// schema. The relabeled phase rows return to 'task' — proving down013
	// reverses exactly the same row set.
	_, err = pool.Exec(ctx, down013)
	require.NoError(t, err, "013 down should apply cleanly")
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM activities WHERE kind = 'task'", 6)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM activities WHERE kind = 'phase'", 0)

	_, err = pool.Exec(ctx, down011)
	require.NoError(t, err, "011 down should apply cleanly")

	// New ontology tables are gone.
	assertTableExists(t, pool, ctx, "activities", false)
	assertTableExists(t, pool, ctx, "activity_kinds", false)
	assertTableExists(t, pool, ctx, "activity_adoptions", false)
	assertTableExists(t, pool, ctx, "activity_managers", false)

	// Two-level model restored 1:1 (no rows lost).
	assertTableExists(t, pool, ctx, "projects", true)
	assertTableExists(t, pool, ctx, "subprojects", true)
	assertTableExists(t, pool, ctx, "project_managers", true)
	assertTableExists(t, pool, ctx, "project_adoptions", true)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM projects", 6)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM subprojects", 6)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM project_managers", 0)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM project_adoptions", 0)

	// time_entries restored with all three FKs populated.
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM time_entries", 12)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM time_entries WHERE project_id IS NULL", 0)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM time_entries WHERE subproject_id IS NULL", 0)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM time_entries WHERE wg_id IS NULL", 0)

	// expenses: 4 keep their project, 2 return to NULL (pre-011 state).
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM expenses", 6)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM expenses WHERE project_id IS NOT NULL", 4)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM expenses WHERE project_id IS NULL", 2)

	// working_groups restored with enforce_unit_tuple default TRUE.
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM working_groups", 6)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM working_groups WHERE enforce_unit_tuple = TRUE", 6)

	// financial_cutoff_periods / budget_caps back to project_id.
	assertCount(t, pool, ctx, `SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'financial_cutoff_periods' AND column_name = 'project_id'`, 1)
	assertCount(t, pool, ctx, `SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'budget_caps' AND column_name = 'project_id'`, 1)
	assertCount(t, pool, ctx, `SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'working_groups' AND column_name = 'enforce_unit_tuple'`, 1)

	// --- UP again (cycle) -----------------------------------------------------
	_, err = pool.Exec(ctx, up011)
	require.NoError(t, err, "011 up should re-apply cleanly after down")
	_, err = pool.Exec(ctx, up013)
	require.NoError(t, err, "013 up should re-apply cleanly after down")
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM activities", 13)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM activities WHERE kind = 'phase'", 6)
	assertCount(t, pool, ctx, "SELECT COUNT(*) FROM activities WHERE kind = 'task'", 0)
	assertCount(t, pool, ctx, `SELECT COUNT(*) FROM time_entries te
		LEFT JOIN activities a ON a.id = te.activity_id WHERE a.id IS NULL`, 0)
	assertCount(t, pool, ctx, `SELECT COUNT(*) FROM expenses e
		LEFT JOIN activities a ON a.id = e.activity_id WHERE a.id IS NULL`, 0)
}
