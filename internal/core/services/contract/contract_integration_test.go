package contract

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

// realRepoFixture creates a real ContractRepository-backed *Service.
func realRepoFixture(t *testing.T, pool *pgxpool.Pool) *Service {
	t.Helper()
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	repo := postgres.NewContractRepository(pool)
	return NewService(repo)
}

func TestContractIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("CreateContractWithCustomer", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Contract Org", "ct-org-"+uuid.New().String()[:8], now, now)
		require.NoError(t, err)

		customerID := uuid.New()
		_, err = pool.Exec(context.Background(),
			`INSERT INTO customers (id, org_id, name, contact_name, email, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, true, $6, $6)`,
			customerID, orgID, "Test Customer", "Contact", "customer@test.com", now)
		require.NoError(t, err)

		req := &contractdomain.CreateContractRequest{
			Name:            "Contract with Customer",
			KmRate:          0.35,
			Currency:        "EUR",
			GovernanceModel: models.GovernanceCreatorControlled,
		}
		resp, err := svc.Create(context.Background(), orgID, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.ID)
		assert.Equal(t, "Contract with Customer", resp.Name)
	})

	t.Run("CreateContractWithoutCustomer", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Org No Cust", "org-nocus-"+uuid.New().String()[:8], now, now)
		require.NoError(t, err)

		resp, err := svc.Create(context.Background(), orgID,
			&contractdomain.CreateContractRequest{
				Name:            "Contract without Customer",
				KmRate:          0.50,
				Currency:        "USD",
				GovernanceModel: models.GovernanceCreatorControlled,
			})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "Contract without Customer", resp.Name)
	})

	t.Run("CreateWithInvalidName", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		_, err := svc.Create(context.Background(), uuid.New(),
			&contractdomain.CreateContractRequest{
				Name:            "",
				GovernanceModel: models.GovernanceCreatorControlled,
			})
		assert.ErrorIs(t, err, contractdomain.ErrInvalidRequest)
	})

	t.Run("GetContract", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Get Contract Org", "getct-org-"+uuid.New().String()[:8], now, now)
		require.NoError(t, err)

		created, err := svc.Create(context.Background(), orgID,
			&contractdomain.CreateContractRequest{
				Name: "Contract to Get",
				KmRate: 0.40,
				Currency: "EUR",
				GovernanceModel: models.GovernanceCreatorControlled,
			})
		require.NoError(t, err)

		got, err := svc.Get(context.Background(), orgID, created.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, created.ID, got.ID)
	})

	t.Run("ListContracts", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "List Contract Org", "listct-org-"+uuid.New().String()[:8], now, now)
		require.NoError(t, err)

		_, err = svc.Create(context.Background(), orgID,
			&contractdomain.CreateContractRequest{
				Name: "Contract 1", KmRate: 0.30, Currency: "EUR",
				GovernanceModel: models.GovernanceCreatorControlled,
			})
		require.NoError(t, err)

		_, err = svc.Create(context.Background(), orgID,
			&contractdomain.CreateContractRequest{
				Name: "Contract 2", KmRate: 0.50, Currency: "USD",
				GovernanceModel: models.GovernanceCreatorControlled,
			})
		require.NoError(t, err)

		contracts, err := svc.List(context.Background(), orgID, "", nil)
		require.NoError(t, err)
		assert.Len(t, contracts, 2)
	})
}
