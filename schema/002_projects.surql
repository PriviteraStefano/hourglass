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
DEFINE FIELD email ON TABLE customers TYPE string;
DEFINE FIELD address ON TABLE customers TYPE string;
DEFINE FIELD vat_number ON TABLE customers TYPE string;
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
DEFINE FIELD description ON TABLE projects TYPE string;
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
DEFINE FIELD description ON TABLE subprojects TYPE string;
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
DEFINE FIELD description ON TABLE working_groups TYPE string;
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
