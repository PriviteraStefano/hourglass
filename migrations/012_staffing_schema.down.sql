-- 012_staffing_schema.down.sql — reverse the staffing schema additions
--
-- Schema restoration is exact: the up migration is purely additive, so the
-- down drops the table/index/columns and restores the original role CHECK
-- verbatim. Data accommodation: any membership already carrying the new 'hr'
-- role (D-4) has no representation in the old CHECK, so it is downgraded to
-- 'employee' (least-privilege default) before the constraint is restored —
-- otherwise the CHECK restore fails on the violating row (SQLSTATE 23514).

-- ============================================================================
-- 1. availability_windows: drop index + table
-- ============================================================================
DROP INDEX IF EXISTS idx_availability_windows_org_user_dates;
DROP TABLE IF EXISTS availability_windows;

-- ============================================================================
-- 2. organization_memberships: restore the original role CHECK (without 'hr')
--    Existing 'hr' rows are downgraded to 'employee' first (see header).
-- ============================================================================
UPDATE organization_memberships SET role = 'employee', updated_at = NOW() WHERE role = 'hr';
ALTER TABLE organization_memberships DROP CONSTRAINT IF EXISTS organization_memberships_role_check;
ALTER TABLE organization_memberships ADD CONSTRAINT organization_memberships_role_check
    CHECK (role IN ('employee', 'manager', 'finance', 'customer'));

-- ============================================================================
-- 3. organization_memberships: drop the validity columns (D-2)
-- ============================================================================
ALTER TABLE organization_memberships
    DROP COLUMN IF EXISTS valid_from,
    DROP COLUMN IF EXISTS valid_until,
    DROP COLUMN IF EXISTS work_permit_expires_at;
