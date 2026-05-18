package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

func TestOrganizationHandler_Create_InvalidBody(t *testing.T) {
	h := NewOrganizationHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/organizations", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestOrganizationHandler_Get_InvalidID(t *testing.T) {
	h := NewOrganizationHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/organizations/invalid", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	req = req.WithContext(ctx)

	h.Get(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestOrganizationHandler_Invite_InvalidBody(t *testing.T) {
	h := NewOrganizationHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/organizations/"+uuid.NewString()+"/invite", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Invite(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestOrganizationHandler_GetSettings_InvalidID(t *testing.T) {
	h := NewOrganizationHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/organizations/invalid/settings", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	req = req.WithContext(ctx)

	h.GetSettings(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestOrganizationHandler_UpdateSettings_InvalidBody(t *testing.T) {
	h := NewOrganizationHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/organizations/"+uuid.NewString()+"/settings", strings.NewReader("{"))
	req.SetPathValue("id", uuid.NewString())
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.UpdateSettings(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestOrganizationHandler_UpdateSettings_InvalidID(t *testing.T) {
	h := NewOrganizationHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/organizations/invalid/settings", strings.NewReader(`{"currency":"USD"}`))
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.UpdateSettings(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestOrganizationHandler_UpdateMemberRoles_InvalidBody(t *testing.T) {
	h := NewOrganizationHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/organizations/"+uuid.NewString()+"/members/"+uuid.NewString()+"/roles", strings.NewReader("{"))
	req.SetPathValue("member_id", uuid.NewString())
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.UpdateMemberRoles(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestOrganizationHandler_UpdateMemberRoles_InvalidMemberID(t *testing.T) {
	h := NewOrganizationHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/organizations/"+uuid.NewString()+"/members/invalid/roles", strings.NewReader(`{"roles":["manager"]}`))
	req.SetPathValue("member_id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.UpdateMemberRoles(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestOrganizationHandler_DeactivateMember_InvalidMemberID(t *testing.T) {
	h := NewOrganizationHandler(nil)

	req := httptest.NewRequest(http.MethodDelete, "/organizations/"+uuid.NewString()+"/members/invalid", nil)
	req.SetPathValue("member_id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.DeactivateMember(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestOrganizationHandler_InviteCustomer_InvalidBody(t *testing.T) {
	h := NewOrganizationHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/organizations/"+uuid.NewString()+"/invite-customer", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.InviteCustomer(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
