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

// newOrgSettingsFixture spins up a fresh schema + fixture for one battery
// test (each test owns its schema lifecycle).
func newOrgSettingsFixture(t *testing.T) (*orgSettingsAPIHelper, uuid.UUID) {
	t.Helper()
	pool := postgres.SetupPackageContainer(t)
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	f := newHandlerFixture(t, pool)
	h := &orgSettingsAPIHelper{f: f}
	mgrLogin := f.registerAndLogin(t, "os-mgr-"+uuid.New().String()[:8]+"@test.com", "osmgr", "TestPass123!", "OrgSettingsOrg")
	orgID, err := uuid.Parse(mgrLogin.Organization.ID)
	require.NoError(t, err)
	return h, orgID
}

// TestOrgSettingsHandler_RouteCoexistence is the Pitfall 6 regression lock
// (D-13-23): the literal GET/PUT /organizations/settings routes (no id) hit
// the NEW org_settings handler while GET /organizations/{uuid}/settings still
// resolves to the TYPED organization_settings surface — ServeMux
// most-specific-wins, both registrations kept.
func TestOrgSettingsHandler_RouteCoexistence(t *testing.T) {
	h, orgID := newOrgSettingsFixture(t)

	// Literal route → the new handler: org_settings map shape with the
	// code-level default for the absent key (D-13-24).
	status, env := h.doJSON(t, http.MethodGet, "/organizations/settings", "")
	require.Equal(t, http.StatusOK, status, "literal GET should return 200: %v", env)
	data, ok := env["data"].(map[string]any)
	require.True(t, ok, "literal GET must carry a data object: %v", env)
	require.Equal(t, 8.0, data["planning_daily_hours"], "new handler applies the planning_daily_hours default")

	// Typed wildcard route → the legacy typed surface: organization_settings
	// row (auto-created by the org trigger), Go field-name serialization.
	status, env = h.doJSON(t, http.MethodGet, "/organizations/"+orgID.String()+"/settings", "")
	require.Equal(t, http.StatusOK, status, "typed GET should return 200: %v", env)
	data, ok = env["data"].(map[string]any)
	require.True(t, ok, "typed GET must carry a data object: %v", env)
	require.Equal(t, "EUR", data["Currency"], "typed surface serializes organization_settings fields")
	require.Equal(t, float64(1), data["WeekStartDay"])
	_, hasSnake := data["planning_daily_hours"]
	require.False(t, hasSnake, "typed surface must NOT carry org_settings keys (different handler)")

	// The literal PUT still lands in org_settings, not the typed table.
	status, env = h.doJSON(t, http.MethodPut, "/organizations/settings", `{"planning_daily_hours": 7.5}`)
	require.Equal(t, http.StatusOK, status, "literal PUT should return 200: %v", env)
	data, ok = env["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, 7.5, data["planning_daily_hours"])

	status, env = h.doJSON(t, http.MethodGet, "/organizations/"+orgID.String()+"/settings", "")
	require.Equal(t, http.StatusOK, status, "typed GET still resolves after the literal PUT")
	data, ok = env["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "EUR", data["Currency"], "typed row untouched by the literal route")
	_, hasSnake = data["planning_daily_hours"]
	require.False(t, hasSnake)
}

// TestOrgSettingsHandler_RequiresAuth — GET/PUT are JWT-gated (401 without
// a session cookie; T-13-10 trust boundary).
func TestOrgSettingsHandler_RequiresAuth(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	f := newHandlerFixture(t, pool)
	// A client with no cookie jar — no auth context at all.
	anonResp, err := f.Client.Get(f.ServerURL + "/organizations/settings")
	require.NoError(t, err)
	anonResp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, anonResp.StatusCode, "anonymous GET must be 401")

	req, err := http.NewRequest(http.MethodPut, f.ServerURL+"/organizations/settings", strings.NewReader(`{"planning_daily_hours": 8.0}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	anonPut, err := f.Client.Do(req)
	require.NoError(t, err)
	anonPut.Body.Close()
	require.Equal(t, http.StatusUnauthorized, anonPut.StatusCode, "anonymous PUT must be 401")
}

// TestOrgSettingsHandler_PutOneOrManyKeys — PUT accepts a single key or a
// multi-key batch, both validated and stored (D-13-18).
func TestOrgSettingsHandler_PutOneOrManyKeys(t *testing.T) {
	h, _ := newOrgSettingsFixture(t)

	// Single key.
	status, env := h.doJSON(t, http.MethodPut, "/organizations/settings", `{"planning_deadline": "2026-12-31"}`)
	require.Equal(t, http.StatusOK, status, "single-key PUT should return 200: %v", env)

	// Many keys in one body.
	status, env = h.doJSON(t, http.MethodPut, "/organizations/settings", `{"planning_horizon": "week", "planning_mode": "self_planned"}`)
	require.Equal(t, http.StatusOK, status, "multi-key PUT should return 200: %v", env)
	data, ok := env["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "week", data["planning_horizon"])
	require.Equal(t, "self_planned", data["planning_mode"])

	// GET reflects the accumulated post-state.
	status, env = h.doJSON(t, http.MethodGet, "/organizations/settings", "")
	require.Equal(t, http.StatusOK, status)
	data, ok = env["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "2026-12-31", data["planning_deadline"])
	require.Equal(t, "week", data["planning_horizon"])
	require.Equal(t, "self_planned", data["planning_mode"])
}

// TestOrgSettingsHandler_ValidationMatrix — the D-13-18/T-13-11 value gates:
// zero/negative daily hours and out-of-vocabulary horizon → 400, nothing
// written.
func TestOrgSettingsHandler_ValidationMatrix(t *testing.T) {
	h, _ := newOrgSettingsFixture(t)

	for _, body := range []string{
		`{"planning_daily_hours": 0}`,
		`{"planning_daily_hours": -1}`,
		`{"planning_horizon": "year"}`,
		`{"planning_mode": "autonomous"}`,
	} {
		status, _ := h.doJSON(t, http.MethodPut, "/organizations/settings", body)
		require.Equal(t, http.StatusBadRequest, status, "invalid value %s must be 400", body)
	}

	// Nothing was written by the rejected batch.
	status, env := h.doJSON(t, http.MethodGet, "/organizations/settings", "")
	require.Equal(t, http.StatusOK, status)
	data, ok := env["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, 8.0, data["planning_daily_hours"], "default remains — rejected values never land")
	_, hasHorizon := data["planning_horizon"]
	require.False(t, hasHorizon, "rejected key must not be stored")
}

// TestOrgSettingsHandler_PermissionMatrix — manager+ only (T-13-10): the
// finance role is 403 on PUT, matching the employee gate.
func TestOrgSettingsHandler_PermissionMatrix(t *testing.T) {
	h, orgID := newOrgSettingsFixture(t)

	finID := h.registerUserInOrg(t, "os-fin-"+uuid.New().String()[:8]+"@test.com", "osfin", "TestPass123!", orgID.String(), "finance")
	require.NotEmpty(t, finID)
	status, _ := h.doJSON(t, http.MethodPut, "/organizations/settings", `{"planning_daily_hours": 8.0}`)
	require.Equal(t, http.StatusForbidden, status, "finance PUT should return 403 (manager+ only)")

	// The finance attempt wrote nothing.
	status, env := h.doJSON(t, http.MethodGet, "/organizations/settings", "")
	require.Equal(t, http.StatusOK, status)
	data, ok := env["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, 8.0, data["planning_daily_hours"], "finance rejection must not change the store")
}

// TestOrgSettingsHandler_MalformedBody — a body that is not a JSON object of
// key/value pairs → 400 before any service call.
func TestOrgSettingsHandler_MalformedBody(t *testing.T) {
	h, _ := newOrgSettingsFixture(t)

	for _, body := range []string{`{not json`, `[]`, `""`, `{}`} {
		status, _ := h.doJSON(t, http.MethodPut, "/organizations/settings", body)
		require.Equal(t, http.StatusBadRequest, status, "malformed body %q must be 400", body)
	}
}
