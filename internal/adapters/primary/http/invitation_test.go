package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

func TestInvitationHandler_Create_InvalidBody(t *testing.T) {
	h := NewInvitationHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/invitations", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestInvitationHandler_Create_MissingOrgID(t *testing.T) {
	h := NewInvitationHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/invitations", strings.NewReader(`{"email":"test@example.com"}`))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestInvitationHandler_Create_InvalidOrgID(t *testing.T) {
	h := NewInvitationHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/invitations", strings.NewReader(`{"organization_id":"not-a-uuid","email":"test@example.com"}`))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestInvitationHandler_Accept_InvalidBody(t *testing.T) {
	h := NewInvitationHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/invitations/accept", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	req = req.WithContext(ctx)

	h.Accept(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestInvitationHandler_Accept_MissingToken(t *testing.T) {
	h := NewInvitationHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/invitations/accept", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	req = req.WithContext(ctx)

	h.Accept(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestInvitationHandler_Accept_WeakPassword(t *testing.T) {
	h := NewInvitationHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/invitations/accept", strings.NewReader(`{"token":"tok123","email":"test@example.com","password":"short"}`))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	req = req.WithContext(ctx)

	h.Accept(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
