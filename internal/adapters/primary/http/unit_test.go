package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

func TestUnitHandler_Create_InvalidBody(t *testing.T) {
	h := NewUnitHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/units", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUnitHandler_Create_MissingName(t *testing.T) {
	h := NewUnitHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/units", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUnitHandler_Get_MissingID(t *testing.T) {
	h := NewUnitHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/units/", nil)
	req.SetPathValue("id", "")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Get(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUnitHandler_Update_InvalidBody(t *testing.T) {
	h := NewUnitHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/units/"+uuid.NewString(), strings.NewReader("{"))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Update(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUnitHandler_Update_MissingID(t *testing.T) {
	h := NewUnitHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/units/", strings.NewReader(`{}`))
	req.SetPathValue("id", "")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Update(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUnitHandler_Delete_MissingID(t *testing.T) {
	h := NewUnitHandler(nil)

	req := httptest.NewRequest(http.MethodDelete, "/units/", nil)
	req.SetPathValue("id", "")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Delete(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUnitHandler_AddMember_InvalidBody(t *testing.T) {
	h := NewUnitHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/units/"+uuid.NewString()+"/members", strings.NewReader("{"))
	req.SetPathValue("id", uuid.NewString())
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.AddMember(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUnitHandler_AddMember_MissingUserID(t *testing.T) {
	h := NewUnitHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/units/"+uuid.NewString()+"/members", strings.NewReader(`{}`))
	req.SetPathValue("id", uuid.NewString())
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.AddMember(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// GetTree and List immediately call h.service, requiring a non-nil service.
// With nil service, they panic — they have no handler-level validation gate.
// Coverage for those methods is provided by the existing auth_test.go integration tests.
