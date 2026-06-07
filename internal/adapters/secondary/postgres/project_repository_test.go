package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	projectdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/project"
	"github.com/stretchr/testify/require"
)

func TestProjectRepository_Create_Get(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewProjectRepository(pool)
	orgID := seedOrg(t, pool, time.Now().UTC())

	req := &projectdomain.CreateProjectRequest{
		Name:            "Test Project",
		Type:            "billable",
		ContractID:      "",
		GovernanceModel: "creator_controlled",
		IsShared:        false,
	}

	created, err := repo.Create(context.Background(), orgID, req)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, "Test Project", created.Name)
	require.Equal(t, "billable", string(created.Type))
	require.Equal(t, orgID, created.CreatedByOrgID)
	require.True(t, created.IsActive)
	require.False(t, created.IsShared)

	// Get the created project
	got, err := repo.Get(context.Background(), orgID, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "Test Project", got.Name)
}

func TestProjectRepository_Get_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewProjectRepository(pool)
	_, err := repo.Get(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	require.ErrorIs(t, err, projectdomain.ErrProjectNotFound)
}

func TestProjectRepository_List_Scopes(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewProjectRepository(pool)
	now := time.Now().UTC()

	// Create two organizations: one creates a shared project, another adopts it
	creatorOrgID := seedOrg(t, pool, now)
	adopterOrgID := seedOrg(t, pool, now)

	// Create a shared project
	req := &projectdomain.CreateProjectRequest{
		Name:            "Shared Project",
		Type:            "billable",
		GovernanceModel: "creator_controlled",
		IsShared:        true,
	}
	sharedProj, err := repo.Create(context.Background(), creatorOrgID, req)
	require.NoError(t, err)

	// Create a private project
	req2 := &projectdomain.CreateProjectRequest{
		Name:            "Private Project",
		Type:            "internal",
		GovernanceModel: "creator_controlled",
		IsShared:        false,
	}
	privateProj, err := repo.Create(context.Background(), creatorOrgID, req2)
	require.NoError(t, err)
	require.False(t, privateProj.IsShared)

	// Adopt the shared project for adopterOrgID
	_, err = repo.Adopt(context.Background(), adopterOrgID, sharedProj.ID)
	require.NoError(t, err)

	// Test default scope (created_by_org_id = orgID)
	projects, err := repo.List(context.Background(), creatorOrgID, "", "")
	require.NoError(t, err)
	require.Len(t, projects, 2)

	// Test adopted scope
	adopted, err := repo.List(context.Background(), adopterOrgID, "adopted", "")
	require.NoError(t, err)
	require.Len(t, adopted, 1)
	require.Equal(t, sharedProj.ID, adopted[0].ID)
	require.True(t, adopted[0].IsAdopted)

	// Test all scope (is_shared = true)
	allShared, err := repo.List(context.Background(), adopterOrgID, "all", "")
	require.NoError(t, err)
	require.Len(t, allShared, 1)
	require.Equal(t, sharedProj.ID, allShared[0].ID)

	// Empty list when no projects match
	empty, err := repo.List(context.Background(), uuid.New(), "adopted", "")
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestProjectRepository_Adopt_ErrAlreadyAdopted(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewProjectRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)

	req := &projectdomain.CreateProjectRequest{
		Name:            "Adoptable Project",
		Type:            "billable",
		GovernanceModel: "creator_controlled",
		IsShared:        true,
	}
	proj, err := repo.Create(context.Background(), orgID, req)
	require.NoError(t, err)

	// First adoption should succeed
	adoption, err := repo.Adopt(context.Background(), orgID, proj.ID)
	require.NoError(t, err)
	require.NotNil(t, adoption)
	require.Equal(t, proj.ID, adoption.ProjectID)
	require.Equal(t, orgID, adoption.OrganizationID)

	// Second adoption should fail
	_, err = repo.Adopt(context.Background(), orgID, proj.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, projectdomain.ErrAlreadyAdopted)
}

func TestProjectRepository_ListManagers_AddManager_RemoveManager(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewProjectRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	req := &projectdomain.CreateProjectRequest{
		Name:            "Manager Test Project",
		Type:            "billable",
		GovernanceModel: "creator_controlled",
		IsShared:        false,
	}
	proj, err := repo.Create(context.Background(), orgID, req)
	require.NoError(t, err)

	// Initially no managers
	managers, err := repo.ListManagers(context.Background(), proj.ID)
	require.NoError(t, err)
	require.Empty(t, managers)

	// Add manager
	pm, err := repo.AddManager(context.Background(), proj.ID, userID)
	require.NoError(t, err)
	require.NotNil(t, pm)
	require.Equal(t, proj.ID, pm.ProjectID)
	require.Equal(t, userID, pm.UserID)
	require.NotEmpty(t, pm.UserName)
	require.NotEmpty(t, pm.Email)

	// List should have one manager
	managers, err = repo.ListManagers(context.Background(), proj.ID)
	require.NoError(t, err)
	require.Len(t, managers, 1)
	require.Equal(t, userID, managers[0].UserID)

	// Remove manager
	err = repo.RemoveManager(context.Background(), proj.ID, userID)
	require.NoError(t, err)

	// List should be empty again
	managers, err = repo.ListManagers(context.Background(), proj.ID)
	require.NoError(t, err)
	require.Empty(t, managers)
}

func TestProjectRepository_RemoveManager_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewProjectRepository(pool)
	err := repo.RemoveManager(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
}
