package handlers

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type ExportHandler struct {
	db *sql.DB
}

func NewExportHandler(db *sql.DB) *ExportHandler {
	return &ExportHandler{db: db}
}

func (h *ExportHandler) Timesheets(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())
	userRole := middleware.GetRole(r.Context())

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0).Add(-time.Second)

	if fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = parsed
		}
	}
	if toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			to = parsed
		}
	}

	query := `
		SELECT te.date, u.name as employee, p.name as project, c.name as contract,
			   COALESCE(cu.company_name, '') as customer, te.hours, te.description, te.status
		FROM time_entries te
		JOIN users u ON te.user_id = u.id
		LEFT JOIN projects p ON te.project_id = p.id
		LEFT JOIN contracts c ON p.contract_id = c.id
		LEFT JOIN customers cu ON c.customer_id = cu.id
		WHERE te.organization_id = $1 AND te.date >= $2 AND te.date <= $3 AND te.deleted_at IS NULL
	`
	args := []interface{}{orgID, from, to}
	argIndex := 4

	if userRole == string(models.RoleEmployee) {
		query += fmt.Sprintf(" AND te.user_id = $%d", argIndex)
		args = append(args, userID)
	} else if userRole == string(models.RoleManager) {
		query += fmt.Sprintf(" AND (te.user_id = $%d OR te.project_id IN (SELECT project_id FROM project_managers WHERE user_id = $%d))", argIndex, argIndex)
		args = append(args, userID)
	}

	query += " ORDER BY te.date DESC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch timesheets")
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=timesheets_%s_%s.csv", from.Format("2006-01-02"), to.Format("2006-01-02")))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"Date", "Employee", "Project", "Contract", "Customer", "Hours", "Description", "Status"})

	for rows.Next() {
		var date time.Time
		var employee, project, contract, customer, description, status string
		var hours sql.NullFloat64
		if err := rows.Scan(&date, &employee, &project, &contract, &customer, &hours, &description, &status); err != nil {
			continue
		}
		hoursStr := ""
		if hours.Valid {
			hoursStr = strconv.FormatFloat(hours.Float64, 'f', 2, 64)
		}
		writer.Write([]string{
			date.Format("2006-01-02"),
			employee,
			project,
			contract,
			customer,
			hoursStr,
			description,
			status,
		})
	}
}

func (h *ExportHandler) Expenses(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())
	userRole := middleware.GetRole(r.Context())

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0).Add(-time.Second)

	if fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = parsed
		}
	}
	if toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			to = parsed
		}
	}

	query := `
		SELECT e.date, u.name as employee, p.name as project, c.name as contract,
			   COALESCE(cu.company_name, '') as customer, e.type, e.amount, e.km_distance, e.description, e.status
		FROM expenses e
		JOIN users u ON e.user_id = u.id
		LEFT JOIN projects p ON e.project_id = p.id
		LEFT JOIN contracts c ON p.contract_id = c.id
		LEFT JOIN customers cu ON e.customer_id = cu.id
		WHERE e.organization_id = $1 AND e.date >= $2 AND e.date <= $3 AND e.deleted_at IS NULL
	`
	args := []interface{}{orgID, from, to}
	argIndex := 4

	if userRole == string(models.RoleEmployee) {
		query += fmt.Sprintf(" AND e.user_id = $%d", argIndex)
		args = append(args, userID)
	} else if userRole == string(models.RoleManager) {
		query += fmt.Sprintf(" AND (e.user_id = $%d OR e.project_id IN (SELECT project_id FROM project_managers WHERE user_id = $%d))", argIndex, argIndex)
		args = append(args, userID)
	}

	query += " ORDER BY e.date DESC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch expenses")
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=expenses_%s_%s.csv", from.Format("2006-01-02"), to.Format("2006-01-02")))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"Date", "Employee", "Project", "Contract", "Customer", "Type", "Amount", "Km Distance", "Description", "Status"})

	for rows.Next() {
		var date time.Time
		var employee, project, contract, customer, description, status string
		var expenseType sql.NullString
		var amount, kmDistance sql.NullFloat64
		if err := rows.Scan(&date, &employee, &project, &contract, &customer, &expenseType, &amount, &kmDistance, &description, &status); err != nil {
			continue
		}
		amountStr := ""
		if amount.Valid {
			amountStr = strconv.FormatFloat(amount.Float64, 'f', 2, 64)
		}
		kmStr := ""
		if kmDistance.Valid {
			kmStr = strconv.FormatFloat(kmDistance.Float64, 'f', 2, 64)
		}
		writer.Write([]string{
			date.Format("2006-01-02"),
			employee,
			project,
			contract,
			customer,
			expenseType.String,
			amountStr,
			kmStr,
			description,
			status,
		})
	}
}

func (h *ExportHandler) Combined(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())
	userRole := middleware.GetRole(r.Context())

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0).Add(-time.Second)

	if fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = parsed
		}
	}
	if toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			to = parsed
		}
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=combined_%s_%s.csv", from.Format("2006-01-02"), to.Format("2006-01-02")))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"Entry Type", "Date", "Employee", "Project", "Contract", "Customer", "Hours", "Amount", "Km Distance", "Type", "Description", "Status"})

	userFilter := ""
	args := []interface{}{orgID, from, to}
	if userRole == string(models.RoleEmployee) {
		userFilter = " AND user_id = $4"
		args = append(args, userID)
	} else if userRole == string(models.RoleManager) {
		userFilter = " AND (user_id = $4 OR project_id IN (SELECT project_id FROM project_managers WHERE user_id = $4))"
		args = append(args, userID)
	}

	timeQuery := `
		SELECT 'time_entry' as entry_type, te.date, u.name, p.name, c.name, COALESCE(cu.company_name, ''),
			   te.hours, NULL, NULL, NULL, te.description, te.status
		FROM time_entries te
		JOIN users u ON te.user_id = u.id
		LEFT JOIN projects p ON te.project_id = p.id
		LEFT JOIN contracts c ON p.contract_id = c.id
		LEFT JOIN customers cu ON c.customer_id = cu.id
		WHERE te.organization_id = $1 AND te.date >= $2 AND te.date <= $3 AND te.deleted_at IS NULL
	` + userFilter + `
		UNION ALL
		SELECT 'expense', e.date, u.name, p.name, c.name, COALESCE(cu.company_name, ''),
			   NULL, e.amount, e.km_distance, e.type, e.description, e.status
		FROM expenses e
		JOIN users u ON e.user_id = u.id
		LEFT JOIN projects p ON e.project_id = p.id
		LEFT JOIN contracts c ON p.contract_id = c.id
		LEFT JOIN customers cu ON e.customer_id = cu.id
		WHERE e.organization_id = $1 AND e.date >= $2 AND e.date <= $3 AND e.deleted_at IS NULL
	` + userFilter + `
		ORDER BY date DESC
	`

	rows, err := h.db.Query(timeQuery, args...)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch entries")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var entryType, employee, project, contract, customer, description, status string
		var date time.Time
		var hours, amount, kmDistance sql.NullFloat64
		var expenseType sql.NullString
		if err := rows.Scan(&entryType, &date, &employee, &project, &contract, &customer, &hours, &amount, &kmDistance, &expenseType, &description, &status); err != nil {
			continue
		}
		hoursStr := ""
		if hours.Valid {
			hoursStr = strconv.FormatFloat(hours.Float64, 'f', 2, 64)
		}
		amountStr := ""
		if amount.Valid {
			amountStr = strconv.FormatFloat(amount.Float64, 'f', 2, 64)
		}
		kmStr := ""
		if kmDistance.Valid {
			kmStr = strconv.FormatFloat(kmDistance.Float64, 'f', 2, 64)
		}
		typeStr := ""
		if expenseType.Valid {
			typeStr = expenseType.String
		}
		writer.Write([]string{
			entryType,
			date.Format("2006-01-02"),
			employee,
			project,
			contract,
			customer,
			hoursStr,
			amountStr,
			kmStr,
			typeStr,
			description,
			status,
		})
	}
}
