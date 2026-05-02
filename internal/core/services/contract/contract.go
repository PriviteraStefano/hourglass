package contract

import (
	"context"

	"github.com/google/uuid"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

type Service struct {
	repo ports.ContractRepository
}

func NewService(repo ports.ContractRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, orgID uuid.UUID, scope string) ([]contractdomain.ContractResponse, error) {
	return s.repo.List(ctx, orgID, scope)
}

func (s *Service) Create(ctx context.Context, orgID uuid.UUID, req *contractdomain.CreateContractRequest) (*contractdomain.ContractResponse, error) {
	if req.Name == "" || !req.GovernanceModel.IsValid() {
		return nil, contractdomain.ErrInvalidRequest
	}
	if req.Currency == "" {
		req.Currency = "EUR"
	}
	return s.repo.Create(ctx, orgID, req)
}

func (s *Service) Get(ctx context.Context, orgID, contractID uuid.UUID) (*contractdomain.ContractResponse, error) {
	return s.repo.Get(ctx, orgID, contractID)
}

func (s *Service) Adopt(ctx context.Context, orgID, contractID uuid.UUID) (*contractdomain.ContractAdoption, error) {
	return s.repo.Adopt(ctx, orgID, contractID)
}

func (s *Service) Update(ctx context.Context, role string, orgID, contractID uuid.UUID, req *contractdomain.UpdateContractRequest) (*contractdomain.ContractResponse, int, error) {
	if role != string(models.RoleFinance) {
		return nil, 0, contractdomain.ErrForbidden
	}
	return s.repo.Update(ctx, orgID, contractID, req)
}

func (s *Service) RecalculateMileage(ctx context.Context, role string, orgID, contractID uuid.UUID, fromDate string, actorUserID uuid.UUID) (int, error) {
	if role != string(models.RoleFinance) {
		return 0, contractdomain.ErrForbidden
	}
	if fromDate == "" {
		return 0, contractdomain.ErrInvalidRequest
	}
	return s.repo.RecalculateMileage(ctx, orgID, contractID, fromDate, actorUserID)
}
