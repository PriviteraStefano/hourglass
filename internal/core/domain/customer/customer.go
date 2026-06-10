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
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	CompanyName    string    `json:"company_name"`
	ContactName    string    `json:"contact_name"`
	Email          string    `json:"email"`
	Phone          string    `json:"phone"`
	VATNumber      string    `json:"vat_number"`
	Address        string    `json:"address"`
	IsActive       bool      `json:"is_active"`
	IsInternal     bool      `json:"is_internal"`
	CreatedAt      time.Time `json:"created_at"`
}

type ContractSummary struct {
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

type CreateCustomerRequest struct {
	CompanyName string `json:"company_name"`
	ContactName string `json:"contact_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	VATNumber   string `json:"vat_number"`
	Address     string `json:"address"`
}

type UpdateCustomerRequest struct {
	CompanyName string    `json:"company_name"`
	ContactName string    `json:"contact_name"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	VATNumber   string    `json:"vat_number"`
	Address     string    `json:"address"`
	IsActive    *bool     `json:"is_active,omitempty"`
}
