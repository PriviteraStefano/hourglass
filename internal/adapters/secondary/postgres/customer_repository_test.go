package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/customer"
	"github.com/stretchr/testify/require"
)

func TestCustomerRepository_Create_GetByID(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCustomerRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	c := &customer.Customer{
		OrganizationID: orgID,
		CompanyName:    "ACME Corp",
		ContactName:    "John Doe",
		Email:          "john@acme.com",
		Phone:          "+1234567890",
		VATNumber:      "VAT123",
		Address:        "123 Main St",
		IsActive:       true,
	}

	created, err := repo.Create(context.Background(), c)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotEmpty(t, created.ID)
	require.Equal(t, "ACME Corp", created.CompanyName)
	require.Equal(t, orgID, created.OrganizationID)
	require.Equal(t, "John Doe", created.ContactName)
	require.True(t, created.IsActive)

	// Get by ID
	got, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "ACME Corp", got.CompanyName)
}

func TestCustomerRepository_GetByID_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCustomerRepository(pool)
	_, err := repo.GetByID(context.Background(), uuid.New())
	require.Error(t, err)
	require.ErrorIs(t, err, customer.ErrCustomerNotFound)
}

func TestCustomerRepository_ListByOrg(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCustomerRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	otherOrgID := seedOrg(t, pool, now)

	// Create two customers for orgID
	c1 := &customer.Customer{OrganizationID: orgID, CompanyName: "Alpha Corp", IsActive: true}
	c2 := &customer.Customer{OrganizationID: orgID, CompanyName: "Beta Inc", IsActive: true}

	_, err := repo.Create(context.Background(), c1)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), c2)
	require.NoError(t, err)

	// List by org should return 2
	list, err := repo.ListByOrg(context.Background(), orgID, 100, 0, "")
	require.NoError(t, err)
	require.Len(t, list, 2)

	// Other org should have 0
	empty, err := repo.ListByOrg(context.Background(), otherOrgID, 100, 0, "")
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestCustomerRepository_Update(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCustomerRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	c := &customer.Customer{
		OrganizationID: orgID,
		CompanyName:    "Old Name",
		ContactName:    "Old Contact",
		IsActive:       true,
	}

	created, err := repo.Create(context.Background(), c)
	require.NoError(t, err)

	// Update company name
	created.CompanyName = "New Name"
	created.ContactName = "New Contact"

	updated, err := repo.Update(context.Background(), created)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, "New Name", updated.CompanyName)
	require.Equal(t, "New Contact", updated.ContactName)
}

func TestCustomerRepository_Deactivate(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCustomerRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	c := &customer.Customer{
		OrganizationID: orgID,
		CompanyName:    "To Deactivate",
		IsActive:       true,
	}

	created, err := repo.Create(context.Background(), c)
	require.NoError(t, err)

	// Deactivate
	err = repo.Deactivate(context.Background(), created.ID)
	require.NoError(t, err)

	// Verify
	got, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.False(t, got.IsActive)
}

func TestCustomerRepository_Deactivate_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCustomerRepository(pool)
	err := repo.Deactivate(context.Background(), uuid.New())
	require.Error(t, err)
	require.ErrorIs(t, err, customer.ErrCustomerNotFound)
}

func TestCustomerRepository_ListContractsByCustomer(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	customerRepo := NewCustomerRepository(pool)
	contractRepo := NewContractRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	// Create a customer
	c := &customer.Customer{OrganizationID: orgID, CompanyName: "Customer Co", IsActive: true}
	created, err := customerRepo.Create(context.Background(), c)
	require.NoError(t, err)

	// Initially 0 contracts
	count, err := customerRepo.CountContractsByCustomer(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	summaries, err := customerRepo.ListContractsByCustomer(context.Background(), created.ID)
	require.NoError(t, err)
	require.Empty(t, summaries)

	// Create a contract referencing this customer
	custID := created.ID
	req := &contractdomain.CreateContractRequest{
		Name:            "Customer Contract",
		KmRate:          0.30,
		Currency:        "EUR",
		GovernanceModel: "creator_controlled",
		IsShared:        false,
	}
	ct, err := contractRepo.Create(context.Background(), orgID, req)
	require.NoError(t, err)

	// Update the contract's customer_id
	isShared := false
	_, _, err = contractRepo.Update(context.Background(), orgID, ct.ID, &contractdomain.UpdateContractRequest{
		Name:     ct.Name,
		KmRate:   &ct.KmRate,
		Currency: ct.Currency,
		IsShared: &isShared,
		CustomerID: func() *string { s := custID.String(); return &s }(),
	})
	require.NoError(t, err)

	// Now should have 1 contract
	count, err = customerRepo.CountContractsByCustomer(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	summaries, err = customerRepo.ListContractsByCustomer(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.Equal(t, ct.ID, summaries[0].ID)
}
