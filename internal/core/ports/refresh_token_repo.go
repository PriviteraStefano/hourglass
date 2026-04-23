package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	UserID    uuid.UUID
	Hash      string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type RefreshTokenRepository interface {
	Add(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	FindByHash(ctx context.Context, hash string) (*RefreshToken, error)
	RevokeByHash(ctx context.Context, hash string) error
}
