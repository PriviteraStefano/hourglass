-- Roll back Phase 2 schema migration

DROP INDEX IF EXISTS idx_expense_receipts_expense_id;
ALTER TABLE expense_receipts DROP COLUMN IF EXISTS mime_type;
ALTER TABLE expense_receipts DROP COLUMN IF EXISTS receipt_data;
ALTER TABLE expense_receipts DROP COLUMN IF EXISTS expense_id;

DROP INDEX IF EXISTS idx_expenses_deleted_at;
DROP INDEX IF EXISTS idx_expenses_type;
DROP INDEX IF EXISTS idx_expenses_customer_id;
DROP INDEX IF EXISTS idx_expenses_project_id;
ALTER TABLE expenses DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE expenses DROP COLUMN IF EXISTS description;
ALTER TABLE expenses DROP COLUMN IF EXISTS km_distance;
ALTER TABLE expenses DROP COLUMN IF EXISTS amount;
ALTER TABLE expenses DROP COLUMN IF EXISTS type;
ALTER TABLE expenses DROP COLUMN IF EXISTS customer_id;
ALTER TABLE expenses DROP COLUMN IF EXISTS project_id;

DROP INDEX IF EXISTS idx_time_entries_deleted_at;
DROP INDEX IF EXISTS idx_time_entries_project_id;
ALTER TABLE time_entries DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE time_entries DROP COLUMN IF EXISTS description;
ALTER TABLE time_entries DROP COLUMN IF EXISTS hours;
ALTER TABLE time_entries DROP COLUMN IF EXISTS project_id;

ALTER TABLE organization_memberships
	DROP CONSTRAINT IF EXISTS organization_memberships_user_org_role_key;
ALTER TABLE organization_memberships
	ADD CONSTRAINT organization_memberships_user_id_organization_id_key
	UNIQUE (user_id, organization_id);

DROP INDEX IF EXISTS idx_contracts_customer;
ALTER TABLE contracts DROP COLUMN IF EXISTS customer_id;

DROP INDEX IF EXISTS idx_project_managers_user;
DROP INDEX IF EXISTS idx_project_managers_project;
DROP TABLE IF EXISTS project_managers;

DROP TRIGGER IF EXISTS create_default_organization_settings_trigger ON organizations;
DROP FUNCTION IF EXISTS create_default_organization_settings();
DROP TRIGGER IF EXISTS update_organization_settings_updated_at ON organization_settings;
DROP TABLE IF EXISTS organization_settings;

DROP INDEX IF EXISTS idx_customers_active;
DROP INDEX IF EXISTS idx_customers_organization;
DROP TABLE IF EXISTS customers;

