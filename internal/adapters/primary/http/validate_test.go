package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
)

// Over-limit input must be rejected with 400 at the handler boundary (audit
// S3) — long before any service call or domain validation. Handlers are
// constructed with nil services: reaching the service would panic, so a 400
// here proves the length cap fired first.
func TestInputLengthCaps_RejectOversizedFields(t *testing.T) {
	longString := func(n int) string { return strings.Repeat("x", n) }

	tests := []struct {
		name    string
		body    string
		handler func(w http.ResponseWriter, r *http.Request)
	}{
		{
			name: "customer company_name with 10000 chars",
			body: `{"company_name":"` + longString(10000) + `"}`,
			handler: func(w http.ResponseWriter, r *http.Request) {
				ctx := middleware.SetUserID(r.Context(), uuid.New())
				ctx = middleware.SetOrganizationID(ctx, uuid.New())
				ctx = middleware.SetRole(ctx, "finance")
				NewCustomerHandler(nil).Create(w, r.WithContext(ctx))
			},
		},
		{
			name: "time entry description with 5000 chars",
			body: `{"project_id":"` + uuid.NewString() + `","subproject_id":"` + uuid.NewString() + `","wg_id":"` + uuid.NewString() + `","unit_id":"` + uuid.NewString() + `","hours":8,"description":"` + longString(5000) + `","date":"2026-01-15"}`,
			handler: func(w http.ResponseWriter, r *http.Request) {
				ctx := middleware.SetUserID(r.Context(), uuid.New())
				ctx = middleware.SetOrganizationID(ctx, uuid.New())
				NewTimeEntryHandler(nil).Create(w, r.WithContext(ctx))
			},
		},
		{
			name: "working group name with 10000 chars",
			body: `{"org_id":"` + uuid.NewString() + `","subproject_id":"` + uuid.NewString() + `","name":"` + longString(10000) + `"}`,
			handler: func(w http.ResponseWriter, r *http.Request) {
				ctx := middleware.SetUserID(r.Context(), uuid.New())
				NewWorkingGroupHandler(nil).Create(w, r.WithContext(ctx))
			},
		},
		{
			name:    "register password with 200 chars",
			body:    `{"email":"cap@test.com","password":"` + longString(200) + `","organization_name":"Cap Org"}`,
			handler: NewAuthHandler(nil, nil).Register,
		},
		{
			name:    "contract name with 10001 chars",
			body:    `{"name":"` + longString(10001) + `"}`,
			handler: NewContractHandler(nil).Create,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			tt.handler(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for over-limit input, got %d (body: %s)", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "exceeds maximum length of") {
				t.Fatalf("expected field-level length message, got: %s", rec.Body.String())
			}
		})
	}
}

// Under-limit values must pass through the length gate untouched — later
// validation (not the length cap) decides the outcome. Here a short
// description must reach the required-field check for project_id instead of
// tripping the length gate.
func TestInputLengthCaps_DoNotRejectNormalLength(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/time-entries",
		strings.NewReader(`{"description":"short desc"}`))
	rec := httptest.NewRecorder()
	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	NewTimeEntryHandler(nil).Create(rec, req.WithContext(ctx))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected existing required-field 400, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "project_id is required") {
		t.Fatalf("length gate must not shadow required-field validation, got: %s", rec.Body.String())
	}
}
