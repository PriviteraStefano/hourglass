package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
)

type OrganizationRepository interface {
	Add(ctx context.Context, org *auth.Organization) error
	GetByID(ctx context.Context, id uuid.UUID) (*auth.Organization, error)
	GetMembership(ctx context.Context, userID, orgID uuid.UUID) (*auth.OrganizationMembership, error)
	AddMembership(ctx context.Context, membership *auth.OrganizationMembership) error
}
