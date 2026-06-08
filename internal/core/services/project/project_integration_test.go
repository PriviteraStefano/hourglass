package project

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	projectdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/project"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

// realRepoFixture creates a real ProjectRepository-backed *Service.
func realRepoFixture(t *testing.T, pool *pgxpool.Pool) *Service {
	t.Helper()
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	repo := postgres.NewProjectRepository(pool)
	return NewService(repo)
}

func TestProjectIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("CreateBillableProject", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Proj Org", "proj-org-"+uuid.New().String()[:8], now, now)
		require.NoError(t, err)

		contractID := uuid.New()
		_, err = pool.Exec(context.Background(),
			`INSERT INTO contracts (id, name, km_rate, currency, governance_model, created_by_org_id, is_shared, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, false, true, $7, $7)`,
			contractID, "Test Contract", 0.42, "EUR", "creator_controlled", orgID, now)
		require.NoError(t, err)

		resp, err := svc.Create(context.Background(), orgID,
			&projectdomain.CreateProjectRequest{
				Name:            "Billable Project",
				Type:            models.ProjectTypeBillable,
				ContractID:      contractID.String(),
				GovernanceModel: models.GovernanceCreatorControlled,
			})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.ID)
		assert.Equal(t, "Billable Project", resp.Name)
		assert.Equal(t, models.ProjectTypeBillable, resp.Type)
	})

	t.Run("CreateInternalProject", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Internal Org", "int-org-"+uuid.New().String()[:8], now, now)
		require.NoError(t, err)

		resp, err := svc.Create(context.Background(), orgID,
			&projectdomain.CreateProjectRequest{
				Name:            "Internal Project",
				Type:            models.ProjectTypeInternal,
				GovernanceModel: models.GovernanceCreatorControlled,
			})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "Internal Project", resp.Name)
		assert.Equal(t, models.ProjectTypeInternal, resp.Type)
	})

	t.Run("GetProject", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Get Org", "get-org-"+uuid.New().String()[:8], now, now)
		require.NoError(t, err)

		created, err := svc.Create(context.Background(), orgID,
			&projectdomain.CreateProjectRequest{
				Name:            "Project to Get",
				Type:            models.ProjectTypeInternal,
				GovernanceModel: models.GovernanceCreatorControlled,
			})
		require.NoError(t, err)

		got, err := svc.Get(context.Background(), orgID, created.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, created.ID, got.ID)
	})

	t.Run("ListProjects", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "List Org", "list-org-"+uuid.New().String()[:8], now, now)
		require.NoError(t, err)

		_, err = svc.Create(context.Background(), orgID,
			&projectdomain.CreateProjectRequest{
				Name: "Project A", Type: models.ProjectTypeInternal,
				GovernanceModel: models.GovernanceCreatorControlled,
			})
		require.NoError(t, err)

		_, err = svc.Create(context.Background(), orgID,
			&projectdomain.CreateProjectRequest{
				Name: "Project B", Type: models.ProjectTypeInternal,
				GovernanceModel: models.GovernanceCreatorControlled,
			})
		require.NoError(t, err)

		projects, err := svc.List(context.Background(), orgID, "", "")
		require.NoError(t, err)
		assert.Len(t, projects, 2)
	})

	t.Run("CreateWithInvalidName", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		_, err := svc.Create(context.Background(), uuid.New(),
			&projectdomain.CreateProjectRequest{
				Name: "",
				Type: models.ProjectTypeInternal,
				GovernanceModel: models.GovernanceCreatorControlled,
			})
		assert.ErrorIs(t, err, projectdomain.ErrInvalidRequest)
	})

	t.Run("GetNotFound", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		_, err := svc.Get(context.Background(), uuid.New(), uuid.New())
		assert.ErrorIs(t, err, projectdomain.ErrProjectNotFound)
	})
}
