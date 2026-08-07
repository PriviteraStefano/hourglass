package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	wgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	"github.com/stefanoprivitera/hourglass/internal/core/services/routing"
	tesvc "github.com/stefanoprivitera/hourglass/internal/core/services/time_entry"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
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

func TestTimeEntryHandler_Create_MissingActivityID(t *testing.T) {
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

func TestTimeEntryHandler_Create_InvalidActivityID(t *testing.T) {
	h := NewTimeEntryHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/time-entries", strings.NewReader(`{"activity_id":"not-a-uuid","unit_id":"`+uuid.NewString()+`","hours":8,"date":"2026-01-15"}`))
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

func TestTimeEntryHandler_Get_InvalidID(t *testing.T) {
	h := NewTimeEntryHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/time-entries/invalid", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "employee")
	req = req.WithContext(ctx)

	h.Get(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestTimeEntryHandler_Update_InvalidBody(t *testing.T) {
	h := NewTimeEntryHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/time-entries/"+uuid.NewString(), strings.NewReader("{"))
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

func TestTimeEntryHandler_Submit_InvalidID(t *testing.T) {
	h := NewTimeEntryHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/time-entries/invalid/submit", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Submit(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestTimeEntryHandler_Reject_NoAuth(t *testing.T) {
	h := NewTimeEntryHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/time-entries/"+uuid.NewString()+"/reject", nil)
	req.SetPathValue("id", uuid.NewString())
	rec := httptest.NewRecorder()

	// No role set — middleware.GetRole returns "" which is != "wg_manager" and != "admin"
	h.Reject(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestTimeEntryHandler_ListPending_EmployeeForbidden(t *testing.T) {
	h := NewTimeEntryHandler(newTEService(t, nil))

	req := httptest.NewRequest(http.MethodGet, "/time-entries/pending", nil)
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "employee")
	req = req.WithContext(ctx)

	h.ListPending(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestTimeEntryHandler_ListPending_EmployeeWGManager_OK(t *testing.T) {
	orgID := uuid.New()
	empID := uuid.New()
	// The employee is a WG manager for the activity — the manager-stage
	// approver set (T-10-05-3). Service admits them via IsWGManager; the
	// handler passes role "wg_manager" and the queue is WG-scoped repo-side.
	h := NewTimeEntryHandler(newTEService(t, func(f *teServiceFixture) {
		f.wgRepo.Groups = map[uuid.UUID]*wgdomain.WorkingGroup{
			uuid.New(): {
				ID: uuid.New(), OrgID: orgID, SubprojectID: uuid.New(),
				Name: "Test WG", ManagerID: empID, IsActive: true,
			},
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/time-entries/pending", nil)
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), empID)
	ctx = middleware.SetOrganizationID(ctx, orgID)
	ctx = middleware.SetRole(ctx, "employee")
	req = req.WithContext(ctx)

	h.ListPending(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestTimeEntryHandler_ListPending_NoAuth(t *testing.T) {
	h := NewTimeEntryHandler(newTEService(t, nil))

	req := httptest.NewRequest(http.MethodGet, "/time-entries/pending", nil)
	rec := httptest.NewRecorder()

	// No role set — GetRole returns "" which is != "wg_manager" and != "admin".
	// The service's IsWGManager resolves an empty WG list → false → 403.
	h.ListPending(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

// teServiceFixture carries the mock repos backing a real time-entry service
// for handler unit tests (the handler's ListPending gate now resolves WG
// membership via Service.IsWGManager — T-10-05-3 — so a nil service would
// panic).
type teServiceFixture struct {
	repo         *testdata.MockTimeEntryRepo
	approvalRepo *testdata.MockTimeEntryApprovalRepo
	wgRepo       *testdata.MockWorkingGroupRepo
	activityRepo *testdata.MockActivityRepo
	unitRepo     *testdata.MockUnitRepo
	svc          *tesvc.Service
}

// newTEService builds a real time-entry service over testdata mocks.
// override, when non-nil, configures the fixture before the service is
// constructed.
func newTEService(t *testing.T, override func(*teServiceFixture)) *tesvc.Service {
	t.Helper()
	f := &teServiceFixture{
		repo:         &testdata.MockTimeEntryRepo{Entries: make(map[uuid.UUID]*time_entry.TimeEntry)},
		approvalRepo: &testdata.MockTimeEntryApprovalRepo{},
		wgRepo:       &testdata.MockWorkingGroupRepo{},
		activityRepo: &testdata.MockActivityRepo{},
		unitRepo:     &testdata.MockUnitRepo{},
	}
	if override != nil {
		override(f)
	}
	f.svc = tesvc.NewService(f.repo, f.approvalRepo, f.wgRepo, f.activityRepo, f.unitRepo, routing.NewService(f.wgRepo, f.activityRepo, f.unitRepo))
	return f.svc
}

func TestTimeEntryHandler_BatchApprove_InvalidBody(t *testing.T) {
	// Note: BatchApprove is not a handler method — this tests the pattern
	// of malformed JSON rejection via Create which does parse JSON.
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

func TestTimeEntryHandler_Delete_InvalidID(t *testing.T) {
	h := NewTimeEntryHandler(nil)

	req := httptest.NewRequest(http.MethodDelete, "/time-entries/invalid", nil)
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
