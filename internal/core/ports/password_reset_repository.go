package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/password_reset"
)

type PasswordResetRepository interface {
	Create(ctx context.Context, pr *password_reset.PasswordReset) (*password_reset.PasswordReset, error)
	FindActiveByUserID(ctx context.Context, userID string) (*password_reset.PasswordReset, error)
	MarkUsed(ctx context.Context, id string) error
	UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
}
