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

const refreshTokenColumns = "user_id, organization_id, token_hash, expires_at, created_at, family_id, rotated_at, revoked_at"

// Add inserts a new refresh token row belonging to a fresh token family.
func (r *RefreshTokenRepository) Add(ctx context.Context, userID, organizationID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	query := `INSERT INTO refresh_tokens (id, user_id, organization_id, token_hash, expires_at, family_id, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, gen_random_uuid(), NOW())`
	_, err := r.pool.Exec(ctx, query, userID, organizationID, tokenHash, expiresAt)
	return wrapPGError(err, "add refresh token")
}

// FindByHash looks up a currently-valid (non-expired, non-rotated,
// non-revoked) refresh token by its hash. Returns nil, nil when no matching
// token exists.
func (r *RefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*ports.RefreshToken, error) {
	query := `SELECT ` + refreshTokenColumns + `
		FROM refresh_tokens WHERE token_hash = $1 AND expires_at > NOW()
		AND rotated_at IS NULL AND revoked_at IS NULL
		LIMIT 1`
	row := r.pool.QueryRow(ctx, query, hash)

	var t ports.RefreshToken
	err := row.Scan(&t.UserID, &t.OrganizationID, &t.Hash, &t.ExpiresAt, &t.CreatedAt, &t.FamilyID, &t.RotatedAt, &t.RevokedAt)
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

// RevokeFamily sets revoked_at on every non-revoked token sharing familyID.
func (r *RefreshTokenRepository) RevokeFamily(ctx context.Context, familyID uuid.UUID) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE family_id = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, query, familyID)
	return wrapPGError(err, "revoke refresh token family")
}

// Rotate atomically validates a refresh token and issues its successor in a
// single transaction:
//
//   - unknown hash: returns (nil, nil)
//   - already rotated/revoked (replay): revokes the entire family and
//     returns the offending token with ports.ErrTokenReuse — the successor
//     issued by the original rotation dies with its family
//   - valid: marks rotated_at on the old row, inserts the successor with the
//     same family_id, returns the consumed token
//
// There is no window where the old token is consumed but no successor exists:
// both statements commit together or not at all.
func (r *RefreshTokenRepository) Rotate(ctx context.Context, hash, newHash string, newExpiresAt time.Time) (*ports.RefreshToken, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin refresh token rotation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var t ports.RefreshToken
	err = tx.QueryRow(ctx,
		`SELECT `+refreshTokenColumns+` FROM refresh_tokens
		 WHERE token_hash = $1 AND expires_at > NOW() FOR UPDATE`, hash,
	).Scan(&t.UserID, &t.OrganizationID, &t.Hash, &t.ExpiresAt, &t.CreatedAt, &t.FamilyID, &t.RotatedAt, &t.RevokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find refresh token for rotation: %w", err)
	}

	// Replay of a rotated or revoked token: kill the whole family.
	if t.RotatedAt != nil || t.RevokedAt != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE refresh_tokens SET revoked_at = NOW() WHERE family_id = $1 AND revoked_at IS NULL`, t.FamilyID,
		); err != nil {
			return nil, wrapPGError(err, "revoke refresh token family")
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit refresh token family revocation: %w", err)
		}
		return &t, ports.ErrTokenReuse
	}

	// Legitimate rotation: mark the old token rotated and insert its successor.
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx,
		`UPDATE refresh_tokens SET rotated_at = $1 WHERE token_hash = $2 AND revoked_at IS NULL`, now, hash,
	); err != nil {
		return nil, wrapPGError(err, "mark refresh token rotated")
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, organization_id, token_hash, expires_at, family_id, created_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)`,
		t.UserID, t.OrganizationID, newHash, newExpiresAt, t.FamilyID, now,
	); err != nil {
		return nil, wrapPGError(err, "insert rotated refresh token")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit refresh token rotation: %w", err)
	}
	return &t, nil
}
