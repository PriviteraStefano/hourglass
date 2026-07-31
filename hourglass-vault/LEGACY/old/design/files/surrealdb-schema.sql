-- ============================================================================
-- SURREALDB SCHEMA: Hourglass Time Tracking & Approval Workflow
-- ============================================================================
-- Comprehensive SurrealDB schema implementing the grill-me architecture
-- Features: Org hierarchy, time entries, expenses, audit trail, financial cutoff
-- Last Updated: 2026-04-12
-- ============================================================================

-- ============================================================================
-- ORGANIZATION LAYER
-- ============================================================================

-- Organizations (top-level tenant)
DEFINE TABLE organizations SCHEMAFULL
  PERMISSIONS
    FOR select ALLOW $auth.org_id == id
    FOR create ALLOW $auth.role == 'super_admin'
    FOR update ALLOW $auth.role == 'super_admin' OR (
      $auth.org_id == id AND $auth.role IN ['admin']
    )
    FOR delete ALLOW $auth.role == 'super_admin'
;

DEFINE FIELD id ON TABLE organizations TYPE string;
DEFINE FIELD name ON TABLE organizations TYPE string;
DEFINE FIELD slug ON TABLE organizations TYPE string ASSERT $value != NONE;
DEFINE FIELD description ON TABLE organizations TYPE string;
DEFINE FIELD created_at ON TABLE organizations TYPE datetime DEFAULT now();
DEFINE FIELD updated_at ON TABLE organizations TYPE datetime DEFAULT now();
DEFINE FIELD financial_cutoff_days ON TABLE organizations TYPE number DEFAULT 7;
DEFINE FIELD financial_cutoff_config ON TABLE organizations TYPE object DEFAULT {
  cutoff_day_of_month: 28,
  grace_days: 7
};

DEFINE INDEX org_slug ON TABLE organizations COLUMNS slug UNIQUE;

-- Units (org hierarchy with unlimited nesting)
DEFINE TABLE units SCHEMAFULL
  PERMISSIONS
    FOR select ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR create ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR update ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR delete ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
;

DEFINE FIELD id ON TABLE units TYPE string;
DEFINE FIELD org_id ON TABLE units TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD name ON TABLE units TYPE string ASSERT $value != NONE;
DEFINE FIELD description ON TABLE units TYPE string;
DEFINE FIELD parent_unit_id ON TABLE units TYPE option<record<units>>;
DEFINE FIELD hierarchy_level ON TABLE units TYPE number DEFAULT 0;
DEFINE FIELD code ON TABLE units TYPE string;
DEFINE FIELD created_at ON TABLE units TYPE datetime DEFAULT now();
DEFINE FIELD updated_at ON TABLE units TYPE datetime DEFAULT now();

DEFINE INDEX unit_org ON TABLE units COLUMNS org_id;
DEFINE INDEX unit_parent ON TABLE units COLUMNS parent_unit_id;
DEFINE INDEX unit_org_code ON TABLE units COLUMNS org_id, code UNIQUE;

-- Users (cross-org users with org membership)
DEFINE TABLE users SCHEMAFULL
  PERMISSIONS
    FOR select ALLOW true
    FOR create ALLOW $auth.role == 'super_admin'
    FOR update ALLOW $auth.id == id OR $auth.role == 'super_admin'
    FOR delete ALLOW $auth.role == 'super_admin'
;

DEFINE FIELD id ON TABLE users TYPE string;
DEFINE FIELD email ON TABLE users TYPE string ASSERT $value != NONE;
DEFINE FIELD name ON TABLE users TYPE string ASSERT $value != NONE;
DEFINE FIELD password_hash ON TABLE users TYPE string;
DEFINE FIELD created_at ON TABLE users TYPE datetime DEFAULT now();
DEFINE FIELD updated_at ON TABLE users TYPE datetime DEFAULT now();

DEFINE INDEX user_email ON TABLE users COLUMNS email UNIQUE;

-- Unit Memberships (users → units, many-to-many)
DEFINE TABLE unit_memberships SCHEMAFULL
  PERMISSIONS
    FOR select ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR create ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR update ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR delete ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
;

DEFINE FIELD id ON TABLE unit_memberships TYPE string;
DEFINE FIELD org_id ON TABLE unit_memberships TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD user_id ON TABLE unit_memberships TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD unit_id ON TABLE unit_memberships TYPE record<units> ASSERT $value != NONE;
DEFINE FIELD is_primary ON TABLE unit_memberships TYPE bool DEFAULT false;
DEFINE FIELD role ON TABLE unit_memberships TYPE string DEFAULT 'employee';
DEFINE FIELD start_date ON TABLE unit_memberships TYPE datetime DEFAULT now();
DEFINE FIELD end_date ON TABLE unit_memberships TYPE option<datetime>;
DEFINE FIELD created_at ON TABLE unit_memberships TYPE datetime DEFAULT now();

DEFINE INDEX um_org ON TABLE unit_memberships COLUMNS org_id;
DEFINE INDEX um_user ON TABLE unit_memberships COLUMNS user_id;
DEFINE INDEX um_unit ON TABLE unit_memberships COLUMNS unit_id;
DEFINE INDEX um_user_org ON TABLE unit_memberships COLUMNS user_id, org_id;
DEFINE INDEX um_primary ON TABLE unit_memberships COLUMNS user_id, is_primary WHERE is_primary == true;

-- ============================================================================
-- PROJECT LAYER (3-level: Project → Subproject → WorkingGroup)
-- ============================================================================

-- Projects (planning level)
DEFINE TABLE projects SCHEMAFULL
  PERMISSIONS
    FOR select ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR create ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR update ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR delete ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
;

DEFINE FIELD id ON TABLE projects TYPE string;
DEFINE FIELD org_id ON TABLE projects TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD name ON TABLE projects TYPE string ASSERT $value != NONE;
DEFINE FIELD description ON TABLE projects TYPE string;
DEFINE FIELD project_type ON TABLE projects TYPE string ENUM ['billable', 'internal'];
DEFINE FIELD customer_id ON TABLE projects TYPE option<record<customers>>;
DEFINE FIELD budget_amount ON TABLE projects TYPE option<number>;
DEFINE FIELD financial_cutoff_config ON TABLE projects TYPE option<object>;
DEFINE FIELD created_at ON TABLE projects TYPE datetime DEFAULT now();
DEFINE FIELD updated_at ON TABLE projects TYPE datetime DEFAULT now();

DEFINE INDEX project_org ON TABLE projects COLUMNS org_id;
DEFINE INDEX project_customer ON TABLE projects COLUMNS customer_id;

-- Subprojects (structure level)
DEFINE TABLE subprojects SCHEMAFULL
  PERMISSIONS
    FOR select ALLOW (
      SELECT id FROM organizations WHERE id == (
        SELECT org_id FROM projects WHERE id == (
          SELECT project_id FROM subprojects WHERE id == $value.id
        )
      )
    )
    FOR create ALLOW true
    FOR update ALLOW true
    FOR delete ALLOW true
;

DEFINE FIELD id ON TABLE subprojects TYPE string;
DEFINE FIELD project_id ON TABLE subprojects TYPE record<projects> ASSERT $value != NONE;
DEFINE FIELD name ON TABLE subprojects TYPE string ASSERT $value != NONE;
DEFINE FIELD description ON TABLE subprojects TYPE string;
DEFINE FIELD sequence_order ON TABLE subprojects TYPE number DEFAULT 0;
DEFINE FIELD created_at ON TABLE subprojects TYPE datetime DEFAULT now();
DEFINE FIELD updated_at ON TABLE subprojects TYPE datetime DEFAULT now();

DEFINE INDEX subproject_project ON TABLE subprojects COLUMNS project_id;

-- Working Groups (execution teams)
DEFINE TABLE working_groups SCHEMAFULL
  PERMISSIONS
    FOR select ALLOW true
    FOR create ALLOW true
    FOR update ALLOW true
    FOR delete ALLOW true
;

DEFINE FIELD id ON TABLE working_groups TYPE string;
DEFINE FIELD org_id ON TABLE working_groups TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD subproject_id ON TABLE working_groups TYPE record<subprojects> ASSERT $value != NONE;
DEFINE FIELD name ON TABLE working_groups TYPE string ASSERT $value != NONE;
DEFINE FIELD description ON TABLE working_groups TYPE string;
DEFINE FIELD unit_ids ON TABLE working_groups TYPE array<record<units>> DEFAULT [];
DEFINE FIELD enforce_unit_tuple ON TABLE working_groups TYPE bool DEFAULT true;
DEFINE FIELD manager_id ON TABLE working_groups TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD delegate_ids ON TABLE working_groups TYPE array<record<users>> DEFAULT [];
DEFINE FIELD created_at ON TABLE working_groups TYPE datetime DEFAULT now();
DEFINE FIELD updated_at ON TABLE working_groups TYPE datetime DEFAULT now();

DEFINE INDEX wg_org ON TABLE working_groups COLUMNS org_id;
DEFINE INDEX wg_subproject ON TABLE working_groups COLUMNS subproject_id;
DEFINE INDEX wg_manager ON TABLE working_groups COLUMNS manager_id;

-- Working Group Members (users assigned to WG from units)
DEFINE TABLE wg_members SCHEMAFULL
  PERMISSIONS
    FOR select ALLOW true
    FOR create ALLOW true
    FOR update ALLOW true
    FOR delete ALLOW true
;

DEFINE FIELD id ON TABLE wg_members TYPE string;
DEFINE FIELD wg_id ON TABLE wg_members TYPE record<working_groups> ASSERT $value != NONE;
DEFINE FIELD user_id ON TABLE wg_members TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD unit_id ON TABLE wg_members TYPE record<units> ASSERT $value != NONE;
DEFINE FIELD role ON TABLE wg_members TYPE string DEFAULT 'member';
DEFINE FIELD is_default_subproject ON TABLE wg_members TYPE bool DEFAULT false;
DEFINE FIELD start_date ON TABLE wg_members TYPE datetime DEFAULT now();
DEFINE FIELD end_date ON TABLE wg_members TYPE option<datetime>;
DEFINE FIELD created_at ON TABLE wg_members TYPE datetime DEFAULT now();

DEFINE INDEX wgm_wg ON TABLE wg_members COLUMNS wg_id;
DEFINE INDEX wgm_user ON TABLE wg_members COLUMNS user_id;
DEFINE INDEX wgm_unit ON TABLE wg_members COLUMNS unit_id;
DEFINE INDEX wgm_wg_user ON TABLE wg_members COLUMNS wg_id, user_id UNIQUE;

-- ============================================================================
-- TIME ENTRY LAYER
-- ============================================================================

-- Time Entries (core time tracking records)
DEFINE TABLE time_entries SCHEMAFULL
  PERMISSIONS
    FOR select ALLOW (
      $auth.id == user_id OR
      $auth.role IN ['wg_manager', 'org_manager', 'finance', 'admin']
    )
    FOR create ALLOW (
      $auth.id == user_id OR
      $auth.role IN ['wg_manager', 'admin']
    )
    FOR update ALLOW (
      $auth.id == user_id OR
      $auth.role IN ['wg_manager', 'org_manager', 'finance', 'admin']
    )
    FOR delete ALLOW false
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
DEFINE FIELD entry_date ON TABLE time_entries TYPE date ASSERT $value != NONE;
DEFINE FIELD status ON TABLE time_entries TYPE string ENUM ['draft', 'submitted', 'approved'] DEFAULT 'draft';
DEFINE FIELD is_deleted ON TABLE time_entries TYPE bool DEFAULT false;
DEFINE FIELD created_from_entry_id ON TABLE time_entries TYPE option<record<time_entries>>;
DEFINE FIELD created_at ON TABLE time_entries TYPE datetime DEFAULT now();
DEFINE FIELD updated_at ON TABLE time_entries TYPE datetime DEFAULT now();

DEFINE INDEX te_org ON TABLE time_entries COLUMNS org_id;
DEFINE INDEX te_user ON TABLE time_entries COLUMNS user_id;
DEFINE INDEX te_project ON TABLE time_entries COLUMNS project_id;
DEFINE INDEX te_wg ON TABLE time_entries COLUMNS wg_id;
DEFINE INDEX te_unit ON TABLE time_entries COLUMNS unit_id;
DEFINE INDEX te_status ON TABLE time_entries COLUMNS status;
DEFINE INDEX te_date ON TABLE time_entries COLUMNS entry_date;
DEFINE INDEX te_user_date ON TABLE time_entries COLUMNS user_id, entry_date;
DEFINE INDEX te_not_deleted ON TABLE time_entries COLUMNS is_deleted WHERE is_deleted == false;

-- ============================================================================
-- EXPENSE LAYER (separate from time entries)
-- ============================================================================

-- Expenses
DEFINE TABLE expenses SCHEMAFULL
  PERMISSIONS
    FOR select ALLOW (
      $auth.id == user_id OR
      $auth.role IN ['finance', 'admin']
    )
    FOR create ALLOW (
      $auth.id == user_id OR
      $auth.role IN ['finance', 'admin']
    )
    FOR update ALLOW (
      ($auth.id == user_id AND status == 'draft') OR
      $auth.role IN ['finance', 'admin']
    )
    FOR delete ALLOW false
;

DEFINE FIELD id ON TABLE expenses TYPE string;
DEFINE FIELD org_id ON TABLE expenses TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD user_id ON TABLE expenses TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD project_id ON TABLE expenses TYPE option<record<projects>>;
DEFINE FIELD unit_id ON TABLE expenses TYPE record<units> ASSERT $value != NONE;
DEFINE FIELD category ON TABLE expenses TYPE string ENUM ['mileage', 'meal', 'accommodation', 'other'] ASSERT $value != NONE;
DEFINE FIELD amount ON TABLE expenses TYPE number ASSERT $value > 0;
DEFINE FIELD currency ON TABLE expenses TYPE string DEFAULT 'EUR';
DEFINE FIELD description ON TABLE expenses TYPE string;
DEFINE FIELD expense_date ON TABLE expenses TYPE date ASSERT $value != NONE;
DEFINE FIELD receipt_url ON TABLE expenses TYPE option<string>;
DEFINE FIELD receipt_ocr_data ON TABLE expenses TYPE option<object>;
DEFINE FIELD status ON TABLE expenses TYPE string ENUM ['draft', 'submitted', 'approved', 'rejected'] DEFAULT 'draft';
DEFINE FIELD is_deleted ON TABLE expenses TYPE bool DEFAULT false;
DEFINE FIELD created_at ON TABLE expenses TYPE datetime DEFAULT now();
DEFINE FIELD updated_at ON TABLE expenses TYPE datetime DEFAULT now();

DEFINE INDEX exp_org ON TABLE expenses COLUMNS org_id;
DEFINE INDEX exp_user ON TABLE expenses COLUMNS user_id;
DEFINE INDEX exp_project ON TABLE expenses COLUMNS project_id;
DEFINE INDEX exp_unit ON TABLE expenses COLUMNS unit_id;
DEFINE INDEX exp_status ON TABLE expenses COLUMNS status;
DEFINE INDEX exp_date ON TABLE expenses COLUMNS expense_date;
DEFINE INDEX exp_user_date ON TABLE expenses COLUMNS user_id, expense_date;

-- ============================================================================
-- AUDIT LAYER (immutable, single source of truth)
-- ============================================================================

-- Audit Log (shared for time entries and expenses)
DEFINE TABLE audit_logs SCHEMAFULL
  PERMISSIONS
    FOR select ALLOW (
      $auth.role IN ['admin', 'finance', 'org_manager'] OR
      $auth.org_id == org_id
    )
    FOR create ALLOW true
    FOR update ALLOW false
    FOR delete ALLOW false
;

DEFINE FIELD id ON TABLE audit_logs TYPE string;
DEFINE FIELD org_id ON TABLE audit_logs TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD entry_id ON TABLE audit_logs TYPE string ASSERT $value != NONE;
DEFINE FIELD entry_type ON TABLE audit_logs TYPE string ENUM ['time_entry', 'expense'] ASSERT $value != NONE;
DEFINE FIELD action ON TABLE audit_logs TYPE string ENUM [
  'created', 'submitted', 'approved', 'rejected', 'split', 'moved',
  'reallocated', 'edited', 'finance_override', 'reverted'
] ASSERT $value != NONE;
DEFINE FIELD actor_role ON TABLE audit_logs TYPE string ENUM [
  'user', 'wg_manager', 'org_manager', 'finance', 'admin'
] ASSERT $value != NONE;
DEFINE FIELD actor_id ON TABLE audit_logs TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD reason ON TABLE audit_logs TYPE option<string>;
DEFINE FIELD changes ON TABLE audit_logs TYPE option<object>;
DEFINE FIELD timestamp ON TABLE audit_logs TYPE datetime DEFAULT now();
DEFINE FIELD ip_address ON TABLE audit_logs TYPE option<string>;

DEFINE INDEX audit_entry ON TABLE audit_logs COLUMNS entry_id;
DEFINE INDEX audit_actor ON TABLE audit_logs COLUMNS actor_id;
DEFINE INDEX audit_actor_role ON TABLE audit_logs COLUMNS actor_role;
DEFINE INDEX audit_timestamp ON TABLE audit_logs COLUMNS timestamp;
DEFINE INDEX audit_org_timestamp ON TABLE audit_logs COLUMNS org_id, timestamp;
DEFINE INDEX audit_entry_type ON TABLE audit_logs COLUMNS entry_type;

-- ============================================================================
-- CONFIGURATION LAYER
-- ============================================================================

-- Financial Cutoff Periods (organization-level config)
DEFINE TABLE financial_cutoff_periods SCHEMAFULL
  PERMISSIONS
    FOR select ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR create ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR update ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR delete ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
;

DEFINE FIELD id ON TABLE financial_cutoff_periods TYPE string;
DEFINE FIELD org_id ON TABLE financial_cutoff_periods TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD project_id ON TABLE financial_cutoff_periods TYPE option<record<projects>>;
DEFINE FIELD period_start ON TABLE financial_cutoff_periods TYPE date ASSERT $value != NONE;
DEFINE FIELD period_end ON TABLE financial_cutoff_periods TYPE date ASSERT $value != NONE;
DEFINE FIELD cutoff_date ON TABLE financial_cutoff_periods TYPE date ASSERT $value != NONE;
DEFINE FIELD is_locked ON TABLE financial_cutoff_periods TYPE bool DEFAULT false;
DEFINE FIELD created_at ON TABLE financial_cutoff_periods TYPE datetime DEFAULT now();

DEFINE INDEX fcp_org ON TABLE financial_cutoff_periods COLUMNS org_id;
DEFINE INDEX fcp_project ON TABLE financial_cutoff_periods COLUMNS project_id;
DEFINE INDEX fcp_dates ON TABLE financial_cutoff_periods COLUMNS period_start, period_end;

-- Budget Caps (expense limits per user/project/category)
DEFINE TABLE budget_caps SCHEMAFULL
  PERMISSIONS
    FOR select ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR create ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR update ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR delete ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
;

DEFINE FIELD id ON TABLE budget_caps TYPE string;
DEFINE FIELD org_id ON TABLE budget_caps TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD user_id ON TABLE budget_caps TYPE option<record<users>>;
DEFINE FIELD project_id ON TABLE budget_caps TYPE option<record<projects>>;
DEFINE FIELD category ON TABLE budget_caps TYPE option<string>;
DEFINE FIELD limit_amount ON TABLE budget_caps TYPE number ASSERT $value > 0;
DEFINE FIELD period ON TABLE budget_caps TYPE string ENUM ['daily', 'weekly', 'monthly', 'yearly'] DEFAULT 'monthly';
DEFINE FIELD currency ON TABLE budget_caps TYPE string DEFAULT 'EUR';
DEFINE FIELD created_at ON TABLE budget_caps TYPE datetime DEFAULT now();

DEFINE INDEX bc_org ON TABLE budget_caps COLUMNS org_id;
DEFINE INDEX bc_user ON TABLE budget_caps COLUMNS user_id;
DEFINE INDEX bc_project ON TABLE budget_caps COLUMNS project_id;

-- Customers (for project billing tracking)
DEFINE TABLE customers SCHEMAFULL
  PERMISSIONS
    FOR select ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR create ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR update ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
    FOR delete ALLOW (
      SELECT id FROM organizations WHERE id == $auth.org_id
    )
;

DEFINE FIELD id ON TABLE customers TYPE string;
DEFINE FIELD org_id ON TABLE customers TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD name ON TABLE customers TYPE string ASSERT $value != NONE;
DEFINE FIELD email ON TABLE customers TYPE string;
DEFINE FIELD address ON TABLE customers TYPE string;
DEFINE FIELD created_at ON TABLE customers TYPE datetime DEFAULT now();
DEFINE FIELD updated_at ON TABLE customers TYPE datetime DEFAULT now();

DEFINE INDEX cust_org ON TABLE customers COLUMNS org_id;

-- ============================================================================
-- END SCHEMA DEFINITION
-- ============================================================================
