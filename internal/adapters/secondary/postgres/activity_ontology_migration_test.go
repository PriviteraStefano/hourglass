package postgres

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

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
	applyMigrations(t, pool, true, "011_activity_ontology.up.sql", "013_activity_kind_phase_fix.up.sql")
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
