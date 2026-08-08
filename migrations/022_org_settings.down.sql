-- 022_org_settings.down.sql — reverse org_settings + planning_mode
-- (drop the column before the table: organization_memberships outlives
-- org_settings and has no dependency on it, but column-first keeps the
-- reverse ordered with the up)
ALTER TABLE organization_memberships DROP COLUMN IF EXISTS planning_mode;
DROP TABLE IF EXISTS org_settings CASCADE;
