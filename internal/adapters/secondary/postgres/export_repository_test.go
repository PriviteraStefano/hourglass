package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func seedFullFKChain(t *testing.T, pool *pgxpool.Pool, now time.Time) (orgID, projectID, contractID, customerID uuid.UUID) {
	t.Helper()
	orgID = seedOrg(t, pool, now)

	// Create a customer
	customerID = uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO customers (id, org_id, name, contact_name, email, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, true, $6, $6)`,
		customerID, orgID, "Export Customer", "Contact", "export@test.com", now)
	require.NoError(t, err)

	// Create a contract
	contractID = uuid.New()
	_, err = pool.Exec(context.Background(),
		`INSERT INTO contracts (id, name, km_rate, currency, customer_id, governance_model, created_by_org_id, is_shared, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, false, true, $8, $8)`,
		contractID, "Export Contract", 0.42, "EUR", customerID, "creator_controlled", orgID, now)
	require.NoError(t, err)

	// Create a project linked to the contract
	projectID = uuid.New()
	_, err = pool.Exec(context.Background(),
		`INSERT INTO projects (id, org_id, name, project_type, type, contract_id, customer_id, governance_model, created_by_org_id, is_shared, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, true, $10, $10)`,
		projectID, orgID, "Export Project", "billable", "billable", contractID, customerID, "creator_controlled", orgID, now)
	require.NoError(t, err)

	return orgID, projectID, contractID, customerID
}

func TestExportRepository_Timesheets(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewExportRepository(pool)
	now := time.Now().UTC()

	orgID, projectID, _, _ := seedFullFKChain(t, pool, now)
	userID := seedUser(t, pool, now)

	// Create a time entry
	_, err := pool.Exec(context.Background(),
		`INSERT INTO time_entries (id, org_id, user_id, project_id, hours, description, entry_date, status, is_deleted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, false, $9, $9)`,
		uuid.New(), orgID, userID, projectID, 7.5, "Export test entry", now, "draft", now)
	require.NoError(t, err)

	from := now.Add(-24 * time.Hour)
	to := now.Add(24 * time.Hour)

	results, err := repo.Timesheets(context.Background(), orgID, from, to, "", uuid.Nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "time_entry", results[0].EntryType)
	require.NotEmpty(t, results[0].Employee)
	require.NotEmpty(t, results[0].Project)
	require.NotNil(t, results[0].Hours)
	require.Equal(t, 7.5, *results[0].Hours)
	require.Nil(t, results[0].Amount)
	require.Nil(t, results[0].KmDistance)
	require.Equal(t, "Export test entry", results[0].Description)
}

func TestExportRepository_Expenses(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewExportRepository(pool)
	now := time.Now().UTC()

	orgID, projectID, _, _ := seedFullFKChain(t, pool, now)
	userID := seedUser(t, pool, now)

	// Create an expense
	_, err := pool.Exec(context.Background(),
		`INSERT INTO expenses (id, org_id, user_id, project_id, unit_id, category, amount, km_distance, description, expense_date, status, is_deleted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, false, $12, $12)`,
		uuid.New(), orgID, userID, projectID, uuid.New(), "mileage", 45.0, 120.5, "Export expense test", now, "draft", now)
	require.NoError(t, err)

	from := now.Add(-24 * time.Hour)
	to := now.Add(24 * time.Hour)

	results, err := repo.Expenses(context.Background(), orgID, from, to, "", uuid.Nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "expense", results[0].EntryType)
	require.NotEmpty(t, results[0].Employee)
	require.NotNil(t, results[0].Amount)
	require.Equal(t, 45.0, *results[0].Amount)
	require.NotNil(t, results[0].KmDistance)
	require.Equal(t, 120.5, *results[0].KmDistance)
	require.Nil(t, results[0].Hours)
	require.Equal(t, "mileage", results[0].Type)
	require.Equal(t, "Export expense test", results[0].Description)
}

func TestExportRepository_Empty(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewExportRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	from := now.Add(-24 * time.Hour)
	to := now.Add(24 * time.Hour)

	// Empty timesheets
	ts, err := repo.Timesheets(context.Background(), orgID, from, to, "", uuid.Nil)
	require.NoError(t, err)
	require.Empty(t, ts)

	// Empty expenses
	ex, err := repo.Expenses(context.Background(), orgID, from, to, "", uuid.Nil)
	require.NoError(t, err)
	require.Empty(t, ex)
}

func TestExportRepository_Timesheets_EmployeeFilter(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewExportRepository(pool)
	now := time.Now().UTC()

	orgID, projectID, _, _ := seedFullFKChain(t, pool, now)
	userID := seedUser(t, pool, now)
	otherUserID := seedUser(t, pool, now)

	// Entry for userID
	_, err := pool.Exec(context.Background(),
		`INSERT INTO time_entries (id, org_id, user_id, project_id, hours, description, entry_date, status, is_deleted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, false, $9, $9)`,
		uuid.New(), orgID, userID, projectID, 7.5, "Employee entry", now, "draft", now)
	require.NoError(t, err)

	// Entry for otherUserID
	_, err = pool.Exec(context.Background(),
		`INSERT INTO time_entries (id, org_id, user_id, project_id, hours, description, entry_date, status, is_deleted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, false, $9, $9)`,
		uuid.New(), orgID, otherUserID, projectID, 5.0, "Other employee entry", now, "draft", now)
	require.NoError(t, err)

	from := now.Add(-24 * time.Hour)
	to := now.Add(24 * time.Hour)

	// Employee filter should only show their entries
	results, err := repo.Timesheets(context.Background(), orgID, from, to, "employee", userID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "Employee entry", results[0].Description)
}

func TestExportRepository_Expenses_EmployeeFilter(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewExportRepository(pool)
	now := time.Now().UTC()

	orgID, projectID, _, _ := seedFullFKChain(t, pool, now)
	userID := seedUser(t, pool, now)
	otherUserID := seedUser(t, pool, now)

	// Entry for userID
	_, err := pool.Exec(context.Background(),
		`INSERT INTO expenses (id, org_id, user_id, project_id, unit_id, category, amount, description, expense_date, status, is_deleted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, false, $11, $11)`,
		uuid.New(), orgID, userID, projectID, uuid.New(), "meal", 32.0, "Employee expense", now, "draft", now)
	require.NoError(t, err)

	// Entry for otherUserID
	_, err = pool.Exec(context.Background(),
		`INSERT INTO expenses (id, org_id, user_id, project_id, unit_id, category, amount, description, expense_date, status, is_deleted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, false, $11, $11)`,
		uuid.New(), orgID, otherUserID, projectID, uuid.New(), "meal", 48.0, "Other employee expense", now, "draft", now)
	require.NoError(t, err)

	from := now.Add(-24 * time.Hour)
	to := now.Add(24 * time.Hour)

	results, err := repo.Expenses(context.Background(), orgID, from, to, "employee", userID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "Employee expense", results[0].Description)
}
