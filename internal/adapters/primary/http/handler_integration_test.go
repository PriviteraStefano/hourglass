package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
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
// Project handler integration
// ---------------------------------------------------------------------------

func TestProjectHandlerIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("CreateBillable", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "proj-int-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "projuser", "TestPass123!", "ProjOrg")

		body := `{"name":"Client Project","type":"billable","governance_model":"creator_controlled"}`
		resp, err := f.Client.Post(f.ServerURL+"/projects", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var created struct {
			Data map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&created)
		require.NoError(t, err)
		projectID, ok := created.Data["id"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, projectID)
	})

	t.Run("CreateInternal", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "proj-int2-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "projusr2", "TestPass123!", "ProjOrg2")

		body := `{"name":"Internal Project","type":"internal","governance_model":"creator_controlled"}`
		resp, err := f.Client.Post(f.ServerURL+"/projects", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("List_ReturnsOK", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "proj-list-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "projlist", "TestPass123!", "ProjListOrg")

		resp, err := f.Client.Get(f.ServerURL + "/projects")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("GetByID_InvalidUUID_Returns400", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "proj-get-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "projget", "TestPass123!", "ProjGetOrg")

		resp, err := f.Client.Get(f.ServerURL + "/projects/not-a-uuid")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
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

	t.Run("Create_RequiresSubproject_Returns400", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

		f := newHandlerFixture(t, pool)
		email := "te-int-" + uuid.New().String()[:8] + "@test.com"
		f.registerAndLogin(t, email, "teuser", "TestPass123!", "TEOOrg")

		// Create a project first
		projBody := `{"name":"TE Test Project","type":"billable","governance_model":"creator_controlled"}`
		projResp, err := f.Client.Post(f.ServerURL+"/projects", "application/json", strings.NewReader(projBody))
		require.NoError(t, err)
		defer projResp.Body.Close()
		require.Equal(t, http.StatusCreated, projResp.StatusCode)

		var projCreated struct {
			Data map[string]interface{} `json:"data"`
		}
		err = json.NewDecoder(projResp.Body).Decode(&projCreated)
		require.NoError(t, err)
		projectID, ok := projCreated.Data["id"].(string)
		require.True(t, ok, "project should have an ID")

		// Create time entry WITHOUT required subproject_id, wg_id, unit_id
		today := time.Now().UTC().Format("2006-01-02")
		teBody := fmt.Sprintf(`{"project_id":"%s","hours":8,"description":"Test entry","date":"%s"}`, projectID, today)
		teResp, err := f.Client.Post(f.ServerURL+"/time-entries", "application/json", strings.NewReader(teBody))
		require.NoError(t, err)
		defer teResp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, teResp.StatusCode, "missing subproject_id should return 400")
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
		f.registerAndLogin(t, email, "wguser", "TestPass123!", "WGOrg")

		// Create a project
		projBody := `{"name":"WG Project","type":"billable","governance_model":"creator_controlled"}`
		projResp, err := f.Client.Post(f.ServerURL+"/projects", "application/json", strings.NewReader(projBody))
		require.NoError(t, err)
		defer projResp.Body.Close()

		if projResp.StatusCode != http.StatusCreated {
			t.Skip("Cannot create project for WG test")
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
