package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type ExpenseHandler struct {
	db          *sql.DB
	uploadDir   string
	maxFileSize int64
}

func NewExpenseHandler(db *sql.DB) *ExpenseHandler {
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads/receipts"
	}
	return &ExpenseHandler{
		db:          db,
		uploadDir:   uploadDir,
		maxFileSize: 10 * 1024 * 1024,
	}
}

func (h *ExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())
	role := middleware.GetRole(r.Context())

	query := `
		SELECT e.id, e.user_id, e.organization_id, e.project_id, e.customer_id, e.date, 
			   e.type, e.amount, e.km_distance, e.description, e.status, 
			   e.current_approver_role, e.submitted_at, e.deleted_at, 
			   e.created_at, e.updated_at
		FROM expenses e
		WHERE e.organization_id = $1 AND e.deleted_at IS NULL
	`
	args := []interface{}{orgID}
	argIndex := 2

	date := r.URL.Query().Get("date")
	if date != "" {
		query += fmt.Sprintf(" AND e.date = $%d", argIndex)
		args = append(args, date)
		argIndex++
	}

	month := r.URL.Query().Get("month")
	year := r.URL.Query().Get("year")
	if month != "" && year != "" {
		query += fmt.Sprintf(" AND EXTRACT(MONTH FROM e.date) = $%d AND EXTRACT(YEAR FROM e.date) = $%d", argIndex, argIndex+1)
		args = append(args, month, year)
		argIndex += 2
	}

	filterUserID := r.URL.Query().Get("user_id")
	if filterUserID != "" {
		if role == string(models.RoleEmployee) && filterUserID != userID.String() {
			api.RespondWithError(w, http.StatusForbidden, "can only view own expenses")
			return
		}
		query += fmt.Sprintf(" AND e.user_id = $%d", argIndex)
		args = append(args, filterUserID)
		argIndex++
	} else if role == string(models.RoleEmployee) {
		query += fmt.Sprintf(" AND e.user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	status := r.URL.Query().Get("status")
	if status != "" {
		query += fmt.Sprintf(" AND e.status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	expenseType := r.URL.Query().Get("type")
	if expenseType != "" {
		query += fmt.Sprintf(" AND e.type = $%d", argIndex)
		args = append(args, expenseType)
		argIndex++
	}

	projectID := r.URL.Query().Get("project_id")
	if projectID != "" {
		query += fmt.Sprintf(" AND e.project_id = $%d", argIndex)
		args = append(args, projectID)
		argIndex++
	}

	query += " ORDER BY e.date DESC, e.created_at DESC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch expenses")
		return
	}
	defer rows.Close()

	var expenses []models.Expense
	for rows.Next() {
		var e models.Expense
		var currentApproverRole sql.NullString
		var submittedAt sql.NullTime
		var deletedAt sql.NullTime
		var projectID sql.NullString
		var customerID sql.NullString
		var expenseType sql.NullString
		var amount sql.NullFloat64
		var kmDistance sql.NullFloat64
		var description sql.NullString
		err := rows.Scan(
			&e.ID, &e.UserID, &e.OrganizationID, &projectID, &customerID, &e.Date,
			&expenseType, &amount, &kmDistance, &description, &e.Status,
			&currentApproverRole, &submittedAt, &deletedAt,
			&e.CreatedAt, &e.UpdatedAt,
		)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to scan expense")
			return
		}
		if currentApproverRole.Valid {
			e.CurrentApproverRole = &currentApproverRole.String
		}
		if submittedAt.Valid {
			e.SubmittedAt = &submittedAt.Time
		}
		if deletedAt.Valid {
			e.DeletedAt = &deletedAt.Time
		}
		if projectID.Valid {
			pid, _ := uuid.Parse(projectID.String)
			e.ProjectID = &pid
		}
		if customerID.Valid {
			cid, _ := uuid.Parse(customerID.String)
			e.CustomerID = &cid
		}
		if expenseType.Valid {
			et := models.ExpenseCategory(expenseType.String)
			e.Type = &et
		}
		if amount.Valid {
			e.Amount = &amount.Float64
		}
		if kmDistance.Valid {
			e.KmDistance = &kmDistance.Float64
		}
		if description.Valid {
			e.Description = description.String
		}
		expenses = append(expenses, e)
	}

	if expenses == nil {
		expenses = []models.Expense{}
	}

	api.RespondWithJSON(w, http.StatusOK, expenses)
}

func (h *ExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	err := r.ParseMultipartForm(h.maxFileSize)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	dateStr := r.FormValue("date")
	if dateStr == "" {
		api.RespondWithError(w, http.StatusBadRequest, "date is required")
		return
	}

	entryDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid date format")
		return
	}

	itemsJSON := r.FormValue("items")
	if itemsJSON == "" {
		api.RespondWithError(w, http.StatusBadRequest, "items are required")
		return
	}

	var items []models.ExpenseItemCreateRequest
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid items format")
		return
	}

	if len(items) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "at least one item is required")
		return
	}

	for i, item := range items {
		if !item.Category.IsValid() {
			api.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid category for item %d", i))
			return
		}
		if item.Amount < 0 {
			api.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("amount must be positive for item %d", i))
			return
		}

		projectID, err := uuid.Parse(item.ProjectID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid project id for item %d", i))
			return
		}

		var accessible bool
		err = h.db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM projects p
				WHERE p.id = $1 AND p.is_active = true
				AND (p.created_by_org_id = $2 OR p.is_shared = true OR p.id IN (
					SELECT project_id FROM project_adoptions WHERE organization_id = $2
				))
			)
		`, projectID, orgID).Scan(&accessible)
		if err != nil || !accessible {
			api.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("project not found or not accessible for item %d", i))
			return
		}

		if item.Category == models.CategoryMileage && item.KmDistance != nil && *item.KmDistance > 0 {
			var kmRate float64
			err = h.db.QueryRow(`
				SELECT COALESCE(c.km_rate, 0)
				FROM projects p
				JOIN contracts c ON p.contract_id = c.id
				WHERE p.id = $1
			`, projectID).Scan(&kmRate)
			if err != nil {
				kmRate = 0
			}
			item.Amount = *item.KmDistance * kmRate
			items[i] = item
		}
	}

	tx, err := h.db.Begin()
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	expenseID := uuid.New()
	now := time.Now()

	_, err = tx.Exec(`
		INSERT INTO expenses (id, user_id, organization_id, date, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
	`, expenseID, userID, orgID, entryDate, models.StatusDraft, now)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create expense")
		return
	}

	receiptFiles := r.MultipartForm.File["receipts"]
	receiptIndex := 0

	for _, item := range items {
		projectID, _ := uuid.Parse(item.ProjectID)
		itemID := uuid.New()

		_, err = tx.Exec(`
			INSERT INTO expense_items (id, expense_id, project_id, category, amount, km_distance, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, itemID, expenseID, projectID, item.Category, item.Amount, item.KmDistance, item.Description)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to create expense item")
			return
		}

		if receiptIndex < len(receiptFiles) {
			fileHeader := receiptFiles[receiptIndex]
			file, err := fileHeader.Open()
			if err != nil {
				api.RespondWithError(w, http.StatusInternalServerError, "failed to open uploaded file")
				return
			}
			defer file.Close()

			filePath, err := h.saveReceipt(file, fileHeader.Filename, entryDate)
			if err != nil {
				api.RespondWithError(w, http.StatusInternalServerError, "failed to save receipt")
				return
			}

			_, err = tx.Exec(`
				INSERT INTO expense_receipts (expense_item_id, file_path, original_filename)
				VALUES ($1, $2, $3)
			`, itemID, filePath, fileHeader.Filename)
			if err != nil {
				api.RespondWithError(w, http.StatusInternalServerError, "failed to save receipt record")
				return
			}
			receiptIndex++
		}
	}

	if err := tx.Commit(); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	expense := models.Expense{
		ID:             expenseID,
		UserID:         userID,
		OrganizationID: orgID,
		Date:           entryDate,
		Status:         models.StatusDraft,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	expense.Items, _ = h.getExpenseItems(expenseID)

	api.RespondWithJSON(w, http.StatusCreated, expense)
}

func (h *ExpenseHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())
	role := middleware.GetRole(r.Context())

	expenseIDStr := r.PathValue("id")
	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	var e models.Expense
	var expenseUserID uuid.UUID
	var currentApproverRole sql.NullString
	var submittedAt sql.NullTime

	err = h.db.QueryRow(`
		SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at
		FROM expenses WHERE id = $1 AND organization_id = $2
	`, expenseID, orgID).Scan(
		&e.ID, &expenseUserID, &e.OrganizationID, &e.Date,
		&e.Status, &currentApproverRole, &submittedAt,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "expense not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch expense")
		return
	}

	if role == string(models.RoleEmployee) && expenseUserID.String() != userID.String() {
		api.RespondWithError(w, http.StatusForbidden, "can only view own expenses")
		return
	}

	if currentApproverRole.Valid {
		e.CurrentApproverRole = &currentApproverRole.String
	}
	if submittedAt.Valid {
		e.SubmittedAt = &submittedAt.Time
	}

	e.Items, err = h.getExpenseItems(e.ID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch expense items")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, e)
}

func (h *ExpenseHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	expenseIDStr := r.PathValue("id")
	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	var status string
	var expenseUserID uuid.UUID
	err = h.db.QueryRow(`
		SELECT status, user_id FROM expenses WHERE id = $1 AND organization_id = $2
	`, expenseID, orgID).Scan(&status, &expenseUserID)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "expense not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch expense")
		return
	}

	if status != string(models.StatusDraft) {
		api.RespondWithError(w, http.StatusBadRequest, "can only update draft expenses")
		return
	}

	if expenseUserID.String() != userID.String() {
		api.RespondWithError(w, http.StatusForbidden, "can only update own expenses")
		return
	}

	err = r.ParseMultipartForm(h.maxFileSize)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	itemsJSON := r.FormValue("items")
	if itemsJSON == "" {
		api.RespondWithError(w, http.StatusBadRequest, "items are required")
		return
	}

	var items []models.ExpenseItemCreateRequest
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid items format")
		return
	}

	if len(items) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "at least one item is required")
		return
	}

	for i, item := range items {
		if !item.Category.IsValid() {
			api.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid category for item %d", i))
			return
		}

		projectID, err := uuid.Parse(item.ProjectID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid project id for item %d", i))
			return
		}

		var accessible bool
		err = h.db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM projects p
				WHERE p.id = $1 AND p.is_active = true
				AND (p.created_by_org_id = $2 OR p.is_shared = true OR p.id IN (
					SELECT project_id FROM project_adoptions WHERE organization_id = $2
				))
			)
		`, projectID, orgID).Scan(&accessible)
		if err != nil || !accessible {
			api.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("project not found for item %d", i))
			return
		}

		if item.Category == models.CategoryMileage && item.KmDistance != nil && *item.KmDistance > 0 {
			var kmRate float64
			err = h.db.QueryRow(`
				SELECT COALESCE(c.km_rate, 0)
				FROM projects p
				JOIN contracts c ON p.contract_id = c.id
				WHERE p.id = $1
			`, projectID).Scan(&kmRate)
			if err != nil {
				kmRate = 0
			}
			item.Amount = *item.KmDistance * kmRate
			items[i] = item
		}
	}

	tx, err := h.db.Begin()
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM expense_items WHERE expense_id = $1`, expenseID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete old items")
		return
	}

	receiptFiles := r.MultipartForm.File["receipts"]
	receiptIndex := 0

	for _, item := range items {
		projectID, _ := uuid.Parse(item.ProjectID)
		itemID := uuid.New()

		_, err = tx.Exec(`
			INSERT INTO expense_items (id, expense_id, project_id, category, amount, km_distance, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, itemID, expenseID, projectID, item.Category, item.Amount, item.KmDistance, item.Description)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to create expense item")
			return
		}

		if receiptIndex < len(receiptFiles) {
			fileHeader := receiptFiles[receiptIndex]
			file, err := fileHeader.Open()
			if err != nil {
				api.RespondWithError(w, http.StatusInternalServerError, "failed to open uploaded file")
				return
			}
			defer file.Close()

			var expense models.Expense
			err = h.db.QueryRow(`SELECT date FROM expenses WHERE id = $1`, expenseID).Scan(&expense.Date)
			if err != nil {
				api.RespondWithError(w, http.StatusInternalServerError, "failed to get expense date")
				return
			}

			filePath, err := h.saveReceipt(file, fileHeader.Filename, expense.Date)
			if err != nil {
				api.RespondWithError(w, http.StatusInternalServerError, "failed to save receipt")
				return
			}

			_, err = tx.Exec(`
				INSERT INTO expense_receipts (expense_item_id, file_path, original_filename)
				VALUES ($1, $2, $3)
			`, itemID, filePath, fileHeader.Filename)
			if err != nil {
				api.RespondWithError(w, http.StatusInternalServerError, "failed to save receipt record")
				return
			}
			receiptIndex++
		}
	}

	if err := tx.Commit(); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	var e models.Expense
	var currentApproverRole sql.NullString
	var submittedAt sql.NullTime
	err = h.db.QueryRow(`
		SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at
		FROM expenses WHERE id = $1
	`, expenseID).Scan(
		&e.ID, &e.UserID, &e.OrganizationID, &e.Date,
		&e.Status, &currentApproverRole, &submittedAt,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch updated expense")
		return
	}

	if currentApproverRole.Valid {
		e.CurrentApproverRole = &currentApproverRole.String
	}
	if submittedAt.Valid {
		e.SubmittedAt = &submittedAt.Time
	}

	e.Items, _ = h.getExpenseItems(e.ID)

	api.RespondWithJSON(w, http.StatusOK, e)
}

func (h *ExpenseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	expenseIDStr := r.PathValue("id")
	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	var status string
	var expenseUserID uuid.UUID
	var deletedAt sql.NullTime
	err = h.db.QueryRow(`
		SELECT status, user_id, deleted_at FROM expenses WHERE id = $1 AND organization_id = $2
	`, expenseID, orgID).Scan(&status, &expenseUserID, &deletedAt)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "expense not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch expense")
		return
	}

	if deletedAt.Valid {
		api.RespondWithError(w, http.StatusNotFound, "expense not found")
		return
	}

	if status != string(models.StatusDraft) && status != string(models.StatusRejected) {
		api.RespondWithError(w, http.StatusBadRequest, "can only delete draft or rejected expenses")
		return
	}

	if expenseUserID.String() != userID.String() {
		api.RespondWithError(w, http.StatusForbidden, "can only delete own expenses")
		return
	}

	now := time.Now()
	_, err = h.db.Exec(`UPDATE expenses SET deleted_at = $1 WHERE id = $2`, now, expenseID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete expense")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ExpenseHandler) MonthlySummary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())
	role := middleware.GetRole(r.Context())

	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")
	userFilter := r.URL.Query().Get("user_id")

	if monthStr == "" || yearStr == "" {
		api.RespondWithError(w, http.StatusBadRequest, "month and year are required")
		return
	}

	month, _ := strconv.Atoi(monthStr)
	year, _ := strconv.Atoi(yearStr)

	if role == string(models.RoleEmployee) {
		userFilter = userID.String()
	}

	query := `
		SELECT e.date, e.id, e.status,
			   ei.project_id, p.name as project_name, ei.category, ei.amount
		FROM expenses e
		JOIN expense_items ei ON e.id = ei.expense_id
		JOIN projects p ON ei.project_id = p.id
		WHERE e.organization_id = $1
		AND EXTRACT(MONTH FROM e.date) = $2
		AND EXTRACT(YEAR FROM e.date) = $3
	`
	args := []interface{}{orgID, month, year}
	argIndex := 4

	if userFilter != "" {
		query += fmt.Sprintf(" AND e.user_id = $%d", argIndex)
		args = append(args, userFilter)
	}

	query += " ORDER BY e.date, p.name"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch summary")
		return
	}
	defer rows.Close()

	dayMap := make(map[string]*models.ExpenseDaySummary)
	projectTotals := make(map[string]float64)
	categoryTotals := make(map[string]float64)

	for rows.Next() {
		var date time.Time
		var entryID uuid.UUID
		var status string
		var projectID uuid.UUID
		var projectName string
		var category string
		var amount float64

		err := rows.Scan(&date, &entryID, &status, &projectID, &projectName, &category, &amount)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to scan summary row")
			return
		}

		dateStr := date.Format("2006-01-02")
		day, exists := dayMap[dateStr]
		if !exists {
			day = &models.ExpenseDaySummary{
				Date:  dateStr,
				Items: []models.ExpenseItemSummary{},
			}
			dayMap[dateStr] = day
		}
		day.TotalAmount += amount
		day.Items = append(day.Items, models.ExpenseItemSummary{
			ProjectID:   projectID.String(),
			ProjectName: projectName,
			Category:    category,
			Amount:      amount,
		})

		projectTotals[projectName] += amount
		categoryTotals[category] += amount
	}

	days := make([]models.ExpenseDaySummary, 0, len(dayMap))
	for _, day := range dayMap {
		days = append(days, *day)
	}

	summary := models.ExpenseMonthlySummary{
		Days:       days,
		Totals:     projectTotals,
		Categories: categoryTotals,
	}

	api.RespondWithJSON(w, http.StatusOK, summary)
}

func (h *ExpenseHandler) GetReceipt(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())
	role := middleware.GetRole(r.Context())

	receiptIDStr := r.PathValue("id")
	receiptID, err := uuid.Parse(receiptIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid receipt id")
		return
	}

	var filePath string
	var originalFilename string
	var expenseUserID uuid.UUID
	err = h.db.QueryRow(`
		SELECT er.file_path, er.original_filename, e.user_id
		FROM expense_receipts er
		JOIN expense_items ei ON er.expense_item_id = ei.id
		JOIN expenses e ON ei.expense_id = e.id
		WHERE er.id = $1 AND e.organization_id = $2
	`, receiptID, orgID).Scan(&filePath, &originalFilename, &expenseUserID)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "receipt not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch receipt")
		return
	}

	if role == string(models.RoleEmployee) && expenseUserID.String() != userID.String() {
		api.RespondWithError(w, http.StatusForbidden, "can only access own receipts")
		return
	}

	http.ServeFile(w, r, filePath)
}

func (h *ExpenseHandler) getExpenseItems(expenseID uuid.UUID) ([]models.ExpenseItem, error) {
	rows, err := h.db.Query(`
		SELECT ei.id, ei.expense_id, ei.project_id, p.name, ei.category, ei.amount, ei.km_distance, ei.description
		FROM expense_items ei
		JOIN projects p ON ei.project_id = p.id
		WHERE ei.expense_id = $1
		ORDER BY ei.created_at
	`, expenseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.ExpenseItem
	for rows.Next() {
		var item models.ExpenseItem
		var kmDistance sql.NullFloat64
		var desc sql.NullString
		err := rows.Scan(
			&item.ID, &item.ExpenseID, &item.ProjectID,
			&item.ProjectName, &item.Category, &item.Amount,
			&kmDistance, &desc,
		)
		if err != nil {
			return nil, err
		}
		if kmDistance.Valid {
			item.KmDistance = &kmDistance.Float64
		}
		if desc.Valid {
			item.Description = desc.String
		}

		receiptRows, err := h.db.Query(`
			SELECT id, expense_item_id, file_path, original_filename, uploaded_at
			FROM expense_receipts WHERE expense_item_id = $1
		`, item.ID)
		if err != nil {
			return nil, err
		}

		var receipts []models.ExpenseReceipt
		for receiptRows.Next() {
			var receipt models.ExpenseReceipt
			var origFilename sql.NullString
			err := receiptRows.Scan(
				&receipt.ID, &receipt.ExpenseItemID, &receipt.FilePath,
				&origFilename, &receipt.UploadedAt,
			)
			if err != nil {
				receiptRows.Close()
				return nil, err
			}
			if origFilename.Valid {
				receipt.OriginalFilename = origFilename.String
			}
			receipts = append(receipts, receipt)
		}
		receiptRows.Close()

		if receipts == nil {
			receipts = []models.ExpenseReceipt{}
		}
		item.Receipts = receipts

		items = append(items, item)
	}

	if items == nil {
		items = []models.ExpenseItem{}
	}

	return items, nil
}

func (h *ExpenseHandler) saveReceipt(file io.Reader, filename string, entryDate time.Time) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".pdf" {
		return "", fmt.Errorf("invalid file type: only jpg, png, and pdf are allowed")
	}

	dir := filepath.Join(h.uploadDir, entryDate.Format("2006"), entryDate.Format("01"))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	newFilename := uuid.New().String() + ext
	filePath := filepath.Join(dir, newFilename)

	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return filePath, nil
}
