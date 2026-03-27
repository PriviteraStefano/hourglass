package models

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleEmployee Role = "employee"
	RoleManager  Role = "manager"
	RoleFinance  Role = "finance"
	RoleCustomer Role = "customer"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleEmployee, RoleManager, RoleFinance, RoleCustomer:
		return true
	default:
		return false
	}
}

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

type OrganizationMembership struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	Role           Role       `json:"role"`
	IsActive       bool       `json:"is_active"`
	InvitedBy      *uuid.UUID `json:"invited_by,omitempty"`
	InvitedAt      *time.Time `json:"invited_at,omitempty"`
	ActivatedAt    *time.Time `json:"activated_at,omitempty"`
}

type UserWithMembership struct {
	User         User                   `json:"user"`
	Membership   OrganizationMembership `json:"membership"`
	Organization Organization           `json:"organization"`
}
