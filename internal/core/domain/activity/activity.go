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
	// ErrActivityNotLoggable is returned at submission time when an entry's
	// activity is commercial (has a contract via the derived chain, D-3) but
	// anchors no working group. Per ADR-BE-014 R-2, commercial activities must
	// anchor a WG before accepting entries; only personal activities (no
	// contract, no WG — D-8) fall back to the unit-manager stage.
	ErrActivityNotLoggable = errors.New("activity not loggable: commercial activities must anchor a working group before accepting entries")
	// ErrActivityCycle rejects a parent assignment that would make the activity
	// its own ancestor — the SPEC in-scope item "Cycle prevention on
	// activities.parent_id (path check on insert/update)". The service walks
	// the repository's GetAncestry of the proposed parent and rejects when the
	// chain contains the activity's own id (ADR-BE-001 sentinel pattern).
	ErrActivityCycle = errors.New("activity parent would create a cycle")
	// ErrOriginImmutable rejects any update request carrying origin fields
	// (D-03): the origin discriminator and its reference set are fixed at
	// creation and never mutate (T-11-10).
	ErrOriginImmutable = errors.New("origin refs are immutable after creation")
)

// Origin type vocabulary (ADR-P-013, D-D) — mirrors the DB CHECK on
// activities.origin_type:
//   - manager_assignment → assigned_by/assigned_to (D-01)
//   - employee_proposal → proposed_by (reviewed_by stays NULL — OQ1; the
//     approver lives in the proposal_approved audit row, D-12)
//   - customer_ticket → ticket_id (D-02)
const (
	OriginTypeManagerAssignment = "manager_assignment"
	OriginTypeEmployeeProposal  = "employee_proposal"
	OriginTypeCustomerTicket    = "customer_ticket"
)

// ActivityKind is a free label from the org-level activity_kinds catalog (D-2).
// It is intentionally NOT an enum: orgs extend the catalog with their own
// kinds, and kind carries no level/ordering semantics.
type ActivityKind string

// Activity is the single recursive work entity (ADR-P-007 D-1/D-2/D-3/D-7).
// Projects and subprojects collapsed into this one type.
type Activity struct {
	ID          uuid.UUID    `json:"id"`
	OrgID       uuid.UUID    `json:"org_id"`
	ParentID    *uuid.UUID   `json:"parent_id,omitempty"` // D-2: nullable, no level meaning
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Kind        ActivityKind `json:"kind"`                  // catalog label, not an enum
	ContractID  *uuid.UUID   `json:"contract_id,omitempty"` // D-3: nullable = internal work
	// BeneficiaryUnitID is the unit the work benefits (COV-05). Nullable and
	// inherited downward like contract_id (D-3) — absorption funding sources
	// default from the nearest ancestor carrying it. Unlike origin refs it
	// stays EDITABLE on Update.
	BeneficiaryUnitID *uuid.UUID             `json:"beneficiary_unit_id,omitempty"`
	GovernanceModel   models.GovernanceModel `json:"governance_model"`
	CreatedByOrgID    uuid.UUID              `json:"created_by_org_id"`
	IsShared          bool                   `json:"is_shared"`
	Billable          *bool                  `json:"billable,omitempty"` // D-7: nil = inherit
	BudgetAmount      *float64               `json:"budget_amount,omitempty"`
	IsActive          bool                   `json:"is_active"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`

	// Origin axis (ADR-P-013, D-D): the discriminator + reference set, set
	// once at creation and immutable afterwards (D-03). Null on legacy rows.
	OriginType *string    `json:"origin_type,omitempty"`
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty"`
	AssignedTo *uuid.UUID `json:"assigned_to,omitempty"`
	ProposedBy *uuid.UUID `json:"proposed_by,omitempty"`
	ReviewedBy *uuid.UUID `json:"reviewed_by,omitempty"` // stays NULL at create (OQ1)
	TicketID   *uuid.UUID `json:"ticket_id,omitempty"`
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

// FundingContext is the derived-not-stored funding chain (D-04): the nearest
// ancestor contract plus its funding attributes — contract_type and
// sold_hours via the contracts JOIN (016). The coverage service consumes it
// as the decision input: contract draw vs support bucket vs service request
// (zero-value contract). All fields nil when the chain has no contract.
type FundingContext struct {
	ContractID   *uuid.UUID `json:"contract_id,omitempty"`
	ContractType *string    `json:"contract_type,omitempty"`
	SoldHours    *float64   `json:"sold_hours,omitempty"`
}

// CreateActivityRequest is the DTO for creating an activity.
type CreateActivityRequest struct {
	ParentID          *uuid.UUID             `json:"parent_id,omitempty"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Kind              ActivityKind           `json:"kind"`
	ContractID        *uuid.UUID             `json:"contract_id,omitempty"`
	BeneficiaryUnitID *uuid.UUID             `json:"beneficiary_unit_id,omitempty"` // COV-05: nullable, editable
	GovernanceModel   models.GovernanceModel `json:"governance_model"`
	IsShared          bool                   `json:"is_shared"`
	Billable          *bool                  `json:"billable,omitempty"`
	BudgetAmount      *float64               `json:"budget_amount,omitempty"`
	IsActive          *bool                  `json:"is_active,omitempty"` // nil → true; employee proposals are forced false (D-12)

	// Origin axis (ADR-P-013): validated per type in the service. Set-once.
	OriginType *string    `json:"origin_type,omitempty"`
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty"`
	AssignedTo *uuid.UUID `json:"assigned_to,omitempty"`
	ProposedBy *uuid.UUID `json:"proposed_by,omitempty"`
	ReviewedBy *uuid.UUID `json:"reviewed_by,omitempty"`
	TicketID   *uuid.UUID `json:"ticket_id,omitempty"`
}

// UpdateActivityRequest is the DTO for updating an activity.
type UpdateActivityRequest struct {
	ParentID          *uuid.UUID             `json:"parent_id,omitempty"`
	Name              string                 `json:"name,omitempty"`
	Description       string                 `json:"description,omitempty"`
	Kind              ActivityKind           `json:"kind,omitempty"`
	ContractID        *uuid.UUID             `json:"contract_id,omitempty"`
	BeneficiaryUnitID *uuid.UUID             `json:"beneficiary_unit_id,omitempty"` // COV-05: editable, not an origin ref
	GovernanceModel   models.GovernanceModel `json:"governance_model,omitempty"`
	IsShared          *bool                  `json:"is_shared,omitempty"`
	Billable          *bool                  `json:"billable,omitempty"`
	BudgetAmount      *float64               `json:"budget_amount,omitempty"`
	IsActive          *bool                  `json:"is_active,omitempty"`

	// Origin fields are present so the service immutability guard can reject
	// them (D-03, T-11-10); the repo UPDATE never touches origin columns.
	OriginType *string    `json:"origin_type,omitempty"`
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty"`
	AssignedTo *uuid.UUID `json:"assigned_to,omitempty"`
	ProposedBy *uuid.UUID `json:"proposed_by,omitempty"`
	ReviewedBy *uuid.UUID `json:"reviewed_by,omitempty"`
	TicketID   *uuid.UUID `json:"ticket_id,omitempty"`
}

// ActivityFilter filters the List query.
type ActivityFilter struct {
	Scope      string // "adopted" | "all" | "own" (default)
	ContractID string
	ParentID   string
	Kind       string
	IsActive   *bool
}
