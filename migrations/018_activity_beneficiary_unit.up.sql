-- 018_activity_beneficiary_unit.up.sql — Activities: beneficiary unit (COV-05)
--
-- Adds a nullable beneficiary_unit_id to activities per COV-05: the unit the
-- work benefits, inherited downward like contract_id (D-3, ADR-P-007). The
-- absorption funding source defaults from it (COV-02 / D-B).
--
-- Single nullable column, no backfill, no 3VL CHECK — mirrors contract_id
-- (011) exactly: legacy rows (NULL) pass untouched. Absorption-source
-- derivation is a read-path resolver, not a column constraint.

-- ============================================================================
-- 1. Beneficiary unit column (nullable, REFERENCES units(id))
-- ============================================================================
ALTER TABLE activities ADD COLUMN beneficiary_unit_id UUID REFERENCES units(id);

-- FK-adjacent access path: beneficiary-unit → activities.
CREATE INDEX idx_activities_beneficiary_unit_id ON activities(beneficiary_unit_id);
