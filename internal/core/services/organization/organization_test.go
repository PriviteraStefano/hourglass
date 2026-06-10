package organization

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	orgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/organization"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
)

func newMockService() (*Service, *testdata.MockOrgMgmtRepo) {
	repo := &testdata.MockOrgMgmtRepo{}
	svc := NewService(repo, nil)
	return svc, repo
}

// ---------------------------------------------------------------------------
// TestService_Create
// ---------------------------------------------------------------------------

func TestService_Create(t *testing.T) {
	userID := uuid.New()

	t.Run("valid org creates organization", func(t *testing.T) {
		svc, repo := newMockService()

		req := &orgdomain.CreateOrganizationRequest{
			Name: "Test Organization",
		}

		org, err := svc.Create(context.Background(), userID, req)
		require.NoError(t, err)
		require.NotNil(t, org)
		assert.NotEqual(t, uuid.Nil, org.ID)
		assert.Equal(t, "Test Organization", org.Name)

		// Verify stored
		stored, err := repo.GetOrganization(context.Background(), org.ID)
		require.NoError(t, err)
		assert.Equal(t, "Test Organization", stored.Name)
	})

	t.Run("missing name returns error", func(t *testing.T) {
		svc, _ := newMockService()

		req := &orgdomain.CreateOrganizationRequest{
			Name: "",
		}

		org, err := svc.Create(context.Background(), userID, req)
		assert.ErrorIs(t, err, orgdomain.ErrInvalidRequest)
		assert.Nil(t, org)
	})

	t.Run("creates with generated slug when slug empty", func(t *testing.T) {
		svc, _ := newMockService()

		req := &orgdomain.CreateOrganizationRequest{
			Name: "My Org Name",
		}

		org, err := svc.Create(context.Background(), userID, req)
		require.NoError(t, err)
		require.NotNil(t, org)
		assert.Equal(t, "my-org-name", org.Slug)
	})

	t.Run("uses provided slug", func(t *testing.T) {
		svc, _ := newMockService()

		req := &orgdomain.CreateOrganizationRequest{
			Name: "My Org Name",
			Slug: "custom-slug",
		}

		org, err := svc.Create(context.Background(), userID, req)
		require.NoError(t, err)
		require.NotNil(t, org)
		assert.Equal(t, "custom-slug", org.Slug)
	})
}

// ---------------------------------------------------------------------------
// TestService_Get
// ---------------------------------------------------------------------------

func TestService_Get(t *testing.T) {
	t.Run("existing org returns organization", func(t *testing.T) {
		svc, repo := newMockService()

		// Pre-seed org
		orgID := uuid.New()
		repo.Orgs = map[uuid.UUID]*orgdomain.Organization{
			orgID: {
				ID:   orgID,
				Name: "Existing Org",
				Slug: "existing-org",
			},
		}

		org, err := svc.Get(context.Background(), orgID)
		require.NoError(t, err)
		require.NotNil(t, org)
		assert.Equal(t, "Existing Org", org.Name)
	})

	t.Run("nonexistent org returns error", func(t *testing.T) {
		svc, _ := newMockService()

		org, err := svc.Get(context.Background(), uuid.New())
		assert.ErrorIs(t, err, orgdomain.ErrOrganizationNotFound)
		assert.Nil(t, org)
	})
}

// ---------------------------------------------------------------------------
// TestService_GetSettings
// ---------------------------------------------------------------------------

func TestService_GetSettings(t *testing.T) {
	t.Run("returns settings for org", func(t *testing.T) {
		svc, repo := newMockService()
		orgID := uuid.New()

		repo.Settings = &orgdomain.Settings{
			OrganizationID: orgID,
			Currency:       "EUR",
			WeekStartDay:   1,
			Timezone:       "UTC",
		}

		settings, err := svc.GetSettings(context.Background(), orgID)
		require.NoError(t, err)
		require.NotNil(t, settings)
		assert.Equal(t, "EUR", settings.Currency)
	})
}

// ---------------------------------------------------------------------------
// TestService_UpdateSettings
// ---------------------------------------------------------------------------

func TestService_UpdateSettings(t *testing.T) {
	orgID := uuid.New()
	weekStartDay := 2

	t.Run("finance role updates settings", func(t *testing.T) {
		svc, repo := newMockService()

		req := &orgdomain.UpdateSettingsRequest{
			Currency:     "USD",
			WeekStartDay: &weekStartDay,
		}

		settings, err := svc.UpdateSettings(context.Background(), "finance", orgID, req)
		require.NoError(t, err)
		require.NotNil(t, settings)
		_ = repo // settings returned from mock
	})

	t.Run("non-finance role is forbidden", func(t *testing.T) {
		svc, _ := newMockService()

		req := &orgdomain.UpdateSettingsRequest{
			Currency: "USD",
		}

		settings, err := svc.UpdateSettings(context.Background(), "employee", orgID, req)
		assert.ErrorIs(t, err, orgdomain.ErrForbidden)
		assert.Nil(t, settings)
	})

	t.Run("manager role is forbidden", func(t *testing.T) {
		svc, _ := newMockService()

		req := &orgdomain.UpdateSettingsRequest{
			Currency: "USD",
		}

		settings, err := svc.UpdateSettings(context.Background(), "manager", orgID, req)
		assert.ErrorIs(t, err, orgdomain.ErrForbidden)
		assert.Nil(t, settings)
	})
}
