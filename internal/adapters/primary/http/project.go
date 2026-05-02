package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	projectdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/project"
	projectsvc "github.com/stefanoprivitera/hourglass/internal/core/services/project"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type ProjectHandler struct {
	service *projectsvc.Service
}

func NewProjectHandler(service *projectsvc.Service) *ProjectHandler {
	return &ProjectHandler{service: service}
}

type CreateProjectRequest struct {
	Name            string                 `json:"name"`
	Type            models.ProjectType     `json:"type"`
	ContractID      string                 `json:"contract_id"`
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	IsShared        bool                   `json:"is_shared"`
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "owned"
	}
	projects, err := h.service.List(r.Context(), orgID, scope, r.URL.Query().Get("contract_id"))
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch projects")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, projects)
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	project, err := h.service.Create(r.Context(), orgID, &projectdomain.CreateProjectRequest{
		Name:            req.Name,
		Type:            req.Type,
		ContractID:      req.ContractID,
		GovernanceModel: req.GovernanceModel,
		IsShared:        req.IsShared,
	})
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid project payload")
		return
	}
	api.RespondWithJSON(w, http.StatusCreated, project)
}

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	projectID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid project id")
		return
	}
	project, err := h.service.Get(r.Context(), orgID, projectID)
	if err != nil {
		api.RespondWithError(w, http.StatusNotFound, "project not found")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, project)
}

func (h *ProjectHandler) Adopt(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	projectID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid project id")
		return
	}
	adoption, err := h.service.Adopt(r.Context(), orgID, projectID)
	if err != nil {
		if err == projectdomain.ErrAlreadyAdopted {
			api.RespondWithError(w, http.StatusConflict, "project already adopted")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to adopt project")
		return
	}
	api.RespondWithJSON(w, http.StatusCreated, adoption)
}

func (h *ProjectHandler) ListManagers(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid project id")
		return
	}
	managers, err := h.service.ListManagers(r.Context(), projectID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch managers")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, managers)
}

func (h *ProjectHandler) AddManager(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid project id")
		return
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	manager, err := h.service.AddManager(r.Context(), middleware.GetRole(r.Context()), projectID, userID)
	if err != nil {
		switch err {
		case projectdomain.ErrForbidden:
			api.RespondWithError(w, http.StatusForbidden, "only finance users can add project managers")
		case projectdomain.ErrUserNotFound:
			api.RespondWithError(w, http.StatusBadRequest, "user not found")
		default:
			api.RespondWithError(w, http.StatusInternalServerError, "failed to add manager")
		}
		return
	}
	api.RespondWithJSON(w, http.StatusCreated, manager)
}

func (h *ProjectHandler) RemoveManager(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid project id")
		return
	}
	userID, err := uuid.Parse(r.PathValue("user_id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	err = h.service.RemoveManager(r.Context(), middleware.GetRole(r.Context()), projectID, userID)
	if err != nil {
		if err == projectdomain.ErrForbidden {
			api.RespondWithError(w, http.StatusForbidden, "only finance users can remove project managers")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to remove manager")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
