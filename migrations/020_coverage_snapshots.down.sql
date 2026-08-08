-- 020_coverage_snapshots.down.sql — reverse the snapshot tables (ADR-BE-004 cycle)
-- Rows first (they reference the header), then the header.
DROP TABLE IF EXISTS coverage_snapshot_rows CASCADE;
DROP TABLE IF EXISTS coverage_period_closes CASCADE;
