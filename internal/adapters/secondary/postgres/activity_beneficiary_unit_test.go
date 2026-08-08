package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TestActivityBeneficiaryUnit* — beneficiary unit persistence + CTE resolvers
// (COV-05, T-12-06/T-12-07). Self-seeded inline (never seed_demo.sql): the
// migration 018 column is exercised end-to-end through the repository —
// create/read/update round-trip, downward inheritance via ResolveBeneficiaryUnit,
// and the ResolveFundingContext contract attrs JOIN (D-04 input).
// ---------------------------------------------------------------------------

// seedTypedContract inserts a contracts row matching the 000/016 column
// shape. project contracts carry sold_period NULL per 016's sold_check.
func seedTypedContract(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, now time.Time, contractType string, soldHours *float64, soldPeriod *string) uuid.UUID {
	t.Helper()
	contractID := uuid.New()
	customerID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO customers (id, org_id, name, contact_name, email, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, true, $6, $6)`,
		customerID, orgID, "Beneficiary Customer", "Contact", "beneficiary@test.com", now)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(),
		`INSERT INTO contracts (id, name, km_rate, currency, customer_id, governance_model,
			created_by_org_id, is_shared, is_active, created_at, updated_at, contract_type, sold_hours, sold_period)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, false, true, $8, $8, $9, $10, $11)`,
		contractID, "Beneficiary Contract", 0, "EUR", customerID, "creator_controlled", orgID, now,
		contractType, soldHours, soldPeriod)
	require.NoError(t, err)
	return contractID
}

// TestActivityBeneficiaryUnit_CreateGetReadBack covers the write path the
// plan's Pitfall-5 guard protects: beneficiary_unit_id set via repo Create
// must persist and round-trip through Get (the column lands in the INSERT
// AND the baseActivityQuery SELECT in the same change).
func TestActivityBeneficiaryUnit_CreateGetReadBack(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	seedActivityKind(t, pool, orgID, "engagement")

	created, err := repo.Create(context.Background(), orgID, &activitydomain.CreateActivityRequest{
		Name:              "Beneficiary Work",
		Kind:              "engagement",
		GovernanceModel:   "creator_controlled",
		BeneficiaryUnitID: &unitID,
	})
	require.NoError(t, err)
	require.NotNil(t, created.BeneficiaryUnitID, "Create must return the beneficiary unit")
	require.Equal(t, unitID, *created.BeneficiaryUnitID)

	got, err := repo.Get(context.Background(), orgID, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got.BeneficiaryUnitID, "Get must read the beneficiary unit back")
	require.Equal(t, unitID, *got.BeneficiaryUnitID)

	// List carries the column too (baseActivityQuery).
	list, err := repo.List(context.Background(), orgID, &activitydomain.ActivityFilter{})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NotNil(t, list[0].BeneficiaryUnitID)
	require.Equal(t, unitID, *list[0].BeneficiaryUnitID)
}

// TestActivityBeneficiaryUnit_UpdateEditable covers the COV-05 editability:
// the Update SET builder gains a beneficiary_unit_id branch (unlike origin
// refs), so a later unit assignment replaces the stored value.
func TestActivityBeneficiaryUnit_UpdateEditable(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	otherUnitID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO units (id, org_id, name, code, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $5)`,
		otherUnitID, orgID, "Other Seed Unit", "SEED2", now)
	require.NoError(t, err)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	// Assign via Update (editable — COV-05).
	updated, err := repo.Update(context.Background(), orgID, activityID, &activitydomain.UpdateActivityRequest{
		BeneficiaryUnitID: &unitID,
	})
	require.NoError(t, err)
	require.NotNil(t, updated.BeneficiaryUnitID)
	require.Equal(t, unitID, *updated.BeneficiaryUnitID)

	// Re-assign to a different unit — the field must be replaceable.
	updated2, err := repo.Update(context.Background(), orgID, activityID, &activitydomain.UpdateActivityRequest{
		BeneficiaryUnitID: &otherUnitID,
	})
	require.NoError(t, err)
	require.Equal(t, otherUnitID, *updated2.BeneficiaryUnitID)

	// Name-only update leaves the beneficiary unit untouched.
	updated3, err := repo.Update(context.Background(), orgID, activityID, &activitydomain.UpdateActivityRequest{
		Name: "Renamed",
	})
	require.NoError(t, err)
	require.NotNil(t, updated3.BeneficiaryUnitID)
	require.Equal(t, otherUnitID, *updated3.BeneficiaryUnitID)
}

// TestResolveBeneficiaryUnit_Inheritance covers the downward-inheritance CTE
// (COV-05): a child without its own unit resolves the PARENT's unit — the
// Pitfall-5 regression guard (the column must be in GetAncestry's three
// SELECTs + scanActivity, or this returns nil forever).
func TestResolveBeneficiaryUnit_Inheritance(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)

	parentID := seedActivity(t, pool, orgID, "engagement", nil, now)
	_, err := pool.Exec(context.Background(),
		`UPDATE activities SET beneficiary_unit_id = $1 WHERE id = $2`, unitID, parentID)
	require.NoError(t, err)
	childID := seedActivity(t, pool, orgID, "phase", &parentID, now)
	grandchildID := seedActivity(t, pool, orgID, "task", &childID, now)

	res, err := repo.ResolveBeneficiaryUnit(context.Background(), grandchildID)
	require.NoError(t, err)
	require.NotNil(t, res, "grandchild must inherit the parent's beneficiary unit")
	require.Equal(t, unitID, *res)

	// The unit-bearing activity itself resolves directly.
	res2, err := repo.ResolveBeneficiaryUnit(context.Background(), parentID)
	require.NoError(t, err)
	require.NotNil(t, res2)
	require.Equal(t, unitID, *res2)
}

// TestResolveBeneficiaryUnit_NoUnitNil covers the nil-when-none path: an
// activity tree with no beneficiary unit anywhere resolves to nil — the
// absorption proposal then has no default unit (D-06 no-source flag).
func TestResolveBeneficiaryUnit_NoUnitNil(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	rootID := seedActivity(t, pool, orgID, "engagement", nil, now)
	childID := seedActivity(t, pool, orgID, "task", &rootID, now)

	res, err := repo.ResolveBeneficiaryUnit(context.Background(), childID)
	require.NoError(t, err)
	require.Nil(t, res, "no beneficiary unit anywhere in the chain → nil")
}

// TestResolveFundingContext_ContractAttrs covers the D-04 input: the resolver
// returns contract_id + contract_type + sold_hours for an activity under a
// seeded contract (LEFT JOIN contracts), inherited through the chain.
func TestResolveFundingContext_ContractAttrs(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	soldHours := 120.0
	soldPeriod := "monthly"
	contractID := seedTypedContract(t, pool, orgID, now, "support", &soldHours, &soldPeriod)
	parentID := seedActivity(t, pool, orgID, "engagement", nil, now)
	_, err := pool.Exec(context.Background(),
		`UPDATE activities SET contract_id = $1 WHERE id = $2`, contractID, parentID)
	require.NoError(t, err)
	childID := seedActivity(t, pool, orgID, "task", &parentID, now)

	fc, err := repo.ResolveFundingContext(context.Background(), childID)
	require.NoError(t, err)
	require.NotNil(t, fc)
	require.NotNil(t, fc.ContractID, "contract must resolve through the chain")
	require.Equal(t, contractID, *fc.ContractID)
	require.NotNil(t, fc.ContractType, "contract_type must come via the contracts JOIN")
	require.Equal(t, "support", *fc.ContractType)
	require.NotNil(t, fc.SoldHours, "sold_hours must come via the contracts JOIN")
	require.Equal(t, 120.0, *fc.SoldHours)
}

// TestResolveFundingContext_NoContractNil covers the all-NULL path: an
// internal tree with no contract resolves to nil — the coverage decision
// falls through to absorption (D-04).
func TestResolveFundingContext_NoContractNil(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	rootID := seedActivity(t, pool, orgID, "internal", nil, now)
	childID := seedActivity(t, pool, orgID, "task", &rootID, now)

	fc, err := repo.ResolveFundingContext(context.Background(), childID)
	require.NoError(t, err)
	require.Nil(t, fc, "no contract anywhere in the chain → nil")
}
