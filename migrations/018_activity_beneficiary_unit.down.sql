-- 018_activity_beneficiary_unit.down.sql — reverse the beneficiary unit extension (ADR-BE-004 cycle)
-- Drop the index first, then the column (016 down shape). The FK constraint
-- activities_beneficiary_unit_id_fkey is dropped with the column.
DROP INDEX IF EXISTS idx_activities_beneficiary_unit_id;

ALTER TABLE activities DROP COLUMN IF EXISTS beneficiary_unit_id;
