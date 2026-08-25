package main

import (
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stefanoprivitera/hourglass/internal/adapters/primary/http"
	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	"github.com/stefanoprivitera/hourglass/internal/auth"
	activitysvc "github.com/stefanoprivitera/hourglass/internal/core/services/activity"
	authsvc "github.com/stefanoprivitera/hourglass/internal/core/services/auth"
	contractsvc "github.com/stefanoprivitera/hourglass/internal/core/services/contract"
	coveragesvc "github.com/stefanoprivitera/hourglass/internal/core/services/coverage"
	customersvc "github.com/stefanoprivitera/hourglass/internal/core/services/customer"
	directionsvc "github.com/stefanoprivitera/hourglass/internal/core/services/direction"
	expsvc "github.com/stefanoprivitera/hourglass/internal/core/services/expense"
	exportsvc "github.com/stefanoprivitera/hourglass/internal/core/services/export"
	invitationsvc "github.com/stefanoprivitera/hourglass/internal/core/services/invitation"
	orgsvc "github.com/stefanoprivitera/hourglass/internal/core/services/organization"
	orgsettingssvc "github.com/stefanoprivitera/hourglass/internal/core/services/orgsettings"
	passwordresetsvc "github.com/stefanoprivitera/hourglass/internal/core/services/password_reset"
	"github.com/stefanoprivitera/hourglass/internal/core/services/routing"
	tesvc "github.com/stefanoprivitera/hourglass/internal/core/services/time_entry"
	ticketsvc "github.com/stefanoprivitera/hourglass/internal/core/services/ticket"
	unitsvc "github.com/stefanoprivitera/hourglass/internal/core/services/unit"
	wgsvc "github.com/stefanoprivitera/hourglass/internal/core/services/working_group"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

// appGraph is the composition root: every repository, service, and handler is
// assembled in one typed place so the wiring is explicit and checkable.
type appGraph struct {
	AuthService *auth.Service

	// Auth + account lifecycle handlers
	AuthHandler           *http.AuthHandler
	InvitationHandler     *http.InvitationHandler
	PasswordResetHandler  *http.PasswordResetHandler
	OrgHandler            *http.OrganizationHandler
	OrgSettingsHandler    *http.OrgSettingsHandler

	// Domain handlers
	UnitHandler       *http.UnitHandler
	WGHandler         *http.WorkingGroupHandler
	CustomerHandler   *http.CustomerHandler
	ContractHandler   *http.ContractHandler
	ActivityHandler   *http.ActivityHandler
	TicketHandler     *http.TicketHandler
	CoverageHandler   *http.CoverageHandler
	DirectionHandler  *http.DirectionHandler
	TEHandler         *http.TimeEntryHandler
	ExpenseHandler    *http.ExpenseHandler
	ExportHandler     *http.ExportHandler

	// Rate limiters
	AuthRateLimiter         *middleware.RateLimiter
	PasswordResetRateLimiter *middleware.RateLimiter
	RateLimiter             *middleware.RateLimiter
}

func buildGraph(pool *pgxpool.Pool, jwtSecret string) (*appGraph, error) {
	authService := auth.NewService(jwtSecret)
	passwordHasher := auth.NewPasswordHasher()
	tokenService := auth.NewTokenService(authService)

	// Repositories
	timeEntryRepo := postgres.NewTimeEntryRepository(pool)
	expenseRepo := postgres.NewExpenseRepository(pool)
	userRepo := postgres.NewUserRepository(pool)
	orgRepo := postgres.NewOrganizationRepository(pool)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(pool)
	invitationRepo := postgres.NewInvitationRepository(pool)
	passwordResetRepo := postgres.NewPasswordResetRepository(pool)
	userFinder := postgres.NewUserFinder(pool)
	wgRepo := postgres.NewWorkingGroupRepository(pool)
	unitRepo := postgres.NewUnitRepository(pool)
	customerRepo := postgres.NewCustomerRepository(pool)
	orgMgmtRepo := postgres.NewOrganizationManagementRepository(pool)
	contractRepo := postgres.NewContractRepository(pool)
	activityRepo := postgres.NewActivityRepository(pool)
	ticketRepo := postgres.NewTicketRepository(pool)
	directionRepo := postgres.NewDirectionRepository(pool)
	coverageRepo := postgres.NewCoverageRepository(pool)
	orgSettingsRepo := postgres.NewOrgSettingsRepository(pool)
	exportRepo := postgres.NewExportRepository(pool)

	// Services
	invitationService := invitationsvc.NewService(invitationRepo)
	hexAuthService := authsvc.NewService(userRepo, orgRepo, tokenService, passwordHasher, refreshTokenRepo)
	passwordResetService := passwordresetsvc.NewService(passwordResetRepo, userRepo, userFinder, passwordHasher, auth.NewTokenService(authService), refreshTokenRepo)
	unitService := unitsvc.NewService(unitRepo)
	wgService := wgsvc.NewService(wgRepo)
	customerService := customersvc.NewService(customerRepo)
	orgMgmtService := orgsvc.NewService(orgMgmtRepo, customerService)
	contractService := contractsvc.NewService(contractRepo)
	routingSvc := routing.NewService(wgRepo, activityRepo, unitRepo)
	// CONCERNS.md #3: the time-entry service takes two ports
	// (TimeEntryRepository + TimeEntryApprovalRepository). The same concrete
	// *TimeEntryRepository satisfies both, so the approval role is injected as
	// an explicit, named dependency rather than reusing the entry variable
	// positionally — making the two roles unambiguous at the call site.
	timeEntryApprovalRepo := postgres.NewTimeEntryRepository(pool)
	teService := tesvc.NewService(timeEntryRepo, timeEntryApprovalRepo, wgRepo, activityRepo, unitRepo, routingSvc)
	activityService := activitysvc.NewService(activityRepo, contractRepo, unitRepo, orgRepo, ticketRepo, directionRepo, routingSvc)
	ticketService := ticketsvc.NewService(ticketRepo, activityRepo, contractRepo, orgRepo)
	coverageService := coveragesvc.NewService(coverageRepo, activityRepo, contractRepo, unitRepo, timeEntryRepo, routingSvc)
	orgSettingsService := orgsettingssvc.NewService(orgSettingsRepo, orgRepo)
	directionService := directionsvc.NewService(directionRepo, activityRepo, wgRepo, unitRepo, orgRepo, orgSettingsService, routingSvc)
	expenseService := expsvc.NewService(expenseRepo, wgRepo, activityRepo, unitRepo)
	exportService := exportsvc.NewService(exportRepo)

	// Handlers
	authHandler := http.NewAuthHandler(hexAuthService, invitationService)
	invitationHandler := http.NewInvitationHandler(invitationService)
	passwordResetHandler := http.NewPasswordResetHandler(passwordResetService)
	unitHandler := http.NewUnitHandler(unitService)
	wgHandler := http.NewWorkingGroupHandler(wgService)
	hexCustomerHandler := http.NewCustomerHandler(customerService)
	orgHandler := http.NewOrganizationHandler(orgMgmtService)
	contractHandler := http.NewContractHandler(contractService)
	activityHandler := http.NewActivityHandler(activityService, activityRepo)
	ticketHandler := http.NewTicketHandler(ticketService)
	coverageHandler := http.NewCoverageHandler(coverageService)
	directionHandler := http.NewDirectionHandler(directionService)
	teHandler := http.NewTimeEntryHandler(teService)
	expenseHandler := http.NewExpenseHandler(expenseService)
	exportHandler := http.NewExportHandler(exportService)
	orgSettingsHandler := http.NewOrgSettingsHandler(orgSettingsService)

	// Rate limiters (env-tunable, same knobs as before)
	rateLimit := 5
	if rl := os.Getenv("RATE_LIMIT"); rl != "" {
		if v, err := strconv.Atoi(rl); err == nil && v > 0 {
			rateLimit = v
		}
	}
	authRateLimiter := middleware.NewRateLimiter(rateLimit, 100)
	passwordResetRateLimiter := middleware.NewRateLimiter(3, 60)

	anonymousRateLimit := 20
	if rl := os.Getenv("ANONYMOUS_RATE_LIMIT"); rl != "" {
		if v, err := strconv.Atoi(rl); err == nil && v > 0 {
			anonymousRateLimit = v
		}
	}
	rateLimiter := middleware.NewRateLimiter(anonymousRateLimit, 100)

	return &appGraph{
		AuthService:               authService,
		AuthHandler:              authHandler,
		InvitationHandler:        invitationHandler,
		PasswordResetHandler:     passwordResetHandler,
		OrgHandler:               orgHandler,
		OrgSettingsHandler:       orgSettingsHandler,
		UnitHandler:              unitHandler,
		WGHandler:                wgHandler,
		CustomerHandler:          hexCustomerHandler,
		ContractHandler:          contractHandler,
		ActivityHandler:          activityHandler,
		TicketHandler:            ticketHandler,
		CoverageHandler:          coverageHandler,
		DirectionHandler:         directionHandler,
		TEHandler:                teHandler,
		ExpenseHandler:           expenseHandler,
		ExportHandler:            exportHandler,
		AuthRateLimiter:          authRateLimiter,
		PasswordResetRateLimiter: passwordResetRateLimiter,
		RateLimiter:              rateLimiter,
	}, nil
}
