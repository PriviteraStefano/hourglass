package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	"github.com/stefanoprivitera/hourglass/internal/auth"
	authsvc "github.com/stefanoprivitera/hourglass/internal/core/services/auth"
	invitationsvc "github.com/stefanoprivitera/hourglass/internal/core/services/invitation"
	passwordresetsvc "github.com/stefanoprivitera/hourglass/internal/core/services/password_reset"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// integrationFixture holds wired services, server, and client for each test.
// ---------------------------------------------------------------------------

type integrationFixture struct {
	server      *httptest.Server
	client      *http.Client
	serverURL   string
	authService *authsvc.Service
	pool        *pgxpool.Pool
}

func newFixture(t *testing.T, pool *pgxpool.Pool) *integrationFixture {
	t.Helper()

	mux := http.NewServeMux()

	jwtSecret := "test-secret"
	authSvc := auth.NewService(jwtSecret)

	// Auth repos
	userRepo := postgres.NewUserRepository(pool)
	orgRepo := postgres.NewOrganizationRepository(pool)
	passwordHasher := auth.NewPasswordHasher()
	tokenService := auth.NewTokenService(authSvc)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(pool)

	// Invitation service
	invitationRepo := postgres.NewInvitationRepository(pool)
	invitationService := invitationsvc.NewService(invitationRepo)

	// Auth service + handler
	hexAuthService := authsvc.NewService(
		userRepo,
		orgRepo,
		tokenService,
		passwordHasher,
		refreshTokenRepo,
	)
	authHandler := NewAuthHandler(hexAuthService, invitationService)

	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("POST /auth/logout", authHandler.Logout)
	mux.HandleFunc("POST /auth/refresh", authHandler.Refresh)
	mux.HandleFunc("GET /auth/me", middleware.Auth(authSvc, authHandler.GetProfile))
	mux.HandleFunc("POST /auth/bootstrap", authHandler.Bootstrap)
	mux.HandleFunc("GET /auth/bootstrap-check", authHandler.BootstrapCheck)
	mux.HandleFunc("POST /auth/switch-organization", middleware.Auth(authSvc, authHandler.SwitchOrganization))
	mux.HandleFunc("GET /auth/memberships", middleware.Auth(authSvc, authHandler.GetMemberships))

	// Password reset
	passwordResetRepo := postgres.NewPasswordResetRepository(pool)
	userFinder := postgres.NewUserFinder(pool)
	passwordResetService := passwordresetsvc.NewService(
		passwordResetRepo,
		userRepo,
		userFinder,
		passwordHasher,
		auth.NewTokenService(authSvc),
		refreshTokenRepo,
	)
	passwordResetHandler := NewPasswordResetHandler(passwordResetService)
	mux.HandleFunc("POST /auth/password-reset/request", passwordResetHandler.Request)
	mux.HandleFunc("POST /auth/password-reset/verify", passwordResetHandler.Verify)

	server := httptest.NewServer(mux)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{Jar: jar}

	t.Cleanup(func() { server.Close() })

	return &integrationFixture{
		server:      server,
		client:      client,
		serverURL:   server.URL,
		authService: hexAuthService,
		pool:        pool,
	}
}

// registerAndLogin registers a user, then logs in to get auth cookies in the jar.
func (f *integrationFixture) registerAndLogin(t *testing.T, email, username, password, orgName string) *authsvc.LoginResponse {
	t.Helper()
	// First register
	regBody := `{"email":"` + email + `","username":"` + username + `","password":"` + password + `","organization_name":"` + orgName + `"}`
	regResp, err := f.client.Post(f.serverURL+"/auth/register", "application/json", strings.NewReader(regBody))
	require.NoError(t, err)
	regResp.Body.Close()
	require.Equal(t, http.StatusOK, regResp.StatusCode, "register should return 200")

	// Then login — this sets auth cookies in the jar
	return f.loginUser(t, email, password)
}

// loginUser is a helper that logs in and returns the parsed response.
func (f *integrationFixture) loginUser(t *testing.T, identifier, password string) *authsvc.LoginResponse {
	t.Helper()
	body := `{"identifier":"` + identifier + `","password":"` + password + `"}`
	resp, err := f.client.Post(f.serverURL+"/auth/login", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "login should return 200")

	// Response is wrapped in {"data": {...}}
	var wrapped struct {
		Data *authsvc.LoginResponse `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&wrapped)
	require.NoError(t, err)
	require.NotNil(t, wrapped.Data, "login response should have 'data' field")
	return wrapped.Data
}

// ---------------------------------------------------------------------------
// TestAuthIntegration — master test that starts one container per package
// and runs all sub-tests.
// ---------------------------------------------------------------------------

func TestAuthIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("GetMemberships_NoNilPointer", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newFixture(t, pool)
		email := fmt.Sprintf("memberships-test-%s@test.com", t.Name())
		_ = f.registerAndLogin(t, email, "membershipuser", "TestPass123!", "Membership Org")

		// GET /auth/memberships should return 200 with memberships (not a nil pointer)
		resp, err := f.client.Get(f.serverURL + "/auth/memberships")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode,
			"GET /auth/memberships should return 200 (no nil pointer)")

		var body struct {
			Data map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)

		innerData := body.Data
		memberships, ok := innerData["memberships"]
		assert.True(t, ok, "response should contain 'memberships' key")
		assert.NotNil(t, memberships, "memberships should not be nil")

		// Verify the memberships list is not empty
		membershipList, ok := memberships.([]interface{})
		require.True(t, ok, "memberships should be a list")
		assert.GreaterOrEqual(t, len(membershipList), 1, "should have at least 1 membership")
		t.Log("PASS: GetMemberships returned memberships without nil pointer")
	})

	t.Run("GetProfile_WithRoleAndOrgID", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newFixture(t, pool)
		email := fmt.Sprintf("profile-test-%s@test.com", t.Name())
		_ = f.registerAndLogin(t, email, "profileuser", "TestPass123!", "Profile Org")

		// GET /auth/me should return profile with non-empty role and org_id
		resp, err := f.client.Get(f.serverURL + "/auth/me")
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode,
			"GET /auth/me should return 200 with valid profile")

		var profileWrapped struct {
			Data map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&profileWrapped)
		require.NoError(t, err)

		profile := profileWrapped.Data
		user, ok := profile["user"].(map[string]interface{})
		require.True(t, ok, "response should have 'user'")

		emailVal, ok := user["email"].(string)
		require.True(t, ok)
		assert.Equal(t, email, emailVal, "user email should match")

		membership, ok := profile["membership"].(map[string]interface{})
		require.True(t, ok, "response should have 'membership'")

		role, ok := membership["role"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, role, "role should not be empty")
		assert.Equal(t, "manager", role)

		orgID, ok := membership["organization_id"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, orgID, "organization_id should not be empty")

		org, ok := profile["organization"].(map[string]interface{})
		require.True(t, ok, "response should have 'organization'")

		orgIDInOrg, ok := org["id"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, orgIDInOrg, "org id should not be empty")

		t.Logf("PASS: GetProfile returns profile with role=%s and org_id=%s", role, orgID)
	})

	t.Run("RefreshTokenRotation", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newFixture(t, pool)
		email := fmt.Sprintf("refresh-rotation-%s@test.com", t.Name())
		loginResp := f.registerAndLogin(t, email, "refreshuser", "TestPass123!", "Refresh Org")

		require.NotEmpty(t, loginResp.RefreshToken, "login should return a refresh token")
		initialRefreshToken := loginResp.RefreshToken

		// Call POST /auth/refresh — cookie jar has the refresh_token cookie from login
		resp, err := f.client.Post(f.serverURL+"/auth/refresh", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode,
			"POST /auth/refresh should return 200")

		var refreshWrapped struct {
			Data *authsvc.RefreshResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&refreshWrapped)
		require.NoError(t, err)
		require.NotNil(t, refreshWrapped.Data, "refresh response should have 'data' field")
		refreshResp := refreshWrapped.Data

		// Verify new refresh token was issued (rotation)
		assert.NotEmpty(t, refreshResp.RefreshToken,
			"refresh response should include a new refresh_token")
		assert.NotEqual(t, initialRefreshToken, refreshResp.RefreshToken,
			"new refresh token should differ from the initial one (rotation)")

		// New access token should also be present and valid
		assert.NotEmpty(t, refreshResp.Token,
			"refresh response should include a new access token")

		t.Logf("PASS: Refresh token rotation works")
	})

	t.Run("PasswordReset_CodeNotInResponse", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newFixture(t, pool)
		email := fmt.Sprintf("pwreset-test-%s@test.com", t.Name())

		// Register user (no login needed for password reset request)
		regBody := `{"email":"` + email + `","username":"pwresetuser","password":"TestPass123!","organization_name":"PWReset Org"}`
		regResp, err := f.client.Post(f.serverURL+"/auth/register", "application/json", strings.NewReader(regBody))
		require.NoError(t, err)
		regResp.Body.Close()
	require.Equal(t, http.StatusOK, regResp.StatusCode)

	// Request password reset
		body := `{"identifier":"` + email + `"}`
		resp, err := f.client.Post(f.serverURL+"/auth/password-reset/request", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var pwResetWrapped struct {
			Data map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&pwResetWrapped)
		require.NoError(t, err)

		result := pwResetWrapped.Data
		// Verify "code" field is NOT in the response
		_, hasCode := result["code"]
		assert.False(t, hasCode, "password reset response should NOT contain 'code' field")

		// Verify "message" field is present
		message, hasMessage := result["message"].(string)
		assert.True(t, hasMessage, "response should have 'message' field")
		assert.Equal(t, "reset code sent", message)

		// Verify "expires_at" field is present
		_, hasExpiresAt := result["expires_at"]
		assert.True(t, hasExpiresAt, "response should have 'expires_at' field")

		t.Log("PASS: Password reset response does not contain 'code' in body")
	})

	t.Run("GetProfile_UnknownUser", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newFixture(t, pool)

		// GET /auth/me without being authenticated should return 401
		resp, err := f.client.Get(f.serverURL + "/auth/me")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
			"GET /auth/me without authentication should return 401")
		t.Log("PASS: GetProfile without auth returns 401")
	})

	t.Run("Login_SetsAuthTokenCookie", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newFixture(t, pool)
		email := fmt.Sprintf("login-cookie-test-%s@test.com", t.Name())

		// Register user first
		regBody := `{"email":"` + email + `","username":"logincookieuser","password":"TestPass123!","organization_name":"Login Cookie Org"}`
		regResp, err := f.client.Post(f.serverURL+"/auth/register", "application/json", strings.NewReader(regBody))
		require.NoError(t, err)
		regResp.Body.Close()
		require.Equal(t, http.StatusOK, regResp.StatusCode)

		// Login and capture response cookies
		body := `{"identifier":"` + email + `","password":"TestPass123!"}`
		resp, err := f.client.Post(f.serverURL+"/auth/login", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify auth_token is set via Set-Cookie header
		hasAuthToken := false
		for _, c := range resp.Cookies() {
			if c.Name == "auth_token" && c.Value != "" {
				hasAuthToken = true
				break
			}
		}
		assert.True(t, hasAuthToken, "login response should set auth_token cookie")

		// Verify refresh_token is set via Set-Cookie header
		hasRefreshToken := false
		for _, c := range resp.Cookies() {
			if c.Name == "refresh_token" && c.Value != "" {
				hasRefreshToken = true
				break
			}
		}
		assert.True(t, hasRefreshToken, "login response should set refresh_token cookie")

		t.Log("PASS: Login sets both auth_token and refresh_token cookies")
	})
}
