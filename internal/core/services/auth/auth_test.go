package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
)

// ---------------------------------------------------------------------------
// TestService_Register
// ---------------------------------------------------------------------------

func TestService_Register(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterRequest
		setup   func(*testdata.MockUserRepo)
		wantErr error
	}{
		{
			name: "valid registration with new org",
			req: RegisterRequest{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "Test",
				LastName:  "User",
				Username:  "testuser",
				OrgName:   "Test Org",
			},
			setup:   func(u *testdata.MockUserRepo) {},
			wantErr: nil,
		},
		{
			name: "duplicate email",
			req: RegisterRequest{
				Email:    "existing@test.com",
				Password: "password123",
			},
			setup: func(u *testdata.MockUserRepo) {
				user := authdomain.NewUser("existing@test.com", "", "Existing", "User", "hash")
				_ = u.Add(context.Background(), user)
			},
			wantErr: ErrEmailExists,
		},
		{
			name: "weak password",
			req: RegisterRequest{
				Email:    "test@example.com",
				Password: "short",
			},
			setup:   func(u *testdata.MockUserRepo) {},
			wantErr: authdomain.ErrWeakPassword,
		},
		{
			name: "invalid email format",
			req: RegisterRequest{
				Email:    "not-an-email",
				Password: "password123",
			},
			setup:   func(u *testdata.MockUserRepo) {},
			wantErr: authdomain.ErrInvalidEmail,
		},
		{
			name: "registration without org",
			req: RegisterRequest{
				Email:     "noorg@example.com",
				Password:  "password123",
				FirstName: "No",
				LastName:  "Org",
			},
			setup:   func(u *testdata.MockUserRepo) {},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &testdata.MockUserRepo{}
			orgRepo := &testdata.MockOrgRepo{}
			tokenSvc := &testdata.MockTokenService{}
			pwHasher := &testdata.MockPasswordHasher{}
			refreshRepo := &testdata.MockRefreshTokenRepo{}
			tt.setup(userRepo)

			svc := NewService(userRepo, orgRepo, tokenSvc, pwHasher, refreshRepo)
			resp, err := svc.Register(context.Background(), tt.req)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.NotEmpty(t, resp.User.ID)
				assert.True(t, resp.User.IsActive)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestService_Login
// ---------------------------------------------------------------------------

func TestService_Login(t *testing.T) {
	tests := []struct {
		name    string
		req     LoginRequest
		setup   func(*testdata.MockUserRepo)
		wantErr error
	}{
		{
			name: "valid credentials",
			req: LoginRequest{
				Identifier: "valid@test.com",
				Password:   "correctpassword",
			},
			setup: func(u *testdata.MockUserRepo) {
				user := authdomain.User{
					ID:           uuid.New(),
					Email:        "valid@test.com",
					Username:     "validuser",
					PasswordHash: "hashed:correctpassword",
					IsActive:     true,
				}
				u.Users = map[uuid.UUID]*authdomain.User{user.ID: &user}
				u.Memberships = map[uuid.UUID][]authdomain.OrganizationMembership{
					user.ID: {
						{ID: uuid.New(), UserID: user.ID, OrganizationID: uuid.New(), Role: "employee", IsActive: true},
					},
				}
			},
			wantErr: nil,
		},
		{
			name: "invalid password",
			req: LoginRequest{
				Identifier: "wrongpass@test.com",
				Password:   "wrongpassword",
			},
			setup: func(u *testdata.MockUserRepo) {
				user := authdomain.User{
					ID:           uuid.New(),
					Email:        "wrongpass@test.com",
					PasswordHash: "hashed:correctpassword",
					IsActive:     true,
				}
				u.Users = map[uuid.UUID]*authdomain.User{user.ID: &user}
			},
			wantErr: ErrInvalidCreds,
		},
		{
			name: "inactive user",
			req: LoginRequest{
				Identifier: "inactive@test.com",
				Password:   "somepassword",
			},
			setup: func(u *testdata.MockUserRepo) {
				user := authdomain.User{
					ID:           uuid.New(),
					Email:        "inactive@test.com",
					PasswordHash: "hashed:somepassword",
					IsActive:     false,
				}
				u.Users = map[uuid.UUID]*authdomain.User{user.ID: &user}
			},
			wantErr: ErrAccountDeactivated,
		},
		{
			name: "nonexistent email",
			req: LoginRequest{
				Identifier: "nonexistent@test.com",
				Password:   "somepassword",
			},
			setup:   func(u *testdata.MockUserRepo) {},
			wantErr: ErrInvalidCreds,
		},
		{
			name: "login with username",
			req: LoginRequest{
				Identifier: "testloginuser",
				Password:   "mypassword",
			},
			setup: func(u *testdata.MockUserRepo) {
				user := authdomain.User{
					ID:           uuid.New(),
					Email:        "loginuser@test.com",
					Username:     "testloginuser",
					PasswordHash: "hashed:mypassword",
					IsActive:     true,
				}
				u.Users = map[uuid.UUID]*authdomain.User{user.ID: &user}
				u.Memberships = map[uuid.UUID][]authdomain.OrganizationMembership{
					user.ID: {
						{ID: uuid.New(), UserID: user.ID, OrganizationID: uuid.New(), Role: "employee", IsActive: true},
					},
				}
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &testdata.MockUserRepo{}
			orgRepo := &testdata.MockOrgRepo{}
			tokenSvc := &testdata.MockTokenService{}
			pwHasher := &testdata.MockPasswordHasher{}
			refreshRepo := &testdata.MockRefreshTokenRepo{}
			tt.setup(userRepo)

			svc := NewService(userRepo, orgRepo, tokenSvc, pwHasher, refreshRepo)
			resp, err := svc.Login(context.Background(), tt.req)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.NotEmpty(t, resp.Token)
				assert.NotEmpty(t, resp.RefreshToken)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestService_Refresh
// ---------------------------------------------------------------------------

func TestService_Refresh(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		setup   func(*testdata.MockUserRepo, *testdata.MockRefreshTokenRepo)
		wantErr error
	}{
		{
			name:  "valid refresh token",
			token: "valid-refresh-token",
			setup: func(u *testdata.MockUserRepo, r *testdata.MockRefreshTokenRepo) {
				userID := uuid.New()
				orgID := uuid.New()
				u.Memberships = map[uuid.UUID][]authdomain.OrganizationMembership{
					userID: {
						{ID: uuid.New(), UserID: userID, OrganizationID: orgID, Role: "employee", IsActive: true},
					},
				}
				_ = u.Add(context.Background(), &authdomain.User{
					ID:       userID,
					Email:    "refreshtest@test.com",
					Username: "refreshtest",
					IsActive: true,
				})
				r.Tokens = map[string]*ports.RefreshToken{
					"hashed-valid-refresh-token": {
						UserID:         userID,
						OrganizationID: orgID,
						Hash:           "hashed-valid-refresh-token",
						ExpiresAt:      time.Now().Add(24 * time.Hour),
						CreatedAt:      time.Now(),
					},
				}
			},
			wantErr: nil,
		},
		{
			name:    "nonexistent token",
			token:   "nonexistent-token",
			setup:   func(u *testdata.MockUserRepo, r *testdata.MockRefreshTokenRepo) {},
			wantErr: ErrInvalidCreds,
		},
		{
			name:  "user not found for token",
			token: "orphaned-token",
			setup: func(u *testdata.MockUserRepo, r *testdata.MockRefreshTokenRepo) {
				r.Tokens = map[string]*ports.RefreshToken{
					"hashed-orphaned-token": {
						UserID:         uuid.New(),
						OrganizationID: uuid.New(),
						Hash:           "hashed-orphaned-token",
						ExpiresAt:      time.Now().Add(24 * time.Hour),
					},
				}
			},
			wantErr: ErrUserNotFound,
		},
		{
			name:  "replay of already-rotated token",
			token: "replayed-token",
			setup: func(u *testdata.MockUserRepo, r *testdata.MockRefreshTokenRepo) {
				now := time.Now()
				rotatedAt := now.Add(-time.Minute)
				r.Tokens = map[string]*ports.RefreshToken{
					"hashed-replayed-token": {
						UserID:         uuid.New(),
						OrganizationID: uuid.New(),
						Hash:           "hashed-replayed-token",
						ExpiresAt:      now.Add(24 * time.Hour),
						CreatedAt:      now.Add(-2 * time.Hour),
						FamilyID:       uuid.New(),
						RotatedAt:      &rotatedAt,
					},
				}
			},
			wantErr: ErrTokenReuse,
		},
		{
			name:  "replay of revoked token",
			token: "revoked-token",
			setup: func(u *testdata.MockUserRepo, r *testdata.MockRefreshTokenRepo) {
				now := time.Now()
				revokedAt := now.Add(-time.Minute)
				r.Tokens = map[string]*ports.RefreshToken{
					"hashed-revoked-token": {
						UserID:         uuid.New(),
						OrganizationID: uuid.New(),
						Hash:           "hashed-revoked-token",
						ExpiresAt:      now.Add(24 * time.Hour),
						CreatedAt:      now.Add(-2 * time.Hour),
						FamilyID:       uuid.New(),
						RevokedAt:      &revokedAt,
					},
				}
			},
			wantErr: ErrTokenReuse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &testdata.MockUserRepo{}
			orgRepo := &testdata.MockOrgRepo{}
			tokenSvc := &testdata.MockTokenService{}
			pwHasher := &testdata.MockPasswordHasher{}
			refreshRepo := &testdata.MockRefreshTokenRepo{}
			tt.setup(userRepo, refreshRepo)

			svc := NewService(userRepo, orgRepo, tokenSvc, pwHasher, refreshRepo)
			resp, err := svc.Refresh(context.Background(), tt.token)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.NotEmpty(t, resp.Token)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestService_Refresh_ReplayRevokesFamily
// ---------------------------------------------------------------------------

// A replayed rotated token must revoke the ENTIRE family — including the
// successor issued by the original rotation — not just the replayed hash.
// The replayed token itself is tombstoned too, and tokens in other families
// are left untouched (RevokeFamily is scoped to the right family).
func TestService_Refresh_ReplayRevokesFamily(t *testing.T) {
	userRepo := &testdata.MockUserRepo{}
	orgRepo := &testdata.MockOrgRepo{}
	tokenSvc := &testdata.MockTokenService{}
	pwHasher := &testdata.MockPasswordHasher{}
	refreshRepo := &testdata.MockRefreshTokenRepo{}

	userID := uuid.New()
	orgID := uuid.New()
	familyID := uuid.New()
	otherFamilyID := uuid.New()
	now := time.Now()
	rotatedAt := now.Add(-time.Minute)

	// Token A was already rotated; token B is its successor (same family).
	// Token C belongs to a completely different family and must be untouched.
	refreshRepo.Tokens = map[string]*ports.RefreshToken{
		"hashed-token-a": {
			UserID:         userID,
			OrganizationID: orgID,
			Hash:           "hashed-token-a",
			ExpiresAt:      now.Add(24 * time.Hour),
			CreatedAt:      now.Add(-2 * time.Hour),
			FamilyID:       familyID,
			RotatedAt:      &rotatedAt,
		},
		"hashed-token-b": {
			UserID:         userID,
			OrganizationID: orgID,
			Hash:           "hashed-token-b",
			ExpiresAt:      now.Add(7 * 24 * time.Hour),
			CreatedAt:      now,
			FamilyID:       familyID,
		},
		"hashed-token-c": {
			UserID:         userID,
			OrganizationID: orgID,
			Hash:           "hashed-token-c",
			ExpiresAt:      now.Add(7 * 24 * time.Hour),
			CreatedAt:      now,
			FamilyID:       otherFamilyID,
		},
	}

	svc := NewService(userRepo, orgRepo, tokenSvc, pwHasher, refreshRepo)

	// Replaying the rotated token A must surface as ErrTokenReuse...
	resp, err := svc.Refresh(context.Background(), "token-a")
	assert.ErrorIs(t, err, ErrTokenReuse)
	assert.Nil(t, resp)

	// ...and must have tombstoned the whole family:
	sibling := refreshRepo.Tokens["hashed-token-b"]
	require.NotNil(t, sibling, "successor token should still exist in store")
	require.NotNil(t, sibling.RevokedAt, "successor token should be revoked with its family")

	replayed := refreshRepo.Tokens["hashed-token-a"]
	require.NotNil(t, replayed)
	require.NotNil(t, replayed.RevokedAt, "replayed token itself should be tombstoned too")

	// A token in a different family is unaffected (right-family scoping).
	other := refreshRepo.Tokens["hashed-token-c"]
	require.NotNil(t, other)
	assert.Nil(t, other.RevokedAt, "token in a different family must not be revoked")
	assert.Nil(t, other.RotatedAt)
}

// ---------------------------------------------------------------------------
// TestService_Refresh_RotateHappyPath
// ---------------------------------------------------------------------------

// The atomic rotate happy path: the old row is marked rotated, the successor
// is inserted with the SAME family_id, and a fresh token pair is returned.
func TestService_Refresh_RotateHappyPath(t *testing.T) {
	userRepo := &testdata.MockUserRepo{}
	orgRepo := &testdata.MockOrgRepo{}
	tokenSvc := &testdata.MockTokenService{}
	pwHasher := &testdata.MockPasswordHasher{}
	refreshRepo := &testdata.MockRefreshTokenRepo{}

	userID := uuid.New()
	orgID := uuid.New()
	familyID := uuid.New()
	now := time.Now()

	refreshRepo.Tokens = map[string]*ports.RefreshToken{
		"hashed-token-a": {
			UserID:         userID,
			OrganizationID: orgID,
			Hash:           "hashed-token-a",
			ExpiresAt:      now.Add(24 * time.Hour),
			CreatedAt:      now.Add(-2 * time.Hour),
			FamilyID:       familyID,
		},
	}
	userRepo.Memberships = map[uuid.UUID][]authdomain.OrganizationMembership{
		userID: {
			{ID: uuid.New(), UserID: userID, OrganizationID: orgID, Role: "employee", IsActive: true},
		},
	}
	_ = userRepo.Add(context.Background(), &authdomain.User{
		ID:       userID,
		Email:    "rotate-happy@test.com",
		Username: "rotatehappy",
		IsActive: true,
	})

	svc := NewService(userRepo, orgRepo, tokenSvc, pwHasher, refreshRepo)
	resp, err := svc.Refresh(context.Background(), "token-a")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEqual(t, "token-a", resp.RefreshToken, "refresh must issue a new token")

	// Old row: rotated, NOT revoked (happy path never tombstones).
	old := refreshRepo.Tokens["hashed-token-a"]
	require.NotNil(t, old)
	require.NotNil(t, old.RotatedAt, "old token must be marked rotated")
	assert.Nil(t, old.RevokedAt, "old token must not be revoked on the happy path")

	// Successor: inserted with the same family id (mock token service always
	// issues "mock-refresh-token", so the successor hash is deterministic).
	succ := refreshRepo.Tokens["hashed-mock-refresh-token"]
	require.NotNil(t, succ, "successor token must be inserted during rotation")
	assert.Equal(t, familyID, succ.FamilyID, "successor must inherit the family id")
	assert.Equal(t, userID, succ.UserID)
}

// ---------------------------------------------------------------------------
// TestService_Refresh_MidRotateFailure_NoPartialState
// ---------------------------------------------------------------------------

// If the repository fails mid-rotation, the service must surface the error and
// leave NO partial state behind: the old token is neither rotated nor revoked
// (the real repository rolls its transaction back — no window where the old
// token is consumed without a successor).
func TestService_Refresh_MidRotateFailure_NoPartialState(t *testing.T) {
	userRepo := &testdata.MockUserRepo{}
	orgRepo := &testdata.MockOrgRepo{}
	tokenSvc := &testdata.MockTokenService{}
	pwHasher := &testdata.MockPasswordHasher{}
	refreshRepo := &testdata.MockRefreshTokenRepo{}

	userID := uuid.New()
	orgID := uuid.New()
	now := time.Now()
	refreshRepo.Tokens = map[string]*ports.RefreshToken{
		"hashed-token-a": {
			UserID:         userID,
			OrganizationID: orgID,
			Hash:           "hashed-token-a",
			ExpiresAt:      now.Add(24 * time.Hour),
			CreatedAt:      now.Add(-2 * time.Hour),
			FamilyID:       uuid.New(),
		},
	}
	// Repo fails mid-rotation (e.g. successor insert violates a constraint and
	// the tx rolls back).
	refreshRepo.RotateErr = errors.New("simulated mid-rotate failure")

	svc := NewService(userRepo, orgRepo, tokenSvc, pwHasher, refreshRepo)
	resp, err := svc.Refresh(context.Background(), "token-a")
	require.Error(t, err)
	assert.Nil(t, resp, "no token pair may be issued when rotation fails")

	old := refreshRepo.Tokens["hashed-token-a"]
	require.NotNil(t, old)
	assert.Nil(t, old.RotatedAt, "old token must NOT be rotated when rotation fails")
	assert.Nil(t, old.RevokedAt, "old token must NOT be revoked when rotation fails")
	assert.Empty(t, refreshRepo.Tokens["hashed-mock-refresh-token"], "no successor may exist when rotation fails")
}

// ---------------------------------------------------------------------------
// TestService_Bootstrap
// ---------------------------------------------------------------------------

func TestService_Bootstrap(t *testing.T) {
	tests := []struct {
		name    string
		req     BootstrapRequest
		setup   func(*testdata.MockUserRepo)
		wantErr error
	}{
		{
			name: "bootstrap when no users exist",
			req: BootstrapRequest{
				Email:    "admin@example.com",
				Password: "password123",
				OrgName:  "Hourglass",
				Username: "admin",
			},
			setup:   func(u *testdata.MockUserRepo) {},
			wantErr: nil,
		},
		{
			name: "bootstrap when users exist",
			req: BootstrapRequest{
				Email:    "admin2@example.com",
				Password: "password123",
				OrgName:  "Hourglass2",
			},
			setup: func(u *testdata.MockUserRepo) {
				user := authdomain.NewUser("existing@example.com", "", "Existing", "User", "hash")
				_ = u.Add(context.Background(), user)
			},
			wantErr: ErrEmailExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &testdata.MockUserRepo{}
			orgRepo := &testdata.MockOrgRepo{}
			tokenSvc := &testdata.MockTokenService{}
			pwHasher := &testdata.MockPasswordHasher{}
			refreshRepo := &testdata.MockRefreshTokenRepo{}
			tt.setup(userRepo)

			svc := NewService(userRepo, orgRepo, tokenSvc, pwHasher, refreshRepo)
			resp, err := svc.Bootstrap(context.Background(), tt.req)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.NotEmpty(t, resp.Token)
				assert.NotEmpty(t, resp.RefreshToken)
			}
		})
	}
}
