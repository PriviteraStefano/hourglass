package surrealdb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/invitation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupInvitationRepo(t *testing.T) (*InvitationRepository, func()) {
	t.Helper()

	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	ns := "test_invitation_" + uuid.New().String()
	db, err := GetTestDBWithNamespace(ns, ns)
	if err != nil {
		t.Skipf("SurrealDB not available: %v", err)
	}

	repo := NewInvitationRepository(db)
	return repo, func() { db.Close(context.Background()) }
}

func seedInvitationOrg(t *testing.T, repo *InvitationRepository) uuid.UUID {
	t.Helper()

	db := repo.db
	orgRepo := NewOrganizationRepository(db)
	orgID := uuid.New()
	org := &auth.Organization{
		ID:        orgID,
		Name:      "Invite Test Org " + uuid.New().String()[:8],
		Slug:      "inv-org-" + uuid.New().String()[:8],
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := orgRepo.Add(context.Background(), org)
	require.NoError(t, err, "failed to seed org")

	return orgID
}

func TestInvitationRepo_Create(t *testing.T) {
	repo, cleanup := setupInvitationRepo(t)
	defer cleanup()
	orgID := seedInvitationOrg(t, repo)

	inv := &invitation.Invitation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Code:           "inv-code-" + uuid.New().String()[:8],
		InviteToken:    "token-" + uuid.New().String(),
		Email:          "user@example.com",
		Status:         invitation.InvitationStatusPending,
		ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
		CreatedBy:      "admin",
		CreatedAt:      time.Now(),
	}

	created, err := repo.Create(context.Background(), inv)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.NotEqual(t, uuid.Nil, created.ID)
	assert.Equal(t, inv.Code, created.Code)
	assert.Equal(t, inv.Email, created.Email)
	assert.Equal(t, invitation.InvitationStatusPending, created.Status)
}

func TestInvitationRepo_FindByCode(t *testing.T) {
	repo, cleanup := setupInvitationRepo(t)
	defer cleanup()
	orgID := seedInvitationOrg(t, repo)

	t.Run("existing", func(t *testing.T) {
		code := "find-code-" + uuid.New().String()[:8]
		inv := &invitation.Invitation{
			ID:             uuid.New(),
			OrganizationID: orgID,
			Code:           code,
			InviteToken:    "find-token-" + uuid.New().String(),
			Email:          "find@example.com",
			Status:         invitation.InvitationStatusPending,
			ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
			CreatedBy:      "admin",
			CreatedAt:      time.Now(),
		}

		_, err := repo.Create(context.Background(), inv)
		require.NoError(t, err)

		found, err := repo.FindByCode(context.Background(), code)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, code, found.Code)
		assert.Equal(t, inv.Email, found.Email)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := repo.FindByCode(context.Background(), "nonexistent-code")
		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestInvitationRepo_FindByID(t *testing.T) {
	repo, cleanup := setupInvitationRepo(t)
	defer cleanup()
	orgID := seedInvitationOrg(t, repo)

	t.Run("existing", func(t *testing.T) {
		invID := uuid.New()
		inv := &invitation.Invitation{
			ID:             invID,
			OrganizationID: orgID,
			Code:           "findbyid-code-" + uuid.New().String()[:8],
			InviteToken:    "findbyid-token-" + uuid.New().String(),
			Email:          "findbyid@example.com",
			Status:         invitation.InvitationStatusPending,
			ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
			CreatedBy:      "admin",
			CreatedAt:      time.Now(),
		}

		_, err := repo.Create(context.Background(), inv)
		require.NoError(t, err)

		found, err := repo.FindByID(context.Background(), invID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, invID, found.ID)
	})
}

func TestInvitationRepo_FindByToken(t *testing.T) {
	repo, cleanup := setupInvitationRepo(t)
	defer cleanup()
	orgID := seedInvitationOrg(t, repo)

	t.Run("existing", func(t *testing.T) {
		token := "token-" + uuid.New().String()
		inv := &invitation.Invitation{
			ID:             uuid.New(),
			OrganizationID: orgID,
			Code:           "fbtoken-code-" + uuid.New().String()[:8],
			InviteToken:    token,
			Email:          "fbtoken@example.com",
			Status:         invitation.InvitationStatusPending,
			ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
			CreatedBy:      "admin",
			CreatedAt:      time.Now(),
		}

		_, err := repo.Create(context.Background(), inv)
		require.NoError(t, err)

		found, err := repo.FindByToken(context.Background(), token)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, token, found.InviteToken)
	})
}

func TestInvitationRepo_Update(t *testing.T) {
	repo, cleanup := setupInvitationRepo(t)
	defer cleanup()
	orgID := seedInvitationOrg(t, repo)

	inv := &invitation.Invitation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Code:           "update-code-" + uuid.New().String()[:8],
		InviteToken:    "update-token-" + uuid.New().String(),
		Email:          "update@example.com",
		Status:         invitation.InvitationStatusPending,
		ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
		CreatedBy:      "admin",
		CreatedAt:      time.Now(),
	}

	created, err := repo.Create(context.Background(), inv)
	require.NoError(t, err)

	created.Status = invitation.InvitationStatusUsed

	updated, err := repo.Update(context.Background(), created)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, invitation.InvitationStatusUsed, updated.Status)
}
