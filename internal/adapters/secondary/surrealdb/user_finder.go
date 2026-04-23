package surrealdb

import (
	"context"

	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	sdb "github.com/surrealdb/surrealdb.go"
)

type UserFinder struct {
	db *sdb.DB
}

func NewUserFinder(db *sdb.DB) *UserFinder {
	return &UserFinder{db: db}
}

func (r *UserFinder) FindByIdentifier(ctx context.Context, identifier string) (string, error) {
	results, err := sdb.Query[[]SurrealUser](ctx, r.db,
		"SELECT id FROM users WHERE email = $identifier OR username = $identifier LIMIT 1",
		map[string]interface{}{"identifier": identifier})
	if err != nil {
		return "", wrapErr(err, "find user by identifier")
	}
	if results == nil || len(*results) == 0 {
		return "", ports.ErrUserNotFound
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		return "", ports.ErrUserNotFound
	}
	return recordIDToUserID(&resultItems[0].ID), nil
}
