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
		"coverage_snapshot_rows",
		"coverage_period_closes",
		"coverage_allocations",
		"time_entries",
		"direction",
		"wg_members",
		"working_groups",
		"subprojects",
		"project_managers",
		"project_adoptions",
		"projects",
		"activity_managers",
		"activity_adoptions",
		"audit_logs",
		"ticket_comments",
		"tickets",
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
		"org_settings",
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

// seedActivityKind ensures the org has a kind in its activity_kinds catalog
// (ADR-P-007 D-2 — kind is an org-level catalog label, not an enum).
func seedActivityKind(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, kind string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO activity_kinds (org_id, name, is_seed) VALUES ($1, $2, true)
		 ON CONFLICT (org_id, name) DO NOTHING`,
		orgID, kind)
	require.NoError(t, err)
}

// seedActivity creates an activity linked to the org's kind catalog.
// parentID nil = root activity; kind defaults to "engagement" when empty.
func seedActivity(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, kind string, parentID *uuid.UUID, now time.Time) uuid.UUID {
	t.Helper()
	if kind == "" {
		kind = "engagement"
	}
	seedActivityKind(t, pool, orgID, kind)
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO activities (id, org_id, parent_id, name, description, kind,
			governance_model, created_by_org_id, is_shared, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 'creator_controlled', $2, false, true, $7, $7)`,
		id, orgID, parentID, "Test Activity", "Test activity description", kind, now)
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

// seedCoverageContract creates a contract owned by the org (016 columns:
// contract_type + sold_hours + sold_period, nullable — pass "" / nil for
// legacy-shaped rows; support contracts need sold_period per
// contracts_sold_check). Named seedCoverageContract to avoid the pre-existing
// seedContract (time_entry_repository_test.go) which lacks the 016 columns.
func seedCoverageContract(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, name, contractType string, soldHours *float64, isShared bool, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	var soldPeriod any
	if contractType == "support" {
		soldPeriod = "monthly"
	}
	_, err := pool.Exec(context.Background(),
		`INSERT INTO contracts (id, name, governance_model, created_by_org_id, is_shared, is_active,
			contract_type, sold_hours, sold_period, created_at, updated_at)
		 VALUES ($1, $2, 'creator_controlled', $3, $4, true, $5, $6, $7, $8, $8)`,
		id, name, orgID, isShared, nullableStr(contractType), soldHours, soldPeriod, now)
	require.NoError(t, err)
	return id
}

// seedContractAdoption adopts a shared contract into an org (the adoption-aware
// visibility predicate on BucketBalance and 12-05 contract refs).
func seedContractAdoption(t *testing.T, pool *pgxpool.Pool, contractID, orgID uuid.UUID, now time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO contract_adoptions (id, contract_id, organization_id, created_at)
		 VALUES ($1, $2, $3, $4)`,
		uuid.New(), contractID, orgID, now)
	require.NoError(t, err)
}

// seedTimeEntry creates an approved time entry on the given activity/unit
// (post-011 shape: activity_id only, no project/wg columns).
func seedTimeEntry(t *testing.T, pool *pgxpool.Pool, orgID, userID, activityID, unitID uuid.UUID, hours float64, entryDate time.Time, status string, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO time_entries (id, org_id, user_id, activity_id, unit_id, hours, description, entry_date, status, is_deleted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, $10, $10)`,
		id, orgID, userID, activityID, unitID, hours, "Coverage test entry", entryDate, status, now)
	require.NoError(t, err)
	return id
}

// seedDirectionRow creates a direction row following the seedTimeEntry shape
// (seeded org/user/activity; returns the row id). directedTo / wgID /
// plannedDate / estHours are nullable — pass nil for NULL. The 021 XOR CHECK
// requires exactly one of directedTo / wgID; status defaults to 'draft' when
// empty.
func seedDirectionRow(t *testing.T, pool *pgxpool.Pool, orgID, directedBy uuid.UUID, directedTo *uuid.UUID, wgID *uuid.UUID, activityID uuid.UUID, plannedDate *time.Time, estHours *float64, status string, now time.Time) uuid.UUID {
	t.Helper()
	if status == "" {
		status = "draft"
	}
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO direction (id, org_id, directed_by, directed_to, wg_id, activity_id, planned_date, est_hours, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)`,
		id, orgID, directedBy, directedTo, wgID, activityID, plannedDate, estHours, status, now)
	require.NoError(t, err)
	return id
}

// nullableStr returns nil for an empty string — the SQL NULL a nullable
// column expects in seeds.
func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
