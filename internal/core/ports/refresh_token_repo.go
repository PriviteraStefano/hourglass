package ports

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrTokenReuse is returned by Rotate when a refresh token that was already
// rotated or revoked is presented again. The entire token family has been
// revoked at this point.
var ErrTokenReuse = errors.New("refresh token reuse detected")

type RefreshToken struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Hash           string
	ExpiresAt      time.Time
	CreatedAt      time.Time
	FamilyID       uuid.UUID
	RotatedAt      *time.Time
	RevokedAt      *time.Time
}

type RefreshTokenRepository interface {
	Add(ctx context.Context, userID, organizationID uuid.UUID, tokenHash string, expiresAt time.Time) error
	FindByHash(ctx context.Context, hash string) (*RefreshToken, error)
	RevokeByHash(ctx context.Context, hash string) error
	RevokeAllByUser(ctx context.Context, userID uuid.UUID) error
	// Rotate atomically validates a refresh token and issues its successor
	// within a single transaction:
	//   - unknown hash          -> (nil, nil)
	//   - already rotated/revoked -> revokes the whole family, returns the
	//     offending token and ErrTokenReuse
	//   - valid                  -> marks rotated_at, inserts the successor
	//     with the same family_id, returns the consumed token
	Rotate(ctx context.Context, hash, newHash string, newExpiresAt time.Time) (*RefreshToken, error)
	// RevokeFamily tombstones every non-revoked token that shares familyID.
	RevokeFamily(ctx context.Context, familyID uuid.UUID) error
}
