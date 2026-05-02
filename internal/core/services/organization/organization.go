package organization

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	orgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/organization"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

type Service struct {
	repo ports.OrganizationManagementRepository
}

func NewService(repo ports.OrganizationManagementRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req *orgdomain.CreateOrganizationRequest) (*orgdomain.Organization, error) {
	if req.Name == "" {
		return nil, orgdomain.ErrInvalidRequest
	}
	slug := req.Slug
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	}
	now := time.Now()
	org := &orgdomain.Organization{
		ID:        uuid.New(),
		Name:      req.Name,
		Slug:      slug,
		CreatedAt: now,
	}
	if err := s.repo.CreateOrganization(ctx, org, userID, models.RoleFinance); err != nil {
		return nil, err
	}
	return org, nil
}

func (s *Service) Get(ctx context.Context, orgID uuid.UUID) (*orgdomain.Organization, error) {
	return s.repo.GetOrganization(ctx, orgID)
}

func (s *Service) Invite(ctx context.Context, orgID, userID uuid.UUID, req *orgdomain.InviteRequest) (uuid.UUID, time.Time, error) {
	if req.Email == "" || !req.Role.IsValid() {
		return uuid.Nil, time.Time{}, orgdomain.ErrInvalidRequest
	}
	return s.repo.InviteMember(ctx, orgID, req, userID)
}

func (s *Service) GetSettings(ctx context.Context, orgID uuid.UUID) (*orgdomain.Settings, error) {
	return s.repo.GetSettings(ctx, orgID)
}

func (s *Service) UpdateSettings(ctx context.Context, role string, orgID uuid.UUID, req *orgdomain.UpdateSettingsRequest) (*orgdomain.Settings, error) {
	if role != string(models.RoleFinance) {
		return nil, orgdomain.ErrForbidden
	}
	return s.repo.UpdateSettings(ctx, orgID, req)
}

func (s *Service) ListMembers(ctx context.Context, orgID uuid.UUID) ([]orgdomain.Member, error) {
	return s.repo.ListMembers(ctx, orgID)
}

func (s *Service) UpdateMemberRoles(ctx context.Context, actorRole string, orgID, memberID uuid.UUID, roles []string) error {
	if actorRole != string(models.RoleFinance) {
		return orgdomain.ErrForbidden
	}
	if len(roles) == 0 {
		return orgdomain.ErrInvalidRequest
	}
	newRole := models.Role(roles[0])
	if !newRole.IsValid() {
		return orgdomain.ErrInvalidRequest
	}
	return s.repo.UpdateMemberRole(ctx, orgID, memberID, newRole)
}

func (s *Service) DeactivateMember(ctx context.Context, actorRole string, orgID, memberID uuid.UUID) error {
	if actorRole != string(models.RoleFinance) {
		return orgdomain.ErrForbidden
	}
	memberRole, err := s.repo.GetMemberRole(ctx, memberID)
	if err != nil {
		return err
	}
	if memberRole == models.RoleFinance {
		financeCount, err := s.repo.CountActiveFinance(ctx, orgID)
		if err != nil {
			return err
		}
		if financeCount <= 1 {
			return orgdomain.ErrLastFinance
		}
	}
	return s.repo.DeactivateMember(ctx, orgID, memberID)
}
