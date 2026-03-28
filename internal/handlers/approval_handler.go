package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type ApprovalHandler struct {
	db *sql.DB
}

func NewApprovalHandler(db *sql.DB) *ApprovalHandler {
	return &ApprovalHandler{db: db}
}

func (h *ApprovalHandler) SubmitTimeEntry(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID:= middleware.GetOrganizationID(r.Context())

	entryIDStr := r.PathValue("id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	var entryStatus string
	var entryUserID uuid.UUID
	err = h.db.QueryRow(`
		SELECT status, user_id FROM time_entries 
		WHERE id = $1 AND organization_id = $2
	`, entryID, orgID).Scan(&entryStatus, &entryUserID)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entry")
		return
	}

	if entryStatus != string(models.StatusDraft) {
		api.RespondWithError(w, http.StatusBadRequest, "can only submit draft entries")
		return
	}

	if entryUserID != userID {
		api.RespondWithError(w, http.StatusForbidden, "can only submit own entries")
		return
	}

	var authorRole string
	err = h.db.QueryRow(`
		SELECT role FROM organization_memberships 
		WHERE user_id = $1 AND organization_id = $2
	`, userID, orgID).Scan(&authorRole)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get user role")
		return
	}

	nextApproverRole, shouldAutoApprove := h.getNextApproverRole(authorRole, userID, orgID)

	now := time.Now()
	var newStatus models.EntryStatus
	var currentApproverRole *string

	if shouldAutoApprove {
		newStatus = models.StatusApproved
	} else {
		newStatus = models.StatusPendingManager
		if nextApproverRole == "finance" {
			newStatus = models.StatusPendingFinance
		}
		currentApproverRole = &nextApproverRole
	}

	tx, err := h.db.Begin()
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE time_entries 
		SET status = $1, current_approver_role = $2, submitted_at = $3, updated_at = $3
		WHERE id = $4
	`, newStatus, currentApproverRole, now, entryID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update time entry")
		return
	}

	action := models.ActionSubmit
	changes := "{}"
	if shouldAutoApprove {
		action = models.ActionApprove
		changes = `{"auto_approved": true}`
	}

	_, err = tx.Exec(`
		INSERT INTO time_entry_approvals (time_entry_id, action, actor_user_id, actor_role, changes)
		VALUES ($1, $2, $3, $4, $5)
	`, entryID, action, userID, authorRole, changes)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create approval record")
		return
	}

	if err := tx.Commit(); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	var te models.TimeEntry
	var submittedAt sql.NullTime
	err = h.db.QueryRow(`
		SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at
		FROM time_entries WHERE id = $1
	`, entryID).Scan(
		&te.ID, &te.UserID, &te.OrganizationID, &te.Date,
		&te.Status, &te.CurrentApproverRole, &submittedAt,
		&te.CreatedAt, &te.UpdatedAt,
	)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch updated entry")
		return
	}

	if submittedAt.Valid {
		te.SubmittedAt = &submittedAt.Time
	}

	api.RespondWithJSON(w, http.StatusOK, te)
}

func (h *ApprovalHandler) SubmitTimeEntryMonth(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")

	if monthStr == "" || yearStr == "" {
		api.RespondWithError(w, http.StatusBadRequest, "month and year are required")
		return
	}

	month, _ := strconv.Atoi(monthStr)
	year, _ := strconv.Atoi(yearStr)

	rows, err := h.db.Query(`
		SELECT id FROM time_entries 
		WHERE user_id = $1 AND organization_id = $2 
		AND EXTRACT(MONTH FROM date) = $3 AND EXTRACT(YEAR FROM date) = $4
		AND status = 'draft'
	`, userID, orgID, month, year)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch draft entries")
		return
	}
	defer rows.Close()

	var entryIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to scan entry id")
			return
		}
		entryIDs = append(entryIDs, id)
	}

	if len(entryIDs) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.TimeEntry{})
		return
	}

	var authorRole string
	err = h.db.QueryRow(`
		SELECT role FROM organization_memberships 
		WHERE user_id = $1 AND organization_id = $2
	`, userID, orgID).Scan(&authorRole)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get user role")
		return
	}

	nextApproverRole, shouldAutoApprove := h.getNextApproverRole(authorRole, userID, orgID)

	now := time.Now()
	var newStatus models.EntryStatus
	var currentApproverRole *string

	if shouldAutoApprove {
		newStatus = models.StatusApproved
	} else {
		newStatus = models.StatusPendingManager
		if nextApproverRole == "finance" {
			newStatus = models.StatusPendingFinance
		}
		currentApproverRole = &nextApproverRole
	}

	tx, err := h.db.Begin()
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	for _, entryID := range entryIDs {
		_, err = tx.Exec(`
			UPDATE time_entries 
			SET status = $1, current_approver_role = $2, submitted_at = $3, updated_at = $3
			WHERE id = $4
		`, newStatus, currentApproverRole, now, entryID)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to update time entry")
			return
		}

		action := models.ActionSubmit
		changes := "{}"
		if shouldAutoApprove {
			action = models.ActionApprove
			changes = `{"auto_approved": true}`
		}

		_, err = tx.Exec(`
			INSERT INTO time_entry_approvals (time_entry_id, action, actor_user_id, actor_role, changes)
			VALUES ($1, $2, $3, $4, $5)
		`, entryID, action, userID, authorRole, changes)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to create approval record")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"submitted_count": len(entryIDs),
		"entry_ids":       entryIDs,
		"status":          newStatus,
	})
}

func (h *ApprovalHandler) SubmitExpense(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	expenseIDStr := r.PathValue("id")
	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	var expenseStatus string
	var expenseUserID uuid.UUID
	err = h.db.QueryRow(`
		SELECT status, user_id FROM expenses 
		WHERE id = $1 AND organization_id = $2
	`, expenseID, orgID).Scan(&expenseStatus, &expenseUserID)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "expense not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch expense")
		return
	}

	if expenseStatus != string(models.StatusDraft) {
		api.RespondWithError(w, http.StatusBadRequest, "can only submit draft expenses")
		return
	}

	if expenseUserID != userID {
		api.RespondWithError(w, http.StatusForbidden, "can only submit own expenses")
		return
	}

	var authorRole string
	err = h.db.QueryRow(`
		SELECT role FROM organization_memberships 
		WHERE user_id = $1 AND organization_id = $2
	`, userID, orgID).Scan(&authorRole)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get user role")
		return
	}

	nextApproverRole, shouldAutoApprove := h.getNextApproverRole(authorRole, userID, orgID)

	now := time.Now()
	var newStatus models.EntryStatus
	var currentApproverRole *string

	if shouldAutoApprove {
		newStatus = models.StatusApproved
	} else {
		newStatus = models.StatusPendingManager
		if nextApproverRole == "finance" {
			newStatus = models.StatusPendingFinance
		}
		currentApproverRole = &nextApproverRole
	}

	tx, err := h.db.Begin()
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE expenses 
		SET status = $1, current_approver_role = $2, submitted_at = $3, updated_at = $3
		WHERE id = $4
	`, newStatus, currentApproverRole, now, expenseID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update expense")
		return
	}

	action := models.ActionSubmit
	changes := "{}"
	if shouldAutoApprove {
		action = models.ActionApprove
		changes = `{"auto_approved": true}`
	}

	_, err = tx.Exec(`
		INSERT INTO expense_approvals (expense_id, action, actor_user_id, actor_role, changes)
		VALUES ($1, $2, $3, $4, $5)
	`, expenseID, action, userID, authorRole, changes)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create approval record")
		return
	}

	if err := tx.Commit(); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	var e models.Expense
	var submittedAt sql.NullTime
	err = h.db.QueryRow(`
		SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at
		FROM expenses WHERE id = $1
	`, expenseID).Scan(
		&e.ID, &e.UserID, &e.OrganizationID, &e.Date,
		&e.Status, &e.CurrentApproverRole, &submittedAt,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch updated expense")
		return
	}

	if submittedAt.Valid {
		e.SubmittedAt = &submittedAt.Time
	}

	api.RespondWithJSON(w, http.StatusOK, e)
}

func (h *ApprovalHandler) SubmitExpenseMonth(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")

	if monthStr == "" || yearStr == "" {
		api.RespondWithError(w, http.StatusBadRequest, "month and year are required")
		return
	}

	month, _ := strconv.Atoi(monthStr)
	year, _ := strconv.Atoi(yearStr)

	rows, err := h.db.Query(`
		SELECT id FROM expenses 
		WHERE user_id = $1 AND organization_id = $2 
		AND EXTRACT(MONTH FROM date) = $3 AND EXTRACT(YEAR FROM date) = $4
		AND status = 'draft'
	`, userID, orgID, month, year)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch draft expenses")
		return
	}
	defer rows.Close()

	var expenseIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to scan expense id")
			return
		}
		expenseIDs = append(expenseIDs, id)
	}

	if len(expenseIDs) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.Expense{})
		return
	}

	var authorRole string
	err = h.db.QueryRow(`
		SELECT role FROM organization_memberships 
		WHERE user_id = $1 AND organization_id = $2
	`, userID, orgID).Scan(&authorRole)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get user role")
		return
	}

	nextApproverRole, shouldAutoApprove := h.getNextApproverRole(authorRole, userID, orgID)

	now := time.Now()
	var newStatus models.EntryStatus
	var currentApproverRole *string

	if shouldAutoApprove {
		newStatus = models.StatusApproved
	} else {
		newStatus = models.StatusPendingManager
		if nextApproverRole == "finance" {
			newStatus = models.StatusPendingFinance
		}
		currentApproverRole = &nextApproverRole
	}

	tx, err := h.db.Begin()
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	for _, expenseID := range expenseIDs {
		_, err = tx.Exec(`
			UPDATE expenses 
			SET status = $1, current_approver_role = $2, submitted_at = $3, updated_at = $3
			WHERE id = $4
		`, newStatus, currentApproverRole, now, expenseID)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to update expense")
			return
		}

		action := models.ActionSubmit
		changes := "{}"
		if shouldAutoApprove {
			action = models.ActionApprove
			changes = `{"auto_approved": true}`
		}

		_, err = tx.Exec(`
			INSERT INTO expense_approvals (expense_id, action, actor_user_id, actor_role, changes)
			VALUES ($1, $2, $3, $4, $5)
		`, expenseID, action, userID, authorRole, changes)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to create approval record")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"submitted_count": len(expenseIDs),
		"expense_ids":     expenseIDs,
		"status":          newStatus,
	})
}

func (h *ApprovalHandler) getNextApproverRole(authorRole string, authorID uuid.UUID, orgID uuid.UUID) (string, bool) {
	if authorRole == string(models.RoleManager) {
		managerCount := h.countUsersWithRole(orgID, string(models.RoleManager))
		if managerCount == 1 {
			financeCount := h.countUsersWithRole(orgID, string(models.RoleFinance))
			if financeCount == 0 {
				return "", true
			}
			return "finance", false
		}
		return "manager", false
	}

	if authorRole == string(models.RoleFinance) {
		managerCount := h.countUsersWithRole(orgID, string(models.RoleManager))
		if managerCount == 0 {
			return "", true
		}
		return "manager", false
	}

	managerCount := h.countUsersWithRole(orgID, string(models.RoleManager))
	if managerCount == 0 {
		financeCount := h.countUsersWithRole(orgID, string(models.RoleFinance))
		if financeCount == 0 {
			return "", true
		}
		return "finance", false
	}

	return "manager", false
}

func (h *ApprovalHandler) countUsersWithRole(orgID uuid.UUID, role string) int {
	var count int
	backupCount := 0

	h.db.QueryRow(`
		SELECT COUNT(*) FROM organization_memberships 
		WHERE organization_id = $1 AND role = $2 AND is_active = true
	`, orgID, role).Scan(&count)

	h.db.QueryRow(`
		SELECT COUNT(*) FROM backup_approvers 
		WHERE organization_id = $1 AND role = $2
	`, orgID, role).Scan(&backupCount)

	return count + backupCount
}

func (h *ApprovalHandler) GetPendingTimeEntries(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	var userRole string
	err := h.db.QueryRow(`
		SELECT role FROM organization_memberships 
		WHERE user_id = $1 AND organization_id = $2 AND is_active = true
	`, userID, orgID).Scan(&userRole)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get user role")
		return
	}

	if userRole != string(models.RoleManager) && userRole != string(models.RoleFinance) {
		api.RespondWithError(w, http.StatusForbidden, "only managers or finance can view pending entries")
		return
	}

	isBackup := h.isBackupApprover(orgID, userID, userRole)
	if !isBackup && userRole != string(models.RoleManager) && userRole != string(models.RoleFinance) {
		api.RespondWithError(w, http.StatusForbidden, "not authorized to approve entries")
		return
	}

	rows, err := h.db.Query(`
		SELECT te.id, te.user_id, te.date, te.status, te.current_approver_role,
		       u.name as user_name,
		       EXTRACT(MONTH FROM te.date) as month, EXTRACT(YEAR FROM te.date) as year
		FROM time_entries te
		JOIN users u ON te.user_id = u.id
		WHERE te.organization_id = $1 
		  AND te.current_approver_role = $2
		  AND te.status IN ('pending_manager', 'pending_finance')
		ORDER BY te.user_id, te.date
	`, orgID, userRole)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch pending entries")
		return
	}
	defer rows.Close()

	groupsMap := make(map[string]*models.PendingEntryGroup)

	for rows.Next() {
		var entry models.PendingEntry
		var userID uuid.UUID
		var userName string
		var month, year int
		var approverRole sql.NullString

		err := rows.Scan(&entry.ID, &userID, &entry.Date, &entry.Status, &approverRole, &userName, &month, &year)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to scan entry")
			return
		}

		if approverRole.Valid {
			entry.CurrentApproverRole = approverRole.String
		}

		itemRows, err := h.db.Query(`
			SELECT tei.id, tei.project_id, p.name as project_name, tei.hours, tei.description
			FROM time_entry_items tei
			JOIN projects p ON tei.project_id = p.id
			WHERE tei.time_entry_id = $1
		`, entry.ID)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch entry items")
			return
		}

		var items []models.TimeEntryItem
		for itemRows.Next() {
			var item models.TimeEntryItem
			var description sql.NullString
			err := itemRows.Scan(&item.ID, &item.ProjectID, &item.ProjectName, &item.Hours, &description)
			if err != nil {
				itemRows.Close()
				api.RespondWithError(w, http.StatusInternalServerError, "failed to scan entry item")
				return
			}
			if description.Valid {
				item.Description = description.String
			}
			items = append(items, item)
		}
		itemRows.Close()
		entry.Items = items

		key := fmt.Sprintf("%s-%d-%d", userID.String(), year, month)
		if group, exists := groupsMap[key]; exists {
			group.Entries = append(group.Entries, entry)
		} else {
			groupsMap[key] = &models.PendingEntryGroup{
				UserID:   userID,
				UserName: userName,
				Month:    month,
				Year:     year,
				Entries:  []models.PendingEntry{entry},
			}
		}
	}

	groups := make([]models.PendingEntryGroup, 0, len(groupsMap))
	for _, g := range groupsMap {
		groups = append(groups, *g)
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"groups": groups,
	})
}

func (h *ApprovalHandler) GetPendingExpenses(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	var userRole string
	err := h.db.QueryRow(`
		SELECT role FROM organization_memberships 
		WHERE user_id = $1 AND organization_id = $2 AND is_active = true
	`, userID, orgID).Scan(&userRole)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get user role")
		return
	}

	if userRole != string(models.RoleManager) && userRole != string(models.RoleFinance) {
		api.RespondWithError(w, http.StatusForbidden, "only managers or finance can view pending expenses")
		return
	}

	rows, err := h.db.Query(`
		SELECT e.id, e.user_id, e.date, e.status, e.current_approver_role,
		       u.name as user_name,
		       EXTRACT(MONTH FROM e.date) as month, EXTRACT(YEAR FROM e.date) as year
		FROM expenses e
		JOIN users u ON e.user_id = u.id
		WHERE e.organization_id = $1 
		  AND e.current_approver_role = $2
		  AND e.status IN ('pending_manager', 'pending_finance')
		ORDER BY e.user_id, e.date
	`, orgID, userRole)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch pending expenses")
		return
	}
	defer rows.Close()

	groupsMap := make(map[string]*models.PendingEntryGroup)

	for rows.Next() {
		var entry models.PendingEntry
		var userID uuid.UUID
		var userName string
		var month, year int
		var approverRole sql.NullString

		err := rows.Scan(&entry.ID, &userID, &entry.Date, &entry.Status, &approverRole, &userName, &month, &year)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to scan expense")
			return
		}

		if approverRole.Valid {
			entry.CurrentApproverRole = approverRole.String
		}

		itemRows, err := h.db.Query(`
			SELECT ei.id, ei.project_id, p.name as project_name, ei.category, ei.amount, ei.km_distance, ei.description
			FROM expense_items ei
			JOIN projects p ON ei.project_id = p.id
			WHERE ei.expense_id = $1
		`, entry.ID)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch expense items")
			return
		}

		var items []models.ExpenseItem
		for itemRows.Next() {
			var item models.ExpenseItem
			var description sql.NullString
			var kmDistance sql.NullFloat64
			err := itemRows.Scan(&item.ID, &item.ProjectID, &item.ProjectName, &item.Category, &item.Amount, &kmDistance, &description)
			if err != nil {
				itemRows.Close()
				api.RespondWithError(w, http.StatusInternalServerError, "failed to scan expense item")
				return
			}
			if description.Valid {
				item.Description = description.String
			}
			if kmDistance.Valid {
				item.KmDistance = &kmDistance.Float64
			}
			items = append(items, item)
		}
		itemRows.Close()
		entry.Items = items

		key := fmt.Sprintf("%s-%d-%d", userID.String(), year, month)
		if group, exists := groupsMap[key]; exists {
			group.Entries = append(group.Entries, entry)
		} else {
			groupsMap[key] = &models.PendingEntryGroup{
				UserID:   userID,
				UserName: userName,
				Month:    month,
				Year:     year,
				Entries:  []models.PendingEntry{entry},
			}
		}
	}

	groups := make([]models.PendingEntryGroup, 0, len(groupsMap))
	for _, g := range groupsMap {
		groups = append(groups, *g)
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"groups": groups,
	})
}

func (h *ApprovalHandler) isBackupApprover(orgID uuid.UUID, userID uuid.UUID, role string) bool {
	var count int
	h.db.QueryRow(`
		SELECT COUNT(*) FROM backup_approvers 
		WHERE organization_id = $1 AND user_id = $2 AND role = $3
	`, orgID, userID, role).Scan(&count)
	return count > 0
}

func (h *ApprovalHandler) ApproveTimeEntry(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	entryIDStr := r.PathValue("id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	var userRole string
	err = h.db.QueryRow(`
		SELECT role FROM organization_memberships 
		WHERE user_id = $1 AND organization_id = $2 AND is_active = true
	`, userID, orgID).Scan(&userRole)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get user role")
		return
	}

	var entryStatus string
	var currentApproverRole sql.NullString
	var entryUserID uuid.UUID
	err = h.db.QueryRow(`
		SELECT status, current_approver_role, user_id FROM time_entries 
		WHERE id = $1 AND organization_id = $2
	`, entryID, orgID).Scan(&entryStatus, &currentApproverRole, &entryUserID)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entry")
		return
	}

	if entryStatus != string(models.StatusPendingManager) && entryStatus != string(models.StatusPendingFinance) {
		api.RespondWithError(w, http.StatusBadRequest, "entry is not pending approval")
		return
	}

	if !currentApproverRole.Valid {
		api.RespondWithError(w, http.StatusBadRequest, "entry has no assigned approver")
		return
	}

	if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
		api.RespondWithError(w, http.StatusForbidden, "not authorized to approve this entry")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	var newStatus models.EntryStatus
	var nextApproverRole *string

	if entryStatus == string(models.StatusPendingManager) {
		financeCount := h.countUsersWithRole(orgID, string(models.RoleFinance))
		if financeCount == 0 {
			newStatus = models.StatusApproved
		} else {
			newStatus = models.StatusPendingFinance
			financeRole := string(models.RoleFinance)
			nextApproverRole = &financeRole
		}
	} else {
		newStatus = models.StatusApproved
	}

	now := time.Now()
	_, err = tx.Exec(`
		UPDATE time_entries 
		SET status = $1, current_approver_role = $2, updated_at = $3
		WHERE id = $4
	`, newStatus, nextApproverRole, now, entryID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update time entry")
		return
	}

	_, err = tx.Exec(`
		INSERT INTO time_entry_approvals (time_entry_id, action, actor_user_id, actor_role)
		VALUES ($1, $2, $3, $4)
	`, entryID, models.ActionApprove, userID, userRole)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create approval record")
		return
	}

	if err := tx.Commit(); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	var te models.TimeEntry
	var submittedAt sql.NullTime
	err = h.db.QueryRow(`
		SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at
		FROM time_entries WHERE id = $1
	`, entryID).Scan(
		&te.ID, &te.UserID, &te.OrganizationID, &te.Date,
		&te.Status, &te.CurrentApproverRole, &submittedAt,
		&te.CreatedAt, &te.UpdatedAt,
	)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch updated entry")
		return
	}

	if submittedAt.Valid {
		te.SubmittedAt = &submittedAt.Time
	}

	api.RespondWithJSON(w, http.StatusOK, te)
}

func (h *ApprovalHandler) RejectTimeEntry(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	entryIDStr := r.PathValue("id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	var req models.RejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Comment == "" {
		api.RespondWithError(w, http.StatusBadRequest, "comment is required for rejection")
		return
	}

	var userRole string
	err = h.db.QueryRow(`
		SELECT role FROM organization_memberships 
		WHERE user_id = $1 AND organization_id = $2 AND is_active = true
	`, userID, orgID).Scan(&userRole)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get user role")
		return
	}

	var entryStatus string
	var currentApproverRole sql.NullString
	err = h.db.QueryRow(`
		SELECT status, current_approver_role FROM time_entries 
		WHERE id = $1 AND organization_id = $2
	`, entryID, orgID).Scan(&entryStatus, &currentApproverRole)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entry")
		return
	}

	if entryStatus != string(models.StatusPendingManager) && entryStatus != string(models.StatusPendingFinance) {
		api.RespondWithError(w, http.StatusBadRequest, "entry is not pending approval")
		return
	}

	if !currentApproverRole.Valid {
		api.RespondWithError(w, http.StatusBadRequest, "entry has no assigned approver")
		return
	}

	if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
		api.RespondWithError(w, http.StatusForbidden, "not authorized to reject this entry")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	now := time.Now()
	_, err = tx.Exec(`
		UPDATE time_entries 
		SET status = $1, current_approver_role = NULL, updated_at = $2
		WHERE id = $3
	`, models.StatusRejected, now, entryID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update time entry")
		return
	}

	_, err = tx.Exec(`
		INSERT INTO time_entry_approvals (time_entry_id, action, actor_user_id, actor_role, comment)
		VALUES ($1, $2, $3, $4, $5)
	`, entryID, models.ActionReject, userID, userRole, req.Comment)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create approval record")
		return
	}

	if err := tx.Commit(); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	var te models.TimeEntry
	var submittedAt sql.NullTime
	err = h.db.QueryRow(`
		SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at
		FROM time_entries WHERE id = $1
	`, entryID).Scan(
		&te.ID, &te.UserID, &te.OrganizationID, &te.Date,
		&te.Status, &te.CurrentApproverRole, &submittedAt,
		&te.CreatedAt, &te.UpdatedAt,
	)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch updated entry")
		return
	}

	if submittedAt.Valid {
		te.SubmittedAt = &submittedAt.Time
	}

	api.RespondWithJSON(w, http.StatusOK, te)
}

func (h *ApprovalHandler) ApproveExpense(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	expenseIDStr := r.PathValue("id")
	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	var userRole string
	err = h.db.QueryRow(`
		SELECT role FROM organization_memberships 
		WHERE user_id = $1 AND organization_id = $2 AND is_active = true
	`, userID, orgID).Scan(&userRole)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get user role")
		return
	}

	var expenseStatus string
	var currentApproverRole sql.NullString
	err = h.db.QueryRow(`
		SELECT status, current_approver_role FROM expenses 
		WHERE id = $1 AND organization_id = $2
	`, expenseID, orgID).Scan(&expenseStatus, &currentApproverRole)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "expense not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch expense")
		return
	}

	if expenseStatus != string(models.StatusPendingManager) && expenseStatus != string(models.StatusPendingFinance) {
		api.RespondWithError(w, http.StatusBadRequest, "expense is not pending approval")
		return
	}

	if !currentApproverRole.Valid {
		api.RespondWithError(w, http.StatusBadRequest, "expense has no assigned approver")
		return
	}

	if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
		api.RespondWithError(w, http.StatusForbidden, "not authorized to approve this expense")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	var newStatus models.EntryStatus
	var nextApproverRole *string

	if expenseStatus == string(models.StatusPendingManager) {
		financeCount := h.countUsersWithRole(orgID, string(models.RoleFinance))
		if financeCount == 0 {
			newStatus = models.StatusApproved
		} else {
			newStatus = models.StatusPendingFinance
			financeRole := string(models.RoleFinance)
			nextApproverRole = &financeRole
		}
	} else {
		newStatus = models.StatusApproved
	}

	now := time.Now()
	_, err = tx.Exec(`
		UPDATE expenses 
		SET status = $1, current_approver_role = $2, updated_at = $3
		WHERE id = $4
	`, newStatus, nextApproverRole, now, expenseID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update expense")
		return
	}

	_, err = tx.Exec(`
		INSERT INTO expense_approvals (expense_id, action, actor_user_id, actor_role)
		VALUES ($1, $2, $3, $4)
	`, expenseID, models.ActionApprove, userID, userRole)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create approval record")
		return
	}

	if err := tx.Commit(); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	var e models.Expense
	var submittedAt sql.NullTime
	err = h.db.QueryRow(`
		SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at
		FROM expenses WHERE id = $1
	`, expenseID).Scan(
		&e.ID, &e.UserID, &e.OrganizationID, &e.Date,
		&e.Status, &e.CurrentApproverRole, &submittedAt,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch updated expense")
		return
	}

	if submittedAt.Valid {
		e.SubmittedAt = &submittedAt.Time
	}

	api.RespondWithJSON(w, http.StatusOK, e)
}

func (h *ApprovalHandler) RejectExpense(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	expenseIDStr := r.PathValue("id")
	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	var req models.RejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Comment == "" {
		api.RespondWithError(w, http.StatusBadRequest, "comment is required for rejection")
		return
	}

	var userRole string
	err = h.db.QueryRow(`
		SELECT role FROM organization_memberships 
		WHERE user_id = $1 AND organization_id = $2 AND is_active = true
	`, userID, orgID).Scan(&userRole)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get user role")
		return
	}

	var expenseStatus string
	var currentApproverRole sql.NullString
	err = h.db.QueryRow(`
		SELECT status, current_approver_role FROM expenses 
		WHERE id = $1 AND organization_id = $2
	`, expenseID, orgID).Scan(&expenseStatus, &currentApproverRole)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "expense not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch expense")
		return
	}

	if expenseStatus != string(models.StatusPendingManager) && expenseStatus != string(models.StatusPendingFinance) {
		api.RespondWithError(w, http.StatusBadRequest, "expense is not pending approval")
		return
	}

	if !currentApproverRole.Valid {
		api.RespondWithError(w, http.StatusBadRequest, "expense has no assigned approver")
		return
	}

	if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
		api.RespondWithError(w, http.StatusForbidden, "not authorized to reject this expense")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	now := time.Now()
	_, err = tx.Exec(`
		UPDATE expenses 
		SET status = $1, current_approver_role = NULL, updated_at = $2
		WHERE id = $3
	`, models.StatusRejected, now, expenseID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update expense")
		return
	}

	_, err = tx.Exec(`
		INSERT INTO expense_approvals (expense_id, action, actor_user_id, actor_role, comment)
		VALUES ($1, $2, $3, $4, $5)
	`, expenseID, models.ActionReject, userID, userRole, req.Comment)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create approval record")
		return
	}

	if err := tx.Commit(); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	var e models.Expense
	var submittedAt sql.NullTime
	err = h.db.QueryRow(`
		SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at
		FROM expenses WHERE id = $1
	`, expenseID).Scan(
		&e.ID, &e.UserID, &e.OrganizationID, &e.Date,
		&e.Status, &e.CurrentApproverRole, &submittedAt,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch updated expense")
		return
	}

	if submittedAt.Valid {
		e.SubmittedAt = &submittedAt.Time
	}

	api.RespondWithJSON(w, http.StatusOK, e)
}

func (h *ApprovalHandler) EditApproveTimeEntry(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	entryIDStr := r.PathValue("id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	var req models.EditApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Items) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "items are required")
		return
	}

	var userRole string
	h.db.QueryRow(`SELECT role FROM organization_memberships WHERE user_id = $1 AND organization_id = $2 AND is_active = true`, userID, orgID).Scan(&userRole)

	var entryStatus string
	var currentApproverRole sql.NullString
	h.db.QueryRow(`SELECT status, current_approver_role FROM time_entries WHERE id = $1 AND organization_id = $2`, entryID, orgID).Scan(&entryStatus, &currentApproverRole)

	if entryStatus != string(models.StatusPendingManager) && entryStatus != string(models.StatusPendingFinance) {
		api.RespondWithError(w, http.StatusBadRequest, "entry is not pending approval")
		return
	}
	if !currentApproverRole.Valid {
		api.RespondWithError(w, http.StatusBadRequest, "entry has no assigned approver")
		return
	}
	if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
		api.RespondWithError(w, http.StatusForbidden, "not authorized to approve this entry")
		return
	}

	var totalHours float64
	for _, item := range req.Items {
		totalHours += item.Hours
	}
	if totalHours > 24 {
		api.RespondWithError(w, http.StatusBadRequest, "total hours cannot exceed 24 per day")
		return
	}

	originalItems, _ := h.getTimeEntryItems(entryID)
	changesJSON, _ := json.Marshal(map[string]interface{}{"original_items": originalItems, "new_items": req.Items})

	tx, _ := h.db.Begin()
	defer tx.Rollback()

	tx.Exec(`DELETE FROM time_entry_items WHERE time_entry_id = $1`, entryID)
	for _, item := range req.Items {
		projectID, _ := uuid.Parse(item.ProjectID)
		tx.Exec(`INSERT INTO time_entry_items (time_entry_id, project_id, hours, description) VALUES ($1, $2, $3, $4)`, entryID, projectID, item.Hours, item.Description)
	}

	var newStatus models.EntryStatus
	var nextApproverRole *string
	if entryStatus == string(models.StatusPendingManager) {
		if h.countUsersWithRole(orgID, string(models.RoleFinance)) == 0 {
			newStatus = models.StatusApproved
		} else {
			newStatus = models.StatusPendingFinance
			fr := string(models.RoleFinance)
			nextApproverRole = &fr
		}
	} else {
		newStatus = models.StatusApproved
	}

	now := time.Now()
	tx.Exec(`UPDATE time_entries SET status = $1, current_approver_role = $2, updated_at = $3 WHERE id = $4`, newStatus, nextApproverRole, now, entryID)
	tx.Exec(`INSERT INTO time_entry_approvals (time_entry_id, action, actor_user_id, actor_role, changes, comment) VALUES ($1, $2, $3, $4, $5, $6)`, entryID, models.ActionEditApprove, userID, userRole, string(changesJSON), req.Comment)
	tx.Commit()

	var te models.TimeEntry
	var submittedAt sql.NullTime
	h.db.QueryRow(`SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at FROM time_entries WHERE id = $1`, entryID).Scan(&te.ID, &te.UserID, &te.OrganizationID, &te.Date, &te.Status, &te.CurrentApproverRole, &submittedAt, &te.CreatedAt, &te.UpdatedAt)
	if submittedAt.Valid {
		te.SubmittedAt = &submittedAt.Time
	}
	te.Items, _ = h.getTimeEntryItems(entryID)
	api.RespondWithJSON(w, http.StatusOK, te)
}

func (h *ApprovalHandler) EditReturnTimeEntry(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	entryIDStr := r.PathValue("id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	var req models.EditReturnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Items) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "items are required")
		return
	}
	if req.Comment == "" {
		api.RespondWithError(w, http.StatusBadRequest, "comment is required for edit-return")
		return
	}

	var userRole string
	h.db.QueryRow(`SELECT role FROM organization_memberships WHERE user_id = $1 AND organization_id = $2 AND is_active = true`, userID, orgID).Scan(&userRole)

	var entryStatus string
	var currentApproverRole sql.NullString
	h.db.QueryRow(`SELECT status, current_approver_role FROM time_entries WHERE id = $1 AND organization_id = $2`, entryID, orgID).Scan(&entryStatus, &currentApproverRole)

	if entryStatus != string(models.StatusPendingManager) && entryStatus != string(models.StatusPendingFinance) {
		api.RespondWithError(w, http.StatusBadRequest, "entry is not pending approval")
		return
	}
	if !currentApproverRole.Valid {
		api.RespondWithError(w, http.StatusBadRequest, "entry has no assigned approver")
		return
	}
	if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
		api.RespondWithError(w, http.StatusForbidden, "not authorized to edit this entry")
		return
	}

	var totalHours float64
	for _, item := range req.Items {
		totalHours += item.Hours
	}
	if totalHours > 24 {
		api.RespondWithError(w, http.StatusBadRequest, "total hours cannot exceed 24 per day")
		return
	}

	originalItems, _ := h.getTimeEntryItems(entryID)
	changesJSON, _ := json.Marshal(map[string]interface{}{"original_items": originalItems, "new_items": req.Items})

	tx, _ := h.db.Begin()
	defer tx.Rollback()

	tx.Exec(`DELETE FROM time_entry_items WHERE time_entry_id = $1`, entryID)
	for _, item := range req.Items {
		projectID, _ := uuid.Parse(item.ProjectID)
		tx.Exec(`INSERT INTO time_entry_items (time_entry_id, project_id, hours, description) VALUES ($1, $2, $3, $4)`, entryID, projectID, item.Hours, item.Description)
	}

	now := time.Now()
	tx.Exec(`UPDATE time_entries SET status = $1, updated_at = $2 WHERE id = $3`, models.StatusSubmitted, now, entryID)
	tx.Exec(`INSERT INTO time_entry_approvals (time_entry_id, action, actor_user_id, actor_role, changes, comment) VALUES ($1, $2, $3, $4, $5, $6)`, entryID, models.ActionEditReturn, userID, userRole, string(changesJSON), req.Comment)
	tx.Commit()

	var te models.TimeEntry
	var submittedAt sql.NullTime
	h.db.QueryRow(`SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at FROM time_entries WHERE id = $1`, entryID).Scan(&te.ID, &te.UserID, &te.OrganizationID, &te.Date, &te.Status, &te.CurrentApproverRole, &submittedAt, &te.CreatedAt, &te.UpdatedAt)
	if submittedAt.Valid {
		te.SubmittedAt = &submittedAt.Time
	}
	te.Items, _ = h.getTimeEntryItems(entryID)
	api.RespondWithJSON(w, http.StatusOK, te)
}

func (h *ApprovalHandler) getTimeEntryItems(entryID uuid.UUID) ([]models.TimeEntryItem, error) {
	rows, err := h.db.Query(`SELECT tei.id, tei.time_entry_id, tei.project_id, p.name as project_name, tei.hours, tei.description FROM time_entry_items tei JOIN projects p ON tei.project_id = p.id WHERE tei.time_entry_id = $1`, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.TimeEntryItem
	for rows.Next() {
		var item models.TimeEntryItem
		var desc sql.NullString
		rows.Scan(&item.ID, &item.TimeEntryID, &item.ProjectID, &item.ProjectName, &item.Hours, &desc)
		if desc.Valid {
			item.Description = desc.String
		}
		items = append(items, item)
	}
	return items, nil
}

func (h *ApprovalHandler) EditApproveExpense(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	expenseIDStr := r.PathValue("id")
	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	var req models.EditApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.ExpenseItems) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "expense_items are required")
		return
	}

	var userRole string
	h.db.QueryRow(`SELECT role FROM organization_memberships WHERE user_id = $1 AND organization_id = $2 AND is_active = true`, userID, orgID).Scan(&userRole)

	var expenseStatus string
	var currentApproverRole sql.NullString
	h.db.QueryRow(`SELECT status, current_approver_role FROM expenses WHERE id = $1 AND organization_id = $2`, expenseID, orgID).Scan(&expenseStatus, &currentApproverRole)

	if expenseStatus != string(models.StatusPendingManager) && expenseStatus != string(models.StatusPendingFinance) {
		api.RespondWithError(w, http.StatusBadRequest, "expense is not pending approval")
		return
	}
	if !currentApproverRole.Valid {
		api.RespondWithError(w, http.StatusBadRequest, "expense has no assigned approver")
		return
	}
	if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
		api.RespondWithError(w, http.StatusForbidden, "not authorized to approve this expense")
		return
	}

	originalItems, _ := h.getExpenseItems(expenseID)
	changesJSON, _ := json.Marshal(map[string]interface{}{"original_items": originalItems, "new_items": req.ExpenseItems})

	tx, _ := h.db.Begin()
	defer tx.Rollback()

	tx.Exec(`DELETE FROM expense_items WHERE expense_id = $1`, expenseID)
	for _, item := range req.ExpenseItems {
		projectID, _ := uuid.Parse(item.ProjectID)
		tx.Exec(`INSERT INTO expense_items (expense_id, project_id, category, amount, km_distance, description) VALUES ($1, $2, $3, $4, $5, $6)`, expenseID, projectID, item.Category, item.Amount, item.KmDistance, item.Description)
	}

	var newStatus models.EntryStatus
	var nextApproverRole *string
	if expenseStatus == string(models.StatusPendingManager) {
		if h.countUsersWithRole(orgID, string(models.RoleFinance)) == 0 {
			newStatus = models.StatusApproved
		} else {
			newStatus = models.StatusPendingFinance
			fr := string(models.RoleFinance)
			nextApproverRole = &fr
		}
	} else {
		newStatus = models.StatusApproved
	}

	now := time.Now()
	tx.Exec(`UPDATE expenses SET status = $1, current_approver_role = $2, updated_at = $3 WHERE id = $4`, newStatus, nextApproverRole, now, expenseID)
	tx.Exec(`INSERT INTO expense_approvals (expense_id, action, actor_user_id, actor_role, changes, comment) VALUES ($1, $2, $3, $4, $5, $6)`, expenseID, models.ActionEditApprove, userID, userRole, string(changesJSON), req.Comment)
	tx.Commit()

	var e models.Expense
	var submittedAt sql.NullTime
	h.db.QueryRow(`SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at FROM expenses WHERE id = $1`, expenseID).Scan(&e.ID, &e.UserID, &e.OrganizationID, &e.Date, &e.Status, &e.CurrentApproverRole, &submittedAt, &e.CreatedAt, &e.UpdatedAt)
	if submittedAt.Valid {
		e.SubmittedAt = &submittedAt.Time
	}
	e.Items, _ = h.getExpenseItems(expenseID)
	api.RespondWithJSON(w, http.StatusOK, e)
}

func (h *ApprovalHandler) EditReturnExpense(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	expenseIDStr := r.PathValue("id")
	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	var req models.EditReturnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.ExpenseItems) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "expense_items are required")
		return
	}
	if req.Comment == "" {
		api.RespondWithError(w, http.StatusBadRequest, "comment is required for edit-return")
		return
	}

	var userRole string
	h.db.QueryRow(`SELECT role FROM organization_memberships WHERE user_id = $1 AND organization_id = $2 AND is_active = true`, userID, orgID).Scan(&userRole)

	var expenseStatus string
	var currentApproverRole sql.NullString
	h.db.QueryRow(`SELECT status, current_approver_role FROM expenses WHERE id = $1 AND organization_id = $2`, expenseID, orgID).Scan(&expenseStatus, &currentApproverRole)

	if expenseStatus != string(models.StatusPendingManager) && expenseStatus != string(models.StatusPendingFinance) {
		api.RespondWithError(w, http.StatusBadRequest, "expense is not pending approval")
		return
	}
	if !currentApproverRole.Valid {
		api.RespondWithError(w, http.StatusBadRequest, "expense has no assigned approver")
		return
	}
	if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
		api.RespondWithError(w, http.StatusForbidden, "not authorized to edit this expense")
		return
	}

	originalItems, _ := h.getExpenseItems(expenseID)
	changesJSON, _ := json.Marshal(map[string]interface{}{"original_items": originalItems, "new_items": req.ExpenseItems})

	tx, _ := h.db.Begin()
	defer tx.Rollback()

	tx.Exec(`DELETE FROM expense_items WHERE expense_id = $1`, expenseID)
	for _, item := range req.ExpenseItems {
		projectID, _ := uuid.Parse(item.ProjectID)
		tx.Exec(`INSERT INTO expense_items (expense_id, project_id, category, amount, km_distance, description) VALUES ($1, $2, $3, $4, $5, $6)`, expenseID, projectID, item.Category, item.Amount, item.KmDistance, item.Description)
	}

	now := time.Now()
	tx.Exec(`UPDATE expenses SET status = $1, updated_at = $2 WHERE id = $3`, models.StatusSubmitted, now, expenseID)
	tx.Exec(`INSERT INTO expense_approvals (expense_id, action, actor_user_id, actor_role, changes, comment) VALUES ($1, $2, $3, $4, $5, $6)`, expenseID, models.ActionEditReturn, userID, userRole, string(changesJSON), req.Comment)
	tx.Commit()

	var e models.Expense
	var submittedAt sql.NullTime
	h.db.QueryRow(`SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at FROM expenses WHERE id = $1`, expenseID).Scan(&e.ID, &e.UserID, &e.OrganizationID, &e.Date, &e.Status, &e.CurrentApproverRole, &submittedAt, &e.CreatedAt, &e.UpdatedAt)
	if submittedAt.Valid {
		e.SubmittedAt = &submittedAt.Time
	}
	e.Items, _ = h.getExpenseItems(expenseID)
	api.RespondWithJSON(w, http.StatusOK, e)
}

func (h *ApprovalHandler) getExpenseItems(expenseID uuid.UUID) ([]models.ExpenseItem, error) {
	rows, err := h.db.Query(`SELECT ei.id, ei.expense_id, ei.project_id, p.name as project_name, ei.category, ei.amount, ei.km_distance, ei.description FROM expense_items ei JOIN projects p ON ei.project_id = p.id WHERE ei.expense_id = $1`, expenseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.ExpenseItem
	for rows.Next() {
		var item models.ExpenseItem
		var desc sql.NullString
		var km sql.NullFloat64
		rows.Scan(&item.ID, &item.ExpenseID, &item.ProjectID, &item.ProjectName, &item.Category, &item.Amount, &km, &desc)
		if desc.Valid {
			item.Description = desc.String
		}
		if km.Valid {
			item.KmDistance = &km.Float64
		}
		items = append(items, item)
	}
	return items, nil
}

func (h *ApprovalHandler) PartialApproveTimeEntry(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	entryIDStr := r.PathValue("id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	var req models.PartialApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.ApprovedItemIDs) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "approved_item_ids are required")
		return
	}

	var userRole string
	h.db.QueryRow(`SELECT role FROM organization_memberships WHERE user_id = $1 AND organization_id = $2 AND is_active = true`, userID, orgID).Scan(&userRole)

	var entryStatus string
	var currentApproverRole sql.NullString
	h.db.QueryRow(`SELECT status, current_approver_role FROM time_entries WHERE id = $1 AND organization_id = $2`, entryID, orgID).Scan(&entryStatus, &currentApproverRole)

	if entryStatus != string(models.StatusPendingManager) && entryStatus != string(models.StatusPendingFinance) {
		api.RespondWithError(w, http.StatusBadRequest, "entry is not pending approval")
		return
	}
	if !currentApproverRole.Valid {
		api.RespondWithError(w, http.StatusBadRequest, "entry has no assigned approver")
		return
	}
	if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
		api.RespondWithError(w, http.StatusForbidden, "not authorized to approve this entry")
		return
	}

	approvedIDs := make([]uuid.UUID, len(req.ApprovedItemIDs))
	for i, idStr := range req.ApprovedItemIDs {
		approvedIDs[i], _ = uuid.Parse(idStr)
	}

	changesJSON, _ := json.Marshal(map[string]interface{}{"approved_item_ids": req.ApprovedItemIDs})

	tx, _ := h.db.Begin()
	defer tx.Rollback()

	now := time.Now()
	tx.Exec(`INSERT INTO time_entry_approvals (time_entry_id, action, actor_user_id, actor_role, changes, comment) VALUES ($1, $2, $3, $4, $5, $6)`, entryID, models.ActionPartialApprove, userID, userRole, string(changesJSON), req.Comment)

	allItems, _ := h.getTimeEntryItems(entryID)
	allApproved := len(approvedIDs) == len(allItems)

	if allApproved {
		var newStatus models.EntryStatus
		var nextApproverRole *string
		if entryStatus == string(models.StatusPendingManager) {
			if h.countUsersWithRole(orgID, string(models.RoleFinance)) == 0 {
				newStatus = models.StatusApproved
			} else {
				newStatus = models.StatusPendingFinance
				fr := string(models.RoleFinance)
				nextApproverRole = &fr
			}
		} else {
			newStatus = models.StatusApproved
		}
		tx.Exec(`UPDATE time_entries SET status = $1, current_approver_role = $2, updated_at = $3 WHERE id = $4`, newStatus, nextApproverRole, now, entryID)
	} else {
		tx.Exec(`UPDATE time_entries SET status = $1, updated_at = $2 WHERE id = $3`, models.StatusSubmitted, now, entryID)
	}
	tx.Commit()

	var te models.TimeEntry
	var submittedAt sql.NullTime
	h.db.QueryRow(`SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at FROM time_entries WHERE id = $1`, entryID).Scan(&te.ID, &te.UserID, &te.OrganizationID, &te.Date, &te.Status, &te.CurrentApproverRole, &submittedAt, &te.CreatedAt, &te.UpdatedAt)
	if submittedAt.Valid {
		te.SubmittedAt = &submittedAt.Time
	}
	te.Items, _ = h.getTimeEntryItems(entryID)
	api.RespondWithJSON(w, http.StatusOK, te)
}

func (h *ApprovalHandler) DelegateTimeEntry(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	entryIDStr := r.PathValue("id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	var req models.DelegateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var userRole string
	h.db.QueryRow(`SELECT role FROM organization_memberships WHERE user_id = $1 AND organization_id = $2 AND is_active = true`, userID, orgID).Scan(&userRole)

	var entryStatus string
	var currentApproverRole sql.NullString
	h.db.QueryRow(`SELECT status, current_approver_role FROM time_entries WHERE id = $1 AND organization_id = $2`, entryID, orgID).Scan(&entryStatus, &currentApproverRole)

	if entryStatus != string(models.StatusPendingManager) && entryStatus != string(models.StatusPendingFinance) {
		api.RespondWithError(w, http.StatusBadRequest, "entry is not pending approval")
		return
	}
	if !currentApproverRole.Valid {
		api.RespondWithError(w, http.StatusBadRequest, "entry has no assigned approver")
		return
	}
	if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
		api.RespondWithError(w, http.StatusForbidden, "not authorized to delegate this entry")
		return
	}

	var delegateRole string
	h.db.QueryRow(`SELECT role FROM organization_memberships WHERE user_id = $1 AND organization_id = $2 AND is_active = true`, req.DelegateToUserID, orgID).Scan(&delegateRole)
	if delegateRole != currentApproverRole.String {
		api.RespondWithError(w, http.StatusBadRequest, "delegate must have same role as current approver")
		return
	}

	changesJSON, _ := json.Marshal(map[string]interface{}{"delegated_to": req.DelegateToUserID.String()})

	tx, _ := h.db.Begin()
	defer tx.Rollback()

	tx.Exec(`INSERT INTO time_entry_approvals (time_entry_id, action, actor_user_id, actor_role, changes, comment) VALUES ($1, $2, $3, $4, $5, $6)`, entryID, models.ActionDelegate, userID, userRole, string(changesJSON), req.Comment)
	tx.Commit()

	var te models.TimeEntry
	var submittedAt sql.NullTime
	h.db.QueryRow(`SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at FROM time_entries WHERE id = $1`, entryID).Scan(&te.ID, &te.UserID, &te.OrganizationID, &te.Date, &te.Status, &te.CurrentApproverRole, &submittedAt, &te.CreatedAt, &te.UpdatedAt)
	if submittedAt.Valid {
		te.SubmittedAt = &submittedAt.Time
	}
	te.Items, _ = h.getTimeEntryItems(entryID)
	api.RespondWithJSON(w, http.StatusOK, te)
}

func (h *ApprovalHandler) BatchApproveTimeEntries(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	var req models.BatchApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.EntryIDs) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "entry_ids are required")
		return
	}

	var userRole string
	h.db.QueryRow(`SELECT role FROM organization_memberships WHERE user_id = $1 AND organization_id = $2 AND is_active = true`, userID, orgID).Scan(&userRole)

	tx, _ := h.db.Begin()
	defer tx.Rollback()

	now := time.Now()
	approvedCount := 0

	for _, entryID := range req.EntryIDs {
		var entryStatus string
		var currentApproverRole sql.NullString
		err := tx.QueryRow(`SELECT status, current_approver_role FROM time_entries WHERE id = $1 AND organization_id = $2 FOR UPDATE`, entryID, orgID).Scan(&entryStatus, &currentApproverRole)
		if err != nil {
			continue
		}

		if entryStatus != string(models.StatusPendingManager) && entryStatus != string(models.StatusPendingFinance) {
			continue
		}
		if !currentApproverRole.Valid {
			continue
		}
		if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
			continue
		}

		var newStatus models.EntryStatus
		var nextApproverRole *string
		if entryStatus == string(models.StatusPendingManager) {
			if h.countUsersWithRole(orgID, string(models.RoleFinance)) == 0 {
				newStatus = models.StatusApproved
			} else {
				newStatus = models.StatusPendingFinance
				fr := string(models.RoleFinance)
				nextApproverRole = &fr
			}
		} else {
			newStatus = models.StatusApproved
		}

		tx.Exec(`UPDATE time_entries SET status = $1, current_approver_role = $2, updated_at = $3 WHERE id = $4`, newStatus, nextApproverRole, now, entryID)
		tx.Exec(`INSERT INTO time_entry_approvals (time_entry_id, action, actor_user_id, actor_role, comment) VALUES ($1, $2, $3, $4, $5)`, entryID, models.ActionApprove, userID, userRole, req.Comment)
		approvedCount++
	}

	tx.Commit()
	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"approved_count": approvedCount, "requested_count": len(req.EntryIDs)})
}

func (h *ApprovalHandler) BatchRejectTimeEntries(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	var req models.BatchRejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.EntryIDs) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "entry_ids are required")
		return
	}
	if req.Comment == "" {
		api.RespondWithError(w, http.StatusBadRequest, "comment is required for batch reject")
		return
	}

	var userRole string
	h.db.QueryRow(`SELECT role FROM organization_memberships WHERE user_id = $1 AND organization_id = $2 AND is_active = true`, userID, orgID).Scan(&userRole)

	tx, _ := h.db.Begin()
	defer tx.Rollback()

	now := time.Now()
	rejectedCount := 0

	for _, entryID := range req.EntryIDs {
		var entryStatus string
		var currentApproverRole sql.NullString
		err := tx.QueryRow(`SELECT status, current_approver_role FROM time_entries WHERE id = $1 AND organization_id = $2 FOR UPDATE`, entryID, orgID).Scan(&entryStatus, &currentApproverRole)
		if err != nil {
			continue
		}

		if entryStatus != string(models.StatusPendingManager) && entryStatus != string(models.StatusPendingFinance) {
			continue
		}
		if !currentApproverRole.Valid {
			continue
		}
		if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
			continue
		}

		tx.Exec(`UPDATE time_entries SET status = $1, current_approver_role = NULL, updated_at = $2 WHERE id = $3`, models.StatusRejected, now, entryID)
		tx.Exec(`INSERT INTO time_entry_approvals (time_entry_id, action, actor_user_id, actor_role, comment) VALUES ($1, $2, $3, $4, $5)`, entryID, models.ActionReject, userID, userRole, req.Comment)
		rejectedCount++
	}

	tx.Commit()
	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"rejected_count": rejectedCount, "requested_count": len(req.EntryIDs)})
}

func (h *ApprovalHandler) PartialApproveExpense(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	expenseIDStr := r.PathValue("id")
	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	var req models.PartialApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.ApprovedItemIDs) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "approved_item_ids are required")
		return
	}

	var userRole string
	h.db.QueryRow(`SELECT role FROM organization_memberships WHERE user_id = $1 AND organization_id = $2 AND is_active = true`, userID, orgID).Scan(&userRole)

	var expenseStatus string
	var currentApproverRole sql.NullString
	h.db.QueryRow(`SELECT status, current_approver_role FROM expenses WHERE id = $1 AND organization_id = $2`, expenseID, orgID).Scan(&expenseStatus, &currentApproverRole)

	if expenseStatus != string(models.StatusPendingManager) && expenseStatus != string(models.StatusPendingFinance) {
		api.RespondWithError(w, http.StatusBadRequest, "expense is not pending approval")
		return
	}
	if !currentApproverRole.Valid {
		api.RespondWithError(w, http.StatusBadRequest, "expense has no assigned approver")
		return
	}
	if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
		api.RespondWithError(w, http.StatusForbidden, "not authorized to approve this expense")
		return
	}

	changesJSON, _ := json.Marshal(map[string]interface{}{"approved_item_ids": req.ApprovedItemIDs})

	tx, _ := h.db.Begin()
	defer tx.Rollback()

	now := time.Now()
	tx.Exec(`INSERT INTO expense_approvals (expense_id, action, actor_user_id, actor_role, changes, comment) VALUES ($1, $2, $3, $4, $5, $6)`, expenseID, models.ActionPartialApprove, userID, userRole, string(changesJSON), req.Comment)

	allItems, _ := h.getExpenseItems(expenseID)
	allApproved := len(req.ApprovedItemIDs) == len(allItems)

	if allApproved {
		var newStatus models.EntryStatus
		var nextApproverRole *string
		if expenseStatus == string(models.StatusPendingManager) {
			if h.countUsersWithRole(orgID, string(models.RoleFinance)) == 0 {
				newStatus = models.StatusApproved
			} else {
				newStatus = models.StatusPendingFinance
				fr := string(models.RoleFinance)
				nextApproverRole = &fr
			}
		} else {
			newStatus = models.StatusApproved
		}
		tx.Exec(`UPDATE expenses SET status = $1, current_approver_role = $2, updated_at = $3 WHERE id = $4`, newStatus, nextApproverRole, now, expenseID)
	} else {
		tx.Exec(`UPDATE expenses SET status = $1, updated_at = $2 WHERE id = $3`, models.StatusSubmitted, now, expenseID)
	}
	tx.Commit()

	var e models.Expense
	var submittedAt sql.NullTime
	h.db.QueryRow(`SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at FROM expenses WHERE id = $1`, expenseID).Scan(&e.ID, &e.UserID, &e.OrganizationID, &e.Date, &e.Status, &e.CurrentApproverRole, &submittedAt, &e.CreatedAt, &e.UpdatedAt)
	if submittedAt.Valid {
		e.SubmittedAt = &submittedAt.Time
	}
	e.Items, _ = h.getExpenseItems(expenseID)
	api.RespondWithJSON(w, http.StatusOK, e)
}

func (h *ApprovalHandler) DelegateExpense(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	expenseIDStr := r.PathValue("id")
	expenseID, err := uuid.Parse(expenseIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	var req models.DelegateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var userRole string
	h.db.QueryRow(`SELECT role FROM organization_memberships WHERE user_id = $1 AND organization_id = $2 AND is_active = true`, userID, orgID).Scan(&userRole)

	var expenseStatus string
	var currentApproverRole sql.NullString
	h.db.QueryRow(`SELECT status, current_approver_role FROM expenses WHERE id = $1 AND organization_id = $2`, expenseID, orgID).Scan(&expenseStatus, &currentApproverRole)

	if expenseStatus != string(models.StatusPendingManager) && expenseStatus != string(models.StatusPendingFinance) {
		api.RespondWithError(w, http.StatusBadRequest, "expense is not pending approval")
		return
	}
	if !currentApproverRole.Valid {
		api.RespondWithError(w, http.StatusBadRequest, "expense has no assigned approver")
		return
	}
	if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
		api.RespondWithError(w, http.StatusForbidden, "not authorized to delegate this expense")
		return
	}

	var delegateRole string
	h.db.QueryRow(`SELECT role FROM organization_memberships WHERE user_id = $1 AND organization_id = $2 AND is_active = true`, req.DelegateToUserID, orgID).Scan(&delegateRole)
	if delegateRole != currentApproverRole.String {
		api.RespondWithError(w, http.StatusBadRequest, "delegate must have same role as current approver")
		return
	}

	changesJSON, _ := json.Marshal(map[string]interface{}{"delegated_to": req.DelegateToUserID.String()})

	tx, _ := h.db.Begin()
	defer tx.Rollback()
	tx.Exec(`INSERT INTO expense_approvals (expense_id, action, actor_user_id, actor_role, changes, comment) VALUES ($1, $2, $3, $4, $5, $6)`, expenseID, models.ActionDelegate, userID, userRole, string(changesJSON), req.Comment)
	tx.Commit()

	var e models.Expense
	var submittedAt sql.NullTime
	h.db.QueryRow(`SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at FROM expenses WHERE id = $1`, expenseID).Scan(&e.ID, &e.UserID, &e.OrganizationID, &e.Date, &e.Status, &e.CurrentApproverRole, &submittedAt, &e.CreatedAt, &e.UpdatedAt)
	if submittedAt.Valid {
		e.SubmittedAt = &submittedAt.Time
	}
	e.Items, _ = h.getExpenseItems(expenseID)
	api.RespondWithJSON(w, http.StatusOK, e)
}

func (h *ApprovalHandler) BatchApproveExpenses(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	var req models.BatchApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.EntryIDs) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "entry_ids are required")
		return
	}

	var userRole string
	h.db.QueryRow(`SELECT role FROM organization_memberships WHERE user_id = $1 AND organization_id = $2 AND is_active = true`, userID, orgID).Scan(&userRole)

	tx, _ := h.db.Begin()
	defer tx.Rollback()

	now := time.Now()
	approvedCount := 0

	for _, expenseID := range req.EntryIDs {
		var expenseStatus string
		var currentApproverRole sql.NullString
		err := tx.QueryRow(`SELECT status, current_approver_role FROM expenses WHERE id = $1 AND organization_id = $2 FOR UPDATE`, expenseID, orgID).Scan(&expenseStatus, &currentApproverRole)
		if err != nil {
			continue
		}

		if expenseStatus != string(models.StatusPendingManager) && expenseStatus != string(models.StatusPendingFinance) {
			continue
		}
		if !currentApproverRole.Valid {
			continue
		}
		if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
			continue
		}

		var newStatus models.EntryStatus
		var nextApproverRole *string
		if expenseStatus == string(models.StatusPendingManager) {
			if h.countUsersWithRole(orgID, string(models.RoleFinance)) == 0 {
				newStatus = models.StatusApproved
			} else {
				newStatus = models.StatusPendingFinance
				fr := string(models.RoleFinance)
				nextApproverRole = &fr
			}
		} else {
			newStatus = models.StatusApproved
		}

		tx.Exec(`UPDATE expenses SET status = $1, current_approver_role = $2, updated_at = $3 WHERE id = $4`, newStatus, nextApproverRole, now, expenseID)
		tx.Exec(`INSERT INTO expense_approvals (expense_id, action, actor_user_id, actor_role, comment) VALUES ($1, $2, $3, $4, $5)`, expenseID, models.ActionApprove, userID, userRole, req.Comment)
		approvedCount++
	}

	tx.Commit()
	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"approved_count": approvedCount, "requested_count": len(req.EntryIDs)})
}

func (h *ApprovalHandler) BatchRejectExpenses(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	var req models.BatchRejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.EntryIDs) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "entry_ids are required")
		return
	}
	if req.Comment == "" {
		api.RespondWithError(w, http.StatusBadRequest, "comment is required for batch reject")
		return
	}

	var userRole string
	h.db.QueryRow(`SELECT role FROM organization_memberships WHERE user_id = $1 AND organization_id = $2 AND is_active = true`, userID, orgID).Scan(&userRole)

	tx, _ := h.db.Begin()
	defer tx.Rollback()

	now := time.Now()
	rejectedCount := 0

	for _, expenseID := range req.EntryIDs {
		var expenseStatus string
		var currentApproverRole sql.NullString
		err := tx.QueryRow(`SELECT status, current_approver_role FROM expenses WHERE id = $1 AND organization_id = $2 FOR UPDATE`, expenseID, orgID).Scan(&expenseStatus, &currentApproverRole)
		if err != nil {
			continue
		}

		if expenseStatus != string(models.StatusPendingManager) && expenseStatus != string(models.StatusPendingFinance) {
			continue
		}
		if !currentApproverRole.Valid {
			continue
		}
		if userRole != currentApproverRole.String && !h.isBackupApprover(orgID, userID, currentApproverRole.String) {
			continue
		}

		tx.Exec(`UPDATE expenses SET status = $1, current_approver_role = NULL, updated_at = $2 WHERE id = $3`, models.StatusRejected, now, expenseID)
		tx.Exec(`INSERT INTO expense_approvals (expense_id, action, actor_user_id, actor_role, comment) VALUES ($1, $2, $3, $4, $5)`, expenseID, models.ActionReject, userID, userRole, req.Comment)
		rejectedCount++
	}

	tx.Commit()
	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"rejected_count": rejectedCount, "requested_count": len(req.EntryIDs)})
}
