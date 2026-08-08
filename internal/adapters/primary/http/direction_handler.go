package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	directiondomain "github.com/stefanoprivitera/hourglass/internal/core/domain/direction"
	directionsvc "github.com/stefanoprivitera/hourglass/internal/core/services/direction"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

// DirectionHandler exposes the plan plane API (ADR-P-015, ADR-BE-018 §7,
// DIR-01..06): the 7 pinned routes — create (user-XOR-WG target, D-13-05),
// the explicit activation endpoint (OQ1 — create-with-planned_date does NOT
// auto-activate), cancel/unclaim with the mandatory reason (D-13-10/16), the
// WG claim (D-13-11..13), and the ListPlan/Coverage read-models (D-13-25/27)
// with the warning overlay (D-13-28). It stays thin per ADR-BE-002 — parse →
// service call → sentinel-to-HTTP mapping (ADR-BE-001). Role/user/org come
// from the middleware claims; every invariant is already decided in the
// service (13-07), so the boundary only maps. All seven routes are
// middleware.Auth-wrapped in the wiring (T-13-28).
//
// Sentinel map (coverage_handler writeError shape):
//
//	ErrDirectionNotFound                               → 404
//	ErrInvalidRequest / ErrInvalidHours / ErrInvalidTarget
//		/ ErrCancelReasonRequired                       → 400
//	ErrForbidden / ErrNotWgMember                      → 403
//	ErrInvalidTransition / ErrClaimOverBudget
//		/ ErrWgRowNotActive                             → 409
//	default                                            → 500
//
// Boundary parsing (T-13-32): uuid.Parse failures → 400; period strings are
// "2006-01-02" parsed at the boundary (OQ5); JSON decode errors → 400 — no
// 500 path for client input.
type DirectionHandler struct {
	service *directionsvc.Service
}

func NewDirectionHandler(service *directionsvc.Service) *DirectionHandler {
	return &DirectionHandler{service: service}
}

// CreateResponse is the POST /direction response body (D-13-03): the created
// row plus the warnings overlay — warnings ride the create response and never
// reject the write (D-13-28).
type CreateResponse struct {
	Row      *directiondomain.Direction `json:"row"`
	Warnings []directiondomain.Warning  `json:"warnings"`
}

// CancelRequest carries the mandatory cancellation reason (D-13-10/16).
type CancelRequest struct {
	Reason string `json:"reason"`
}

// ClaimRequest is the POST /direction/claims body (D-13-11/13): the WG row id
// and the claimed hours.
type ClaimRequest struct {
	WgRowID  uuid.UUID `json:"wg_row_id"`
	EstHours float64   `json:"est_hours"`
}

// Create handles POST /direction: parses the create request (directed_to XOR
// wg_id, activity_id, planned_date, est_hours, priority, due_date,
// supersedes_id — the 13-07 DTO shape), delegates to the service gate chain,
// and returns 200 {row, warnings}.
func (h *DirectionHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	var req directionsvc.CreateDirectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	row, warnings, err := h.service.Create(ctx, orgID, userID, role, &req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, CreateResponse{Row: row, Warnings: warnings})
}

// Activate handles POST /direction/{id}/activate (OQ1 — the explicit
// activation endpoint; draft → active, one audit row per transition).
func (h *DirectionHandler) Activate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid direction id")
		return
	}

	row, err := h.service.Activate(ctx, orgID, userID, role, id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, row)
}

// Cancel handles POST /direction/{id}/cancel (D-13-10): the reason is
// mandatory — the service rejects a missing reason with 400 before any state
// change.
func (h *DirectionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid direction id")
		return
	}

	var req CancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	row, err := h.service.Cancel(ctx, orgID, userID, role, id, req.Reason)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, row)
}

// Claim handles POST /direction/claims (D-13-11..13): a WG member splits off
// a user-targeted claim row with origin_direction_id = the WG row. The Σ
// budget guard runs in the repo tx under the WG-row lock (CR-01) — the
// service fast-fails membership/hours first.
func (h *DirectionHandler) Claim(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	var req ClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	row, err := h.service.Claim(ctx, orgID, userID, role, req.WgRowID, req.EstHours)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, row)
}

// Unclaim handles POST /direction/claims/{id}/cancel (D-13-16): cancels a
// CLAIM row with a mandatory reason — hours return to the WG budget
// automatically since consumption is Σ-derived.
func (h *DirectionHandler) Unclaim(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid direction id")
		return
	}

	var req CancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	row, err := h.service.Unclaim(ctx, orgID, userID, role, id, req.Reason)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, row)
}

// ListPlan handles GET /direction (D-13-27): the plan read-model with the
// derived-on-read states and the warning overlay. employee_id is optional
// and means the org-wide default view — which the service gate restricts to
// managers (OQ5, T-13-26): non-managers MUST pass their own employee_id.
// period_start + period_end are required, parsed as 2006-01-02 at the
// boundary (OQ5 — malformed or missing bounds → 400).
func (h *DirectionHandler) ListPlan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	employeeID, err := parseOptionalQueryUUID(r, "employee_id")
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid employee_id")
		return
	}
	start, end, err := parsePeriod(r)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid period")
		return
	}

	plan, err := h.service.ListPlan(ctx, orgID, userID, role, employeeID, start, end)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, plan)
}

// Coverage handles GET /direction/coverage (DIR-06, D-13-25): the
// planned-vs-capacity read-model with scope params (employee|unit|wg) and
// period bounds, all required. Unit/wg scopes are manager-only and the
// employee scope allows the non-manager self-view (scope_id == actorID) —
// the service gate (T-13-26/31).
func (h *DirectionHandler) Coverage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	scope := r.URL.Query().Get("scope")
	scopeID := r.URL.Query().Get("scope_id")
	if scope == "" || scopeID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "scope and scope_id are required")
		return
	}
	start, end, err := parsePeriod(r)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid period")
		return
	}

	coverage, err := h.service.Coverage(ctx, orgID, userID, role, scope, scopeID, start, end)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, coverage)
}

// parseOptionalQueryUUID parses an optional uuid query parameter ("" → nil).
func parseOptionalQueryUUID(r *http.Request, key string) (*uuid.UUID, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// parsePeriod parses the required period_start/period_end query params as
// 2006-01-02 (OQ5 — the read-model day semantics are timezone-free, UTC
// midnight).
func parsePeriod(r *http.Request) (time.Time, time.Time, error) {
	startRaw := r.URL.Query().Get("period_start")
	endRaw := r.URL.Query().Get("period_end")
	if startRaw == "" || endRaw == "" {
		return time.Time{}, time.Time{}, errors.New("period_start and period_end are required")
	}
	start, err := time.Parse("2006-01-02", startRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := time.Parse("2006-01-02", endRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start, end, nil
}

// writeError maps the direction service sentinels to HTTP status codes
// (ADR-BE-001, coverage_handler writeError shape): 404/400/403/409 — never a
// 500 for client input (T-13-29/32). The default is 500 — never leaks
// internals.
func (h *DirectionHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, directiondomain.ErrDirectionNotFound):
		api.RespondWithError(w, http.StatusNotFound, "direction not found")
	case errors.Is(err, directiondomain.ErrInvalidRequest),
		errors.Is(err, directiondomain.ErrInvalidHours),
		errors.Is(err, directiondomain.ErrInvalidTarget),
		errors.Is(err, directiondomain.ErrCancelReasonRequired):
		api.RespondWithError(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, directiondomain.ErrForbidden),
		errors.Is(err, directiondomain.ErrNotWgMember):
		api.RespondWithError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, directiondomain.ErrInvalidTransition),
		errors.Is(err, directiondomain.ErrClaimOverBudget),
		errors.Is(err, directiondomain.ErrWgRowNotActive):
		api.RespondWithError(w, http.StatusConflict, "conflict")
	default:
		api.RespondWithError(w, http.StatusInternalServerError, "internal server error")
	}
}
