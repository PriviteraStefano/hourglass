-- 016_contract_sold_hours.down.sql — reverse the sold-hours extension (ADR-BE-004 cycle)
ALTER TABLE contracts DROP CONSTRAINT IF EXISTS contracts_sold_check;

ALTER TABLE contracts DROP COLUMN IF EXISTS sold_period;
ALTER TABLE contracts DROP COLUMN IF EXISTS sold_hours;
ALTER TABLE contracts DROP COLUMN IF EXISTS contract_type;
