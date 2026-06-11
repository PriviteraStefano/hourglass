package contract

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

var (
	ErrContractNotFound    = errors.New("contract not found")
	ErrInvalidRequest      = errors.New("invalid request")
	ErrForbidden           = errors.New("forbidden")
	ErrAlreadyAdopted      = errors.New("already adopted")
	ErrHasTimeEntries      = errors.New("contract has time entries")
	ErrHasActiveProjects   = errors.New("contract has active projects")
)

type Contract struct {
	ID              uuid.UUID           `json:"id"`
	Name            string              `json:"name"`
	KmRate          float64             `json:"km_rate"`
	Currency        string              `json:"currency"`
	CustomerID      *uuid.UUID          `json:"customer_id,omitempty"`
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	CreatedByOrgID  uuid.UUID           `json:"created_by_org_id"`
	IsShared        bool                `json:"is_shared"`
	IsActive        bool                `json:"is_active"`
	CreatedAt       time.Time           `json:"created_at"`
}

type ContractResponse struct {
	Contract
	CreatedByOrgName  string `json:"created_by_org_name"`
	AdoptionCount     int    `json:"adoption_count"`
	IsAdopted         bool   `json:"is_adopted"`
	CustomerName      string `json:"customer_name"`
	TimeEntriesCount  int    `json:"time_entries_count"`
}

type ContractAdoption struct {
	ID             uuid.UUID `json:"id"`
	ContractID     uuid.UUID `json:"contract_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	AdoptedAt      time.Time `json:"adopted_at"`
}

type CreateContractRequest struct {
	Name            string              `json:"name"`
	KmRate          float64             `json:"km_rate"`
	Currency        string              `json:"currency"`
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	IsShared        bool                `json:"is_shared"`
	CustomerID      *uuid.UUID          `json:"customer_id,omitempty"`
}

type UpdateContractRequest struct {
	Name            string              `json:"name"`
	KmRate          *float64            `json:"km_rate,omitempty"`
	Currency        string              `json:"currency"`
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	IsShared        *bool               `json:"is_shared,omitempty"`
	IsActive        *bool               `json:"is_active,omitempty"`
	CustomerID      *string             `json:"customer_id,omitempty"`
}
