package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	"github.com/stretchr/testify/require"
)

// coverageAPIHelper wraps the fixture with coverage-scoped HTTP helpers.
type coverageAPIHelper struct {
	f *handlerFixture
}

// doJSON performs a request and decodes the {data,...}/{error,...} envelope.
func (h *coverageAPIHelper) doJSON(t *testing.T, method, path, body string) (int, map[string]any) {
	t.Helper()
	var req *http.Request
	var err error
	if body == "" {
		req, err = http.NewRequest(method, h.f.ServerURL+path, nil)
	} else {
		req, err = http.NewRequest(method, h.f.ServerURL+path, strings.NewReader(body))
	}
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.f.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var envelope map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	return resp.StatusCode, envelope
}

// registerUserInOrg registers a brand-new user and switches them into orgID
// with the given role (inserts the membership row directly). Returns the
// user id. Mirrors ticketAPIHelper.
func (h *coverageAPIHelper) registerUserInOrg(t *testing.T, email, username, password, orgID string, role string) string {
	t.Helper()
	h.f.registerAndLogin(t, email, username, password, "OwnOrg_"+uuid.New().String()[:8])

	var userID string
	err := h.f.Pool.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, email).Scan(&userID)
	require.NoError(t, err)

	_, err = h.f.Pool.Exec(context.Background(),
		`INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, created_at, updated_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, true, NOW(), NOW())`, userID, orgID, role)
	require.NoError(t, err)

	switchBody := fmt.Sprintf(`{"organization_id":"%s"}`, orgID)
	resp, err := h.f.Client.Post(h.f.ServerURL+"/auth/switch-organization", "application/json", strings.NewReader(switchBody))
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "switch-organization should return 200")
	return userID
}

// TestCoverageHandler is the full API contract test for the coverage plane
// (COV-01..05, D-07/D-08/D-12): the replace-set write with the D-08
// permission matrix (approver manager 200; owner/employee/finance/customer
// 403), the sentinel map at the boundary (Σ mismatch 400, overlapping close
// 409, nonexistent entry 404, close date parse 400), the manager|finance
// read gates, and the frozen close returning its rows (OQ4).
func TestCoverageHandler(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	f := newHandlerFixture(t, pool)
	h := &coverageAPIHelper{f: f}

	// --- Setup: owner employee creates the org + a personal activity ---
	empEmail := "cov-emp-" + uuid.New().String()[:8] + "@test.com"
	empPassword := "TestPass123!"
	empLogin := f.registerAndLogin(t, empEmail, "covemp", empPassword, "CoverageOrg")
	orgID := empLogin.Organization.ID

	var empUUID uuid.UUID
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, empEmail).Scan(&empUUID))
	orgUUID, err := uuid.Parse(orgID)
	require.NoError(t, err)

	// The triage plans/activities need a kind in the org's catalog.
	_, err = pool.Exec(context.Background(),
		`INSERT INTO activity_kinds (org_id, name, is_seed) VALUES ($1, 'engagement', true)
		 ON CONFLICT (org_id, name) DO NOTHING`, orgUUID)
	require.NoError(t, err)

	// Personal activity (no contract → R-2 fallback routing for D-08).
	status, env := h.doJSON(t, http.MethodPost, "/activities",
		`{"name":"Coverage work","kind":"engagement","governance_model":"creator_controlled"}`)
	require.Equal(t, http.StatusCreated, status, "activity create: %v", env)
	activityID := env["data"].(map[string]any)["id"].(string)
	activityUUID, err := uuid.Parse(activityID)
	require.NoError(t, err)

	// A unit + the manager's unit membership — the D-08 approver resolution
	// path (ResolveUnitManager upward walk finds the manager member).
	var unitID string
	require.NoError(t, pool.QueryRow(context.Background(),
		`INSERT INTO units (id, org_id, name, code, created_at, updated_at)
		 VALUES (gen_random_uuid(), $1, 'CoverageUnit', 'CU', NOW(), NOW()) RETURNING id`, orgUUID).Scan(&unitID))

	mgrEmail := "cov-mgr-" + uuid.New().String()[:8] + "@test.com"
	mgrPassword := "TestPass123!"
	managerID := h.registerUserInOrg(t, mgrEmail, "covmgr", mgrPassword, orgID, "manager")
	managerUUID, err := uuid.Parse(managerID)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(),
		`INSERT INTO unit_memberships (id, org_id, user_id, unit_id, is_primary, role, start_date, created_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, true, 'manager', NOW(), NOW())`,
		orgUUID, managerUUID, unitID)
	require.NoError(t, err)

	// A support contract (the D-04 budget-draw funding target).
	var contractID string
	require.NoError(t, pool.QueryRow(context.Background(),
		`INSERT INTO contracts (id, name, km_rate, currency, governance_model, created_by_org_id,
			is_shared, is_active, contract_type, sold_hours, sold_period, created_at, updated_at)
		 VALUES (gen_random_uuid(), 'Support bucket', 0, 'EUR', 'creator_controlled', $1,
			false, true, 'support', 100, 'monthly', NOW(), NOW()) RETURNING id`, orgUUID).Scan(&contractID))

	// Two approved entries: entry1 (8h, will get a full replace-set) and
	// entry2 (4h, no allocations — the to-cover queue row).
	insertEntry := func(hours float64, date string) string {
		var id string
		require.NoError(t, pool.QueryRow(context.Background(),
			`INSERT INTO time_entries (id, org_id, user_id, activity_id, unit_id, hours,
				description, entry_date, status, is_deleted, created_at, updated_at)
			 VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, 'covered work', $6, 'approved', false, NOW(), NOW())
			 RETURNING id`, orgUUID, empUUID, activityUUID, unitID, hours, date).Scan(&id))
		return id
	}
	entry1 := insertEntry(8.0, "2026-01-15")
	insertEntry(4.0, "2026-01-20")

	restoreManager := func() {
		f.loginUser(t, mgrEmail, mgrPassword)
		status, env := h.doJSON(t, http.MethodPost, "/auth/switch-organization",
			fmt.Sprintf(`{"organization_id":"%s"}`, orgID))
		require.Equal(t, http.StatusOK, status, "switch manager back: %v", env)
	}

	putBody := fmt.Sprintf(
		`{"allocations":[{"entry_type":"time","source_type":"contract","contract_id":"%s","hours":8}]}`, contractID)

	// --- 1. Entry owner PUT → 403 (structural self-barrier, A9) ---
	// Restore the owner session (the manager's registration above took the
	// cookie jar) — the owner is the entry's user_id.
	f.loginUser(t, empEmail, empPassword)
	status, env = h.doJSON(t, http.MethodPut, "/time-entries/"+entry1+"/allocations", putBody)
	require.Equal(t, http.StatusForbidden, status, "owner PUT should be 403: %v", env)
	_, hasErr := env["error"]
	require.True(t, hasErr, "403 must carry the error envelope: %v", env)

	// --- 2. Manager (approver resolution) PUT → 200 + stored set (D-07) ---
	restoreManager()
	status, env = h.doJSON(t, http.MethodPut, "/time-entries/"+entry1+"/allocations", putBody)
	require.Equal(t, http.StatusOK, status, "manager PUT should be 200: %v", env)
	stored, ok := env["data"].([]any)
	require.True(t, ok, "PUT must return the stored set in data: %v", env)
	require.Len(t, stored, 1)
	row := stored[0].(map[string]any)
	require.Equal(t, "contract", row["source_type"])
	require.Equal(t, float64(8), row["hours"])

	// --- 3. Σ mismatch → 400 (COV-01 sentinel) ---
	status, env = h.doJSON(t, http.MethodPut, "/time-entries/"+entry1+"/allocations",
		fmt.Sprintf(`{"allocations":[{"entry_type":"time","source_type":"contract","contract_id":"%s","hours":7}]}`, contractID))
	require.Equal(t, http.StatusBadRequest, status, "Σ mismatch should be 400: %v", env)

	// --- 4. Finance PUT → 403 (read-only, D-L) ---
	h.registerUserInOrg(t, "cov-fin-"+uuid.New().String()[:8]+"@test.com", "covfin", "TestPass123!", orgID, "finance")
	status, env = h.doJSON(t, http.MethodPut, "/time-entries/"+entry1+"/allocations", putBody)
	require.Equal(t, http.StatusForbidden, status, "finance PUT should be 403: %v", env)

	// --- 5. Non-owner employee PUT → 403 (not in the approver set) ---
	h.registerUserInOrg(t, "cov-emp2-"+uuid.New().String()[:8]+"@test.com", "covemp2", "TestPass123!", orgID, "employee")
	status, env = h.doJSON(t, http.MethodPut, "/time-entries/"+entry1+"/allocations", putBody)
	require.Equal(t, http.StatusForbidden, status, "employee PUT should be 403: %v", env)

	// --- 6. Customer PUT → 403 (T-12-18) ---
	h.registerUserInOrg(t, "cov-cust-"+uuid.New().String()[:8]+"@test.com", "covcust", "TestPass123!", orgID, "customer")
	status, env = h.doJSON(t, http.MethodPut, "/time-entries/"+entry1+"/allocations", putBody)
	require.Equal(t, http.StatusForbidden, status, "customer PUT should be 403: %v", env)

	// --- 7. Finance GET /coverage/to-cover → 200 with the uncovered row (D-06) ---
	h.registerUserInOrg(t, "cov-fin2-"+uuid.New().String()[:8]+"@test.com", "covfin2", "TestPass123!", orgID, "finance")
	status, env = h.doJSON(t, http.MethodGet, "/coverage/to-cover", "")
	require.Equal(t, http.StatusOK, status, "finance to-cover should be 200: %v", env)
	queue := env["data"].([]any)
	require.GreaterOrEqual(t, len(queue), 1, "entry2 (4h, unallocated) must be in the queue")
	foundUncovered := false
	for _, q := range queue {
		qr := q.(map[string]any)
		if qr["uncovered_hours"] == float64(4) {
			foundUncovered = true
			proposal, ok := qr["proposal"].(map[string]any)
			require.True(t, ok, "queue rows carry the D-04 proposal: %v", qr)
			require.True(t, proposal["flagged"] == true, "no-source activity must be flagged: %v", proposal)
		}
	}
	require.True(t, foundUncovered, "entry2 must appear with uncovered_hours 4")

	// --- 8. Employee / customer GET to-cover → 403 (read gate) ---
	h.registerUserInOrg(t, "cov-emp3-"+uuid.New().String()[:8]+"@test.com", "covemp3", "TestPass123!", orgID, "employee")
	status, env = h.doJSON(t, http.MethodGet, "/coverage/to-cover", "")
	require.Equal(t, http.StatusForbidden, status, "employee to-cover should be 403: %v", env)
	h.registerUserInOrg(t, "cov-cust2-"+uuid.New().String()[:8]+"@test.com", "covcust2", "TestPass123!", orgID, "customer")
	status, env = h.doJSON(t, http.MethodGet, "/coverage/to-cover", "")
	require.Equal(t, http.StatusForbidden, status, "customer to-cover should be 403: %v", env)

	// --- 9. Proposal read: 200 for an existing entry, 404 for a ghost ---
	restoreManager()
	status, env = h.doJSON(t, http.MethodGet, "/coverage/proposals/"+entry1, "")
	require.Equal(t, http.StatusOK, status, "proposal for existing entry: %v", env)
	proposalData, ok := env["data"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, proposalData, "proposal")
	require.Contains(t, proposalData, "allocations")
	ghost := uuid.NewString()
	status, env = h.doJSON(t, http.MethodGet, "/coverage/proposals/"+ghost, "")
	require.Equal(t, http.StatusNotFound, status, "proposal for nonexistent entry should be 404: %v", env)

	// --- 10. Read-back + bucket balance reads ---
	status, env = h.doJSON(t, http.MethodGet, "/time-entries/"+entry1+"/allocations", "")
	require.Equal(t, http.StatusOK, status, "allocations read-back: %v", env)
	require.Len(t, env["data"].([]any), 1)
	status, env = h.doJSON(t, http.MethodGet, "/coverage/buckets/"+contractID+"/balance", "")
	require.Equal(t, http.StatusOK, status, "bucket balance: %v", env)
	require.Equal(t, float64(92), env["data"].(map[string]any)["balance"], "100 sold − 8 drawn")

	// --- 11. Close: 201 with rows, then 409 on overlap, 400 on bad dates ---
	closeBody := `{"period_start":"2026-01-01","period_end":"2026-01-31"}`
	status, env = h.doJSON(t, http.MethodPost, "/coverage/close", closeBody)
	require.Equal(t, http.StatusCreated, status, "close should be 201: %v", env)
	closeData, ok := env["data"].(map[string]any)
	require.True(t, ok, "close must return the frozen PeriodClose in data: %v", env)
	closeID := closeData["id"].(string)
	require.NotEmpty(t, closeID)
	rowsArr, ok := closeData["rows"].([]any)
	require.True(t, ok, "close must return rows in one call (OQ4): %v", closeData)
	require.GreaterOrEqual(t, len(rowsArr), 1, "the allocated entry must be frozen in the snapshot")

	// Overlapping close (contained period) → 409 (A6).
	status, env = h.doJSON(t, http.MethodPost, "/coverage/close",
		`{"period_start":"2026-01-15","period_end":"2026-01-31"}`)
	require.Equal(t, http.StatusConflict, status, "overlapping close should be 409: %v", env)

	// Date parse errors → 400 (T-12-19).
	status, env = h.doJSON(t, http.MethodPost, "/coverage/close",
		`{"period_start":"not-a-date","period_end":"2026-02-28"}`)
	require.Equal(t, http.StatusBadRequest, status, "bad period_start should be 400: %v", env)
	status, env = h.doJSON(t, http.MethodPost, "/coverage/close",
		`{"period_start":"2026-02-01","period_end":"nope"}`)
	require.Equal(t, http.StatusBadRequest, status, "bad period_end should be 400: %v", env)

	// --- 12. Snapshot + history reads ---
	status, env = h.doJSON(t, http.MethodGet, "/coverage/snapshots/"+closeID, "")
	require.Equal(t, http.StatusOK, status, "snapshot read: %v", env)
	require.Equal(t, closeID, env["data"].(map[string]any)["id"])

	status, env = h.doJSON(t, http.MethodGet, "/coverage/allocations/"+entry1+"/history", "")
	require.Equal(t, http.StatusOK, status, "history read: %v", env)
	history := env["data"].([]any)
	require.GreaterOrEqual(t, len(history), 1)
	require.Equal(t, "allocations-set", history[0].(map[string]any)["action"])

	// --- 13. No DELETE /allocations route exists (D-07 prohibition) ---
	req, err := http.NewRequest(http.MethodDelete, f.ServerURL+"/time-entries/"+entry1+"/allocations", nil)
	require.NoError(t, err)
	resp, err := f.Client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode,
		"DELETE /time-entries/{id}/allocations must not be registered (D-07)")
}
