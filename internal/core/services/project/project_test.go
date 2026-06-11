package project

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	projectdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/project"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

func setupService(t *testing.T) (*Service, *testdata.MockProjectRepo) {
	t.Helper()
	repo := &testdata.MockProjectRepo{}
	svc := NewService(repo)
	return svc, repo
}

func seedProject(repo *testdata.MockProjectRepo, overrides ...func(*projectdomain.ProjectResponse)) *projectdomain.ProjectResponse {
	p := &projectdomain.ProjectResponse{
		Project: projectdomain.Project{
			ID:              uuid.New(),
			Name:            "Test Project",
			Type:            models.ProjectTypeBillable,
			ContractID:      uuid.New(),
			GovernanceModel: models.GovernanceCreatorControlled,
			CreatedByOrgID:  uuid.New(),
			IsActive:        true,
			CreatedAt:       time.Now(),
		},
	}
	for _, o := range overrides {
		o(p)
	}
	if repo.Projects == nil {
		repo.Projects = make(map[uuid.UUID]*projectdomain.ProjectResponse)
	}
	repo.Projects[p.ID] = p
	return p
}

func TestService_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     *projectdomain.CreateProjectRequest
		wantErr error
	}{
		{
			name: "valid billable project",
			req: &projectdomain.CreateProjectRequest{
				Name:            "Billable Project",
				Type:            models.ProjectTypeBillable,
				GovernanceModel: models.GovernanceCreatorControlled,
			},
			wantErr: nil,
		},
		{
			name: "valid internal project",
			req: &projectdomain.CreateProjectRequest{
				Name:            "Internal Project",
				Type:            models.ProjectTypeInternal,
				GovernanceModel: models.GovernanceCreatorControlled,
			},
			wantErr: nil,
		},
		{
			name: "with contract",
			req: &projectdomain.CreateProjectRequest{
				Name:            "Project With Contract",
				Type:            models.ProjectTypeBillable,
				ContractID:      uuid.New().String(),
				GovernanceModel: models.GovernanceCreatorControlled,
			},
			wantErr: nil,
		},
		{
			name: "missing name",
			req: &projectdomain.CreateProjectRequest{
				Name:            "",
				Type:            models.ProjectTypeBillable,
				GovernanceModel: models.GovernanceCreatorControlled,
			},
			wantErr: projectdomain.ErrInvalidRequest,
		},
		{
			name: "invalid type",
			req: &projectdomain.CreateProjectRequest{
				Name:            "Bad Type",
				Type:            "invalid_type",
				GovernanceModel: models.GovernanceCreatorControlled,
			},
			wantErr: projectdomain.ErrInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := setupService(t)
			result, err := svc.Create(context.Background(), uuid.New(), tt.req)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestService_List(t *testing.T) {
	svc, repo := setupService(t)
	orgID := uuid.New()
	seedProject(repo, func(p *projectdomain.ProjectResponse) { p.CreatedByOrgID = orgID })
	seedProject(repo, func(p *projectdomain.ProjectResponse) { p.CreatedByOrgID = orgID })

	results, err := svc.List(context.Background(), orgID, "", "")
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestService_Get(t *testing.T) {
	svc, repo := setupService(t)
	orgID := uuid.New()
	seeded := seedProject(repo, func(p *projectdomain.ProjectResponse) { p.CreatedByOrgID = orgID })

	t.Run("existing", func(t *testing.T) {
		result, err := svc.Get(context.Background(), orgID, seeded.ID)
		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, seeded.ID, result.ID)
		assert.Equal(t, seeded.Name, result.Name)
	})

	t.Run("not found", func(t *testing.T) {
		result, err := svc.Get(context.Background(), orgID, uuid.New())
		assert.ErrorIs(t, err, projectdomain.ErrProjectNotFound)
		assert.Nil(t, result)
	})
}

func TestService_AddManager(t *testing.T) {
	t.Parallel()

	t.Run("finance adds manager", func(t *testing.T) {
		svc, _ := setupService(t)
		manager, err := svc.AddManager(context.Background(), string(models.RoleFinance), uuid.New(), uuid.New())
		assert.NoError(t, err)
		assert.NotNil(t, manager)
	})

	t.Run("non-finance forbidden", func(t *testing.T) {
		svc, _ := setupService(t)
		manager, err := svc.AddManager(context.Background(), string(models.RoleEmployee), uuid.New(), uuid.New())
		assert.ErrorIs(t, err, projectdomain.ErrForbidden)
		assert.Nil(t, manager)
	})
}

func TestService_RemoveManager(t *testing.T) {
	t.Parallel()

	t.Run("finance removes manager", func(t *testing.T) {
		svc, _ := setupService(t)
		err := svc.RemoveManager(context.Background(), string(models.RoleFinance), uuid.New(), uuid.New())
		assert.NoError(t, err)
	})

	t.Run("non-finance forbidden", func(t *testing.T) {
		svc, _ := setupService(t)
		err := svc.RemoveManager(context.Background(), string(models.RoleEmployee), uuid.New(), uuid.New())
		assert.ErrorIs(t, err, projectdomain.ErrForbidden)
	})
}

func TestService_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		role    string
		wantErr error
	}{
		{name: "finance role updates", role: string(models.RoleFinance), wantErr: nil},
		{name: "non-finance role forbidden", role: string(models.RoleEmployee), wantErr: projectdomain.ErrForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := setupService(t)
			orgID := uuid.New()
			seeded := seedProject(repo, func(p *projectdomain.ProjectResponse) { p.CreatedByOrgID = orgID })

			result, err := svc.Update(context.Background(), tt.role, orgID, seeded.ID, &projectdomain.UpdateProjectRequest{
				Name:            "Updated Project",
				Type:            models.ProjectTypeInternal,
				GovernanceModel: models.GovernanceCreatorControlled,
				IsShared:        false,
			})
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestService_Delete(t *testing.T) {
	t.Parallel()

	t.Run("finance role deletes", func(t *testing.T) {
		svc, repo := setupService(t)
		orgID := uuid.New()
		seeded := seedProject(repo, func(p *projectdomain.ProjectResponse) { p.CreatedByOrgID = orgID })
		err := svc.Delete(context.Background(), string(models.RoleFinance), orgID, seeded.ID)
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		svc, _ := setupService(t)
		err := svc.Delete(context.Background(), string(models.RoleFinance), uuid.New(), uuid.New())
		assert.Error(t, err)
	})

	t.Run("unauthorized role", func(t *testing.T) {
		svc, repo := setupService(t)
		orgID := uuid.New()
		seeded := seedProject(repo, func(p *projectdomain.ProjectResponse) { p.CreatedByOrgID = orgID })
		err := svc.Delete(context.Background(), string(models.RoleEmployee), orgID, seeded.ID)
		assert.ErrorIs(t, err, projectdomain.ErrForbidden)
	})

	t.Run("not owner", func(t *testing.T) {
		svc, repo := setupService(t)
		orgID := uuid.New()
		otherOrgID := uuid.New()
		seeded := seedProject(repo, func(p *projectdomain.ProjectResponse) { p.CreatedByOrgID = otherOrgID })
		err := svc.Delete(context.Background(), string(models.RoleFinance), orgID, seeded.ID)
		assert.ErrorIs(t, err, projectdomain.ErrForbidden)
	})

	t.Run("blocked by time entries", func(t *testing.T) {
		svc, repo := setupService(t)
		orgID := uuid.New()
		seeded := seedProject(repo, func(p *projectdomain.ProjectResponse) { p.CreatedByOrgID = orgID })
		repo.HasActiveTimeEntriesFn = func(ctx context.Context, projectID uuid.UUID) (bool, bool, error) {
			return true, false, nil
		}
		err := svc.Delete(context.Background(), string(models.RoleFinance), orgID, seeded.ID)
		assert.ErrorIs(t, err, projectdomain.ErrHasActiveTimeEntries)
	})

	t.Run("blocked by subproject entries", func(t *testing.T) {
		svc, repo := setupService(t)
		orgID := uuid.New()
		seeded := seedProject(repo, func(p *projectdomain.ProjectResponse) { p.CreatedByOrgID = orgID })
		repo.HasActiveTimeEntriesFn = func(ctx context.Context, projectID uuid.UUID) (bool, bool, error) {
			return false, true, nil
		}
		err := svc.Delete(context.Background(), string(models.RoleFinance), orgID, seeded.ID)
		assert.ErrorIs(t, err, projectdomain.ErrHasActiveSubprojectEntries)
	})
}
