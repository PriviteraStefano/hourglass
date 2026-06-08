package unit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
)

// realRepoFixture creates a real UnitRepository-backed *Service.
func realRepoFixture(t *testing.T, pool *pgxpool.Pool) *Service {
	t.Helper()
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	repo := postgres.NewUnitRepository(pool)
	return NewService(repo)
}

func TestUnitIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("CreateAndGetByID", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Org for Units", "org-units-"+uuid.New().String()[:8], time.Now(), time.Now())
		require.NoError(t, err)

		req := &unit.CreateUnitRequest{
			OrgID: orgID,
			Name:  "Engineering",
			Code:  "ENG",
		}
		u, err := svc.Create(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, u)
		assert.NotEmpty(t, u.ID)
		assert.Equal(t, "Engineering", u.Name)
		assert.Equal(t, "ENG", u.Code)
		assert.Equal(t, 0, u.HierarchyLevel)

		got, err := svc.Get(context.Background(), u.ID)
		require.NoError(t, err)
		assert.Equal(t, u.ID, got.ID)
		assert.Equal(t, u.Name, got.Name)
	})

	t.Run("ListByOrg", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Org for List", "org-list-"+uuid.New().String()[:8], time.Now(), time.Now())
		require.NoError(t, err)

		// Create two units
		_, err = svc.Create(context.Background(), &unit.CreateUnitRequest{
			OrgID: orgID, Name: "Unit A", Code: "UA",
		})
		require.NoError(t, err)
		_, err = svc.Create(context.Background(), &unit.CreateUnitRequest{
			OrgID: orgID, Name: "Unit B", Code: "UB",
		})
		require.NoError(t, err)

		units, err := svc.ListByOrg(context.Background(), orgID)
		require.NoError(t, err)
		assert.Len(t, units, 2)
	})

	t.Run("Update", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Org for Update", "org-upd-"+uuid.New().String()[:8], time.Now(), time.Now())
		require.NoError(t, err)

		u, err := svc.Create(context.Background(), &unit.CreateUnitRequest{
			OrgID: orgID, Name: "Original", Code: "ORIG",
		})
		require.NoError(t, err)

		updated, err := svc.Update(context.Background(), u.ID, &unit.UpdateUnitRequest{
			Name: "Updated Name",
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name)

		got, err := svc.Get(context.Background(), u.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", got.Name)
	})

	t.Run("CreateWithParent", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Org for Parent", "org-par-"+uuid.New().String()[:8], time.Now(), time.Now())
		require.NoError(t, err)

		parent, err := svc.Create(context.Background(), &unit.CreateUnitRequest{
			OrgID: orgID, Name: "Parent Unit", Code: "PAR",
		})
		require.NoError(t, err)

		child, err := svc.Create(context.Background(), &unit.CreateUnitRequest{
			OrgID:        orgID,
			Name:         "Child Unit",
			Code:         "CHILD",
			ParentUnitID: parent.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, parent.ID, child.ParentUnitID)
		assert.Equal(t, 1, child.HierarchyLevel)
	})

	t.Run("Delete", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Org for Delete", "org-del-"+uuid.New().String()[:8], time.Now(), time.Now())
		require.NoError(t, err)

		u, err := svc.Create(context.Background(), &unit.CreateUnitRequest{
			OrgID: orgID, Name: "To Delete", Code: "DEL",
		})
		require.NoError(t, err)

		err = svc.Delete(context.Background(), u.ID)
		require.NoError(t, err)

		got, err := svc.Get(context.Background(), u.ID)
		assert.ErrorIs(t, err, unit.ErrUnitNotFound)
		assert.Nil(t, got)
	})

	t.Run("GetInvalidID", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		_, err := svc.Get(context.Background(), uuid.New().String())
		assert.ErrorIs(t, err, unit.ErrUnitNotFound)
	})
}
