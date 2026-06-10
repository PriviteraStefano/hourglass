-- 009_add_is_internal_to_customers.up.sql
-- Adds the is_internal boolean column to customers (default false).
-- Seeds an internal customer record for each existing organization.

ALTER TABLE customers ADD COLUMN IF NOT EXISTS is_internal BOOLEAN NOT NULL DEFAULT false;

-- Seed: create internal customer records for existing organizations.
-- Idempotent: skips orgs that already have an internal customer record.
INSERT INTO customers (id, org_id, name, contact_name, email, phone, address, vat_number, is_active, is_internal, created_at, updated_at)
SELECT gen_random_uuid(), id, name, '', '', '', '', '', true, true, NOW(), NOW()
FROM organizations
WHERE NOT EXISTS (
    SELECT 1 FROM customers WHERE customers.org_id = organizations.id AND customers.is_internal = true
);
