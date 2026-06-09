-- 002_full_schema.up.sql — Full PostgreSQL schema migration
--
-- Creates all 24 tables with proper types, constraints, indexes, and FKs.
-- Dependency-ordered: parents before children.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- 1. organizations
-- ============================================================================
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    financial_cutoff_days INT,
    financial_cutoff_config JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug);

-- ============================================================================
-- 2. users
-- ============================================================================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    firstname VARCHAR(255),
    lastname VARCHAR(255),
    username VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- ============================================================================
-- 3. customers
-- ============================================================================
CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(255) NOT NULL,
    contact_name VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(255),
    address TEXT,
    vat_number VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_customers_org_id ON customers(org_id);

-- ============================================================================
-- 4. units (self-referencing FK: parent_unit_id -> units(id))
-- ============================================================================
CREATE TABLE IF NOT EXISTS units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    parent_unit_id UUID REFERENCES units(id) ON DELETE RESTRICT,
    hierarchy_level INT NOT NULL DEFAULT 0,
    code VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_units_org_id ON units(org_id);
CREATE INDEX IF NOT EXISTS idx_units_parent_unit_id ON units(parent_unit_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_units_org_code ON units(org_id, code);

-- ============================================================================
-- 5. organization_memberships
-- ============================================================================
CREATE TABLE IF NOT EXISTS organization_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('employee', 'manager', 'finance', 'customer')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    invited_by UUID REFERENCES users(id),
    invited_at TIMESTAMPTZ,
    activated_at TIMESTAMPTZ,
    activation_token VARCHAR(255) UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, organization_id)
);

CREATE INDEX IF NOT EXISTS idx_organization_memberships_user_id ON organization_memberships(user_id);
CREATE INDEX IF NOT EXISTS idx_organization_memberships_organization_id ON organization_memberships(organization_id);

-- ============================================================================
-- 6. unit_memberships
-- ============================================================================
CREATE TABLE IF NOT EXISTS unit_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    user_id UUID NOT NULL REFERENCES users(id),
    unit_id UUID NOT NULL REFERENCES units(id),
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    role VARCHAR(50) NOT NULL DEFAULT 'employee',
    start_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    end_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_unit_memberships_org_id ON unit_memberships(org_id);
CREATE INDEX IF NOT EXISTS idx_unit_memberships_user_id ON unit_memberships(user_id);
CREATE INDEX IF NOT EXISTS idx_unit_memberships_unit_id ON unit_memberships(unit_id);
CREATE INDEX IF NOT EXISTS idx_unit_memberships_user_org ON unit_memberships(user_id, org_id);
CREATE INDEX IF NOT EXISTS idx_unit_memberships_user_primary ON unit_memberships(user_id, is_primary);

-- ============================================================================
-- 7. contracts
-- ============================================================================
CREATE TABLE IF NOT EXISTS contracts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    km_rate DECIMAL(10,2) NOT NULL DEFAULT 0,
    currency VARCHAR(10) NOT NULL DEFAULT 'EUR',
    customer_id UUID REFERENCES customers(id) ON DELETE RESTRICT,
    governance_model VARCHAR(50) NOT NULL CHECK (governance_model IN ('creator_controlled', 'unanimous', 'majority')),
    created_by_org_id UUID NOT NULL REFERENCES organizations(id),
    is_shared BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_contracts_customer_id ON contracts(customer_id);
CREATE INDEX IF NOT EXISTS idx_contracts_created_by_org_id ON contracts(created_by_org_id);

-- ============================================================================
-- 8. contract_adoptions
-- ============================================================================
CREATE TABLE IF NOT EXISTS contract_adoptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(contract_id, organization_id)
);

CREATE INDEX IF NOT EXISTS idx_contract_adoptions_contract_id ON contract_adoptions(contract_id);
CREATE INDEX IF NOT EXISTS idx_contract_adoptions_organization_id ON contract_adoptions(organization_id);

-- ============================================================================
-- 9. projects
-- ============================================================================
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    project_type VARCHAR(50) NOT NULL CHECK (project_type IN ('billable', 'internal')),
    type VARCHAR(50) NOT NULL CHECK (type IN ('billable', 'internal')),
    contract_id UUID REFERENCES contracts(id) ON DELETE RESTRICT,
    customer_id UUID REFERENCES customers(id),
    governance_model VARCHAR(50) NOT NULL CHECK (governance_model IN ('creator_controlled', 'unanimous', 'majority')),
    created_by_org_id UUID NOT NULL REFERENCES organizations(id),
    is_shared BOOLEAN NOT NULL DEFAULT FALSE,
    budget_amount DECIMAL(12,2),
    financial_cutoff_config JSONB,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_projects_org_id ON projects(org_id);
CREATE INDEX IF NOT EXISTS idx_projects_customer_id ON projects(customer_id);

-- ============================================================================
-- 10. project_adoptions
-- ============================================================================
CREATE TABLE IF NOT EXISTS project_adoptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, organization_id)
);

CREATE INDEX IF NOT EXISTS idx_project_adoptions_project_id ON project_adoptions(project_id);
CREATE INDEX IF NOT EXISTS idx_project_adoptions_organization_id ON project_adoptions(organization_id);

-- ============================================================================
-- 11. project_managers
-- ============================================================================
CREATE TABLE IF NOT EXISTS project_managers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_project_managers_project_id ON project_managers(project_id);
CREATE INDEX IF NOT EXISTS idx_project_managers_user_id ON project_managers(user_id);

-- ============================================================================
-- 12. subprojects
-- ============================================================================
CREATE TABLE IF NOT EXISTS subprojects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    sequence_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subprojects_project_id ON subprojects(project_id);

-- ============================================================================
-- 13. working_groups
-- ============================================================================
CREATE TABLE IF NOT EXISTS working_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    subproject_id UUID NOT NULL REFERENCES subprojects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    unit_ids UUID[] NOT NULL DEFAULT '{}',
    enforce_unit_tuple BOOLEAN NOT NULL DEFAULT TRUE,
    manager_id UUID NOT NULL REFERENCES users(id),
    delegate_ids UUID[] NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_working_groups_org_id ON working_groups(org_id);
CREATE INDEX IF NOT EXISTS idx_working_groups_subproject_id ON working_groups(subproject_id);
CREATE INDEX IF NOT EXISTS idx_working_groups_manager_id ON working_groups(manager_id);

-- ============================================================================
-- 14. wg_members
-- ============================================================================
CREATE TABLE IF NOT EXISTS wg_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wg_id UUID NOT NULL REFERENCES working_groups(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    unit_id UUID NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    is_default_subproject BOOLEAN NOT NULL DEFAULT FALSE,
    start_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    end_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(wg_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_wg_members_wg_id ON wg_members(wg_id);
CREATE INDEX IF NOT EXISTS idx_wg_members_user_id ON wg_members(user_id);
CREATE INDEX IF NOT EXISTS idx_wg_members_unit_id ON wg_members(unit_id);

-- ============================================================================
-- 15. time_entries
-- ============================================================================
CREATE TABLE IF NOT EXISTS time_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    user_id UUID NOT NULL REFERENCES users(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    subproject_id UUID NOT NULL REFERENCES subprojects(id),
    wg_id UUID NOT NULL REFERENCES working_groups(id),
    unit_id UUID NOT NULL REFERENCES units(id),
    hours DECIMAL(8,2) NOT NULL CHECK (hours > 0),
    description TEXT NOT NULL,
    entry_date TIMESTAMPTZ NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('draft', 'submitted', 'approved')) DEFAULT 'draft',
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_from_entry_id UUID REFERENCES time_entries(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_time_entries_org_id ON time_entries(org_id);
CREATE INDEX IF NOT EXISTS idx_time_entries_user_id ON time_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_time_entries_project_id ON time_entries(project_id);
CREATE INDEX IF NOT EXISTS idx_time_entries_wg_id ON time_entries(wg_id);
CREATE INDEX IF NOT EXISTS idx_time_entries_unit_id ON time_entries(unit_id);
CREATE INDEX IF NOT EXISTS idx_time_entries_status ON time_entries(status);
CREATE INDEX IF NOT EXISTS idx_time_entries_entry_date ON time_entries(entry_date);
CREATE INDEX IF NOT EXISTS idx_time_entries_user_date ON time_entries(user_id, entry_date);
CREATE INDEX IF NOT EXISTS idx_time_entries_is_deleted ON time_entries(is_deleted);

-- ============================================================================
-- 16. time_entry_approvals
-- ============================================================================
CREATE TABLE IF NOT EXISTS time_entry_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    time_entry_id UUID NOT NULL REFERENCES time_entries(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    action VARCHAR(50) NOT NULL CHECK (action IN ('submit', 'approve', 'reject', 'edit_approve', 'edit_return', 'partial_approve', 'delegate')),
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_time_entry_approvals_time_entry_id ON time_entry_approvals(time_entry_id);
CREATE INDEX IF NOT EXISTS idx_time_entry_approvals_user_id ON time_entry_approvals(user_id);

-- ============================================================================
-- 17. expenses
-- ============================================================================
CREATE TABLE IF NOT EXISTS expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    user_id UUID NOT NULL REFERENCES users(id),
    project_id UUID REFERENCES projects(id),
    customer_id UUID REFERENCES customers(id),
    unit_id UUID NOT NULL REFERENCES units(id),
    category VARCHAR(50) NOT NULL CHECK (category IN ('mileage', 'meal', 'accommodation', 'parking', 'travel_tickets', 'tolls', 'taxi', 'equipment', 'other')),
    amount DECIMAL(12,2) NOT NULL CHECK (amount > 0),
    km_distance DECIMAL(10,2),
    currency VARCHAR(10) NOT NULL DEFAULT 'EUR',
    description TEXT,
    expense_date TIMESTAMPTZ NOT NULL,
    receipt_url VARCHAR(500),
    receipt_ocr_data JSONB,
    status VARCHAR(50) NOT NULL CHECK (status IN ('draft', 'submitted', 'approved', 'rejected')) DEFAULT 'draft',
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_expenses_org_id ON expenses(org_id);
CREATE INDEX IF NOT EXISTS idx_expenses_user_id ON expenses(user_id);
CREATE INDEX IF NOT EXISTS idx_expenses_project_id ON expenses(project_id);
CREATE INDEX IF NOT EXISTS idx_expenses_unit_id ON expenses(unit_id);
CREATE INDEX IF NOT EXISTS idx_expenses_status ON expenses(status);
CREATE INDEX IF NOT EXISTS idx_expenses_expense_date ON expenses(expense_date);
CREATE INDEX IF NOT EXISTS idx_expenses_user_date ON expenses(user_id, expense_date);

-- ============================================================================
-- 18. expense_approvals
-- ============================================================================
CREATE TABLE IF NOT EXISTS expense_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    expense_id UUID NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    action VARCHAR(50) NOT NULL CHECK (action IN ('submit', 'approve', 'reject', 'edit_approve', 'edit_return', 'partial_approve', 'delegate')),
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_expense_approvals_expense_id ON expense_approvals(expense_id);
CREATE INDEX IF NOT EXISTS idx_expense_approvals_user_id ON expense_approvals(user_id);

-- ============================================================================
-- 19. invitations
-- ============================================================================
CREATE TABLE IF NOT EXISTS invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    code VARCHAR(255) NOT NULL UNIQUE,
    invite_token VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'expired')),
    created_by UUID NOT NULL REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invitations_organization_id ON invitations(organization_id);

-- ============================================================================
-- 20. password_resets
-- ============================================================================
CREATE TABLE IF NOT EXISTS password_resets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_resets_user_id ON password_resets(user_id);

-- ============================================================================
-- 21. refresh_tokens
-- ============================================================================
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- ============================================================================
-- 22. financial_cutoff_periods
-- ============================================================================
CREATE TABLE IF NOT EXISTS financial_cutoff_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    project_id UUID REFERENCES projects(id),
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    cutoff_date TIMESTAMPTZ NOT NULL,
    is_locked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_financial_cutoff_periods_org_id ON financial_cutoff_periods(org_id);
CREATE INDEX IF NOT EXISTS idx_financial_cutoff_periods_project_id ON financial_cutoff_periods(project_id);
CREATE INDEX IF NOT EXISTS idx_financial_cutoff_periods_dates ON financial_cutoff_periods(period_start, period_end);

-- ============================================================================
-- 23. budget_caps
-- ============================================================================
CREATE TABLE IF NOT EXISTS budget_caps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    user_id UUID REFERENCES users(id),
    project_id UUID REFERENCES projects(id),
    category VARCHAR(50) CHECK (category IN ('mileage', 'meal', 'accommodation', 'other')),
    limit_amount DECIMAL(12,2) NOT NULL CHECK (limit_amount > 0),
    period VARCHAR(50) NOT NULL CHECK (period IN ('daily', 'weekly', 'monthly', 'yearly')) DEFAULT 'monthly',
    currency VARCHAR(10) NOT NULL DEFAULT 'EUR',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_budget_caps_org_id ON budget_caps(org_id);
CREATE INDEX IF NOT EXISTS idx_budget_caps_user_id ON budget_caps(user_id);
CREATE INDEX IF NOT EXISTS idx_budget_caps_project_id ON budget_caps(project_id);

-- ============================================================================
-- 24. backup_approvers
-- ============================================================================
CREATE TABLE IF NOT EXISTS backup_approvers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('employee', 'manager', 'finance', 'customer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_backup_approvers_org_id ON backup_approvers(org_id);
CREATE INDEX IF NOT EXISTS idx_backup_approvers_user_id ON backup_approvers(user_id);

-- ============================================================================
-- 25. organization_settings
-- ============================================================================
CREATE TABLE IF NOT EXISTS organization_settings (
    organization_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    default_km_rate DECIMAL(10,2),
    currency VARCHAR(10) NOT NULL DEFAULT 'EUR',
    week_start_day SMALLINT NOT NULL DEFAULT 1,
    timezone VARCHAR(100) NOT NULL DEFAULT 'UTC',
    show_approval_history BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auto-create settings row when a new organization is created
CREATE OR REPLACE FUNCTION auto_create_org_settings()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO organization_settings (organization_id, currency, week_start_day, timezone, show_approval_history, created_at, updated_at)
    VALUES (NEW.id, 'EUR', 1, 'UTC', TRUE, NOW(), NOW());
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_auto_create_org_settings ON organizations;
CREATE TRIGGER trg_auto_create_org_settings
AFTER INSERT ON organizations
FOR EACH ROW
EXECUTE FUNCTION auto_create_org_settings();
