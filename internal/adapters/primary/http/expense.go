package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/expense"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	expsvc "github.com/stefanoprivitera/hourglass/internal/core/services/expense"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

// ExpenseHandler handles HTTP requests for expense CRUD and approval workflow.
type ExpenseHandler struct {
	service *expsvc.Service
}

// NewExpenseHandler creates a new ExpenseHandler.
func NewExpenseHandler(service *expsvc.Service) *ExpenseHandler {
	return &ExpenseHandler{service: service}
}

// --- Request types ---

type CreateExpenseRequest struct {
	ActivityID  string   `json:"activity_id"`
	Category    string   `json:"category"`
	Amount      float64  `json:"amount"`
	KmDistance  *float64 `json:"km_distance,omitempty"`
	Description string   `json:"description"`
	Date        string   `json:"date"`
}

type UpdateExpenseRequest struct {
	ActivityID  *string  `json:"activity_id,omitempty"`
	Category    *string  `json:"category,omitempty"`
	Amount      *float64 `json:"amount,omitempty"`
	KmDistance  *float64 `json:"km_distance,omitempty"`
	Description *string  `json:"description,omitempty"`
	Date        *string  `json:"date,omitempty"`
}

type RejectExpenseRequest struct {
	Reason string `json:"reason"`
}

// --- Handlers ---

// List returns expenses for an org with optional filters.
func (h *ExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)
	userID := middleware.GetUserID(ctx)

	filters := ports.ExpenseListFilters{
		Role:          role,
		RequestUserID: userID.String(),
		IsDeleted:     false,
	}

	if date := r.URL.Query().Get("date"); date != "" {
		filters.Date = date
	}
	if month := r.URL.Query().Get("month"); month != "" {
		filters.Month = month
	}
	if year := r.URL.Query().Get("year"); year != "" {
		filters.Year = year
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filters.Status = status
	}
	if activityID := r.URL.Query().Get("activity_id"); activityID != "" {
		filters.ActivityID = activityID
	}
	if filterUserID := r.URL.Query().Get("user_id"); filterUserID != "" {
		filters.UserID = filterUserID
	}

	entries, err := h.service.List(ctx, orgID, filters)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch expenses")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, entries)
}

// Get returns a single expense by ID, with org ownership check.
func (h *ExpenseHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)
	userID := middleware.GetUserID(ctx)
	expenseIDStr := r.PathValue("id")

	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	e, err := h.service.Get(ctx, expenseID)
	if err != nil {
		if err == expense.ErrExpenseNotFound {
			api.RespondWithError(w, http.StatusNotFound, "expense not found")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get expense")
		return
	}

	if e.OrgID != orgID {
		api.RespondWithError(w, http.StatusNotFound, "expense not found")
		return
	}

	if role == "employee" && e.UserID != userID {
		api.RespondWithError(w, http.StatusForbidden, "can only view own expenses")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, e)
}

// Create creates a new expense.
func (h *ExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)

	var req CreateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !validateStringLengths(w,
		lengthField("description", req.Description, MaxDescriptionLength),
		lengthField("date", req.Date, MaxShortStringLength),
	) {
		return
	}

	if req.ActivityID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "activity_id is required")
		return
	}
	if req.Category == "" {
		api.RespondWithError(w, http.StatusBadRequest, "category is required")
		return
	}
	if !expense.IsValidCategory(req.Category) {
		api.RespondWithError(w, http.StatusBadRequest, "invalid category")
		return
	}
	if req.Amount <= 0 {
		api.RespondWithError(w, http.StatusBadRequest, "amount must be greater than 0")
		return
	}
	if req.Date == "" {
		api.RespondWithError(w, http.StatusBadRequest, "date is required")
		return
	}

	activityID, err := uuid.Parse(req.ActivityID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid activity_id")
		return
	}

	svcReq := &expense.CreateExpenseRequest{
		OrgID:       orgID,
		UserID:      userID,
		ActivityID:  activityID,
		Category:    req.Category,
		Amount:      req.Amount,
		KmDistance:  req.KmDistance,
		Description: req.Description,
		Date:        req.Date,
	}

	e, err := h.service.Create(ctx, svcReq)
	if err != nil {
		if err == expense.ErrPeriodLocked {
			api.RespondWithError(w, http.StatusBadRequest, "cannot create expense for locked period")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create expense")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, e)
}

// Update updates an existing expense.
func (h *ExpenseHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	expenseIDStr := r.PathValue("id")

	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	var req UpdateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var description, date string
	if req.Description != nil {
		description = *req.Description
	}
	if req.Date != nil {
		date = *req.Date
	}
	if !validateStringLengths(w,
		lengthField("description", description, MaxDescriptionLength),
		lengthField("date", date, MaxShortStringLength),
	) {
		return
	}

	svcReq := &expense.UpdateExpenseRequest{}
	if req.ActivityID != nil {
		aid, err := uuid.Parse(*req.ActivityID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid activity_id")
			return
		}
		svcReq.ActivityID = &aid
	}
	if req.Category != nil {
		svcReq.Category = req.Category
	}
	if req.Amount != nil {
		svcReq.Amount = req.Amount
	}
	if req.KmDistance != nil {
		svcReq.KmDistance = req.KmDistance
	}
	if req.Description != nil {
		svcReq.Description = req.Description
	}
	if req.Date != nil {
		svcReq.Date = req.Date
	}

	e, err := h.service.Update(ctx, expenseID, userID, svcReq)
	if err != nil {
		if err == expense.ErrExpenseNotFound {
			api.RespondWithError(w, http.StatusNotFound, "expense not found")
			return
		}
		if err == expense.ErrEntryNotDraft {
			api.RespondWithError(w, http.StatusBadRequest, "can only update draft, submitted or rejected expenses")
			return
		}
		if err == expense.ErrNotOwner {
			api.RespondWithError(w, http.StatusForbidden, "can only update own expenses")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update expense")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, e)
}

// Delete soft-deletes an expense.
func (h *ExpenseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	expenseIDStr := r.PathValue("id")

	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	err = h.service.Delete(ctx, expenseID, userID)
	if err != nil {
		if err == expense.ErrExpenseNotFound {
			api.RespondWithError(w, http.StatusNotFound, "expense not found")
			return
		}
		if err == expense.ErrEntryNotDraft {
			api.RespondWithError(w, http.StatusBadRequest, "can only delete draft expenses")
			return
		}
		if err == expense.ErrNotOwner {
			api.RespondWithError(w, http.StatusForbidden, "can only delete own expenses")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete expense")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Submit submits an expense for approval.
func (h *ExpenseHandler) Submit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	expenseIDStr := r.PathValue("id")

	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	e, err := h.service.Submit(ctx, expenseID, userID)
	if err != nil {
		if err == expense.ErrExpenseNotFound {
			api.RespondWithError(w, http.StatusNotFound, "expense not found")
			return
		}
		if err == expense.ErrEntryNotDraft {
			api.RespondWithError(w, http.StatusBadRequest, "can only submit draft or rejected expenses")
			return
		}
		if err == expense.ErrNotOwner {
			api.RespondWithError(w, http.StatusForbidden, "can only submit own expenses")
			return
		}
		if errors.Is(err, activitydomain.ErrActivityNotLoggable) {
			api.RespondWithError(w, http.StatusConflict, "this activity requires a working group before entries can be logged")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to submit expense")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, e)
}

// Approve approves an expense (manager→pending_finance, finance→approved).
func (h *ExpenseHandler) Approve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)
	expenseIDStr := r.PathValue("id")

	if role != "manager" && role != "finance" {
		api.RespondWithError(w, http.StatusForbidden, "only managers and finance can approve expenses")
		return
	}

	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	e, err := h.service.Approve(ctx, expenseID, userID, role)
	if err != nil {
		if err == expense.ErrExpenseNotFound {
			api.RespondWithError(w, http.StatusNotFound, "expense not found")
			return
		}
		if err == expense.ErrEntryNotSubmitted {
			api.RespondWithError(w, http.StatusBadRequest, "expense cannot be approved at current status")
			return
		}
		if err == expense.ErrForbidden {
			api.RespondWithError(w, http.StatusForbidden, "only managers and finance can approve expenses")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to approve expense")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, e)
}

// Reject rejects an expense with a reason.
func (h *ExpenseHandler) Reject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	role := middleware.GetRole(ctx)
	expenseIDStr := r.PathValue("id")

	if role != "manager" && role != "finance" {
		api.RespondWithError(w, http.StatusForbidden, "only managers and finance can reject expenses")
		return
	}

	var req RejectExpenseRequest
	json.NewDecoder(r.Body).Decode(&req)

	if !validateStringLengths(w,
		lengthField("reason", req.Reason, MaxShortStringLength),
	) {
		return
	}

	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	e, err := h.service.Reject(ctx, expenseID, userID, role, req.Reason)
	if err != nil {
		if err == expense.ErrExpenseNotFound {
			api.RespondWithError(w, http.StatusNotFound, "expense not found")
			return
		}
		if err == expense.ErrEntryNotSubmitted {
			api.RespondWithError(w, http.StatusBadRequest, "expense cannot be rejected at current status")
			return
		}
		if err == expense.ErrForbidden {
			api.RespondWithError(w, http.StatusForbidden, "only managers and finance can reject expenses")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to reject expense")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, e)
}

// ListPending returns pending expenses for the current user's role.
func (h *ExpenseHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)
	userID := middleware.GetUserID(ctx)

	if role != "manager" && role != "finance" {
		api.RespondWithError(w, http.StatusForbidden, "only managers and finance can view pending expenses")
		return
	}

	entries, err := h.service.ListPending(ctx, orgID, role, userID.String())
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch pending expenses")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, entries)
}

// ReceiptUpload handles file upload for an expense receipt.
func (h *ExpenseHandler) ReceiptUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	expenseIDStr := r.PathValue("id")

	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	// Limit upload size to 10MB
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "file field is required")
		return
	}
	defer file.Close()

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".pdf" && ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		api.RespondWithError(w, http.StatusBadRequest, "only PDF, JPEG, and PNG files are allowed")
		return
	}

	// Create directory: uploads/receipts/{org_id}/{expense_id}/
	uploadDir := filepath.Join("uploads", "receipts", orgID.String(), expenseID.String())
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create upload directory")
		return
	}

	// Save file
	filename := uuid.New().String() + ext
	dst, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to write file")
		return
	}

	// Build receipt URL
	receiptURL := filepath.Join("uploads", "receipts", orgID.String(), expenseID.String(), filename)

	// Update expense with receipt URL via service
	e, err := h.service.SetReceiptURL(ctx, expenseID, receiptURL)
	if err != nil {
		if err == expense.ErrExpenseNotFound {
			api.RespondWithError(w, http.StatusNotFound, "expense not found")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update receipt url")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, e)
}
