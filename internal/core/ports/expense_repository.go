package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

type ExpenseRepository interface {
	Create(ctx context.Context, e *models.Expense) (*models.Expense, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Expense, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]models.Expense, error)
	Update(ctx context.Context, e *models.Expense) (*models.Expense, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
