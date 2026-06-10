package unit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	unitdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
)

func setupService(t *testing.T) (*Service, *testdata.MockUnitRepo) {
	t.Helper()
	repo := &testdata.MockUnitRepo{}
	svc := NewService(repo)
	return svc, repo
}

func seedUnit(repo *testdata.MockUnitRepo, overrides ...func(*unitdomain.Unit)) *unitdomain.Unit {
	u := &unitdomain.Unit{
		ID:             uuid.New().String(),
		OrgID:          uuid.New(),
		Name:           "Test Unit",
		HierarchyLevel: 1,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	for _, o := range overrides {
		o(u)
	}
	if repo.Units == nil {
		repo.Units = make(map[string]*unitdomain.Unit)
	}
	repo.Units[u.ID] = u
	return u
}

func TestService_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     *unitdomain.CreateUnitRequest
		wantErr error
	}{
		{
			name: "valid unit",
			req: &unitdomain.CreateUnitRequest{
				OrgID: uuid.New(),
				Name:  "Engineering",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := setupService(t)
			result, err := svc.Create(context.Background(), tt.req)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
				return
			}
			assert.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.req.Name, result.Name)
			assert.Equal(t, tt.req.OrgID, result.OrgID)
		})
	}
}

func TestService_ListByOrg(t *testing.T) {
	svc, repo := setupService(t)
	orgID := uuid.New()
	seedUnit(repo, func(u *unitdomain.Unit) { u.OrgID = orgID })
	seedUnit(repo, func(u *unitdomain.Unit) { u.OrgID = orgID })
	seedUnit(repo, func(u *unitdomain.Unit) { u.OrgID = orgID })

	results, err := svc.ListByOrg(context.Background(), orgID)
	assert.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestService_Get(t *testing.T) {
	svc, repo := setupService(t)
	seeded := seedUnit(repo)

	t.Run("existing", func(t *testing.T) {
		result, err := svc.Get(context.Background(), seeded.ID)
		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, seeded.ID, result.ID)
		assert.Equal(t, seeded.Name, result.Name)
	})

	t.Run("not found", func(t *testing.T) {
		result, err := svc.Get(context.Background(), "nonexistent-id")
		assert.ErrorIs(t, err, unitdomain.ErrUnitNotFound)
		assert.Nil(t, result)
	})
}

func TestService_Update(t *testing.T) {
	svc, repo := setupService(t)
	seeded := seedUnit(repo)

	result, err := svc.Update(context.Background(), seeded.ID, &unitdomain.UpdateUnitRequest{
		Name: "Updated Engineering",
		Code: "ENG-2",
	})
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Updated Engineering", result.Name)
	assert.Equal(t, "ENG-2", result.Code)
}

func TestService_Delete(t *testing.T) {
	svc, repo := setupService(t)

	t.Run("delete empty unit", func(t *testing.T) {
		seeded := seedUnit(repo)
		err := svc.Delete(context.Background(), seeded.ID)
		assert.NoError(t, err)
	})
}
