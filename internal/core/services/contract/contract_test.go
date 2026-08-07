package contract

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			name: "valid contract with customer",
			req: &contractdomain.CreateContractRequest{
				Name:            "Customer Contract",
				GovernanceModel: models.GovernanceCreatorControlled,
				CustomerID:      &uuid.UUID{1},
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
		{
			name: "support contract without sold period",
			req: &contractdomain.CreateContractRequest{
				Name:            "Support No Period",
				GovernanceModel: models.GovernanceCreatorControlled,
				ContractType:    strPtr(contractdomain.ContractTypeSupport),
				SoldHours:       f64Ptr(100),
			},
			wantErr: contractdomain.ErrInvalidSoldConfig,
		},
		{
			name: "support contract without sold hours",
			req: &contractdomain.CreateContractRequest{
				Name:            "Support No Hours",
				GovernanceModel: models.GovernanceCreatorControlled,
				ContractType:    strPtr(contractdomain.ContractTypeSupport),
				SoldPeriod:      strPtr(contractdomain.SoldPeriodMonth),
			},
			wantErr: contractdomain.ErrInvalidSoldConfig,
		},
		{
			name: "project contract with sold period",
			req: &contractdomain.CreateContractRequest{
				Name:            "Project With Period",
				GovernanceModel: models.GovernanceCreatorControlled,
				ContractType:    strPtr(contractdomain.ContractTypeProject),
				SoldHours:       f64Ptr(100),
				SoldPeriod:      strPtr(contractdomain.SoldPeriodMonth),
			},
			wantErr: contractdomain.ErrInvalidSoldConfig,
		},
		{
			name: "unknown contract type",
			req: &contractdomain.CreateContractRequest{
				Name:            "Unknown Type",
				GovernanceModel: models.GovernanceCreatorControlled,
				ContractType:    strPtr("subscription"),
			},
			wantErr: contractdomain.ErrInvalidSoldConfig,
		},
		{
			name: "invalid sold period value",
			req: &contractdomain.CreateContractRequest{
				Name:            "Bad Period",
				GovernanceModel: models.GovernanceCreatorControlled,
				ContractType:    strPtr(contractdomain.ContractTypeSupport),
				SoldHours:       f64Ptr(100),
				SoldPeriod:      strPtr("decade"),
			},
			wantErr: contractdomain.ErrInvalidSoldConfig,
		},
		{
			name: "legacy contract with sold hours only",
			req: &contractdomain.CreateContractRequest{
				Name:            "Legacy Sold Hours",
				GovernanceModel: models.GovernanceCreatorControlled,
				SoldHours:       f64Ptr(250),
			},
			wantErr: nil,
		},
		{
			name: "valid support contract",
			req: &contractdomain.CreateContractRequest{
				Name:            "Valid Support",
				GovernanceModel: models.GovernanceCreatorControlled,
				ContractType:    strPtr(contractdomain.ContractTypeSupport),
				SoldHours:       f64Ptr(100),
				SoldPeriod:      strPtr(contractdomain.SoldPeriodMonth),
			},
			wantErr: nil,
		},
		{
			name: "valid project contract",
			req: &contractdomain.CreateContractRequest{
				Name:            "Valid Project",
				GovernanceModel: models.GovernanceCreatorControlled,
				ContractType:    strPtr(contractdomain.ContractTypeProject),
				SoldHours:       f64Ptr(1200),
			},
			wantErr: nil,
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
		req     *contractdomain.UpdateContractRequest
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
		{
			name: "finance sets support without sold period",
			role: string(models.RoleFinance),
			req: &contractdomain.UpdateContractRequest{
				ContractType: strPtr(contractdomain.ContractTypeSupport),
				SoldHours:    f64Ptr(100),
			},
			wantErr: contractdomain.ErrInvalidSoldConfig,
		},
		{
			name: "finance sets project with sold period",
			role: string(models.RoleFinance),
			req: &contractdomain.UpdateContractRequest{
				ContractType: strPtr(contractdomain.ContractTypeProject),
				SoldPeriod:   strPtr(contractdomain.SoldPeriodQuarter),
			},
			wantErr: contractdomain.ErrInvalidSoldConfig,
		},
		{
			name: "finance sets invalid sold period",
			role: string(models.RoleFinance),
			req: &contractdomain.UpdateContractRequest{
				ContractType: strPtr(contractdomain.ContractTypeSupport),
				SoldHours:    f64Ptr(100),
				SoldPeriod:   strPtr("decade"),
			},
			wantErr: contractdomain.ErrInvalidSoldConfig,
		},
		{
			name: "finance sets valid support config",
			role: string(models.RoleFinance),
			req: &contractdomain.UpdateContractRequest{
				ContractType: strPtr(contractdomain.ContractTypeSupport),
				SoldHours:    f64Ptr(100),
				SoldPeriod:   strPtr(contractdomain.SoldPeriodYear),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := setupService(t)
			orgID := uuid.New()
			seeded := seedContract(repo, func(c *contractdomain.ContractResponse) { c.CreatedByOrgID = orgID })
			req := tt.req
			if req == nil {
				req = &contractdomain.UpdateContractRequest{Name: "Updated Contract"}
			}
			result, _, err := svc.Update(context.Background(), tt.role, orgID, seeded.ID, req)
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

func TestService_Update_SupportToProjectConversion(t *testing.T) {
	// WR-03: a support→project conversion must validate the MERGED config
	// (current row + request deltas) and accept sold_period: "" as the
	// explicit clear — otherwise the conversion 500s on the DB CHECK
	// (contracts_sold_check) because sold_period stays set in the row.
	svc, repo := setupService(t)
	orgID := uuid.New()
	support := strPtr(contractdomain.ContractTypeSupport)
	month := strPtr(contractdomain.SoldPeriodMonth)
	hours := f64Ptr(100)
	seeded := seedContract(repo, func(c *contractdomain.ContractResponse) {
		c.CreatedByOrgID = orgID
		c.ContractType = support
		c.SoldHours = hours
		c.SoldPeriod = month
	})

	t.Run("conversion clears sold_period in the same request", func(t *testing.T) {
		project := strPtr(contractdomain.ContractTypeProject)
		clear := ""
		result, _, err := svc.Update(context.Background(), string(models.RoleFinance), orgID, seeded.ID, &contractdomain.UpdateContractRequest{
			ContractType: project,
			SoldPeriod:   &clear,
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("conversion without clearing sold_period rejected", func(t *testing.T) {
		project := strPtr(contractdomain.ContractTypeProject)
		result, _, err := svc.Update(context.Background(), string(models.RoleFinance), orgID, seeded.ID, &contractdomain.UpdateContractRequest{
			ContractType: project,
		})
		assert.ErrorIs(t, err, contractdomain.ErrInvalidSoldConfig)
		assert.Nil(t, result)
	})

	t.Run("merged validation keeps untouched sold config valid", func(t *testing.T) {
		// Name-only update on a support contract: the merged row keeps
		// support + hours + month → still valid.
		result, _, err := svc.Update(context.Background(), string(models.RoleFinance), orgID, seeded.ID, &contractdomain.UpdateContractRequest{
			Name: "Renamed support",
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("missing contract surfaces ErrContractNotFound", func(t *testing.T) {
		result, _, err := svc.Update(context.Background(), string(models.RoleFinance), orgID, uuid.New(), &contractdomain.UpdateContractRequest{Name: "Ghost"})
		assert.ErrorIs(t, err, contractdomain.ErrContractNotFound)
		assert.Nil(t, result)
	})
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

	t.Run("blocked by projects", func(t *testing.T) {
		svc, repo := setupService(t)
		orgID := uuid.New()
		seeded := seedContract(repo, func(c *contractdomain.ContractResponse) { c.CreatedByOrgID = orgID })
		repo.HasProjectsFn = func(ctx context.Context, contractID uuid.UUID) (int, error) {
			return 1, nil
		}
		err := svc.Delete(context.Background(), string(models.RoleFinance), orgID, seeded.ID)
		assert.ErrorIs(t, err, contractdomain.ErrHasActiveProjects)
	})
}

func strPtr(s string) *string {
	return &s
}

func f64Ptr(v float64) *float64 {
	return &v
}
