package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stefanoprivitera/hourglass/internal/auth"
	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// realRepoFixture creates real PostgreSQL-backed repos and a real *Service.
// Each subtest gets a fresh schema. Call once per subtest.
func realRepoFixture(t *testing.T, pool *pgxpool.Pool) *Service {
	t.Helper()
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	userRepo := postgres.NewUserRepository(pool)
	orgRepo := postgres.NewOrganizationRepository(pool)
	refreshRepo := postgres.NewRefreshTokenRepository(pool)
	jwtSvc := auth.NewService("dev-secret-change-in-production")
	tokenSvc := auth.NewTokenService(jwtSvc)
	pwHasher := auth.NewPasswordHasher()

	return NewService(userRepo, orgRepo, tokenSvc, pwHasher, refreshRepo)
}

func TestAuthIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("RegisterWithRealDB", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		req := RegisterRequest{
			Email:     uuid.New().String() + "@test.com",
			Password:  "SecurePass123!",
			FirstName: "John",
			LastName:  "Doe",
			Username:  "john_" + uuid.New().String()[:8],
			OrgName:   "Acme Corp",
		}

		resp, err := svc.Register(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.User.ID)
		assert.Equal(t, "John Doe", resp.User.Name)
		assert.True(t, resp.User.IsActive)
		assert.NotEmpty(t, resp.Membership.ID)
		assert.Equal(t, "manager", resp.Membership.Role)
		assert.NotEmpty(t, resp.Organization.ID)
		assert.Equal(t, "Acme Corp", resp.Organization.Name)
	})

	t.Run("RegisterWithExistingOrg", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Existing Org", "existing-org-"+uuid.New().String()[:8], time.Now(), time.Now())
		require.NoError(t, err)

		resp, err := svc.Register(context.Background(), RegisterRequest{
			Email:     uuid.New().String() + "@test.com",
			Password:  "SecurePass123!",
			FirstName: "Jane",
			LastName:  "Smith",
			Username:  "jane_" + uuid.New().String()[:8],
			OrgID:     orgID.String(),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, orgID.String(), resp.Membership.OrganizationID)
		assert.Equal(t, orgID.String(), resp.Organization.ID)
	})

	t.Run("RegisterDuplicateEmail", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		email := uuid.New().String() + "@test.com"
		username := "dup_" + uuid.New().String()[:8]

		_, err := svc.Register(context.Background(), RegisterRequest{
			Email:     email,
			Password:  "SecurePass123!",
			FirstName: "First",
			LastName:  "User",
			Username:  username,
			OrgName:   "Org Alpha",
		})
		require.NoError(t, err)

		_, err = svc.Register(context.Background(), RegisterRequest{
			Email:     email,
			Password:  "SecurePass123!",
			FirstName: "Second",
			LastName:  "User",
			Username:  "diff_" + uuid.New().String()[:8],
			OrgName:   "Org Beta",
		})
		assert.ErrorIs(t, err, ErrEmailExists)
	})

	t.Run("LoginWithRealDB", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		email := uuid.New().String() + "@test.com"
		password := "SecurePass123!"
		username := "login_" + uuid.New().String()[:8]

		regResp, err := svc.Register(context.Background(), RegisterRequest{
			Email:     email,
			Password:  password,
			FirstName: "Login",
			LastName:  "Test",
			Username:  username,
			OrgName:   "Login Corp",
		})
		require.NoError(t, err)
		require.NotNil(t, regResp)

		loginResp, err := svc.Login(context.Background(), LoginRequest{
			Identifier: email,
			Password:   password,
		})
		require.NoError(t, err)
		require.NotNil(t, loginResp)
		assert.NotEmpty(t, loginResp.Token)
		assert.NotEmpty(t, loginResp.RefreshToken)
		assert.Equal(t, regResp.User.ID, loginResp.User.ID)
		assert.Equal(t, regResp.Membership.OrganizationID, loginResp.Membership.OrganizationID)

		refreshHash := auth.HashRefreshToken(loginResp.RefreshToken)
		refreshRepo := postgres.NewRefreshTokenRepository(pool)
		storedToken, err := refreshRepo.FindByHash(context.Background(), refreshHash)
		require.NoError(t, err)
		require.NotNil(t, storedToken)
		assert.Equal(t, refreshHash, storedToken.Hash)
	})

	t.Run("LoginWithWrongPassword", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		email := uuid.New().String() + "@test.com"
		username := "wrong_" + uuid.New().String()[:8]

		_, err := svc.Register(context.Background(), RegisterRequest{
			Email:     email,
			Password:  "SecurePass123!",
			FirstName: "Wrong",
			LastName:  "Pass",
			Username:  username,
			OrgName:   "Wrong Corp",
		})
		require.NoError(t, err)

		_, err = svc.Login(context.Background(), LoginRequest{
			Identifier: email,
			Password:   "wrongpassword",
		})
		assert.ErrorIs(t, err, ErrInvalidCreds)
	})

	t.Run("GetProfileWithMembership", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		email := uuid.New().String() + "@test.com"
		username := "prof_" + uuid.New().String()[:8]

		regResp, err := svc.Register(context.Background(), RegisterRequest{
			Email:     email,
			Password:  "SecurePass123!",
			FirstName: "Profile",
			LastName:  "Test",
			Username:  username,
			OrgName:   "Profile Corp",
		})
		require.NoError(t, err)

		userID := uuid.MustParse(regResp.User.ID)
		orgID := uuid.MustParse(regResp.Membership.OrganizationID)

		profile, err := svc.GetProfile(context.Background(), userID, orgID)
		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, regResp.User.ID, profile.User.ID)
		assert.Equal(t, "manager", profile.Membership.Role)
		assert.Equal(t, regResp.Membership.OrganizationID, profile.Membership.OrganizationID)
	})

	t.Run("RefreshTokenRotation", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		email := uuid.New().String() + "@test.com"
		password := "SecurePass123!"
		username := "refresh_" + uuid.New().String()[:8]

		_, err := svc.Register(context.Background(), RegisterRequest{
			Email:     email,
			Password:  password,
			FirstName: "Refresh",
			LastName:  "Test",
			Username:  username,
			OrgName:   "Refresh Corp",
		})
		require.NoError(t, err)

		loginResp, err := svc.Login(context.Background(), LoginRequest{
			Identifier: email,
			Password:   password,
		})
		require.NoError(t, err)
		require.NotNil(t, loginResp)

		oldRefreshToken := loginResp.RefreshToken
		oldHash := auth.HashRefreshToken(oldRefreshToken)

		refreshResp, err := svc.Refresh(context.Background(), oldRefreshToken)
		require.NoError(t, err)
		require.NotNil(t, refreshResp)
		assert.NotEmpty(t, refreshResp.Token)
		assert.NotEmpty(t, refreshResp.RefreshToken)
		assert.NotEqual(t, oldRefreshToken, refreshResp.RefreshToken, "refresh token should be rotated")

		refreshRepo := postgres.NewRefreshTokenRepository(pool)
		storedOld, err := refreshRepo.FindByHash(context.Background(), oldHash)
		require.NoError(t, err)
		assert.Nil(t, storedOld, "old refresh token hash should be revoked after rotation")

		newHash := auth.HashRefreshToken(refreshResp.RefreshToken)
		storedNew, err := refreshRepo.FindByHash(context.Background(), newHash)
		require.NoError(t, err)
		require.NotNil(t, storedNew)
		assert.Equal(t, newHash, storedNew.Hash)
	})

	t.Run("RefreshWithRevokedToken", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		email := uuid.New().String() + "@test.com"
		password := "SecurePass123!"
		username := "revoke_" + uuid.New().String()[:8]

		_, err := svc.Register(context.Background(), RegisterRequest{
			Email:     email,
			Password:  password,
			FirstName: "Revoke",
			LastName:  "Test",
			Username:  username,
			OrgName:   "Revoke Corp",
		})
		require.NoError(t, err)

		loginResp, err := svc.Login(context.Background(), LoginRequest{
			Identifier: email,
			Password:   password,
		})
		require.NoError(t, err)

		_, err = svc.Refresh(context.Background(), loginResp.RefreshToken)
		require.NoError(t, err)

		_, err = svc.Refresh(context.Background(), loginResp.RefreshToken)
		require.Error(t, err)
	})

	t.Run("RefreshTokenReuse_ReturnsErrTokenReuse_AndRevokesFamily", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		email := uuid.New().String() + "@test.com"
		password := "SecurePass123!"
		username := "reuse_" + uuid.New().String()[:8]

		_, err := svc.Register(context.Background(), RegisterRequest{
			Email:     email,
			Password:  password,
			FirstName: "Reuse",
			LastName:  "Test",
			Username:  username,
			OrgName:   "Reuse Corp",
		})
		require.NoError(t, err)

		loginResp, err := svc.Login(context.Background(), LoginRequest{
			Identifier: email,
			Password:   password,
		})
		require.NoError(t, err)

		// Legitimate rotation: token A -> token B (same family).
		refreshResp, err := svc.Refresh(context.Background(), loginResp.RefreshToken)
		require.NoError(t, err)
		require.NotEqual(t, loginResp.RefreshToken, refreshResp.RefreshToken)

		// Replaying the rotated token A is detected as reuse...
		_, err = svc.Refresh(context.Background(), loginResp.RefreshToken)
		require.ErrorIs(t, err, ErrTokenReuse, "replay of rotated token should surface ErrTokenReuse")

		// ...and the whole family dies: the successor B is revoked too.
		_, err = svc.Refresh(context.Background(), refreshResp.RefreshToken)
		require.ErrorIs(t, err, ErrTokenReuse, "successor token should be revoked with its family")
	})

	t.Run("GetProfileWithoutOrg", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		email := uuid.New().String() + "@test.com"
		username := "noor_" + uuid.New().String()[:8]

		regResp, err := svc.Register(context.Background(), RegisterRequest{
			Email:     email,
			Password:  "SecurePass123!",
			FirstName: "No",
			LastName:  "Org",
			Username:  username,
		})
		require.NoError(t, err)

		userID := uuid.MustParse(regResp.User.ID)
		profile, err := svc.GetProfile(context.Background(), userID, uuid.Nil)
		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, regResp.User.ID, profile.User.ID)
		assert.Empty(t, profile.Membership.ID)
	})

	t.Run("RegisterFailsWithDuplicateUsername", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		email1 := uuid.New().String() + "@test.com"
		email2 := uuid.New().String() + "@test.com"
		sameUsername := "dupuser_" + uuid.New().String()[:8]

		_, err := svc.Register(context.Background(), RegisterRequest{
			Email:     email1,
			Password:  "SecurePass123!",
			FirstName: "First",
			LastName:  "User",
			Username:  sameUsername,
			OrgName:   "Org A",
		})
		require.NoError(t, err)

		_, err = svc.Register(context.Background(), RegisterRequest{
			Email:     email2,
			Password:  "SecurePass123!",
			FirstName: "Second",
			LastName:  "User",
			Username:  sameUsername,
			OrgName:   "Org B",
		})
		assert.ErrorIs(t, err, ErrUsernameExists)
	})
}

// ---------------------------------------------------------------------------
// Compile-time checks that real implementations satisfy port interfaces
// ---------------------------------------------------------------------------

var _ ports.TokenService = (*auth.TokenService)(nil)
var _ ports.PasswordHasher = (*auth.PasswordHasher)(nil)
