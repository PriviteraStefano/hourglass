-- Remove two-stage approval columns from time_entries and expenses
ALTER TABLE time_entries DROP COLUMN IF EXISTS current_approver_role;
ALTER TABLE time_entries DROP COLUMN IF EXISTS submitted_at;

ALTER TABLE expenses DROP COLUMN IF EXISTS current_approver_role;
ALTER TABLE expenses DROP COLUMN IF EXISTS submitted_at;
