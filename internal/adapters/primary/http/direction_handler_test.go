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

// directionAPIHelper wraps the fixture with direction-scoped HTTP helpers
// (the 7 pinned routes, ADR-BE-018 §7).
type directionAPIHelper struct {
	f *handlerFixture
}

// doJSON performs a request and decodes the {data,...}/{error,...} envelope.
func (h *directionAPIHelper) doJSON(t *testing.T, method, path, body string) (int, map[string]any) {
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
// user id. Mirrors the coverage/ticket helper pattern.
func (h *directionAPIHelper) registerUserInOrg(t *testing.T, email, username, password, orgID string, role string) string {
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

// switchToOrg posts the switch-organization request (a fresh loginUser issues
// a token for the user's PRIMARY org — the switched org is re-established
// here).
func (h *directionAPIHelper) switchToOrg(t *testing.T, orgID string) {
	t.Helper()
	switchBody := fmt.Sprintf(`{"organization_id":"%s"}`, orgID)
	resp, err := h.f.Client.Post(h.f.ServerURL+"/auth/switch-organization", "application/json", strings.NewReader(switchBody))
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "switch-organization should return 200")
}

// seedWorkingGroup inserts a working group anchored to the activity (the
// WG-scope predicate D-13-17: activity == the WG's anchored activity).
func (h *directionAPIHelper) seedWorkingGroup(t *testing.T, orgID, activityID, managerID uuid.UUID) uuid.UUID {
	t.Helper()
	var wgID uuid.UUID
	err := h.f.Pool.QueryRow(context.Background(),
		`INSERT INTO working_groups (id, org_id, activity_id, name, description, unit_ids, manager_id, delegate_ids, is_active, created_at, updated_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, true, NOW(), NOW()) RETURNING id`,
		orgID, activityID, "Direction WG", "Test WG", []any{}, managerID, []any{}).Scan(&wgID)
	require.NoError(t, err)
	return wgID
}

// seedWorkingGroupMember adds a member to a working group (wg_members —
// unit_id required by the schema).
func (h *directionAPIHelper) seedWorkingGroupMember(t *testing.T, orgID, wgID, userID uuid.UUID) {
	t.Helper()
	var unitID uuid.UUID
	err := h.f.Pool.QueryRow(context.Background(),
		`INSERT INTO units (id, org_id, name, code, created_at, updated_at)
		 VALUES (gen_random_uuid(), $1, 'DirUnit', $2, NOW(), NOW()) RETURNING id`,
		orgID, "DU"+uuid.New().String()[:6]).Scan(&unitID)
	require.NoError(t, err)
	_, err = h.f.Pool.Exec(context.Background(),
		`INSERT INTO wg_members (id, wg_id, user_id, unit_id) VALUES (gen_random_uuid(), $1, $2, $3)`,
		wgID, userID, unitID)
	require.NoError(t, err)
}

// seedDirectionRow inserts a direction row directly (the fallback contract
// seed — the row's directed_by/directed_to feed FirstDirectionRefs).
func (h *directionAPIHelper) seedDirectionRow(t *testing.T, orgID, directedBy uuid.UUID, directedTo *uuid.UUID, activityID uuid.UUID, estHours float64) {
	t.Helper()
	_, err := h.f.Pool.Exec(context.Background(),
		`INSERT INTO direction (id, org_id, directed_by, directed_to, wg_id, activity_id, planned_date, est_hours, status, created_at, updated_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, NULL, $4, NULL, $5, 'draft', NOW(), NOW())`,
		orgID, directedBy, directedTo, activityID, estHours)
	require.NoError(t, err)
}

// TestDirectionHandler is the full API contract test for the plan plane
// (DIR-01..06, ADR-BE-018 §7): the permission matrix end-to-end (self-
// direction 200, cross-employee manager_planned 403, non-member claim 403,
// over-budget claim 409, double-activation 409, unauthenticated 401), the
// sentinel map at the boundary (404/400/409/500), the warnings array riding
// the create response (D-13-03), the supersede chain (D-13-08) incl. claim-
// row superseding (origin_direction_id carried), the read gates on the
// org-wide plan view and coverage scopes (T-13-26/T-13-31), and the origin
// fallback e2e at the HTTP boundary (FND-04, Pitfall 5 — the GET /activities
// read path derives assigned_by/assigned_to from the first direction row).
func TestDirectionHandler(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	f := newHandlerFixture(t, pool)
	h := &directionAPIHelper{f: f}

	// --- Setup: the org owner (manager) + a plan activity ---
	ownerEmail := "dir-owner-" + uuid.New().String()[:8] + "@test.com"
	owner := f.registerAndLogin(t, ownerEmail, "diro", "TestPass123!", "DirectionOrg")
	orgID := owner.Organization.ID
	orgUUID, err := uuid.Parse(orgID)
	require.NoError(t, err)

	var ownerUUID uuid.UUID
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, ownerEmail).Scan(&ownerUUID))

	_, err = pool.Exec(context.Background(),
		`INSERT INTO activity_kinds (org_id, name, is_seed) VALUES ($1, 'engagement', true)
		 ON CONFLICT (org_id, name) DO NOTHING`, orgUUID)
	require.NoError(t, err)

	status, env := h.doJSON(t, http.MethodPost, "/activities",
		`{"name":"Plan work","kind":"engagement","governance_model":"creator_controlled"}`)
	require.Equal(t, http.StatusCreated, status, "activity create: %v", env)
	activityID := env["data"].(map[string]any)["id"].(string)

	period := "period_start=2026-08-10&period_end=2026-08-16"
	periodDay := "period_start=2026-08-15&period_end=2026-08-15"

	// ---------------------------------------------------------------------
	// (a) Permission matrix
	// ---------------------------------------------------------------------

	t.Run("self-direction create returns 200 with row + warnings array", func(t *testing.T) {
		status, env := h.doJSON(t, http.MethodPost, "/direction",
			fmt.Sprintf(`{"directed_to":"%s","activity_id":"%s","planned_date":"2026-08-10T00:00:00Z","est_hours":4}`,
				ownerUUID.String(), activityID))
		require.Equal(t, http.StatusOK, status, "self-direction create: %v", env)
		data := env["data"].(map[string]any)
		row := data["row"].(map[string]any)
		require.NotEmpty(t, row["id"], "created row id")
		require.NotNil(t, data["warnings"], "create response must carry the warnings array (D-13-03)")
		require.IsType(t, []any{}, data["warnings"])
	})

	t.Run("employee creating for a colleague in manager_planned org is forbidden", func(t *testing.T) {
		h.registerUserInOrg(t, "dir-emp1-"+uuid.New().String()[:8]+"@test.com", "diremp1", "TestPass123!", orgID, "employee")
		emp2 := h.registerUserInOrg(t, "dir-emp2-"+uuid.New().String()[:8]+"@test.com", "diremp2", "TestPass123!", orgID, "employee")

		status, env := h.doJSON(t, http.MethodPost, "/direction",
			fmt.Sprintf(`{"directed_to":"%s","activity_id":"%s","planned_date":"2026-08-11T00:00:00Z","est_hours":4}`,
				emp2, activityID))
		require.Equal(t, http.StatusForbidden, status, "cross-employee create in manager_planned mode: %v", env)
	})

	t.Run("directed_to a user outside the org is 400 (WR-01)", func(t *testing.T) {
		// A user whose ONLY membership is in a different org: the fixture org
		// has no membership row for them → GetMembership returns (nil, nil) →
		// the service rejects before any mode/routing decision (T-13g-04).
		foreignEmail := "dir-xorg-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, foreignEmail, "dirxorg", "TestPass123!", "OtherOrg_"+uuid.New().String()[:8])
		var foreignID string
		require.NoError(t, h.f.Pool.QueryRow(context.Background(),
			`SELECT id FROM users WHERE email = $1`, foreignEmail).Scan(&foreignID))

		f.loginUser(t, ownerEmail, "TestPass123!")

		status, env := h.doJSON(t, http.MethodPost, "/direction",
			fmt.Sprintf(`{"directed_to":"%s","activity_id":"%s","planned_date":"2026-08-11T00:00:00Z","est_hours":4}`,
				foreignID, activityID))
		require.Equal(t, http.StatusBadRequest, status, "cross-org directed_to must be 400 (WR-01): %v", env)
	})

	t.Run("wg claim by non-member is forbidden", func(t *testing.T) {
		wgID := h.seedWorkingGroup(t, orgUUID, mustParse(t, activityID), ownerUUID)
		h.seedWorkingGroupMember(t, orgUUID, wgID, ownerUUID)
		// The jar is on an employee from the previous subtest — back to owner.
		f.loginUser(t, ownerEmail, "TestPass123!")

		status, env := h.doJSON(t, http.MethodPost, "/direction",
			fmt.Sprintf(`{"wg_id":"%s","activity_id":"%s","est_hours":8}`, wgID.String(), activityID))
		require.Equal(t, http.StatusOK, status, "wg row create: %v", env)
		wgRowID := env["data"].(map[string]any)["row"].(map[string]any)["id"].(string)

		status, env = h.doJSON(t, http.MethodPost, "/direction/"+wgRowID+"/activate", "")
		require.Equal(t, http.StatusOK, status, "activate wg row: %v", env)

		h.registerUserInOrg(t, "dir-nm-"+uuid.New().String()[:8]+"@test.com", "dirnm", "TestPass123!", orgID, "employee")
		status, env = h.doJSON(t, http.MethodPost, "/direction/claims",
			fmt.Sprintf(`{"wg_row_id":"%s","est_hours":4}`, wgRowID))
		require.Equal(t, http.StatusForbidden, status, "non-member claim: %v", env)
	})

	t.Run("claim over budget is 409", func(t *testing.T) {
		wgID := h.seedWorkingGroup(t, orgUUID, mustParse(t, activityID), ownerUUID)
		memberEmail := "dir-mem-" + uuid.New().String()[:8] + "@test.com"
		member := h.registerUserInOrg(t, memberEmail, "dirmem", "TestPass123!", orgID, "employee")
		memberUUID := mustParse(t, member)
		h.seedWorkingGroupMember(t, orgUUID, wgID, memberUUID)
		f.loginUser(t, ownerEmail, "TestPass123!") // WG-row create is the manager's

		status, env := h.doJSON(t, http.MethodPost, "/direction",
			fmt.Sprintf(`{"wg_id":"%s","activity_id":"%s","est_hours":8}`, wgID.String(), activityID))
		require.Equal(t, http.StatusOK, status, "wg row create: %v", env)
		wgRowID := env["data"].(map[string]any)["row"].(map[string]any)["id"].(string)
		status, env = h.doJSON(t, http.MethodPost, "/direction/"+wgRowID+"/activate", "")
		require.Equal(t, http.StatusOK, status, "activate wg row: %v", env)

		f.loginUser(t, memberEmail, "TestPass123!") // the claim is the member's
		h.switchToOrg(t, orgID)

		status, env = h.doJSON(t, http.MethodPost, "/direction/claims",
			fmt.Sprintf(`{"wg_row_id":"%s","est_hours":8}`, wgRowID))
		require.Equal(t, http.StatusOK, status, "full-budget claim: %v", env)

		status, env = h.doJSON(t, http.MethodPost, "/direction/claims",
			fmt.Sprintf(`{"wg_row_id":"%s","est_hours":4}`, wgRowID))
		require.Equal(t, http.StatusConflict, status, "over-budget claim must be 409 (D-13-13): %v", env)
	})

	t.Run("activate a cancelled row is 409", func(t *testing.T) {
		f.loginUser(t, ownerEmail, "TestPass123!")
		status, env := h.doJSON(t, http.MethodPost, "/direction",
			fmt.Sprintf(`{"directed_to":"%s","activity_id":"%s","est_hours":4}`, ownerUUID.String(), activityID))
		require.Equal(t, http.StatusOK, status, "create: %v", env)
		rowID := env["data"].(map[string]any)["row"].(map[string]any)["id"].(string)

		status, env = h.doJSON(t, http.MethodPost, "/direction/"+rowID+"/cancel", `{"reason":"no longer needed"}`)
		require.Equal(t, http.StatusOK, status, "cancel: %v", env)

		status, env = h.doJSON(t, http.MethodPost, "/direction/"+rowID+"/activate", "")
		require.Equal(t, http.StatusConflict, status, "activate cancelled row must be 409: %v", env)
	})

	t.Run("unauthenticated requests are rejected", func(t *testing.T) {
		// A fresh client without the fixture cookie jar.
		resp, err := http.Get(f.ServerURL + "/direction?" + period)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// ---------------------------------------------------------------------
	// (b) Sentinel mapping at the boundary
	// ---------------------------------------------------------------------

	t.Run("sentinel mapping: 404 unknown id / 400 reason-less cancel / 400 malformed body", func(t *testing.T) {
		f.loginUser(t, ownerEmail, "TestPass123!")
		status, env := h.doJSON(t, http.MethodPost, "/direction/"+uuid.New().String()+"/activate", "")
		require.Equal(t, http.StatusNotFound, status, "unknown activate target: %v", env)

		status, env = h.doJSON(t, http.MethodPost, "/direction",
			fmt.Sprintf(`{"directed_to":"%s","activity_id":"%s","est_hours":4}`, ownerUUID.String(), activityID))
		require.Equal(t, http.StatusOK, status, "create: %v", env)
		rowID := env["data"].(map[string]any)["row"].(map[string]any)["id"].(string)

		status, env = h.doJSON(t, http.MethodPost, "/direction/"+rowID+"/cancel", `{}`)
		require.Equal(t, http.StatusBadRequest, status, "reason-less cancel must be 400 (D-13-10): %v", env)

		status, env = h.doJSON(t, http.MethodPost, "/direction", `{"not json`)
		require.Equal(t, http.StatusBadRequest, status, "malformed body must be 400: %v", env)
	})

	t.Run("est_hours above the DECIMAL(8,2) ceiling is 400, never 500 (WR-02)", func(t *testing.T) {
		f.loginUser(t, ownerEmail, "TestPass123!")
		status, env := h.doJSON(t, http.MethodPost, "/direction",
			fmt.Sprintf(`{"directed_to":"%s","activity_id":"%s","planned_date":"2026-08-13T00:00:00Z","est_hours":1000000}`,
				ownerUUID.String(), activityID))
		require.Equal(t, http.StatusBadRequest, status, "absurd est_hours must map to 400 (WR-02): %v", env)
	})

	t.Run("claims on a user-targeted row are 404 — no panic, connection stays up (CR-01)", func(t *testing.T) {
		f.loginUser(t, ownerEmail, "TestPass123!")
		status, env := h.doJSON(t, http.MethodPost, "/direction",
			fmt.Sprintf(`{"directed_to":"%s","activity_id":"%s","planned_date":"2026-08-12T00:00:00Z","est_hours":4}`,
				ownerUUID.String(), activityID))
		require.Equal(t, http.StatusOK, status, "self-direction create: %v", env)
		rowID := mustParse(t, env["data"].(map[string]any)["row"].(map[string]any)["id"].(string))

		status, env = h.doJSON(t, http.MethodPost, "/direction/"+rowID.String()+"/activate", "")
		require.Equal(t, http.StatusOK, status, "activate self-direction row: %v", env)

		// The guard sits after the status fast-fail, so the 404 contract
		// needs an ACTIVE user-targeted row (the deref path).
		status, env = h.doJSON(t, http.MethodPost, "/direction/claims",
			fmt.Sprintf(`{"wg_row_id":"%s","est_hours":4}`, rowID.String()))
		require.Equal(t, http.StatusNotFound, status, "claim on a user-targeted row must be 404 (CR-01): %v", env)
	})

	// ---------------------------------------------------------------------
	// (c) Supersede chain (D-13-08) — incl. claim-row superseding
	// ---------------------------------------------------------------------

	t.Run("create-with-supersedes flips the target to superseded (history via GET /direction)", func(t *testing.T) {
		f.loginUser(t, ownerEmail, "TestPass123!")
		status, env := h.doJSON(t, http.MethodPost, "/direction",
			fmt.Sprintf(`{"directed_to":"%s","activity_id":"%s","est_hours":4}`, ownerUUID.String(), activityID))
		require.Equal(t, http.StatusOK, status, "row1 create: %v", env)
		row1ID := env["data"].(map[string]any)["row"].(map[string]any)["id"].(string)

		status, env = h.doJSON(t, http.MethodPost, "/direction",
			fmt.Sprintf(`{"directed_to":"%s","activity_id":"%s","est_hours":6,"supersedes_id":"%s"}`,
				ownerUUID.String(), activityID, row1ID))
		require.Equal(t, http.StatusOK, status, "row2 create-with-supersedes: %v", env)
		row2ID := env["data"].(map[string]any)["row"].(map[string]any)["id"].(string)

		status, env = h.doJSON(t, http.MethodGet, "/direction?"+period, "")
		require.Equal(t, http.StatusOK, status, "plan read: %v", env)
		rows := env["data"].(map[string]any)["rows"].([]any)
		ids := map[string]bool{}
		for _, r := range rows {
			ids[r.(map[string]any)["id"].(string)] = true
		}
		require.True(t, ids[row2ID], "the superseding row is in the plan")
		require.False(t, ids[row1ID], "the superseded row is history, not in the plan (D-13-08)")
	})

	t.Run("supersede-on-create writes BOTH audit rows in the tx (CR-02)", func(t *testing.T) {
		f.loginUser(t, ownerEmail, "TestPass123!")
		status, env := h.doJSON(t, http.MethodPost, "/direction",
			fmt.Sprintf(`{"directed_to":"%s","activity_id":"%s","est_hours":4}`, ownerUUID.String(), activityID))
		require.Equal(t, http.StatusOK, status, "row1 create: %v", env)
		row1ID := mustParse(t, env["data"].(map[string]any)["row"].(map[string]any)["id"].(string))

		status, env = h.doJSON(t, http.MethodPost, "/direction",
			fmt.Sprintf(`{"directed_to":"%s","activity_id":"%s","est_hours":6,"supersedes_id":"%s"}`,
				ownerUUID.String(), activityID, row1ID.String()))
		require.Equal(t, http.StatusOK, status, "row2 create-with-supersedes: %v", env)
		row2ID := mustParse(t, env["data"].(map[string]any)["row"].(map[string]any)["id"].(string))

		var supersededCount int
		require.NoError(t, h.f.Pool.QueryRow(context.Background(),
			`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'direction' AND entity_id = $1 AND action = 'superseded'`,
			row1ID).Scan(&supersededCount))
		require.Equal(t, 1, supersededCount, "the superseded audit must address the flipped target (CR-02)")

		var createdCount int
		require.NoError(t, h.f.Pool.QueryRow(context.Background(),
			`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'direction' AND entity_id = $1 AND action = 'created'`,
			row2ID).Scan(&createdCount))
		require.Equal(t, 1, createdCount, "the created audit addresses the new row")
	})

	t.Run("superseding a claim-row target carries origin_direction_id", func(t *testing.T) {
		wgID := h.seedWorkingGroup(t, orgUUID, mustParse(t, activityID), ownerUUID)
		memberEmail := "dir-cs-" + uuid.New().String()[:8] + "@test.com"
		member := h.registerUserInOrg(t, memberEmail, "dircs", "TestPass123!", orgID, "employee")
		memberUUID := mustParse(t, member)
		h.seedWorkingGroupMember(t, orgUUID, wgID, memberUUID)
		f.loginUser(t, ownerEmail, "TestPass123!")

		status, env := h.doJSON(t, http.MethodPost, "/direction",
			fmt.Sprintf(`{"wg_id":"%s","activity_id":"%s","est_hours":8}`, wgID.String(), activityID))
		require.Equal(t, http.StatusOK, status, "wg row create: %v", env)
		wgRowID := env["data"].(map[string]any)["row"].(map[string]any)["id"].(string)
		status, env = h.doJSON(t, http.MethodPost, "/direction/"+wgRowID+"/activate", "")
		require.Equal(t, http.StatusOK, status, "activate wg row: %v", env)

		f.loginUser(t, memberEmail, "TestPass123!")
		h.switchToOrg(t, orgID)
		status, env = h.doJSON(t, http.MethodPost, "/direction/claims",
			fmt.Sprintf(`{"wg_row_id":"%s","est_hours":4}`, wgRowID))
		require.Equal(t, http.StatusOK, status, "claim: %v", env)
		claimRowID := env["data"].(map[string]any)["id"].(string) // the claim returns the row directly

		// The manager supersedes the claim row with a new user-targeted row:
		// the repo tx inherits origin_direction_id (ADR-BE-018 §5).
		f.loginUser(t, ownerEmail, "TestPass123!")
		status, env = h.doJSON(t, http.MethodPost, "/direction",
			fmt.Sprintf(`{"directed_to":"%s","activity_id":"%s","est_hours":4,"supersedes_id":"%s"}`,
				member, activityID, claimRowID))
		require.Equal(t, http.StatusOK, status, "supersede claim row: %v", env)
		row := env["data"].(map[string]any)["row"].(map[string]any)
		require.Equal(t, wgRowID, row["origin_direction_id"], "claim lineage carries origin_direction_id (ADR-BE-018 §5)")
	})

	// ---------------------------------------------------------------------
	// (d) Period bounds + (e) read gates at the HTTP boundary
	// ---------------------------------------------------------------------

	t.Run("GET /direction requires period bounds", func(t *testing.T) {
		status, env := h.doJSON(t, http.MethodGet, "/direction", "")
		require.Equal(t, http.StatusBadRequest, status, "missing period bounds: %v", env)
		status, env = h.doJSON(t, http.MethodGet, "/direction?period_start=2026-08-10", "")
		require.Equal(t, http.StatusBadRequest, status, "partial period bounds: %v", env)
	})

	t.Run("read gates: org-wide plan manager-only, employee_id=self allowed", func(t *testing.T) {
		emp := h.registerUserInOrg(t, "dir-rg-"+uuid.New().String()[:8]+"@test.com", "dirrg", "TestPass123!", orgID, "employee")

		status, env := h.doJSON(t, http.MethodGet, "/direction?"+period, "")
		require.Equal(t, http.StatusForbidden, status, "non-manager org-wide view: %v", env)

		status, env = h.doJSON(t, http.MethodGet, "/direction?employee_id="+ownerUUID.String()+"&"+period, "")
		require.Equal(t, http.StatusForbidden, status, "non-manager other-employee view: %v", env)

		status, env = h.doJSON(t, http.MethodGet, "/direction?employee_id="+emp+"&"+period, "")
		require.Equal(t, http.StatusOK, status, "non-manager self-view: %v", env)
		_, hasWarnings := env["data"].(map[string]any)["warnings"]
		require.True(t, hasWarnings, "warnings ride the plan read response (D-13-28)")
	})

	t.Run("read gates: coverage employee scope self-only for non-managers", func(t *testing.T) {
		emp := h.registerUserInOrg(t, "dir-cg-"+uuid.New().String()[:8]+"@test.com", "dircg", "TestPass123!", orgID, "employee")

		status, env := h.doJSON(t, http.MethodGet, "/direction/coverage?scope=employee&scope_id="+ownerUUID.String()+"&"+periodDay, "")
		require.Equal(t, http.StatusForbidden, status, "non-manager coverage other scope_id: %v", env)

		status, env = h.doJSON(t, http.MethodGet, "/direction/coverage?scope=employee&scope_id="+emp+"&"+periodDay, "")
		require.Equal(t, http.StatusOK, status, "non-manager coverage self-view: %v", env)
	})

	// ---------------------------------------------------------------------
	// (f) Coverage read-model
	// ---------------------------------------------------------------------

	t.Run("coverage scope=employee returns rows + totals + warnings", func(t *testing.T) {
		f.loginUser(t, ownerEmail, "TestPass123!")
		status, env := h.doJSON(t, http.MethodGet,
			"/direction/coverage?scope=employee&scope_id="+ownerUUID.String()+"&"+periodDay, "")
		require.Equal(t, http.StatusOK, status, "coverage read: %v", env)
		data := env["data"].(map[string]any)
		require.IsType(t, []any{}, data["rows"])
		require.NotEmpty(t, data["rows"], "an uncovered day is surfaced (gap == capacity)")
		require.IsType(t, []any{}, data["totals"])
		require.IsType(t, []any{}, data["warnings"])
	})

	t.Run("coverage scope=unknown is 400", func(t *testing.T) {
		f.loginUser(t, ownerEmail, "TestPass123!")
		status, env := h.doJSON(t, http.MethodGet,
			"/direction/coverage?scope=bogus&scope_id="+ownerUUID.String()+"&"+periodDay, "")
		require.Equal(t, http.StatusBadRequest, status, "unknown scope: %v", env)
	})

	// ---------------------------------------------------------------------
	// (g) Origin fallback at the HTTP boundary (FND-04 e2e, Pitfall 5)
	// ---------------------------------------------------------------------

	t.Run("GET /activities derives assigned_by/assigned_to from the first direction row", func(t *testing.T) {
		emp := h.registerUserInOrg(t, "dir-fb-"+uuid.New().String()[:8]+"@test.com", "dirfb", "TestPass123!", orgID, "employee")
		// The jar is on the employee (registerUserInOrg switched orgs) —
		// the manager_assignment origin create requires manager|finance.
		f.loginUser(t, ownerEmail, "TestPass123!")

		// (1) Empty-origin activity + seeded direction row → derived refs.
		status, env := h.doJSON(t, http.MethodPost, "/activities",
			`{"name":"Fallback derive","kind":"engagement","governance_model":"creator_controlled"}`)
		require.Equal(t, http.StatusCreated, status, "activity create: %v", env)
		deriveActivityID := env["data"].(map[string]any)["id"].(string)
		h.seedDirectionRow(t, orgUUID, ownerUUID, ptrUUID(mustParse(t, emp)), mustParse(t, deriveActivityID), 4)

		// (2) Empty-origin activity WITHOUT direction rows → refs stay empty.
		status, env = h.doJSON(t, http.MethodPost, "/activities",
			`{"name":"Fallback empty","kind":"engagement","governance_model":"creator_controlled"}`)
		require.Equal(t, http.StatusCreated, status, "activity create: %v", env)
		emptyActivityID := env["data"].(map[string]any)["id"].(string)

		// (3) Stored origin refs stay authoritative — even with a direction row.
		status, env = h.doJSON(t, http.MethodPost, "/activities",
			fmt.Sprintf(`{"name":"Fallback stored","kind":"engagement","governance_model":"creator_controlled",
				"origin_type":"manager_assignment","assigned_by":"%s","assigned_to":"%s"}`,
				ownerUUID.String(), emp))
		require.Equal(t, http.StatusCreated, status, "origin activity create: %v", env)
		storedActivityID := env["data"].(map[string]any)["id"].(string)
		h.seedDirectionRow(t, orgUUID, mustParse(t, emp), ptrUUID(ownerUUID), mustParse(t, storedActivityID), 4)

		status, env = h.doJSON(t, http.MethodGet, "/activities", "")
		require.Equal(t, http.StatusOK, status, "activities read: %v", env)
		activities := env["data"].([]any)
		byID := map[string]map[string]any{}
		for _, a := range activities {
			am := a.(map[string]any)
			byID[am["id"].(string)] = am
		}

		derived := byID[deriveActivityID]
		require.NotNil(t, derived["assigned_by"], "empty-origin activity derives assigned_by")
		require.Equal(t, ownerUUID.String(), derived["assigned_by"])
		require.Equal(t, emp, derived["assigned_to"])

		empty := byID[emptyActivityID]
		require.Nil(t, empty["assigned_by"], "no direction rows → refs stay empty (D-13-34)")
		require.Nil(t, empty["assigned_to"])

		stored := byID[storedActivityID]
		require.Equal(t, ownerUUID.String(), stored["assigned_by"], "stored refs are authoritative (Pitfall 5)")
		require.Equal(t, emp, stored["assigned_to"])
	})
}

// mustParse parses a uuid string for test setup.
func mustParse(t *testing.T, s string) uuid.UUID {
	t.Helper()
	u, err := uuid.Parse(s)
	require.NoError(t, err)
	return u
}

// ptrUUID returns a pointer to the uuid.
func ptrUUID(u uuid.UUID) *uuid.UUID { return &u }
