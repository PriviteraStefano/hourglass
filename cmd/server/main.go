package main

import (
	"log"
	stdhttp "net/http"
	"os"
	"strings"

	"github.com/stefanoprivitera/hourglass/internal/adapters/primary/http"
	hexauth "github.com/stefanoprivitera/hourglass/internal/adapters/secondary/surrealdb"
	"github.com/stefanoprivitera/hourglass/internal/auth"
	hexsvc "github.com/stefanoprivitera/hourglass/internal/core/services/auth"
	invitationsvc "github.com/stefanoprivitera/hourglass/internal/core/services/invitation"
	passwordresetsvc "github.com/stefanoprivitera/hourglass/internal/core/services/password_reset"
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

	timeEntryHandler := handlers.NewSurrealTimeEntryHandler(sdbConn.DB())

	userRepo := hexauth.NewUserRepository(sdbConn.DB())
	orgRepo := hexauth.NewOrganizationRepository(sdbConn.DB())
	passwordHasher := hexauth.NewPasswordHasher()
	tokenService := hexauth.NewTokenService(authService)
	refreshTokenRepo := hexauth.NewRefreshTokenRepository(sdbConn.DB())

	hexAuthService := hexsvc.NewService(
		userRepo,
		orgRepo,
		tokenService,
		passwordHasher,
		refreshTokenRepo,
	)
	hexAuthHandler := http.NewAuthHandler(hexAuthService)

	mux.HandleFunc("POST /auth/register", hexAuthHandler.Register)
	mux.HandleFunc("POST /auth/login", hexAuthHandler.Login)
	mux.HandleFunc("POST /auth/logout", hexAuthHandler.Logout)
	mux.HandleFunc("POST /auth/refresh", hexAuthHandler.Refresh)
	mux.HandleFunc("GET /auth/me", middleware.Auth(authService, hexAuthHandler.GetProfile))
	mux.HandleFunc("POST /auth/bootstrap", hexAuthHandler.Bootstrap)

	invitationRepo := hexauth.NewInvitationRepository(sdbConn.DB())
	invitationService := invitationsvc.NewService(invitationRepo)
	hexInvitationHandler := http.NewInvitationHandler(invitationService)

	passwordResetRepo := hexauth.NewPasswordResetRepository(sdbConn.DB())
	userFinder := hexauth.NewUserFinder(sdbConn.DB())
	passwordResetService := passwordresetsvc.NewService(passwordResetRepo, userRepo, userFinder, passwordHasher, hexauth.NewTokenService(authService))
	hexPasswordResetHandler := http.NewPasswordResetHandler(passwordResetService)

	unitRepo := hexauth.NewUnitRepository(sdbConn.DB())
	unitService := unitsvc.NewService(unitRepo)
	hexUnitHandler := http.NewUnitHandler(unitService)

	wgRepo := hexauth.NewWorkingGroupRepository(sdbConn.DB())
	wgService := wgsvc.NewService(wgRepo)
	hexWGHandler := http.NewWorkingGroupHandler(wgService)

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

	mux.HandleFunc("GET /time-entries", middleware.Auth(authService, timeEntryHandler.List))
	mux.HandleFunc("POST /time-entries", middleware.Auth(authService, timeEntryHandler.Create))
	mux.HandleFunc("GET /time-entries/{id}", middleware.Auth(authService, timeEntryHandler.Get))
	mux.HandleFunc("PUT /time-entries/{id}", middleware.Auth(authService, timeEntryHandler.Update))
	mux.HandleFunc("DELETE /time-entries/{id}", middleware.Auth(authService, timeEntryHandler.Delete))
	mux.HandleFunc("POST /time-entries/{id}/submit", middleware.Auth(authService, timeEntryHandler.Submit))
	mux.HandleFunc("POST /time-entries/{id}/approve", middleware.Auth(authService, timeEntryHandler.Approve))
	mux.HandleFunc("POST /time-entries/{id}/reject", middleware.Auth(authService, timeEntryHandler.Reject))
	mux.HandleFunc("GET /time-entries/pending", middleware.Auth(authService, timeEntryHandler.ListPending))

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
