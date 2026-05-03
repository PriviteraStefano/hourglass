-- ============================================================================
-- ORGANIZATION LAYER SCHEMA
-- ============================================================================

-- Organizations (top-level tenant)
DEFINE TABLE organizations SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.org_id == id
        FOR create WHERE $auth.role == 'super_admin'
        FOR update WHERE $auth.role == 'super_admin' OR $auth.org_id == id
        FOR delete WHERE $auth.role == 'super_admin'
;

DEFINE FIELD id ON TABLE organizations TYPE uuid;
DEFINE FIELD name ON TABLE organizations TYPE string ASSERT $value != NONE;
DEFINE FIELD slug ON TABLE organizations TYPE string ASSERT $value != NONE;
DEFINE FIELD description ON TABLE organizations TYPE option<string>;
DEFINE FIELD financial_cutoff_days ON TABLE organizations TYPE option<number>;
DEFINE FIELD financial_cutoff_config ON TABLE organizations TYPE option<object>;
DEFINE FIELD financial_cutoff_config.cutoff_day_of_month ON TABLE organizations TYPE int;
DEFINE FIELD financial_cutoff_config.grace_days ON TABLE organizations TYPE int;
DEFINE FIELD created_at ON TABLE organizations TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON TABLE organizations TYPE datetime DEFAULT time::now();

DEFINE INDEX org_slug ON TABLE organizations COLUMNS slug UNIQUE;

-- Units (org hierarchy with unlimited nesting)
DEFINE TABLE units SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.org_id == org_id
        FOR create WHERE $auth.org_id == org_id
        FOR update WHERE $auth.org_id == org_id
        FOR delete WHERE $auth.org_id == org_id
;

DEFINE FIELD id ON TABLE units TYPE string;
DEFINE FIELD org_id ON TABLE units TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD name ON TABLE units TYPE string ASSERT $value != NONE;
DEFINE FIELD description ON TABLE units TYPE option<string>;
DEFINE FIELD parent_unit_id ON TABLE units TYPE option<record<units>>;
DEFINE FIELD hierarchy_level ON TABLE units TYPE number DEFAULT 0;
DEFINE FIELD code ON TABLE units TYPE option<string>;
DEFINE FIELD created_at ON TABLE units TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON TABLE units TYPE datetime DEFAULT time::now();

DEFINE INDEX unit_org ON TABLE units COLUMNS org_id;
DEFINE INDEX unit_parent ON TABLE units COLUMNS parent_unit_id;
DEFINE INDEX unit_org_code ON TABLE units COLUMNS org_id, code UNIQUE;

-- Users (cross-org users)
DEFINE TABLE users SCHEMAFULL
    PERMISSIONS
        FOR select WHERE true
        FOR create WHERE $auth.role == 'super_admin'
        FOR update WHERE $auth.id == id OR $auth.role == 'super_admin'
        FOR delete WHERE $auth.role == 'super_admin'
;

DEFINE FIELD id ON TABLE users TYPE uuid;
DEFINE FIELD email ON TABLE users TYPE string ASSERT $value != NONE;
DEFINE FIELD name ON TABLE users TYPE string ASSERT $value != NONE;
DEFINE FIELD password_hash ON TABLE users TYPE option<string>;
DEFINE FIELD is_active ON TABLE users TYPE bool DEFAULT true;
DEFINE FIELD created_at ON TABLE users TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON TABLE users TYPE datetime DEFAULT time::now();

DEFINE INDEX user_email ON TABLE users COLUMNS email UNIQUE;

-- Organization Memberships (users → organizations, many-to-many)
DEFINE TABLE organization_memberships SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.role == 'super_admin' OR $auth.org_id == organization_id
        FOR create WHERE $auth.role == 'super_admin' OR $auth.org_id == organization_id
        FOR update WHERE $auth.role == 'super_admin' OR $auth.org_id == organization_id
        FOR delete WHERE $auth.role == 'super_admin' OR $auth.org_id == organization_id
;

DEFINE FIELD id ON TABLE organization_memberships TYPE uuid;
DEFINE FIELD user_id ON TABLE organization_memberships TYPE option<record<users>>;
DEFINE FIELD organization_id ON TABLE organization_memberships TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD role ON TABLE organization_memberships TYPE string ASSERT $value IN ['employee', 'manager', 'finance', 'customer'] DEFAULT 'employee';
DEFINE FIELD is_active ON TABLE organization_memberships TYPE bool DEFAULT true;
DEFINE FIELD invited_by ON TABLE organization_memberships TYPE option<record<users>>;
DEFINE FIELD invited_at ON TABLE organization_memberships TYPE option<datetime>;
DEFINE FIELD activated_at ON TABLE organization_memberships TYPE option<datetime>;
DEFINE FIELD created_at ON TABLE organization_memberships TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON TABLE organization_memberships TYPE datetime DEFAULT time::now();

DEFINE INDEX om_user ON TABLE organization_memberships COLUMNS user_id;
DEFINE INDEX om_org ON TABLE organization_memberships COLUMNS organization_id;
DEFINE INDEX om_user_org ON TABLE organization_memberships COLUMNS user_id, organization_id UNIQUE;

-- Unit Memberships (users → units, many-to-many)
DEFINE TABLE unit_memberships SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.org_id == org_id
        FOR create WHERE $auth.org_id == org_id
        FOR update WHERE $auth.org_id == org_id
        FOR delete WHERE $auth.org_id == org_id
;

DEFINE FIELD id ON TABLE unit_memberships TYPE string;
DEFINE FIELD org_id ON TABLE unit_memberships TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD user_id ON TABLE unit_memberships TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD unit_id ON TABLE unit_memberships TYPE record<units> ASSERT $value != NONE;
DEFINE FIELD is_primary ON TABLE unit_memberships TYPE bool DEFAULT false;
DEFINE FIELD role ON TABLE unit_memberships TYPE string DEFAULT 'employee';
DEFINE FIELD start_date ON TABLE unit_memberships TYPE datetime DEFAULT time::now();
DEFINE FIELD end_date ON TABLE unit_memberships TYPE option<datetime>;
DEFINE FIELD created_at ON TABLE unit_memberships TYPE datetime DEFAULT time::now();

DEFINE INDEX um_org ON TABLE unit_memberships COLUMNS org_id;
DEFINE INDEX um_user ON TABLE unit_memberships COLUMNS user_id;
DEFINE INDEX um_unit ON TABLE unit_memberships COLUMNS unit_id;
DEFINE INDEX um_user_org ON TABLE unit_memberships COLUMNS user_id, org_id;
DEFINE INDEX um_primary ON TABLE unit_memberships COLUMNS user_id, is_primary;

-- Verification Tokens (email verification, password reset)
DEFINE TABLE verification_tokens SCHEMAFULL;

DEFINE FIELD id ON TABLE verification_tokens TYPE string;
DEFINE FIELD user_id ON TABLE verification_tokens TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD token ON TABLE verification_tokens TYPE string ASSERT $value != NONE;
DEFINE FIELD type ON TABLE verification_tokens TYPE string ASSERT $value IN ['email_verification', 'password_reset'];
DEFINE FIELD expires_at ON TABLE verification_tokens TYPE datetime ASSERT $value != NONE;
DEFINE FIELD created_at ON TABLE verification_tokens TYPE datetime DEFAULT time::now();

DEFINE INDEX vt_token ON TABLE verification_tokens COLUMNS token UNIQUE;
DEFINE INDEX vt_user ON TABLE verification_tokens COLUMNS user_id;

-- Refresh Tokens (JWT refresh)
DEFINE TABLE refresh_tokens SCHEMAFULL;

DEFINE FIELD id ON TABLE refresh_tokens TYPE string;
DEFINE FIELD user_id ON TABLE refresh_tokens TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD organization_id ON TABLE refresh_tokens TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD token_hash ON TABLE refresh_tokens TYPE string ASSERT $value != NONE;
DEFINE FIELD expires_at ON TABLE refresh_tokens TYPE datetime ASSERT $value != NONE;
DEFINE FIELD revoked_at ON TABLE refresh_tokens TYPE option<datetime>;
DEFINE FIELD created_at ON TABLE refresh_tokens TYPE datetime DEFAULT time::now();

DEFINE INDEX rt_token_hash ON TABLE refresh_tokens COLUMNS token_hash UNIQUE;
DEFINE INDEX rt_user ON TABLE refresh_tokens COLUMNS user_id;

-- ============================================================================
-- PROJECT LAYER SCHEMA
-- ============================================================================

-- Customers (for project billing tracking)
DEFINE TABLE customers SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.org_id == org_id
        FOR create WHERE $auth.org_id == org_id
        FOR update WHERE $auth.org_id == org_id
        FOR delete WHERE $auth.org_id == org_id
;

DEFINE FIELD id ON TABLE customers TYPE string;
DEFINE FIELD org_id ON TABLE customers TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD name ON TABLE customers TYPE string ASSERT $value != NONE;
DEFINE FIELD email ON TABLE customers TYPE option<string>;
DEFINE FIELD address ON TABLE customers TYPE option<string>;
DEFINE FIELD vat_number ON TABLE customers TYPE option<string>;
DEFINE FIELD is_active ON TABLE customers TYPE bool DEFAULT true;
DEFINE FIELD created_at ON TABLE customers TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON TABLE customers TYPE datetime DEFAULT time::now();

DEFINE INDEX cust_org ON TABLE customers COLUMNS org_id;

-- Projects (planning level)
DEFINE TABLE projects SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.org_id == org_id
        FOR create WHERE $auth.org_id == org_id
        FOR update WHERE $auth.org_id == org_id
        FOR delete WHERE $auth.org_id == org_id
;

DEFINE FIELD id ON TABLE projects TYPE string;
DEFINE FIELD org_id ON TABLE projects TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD name ON TABLE projects TYPE string ASSERT $value != NONE;
DEFINE FIELD description ON TABLE projects TYPE option<string>;
DEFINE FIELD project_type ON TABLE projects TYPE string ASSERT $value IN ['billable', 'internal'];
DEFINE FIELD customer_id ON TABLE projects TYPE option<record<customers>>;
DEFINE FIELD budget_amount ON TABLE projects TYPE option<number>;
DEFINE FIELD financial_cutoff_config ON TABLE projects TYPE option<object>;
DEFINE FIELD is_active ON TABLE projects TYPE bool DEFAULT true;
DEFINE FIELD created_at ON TABLE projects TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON TABLE projects TYPE datetime DEFAULT time::now();

DEFINE INDEX project_org ON TABLE projects COLUMNS org_id;
DEFINE INDEX project_customer ON TABLE projects COLUMNS customer_id;

-- Subprojects (structure level)
DEFINE TABLE subprojects SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.org_id == (SELECT VALUE org_id FROM $value.project_id)[0]
        FOR create WHERE $auth.org_id == (SELECT VALUE org_id FROM $value.project_id)[0]
        FOR update WHERE $auth.org_id == (SELECT VALUE org_id FROM $value.project_id)[0]
        FOR delete WHERE $auth.org_id == (SELECT VALUE org_id FROM $value.project_id)[0]
;

DEFINE FIELD id ON TABLE subprojects TYPE string;
DEFINE FIELD project_id ON TABLE subprojects TYPE record<projects> ASSERT $value != NONE;
DEFINE FIELD name ON TABLE subprojects TYPE string ASSERT $value != NONE;
DEFINE FIELD description ON TABLE subprojects TYPE option<string>;
DEFINE FIELD sequence_order ON TABLE subprojects TYPE number DEFAULT 0;
DEFINE FIELD is_active ON TABLE subprojects TYPE bool DEFAULT true;
DEFINE FIELD created_at ON TABLE subprojects TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON TABLE subprojects TYPE datetime DEFAULT time::now();

DEFINE INDEX subproject_project ON TABLE subprojects COLUMNS project_id;

-- Working Groups (execution teams)
DEFINE TABLE working_groups SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.org_id == org_id
        FOR create WHERE $auth.org_id == org_id
        FOR update WHERE $auth.org_id == org_id
        FOR delete WHERE $auth.org_id == org_id
;

DEFINE FIELD id ON TABLE working_groups TYPE string;
DEFINE FIELD org_id ON TABLE working_groups TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD subproject_id ON TABLE working_groups TYPE record<subprojects> ASSERT $value != NONE;
DEFINE FIELD name ON TABLE working_groups TYPE string ASSERT $value != NONE;
DEFINE FIELD description ON TABLE working_groups TYPE option<string>;
DEFINE FIELD unit_ids ON TABLE working_groups TYPE array<record<units>> DEFAULT [];
DEFINE FIELD enforce_unit_tuple ON TABLE working_groups TYPE bool DEFAULT true;
DEFINE FIELD manager_id ON TABLE working_groups TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD delegate_ids ON TABLE working_groups TYPE array<record<users>> DEFAULT [];
DEFINE FIELD is_active ON TABLE working_groups TYPE bool DEFAULT true;
DEFINE FIELD created_at ON TABLE working_groups TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON TABLE working_groups TYPE datetime DEFAULT time::now();

DEFINE INDEX wg_org ON TABLE working_groups COLUMNS org_id;
DEFINE INDEX wg_subproject ON TABLE working_groups COLUMNS subproject_id;
DEFINE INDEX wg_manager ON TABLE working_groups COLUMNS manager_id;

-- Working Group Members (users assigned to WG from units)
DEFINE TABLE wg_members SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.org_id == (SELECT VALUE org_id FROM $value.wg_id)[0]
        FOR create WHERE $auth.org_id == (SELECT VALUE org_id FROM $value.wg_id)[0]
        FOR update WHERE $auth.org_id == (SELECT VALUE org_id FROM $value.wg_id)[0]
        FOR delete WHERE $auth.org_id == (SELECT VALUE org_id FROM $value.wg_id)[0]
;

DEFINE FIELD id ON TABLE wg_members TYPE string;
DEFINE FIELD wg_id ON TABLE wg_members TYPE record<working_groups> ASSERT $value != NONE;
DEFINE FIELD user_id ON TABLE wg_members TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD unit_id ON TABLE wg_members TYPE record<units> ASSERT $value != NONE;
DEFINE FIELD role ON TABLE wg_members TYPE string DEFAULT 'member';
DEFINE FIELD is_default_subproject ON TABLE wg_members TYPE bool DEFAULT false;
DEFINE FIELD start_date ON TABLE wg_members TYPE datetime DEFAULT time::now();
DEFINE FIELD end_date ON TABLE wg_members TYPE option<datetime>;
DEFINE FIELD created_at ON TABLE wg_members TYPE datetime DEFAULT time::now();

DEFINE INDEX wgm_wg ON TABLE wg_members COLUMNS wg_id;
DEFINE INDEX wgm_user ON TABLE wg_members COLUMNS user_id;
DEFINE INDEX wgm_unit ON TABLE wg_members COLUMNS unit_id;
DEFINE INDEX wgm_wg_user ON TABLE wg_members COLUMNS wg_id, user_id UNIQUE;

-- ============================================================================
-- TIME ENTRY LAYER SCHEMA
-- ============================================================================

-- Time Entries (core time tracking records)
DEFINE TABLE time_entries SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.id == user_id OR $auth.role IN ['wg_manager', 'org_manager', 'finance', 'admin']
        FOR create WHERE $auth.id == user_id OR $auth.role IN ['wg_manager', 'admin']
        FOR update WHERE $auth.id == user_id OR $auth.role IN ['wg_manager', 'org_manager', 'finance', 'admin']
        FOR delete WHERE $auth.id == user_id OR $auth.role IN ['admin']
;

DEFINE FIELD id ON TABLE time_entries TYPE string;
DEFINE FIELD org_id ON TABLE time_entries TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD user_id ON TABLE time_entries TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD project_id ON TABLE time_entries TYPE record<projects> ASSERT $value != NONE;
DEFINE FIELD subproject_id ON TABLE time_entries TYPE record<subprojects> ASSERT $value != NONE;
DEFINE FIELD wg_id ON TABLE time_entries TYPE record<working_groups> ASSERT $value != NONE;
DEFINE FIELD unit_id ON TABLE time_entries TYPE record<units> ASSERT $value != NONE;
DEFINE FIELD hours ON TABLE time_entries TYPE number ASSERT $value > 0;
DEFINE FIELD description ON TABLE time_entries TYPE string ASSERT $value != NONE;
DEFINE FIELD entry_date ON TABLE time_entries TYPE datetime ASSERT $value != NONE;
DEFINE FIELD status ON TABLE time_entries TYPE string ASSERT $value IN ['draft', 'submitted', 'approved'] DEFAULT 'draft';
DEFINE FIELD is_deleted ON TABLE time_entries TYPE bool DEFAULT false;
DEFINE FIELD created_from_entry_id ON TABLE time_entries TYPE option<record<time_entries>>;
DEFINE FIELD created_at ON TABLE time_entries TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON TABLE time_entries TYPE datetime DEFAULT time::now();

DEFINE INDEX te_org ON TABLE time_entries COLUMNS org_id;
DEFINE INDEX te_user ON TABLE time_entries COLUMNS user_id;
DEFINE INDEX te_project ON TABLE time_entries COLUMNS project_id;
DEFINE INDEX te_wg ON TABLE time_entries COLUMNS wg_id;
DEFINE INDEX te_unit ON TABLE time_entries COLUMNS unit_id;
DEFINE INDEX te_status ON TABLE time_entries COLUMNS status;
DEFINE INDEX te_date ON TABLE time_entries COLUMNS entry_date;
DEFINE INDEX te_user_date ON TABLE time_entries COLUMNS user_id, entry_date;
DEFINE INDEX te_not_deleted ON TABLE time_entries COLUMNS is_deleted;

-- ============================================================================
-- AUDIT LAYER (immutable, single source of truth)
-- ============================================================================

-- Audit Log (shared for time entries and expenses)
DEFINE TABLE audit_logs SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.role IN ['admin', 'finance', 'org_manager'] OR $auth.org_id == org_id
        FOR create WHERE true
        FOR update WHERE false
        FOR delete WHERE false
;

DEFINE FIELD id ON TABLE audit_logs TYPE string;
DEFINE FIELD org_id ON TABLE audit_logs TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD entry_id ON TABLE audit_logs TYPE string ASSERT $value != NONE;
DEFINE FIELD entry_type ON TABLE audit_logs TYPE string ASSERT $value IN ['time_entry', 'expense'] AND $value != NONE;
DEFINE FIELD action ON TABLE audit_logs TYPE string ASSERT $value IN [
    'created', 'submitted', 'approved', 'rejected', 'split', 'moved',
    'reallocated', 'edited', 'finance_override', 'reverted'
] AND $value != NONE;
DEFINE FIELD actor_role ON TABLE audit_logs TYPE string ASSERT $value IN [
    'user', 'wg_manager', 'org_manager', 'finance', 'admin'
] AND $value != NONE;
DEFINE FIELD actor_id ON TABLE audit_logs TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD reason ON TABLE audit_logs TYPE option<string>;
DEFINE FIELD changes ON TABLE audit_logs TYPE option<object>;
DEFINE FIELD timestamp ON TABLE audit_logs TYPE datetime DEFAULT time::now();
DEFINE FIELD ip_address ON TABLE audit_logs TYPE option<string>;

DEFINE INDEX audit_entry ON TABLE audit_logs COLUMNS entry_id;
DEFINE INDEX audit_actor ON TABLE audit_logs COLUMNS actor_id;
DEFINE INDEX audit_actor_role ON TABLE audit_logs COLUMNS actor_role;
DEFINE INDEX audit_timestamp ON TABLE audit_logs COLUMNS timestamp;
DEFINE INDEX audit_org_timestamp ON TABLE audit_logs COLUMNS org_id, timestamp;
DEFINE INDEX audit_entry_type ON TABLE audit_logs COLUMNS entry_type;

-- ============================================================================
-- EXPENSE LAYER SCHEMA
-- ============================================================================

-- Expenses
DEFINE TABLE expenses SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.id == user_id OR $auth.role IN ['finance', 'admin']
        FOR create WHERE $auth.id == user_id OR $auth.role IN ['finance', 'admin']
        FOR update WHERE ($auth.id == user_id AND status == 'draft') OR $auth.role IN ['finance', 'admin']
        FOR delete WHERE false
;

DEFINE FIELD id ON TABLE expenses TYPE string;
DEFINE FIELD org_id ON TABLE expenses TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD user_id ON TABLE expenses TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD project_id ON TABLE expenses TYPE option<record<projects>>;
DEFINE FIELD unit_id ON TABLE expenses TYPE record<units> ASSERT $value != NONE;
DEFINE FIELD category ON TABLE expenses TYPE string ASSERT $value IN ['mileage', 'meal', 'accommodation', 'other'] AND $value != NONE;
DEFINE FIELD amount ON TABLE expenses TYPE number ASSERT $value > 0;
DEFINE FIELD currency ON TABLE expenses TYPE string DEFAULT 'EUR';
DEFINE FIELD description ON TABLE expenses TYPE option<string>;
DEFINE FIELD expense_date ON TABLE expenses TYPE datetime ASSERT $value != NONE;
DEFINE FIELD receipt_url ON TABLE expenses TYPE option<string>;
DEFINE FIELD receipt_ocr_data ON TABLE expenses TYPE option<object>;
DEFINE FIELD status ON TABLE expenses TYPE string ASSERT $value IN ['draft', 'submitted', 'approved', 'rejected'] DEFAULT 'draft';
DEFINE FIELD is_deleted ON TABLE expenses TYPE bool DEFAULT false;
DEFINE FIELD created_at ON TABLE expenses TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON TABLE expenses TYPE datetime DEFAULT time::now();

DEFINE INDEX exp_org ON TABLE expenses COLUMNS org_id;
DEFINE INDEX exp_user ON TABLE expenses COLUMNS user_id;
DEFINE INDEX exp_project ON TABLE expenses COLUMNS project_id;
DEFINE INDEX exp_unit ON TABLE expenses COLUMNS unit_id;
DEFINE INDEX exp_status ON TABLE expenses COLUMNS status;
DEFINE INDEX exp_date ON TABLE expenses COLUMNS expense_date;
DEFINE INDEX exp_user_date ON TABLE expenses COLUMNS user_id, expense_date;

-- ============================================================================
-- CONFIGURATION LAYER
-- ============================================================================

-- Financial Cutoff Periods (organization-level config)
DEFINE TABLE financial_cutoff_periods SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.org_id == org_id
        FOR create WHERE $auth.org_id == org_id
        FOR update WHERE $auth.org_id == org_id
        FOR delete WHERE $auth.org_id == org_id
;

DEFINE FIELD id ON TABLE financial_cutoff_periods TYPE string;
DEFINE FIELD org_id ON TABLE financial_cutoff_periods TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD project_id ON TABLE financial_cutoff_periods TYPE option<record<projects>>;
DEFINE FIELD period_start ON TABLE financial_cutoff_periods TYPE datetime ASSERT $value != NONE;
DEFINE FIELD period_end ON TABLE financial_cutoff_periods TYPE datetime ASSERT $value != NONE;
DEFINE FIELD cutoff_date ON TABLE financial_cutoff_periods TYPE datetime ASSERT $value != NONE;
DEFINE FIELD is_locked ON TABLE financial_cutoff_periods TYPE bool DEFAULT false;
DEFINE FIELD created_at ON TABLE financial_cutoff_periods TYPE datetime DEFAULT time::now();

DEFINE INDEX fcp_org ON TABLE financial_cutoff_periods COLUMNS org_id;
DEFINE INDEX fcp_project ON TABLE financial_cutoff_periods COLUMNS project_id;
DEFINE INDEX fcp_dates ON TABLE financial_cutoff_periods COLUMNS period_start, period_end;

-- Budget Caps (expense limits per user/project/category)
DEFINE TABLE budget_caps SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.org_id == org_id
        FOR create WHERE $auth.org_id == org_id
        FOR update WHERE $auth.org_id == org_id
        FOR delete WHERE $auth.org_id == org_id
;

DEFINE FIELD id ON TABLE budget_caps TYPE string;
DEFINE FIELD org_id ON TABLE budget_caps TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD user_id ON TABLE budget_caps TYPE option<record<users>>;
DEFINE FIELD project_id ON TABLE budget_caps TYPE option<record<projects>>;
DEFINE FIELD category ON TABLE budget_caps TYPE option<string>;
DEFINE FIELD limit_amount ON TABLE budget_caps TYPE number ASSERT $value > 0;
DEFINE FIELD period ON TABLE budget_caps TYPE string ASSERT $value IN ['daily', 'weekly', 'monthly', 'yearly'] DEFAULT 'monthly';
DEFINE FIELD currency ON TABLE budget_caps TYPE string DEFAULT 'EUR';
DEFINE FIELD created_at ON TABLE budget_caps TYPE datetime DEFAULT time::now();

DEFINE INDEX bc_org ON TABLE budget_caps COLUMNS org_id;
DEFINE INDEX bc_user ON TABLE budget_caps COLUMNS user_id;
DEFINE INDEX bc_project ON TABLE budget_caps COLUMNS project_id;

-- ============================================================================
-- USER FIELDS: username, firstname, lastname
-- ============================================================================

DEFINE FIELD username ON TABLE users TYPE option<string>;
DEFINE FIELD firstname ON TABLE users TYPE option<string>;
DEFINE FIELD lastname ON TABLE users TYPE option<string>;

DEFINE INDEX user_username ON TABLE users COLUMNS username UNIQUE;

-- ============================================================================
-- INVITATIONS TABLE
-- ============================================================================

DEFINE TABLE invitations SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.role == 'admin' OR $auth.role == 'manager'
        FOR create WHERE $auth.role == 'admin' OR $auth.role == 'manager'
        FOR delete WHERE $auth.role == 'admin'
;

DEFINE FIELD id ON invitations TYPE string;
DEFINE FIELD organization_id ON invitations TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD code ON invitations TYPE string ASSERT $value != NONE;
DEFINE FIELD invite_token ON invitations TYPE string ASSERT $value != NONE;
DEFINE FIELD email ON invitations TYPE option<string>;
DEFINE FIELD status ON invitations TYPE string DEFAULT 'pending' ASSERT $value IN ['pending', 'accepted', 'expired'];
DEFINE FIELD expires_at ON invitations TYPE datetime ASSERT $value != NONE;
DEFINE FIELD created_by ON invitations TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD created_at ON invitations TYPE datetime DEFAULT time::now();

DEFINE INDEX invite_code ON invitations COLUMNS code UNIQUE;
DEFINE INDEX invite_token ON invitations COLUMNS invite_token UNIQUE;
DEFINE INDEX invite_org ON invitations COLUMNS organization_id;

-- ============================================================================
-- PASSWORD RESETS TABLE
-- ============================================================================

DEFINE TABLE password_resets SCHEMAFULL;

DEFINE FIELD id ON password_resets TYPE string;
DEFINE FIELD user_id ON password_resets TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD code_hash ON password_resets TYPE string ASSERT $value != NONE;
DEFINE FIELD expires_at ON password_resets TYPE datetime ASSERT $value != NONE;
DEFINE FIELD used_at ON password_resets TYPE option<datetime>;
DEFINE FIELD created_at ON password_resets TYPE datetime DEFAULT time::now();

DEFINE INDEX pr_user ON password_resets COLUMNS user_id;
