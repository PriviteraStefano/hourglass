package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	orgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/organization"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

type OrganizationManagementRepository interface {
	CreateOrganization(ctx context.Context, org *orgdomain.Organization, ownerUserID uuid.UUID, ownerRole models.Role) error
	GetOrganization(ctx context.Context, id uuid.UUID) (*orgdomain.Organization, error)

	InviteMember(ctx context.Context, orgID uuid.UUID, req *orgdomain.InviteRequest, invitedBy uuid.UUID) (uuid.UUID, time.Time, error)
	GetSettings(ctx context.Context, orgID uuid.UUID) (*orgdomain.Settings, error)
	UpdateSettings(ctx context.Context, orgID uuid.UUID, req *orgdomain.UpdateSettingsRequest) (*orgdomain.Settings, error)

	ListMembers(ctx context.Context, orgID uuid.UUID) ([]orgdomain.Member, error)
	UpdateMemberRole(ctx context.Context, orgID, memberID uuid.UUID, role models.Role) error
	DeactivateMember(ctx context.Context, orgID, memberID uuid.UUID) error
	CountActiveFinance(ctx context.Context, orgID uuid.UUID) (int, error)
	GetMemberRole(ctx context.Context, memberID uuid.UUID) (models.Role, error)
}
