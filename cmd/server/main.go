package main

import (
	"context"
	"log"
	stdhttp "net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/stefanoprivitera/hourglass/internal/db"
	"github.com/stefanoprivitera/hourglass/internal/handlers"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

func main() {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		// Fail closed: never boot with the publicly-known dev secret unless an
		// operator explicitly opts in via ALLOW_INSECURE_AUTH=1. A misconfigured
		// or unset GO_ENV in production must NOT silently enable auth bypass
		// (CONCERNS.md #11).
		if os.Getenv("ALLOW_INSECURE_AUTH") == "1" {
			log.Println("WARNING: Using default insecure JWT_SECRET because ALLOW_INSECURE_AUTH=1 is set. Never enable this outside local development.")
			jwtSecret = "dev-secret-change-in-production"
		} else {
			log.Fatal("FATAL: JWT_SECRET is not set. Set JWT_SECRET in production, or set ALLOW_INSECURE_AUTH=1 to run with an insecure dev secret.")
		}
	}

	pool, err := db.NewPool()
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL pool: %v", err)
	}
	defer db.ClosePool(pool)
	log.Println("PostgreSQL pool initialized")

	g, err := buildGraph(pool, jwtSecret)
	if err != nil {
		log.Fatalf("Failed to build application graph: %v", err)
	}

	mux := stdhttp.NewServeMux()

	mux.HandleFunc("GET /health", handlers.NewHealthHandler().ServeHTTP)

	// Auth + account lifecycle
	mux.Handle("POST /auth/register", g.AuthRateLimiter.Middleware(stdhttp.HandlerFunc(g.AuthHandler.Register)))
	mux.Handle("POST /auth/login", g.AuthRateLimiter.Middleware(stdhttp.HandlerFunc(g.AuthHandler.Login)))
	mux.HandleFunc("POST /auth/logout", g.AuthHandler.Logout)
	mux.HandleFunc("POST /auth/refresh", g.AuthHandler.Refresh)
	mux.HandleFunc("GET /auth/me", middleware.Auth(g.AuthService, g.AuthHandler.GetProfile))
	mux.HandleFunc("POST /auth/bootstrap", g.AuthHandler.Bootstrap)
	mux.HandleFunc("GET /auth/bootstrap-check", g.AuthHandler.BootstrapCheck)
	mux.HandleFunc("POST /auth/switch-organization", middleware.Auth(g.AuthService, g.AuthHandler.SwitchOrganization))
	mux.HandleFunc("GET /auth/memberships", middleware.Auth(g.AuthService, g.AuthHandler.GetMemberships))

	mux.Handle("POST /auth/password-reset/request", g.PasswordResetRateLimiter.Middleware(stdhttp.HandlerFunc(g.PasswordResetHandler.Request)))
	mux.Handle("POST /auth/password-reset/verify", g.PasswordResetRateLimiter.Middleware(stdhttp.HandlerFunc(g.PasswordResetHandler.Verify)))

	mux.HandleFunc("POST /invitations", g.InvitationHandler.Create)
	mux.HandleFunc("GET /invitations/validate/code/{code}", g.InvitationHandler.ValidateCode)
	mux.HandleFunc("GET /invitations/validate/token/{token}", g.InvitationHandler.ValidateToken)
	mux.HandleFunc("POST /invitations/accept", g.InvitationHandler.Accept)

	// Units
	mux.HandleFunc("GET /units", middleware.Auth(g.AuthService, g.UnitHandler.List))
	mux.HandleFunc("POST /units", middleware.Auth(g.AuthService, g.UnitHandler.Create))
	mux.HandleFunc("GET /units/{id}", middleware.Auth(g.AuthService, g.UnitHandler.Get))
	mux.HandleFunc("PUT /units/{id}", middleware.Auth(g.AuthService, g.UnitHandler.Update))
	mux.HandleFunc("DELETE /units/{id}", middleware.Auth(g.AuthService, g.UnitHandler.Delete))
	mux.HandleFunc("GET /units/tree", middleware.Auth(g.AuthService, g.UnitHandler.GetTree))
	mux.HandleFunc("GET /units/{id}/descendants", middleware.Auth(g.AuthService, g.UnitHandler.GetDescendants))
	mux.HandleFunc("GET /units/{id}/members", middleware.Auth(g.AuthService, g.UnitHandler.ListMembers))
	mux.HandleFunc("POST /units/{id}/members", middleware.Auth(g.AuthService, g.UnitHandler.AddMember))
	mux.HandleFunc("DELETE /units/{id}/members/{membership_id}", middleware.Auth(g.AuthService, g.UnitHandler.RemoveMember))
	mux.HandleFunc("PUT /units/{id}/members/{membership_id}", middleware.Auth(g.AuthService, g.UnitHandler.UpdateMember))
	mux.HandleFunc("GET /units/members/batch", middleware.Auth(g.AuthService, g.UnitHandler.ListMembersBatch))

	// Working groups
	mux.HandleFunc("GET /working-groups", middleware.Auth(g.AuthService, g.WGHandler.List))
	mux.HandleFunc("POST /working-groups", middleware.Auth(g.AuthService, g.WGHandler.Create))
	mux.HandleFunc("GET /working-groups/{id}", middleware.Auth(g.AuthService, g.WGHandler.Get))
	mux.HandleFunc("PUT /working-groups/{id}", middleware.Auth(g.AuthService, g.WGHandler.Update))
	mux.HandleFunc("DELETE /working-groups/{id}", middleware.Auth(g.AuthService, g.WGHandler.Delete))
	mux.HandleFunc("GET /working-groups/{id}/members", middleware.Auth(g.AuthService, g.WGHandler.ListMembers))
	mux.HandleFunc("POST /working-groups/{id}/members", middleware.Auth(g.AuthService, g.WGHandler.AddMember))
	mux.HandleFunc("DELETE /working-groups/{id}/members/{member_id}", middleware.Auth(g.AuthService, g.WGHandler.RemoveMember))

	// Customers
	mux.HandleFunc("GET /customers", middleware.Auth(g.AuthService, g.CustomerHandler.List))
	mux.HandleFunc("POST /customers", middleware.Auth(g.AuthService, g.CustomerHandler.Create))
	mux.HandleFunc("GET /customers/{id}", middleware.Auth(g.AuthService, g.CustomerHandler.Get))
	mux.HandleFunc("PUT /customers/{id}", middleware.Auth(g.AuthService, g.CustomerHandler.Update))
	mux.HandleFunc("DELETE /customers/{id}", middleware.Auth(g.AuthService, g.CustomerHandler.Delete))

	// Organizations
	mux.HandleFunc("POST /organizations", middleware.Auth(g.AuthService, g.OrgHandler.Create))
	mux.HandleFunc("GET /organizations/{id}", middleware.Auth(g.AuthService, g.OrgHandler.Get))
	mux.HandleFunc("POST /organizations/invite", middleware.Auth(g.AuthService, g.OrgHandler.Invite))
	mux.HandleFunc("POST /organizations/invite-customer", middleware.Auth(g.AuthService, g.OrgHandler.InviteCustomer))
	mux.HandleFunc("GET /organizations/{id}/settings", middleware.Auth(g.AuthService, g.OrgHandler.GetSettings))
	mux.HandleFunc("PUT /organizations/{id}/settings", middleware.Auth(g.AuthService, g.OrgHandler.UpdateSettings))
	mux.HandleFunc("GET /organizations/settings", middleware.Auth(g.AuthService, g.OrgSettingsHandler.Get))
	mux.HandleFunc("PUT /organizations/settings", middleware.Auth(g.AuthService, g.OrgSettingsHandler.Put))
	mux.HandleFunc("GET /organizations/members", middleware.Auth(g.AuthService, g.OrgHandler.ListMembers))
	mux.HandleFunc("PUT /organizations/members/{member_id}/roles", middleware.Auth(g.AuthService, g.OrgHandler.UpdateMemberRoles))
	mux.HandleFunc("DELETE /organizations/members/{member_id}", middleware.Auth(g.AuthService, g.OrgHandler.DeactivateMember))

	// Activities
	mux.HandleFunc("GET /activities", middleware.Auth(g.AuthService, g.ActivityHandler.List))
	mux.HandleFunc("POST /activities", middleware.Auth(g.AuthService, g.ActivityHandler.Create))
	mux.HandleFunc("GET /activities/{id}", middleware.Auth(g.AuthService, g.ActivityHandler.Get))
	mux.HandleFunc("PUT /activities/{id}", middleware.Auth(g.AuthService, g.ActivityHandler.Update))
	mux.HandleFunc("DELETE /activities/{id}", middleware.Auth(g.AuthService, g.ActivityHandler.Delete))
	mux.HandleFunc("GET /activities/{id}/children", middleware.Auth(g.AuthService, g.ActivityHandler.ListChildren))
	mux.HandleFunc("POST /activities/{id}/approve-proposal", middleware.Auth(g.AuthService, g.ActivityHandler.ApproveProposal))
	mux.HandleFunc("GET /activity-kinds", middleware.Auth(g.AuthService, g.ActivityHandler.ListKinds))

	// Tickets
	mux.HandleFunc("POST /tickets", middleware.Auth(g.AuthService, g.TicketHandler.Create))
	mux.HandleFunc("GET /tickets", middleware.Auth(g.AuthService, g.TicketHandler.List))
	mux.HandleFunc("GET /tickets/{id}", middleware.Auth(g.AuthService, g.TicketHandler.Get))
	mux.HandleFunc("PUT /tickets/{id}", middleware.Auth(g.AuthService, g.TicketHandler.Update))
	mux.HandleFunc("POST /tickets/{id}/triage", middleware.Auth(g.AuthService, g.TicketHandler.Triage))
	mux.HandleFunc("POST /tickets/{id}/dismiss", middleware.Auth(g.AuthService, g.TicketHandler.Dismiss))
	mux.HandleFunc("POST /tickets/{id}/transition", middleware.Auth(g.AuthService, g.TicketHandler.Transition))
	mux.HandleFunc("POST /tickets/{id}/comments", middleware.Auth(g.AuthService, g.TicketHandler.AddComment))
	mux.HandleFunc("GET /tickets/{id}/history", middleware.Auth(g.AuthService, g.TicketHandler.History))

	// Coverage
	mux.HandleFunc("PUT /time-entries/{id}/allocations", middleware.Auth(g.AuthService, g.CoverageHandler.PutAllocations))
	mux.HandleFunc("GET /time-entries/{id}/allocations", middleware.Auth(g.AuthService, g.CoverageHandler.GetAllocations))
	mux.HandleFunc("GET /coverage/proposals/{entry_id}", middleware.Auth(g.AuthService, g.CoverageHandler.GetProposal))
	mux.HandleFunc("GET /coverage/own/{entry_id}", middleware.Auth(g.AuthService, g.CoverageHandler.GetOwn))
	mux.HandleFunc("GET /coverage/to-cover", middleware.Auth(g.AuthService, g.CoverageHandler.GetToCoverQueue))
	mux.HandleFunc("GET /coverage/buckets/{contract_id}/balance", middleware.Auth(g.AuthService, g.CoverageHandler.GetBucketBalance))
	mux.HandleFunc("POST /coverage/close", middleware.Auth(g.AuthService, g.CoverageHandler.PostClose))
	mux.HandleFunc("GET /coverage/snapshots/{close_id}", middleware.Auth(g.AuthService, g.CoverageHandler.GetSnapshot))
	mux.HandleFunc("GET /coverage/allocations/{entry_id}/history", middleware.Auth(g.AuthService, g.CoverageHandler.GetHistory))

	// Direction
	mux.HandleFunc("POST /direction", middleware.Auth(g.AuthService, g.DirectionHandler.Create))
	mux.HandleFunc("POST /direction/{id}/activate", middleware.Auth(g.AuthService, g.DirectionHandler.Activate))
	mux.HandleFunc("POST /direction/{id}/cancel", middleware.Auth(g.AuthService, g.DirectionHandler.Cancel))
	mux.HandleFunc("POST /direction/claims", middleware.Auth(g.AuthService, g.DirectionHandler.Claim))
	mux.HandleFunc("POST /direction/claims/{id}/cancel", middleware.Auth(g.AuthService, g.DirectionHandler.Unclaim))
	mux.HandleFunc("GET /direction", middleware.Auth(g.AuthService, g.DirectionHandler.ListPlan))
	mux.HandleFunc("GET /direction/coverage", middleware.Auth(g.AuthService, g.DirectionHandler.Coverage))

	// Contracts
	mux.HandleFunc("GET /contracts", middleware.Auth(g.AuthService, g.ContractHandler.List))
	mux.HandleFunc("POST /contracts", middleware.Auth(g.AuthService, g.ContractHandler.Create))
	mux.HandleFunc("GET /contracts/{id}", middleware.Auth(g.AuthService, g.ContractHandler.Get))
	mux.HandleFunc("POST /contracts/{id}/adopt", middleware.Auth(g.AuthService, g.ContractHandler.Adopt))
	mux.HandleFunc("PUT /contracts/{id}", middleware.Auth(g.AuthService, g.ContractHandler.Update))
	mux.HandleFunc("POST /contracts/{id}/recalculate-mileage", middleware.Auth(g.AuthService, g.ContractHandler.RecalculateMileage))
	mux.HandleFunc("DELETE /contracts/{id}", middleware.Auth(g.AuthService, g.ContractHandler.Delete))

	// Exports
	mux.HandleFunc("GET /exports/timesheets", middleware.Auth(g.AuthService, g.ExportHandler.Timesheets))
	mux.HandleFunc("GET /exports/expenses", middleware.Auth(g.AuthService, g.ExportHandler.Expenses))
	mux.HandleFunc("GET /exports/combined", middleware.Auth(g.AuthService, g.ExportHandler.Combined))
	mux.HandleFunc("GET /exports/timesheets/count", middleware.Auth(g.AuthService, g.ExportHandler.CountTimesheets))
	mux.HandleFunc("GET /exports/expenses/count", middleware.Auth(g.AuthService, g.ExportHandler.CountExpenses))
	mux.HandleFunc("GET /exports/combined/count", middleware.Auth(g.AuthService, g.ExportHandler.CountCombined))

	// Time entries
	mux.HandleFunc("GET /time-entries", middleware.Auth(g.AuthService, g.TEHandler.List))
	mux.HandleFunc("POST /time-entries", middleware.Auth(g.AuthService, g.TEHandler.Create))
	mux.HandleFunc("GET /time-entries/{id}", middleware.Auth(g.AuthService, g.TEHandler.Get))
	mux.HandleFunc("PUT /time-entries/{id}", middleware.Auth(g.AuthService, g.TEHandler.Update))
	mux.HandleFunc("DELETE /time-entries/{id}", middleware.Auth(g.AuthService, g.TEHandler.Delete))
	mux.HandleFunc("POST /time-entries/{id}/submit", middleware.Auth(g.AuthService, g.TEHandler.Submit))
	mux.HandleFunc("POST /time-entries/{id}/approve", middleware.Auth(g.AuthService, g.TEHandler.Approve))
	mux.HandleFunc("POST /time-entries/{id}/reject", middleware.Auth(g.AuthService, g.TEHandler.Reject))
	mux.HandleFunc("GET /time-entries/pending", middleware.Auth(g.AuthService, g.TEHandler.ListPending))

	// Expenses
	mux.HandleFunc("GET /expenses", middleware.Auth(g.AuthService, g.ExpenseHandler.List))
	mux.HandleFunc("POST /expenses", middleware.Auth(g.AuthService, g.ExpenseHandler.Create))
	mux.HandleFunc("GET /expenses/{id}", middleware.Auth(g.AuthService, g.ExpenseHandler.Get))
	mux.HandleFunc("PUT /expenses/{id}", middleware.Auth(g.AuthService, g.ExpenseHandler.Update))
	mux.HandleFunc("DELETE /expenses/{id}", middleware.Auth(g.AuthService, g.ExpenseHandler.Delete))
	mux.HandleFunc("POST /expenses/{id}/submit", middleware.Auth(g.AuthService, g.ExpenseHandler.Submit))
	mux.HandleFunc("POST /expenses/{id}/approve", middleware.Auth(g.AuthService, g.ExpenseHandler.Approve))
	mux.HandleFunc("POST /expenses/{id}/reject", middleware.Auth(g.AuthService, g.ExpenseHandler.Reject))
	mux.HandleFunc("GET /expenses/pending", middleware.Auth(g.AuthService, g.ExpenseHandler.ListPending))
	mux.HandleFunc("POST /expenses/{id}/receipt", middleware.Auth(g.AuthService, g.ExpenseHandler.ReceiptUpload))

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

	// Body-size cap (CONCERNS.md #8): 1 MB default for JSON endpoints, tunable
	// via MAX_BODY_BYTES; multipart (receipt uploads) keeps its own 10 MB limit.
	maxBodyBytes := int64(1 << 20)
	if v := os.Getenv("MAX_BODY_BYTES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxBodyBytes = n
		}
	}

	log.Printf("Server starting on port %s", port)
	handler := middleware.Recovery(middleware.MaxBody(maxBodyBytes)(middleware.TryAuth(g.AuthService, g.RateLimiter.Middleware(middleware.Logging(middleware.CORS(allowedOrigins)(mux))))))

	srv := &stdhttp.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM: drain in-flight requests instead of
	// dropping them (CONCERNS.md #14).
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != stdhttp.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
