package ports

import (
	"time"

	"github.com/google/uuid"
)

type TokenService interface {
	GenerateToken(userID, organizationID uuid.UUID, role, email string) (string, error)
	ValidateToken(tokenString string) (*Claims, error)
	GenerateRefreshToken() (string, error)
	HashRefreshToken(token string) string
}

type Claims struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Role           string
	Email          string
	ExpiresAt      time.Time
}
