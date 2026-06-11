package testdata

import (
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/customer"
	domainexpense "github.com/stefanoprivitera/hourglass/internal/core/domain/expense"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/invitation"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/organization"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/password_reset"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/project"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

func UniqueID() string {
	return uuid.New().String()
}

func UniqueEmail() string {
	return uuid.New().String() + "@test.com"
}

func UniqueOrgName() string {
	return "Org_" + uuid.New().String()
}

func NewUser(overrides ...func(*auth.User)) auth.User {
	u := auth.User{
		ID:       uuid.New(),
		Email:    UniqueEmail(),
		Username: "testuser_" + UniqueID(),
		IsActive: true,
	}
	for _, o := range overrides {
		o(&u)
	}
	return u
}

func NewOrganization(overrides ...func(*organization.Organization)) organization.Organization {
	o := organization.Organization{
		ID:   uuid.New(),
		Name: UniqueOrgName(),
		Slug: "org-" + uuid.New().String(),
	}
	for _, o2 := range overrides {
		o2(&o)
	}
	return o
}

func NewOrganizationMembership(overrides ...func(*auth.OrganizationMembership)) auth.OrganizationMembership {
	now := time.Now()
	m := auth.OrganizationMembership{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Role:           "employee",
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	for _, o := range overrides {
		o(&m)
	}
	return m
}

func NewTimeEntry(overrides ...func(*time_entry.TimeEntry)) time_entry.TimeEntry {
	now := time.Now()
	e := time_entry.TimeEntry{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		OrgID:     uuid.New(),
		Hours:     8.0,
		Status:    time_entry.StatusDraft,
		EntryDate: now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	for _, o := range overrides {
		o(&e)
	}
	return e
}

func NewContract(overrides ...func(*contract.Contract)) contract.Contract {
	c := contract.Contract{
		ID:             uuid.New(),
		Name:           "Test Contract",
		KmRate:         0.50,
		Currency:       "EUR",
		CreatedByOrgID: uuid.New(),
		IsActive:       true,
		CreatedAt:      time.Now(),
	}
	for _, o := range overrides {
		o(&c)
	}
	return c
}

func NewCustomer(overrides ...func(*customer.Customer)) customer.Customer {
	c := customer.Customer{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		CompanyName:    "Test Company",
		Email:          UniqueEmail(),
		IsActive:       true,
		CreatedAt:      time.Now(),
	}
	for _, o := range overrides {
		o(&c)
	}
	return c
}

func NewProject(overrides ...func(*project.Project)) project.Project {
	p := project.Project{
		ID:        uuid.New(),
		Name:      "Test Project",
		Type:      models.ProjectTypeBillable,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	for _, o := range overrides {
		o(&p)
	}
	return p
}

func NewUnit(overrides ...func(*unit.Unit)) unit.Unit {
	u := unit.Unit{
		ID:        uuid.New().String(),
		OrgID:     uuid.New(),
		Name:      "Test Unit",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	for _, o := range overrides {
		o(&u)
	}
	return u
}

func NewWorkingGroup(overrides ...func(*working_group.WorkingGroup)) working_group.WorkingGroup {
	wg := working_group.WorkingGroup{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		Name:      "Test Working Group",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	for _, o := range overrides {
		o(&wg)
	}
	return wg
}

func NewInvitation(overrides ...func(*invitation.Invitation)) invitation.Invitation {
	inv := invitation.Invitation{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Email:          UniqueEmail(),
		Status:         invitation.InvitationStatusPending,
		ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
		CreatedAt:      time.Now(),
	}
	for _, o := range overrides {
		o(&inv)
	}
	return inv
}

func NewPasswordReset(overrides ...func(*password_reset.PasswordReset)) password_reset.PasswordReset {
	pr := password_reset.PasswordReset{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		CodeHash:  uuid.New().String(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	for _, o := range overrides {
		o(&pr)
	}
	return pr
}

func NewExpense(overrides ...func(*models.Expense)) models.Expense {
	now := time.Now()
	e := models.Expense{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Status:         models.StatusDraft,
		Date:           now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	for _, o := range overrides {
		o(&e)
	}
	return e
}

func NewExpenseDomain(overrides ...func(*domainexpense.Expense)) domainexpense.Expense {
	now := time.Now()
	e := domainexpense.Expense{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		UserID:    uuid.New(),
		ProjectID: uuid.New(),
		Category:  domainexpense.CategoryMileage,
		Amount:    100.0,
		Status:    domainexpense.StatusDraft,
		EntryDate: now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	for _, o := range overrides {
		o(&e)
	}
	return e
}
