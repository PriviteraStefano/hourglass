package contract

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

var (
	ErrContractNotFound = errors.New("contract not found")
	ErrInvalidRequest   = errors.New("invalid request")
	ErrForbidden        = errors.New("forbidden")
	ErrAlreadyAdopted   = errors.New("already adopted")
)

type Contract struct {
	ID              uuid.UUID
	Name            string
	KmRate          float64
	Currency        string
	CustomerID      *uuid.UUID
	GovernanceModel models.GovernanceModel
	CreatedByOrgID  uuid.UUID
	IsShared        bool
	IsActive        bool
	CreatedAt       time.Time
}

type ContractResponse struct {
	Contract
	CreatedByOrgName string
	AdoptionCount    int
	IsAdopted        bool
}

type ContractAdoption struct {
	ID             uuid.UUID
	ContractID     uuid.UUID
	OrganizationID uuid.UUID
	AdoptedAt      time.Time
}

type CreateContractRequest struct {
	Name            string
	KmRate          float64
	Currency        string
	GovernanceModel models.GovernanceModel
	IsShared        bool
}

type UpdateContractRequest struct {
	Name            string
	KmRate          *float64
	Currency        string
	GovernanceModel models.GovernanceModel
	IsShared        *bool
	IsActive        *bool
	CustomerID      *string
}
