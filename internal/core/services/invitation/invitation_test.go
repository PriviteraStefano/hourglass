package invitation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	invitationdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/invitation"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
)

func setupService(t *testing.T) (*Service, *testdata.MockInvitationRepo) {
	t.Helper()
	repo := &testdata.MockInvitationRepo{}
	svc := NewService(repo)
	return svc, repo
}

func seedInvitation(repo *testdata.MockInvitationRepo, overrides ...func(*invitationdomain.Invitation)) *invitationdomain.Invitation {
	inv := &invitationdomain.Invitation{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Code:           uuid.New().String()[:8],
		InviteToken:    "tok_" + uuid.New().String(),
		Email:          "invited@example.com",
		Status:         invitationdomain.InvitationStatusPending,
		ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
		CreatedBy:      "system",
		CreatedAt:      time.Now(),
	}
	for _, o := range overrides {
		o(inv)
	}
	if repo.Invitations == nil {
		repo.Invitations = make(map[uuid.UUID]*invitationdomain.Invitation)
	}
	repo.Invitations[inv.ID] = inv
	return inv
}

func TestService_Create(t *testing.T) {
	t.Parallel()

	t.Run("valid invitation", func(t *testing.T) {
		svc, _ := setupService(t)
		result, err := svc.Create(context.Background(), &invitationdomain.CreateInvitationRequest{
			OrganizationID: uuid.New(),
			Email:          "newmember@example.com",
			ExpiresInDays:  7,
			CreatedBy:      uuid.New(),
		})
		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, invitationdomain.InvitationStatusPending, result.Status)
		assert.NotEmpty(t, result.InviteToken)
		assert.NotEmpty(t, result.Code)
	})
}

func TestService_ValidateCode(t *testing.T) {
	svc, repo := setupService(t)

	t.Run("valid code", func(t *testing.T) {
		seeded := seedInvitation(repo)
		result, err := svc.ValidateCode(context.Background(), seeded.Code)
		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, seeded.ID, result.ID)
	})

	t.Run("expired code", func(t *testing.T) {
		seeded := seedInvitation(repo, func(inv *invitationdomain.Invitation) {
			inv.ExpiresAt = time.Now().Add(-1 * time.Hour)
		})
		result, err := svc.ValidateCode(context.Background(), seeded.Code)
		assert.ErrorIs(t, err, invitationdomain.ErrInvitationExpired)
		assert.Nil(t, result)
	})

	t.Run("not found", func(t *testing.T) {
		result, err := svc.ValidateCode(context.Background(), "NONEXISTENT")
		assert.ErrorIs(t, err, invitationdomain.ErrInvitationNotFound)
		assert.Nil(t, result)
	})
}

func TestService_ValidateToken(t *testing.T) {
	svc, repo := setupService(t)

	t.Run("valid token", func(t *testing.T) {
		seeded := seedInvitation(repo)
		result, err := svc.ValidateToken(context.Background(), seeded.InviteToken)
		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, seeded.ID, result.ID)
	})

	t.Run("expired token", func(t *testing.T) {
		seeded := seedInvitation(repo, func(inv *invitationdomain.Invitation) {
			inv.ExpiresAt = time.Now().Add(-1 * time.Hour)
		})
		result, err := svc.ValidateToken(context.Background(), seeded.InviteToken)
		assert.ErrorIs(t, err, invitationdomain.ErrInvitationExpired)
		assert.Nil(t, result)
	})
}

func TestService_Accept(t *testing.T) {
	svc, repo := setupService(t)

	t.Run("accept pending", func(t *testing.T) {
		seeded := seedInvitation(repo)
		result, err := svc.Accept(context.Background(), seeded.InviteToken, "user@example.com", "testuser", "password123")
		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, invitationdomain.InvitationStatusUsed, result.Status)
	})

	t.Run("already accepted", func(t *testing.T) {
		seeded := seedInvitation(repo, func(inv *invitationdomain.Invitation) {
			inv.Status = invitationdomain.InvitationStatusUsed
		})
		result, err := svc.Accept(context.Background(), seeded.InviteToken, "user@example.com", "testuser", "password123")
		assert.ErrorIs(t, err, invitationdomain.ErrInvitationUsed)
		assert.Nil(t, result)
	})

	t.Run("expired", func(t *testing.T) {
		seeded := seedInvitation(repo, func(inv *invitationdomain.Invitation) {
			inv.ExpiresAt = time.Now().Add(-1 * time.Hour)
		})
		result, err := svc.Accept(context.Background(), seeded.InviteToken, "user@example.com", "testuser", "password123")
		assert.ErrorIs(t, err, invitationdomain.ErrInvitationExpired)
		assert.Nil(t, result)
	})

	t.Run("not found", func(t *testing.T) {
		result, err := svc.Accept(context.Background(), "nonexistent-token", "user@example.com", "testuser", "password123")
		assert.ErrorIs(t, err, invitationdomain.ErrInvitationNotFound)
		assert.Nil(t, result)
	})
}
