package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	domainexpense "github.com/stefanoprivitera/hourglass/internal/core/domain/expense"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stretchr/testify/require"
)

func seedExpenseProject(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO projects (id, org_id, name, project_type, type, governance_model, created_by_org_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
		id, orgID, "Expense Test Project", "billable", "billable", "creator_controlled", orgID, now)
	require.NoError(t, err)
	return id
}

func TestExpenseRepository_Create_GetByID(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewExpenseRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	projectID := seedExpenseProject(t, pool, orgID, now)
	unitID := seedUnit(t, pool, orgID, now)

	kmDist := 45.0

	expense := &domainexpense.Expense{
		OrgID:       orgID,
		UserID:      userID,
		ProjectID:   projectID,
		UnitID:      unitID,
		Category:    "mileage",
		Amount:      150.50,
		KmDistance:  &kmDist,
		Description: "Test mileage expense",
		EntryDate:   now,
		Status:      domainexpense.StatusDraft,
	}

	created, err := repo.Create(context.Background(), expense)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotEqual(t, uuid.Nil, created.ID)
	require.Equal(t, orgID, created.OrgID)
	require.Equal(t, userID, created.UserID)
	require.Equal(t, projectID, created.ProjectID)
	require.Equal(t, "mileage", created.Category)
	require.Equal(t, 150.50, created.Amount)
	require.NotNil(t, created.KmDistance)
	require.Equal(t, 45.0, *created.KmDistance)
	require.Equal(t, "Test mileage expense", created.Description)
	require.Equal(t, domainexpense.StatusDraft, created.Status)
	require.False(t, created.CreatedAt.IsZero())

	// GetByID
	got, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.Amount, got.Amount)
}

func TestExpenseRepository_GetByID_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewExpenseRepository(pool)
	_, err := repo.GetByID(context.Background(), uuid.New())
	require.Error(t, err)
	require.ErrorIs(t, err, domainexpense.ErrExpenseNotFound)
}

func TestExpenseRepository_List(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewExpenseRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	orgID2 := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	projectID := seedExpenseProject(t, pool, orgID, now)
	unitID := seedUnit(t, pool, orgID, now)

	// Create two expenses for org1
	for i := 0; i < 2; i++ {
		e := &domainexpense.Expense{
			OrgID:       orgID,
			UserID:      userID,
			ProjectID:   projectID,
			UnitID:      unitID,
			Category:    "meal",
			Amount:      100.0,
			Description: "List test expense",
			EntryDate:   now,
			Status:      domainexpense.StatusDraft,
		}
		_, err := repo.Create(context.Background(), e)
		require.NoError(t, err)
	}

	// List by org
	expenses, err := repo.List(context.Background(), orgID, ports.ExpenseListFilters{IsDeleted: false})
	require.NoError(t, err)
	require.Len(t, expenses, 2)

	// List by org2 (no expenses)
	empty, err := repo.List(context.Background(), orgID2, ports.ExpenseListFilters{IsDeleted: false})
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestExpenseRepository_Update(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewExpenseRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	projectID := seedExpenseProject(t, pool, orgID, now)
	unitID := seedUnit(t, pool, orgID, now)

	kmDist := 0.0

	expense := &domainexpense.Expense{
		OrgID:       orgID,
		UserID:      userID,
		ProjectID:   projectID,
		UnitID:      unitID,
		Category:    "mileage",
		Amount:      100.0,
		KmDistance:  &kmDist,
		Description: "Original",
		EntryDate:   now,
		Status:      domainexpense.StatusDraft,
	}
	created, err := repo.Create(context.Background(), expense)
	require.NoError(t, err)

	// Update
	newKmDist := 50.0
	created.Category = "meal"
	created.Amount = 200.0
	created.KmDistance = &newKmDist
	created.Description = "Updated expense"

	updated, err := repo.Update(context.Background(), created)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, 200.0, updated.Amount)
	require.NotNil(t, updated.KmDistance)
	require.Equal(t, 50.0, *updated.KmDistance)
	require.Equal(t, "meal", updated.Category)
	require.Equal(t, "Updated expense", updated.Description)
}

func TestExpenseRepository_Delete(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewExpenseRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	projectID := seedExpenseProject(t, pool, orgID, now)
	unitID := seedUnit(t, pool, orgID, now)

	expense := &domainexpense.Expense{
		OrgID:       orgID,
		UserID:      userID,
		ProjectID:   projectID,
		UnitID:      unitID,
		Category:    "other",
		Amount:      75.0,
		Description: "To delete",
		EntryDate:   now,
		Status:      domainexpense.StatusDraft,
	}
	created, err := repo.Create(context.Background(), expense)
	require.NoError(t, err)

	// Soft delete
	err = repo.Delete(context.Background(), created.ID)
	require.NoError(t, err)

	// List should exclude deleted
	expenses, err := repo.List(context.Background(), orgID, ports.ExpenseListFilters{IsDeleted: false})
	require.NoError(t, err)
	require.Empty(t, expenses)

	// Delete on non-existent ID
	err = repo.Delete(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestExpenseRepository_CreateApproval(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewExpenseRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	projectID := seedExpenseProject(t, pool, orgID, now)
	unitID := seedUnit(t, pool, orgID, now)

	expense := &domainexpense.Expense{
		OrgID:       orgID,
		UserID:      userID,
		ProjectID:   projectID,
		UnitID:      unitID,
		Category:    "mileage",
		Amount:      50.0,
		Description: "Approval test",
		EntryDate:   now,
		Status:      domainexpense.StatusDraft,
	}
	created, err := repo.Create(context.Background(), expense)
	require.NoError(t, err)

	// Create approval
	approval := &domainexpense.Approval{
		ID:          uuid.New(),
		EntryID:     created.ID,
		Action:      "submit",
		ActorUserID: userID,
		Comment:     "submitting for approval",
		CreatedAt:   now,
	}
	err = repo.CreateApproval(context.Background(), approval)
	require.NoError(t, err)

	// Verify via raw SQL
	var count int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM expense_approvals WHERE expense_id = $1 AND action = 'submit'`,
		created.ID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestExpenseRepository_IsPeriodLocked(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewExpenseRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	projectID := seedExpenseProject(t, pool, orgID, now)

	// Create a locked cutoff period
	start := now.Add(-72 * time.Hour)
	end := now.Add(72 * time.Hour)
	seedFinancialCutoffPeriod(t, pool, orgID, projectID, start, end, true)

	// Check a date within the locked period
	locked, err := repo.IsPeriodLocked(context.Background(), orgID, projectID, now.Format(time.RFC3339))
	require.NoError(t, err)
	require.True(t, locked)

	// Check a date outside the locked period
	outsideDate := now.Add(168 * time.Hour)
	locked, err = repo.IsPeriodLocked(context.Background(), orgID, projectID, outsideDate.Format(time.RFC3339))
	require.NoError(t, err)
	require.False(t, locked)
}

func TestExpenseRepository_ListPending(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewExpenseRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	projectID := seedExpenseProject(t, pool, orgID, now)
	unitID := seedUnit(t, pool, orgID, now)

	// Create a submitted expense
	e1 := &domainexpense.Expense{
		OrgID:       orgID,
		UserID:      userID,
		ProjectID:   projectID,
		UnitID:      unitID,
		Category:    "meal",
		Amount:      25.0,
		Description: "Pending expense",
		EntryDate:   now,
		Status:      domainexpense.StatusSubmitted,
	}
	_, err := repo.Create(context.Background(), e1)
	require.NoError(t, err)

	// Create a draft expense (should not appear in pending)
	e2 := &domainexpense.Expense{
		OrgID:       orgID,
		UserID:      userID,
		ProjectID:   projectID,
		UnitID:      unitID,
		Category:    "meal",
		Amount:      30.0,
		Description: "Draft expense",
		EntryDate:   now,
		Status:      domainexpense.StatusDraft,
	}
	_, err = repo.Create(context.Background(), e2)
	require.NoError(t, err)

	// List all pending (no role filter — default shows submitted)
	allPending, err := repo.ListPending(context.Background(), orgID, "", "")
	require.NoError(t, err)
	require.Len(t, allPending, 1)
}
