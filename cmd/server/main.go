package main

import (
	"log"
	stdhttp "net/http"
	"os"
	"strings"

	"github.com/stefanoprivitera/hourglass/internal/adapters/primary/http"
	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/surrealdb"
	"github.com/stefanoprivitera/hourglass/internal/auth"
	authsvc "github.com/stefanoprivitera/hourglass/internal/core/services/auth"
	contractsvc "github.com/stefanoprivitera/hourglass/internal/core/services/contract"
	customersvc "github.com/stefanoprivitera/hourglass/internal/core/services/customer"
	exportsvc "github.com/stefanoprivitera/hourglass/internal/core/services/export"
	invitationsvc "github.com/stefanoprivitera/hourglass/internal/core/services/invitation"
	orgsvc "github.com/stefanoprivitera/hourglass/internal/core/services/organization"
	passwordresetsvc "github.com/stefanoprivitera/hourglass/internal/core/services/password_reset"
	projectsvc "github.com/stefanoprivitera/hourglass/internal/core/services/project"
	tesvc "github.com/stefanoprivitera/hourglass/internal/core/services/time_entry"
	unitsvc "github.com/stefanoprivitera/hourglass/internal/core/services/unit"
	wgsvc "github.com/stefanoprivitera/hourglass/internal/core/services/working_group"
	"github.com/stefanoprivitera/hourglass/internal/db"
	"github.com/stefanoprivitera/hourglass/internal/handlers"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

func main() {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}

	authService := auth.NewService(jwtSecret)

	sdbConn, err := db.NewSurrealDB()
	if err != nil {
		log.Fatalf("Failed to connect to SurrealDB: %v", err)
	}
	defer sdbConn.Close()
	log.Println("Using SurrealDB")

	healthHandler := handlers.NewHealthHandler()

	mux := stdhttp.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler.ServeHTTP)

	timeEntryRepo := surrealdb.NewTimeEntryRepository(sdbConn.DB())
	auditLogRepo := surrealdb.NewAuditLogRepository(sdbConn.DB())
	teService := tesvc.NewService(timeEntryRepo, auditLogRepo)
	hexTEHandler := http.NewTimeEntryHandler(teService)

	userRepo := surrealdb.NewUserRepository(sdbConn.DB())
	orgRepo := surrealdb.NewOrganizationRepository(sdbConn.DB())
	passwordHasher := surrealdb.NewPasswordHasher()
	tokenService := surrealdb.NewTokenService(authService)
	refreshTokenRepo := surrealdb.NewRefreshTokenRepository(sdbConn.DB())

	invitationRepo := surrealdb.NewInvitationRepository(sdbConn.DB())
	invitationService := invitationsvc.NewService(invitationRepo)

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
	mux.HandleFunc("GET /auth/me", middleware.Auth(authService, authHandler.GetProfile))
	mux.HandleFunc("POST /auth/bootstrap", authHandler.Bootstrap)
	mux.HandleFunc("GET /auth/bootstrap-check", authHandler.BootstrapCheck)
	mux.HandleFunc("POST /auth/switch-organization", middleware.Auth(authService, authHandler.SwitchOrganization))
	mux.HandleFunc("GET /auth/memberships", middleware.Auth(authService, authHandler.GetMemberships))

	hexInvitationHandler := http.NewInvitationHandler(invitationService)

	passwordResetRepo := surrealdb.NewPasswordResetRepository(sdbConn.DB())
	userFinder := surrealdb.NewUserFinder(sdbConn.DB())
	passwordResetService := passwordresetsvc.NewService(passwordResetRepo, userRepo, userFinder, passwordHasher, surrealdb.NewTokenService(authService), refreshTokenRepo)
	hexPasswordResetHandler := http.NewPasswordResetHandler(passwordResetService)

	unitRepo := surrealdb.NewUnitRepository(sdbConn.DB())
	unitService := unitsvc.NewService(unitRepo)
	hexUnitHandler := http.NewUnitHandler(unitService)

	wgRepo := surrealdb.NewWorkingGroupRepository(sdbConn.DB())
	wgService := wgsvc.NewService(wgRepo)
	hexWGHandler := http.NewWorkingGroupHandler(wgService)

	customerRepo := surrealdb.NewCustomerRepository(sdbConn.DB())
	customerService := customersvc.NewService(customerRepo)
	hexCustomerHandler := http.NewCustomerHandler(customerService)

	orgMgmtRepo := surrealdb.NewOrganizationManagementRepository(sdbConn.DB())
	orgMgmtService := orgsvc.NewService(orgMgmtRepo)
	hexOrgHandler := http.NewOrganizationHandler(orgMgmtService)

	projectRepo := surrealdb.NewProjectRepository(sdbConn.DB())
	projectService := projectsvc.NewService(projectRepo)
	hexProjectHandler := http.NewProjectHandler(projectService)

	contractRepo := surrealdb.NewContractRepository(sdbConn.DB())
	contractService := contractsvc.NewService(contractRepo)
	hexContractHandler := http.NewContractHandler(contractService)

	exportRepo := surrealdb.NewExportRepository(sdbConn.DB())
	exportService := exportsvc.NewService(exportRepo)
	hexExportHandler := http.NewExportHandler(exportService)

	mux.HandleFunc("POST /auth/password-reset/request", hexPasswordResetHandler.Request)
	mux.HandleFunc("POST /auth/password-reset/verify", hexPasswordResetHandler.Verify)

	mux.HandleFunc("POST /invitations", hexInvitationHandler.Create)
	mux.HandleFunc("GET /invitations/validate/code/{code}", hexInvitationHandler.ValidateCode)
	mux.HandleFunc("GET /invitations/validate/token/{token}", hexInvitationHandler.ValidateToken)
	mux.HandleFunc("POST /invitations/accept", hexInvitationHandler.Accept)

	mux.HandleFunc("GET /units", middleware.Auth(authService, hexUnitHandler.List))
	mux.HandleFunc("POST /units", middleware.Auth(authService, hexUnitHandler.Create))
	mux.HandleFunc("GET /units/{id}", middleware.Auth(authService, hexUnitHandler.Get))
	mux.HandleFunc("PUT /units/{id}", middleware.Auth(authService, hexUnitHandler.Update))
	mux.HandleFunc("DELETE /units/{id}", middleware.Auth(authService, hexUnitHandler.Delete))
	mux.HandleFunc("GET /units/tree", middleware.Auth(authService, hexUnitHandler.GetTree))
	mux.HandleFunc("GET /units/{id}/descendants", middleware.Auth(authService, hexUnitHandler.GetDescendants))

	mux.HandleFunc("GET /working-groups", middleware.Auth(authService, hexWGHandler.List))
	mux.HandleFunc("POST /working-groups", middleware.Auth(authService, hexWGHandler.Create))
	mux.HandleFunc("GET /working-groups/{id}", middleware.Auth(authService, hexWGHandler.Get))
	mux.HandleFunc("PUT /working-groups/{id}", middleware.Auth(authService, hexWGHandler.Update))
	mux.HandleFunc("DELETE /working-groups/{id}", middleware.Auth(authService, hexWGHandler.Delete))
	mux.HandleFunc("GET /working-groups/{id}/members", middleware.Auth(authService, hexWGHandler.ListMembers))
	mux.HandleFunc("POST /working-groups/{id}/members", middleware.Auth(authService, hexWGHandler.AddMember))
	mux.HandleFunc("DELETE /working-groups/{id}/members/{member_id}", middleware.Auth(authService, hexWGHandler.RemoveMember))

	mux.HandleFunc("GET /customers", middleware.Auth(authService, hexCustomerHandler.List))
	mux.HandleFunc("POST /customers", middleware.Auth(authService, hexCustomerHandler.Create))
	mux.HandleFunc("GET /customers/{id}", middleware.Auth(authService, hexCustomerHandler.Get))
	mux.HandleFunc("PUT /customers/{id}", middleware.Auth(authService, hexCustomerHandler.Update))
	mux.HandleFunc("DELETE /customers/{id}", middleware.Auth(authService, hexCustomerHandler.Delete))

	mux.HandleFunc("POST /organizations", middleware.Auth(authService, hexOrgHandler.Create))
	mux.HandleFunc("GET /organizations/{id}", middleware.Auth(authService, hexOrgHandler.Get))
	mux.HandleFunc("POST /organizations/invite", middleware.Auth(authService, hexOrgHandler.Invite))
	mux.HandleFunc("POST /organizations/invite-customer", middleware.Auth(authService, hexOrgHandler.InviteCustomer))
	mux.HandleFunc("GET /organizations/{id}/settings", middleware.Auth(authService, hexOrgHandler.GetSettings))
	mux.HandleFunc("PUT /organizations/{id}/settings", middleware.Auth(authService, hexOrgHandler.UpdateSettings))
	mux.HandleFunc("GET /organizations/members", middleware.Auth(authService, hexOrgHandler.ListMembers))
	mux.HandleFunc("PUT /organizations/members/{member_id}/roles", middleware.Auth(authService, hexOrgHandler.UpdateMemberRoles))
	mux.HandleFunc("DELETE /organizations/members/{member_id}", middleware.Auth(authService, hexOrgHandler.DeactivateMember))

	mux.HandleFunc("GET /projects", middleware.Auth(authService, hexProjectHandler.List))
	mux.HandleFunc("POST /projects", middleware.Auth(authService, hexProjectHandler.Create))
	mux.HandleFunc("GET /projects/{id}", middleware.Auth(authService, hexProjectHandler.Get))
	mux.HandleFunc("POST /projects/{id}/adopt", middleware.Auth(authService, hexProjectHandler.Adopt))
	mux.HandleFunc("GET /projects/{id}/managers", middleware.Auth(authService, hexProjectHandler.ListManagers))
	mux.HandleFunc("POST /projects/{id}/managers", middleware.Auth(authService, hexProjectHandler.AddManager))
	mux.HandleFunc("DELETE /projects/{id}/managers/{user_id}", middleware.Auth(authService, hexProjectHandler.RemoveManager))

	mux.HandleFunc("GET /contracts", middleware.Auth(authService, hexContractHandler.List))
	mux.HandleFunc("POST /contracts", middleware.Auth(authService, hexContractHandler.Create))
	mux.HandleFunc("GET /contracts/{id}", middleware.Auth(authService, hexContractHandler.Get))
	mux.HandleFunc("POST /contracts/{id}/adopt", middleware.Auth(authService, hexContractHandler.Adopt))
	mux.HandleFunc("PUT /contracts/{id}", middleware.Auth(authService, hexContractHandler.Update))
	mux.HandleFunc("POST /contracts/{id}/recalculate-mileage", middleware.Auth(authService, hexContractHandler.RecalculateMileage))

	mux.HandleFunc("GET /exports/timesheets", middleware.Auth(authService, hexExportHandler.Timesheets))
	mux.HandleFunc("GET /exports/expenses", middleware.Auth(authService, hexExportHandler.Expenses))
	mux.HandleFunc("GET /exports/combined", middleware.Auth(authService, hexExportHandler.Combined))

	mux.HandleFunc("GET /time-entries", middleware.Auth(authService, hexTEHandler.List))
	mux.HandleFunc("POST /time-entries", middleware.Auth(authService, hexTEHandler.Create))
	mux.HandleFunc("GET /time-entries/{id}", middleware.Auth(authService, hexTEHandler.Get))
	mux.HandleFunc("PUT /time-entries/{id}", middleware.Auth(authService, hexTEHandler.Update))
	mux.HandleFunc("DELETE /time-entries/{id}", middleware.Auth(authService, hexTEHandler.Delete))
	mux.HandleFunc("POST /time-entries/{id}/submit", middleware.Auth(authService, hexTEHandler.Submit))
	mux.HandleFunc("POST /time-entries/{id}/approve", middleware.Auth(authService, hexTEHandler.Approve))
	mux.HandleFunc("POST /time-entries/{id}/reject", middleware.Auth(authService, hexTEHandler.Reject))
	mux.HandleFunc("GET /time-entries/pending", middleware.Auth(authService, hexTEHandler.ListPending))

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

	rateLimiter := middleware.NewRateLimiter(10, 100)

	log.Printf("Server starting on port %s", port)
	handler := rateLimiter.Middleware(middleware.Logging(middleware.APIVersion(corsMiddleware(allowedOrigins)(mux))))
	if err := stdhttp.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func corsMiddleware(allowedOrigins []string) func(stdhttp.Handler) stdhttp.Handler {
	return func(next stdhttp.Handler) stdhttp.Handler {
		return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			origin := r.Header.Get("Origin")
			allowed := false
			for _, o := range allowedOrigins {
				if o == origin || o == "*" {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(stdhttp.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
