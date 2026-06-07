package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

type SubprojectRepository interface {
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.Subproject, error)
	GetByID(ctx context.Context, id string) (*models.Subproject, error)
	Create(ctx context.Context, sp *models.Subproject) (*models.Subproject, error)
	Update(ctx context.Context, sp *models.Subproject) (*models.Subproject, error)
	Delete(ctx context.Context, id string) error
}
