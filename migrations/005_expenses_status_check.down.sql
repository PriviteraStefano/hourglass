ALTER TABLE expenses DROP CONSTRAINT IF EXISTS expenses_status_check;
ALTER TABLE expenses ADD CONSTRAINT expenses_status_check
    CHECK (status IN ('draft', 'submitted', 'approved', 'rejected'));
