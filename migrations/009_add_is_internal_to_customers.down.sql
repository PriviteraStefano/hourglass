-- 009_add_is_internal_to_customers.down.sql
ALTER TABLE customers DROP COLUMN IF EXISTS is_internal;
