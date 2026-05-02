package ports

import (
	"context"

	"github.com/google/uuid"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
)

type ContractRepository interface {
	List(ctx context.Context, orgID uuid.UUID, scope string) ([]contractdomain.ContractResponse, error)
	Create(ctx context.Context, orgID uuid.UUID, req *contractdomain.CreateContractRequest) (*contractdomain.ContractResponse, error)
	Get(ctx context.Context, orgID, contractID uuid.UUID) (*contractdomain.ContractResponse, error)
	Adopt(ctx context.Context, orgID, contractID uuid.UUID) (*contractdomain.ContractAdoption, error)
	Update(ctx context.Context, orgID, contractID uuid.UUID, req *contractdomain.UpdateContractRequest) (*contractdomain.ContractResponse, int, error)
	RecalculateMileage(ctx context.Context, orgID, contractID uuid.UUID, fromDate string, actorUserID uuid.UUID) (int, error)
}
