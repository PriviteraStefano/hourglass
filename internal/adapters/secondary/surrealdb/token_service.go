package surrealdb

import (
	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

type TokenService struct {
	authService *auth.Service
}

func NewTokenService(authService *auth.Service) *TokenService {
	return &TokenService{authService: authService}
}

func (s *TokenService) GenerateToken(userID, organizationID uuid.UUID, role, email string) (string, error) {
	return s.authService.GenerateToken(userID, organizationID, role, email)
}

func (s *TokenService) ValidateToken(tokenString string) (*ports.Claims, error) {
	claims, err := s.authService.ValidateToken(tokenString)
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
	return s.authService.GenerateRefreshToken()
}

func (s *TokenService) HashRefreshToken(token string) string {
	return auth.HashRefreshToken(token)
}
