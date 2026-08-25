package http

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/core/services/export"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

// maxExportRange bounds how much history a single export may span. Without a
// cap an org with years of history can exhaust server memory building the full
// result set (CONCERNS.md #7). 731 days ≈ 2 years.
const maxExportRange = 731 * 24 * time.Hour

// xlsxSheet describes a single sheet in an XLSX workbook.
type xlsxSheet struct {
	Name   string
	Rows   []ports.ExportRow
	Conv   func(ports.ExportRow) csvRow
	Header []string
}

type ExportHandler struct {
	service *export.Service
}

func NewExportHandler(service *export.Service) *ExportHandler {
	return &ExportHandler{service: service}
}

func (h *ExportHandler) Timesheets(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "xlsx" {
		h.writeTimesheetsXLSX(w, r)
		return
	}

	h.writeCSV(w, r, "timesheets",
		func(from, to time.Time, role string) ([]ports.ExportRow, error) {
			orgID := middleware.GetOrganizationID(r.Context())
			userID := middleware.GetUserID(r.Context())
			return h.service.Timesheets(r.Context(), orgID, from, to, role, userID)
		},
		timeSheetRow,
		[]string{"Date", "Employee", "Project", "Contract", "Customer", "Hours", "Description", "Status"})
}

func (h *ExportHandler) Expenses(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "xlsx" {
		h.writeExpensesXLSX(w, r)
		return
	}

	h.writeCSV(w, r, "expenses",
		func(from, to time.Time, role string) ([]ports.ExportRow, error) {
			orgID := middleware.GetOrganizationID(r.Context())
			userID := middleware.GetUserID(r.Context())
			return h.service.Expenses(r.Context(), orgID, from, to, role, userID)
		},
		expenseRow,
		[]string{"Date", "Employee", "Project", "Contract", "Customer", "Type", "Amount", "Km Distance", "Description", "Status"})
}

func (h *ExportHandler) Combined(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "xlsx" {
		h.writeCombinedXLSX(w, r)
		return
	}

	h.writeCSV(w, r, "combined",
		func(from, to time.Time, role string) ([]ports.ExportRow, error) {
			orgID := middleware.GetOrganizationID(r.Context())
			userID := middleware.GetUserID(r.Context())
			role = middleware.GetRole(r.Context())
			return h.service.Combined(r.Context(), orgID, from, to, role, userID)
		},
		combinedRow,
		[]string{"Entry Type", "Date", "Employee", "Project", "Contract", "Customer", "Hours", "Amount", "Km Distance", "Type", "Description", "Status"})
}

type csvRow []string

func (h *ExportHandler) CountTimesheets(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseExportRange(r)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid export range")
		return
	}
	role := middleware.GetRole(r.Context())
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	count, err := h.service.CountTimesheets(r.Context(), orgID, from, to, role, userID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to count export data")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *ExportHandler) CountExpenses(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseExportRange(r)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid export range")
		return
	}
	role := middleware.GetRole(r.Context())
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	count, err := h.service.CountExpenses(r.Context(), orgID, from, to, role, userID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to count export data")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *ExportHandler) CountCombined(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseExportRange(r)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid export range")
		return
	}
	role := middleware.GetRole(r.Context())
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	count, err := h.service.CountCombined(r.Context(), orgID, from, to, role, userID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to count export data")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, map[string]int{"count": count})
}

// writeXLSX streams an XLSX workbook to the response using excelize's
// StreamWriter so the full result set is never held in the workbook model at
// once (CONCERNS.md #7). Rows are converted and written one at a time.
func (h *ExportHandler) writeXLSX(w http.ResponseWriter, r *http.Request, prefix string, from, to time.Time, sheets []xlsxSheet) {
	f := excelize.NewFile()
	defer f.Close()

	boldStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})

	for i, sheet := range sheets {
		var sheetName string
		if i == 0 {
			sheetName = f.GetSheetName(0)
			f.SetSheetName(sheetName, sheet.Name)
			sheetName = sheet.Name
		} else {
			sheetName = sheet.Name
			f.NewSheet(sheetName)
		}

		sw, err := f.NewStreamWriter(sheetName)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to build export")
			return
		}

		// Header row (bold)
		headerCells := make([]interface{}, len(sheet.Header))
		for j, hdr := range sheet.Header {
			headerCells[j] = excelize.Cell{Value: hdr, StyleID: boldStyle}
		}
		startCell, _ := excelize.CoordinatesToCellName(1, 1)
		if err := sw.SetRow(startCell, headerCells); err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to build export")
			return
		}

		// Data rows streamed one at a time
		for rowIdx, row := range sheet.Rows {
			cell, _ := excelize.CoordinatesToCellName(1, rowIdx+2)
			values := make([]interface{}, len(sheet.Header))
			for k, v := range sheet.Conv(row) {
				values[k] = v
			}
			if err := sw.SetRow(cell, values); err != nil {
				api.RespondWithError(w, http.StatusInternalServerError, "failed to build export")
				return
			}
		}

		if err := sw.Flush(); err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to build export")
			return
		}

		for j := range sheet.Header {
			colName, _ := excelize.ColumnNumberToName(j + 1)
			f.SetColWidth(sheetName, colName, colName, 18)
		}
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%s_%s_%s.xlsx", prefix, from.Format("2006-01-02"), to.Format("2006-01-02")))
	f.Write(w)
}

// writeTimesheetsXLSX fetches timesheet data and writes it as an XLSX file.
func (h *ExportHandler) writeTimesheetsXLSX(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())
	userID := middleware.GetUserID(r.Context())
	from, to, err := parseExportRange(r)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid export range")
		return
	}

	rows, err := h.service.Timesheets(r.Context(), orgID, from, to, role, userID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch export data")
		return
	}

	h.writeXLSX(w, r, "timesheets", from, to, []xlsxSheet{
		{
			Name:   "Timesheets",
			Rows:   rows,
			Conv:   timeSheetRow,
			Header: []string{"Date", "Employee", "Project", "Contract", "Customer", "Hours", "Description", "Status"},
		},
	})
}

// writeExpensesXLSX fetches expense data and writes it as an XLSX file.
func (h *ExportHandler) writeExpensesXLSX(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())
	userID := middleware.GetUserID(r.Context())
	from, to, err := parseExportRange(r)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid export range")
		return
	}

	rows, err := h.service.Expenses(r.Context(), orgID, from, to, role, userID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch export data")
		return
	}

	h.writeXLSX(w, r, "expenses", from, to, []xlsxSheet{
		{
			Name:   "Expenses",
			Rows:   rows,
			Conv:   expenseRow,
			Header: []string{"Date", "Employee", "Project", "Contract", "Customer", "Type", "Amount", "Km Distance", "Description", "Status"},
		},
	})
}

// writeCombinedXLSX fetches timesheet and expense data and writes them as two sheets in an XLSX file.
func (h *ExportHandler) writeCombinedXLSX(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())
	userID := middleware.GetUserID(r.Context())
	from, to, err := parseExportRange(r)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid export range")
		return
	}

	teRows, err := h.service.Timesheets(r.Context(), orgID, from, to, role, userID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch export data")
		return
	}

	expRows, err := h.service.Expenses(r.Context(), orgID, from, to, role, userID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch export data")
		return
	}

	h.writeXLSX(w, r, "combined", from, to, []xlsxSheet{
		{
			Name:   "Timesheets",
			Rows:   teRows,
			Conv:   timeSheetRow,
			Header: []string{"Date", "Employee", "Project", "Contract", "Customer", "Hours", "Description", "Status"},
		},
		{
			Name:   "Expenses",
			Rows:   expRows,
			Conv:   expenseRow,
			Header: []string{"Date", "Employee", "Project", "Contract", "Customer", "Type", "Amount", "Km Distance", "Description", "Status"},
		},
	})
}

// writeCSV streams CSV rows to the response. Rows are converted and written one
// at a time via csv.Writer, so the full result set is never buffered as an
// intermediate []csvRow slice (CONCERNS.md #7).
func (h *ExportHandler) writeCSV(w http.ResponseWriter, r *http.Request, prefix string, fetch func(time.Time, time.Time, string) ([]ports.ExportRow, error), conv func(ports.ExportRow) csvRow, header []string) {
	role := middleware.GetRole(r.Context())
	from, to, err := parseExportRange(r)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid export range")
		return
	}

	rows, err := fetch(from, to, role)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch export data")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s_%s_%s.csv", prefix, from.Format("2006-01-02"), to.Format("2006-01-02")))
	writer := csv.NewWriter(w)
	defer writer.Flush()
	if err := writer.Write(header); err != nil {
		return
	}
	for _, row := range rows {
		if err := writer.Write(conv(row)); err != nil {
			return
		}
	}
}

func parseExportRange(r *http.Request) (time.Time, time.Time, error) {
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0).Add(-time.Second)
	if v := r.URL.Query().Get("from"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			from = parsed
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			to = parsed
		}
	}
	if to.Before(from) {
		return from, to, fmt.Errorf("to before from")
	}
	if to.Sub(from) > maxExportRange {
		return from, to, fmt.Errorf("range exceeds maximum")
	}
	return from, to, nil
}

// queryProjectID returns the ?project_id= query param value, if any.
func queryProjectID(r *http.Request) string {
	return r.URL.Query().Get("project_id")
}

// queryUserID returns the ?user_id= query param value, if any.
func queryUserID(r *http.Request) string {
	return r.URL.Query().Get("user_id")
}

func timeSheetRow(row ports.ExportRow) csvRow {
	return csvRow{
		row.Date.Format("2006-01-02"),
		row.Employee,
		row.Project,
		row.Contract,
		row.Customer,
		formatFloat(row.Hours),
		row.Description,
		row.Status,
	}
}

func expenseRow(row ports.ExportRow) csvRow {
	return csvRow{
		row.Date.Format("2006-01-02"),
		row.Employee,
		row.Project,
		row.Contract,
		row.Customer,
		row.Type,
		formatFloat(row.Amount),
		formatFloat(row.KmDistance),
		row.Description,
		row.Status,
	}
}

func combinedRow(row ports.ExportRow) csvRow {
	return csvRow{
		row.EntryType,
		row.Date.Format("2006-01-02"),
		row.Employee,
		row.Project,
		row.Contract,
		row.Customer,
		formatFloat(row.Hours),
		formatFloat(row.Amount),
		formatFloat(row.KmDistance),
		row.Type,
		row.Description,
		row.Status,
	}
}

func formatFloat(v *float64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'f', 2, 64)
}
