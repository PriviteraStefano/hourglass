package main

import (
	"log"
	"net/http"
	"os"

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

	mux := http.NewServeMux()

	mux.HandleFunc("POST /auth/register", userHandler.Register)
	mux.HandleFunc("POST /auth/login", userHandler.Login)
	mux.HandleFunc("POST /auth/logout", userHandler.Logout)
	mux.HandleFunc("POST /auth/activate", userHandler.Activate)

	mux.HandleFunc("POST /organizations", middleware.Auth(authService, orgHandler.Create))
	mux.HandleFunc("GET /organizations/{id}", middleware.Auth(authService, orgHandler.Get))
	mux.HandleFunc("POST /organizations/{id}/invite", middleware.Auth(authService, orgHandler.Invite))

	mux.HandleFunc("GET /contracts", middleware.Auth(authService, contractHandler.List))
	mux.HandleFunc("POST /contracts", middleware.Auth(authService, contractHandler.Create))
	mux.HandleFunc("GET /contracts/{id}", middleware.Auth(authService, contractHandler.Get))
	mux.HandleFunc("POST /contracts/{id}/adopt", middleware.Auth(authService, contractHandler.Adopt))

	mux.HandleFunc("GET /projects", middleware.Auth(authService, projectHandler.List))
	mux.HandleFunc("POST /projects", middleware.Auth(authService, projectHandler.Create))
	mux.HandleFunc("GET /projects/{id}", middleware.Auth(authService, projectHandler.Get))
	mux.HandleFunc("POST /projects/{id}/adopt", middleware.Auth(authService, projectHandler.Adopt))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, corsMiddleware(mux)); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
