package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_MissingCookie(t *testing.T) {
	authSvc := auth.NewService("test-secret")
	nextCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Auth(authSvc, handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, nextCalled, "next handler should not be called")
}

func TestAuth_InvalidToken(t *testing.T) {
	authSvc := auth.NewService("test-secret")
	nextCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Auth(authSvc, handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: "invalid-token-string"})
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, nextCalled, "next handler should not be called")
}

func TestAuth_ValidToken(t *testing.T) {
	authSvc := auth.NewService("test-secret")
	userID := uuid.New()
	orgID := uuid.New()

	token, err := authSvc.GenerateToken(userID, orgID, "employee", "test@example.com")
	require.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, userID, GetUserID(r.Context()), "userID should match")
		assert.Equal(t, orgID, GetOrganizationID(r.Context()), "orgID should match")
		assert.Equal(t, "employee", GetRole(r.Context()), "role should match")
		assert.Equal(t, "test@example.com", GetEmail(r.Context()), "email should match")
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Auth(authSvc, handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireRole_Allowed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RequireRole([]string{"manager", "finance"}, handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(SetRole(req.Context(), "manager"))
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireRole_Forbidden(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RequireRole([]string{"manager", "finance"}, handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(SetRole(req.Context(), "employee"))
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}
