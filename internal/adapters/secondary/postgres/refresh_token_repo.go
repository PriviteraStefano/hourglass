package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// RefreshTokenRepository implements ports.RefreshTokenRepository using a pgxpool.
type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

// Add inserts a new refresh token row.
func (r *RefreshTokenRepository) Add(ctx context.Context, userID, organizationID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	query := `INSERT INTO refresh_tokens (id, user_id, organization_id, token_hash, expires_at, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW())`
	_, err := r.pool.Exec(ctx, query, userID, organizationID, tokenHash, expiresAt)
	return wrapPGError(err, "add refresh token")
}

// FindByHash looks up a non-expired, non-revoked refresh token by its hash.
// Returns nil, nil when no matching token exists.
func (r *RefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*ports.RefreshToken, error) {
	query := `SELECT user_id, organization_id, token_hash, expires_at, created_at
		FROM refresh_tokens WHERE token_hash = $1 AND expires_at > NOW() AND revoked_at IS NULL
		LIMIT 1`
	row := r.pool.QueryRow(ctx, query, hash)

	var t ports.RefreshToken
	err := row.Scan(&t.UserID, &t.OrganizationID, &t.Hash, &t.ExpiresAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find refresh token by hash: %w", err)
	}
	return &t, nil
}

// RevokeByHash sets revoked_at on a non-revoked refresh token.
func (r *RefreshTokenRepository) RevokeByHash(ctx context.Context, hash string) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, query, hash)
	return wrapPGError(err, "revoke refresh token")
}

// RevokeAllByUser sets revoked_at on all non-revoked tokens for a user.
func (r *RefreshTokenRepository) RevokeAllByUser(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, query, userID)
	return wrapPGError(err, "revoke all refresh tokens")
}
