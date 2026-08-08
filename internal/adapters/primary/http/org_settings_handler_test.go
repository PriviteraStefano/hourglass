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

// orgSettingsAPIHelper wraps the fixture with org_settings-scoped HTTP
// helpers (literal /organizations/settings routes, D-13-23).
type orgSettingsAPIHelper struct {
	f *handlerFixture
}

// doJSON performs a request and decodes the {data,...}/{error,...} envelope.
func (h *orgSettingsAPIHelper) doJSON(t *testing.T, method, path, body string) (int, map[string]any) {
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
// user id. Mirrors the coverageAPIHelper pattern.
func (h *orgSettingsAPIHelper) registerUserInOrg(t *testing.T, email, username, password, orgID string, role string) string {
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

// TestOrgSettingsHandler is the phase-tracer e2e: one vertical slice through
// HTTP → service validation → tx upsert + in-tx audit → read-back
// (D-13-18..23, ADR-BE-018). It proves repo-tx-audit + known-key validation +
// manager+ gate + literal routes + cmd/server wiring together. Task 3
// extends this file with the coexistence + validation/permission battery.
func TestOrgSettingsHandler(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	f := newHandlerFixture(t, pool)
	h := &orgSettingsAPIHelper{f: f}

	// registerAndLogin with an org name makes the creator a manager (auth
	// service: role = "manager" when a new org is created).
	mgrLogin := f.registerAndLogin(t, "os-mgr-"+uuid.New().String()[:8]+"@test.com", "osmgr", "TestPass123!", "OrgSettingsOrg")
	orgID := mgrLogin.Organization.ID
	orgUUID, err := uuid.Parse(orgID)
	require.NoError(t, err)

	// --- PUT a known key as manager → 200 with the post-state map ---
	status, env := h.doJSON(t, http.MethodPut, "/organizations/settings", `{"planning_daily_hours": 7.5}`)
	require.Equal(t, http.StatusOK, status, "manager PUT should return 200: %v", env)
	data, ok := env["data"].(map[string]any)
	require.True(t, ok, "PUT response must carry a data object: %v", env)
	require.Equal(t, 7.5, data["planning_daily_hours"])

	// --- GET reflects the stored value ---
	status, env = h.doJSON(t, http.MethodGet, "/organizations/settings", "")
	require.Equal(t, http.StatusOK, status, "GET should return 200: %v", env)
	data, ok = env["data"].(map[string]any)
	require.True(t, ok, "GET response must carry a data object: %v", env)
	require.Equal(t, 7.5, data["planning_daily_hours"])

	// --- audit row written in the same tx as the upsert (D-13-22) ---
	var payload []byte
	err = pool.QueryRow(context.Background(),
		`SELECT payload FROM audit_logs
		 WHERE org_id = $1 AND entity_type = 'org_settings' AND entity_id = $1 AND action = 'settings-updated'
		 ORDER BY created_at DESC LIMIT 1`, orgUUID).Scan(&payload)
	require.NoError(t, err, "settings-updated audit row must exist")
	var auditPayload map[string]any
	require.NoError(t, json.Unmarshal(payload, &auditPayload))
	require.Equal(t, "planning_daily_hours", auditPayload["key"])
	require.Nil(t, auditPayload["before"], "first write has no before value")
	require.Equal(t, 7.5, auditPayload["after"])

	// --- unknown key → 400 (D-13-18 code-enforced vocabulary) ---
	status, _ = h.doJSON(t, http.MethodPut, "/organizations/settings", `{"not_a_known_key": 1}`)
	require.Equal(t, http.StatusBadRequest, status, "unknown key should return 400")

	// --- PUT as employee → 403 (manager+ gate, T-13-10) ---
	empID := h.registerUserInOrg(t, "os-emp-"+uuid.New().String()[:8]+"@test.com", "osemp", "TestPass123!", orgID, "employee")
	require.NotEmpty(t, empID)
	status, _ = h.doJSON(t, http.MethodPut, "/organizations/settings", `{"planning_daily_hours": 8.0}`)
	require.Equal(t, http.StatusForbidden, status, "employee PUT should return 403")
}
