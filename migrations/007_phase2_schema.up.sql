-- Phase 2 schema migration: customers, org settings, project managers, and flattened entry columns

CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    company_name VARCHAR(255) NOT NULL,
    contact_name VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(50),
    vat_number VARCHAR(50),
    address TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_customers_organization ON customers(organization_id);
CREATE INDEX IF NOT EXISTS idx_customers_active ON customers(is_active);

CREATE TABLE IF NOT EXISTS organization_settings (
    organization_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    default_km_rate NUMERIC(10, 2),
    currency VARCHAR(3) NOT NULL DEFAULT 'EUR',
    week_start_day SMALLINT NOT NULL DEFAULT 1,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    show_approval_history BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_organization_settings_updated_at
    BEFORE UPDATE ON organization_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE FUNCTION create_default_organization_settings()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO organization_settings (organization_id)
    VALUES (NEW.id)
    ON CONFLICT (organization_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS create_default_organization_settings_trigger ON organizations;
CREATE TRIGGER create_default_organization_settings_trigger
    AFTER INSERT ON organizations
    FOR EACH ROW
    EXECUTE FUNCTION create_default_organization_settings();

INSERT INTO organization_settings (organization_id)
SELECT id FROM organizations
ON CONFLICT (organization_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS project_managers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_project_managers_project ON project_managers(project_id);
CREATE INDEX IF NOT EXISTS idx_project_managers_user ON project_managers(user_id);

ALTER TABLE contracts ADD COLUMN IF NOT EXISTS customer_id UUID REFERENCES customers(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_contracts_customer ON contracts(customer_id);

ALTER TABLE organization_memberships
    DROP CONSTRAINT IF EXISTS organization_memberships_user_id_organization_id_key;

ALTER TABLE organization_memberships
    DROP CONSTRAINT IF EXISTS organization_memberships_user_org_role_key;

ALTER TABLE organization_memberships
    ADD CONSTRAINT organization_memberships_user_org_role_key
    UNIQUE (user_id, organization_id, role);

ALTER TABLE time_entries ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE SET NULL;
ALTER TABLE time_entries ADD COLUMN IF NOT EXISTS hours NUMERIC(5, 2);
ALTER TABLE time_entries ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE time_entries ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_time_entries_project_id ON time_entries(project_id);
CREATE INDEX IF NOT EXISTS idx_time_entries_deleted_at ON time_entries(deleted_at);

ALTER TABLE expenses ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE SET NULL;
ALTER TABLE expenses ADD COLUMN IF NOT EXISTS customer_id UUID REFERENCES customers(id) ON DELETE SET NULL;
ALTER TABLE expenses ADD COLUMN IF NOT EXISTS type VARCHAR(50);
ALTER TABLE expenses ADD COLUMN IF NOT EXISTS amount NUMERIC(10, 2);
ALTER TABLE expenses ADD COLUMN IF NOT EXISTS km_distance NUMERIC(10, 2);
ALTER TABLE expenses ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE expenses ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_expenses_project_id ON expenses(project_id);
CREATE INDEX IF NOT EXISTS idx_expenses_customer_id ON expenses(customer_id);
CREATE INDEX IF NOT EXISTS idx_expenses_type ON expenses(type);
CREATE INDEX IF NOT EXISTS idx_expenses_deleted_at ON expenses(deleted_at);

ALTER TABLE expense_receipts ADD COLUMN IF NOT EXISTS expense_id UUID REFERENCES expenses(id) ON DELETE CASCADE;
ALTER TABLE expense_receipts ADD COLUMN IF NOT EXISTS receipt_data BYTEA;
ALTER TABLE expense_receipts ADD COLUMN IF NOT EXISTS mime_type VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_expense_receipts_expense_id ON expense_receipts(expense_id);

