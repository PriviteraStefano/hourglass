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

type GovernanceModel string

const (
	GovernanceCreatorControlled GovernanceModel = "creator_controlled"
	GovernanceUnanimous         GovernanceModel = "unanimous"
	GovernanceMajority          GovernanceModel = "majority"
)

func (g GovernanceModel) IsValid() bool {
	switch g {
	case GovernanceCreatorControlled, GovernanceUnanimous, GovernanceMajority:
		return true
	default:
		return false
	}
}

type ProjectType string

const (
	ProjectTypeBillable ProjectType = "billable"
	ProjectTypeInternal ProjectType = "internal"
)

func (p ProjectType) IsValid() bool {
	switch p {
	case ProjectTypeBillable, ProjectTypeInternal:
		return true
	default:
		return false
	}
}

type Contract struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	KmRate          float64         `json:"km_rate"`
	Currency        string          `json:"currency"`
	GovernanceModel GovernanceModel `json:"governance_model"`
	CreatedByOrgID  uuid.UUID       `json:"created_by_org_id"`
	IsShared        bool            `json:"is_shared"`
	IsActive        bool            `json:"is_active"`
	CreatedAt       time.Time       `json:"created_at"`
}

type Project struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	Type            ProjectType     `json:"type"`
	ContractID      uuid.UUID       `json:"contract_id"`
	GovernanceModel GovernanceModel `json:"governance_model"`
	CreatedByOrgID  uuid.UUID       `json:"created_by_org_id"`
	IsShared        bool            `json:"is_shared"`
	IsActive        bool            `json:"is_active"`
	CreatedAt       time.Time       `json:"created_at"`
}

type ContractAdoption struct {
	ID             uuid.UUID `json:"id"`
	ContractID     uuid.UUID `json:"contract_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	AdoptedAt      time.Time `json:"adopted_at"`
}

type ProjectAdoption struct {
	ID             uuid.UUID `json:"id"`
	ProjectID      uuid.UUID `json:"project_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	AdoptedAt      time.Time `json:"adopted_at"`
}
