package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

type UserFinder struct {
	pool *pgxpool.Pool
}

func NewUserFinder(pool *pgxpool.Pool) *UserFinder {
	return &UserFinder{pool: pool}
}

func (f *UserFinder) FindByIdentifier(ctx context.Context, identifier string) (string, error) {
	const query = `SELECT id FROM users WHERE email = $1 OR username = $1 LIMIT 1`
	var id uuid.UUID
	err := f.pool.QueryRow(ctx, query, identifier).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ports.ErrUserNotFound
		}
		return "", fmt.Errorf("find user by identifier: %w", err)
	}
	return id.String(), nil
}
