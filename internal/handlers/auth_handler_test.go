package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stefanoprivitera/hourglass/internal/auth"
	"github.com/stefanoprivitera/hourglass/internal/db"
)

func uniqueID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Intn(99999))
}

func TestSurrealAuthHandler_Register(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	sdb, err := db.NewSurrealDB()
	if err != nil {
		t.Fatalf("Failed to connect to SurrealDB: %v", err)
	}
	defer sdb.Close()

	authService := auth.NewService("test-secret-key")
	handler := NewAuthHandler(sdb, authService)

	tests := []struct {
		name       string
		payload    interface{}
		wantStatus int
	}{
		{
			name: "successful registration",
			payload: RegisterRequest{
				Email:    "test@example.com",
				Name:     "Test User",
				Password: "password123",
				OrgName:  "Test Org",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "missing email",
			payload: RegisterRequest{
				Name:     "Test User",
				Password: "password123",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			payload: RegisterRequest{
				Email: "test2@example.com",
				Name:  "Test User",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "short password",
			payload: RegisterRequest{
				Email:    "test3@example.com",
				Name:     "Test User",
				Password: "short",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.Register(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestSurrealAuthHandler_Login(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	sdb, err := db.NewSurrealDB()
	if err != nil {
		t.Fatalf("Failed to connect to SurrealDB: %v", err)
	}
	defer sdb.Close()

	authService := auth.NewService("test-secret-key")
	handler := NewAuthHandler(sdb, authService)

	// First, register a user
	registerPayload := RegisterRequest{
		Email:    "login-test-" + uniqueID() + "@example.com",
		Name:     "Login Test User",
		Password: "password123",
	}
	body, _ := json.Marshal(registerPayload)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Failed to register user: %s", rec.Body.String())
	}

	// Now test login
	loginIdentifier := registerPayload.Email // Use the email we just registered with
	tests := []struct {
		name       string
		payload    interface{}
		wantStatus int
	}{
		{
			name: "successful login",
			payload: SurrealLoginRequest{
				Identifier: loginIdentifier,
				Password:   "password123",
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "wrong password",
			payload: SurrealLoginRequest{
				Identifier: loginIdentifier,
				Password:   "wrongpassword",
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "non-existent user",
			payload: SurrealLoginRequest{
				Identifier: "nonexistent@example.com",
				Password:   "password123",
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "missing credentials",
			payload: SurrealLoginRequest{
				Identifier: "",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.Login(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			// For successful login, verify token is returned
			if tt.wantStatus == http.StatusOK {
				var responseBody map[string]interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
					t.Errorf("failed to parse response: %v", err)
				}
				data, ok := responseBody["data"].(map[string]interface{})
				if !ok {
					t.Errorf("expected data in response, got: %+v", responseBody)
				}
				if data != nil && data["token"] == "" {
					t.Error("expected token in response")
				}
				if data != nil && data["refresh_token"] == "" {
					t.Error("expected refresh_token in response")
				}
			}
		})
	}
}

func TestSurrealAuthHandler_LoginWithUsername(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	sdb, err := db.NewSurrealDB()
	if err != nil {
		t.Fatalf("Failed to connect to SurrealDB: %v", err)
	}
	defer sdb.Close()

	authService := auth.NewService("test-secret-key")
	handler := NewAuthHandler(sdb, authService)

	// Register a user with username
	registerPayload := RegisterRequest{
		Email:     "user-login-" + uniqueID() + "@example.com",
		Username:  "user-" + uniqueID(),
		Firstname: "Test",
		Lastname:  "User",
		Password:  "password123",
	}
	body, _ := json.Marshal(registerPayload)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Failed to register user: %s", rec.Body.String())
	}

	// Login with username
	loginPayload := SurrealLoginRequest{
		Identifier: registerPayload.Username,
		Password:   "password123",
	}
	body, _ = json.Marshal(loginPayload)
	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()

	handler.Login(rec, req)

	t.Logf("Response status: %d, body: %s", rec.Code, rec.Body.String())

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var responseBody map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Errorf("failed to parse response: %v", err)
	}
	data, ok := responseBody["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data in response, got: %+v", responseBody)
	}
	if data["token"] == "" {
		t.Error("expected token in response")
	}
}

func TestSurrealAuthHandler_RefreshToken(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	sdb, err := db.NewSurrealDB()
	if err != nil {
		t.Fatalf("Failed to connect to SurrealDB: %v", err)
	}
	defer sdb.Close()

	authService := auth.NewService("test-secret-key")
	handler := NewAuthHandler(sdb, authService)

	// Register and login
	registerPayload := RegisterRequest{
		Email:    "refresh-test@example.com",
		Name:     "Refresh Test User",
		Password: "password123",
	}
	body, _ := json.Marshal(registerPayload)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.Register(rec, req)

	// Login
	loginPayload := SurrealLoginRequest{
		Identifier: "refresh-test@example.com",
		Password:   "password123",
	}
	body, _ = json.Marshal(loginPayload)
	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.Login(rec, req)

	var loginResponse SurrealLoginResponse
	json.Unmarshal(rec.Body.Bytes(), &loginResponse)

	// Test refresh
	refreshPayload := map[string]string{
		"refresh_token": loginResponse.RefreshToken,
	}
	body, _ = json.Marshal(refreshPayload)
	req = httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()

	handler.Refresh(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var refreshResponse map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &refreshResponse)
	if refreshResponse["token"] == "" {
		t.Error("expected new token in refresh response")
	}
}
