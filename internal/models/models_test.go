package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// TestRoleIsValid
// ---------------------------------------------------------------------------

func TestRoleIsValid(t *testing.T) {
	validCases := []Role{RoleEmployee, RoleManager, RoleFinance, RoleCustomer}
	for _, r := range validCases {
		t.Run(string(r)+"_valid", func(t *testing.T) {
			assert.True(t, r.IsValid(), "role %q should be valid", r)
		})
	}

	invalidCases := []Role{"admin", "superuser", "", "ceo"}
	for _, r := range invalidCases {
		t.Run(string(r)+"_invalid", func(t *testing.T) {
			assert.False(t, r.IsValid(), "role %q should be invalid", r)
		})
	}
}

// ---------------------------------------------------------------------------
// TestEntryStatusIsValid
// ---------------------------------------------------------------------------

func TestEntryStatusIsValid(t *testing.T) {
	validCases := []EntryStatus{
		StatusDraft, StatusSubmitted,
		StatusPendingManager, StatusPendingFinance,
		StatusApproved, StatusRejected,
	}
	for _, s := range validCases {
		t.Run(string(s)+"_valid", func(t *testing.T) {
			assert.True(t, s.IsValid(), "entry status %q should be valid", s)
		})
	}

	invalidCases := []EntryStatus{"deleted", "", "pending"}
	for _, s := range invalidCases {
		t.Run(string(s)+"_invalid", func(t *testing.T) {
			assert.False(t, s.IsValid(), "entry status %q should be invalid", s)
		})
	}
}

// ---------------------------------------------------------------------------
// TestGovernanceModelIsValid
// ---------------------------------------------------------------------------

func TestGovernanceModelIsValid(t *testing.T) {
	validCases := []GovernanceModel{
		GovernanceCreatorControlled, GovernanceUnanimous, GovernanceMajority,
	}
	for _, g := range validCases {
		t.Run(string(g)+"_valid", func(t *testing.T) {
			assert.True(t, g.IsValid(), "governance model %q should be valid", g)
		})
	}

	invalidCases := []GovernanceModel{"democracy", "", "dictatorship"}
	for _, g := range invalidCases {
		t.Run(string(g)+"_invalid", func(t *testing.T) {
			assert.False(t, g.IsValid(), "governance model %q should be invalid", g)
		})
	}
}

// ---------------------------------------------------------------------------
// TestProjectTypeIsValid
// ---------------------------------------------------------------------------

func TestProjectTypeIsValid(t *testing.T) {
	validCases := []ProjectType{ProjectTypeBillable, ProjectTypeInternal}
	for _, p := range validCases {
		t.Run(string(p)+"_valid", func(t *testing.T) {
			assert.True(t, p.IsValid(), "project type %q should be valid", p)
		})
	}

	invalidCases := []ProjectType{"external", ""}
	for _, p := range invalidCases {
		t.Run(string(p)+"_invalid", func(t *testing.T) {
			assert.False(t, p.IsValid(), "project type %q should be invalid", p)
		})
	}
}

// ---------------------------------------------------------------------------
// TestExpenseCategoryIsValid
// ---------------------------------------------------------------------------

func TestExpenseCategoryIsValid(t *testing.T) {
	validCases := []ExpenseCategory{
		CategoryMileage, CategoryMeal, CategoryAccommodation,
		CategoryParking, CategoryTravelTickets, CategoryTolls,
		CategoryTaxi, CategoryEquipment, CategoryOther,
	}
	for _, c := range validCases {
		t.Run(string(c)+"_valid", func(t *testing.T) {
			assert.True(t, c.IsValid(), "expense category %q should be valid", c)
		})
	}

	invalidCases := []ExpenseCategory{"invalid", ""}
	for _, c := range invalidCases {
		t.Run(string(c)+"_invalid", func(t *testing.T) {
			assert.False(t, c.IsValid(), "expense category %q should be invalid", c)
		})
	}
}
