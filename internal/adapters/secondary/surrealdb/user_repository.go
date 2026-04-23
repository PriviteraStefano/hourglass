package surrealdb

import (
	"context"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	sdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type UserRepository struct {
	db *sdb.DB
}

func NewUserRepository(db *sdb.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Add(ctx context.Context, user *auth.User) error {
	su := SurrealUserFromDomain(user)
	_, err := sdb.Create[SurrealUser](ctx, r.db, models.Table("users"), su)
	if err != nil {
		return wrapErr(err, "create user")
	}
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*auth.User, error) {
	results, err := sdb.Query[[]SurrealUser](ctx, r.db,
		"SELECT * FROM users WHERE email = $email LIMIT 1",
		map[string]any{"email": email})
	if err != nil {
		return nil, wrapErr(err, "get user by email")
	}
	if results == nil || len(*results) == 0 {
		return nil, ports.ErrUserNotFound
	}
	resultData := *results
	if len(resultData) == 0 {
		return nil, ports.ErrUserNotFound
	}
	resultItems := resultData[0].Result
	if len(resultItems) == 0 {
		return nil, ports.ErrUserNotFound
	}
	return resultItems[0].ToDomain(), nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	recordID := uuidToRecordID("users", id)
	result, err := sdb.Select[SurrealUser](ctx, r.db, recordID)
	if err != nil {
		return nil, wrapErr(err, "get user by id")
	}
	return result.ToDomain(), nil
}

func (r *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	results, err := sdb.Query[[]SurrealUserCount](ctx, r.db,
		"SELECT count() FROM users WHERE email = $email GROUP ALL",
		map[string]any{"email": email})
	if err != nil {
		return false, wrapErr(err, "check email exists")
	}
	if results == nil || len(*results) == 0 {
		return false, nil
	}
	resultData := *results
	if len(resultData) == 0 {
		return false, nil
	}
	resultItems := resultData[0].Result
	if len(resultItems) == 0 {
		return false, nil
	}
	return resultItems[0].Count > 0, nil
}

func (r *UserRepository) UsernameExists(ctx context.Context, username string) (bool, error) {
	result, err := sdb.Query[[]SurrealUser](ctx, r.db,
		"SELECT id FROM users WHERE username = $username LIMIT 1",
		map[string]interface{}{"username": username})
	if err != nil {
		return false, wrapErr(err, "check username exists")
	}
	if result == nil || len(*result) == 0 {
		return false, nil
	}
	return len((*result)[0].Result) > 0, nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	recordID := uuidToRecordID("users", userID)
	_, err := sdb.Merge[map[string]interface{}](ctx, r.db, recordID, map[string]interface{}{
		"password_hash": passwordHash,
	})
	return wrapErr(err, "update password")
}
