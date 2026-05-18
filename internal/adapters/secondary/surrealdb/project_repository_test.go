package surrealdb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	projectdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/project"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupProjectRepo(t *testing.T) (*ProjectRepository, *ContractRepository, func()) {
	t.Helper()

	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	ns := "test_project_" + uuid.New().String()
	db, err := GetTestDBWithNamespace(ns, ns)
	if err != nil {
		t.Skipf("SurrealDB not available: %v", err)
	}

	projectRepo := NewProjectRepository(db)
	contractRepo := NewContractRepository(db)
	return projectRepo, contractRepo, func() { db.Close(context.Background()) }
}

func seedProjectOrgAndContract(t *testing.T, projectRepo *ProjectRepository, contractRepo *ContractRepository) (uuid.UUID, uuid.UUID) {
	t.Helper()

	db := projectRepo.db

	orgRepo := NewOrganizationRepository(db)
	orgID := uuid.New()
	org := &auth.Organization{
		ID:        orgID,
		Name:      "Project Test Org " + uuid.New().String()[:8],
		Slug:      "proj-org-" + uuid.New().String()[:8],
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := orgRepo.Add(context.Background(), org)
	require.NoError(t, err, "failed to seed org")

	contractReq := &contractdomain.CreateContractRequest{
		Name:            "Base Contract " + uuid.New().String()[:8],
		KmRate:          0.40,
		Currency:        "EUR",
		GovernanceModel: models.GovernanceCreatorControlled,
		IsShared:        false,
	}
	createdContract, err := contractRepo.Create(context.Background(), orgID, contractReq)
	require.NoError(t, err, "failed to seed contract")

	return orgID, createdContract.ID
}

func TestProjectRepo_Create(t *testing.T) {
	projectRepo, contractRepo, cleanup := setupProjectRepo(t)
	defer cleanup()
	orgID, contractID := seedProjectOrgAndContract(t, projectRepo, contractRepo)

	req := &projectdomain.CreateProjectRequest{
		Name:            "Test Project " + uuid.New().String()[:8],
		Type:            models.ProjectTypeBillable,
		ContractID:      contractID.String(),
		GovernanceModel: models.GovernanceCreatorControlled,
		IsShared:        false,
	}

	result, err := projectRepo.Create(context.Background(), orgID, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, req.Name, result.Name)
	assert.Equal(t, models.ProjectTypeBillable, result.Type)
	assert.True(t, result.IsActive)
}

func TestProjectRepo_Get(t *testing.T) {
	projectRepo, contractRepo, cleanup := setupProjectRepo(t)
	defer cleanup()
	orgID, contractID := seedProjectOrgAndContract(t, projectRepo, contractRepo)

	t.Run("existing", func(t *testing.T) {
		req := &projectdomain.CreateProjectRequest{
			Name:            "Get Test Project " + uuid.New().String()[:8],
			Type:            models.ProjectTypeBillable,
			ContractID:      contractID.String(),
			GovernanceModel: models.GovernanceCreatorControlled,
			IsShared:        false,
		}

		created, err := projectRepo.Create(context.Background(), orgID, req)
		require.NoError(t, err)

		found, err := projectRepo.Get(context.Background(), orgID, created.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, req.Name, found.Name)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := projectRepo.Get(context.Background(), orgID, uuid.New())
		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestProjectRepo_List(t *testing.T) {
	projectRepo, contractRepo, cleanup := setupProjectRepo(t)
	defer cleanup()
	orgID, contractID := seedProjectOrgAndContract(t, projectRepo, contractRepo)

	projectNames := []string{"Project Alpha", "Project Beta"}
	for _, name := range projectNames {
		req := &projectdomain.CreateProjectRequest{
			Name:            name,
			Type:            models.ProjectTypeBillable,
			ContractID:      contractID.String(),
			GovernanceModel: models.GovernanceCreatorControlled,
			IsShared:        false,
		}
		_, err := projectRepo.Create(context.Background(), orgID, req)
		require.NoError(t, err)
	}

	results, err := projectRepo.List(context.Background(), orgID, "own", "")
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestProjectRepo_ListByContract(t *testing.T) {
	projectRepo, contractRepo, cleanup := setupProjectRepo(t)
	defer cleanup()
	orgID, contractID := seedProjectOrgAndContract(t, projectRepo, contractRepo)

	req := &projectdomain.CreateProjectRequest{
		Name:            "Contract Specific Project",
		Type:            models.ProjectTypeBillable,
		ContractID:      contractID.String(),
		GovernanceModel: models.GovernanceCreatorControlled,
		IsShared:        false,
	}
	_, err := projectRepo.Create(context.Background(), orgID, req)
	require.NoError(t, err)

	results, err := projectRepo.List(context.Background(), orgID, "own", contractID.String())
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestProjectRepo_Update(t *testing.T) {
	projectRepo, contractRepo, cleanup := setupProjectRepo(t)
	defer cleanup()
	orgID, contractID := seedProjectOrgAndContract(t, projectRepo, contractRepo)

	req := &projectdomain.CreateProjectRequest{
		Name:            "Project to Update",
		Type:            models.ProjectTypeBillable,
		ContractID:      contractID.String(),
		GovernanceModel: models.GovernanceCreatorControlled,
		IsShared:        false,
	}

	created, err := projectRepo.Create(context.Background(), orgID, req)
	require.NoError(t, err)

	// Project Update is done via the contract repo's UpdateContractRequest approach
	// but the project repo doesn't expose a direct update.
	// Instead, we test that the project is retrievable after creation and verify
	// the project can be created, listed, and retrieved correctly.
	found, err := projectRepo.Get(context.Background(), orgID, created.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, req.Name, found.Name)
	assert.Equal(t, models.ProjectTypeBillable, found.Type)
}
