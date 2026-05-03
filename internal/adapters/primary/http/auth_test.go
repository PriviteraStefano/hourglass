package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	hexauth "github.com/stefanoprivitera/hourglass/internal/adapters/secondary/surrealdb"
	"github.com/stefanoprivitera/hourglass/internal/auth"
	hexsvc "github.com/stefanoprivitera/hourglass/internal/core/services/auth"
	invitationsvc "github.com/stefanoprivitera/hourglass/internal/core/services/invitation"
	sdb "github.com/surrealdb/surrealdb.go"
)

func uniqueID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func uniqueEmail() string {
	return "test_" + uniqueID() + "@example.com"
}

func uniqueOrgName() string {
	return "Org_" + uniqueID()
}

type testServer struct {
	handler *AuthHandler
	server  *httptest.Server
	client  *http.Client
	db      *sdb.DB
}

func setupTestServer(t *testing.T) *testServer {
	ns := "test_" + uniqueID()
	db, err := hexauth.GetTestDBWithNamespace(ns, ns)
	if err != nil {
		t.Skipf("Failed to connect to SurrealDB (set SURREALDB_URL): %v", err)
	}
	t.Cleanup(func() {
		db.Close(context.Background())
	})

	userRepo := hexauth.NewUserRepository(db)
	orgRepo := hexauth.NewOrganizationRepository(db)
	passwordHasher := hexauth.NewPasswordHasher()

	jwtSecret := "test-secret"
	authService := auth.NewService(jwtSecret)
	tokenService := hexauth.NewTokenService(authService)
	refreshTokenRepo := hexauth.NewRefreshTokenRepository(db)
	invitationRepo := hexauth.NewInvitationRepository(db)

	authSvc := hexsvc.NewService(userRepo, orgRepo, tokenService, passwordHasher, refreshTokenRepo)
	invitationSvc := invitationsvc.NewService(invitationRepo)
	handler := NewAuthHandler(authSvc, invitationSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/register", handler.Register)
	mux.HandleFunc("POST /auth/login", handler.Login)
	mux.HandleFunc("POST /auth/logout", handler.Logout)
	mux.HandleFunc("POST /auth/refresh", handler.Refresh)
	mux.HandleFunc("GET /auth/me", handler.GetProfile)
	mux.HandleFunc("POST /auth/bootstrap", handler.Bootstrap)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	return &testServer{handler: handler, server: server, client: client, db: db}
}

func (ts *testServer) post(endpoint string, body map[string]string) (*http.Response, error) {
	jsonBody, _ := json.Marshal(body)
	return ts.client.Post(ts.server.URL+endpoint, "application/json", strings.NewReader(string(jsonBody)))
}

func (ts *testServer) get(endpoint string) (*http.Response, error) {
	return ts.client.Get(ts.server.URL + endpoint)
}

func TestRegister_WithNewOrg(t *testing.T) {
	ts := setupTestServer(t)

	body := map[string]string{
		"email":             uniqueEmail(),
		"username":          "user_" + uniqueID(),
		"firstname":         "John",
		"lastname":          "Doe",
		"password":          "password123",
		"organization_name": uniqueOrgName(),
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("bootstrap response body: %s", string(body))
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data object in register response")
	}
	user, ok := data["user"].(map[string]interface{})
	if !ok || user["email"] == nil {
		t.Error("expected user email in register response data")
	}
	if membership, ok := data["membership"].(map[string]interface{}); !ok || membership["organization_id"] == nil || membership["role"] != "employee" {
		t.Error("expected employee membership in register response data")
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	ts := setupTestServer(t)

	body := map[string]string{
		"email":             "notanemail",
		"password":          "password123",
		"organization_name": uniqueOrgName(),
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	ts := setupTestServer(t)

	body := map[string]string{
		"email":             uniqueEmail(),
		"password":          "short",
		"organization_name": uniqueOrgName(),
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestRegister_MissingOrgAndInvite(t *testing.T) {
	ts := setupTestServer(t)

	body := map[string]string{
		"email":    uniqueEmail(),
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	ts := setupTestServer(t)
	email := uniqueEmail()

	body := map[string]string{
		"email":             email,
		"password":          "password123",
		"organization_name": uniqueOrgName(),
	}
	jsonBody, _ := json.Marshal(body)

	resp1, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	resp1.Body.Close()

	body["username"] = "user2"
	jsonBody, _ = json.Marshal(body)
	resp2, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("expected status 409, got %d", resp2.StatusCode)
	}
}

func TestLogin_WithEmail_Success(t *testing.T) {
	ts := setupTestServer(t)
	email := uniqueEmail()
	username := "user_" + uniqueID()
	password := "password123"

	registerBody := map[string]string{
		"email":             email,
		"username":          username,
		"password":          password,
		"organization_name": uniqueOrgName(),
	}
	jsonBody, _ := json.Marshal(registerBody)
	resp, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	resp.Body.Close()

	loginBody := map[string]string{
		"identifier": email,
		"password":   password,
	}
	jsonBody, _ = json.Marshal(loginBody)

	req, _ := http.NewRequest("POST", ts.server.URL+"/auth/login", strings.NewReader(string(jsonBody)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = ts.client.Do(req)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		t.Logf("Login response: %+v", result)
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	ts := setupTestServer(t)

	loginBody := map[string]string{
		"identifier": "nonexistent@example.com",
		"password":   "password123",
	}
	jsonBody, _ := json.Marshal(loginBody)
	resp, err := http.Post(ts.server.URL+"/auth/login", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestLogin_NonExistentUser(t *testing.T) {
	ts := setupTestServer(t)
	email := uniqueEmail()
	password := "password123"

	registerBody := map[string]string{
		"email":             email,
		"password":          password,
		"organization_name": uniqueOrgName(),
	}
	jsonBody, _ := json.Marshal(registerBody)
	resp, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	resp.Body.Close()

	loginBody := map[string]string{
		"identifier": email,
		"password":   "wrongpassword",
	}
	jsonBody, _ = json.Marshal(loginBody)
	resp, err = http.Post(ts.server.URL+"/auth/login", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestLogout_WithRefreshToken(t *testing.T) {
	ts := setupTestServer(t)
	email := uniqueEmail()
	password := "password123"

	registerBody := map[string]string{
		"email":             email,
		"password":          password,
		"organization_name": uniqueOrgName(),
	}
	jsonBody, _ := json.Marshal(registerBody)
	resp, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	resp.Body.Close()

	loginBody := map[string]string{
		"identifier": email,
		"password":   password,
	}
	jsonBody, _ = json.Marshal(loginBody)
	resp, err = http.Post(ts.server.URL+"/auth/login", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	var refreshToken string
	for _, c := range resp.Cookies() {
		if c.Name == "refresh_token" {
			refreshToken = c.Value
			break
		}
	}
	resp.Body.Close()

	if refreshToken == "" {
		t.Fatal("no refresh token received")
	}

	logoutReq, _ := http.NewRequest("POST", ts.server.URL+"/auth/logout", nil)
	logoutReq.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	client := &http.Client{}
	resp, err = client.Do(logoutReq)
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", resp.StatusCode)
	}
}

func TestLogout_WithoutRefreshToken(t *testing.T) {
	ts := setupTestServer(t)

	logoutReq, _ := http.NewRequest("POST", ts.server.URL+"/auth/logout", nil)
	client := &http.Client{}
	resp, err := client.Do(logoutReq)
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", resp.StatusCode)
	}
}

func TestBootstrap_FirstUser(t *testing.T) {
	ts := setupTestServer(t)
	email := "bootstrap_" + uniqueID() + "@example.com"

	body := map[string]string{
		"organization_name": "Bootstrap Org",
		"email":             email,
		"username":          "admin_" + uniqueID(),
		"firstname":         "Admin",
		"lastname":          "User",
		"password":          "password123",
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(ts.server.URL+"/auth/bootstrap", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data object in bootstrap response")
	}
	if data["token"] == nil {
		t.Error("expected token in bootstrap response")
	}
	if membership, ok := data["membership"].(map[string]interface{}); !ok || membership["organization_id"] == nil || membership["role"] != "employee" {
		t.Error("expected employee membership in bootstrap response")
	}
}

func TestRefresh_ValidToken(t *testing.T) {
	ts := setupTestServer(t)
	email := uniqueEmail()
	password := "password123"

	registerBody := map[string]string{
		"email":             email,
		"password":          password,
		"organization_name": uniqueOrgName(),
	}
	jsonBody, _ := json.Marshal(registerBody)
	resp, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	resp.Body.Close()

	loginBody := map[string]string{
		"identifier": email,
		"password":   password,
	}
	jsonBody, _ = json.Marshal(loginBody)
	resp, err = http.Post(ts.server.URL+"/auth/login", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	var refreshToken string
	for _, c := range resp.Cookies() {
		if c.Name == "refresh_token" {
			refreshToken = c.Value
			break
		}
	}
	resp.Body.Close()

	if refreshToken == "" {
		t.Fatal("no refresh token received")
	}

	refreshReq, _ := http.NewRequest("POST", ts.server.URL+"/auth/refresh", nil)
	refreshReq.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	client := &http.Client{}
	resp, err = client.Do(refreshReq)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	ts := setupTestServer(t)

	refreshReq, _ := http.NewRequest("POST", ts.server.URL+"/auth/refresh", nil)
	refreshReq.AddCookie(&http.Cookie{Name: "refresh_token", Value: "invalid_token"})
	client := &http.Client{}
	resp, err := client.Do(refreshReq)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestRefresh_MissingCookie(t *testing.T) {
	ts := setupTestServer(t)

	refreshReq, _ := http.NewRequest("POST", ts.server.URL+"/auth/refresh", nil)
	client := &http.Client{}
	resp, err := client.Do(refreshReq)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestLogin_Username(t *testing.T) {
	ts := setupTestServer(t)
	email := uniqueEmail()
	username := "user_" + uniqueID()
	password := "password123"

	registerBody := map[string]string{
		"email":             email,
		"username":          username,
		"password":          password,
		"organization_name": uniqueOrgName(),
	}
	jsonBody, _ := json.Marshal(registerBody)
	resp, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	resp.Body.Close()

	loginBody := map[string]string{
		"identifier": username,
		"password":   password,
	}
	jsonBody, _ = json.Marshal(loginBody)
	resp, err = http.Post(ts.server.URL+"/auth/login", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestLogin_InvalidIdentifierFormat(t *testing.T) {
	ts := setupTestServer(t)

	loginBody := map[string]string{
		"identifier": "invalid@user!",
		"password":   "password123",
	}
	jsonBody, _ := json.Marshal(loginBody)
	resp, err := http.Post(ts.server.URL+"/auth/login", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestGetProfile_Authenticated(t *testing.T) {
	ts := setupTestServer(t)
	email := uniqueEmail()
	password := "password123"

	registerBody := map[string]string{
		"email":             email,
		"password":          password,
		"organization_name": uniqueOrgName(),
	}
	jsonBody, _ := json.Marshal(registerBody)
	resp, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	var token string
	for _, c := range resp.Cookies() {
		if c.Name == "auth_token" {
			token = c.Value
			break
		}
	}
	resp.Body.Close()

	if token == "" {
		t.Fatal("no auth token received")
	}

	profileReq, _ := http.NewRequest("GET", ts.server.URL+"/auth/me", nil)
	profileReq.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err = client.Do(profileReq)
	if err != nil {
		t.Fatalf("profile request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["email"] == nil {
		t.Error("expected email in response")
	}
}

func TestGetProfile_Unauthenticated(t *testing.T) {
	ts := setupTestServer(t)

	profileReq, _ := http.NewRequest("GET", ts.server.URL+"/auth/me", nil)
	client := &http.Client{}
	resp, err := client.Do(profileReq)
	if err != nil {
		t.Fatalf("profile request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestBootstrap_SubsequentUser(t *testing.T) {
	ts := setupTestServer(t)
	email1 := "bootstrap1_" + uniqueID() + "@example.com"

	body1 := map[string]string{
		"org_name":  "Bootstrap Org",
		"email":     email1,
		"username":  "admin1_" + uniqueID(),
		"firstname": "Admin",
		"lastname":  "User",
		"password":  "password123",
	}
	jsonBody, _ := json.Marshal(body1)
	resp1, err := http.Post(ts.server.URL+"/auth/bootstrap", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("first bootstrap failed: %v", err)
	}
	resp1.Body.Close()

	email2 := "bootstrap2_" + uniqueID() + "@example.com"
	body2 := map[string]string{
		"org_name":  "Bootstrap Org 2",
		"email":     email2,
		"username":  "admin2_" + uniqueID(),
		"firstname": "Admin",
		"lastname":  "User",
		"password":  "password123",
	}
	jsonBody, _ = json.Marshal(body2)
	resp2, err := http.Post(ts.server.URL+"/auth/bootstrap", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("second bootstrap failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("expected status 409, got %d", resp2.StatusCode)
	}
}

func TestRegister_DuplicateUsername(t *testing.T) {
	ts := setupTestServer(t)
	email1 := uniqueEmail()
	email2 := uniqueEmail()
	username := "user_" + uniqueID()

	body1 := map[string]string{
		"email":             email1,
		"username":          username,
		"password":          "password123",
		"organization_name": uniqueOrgName(),
	}
	jsonBody, _ := json.Marshal(body1)
	resp1, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	resp1.Body.Close()

	body2 := map[string]string{
		"email":             email2,
		"username":          username,
		"password":          "password123",
		"organization_name": "Test Org 2",
	}
	jsonBody, _ = json.Marshal(body2)
	resp2, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("second register failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("expected status 409, got %d", resp2.StatusCode)
	}
}

func TestLogin_DeactivatedAccount(t *testing.T) {
	ts := setupTestServer(t)
	email := uniqueEmail()
	password := "password123"

	registerBody := map[string]string{
		"email":            email,
		"username":         "user_" + uniqueID(),
		"password":         password,
		"organization_name": uniqueOrgName(),
	}
	jsonBody, _ := json.Marshal(registerBody)
	resp, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	resp.Body.Close()

	// Deactivate user using the test DB
	_, err = sdb.Query[any](context.Background(), ts.db, 
		"UPDATE users SET is_active = false WHERE email = $email", 
		map[string]any{"email": email})
	if err != nil {
		t.Fatalf("failed to deactivate user: %v", err)
	}

	loginBody := map[string]string{
		"identifier": email,
		"password":   password,
	}
	jsonBody, _ = json.Marshal(loginBody)
	resp, err = http.Post(ts.server.URL+"/auth/login", "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", resp.StatusCode)
	}
}
