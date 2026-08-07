package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	activitysvc "github.com/stefanoprivitera/hourglass/internal/core/services/activity"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

// ActivityHandler exposes the collapsed activity API surface (ADR-P-007,
// ADR-BE-014 R-6) — the single recursive work entity replacing the old
// projects + subprojects handlers. It stays thin per ADR-BE-002: parse →
// service call → sentinel-to-HTTP mapping (ADR-BE-001).
//
// The detail endpoint additionally composes GetAncestry / ResolveCommercialContext
// / ResolveBillability from the repository — the service exposes the CRUD
// surface (Plan 09-04 scope) and those three are read-only derived queries the
// repo owns, mirroring how the old ProjectHandler held the subproject repo
// directly.
type ActivityHandler struct {
	service *activitysvc.Service
	repo    ports.ActivityRepository
}

func NewActivityHandler(service *activitysvc.Service, repo ports.ActivityRepository) *ActivityHandler {
	return &ActivityHandler{service: service, repo: repo}
}

type CreateActivityRequest struct {
	ParentID        string                 `json:"parent_id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Kind            string                 `json:"kind"`
	ContractID      string                 `json:"contract_id"`
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	IsShared        bool                   `json:"is_shared"`
	Billable        *bool                  `json:"billable"`
	BudgetAmount    *float64               `json:"budget_amount"`
	IsActive        *bool                  `json:"is_active"`

	// Origin axis (ADR-P-013): strings parsed to pointers; empty → nil.
	OriginType string `json:"origin_type"`
	AssignedBy string `json:"assigned_by"`
	AssignedTo string `json:"assigned_to"`
	ProposedBy string `json:"proposed_by"`
	ReviewedBy string `json:"reviewed_by"`
	TicketID   string `json:"ticket_id"`
}

type UpdateActivityRequest struct {
	ParentID        *string                 `json:"parent_id,omitempty"`
	Name            *string                 `json:"name,omitempty"`
	Description     *string                 `json:"description,omitempty"`
	Kind            *string                 `json:"kind,omitempty"`
	ContractID      *string                 `json:"contract_id,omitempty"`
	GovernanceModel *models.GovernanceModel `json:"governance_model,omitempty"`
	IsShared        *bool                   `json:"is_shared,omitempty"`
	Billable        *bool                   `json:"billable,omitempty"`
	BudgetAmount    *float64                `json:"budget_amount,omitempty"`
	IsActive        *bool                   `json:"is_active,omitempty"`

	// Origin fields present so the service immutability guard can reject
	// them (D-03); the repo UPDATE never touches origin columns.
	OriginType *string `json:"origin_type,omitempty"`
	AssignedBy *string `json:"assigned_by,omitempty"`
	AssignedTo *string `json:"assigned_to,omitempty"`
	ProposedBy *string `json:"proposed_by,omitempty"`
	ReviewedBy *string `json:"reviewed_by,omitempty"`
	TicketID   *string `json:"ticket_id,omitempty"`
}

// parseOptionalUUID parses an ID string into a pointer, returning nil for
// empty strings and surfacing malformed UUIDs to the caller.
func parseOptionalUUID(s string) (*uuid.UUID, error) {
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// ActivityDetailResponse is the GET /api/activities/:id payload: the activity
// itself plus the derived reads — ancestry chain (ordered parent → root),
// derived commercial context (nearest contract-bearing ancestor, D-3, or nil
// for internal trees), and resolved billability (inheritance walk, D-7).
type ActivityDetailResponse struct {
	Activity          activitydomain.Activity           `json:"activity"`
	Ancestry          []activitydomain.Activity         `json:"ancestry"`
	CommercialContext *activitydomain.CommercialContext `json:"commercial_context"`
	Billable          *bool                             `json:"billable"`
}

// List returns the org's activities with optional parent_id / contract_id /
// kind filters (scope defaults to "owned" like the old project handler).
func (h *ActivityHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "owned"
	}
	filter := &activitydomain.ActivityFilter{
		Scope:      scope,
		ContractID: r.URL.Query().Get("contract_id"),
		ParentID:   r.URL.Query().Get("parent_id"),
		Kind:       r.URL.Query().Get("kind"),
	}
	activities, err := h.service.List(r.Context(), orgID, filter)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch activities")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, activities)
}

// Create validates the payload at the boundary (name/kind/governance_model
// required, IDs parseable) and delegates to the service, which validates the
// kind against the org's activity_kinds catalog (D-2) and the parent/contract
// against the org's context (D-2/D-3).
func (h *ActivityHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	var req CreateActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !validateStringLengths(w,
		lengthField("name", req.Name, MaxNameLength),
		lengthField("description", req.Description, MaxDescriptionLength),
	) {
		return
	}
	if req.Name == "" {
		api.RespondWithError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Kind == "" {
		api.RespondWithError(w, http.StatusBadRequest, "kind is required")
		return
	}
	if !req.GovernanceModel.IsValid() {
		api.RespondWithError(w, http.StatusBadRequest, "invalid governance_model")
		return
	}

	var err error
	svcReq := &activitydomain.CreateActivityRequest{
		Name:            req.Name,
		Description:     req.Description,
		Kind:            activitydomain.ActivityKind(req.Kind),
		GovernanceModel: req.GovernanceModel,
		IsShared:        req.IsShared,
		Billable:        req.Billable,
		BudgetAmount:    req.BudgetAmount,
		IsActive:        req.IsActive,
	}
	if req.ParentID != "" {
		pid, err := uuid.Parse(req.ParentID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		svcReq.ParentID = &pid
	}
	if req.ContractID != "" {
		cid, err := uuid.Parse(req.ContractID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid contract_id")
			return
		}
		svcReq.ContractID = &cid
	}

	// Origin payload (ADR-P-013): empty strings parse to nil; malformed
	// UUIDs are rejected at the boundary.
	if req.OriginType != "" {
		svcReq.OriginType = &req.OriginType
	}
	svcReq.AssignedBy, err = parseOptionalUUID(req.AssignedBy)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid assigned_by")
		return
	}
	svcReq.AssignedTo, err = parseOptionalUUID(req.AssignedTo)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid assigned_to")
		return
	}
	svcReq.ProposedBy, err = parseOptionalUUID(req.ProposedBy)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid proposed_by")
		return
	}
	svcReq.ReviewedBy, err = parseOptionalUUID(req.ReviewedBy)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid reviewed_by")
		return
	}
	svcReq.TicketID, err = parseOptionalUUID(req.TicketID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid ticket_id")
		return
	}

	activity, err := h.service.Create(r.Context(), middleware.GetRole(r.Context()), orgID, middleware.GetUserID(r.Context()), svcReq)
	if err != nil {
		switch {
		case errors.Is(err, activitydomain.ErrInvalidRequest):
			api.RespondWithError(w, http.StatusBadRequest, "invalid activity payload")
		case errors.Is(err, activitydomain.ErrForbidden):
			api.RespondWithError(w, http.StatusForbidden, "not allowed to create this activity")
		case errors.Is(err, activitydomain.ErrActivityNotFound):
			api.RespondWithError(w, http.StatusBadRequest, "parent activity not found")
		default:
			api.RespondWithError(w, http.StatusInternalServerError, "failed to create activity")
		}
		return
	}
	api.RespondWithJSON(w, http.StatusCreated, activity)
}

// Get returns the activity detail: the activity itself plus its ancestry
// chain, derived commercial context, and resolved billability.
func (h *ActivityHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	activityID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid activity id")
		return
	}

	activity, err := h.service.GetByID(r.Context(), orgID, activityID)
	if err != nil {
		if errors.Is(err, activitydomain.ErrActivityNotFound) {
			api.RespondWithError(w, http.StatusNotFound, "activity not found")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch activity")
		return
	}

	ctx := r.Context()
	ancestry, err := h.repo.GetAncestry(ctx, activityID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch activity ancestry")
		return
	}
	commercialContext, err := h.repo.ResolveCommercialContext(ctx, activityID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to resolve commercial context")
		return
	}
	billable, err := h.repo.ResolveBillability(ctx, activityID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to resolve billability")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, ActivityDetailResponse{
		Activity:          activity.Activity,
		Ancestry:          ancestry,
		CommercialContext: commercialContext,
		Billable:          billable,
	})
}

// Update is finance-role gated (service enforces + owner check).
func (h *ActivityHandler) Update(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	activityID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid activity id")
		return
	}

	var req UpdateActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var name, description string
	if req.Name != nil {
		name = *req.Name
	}
	if req.Description != nil {
		description = *req.Description
	}
	if !validateStringLengths(w,
		lengthField("name", name, MaxNameLength),
		lengthField("description", description, MaxDescriptionLength),
	) {
		return
	}

	svcReq := &activitydomain.UpdateActivityRequest{}
	if req.ParentID != nil {
		pid, err := uuid.Parse(*req.ParentID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		svcReq.ParentID = &pid
	}
	if req.ContractID != nil {
		cid, err := uuid.Parse(*req.ContractID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid contract_id")
			return
		}
		svcReq.ContractID = &cid
	}
	if req.Name != nil {
		svcReq.Name = *req.Name
	}
	if req.Description != nil {
		svcReq.Description = *req.Description
	}
	if req.Kind != nil {
		svcReq.Kind = activitydomain.ActivityKind(*req.Kind)
	}
	if req.GovernanceModel != nil {
		svcReq.GovernanceModel = *req.GovernanceModel
	}
	if req.IsShared != nil {
		svcReq.IsShared = req.IsShared
	}
	if req.Billable != nil {
		svcReq.Billable = req.Billable
	}
	if req.BudgetAmount != nil {
		svcReq.BudgetAmount = req.BudgetAmount
	}
	if req.IsActive != nil {
		svcReq.IsActive = req.IsActive
	}

	// Origin fields pass through so the service immutability guard rejects
	// them with ErrOriginImmutable (D-03) — malformed UUIDs are boundary 400s.
	originFields := []struct {
		src  *string
		dst  **uuid.UUID
		name string
	}{
		{req.AssignedBy, &svcReq.AssignedBy, "assigned_by"},
		{req.AssignedTo, &svcReq.AssignedTo, "assigned_to"},
		{req.ProposedBy, &svcReq.ProposedBy, "proposed_by"},
		{req.ReviewedBy, &svcReq.ReviewedBy, "reviewed_by"},
		{req.TicketID, &svcReq.TicketID, "ticket_id"},
	}
	if req.OriginType != nil {
		svcReq.OriginType = req.OriginType
	}
	for _, f := range originFields {
		if f.src == nil {
			continue
		}
		parsed, err := parseOptionalUUID(*f.src)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid "+f.name)
			return
		}
		*f.dst = parsed
	}

	updated, err := h.service.Update(r.Context(), middleware.GetRole(r.Context()), orgID, activityID, svcReq)
	if err != nil {
		switch {
		case errors.Is(err, activitydomain.ErrForbidden):
			api.RespondWithError(w, http.StatusForbidden, "only finance users can update activities")
		case errors.Is(err, activitydomain.ErrActivityNotFound):
			api.RespondWithError(w, http.StatusNotFound, "activity not found")
		case errors.Is(err, activitydomain.ErrOriginImmutable):
			api.RespondWithError(w, http.StatusConflict, "origin refs are immutable after creation")
		case errors.Is(err, activitydomain.ErrInvalidRequest):
			api.RespondWithError(w, http.StatusBadRequest, "invalid activity payload")
		case errors.Is(err, activitydomain.ErrActivityCycle):
			api.RespondWithError(w, http.StatusBadRequest, "activity parent would create a cycle")
		default:
			api.RespondWithError(w, http.StatusInternalServerError, "failed to update activity")
		}
		return
	}
	api.RespondWithJSON(w, http.StatusOK, updated)
}

// Delete is finance-role gated with guard sentinels mapped to 409
// (has-children / active time entries / active expenses, ADR-BE-001).
func (h *ActivityHandler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	activityID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid activity id")
		return
	}

	err = h.service.Delete(r.Context(), middleware.GetRole(r.Context()), orgID, activityID)
	if err != nil {
		switch {
		case errors.Is(err, activitydomain.ErrForbidden):
			api.RespondWithError(w, http.StatusForbidden, "only finance users can delete activities")
		case errors.Is(err, activitydomain.ErrActivityNotFound):
			api.RespondWithError(w, http.StatusNotFound, "activity not found")
		case errors.Is(err, activitydomain.ErrHasChildren):
			api.RespondWithError(w, http.StatusConflict, "activity has children and cannot be deleted")
		case errors.Is(err, activitydomain.ErrHasActiveTimeEntries):
			api.RespondWithError(w, http.StatusConflict, "activity has active time entries and cannot be deleted")
		case errors.Is(err, activitydomain.ErrHasActiveExpenses):
			api.RespondWithError(w, http.StatusConflict, "activity has active expenses and cannot be deleted")
		default:
			api.RespondWithError(w, http.StatusInternalServerError, "failed to delete activity")
		}
		return
	}
	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "activity deleted"})
}

// ListChildren returns the direct children of an activity (the old
// ListSubprojects replacement — subprojects are now task-kind children).
func (h *ActivityHandler) ListChildren(w http.ResponseWriter, r *http.Request) {
	activityID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid activity id")
		return
	}
	children, err := h.service.ListChildren(r.Context(), activityID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch children")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, children)
}

// ListKinds returns the org's activity_kinds catalog (ADR-P-007 D-2).
func (h *ActivityHandler) ListKinds(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	kinds, err := h.service.ListKinds(r.Context(), orgID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch activity kinds")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, kinds)
}
