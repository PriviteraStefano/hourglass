package surrealdb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupContractRepo(t *testing.T) (*ContractRepository, func()) {
	t.Helper()

	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	ns := "test_contract_" + uuid.New().String()
	db, err := GetTestDBWithNamespace(ns, ns)
	if err != nil {
		t.Skipf("SurrealDB not available: %v", err)
	}

	repo := NewContractRepository(db)
	return repo, func() { db.Close(context.Background()) }
}

func seedContractOrg(t *testing.T, repo *ContractRepository) uuid.UUID {
	t.Helper()

	db := repo.db
	orgRepo := NewOrganizationRepository(db)
	orgID := uuid.New()
	org := &auth.Organization{
		ID:        orgID,
		Name:      "Contract Test Org " + uuid.New().String()[:8],
		Slug:      "ct-org-" + uuid.New().String()[:8],
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := orgRepo.Add(context.Background(), org)
	require.NoError(t, err, "failed to seed org")

	return orgID
}

func TestContractRepo_Create(t *testing.T) {
	repo, cleanup := setupContractRepo(t)
	defer cleanup()
	orgID := seedContractOrg(t, repo)

	req := &contractdomain.CreateContractRequest{
		Name:            "Test Contract " + uuid.New().String()[:8],
		KmRate:          0.45,
		Currency:        "EUR",
		GovernanceModel: models.GovernanceCreatorControlled,
		IsShared:        false,
	}

	result, err := repo.Create(context.Background(), orgID, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, req.Name, result.Name)
	assert.Equal(t, req.KmRate, result.KmRate)
	assert.Equal(t, req.Currency, result.Currency)
	assert.True(t, result.IsActive)
}

func TestContractRepo_Get(t *testing.T) {
	repo, cleanup := setupContractRepo(t)
	defer cleanup()
	orgID := seedContractOrg(t, repo)

	t.Run("existing", func(t *testing.T) {
		req := &contractdomain.CreateContractRequest{
			Name:            "Get Test Contract " + uuid.New().String()[:8],
			KmRate:          0.50,
			Currency:        "USD",
			GovernanceModel: models.GovernanceCreatorControlled,
			IsShared:        false,
		}

		created, err := repo.Create(context.Background(), orgID, req)
		require.NoError(t, err)

		found, err := repo.Get(context.Background(), orgID, created.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, req.Name, found.Name)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := repo.Get(context.Background(), orgID, uuid.New())
		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestContractRepo_List(t *testing.T) {
	repo, cleanup := setupContractRepo(t)
	defer cleanup()
	orgID := seedContractOrg(t, repo)

	contractNames := []string{"List Test A", "List Test B"}
	for _, name := range contractNames {
		req := &contractdomain.CreateContractRequest{
			Name:            name,
			KmRate:          0.40,
			Currency:        "EUR",
			GovernanceModel: models.GovernanceCreatorControlled,
			IsShared:        false,
		}
		_, err := repo.Create(context.Background(), orgID, req)
		require.NoError(t, err)
	}

	results, err := repo.List(context.Background(), orgID, "own", boolPtr(true))
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestContractRepo_Update(t *testing.T) {
	repo, cleanup := setupContractRepo(t)
	defer cleanup()
	orgID := seedContractOrg(t, repo)

	req := &contractdomain.CreateContractRequest{
		Name:            "Update Test",
		KmRate:          0.35,
		Currency:        "EUR",
		GovernanceModel: models.GovernanceCreatorControlled,
		IsShared:        false,
	}

	created, err := repo.Create(context.Background(), orgID, req)
	require.NoError(t, err)

	updateReq := &contractdomain.UpdateContractRequest{
		Name:     "Updated Contract Name",
		KmRate:   float64Ptr(0.55),
		Currency: "USD",
		IsActive: boolPtr(true),
	}

	updated, _, err := repo.Update(context.Background(), orgID, created.ID, updateReq)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "Updated Contract Name", updated.Name)
	assert.Equal(t, 0.55, updated.KmRate)
	assert.Equal(t, "USD", updated.Currency)
}

func TestContractRepo_Delete(t *testing.T) {
	repo, cleanup := setupContractRepo(t)
	defer cleanup()
	orgID := seedContractOrg(t, repo)

	req := &contractdomain.CreateContractRequest{
		Name:            "Delete Test",
		KmRate:          0.30,
		Currency:        "EUR",
		GovernanceModel: models.GovernanceCreatorControlled,
		IsShared:        false,
	}

	created, err := repo.Create(context.Background(), orgID, req)
	require.NoError(t, err)

	err = repo.Delete(context.Background(), orgID, created.ID)
	require.NoError(t, err)

	found, err := repo.Get(context.Background(), orgID, created.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

func boolPtr(b bool) *bool {
	return &b
}

func float64Ptr(f float64) *float64 {
	return &f
}
