package surrealdb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupWorkingGroupRepo(t *testing.T) (*WorkingGroupRepository, func()) {
	t.Helper()

	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	ns := "test_wg_" + uuid.New().String()
	db, err := GetTestDBWithNamespace(ns, ns)
	if err != nil {
		t.Skipf("SurrealDB not available: %v", err)
	}

	repo := NewWorkingGroupRepository(db)
	return repo, func() { db.Close(context.Background()) }
}

func seedWorkingGroupOrgAndUser(t *testing.T, repo *WorkingGroupRepository) (uuid.UUID, uuid.UUID) {
	t.Helper()

	db := repo.db
	orgRepo := NewOrganizationRepository(db)
	orgID := uuid.New()
	org := &auth.Organization{
		ID:        orgID,
		Name:      "WG Test Org " + uuid.New().String()[:8],
		Slug:      "wg-org-" + uuid.New().String()[:8],
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := orgRepo.Add(context.Background(), org)
	require.NoError(t, err, "failed to seed org")

	userRepo := NewUserRepository(db)
	userID := uuid.New()
	user := &auth.User{
		ID:           userID,
		Email:        uuid.New().String() + "@test.com",
		Username:     "wg_user_" + uuid.New().String()[:8],
		PasswordHash: "hash",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = userRepo.Add(context.Background(), user)
	require.NoError(t, err, "failed to seed user")

	return orgID, userID
}

func TestWorkingGroupRepo_Create(t *testing.T) {
	repo, cleanup := setupWorkingGroupRepo(t)
	defer cleanup()
	orgID, userID := seedWorkingGroupOrgAndUser(t, repo)

	wg := &working_group.WorkingGroup{
		ID:               uuid.New(),
		OrgID:            orgID,
		Name:             "Engineering WG",
		Description:      "Engineering working group",
		ManagerID:        userID,
		EnforceUnitTuple: true,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	created, err := repo.Create(context.Background(), wg)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.NotEqual(t, uuid.Nil, created.ID, "expected ID to be set")
	assert.Equal(t, wg.Name, created.Name)
	assert.True(t, created.IsActive)
}

func TestWorkingGroupRepo_GetByID(t *testing.T) {
	repo, cleanup := setupWorkingGroupRepo(t)
	defer cleanup()
	orgID, userID := seedWorkingGroupOrgAndUser(t, repo)

	t.Run("existing", func(t *testing.T) {
		wg := &working_group.WorkingGroup{
			ID:               uuid.New(),
			OrgID:            orgID,
			Name:             "Design WG",
			Description:      "Design team",
			ManagerID:        userID,
			EnforceUnitTuple: false,
			IsActive:         true,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		created, err := repo.Create(context.Background(), wg)
		require.NoError(t, err)

		found, err := repo.GetByID(context.Background(), created.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, wg.Name, found.Name)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := repo.GetByID(context.Background(), uuid.New())
		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestWorkingGroupRepo_ListByOrg(t *testing.T) {
	repo, cleanup := setupWorkingGroupRepo(t)
	defer cleanup()
	orgID, userID := seedWorkingGroupOrgAndUser(t, repo)

	wgNames := []string{"Frontend WG", "Backend WG", "QA WG"}
	for _, name := range wgNames {
		wg := &working_group.WorkingGroup{
			ID:               uuid.New(),
			OrgID:            orgID,
			Name:             name,
			ManagerID:        userID,
			EnforceUnitTuple: false,
			IsActive:         true,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		_, err := repo.Create(context.Background(), wg)
		require.NoError(t, err)
	}

	results, err := repo.ListByOrg(context.Background(), orgID, nil)
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestWorkingGroupRepo_Update(t *testing.T) {
	repo, cleanup := setupWorkingGroupRepo(t)
	defer cleanup()
	orgID, userID := seedWorkingGroupOrgAndUser(t, repo)

	wg := &working_group.WorkingGroup{
		ID:               uuid.New(),
		OrgID:            orgID,
		Name:             "Original Name",
		Description:      "Original description",
		ManagerID:        userID,
		EnforceUnitTuple: false,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	created, err := repo.Create(context.Background(), wg)
	require.NoError(t, err)

	created.Name = "Updated Name"
	created.Description = "Updated description"
	created.EnforceUnitTuple = true
	created.UpdatedAt = time.Now()

	updated, err := repo.Update(context.Background(), created)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, "Updated description", updated.Description)
	assert.True(t, updated.EnforceUnitTuple)
}

func TestWorkingGroupRepo_Delete(t *testing.T) {
	repo, cleanup := setupWorkingGroupRepo(t)
	defer cleanup()
	orgID, userID := seedWorkingGroupOrgAndUser(t, repo)

	wg := &working_group.WorkingGroup{
		ID:               uuid.New(),
		OrgID:            orgID,
		Name:             "Temporary WG",
		ManagerID:        userID,
		EnforceUnitTuple: false,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	created, err := repo.Create(context.Background(), wg)
	require.NoError(t, err)

	err = repo.Delete(context.Background(), created.ID)
	require.NoError(t, err)

	found, err := repo.GetByID(context.Background(), created.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}
