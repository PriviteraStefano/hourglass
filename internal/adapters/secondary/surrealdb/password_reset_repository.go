package surrealdb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/password_reset"
	sdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type PasswordResetRepository struct {
	db *sdb.DB
}

func NewPasswordResetRepository(db *sdb.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

func (r *PasswordResetRepository) Create(ctx context.Context, pr *password_reset.PasswordReset) (*password_reset.PasswordReset, error) {
	spr := surrealPasswordResetFromDomain(pr)
	created, err := sdb.Create[SurrealPasswordReset](ctx, r.db, models.Table("password_resets"), spr)
	if err != nil {
		return nil, wrapErr(err, "create password reset")
	}
	return created.ToDomain(), nil
}

func (r *PasswordResetRepository) FindActiveByUserID(ctx context.Context, userID string) (*password_reset.PasswordReset, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, password_reset.ErrResetNotFound
	}
	recordID := uuidToRecordID("users", uid)
	results, err := sdb.Query[[]SurrealPasswordReset](ctx, r.db,
		`SELECT * FROM password_resets WHERE user_id = $user_id AND expires_at > $now AND used_at = NONE LIMIT 1`,
		map[string]interface{}{
			"user_id": recordID,
			"now":     time.Now(),
		})
	if err != nil {
		return nil, wrapErr(err, "find active password reset")
	}
	if results == nil || len(*results) == 0 {
		return nil, password_reset.ErrResetNotFound
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		return nil, password_reset.ErrResetNotFound
	}
	return resultItems[0].ToDomain(), nil
}

func (r *PasswordResetRepository) MarkUsed(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	recordID := uuidToRecordID("password_resets", uid)
	_, err = sdb.Merge[map[string]interface{}](ctx, r.db, recordID, map[string]interface{}{
		"used_at": time.Now(),
	})
	return wrapErr(err, "mark password reset used")
}

func (r *PasswordResetRepository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	recordID := uuidToRecordID("users", userID)
	_, err := sdb.Merge[map[string]interface{}](ctx, r.db, recordID, map[string]interface{}{
		"password_hash": passwordHash,
	})
	return wrapErr(err, "update user password")
}

func surrealPasswordResetFromDomain(pr *password_reset.PasswordReset) *SurrealPasswordReset {
	if pr == nil {
		return nil
	}
	return &SurrealPasswordReset{
		ID:        uuidToRecordID("password_resets", pr.ID),
		UserID:    uuidToRecordID("users", pr.UserID),
		CodeHash:  pr.CodeHash,
		ExpiresAt: pr.ExpiresAt,
		UsedAt:    pr.UsedAt,
		CreatedAt: pr.CreatedAt,
	}
}
