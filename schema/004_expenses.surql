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
DEFINE FIELD description ON TABLE expenses TYPE string;
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