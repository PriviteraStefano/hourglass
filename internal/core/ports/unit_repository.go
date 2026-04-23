package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
)

type UnitRepository interface {
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]unit.Unit, error)
	GetByID(ctx context.Context, id uuid.UUID) (*unit.Unit, error)
	Create(ctx context.Context, u *unit.Unit) (*unit.Unit, error)
	Update(ctx context.Context, u *unit.Unit) (*unit.Unit, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetDescendants(ctx context.Context, id uuid.UUID) ([]unit.Unit, error)
	HasMembers(ctx context.Context, id uuid.UUID) (bool, error)
}
