-- 021_direction_rows.up.sql — Direction: per-day plan rows (P-015)
--
-- First-class direction plane per ADR-P-015 (D-R: mode derived from
-- planned_date — set = scheduled, NULL = queued; D-W/D-AA: per-day rows
-- always, multiple rows may share a day — the ROW id is the identity noun,
-- no (org_id, directed_to, activity_id, planned_date) constraint, per-day
-- multiplicity is legal; D-13-04/08/10: supersede chain is append-only
-- history, cancelled is terminal, no is_deleted soft-delete column).
--
-- Every CHECK is written with explicit IS [NOT] NULL on both sides (research
-- Pitfall 2): this is a NEW table — no legacy rows — so no three-valued-logic
-- guard is needed; the never-NULL-satisfiable forms are the constraint
-- correctness guarantee (T-13-01). est_hours DECIMAL(8,2) mirrors
-- time_entries.hours exactly (000_full_schema.up.sql:278, D-13-02).

-- ============================================================================
-- 1. direction
-- ============================================================================
CREATE TABLE direction (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id              UUID NOT NULL REFERENCES organizations(id),
    directed_by         UUID NOT NULL REFERENCES users(id),    -- creator/manager attribution
    directed_to         UUID REFERENCES users(id),             -- XOR with wg_id (D-13-05)
    wg_id               UUID REFERENCES working_groups(id),    -- WG target (D-T)
    activity_id         UUID NOT NULL REFERENCES activities(id),
    planned_date        DATE,                                  -- NULL = queued (D-R)
    est_hours           DECIMAL(8,2),                          -- required on scheduled; optional queued budget (D-13-02)
    priority            INT,                                   -- queued ordering, lower = higher (D-13-06)
    due_date            DATE,                                  -- queued ordering (D-13-06)
    status              VARCHAR(20) NOT NULL DEFAULT 'draft',                       -- vocab: direction_status_check (D-13-07)
    supersedes_id       UUID REFERENCES direction(id),         -- replanning chain (D-13-04/08)
    origin_direction_id UUID REFERENCES direction(id),         -- claim chain: WG row → claim row (D-13-11)
    reason              TEXT,                                  -- mandatory for cancel/unclaim (D-13-10/16)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- 2. Shape constraints (new table — explicit IS [NOT] NULL, no 3VL guard)
-- ============================================================================
-- User-XOR-WG (D-13-05): exactly one of directed_to / wg_id is set.
ALTER TABLE direction ADD CONSTRAINT direction_target_check
    CHECK ((directed_to IS NOT NULL AND wg_id IS NULL)
        OR (directed_to IS NULL AND wg_id IS NOT NULL));
-- WG rows are queued-only (D-13-17): a WG target never carries a planned_date.
ALTER TABLE direction ADD CONSTRAINT direction_wg_queued_check
    CHECK (wg_id IS NULL OR planned_date IS NULL);
-- est_hours: > 0 when present (D-13-02/03, mirrors time_entries.hours).
ALTER TABLE direction ADD CONSTRAINT direction_est_hours_check
    CHECK (est_hours IS NULL OR est_hours > 0);
-- Scheduled rows must carry est_hours (D-13-02).
ALTER TABLE direction ADD CONSTRAINT direction_scheduled_hours_check
    CHECK (planned_date IS NULL OR est_hours IS NOT NULL);
-- Closed status vocabulary (D-13-07).
ALTER TABLE direction ADD CONSTRAINT direction_status_check
    CHECK (status IN ('draft','active','superseded','cancelled'));
-- Cancellation requires a reason (D-13-10) — `reason IS NOT NULL` is 2VL, so
-- this form is never NULL-satisfiable: a cancelled row with NULL reason is
-- FALSE OR FALSE and rejected (Pitfall 2).
ALTER TABLE direction ADD CONSTRAINT direction_cancel_reason_check
    CHECK (status <> 'cancelled' OR reason IS NOT NULL);

-- ============================================================================
-- 3. Access-path indexes
-- ============================================================================
CREATE INDEX idx_direction_org_employee_day ON direction(org_id, directed_to, planned_date);
CREATE INDEX idx_direction_org_wg ON direction(org_id, wg_id);
CREATE INDEX idx_direction_activity_created ON direction(activity_id, created_at);  -- origin fallback read (D-13-33)
CREATE INDEX idx_direction_supersedes ON direction(supersedes_id);
