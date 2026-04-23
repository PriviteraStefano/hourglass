package ports

import (
	"context"
)

type UserFinder interface {
	FindByIdentifier(ctx context.Context, identifier string) (userID string, err error)
}
