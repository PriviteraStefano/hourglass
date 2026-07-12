package organization

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

var (
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrMemberNotFound       = errors.New("member not found")
	ErrForbidden            = errors.New("forbidden")
	ErrInvalidRequest       = errors.New("invalid request")
	ErrLastFinance          = errors.New("cannot deactivate last finance member")
)

type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

type Settings struct {
	OrganizationID      uuid.UUID
	DefaultKmRate       *float64
	Currency            string
	WeekStartDay        int
	Timezone            string
	ShowApprovalHistory bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type Member struct {
	ID          uuid.UUID    `json:"id"`
	UserID      *uuid.UUID   `json:"user_id"`
	Role        models.Role  `json:"role"`
	IsActive    bool         `json:"is_active"`
	InvitedBy   *uuid.UUID   `json:"invited_by,omitempty"`
	InvitedAt   *time.Time   `json:"invited_at,omitempty"`
	ActivatedAt *time.Time   `json:"activated_at,omitempty"`
	UserName    string       `json:"user_name"`
	UserEmail   string       `json:"user_email"`
}

type CreateOrganizationRequest struct {
	Name string
	Slug string
}

type InviteRequest struct {
	Email string
	Role  models.Role
}

type UpdateSettingsRequest struct {
	DefaultKmRate       *float64
	Currency            string
	WeekStartDay        *int
	Timezone            string
	ShowApprovalHistory *bool
}
