package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Handler-level validation tests (no DB needed)
// ---------------------------------------------------------------------------

func TestPasswordResetHandler_Request_InvalidBody(t *testing.T) {
	h := NewPasswordResetHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/password-reset/request", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	req = req.WithContext(ctx)

	h.Request(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestPasswordResetHandler_Request_MissingIdentifier(t *testing.T) {
	h := NewPasswordResetHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/password-reset/request", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	req = req.WithContext(ctx)

	h.Request(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestPasswordResetHandler_Verify_InvalidBody(t *testing.T) {
	h := NewPasswordResetHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/password-reset/verify", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	req = req.WithContext(ctx)

	h.Verify(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestPasswordResetHandler_Verify_MissingFields(t *testing.T) {
	h := NewPasswordResetHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/password-reset/verify", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	req = req.WithContext(ctx)

	h.Verify(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestPasswordResetHandler_Verify_WeakPassword(t *testing.T) {
	h := NewPasswordResetHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/password-reset/verify", strings.NewReader(`{"identifier":"test@example.com","code":"123456","password":"short"}`))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	req = req.WithContext(ctx)

	h.Verify(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Integration tests against testcontainers-backed PostgreSQL
// ---------------------------------------------------------------------------

func TestPasswordResetIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("RequestWithValidEmail_Returns200", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := fmt.Sprintf("pwreset-%s@test.com", uuid.New().String()[:8])

		// Register user first
		regBody := fmt.Sprintf(`{"email":"%s","username":"pwrusr","password":"TestPass123!","organization_name":"PWResetOrg"}`, email)
		regResp, err := f.Client.Post(f.ServerURL+"/auth/register", "application/json", strings.NewReader(regBody))
		require.NoError(t, err)
		regResp.Body.Close()

		// Request password reset
		body := fmt.Sprintf(`{"identifier":"%s"}`, email)
		resp, err := f.Client.Post(f.ServerURL+"/auth/password-reset/request", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var wrapped struct {
			Data map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&wrapped)
		require.NoError(t, err)

		result := wrapped.Data
		// Verify "code" field is NOT in the response (per D-16)
		_, hasCode := result["code"]
		assert.False(t, hasCode, "password reset response should NOT contain 'code' field")

		// Verify "message" field is present
		message, hasMessage := result["message"].(string)
		assert.True(t, hasMessage, "response should have 'message' field")
		assert.Equal(t, "reset code sent", message)

		// Verify "expires_at" field is present
		_, hasExpiresAt := result["expires_at"]
		assert.True(t, hasExpiresAt, "response should have 'expires_at' field")
	})

	t.Run("RequestWithUnknownEmail_Returns404", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		body := fmt.Sprintf(`{"identifier":"%s"}`, "unknown-"+uuid.New().String()+"@test.com")
		resp, err := f.Client.Post(f.ServerURL+"/auth/password-reset/request", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("VerifyWithWrongCode_Returns401", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := fmt.Sprintf("pwreset-wrong-%s@test.com", uuid.New().String()[:8])

		// Register user
		regBody := fmt.Sprintf(`{"email":"%s","username":"pwwrong","password":"TestPass123!","organization_name":"PWWrongOrg"}`, email)
		regResp, err := f.Client.Post(f.ServerURL+"/auth/register", "application/json", strings.NewReader(regBody))
		require.NoError(t, err)
		regResp.Body.Close()

		// Request password reset
		reqBody := fmt.Sprintf(`{"identifier":"%s"}`, email)
		resp, err := f.Client.Post(f.ServerURL+"/auth/password-reset/request", "application/json", strings.NewReader(reqBody))
		require.NoError(t, err)
		resp.Body.Close()

		// Verify with wrong code
		verifyBody := fmt.Sprintf(`{"identifier":"%s","code":"WRONGCODE","password":"NewPass123!"}`, email)
		verifyResp, err := f.Client.Post(f.ServerURL+"/auth/password-reset/verify", "application/json", strings.NewReader(verifyBody))
		require.NoError(t, err)
		defer verifyResp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, verifyResp.StatusCode, "wrong code should return 401")
	})
}
