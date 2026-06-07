package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stretchr/testify/require"
)

func TestSubprojectRepository_Create_GetByID(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewSubprojectRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	projectID := seedProject(t, pool, orgID, now)

	sp := &models.Subproject{
		ProjectID:     projectID.String(),
		Name:          "Test Subproject",
		Description:   "A test subproject",
		SequenceOrder: 1,
		IsActive:      true,
	}

	created, err := repo.Create(context.Background(), sp)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotEmpty(t, created.ID)
	require.Equal(t, "Test Subproject", created.Name)
	require.Equal(t, projectID.String(), created.ProjectID)
	require.Equal(t, 1, created.SequenceOrder)
	require.True(t, created.IsActive)

	// Get by ID
	got, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.Name, got.Name)
}

func TestSubprojectRepository_GetByID_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewSubprojectRepository(pool)
	got, err := repo.GetByID(context.Background(), uuid.New().String())
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestSubprojectRepository_ListByProject(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewSubprojectRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	projectID := seedProject(t, pool, orgID, now)
	otherProjectID := seedProject(t, pool, orgID, now)

	// Create two subprojects under projectID
	sp1 := &models.Subproject{ProjectID: projectID.String(), Name: "Alpha", SequenceOrder: 1, IsActive: true}
	sp2 := &models.Subproject{ProjectID: projectID.String(), Name: "Beta", SequenceOrder: 2, IsActive: true}

	_, err := repo.Create(context.Background(), sp1)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), sp2)
	require.NoError(t, err)

	// List by project should return 2
	list, err := repo.ListByProject(context.Background(), projectID)
	require.NoError(t, err)
	require.Len(t, list, 2)

	// Other project should have 0
	empty, err := repo.ListByProject(context.Background(), otherProjectID)
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestSubprojectRepository_Update(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewSubprojectRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	projectID := seedProject(t, pool, orgID, now)

	sp := &models.Subproject{
		ProjectID:     projectID.String(),
		Name:          "Original Name",
		Description:   "Original desc",
		SequenceOrder: 1,
		IsActive:      true,
	}

	created, err := repo.Create(context.Background(), sp)
	require.NoError(t, err)

	// Update
	created.Name = "Updated Name"
	created.Description = "Updated desc"
	created.SequenceOrder = 5

	updated, err := repo.Update(context.Background(), created)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, "Updated Name", updated.Name)
	require.Equal(t, "Updated desc", updated.Description)
	require.Equal(t, 5, updated.SequenceOrder)
}

func TestSubprojectRepository_Delete(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewSubprojectRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	projectID := seedProject(t, pool, orgID, now)

	sp := &models.Subproject{
		ProjectID:     projectID.String(),
		Name:          "To Delete",
		SequenceOrder: 1,
		IsActive:      true,
	}

	created, err := repo.Create(context.Background(), sp)
	require.NoError(t, err)

	// Delete
	err = repo.Delete(context.Background(), created.ID)
	require.NoError(t, err)

	// Should not be found
	got, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Nil(t, got)
}
