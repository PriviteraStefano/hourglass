package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	"github.com/stretchr/testify/require"
)

func TestContractRepository_Create_Get(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewContractRepository(pool)
	orgID := seedOrg(t, pool, time.Now().UTC())

	req := &contractdomain.CreateContractRequest{
		Name:            "Test Contract",
		KmRate:          0.35,
		Currency:        "EUR",
		GovernanceModel: "creator_controlled",
		IsShared:        false,
	}

	created, err := repo.Create(context.Background(), orgID, req)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, "Test Contract", created.Name)
	require.Equal(t, 0.35, created.KmRate)
	require.Equal(t, "EUR", created.Currency)
	require.Equal(t, orgID, created.CreatedByOrgID)
	require.True(t, created.IsActive)
	require.False(t, created.IsShared)
	require.Empty(t, created.CustomerName)
	require.Equal(t, 0, created.TimeEntriesCount)
	require.Equal(t, 0, created.AdoptionCount)
	require.False(t, created.IsAdopted)

	// Get the created contract
	got, err := repo.Get(context.Background(), orgID, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "Test Contract", got.Name)
}

func TestContractRepository_Get_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewContractRepository(pool)
	_, err := repo.Get(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	require.ErrorIs(t, err, contractdomain.ErrContractNotFound)
}

func TestContractRepository_List_Scope(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewContractRepository(pool)
	now := time.Now().UTC()

	creatorOrgID := seedOrg(t, pool, now)
	adopterOrgID := seedOrg(t, pool, now)

	// Create a shared contract
	req := &contractdomain.CreateContractRequest{
		Name:            "Shared Contract",
		KmRate:          0.50,
		Currency:        "EUR",
		GovernanceModel: "creator_controlled",
		IsShared:        true,
	}
	sharedCon, err := repo.Create(context.Background(), creatorOrgID, req)
	require.NoError(t, err)

	// Create a private contract
	req2 := &contractdomain.CreateContractRequest{
		Name:            "Private Contract",
		KmRate:          0.30,
		Currency:        "USD",
		GovernanceModel: "creator_controlled",
		IsShared:        false,
	}
	_, err = repo.Create(context.Background(), creatorOrgID, req2)
	require.NoError(t, err)

	// Adopt shared contract for adopterOrgID
	_, err = repo.Adopt(context.Background(), adopterOrgID, sharedCon.ID)
	require.NoError(t, err)

	// Default scope (created_by_org_id = orgID)
	contracts, err := repo.List(context.Background(), creatorOrgID, "", nil)
	require.NoError(t, err)
	require.Len(t, contracts, 2)

	// Adopted scope
	adopted, err := repo.List(context.Background(), adopterOrgID, "adopted", nil)
	require.NoError(t, err)
	require.Len(t, adopted, 1)
	require.Equal(t, sharedCon.ID, adopted[0].ID)
	require.True(t, adopted[0].IsAdopted)

	// All scope
	allShared, err := repo.List(context.Background(), adopterOrgID, "all", nil)
	require.NoError(t, err)
	require.Len(t, allShared, 1)
	require.Equal(t, sharedCon.ID, allShared[0].ID)

	// Filter by isActive=false should return empty (all are active)
	inactive := false
	activeList, err := repo.List(context.Background(), creatorOrgID, "", &inactive)
	require.NoError(t, err)
	require.Empty(t, activeList)

	// Empty list
	empty, err := repo.List(context.Background(), uuid.New(), "adopted", nil)
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestContractRepository_Adopt_ErrAlreadyAdopted(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewContractRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)

	req := &contractdomain.CreateContractRequest{
		Name:            "Adoptable Contract",
		KmRate:          0.40,
		Currency:        "EUR",
		GovernanceModel: "creator_controlled",
		IsShared:        true,
	}
	con, err := repo.Create(context.Background(), orgID, req)
	require.NoError(t, err)

	// First adoption succeeds
	adoption, err := repo.Adopt(context.Background(), orgID, con.ID)
	require.NoError(t, err)
	require.NotNil(t, adoption)
	require.Equal(t, con.ID, adoption.ContractID)
	require.Equal(t, orgID, adoption.OrganizationID)

	// Second adoption fails
	_, err = repo.Adopt(context.Background(), orgID, con.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, contractdomain.ErrAlreadyAdopted)
}

func TestContractRepository_Update(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewContractRepository(pool)
	orgID := seedOrg(t, pool, time.Now().UTC())

	req := &contractdomain.CreateContractRequest{
		Name:            "Original Contract",
		KmRate:          0.25,
		Currency:        "EUR",
		GovernanceModel: "creator_controlled",
		IsShared:        false,
	}
	created, err := repo.Create(context.Background(), orgID, req)
	require.NoError(t, err)

	// Update name and km_rate
	isShared := true
	updateReq := &contractdomain.UpdateContractRequest{
		Name:     "Updated Contract",
		KmRate:   float64Ptr(0.50),
		Currency: "USD",
		IsShared: &isShared,
	}

	updated, _, err := repo.Update(context.Background(), orgID, created.ID, updateReq)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, "Updated Contract", updated.Name)
	require.Equal(t, 0.50, updated.KmRate)
	require.Equal(t, "USD", updated.Currency)
	require.True(t, updated.IsShared)
}

func TestContractRepository_Update_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewContractRepository(pool)
	isShared := true
	_, _, err := repo.Update(context.Background(), uuid.New(), uuid.New(), &contractdomain.UpdateContractRequest{
		Name: "Nope", IsShared: &isShared,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, contractdomain.ErrContractNotFound)
}

func TestContractRepository_HasTimeEntries(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewContractRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	req := &contractdomain.CreateContractRequest{
		Name:            "TE Test Contract",
		KmRate:          0.35,
		Currency:        "EUR",
		GovernanceModel: "creator_controlled",
		IsShared:        false,
	}
	created, err := repo.Create(context.Background(), orgID, req)
	require.NoError(t, err)

	// Should have 0 time entries initially
	count, err := repo.HasTimeEntries(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestContractRepository_Delete(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewContractRepository(pool)
	orgID := seedOrg(t, pool, time.Now().UTC())

	req := &contractdomain.CreateContractRequest{
		Name:            "To Delete",
		KmRate:          0.10,
		Currency:        "EUR",
		GovernanceModel: "creator_controlled",
		IsShared:        false,
	}
	created, err := repo.Create(context.Background(), orgID, req)
	require.NoError(t, err)

	// Delete
	err = repo.Delete(context.Background(), orgID, created.ID)
	require.NoError(t, err)

	// Should not be found
	_, err = repo.Get(context.Background(), orgID, created.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, contractdomain.ErrContractNotFound)
}

func TestContractRepository_Delete_WrongOrg(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewContractRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	wrongOrgID := seedOrg(t, pool, now)

	req := &contractdomain.CreateContractRequest{
		Name:            "Wrong Org Delete",
		KmRate:          0.10,
		Currency:        "EUR",
		GovernanceModel: "creator_controlled",
		IsShared:        false,
	}
	created, err := repo.Create(context.Background(), orgID, req)
	require.NoError(t, err)

	// Try to delete with wrong org
	err = repo.Delete(context.Background(), wrongOrgID, created.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, contractdomain.ErrContractNotFound)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func float64Ptr(v float64) *float64 {
	return &v
}
