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

func (s *Service) List(ctx context.Context, orgID uuid.UUID, scope string, isActive *bool) ([]contractdomain.ContractResponse, error) {
	return s.repo.List(ctx, orgID, scope, isActive)
}

func (s *Service) Create(ctx context.Context, orgID uuid.UUID, req *contractdomain.CreateContractRequest) (*contractdomain.ContractResponse, error) {
	if req.Name == "" || !req.GovernanceModel.IsValid() {
		return nil, contractdomain.ErrInvalidRequest
	}
	if req.Currency == "" {
		req.Currency = "EUR"
	}
	if err := validateSoldConfig(req.ContractType, req.SoldHours, req.SoldPeriod); err != nil {
		return nil, err
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
	if err := validateSoldConfig(req.ContractType, req.SoldHours, req.SoldPeriod); err != nil {
		return nil, 0, err
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

func (s *Service) Delete(ctx context.Context, role string, orgID, contractID uuid.UUID) error {
	if role != string(models.RoleFinance) {
		return contractdomain.ErrForbidden
	}
	existing, err := s.repo.Get(ctx, orgID, contractID)
	if err != nil {
		return err
	}
	if existing.CreatedByOrgID != orgID {
		return contractdomain.ErrForbidden
	}
	count, err := s.repo.HasTimeEntries(ctx, contractID)
	if err != nil {
		return err
	}
	if count > 0 {
		return contractdomain.ErrHasTimeEntries
	}

	projectCount, err := s.repo.HasProjects(ctx, contractID)
	if err != nil {
		return err
	}
	if projectCount > 0 {
		return contractdomain.ErrHasActiveProjects
	}

	return s.repo.Delete(ctx, orgID, contractID)
}

// validateSoldConfig enforces the sold-hours semantics (D-08/D-09):
//   - contract_type NULL (legacy, D-16) → OK, treated as project
//   - 'support' → sold_hours AND sold_period required
//   - 'project' → sold_period must stay NULL
//   - any other contract_type → rejected (closed set)
//   - sold_period, when set, must be one of month/quarter/year
//
// Both Create and Update validate the fields present in the request. The DB
// CHECK contracts_sold_check backstops this at the row level (migration 016).
func validateSoldConfig(contractType *string, soldHours *float64, soldPeriod *string) error {
	if contractType == nil {
		return nil // legacy contract — treated as project
	}
	switch *contractType {
	case contractdomain.ContractTypeSupport:
		if soldHours == nil || soldPeriod == nil {
			return contractdomain.ErrInvalidSoldConfig
		}
	case contractdomain.ContractTypeProject:
		if soldPeriod != nil {
			return contractdomain.ErrInvalidSoldConfig
		}
	default:
		return contractdomain.ErrInvalidSoldConfig
	}
	if soldPeriod != nil {
		switch *soldPeriod {
		case contractdomain.SoldPeriodMonth, contractdomain.SoldPeriodQuarter, contractdomain.SoldPeriodYear:
		default:
			return contractdomain.ErrInvalidSoldConfig
		}
	}
	return nil
}
