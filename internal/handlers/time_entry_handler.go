package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type TimeEntryHandler struct {
	db *sql.DB
}

func NewTimeEntryHandler(db *sql.DB) *TimeEntryHandler {
	return &TimeEntryHandler{db: db}
}

func (h *TimeEntryHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())
	role := middleware.GetRole(r.Context())

	query := `
		SELECT te.id, te.user_id, te.organization_id, te.date, te.status, 
			   te.current_approver_role, te.submitted_at, te.created_at, te.updated_at
		FROM time_entries te
		WHERE te.organization_id = $1
	`
	args := []interface{}{orgID}
	argIndex := 2

	date := r.URL.Query().Get("date")
	if date != "" {
		query += fmt.Sprintf(" AND te.date = $%d", argIndex)
		args = append(args, date)
		argIndex++
	}

	month := r.URL.Query().Get("month")
	year := r.URL.Query().Get("year")
	if month != "" && year != "" {
		query += fmt.Sprintf(" AND EXTRACT(MONTH FROM te.date) = $%d AND EXTRACT(YEAR FROM te.date) = $%d", argIndex, argIndex+1)
		args = append(args, month, year)
		argIndex += 2
	}

	filterUserID := r.URL.Query().Get("user_id")
	if filterUserID != "" {
		if role == string(models.RoleEmployee) && filterUserID != userID.String() {
			api.RespondWithError(w, http.StatusForbidden, "can only view own entries")
			return
		}
		query += fmt.Sprintf(" AND te.user_id = $%d", argIndex)
		args = append(args, filterUserID)
		argIndex++
	} else if role == string(models.RoleEmployee) {
		query += fmt.Sprintf(" AND te.user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	status := r.URL.Query().Get("status")
	if status != "" {
		query += fmt.Sprintf(" AND te.status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	query += " ORDER BY te.date DESC, te.created_at DESC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entries")
		return
	}
	defer rows.Close()

	var entries []models.TimeEntry
	for rows.Next() {
		var te models.TimeEntry
		var currentApproverRole sql.NullString
		var submittedAt sql.NullTime
		err := rows.Scan(
			&te.ID, &te.UserID, &te.OrganizationID, &te.Date,
			&te.Status, &currentApproverRole, &submittedAt,
			&te.CreatedAt, &te.UpdatedAt,
		)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to scan time entry")
			return
		}
		if currentApproverRole.Valid {
			te.CurrentApproverRole = &currentApproverRole.String
		}
		if submittedAt.Valid {
			te.SubmittedAt = &submittedAt.Time
		}
		entries = append(entries, te)
	}

	for i := range entries {
		items, err := h.getItems(entries[i].ID)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch entry items")
			return
		}
		entries[i].Items = items
	}

	if entries == nil {
		entries = []models.TimeEntry{}
	}

	api.RespondWithJSON(w, http.StatusOK, entries)
}

func (h *TimeEntryHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	var req models.TimeEntryCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Date == "" || len(req.Items) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "date and items are required")
		return
	}

	entryDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid date format")
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

	for _, item := range req.Items {
		projectID, err := uuid.Parse(item.ProjectID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid project id")
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
			api.RespondWithError(w, http.StatusBadRequest, "project not found or not accessible")
			return
		}
	}

	tx, err := h.db.Begin()
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	entryID := uuid.New()
	now := time.Now()

	_, err = tx.Exec(`
		INSERT INTO time_entries (id, user_id, organization_id, date, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
	`, entryID, userID, orgID, entryDate, models.StatusDraft, now)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create time entry")
		return
	}

	for _, item := range req.Items {
		projectID, _ := uuid.Parse(item.ProjectID)
		_, err = tx.Exec(`
			INSERT INTO time_entry_items (time_entry_id, project_id, hours, description)
			VALUES ($1, $2, $3, $4)
		`, entryID, projectID, item.Hours, item.Description)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to create time entry item")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	entry := models.TimeEntry{
		ID:             entryID,
		UserID:         userID,
		OrganizationID: orgID,
		Date:           entryDate,
		Status:         models.StatusDraft,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	entry.Items, _ = h.getItems(entryID)

	api.RespondWithJSON(w, http.StatusCreated, entry)
}

func (h *TimeEntryHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())
	role := middleware.GetRole(r.Context())

	entryIDStr := r.PathValue("id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	var te models.TimeEntry
	var entryUserID uuid.UUID
	var currentApproverRole sql.NullString
	var submittedAt sql.NullTime

	err = h.db.QueryRow(`
		SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at
		FROM time_entries WHERE id = $1 AND organization_id = $2
	`, entryID, orgID).Scan(
		&te.ID, &entryUserID, &te.OrganizationID, &te.Date,
		&te.Status, &currentApproverRole, &submittedAt,
		&te.CreatedAt, &te.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entry")
		return
	}

	if role == string(models.RoleEmployee) && entryUserID.String() != userID.String() {
		api.RespondWithError(w, http.StatusForbidden, "can only view own entries")
		return
	}

	if currentApproverRole.Valid {
		te.CurrentApproverRole = &currentApproverRole.String
	}
	if submittedAt.Valid {
		te.SubmittedAt = &submittedAt.Time
	}

	te.Items, err = h.getItems(te.ID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch entry items")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, te)
}

func (h *TimeEntryHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	entryIDStr := r.PathValue("id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	var status string
	var entryUserID uuid.UUID
	err = h.db.QueryRow(`
		SELECT status, user_id FROM time_entries WHERE id = $1 AND organization_id = $2
	`, entryID, orgID).Scan(&status, &entryUserID)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entry")
		return
	}

	if status != string(models.StatusDraft) {
		api.RespondWithError(w, http.StatusBadRequest, "can only update draft entries")
		return
	}

	if entryUserID.String() != userID.String() {
		api.RespondWithError(w, http.StatusForbidden, "can only update own entries")
		return
	}

	var req models.TimeEntryUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Items) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "items are required")
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

	for _, item := range req.Items {
		projectID, err := uuid.Parse(item.ProjectID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid project id")
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
			api.RespondWithError(w, http.StatusBadRequest, "project not found or not accessible")
			return
		}
	}

	tx, err := h.db.Begin()
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM time_entry_items WHERE time_entry_id = $1`, entryID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete old items")
		return
	}

	for _, item := range req.Items {
		projectID, _ := uuid.Parse(item.ProjectID)
		_, err = tx.Exec(`
			INSERT INTO time_entry_items (time_entry_id, project_id, hours, description)
			VALUES ($1, $2, $3, $4)
		`, entryID, projectID, item.Hours, item.Description)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to create time entry item")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	var te models.TimeEntry
	var currentApproverRole sql.NullString
	var submittedAt sql.NullTime
	err = h.db.QueryRow(`
		SELECT id, user_id, organization_id, date, status, current_approver_role, submitted_at, created_at, updated_at
		FROM time_entries WHERE id = $1
	`, entryID).Scan(
		&te.ID, &te.UserID, &te.OrganizationID, &te.Date,
		&te.Status, &currentApproverRole, &submittedAt,
		&te.CreatedAt, &te.UpdatedAt,
	)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch updated entry")
		return
	}

	if currentApproverRole.Valid {
		te.CurrentApproverRole = &currentApproverRole.String
	}
	if submittedAt.Valid {
		te.SubmittedAt = &submittedAt.Time
	}

	te.Items, _ = h.getItems(te.ID)

	api.RespondWithJSON(w, http.StatusOK, te)
}

func (h *TimeEntryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	entryIDStr := r.PathValue("id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	var status string
	var entryUserID uuid.UUID
	err = h.db.QueryRow(`
		SELECT status, user_id FROM time_entries WHERE id = $1 AND organization_id = $2
	`, entryID, orgID).Scan(&status, &entryUserID)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entry")
		return
	}

	if status != string(models.StatusDraft) {
		api.RespondWithError(w, http.StatusBadRequest, "can only delete draft entries")
		return
	}

	if entryUserID.String() != userID.String() {
		api.RespondWithError(w, http.StatusForbidden, "can only delete own entries")
		return
	}

	_, err = h.db.Exec(`DELETE FROM time_entries WHERE id = $1`, entryID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete time entry")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *TimeEntryHandler) MonthlySummary(w http.ResponseWriter, r *http.Request) {
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
		SELECT te.date, te.id, te.status,
			   tei.project_id, p.name as project_name, tei.hours
		FROM time_entries te
		JOIN time_entry_items tei ON te.id = tei.time_entry_id
		JOIN projects p ON tei.project_id = p.id
		WHERE te.organization_id = $1
		AND EXTRACT(MONTH FROM te.date) = $2
		AND EXTRACT(YEAR FROM te.date) = $3
	`
	args := []interface{}{orgID, month, year}
	argIndex := 4

	if userFilter != "" {
		query += fmt.Sprintf(" AND te.user_id = $%d", argIndex)
		args = append(args, userFilter)
	}

	query += " ORDER BY te.date, p.name"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch summary")
		return
	}
	defer rows.Close()

	dayMap := make(map[string]*models.TimeEntryDaySummary)
	projectTotals := make(map[string]float64)
	matrixMap := make(map[string]map[string]float64)
	matrixProjects := make(map[string]string)

	for rows.Next() {
		var date time.Time
		var entryID uuid.UUID
		var status string
		var projectID uuid.UUID
		var projectName string
		var hours float64

		err := rows.Scan(&date, &entryID, &status, &projectID, &projectName, &hours)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to scan summary row")
			return
		}

		dateStr := date.Format("2006-01-02")
		day, exists := dayMap[dateStr]
		if !exists {
			day = &models.TimeEntryDaySummary{
				Date:     dateStr,
				Projects: []models.TimeEntryProjectSummary{},
			}
			dayMap[dateStr] = day
		}
		day.TotalHours += hours
		day.Projects = append(day.Projects, models.TimeEntryProjectSummary{
			ProjectID:   projectID.String(),
			ProjectName: projectName,
			Hours:       hours,
		})

		projectKey := projectID.String()
		projectTotals[projectKey] += hours
		matrixProjects[projectKey] = projectName

		if matrixMap[projectKey] == nil {
			matrixMap[projectKey] = make(map[string]float64)
		}
		matrixMap[projectKey][dateStr] += hours
	}

	days := make([]models.TimeEntryDaySummary, 0, len(dayMap))
	for _, day := range dayMap {
		days = append(days, *day)
	}

	totals := make(map[string]float64)
	for projectID, total := range projectTotals {
		totals[matrixProjects[projectID]] = total
	}

	matrix := make([]models.TimeEntryMatrixRow, 0)
	for projectID, dayHours := range matrixMap {
		matrix = append(matrix, models.TimeEntryMatrixRow{
			Project: matrixProjects[projectID],
			Days:    dayHours,
			Total:   projectTotals[projectID],
		})
	}

	summary := models.TimeEntryMonthlySummary{
		Days:   days,
		Totals: totals,
		Matrix: matrix,
	}

	api.RespondWithJSON(w, http.StatusOK, summary)
}

func (h *TimeEntryHandler) getItems(entryID uuid.UUID) ([]models.TimeEntryItem, error) {
	rows, err := h.db.Query(`
		SELECT tei.id, tei.time_entry_id, tei.project_id, p.name, tei.hours, tei.description
		FROM time_entry_items tei
		JOIN projects p ON tei.project_id = p.id
		WHERE tei.time_entry_id = $1
		ORDER BY tei.created_at
	`, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.TimeEntryItem
	for rows.Next() {
		var item models.TimeEntryItem
		var desc sql.NullString
		err := rows.Scan(
			&item.ID, &item.TimeEntryID, &item.ProjectID,
			&item.ProjectName, &item.Hours, &desc,
		)
		if err != nil {
			return nil, err
		}
		if desc.Valid {
			item.Description = desc.String
		}
		items = append(items, item)
	}

	if items == nil {
		items = []models.TimeEntryItem{}
	}

	return items, nil
}

func parseMonthYear(r *http.Request) (int, int, error) {
	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")

	if monthStr == "" || yearStr == "" {
		return 0, 0, fmt.Errorf("month and year required")
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid month")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid year")
	}

	return month, year, nil
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func stringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, str) {
			return true
		}
	}
	return false
}
