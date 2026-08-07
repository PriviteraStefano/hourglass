-- 016_contract_sold_hours.up.sql — Contracts: type + sold hours (D-N, D-08/D-09)
--
-- Contracts carry a sold-hours commitment in v0.2 (D-N). The discriminator
-- is contract_type with per-type semantics:
--   * 'support'  — sold_hours + sold_period required (support bucket,
--                  hours-only, carry-over, no expiry per D-P)
--   * 'project'  — sold_period must stay NULL (D-08/D-09: the project
--                  commitment is the total contract sold_hours)
--   * NULL       — legacy row (no funding commitment yet); passes every
--                  CHECK via the three-valued-logic guard (D-01/Pitfall 1)
--
-- All columns nullable, additive per D-16, no backfill.
-- sold_hours DECIMAL(8,2) aligns with time_entries.hours scale (research).

-- ============================================================================
-- 1. Discriminator + sold-hours columns (all nullable)
-- ============================================================================
ALTER TABLE contracts ADD COLUMN contract_type VARCHAR(50);
ALTER TABLE contracts ADD COLUMN sold_hours    DECIMAL(8,2);
ALTER TABLE contracts ADD COLUMN sold_period   VARCHAR(10);

-- ============================================================================
-- 2. Type-consistency check — three-valued-logic guard for legacy rows
-- ============================================================================
ALTER TABLE contracts ADD CONSTRAINT contracts_sold_check
    CHECK (contract_type IS NULL OR
           (contract_type = 'support' AND
            sold_hours IS NOT NULL AND sold_period IS NOT NULL) OR
           (contract_type = 'project' AND sold_period IS NULL));
