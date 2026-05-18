package working_group

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	wgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
)

func setupService(t *testing.T) (*Service, *testdata.MockWorkingGroupRepo) {
	t.Helper()
	repo := &testdata.MockWorkingGroupRepo{}
	svc := NewService(repo)
	return svc, repo
}

func seedWorkingGroup(repo *testdata.MockWorkingGroupRepo, overrides ...func(*wgdomain.WorkingGroup)) *wgdomain.WorkingGroup {
	wg := &wgdomain.WorkingGroup{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		Name:      "Test Working Group",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	for _, o := range overrides {
		o(wg)
	}
	if repo.Groups == nil {
		repo.Groups = make(map[uuid.UUID]*wgdomain.WorkingGroup)
	}
	repo.Groups[wg.ID] = wg
	return wg
}

func TestService_Create(t *testing.T) {
	t.Parallel()

	t.Run("valid working group", func(t *testing.T) {
		svc, _ := setupService(t)
		result, err := svc.Create(context.Background(), &wgdomain.CreateWorkingGroupRequest{
			OrgID: uuid.New(),
			Name:  "Design Team",
		})
		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "Design Team", result.Name)
		assert.True(t, result.IsActive)
	})
}

func TestService_ListByOrg(t *testing.T) {
	svc, repo := setupService(t)
	orgID := uuid.New()
	seedWorkingGroup(repo, func(wg *wgdomain.WorkingGroup) { wg.OrgID = orgID })
	seedWorkingGroup(repo, func(wg *wgdomain.WorkingGroup) { wg.OrgID = orgID })

	results, err := svc.ListByOrg(context.Background(), orgID, nil)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestService_Get(t *testing.T) {
	svc, repo := setupService(t)
	seeded := seedWorkingGroup(repo)

	t.Run("existing", func(t *testing.T) {
		result, err := svc.Get(context.Background(), seeded.ID)
		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, seeded.ID, result.ID)
		assert.Equal(t, seeded.Name, result.Name)
	})

	t.Run("not found", func(t *testing.T) {
		result, err := svc.Get(context.Background(), uuid.New())
		assert.ErrorIs(t, err, wgdomain.ErrWorkingGroupNotFound)
		assert.Nil(t, result)
	})
}

func TestService_Update(t *testing.T) {
	svc, repo := setupService(t)
	seeded := seedWorkingGroup(repo)

	result, err := svc.Update(context.Background(), seeded.ID, &wgdomain.UpdateWorkingGroupRequest{
		Name: "Updated Team",
	})
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Updated Team", result.Name)
}

func TestService_Delete(t *testing.T) {
	svc, repo := setupService(t)

	t.Run("delete empty group", func(t *testing.T) {
		seeded := seedWorkingGroup(repo)
		err := svc.Delete(context.Background(), seeded.ID)
		assert.NoError(t, err)
	})
}

func TestService_AddMember(t *testing.T) {
	svc, _ := setupService(t)
	result, err := svc.AddMember(context.Background(), &wgdomain.AddMemberRequest{
		WGID:   uuid.New(),
		UserID: uuid.New(),
		Role:   "employee",
	})
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "employee", result.Role)
}

func TestService_RemoveMember(t *testing.T) {
	svc, _ := setupService(t)
	err := svc.RemoveMember(context.Background(), uuid.New())
	assert.NoError(t, err)
}
