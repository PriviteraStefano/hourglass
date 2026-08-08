package main

import (
	"encoding/json"
	stdhttp "net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stefanoprivitera/hourglass/internal/adapters/primary/http"
	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	"github.com/stefanoprivitera/hourglass/internal/auth"
	activitysvc "github.com/stefanoprivitera/hourglass/internal/core/services/activity"
	authsvc "github.com/stefanoprivitera/hourglass/internal/core/services/auth"
	contractsvc "github.com/stefanoprivitera/hourglass/internal/core/services/contract"
	customersvc "github.com/stefanoprivitera/hourglass/internal/core/services/customer"
	"github.com/stefanoprivitera/hourglass/internal/core/services/export"
	invitationsvc "github.com/stefanoprivitera/hourglass/internal/core/services/invitation"
	orgsvc "github.com/stefanoprivitera/hourglass/internal/core/services/organization"
	passwordresetsvc "github.com/stefanoprivitera/hourglass/internal/core/services/password_reset"
	"github.com/stefanoprivitera/hourglass/internal/core/services/routing"
	tesvc "github.com/stefanoprivitera/hourglass/internal/core/services/time_entry"
	unitsvc "github.com/stefanoprivitera/hourglass/internal/core/services/unit"
	wgsvc "github.com/stefanoprivitera/hourglass/internal/core/services/working_group"
	"github.com/stefanoprivitera/hourglass/internal/handlers"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

// TestSmoke is an integration smoke test that verifies the full server wiring works
// with PostgreSQL. It tests health endpoint, user registration, login, and
// authenticated data access.
//
// This test starts a PostgreSQL container via testcontainers-go. Docker must be running.
// The test is automatically skipped if Docker is not available (testcontainers self-detects).
func TestSmoke(t *testing.T) {
	pool := postgres.TestPool(t)
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	// Server wiring — matches the same pattern as main.go
	jwtSecret := "test-secret"
	authSvc := auth.NewService(jwtSecret)

	mux := stdhttp.NewServeMux()

	// Health endpoint (unauthenticated)
	healthHandler := handlers.NewHealthHandler()
	mux.HandleFunc("GET /health", healthHandler.ServeHTTP)

	// Time entry
	timeEntryRepo := postgres.NewTimeEntryRepository(pool)

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
	authHandler := http.NewAuthHandler(hexAuthService, invitationService)

	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("POST /auth/logout", authHandler.Logout)
	mux.HandleFunc("POST /auth/refresh", authHandler.Refresh)
	mux.HandleFunc("GET /auth/me", middleware.Auth(authSvc, authHandler.GetProfile))
	mux.HandleFunc("POST /auth/bootstrap", authHandler.Bootstrap)
	mux.HandleFunc("GET /auth/bootstrap-check", authHandler.BootstrapCheck)
	mux.HandleFunc("POST /auth/switch-organization", middleware.Auth(authSvc, authHandler.SwitchOrganization))
	mux.HandleFunc("GET /auth/memberships", middleware.Auth(authSvc, authHandler.GetMemberships))

	// Invitations
	invitationHandler := http.NewInvitationHandler(invitationService)
	mux.HandleFunc("POST /invitations", invitationHandler.Create)
	mux.HandleFunc("GET /invitations/validate/code/{code}", invitationHandler.ValidateCode)
	mux.HandleFunc("GET /invitations/validate/token/{token}", invitationHandler.ValidateToken)
	mux.HandleFunc("POST /invitations/accept", invitationHandler.Accept)

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
	passwordResetHandler := http.NewPasswordResetHandler(passwordResetService)
	mux.HandleFunc("POST /auth/password-reset/request", passwordResetHandler.Request)
	mux.HandleFunc("POST /auth/password-reset/verify", passwordResetHandler.Verify)

	// Units
	unitRepo := postgres.NewUnitRepository(pool)
	unitService := unitsvc.NewService(unitRepo)
	unitHandler := http.NewUnitHandler(unitService)

	// Working groups
	wgRepo := postgres.NewWorkingGroupRepository(pool)
	wgService := wgsvc.NewService(wgRepo)
	wgHandler := http.NewWorkingGroupHandler(wgService)

	// Customers
	customerRepo := postgres.NewCustomerRepository(pool)
	customerService := customersvc.NewService(customerRepo)
	hexCustomerHandler := http.NewCustomerHandler(customerService)

	// Organizations
	orgMgmtRepo := postgres.NewOrganizationManagementRepository(pool)
	orgMgmtService := orgsvc.NewService(orgMgmtRepo, customerService)
	orgHandler := http.NewOrganizationHandler(orgMgmtService)

	// Contracts
	contractRepo := postgres.NewContractRepository(pool)
	contractService := contractsvc.NewService(contractRepo)
	contractHandler := http.NewContractHandler(contractService)

	// Activities
	activityRepo := postgres.NewActivityRepository(pool)
	routingSvc := routing.NewService(wgRepo, activityRepo, unitRepo)
	activityService := activitysvc.NewService(activityRepo, contractRepo, unitRepo, orgRepo, postgres.NewTicketRepository(pool), postgres.NewDirectionRepository(pool), routingSvc)
	activityHandler := http.NewActivityHandler(activityService, activityRepo)

	// Export
	exportRepo := postgres.NewExportRepository(pool)
	exportService := export.NewService(exportRepo)
	exportHandler := http.NewExportHandler(exportService)

	teService := tesvc.NewService(timeEntryRepo, timeEntryRepo, wgRepo, activityRepo, unitRepo, routingSvc)
	hexTEHandler := http.NewTimeEntryHandler(teService)

	// Register protected routes
	mux.HandleFunc("GET /units", middleware.Auth(authSvc, unitHandler.List))
	mux.HandleFunc("POST /units", middleware.Auth(authSvc, unitHandler.Create))
	mux.HandleFunc("GET /units/{id}", middleware.Auth(authSvc, unitHandler.Get))
	mux.HandleFunc("PUT /units/{id}", middleware.Auth(authSvc, unitHandler.Update))
	mux.HandleFunc("DELETE /units/{id}", middleware.Auth(authSvc, unitHandler.Delete))
	mux.HandleFunc("GET /units/tree", middleware.Auth(authSvc, unitHandler.GetTree))
	mux.HandleFunc("GET /units/{id}/descendants", middleware.Auth(authSvc, unitHandler.GetDescendants))
	mux.HandleFunc("GET /units/{id}/members", middleware.Auth(authSvc, unitHandler.ListMembers))
	mux.HandleFunc("POST /units/{id}/members", middleware.Auth(authSvc, unitHandler.AddMember))
	mux.HandleFunc("DELETE /units/{id}/members/{membership_id}", middleware.Auth(authSvc, unitHandler.RemoveMember))

	mux.HandleFunc("GET /working-groups", middleware.Auth(authSvc, wgHandler.List))
	mux.HandleFunc("POST /working-groups", middleware.Auth(authSvc, wgHandler.Create))
	mux.HandleFunc("GET /working-groups/{id}", middleware.Auth(authSvc, wgHandler.Get))
	mux.HandleFunc("PUT /working-groups/{id}", middleware.Auth(authSvc, wgHandler.Update))
	mux.HandleFunc("DELETE /working-groups/{id}", middleware.Auth(authSvc, wgHandler.Delete))
	mux.HandleFunc("GET /working-groups/{id}/members", middleware.Auth(authSvc, wgHandler.ListMembers))
	mux.HandleFunc("POST /working-groups/{id}/members", middleware.Auth(authSvc, wgHandler.AddMember))
	mux.HandleFunc("DELETE /working-groups/{id}/members/{member_id}", middleware.Auth(authSvc, wgHandler.RemoveMember))

	mux.HandleFunc("GET /customers", middleware.Auth(authSvc, hexCustomerHandler.List))
	mux.HandleFunc("POST /customers", middleware.Auth(authSvc, hexCustomerHandler.Create))
	mux.HandleFunc("GET /customers/{id}", middleware.Auth(authSvc, hexCustomerHandler.Get))
	mux.HandleFunc("PUT /customers/{id}", middleware.Auth(authSvc, hexCustomerHandler.Update))
	mux.HandleFunc("DELETE /customers/{id}", middleware.Auth(authSvc, hexCustomerHandler.Delete))

	mux.HandleFunc("POST /organizations", middleware.Auth(authSvc, orgHandler.Create))
	mux.HandleFunc("GET /organizations/{id}", middleware.Auth(authSvc, orgHandler.Get))
	mux.HandleFunc("POST /organizations/invite", middleware.Auth(authSvc, orgHandler.Invite))
	mux.HandleFunc("POST /organizations/invite-customer", middleware.Auth(authSvc, orgHandler.InviteCustomer))
	mux.HandleFunc("GET /organizations/{id}/settings", middleware.Auth(authSvc, orgHandler.GetSettings))
	mux.HandleFunc("PUT /organizations/{id}/settings", middleware.Auth(authSvc, orgHandler.UpdateSettings))
	mux.HandleFunc("GET /organizations/members", middleware.Auth(authSvc, orgHandler.ListMembers))
	mux.HandleFunc("PUT /organizations/members/{member_id}/roles", middleware.Auth(authSvc, orgHandler.UpdateMemberRoles))
	mux.HandleFunc("DELETE /organizations/members/{member_id}", middleware.Auth(authSvc, orgHandler.DeactivateMember))

	mux.HandleFunc("GET /activities", middleware.Auth(authSvc, activityHandler.List))
	mux.HandleFunc("POST /activities", middleware.Auth(authSvc, activityHandler.Create))
	mux.HandleFunc("GET /activities/{id}", middleware.Auth(authSvc, activityHandler.Get))
	mux.HandleFunc("PUT /activities/{id}", middleware.Auth(authSvc, activityHandler.Update))
	mux.HandleFunc("DELETE /activities/{id}", middleware.Auth(authSvc, activityHandler.Delete))
	mux.HandleFunc("GET /activities/{id}/children", middleware.Auth(authSvc, activityHandler.ListChildren))
	mux.HandleFunc("GET /activity-kinds", middleware.Auth(authSvc, activityHandler.ListKinds))

	mux.HandleFunc("GET /contracts", middleware.Auth(authSvc, contractHandler.List))
	mux.HandleFunc("POST /contracts", middleware.Auth(authSvc, contractHandler.Create))
	mux.HandleFunc("GET /contracts/{id}", middleware.Auth(authSvc, contractHandler.Get))
	mux.HandleFunc("PUT /contracts/{id}", middleware.Auth(authSvc, contractHandler.Update))
	mux.HandleFunc("POST /contracts/{id}/recalculate-mileage", middleware.Auth(authSvc, contractHandler.RecalculateMileage))
	mux.HandleFunc("DELETE /contracts/{id}", middleware.Auth(authSvc, contractHandler.Delete))

	mux.HandleFunc("GET /exports/timesheets", middleware.Auth(authSvc, exportHandler.Timesheets))
	mux.HandleFunc("GET /exports/expenses", middleware.Auth(authSvc, exportHandler.Expenses))
	mux.HandleFunc("GET /exports/combined", middleware.Auth(authSvc, exportHandler.Combined))

	mux.HandleFunc("GET /time-entries", middleware.Auth(authSvc, hexTEHandler.List))
	mux.HandleFunc("POST /time-entries", middleware.Auth(authSvc, hexTEHandler.Create))
	mux.HandleFunc("GET /time-entries/{id}", middleware.Auth(authSvc, hexTEHandler.Get))
	mux.HandleFunc("PUT /time-entries/{id}", middleware.Auth(authSvc, hexTEHandler.Update))
	mux.HandleFunc("DELETE /time-entries/{id}", middleware.Auth(authSvc, hexTEHandler.Delete))
	mux.HandleFunc("POST /time-entries/{id}/submit", middleware.Auth(authSvc, hexTEHandler.Submit))
	mux.HandleFunc("POST /time-entries/{id}/approve", middleware.Auth(authSvc, hexTEHandler.Approve))
	mux.HandleFunc("POST /time-entries/{id}/reject", middleware.Auth(authSvc, hexTEHandler.Reject))
	mux.HandleFunc("GET /time-entries/pending", middleware.Auth(authSvc, hexTEHandler.ListPending))

	// Start test server
	server := httptest.NewServer(mux)
	defer server.Close()

	// Create HTTP client with cookie jar for auth cookie tracking
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}
	client := &stdhttp.Client{Jar: jar}

	// --- Step 1: Health check ---
	t.Log("=== Step 1: GET /health ===")
	{
		resp, err := client.Get(server.URL + "/health")
		if err != nil {
			t.Fatalf("GET /health: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != stdhttp.StatusOK {
			t.Fatalf("GET /health: expected 200, got %d", resp.StatusCode)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode health response: %v", err)
		}
		if body["status"] != "ok" {
			t.Fatalf("health: expected status=ok, got %v", body["status"])
		}
		t.Log("PASS: health endpoint returns 200 with status=ok")
	}

	// --- Step 2: Register a new user ---
	email := "smoke-" + t.Name() + "@test.com"
	username := "smokeuser"
	password := "TestPass123!"
	orgName := "Smoke Test Org"

	t.Log("=== Step 2: POST /auth/register ===")
	{
		registerBody := `{"email":"` + email + `","username":"` + username + `","password":"` + password + `","organization_name":"` + orgName + `"}`
		resp, err := client.Post(server.URL+"/auth/register", "application/json", strings.NewReader(registerBody))
		if err != nil {
			t.Fatalf("POST /auth/register: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != stdhttp.StatusOK {
			// Read body for debug info
			var errBody map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errBody)
			t.Fatalf("POST /auth/register: expected 200, got %d: %v", resp.StatusCode, errBody)
		}
		t.Log("PASS: register returns 200")
	}

	// --- Step 3: Login with registered credentials ---
	t.Log("=== Step 3: POST /auth/login ===")
	{
		loginBody := `{"identifier":"` + email + `","password":"` + password + `"}`
		resp, err := client.Post(server.URL+"/auth/login", "application/json", strings.NewReader(loginBody))
		if err != nil {
			t.Fatalf("POST /auth/login: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != stdhttp.StatusOK {
			t.Fatalf("POST /auth/login: expected 200, got %d", resp.StatusCode)
		}

		// Verify auth_token was set in response cookies
		hasAuthToken := false
		for _, c := range resp.Cookies() {
			if c.Name == "auth_token" && c.Value != "" {
				hasAuthToken = true
				break
			}
		}
		if !hasAuthToken {
			t.Fatal("POST /auth/login: no auth_token cookie set in response")
		}

		// Verify cookie jar captured it for subsequent requests
		serverURL, err := url.Parse(server.URL)
		if err != nil {
			t.Fatalf("parse server URL: %v", err)
		}
		jarCookies := jar.Cookies(serverURL)
		jarHasToken := false
		for _, c := range jarCookies {
			if c.Name == "auth_token" && c.Value != "" {
				jarHasToken = true
				break
			}
		}
		if !jarHasToken {
			t.Fatal("POST /auth/login: auth_token cookie not captured by jar")
		}
		t.Log("PASS: login returns 200 with auth_token cookie")
	}

	// --- Step 4: Authenticated GET /units ---
	t.Log("=== Step 4: GET /units (authenticated) ===")
	{
		resp, err := client.Get(server.URL + "/units")
		if err != nil {
			t.Fatalf("GET /units: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != stdhttp.StatusOK {
			t.Fatalf("GET /units: expected 200, got %d", resp.StatusCode)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode /units response: %v", err)
		}

		// Verify response has a data field (may be an empty array for fresh DB)
		if _, ok := body["data"]; !ok {
			t.Fatal("GET /units: expected 'data' field in response")
		}
		t.Log("PASS: authenticated GET /units returns 200 with data")
	}
}
