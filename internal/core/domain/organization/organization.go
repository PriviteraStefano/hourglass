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
	ID        uuid.UUID
	Name      string
	Slug      string
	CreatedAt time.Time
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
	ID          uuid.UUID
	UserID      *uuid.UUID
	Role        models.Role
	IsActive    bool
	InvitedBy   *uuid.UUID
	InvitedAt   *time.Time
	ActivatedAt *time.Time
	UserName    string
	UserEmail   string
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
