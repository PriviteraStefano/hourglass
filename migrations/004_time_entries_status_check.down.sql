ALTER TABLE time_entries DROP CONSTRAINT IF EXISTS time_entries_status_check;
ALTER TABLE time_entries ADD CONSTRAINT time_entries_status_check
    CHECK (status IN ('draft', 'submitted', 'approved'));
