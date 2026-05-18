package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

func TestWorkingGroupHandler_Create_InvalidBody(t *testing.T) {
	h := NewWorkingGroupHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/working-groups", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWorkingGroupHandler_Create_MissingName(t *testing.T) {
	h := NewWorkingGroupHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/working-groups", strings.NewReader(`{"subproject_id":"`+uuid.NewString()+`","org_id":"`+uuid.NewString()+`"}`))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWorkingGroupHandler_Create_MissingSubprojectID(t *testing.T) {
	h := NewWorkingGroupHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/working-groups", strings.NewReader(`{"name":"Test WG","org_id":"`+uuid.NewString()+`"}`))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWorkingGroupHandler_Create_InvalidOrgID(t *testing.T) {
	h := NewWorkingGroupHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/working-groups", strings.NewReader(`{"name":"Test WG","subproject_id":"`+uuid.NewString()+`","org_id":"not-a-uuid"}`))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWorkingGroupHandler_Create_InvalidSubprojectID(t *testing.T) {
	h := NewWorkingGroupHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/working-groups", strings.NewReader(`{"name":"Test WG","subproject_id":"not-a-uuid","org_id":"`+uuid.NewString()+`"}`))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWorkingGroupHandler_Get_InvalidID(t *testing.T) {
	h := NewWorkingGroupHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/working-groups/invalid", nil)
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

func TestWorkingGroupHandler_Update_InvalidBody(t *testing.T) {
	h := NewWorkingGroupHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/working-groups/"+uuid.NewString(), strings.NewReader("{"))
	req.SetPathValue("id", uuid.NewString())
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Update(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWorkingGroupHandler_Update_InvalidID(t *testing.T) {
	h := NewWorkingGroupHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/working-groups/invalid", strings.NewReader(`{"name":"Updated WG"}`))
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Update(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWorkingGroupHandler_Delete_InvalidID(t *testing.T) {
	h := NewWorkingGroupHandler(nil)

	req := httptest.NewRequest(http.MethodDelete, "/working-groups/invalid", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Delete(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWorkingGroupHandler_AddMember_InvalidBody(t *testing.T) {
	h := NewWorkingGroupHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/working-groups/"+uuid.NewString()+"/members", strings.NewReader("{"))
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

func TestWorkingGroupHandler_ListMembers_InvalidID(t *testing.T) {
	h := NewWorkingGroupHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/working-groups/invalid/members", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.ListMembers(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWorkingGroupHandler_RemoveMember_InvalidMemberID(t *testing.T) {
	h := NewWorkingGroupHandler(nil)

	req := httptest.NewRequest(http.MethodDelete, "/working-groups/"+uuid.NewString()+"/members/invalid", nil)
	req.SetPathValue("member_id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.RemoveMember(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
