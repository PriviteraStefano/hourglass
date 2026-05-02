package http

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/core/services/export"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type ExportHandler struct {
	service *export.Service
}

func NewExportHandler(service *export.Service) *ExportHandler {
	return &ExportHandler{service: service}
}

func (h *ExportHandler) Timesheets(w http.ResponseWriter, r *http.Request) {
	h.writeCSV(w, r, "timesheets", func(from, to time.Time, role string) ([]csvRow, error) {
		rows, err := h.service.Timesheets(r.Context(), middleware.GetOrganizationID(r.Context()), from, to, role, middleware.GetUserID(r.Context()))
		if err != nil {
			return nil, err
		}
		return toCSVRows(rows), nil
	}, []string{"Date", "Employee", "Project", "Contract", "Customer", "Hours", "Description", "Status"})
}

func (h *ExportHandler) Expenses(w http.ResponseWriter, r *http.Request) {
	h.writeCSV(w, r, "expenses", func(from, to time.Time, role string) ([]csvRow, error) {
		rows, err := h.service.Expenses(r.Context(), middleware.GetOrganizationID(r.Context()), from, to, role, middleware.GetUserID(r.Context()))
		if err != nil {
			return nil, err
		}
		return toExpenseCSVRows(rows), nil
	}, []string{"Date", "Employee", "Project", "Contract", "Customer", "Type", "Amount", "Km Distance", "Description", "Status"})
}

func (h *ExportHandler) Combined(w http.ResponseWriter, r *http.Request) {
	h.writeCSV(w, r, "combined", func(from, to time.Time, role string) ([]csvRow, error) {
		rows, err := h.service.Combined(r.Context(), middleware.GetOrganizationID(r.Context()), from, to, middleware.GetRole(r.Context()), middleware.GetUserID(r.Context()))
		if err != nil {
			return nil, err
		}
		return toCombinedCSVRows(rows), nil
	}, []string{"Entry Type", "Date", "Employee", "Project", "Contract", "Customer", "Hours", "Amount", "Km Distance", "Type", "Description", "Status"})
}

type csvRow []string

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
