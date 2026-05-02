package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/customer"
)

type CustomerRepository interface {
	ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]customer.Customer, error)
	Create(ctx context.Context, c *customer.Customer) (*customer.Customer, error)
	GetByID(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
	Update(ctx context.Context, c *customer.Customer) (*customer.Customer, error)
	Deactivate(ctx context.Context, id uuid.UUID) error

	ListContractsByCustomer(ctx context.Context, customerID uuid.UUID) ([]customer.ContractSummary, error)
	CountContractsByCustomer(ctx context.Context, customerID uuid.UUID) (int, error)
}
