package project

import (
	"context"

	"github.com/google/uuid"
	projectdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/project"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

type Service struct {
	repo ports.ProjectRepository
}

func NewService(repo ports.ProjectRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, orgID uuid.UUID, scope, contractID string) ([]projectdomain.ProjectResponse, error) {
	return s.repo.List(ctx, orgID, scope, contractID)
}

func (s *Service) Create(ctx context.Context, orgID uuid.UUID, req *projectdomain.CreateProjectRequest) (*projectdomain.ProjectResponse, error) {
	if req.Name == "" || !req.Type.IsValid() || !req.GovernanceModel.IsValid() {
		return nil, projectdomain.ErrInvalidRequest
	}
	return s.repo.Create(ctx, orgID, req)
}

func (s *Service) Get(ctx context.Context, orgID, projectID uuid.UUID) (*projectdomain.ProjectResponse, error) {
	return s.repo.Get(ctx, orgID, projectID)
}

func (s *Service) Adopt(ctx context.Context, orgID, projectID uuid.UUID) (*projectdomain.ProjectAdoption, error) {
	return s.repo.Adopt(ctx, orgID, projectID)
}

func (s *Service) ListManagers(ctx context.Context, projectID uuid.UUID) ([]projectdomain.ProjectManager, error) {
	return s.repo.ListManagers(ctx, projectID)
}

func (s *Service) AddManager(ctx context.Context, actorRole string, projectID, userID uuid.UUID) (*projectdomain.ProjectManager, error) {
	if actorRole != string(models.RoleFinance) {
		return nil, projectdomain.ErrForbidden
	}
	return s.repo.AddManager(ctx, projectID, userID)
}

func (s *Service) RemoveManager(ctx context.Context, actorRole string, projectID, userID uuid.UUID) error {
	if actorRole != string(models.RoleFinance) {
		return projectdomain.ErrForbidden
	}
	return s.repo.RemoveManager(ctx, projectID, userID)
}
