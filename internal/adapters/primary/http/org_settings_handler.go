package http

import (
	"encoding/json"
	"errors"
	"net/http"

	orgsettingsdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/orgsettings"
	orgsettingssvc "github.com/stefanoprivitera/hourglass/internal/core/services/orgsettings"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

// OrgSettingsHandler serves the literal GET/PUT /organizations/settings
// routes (D-13-23, ADR-BE-018): the org comes from JWT claims via
// middleware.GetOrganizationID — NO org path param (no org spoofing,
// T-13-10). The routes coexist with the typed GET/PUT
// /organizations/{id}/settings wildcard registrations (Pitfall 6 — ServeMux
// most-specific-wins).
type OrgSettingsHandler struct {
	service *orgsettingssvc.Service
}

func NewOrgSettingsHandler(service *orgsettingssvc.Service) *OrgSettingsHandler {
	return &OrgSettingsHandler{service: service}
}

// Get handles GET /organizations/settings: the org's known keys with
// code-level defaults applied for absent keys (D-13-24).
func (h *OrgSettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)

	settings, err := h.service.Get(ctx, orgID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, settings)
}

// Put handles PUT /organizations/settings: one-or-many {key: value} pairs,
// each validated against the known-key vocabulary (unknown key or invalid
// value → 400), manager+ gated (non-manager → 403), every write audited
// in-tx (D-13-18/22/23).
func (h *OrgSettingsHandler) Put(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	actorID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	var values map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(values) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "at least one settings key is required")
		return
	}

	settings, err := h.service.Put(ctx, orgID, actorID, role, values)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, settings)
}

// writeError maps the orgsettings service sentinels to HTTP status codes
// (ADR-BE-001, house sentinel map — coverage_handler analog). The default is
// 500 — never leaks internals.
func (h *OrgSettingsHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, orgsettingsdomain.ErrUnknownKey), errors.Is(err, orgsettingsdomain.ErrInvalidValue):
		api.RespondWithError(w, http.StatusBadRequest, "invalid settings key or value")
	case errors.Is(err, orgsettingsdomain.ErrForbidden):
		api.RespondWithError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, orgsettingsdomain.ErrNotFound):
		api.RespondWithError(w, http.StatusNotFound, "not found")
	default:
		api.RespondWithError(w, http.StatusInternalServerError, "internal server error")
	}
}
