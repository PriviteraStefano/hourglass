package contract

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

func setupService(t *testing.T) (*Service, *testdata.MockContractRepo) {
	t.Helper()
	repo := &testdata.MockContractRepo{}
	svc := NewService(repo)
	return svc, repo
}

func seedContract(repo *testdata.MockContractRepo, overrides ...func(*contractdomain.ContractResponse)) *contractdomain.ContractResponse {
	c := &contractdomain.ContractResponse{
		Contract: contractdomain.Contract{
			ID:              uuid.New(),
			Name:            "Test Contract",
			KmRate:          0.50,
			Currency:        "EUR",
			GovernanceModel: models.GovernanceCreatorControlled,
			CreatedByOrgID:  uuid.New(),
			IsActive:        true,
			CreatedAt:       time.Now(),
		},
	}
	for _, o := range overrides {
		o(c)
	}
	if repo.Contracts == nil {
		repo.Contracts = make(map[uuid.UUID]*contractdomain.ContractResponse)
	}
	repo.Contracts[c.ID] = c
	return c
}

func TestService_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     *contractdomain.CreateContractRequest
		wantErr error
	}{
		{
			name: "valid contract",
			req: &contractdomain.CreateContractRequest{
				Name:            "New Contract",
				GovernanceModel: models.GovernanceCreatorControlled,
			},
			wantErr: nil,
		},
		{
			name: "missing name",
			req: &contractdomain.CreateContractRequest{
				Name:            "",
				GovernanceModel: models.GovernanceCreatorControlled,
			},
			wantErr: contractdomain.ErrInvalidRequest,
		},
		{
			name: "invalid governance model",
			req: &contractdomain.CreateContractRequest{
				Name:            "Bad Gov",
				GovernanceModel: "invalid_gov",
			},
			wantErr: contractdomain.ErrInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := setupService(t)
			result, err := svc.Create(context.Background(), uuid.New(), tt.req)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestService_List_byOrg(t *testing.T) {
	svc, repo := setupService(t)
	orgID := uuid.New()
	seedContract(repo, func(c *contractdomain.ContractResponse) { c.CreatedByOrgID = orgID })
	seedContract(repo, func(c *contractdomain.ContractResponse) { c.CreatedByOrgID = orgID })

	results, err := svc.List(context.Background(), orgID, "", nil)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestService_List_empty(t *testing.T) {
	svc, _ := setupService(t)
	results, err := svc.List(context.Background(), uuid.New(), "", nil)
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestService_Get(t *testing.T) {
	svc, repo := setupService(t)
	orgID := uuid.New()
	seeded := seedContract(repo, func(c *contractdomain.ContractResponse) { c.CreatedByOrgID = orgID })

	t.Run("existing", func(t *testing.T) {
		result, err := svc.Get(context.Background(), orgID, seeded.ID)
		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, seeded.ID, result.ID)
		assert.Equal(t, seeded.Name, result.Name)
	})

	t.Run("not found", func(t *testing.T) {
		result, err := svc.Get(context.Background(), orgID, uuid.New())
		assert.ErrorIs(t, err, contractdomain.ErrContractNotFound)
		assert.Nil(t, result)
	})
}

func TestService_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		role    string
		wantErr error
	}{
		{
			name:    "finance role updates",
			role:    string(models.RoleFinance),
			wantErr: nil,
		},
		{
			name:    "non-finance role forbidden",
			role:    string(models.RoleEmployee),
			wantErr: contractdomain.ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := setupService(t)
			orgID := uuid.New()
			seeded := seedContract(repo, func(c *contractdomain.ContractResponse) { c.CreatedByOrgID = orgID })
			result, _, err := svc.Update(context.Background(), tt.role, orgID, seeded.ID, &contractdomain.UpdateContractRequest{
				Name: "Updated Contract",
			})
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestService_Delete(t *testing.T) {
	t.Parallel()

	t.Run("finance role deletes", func(t *testing.T) {
		svc, repo := setupService(t)
		orgID := uuid.New()
		seeded := seedContract(repo, func(c *contractdomain.ContractResponse) { c.CreatedByOrgID = orgID })
		err := svc.Delete(context.Background(), string(models.RoleFinance), orgID, seeded.ID)
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		svc, _ := setupService(t)
		err := svc.Delete(context.Background(), string(models.RoleFinance), uuid.New(), uuid.New())
		assert.Error(t, err)
	})

	t.Run("unauthorized role", func(t *testing.T) {
		svc, repo := setupService(t)
		orgID := uuid.New()
		seeded := seedContract(repo, func(c *contractdomain.ContractResponse) { c.CreatedByOrgID = orgID })
		err := svc.Delete(context.Background(), string(models.RoleEmployee), orgID, seeded.ID)
		assert.ErrorIs(t, err, contractdomain.ErrForbidden)
	})
}
