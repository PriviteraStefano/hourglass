package ports

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	Add(ctx context.Context, user *auth.User) error
	GetByEmail(ctx context.Context, email string) (*auth.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*auth.User, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
}
