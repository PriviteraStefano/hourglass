-- 020_coverage_snapshots.up.sql — Coverage: period-close snapshots (D-10/D-11/D-12)
--
-- A reported period never changes retroactively: the close operation (12-06)
-- freezes the period's allocation state into entry-level rows here; reports
-- and any future invoicing read the snapshot, never live rows (D-10).
-- Allocations stay editable indefinitely — the snapshot is frozen, not a lock
-- (D-F, Q10 amendment).
--
-- * Entry-level rows only, no aggregate columns (D-11): bucket levels,
--   billing totals and per-unit aggregates are computed from these rows on
--   read when the Phase 17 surfaces land.
-- * Append-only by construction (COV-04): no UPDATE/DELETE paths exist
--   anywhere; the only delete path is the CASCADE from the close header.
-- * Duplicate/overlapping close rejection is a repo-level in-tx check
--   returning 409 (A6, 12-06), not a DB unique constraint — deliberately no
--   UNIQUE(org_id, period_start, period_end).
-- * financial_cutoff_periods stays facts-only; this close is a separate
--   mechanism (D-12, Q10 amendment).

-- ============================================================================
-- 1. coverage_period_closes — close header
-- ============================================================================
CREATE TABLE coverage_period_closes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(id),
    period_start DATE NOT NULL,
    period_end   DATE NOT NULL,
    closed_by    UUID NOT NULL REFERENCES users(id),
    closed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- 2. coverage_snapshot_rows — frozen per-entry allocation state (D-11)
-- ============================================================================
CREATE TABLE coverage_snapshot_rows (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    close_id      UUID NOT NULL REFERENCES coverage_period_closes(id) ON DELETE CASCADE,
    entry_id      UUID NOT NULL,            -- polymorphic entry (time in v0.2)
    employee_id   UUID NOT NULL,            -- entry owner at close (D-11)
    entry_date    DATE NOT NULL,            -- entry_date at close (D-11)
    activity_id   UUID NOT NULL,            -- activity chain snapshot (D-11)
    source_type   VARCHAR(50) NOT NULL,
    contract_id   UUID,                     -- resolved contract at close (frozen)
    unit_id       UUID,                     -- beneficiary unit at close (frozen)
    hours         DECIMAL(8,2) NOT NULL CHECK (hours > 0),
    reason        VARCHAR(50),
    justification TEXT
);

-- Access paths: per-close reads (reports), per-entry lookups.
CREATE INDEX idx_coverage_snapshot_rows_close ON coverage_snapshot_rows(close_id);
CREATE INDEX idx_coverage_snapshot_rows_entry ON coverage_snapshot_rows(entry_id);
