package customer

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	customerdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/customer"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

func setupService(t *testing.T) (*Service, *testdata.MockCustomerRepo) {
	t.Helper()
	repo := &testdata.MockCustomerRepo{}
	svc := NewService(repo)
	return svc, repo
}

func seedCustomer(repo *testdata.MockCustomerRepo, overrides ...func(*customerdomain.Customer)) *customerdomain.Customer {
	c := &customerdomain.Customer{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		CompanyName:    "Test Company",
		Email:          "test@example.com",
		Phone:          "+1-555-0100",
		ContactName:    "John Contact",
		VATNumber:      "VAT123",
		Address:        "123 Test St",
		IsActive:       true,
		CreatedAt:      time.Now(),
	}
	for _, o := range overrides {
		o(c)
	}
	if repo.Customers == nil {
		repo.Customers = make(map[uuid.UUID]*customerdomain.Customer)
	}
	repo.Customers[c.ID] = c
	return c
}

func TestService_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		role    string
		req     *customerdomain.CreateCustomerRequest
		wantErr error
	}{
		{
			name: "valid customer",
			role: string(models.RoleFinance),
			req: &customerdomain.CreateCustomerRequest{
				CompanyName: "New Company",
				Email:       "newco@example.com",
			},
			wantErr: nil,
		},
		{
			name: "missing company name",
			role: string(models.RoleFinance),
			req: &customerdomain.CreateCustomerRequest{
				CompanyName: "",
				Email:       "bad@example.com",
			},
			wantErr: customerdomain.ErrInvalidCustomer,
		},
		{
			name: "unauthorized role",
			role: string(models.RoleEmployee),
			req: &customerdomain.CreateCustomerRequest{
				CompanyName: "Sneaky Co",
			},
			wantErr: customerdomain.ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := setupService(t)
			orgID := uuid.New()
			result, err := svc.Create(context.Background(), orgID, tt.role, tt.req)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
				return
			}
			assert.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, orgID, result.OrganizationID)
			assert.True(t, result.IsActive)
		})
	}
}

func TestService_List_byOrg(t *testing.T) {
	svc, repo := setupService(t)
	orgID := uuid.New()
	seedCustomer(repo, func(c *customerdomain.Customer) { c.OrganizationID = orgID })
	seedCustomer(repo, func(c *customerdomain.Customer) { c.OrganizationID = orgID })
	seedCustomer(repo, func(c *customerdomain.Customer) { c.OrganizationID = orgID })

	results, err := svc.List(context.Background(), orgID, 100, 0)
	assert.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestService_List_empty(t *testing.T) {
	svc, _ := setupService(t)
	results, err := svc.List(context.Background(), uuid.New(), 100, 0)
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestService_Get(t *testing.T) {
	svc, repo := setupService(t)
	seeded := seedCustomer(repo)

	t.Run("existing", func(t *testing.T) {
		result, contracts, err := svc.Get(context.Background(), seeded.ID)
		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, seeded.ID, result.ID)
		assert.Equal(t, seeded.CompanyName, result.CompanyName)
		// contracts is nil because mock returns nil (no linked contracts)
		assert.Nil(t, contracts)
	})

	t.Run("not found", func(t *testing.T) {
		result, contracts, err := svc.Get(context.Background(), uuid.New())
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Nil(t, contracts)
	})
}

func TestService_Update(t *testing.T) {
	t.Parallel()

	t.Run("update contact info", func(t *testing.T) {
		svc, repo := setupService(t)
		orgID := uuid.New()
		seeded := seedCustomer(repo, func(c *customerdomain.Customer) {
			c.OrganizationID = orgID
		})
		result, err := svc.Update(context.Background(), seeded.ID, orgID, string(models.RoleFinance), &customerdomain.UpdateCustomerRequest{
			Email: "updated@example.com",
			Phone: "+1-555-0200",
		})
		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "updated@example.com", result.Email)
		assert.Equal(t, "+1-555-0200", result.Phone)
	})

	t.Run("unauthorized role", func(t *testing.T) {
		svc, repo := setupService(t)
		orgID := uuid.New()
		seeded := seedCustomer(repo, func(c *customerdomain.Customer) {
			c.OrganizationID = orgID
		})
		result, err := svc.Update(context.Background(), seeded.ID, orgID, string(models.RoleEmployee), &customerdomain.UpdateCustomerRequest{
			Email: "hacker@example.com",
		})
		assert.ErrorIs(t, err, customerdomain.ErrForbidden)
		assert.Nil(t, result)
	})

	t.Run("wrong org forbidden", func(t *testing.T) {
		svc, repo := setupService(t)
		orgID := uuid.New()
		seeded := seedCustomer(repo, func(c *customerdomain.Customer) {
			c.OrganizationID = uuid.New()
		})
		result, err := svc.Update(context.Background(), seeded.ID, orgID, string(models.RoleFinance), &customerdomain.UpdateCustomerRequest{
			Email: "cross@example.com",
		})
		assert.ErrorIs(t, err, customerdomain.ErrForbidden)
		assert.Nil(t, result)
	})
}

func TestService_Deactivate(t *testing.T) {
	t.Parallel()

	t.Run("finance deactivates", func(t *testing.T) {
		svc, repo := setupService(t)
		orgID := uuid.New()
		seeded := seedCustomer(repo, func(c *customerdomain.Customer) {
			c.OrganizationID = orgID
		})
		err := svc.Delete(context.Background(), seeded.ID, orgID, string(models.RoleFinance))
		assert.NoError(t, err)
	})

	t.Run("unauthorized role", func(t *testing.T) {
		svc, repo := setupService(t)
		orgID := uuid.New()
		seeded := seedCustomer(repo, func(c *customerdomain.Customer) {
			c.OrganizationID = orgID
		})
		err := svc.Delete(context.Background(), seeded.ID, orgID, string(models.RoleEmployee))
		assert.ErrorIs(t, err, customerdomain.ErrForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		svc, _ := setupService(t)
		err := svc.Delete(context.Background(), uuid.New(), uuid.New(), string(models.RoleFinance))
		assert.Error(t, err)
	})
}
