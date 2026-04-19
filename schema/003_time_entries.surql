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