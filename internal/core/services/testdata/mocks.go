package testdata

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	customerdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/customer"
	domainexpense "github.com/stefanoprivitera/hourglass/internal/core/domain/expense"
	invitationdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/invitation"
	orgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/organization"
	pwdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/password_reset"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	unitdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	wgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

type MockTimeEntryRepo struct {
	mu           sync.Mutex
	Entries      map[uuid.UUID]*time_entry.TimeEntry
	PeriodLocked bool
}

func (m *MockTimeEntryRepo) List(ctx context.Context, orgID uuid.UUID, filters ports.ListFilters) ([]time_entry.TimeEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []time_entry.TimeEntry
	for _, e := range m.Entries {
		if e.OrgID == orgID {
			result = append(result, *e)
		}
	}
	return result, nil
}

func (m *MockTimeEntryRepo) GetByID(ctx context.Context, id uuid.UUID) (*time_entry.TimeEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.Entries[id]
	if !ok {
		return nil, time_entry.ErrTimeEntryNotFound
	}
	return e, nil
}

func (m *MockTimeEntryRepo) Create(ctx context.Context, e *time_entry.TimeEntry) (*time_entry.TimeEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Entries == nil {
		m.Entries = make(map[uuid.UUID]*time_entry.TimeEntry)
	}
	m.Entries[e.ID] = e
	return e, nil
}

func (m *MockTimeEntryRepo) Update(ctx context.Context, e *time_entry.TimeEntry) (*time_entry.TimeEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.Entries[e.ID]
	if !ok {
		return nil, time_entry.ErrTimeEntryNotFound
	}
	m.Entries[e.ID] = e
	return e, nil
}

func (m *MockTimeEntryRepo) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Entries, id)
	return nil
}

func (m *MockTimeEntryRepo) IsPeriodLocked(ctx context.Context, orgID, projectID uuid.UUID, entryDate string) (bool, error) {
	return m.PeriodLocked, nil
}

func (m *MockTimeEntryRepo) ListPending(ctx context.Context, orgID uuid.UUID, role, userID string) ([]time_entry.TimeEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []time_entry.TimeEntry
	for _, e := range m.Entries {
		if e.OrgID == orgID {
			result = append(result, *e)
		}
	}
	return result, nil
}

// MockTimeEntryAuditLogRepo records entry-scoped audit rows (time_entry
// domain type). Named TimeEntry* to distinguish it from the general
// MockAuditLogRepo (audit.AuditLog, D-05) in mock_audit_log_repo.go.
type MockTimeEntryAuditLogRepo struct {
	mu        sync.Mutex
	AuditLogs []*time_entry.AuditLog
}

func (m *MockTimeEntryAuditLogRepo) Create(ctx context.Context, log *time_entry.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AuditLogs = append(m.AuditLogs, log)
	return nil
}

type MockUserRepo struct {
	mu          sync.Mutex
	Users       map[uuid.UUID]*auth.User
	Memberships map[uuid.UUID][]auth.OrganizationMembership
}

func (m *MockUserRepo) Add(ctx context.Context, user *auth.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Users == nil {
		m.Users = make(map[uuid.UUID]*auth.User)
	}
	m.Users[user.ID] = user
	return nil
}

func (m *MockUserRepo) AddWithMembership(ctx context.Context, user *auth.User, membership *auth.OrganizationMembership) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Users == nil {
		m.Users = make(map[uuid.UUID]*auth.User)
	}
	m.Users[user.ID] = user
	return nil
}

func (m *MockUserRepo) AddWithOrgAndMembership(ctx context.Context, user *auth.User, org *auth.Organization, membership *auth.OrganizationMembership) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Users == nil {
		m.Users = make(map[uuid.UUID]*auth.User)
	}
	m.Users[user.ID] = user
	return nil
}

func (m *MockUserRepo) GetByEmail(ctx context.Context, email string) (*auth.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.Users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, ports.ErrUserNotFound
}

func (m *MockUserRepo) GetByUsername(ctx context.Context, username string) (*auth.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.Users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, ports.ErrUserNotFound
}

func (m *MockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.Users[id]
	if !ok {
		return nil, ports.ErrUserNotFound
	}
	return u, nil
}

func (m *MockUserRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.Users {
		if u.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockUserRepo) UsernameExists(ctx context.Context, username string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.Users {
		if u.Username == username {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockUserRepo) AnyExists(ctx context.Context) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Users) > 0, nil
}

func (m *MockUserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.Users[userID]
	if !ok {
		return ports.ErrUserNotFound
	}
	u.PasswordHash = passwordHash
	return nil
}

func (m *MockUserRepo) GetMemberships(ctx context.Context, userID uuid.UUID) ([]auth.OrganizationMembership, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Memberships == nil {
		return nil, nil
	}
	memberships, ok := m.Memberships[userID]
	if !ok {
		return nil, nil
	}
	return memberships, nil
}

type MockOrgRepo struct {
	mu          sync.Mutex
	Orgs        map[uuid.UUID]*auth.Organization
	Memberships map[string]*auth.OrganizationMembership // key = userID+":"+orgID
}

func (m *MockOrgRepo) Add(ctx context.Context, org *auth.Organization) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Orgs == nil {
		m.Orgs = make(map[uuid.UUID]*auth.Organization)
	}
	m.Orgs[org.ID] = org
	return nil
}

func (m *MockOrgRepo) GetByID(ctx context.Context, id uuid.UUID) (*auth.Organization, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.Orgs[id]
	if !ok {
		return nil, nil
	}
	return o, nil
}

func (m *MockOrgRepo) GetMembership(ctx context.Context, userID, orgID uuid.UUID) (*auth.OrganizationMembership, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Memberships == nil {
		return nil, nil
	}
	key := userID.String() + ":" + orgID.String()
	membership, ok := m.Memberships[key]
	if !ok {
		return nil, nil
	}
	return membership, nil
}

func (m *MockOrgRepo) AddMembership(ctx context.Context, membership *auth.OrganizationMembership) error {
	return nil
}

type MockOrgMgmtRepo struct {
	mu       sync.Mutex
	Orgs     map[uuid.UUID]*orgdomain.Organization
	Members  []orgdomain.Member
	Settings *orgdomain.Settings
}

func (m *MockOrgMgmtRepo) CreateOrganization(ctx context.Context, org *orgdomain.Organization, ownerUserID uuid.UUID, ownerRole models.Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Orgs == nil {
		m.Orgs = make(map[uuid.UUID]*orgdomain.Organization)
	}
	m.Orgs[org.ID] = org
	return nil
}

func (m *MockOrgMgmtRepo) GetOrganization(ctx context.Context, id uuid.UUID) (*orgdomain.Organization, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.Orgs[id]
	if !ok {
		return nil, orgdomain.ErrOrganizationNotFound
	}
	return o, nil
}

func (m *MockOrgMgmtRepo) InviteMember(ctx context.Context, orgID uuid.UUID, req *orgdomain.InviteRequest, invitedBy uuid.UUID) (uuid.UUID, time.Time, error) {
	return uuid.New(), time.Now(), nil
}

func (m *MockOrgMgmtRepo) GetSettings(ctx context.Context, orgID uuid.UUID) (*orgdomain.Settings, error) {
	if m.Settings != nil {
		return m.Settings, nil
	}
	return &orgdomain.Settings{}, nil
}

func (m *MockOrgMgmtRepo) UpdateSettings(ctx context.Context, orgID uuid.UUID, req *orgdomain.UpdateSettingsRequest) (*orgdomain.Settings, error) {
	return &orgdomain.Settings{}, nil
}

func (m *MockOrgMgmtRepo) ListMembers(ctx context.Context, orgID uuid.UUID) ([]orgdomain.Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Members, nil
}

func (m *MockOrgMgmtRepo) UpdateMemberRole(ctx context.Context, orgID, memberID uuid.UUID, role models.Role) error {
	return nil
}

func (m *MockOrgMgmtRepo) DeactivateMember(ctx context.Context, orgID, memberID uuid.UUID) error {
	return nil
}

func (m *MockOrgMgmtRepo) CountActiveFinance(ctx context.Context, orgID uuid.UUID) (int, error) {
	return 1, nil
}

func (m *MockOrgMgmtRepo) GetMemberRole(ctx context.Context, memberID uuid.UUID) (models.Role, error) {
	return models.RoleEmployee, nil
}

type MockContractRepo struct {
	mu            sync.Mutex
	Contracts     map[uuid.UUID]*contractdomain.ContractResponse
	HasProjectsFn func(ctx context.Context, contractID uuid.UUID) (int, error)
}

func (m *MockContractRepo) List(ctx context.Context, orgID uuid.UUID, scope string, isActive *bool) ([]contractdomain.ContractResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []contractdomain.ContractResponse
	for _, c := range m.Contracts {
		result = append(result, *c)
	}
	return result, nil
}

func (m *MockContractRepo) Create(ctx context.Context, orgID uuid.UUID, req *contractdomain.CreateContractRequest) (*contractdomain.ContractResponse, error) {
	return &contractdomain.ContractResponse{}, nil
}

func (m *MockContractRepo) Get(ctx context.Context, orgID, contractID uuid.UUID) (*contractdomain.ContractResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.Contracts[contractID]
	if !ok {
		return nil, contractdomain.ErrContractNotFound
	}
	return c, nil
}

func (m *MockContractRepo) Adopt(ctx context.Context, orgID, contractID uuid.UUID) (*contractdomain.ContractAdoption, error) {
	return &contractdomain.ContractAdoption{}, nil
}

func (m *MockContractRepo) Update(ctx context.Context, orgID, contractID uuid.UUID, req *contractdomain.UpdateContractRequest) (*contractdomain.ContractResponse, int, error) {
	return &contractdomain.ContractResponse{}, 0, nil
}

func (m *MockContractRepo) RecalculateMileage(ctx context.Context, orgID, contractID uuid.UUID, fromDate string, actorUserID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *MockContractRepo) Delete(ctx context.Context, orgID, contractID uuid.UUID) error {
	return nil
}

func (m *MockContractRepo) HasTimeEntries(ctx context.Context, contractID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *MockContractRepo) HasProjects(ctx context.Context, contractID uuid.UUID) (int, error) {
	if m.HasProjectsFn != nil {
		return m.HasProjectsFn(ctx, contractID)
	}
	return 0, nil
}

type MockCustomerRepo struct {
	mu        sync.Mutex
	Customers map[uuid.UUID]*customerdomain.Customer
}

func (m *MockCustomerRepo) ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int, search string) ([]customerdomain.Customer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []customerdomain.Customer
	for _, c := range m.Customers {
		if search == "" || strings.Contains(c.CompanyName, search) || strings.Contains(c.ContactName, search) || strings.Contains(c.Email, search) {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (m *MockCustomerRepo) CreateInternal(ctx context.Context, orgID uuid.UUID, companyName string) (*customerdomain.Customer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := &customerdomain.Customer{
		ID:             uuid.New(),
		OrganizationID: orgID,
		CompanyName:    companyName,
		IsActive:       true,
		IsInternal:     true,
	}
	if m.Customers == nil {
		m.Customers = make(map[uuid.UUID]*customerdomain.Customer)
	}
	m.Customers[c.ID] = c
	return c, nil
}

func (m *MockCustomerRepo) Create(ctx context.Context, c *customerdomain.Customer) (*customerdomain.Customer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Customers == nil {
		m.Customers = make(map[uuid.UUID]*customerdomain.Customer)
	}
	m.Customers[c.ID] = c
	return c, nil
}

func (m *MockCustomerRepo) GetByID(ctx context.Context, id uuid.UUID) (*customerdomain.Customer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.Customers[id]
	if !ok {
		return nil, customerdomain.ErrCustomerNotFound
	}
	return c, nil
}

func (m *MockCustomerRepo) Update(ctx context.Context, c *customerdomain.Customer) (*customerdomain.Customer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.Customers[c.ID]
	if !ok {
		return nil, customerdomain.ErrCustomerNotFound
	}
	m.Customers[c.ID] = c
	return c, nil
}

func (m *MockCustomerRepo) Deactivate(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Customers, id)
	return nil
}

func (m *MockCustomerRepo) ListContractsByCustomer(ctx context.Context, customerID uuid.UUID) ([]customerdomain.ContractSummary, error) {
	return nil, nil
}

func (m *MockCustomerRepo) CountContractsByCustomer(ctx context.Context, customerID uuid.UUID) (int, error) {
	return 0, nil
}

// MockActivityRepo implements ports.ActivityRepository for service unit tests.
// Kinds is keyed by orgID.String()+":"+kind — seed it to model the org's
// activity_kinds catalog (ADR-P-007 D-2).
type MockActivityRepo struct {
	mu                         sync.Mutex
	Activities                 map[uuid.UUID]*activitydomain.ActivityResponse
	Kinds                      map[string]bool
	HasChildrenFn              func(ctx context.Context, activityID uuid.UUID) (bool, error)
	HasActiveTimeEntriesFn     func(ctx context.Context, activityID uuid.UUID) (bool, bool, error)
	HasActiveExpensesFn        func(ctx context.Context, activityID uuid.UUID) (bool, error)
	ResolveCommercialContextFn func(ctx context.Context, activityID uuid.UUID) (*activitydomain.CommercialContext, error)
	GetAncestryFn              func(ctx context.Context, id uuid.UUID) ([]activitydomain.Activity, error)
}

func (m *MockActivityRepo) List(ctx context.Context, orgID uuid.UUID, filter *activitydomain.ActivityFilter) ([]activitydomain.ActivityResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []activitydomain.ActivityResponse
	for _, a := range m.Activities {
		if a.OrgID == orgID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *MockActivityRepo) Get(ctx context.Context, orgID, activityID uuid.UUID) (*activitydomain.ActivityResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.Activities[activityID]
	if !ok {
		return nil, activitydomain.ErrActivityNotFound
	}
	return a, nil
}

func (m *MockActivityRepo) Create(ctx context.Context, orgID uuid.UUID, req *activitydomain.CreateActivityRequest) (*activitydomain.ActivityResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Activities == nil {
		m.Activities = make(map[uuid.UUID]*activitydomain.ActivityResponse)
	}
	a := &activitydomain.ActivityResponse{}
	a.ID = uuid.New()
	a.OrgID = orgID
	a.ParentID = req.ParentID
	a.Name = req.Name
	a.Description = req.Description
	a.Kind = req.Kind
	a.ContractID = req.ContractID
	a.GovernanceModel = req.GovernanceModel
	a.CreatedByOrgID = orgID
	a.IsShared = req.IsShared
	a.Billable = req.Billable
	a.BudgetAmount = req.BudgetAmount
	a.IsActive = true
	m.Activities[a.ID] = a
	return a, nil
}

func (m *MockActivityRepo) Update(ctx context.Context, orgID, activityID uuid.UUID, req *activitydomain.UpdateActivityRequest) (*activitydomain.ActivityResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.Activities[activityID]
	if !ok {
		return nil, activitydomain.ErrActivityNotFound
	}
	if req.Name != "" {
		a.Name = req.Name
	}
	if req.Kind != "" {
		a.Kind = req.Kind
	}
	if req.IsActive != nil {
		a.IsActive = *req.IsActive
	}
	return a, nil
}

func (m *MockActivityRepo) Delete(ctx context.Context, orgID, activityID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Activities[activityID]; !ok {
		return activitydomain.ErrActivityNotFound
	}
	delete(m.Activities, activityID)
	return nil
}

func (m *MockActivityRepo) Adopt(ctx context.Context, orgID, activityID uuid.UUID) (*activitydomain.ActivityAdoption, error) {
	return &activitydomain.ActivityAdoption{ID: uuid.New(), ActivityID: activityID, OrganizationID: orgID}, nil
}

func (m *MockActivityRepo) ListChildren(ctx context.Context, parentID uuid.UUID) ([]activitydomain.ActivityResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []activitydomain.ActivityResponse
	for _, a := range m.Activities {
		if a.ParentID != nil && *a.ParentID == parentID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *MockActivityRepo) ListByContract(ctx context.Context, contractID uuid.UUID) ([]activitydomain.ActivityResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []activitydomain.ActivityResponse
	for _, a := range m.Activities {
		if a.ContractID != nil && *a.ContractID == contractID {
			result = append(result, *a)
		}
	}
	return result, nil
}

// GetAncestry walks parent_id upward from the given node to the root,
// mirroring the adapter's recursive CTE (the chain includes the starting node
// itself — required for self-parent cycle detection in the service's
// validateParent). Override with GetAncestryFn when a test needs a
// non-derived answer.
func (m *MockActivityRepo) GetAncestry(ctx context.Context, id uuid.UUID) ([]activitydomain.Activity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.GetAncestryFn != nil {
		return m.GetAncestryFn(ctx, id)
	}
	seen := map[uuid.UUID]bool{}
	var chain []activitydomain.Activity
	cur := id
	for cur != uuid.Nil && !seen[cur] {
		seen[cur] = true
		a, ok := m.Activities[cur]
		if !ok {
			break
		}
		chain = append(chain, a.Activity)
		if a.ParentID == nil {
			break
		}
		cur = *a.ParentID
	}
	return chain, nil
}

// ResolveCommercialContext derives the nearest contract-bearing ancestor from
// the stored activities (mirrors the adapter's recursive walk). Override with
// ResolveCommercialContextFn when a test needs a non-derived answer.
func (m *MockActivityRepo) ResolveCommercialContext(ctx context.Context, activityID uuid.UUID) (*activitydomain.CommercialContext, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ResolveCommercialContextFn != nil {
		return m.ResolveCommercialContextFn(ctx, activityID)
	}
	seen := map[uuid.UUID]bool{}
	cur := activityID
	for cur != uuid.Nil && !seen[cur] {
		seen[cur] = true
		a, ok := m.Activities[cur]
		if !ok {
			return nil, nil
		}
		if a.ContractID != nil {
			return &activitydomain.CommercialContext{ContractID: a.ContractID}, nil
		}
		if a.ParentID == nil {
			break
		}
		cur = *a.ParentID
	}
	return nil, nil
}

func (m *MockActivityRepo) ResolveBillability(ctx context.Context, activityID uuid.UUID) (*bool, error) {
	f := false
	return &f, nil
}

func (m *MockActivityRepo) ListManagers(ctx context.Context, activityID uuid.UUID) ([]activitydomain.ActivityManager, error) {
	return nil, nil
}

func (m *MockActivityRepo) AddManager(ctx context.Context, activityID, userID uuid.UUID) (*activitydomain.ActivityManager, error) {
	return &activitydomain.ActivityManager{ID: uuid.New(), ActivityID: activityID, UserID: userID}, nil
}

func (m *MockActivityRepo) RemoveManager(ctx context.Context, activityID, userID uuid.UUID) error {
	return nil
}

func (m *MockActivityRepo) KindExists(ctx context.Context, orgID uuid.UUID, kind string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Kinds[orgID.String()+":"+kind], nil
}

// ListKinds returns the org's catalog from the Kinds map (keyed
// orgID.String()+":"+kind).
func (m *MockActivityRepo) ListKinds(ctx context.Context, orgID uuid.UUID) ([]activitydomain.ActivityKind, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := orgID.String() + ":"
	var kinds []activitydomain.ActivityKind
	for k := range m.Kinds {
		if strings.HasPrefix(k, prefix) {
			kinds = append(kinds, activitydomain.ActivityKind(strings.TrimPrefix(k, prefix)))
		}
	}
	if kinds == nil {
		kinds = []activitydomain.ActivityKind{}
	}
	return kinds, nil
}

func (m *MockActivityRepo) HasChildren(ctx context.Context, activityID uuid.UUID) (bool, error) {
	if m.HasChildrenFn != nil {
		return m.HasChildrenFn(ctx, activityID)
	}
	return false, nil
}

func (m *MockActivityRepo) HasActiveTimeEntries(ctx context.Context, activityID uuid.UUID) (bool, bool, error) {
	if m.HasActiveTimeEntriesFn != nil {
		return m.HasActiveTimeEntriesFn(ctx, activityID)
	}
	return false, false, nil
}

func (m *MockActivityRepo) HasActiveExpenses(ctx context.Context, activityID uuid.UUID) (bool, error) {
	if m.HasActiveExpensesFn != nil {
		return m.HasActiveExpensesFn(ctx, activityID)
	}
	return false, nil
}

type MockUnitRepo struct {
	mu          sync.Mutex
	Units       map[string]*unitdomain.Unit
	Descendants map[string][]unitdomain.Unit       // key = unitID
	UnitMembers map[string][]unitdomain.UnitMember // key = unitID
}

func (m *MockUnitRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]unitdomain.Unit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []unitdomain.Unit
	for _, u := range m.Units {
		result = append(result, *u)
	}
	return result, nil
}

func (m *MockUnitRepo) GetByID(ctx context.Context, id string) (*unitdomain.Unit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.Units[id]
	if !ok {
		return nil, unitdomain.ErrUnitNotFound
	}
	return u, nil
}

func (m *MockUnitRepo) Create(ctx context.Context, u *unitdomain.Unit) (*unitdomain.Unit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Units == nil {
		m.Units = make(map[string]*unitdomain.Unit)
	}
	m.Units[u.ID] = u
	return u, nil
}

func (m *MockUnitRepo) Update(ctx context.Context, u *unitdomain.Unit) (*unitdomain.Unit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.Units[u.ID]
	if !ok {
		return nil, unitdomain.ErrUnitNotFound
	}
	m.Units[u.ID] = u
	return u, nil
}

func (m *MockUnitRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Units, id)
	return nil
}

func (m *MockUnitRepo) GetDescendants(ctx context.Context, id string) ([]unitdomain.Unit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Descendants == nil {
		return nil, nil
	}
	descendants, ok := m.Descendants[id]
	if !ok {
		return nil, nil
	}
	return descendants, nil
}

func (m *MockUnitRepo) HasMembers(ctx context.Context, id string) (bool, error) {
	return false, nil
}

func (m *MockUnitRepo) ListMembers(ctx context.Context, unitID string) ([]unitdomain.UnitMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.UnitMembers == nil {
		return nil, nil
	}
	members, ok := m.UnitMembers[unitID]
	if !ok {
		return nil, nil
	}
	return members, nil
}

func (m *MockUnitRepo) AddMember(ctx context.Context, mm *unitdomain.UnitMember) (*unitdomain.UnitMember, error) {
	return mm, nil
}

func (m *MockUnitRepo) RemoveMember(ctx context.Context, id string) error {
	return nil
}

func (m *MockUnitRepo) HasChildren(ctx context.Context, id string) (bool, error) {
	return false, nil
}

func (m *MockUnitRepo) UpdateMember(ctx context.Context, mm *unitdomain.UnitMember) (*unitdomain.UnitMember, error) {
	return mm, nil
}

func (m *MockUnitRepo) ListMembersByUnitIDs(ctx context.Context, orgID uuid.UUID, unitIDs []string) ([]unitdomain.UnitMember, error) {
	return nil, nil
}

func (m *MockUnitRepo) ListMembershipsForUser(ctx context.Context, userID uuid.UUID) ([]unitdomain.UnitMember, error) {
	return nil, nil
}

func (m *MockUnitRepo) GetMemberCountsByOrg(ctx context.Context, orgID uuid.UUID) (map[string]int, error) {
	return nil, nil
}

type MockWorkingGroupRepo struct {
	mu        sync.Mutex
	Groups    map[uuid.UUID]*wgdomain.WorkingGroup
	WGMembers map[string][]wgdomain.WorkingGroupMember // key = wgID.String()
}

// ListByOrg returns the org's working groups, optionally filtered by the
// anchored activity (working_groups.activity_id — ADR-P-007 D-5). The filter
// honors the WorkingGroup.SubprojectID field, which maps to activity_id
// (legacy field name kept for compile safety).
func (m *MockWorkingGroupRepo) ListByOrg(ctx context.Context, orgID uuid.UUID, activityID *uuid.UUID) ([]wgdomain.WorkingGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []wgdomain.WorkingGroup
	for _, g := range m.Groups {
		if g.OrgID != orgID {
			continue
		}
		if activityID != nil && g.SubprojectID != *activityID {
			continue
		}
		result = append(result, *g)
	}
	return result, nil
}

func (m *MockWorkingGroupRepo) GetByID(ctx context.Context, id uuid.UUID) (*wgdomain.WorkingGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.Groups[id]
	if !ok {
		return nil, wgdomain.ErrWorkingGroupNotFound
	}
	return g, nil
}

func (m *MockWorkingGroupRepo) Create(ctx context.Context, g *wgdomain.WorkingGroup) (*wgdomain.WorkingGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Groups == nil {
		m.Groups = make(map[uuid.UUID]*wgdomain.WorkingGroup)
	}
	m.Groups[g.ID] = g
	return g, nil
}

func (m *MockWorkingGroupRepo) Update(ctx context.Context, g *wgdomain.WorkingGroup) (*wgdomain.WorkingGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.Groups[g.ID]
	if !ok {
		return nil, wgdomain.ErrWorkingGroupNotFound
	}
	m.Groups[g.ID] = g
	return g, nil
}

func (m *MockWorkingGroupRepo) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Groups, id)
	return nil
}

func (m *MockWorkingGroupRepo) HasMembers(ctx context.Context, id uuid.UUID) (bool, error) {
	return false, nil
}

func (m *MockWorkingGroupRepo) ListMembers(ctx context.Context, wgID uuid.UUID) ([]wgdomain.WorkingGroupMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.WGMembers == nil {
		return nil, nil
	}
	key := wgID.String()
	members, ok := m.WGMembers[key]
	if !ok {
		return nil, nil
	}
	return members, nil
}

func (m *MockWorkingGroupRepo) AddMember(ctx context.Context, mm *wgdomain.WorkingGroupMember) (*wgdomain.WorkingGroupMember, error) {
	return mm, nil
}

func (m *MockWorkingGroupRepo) RemoveMember(ctx context.Context, id uuid.UUID) error {
	return nil
}

type MockInvitationRepo struct {
	mu          sync.Mutex
	Invitations map[uuid.UUID]*invitationdomain.Invitation
}

func (m *MockInvitationRepo) Create(ctx context.Context, inv *invitationdomain.Invitation) (*invitationdomain.Invitation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Invitations == nil {
		m.Invitations = make(map[uuid.UUID]*invitationdomain.Invitation)
	}
	m.Invitations[inv.ID] = inv
	return inv, nil
}

func (m *MockInvitationRepo) FindByCode(ctx context.Context, code string) (*invitationdomain.Invitation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, inv := range m.Invitations {
		if inv.Code == code {
			return inv, nil
		}
	}
	return nil, invitationdomain.ErrInvitationNotFound
}

func (m *MockInvitationRepo) FindByToken(ctx context.Context, token string) (*invitationdomain.Invitation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, inv := range m.Invitations {
		if inv.InviteToken == token {
			return inv, nil
		}
	}
	return nil, invitationdomain.ErrInvitationNotFound
}

func (m *MockInvitationRepo) Update(ctx context.Context, inv *invitationdomain.Invitation) (*invitationdomain.Invitation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.Invitations[inv.ID]
	if !ok {
		return nil, invitationdomain.ErrInvitationNotFound
	}
	m.Invitations[inv.ID] = inv
	return inv, nil
}

type MockPasswordResetRepo struct {
	mu               sync.Mutex
	Resets           map[uuid.UUID]*pwdomain.PasswordReset
	FindActiveResets map[string]*pwdomain.PasswordReset // key = userID string
}

func (m *MockPasswordResetRepo) Create(ctx context.Context, pr *pwdomain.PasswordReset) (*pwdomain.PasswordReset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Resets == nil {
		m.Resets = make(map[uuid.UUID]*pwdomain.PasswordReset)
	}
	m.Resets[pr.ID] = pr
	return pr, nil
}

func (m *MockPasswordResetRepo) FindActiveByUserID(ctx context.Context, userID string) (*pwdomain.PasswordReset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FindActiveResets == nil {
		return nil, pwdomain.ErrResetNotFound
	}
	reset, ok := m.FindActiveResets[userID]
	if !ok {
		return nil, pwdomain.ErrResetNotFound
	}
	return reset, nil
}

func (m *MockPasswordResetRepo) MarkUsed(ctx context.Context, id string) error {
	return nil
}

func (m *MockPasswordResetRepo) UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	return nil
}

type MockExportRepo struct {
	mu sync.Mutex
}

func (m *MockExportRepo) Timesheets(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ports.ExportRow, error) {
	return nil, nil
}

func (m *MockExportRepo) Expenses(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ports.ExportRow, error) {
	return nil, nil
}

func (m *MockExportRepo) CountTimesheets(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *MockExportRepo) CountExpenses(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) (int, error) {
	return 0, nil
}

type MockRefreshTokenRepo struct {
	mu        sync.Mutex
	Tokens    map[string]*ports.RefreshToken
	RotateErr error // when set, Rotate returns this error without mutating state
}

func (m *MockRefreshTokenRepo) Add(ctx context.Context, userID, organizationID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Tokens == nil {
		m.Tokens = make(map[string]*ports.RefreshToken)
	}
	m.Tokens[tokenHash] = &ports.RefreshToken{
		UserID:         userID,
		OrganizationID: organizationID,
		Hash:           tokenHash,
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now(),
		FamilyID:       uuid.New(),
	}
	return nil
}

func (m *MockRefreshTokenRepo) FindByHash(ctx context.Context, hash string) (*ports.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Tokens == nil {
		return nil, nil
	}
	token, ok := m.Tokens[hash]
	if !ok || token.RotatedAt != nil || token.RevokedAt != nil {
		return nil, nil
	}
	return token, nil
}

func (m *MockRefreshTokenRepo) RevokeByHash(ctx context.Context, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Tokens == nil {
		return nil
	}
	if token, ok := m.Tokens[hash]; ok && token.RevokedAt == nil {
		now := time.Now()
		token.RevokedAt = &now
		m.Tokens[hash] = token
	}
	return nil
}

func (m *MockRefreshTokenRepo) RevokeAllByUser(ctx context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for h, t := range m.Tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &now
			m.Tokens[h] = t
		}
	}
	return nil
}

func (m *MockRefreshTokenRepo) RevokeFamily(ctx context.Context, familyID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for h, t := range m.Tokens {
		if t.FamilyID == familyID && t.RevokedAt == nil {
			t.RevokedAt = &now
			m.Tokens[h] = t
		}
	}
	return nil
}

// Rotate mirrors the postgres implementation's semantics:
//   - unknown hash -> (nil, nil)
//   - already rotated/revoked -> revoke the whole family, return token + ports.ErrTokenReuse
//   - valid -> mark rotated_at, insert successor with the same family_id
func (m *MockRefreshTokenRepo) Rotate(ctx context.Context, hash, newHash string, newExpiresAt time.Time) (*ports.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.RotateErr != nil {
		// Simulate a repo-level failure mid-rotation (e.g. the successor
		// insert fails and the real transaction rolls back). No state may be
		// mutated: the old token stays exactly as it was.
		return nil, m.RotateErr
	}
	if m.Tokens == nil {
		return nil, nil
	}
	token, ok := m.Tokens[hash]
	if !ok {
		return nil, nil
	}
	if token.RotatedAt != nil || token.RevokedAt != nil {
		now := time.Now()
		for h, t := range m.Tokens {
			if t.FamilyID == token.FamilyID && t.RevokedAt == nil {
				t.RevokedAt = &now
				m.Tokens[h] = t
			}
		}
		return token, ports.ErrTokenReuse
	}
	now := time.Now()
	token.RotatedAt = &now
	m.Tokens[hash] = token
	m.Tokens[newHash] = &ports.RefreshToken{
		UserID:         token.UserID,
		OrganizationID: token.OrganizationID,
		Hash:           newHash,
		ExpiresAt:      newExpiresAt,
		CreatedAt:      now,
		FamilyID:       token.FamilyID,
	}
	return token, nil
}

type MockTokenService struct {
	mu sync.Mutex
}

func (m *MockTokenService) GenerateToken(userID, organizationID uuid.UUID, role, email string) (string, error) {
	return "mock-token", nil
}

func (m *MockTokenService) ValidateToken(tokenString string) (*ports.Claims, error) {
	return &ports.Claims{}, nil
}

func (m *MockTokenService) GenerateRefreshToken() (string, error) {
	return "mock-refresh-token", nil
}

func (m *MockTokenService) HashRefreshToken(token string) string {
	return "hashed-" + token
}

type MockPasswordHasher struct {
	mu sync.Mutex
}

func (m *MockPasswordHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (m *MockPasswordHasher) Check(password, hash string) bool {
	return hash == "hashed:"+password
}

type MockTimeEntryApprovalRepo struct {
	mu        sync.Mutex
	Approvals []*time_entry.Approval
}

func (m *MockTimeEntryApprovalRepo) CreateApproval(ctx context.Context, a *time_entry.Approval) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Approvals = append(m.Approvals, a)
	return nil
}

type MockExpenseRepo struct {
	mu           sync.Mutex
	Expenses     map[uuid.UUID]*domainexpense.Expense
	PeriodLocked bool
	Approvals    []*domainexpense.Approval
}

func (m *MockExpenseRepo) List(ctx context.Context, orgID uuid.UUID, filters ports.ExpenseListFilters) ([]domainexpense.Expense, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domainexpense.Expense
	for _, e := range m.Expenses {
		if e.OrgID == orgID {
			result = append(result, *e)
		}
	}
	return result, nil
}

func (m *MockExpenseRepo) GetByID(ctx context.Context, id uuid.UUID) (*domainexpense.Expense, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.Expenses[id]
	if !ok {
		return nil, domainexpense.ErrExpenseNotFound
	}
	return e, nil
}

func (m *MockExpenseRepo) Create(ctx context.Context, e *domainexpense.Expense) (*domainexpense.Expense, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Expenses == nil {
		m.Expenses = make(map[uuid.UUID]*domainexpense.Expense)
	}
	m.Expenses[e.ID] = e
	return e, nil
}

func (m *MockExpenseRepo) Update(ctx context.Context, e *domainexpense.Expense) (*domainexpense.Expense, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.Expenses[e.ID]
	if !ok {
		return nil, domainexpense.ErrExpenseNotFound
	}
	m.Expenses[e.ID] = e
	return e, nil
}

func (m *MockExpenseRepo) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Expenses, id)
	return nil
}

func (m *MockExpenseRepo) IsPeriodLocked(ctx context.Context, orgID, projectID uuid.UUID, entryDate string) (bool, error) {
	return m.PeriodLocked, nil
}

func (m *MockExpenseRepo) ListPending(ctx context.Context, orgID uuid.UUID, role, userID string) ([]domainexpense.Expense, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domainexpense.Expense
	for _, e := range m.Expenses {
		if e.OrgID == orgID {
			result = append(result, *e)
		}
	}
	return result, nil
}

func (m *MockExpenseRepo) CreateApproval(ctx context.Context, a *domainexpense.Approval) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Approvals = append(m.Approvals, a)
	return nil
}
