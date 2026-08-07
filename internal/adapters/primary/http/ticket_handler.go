package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	ticketdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/ticket"
	ticketsvc "github.com/stefanoprivitera/hourglass/internal/core/services/ticket"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

// TicketHandler exposes the ticket lifecycle API (ADR-P-003 rev, TICK-01..05):
// create/list/get/update plus the action endpoints (triage, dismiss,
// transition, comments) and the append-only history read. It stays thin per
// ADR-BE-002 — parse → service call → sentinel-to-HTTP mapping
// (ADR-BE-001). String IDs are parsed at the boundary (activity_handler
// pattern).
//
// Sentinel map (house style, mirrors ErrHasChildren → 409):
//
//	ErrTicketNotFound              → 404
//	ErrInvalidRequest              → 400
//	ErrInvalidTransition           → 400
//	ErrForbidden                   → 403
//	ErrDismissalBlocked            → 409 (guard-style)
//	ErrActivityNotTerminal         → 409 (guard-style)
//	default                        → 500
//
// There is deliberately NO DELETE /tickets route and no update/delete path
// for comments or history rows (TICK-05 — the stream is append-only).
type TicketHandler struct {
	service *ticketsvc.Service
}

func NewTicketHandler(service *ticketsvc.Service) *TicketHandler {
	return &TicketHandler{service: service}
}

// CreateTicketRequest is the boundary DTO: string IDs parsed to UUIDs.
type CreateTicketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
	AssigneeID  string `json:"assignee_id"`
}

// UpdateTicketRequest uses pointers so absent fields are not touched.
type UpdateTicketRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	AssigneeID  *string `json:"assignee_id,omitempty"`
}

// TransitionRequest carries the target status + optional note.
type TransitionRequest struct {
	Status string  `json:"status"`
	Note   *string `json:"note,omitempty"`
}

// TriageActivityPlanDTO is the boundary shape of one triage plan; IDs are
// strings parsed to UUIDs here.
type TriageActivityPlanDTO struct {
	Name            string  `json:"name"`
	Kind            string  `json:"kind"`
	ParentID        string  `json:"parent_id"`
	ContractID      string  `json:"contract_id"`
	GovernanceModel string  `json:"governance_model"`
	Description     string  `json:"description"`
	Billable        *bool   `json:"billable"`
	IsShared        *bool   `json:"is_shared"`
	BudgetAmount    *float64 `json:"budget_amount"`
}

// TriageRequest is the triage payload: optional kind override + 1..N plans.
type TriageRequest struct {
	Kind       *string                `json:"kind,omitempty"`
	Activities []TriageActivityPlanDTO `json:"activities"`
}

// CommentRequest carries the comment body.
type CommentRequest struct {
	Body string `json:"body"`
}

// ticketDetailResponse is GET /tickets/{id}: the ticket plus its comments.
type ticketDetailResponse struct {
	Ticket   ticketdomain.Ticket          `json:"ticket"`
	Comments []ticketdomain.TicketComment `json:"comments"`
}

// Create handles POST /tickets (any employee; customer rejected, T-11-04).
func (h *TicketHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	assigneeID, err := parseOptionalUUID(req.AssigneeID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid assignee_id")
		return
	}

	created, err := h.service.Create(ctx, orgID, userID, role, &ticketsvc.CreateTicketRequest{
		Title:       req.Title,
		Description: req.Description,
		Kind:        req.Kind,
		AssigneeID:  assigneeID,
	})
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusCreated, created)
}

// List handles GET /tickets with optional status/kind query filters.
func (h *TicketHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)

	tickets, err := h.service.List(ctx, orgID, role, r.URL.Query().Get("status"), r.URL.Query().Get("kind"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, tickets)
}

// Get handles GET /tickets/{id} (ticket + comments).
func (h *TicketHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)

	ticketID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	t, comments, err := h.service.Get(ctx, orgID, role, ticketID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, ticketDetailResponse{Ticket: *t, Comments: comments})
}

// Update handles PUT /tickets/{id} (owner/assignee/manager+ per D-15).
func (h *TicketHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	ticketID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var req UpdateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	assigneeID, err := parseOptionalUUIDPtr(req.AssigneeID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid assignee_id")
		return
	}

	updated, err := h.service.UpdateDetails(ctx, orgID, userID, role, ticketID, req.Title, req.Description, assigneeID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, updated)
}

// Triage handles POST /tickets/{id}/triage (manager|finance, D-11).
func (h *TicketHandler) Triage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	ticketID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var req TriageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	plans := make([]*ticketsvc.TriageActivityPlan, 0, len(req.Activities))
	for _, dto := range req.Activities {
		parentID, err := parseOptionalUUID(dto.ParentID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid parent_id in activities")
			return
		}
		contractID, err := parseOptionalUUID(dto.ContractID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid contract_id in activities")
			return
		}
		plans = append(plans, &ticketsvc.TriageActivityPlan{
			Name:            dto.Name,
			Description:     dto.Description,
			Kind:            dto.Kind,
			ParentID:        parentID,
			ContractID:      contractID,
			GovernanceModel: models.GovernanceModel(dto.GovernanceModel),
			IsShared:        dto.IsShared != nil && *dto.IsShared,
			Billable:        dto.Billable,
			BudgetAmount:    dto.BudgetAmount,
		})
	}

	t, activities, err := h.service.Triage(ctx, orgID, userID, role, ticketID, req.Kind, plans)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, map[string]any{
		"ticket":     t,
		"activities": activities,
	})
}

// Dismiss handles POST /tickets/{id}/dismiss (manager|finance, guarded).
func (h *TicketHandler) Dismiss(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	ticketID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	t, err := h.service.Dismiss(ctx, orgID, userID, role, ticketID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, t)
}

// Transition handles POST /tickets/{id}/transition (D-14 matrix).
func (h *TicketHandler) Transition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	ticketID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var req TransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t, err := h.service.Transition(ctx, orgID, userID, role, ticketID, req.Status, req.Note)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, t)
}

// AddComment handles POST /tickets/{id}/comments (owner/assignee/manager+).
func (h *TicketHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	ticketID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var req CommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	c, err := h.service.AddComment(ctx, orgID, userID, role, ticketID, req.Body)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, c)
}

// History handles GET /tickets/{id}/history — the append-only audit stream
// (TICK-05). Read-only; no mutations exist on this endpoint.
func (h *TicketHandler) History(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)

	ticketID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	history, err := h.service.ListHistory(ctx, orgID, role, ticketID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, history)
}

// writeError maps ticket service sentinels to HTTP status codes (ADR-BE-001,
// house sentinel map). The default is 500 — never leaks internals.
func (h *TicketHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ticketdomain.ErrTicketNotFound):
		api.RespondWithError(w, http.StatusNotFound, "ticket not found")
	case errors.Is(err, ticketdomain.ErrInvalidRequest):
		api.RespondWithError(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, ticketdomain.ErrInvalidTransition):
		api.RespondWithError(w, http.StatusBadRequest, "invalid ticket status transition")
	case errors.Is(err, ticketdomain.ErrForbidden):
		api.RespondWithError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ticketdomain.ErrDismissalBlocked):
		api.RespondWithError(w, http.StatusConflict, "dismissal blocked: linked activities have logged hours")
	case errors.Is(err, ticketdomain.ErrActivityNotTerminal):
		api.RespondWithError(w, http.StatusConflict, "ticket activity is not in a terminal state")
	default:
		api.RespondWithError(w, http.StatusInternalServerError, "internal server error")
	}
}

// parseOptionalUUIDPtr parses an optional *string into a *uuid.UUID, keeping
// nil for absent values (update DTOs use pointer-to-string).
func parseOptionalUUIDPtr(s *string) (*uuid.UUID, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
