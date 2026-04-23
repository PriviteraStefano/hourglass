package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/db"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

func TestSurrealTimeEntryHandler_Create(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	sdb, err := db.NewSurrealDB()
	if err != nil {
		t.Fatalf("Failed to connect to SurrealDB: %v", err)
	}
	defer sdb.Close()

	handler := NewSurrealTimeEntryHandler(sdb)

	tests := []struct {
		name       string
		payload    interface{}
		wantStatus int
	}{
		{
			name: "missing project_id",
			payload: CreateTimeEntryRequest{
				SubprojectID: "subproject:backend",
				WGID:         "wg:backend",
				UnitID:       "unit:engineering",
				Hours:        8,
				Description:  "Test entry",
				Date:         "2026-01-15",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing hours",
			payload: CreateTimeEntryRequest{
				ProjectID:    "project:demo",
				SubprojectID: "subproject:backend",
				WGID:         "wg:backend",
				UnitID:       "unit:engineering",
				Description:  "Test entry",
				Date:         "2026-01-15",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid hours - too many",
			payload: CreateTimeEntryRequest{
				ProjectID:    "project:demo",
				SubprojectID: "subproject:backend",
				WGID:         "wg:backend",
				UnitID:       "unit:engineering",
				Hours:        25,
				Description:  "Test entry",
				Date:         "2026-01-15",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid date format",
			payload: CreateTimeEntryRequest{
				ProjectID:    "project:demo",
				SubprojectID: "subproject:backend",
				WGID:         "wg:backend",
				UnitID:       "unit:engineering",
				Hours:        8,
				Description:  "Test entry",
				Date:         "15-01-2026",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/time-entries", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Add auth context
			userID := uuid.New()
			orgID := uuid.New()
			ctx := middleware.SetUserID(req.Context(), userID)
			ctx = middleware.SetOrganizationID(ctx, orgID)
			ctx = middleware.SetRole(ctx, "employee")
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			handler.Create(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestSurrealTimeEntryHandler_Submit(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	sdb, err := db.NewSurrealDB()
	if err != nil {
		t.Fatalf("Failed to connect to SurrealDB: %v", err)
	}
	defer sdb.Close()

	handler := NewSurrealTimeEntryHandler(sdb)

	// Note: This test assumes a time entry exists
	// In a real test, we would create one first

	t.Run("unauthorized user cannot submit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/time-entries/time_entries:test/submit", nil)
		rec := httptest.NewRecorder()

		// Different user than owner
		userID := uuid.New()
		orgID := uuid.New()
		ctx := middleware.SetUserID(req.Context(), userID)
		ctx = middleware.SetOrganizationID(ctx, orgID)
		ctx = middleware.SetRole(ctx, "employee")
		req = req.WithContext(ctx)

		handler.Submit(rec, req)

		// Should fail either with not found or forbidden
		if rec.Code != http.StatusNotFound && rec.Code != http.StatusForbidden && rec.Code != http.StatusInternalServerError {
			t.Errorf("expected error status, got %d", rec.Code)
		}
	})
}

func TestSurrealTimeEntryHandler_List(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	sdb, err := db.NewSurrealDB()
	if err != nil {
		t.Fatalf("Failed to connect to SurrealDB: %v", err)
	}
	defer sdb.Close()

	handler := NewSurrealTimeEntryHandler(sdb)

	req := httptest.NewRequest(http.MethodGet, "/time-entries?month=1&year=2026", nil)
	rec := httptest.NewRecorder()

	userID := uuid.New()
	orgID := uuid.New()
	ctx := middleware.SetUserID(req.Context(), userID)
	ctx = middleware.SetOrganizationID(ctx, orgID)
	ctx = middleware.SetRole(ctx, "employee")
	req = req.WithContext(ctx)

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify response is an array
	var entries []interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Errorf("failed to parse response as array: %v", err)
	}
}

func TestSurrealTimeEntryHandler_ApproveReject(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	sdb, err := db.NewSurrealDB()
	if err != nil {
		t.Fatalf("Failed to connect to SurrealDB: %v", err)
	}
	defer sdb.Close()

	handler := NewSurrealTimeEntryHandler(sdb)

	t.Run("employee cannot approve entries", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/time-entries/time_entries:test/approve", nil)
		rec := httptest.NewRecorder()

		userID := uuid.New()
		orgID := uuid.New()
		ctx := middleware.SetUserID(req.Context(), userID)
		ctx = middleware.SetOrganizationID(ctx, orgID)
		ctx = middleware.SetRole(ctx, "employee")
		req = req.WithContext(ctx)

		handler.Approve(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("wg_manager can approve submitted entries", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/time-entries/time_entries:test/approve", nil)
		rec := httptest.NewRecorder()

		userID := uuid.New()
		orgID := uuid.New()
		ctx := middleware.SetUserID(req.Context(), userID)
		ctx = middleware.SetOrganizationID(ctx, orgID)
		ctx = middleware.SetRole(ctx, "wg_manager")
		req = req.WithContext(ctx)

		handler.Approve(rec, req)

		// Will fail with not found or internal error since entry doesn't exist
		// but the permission check should pass
		if rec.Code == http.StatusForbidden {
			t.Errorf("wg_manager should be allowed to approve")
		}
	})

	t.Run("reject with reason", func(t *testing.T) {
		payload := map[string]string{
			"reason": "Insufficient documentation",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/time-entries/time_entries:test/reject", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		userID := uuid.New()
		orgID := uuid.New()
		ctx := middleware.SetUserID(req.Context(), userID)
		ctx = middleware.SetOrganizationID(ctx, orgID)
		ctx = middleware.SetRole(ctx, "wg_manager")
		req = req.WithContext(ctx)

		handler.Reject(rec, req)

		// Will fail with not found since entry doesn't exist
		// but verifies the handler accepts the request format
		if rec.Code == http.StatusBadRequest {
			t.Errorf("request format should be valid")
		}
	})
}

func TestSurrealTimeEntryHandler_ListPending(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	sdb, err := db.NewSurrealDB()
	if err != nil {
		t.Fatalf("Failed to connect to SurrealDB: %v", err)
	}
	defer sdb.Close()

	handler := NewSurrealTimeEntryHandler(sdb)

	t.Run("employee cannot list pending", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/time-entries/pending", nil)
		rec := httptest.NewRecorder()

		userID := uuid.New()
		orgID := uuid.New()
		ctx := middleware.SetUserID(req.Context(), userID)
		ctx = middleware.SetOrganizationID(ctx, orgID)
		ctx = middleware.SetRole(ctx, "employee")
		req = req.WithContext(ctx)

		handler.ListPending(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("wg_manager can list pending", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/time-entries/pending", nil)
		rec := httptest.NewRecorder()

		userID := uuid.New()
		orgID := uuid.New()
		ctx := middleware.SetUserID(req.Context(), userID)
		ctx = middleware.SetOrganizationID(ctx, orgID)
		ctx = middleware.SetRole(ctx, "wg_manager")
		req = req.WithContext(ctx)

		handler.ListPending(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var entries []interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
			t.Errorf("failed to parse response as array: %v", err)
		}
	})
}
