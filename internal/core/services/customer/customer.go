package customer

import (
	"context"
	"time"

	"github.com/google/uuid"
	customerdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/customer"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

type Service struct {
	repo ports.CustomerRepository
}

func NewService(repo ports.CustomerRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]customerdomain.Customer, error) {
	return s.repo.ListByOrg(ctx, orgID, limit, offset)
}

func (s *Service) Create(ctx context.Context, orgID uuid.UUID, role string, req *customerdomain.CreateCustomerRequest) (*customerdomain.Customer, error) {
	if role != string(models.RoleFinance) {
		return nil, customerdomain.ErrForbidden
	}
	if req.CompanyName == "" {
		return nil, customerdomain.ErrInvalidCustomer
	}

	now := time.Now()
	c := &customerdomain.Customer{
		ID:             uuid.New(),
		OrganizationID: orgID,
		CompanyName:    req.CompanyName,
		ContactName:    req.ContactName,
		Email:          req.Email,
		Phone:          req.Phone,
		VATNumber:      req.VATNumber,
		Address:        req.Address,
		IsActive:       true,
		CreatedAt:      now,
	}

	return s.repo.Create(ctx, c)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*customerdomain.Customer, []customerdomain.ContractSummary, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	contracts, err := s.repo.ListContractsByCustomer(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return c, contracts, nil
}

func (s *Service) Update(ctx context.Context, id, orgID uuid.UUID, role string, req *customerdomain.UpdateCustomerRequest) (*customerdomain.Customer, error) {
	if role != string(models.RoleFinance) {
		return nil, customerdomain.ErrForbidden
	}

	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.OrganizationID != orgID {
		return nil, customerdomain.ErrForbidden
	}

	if req.CompanyName != "" {
		current.CompanyName = req.CompanyName
	}
	if req.ContactName != "" {
		current.ContactName = req.ContactName
	}
	if req.Email != "" {
		current.Email = req.Email
	}
	if req.Phone != "" {
		current.Phone = req.Phone
	}
	if req.VATNumber != "" {
		current.VATNumber = req.VATNumber
	}
	if req.Address != "" {
		current.Address = req.Address
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}

	return s.repo.Update(ctx, current)
}

func (s *Service) Delete(ctx context.Context, id, orgID uuid.UUID, role string) error {
	if role != string(models.RoleFinance) {
		return customerdomain.ErrForbidden
	}

	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if current.OrganizationID != orgID {
		return customerdomain.ErrForbidden
	}

	linkedCount, err := s.repo.CountContractsByCustomer(ctx, id)
	if err != nil {
		return err
	}
	if linkedCount > 0 {
		return customerdomain.ErrCustomerLinkedContract
	}

	return s.repo.Deactivate(ctx, id)
}
