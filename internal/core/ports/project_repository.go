package ports

import (
	"context"

	"github.com/google/uuid"
	projectdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/project"
)

type ProjectRepository interface {
	List(ctx context.Context, orgID uuid.UUID, scope, contractID string) ([]projectdomain.ProjectResponse, error)
	Create(ctx context.Context, orgID uuid.UUID, req *projectdomain.CreateProjectRequest) (*projectdomain.ProjectResponse, error)
	Get(ctx context.Context, orgID, projectID uuid.UUID) (*projectdomain.ProjectResponse, error)
	Adopt(ctx context.Context, orgID, projectID uuid.UUID) (*projectdomain.ProjectAdoption, error)

	ListManagers(ctx context.Context, projectID uuid.UUID) ([]projectdomain.ProjectManager, error)
	AddManager(ctx context.Context, projectID, userID uuid.UUID) (*projectdomain.ProjectManager, error)
	RemoveManager(ctx context.Context, projectID, userID uuid.UUID) error
}
