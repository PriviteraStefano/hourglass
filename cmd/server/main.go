package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/stefanoprivitera/hourglass/internal/auth"
	"github.com/stefanoprivitera/hourglass/internal/db"
	"github.com/stefanoprivitera/hourglass/internal/handlers"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}

	database, err := db.New(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	authService := auth.NewService(jwtSecret)

	userHandler := handlers.NewUserHandler(database.DB, authService)
	orgHandler := handlers.NewOrganizationHandler(database.DB, authService)
	contractHandler := handlers.NewContractHandler(database.DB)
	projectHandler := handlers.NewProjectHandler(database.DB)
	timeEntryHandler := handlers.NewTimeEntryHandler(database.DB)
	expenseHandler := handlers.NewExpenseHandler(database.DB)
	approvalHandler := handlers.NewApprovalHandler(database.DB)
	healthHandler := handlers.NewHealthHandler()
	customerHandler := handlers.NewCustomerHandler(database.DB)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler.ServeHTTP)

	mux.HandleFunc("POST /auth/register", userHandler.Register)
	mux.HandleFunc("POST /auth/verify", userHandler.Verify)
	mux.HandleFunc("POST /auth/forgot-password", userHandler.ForgotPassword)
	mux.HandleFunc("POST /auth/reset-password", userHandler.ResetPassword)
	mux.HandleFunc("POST /auth/login", userHandler.Login)
	mux.HandleFunc("POST /auth/logout", userHandler.Logout)
	mux.HandleFunc("POST /auth/activate", userHandler.Activate)
	mux.HandleFunc("POST /auth/refresh", userHandler.Refresh)
	mux.HandleFunc("GET /auth/me", middleware.Auth(authService, userHandler.GetProfile))

	mux.HandleFunc("POST /organizations", middleware.Auth(authService, orgHandler.Create))
	mux.HandleFunc("GET /organizations/{id}", middleware.Auth(authService, orgHandler.Get))
	mux.HandleFunc("GET /organizations/{id}/settings", middleware.Auth(authService, orgHandler.GetSettings))
	mux.HandleFunc("PUT /organizations/{id}/settings", middleware.Auth(authService, orgHandler.UpdateSettings))
	mux.HandleFunc("POST /organizations/{id}/invite", middleware.Auth(authService, orgHandler.Invite))

	mux.HandleFunc("GET /customers", middleware.Auth(authService, customerHandler.List))
	mux.HandleFunc("POST /customers", middleware.Auth(authService, customerHandler.Create))
	mux.HandleFunc("GET /customers/{id}", middleware.Auth(authService, customerHandler.Get))
	mux.HandleFunc("PUT /customers/{id}", middleware.Auth(authService, customerHandler.Update))
	mux.HandleFunc("DELETE /customers/{id}", middleware.Auth(authService, customerHandler.Delete))

	mux.HandleFunc("GET /contracts", middleware.Auth(authService, contractHandler.List))
	mux.HandleFunc("POST /contracts", middleware.Auth(authService, contractHandler.Create))
	mux.HandleFunc("GET /contracts/{id}", middleware.Auth(authService, contractHandler.Get))
	mux.HandleFunc("POST /contracts/{id}/adopt", middleware.Auth(authService, contractHandler.Adopt))

	mux.HandleFunc("GET /projects", middleware.Auth(authService, projectHandler.List))
	mux.HandleFunc("POST /projects", middleware.Auth(authService, projectHandler.Create))
	mux.HandleFunc("GET /projects/{id}", middleware.Auth(authService, projectHandler.Get))
	mux.HandleFunc("POST /projects/{id}/adopt", middleware.Auth(authService, projectHandler.Adopt))
	mux.HandleFunc("GET /projects/{id}/managers", middleware.Auth(authService, projectHandler.ListManagers))
	mux.HandleFunc("POST /projects/{id}/managers", middleware.Auth(authService, projectHandler.AddManager))
	mux.HandleFunc("DELETE /projects/{id}/managers/{user_id}", middleware.Auth(authService, projectHandler.RemoveManager))

	mux.HandleFunc("GET /time-entries", middleware.Auth(authService, timeEntryHandler.List))
	mux.HandleFunc("POST /time-entries", middleware.Auth(authService, timeEntryHandler.Create))
	mux.HandleFunc("GET /time-entries/{id}", middleware.Auth(authService, timeEntryHandler.Get))
	mux.HandleFunc("PUT /time-entries/{id}", middleware.Auth(authService, timeEntryHandler.Update))
	mux.HandleFunc("DELETE /time-entries/{id}", middleware.Auth(authService, timeEntryHandler.Delete))
	mux.HandleFunc("GET /time-entries/monthly-summary", middleware.Auth(authService, timeEntryHandler.MonthlySummary))

	mux.HandleFunc("GET /expenses", middleware.Auth(authService, expenseHandler.List))
	mux.HandleFunc("POST /expenses", middleware.Auth(authService, expenseHandler.Create))
	mux.HandleFunc("GET /expenses/{id}", middleware.Auth(authService, expenseHandler.Get))
	mux.HandleFunc("PUT /expenses/{id}", middleware.Auth(authService, expenseHandler.Update))
	mux.HandleFunc("DELETE /expenses/{id}", middleware.Auth(authService, expenseHandler.Delete))
	mux.HandleFunc("GET /expenses/monthly-summary", middleware.Auth(authService, expenseHandler.MonthlySummary))
	mux.HandleFunc("GET /expenses/receipts/{id}", middleware.Auth(authService, expenseHandler.GetReceipt))

	mux.HandleFunc("POST /time-entries/{id}/submit", middleware.Auth(authService, approvalHandler.SubmitTimeEntry))
	mux.HandleFunc("POST /time-entries/submit-month", middleware.Auth(authService, approvalHandler.SubmitTimeEntryMonth))
	mux.HandleFunc("POST /expenses/{id}/submit", middleware.Auth(authService, approvalHandler.SubmitExpense))
	mux.HandleFunc("POST /expenses/submit-month", middleware.Auth(authService, approvalHandler.SubmitExpenseMonth))

	mux.HandleFunc("GET /time-entries/pending-approval", middleware.Auth(authService, approvalHandler.GetPendingTimeEntries))
	mux.HandleFunc("GET /expenses/pending-approval", middleware.Auth(authService, approvalHandler.GetPendingExpenses))

	mux.HandleFunc("POST /time-entries/{id}/approve", middleware.Auth(authService, approvalHandler.ApproveTimeEntry))
	mux.HandleFunc("POST /time-entries/{id}/reject", middleware.Auth(authService, approvalHandler.RejectTimeEntry))
	mux.HandleFunc("POST /expenses/{id}/approve", middleware.Auth(authService, approvalHandler.ApproveExpense))
	mux.HandleFunc("POST /expenses/{id}/reject", middleware.Auth(authService, approvalHandler.RejectExpense))

	mux.HandleFunc("POST /time-entries/{id}/edit-approve", middleware.Auth(authService, approvalHandler.EditApproveTimeEntry))
	mux.HandleFunc("POST /time-entries/{id}/edit-return", middleware.Auth(authService, approvalHandler.EditReturnTimeEntry))
	mux.HandleFunc("POST /expenses/{id}/edit-approve", middleware.Auth(authService, approvalHandler.EditApproveExpense))
	mux.HandleFunc("POST /expenses/{id}/edit-return", middleware.Auth(authService, approvalHandler.EditReturnExpense))

	mux.HandleFunc("POST /time-entries/{id}/partial-approve", middleware.Auth(authService, approvalHandler.PartialApproveTimeEntry))
	mux.HandleFunc("POST /time-entries/{id}/delegate", middleware.Auth(authService, approvalHandler.DelegateTimeEntry))
	mux.HandleFunc("POST /time-entries/batch-approve", middleware.Auth(authService, approvalHandler.BatchApproveTimeEntries))
	mux.HandleFunc("POST /time-entries/batch-reject", middleware.Auth(authService, approvalHandler.BatchRejectTimeEntries))
	mux.HandleFunc("POST /expenses/{id}/partial-approve", middleware.Auth(authService, approvalHandler.PartialApproveExpense))
	mux.HandleFunc("POST /expenses/{id}/delegate", middleware.Auth(authService, approvalHandler.DelegateExpense))
	mux.HandleFunc("POST /expenses/batch-approve", middleware.Auth(authService, approvalHandler.BatchApproveExpenses))
	mux.HandleFunc("POST /expenses/batch-reject", middleware.Auth(authService, approvalHandler.BatchRejectExpenses))

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
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
