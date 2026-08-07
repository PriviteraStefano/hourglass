package main

import (
	"log"
	stdhttp "net/http"
	"os"
	"strconv"
	"strings"

	"github.com/stefanoprivitera/hourglass/internal/adapters/primary/http"
	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	"github.com/stefanoprivitera/hourglass/internal/auth"
	activitysvc "github.com/stefanoprivitera/hourglass/internal/core/services/activity"
	authsvc "github.com/stefanoprivitera/hourglass/internal/core/services/auth"
	contractsvc "github.com/stefanoprivitera/hourglass/internal/core/services/contract"
	customersvc "github.com/stefanoprivitera/hourglass/internal/core/services/customer"
	expsvc "github.com/stefanoprivitera/hourglass/internal/core/services/expense"
	exportsvc "github.com/stefanoprivitera/hourglass/internal/core/services/export"
	invitationsvc "github.com/stefanoprivitera/hourglass/internal/core/services/invitation"
	orgsvc "github.com/stefanoprivitera/hourglass/internal/core/services/organization"
	passwordresetsvc "github.com/stefanoprivitera/hourglass/internal/core/services/password_reset"
	"github.com/stefanoprivitera/hourglass/internal/core/services/routing"
	tesvc "github.com/stefanoprivitera/hourglass/internal/core/services/time_entry"
	unitsvc "github.com/stefanoprivitera/hourglass/internal/core/services/unit"
	wgsvc "github.com/stefanoprivitera/hourglass/internal/core/services/working_group"
	"github.com/stefanoprivitera/hourglass/internal/db"
	"github.com/stefanoprivitera/hourglass/internal/handlers"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

func main() {
	env := os.Getenv("GO_ENV")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		if env == "production" || env == "staging" {
			log.Fatal("FATAL: JWT_SECRET is required in production/staging environments")
		}
		log.Println("WARNING: Using default JWT_SECRET. Set JWT_SECRET in production.")
		jwtSecret = "dev-secret-change-in-production"
	}

	authService := auth.NewService(jwtSecret)

	pool, err := db.NewPool()
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL pool: %v", err)
	}
	defer db.ClosePool(pool)
	log.Println("PostgreSQL pool initialized")

	healthHandler := handlers.NewHealthHandler()

	mux := stdhttp.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler.ServeHTTP)

	timeEntryRepo := postgres.NewTimeEntryRepository(pool)

	expenseRepo := postgres.NewExpenseRepository(pool)

	userRepo := postgres.NewUserRepository(pool)
	orgRepo := postgres.NewOrganizationRepository(pool)
	passwordHasher := auth.NewPasswordHasher()
	tokenService := auth.NewTokenService(authService)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(pool)

	invitationRepo := postgres.NewInvitationRepository(pool)
	invitationService := invitationsvc.NewService(invitationRepo)

	hexAuthService := authsvc.NewService(
		userRepo,
		orgRepo,
		tokenService,
		passwordHasher,
		refreshTokenRepo,
	)
	authHandler := http.NewAuthHandler(hexAuthService, invitationService)

	rateLimit := 5
	if rl := os.Getenv("RATE_LIMIT"); rl != "" {
		if v, err := strconv.Atoi(rl); err == nil && v > 0 {
			rateLimit = v
		}
	}
	authRateLimiter := middleware.NewRateLimiter(rateLimit, 100)
	passwordResetRateLimiter := middleware.NewRateLimiter(3, 60)

	mux.Handle("POST /auth/register", authRateLimiter.Middleware(stdhttp.HandlerFunc(authHandler.Register)))
	mux.Handle("POST /auth/login", authRateLimiter.Middleware(stdhttp.HandlerFunc(authHandler.Login)))
	mux.HandleFunc("POST /auth/logout", authHandler.Logout)
	mux.HandleFunc("POST /auth/refresh", authHandler.Refresh)
	mux.HandleFunc("GET /auth/me", middleware.Auth(authService, authHandler.GetProfile))
	mux.HandleFunc("POST /auth/bootstrap", authHandler.Bootstrap)
	mux.HandleFunc("GET /auth/bootstrap-check", authHandler.BootstrapCheck)
	mux.HandleFunc("POST /auth/switch-organization", middleware.Auth(authService, authHandler.SwitchOrganization))
	mux.HandleFunc("GET /auth/memberships", middleware.Auth(authService, authHandler.GetMemberships))

	invitationHandler := http.NewInvitationHandler(invitationService)

	passwordResetRepo := postgres.NewPasswordResetRepository(pool)
	userFinder := postgres.NewUserFinder(pool)
	passwordResetService := passwordresetsvc.NewService(passwordResetRepo, userRepo, userFinder, passwordHasher, auth.NewTokenService(authService), refreshTokenRepo)
	passwordResetHandler := http.NewPasswordResetHandler(passwordResetService)

	unitRepo := postgres.NewUnitRepository(pool)
	unitService := unitsvc.NewService(unitRepo)
	unitHandler := http.NewUnitHandler(unitService)

	wgRepo := postgres.NewWorkingGroupRepository(pool)
	wgService := wgsvc.NewService(wgRepo)
	wgHandler := http.NewWorkingGroupHandler(wgService)

	customerRepo := postgres.NewCustomerRepository(pool)
	customerService := customersvc.NewService(customerRepo)
	hexCustomerHandler := http.NewCustomerHandler(customerService)

	orgMgmtRepo := postgres.NewOrganizationManagementRepository(pool)
	orgMgmtService := orgsvc.NewService(orgMgmtRepo, customerService)
	orgHandler := http.NewOrganizationHandler(orgMgmtService)

	contractRepo := postgres.NewContractRepository(pool)
	contractService := contractsvc.NewService(contractRepo)
	contractHandler := http.NewContractHandler(contractService)

	// Activities — the single recursive work entity replacing projects +
	// subprojects (ADR-P-007, ADR-BE-014 R-6). The handler holds the repo for
	// the detail endpoint's derived reads (ancestry / commercial context /
	// billability); the service owns the CRUD surface.
	activityRepo := postgres.NewActivityRepository(pool)
	activityService := activitysvc.NewService(activityRepo, contractRepo, unitRepo)
	activityHandler := http.NewActivityHandler(activityService, activityRepo)

	// Entry services — routing resolves through activity → WG chain
	// (ADR-BE-014 R-1/R-2), so they take the working-group + activity + unit
	// repos. The routing service is shared: proposal approval (plan 05)
	// consumes the same resolution so entry and proposal routing cannot drift
	// (D-G).
	routingSvc := routing.NewService(wgRepo, activityRepo, unitRepo)
	teService := tesvc.NewService(timeEntryRepo, timeEntryRepo, wgRepo, activityRepo, unitRepo, routingSvc)
	hexTEHandler := http.NewTimeEntryHandler(teService)

	expenseService := expsvc.NewService(expenseRepo, wgRepo, activityRepo, unitRepo)
	expenseHandler := http.NewExpenseHandler(expenseService)

	exportRepo := postgres.NewExportRepository(pool)
	exportService := exportsvc.NewService(exportRepo)
	exportHandler := http.NewExportHandler(exportService)

	mux.Handle("POST /auth/password-reset/request", passwordResetRateLimiter.Middleware(stdhttp.HandlerFunc(passwordResetHandler.Request)))
	mux.Handle("POST /auth/password-reset/verify", passwordResetRateLimiter.Middleware(stdhttp.HandlerFunc(passwordResetHandler.Verify)))

	mux.HandleFunc("POST /invitations", invitationHandler.Create)
	mux.HandleFunc("GET /invitations/validate/code/{code}", invitationHandler.ValidateCode)
	mux.HandleFunc("GET /invitations/validate/token/{token}", invitationHandler.ValidateToken)
	mux.HandleFunc("POST /invitations/accept", invitationHandler.Accept)

	mux.HandleFunc("GET /units", middleware.Auth(authService, unitHandler.List))
	mux.HandleFunc("POST /units", middleware.Auth(authService, unitHandler.Create))
	mux.HandleFunc("GET /units/{id}", middleware.Auth(authService, unitHandler.Get))
	mux.HandleFunc("PUT /units/{id}", middleware.Auth(authService, unitHandler.Update))
	mux.HandleFunc("DELETE /units/{id}", middleware.Auth(authService, unitHandler.Delete))
	mux.HandleFunc("GET /units/tree", middleware.Auth(authService, unitHandler.GetTree))
	mux.HandleFunc("GET /units/{id}/descendants", middleware.Auth(authService, unitHandler.GetDescendants))
	mux.HandleFunc("GET /units/{id}/members", middleware.Auth(authService, unitHandler.ListMembers))
	mux.HandleFunc("POST /units/{id}/members", middleware.Auth(authService, unitHandler.AddMember))
	mux.HandleFunc("DELETE /units/{id}/members/{membership_id}", middleware.Auth(authService, unitHandler.RemoveMember))
	mux.HandleFunc("PUT /units/{id}/members/{membership_id}", middleware.Auth(authService, unitHandler.UpdateMember))
	mux.HandleFunc("GET /units/members/batch", middleware.Auth(authService, unitHandler.ListMembersBatch))

	mux.HandleFunc("GET /working-groups", middleware.Auth(authService, wgHandler.List))
	mux.HandleFunc("POST /working-groups", middleware.Auth(authService, wgHandler.Create))
	mux.HandleFunc("GET /working-groups/{id}", middleware.Auth(authService, wgHandler.Get))
	mux.HandleFunc("PUT /working-groups/{id}", middleware.Auth(authService, wgHandler.Update))
	mux.HandleFunc("DELETE /working-groups/{id}", middleware.Auth(authService, wgHandler.Delete))
	mux.HandleFunc("GET /working-groups/{id}/members", middleware.Auth(authService, wgHandler.ListMembers))
	mux.HandleFunc("POST /working-groups/{id}/members", middleware.Auth(authService, wgHandler.AddMember))
	mux.HandleFunc("DELETE /working-groups/{id}/members/{member_id}", middleware.Auth(authService, wgHandler.RemoveMember))

	mux.HandleFunc("GET /customers", middleware.Auth(authService, hexCustomerHandler.List))
	mux.HandleFunc("POST /customers", middleware.Auth(authService, hexCustomerHandler.Create))
	mux.HandleFunc("GET /customers/{id}", middleware.Auth(authService, hexCustomerHandler.Get))
	mux.HandleFunc("PUT /customers/{id}", middleware.Auth(authService, hexCustomerHandler.Update))
	mux.HandleFunc("DELETE /customers/{id}", middleware.Auth(authService, hexCustomerHandler.Delete))

	mux.HandleFunc("POST /organizations", middleware.Auth(authService, orgHandler.Create))
	mux.HandleFunc("GET /organizations/{id}", middleware.Auth(authService, orgHandler.Get))
	mux.HandleFunc("POST /organizations/invite", middleware.Auth(authService, orgHandler.Invite))
	mux.HandleFunc("POST /organizations/invite-customer", middleware.Auth(authService, orgHandler.InviteCustomer))
	mux.HandleFunc("GET /organizations/{id}/settings", middleware.Auth(authService, orgHandler.GetSettings))
	mux.HandleFunc("PUT /organizations/{id}/settings", middleware.Auth(authService, orgHandler.UpdateSettings))
	mux.HandleFunc("GET /organizations/members", middleware.Auth(authService, orgHandler.ListMembers))
	mux.HandleFunc("PUT /organizations/members/{member_id}/roles", middleware.Auth(authService, orgHandler.UpdateMemberRoles))
	mux.HandleFunc("DELETE /organizations/members/{member_id}", middleware.Auth(authService, orgHandler.DeactivateMember))

	mux.HandleFunc("GET /activities", middleware.Auth(authService, activityHandler.List))
	mux.HandleFunc("POST /activities", middleware.Auth(authService, activityHandler.Create))
	mux.HandleFunc("GET /activities/{id}", middleware.Auth(authService, activityHandler.Get))
	mux.HandleFunc("PUT /activities/{id}", middleware.Auth(authService, activityHandler.Update))
	mux.HandleFunc("DELETE /activities/{id}", middleware.Auth(authService, activityHandler.Delete))
	mux.HandleFunc("GET /activities/{id}/children", middleware.Auth(authService, activityHandler.ListChildren))
	mux.HandleFunc("GET /activity-kinds", middleware.Auth(authService, activityHandler.ListKinds))

	mux.HandleFunc("GET /contracts", middleware.Auth(authService, contractHandler.List))
	mux.HandleFunc("POST /contracts", middleware.Auth(authService, contractHandler.Create))
	mux.HandleFunc("GET /contracts/{id}", middleware.Auth(authService, contractHandler.Get))
	mux.HandleFunc("POST /contracts/{id}/adopt", middleware.Auth(authService, contractHandler.Adopt))
	mux.HandleFunc("PUT /contracts/{id}", middleware.Auth(authService, contractHandler.Update))
	mux.HandleFunc("POST /contracts/{id}/recalculate-mileage", middleware.Auth(authService, contractHandler.RecalculateMileage))
	mux.HandleFunc("DELETE /contracts/{id}", middleware.Auth(authService, contractHandler.Delete))

	mux.HandleFunc("GET /exports/timesheets", middleware.Auth(authService, exportHandler.Timesheets))
	mux.HandleFunc("GET /exports/expenses", middleware.Auth(authService, exportHandler.Expenses))
	mux.HandleFunc("GET /exports/combined", middleware.Auth(authService, exportHandler.Combined))
	mux.HandleFunc("GET /exports/timesheets/count", middleware.Auth(authService, exportHandler.CountTimesheets))
	mux.HandleFunc("GET /exports/expenses/count", middleware.Auth(authService, exportHandler.CountExpenses))
	mux.HandleFunc("GET /exports/combined/count", middleware.Auth(authService, exportHandler.CountCombined))

	mux.HandleFunc("GET /time-entries", middleware.Auth(authService, hexTEHandler.List))
	mux.HandleFunc("POST /time-entries", middleware.Auth(authService, hexTEHandler.Create))
	mux.HandleFunc("GET /time-entries/{id}", middleware.Auth(authService, hexTEHandler.Get))
	mux.HandleFunc("PUT /time-entries/{id}", middleware.Auth(authService, hexTEHandler.Update))
	mux.HandleFunc("DELETE /time-entries/{id}", middleware.Auth(authService, hexTEHandler.Delete))
	mux.HandleFunc("POST /time-entries/{id}/submit", middleware.Auth(authService, hexTEHandler.Submit))
	mux.HandleFunc("POST /time-entries/{id}/approve", middleware.Auth(authService, hexTEHandler.Approve))
	mux.HandleFunc("POST /time-entries/{id}/reject", middleware.Auth(authService, hexTEHandler.Reject))
	mux.HandleFunc("GET /time-entries/pending", middleware.Auth(authService, hexTEHandler.ListPending))

	// Expense routes
	mux.HandleFunc("GET /expenses", middleware.Auth(authService, expenseHandler.List))
	mux.HandleFunc("POST /expenses", middleware.Auth(authService, expenseHandler.Create))
	mux.HandleFunc("GET /expenses/{id}", middleware.Auth(authService, expenseHandler.Get))
	mux.HandleFunc("PUT /expenses/{id}", middleware.Auth(authService, expenseHandler.Update))
	mux.HandleFunc("DELETE /expenses/{id}", middleware.Auth(authService, expenseHandler.Delete))
	mux.HandleFunc("POST /expenses/{id}/submit", middleware.Auth(authService, expenseHandler.Submit))
	mux.HandleFunc("POST /expenses/{id}/approve", middleware.Auth(authService, expenseHandler.Approve))
	mux.HandleFunc("POST /expenses/{id}/reject", middleware.Auth(authService, expenseHandler.Reject))
	mux.HandleFunc("GET /expenses/pending", middleware.Auth(authService, expenseHandler.ListPending))
	mux.HandleFunc("POST /expenses/{id}/receipt", middleware.Auth(authService, expenseHandler.ReceiptUpload))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
	var allowedOrigins []string
	if allowedOriginsEnv != "" {
		allowedOrigins = strings.Split(allowedOriginsEnv, ",")
	} else {
		allowedOrigins = []string{"http://localhost:3000"}
	}

	// Outer rate limiter covering all routes: anonymous clients get
	// anonymousRateLimit requests/min (default 20), authenticated clients 100.
	// ANONYMOUS_RATE_LIMIT is a deployment knob (same pattern as RATE_LIMIT
	// for the auth endpoints) — e2e suites raise it to run full specs.
	anonymousRateLimit := 20
	if rl := os.Getenv("ANONYMOUS_RATE_LIMIT"); rl != "" {
		if v, err := strconv.Atoi(rl); err == nil && v > 0 {
			anonymousRateLimit = v
		}
	}
	rateLimiter := middleware.NewRateLimiter(anonymousRateLimit, 100)

	log.Printf("Server starting on port %s", port)
	handler := middleware.TryAuth(authService, rateLimiter.Middleware(middleware.Logging(middleware.APIVersion(middleware.CORS(allowedOrigins)(mux)))))
	if err := stdhttp.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
