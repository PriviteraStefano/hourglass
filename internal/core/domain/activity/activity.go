package activity

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

var (
	ErrActivityNotFound     = errors.New("activity not found")
	ErrForbidden            = errors.New("forbidden")
	ErrInvalidRequest       = errors.New("invalid request")
	ErrAlreadyAdopted       = errors.New("already adopted")
	ErrUserNotFound         = errors.New("user not found")
	ErrHasChildren          = errors.New("activity has children")
	ErrHasActiveTimeEntries = errors.New("activity has active time entries")
	ErrHasActiveExpenses    = errors.New("activity has active expenses")
)

// ActivityKind is a free label from the org-level activity_kinds catalog (D-2).
// It is intentionally NOT an enum: orgs extend the catalog with their own
// kinds, and kind carries no level/ordering semantics.
type ActivityKind string

// Activity is the single recursive work entity (ADR-P-007 D-1/D-2/D-3/D-7).
// Projects and subprojects collapsed into this one type.
type Activity struct {
	ID              uuid.UUID              `json:"id"`
	OrgID           uuid.UUID              `json:"org_id"`
	ParentID        *uuid.UUID             `json:"parent_id,omitempty"` // D-2: nullable, no level meaning
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Kind            ActivityKind           `json:"kind"`          // catalog label, not an enum
	ContractID      *uuid.UUID             `json:"contract_id,omitempty"` // D-3: nullable = internal work
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	CreatedByOrgID  uuid.UUID              `json:"created_by_org_id"`
	IsShared        bool                   `json:"is_shared"`
	Billable        *bool                  `json:"billable,omitempty"` // D-7: nil = inherit
	BudgetAmount    *float64               `json:"budget_amount,omitempty"`
	IsActive        bool                   `json:"is_active"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// ActivityResponse decorates an Activity with joined display fields.
type ActivityResponse struct {
	Activity
	ParentName       string `json:"parent_name,omitempty"`
	ContractName     string `json:"contract_name"`
	CreatedByOrgName string `json:"created_by_org_name"`
	AdoptionCount    int    `json:"adoption_count"`
	IsAdopted        bool   `json:"is_adopted"`
}

// ActivityAdoption mirrors project_adoptions (sharing preserved, D-6).
type ActivityAdoption struct {
	ID             uuid.UUID `json:"id"`
	ActivityID     uuid.UUID `json:"activity_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	AdoptedAt      time.Time `json:"adopted_at"`
}

// ActivityManager mirrors project_managers, renamed per ADR-P-007 D-1 note.
type ActivityManager struct {
	ID         uuid.UUID `json:"id"`
	ActivityID uuid.UUID `json:"activity_id"`
	UserID     uuid.UUID `json:"user_id"`
	UserName   string    `json:"user_name"`
	Email      string    `json:"email"`
	CreatedAt  time.Time `json:"created_at"`
}

// CommercialContext is the derived-not-stored commercial chain (D-3):
// the nearest ancestor activity carrying a contract, resolved by walking
// parent_id upward. Nil ContractID means a purely internal tree.
type CommercialContext struct {
	ContractID *uuid.UUID `json:"contract_id,omitempty"`
	CustomerID *uuid.UUID `json:"customer_id,omitempty"`
}

// CreateActivityRequest is the DTO for creating an activity.
type CreateActivityRequest struct {
	ParentID        *uuid.UUID              `json:"parent_id,omitempty"`
	Name            string                  `json:"name"`
	Description     string                  `json:"description"`
	Kind            ActivityKind            `json:"kind"`
	ContractID      *uuid.UUID              `json:"contract_id,omitempty"`
	GovernanceModel models.GovernanceModel  `json:"governance_model"`
	IsShared        bool                    `json:"is_shared"`
	Billable        *bool                   `json:"billable,omitempty"`
	BudgetAmount    *float64                `json:"budget_amount,omitempty"`
}

// UpdateActivityRequest is the DTO for updating an activity.
type UpdateActivityRequest struct {
	ParentID        *uuid.UUID              `json:"parent_id,omitempty"`
	Name            string                  `json:"name,omitempty"`
	Description     string                  `json:"description,omitempty"`
	Kind            ActivityKind            `json:"kind,omitempty"`
	ContractID      *uuid.UUID              `json:"contract_id,omitempty"`
	GovernanceModel models.GovernanceModel  `json:"governance_model,omitempty"`
	IsShared        *bool                   `json:"is_shared,omitempty"`
	Billable        *bool                   `json:"billable,omitempty"`
	BudgetAmount    *float64                `json:"budget_amount,omitempty"`
	IsActive        *bool                   `json:"is_active,omitempty"`
}

// ActivityFilter filters the List query.
type ActivityFilter struct {
	Scope      string // "adopted" | "all" | "own" (default)
	ContractID string
	ParentID   string
	Kind       string
	IsActive   *bool
}
