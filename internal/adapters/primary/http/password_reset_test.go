package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

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
