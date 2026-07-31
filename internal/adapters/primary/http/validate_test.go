package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateStringLengths_Boundaries is the helper-level boundary table:
// every limit class enforced by validate.go is exercised at N-1, N (both
// accepted — no false positives) and N+1 (rejected with a field-level 400).
func TestValidateStringLengths_Boundaries(t *testing.T) {
	longString := func(n int) string { return strings.Repeat("x", n) }

	limitClasses := []struct {
		name      string
		fieldName string
		max       int
	}{
		{"email", "email", MaxEmailLength},
		{"name", "firstname", MaxNameLength},
		{"description", "description", MaxDescriptionLength},
		{"address", "address", MaxAddressLength},
		{"vat", "vat_number", MaxVATLength},
		{"phone", "phone", MaxPhoneLength},
		{"password", "password", MaxPasswordLength},
		{"short string", "short", MaxShortStringLength},
	}

	for _, tt := range limitClasses {
		t.Run(tt.name+"_under_limit", func(t *testing.T) {
			rec := httptest.NewRecorder()
			ok := validateStringLengths(rec, lengthField(tt.fieldName, longString(tt.max-1), tt.max))
			assert.True(t, ok, "N-1 must be accepted")
			assert.Equal(t, http.StatusOK, rec.Code, "no response may be written for accepted input")
			assert.Empty(t, rec.Body.String())
		})
		t.Run(tt.name+"_at_limit", func(t *testing.T) {
			rec := httptest.NewRecorder()
			ok := validateStringLengths(rec, lengthField(tt.fieldName, longString(tt.max), tt.max))
			assert.True(t, ok, "exactly N must be accepted")
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Empty(t, rec.Body.String())
		})
		t.Run(tt.name+"_over_limit", func(t *testing.T) {
			rec := httptest.NewRecorder()
			ok := validateStringLengths(rec, lengthField(tt.fieldName, longString(tt.max+1), tt.max))
			assert.False(t, ok, "N+1 must be rejected")
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			want := tt.fieldName + " exceeds maximum length of " + strconv.Itoa(tt.max)
			assert.Contains(t, rec.Body.String(), want, "field-level message must name the field and cap")
		})
	}
}

// Caps are rune-count based (user-facing character semantics), so multi-byte
// characters must not trip them: 200 é's (400 bytes) pass a 200-rune name cap,
// 201 é's fail it.
func TestValidateStringLengths_RuneCount(t *testing.T) {
	accents := func(n int) string { return strings.Repeat("é", n) }

	rec := httptest.NewRecorder()
	ok := validateStringLengths(rec, lengthField("firstname", accents(MaxNameLength), MaxNameLength))
	assert.True(t, ok, "200 multi-byte runes must pass a 200-rune cap (byte length 400)")
	assert.Equal(t, http.StatusOK, rec.Code)

	rec2 := httptest.NewRecorder()
	ok = validateStringLengths(rec2, lengthField("firstname", accents(MaxNameLength+1), MaxNameLength))
	assert.False(t, ok, "201 multi-byte runes must be rejected")
	assert.Equal(t, http.StatusBadRequest, rec2.Code)
	assert.Contains(t, rec2.Body.String(), "firstname exceeds maximum length of "+strconv.Itoa(MaxNameLength))
}

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
		{
			name: "expense description with 5000 chars",
			body: `{"project_id":"` + uuid.NewString() + `","category":"mileage","amount":10,"description":"` + longString(5000) + `","date":"2026-01-15"}`,
			handler: func(w http.ResponseWriter, r *http.Request) {
				ctx := middleware.SetUserID(r.Context(), uuid.New())
				ctx = middleware.SetOrganizationID(ctx, uuid.New())
				NewExpenseHandler(nil).Create(w, r.WithContext(ctx))
			},
		},
		{
			name:    "register firstname with 10000 chars",
			body:    `{"email":"capname@test.com","firstname":"` + longString(10000) + `","lastname":"Doe","password":"password123","organization_name":"Cap Org"}`,
			handler: NewAuthHandler(nil, nil).Register,
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
// description must reach the required-field check for activity_id instead of
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
	if !strings.Contains(rec.Body.String(), "activity_id is required") {
		t.Fatalf("length gate must not shadow required-field validation, got: %s", rec.Body.String())
	}
}

// Same no-false-positive guarantee for the expense endpoint: a normal-length
// description passes the length gate and the handler falls through to its own
// required-field validation.
func TestInputLengthCaps_DoNotRejectNormalLength_Expense(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/expenses",
		strings.NewReader(`{"description":"short desc"}`))
	rec := httptest.NewRecorder()
	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	NewExpenseHandler(nil).Create(rec, req.WithContext(ctx))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected existing required-field 400, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "activity_id is required") {
		t.Fatalf("length gate must not shadow required-field validation, got: %s", rec.Body.String())
	}
}

// End-to-end no-false-positive: a real registration whose name sits exactly at
// the boundary cap (200 runes) succeeds — the length gate must not reject
// input at the limit. Backed by real PostgreSQL via the handler fixture.
func TestRegister_BoundaryLengthName_Succeeds(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	f := newHandlerFixture(t, pool)
	body := fmt.Sprintf(`{"email":"%s","username":"bnd_%s","firstname":"%s","lastname":"Doe","password":"password123","organization_name":"%s"}`,
		uniqueEmail(), uniqueID(), strings.Repeat("x", MaxNameLength), uniqueOrgName())

	resp, err := f.Client.Post(f.ServerURL+"/auth/register", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"a name at exactly the cap must be accepted, not rejected by the length gate")
}
