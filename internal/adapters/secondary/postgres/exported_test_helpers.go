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
	"github.com/stefanoprivitera/hourglass/internal/db"
	"github.com/stretchr/testify/require"
)

// TestPool returns a pool connected to the test database.
// Skips test if DATABASE_URL is not set.
func TestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	pool, err := db.NewPool()
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	return pool
}

// SetupTestSchema reads and applies all migration .up.sql files from the
// migrations directory (excluding seed files), sorted alphabetically.
func SetupTestSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	files, err := filepath.Glob("../../../../migrations/*.up.sql")
	require.NoError(t, err)
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
		"contract_adoptions",
		"contracts",
		"customers",
		"unit_memberships",
		"units",
		"organization_memberships",
		"users",
		"organizations",
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
