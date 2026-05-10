package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
)

type UnitRepository interface {
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]unit.Unit, error)
	GetByID(ctx context.Context, id string) (*unit.Unit, error)
	Create(ctx context.Context, u *unit.Unit) (*unit.Unit, error)
	Update(ctx context.Context, u *unit.Unit) (*unit.Unit, error)
	Delete(ctx context.Context, id string) error
	GetDescendants(ctx context.Context, id string) ([]unit.Unit, error)
	HasMembers(ctx context.Context, id string) (bool, error)
	ListMembers(ctx context.Context, unitID string) ([]unit.UnitMember, error)
	AddMember(ctx context.Context, m *unit.UnitMember) (*unit.UnitMember, error)
	RemoveMember(ctx context.Context, id string) error
	GetMemberCountsByOrg(ctx context.Context, orgID uuid.UUID) (map[string]int, error)
}
