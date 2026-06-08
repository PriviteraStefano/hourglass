package working_group

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
)

// realRepoFixture creates a real WorkingGroupRepository-backed *Service.
func realRepoFixture(t *testing.T, pool *pgxpool.Pool) *Service {
	t.Helper()
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	repo := postgres.NewWorkingGroupRepository(pool)
	return NewService(repo)
}

// seedWGData creates org, manager user, project, and subproject.
// Returns (orgID, managerID, subprojectID).
func seedWGData(t *testing.T, pool *pgxpool.Pool, suffix string) (uuid.UUID, uuid.UUID, uuid.UUID) {
	now := time.Now()

	orgID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		orgID, "WG Org "+suffix, "wg-"+suffix+"-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)

	managerID := uuid.New()
	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7)`,
		managerID, "mgr-"+suffix+"@test.com", "mgr-"+suffix+"-"+uuid.New().String()[:8],
		"Manager", suffix, "hash", now)
	require.NoError(t, err)

	projectID := uuid.New()
	_, err = pool.Exec(context.Background(),
		`INSERT INTO projects (id, org_id, name, project_type, type, governance_model, created_by_org_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $4, $5, $2, $6, $6)`,
		projectID, orgID, "Project "+suffix, "billable", "creator_controlled", now)
	require.NoError(t, err)

	subprojectID := uuid.New()
	_, err = pool.Exec(context.Background(),
		`INSERT INTO subprojects (id, project_id, name, created_at, updated_at) VALUES ($1, $2, $3, $4, $4)`,
		subprojectID, projectID, "Subproject "+suffix, now)
	require.NoError(t, err)

	return orgID, managerID, subprojectID
}

func TestWorkingGroupIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("CreateAndGetByID", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		orgID, managerID, subprojectID := seedWGData(t, pool, "create")

		wg, err := svc.Create(context.Background(), &working_group.CreateWorkingGroupRequest{
			OrgID:        orgID,
			SubprojectID: subprojectID,
			Name:         "Alpha Team",
			Description:  "The alpha team",
			ManagerID:    managerID,
		})
		require.NoError(t, err)
		require.NotNil(t, wg)
		assert.NotEmpty(t, wg.ID)
		assert.Equal(t, "Alpha Team", wg.Name)
		assert.True(t, wg.IsActive)
		assert.Equal(t, subprojectID, wg.SubprojectID)

		got, err := svc.Get(context.Background(), wg.ID)
		require.NoError(t, err)
		assert.Equal(t, wg.ID, got.ID)
		assert.Equal(t, "Alpha Team", got.Name)
	})

	t.Run("ListByOrg", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		orgID, managerID, subprojectID := seedWGData(t, pool, "list")

		// Create two WGs
		_, err := svc.Create(context.Background(), &working_group.CreateWorkingGroupRequest{
			OrgID: orgID, SubprojectID: subprojectID, ManagerID: managerID, Name: "WG1",
		})
		require.NoError(t, err)
		_, err = svc.Create(context.Background(), &working_group.CreateWorkingGroupRequest{
			OrgID: orgID, SubprojectID: subprojectID, ManagerID: managerID, Name: "WG2",
		})
		require.NoError(t, err)

		groups, err := svc.ListByOrg(context.Background(), orgID, nil)
		require.NoError(t, err)
		assert.Len(t, groups, 2)
	})

	t.Run("GetNotFound", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		_, err := svc.Get(context.Background(), uuid.New())
		assert.ErrorIs(t, err, working_group.ErrWorkingGroupNotFound)
	})

	t.Run("Delete", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		orgID, managerID, subprojectID := seedWGData(t, pool, "delete")

		wg, err := svc.Create(context.Background(), &working_group.CreateWorkingGroupRequest{
			OrgID: orgID, SubprojectID: subprojectID, ManagerID: managerID, Name: "To Delete",
		})
		require.NoError(t, err)

		err = svc.Delete(context.Background(), wg.ID)
		require.NoError(t, err)

		_, err = svc.Get(context.Background(), wg.ID)
		assert.ErrorIs(t, err, working_group.ErrWorkingGroupNotFound)
	})
}
