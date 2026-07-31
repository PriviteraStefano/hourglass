package postgres

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// TestPool returns a pool connected to the test database.
// It starts a PostgreSQL container via testcontainers-go (via SetupPackageContainer)
// so Docker must be running. No DATABASE_URL env var is needed.
func TestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return SetupPackageContainer(t)
}

// findProjectRoot walks up from CWD to locate the Go module root (where go.mod lives).
// This ensures migration files resolve correctly regardless of the calling package's test directory.
func findProjectRoot(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("project root not found (no go.mod in any parent directory from %s)", dir)
			return ""
		}
		dir = parent
	}
}

func migrationGlob(t testing.TB) []string {
	t.Helper()
	root := findProjectRoot(t)
	pattern := filepath.Join(root, "migrations", "*.up.sql")
	files, err := filepath.Glob(pattern)
	require.NoError(t, err, "failed to glob migration files with pattern %s", pattern)
	if len(files) == 0 {
		t.Fatalf("no migration .up.sql files found in %s", filepath.Join(root, "migrations"))
	}
	return files
}

// SetupTestSchema reads and applies all migration .up.sql files from the
// migrations directory (excluding seed files), sorted alphabetically.
// The migration path is resolved relative to the Go module root, so this
// works correctly from any package's test directory.
func SetupTestSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	files := migrationGlob(t)
	sort.Strings(files)
	for _, f := range files {
		if strings.Contains(f, "seed") {
			continue
		}
		content, err := os.ReadFile(f)
		require.NoError(t, err)
		_, err = pool.Exec(context.Background(), string(content))
		if err != nil {
			t.Logf("Migration %s: %v", filepath.Base(f), err)
		}
	}
}

// TeardownTestSchema drops all tables in dependency order for cleanup
// between test packages.
func TeardownTestSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	tables := []string{
		"backup_approvers",
		"budget_caps",
		"financial_cutoff_periods",
		"availability_windows",
		"refresh_tokens",
		"expense_approvals",
		"password_resets",
		"invitations",
		"time_entry_approvals",
		"expenses",
		"time_entries",
		"wg_members",
		"working_groups",
		"subprojects",
		"project_managers",
		"project_adoptions",
		"projects",
		"activity_managers",
		"activity_adoptions",
		"activities",
		"activity_kinds",
		"contract_adoptions",
		"contracts",
		"customers",
		"unit_memberships",
		"units",
		"organization_memberships",
		"users",
		"organization_settings",
		"organizations",
		"verification_tokens",
	}
	ctx := context.Background()
	for _, tname := range tables {
		_, err := pool.Exec(ctx, "DROP TABLE IF EXISTS "+tname+" CASCADE")
		if err != nil {
			t.Logf("Dropping table %s: %v", tname, err)
		}
	}
}

func uniqueEmail() string {
	return uuid.New().String() + "@test.com"
}

func uniqueUsername() string {
	return "user_" + uuid.New().String()[:8]
}

func uniqueCode() string {
	return uuid.New().String()[:12]
}

func seedOrg(t *testing.T, pool *pgxpool.Pool, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		id, "Test Org", "test-org-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)
	return id
}

func seedUser(t *testing.T, pool *pgxpool.Pool, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, uniqueEmail(), uniqueUsername(), "Test", "User", "hash", true, now, now)
	require.NoError(t, err)
	return id
}

func seedProject(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO projects (id, org_id, name, project_type, type, governance_model, created_by_org_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, orgID, "Test Project", "billable", "billable", "creator_controlled", orgID, now, now)
	require.NoError(t, err)
	return id
}

func seedSubproject(t *testing.T, pool *pgxpool.Pool, projectID uuid.UUID, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO subprojects (id, project_id, name, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, projectID, "Test Subproject", now, now)
	require.NoError(t, err)
	return id
}

func seedUnit(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO units (id, org_id, name, code, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, orgID, "Seed Unit", "SEED", now, now)
	require.NoError(t, err)
	return id
}
