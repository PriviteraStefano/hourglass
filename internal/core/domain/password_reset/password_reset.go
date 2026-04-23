package password_reset

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrResetNotFound = errors.New("reset not found")
	ErrResetExpired  = errors.New("reset code expired or not found")
	ErrInvalidCode   = errors.New("invalid code")
)

type PasswordReset struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CodeHash  string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (r *PasswordReset) IsValid() bool {
	if r.UsedAt != nil {
		return false
	}
	return time.Now().Before(r.ExpiresAt)
}

type RequestResetRequest struct {
	Identifier string
}

type VerifyResetRequest struct {
	Identifier string
	Code       string
	Password   string
}
