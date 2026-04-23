package surrealdb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	sdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type RefreshTokenRepository struct {
	db *sdb.DB
}

func NewRefreshTokenRepository(db *sdb.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Add(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	token := SurrealRefreshToken{
		UserID:    uuidToRecordID("users", userID),
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	_, err := sdb.Create[SurrealRefreshToken](ctx, r.db, models.Table("refresh_tokens"), token)
	if err != nil {
		return wrapErr(err, "create refresh token")
	}
	return nil
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*ports.RefreshToken, error) {
	results, err := sdb.Query[[]SurrealRefreshToken](ctx, r.db,
		"SELECT * FROM refresh_tokens WHERE token_hash = $token_hash AND expires_at > $now AND revoked_at = NONE LIMIT 1",
		map[string]any{"token_hash": hash, "now": time.Now()})
	if err != nil {
		return nil, wrapErr(err, "find refresh token by hash")
	}
	if results == nil || len(*results) == 0 {
		return nil, nil
	}
	resultData := *results
	if len(resultData) == 0 {
		return nil, nil
	}
	resultItems := resultData[0].Result
	if len(resultItems) == 0 {
		return nil, nil
	}
	token := resultItems[0]
	return &ports.RefreshToken{
		UserID:    recordIDToUUID(token.UserID),
		Hash:      token.TokenHash,
		ExpiresAt: token.ExpiresAt,
		CreatedAt: token.CreatedAt,
	}, nil
}

func (r *RefreshTokenRepository) RevokeByHash(ctx context.Context, hash string) error {
	_, err := sdb.Query[any](ctx, r.db,
		"UPDATE refresh_tokens SET revoked_at = $now WHERE token_hash = $token_hash",
		map[string]any{"token_hash": hash, "now": time.Now()})
	if err != nil {
		return wrapErr(err, "revoke refresh token")
	}
	return nil
}
