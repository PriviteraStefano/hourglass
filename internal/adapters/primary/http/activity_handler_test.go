package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

// Invalid-input boundary tests for the activity handler (the valid-path
// coverage lives in TestActivityHandlerIntegration against testcontainers).

func TestActivityHandler_Create_InvalidBody(t *testing.T) {
	h := NewActivityHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/activities", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestActivityHandler_Create_MissingKind(t *testing.T) {
	h := NewActivityHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/activities",
		strings.NewReader(`{"name":"No Kind","governance_model":"creator_controlled"}`))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestActivityHandler_Create_InvalidGovernanceModel(t *testing.T) {
	h := NewActivityHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/activities",
		strings.NewReader(`{"name":"Bad Gov","kind":"engagement","governance_model":"democracy"}`))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestActivityHandler_Get_InvalidID(t *testing.T) {
	h := NewActivityHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/activities/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Get(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestActivityHandler_Update_InvalidBody(t *testing.T) {
	h := NewActivityHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/activities/"+uuid.NewString(), strings.NewReader("{"))
	req.SetPathValue("id", uuid.NewString())
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.Update(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestActivityHandler_Delete_InvalidID(t *testing.T) {
	h := NewActivityHandler(nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/activities/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.Delete(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestActivityHandler_ListChildren_InvalidID(t *testing.T) {
	h := NewActivityHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/activities/not-a-uuid/children", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.ListChildren(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
