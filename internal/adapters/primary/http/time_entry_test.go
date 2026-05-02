package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

func TestTimeEntryHandler_Create_InvalidBody(t *testing.T) {
	h := NewTimeEntryHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/time-entries", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestTimeEntryHandler_Create_MissingProjectID(t *testing.T) {
	h := NewTimeEntryHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/time-entries", strings.NewReader(`{"hours":8,"date":"2026-01-15"}`))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestTimeEntryHandler_Approve_EmployeeForbidden(t *testing.T) {
	h := NewTimeEntryHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/time-entries/"+uuid.NewString()+"/approve", nil)
	req.SetPathValue("id", uuid.NewString())
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "employee")
	req = req.WithContext(ctx)

	h.Approve(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}
