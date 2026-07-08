package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/core/services/export"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

// mockExportRepoForTests returns known data for CSV and count tests.
type mockExportRepoForTests struct{}

func (m *mockExportRepoForTests) Timesheets(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ports.ExportRow, error) {
	return []ports.ExportRow{
		{Date: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), Employee: "Alice", Project: "Project A", Hours: float64Ptr(8), Description: "Work", Status: "approved"},
	}, nil
}

func (m *mockExportRepoForTests) Expenses(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ports.ExportRow, error) {
	return []ports.ExportRow{
		{Date: time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC), Employee: "Alice", Project: "Project A", Amount: float64Ptr(50), Type: "meal", Description: "Lunch", Status: "approved"},
	}, nil
}

func (m *mockExportRepoForTests) CountTimesheets(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) (int, error) {
	return 5, nil
}

func (m *mockExportRepoForTests) CountExpenses(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) (int, error) {
	return 3, nil
}

func float64Ptr(v float64) *float64 { return &v }

func newExportHandlerForTests() *ExportHandler {
	svc := export.NewService(&mockExportRepoForTests{})
	return NewExportHandler(svc)
}

func setAuthContext(req *http.Request) *http.Request {
	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	return req.WithContext(ctx)
}

// ---- CSV Success Tests ----

func TestExportHandler_Timesheets_CSV_Success(t *testing.T) {
	h := newExportHandlerForTests()

	req := httptest.NewRequest(http.MethodGet, "/exports/timesheets?from=2026-01-01&to=2026-01-31", nil)
	req = setAuthContext(req)
	rec := httptest.NewRecorder()

	h.Timesheets(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/csv", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment; filename=timesheets_")
	assert.Contains(t, rec.Body.String(), "Date,Employee,Project,Contract,Customer,Hours,Description,Status")
	assert.Contains(t, rec.Body.String(), "Alice")
}

func TestExportHandler_Expenses_CSV_Success(t *testing.T) {
	h := newExportHandlerForTests()

	req := httptest.NewRequest(http.MethodGet, "/exports/expenses?from=2026-01-01&to=2026-01-31", nil)
	req = setAuthContext(req)
	rec := httptest.NewRecorder()

	h.Expenses(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/csv", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment; filename=expenses_")
	assert.Contains(t, rec.Body.String(), "Date,Employee,Project,Contract,Customer,Type,Amount,Km Distance,Description,Status")
	assert.Contains(t, rec.Body.String(), "meal")
}

func TestExportHandler_Combined_CSV_Success(t *testing.T) {
	h := newExportHandlerForTests()

	req := httptest.NewRequest(http.MethodGet, "/exports/combined?from=2026-01-01&to=2026-01-31", nil)
	req = setAuthContext(req)
	rec := httptest.NewRecorder()

	h.Combined(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/csv", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment; filename=combined_")
	assert.Contains(t, rec.Body.String(), "Entry Type,Date,Employee,Project,Contract,Customer,Hours,Amount,Km Distance,Type,Description,Status")
}

// ---- Count Success Tests ----

func TestExportHandler_CountTimesheets_Success(t *testing.T) {
	h := newExportHandlerForTests()

	req := httptest.NewRequest(http.MethodGet, "/exports/timesheets/count?from=2026-01-01&to=2026-01-31", nil)
	req = setAuthContext(req)
	rec := httptest.NewRecorder()

	h.CountTimesheets(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Count int `json:"count"`
		} `json:"data"`
	}
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 5, resp.Data.Count)
}

func TestExportHandler_CountExpenses_Success(t *testing.T) {
	h := newExportHandlerForTests()

	req := httptest.NewRequest(http.MethodGet, "/exports/expenses/count?from=2026-01-01&to=2026-01-31", nil)
	req = setAuthContext(req)
	rec := httptest.NewRecorder()

	h.CountExpenses(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Count int `json:"count"`
		} `json:"data"`
	}
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Data.Count)
}

func TestExportHandler_CountCombined_Success(t *testing.T) {
	h := newExportHandlerForTests()

	req := httptest.NewRequest(http.MethodGet, "/exports/combined/count?from=2026-01-01&to=2026-01-31", nil)
	req = setAuthContext(req)
	rec := httptest.NewRecorder()

	h.CountCombined(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Count int `json:"count"`
		} `json:"data"`
	}
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 8, resp.Data.Count) // 5 + 3 from mock
}

// ---- Missing Auth Test ----

func TestExportHandler_Timesheets_MissingAuth(t *testing.T) {
	h := newExportHandlerForTests()

	req := httptest.NewRequest(http.MethodGet, "/exports/timesheets?from=2026-01-01&to=2026-01-31", nil)
	rec := httptest.NewRecorder()

	// No auth context set — middleware.GetUserID returns zero UUID,
	// middleware.GetRole returns "". With nil role the roleFilter
	// returns default (admin/finance) which allows all data.
	h.Timesheets(rec, req)

	// Without auth context the handler still executes but gets empty
	// orgID/userID/role values. The mock service returns data regardless,
	// so we expect a 200 with CSV output, not a panic.
	assert.Equal(t, http.StatusOK, rec.Code)
}
