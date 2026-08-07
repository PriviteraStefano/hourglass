package contract

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

var (
	ErrContractNotFound  = errors.New("contract not found")
	ErrInvalidRequest    = errors.New("invalid request")
	ErrForbidden         = errors.New("forbidden")
	ErrAlreadyAdopted    = errors.New("already adopted")
	ErrHasTimeEntries    = errors.New("contract has time entries")
	ErrHasActiveProjects = errors.New("contract has active projects")
	ErrInvalidSoldConfig = errors.New("invalid sold hours configuration")
)

// Contract type and sold-period vocabulary (D-08/D-09). contract_type NULL
// stays ambiguous — legacy v0.1 contracts are treated as project (D-16).
const (
	ContractTypeProject = "project"
	ContractTypeSupport = "support"

	SoldPeriodMonth   = "month"
	SoldPeriodQuarter = "quarter"
	SoldPeriodYear    = "year"
)

type Contract struct {
	ID              uuid.UUID              `json:"id"`
	Name            string                 `json:"name"`
	KmRate          float64                `json:"km_rate"`
	Currency        string                 `json:"currency"`
	CustomerID      *uuid.UUID             `json:"customer_id,omitempty"`
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	CreatedByOrgID  uuid.UUID              `json:"created_by_org_id"`
	IsShared        bool                   `json:"is_shared"`
	IsActive        bool                   `json:"is_active"`
	ContractType    *string                `json:"contract_type,omitempty"`
	SoldHours       *float64               `json:"sold_hours,omitempty"`
	SoldPeriod      *string                `json:"sold_period,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
}

type ContractResponse struct {
	Contract
	CreatedByOrgName string `json:"created_by_org_name"`
	AdoptionCount    int    `json:"adoption_count"`
	IsAdopted        bool   `json:"is_adopted"`
	CustomerName     string `json:"customer_name"`
	TimeEntriesCount int    `json:"time_entries_count"`
}

type ContractAdoption struct {
	ID             uuid.UUID `json:"id"`
	ContractID     uuid.UUID `json:"contract_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	AdoptedAt      time.Time `json:"adopted_at"`
}

type CreateContractRequest struct {
	Name            string                 `json:"name"`
	KmRate          float64                `json:"km_rate"`
	Currency        string                 `json:"currency"`
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	IsShared        bool                   `json:"is_shared"`
	CustomerID      *uuid.UUID             `json:"customer_id,omitempty"`
	ContractType    *string                `json:"contract_type,omitempty"`
	SoldHours       *float64               `json:"sold_hours,omitempty"`
	SoldPeriod      *string                `json:"sold_period,omitempty"`
}

type UpdateContractRequest struct {
	Name            string                 `json:"name"`
	KmRate          *float64               `json:"km_rate,omitempty"`
	Currency        string                 `json:"currency"`
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	IsShared        *bool                  `json:"is_shared,omitempty"`
	IsActive        *bool                  `json:"is_active,omitempty"`
	CustomerID      *string                `json:"customer_id,omitempty"`
	ContractType    *string                `json:"contract_type,omitempty"`
	SoldHours       *float64               `json:"sold_hours,omitempty"`
	SoldPeriod      *string                `json:"sold_period,omitempty"`
}
