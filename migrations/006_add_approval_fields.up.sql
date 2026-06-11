-- Add two-stage approval columns to time_entries and expenses
ALTER TABLE time_entries ADD COLUMN IF NOT EXISTS current_approver_role VARCHAR(50);
ALTER TABLE time_entries ADD COLUMN IF NOT EXISTS submitted_at TIMESTAMPTZ;

ALTER TABLE expenses ADD COLUMN IF NOT EXISTS current_approver_role VARCHAR(50);
ALTER TABLE expenses ADD COLUMN IF NOT EXISTS submitted_at TIMESTAMPTZ;
