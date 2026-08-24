package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	coveragedomain "github.com/stefanoprivitera/hourglass/internal/core/domain/coverage"
	coveragesvc "github.com/stefanoprivitera/hourglass/internal/core/services/coverage"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

// CoverageHandler exposes the coverage plane API (ADR-P-012, ADR-BE-017,
// COV-01..05): the replace-set allocation write (D-07), the read-back and
// audit surfaces, the computed-on-read proposal and to-cover queue (D-04/
// D-06/D-I), the derived bucket balance (D-02), and the period-close
// snapshot (D-10/D-11/D-12). It stays thin per ADR-BE-002 — parse → service
// call → sentinel-to-HTTP mapping (ADR-BE-001). Role/user/org come from the
// middleware claims; every invariant is already decided in the service
// (12-05), so the boundary only maps.
//
// Sentinel map (house style, mirrors ticket_handler.writeError):
//
//	ErrEntryNotCoverable        → 404
//	ErrNotFound                 → 404
//	ErrAllocationSumMismatch    → 400
//	ErrInvalidRequest           → 400
//	ErrForbidden                → 403
//	ErrPeriodAlreadyClosed      → 409
//	default                     → 500
//
// Route surface (D-07/COV-04 prohibitions): exactly the eight routes
// registered in cmd/server/main.go — deliberately NO incremental allocation
// CRUD (no PUT /allocations/{id}, no DELETE), no snapshot/audit mutation
// routes, no finance confirm step (D-L one-step), and the close body never
// carries org_id (T-12-19 — org comes from claims).
type CoverageHandler struct {
	service *coveragesvc.Service
}

func NewCoverageHandler(service *coveragesvc.Service) *CoverageHandler {
	return &CoverageHandler{service: service}
}

// ReplaceAllocationsRequest is the boundary DTO for PUT
// /time-entries/{id}/allocations (D-07): the FULL allocation set, 1..N rows.
// IDs are strings parsed at the boundary; entry_id comes from the path.
type ReplaceAllocationsRequest struct {
	Allocations []AllocationRequest `json:"allocations"`
}

// AllocationRequest is one row of the replace-set. entry_type is carried by
// the client (the D-K branch in the service rejects anything but 'time');
// exactly one ref pins per source_type (contract_id for contract/transfer,
// unit_id for absorption) — the service validates the pinning.
type AllocationRequest struct {
	EntryType     string   `json:"entry_type"`
	SourceType    string   `json:"source_type"`
	ContractID    *string  `json:"contract_id"`
	UnitID        *string  `json:"unit_id"`
	Hours         float64  `json:"hours"`
	Reason        *string  `json:"reason"`
	Justification *string  `json:"justification"`
}

// ClosePeriodRequest carries only the reporting period (D-12, T-12-19): the
// org comes from middleware claims, never from the client. Dates are
// "2006-01-02" strings parsed at the boundary.
type ClosePeriodRequest struct {
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
}

// PutAllocations handles PUT /time-entries/{id}/allocations (D-07): the
// atomic replace-set. The D-08 manager gate runs in the service (approver
// resolution via BE-014); the handler maps the sentinels — manager 200,
// owner/employee/finance/customer 403, Σ mismatch 400, non-coverable 404.
func (h *CoverageHandler) PutAllocations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	entryID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	var req ReplaceAllocationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	allocs := make([]*coveragedomain.CoverageAllocation, 0, len(req.Allocations))
	for _, a := range req.Allocations {
		contractID, err := parseOptionalUUID(derefStr(a.ContractID))
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid contract_id in allocations")
			return
		}
		unitID, err := parseOptionalUUID(derefStr(a.UnitID))
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid unit_id in allocations")
			return
		}
		allocs = append(allocs, &coveragedomain.CoverageAllocation{
			EntryType:     a.EntryType,
			EntryID:       entryID,
			SourceType:    a.SourceType,
			ContractID:    contractID,
			UnitID:        unitID,
			Hours:         a.Hours,
			Reason:        a.Reason,
			Justification: a.Justification,
		})
	}

	stored, err := h.service.ReplaceAllocations(ctx, orgID, entryID, allocs, userID.String(), role)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, stored)
}

// GetAllocations handles GET /time-entries/{id}/allocations — the read-back
// of an entry's current allocation set (the "optional; queue may embed"
// read-model, kept for Phase 17). The service read gate (manager|finance)
// applies; the current set comes from the same Propose read path that
// returns it alongside the computed proposal (D-I — no dedicated read method
// exists on the service, so the thin handler reuses the pinned one).
func (h *CoverageHandler) GetAllocations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	entryID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	_, allocs, err := h.service.Propose(ctx, orgID, entryID, role, userID.String())
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, allocs)
}

// GetOwn handles GET /coverage/own/{entry_id}: the employee self-read path
// (Phase 16). The caller may only read coverage on an entry they own; the
// service enforces the self-scope + org gate, so this never widens the
// manager|finance read paths.
func (h *CoverageHandler) GetOwn(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)

	entryID, err := uuid.Parse(r.PathValue("entry_id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	proposal, allocs, err := h.service.GetOwnCoverage(ctx, orgID, entryID, userID.String())
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, map[string]any{
		"proposal":    proposal,
		"allocations": allocs,
	})
}

// GetProposal handles GET /coverage/proposals/{entry_id}: the computed-on-
// read D-04 default proposal (D-I) plus the entry's current allocations.
func (h *CoverageHandler) GetProposal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	entryID, err := uuid.Parse(r.PathValue("entry_id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	proposal, allocs, err := h.service.Propose(ctx, orgID, entryID, role, userID.String())
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, map[string]any{
		"proposal":    proposal,
		"allocations": allocs,
	})
}

// GetToCoverQueue handles GET /coverage/to-cover (D-06, COV-01): every
// approved, non-deleted 'time' entry with uncovered hours — no-source
// entries included, flagged. Read gate: manager|finance.
func (h *CoverageHandler) GetToCoverQueue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	rows, err := h.service.ToCoverQueue(ctx, orgID, role, userID.String())
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, rows)
}

// GetBucketBalance handles GET /coverage/buckets/{contract_id}/balance
// (D-02/D-03): the derived support-bucket balance — sold_hours minus the Σ
// allocations drawn from the contract, computed on read. Negative balances
// are returned as-is (overdraw is report-visible, never a gate).
func (h *CoverageHandler) GetBucketBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	contractID, err := uuid.Parse(r.PathValue("contract_id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid contract id")
		return
	}

	balance, err := h.service.BucketBalance(ctx, orgID, contractID, role, userID.String())
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, map[string]any{"balance": balance})
}

// PostClose handles POST /coverage/close (D-12, COV-04): freezes the
// reporting period's allocation state into an immutable snapshot and returns
// it incl. rows in one call (OQ4). Manager-only (the service gate); the org
// comes from claims (T-12-19 — never from the body). Date parse errors → 400.
func (h *CoverageHandler) PostClose(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	var req ClosePeriodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	start, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid period_start")
		return
	}
	end, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid period_end")
		return
	}

	closeResult, err := h.service.ClosePeriod(ctx, orgID, start, end, userID, role)
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusCreated, closeResult)
}

// GetSnapshot handles GET /coverage/snapshots/{close_id}: a frozen
// period-close snapshot (header + rows). Read gate: manager|finance.
func (h *CoverageHandler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	closeID, err := uuid.Parse(r.PathValue("close_id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid close id")
		return
	}

	snapshot, err := h.service.GetSnapshot(ctx, orgID, closeID, role, userID.String())
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, snapshot)
}

// GetHistory handles GET /coverage/allocations/{entry_id}/history: the
// append-only audit stream behind an entry's allocations (A7,
// entity_type='coverage_allocation'). Read gate: manager|finance.
func (h *CoverageHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)

	entryID, err := uuid.Parse(r.PathValue("entry_id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	history, err := h.service.ListHistory(ctx, orgID, entryID, role, userID.String())
	if err != nil {
		h.writeError(w, err)
		return
	}
	api.RespondWithJSON(w, http.StatusOK, history)
}

// writeError maps coverage service sentinels to HTTP status codes
// (ADR-BE-001, house sentinel map — ticket_handler analog). The default is
// 500 — never leaks internals.
func (h *CoverageHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, coveragedomain.ErrEntryNotCoverable):
		api.RespondWithError(w, http.StatusNotFound, "entry not coverable")
	case errors.Is(err, coveragedomain.ErrNotFound):
		api.RespondWithError(w, http.StatusNotFound, "not found")
	case errors.Is(err, coveragedomain.ErrAllocationSumMismatch):
		api.RespondWithError(w, http.StatusBadRequest, "allocation sum does not match entry hours")
	case errors.Is(err, coveragedomain.ErrInvalidRequest):
		api.RespondWithError(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, coveragedomain.ErrForbidden):
		api.RespondWithError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, coveragedomain.ErrPeriodAlreadyClosed):
		api.RespondWithError(w, http.StatusConflict, "period already closed")
	default:
		api.RespondWithError(w, http.StatusInternalServerError, "internal server error")
	}
}

// derefStr returns the value of a *string ("" when nil) — the boundary DTOs
// carry pointer strings for optional ids, and parseOptionalUUID takes a
// plain string.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
