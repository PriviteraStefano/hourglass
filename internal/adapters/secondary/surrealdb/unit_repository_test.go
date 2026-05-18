package surrealdb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUnitRepo(t *testing.T) (*UnitRepository, func()) {
	t.Helper()

	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	ns := "test_unit_" + uuid.New().String()
	db, err := GetTestDBWithNamespace(ns, ns)
	if err != nil {
		t.Skipf("SurrealDB not available: %v", err)
	}

	repo := NewUnitRepository(db)
	return repo, func() { db.Close(context.Background()) }
}

func seedUnitOrg(t *testing.T, repo *UnitRepository) uuid.UUID {
	t.Helper()

	orgRepo := NewOrganizationRepository(repo.db)
	orgID := uuid.New()
	org := &auth.Organization{
		ID:        orgID,
		Name:      "Unit Test Org " + uuid.New().String()[:8],
		Slug:      "unit-org-" + uuid.New().String()[:8],
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := orgRepo.Add(context.Background(), org)
	require.NoError(t, err, "failed to seed org")

	return orgID
}

func TestUnitRepo_Create(t *testing.T) {
	repo, cleanup := setupUnitRepo(t)
	defer cleanup()
	orgID := seedUnitOrg(t, repo)

	u := &unit.Unit{
		OrgID:          orgID,
		Name:           "Engineering",
		Description:    "Engineering department",
		HierarchyLevel: 1,
		Code:           "ENG",
	}

	created, err := repo.Create(context.Background(), u)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.NotEmpty(t, created.ID, "expected ID to be set")
	assert.Equal(t, u.Name, created.Name)
	assert.Equal(t, u.HierarchyLevel, created.HierarchyLevel)
	assert.Equal(t, u.Code, created.Code)
}

func TestUnitRepo_GetByID(t *testing.T) {
	repo, cleanup := setupUnitRepo(t)
	defer cleanup()
	orgID := seedUnitOrg(t, repo)

	t.Run("existing", func(t *testing.T) {
		u := &unit.Unit{
			OrgID:          orgID,
			Name:           "Design",
			Description:    "Design team",
			HierarchyLevel: 2,
			Code:           "DES",
		}

		created, err := repo.Create(context.Background(), u)
		require.NoError(t, err)

		found, err := repo.GetByID(context.Background(), created.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, created.Name, found.Name)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := repo.GetByID(context.Background(), "nonexistent-id")
		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestUnitRepo_ListByOrg(t *testing.T) {
	repo, cleanup := setupUnitRepo(t)
	defer cleanup()
	orgID := seedUnitOrg(t, repo)

	unitNames := []string{"HR", "Finance", "IT"}
	for _, name := range unitNames {
		u := &unit.Unit{
			OrgID:          orgID,
			Name:           name,
			HierarchyLevel: 1,
			Code:           name[:3],
		}
		_, err := repo.Create(context.Background(), u)
		require.NoError(t, err)
	}

	results, err := repo.ListByOrg(context.Background(), orgID)
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestUnitRepo_Update(t *testing.T) {
	repo, cleanup := setupUnitRepo(t)
	defer cleanup()
	orgID := seedUnitOrg(t, repo)

	u := &unit.Unit{
		OrgID:          orgID,
		Name:           "Old Department",
		Description:    "Old description",
		HierarchyLevel: 1,
		Code:           "OLD",
	}

	created, err := repo.Create(context.Background(), u)
	require.NoError(t, err)

	created.Name = "New Department"
	created.Description = "Updated description"
	created.HierarchyLevel = 2
	created.Code = "NEW"
	created.UpdatedAt = time.Now()

	updated, err := repo.Update(context.Background(), created)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "New Department", updated.Name)
	assert.Equal(t, "Updated description", updated.Description)
	assert.Equal(t, 2, updated.HierarchyLevel)
	assert.Equal(t, "NEW", updated.Code)
}

func TestUnitRepo_Delete(t *testing.T) {
	repo, cleanup := setupUnitRepo(t)
	defer cleanup()
	orgID := seedUnitOrg(t, repo)

	u := &unit.Unit{
		OrgID:          orgID,
		Name:           "Temporary Unit",
		HierarchyLevel: 1,
		Code:           "TMP",
	}

	created, err := repo.Create(context.Background(), u)
	require.NoError(t, err)

	err = repo.Delete(context.Background(), created.ID)
	require.NoError(t, err)

	found, err := repo.GetByID(context.Background(), created.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}
