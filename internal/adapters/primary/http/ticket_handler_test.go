package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	"github.com/stretchr/testify/require"
)

// ticketAPIHelper wraps the fixture with ticket-scoped HTTP helpers.
type ticketAPIHelper struct {
	f *handlerFixture
}

// doJSON performs a request and decodes the {data,...}/{error,...} envelope.
func (h *ticketAPIHelper) doJSON(t *testing.T, method, path, body string) (int, map[string]any) {
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
	decErr := json.NewDecoder(resp.Body).Decode(&envelope)
	require.NoError(t, decErr)
	return resp.StatusCode, envelope
}

// registerUserInOrg registers a brand-new user and switches them into orgID
// with the given role (inserts the membership row directly — the switch
// endpoint issues fresh tokens). Returns the user id.
func (h *ticketAPIHelper) registerUserInOrg(t *testing.T, email, username, password, orgID string, role string) string {
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

// TestTicketAPI is the full API contract test (TICK-01..05): employee
// create → manager lifecycle incl. triage + reopen → permission gates
// (non-owner employee, customer) → guarded dismissal (409 with logged
// hours, 200 with 0) → comments + ordered history → and the TICK-05
// assertion that NO DELETE /tickets route exists in the mux.
func TestTicketAPI(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	f := newHandlerFixture(t, pool)
	h := &ticketAPIHelper{f: f}

	// --- Employee (owner) registers and creates the org ---
	empEmail := "tick-emp-" + uuid.New().String()[:8] + "@test.com"
	empLogin := f.registerAndLogin(t, empEmail, "tickemp", "TestPass123!", "TicketOrg")
	orgID := empLogin.Organization.ID

	// The triage plans need a kind in the org's catalog.
	orgUUID, err := uuid.Parse(orgID)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(),
		`INSERT INTO activity_kinds (org_id, name, is_seed) VALUES ($1, 'engagement', true)
		 ON CONFLICT (org_id, name) DO NOTHING`, orgUUID)
	require.NoError(t, err)

	// --- 1. Employee creates a ticket (201, status open, TICK-01) ---
	status, env := h.doJSON(t, http.MethodPost, "/tickets",
		`{"title":"Billing bug","description":"Off by a cent","kind":"bug"}`)
	require.Equal(t, http.StatusCreated, status, "employee create should be 201: %v", env)
	ticketData, ok := env["data"].(map[string]any)
	require.True(t, ok, "create should return data envelope: %v", env)
	require.Equal(t, "open", ticketData["status"])
	ticketID := ticketData["id"].(string)
	require.NotEmpty(t, ticketID)

	// --- 2. Manager joins the org and runs the lifecycle ---
	mgrEmail := "tick-mgr-" + uuid.New().String()[:8] + "@test.com"
	mgrPassword := "TestPass123!"
	managerID := h.registerUserInOrg(t, mgrEmail, "tickmgr", mgrPassword, orgID, "manager")

	// Restore the manager session after another user has taken the cookie
	// jar: login lands on the manager's own org (first active membership),
	// so switch into the ticket org.
	restoreManager := func() {
		f.loginUser(t, mgrEmail, mgrPassword)
		status, env := h.doJSON(t, http.MethodPost, "/auth/switch-organization",
			fmt.Sprintf(`{"organization_id":"%s"}`, orgID))
		require.Equal(t, http.StatusOK, status, "switch manager back: %v", env)
	}

	// open → triage (transition endpoint)
	status, env = h.doJSON(t, http.MethodPost, "/tickets/"+ticketID+"/transition", `{"status":"triage"}`)
	require.Equal(t, http.StatusOK, status, "open→triage: %v", env)
	require.Equal(t, "triage", env["data"].(map[string]any)["status"])

	// triage → planned via the atomic triage endpoint (creates an activity)
	status, env = h.doJSON(t, http.MethodPost, "/tickets/"+ticketID+"/triage",
		`{"kind":"bug","activities":[{"name":"Investigate billing","kind":"engagement","governance_model":"creator_controlled"}]}`)
	require.Equal(t, http.StatusOK, status, "triage should be 200: %v", env)
	require.Equal(t, "planned", env["data"].(map[string]any)["ticket"].(map[string]any)["status"])

	// planned → in_progress → resolved (activity has no entries → terminal)
	status, env = h.doJSON(t, http.MethodPost, "/tickets/"+ticketID+"/transition", `{"status":"in_progress"}`)
	require.Equal(t, http.StatusOK, status, "planned→in_progress: %v", env)
	status, env = h.doJSON(t, http.MethodPost, "/tickets/"+ticketID+"/transition", `{"status":"resolved","note":"done"}`)
	require.Equal(t, http.StatusOK, status, "in_progress→resolved: %v", env)
	require.Equal(t, "resolved", env["data"].(map[string]any)["status"])

	// reopen resolved → in_progress (D-A)
	status, env = h.doJSON(t, http.MethodPost, "/tickets/"+ticketID+"/transition", `{"status":"in_progress"}`)
	require.Equal(t, http.StatusOK, status, "reopen resolved→in_progress: %v", env)

	// --- 3. Non-owner employee transition → 403 (T-11-05) ---
	h.registerUserInOrg(t, "tick-emp2-"+uuid.New().String()[:8]+"@test.com", "tickemp2", "TestPass123!", orgID, "employee")
	status, env = h.doJSON(t, http.MethodPost, "/tickets/"+ticketID+"/transition", `{"status":"triage"}`)
	require.Equal(t, http.StatusForbidden, status, "non-owner employee transition: %v", env)

	restoreManager()

	// --- 4. Customer role create → 403 (internal-only, D-E) ---
	h.registerUserInOrg(t, "tick-cust-"+uuid.New().String()[:8]+"@test.com", "tickcust", "TestPass123!", orgID, "customer")
	status, env = h.doJSON(t, http.MethodPost, "/tickets", `{"title":"Customer ticket","kind":"bug"}`)
	require.Equal(t, http.StatusForbidden, status, "customer create should be 403: %v", env)

	restoreManager()

	// --- 5. Dismiss with logged hours → 409 (TICK-04, D-13 guard) ---
	// The guard applies while the ticket is still open|triage (dismissal is
	// only legal from those states). An activity with the customer_ticket
	// origin can be linked to an open ticket (OQ5) via the activity API —
	// that is the shape the guard must catch.
	status, env = h.doJSON(t, http.MethodPost, "/tickets", `{"title":"Guarded dismissal","kind":"change"}`)
	require.Equal(t, http.StatusCreated, status)
	ticket2 := env["data"].(map[string]any)["id"].(string)

	status, env = h.doJSON(t, http.MethodPost, "/activities",
		`{"name":"Work on ticket","kind":"engagement","governance_model":"creator_controlled","origin_type":"customer_ticket","ticket_id":"`+ticket2+`"}`)
	require.Equal(t, http.StatusCreated, status, "link activity to open ticket (OQ5): %v", env)
	activityID := env["data"].(map[string]any)["id"].(string)

	// Seed a unit + a submitted time entry on the linked activity.
	var unitID string
	require.NoError(t, pool.QueryRow(context.Background(),
		`INSERT INTO units (id, org_id, name, code, created_at, updated_at)
		 VALUES (gen_random_uuid(), $1, 'Unit', 'U', NOW(), NOW()) RETURNING id`, orgUUID).Scan(&unitID))
	var managerUUID uuid.UUID
	managerUUID, err = uuid.Parse(managerID)
	require.NoError(t, err)
	var activityUUID uuid.UUID
	activityUUID, err = uuid.Parse(activityID)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(),
		`INSERT INTO time_entries (id, org_id, user_id, activity_id, unit_id, hours,
			description, entry_date, status, is_deleted, created_at, updated_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4, 4.0, 'logged', NOW(), 'submitted', false, NOW(), NOW())`,
		orgUUID, managerUUID, activityUUID, unitID)
	require.NoError(t, err)

	status, env = h.doJSON(t, http.MethodPost, "/tickets/"+ticket2+"/dismiss", "")
	require.Equal(t, http.StatusConflict, status, "dismiss with hours should be 409: %v", env)

	// --- 6. Dismiss with 0 hours → 200 + dismissed_hours 0 + history ---
	status, env = h.doJSON(t, http.MethodPost, "/tickets", `{"title":"Empty ticket","kind":"question"}`)
	require.Equal(t, http.StatusCreated, status)
	ticket3 := env["data"].(map[string]any)["id"].(string)

	status, env = h.doJSON(t, http.MethodPost, "/tickets/"+ticket3+"/dismiss", "")
	require.Equal(t, http.StatusOK, status, "dismiss with 0 hours: %v", env)
	dismissed := env["data"].(map[string]any)
	require.Equal(t, "dismissed", dismissed["status"])
	require.Equal(t, float64(0), dismissed["dismissed_hours"])
	// IN-02: the note is server-derived (TICK-04) — the dismiss response
	// carries "dismissed with N h logged" rendered from dismissed_hours.
	require.Equal(t, "dismissed with 0 h logged", dismissed["dismissed_note"],
		"dismiss response must carry the derived note")

	// The note must also render on the detail read (GET /tickets/{id}).
	status, env = h.doJSON(t, http.MethodGet, "/tickets/"+ticket3, "")
	require.Equal(t, http.StatusOK, status)
	detail3 := env["data"].(map[string]any)["ticket"].(map[string]any)
	require.Equal(t, "dismissed with 0 h logged", detail3["dismissed_note"],
		"GET /tickets/{id} must carry the derived note")

	// ... and on the list read (GET /tickets) — every read of a dismissed
	// ticket carries the note (TICK-04, IN-02).
	status, env = h.doJSON(t, http.MethodGet, "/tickets", "")
	require.Equal(t, http.StatusOK, status)
	list := env["data"].([]any)
	require.GreaterOrEqual(t, len(list), 1)
	found := false
	for _, item := range list {
		it, ok := item.(map[string]any)
		if ok && it["id"] == ticket3 {
			require.Equal(t, "dismissed with 0 h logged", it["dismissed_note"],
				"GET /tickets list must carry the derived note for dismissed tickets")
			found = true
		}
	}
	require.True(t, found, "dismissed ticket must appear in the GET /tickets list")

	status, env = h.doJSON(t, http.MethodGet, "/tickets/"+ticket3+"/history", "")
	require.Equal(t, http.StatusOK, status)
	history3 := env["data"].([]any)
	require.GreaterOrEqual(t, len(history3), 2)
	actions3 := []string{history3[0].(map[string]any)["action"].(string), history3[1].(map[string]any)["action"].(string)}
	require.Contains(t, actions3, "created")
	require.Contains(t, actions3, "dismissed")

	// --- 7. Comments POST → 200 + 'comment_added' in history (D-06, TICK-05) ---
	status, env = h.doJSON(t, http.MethodPost, "/tickets/"+ticketID+"/comments", `{"body":"please look at this"}`)
	require.Equal(t, http.StatusOK, status, "comment should be 200: %v", env)

	status, env = h.doJSON(t, http.MethodGet, "/tickets/"+ticketID+"/history", "")
	require.Equal(t, http.StatusOK, status)
	history := env["data"].([]any)
	require.GreaterOrEqual(t, len(history), 4) // created, status_changed x4, triaged, activities_created, comment_added
	var lastAction string
	for _, e := range history {
		lastAction = e.(map[string]any)["action"].(string)
	}
	require.Equal(t, "comment_added", lastAction, "comment should be the latest history row")

	// --- 8. History is ordered (created_at ascending) ---
	for i := 1; i < len(history); i++ {
		prev := history[i-1].(map[string]any)["created_at"].(string)
		cur := history[i].(map[string]any)["created_at"].(string)
		pt, err1 := time.Parse(time.RFC3339Nano, prev)
		ct, err2 := time.Parse(time.RFC3339Nano, cur)
		require.NoError(t, err1)
		require.NoError(t, err2)
		require.False(t, ct.Before(pt), "history must be ordered by created_at")
	}

	// --- 9. GET /tickets/{id} returns ticket + comments (detail) ---
	status, env = h.doJSON(t, http.MethodGet, "/tickets/"+ticketID, "")
	require.Equal(t, http.StatusOK, status)
	detail := env["data"].(map[string]any)
	require.Contains(t, detail, "ticket")
	require.Contains(t, detail, "comments")

	// --- 10. No DELETE /tickets route exists (TICK-05) ---
	// Go 1.22 ServeMux answers 405 (method not allowed) when the path matches
	// but the method is not registered — proving no DELETE route is served.
	req, err := http.NewRequest(http.MethodDelete, f.ServerURL+"/tickets/"+ticketID, nil)
	require.NoError(t, err)
	resp, err := f.Client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "DELETE /tickets/{id} must not be registered (TICK-05)")
}
