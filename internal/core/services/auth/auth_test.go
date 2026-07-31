package auth

import (
	"context"
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
func TestService_Refresh_ReplayRevokesFamily(t *testing.T) {
	userRepo := &testdata.MockUserRepo{}
	orgRepo := &testdata.MockOrgRepo{}
	tokenSvc := &testdata.MockTokenService{}
	pwHasher := &testdata.MockPasswordHasher{}
	refreshRepo := &testdata.MockRefreshTokenRepo{}

	userID := uuid.New()
	orgID := uuid.New()
	familyID := uuid.New()
	now := time.Now()
	rotatedAt := now.Add(-time.Minute)

	// Token A was already rotated; token B is its successor (same family).
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
	}

	svc := NewService(userRepo, orgRepo, tokenSvc, pwHasher, refreshRepo)

	// Replaying the rotated token A must surface as ErrTokenReuse...
	resp, err := svc.Refresh(context.Background(), "token-a")
	assert.ErrorIs(t, err, ErrTokenReuse)
	assert.Nil(t, resp)

	// ...and must have tombstoned the successor B too.
	sibling := refreshRepo.Tokens["hashed-token-b"]
	require.NotNil(t, sibling, "successor token should still exist in store")
	require.NotNil(t, sibling.RevokedAt, "successor token should be revoked with its family")
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
