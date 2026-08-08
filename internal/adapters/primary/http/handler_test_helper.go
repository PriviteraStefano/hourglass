package http

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	"github.com/stefanoprivitera/hourglass/internal/auth"
	activitysvc "github.com/stefanoprivitera/hourglass/internal/core/services/activity"
	authsvc "github.com/stefanoprivitera/hourglass/internal/core/services/auth"
	contractsvc "github.com/stefanoprivitera/hourglass/internal/core/services/contract"
	coveragesvc "github.com/stefanoprivitera/hourglass/internal/core/services/coverage"
	customersvc "github.com/stefanoprivitera/hourglass/internal/core/services/customer"
	exportsvc "github.com/stefanoprivitera/hourglass/internal/core/services/export"
	invitationsvc "github.com/stefanoprivitera/hourglass/internal/core/services/invitation"
	orgsettingssvc "github.com/stefanoprivitera/hourglass/internal/core/services/orgsettings"
	orgsvc "github.com/stefanoprivitera/hourglass/internal/core/services/organization"
	passwordresetsvc "github.com/stefanoprivitera/hourglass/internal/core/services/password_reset"
	"github.com/stefanoprivitera/hourglass/internal/core/services/routing"
	tesvc "github.com/stefanoprivitera/hourglass/internal/core/services/time_entry"
	ticketsvc "github.com/stefanoprivitera/hourglass/internal/core/services/ticket"
	unitsvc "github.com/stefanoprivitera/hourglass/internal/core/services/unit"
	wgsvc "github.com/stefanoprivitera/hourglass/internal/core/services/working_group"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stretchr/testify/require"
)

// handlerFixture provides a fully-wired test server backed by
// testcontainers-managed PostgreSQL.  The caller is responsible for
// schema lifecycle (SetupTestSchema / TeardownTestSchema) via the
// pool.  See TestAuthHandlerIntegration for usage.
type handlerFixture struct {
	Client    *http.Client
	Server    *httptest.Server
	ServerURL string
	Pool      *pgxpool.Pool
	authSvc   *auth.Service // exposed for direct token generation if needed
}

// newHandlerFixture creates a test server with real PostgreSQL-backed
// services, all handlers wired exactly as in cmd/server/main.go.
// pool must be obtained from postgres.SetupPackageContainer; schema
// lifecycle is the caller's responsibility.
func newHandlerFixture(t *testing.T, pool *pgxpool.Pool) *handlerFixture {
	t.Helper()

	// -- shared auth primitives --
	jwtSecret := "test-secret"
	authSvc := auth.NewService(jwtSecret)

	mux := http.NewServeMux()

	// Repositories
	userRepo := postgres.NewUserRepository(pool)
	orgRepo := postgres.NewOrganizationRepository(pool)
	passwordHasher := auth.NewPasswordHasher()
	tokenService := auth.NewTokenService(authSvc)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(pool)
	invitationRepo := postgres.NewInvitationRepository(pool)
	passwordResetRepo := postgres.NewPasswordResetRepository(pool)
	userFinder := postgres.NewUserFinder(pool)
	unitRepo := postgres.NewUnitRepository(pool)
	wgRepo := postgres.NewWorkingGroupRepository(pool)
	customerRepo := postgres.NewCustomerRepository(pool)
	orgMgmtRepo := postgres.NewOrganizationManagementRepository(pool)
	activityRepo := postgres.NewActivityRepository(pool)
	contractRepo := postgres.NewContractRepository(pool)
	timeEntryRepo := postgres.NewTimeEntryRepository(pool)
	exportRepo := postgres.NewExportRepository(pool)

	// Services
	hexAuthService := authsvc.NewService(userRepo, orgRepo, tokenService, passwordHasher, refreshTokenRepo)
	invitationService := invitationsvc.NewService(invitationRepo)
	passwordResetService := passwordresetsvc.NewService(passwordResetRepo, userRepo, userFinder, passwordHasher, auth.NewTokenService(authSvc), refreshTokenRepo)
	invitationHandler := NewInvitationHandler(invitationService)
	unitService := unitsvc.NewService(unitRepo)
	wgService := wgsvc.NewService(wgRepo)
	customerService := customersvc.NewService(customerRepo)
	orgMgmtService := orgsvc.NewService(orgMgmtRepo, customerService)
	routingSvc := routing.NewService(wgRepo, activityRepo, unitRepo)
	activityService := activitysvc.NewService(activityRepo, contractRepo, unitRepo, orgRepo, postgres.NewTicketRepository(pool), routingSvc)
	contractService := contractsvc.NewService(contractRepo)
	teService := tesvc.NewService(timeEntryRepo, timeEntryRepo, wgRepo, activityRepo, unitRepo, routingSvc)
	exportService := exportsvc.NewService(exportRepo)
	ticketService := ticketsvc.NewService(postgres.NewTicketRepository(pool), activityRepo, contractRepo, orgRepo)
	coverageRepo := postgres.NewCoverageRepository(pool)
	coverageService := coveragesvc.NewService(coverageRepo, activityRepo, contractRepo, unitRepo, timeEntryRepo, routingSvc)
	coverageHandler := NewCoverageHandler(coverageService)

	// Org settings — the org policy key/value store (D-13-18..23): literal
	// /organizations/settings routes mirroring cmd/server/main.go, with the
	// typed /organizations/{id}/settings wildcard still registered below
	// (Pitfall 6 — ServeMux most-specific-wins).
	orgSettingsRepo := postgres.NewOrgSettingsRepository(pool)
	orgSettingsService := orgsettingssvc.NewService(orgSettingsRepo, orgRepo)
	orgSettingsHandler := NewOrgSettingsHandler(orgSettingsService)

	// Handlers
	authHandler := NewAuthHandler(hexAuthService, invitationService)
	passwordResetHandler := NewPasswordResetHandler(passwordResetService)
	unitHandler := NewUnitHandler(unitService)
	wgHandler := NewWorkingGroupHandler(wgService)
	customerHandler := NewCustomerHandler(customerService)
	orgHandler := NewOrganizationHandler(orgMgmtService)
	activityHandler := NewActivityHandler(activityService, activityRepo)
	contractHandler := NewContractHandler(contractService)
	teHandler := NewTimeEntryHandler(teService)
	exportHandler := NewExportHandler(exportService)
	ticketHandler := NewTicketHandler(ticketService)

	// ---- Unauthenticated routes ----
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("POST /auth/logout", authHandler.Logout)
	mux.HandleFunc("POST /auth/refresh", authHandler.Refresh)
	mux.HandleFunc("POST /auth/bootstrap", authHandler.Bootstrap)
	mux.HandleFunc("GET /auth/bootstrap-check", authHandler.BootstrapCheck)
	mux.HandleFunc("POST /auth/password-reset/request", passwordResetHandler.Request)
	mux.HandleFunc("POST /auth/password-reset/verify", passwordResetHandler.Verify)

	// ---- Authenticated auth routes ----
	mux.HandleFunc("GET /auth/me", middleware.Auth(authSvc, authHandler.GetProfile))
	mux.HandleFunc("POST /auth/switch-organization", middleware.Auth(authSvc, authHandler.SwitchOrganization))
	mux.HandleFunc("GET /auth/memberships", middleware.Auth(authSvc, authHandler.GetMemberships))

	// Invitations
	mux.HandleFunc("POST /invitations", invitationHandler.Create)
	mux.HandleFunc("GET /invitations/validate/code/{code}", invitationHandler.ValidateCode)
	mux.HandleFunc("GET /invitations/validate/token/{token}", invitationHandler.ValidateToken)
	mux.HandleFunc("POST /invitations/accept", invitationHandler.Accept)

	// Units
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

	// Working groups
	mux.HandleFunc("GET /working-groups", middleware.Auth(authSvc, wgHandler.List))
	mux.HandleFunc("POST /working-groups", middleware.Auth(authSvc, wgHandler.Create))
	mux.HandleFunc("GET /working-groups/{id}", middleware.Auth(authSvc, wgHandler.Get))
	mux.HandleFunc("PUT /working-groups/{id}", middleware.Auth(authSvc, wgHandler.Update))
	mux.HandleFunc("DELETE /working-groups/{id}", middleware.Auth(authSvc, wgHandler.Delete))
	mux.HandleFunc("GET /working-groups/{id}/members", middleware.Auth(authSvc, wgHandler.ListMembers))
	mux.HandleFunc("POST /working-groups/{id}/members", middleware.Auth(authSvc, wgHandler.AddMember))
	mux.HandleFunc("DELETE /working-groups/{id}/members/{member_id}", middleware.Auth(authSvc, wgHandler.RemoveMember))

	// Customers
	mux.HandleFunc("GET /customers", middleware.Auth(authSvc, customerHandler.List))
	mux.HandleFunc("POST /customers", middleware.Auth(authSvc, customerHandler.Create))
	mux.HandleFunc("GET /customers/{id}", middleware.Auth(authSvc, customerHandler.Get))
	mux.HandleFunc("PUT /customers/{id}", middleware.Auth(authSvc, customerHandler.Update))
	mux.HandleFunc("DELETE /customers/{id}", middleware.Auth(authSvc, customerHandler.Delete))

	// Organizations
	mux.HandleFunc("POST /organizations", middleware.Auth(authSvc, orgHandler.Create))
	mux.HandleFunc("GET /organizations/{id}", middleware.Auth(authSvc, orgHandler.Get))
	mux.HandleFunc("POST /organizations/invite", middleware.Auth(authSvc, orgHandler.Invite))
	mux.HandleFunc("POST /organizations/invite-customer", middleware.Auth(authSvc, orgHandler.InviteCustomer))
	mux.HandleFunc("GET /organizations/{id}/settings", middleware.Auth(authSvc, orgHandler.GetSettings))
	mux.HandleFunc("PUT /organizations/{id}/settings", middleware.Auth(authSvc, orgHandler.UpdateSettings))
	// Literal org_settings routes (D-13-23) — coexisting with the typed
	// wildcard registrations above (Pitfall 6).
	mux.HandleFunc("GET /organizations/settings", middleware.Auth(authSvc, orgSettingsHandler.Get))
	mux.HandleFunc("PUT /organizations/settings", middleware.Auth(authSvc, orgSettingsHandler.Put))
	mux.HandleFunc("GET /organizations/members", middleware.Auth(authSvc, orgHandler.ListMembers))
	mux.HandleFunc("PUT /organizations/members/{member_id}/roles", middleware.Auth(authSvc, orgHandler.UpdateMemberRoles))
	mux.HandleFunc("DELETE /organizations/members/{member_id}", middleware.Auth(authSvc, orgHandler.DeactivateMember))

	// Activities
	mux.HandleFunc("GET /activities", middleware.Auth(authSvc, activityHandler.List))
	mux.HandleFunc("POST /activities", middleware.Auth(authSvc, activityHandler.Create))
	mux.HandleFunc("GET /activities/{id}", middleware.Auth(authSvc, activityHandler.Get))
	mux.HandleFunc("PUT /activities/{id}", middleware.Auth(authSvc, activityHandler.Update))
	mux.HandleFunc("DELETE /activities/{id}", middleware.Auth(authSvc, activityHandler.Delete))
	mux.HandleFunc("GET /activities/{id}/children", middleware.Auth(authSvc, activityHandler.ListChildren))
	mux.HandleFunc("GET /activity-kinds", middleware.Auth(authSvc, activityHandler.ListKinds))

	// Contracts
	mux.HandleFunc("GET /contracts", middleware.Auth(authSvc, contractHandler.List))
	mux.HandleFunc("POST /contracts", middleware.Auth(authSvc, contractHandler.Create))
	mux.HandleFunc("GET /contracts/{id}", middleware.Auth(authSvc, contractHandler.Get))
	mux.HandleFunc("PUT /contracts/{id}", middleware.Auth(authSvc, contractHandler.Update))
	mux.HandleFunc("POST /contracts/{id}/recalculate-mileage", middleware.Auth(authSvc, contractHandler.RecalculateMileage))
	mux.HandleFunc("DELETE /contracts/{id}", middleware.Auth(authSvc, contractHandler.Delete))

	// Exports
	mux.HandleFunc("GET /exports/timesheets", middleware.Auth(authSvc, exportHandler.Timesheets))
	mux.HandleFunc("GET /exports/expenses", middleware.Auth(authSvc, exportHandler.Expenses))
	mux.HandleFunc("GET /exports/combined", middleware.Auth(authSvc, exportHandler.Combined))

	// Time entries
	mux.HandleFunc("GET /time-entries", middleware.Auth(authSvc, teHandler.List))
	mux.HandleFunc("POST /time-entries", middleware.Auth(authSvc, teHandler.Create))
	mux.HandleFunc("GET /time-entries/{id}", middleware.Auth(authSvc, teHandler.Get))
	mux.HandleFunc("PUT /time-entries/{id}", middleware.Auth(authSvc, teHandler.Update))
	mux.HandleFunc("DELETE /time-entries/{id}", middleware.Auth(authSvc, teHandler.Delete))
	mux.HandleFunc("POST /time-entries/{id}/submit", middleware.Auth(authSvc, teHandler.Submit))
	mux.HandleFunc("POST /time-entries/{id}/approve", middleware.Auth(authSvc, teHandler.Approve))
	mux.HandleFunc("POST /time-entries/{id}/reject", middleware.Auth(authSvc, teHandler.Reject))
	mux.HandleFunc("GET /time-entries/pending", middleware.Auth(authSvc, teHandler.ListPending))

	// Tickets — the 9 registered routes (TICK-05: no DELETE route exists).
	mux.HandleFunc("POST /tickets", middleware.Auth(authSvc, ticketHandler.Create))
	mux.HandleFunc("GET /tickets", middleware.Auth(authSvc, ticketHandler.List))
	mux.HandleFunc("GET /tickets/{id}", middleware.Auth(authSvc, ticketHandler.Get))
	mux.HandleFunc("PUT /tickets/{id}", middleware.Auth(authSvc, ticketHandler.Update))
	mux.HandleFunc("POST /tickets/{id}/triage", middleware.Auth(authSvc, ticketHandler.Triage))
	mux.HandleFunc("POST /tickets/{id}/dismiss", middleware.Auth(authSvc, ticketHandler.Dismiss))
	mux.HandleFunc("POST /tickets/{id}/transition", middleware.Auth(authSvc, ticketHandler.Transition))
	mux.HandleFunc("POST /tickets/{id}/comments", middleware.Auth(authSvc, ticketHandler.AddComment))
	mux.HandleFunc("GET /tickets/{id}/history", middleware.Auth(authSvc, ticketHandler.History))

	// Coverage — the 8 registered routes (D-07/COV-04: no incremental CRUD,
	// no snapshot/audit mutation routes, no finance confirm step; the close
	// body carries only the period — org comes from claims).
	mux.HandleFunc("PUT /time-entries/{id}/allocations", middleware.Auth(authSvc, coverageHandler.PutAllocations))
	mux.HandleFunc("GET /time-entries/{id}/allocations", middleware.Auth(authSvc, coverageHandler.GetAllocations))
	mux.HandleFunc("GET /coverage/proposals/{entry_id}", middleware.Auth(authSvc, coverageHandler.GetProposal))
	mux.HandleFunc("GET /coverage/to-cover", middleware.Auth(authSvc, coverageHandler.GetToCoverQueue))
	mux.HandleFunc("GET /coverage/buckets/{contract_id}/balance", middleware.Auth(authSvc, coverageHandler.GetBucketBalance))
	mux.HandleFunc("POST /coverage/close", middleware.Auth(authSvc, coverageHandler.PostClose))
	mux.HandleFunc("GET /coverage/snapshots/{close_id}", middleware.Auth(authSvc, coverageHandler.GetSnapshot))
	mux.HandleFunc("GET /coverage/allocations/{entry_id}/history", middleware.Auth(authSvc, coverageHandler.GetHistory))

	server := httptest.NewServer(mux)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{Jar: jar}

	t.Cleanup(server.Close)

	return &handlerFixture{
		Client:    client,
		Server:    server,
		ServerURL: server.URL,
		Pool:      pool,
		authSvc:   authSvc,
	}
}

// registerAndLogin registers a user and logs in, returning the parsed login
// response.  The fixture's cookie jar captures auth cookies automatically.
func (f *handlerFixture) registerAndLogin(t *testing.T, email, username, password, orgName string) *authsvc.LoginResponse {
	t.Helper()
	regBody := `{"email":"` + email + `","username":"` + username + `","password":"` + password + `","organization_name":"` + orgName + `"}`
	regResp, err := f.Client.Post(f.ServerURL+"/auth/register", "application/json", strings.NewReader(regBody))
	require.NoError(t, err)
	regResp.Body.Close()
	require.Equal(t, http.StatusOK, regResp.StatusCode, "register should return 200")
	return f.loginUser(t, email, password)
}

// loginUser logs in and returns the parsed login response.
func (f *handlerFixture) loginUser(t *testing.T, identifier, password string) *authsvc.LoginResponse {
	t.Helper()
	body := `{"identifier":"` + identifier + `","password":"` + password + `"}`
	resp, err := f.Client.Post(f.ServerURL+"/auth/login", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "login should return 200")

	var wrapped struct {
		Data *authsvc.LoginResponse `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&wrapped)
	require.NoError(t, err)
	require.NotNil(t, wrapped.Data, "login response should have 'data' field")
	return wrapped.Data
}
