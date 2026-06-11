-- Update CHECK constraint on expenses to support two-stage approval workflow
ALTER TABLE expenses DROP CONSTRAINT IF EXISTS expenses_status_check;
ALTER TABLE expenses ADD CONSTRAINT expenses_status_check
    CHECK (status IN ('draft', 'submitted', 'pending_manager', 'pending_finance', 'approved', 'rejected'));
