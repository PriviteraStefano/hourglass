package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
)

type WorkingGroupRepository interface {
	ListByOrg(ctx context.Context, orgID uuid.UUID, subprojectID *uuid.UUID) ([]working_group.WorkingGroup, error)
	GetByID(ctx context.Context, id uuid.UUID) (*working_group.WorkingGroup, error)
	Create(ctx context.Context, wg *working_group.WorkingGroup) (*working_group.WorkingGroup, error)
	Update(ctx context.Context, wg *working_group.WorkingGroup) (*working_group.WorkingGroup, error)
	Delete(ctx context.Context, id uuid.UUID) error
	HasMembers(ctx context.Context, id uuid.UUID) (bool, error)
	ListMembers(ctx context.Context, wgID uuid.UUID) ([]working_group.WorkingGroupMember, error)
	AddMember(ctx context.Context, m *working_group.WorkingGroupMember) (*working_group.WorkingGroupMember, error)
	RemoveMember(ctx context.Context, id uuid.UUID) error
}
