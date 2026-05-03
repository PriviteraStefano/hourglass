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
	AddWithMembership(ctx context.Context, user *auth.User, membership *auth.OrganizationMembership) error
	AddWithOrgAndMembership(ctx context.Context, user *auth.User, org *auth.Organization, membership *auth.OrganizationMembership) error
	GetByEmail(ctx context.Context, email string) (*auth.User, error)
	GetByUsername(ctx context.Context, username string) (*auth.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*auth.User, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
	AnyExists(ctx context.Context) (bool, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	GetMemberships(ctx context.Context, userID uuid.UUID) ([]auth.OrganizationMembership, error)
}
