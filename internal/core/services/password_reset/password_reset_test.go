package password_reset

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	pwdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/password_reset"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
)

// mockUserFinder implements ports.UserFinder for testing
type mockUserFinder struct {
	userID string
	err    error
}

func (m *mockUserFinder) FindByIdentifier(ctx context.Context, identifier string) (string, error) {
	return m.userID, m.err
}

// mockPasswordResetRepo wraps testdata.MockPasswordResetRepo with overridable FindActiveByUserID
type mockPasswordResetRepo struct {
	*testdata.MockPasswordResetRepo
	findActiveFunc func(ctx context.Context, userID string) (*pwdomain.PasswordReset, error)
}

func (m *mockPasswordResetRepo) FindActiveByUserID(ctx context.Context, userID string) (*pwdomain.PasswordReset, error) {
	if m.findActiveFunc != nil {
		return m.findActiveFunc(ctx, userID)
	}
	return m.MockPasswordResetRepo.FindActiveByUserID(ctx, userID)
}

func newService() (*Service, *mockPasswordResetRepo, *testdata.MockUserRepo, *mockUserFinder) {
	pwRepo := &mockPasswordResetRepo{
		MockPasswordResetRepo: &testdata.MockPasswordResetRepo{},
	}
	userRepo := &testdata.MockUserRepo{}
	userFinder := &mockUserFinder{}
	hasher := &testdata.MockPasswordHasher{}
	tokenSvc := &testdata.MockTokenService{}
	refreshRepo := &testdata.MockRefreshTokenRepo{}
	svc := NewService(pwRepo, userRepo, userFinder, hasher, tokenSvc, refreshRepo)
	return svc, pwRepo, userRepo, userFinder
}

func seedUser(userRepo *testdata.MockUserRepo, uid uuid.UUID) {
	_ = userRepo.Add(context.Background(), &auth.User{
		ID:       uid,
		Email:    "user@example.com",
		Username: "testuser",
		IsActive: true,
	})
}

func TestService_RequestReset(t *testing.T) {
	t.Run("valid email", func(t *testing.T) {
		svc, _, _, userFinder := newService()
		userID := uuid.New().String()
		userFinder.userID = userID
		userFinder.err = nil

		code, expiresAt, err := svc.Request(context.Background(), "user@example.com")
		assert.NoError(t, err)
		assert.NotEmpty(t, code)
		assert.False(t, expiresAt.IsZero())
		assert.True(t, expiresAt.After(time.Now()))
	})

	t.Run("nonexistent email", func(t *testing.T) {
		svc, _, _, userFinder := newService()
		userFinder.userID = ""
		userFinder.err = pwdomain.ErrUserNotFound

		code, expiresAt, err := svc.Request(context.Background(), "unknown@example.com")
		assert.ErrorIs(t, err, pwdomain.ErrUserNotFound)
		assert.Empty(t, code)
		assert.True(t, expiresAt.IsZero())
	})
}

func TestService_Verify(t *testing.T) {
	hasher := &testdata.MockPasswordHasher{}
	userID := uuid.New()

	// Pre-seed user data needed by all subtests that do verification
	sharedSvc, pwRepo, userRepo, userFinder := newService()
	seedUser(userRepo, userID)
	userFinder.userID = userID.String()
	userFinder.err = nil

	t.Run("valid token", func(t *testing.T) {
		pwRepo.findActiveFunc = func(ctx context.Context, uid string) (*pwdomain.PasswordReset, error) {
			hash, _ := hasher.Hash("123456")
			return &pwdomain.PasswordReset{
				ID:        uuid.New(),
				UserID:    uuid.MustParse(uid),
				CodeHash:  hash,
				ExpiresAt: time.Now().Add(1 * time.Hour),
				CreatedAt: time.Now(),
			}, nil
		}

		err := sharedSvc.Verify(context.Background(), "user@example.com", "123456", "newpassword")
		assert.NoError(t, err)
	})

	t.Run("invalid code", func(t *testing.T) {
		pwRepo.findActiveFunc = func(ctx context.Context, uid string) (*pwdomain.PasswordReset, error) {
			hash, _ := hasher.Hash("999999") // Different hash
			return &pwdomain.PasswordReset{
				ID:        uuid.New(),
				UserID:    uuid.MustParse(uid),
				CodeHash:  hash,
				ExpiresAt: time.Now().Add(1 * time.Hour),
				CreatedAt: time.Now(),
			}, nil
		}

		err := sharedSvc.Verify(context.Background(), "user@example.com", "000000", "newpassword")
		assert.ErrorIs(t, err, pwdomain.ErrInvalidCode)
	})

	t.Run("expired reset", func(t *testing.T) {
		pwRepo.findActiveFunc = func(ctx context.Context, uid string) (*pwdomain.PasswordReset, error) {
			return nil, pwdomain.ErrResetNotFound
		}

		err := sharedSvc.Verify(context.Background(), "user@example.com", "123456", "newpassword")
		assert.ErrorIs(t, err, pwdomain.ErrResetExpired)
	})
}

func TestService_Verify_UnknownIdentifier(t *testing.T) {
	svc, _, _, userFinder := newService()
	userFinder.userID = ""
	userFinder.err = pwdomain.ErrUserNotFound

	err := svc.Verify(context.Background(), "unknown@example.com", "123456", "newpassword")
	assert.ErrorIs(t, err, pwdomain.ErrUserNotFound)
}

func TestService_Verify_NoActiveReset(t *testing.T) {
	svc, pwRepo, userRepo, userFinder := newService()
	userID := uuid.New()
	seedUser(userRepo, userID)
	userFinder.userID = userID.String()
	userFinder.err = nil

	pwRepo.findActiveFunc = func(ctx context.Context, uid string) (*pwdomain.PasswordReset, error) {
		return nil, pwdomain.ErrResetNotFound
	}

	err := svc.Verify(context.Background(), "user@example.com", "123456", "newpassword")
	assert.ErrorIs(t, err, pwdomain.ErrResetExpired)
}

func TestService_RequestAndVerify_Integration(t *testing.T) {
	svc, pwRepo, userRepo, userFinder := newService()
	userID := uuid.New()
	seedUser(userRepo, userID)
	userFinder.userID = userID.String()
	userFinder.err = nil

	// Request a reset
	code, expiresAt, err := svc.Request(context.Background(), "user@example.com")
	assert.NoError(t, err)
	assert.NotEmpty(t, code)
	assert.False(t, expiresAt.IsZero())

	// Override FindActiveByUserID to return a reset with the hashed code
	hasher := &testdata.MockPasswordHasher{}
	hash, _ := hasher.Hash(code)

	pwRepo.findActiveFunc = func(ctx context.Context, uid string) (*pwdomain.PasswordReset, error) {
		return &pwdomain.PasswordReset{
			ID:        uuid.New(),
			UserID:    userID,
			CodeHash:  hash,
			ExpiresAt: time.Now().Add(1 * time.Hour),
			CreatedAt: time.Now(),
		}, nil
	}

	// Verify with the correct code
	err = svc.Verify(context.Background(), "user@example.com", code, "newpassword")
	assert.NoError(t, err)
}
