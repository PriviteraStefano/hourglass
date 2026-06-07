package auth

import (
	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// TokenService wraps auth.Service to implement ports.TokenService.
type TokenService struct {
	svc *Service
}

func NewTokenService(svc *Service) *TokenService {
	return &TokenService{svc: svc}
}

func (s *TokenService) GenerateToken(userID, organizationID uuid.UUID, role, email string) (string, error) {
	return s.svc.GenerateToken(userID, organizationID, role, email)
}

func (s *TokenService) ValidateToken(tokenString string) (*ports.Claims, error) {
	claims, err := s.svc.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	return &ports.Claims{
		UserID:         claims.UserID,
		OrganizationID: claims.OrganizationID,
		Role:           claims.Role,
		Email:          claims.Email,
		ExpiresAt:      claims.ExpiresAt.Time,
	}, nil
}

func (s *TokenService) GenerateRefreshToken() (string, error) {
	return s.svc.GenerateRefreshToken()
}

func (s *TokenService) HashRefreshToken(token string) string {
	return HashRefreshToken(token)
}
