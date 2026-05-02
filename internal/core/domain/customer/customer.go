package customer

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

var (
	ErrCustomerNotFound       = errors.New("customer not found")
	ErrInvalidCustomer        = errors.New("invalid customer")
	ErrForbidden              = errors.New("forbidden")
	ErrCustomerLinkedContract = errors.New("customer linked to contracts")
)

type Customer struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	CompanyName    string
	ContactName    string
	Email          string
	Phone          string
	VATNumber      string
	Address        string
	IsActive       bool
	CreatedAt      time.Time
}

type ContractSummary struct {
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

type CreateCustomerRequest struct {
	CompanyName string
	ContactName string
	Email       string
	Phone       string
	VATNumber   string
	Address     string
}

type UpdateCustomerRequest struct {
	CompanyName string
	ContactName string
	Email       string
	Phone       string
	VATNumber   string
	Address     string
	IsActive    *bool
}
