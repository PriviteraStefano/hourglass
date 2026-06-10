package customer

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	customerdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/customer"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

// realRepoFixture creates a real CustomerRepository-backed *Service.
func realRepoFixture(t *testing.T, pool *pgxpool.Pool) *Service {
	t.Helper()
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	repo := postgres.NewCustomerRepository(pool)
	return NewService(repo)
}

func TestCustomerIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("CreateCustomer", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Cust Org", "cust-org-"+uuid.New().String()[:8], now, now)
		require.NoError(t, err)

		c, err := svc.Create(context.Background(), orgID, string(models.RoleFinance),
			&customerdomain.CreateCustomerRequest{
				CompanyName: "Acme Inc",
				ContactName: "John Contact",
				Email:       "acme@example.com",
			})
		require.NoError(t, err)
		require.NotNil(t, c)
		assert.NotEmpty(t, c.ID)
		assert.Equal(t, "Acme Inc", c.CompanyName)
		assert.True(t, c.IsActive)
	})

	t.Run("CreateCustomerForbiddenForNonFinance", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		_, err := svc.Create(context.Background(), uuid.New(), string(models.RoleEmployee),
			&customerdomain.CreateCustomerRequest{CompanyName: "Should Fail"})
		assert.ErrorIs(t, err, customerdomain.ErrForbidden)
	})

	t.Run("CreateCustomerEmptyName", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		_, err := svc.Create(context.Background(), uuid.New(), string(models.RoleFinance),
			&customerdomain.CreateCustomerRequest{CompanyName: ""})
		assert.ErrorIs(t, err, customerdomain.ErrInvalidCustomer)
	})

	t.Run("GetCustomer", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Get Cust Org", "getcust-org-"+uuid.New().String()[:8], now, now)
		require.NoError(t, err)

		created, err := svc.Create(context.Background(), orgID, string(models.RoleFinance),
			&customerdomain.CreateCustomerRequest{
				CompanyName: "Get Me Inc",
				ContactName: "Contact Person",
				Email:       "getme@example.com",
			})
		require.NoError(t, err)

		got, contracts, err := svc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Get Me Inc", got.CompanyName)
		assert.Empty(t, contracts)
	})

	t.Run("ListCustomers", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "List Cust Org", "listcust-org-"+uuid.New().String()[:8], now, now)
		require.NoError(t, err)

		_, err = svc.Create(context.Background(), orgID, string(models.RoleFinance),
			&customerdomain.CreateCustomerRequest{CompanyName: "Customer A"})
		require.NoError(t, err)
		_, err = svc.Create(context.Background(), orgID, string(models.RoleFinance),
			&customerdomain.CreateCustomerRequest{CompanyName: "Customer B"})
		require.NoError(t, err)

		customers, err := svc.List(context.Background(), orgID, 100, 0, "")
		require.NoError(t, err)
		assert.Len(t, customers, 2)
	})

	t.Run("DeactivateCustomer", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Deact Cust Org", "deact-org-"+uuid.New().String()[:8], now, now)
		require.NoError(t, err)

		created, err := svc.Create(context.Background(), orgID, string(models.RoleFinance),
			&customerdomain.CreateCustomerRequest{CompanyName: "To Deactivate"})
		require.NoError(t, err)

		err = svc.Delete(context.Background(), created.ID, orgID, string(models.RoleFinance))
		require.NoError(t, err)

		// Deactivate sets is_active to false. Customer is still gettable.
		got, _, err := svc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.False(t, got.IsActive)
	})
}
