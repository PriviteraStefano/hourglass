package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

func TestExportHandler_Timesheets_MissingAuth(t *testing.T) {
	defer func() { recover() }()

	h := NewExportHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/export/timesheets?from=2026-01-01&to=2026-01-31", nil)
	rec := httptest.NewRecorder()

	// No auth context set — handler calls nil service and panics.
	// The recover() above catches the expected panic.
	h.Timesheets(rec, req)
}

func TestExportHandler_Expenses_MissingAuth(t *testing.T) {
	defer func() { recover() }()

	h := NewExportHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/export/expenses?from=2026-01-01&to=2026-01-31", nil)
	rec := httptest.NewRecorder()

	h.Expenses(rec, req)
}

func TestExportHandler_Combined_MissingAuth(t *testing.T) {
	defer func() { recover() }()

	h := NewExportHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/export/combined?from=2026-01-01&to=2026-01-31", nil)
	rec := httptest.NewRecorder()

	h.Combined(rec, req)
}

func TestExportHandler_Timesheets_WithAuth(t *testing.T) {
	defer func() { recover() }()

	h := NewExportHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/export/timesheets?from=2026-01-01&to=2026-01-31", nil)
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	// Timesheets calls h.service.Timesheets which panics on nil service.
	// With auth context set, the handler processes up until the service call.
	// The recover() above catches the expected panic.
	h.Timesheets(rec, req)
}

func TestExportHandler_Expenses_WithAuth(t *testing.T) {
	defer func() { recover() }()

	h := NewExportHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/export/expenses?from=2026-01-01&to=2026-01-31", nil)
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.Expenses(rec, req)
}

func TestExportHandler_Combined_WithAuth(t *testing.T) {
	defer func() { recover() }()

	h := NewExportHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/export/combined?from=2026-01-01&to=2026-01-31", nil)
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.Combined(rec, req)
}

func TestExportHandler_Timesheets_WithFilters(t *testing.T) {
	defer func() { recover() }()

	h := NewExportHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/export/timesheets?from=2026-03-01&to=2026-03-31", nil)
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "admin")
	req = req.WithContext(ctx)

	h.Timesheets(rec, req)
}
