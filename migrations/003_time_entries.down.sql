-- Drop trigger and function
DROP TRIGGER IF EXISTS update_time_entries_updated_at ON time_entries;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_time_entry_items_project_id;
DROP INDEX IF EXISTS idx_time_entry_items_time_entry_id;
DROP INDEX IF EXISTS idx_time_entries_status;
DROP INDEX IF EXISTS idx_time_entries_date;
DROP INDEX IF EXISTS idx_time_entries_organization_id;
DROP INDEX IF EXISTS idx_time_entries_user_id;

-- Drop tables
DROP TABLE IF EXISTS time_entry_items;
DROP TABLE IF EXISTS time_entries;