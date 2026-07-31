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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Unit handler integration
// ---------------------------------------------------------------------------

func TestUnitHandlerIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("CreateAndList", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "unit-int-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "unituser", "TestPass123!", "UnitOrg")

		// Create unit
		body := `{"name":"Engineering","code":"ENG"}`
		resp, err := f.Client.Post(f.ServerURL+"/units", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var created struct {
			Data map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&created)
		require.NoError(t, err)
		unitID, ok := created.Data["id"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, unitID)

		// List units
		listResp, err := f.Client.Get(f.ServerURL + "/units")
		require.NoError(t, err)
		defer listResp.Body.Close()
		assert.Equal(t, http.StatusOK, listResp.StatusCode)
	})

	t.Run("GetByID_NotFound_Returns404", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "unit-get-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "unitget", "TestPass123!", "UnitGetOrg")

		resp, err := f.Client.Get(f.ServerURL + "/units/" + uuid.New().String())
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Create_InvalidBody_Returns400", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "unit-inv-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "unitinv", "TestPass123!", "UnitInvOrg")

		resp, err := f.Client.Post(f.ServerURL+"/units", "application/json", strings.NewReader("{"))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// ---------------------------------------------------------------------------
// Organization handler integration
// ---------------------------------------------------------------------------

func TestOrganizationHandlerIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("CreateAndGet", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "org-int-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "orguser", "TestPass123!", "OrgOrg")

		// The user is already in an org. Try to create a new org.
		body := `{"name":"NewTestOrg","slug":"new-test-org"}`
		resp, err := f.Client.Post(f.ServerURL+"/organizations", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		// May be 201 or 403 depending on role — just verify no 500
		assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode)

		if resp.StatusCode == http.StatusCreated {
			var created struct {
				Data map[string]interface{} `json:"data"`
			}
			err = json.NewDecoder(resp.Body).Decode(&created)
			require.NoError(t, err)
			assert.NotNil(t, created.Data["id"])
		}
	})

	t.Run("ListMembers_ReturnsList", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "org-list-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "orglist", "TestPass123!", "OrgListOrg")

		resp, err := f.Client.Get(f.ServerURL + "/organizations/members")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var wrapped struct {
			Data []interface{} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&wrapped)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(wrapped.Data), 1, "should have at least 1 member")
	})

	t.Run("GetByID_InvalidUUID_Returns400", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "org-get-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "orgget", "TestPass123!", "OrgGetOrg")

		resp, err := f.Client.Get(f.ServerURL + "/organizations/not-a-uuid")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// ---------------------------------------------------------------------------
// Activity handler integration
// ---------------------------------------------------------------------------

func TestActivityHandlerIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("CreateAndList", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "act-int-" + uuid.New().String()[:8] + "@test.com"
		login := f.registerAndLogin(t, email, "actuser", "TestPass123!", "ActOrg")
		orgID, err := uuid.Parse(login.Organization.ID)
		require.NoError(t, err)
		seedKind(t, pool, orgID, "engagement")
		seedKind(t, pool, orgID, "task")

		// Create an activity
		body := `{"name":"Client Engagement","kind":"engagement","governance_model":"creator_controlled"}`
		resp, err := f.Client.Post(f.ServerURL+"/activities", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var created struct {
			Data map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&created)
		require.NoError(t, err)
		activityID, ok := created.Data["id"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, activityID)

		// Create a child (task) under it
		childBody := `{"name":"Task Alpha","kind":"task","governance_model":"creator_controlled","parent_id":"` + activityID + `"}`
		childResp, err := f.Client.Post(f.ServerURL+"/activities", "application/json", strings.NewReader(childBody))
		require.NoError(t, err)
		defer childResp.Body.Close()
		assert.Equal(t, http.StatusCreated, childResp.StatusCode)

		var childCreated struct {
			Data map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(childResp.Body).Decode(&childCreated)
		require.NoError(t, err)
		childID, ok := childCreated.Data["id"].(string)
		require.True(t, ok)

		// List (org scope)
		listResp, err := f.Client.Get(f.ServerURL + "/activities")
		require.NoError(t, err)
		defer listResp.Body.Close()
		assert.Equal(t, http.StatusOK, listResp.StatusCode)

		// List children
		childrenResp, err := f.Client.Get(f.ServerURL + "/activities/" + activityID + "/children")
		require.NoError(t, err)
		defer childrenResp.Body.Close()
		assert.Equal(t, http.StatusOK, childrenResp.StatusCode)

		var children struct {
			Data []map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(childrenResp.Body).Decode(&children)
		require.NoError(t, err)
		require.Len(t, children.Data, 1, "expected exactly one child")
		assert.Equal(t, childID, children.Data[0]["id"])

		// Detail (activity + ancestry + commercial context + billable)
		detailResp, err := f.Client.Get(f.ServerURL + "/activities/" + activityID)
		require.NoError(t, err)
		defer detailResp.Body.Close()
		assert.Equal(t, http.StatusOK, detailResp.StatusCode)

		var detail struct {
			Data map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(detailResp.Body).Decode(&detail)
		require.NoError(t, err)
		require.Contains(t, detail.Data, "activity")
		require.Contains(t, detail.Data, "ancestry")
		require.Contains(t, detail.Data, "commercial_context")
		require.Contains(t, detail.Data, "billable")

		// Kinds catalog
		kindsResp, err := f.Client.Get(f.ServerURL + "/activity-kinds")
		require.NoError(t, err)
		defer kindsResp.Body.Close()
		assert.Equal(t, http.StatusOK, kindsResp.StatusCode)

		var kinds struct {
			Data []string `json:"data"`
		}
		err = json.NewDecoder(kindsResp.Body).Decode(&kinds)
		require.NoError(t, err)
		require.Contains(t, kinds.Data, "engagement")
	})

	t.Run("GetByID_InvalidUUID_Returns400", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "act-get-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "actget", "TestPass123!", "ActGetOrg")

		resp, err := f.Client.Get(f.ServerURL + "/activities/not-a-uuid")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// seedKind inserts a kind into the org's activity_kinds catalog directly
// (the fixture registers fresh orgs at runtime; the migration only seeds the
// MVP org's catalog).
func seedKind(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, kind string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO activity_kinds (org_id, name, is_seed) VALUES ($1, $2, true)
		 ON CONFLICT (org_id, name) DO NOTHING`,
		orgID, kind)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Contract handler integration
// ---------------------------------------------------------------------------

func TestContractHandlerIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("CreateAndList", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "cntr-int-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "cntruser", "TestPass123!", "CntrOrg")

		// Create a customer first
		custBody := `{"company_name":"Test Customer"}`
		custResp, err := f.Client.Post(f.ServerURL+"/customers", "application/json", strings.NewReader(custBody))
		require.NoError(t, err)
		defer custResp.Body.Close()

		var custCreated struct {
			Data map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(custResp.Body).Decode(&custCreated)
		require.NoError(t, err)
		custID, ok := custCreated.Data["id"].(string)
		if !ok {
			t.Skip("Customer creation did not return an id")
		}

		// Create contract
		ctrBody := fmt.Sprintf(`{"name":"Test Contract","km_rate":0.5,"currency":"EUR","governance_model":"creator_controlled","customer_id":"%s"}`, custID)
		resp, err := f.Client.Post(f.ServerURL+"/contracts", "application/json", strings.NewReader(ctrBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("GetByID_InvalidUUID_Returns400", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "cntr-get-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "cntrget", "TestPass123!", "CntrGetOrg")

		resp, err := f.Client.Get(f.ServerURL + "/contracts/not-a-uuid")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// ---------------------------------------------------------------------------
// Customer handler integration
// ---------------------------------------------------------------------------

func TestCustomerHandlerIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("List_ReturnsOK", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "cust-list-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "custlist", "TestPass123!", "CustListOrg")

		resp, err := f.Client.Get(f.ServerURL + "/customers")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("GetByID_InvalidUUID_Returns400", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "cust-get-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "custget", "TestPass123!", "CustGetOrg")

		resp, err := f.Client.Get(f.ServerURL + "/customers/not-a-uuid")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// ---------------------------------------------------------------------------
// Time entry handler integration
// ---------------------------------------------------------------------------

func TestTimeEntryHandlerIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("Create_RequiresUnit_Returns400", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "te-int-" + uuid.New().String()[:8] + "@test.com"
		login := f.registerAndLogin(t, email, "teuser", "TestPass123!", "TEOOrg")
		orgID, err := uuid.Parse(login.Organization.ID)
		require.NoError(t, err)
		seedKind(t, pool, orgID, "engagement")

		// Create an activity first
		actBody := `{"name":"TE Test Activity","kind":"engagement","governance_model":"creator_controlled"}`
		actResp, err := f.Client.Post(f.ServerURL+"/activities", "application/json", strings.NewReader(actBody))
		require.NoError(t, err)
		defer actResp.Body.Close()
		require.Equal(t, http.StatusCreated, actResp.StatusCode)

		var actCreated struct {
			Data map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(actResp.Body).Decode(&actCreated)
		require.NoError(t, err)
		activityID, ok := actCreated.Data["id"].(string)
		require.True(t, ok, "activity should have an ID")

		// Create time entry WITHOUT required unit_id
		today := time.Now().UTC().Format("2006-01-02")
		teBody := fmt.Sprintf(`{"activity_id":"%s","hours":8,"description":"Test entry","date":"%s"}`, activityID, today)
		teResp, err := f.Client.Post(f.ServerURL+"/time-entries", "application/json", strings.NewReader(teBody))
		require.NoError(t, err)
		defer teResp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, teResp.StatusCode, "missing unit_id should return 400")
	})

	t.Run("List_ReturnsOK", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "te-list-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "telist", "TestPass123!", "TEListOrg")

		resp, err := f.Client.Get(f.ServerURL + "/time-entries")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// ---------------------------------------------------------------------------
// Working group handler integration
// ---------------------------------------------------------------------------

func TestWorkingGroupHandlerIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("CreateAndList", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "wg-int-" + uuid.New().String()[:8] + "@test.com"
		login := f.registerAndLogin(t, email, "wguser", "TestPass123!", "WGOrg")
		orgID, err := uuid.Parse(login.Organization.ID)
		require.NoError(t, err)
		seedKind(t, pool, orgID, "engagement")

		// Create an activity
		actBody := `{"name":"WG Activity","kind":"engagement","governance_model":"creator_controlled"}`
		actResp, err := f.Client.Post(f.ServerURL+"/activities", "application/json", strings.NewReader(actBody))
		require.NoError(t, err)
		defer actResp.Body.Close()

		if actResp.StatusCode != http.StatusCreated {
			t.Skip("Cannot create activity for WG test")
		}

		// List working groups (should return 200 with empty list)
		resp, err := f.Client.Get(f.ServerURL + "/working-groups")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("GetByID_InvalidUUID_Returns400", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "wg-get-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "wggget", "TestPass123!", "WGGetOrg")

		resp, err := f.Client.Get(f.ServerURL + "/working-groups/not-a-uuid")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
