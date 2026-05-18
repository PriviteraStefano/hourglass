package surrealdb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	customerdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/customer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCustomerRepo(t *testing.T) (*CustomerRepository, func()) {
	t.Helper()

	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	ns := "test_customer_" + uuid.New().String()
	db, err := GetTestDBWithNamespace(ns, ns)
	if err != nil {
		t.Skipf("SurrealDB not available: %v", err)
	}

	repo := NewCustomerRepository(db)
	return repo, func() { db.Close(context.Background()) }
}

func seedCustomerOrg(t *testing.T, repo *CustomerRepository) uuid.UUID {
	t.Helper()

	orgRepo := NewOrganizationRepository(repo.db)
	orgID := uuid.New()
	org := &auth.Organization{
		ID:        orgID,
		Name:      "Customer Test Org " + uuid.New().String()[:8],
		Slug:      "cust-org-" + uuid.New().String()[:8],
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := orgRepo.Add(context.Background(), org)
	require.NoError(t, err, "failed to seed org")

	return orgID
}

func TestCustomerRepo_Create(t *testing.T) {
	repo, cleanup := setupCustomerRepo(t)
	defer cleanup()
	orgID := seedCustomerOrg(t, repo)

	customer := &customerdomain.Customer{
		ID:             uuid.New(),
		OrganizationID: orgID,
		CompanyName:    "Acme Corp " + uuid.New().String()[:8],
		ContactName:    "John Doe",
		Email:          "john@acme.com",
		Phone:          "+1234567890",
		VATNumber:      "VAT123",
		Address:        "123 Main St",
		IsActive:       true,
		CreatedAt:      time.Now(),
	}

	created, err := repo.Create(context.Background(), customer)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, customer.CompanyName, created.CompanyName)
	assert.Equal(t, customer.ContactName, created.ContactName)
	assert.Equal(t, customer.Email, created.Email)
	assert.True(t, created.IsActive)
}

func TestCustomerRepo_GetByID(t *testing.T) {
	repo, cleanup := setupCustomerRepo(t)
	defer cleanup()
	orgID := seedCustomerOrg(t, repo)

	t.Run("existing", func(t *testing.T) {
		customer := &customerdomain.Customer{
			ID:             uuid.New(),
			OrganizationID: orgID,
			CompanyName:    "Get Test Corp",
			ContactName:    "Jane Smith",
			IsActive:       true,
			CreatedAt:      time.Now(),
		}

		created, err := repo.Create(context.Background(), customer)
		require.NoError(t, err)

		found, err := repo.GetByID(context.Background(), created.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, created.CompanyName, found.CompanyName)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := repo.GetByID(context.Background(), uuid.New())
		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestCustomerRepo_ListByOrg(t *testing.T) {
	repo, cleanup := setupCustomerRepo(t)
	defer cleanup()
	orgID := seedCustomerOrg(t, repo)

	customers := []*customerdomain.Customer{
		{
			ID:             uuid.New(),
			OrganizationID: orgID,
			CompanyName:    "Company Alpha",
			IsActive:       true,
			CreatedAt:      time.Now(),
		},
		{
			ID:             uuid.New(),
			OrganizationID: orgID,
			CompanyName:    "Company Beta",
			IsActive:       true,
			CreatedAt:      time.Now(),
		},
	}

	for _, c := range customers {
		_, err := repo.Create(context.Background(), c)
		require.NoError(t, err)
	}

	results, err := repo.ListByOrg(context.Background(), orgID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestCustomerRepo_Update(t *testing.T) {
	repo, cleanup := setupCustomerRepo(t)
	defer cleanup()
	orgID := seedCustomerOrg(t, repo)

	customer := &customerdomain.Customer{
		ID:             uuid.New(),
		OrganizationID: orgID,
		CompanyName:    "Old Name Inc",
		ContactName:    "Old Contact",
		Email:          "old@example.com",
		IsActive:       true,
		CreatedAt:      time.Now(),
	}

	created, err := repo.Create(context.Background(), customer)
	require.NoError(t, err)

	created.CompanyName = "New Name Inc"
	created.ContactName = "New Contact"
	created.Email = "new@example.com"

	updated, err := repo.Update(context.Background(), created)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "New Name Inc", updated.CompanyName)
	assert.Equal(t, "New Contact", updated.ContactName)
	assert.Equal(t, "new@example.com", updated.Email)
}

func TestCustomerRepo_Deactivate(t *testing.T) {
	repo, cleanup := setupCustomerRepo(t)
	defer cleanup()
	orgID := seedCustomerOrg(t, repo)

	customer := &customerdomain.Customer{
		ID:             uuid.New(),
		OrganizationID: orgID,
		CompanyName:    "To Deactivate",
		IsActive:       true,
		CreatedAt:      time.Now(),
	}

	created, err := repo.Create(context.Background(), customer)
	require.NoError(t, err)

	err = repo.Deactivate(context.Background(), created.ID)
	require.NoError(t, err)

	found, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.False(t, found.IsActive)
}
