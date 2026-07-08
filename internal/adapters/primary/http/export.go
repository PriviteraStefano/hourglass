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

// xlsxSheet describes a single sheet in an XLSX workbook.
type xlsxSheet struct {
	Name   string
	Rows   []csvRow
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

	h.writeCSV(w, r, "timesheets", func(from, to time.Time, role string) ([]csvRow, error) {
		orgID := middleware.GetOrganizationID(r.Context())
		userID := middleware.GetUserID(r.Context())
		rows, err := h.service.Timesheets(r.Context(), orgID, from, to, role, userID)
		if err != nil {
			return nil, err
		}
		return toCSVRows(rows), nil
	}, []string{"Date", "Employee", "Project", "Contract", "Customer", "Hours", "Description", "Status"})
}

func (h *ExportHandler) Expenses(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "xlsx" {
		h.writeExpensesXLSX(w, r)
		return
	}

	h.writeCSV(w, r, "expenses", func(from, to time.Time, role string) ([]csvRow, error) {
		orgID := middleware.GetOrganizationID(r.Context())
		userID := middleware.GetUserID(r.Context())
		rows, err := h.service.Expenses(r.Context(), orgID, from, to, role, userID)
		if err != nil {
			return nil, err
		}
		return toExpenseCSVRows(rows), nil
	}, []string{"Date", "Employee", "Project", "Contract", "Customer", "Type", "Amount", "Km Distance", "Description", "Status"})
}

func (h *ExportHandler) Combined(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "xlsx" {
		h.writeCombinedXLSX(w, r)
		return
	}

	h.writeCSV(w, r, "combined", func(from, to time.Time, role string) ([]csvRow, error) {
		orgID := middleware.GetOrganizationID(r.Context())
		userID := middleware.GetUserID(r.Context())
		role = middleware.GetRole(r.Context())
		rows, err := h.service.Combined(r.Context(), orgID, from, to, role, userID)
		if err != nil {
			return nil, err
		}
		return toCombinedCSVRows(rows), nil
	}, []string{"Entry Type", "Date", "Employee", "Project", "Contract", "Customer", "Hours", "Amount", "Km Distance", "Type", "Description", "Status"})
}

type csvRow []string

// writeXLSX generates an XLSX workbook with the given sheets and writes it to the response.
// The first sheet renames the default "Sheet1"; subsequent sheets are added via f.NewSheet.
// Headers are bold, columns auto-sized to width 18, Content-Type and Content-Disposition are set.
func (h *ExportHandler) writeXLSX(w http.ResponseWriter, r *http.Request, prefix string, sheets []xlsxSheet) {
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

		// Write header row with bold style
		for j, hdr := range sheet.Header {
			cell, _ := excelize.CoordinatesToCellName(j+1, 1)
			f.SetCellValue(sheetName, cell, hdr)
			f.SetCellStyle(sheetName, cell, cell, boldStyle)
		}

		// Write data rows
		for rowIdx, row := range sheet.Rows {
			for colIdx, val := range row {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
				f.SetCellValue(sheetName, cell, val)
			}
		}

		// Auto-size columns
		for j := range sheet.Header {
			colName, _ := excelize.ColumnNumberToName(j + 1)
			f.SetColWidth(sheetName, colName, colName, 18)
		}
	}

	from, to := parseExportRange(r)
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
	from, to := parseExportRange(r)

	rows, err := h.service.Timesheets(r.Context(), orgID, from, to, role, userID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch export data")
		return
	}

	h.writeXLSX(w, r, "timesheets", []xlsxSheet{
		{
			Name:   "Timesheets",
			Rows:   toCSVRows(rows),
			Header: []string{"Date", "Employee", "Project", "Contract", "Customer", "Hours", "Description", "Status"},
		},
	})
}

// writeExpensesXLSX fetches expense data and writes it as an XLSX file.
func (h *ExportHandler) writeExpensesXLSX(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())
	userID := middleware.GetUserID(r.Context())
	from, to := parseExportRange(r)

	rows, err := h.service.Expenses(r.Context(), orgID, from, to, role, userID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch export data")
		return
	}

	h.writeXLSX(w, r, "expenses", []xlsxSheet{
		{
			Name:   "Expenses",
			Rows:   toExpenseCSVRows(rows),
			Header: []string{"Date", "Employee", "Project", "Contract", "Customer", "Type", "Amount", "Km Distance", "Description", "Status"},
		},
	})
}

// writeCombinedXLSX fetches timesheet and expense data and writes them as two sheets in an XLSX file.
func (h *ExportHandler) writeCombinedXLSX(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())
	userID := middleware.GetUserID(r.Context())
	from, to := parseExportRange(r)

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

	h.writeXLSX(w, r, "combined", []xlsxSheet{
		{
			Name:   "Timesheets",
			Rows:   toCSVRows(teRows),
			Header: []string{"Date", "Employee", "Project", "Contract", "Customer", "Hours", "Description", "Status"},
		},
		{
			Name:   "Expenses",
			Rows:   toExpenseCSVRows(expRows),
			Header: []string{"Date", "Employee", "Project", "Contract", "Customer", "Type", "Amount", "Km Distance", "Description", "Status"},
		},
	})
}

// writeCSV writes CSV data to the response writer.
// Note: The current implementation fetches all rows first, then writes them.
// For streaming (writing rows as the query returns them), rows would need to
// be written directly from the database cursor to the csv.Writer without
// intermediate buffering. This is acceptable for typical org sizes; a future
// optimization could add true streaming for orgs with very large exports.
func (h *ExportHandler) writeCSV(w http.ResponseWriter, r *http.Request, prefix string, fn func(time.Time, time.Time, string) ([]csvRow, error), header []string) {
	role := middleware.GetRole(r.Context())
	from, to := parseExportRange(r)
	rows, err := fn(from, to, role)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch export data")
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s_%s_%s.csv", prefix, from.Format("2006-01-02"), to.Format("2006-01-02")))
	writer := csv.NewWriter(w)
	defer writer.Flush()
	_ = writer.Write(header)
	for _, row := range rows {
		_ = writer.Write(row)
	}
}

func parseExportRange(r *http.Request) (time.Time, time.Time) {
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
	return from, to
}

// queryProjectID returns the ?project_id= query param value, if any.
func queryProjectID(r *http.Request) string {
	return r.URL.Query().Get("project_id")
}

// queryUserID returns the ?user_id= query param value, if any.
func queryUserID(r *http.Request) string {
	return r.URL.Query().Get("user_id")
}

func toCSVRows(rows []ports.ExportRow) []csvRow {
	out := make([]csvRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, csvRow{
			row.Date.Format("2006-01-02"),
			row.Employee,
			row.Project,
			row.Contract,
			row.Customer,
			formatFloat(row.Hours),
			row.Description,
			row.Status,
		})
	}
	return out
}

func toExpenseCSVRows(rows []ports.ExportRow) []csvRow {
	out := make([]csvRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, csvRow{
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
		})
	}
	return out
}

func toCombinedCSVRows(rows []ports.ExportRow) []csvRow {
	out := make([]csvRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, csvRow{
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
		})
	}
	return out
}

func formatFloat(v *float64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'f', 2, 64)
}
