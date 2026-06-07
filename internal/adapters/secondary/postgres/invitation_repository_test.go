package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/invitation"
	"github.com/stretchr/testify/require"
)

func TestInvitationRepository_Create_FindByCode(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewInvitationRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	inv := &invitation.Invitation{
		OrganizationID: orgID,
		Code:           uniqueCode(),
		InviteToken:    uuid.New().String(),
		Email:          "test@example.com",
		Status:         invitation.InvitationStatusPending,
		CreatedBy:      userID.String(),
		ExpiresAt:      now.Add(72 * time.Hour),
	}

	created, err := repo.Create(context.Background(), inv)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, created.ID)
	require.Equal(t, inv.Code, created.Code)
	require.Equal(t, inv.InviteToken, created.InviteToken)
	require.Equal(t, inv.Email, created.Email)
	require.Equal(t, invitation.InvitationStatusPending, created.Status)
	require.Equal(t, inv.CreatedBy, created.CreatedBy)

	// Find by code
	got, err := repo.FindByCode(context.Background(), inv.Code)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.Code, got.Code)
}

func TestInvitationRepository_Create_FindByToken(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewInvitationRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	inv := &invitation.Invitation{
		OrganizationID: orgID,
		Code:           uniqueCode(),
		InviteToken:    uuid.New().String(),
		Email:          "token-test@example.com",
		Status:         invitation.InvitationStatusPending,
		CreatedBy:      userID.String(),
		ExpiresAt:      now.Add(72 * time.Hour),
	}

	created, err := repo.Create(context.Background(), inv)
	require.NoError(t, err)

	got, err := repo.FindByToken(context.Background(), created.InviteToken)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.InviteToken, got.InviteToken)
}

func TestInvitationRepository_FindByCode_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewInvitationRepository(pool)
	_, err := repo.FindByCode(context.Background(), "nonexistent-code")
	require.ErrorIs(t, err, invitation.ErrInvitationNotFound)
}

func TestInvitationRepository_FindByToken_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewInvitationRepository(pool)
	_, err := repo.FindByToken(context.Background(), "nonexistent-token")
	require.ErrorIs(t, err, invitation.ErrInvitationNotFound)
}

func TestInvitationRepository_Update(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewInvitationRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	inv := &invitation.Invitation{
		OrganizationID: orgID,
		Code:           uniqueCode(),
		InviteToken:    uuid.New().String(),
		Email:          "update-test@example.com",
		Status:         invitation.InvitationStatusPending,
		CreatedBy:      userID.String(),
		ExpiresAt:      now.Add(72 * time.Hour),
	}

	created, err := repo.Create(context.Background(), inv)
	require.NoError(t, err)

	// Update status to expired (valid in both domain and DB CHECK)
	created.Status = invitation.InvitationStatusExpired
	updated, err := repo.Update(context.Background(), created)
	require.NoError(t, err)
	require.Equal(t, invitation.InvitationStatusExpired, updated.Status)

	// Verify via FindByCode
	got, err := repo.FindByCode(context.Background(), created.Code)
	require.NoError(t, err)
	require.Equal(t, invitation.InvitationStatusExpired, got.Status)
}
