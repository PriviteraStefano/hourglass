package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	projectdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/project"
	projectsvc "github.com/stefanoprivitera/hourglass/internal/core/services/project"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

func TestProjectHandler_Create_InvalidBody(t *testing.T) {
	h := NewProjectHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestProjectHandler_Get_InvalidID(t *testing.T) {
	h := NewProjectHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/projects/invalid", nil)
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

func TestProjectHandler_Adopt_InvalidID(t *testing.T) {
	h := NewProjectHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/projects/invalid/adopt", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Adopt(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestProjectHandler_ListManagers_InvalidID(t *testing.T) {
	h := NewProjectHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/projects/invalid/managers", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.ListManagers(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestProjectHandler_AddManager_InvalidBody(t *testing.T) {
	h := NewProjectHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+uuid.NewString()+"/managers", strings.NewReader("{"))
	req.SetPathValue("id", uuid.NewString())
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.AddManager(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestProjectHandler_AddManager_InvalidID(t *testing.T) {
	h := NewProjectHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/projects/invalid/managers", strings.NewReader(`{"user_id":"`+uuid.NewString()+`"}`))
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.AddManager(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestProjectHandler_RemoveManager_InvalidID(t *testing.T) {
	h := NewProjectHandler(nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/projects/invalid/managers/"+uuid.NewString(), nil)
	req.SetPathValue("id", "not-a-uuid")
	req.SetPathValue("user_id", uuid.NewString())
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.RemoveManager(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestProjectHandler_RemoveManager_InvalidUserID(t *testing.T) {
	h := NewProjectHandler(nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+uuid.NewString()+"/managers/invalid", nil)
	req.SetPathValue("id", uuid.NewString())
	req.SetPathValue("user_id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.RemoveManager(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func setupProjectHandler(t *testing.T) (*ProjectHandler, *testdata.MockProjectRepo) {
	t.Helper()
	mockRepo := &testdata.MockProjectRepo{}
	svc := projectsvc.NewService(mockRepo)
	subprojectRepo := &mockSubprojectRepo{}
	handler := NewProjectHandler(svc, subprojectRepo)
	return handler, mockRepo
}

type mockSubprojectRepo struct{}

func (m *mockSubprojectRepo) ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.Subproject, error) {
	return []models.Subproject{}, nil
}
func (m *mockSubprojectRepo) GetByID(ctx context.Context, id string) (*models.Subproject, error) { return nil, nil }
func (m *mockSubprojectRepo) Create(ctx context.Context, sp *models.Subproject) (*models.Subproject, error) { return nil, nil }
func (m *mockSubprojectRepo) Update(ctx context.Context, sp *models.Subproject) (*models.Subproject, error) { return nil, nil }
func (m *mockSubprojectRepo) Delete(ctx context.Context, id string) error { return nil }

func TestProjectHandler_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, _ := setupProjectHandler(t)
		orgID := uuid.New()
		projectID := uuid.New()

		body := `{"name":"Updated","type":"internal","contract_id":"` + uuid.New().String() + `","governance_model":"creator_controlled","is_shared":false}`
		req := httptest.NewRequest("PUT", "/projects/"+projectID.String(), strings.NewReader(body))
		req.SetPathValue("id", projectID.String())
		ctx := context.Background()
		ctx = middleware.SetOrganizationID(ctx, orgID)
		ctx = middleware.SetRole(ctx, string(models.RoleFinance))
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.Update(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("forbidden", func(t *testing.T) {
		handler, _ := setupProjectHandler(t)
		orgID := uuid.New()
		projectID := uuid.New()

		body := `{"name":"Updated","type":"internal","contract_id":"` + uuid.New().String() + `","governance_model":"creator_controlled","is_shared":false}`
		req := httptest.NewRequest("PUT", "/projects/"+projectID.String(), strings.NewReader(body))
		req.SetPathValue("id", projectID.String())
		ctx := context.Background()
		ctx = middleware.SetRole(ctx, string(models.RoleEmployee))
		ctx = middleware.SetOrganizationID(ctx, orgID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.Update(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestProjectHandler_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, mockRepo := setupProjectHandler(t)
		orgID := uuid.New()
		projectID := uuid.New()
		if mockRepo.Projects == nil {
			mockRepo.Projects = make(map[uuid.UUID]*projectdomain.ProjectResponse)
		}
		mockRepo.Projects[projectID] = &projectdomain.ProjectResponse{
			Project: projectdomain.Project{ID: projectID, CreatedByOrgID: orgID},
		}
		req := httptest.NewRequest("DELETE", "/projects/"+projectID.String(), nil)
		req.SetPathValue("id", projectID.String())
		ctx := context.Background()
		ctx = middleware.SetRole(ctx, string(models.RoleFinance))
		ctx = middleware.SetOrganizationID(ctx, orgID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.Delete(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("forbidden non-finance role", func(t *testing.T) {
		handler, _ := setupProjectHandler(t)
		projectID := uuid.New()
		req := httptest.NewRequest("DELETE", "/projects/"+projectID.String(), nil)
		req.SetPathValue("id", projectID.String())
		ctx := context.Background()
		ctx = middleware.SetRole(ctx, string(models.RoleEmployee))
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.Delete(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("conflict time entries", func(t *testing.T) {
		handler, mockRepo := setupProjectHandler(t)
		orgID := uuid.New()
		projectID := uuid.New()
		if mockRepo.Projects == nil {
			mockRepo.Projects = make(map[uuid.UUID]*projectdomain.ProjectResponse)
		}
		mockRepo.Projects[projectID] = &projectdomain.ProjectResponse{
			Project: projectdomain.Project{ID: projectID, CreatedByOrgID: orgID},
		}
		mockRepo.HasActiveTimeEntriesFn = func(ctx context.Context, pid uuid.UUID) (bool, bool, error) {
			return true, false, nil
		}
		req := httptest.NewRequest("DELETE", "/projects/"+projectID.String(), nil)
		req.SetPathValue("id", projectID.String())
		ctx := context.Background()
		ctx = middleware.SetRole(ctx, string(models.RoleFinance))
		ctx = middleware.SetOrganizationID(ctx, orgID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.Delete(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("conflict subproject entries", func(t *testing.T) {
		handler, mockRepo := setupProjectHandler(t)
		orgID := uuid.New()
		projectID := uuid.New()
		if mockRepo.Projects == nil {
			mockRepo.Projects = make(map[uuid.UUID]*projectdomain.ProjectResponse)
		}
		mockRepo.Projects[projectID] = &projectdomain.ProjectResponse{
			Project: projectdomain.Project{ID: projectID, CreatedByOrgID: orgID},
		}
		mockRepo.HasActiveTimeEntriesFn = func(ctx context.Context, pid uuid.UUID) (bool, bool, error) {
			return false, true, nil
		}
		req := httptest.NewRequest("DELETE", "/projects/"+projectID.String(), nil)
		req.SetPathValue("id", projectID.String())
		ctx := context.Background()
		ctx = middleware.SetRole(ctx, string(models.RoleFinance))
		ctx = middleware.SetOrganizationID(ctx, orgID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.Delete(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestProjectHandler_ListSubprojects(t *testing.T) {
	t.Run("returns subprojects", func(t *testing.T) {
		handler, _ := setupProjectHandler(t)
		projectID := uuid.New()
		req := httptest.NewRequest("GET", "/projects/"+projectID.String()+"/subprojects", nil)
		req.SetPathValue("id", projectID.String())
		w := httptest.NewRecorder()
		handler.ListSubprojects(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
