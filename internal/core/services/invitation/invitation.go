package invitation

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/invitation"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

type Service struct {
	repo ports.InvitationRepository
}

func NewService(repo ports.InvitationRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req *invitation.CreateInvitationRequest) (*invitation.Invitation, error) {
	code := generateInviteCode()
	token := generateToken()
	expiresInDays := req.ExpiresInDays
	if expiresInDays <= 0 {
		expiresInDays = 7
	}

	inv := &invitation.Invitation{
		ID:             uuid.New(),
		OrganizationID: req.OrganizationID,
		Code:           code,
		InviteToken:    token,
		Email:          req.Email,
		Status:         invitation.InvitationStatusPending,
		ExpiresAt:      time.Now().Add(time.Duration(expiresInDays) * 24 * time.Hour),
		CreatedBy:      "system",
		CreatedAt:      time.Now(),
	}

	created, err := s.repo.Create(ctx, inv)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) ValidateCode(ctx context.Context, code string) (*invitation.Invitation, error) {
	inv, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	if inv.IsExpired() {
		return nil, invitation.ErrInvitationExpired
	}

	return inv, nil
}

func (s *Service) ValidateToken(ctx context.Context, token string) (*invitation.Invitation, error) {
	inv, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if inv.IsExpired() {
		return nil, invitation.ErrInvitationExpired
	}

	return inv, nil
}

func (s *Service) Accept(ctx context.Context, token, email, username, password string) (*invitation.Invitation, error) {
	inv, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return nil, invitation.ErrInvitationNotFound
	}

	if !inv.IsUsable() {
		if inv.Status == invitation.InvitationStatusUsed {
			return nil, invitation.ErrInvitationUsed
		}
		return nil, invitation.ErrInvitationExpired
	}

	inv.Status = invitation.InvitationStatusUsed
	updated, err := s.repo.Update(ctx, inv)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func generateInviteCode() string {
	chars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 6)
	rand.Read(code)
	for i := range code {
		code[i] = chars[int(code[i])%len(chars)]
	}
	return string(code)
}

func generateToken() string {
	token := make([]byte, 16)
	rand.Read(token)
	return base64.URLEncoding.EncodeToString(token)
}
