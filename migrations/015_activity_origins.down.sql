-- 015_activity_origins.down.sql — reverse the origins extension (ADR-BE-004 cycle)
DROP INDEX IF EXISTS idx_activities_ticket_id;

ALTER TABLE activities DROP CONSTRAINT IF EXISTS activities_origin_refs_check;
ALTER TABLE activities DROP CONSTRAINT IF EXISTS activities_origin_type_check;

ALTER TABLE activities DROP COLUMN IF EXISTS ticket_id;
ALTER TABLE activities DROP COLUMN IF EXISTS reviewed_by;
ALTER TABLE activities DROP COLUMN IF EXISTS proposed_by;
ALTER TABLE activities DROP COLUMN IF EXISTS assigned_to;
ALTER TABLE activities DROP COLUMN IF EXISTS assigned_by;
ALTER TABLE activities DROP COLUMN IF EXISTS origin_type;
