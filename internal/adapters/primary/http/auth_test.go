package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func uniqueID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func uniqueEmail() string {
	return "test_" + uniqueID() + "@example.com"
}

func uniqueOrgName() string {
	return "Org_" + uniqueID()
}

// TestAuthHandlerIntegration is the master test for all auth handler
// integration scenarios.  It starts one PostgreSQL container for the entire
// package (via SetupPackageContainer + sync.Once) and creates/drops schemas
// per subtest for perfect isolation.
func TestAuthHandlerIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	// -----------------------------------------------------------------------
	// Registration
	// -----------------------------------------------------------------------

	t.Run("Register_WithNewOrg_Returns201WithUserData", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		body := fmt.Sprintf(`{"email":"%s","username":"user_%s","firstname":"John","lastname":"Doe","password":"password123","organization_name":"%s"}`,
			uniqueEmail(), uniqueID(), uniqueOrgName())

		resp, err := f.Client.Post(f.ServerURL+"/auth/register", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		data, ok := result["data"].(map[string]interface{})
		require.True(t, ok)

		user, ok := data["user"].(map[string]interface{})
		require.True(t, ok, "expected 'user' in register response data")
		assert.NotNil(t, user["email"], "expected user email")

		membership, ok := data["membership"].(map[string]interface{})
		require.True(t, ok)
		assert.NotEmpty(t, membership["organization_id"])
		assert.Equal(t, "employee", membership["role"])
	})

	t.Run("Register_InvalidEmail_Returns400", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		body := fmt.Sprintf(`{"email":"notanemail","password":"password123","organization_name":"%s"}`, uniqueOrgName())
		resp, err := f.Client.Post(f.ServerURL+"/auth/register", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Register_WeakPassword_Returns400", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		body := fmt.Sprintf(`{"email":"%s","password":"short","organization_name":"%s"}`, uniqueEmail(), uniqueOrgName())
		resp, err := f.Client.Post(f.ServerURL+"/auth/register", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Register_WithoutOrg_Returns201", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		body := fmt.Sprintf(`{"email":"%s","password":"password123"}`, uniqueEmail())
		resp, err := f.Client.Post(f.ServerURL+"/auth/register", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		// Register without org creates a user without membership (valid)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("Register_DuplicateEmail_Returns409", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := uniqueEmail()
		org := uniqueOrgName()

		body1 := fmt.Sprintf(`{"email":"%s","password":"password123","organization_name":"%s"}`, email, org)
		r1, err := f.Client.Post(f.ServerURL+"/auth/register", "application/json", strings.NewReader(body1))
		require.NoError(t, err)
		r1.Body.Close()
		assert.Equal(t, http.StatusCreated, r1.StatusCode)

		body2 := fmt.Sprintf(`{"email":"%s","username":"dup","password":"password123","organization_name":"%s"}`, email, org)
		r2, err := f.Client.Post(f.ServerURL+"/auth/register", "application/json", strings.NewReader(body2))
		require.NoError(t, err)
		defer r2.Body.Close()
		assert.Equal(t, http.StatusConflict, r2.StatusCode)
	})

	t.Run("Register_DuplicateUsername_Returns409", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email1 := uniqueEmail()
		email2 := uniqueEmail()
		username := "user_" + uniqueID()

		body1 := fmt.Sprintf(`{"email":"%s","username":"%s","password":"password123","organization_name":"%s"}`, email1, username, uniqueOrgName())
		r1, err := f.Client.Post(f.ServerURL+"/auth/register", "application/json", strings.NewReader(body1))
		require.NoError(t, err)
		r1.Body.Close()

		body2 := fmt.Sprintf(`{"email":"%s","username":"%s","password":"password123","organization_name":"%s"}`, email2, username, uniqueOrgName())
		r2, err := f.Client.Post(f.ServerURL+"/auth/register", "application/json", strings.NewReader(body2))
		require.NoError(t, err)
		defer r2.Body.Close()
		assert.Equal(t, http.StatusConflict, r2.StatusCode)
	})

	// -----------------------------------------------------------------------
	// Login
	// -----------------------------------------------------------------------

	t.Run("Login_ValidCredentials_Returns200WithCookies", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := uniqueEmail()
		password := "password123"

		regBody := fmt.Sprintf(`{"email":"%s","username":"user_%s","password":"%s","organization_name":"%s"}`, email, uniqueID(), password, uniqueOrgName())
		regResp, err := f.Client.Post(f.ServerURL+"/auth/register", "application/json", strings.NewReader(regBody))
		require.NoError(t, err)
		regResp.Body.Close()

		loginBody := fmt.Sprintf(`{"identifier":"%s","password":"%s"}`, email, password)
		resp, err := f.Client.Post(f.ServerURL+"/auth/login", "application/json", strings.NewReader(loginBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		hasAuthToken := false
		hasRefreshToken := false
		for _, c := range resp.Cookies() {
			if c.Name == "auth_token" && c.Value != "" {
				hasAuthToken = true
			}
			if c.Name == "refresh_token" && c.Value != "" {
				hasRefreshToken = true
			}
		}
		assert.True(t, hasAuthToken, "login should set auth_token cookie")
		assert.True(t, hasRefreshToken, "login should set refresh_token cookie")
	})

	t.Run("Login_WithUsername_Returns200", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := uniqueEmail()
		username := "user_" + uniqueID()
		password := "password123"

		regBody := fmt.Sprintf(`{"email":"%s","username":"%s","password":"%s","organization_name":"%s"}`, email, username, password, uniqueOrgName())
		regResp, err := f.Client.Post(f.ServerURL+"/auth/register", "application/json", strings.NewReader(regBody))
		require.NoError(t, err)
		regResp.Body.Close()

		loginBody := fmt.Sprintf(`{"identifier":"%s","password":"%s"}`, username, password)
		resp, err := f.Client.Post(f.ServerURL+"/auth/login", "application/json", strings.NewReader(loginBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Login_WrongPassword_Returns401", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		body := fmt.Sprintf(`{"identifier":"%s","password":"wrong"}`, uniqueEmail())
		resp, err := f.Client.Post(f.ServerURL+"/auth/login", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Login_NonExistentUser_Returns401", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		body := fmt.Sprintf(`{"identifier":"%s","password":"password123"}`, uniqueEmail())
		resp, err := f.Client.Post(f.ServerURL+"/auth/login", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Login_InvalidIdentifierFormat_Returns401", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		// "invalid@user!" is treated as an email (has @). Service returns ErrInvalidCreds → 401.
		body := `{"identifier":"invalid@user!","password":"password123"}`
		resp, err := f.Client.Post(f.ServerURL+"/auth/login", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Login_DeactivatedAccount_Returns403", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := uniqueEmail()
		password := "password123"

		regBody := fmt.Sprintf(`{"email":"%s","username":"user_%s","password":"%s","organization_name":"%s"}`, email, uniqueID(), password, uniqueOrgName())
		regResp, err := f.Client.Post(f.ServerURL+"/auth/register", "application/json", strings.NewReader(regBody))
		require.NoError(t, err)
		regResp.Body.Close()

		_, err = f.Pool.Exec(context.Background(), "UPDATE users SET is_active = false WHERE email = $1", email)
		require.NoError(t, err)

		loginBody := fmt.Sprintf(`{"identifier":"%s","password":"%s"}`, email, password)
		resp, err := f.Client.Post(f.ServerURL+"/auth/login", "application/json", strings.NewReader(loginBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	// -----------------------------------------------------------------------
	// Logout
	// -----------------------------------------------------------------------

	t.Run("Logout_WithValidSession_Returns200", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := uniqueEmail()
		f.registerAndLogin(t, email, "user_"+uniqueID(), "TestPass123!", uniqueOrgName())

		resp, err := f.Client.Post(f.ServerURL+"/auth/logout", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// After logout, profile should be unauthorized
		profileResp, err := f.Client.Get(f.ServerURL + "/auth/me")
		require.NoError(t, err)
		defer profileResp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, profileResp.StatusCode)
	})

	t.Run("Logout_WithoutSession_Returns200", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		resp, err := f.Client.Post(f.ServerURL+"/auth/logout", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// -----------------------------------------------------------------------
	// Refresh
	// -----------------------------------------------------------------------

	t.Run("Refresh_ValidToken_Returns200WithRotation", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := uniqueEmail()
		loginResp := f.registerAndLogin(t, email, "refreshuser", "TestPass123!", "Refresh Org")
		initialRefresh := loginResp.RefreshToken

		resp, err := f.Client.Post(f.ServerURL+"/auth/refresh", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var wrapped struct {
			Data struct {
				Token        string `json:"token"`
				RefreshToken string `json:"refresh_token"`
			} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&wrapped)
		require.NoError(t, err)

		assert.NotEqual(t, initialRefresh, wrapped.Data.RefreshToken, "refresh token should rotate")
		assert.NotEmpty(t, wrapped.Data.Token, "new access token should be present")
	})

	t.Run("Refresh_InvalidToken_Returns401", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		req, err := http.NewRequest("POST", f.ServerURL+"/auth/refresh", nil)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "invalid"})
		cleanClient := &http.Client{}
		resp, err := cleanClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Refresh_MissingCookie_Returns401", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		cleanClient := &http.Client{}
		resp, err := cleanClient.Post(f.ServerURL+"/auth/refresh", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// -----------------------------------------------------------------------
	// Profile
	// -----------------------------------------------------------------------

	t.Run("GetProfile_Authenticated_ReturnsUserWithRole", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := uniqueEmail()
		f.registerAndLogin(t, email, "profileuser", "TestPass123!", "Profile Org")

		resp, err := f.Client.Get(f.ServerURL + "/auth/me")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var wrapped struct {
			Data map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&wrapped)
		require.NoError(t, err)

		membership, ok := wrapped.Data["membership"].(map[string]interface{})
		require.True(t, ok, "response should have 'membership'")
		assert.NotEmpty(t, membership["role"])
		assert.NotEmpty(t, membership["organization_id"])
	})

	t.Run("GetProfile_Unauthenticated_Returns401", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		cleanClient := &http.Client{}
		resp, err := cleanClient.Get(f.ServerURL + "/auth/me")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// -----------------------------------------------------------------------
	// Memberships
	// -----------------------------------------------------------------------

	t.Run("GetMemberships_ReturnsList", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := uniqueEmail()
		f.registerAndLogin(t, email, "memberuser", "TestPass123!", "Member Org")

		resp, err := f.Client.Get(f.ServerURL + "/auth/memberships")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var wrapped struct {
			Data map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&wrapped)
		require.NoError(t, err)

		memberships, ok := wrapped.Data["memberships"].([]interface{})
		require.True(t, ok, "memberships should be a list")
		assert.GreaterOrEqual(t, len(memberships), 1)
	})

	// -----------------------------------------------------------------------
	// Bootstrap
	// -----------------------------------------------------------------------

	t.Run("Bootstrap_FirstUser_Returns200", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := uniqueEmail()
		body := fmt.Sprintf(`{"email":"%s","username":"boot_%s","password":"TestPass123!","organization_name":"Bootstrap"}`, email, uniqueID())
		resp, err := f.Client.Post(f.ServerURL+"/auth/bootstrap", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var wrapped struct {
			Data map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&wrapped)
		require.NoError(t, err)
		assert.NotNil(t, wrapped.Data["token"], "bootstrap should return a token")
	})

	t.Run("Bootstrap_AlreadyBootstrapped_Returns409", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email1 := uniqueEmail()
		body1 := fmt.Sprintf(`{"email":"%s","username":"boot1_%s","password":"TestPass123!","organization_name":"BootOrg"}`, email1, uniqueID())
		r1, err := f.Client.Post(f.ServerURL+"/auth/bootstrap", "application/json", strings.NewReader(body1))
		require.NoError(t, err)
		r1.Body.Close()
		assert.Equal(t, http.StatusOK, r1.StatusCode)

		email2 := uniqueEmail()
		body2 := fmt.Sprintf(`{"email":"%s","username":"boot2_%s","password":"TestPass123!","organization_name":"BootOrg2"}`, email2, uniqueID())
		r2, err := f.Client.Post(f.ServerURL+"/auth/bootstrap", "application/json", strings.NewReader(body2))
		require.NoError(t, err)
		defer r2.Body.Close()
		assert.Equal(t, http.StatusConflict, r2.StatusCode)
	})

	// -----------------------------------------------------------------------
	// Switch Organization
	// -----------------------------------------------------------------------

	t.Run("SwitchOrganization_Valid_Returns200", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := uniqueEmail()
		f.registerAndLogin(t, email, "switchuser", "TestPass123!", "Switch Org")

		// Create a second org via DB
		var orgID string
		err := f.Pool.QueryRow(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) 
			 VALUES (gen_random_uuid(), 'Second Org', 'second-org', NOW(), NOW()) 
			 RETURNING id`).Scan(&orgID)
		require.NoError(t, err)

		var userID string
		err = f.Pool.QueryRow(context.Background(),
			`SELECT id FROM users WHERE email = $1`, email).Scan(&userID)
		require.NoError(t, err)

		_, err = f.Pool.Exec(context.Background(),
			`INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, created_at, updated_at)
			 VALUES (gen_random_uuid(), $1, $2, 'employee', true, NOW(), NOW())`, userID, orgID)
		require.NoError(t, err)

		switchBody := fmt.Sprintf(`{"organization_id":"%s"}`, orgID)
		resp, err := f.Client.Post(f.ServerURL+"/auth/switch-organization", "application/json", strings.NewReader(switchBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "switch-organization should return 200")
	})


}
