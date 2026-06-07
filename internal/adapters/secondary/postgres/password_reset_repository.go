package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/password_reset"
)

// PasswordResetRepository implements ports.PasswordResetRepository using a pgxpool.
type PasswordResetRepository struct {
	pool *pgxpool.Pool
}

func NewPasswordResetRepository(pool *pgxpool.Pool) *PasswordResetRepository {
	return &PasswordResetRepository{pool: pool}
}

// Create inserts a new password reset row and returns the full record.
func (r *PasswordResetRepository) Create(ctx context.Context, pr *password_reset.PasswordReset) (*password_reset.PasswordReset, error) {
	query := `INSERT INTO password_resets (id, user_id, code_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, user_id, code_hash, expires_at, used_at, created_at`
	return scanPasswordReset(r.pool.QueryRow(ctx, query, pr.ID, pr.UserID, pr.CodeHash, pr.ExpiresAt))
}

// FindActiveByUserID returns the latest active (non-expired, unused) reset for the user.
// Returns password_reset.ErrResetNotFound when no active reset exists.
func (r *PasswordResetRepository) FindActiveByUserID(ctx context.Context, userID string) (*password_reset.PasswordReset, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	query := `SELECT id, user_id, code_hash, expires_at, used_at, created_at
		FROM password_resets WHERE user_id = $1 AND expires_at > NOW() AND used_at IS NULL
		ORDER BY created_at DESC LIMIT 1`
	pr, err := scanPasswordReset(r.pool.QueryRow(ctx, query, uid))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, password_reset.ErrResetNotFound
		}
		return nil, fmt.Errorf("find active password reset: %w", err)
	}
	return pr, nil
}

// MarkUsed sets used_at to NOW() for the given reset id.
func (r *PasswordResetRepository) MarkUsed(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("parse reset id: %w", err)
	}
	query := `UPDATE password_resets SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`
	_, err = r.pool.Exec(ctx, query, uid)
	return wrapPGError(err, "mark password reset used")
}

// UpdateUserPassword sets the password_hash for the given user.
func (r *PasswordResetRepository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, passwordHash, userID)
	return wrapPGError(err, "update user password")
}

// prScanner is satisfied by both pgx.Row and pgx.Rows.
type prScanner interface {
	Scan(dest ...any) error
}

// scanPasswordReset scans a single password_reset row, handling nullable used_at.
func scanPasswordReset(s prScanner) (*password_reset.PasswordReset, error) {
	var pr password_reset.PasswordReset
	err := s.Scan(&pr.ID, &pr.UserID, &pr.CodeHash, &pr.ExpiresAt, &pr.UsedAt, &pr.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &pr, nil
}
