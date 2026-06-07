package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/models"
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
	unitID := seedUnit(t, pool, orgID, now)
	projectID := seedExpenseProject(t, pool, orgID, now)

	amount := 150.50
	kmDist := 45.0
	cat := models.CategoryMileage

	expense := &models.Expense{
		OrganizationID: orgID,
		UserID:         userID,
		UnitID:         &unitID,
		ProjectID:      &projectID,
		Date:           now,
		Type:           &cat,
		Amount:         &amount,
		KmDistance:     &kmDist,
		Description:    "Test mileage expense",
		Status:         models.StatusDraft,
	}

	created, err := repo.Create(context.Background(), expense)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotEqual(t, uuid.Nil, created.ID)
	require.Equal(t, orgID, created.OrganizationID)
	require.Equal(t, userID, created.UserID)
	require.NotNil(t, created.UnitID)
	require.Equal(t, unitID, *created.UnitID)
	require.NotNil(t, created.ProjectID)
	require.Equal(t, projectID, *created.ProjectID)
	require.Equal(t, models.CategoryMileage, *created.Type)
	require.NotNil(t, created.Amount)
	require.Equal(t, 150.50, *created.Amount)
	require.NotNil(t, created.KmDistance)
	require.Equal(t, 45.0, *created.KmDistance)
	require.Equal(t, "Test mileage expense", created.Description)
	require.Equal(t, models.StatusDraft, created.Status)
	require.False(t, created.CreatedAt.IsZero())

	// GetByID
	got, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, *created.Amount, *got.Amount)
	require.NotNil(t, got.KmDistance)
	require.Equal(t, *created.KmDistance, *got.KmDistance)
}

func TestExpenseRepository_GetByID_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewExpenseRepository(pool)
	_, err := repo.GetByID(context.Background(), uuid.New())
	require.Error(t, err)
	require.ErrorIs(t, err, ports.ErrNotFound)
}

func TestExpenseRepository_ListByOrg(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewExpenseRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	orgID2 := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	projectID := seedExpenseProject(t, pool, orgID, now)

	amount := 100.0
	cat := models.CategoryMeal

	// Create two expenses for org1
	for i := 0; i < 2; i++ {
		e := &models.Expense{
			OrganizationID: orgID,
			UserID:         userID,
			UnitID:         &unitID,
			ProjectID:      &projectID,
			Date:           now,
			Type:           &cat,
			Amount:         &amount,
			Description:    "List test expense",
			Status:         models.StatusDraft,
		}
		_, err := repo.Create(context.Background(), e)
		require.NoError(t, err)
	}

	// List by org
	expenses, err := repo.ListByOrg(context.Background(), orgID, 10, 0)
	require.NoError(t, err)
	require.Len(t, expenses, 2)

	// List by org2 (no expenses)
	empty, err := repo.ListByOrg(context.Background(), orgID2, 10, 0)
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
	unitID := seedUnit(t, pool, orgID, now)
	projectID := seedExpenseProject(t, pool, orgID, now)

	amount := 100.0
	kmDist := 0.0
	cat := models.CategoryMileage

	expense := &models.Expense{
		OrganizationID: orgID,
		UserID:         userID,
		UnitID:         &unitID,
		ProjectID:      &projectID,
		Date:           now,
		Type:           &cat,
		Amount:         &amount,
		KmDistance:     &kmDist,
		Description:    "Original",
		Status:         models.StatusDraft,
	}
	created, err := repo.Create(context.Background(), expense)
	require.NoError(t, err)

	// Update
	newAmount := 200.0
	newKmDist := 50.0
	newCat := models.CategoryMeal
	created.Amount = &newAmount
	created.KmDistance = &newKmDist
	created.Type = &newCat
	created.Description = "Updated expense"

	updated, err := repo.Update(context.Background(), created)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, 200.0, *updated.Amount)
	require.NotNil(t, updated.KmDistance)
	require.Equal(t, 50.0, *updated.KmDistance)
	require.Equal(t, models.CategoryMeal, *updated.Type)
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
	unitID := seedUnit(t, pool, orgID, now)
	projectID := seedExpenseProject(t, pool, orgID, now)

	amount := 75.0
	cat := models.CategoryOther

	expense := &models.Expense{
		OrganizationID: orgID,
		UserID:         userID,
		UnitID:         &unitID,
		ProjectID:      &projectID,
		Date:           now,
		Type:           &cat,
		Amount:         &amount,
		Description:    "To delete",
		Status:         models.StatusDraft,
	}
	created, err := repo.Create(context.Background(), expense)
	require.NoError(t, err)

	// Soft delete
	err = repo.Delete(context.Background(), created.ID)
	require.NoError(t, err)

	// ListByOrg should exclude deleted
	expenses, err := repo.ListByOrg(context.Background(), orgID, 10, 0)
	require.NoError(t, err)
	require.Empty(t, expenses)

	// Delete on non-existent ID
	err = repo.Delete(context.Background(), uuid.New())
	require.Error(t, err)
}
