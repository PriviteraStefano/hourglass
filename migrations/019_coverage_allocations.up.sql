-- 019_coverage_allocations.up.sql — Coverage: allocation ledger (D-01, ADR-BE-017)
--
-- The tagged-union allocation ledger per ADR-P-012: 1..N rows per entry with
-- the Σ invariant (Σ allocations = entry hours) enforced by the service inside
-- the replace-set transaction (D-07, CR-01 in-tx re-validation).
--
-- Funding-source shape per D-01: a source_type discriminator + nullable ref
-- columns (contract_id / unit_id) + a refs-to-type CHECK — mirroring the 015
-- origins / 016 sold-hours three-valued-logic house rule (ADR-BE-016): legacy
-- rows (source_type NULL) pass every CHECK. The five funding sources are
-- derived semantics of the three row-level values (contract draw = contract
-- budget / support bucket / zero-value service-request contract).
--
-- * entry_id has NO FK — polymorphic entry reference (D-K: 'time' only in
--   v0.2); the service validates the referenced entry exists, same org,
--   approved, not deleted. The entry_type CHECK ('time') is the schema side
--   of the costed D-K belt-and-braces pair (12-04 service branch).
-- * hours DECIMAL(8,2) NOT NULL CHECK (hours > 0) matches time_entries.hours
--   exactly (000 line 278).
-- * reason mandatory for absorption, justification mandatory for transfer
--   (COV-02); both use the `source_type <> 'x' OR col IS NOT NULL` form so
--   NULL can never satisfy the guard (Pitfall 2).
-- * Append-only ledger: the schema exposes no UPDATE/DELETE paths; the
--   replace-set write is DELETE+INSERT inside one transaction (D-07).
-- * No is_locked / closed flag — period close is a frozen snapshot (020),
--   never a lock (D-F, D-10). Balances and proposals are derived on read
--   (D-02, D-I); the ledger stores only confirmed allocations.

-- ============================================================================
-- 1. coverage_allocations
-- ============================================================================
CREATE TABLE coverage_allocations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(id),
    entry_type    VARCHAR(50) NOT NULL,                -- D-K: 'time' only in v0.2
    entry_id      UUID NOT NULL,                       -- polymorphic; no FK (D-K)
    source_type   VARCHAR(50),                         -- nullable: legacy rows pass (3VL)
    contract_id   UUID REFERENCES contracts(id),
    unit_id       UUID REFERENCES units(id),
    hours         DECIMAL(8,2) NOT NULL CHECK (hours > 0),
    reason        VARCHAR(50),                         -- mandatory for absorption (COV-02)
    justification TEXT,                                -- mandatory for transfer (COV-02)
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- 2. Refs-to-type CHECK — three-valued-logic guard (Pitfall 2 / ADR-BE-016):
--    NULL source_type passes; each type pins exactly one ref.
-- ============================================================================
ALTER TABLE coverage_allocations ADD CONSTRAINT coverage_allocations_source_check
    CHECK (source_type IS NULL OR
           (source_type = 'contract'   AND contract_id IS NOT NULL AND unit_id IS NULL) OR
           (source_type = 'absorption' AND unit_id IS NOT NULL AND contract_id IS NULL) OR
           (source_type = 'transfer'   AND contract_id IS NOT NULL AND unit_id IS NULL));

-- ============================================================================
-- 3. Closed vocabularies (house style, BE-016)
-- ============================================================================
ALTER TABLE coverage_allocations ADD CONSTRAINT coverage_allocations_source_type_check
    CHECK (source_type IN ('contract', 'absorption', 'transfer'));
-- Absorption reasons per COV-02 — exactly these three; the Part-5 "plain
-- internal" fourth value is superseded.
ALTER TABLE coverage_allocations ADD CONSTRAINT coverage_allocations_reason_vocab_check
    CHECK (reason IS NULL OR reason IN ('WarrantyBug', 'UnderEstimate', 'Goodwill'));
-- D-K: polymorphic entry_type — 'time' only in v0.2 (schema side of the pair).
ALTER TABLE coverage_allocations ADD CONSTRAINT coverage_allocations_entry_type_check
    CHECK (entry_type IN ('time'));

-- ============================================================================
-- 4. Mandatory-field CHECKs — never NULL-satisfiable (Pitfall 2)
-- ============================================================================
ALTER TABLE coverage_allocations ADD CONSTRAINT coverage_allocations_reason_check
    CHECK (source_type <> 'absorption' OR reason IS NOT NULL);
ALTER TABLE coverage_allocations ADD CONSTRAINT coverage_allocations_justification_check
    CHECK (source_type <> 'transfer' OR justification IS NOT NULL);

-- ============================================================================
-- 5. Access paths: org-scoped reads, entry lookups, ref-resolution reads
-- ============================================================================
CREATE INDEX idx_coverage_allocations_org      ON coverage_allocations(org_id);
CREATE INDEX idx_coverage_allocations_entry    ON coverage_allocations(entry_type, entry_id);
CREATE INDEX idx_coverage_allocations_contract ON coverage_allocations(contract_id);
CREATE INDEX idx_coverage_allocations_unit     ON coverage_allocations(unit_id);
