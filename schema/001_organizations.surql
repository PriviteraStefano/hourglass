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

DEFINE FIELD id ON TABLE organizations TYPE string;
DEFINE FIELD name ON TABLE organizations TYPE string ASSERT $value != NONE;
DEFINE FIELD slug ON TABLE organizations TYPE string ASSERT $value != NONE;
DEFINE FIELD description ON TABLE organizations TYPE string;
DEFINE FIELD financial_cutoff_days ON TABLE organizations TYPE number DEFAULT 7;
DEFINE FIELD financial_cutoff_config ON TABLE organizations TYPE object DEFAULT {
    cutoff_day_of_month: 28,
    grace_days: 7
};
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
DEFINE FIELD description ON TABLE units TYPE string;
DEFINE FIELD parent_unit_id ON TABLE units TYPE option<record<units>>;
DEFINE FIELD hierarchy_level ON TABLE units TYPE number DEFAULT 0;
DEFINE FIELD code ON TABLE units TYPE string;
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

DEFINE FIELD id ON TABLE users TYPE string;
DEFINE FIELD email ON TABLE users TYPE string ASSERT $value != NONE;
DEFINE FIELD name ON TABLE users TYPE string ASSERT $value != NONE;
DEFINE FIELD password_hash ON TABLE users TYPE string;
DEFINE FIELD is_active ON TABLE users TYPE bool DEFAULT true;
DEFINE FIELD created_at ON TABLE users TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON TABLE users TYPE datetime DEFAULT time::now();

DEFINE INDEX user_email ON TABLE users COLUMNS email UNIQUE;

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
DEFINE INDEX um_primary ON TABLE unit_memberships COLUMNS user_id, is_primary WHERE is_primary == true;
