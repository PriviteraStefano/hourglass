-- 012_staffing_schema.up.sql — Staffing: availability windows + membership validity + hr role
--
-- Additive staffing structure data per ADR-P-008 (D-1, D-1a, D-2, D-4), zero
-- FK coupling to the activity ontology rewrite (P-008 consequence §103).
--   * availability_windows — typed absence windows (holiday/permit/medical/unavailable)
--   * organization_memberships + valid_from / valid_until / work_permit_expires_at
--   * organization_memberships.role CHECK extended with 'hr'
--
-- NOTE: named 012 not 011 — 011 is taken by 011_activity_ontology (plan 09-01);
-- per ADR-BE-004, new migration files continue from the max.

-- ============================================================================
-- 1. availability_windows (ADR-P-008 D-1, D-1a)
-- ============================================================================
CREATE TABLE availability_windows (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    kind            VARCHAR(20) NOT NULL DEFAULT 'unavailable'
                    CHECK (kind IN ('holiday', 'permit', 'medical', 'unavailable')),  -- D-1
    starts_on       DATE NOT NULL,
    ends_on         DATE NOT NULL CHECK (ends_on >= starts_on),
    hours           DECIMAL(4,2),                    -- D-1: partial-day permits
    certificate_ref VARCHAR(100),                    -- D-1a: medical only, INPS protocol no.; never the document
    note            TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'declared'
                    CHECK (status IN ('declared', 'confirmed')),  -- D-1a: holiday -> confirmed when both lines confirm
    created_by      UUID NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Per-person date-range lookups for the assignment-time surfaces (D-3).
CREATE INDEX idx_availability_windows_org_user_dates
    ON availability_windows(org_id, user_id, starts_on, ends_on);

-- ============================================================================
-- 2. organization_memberships: employment validity dates (D-2)
-- ============================================================================
ALTER TABLE organization_memberships
    ADD COLUMN valid_from DATE,
    ADD COLUMN valid_until DATE,              -- NULL = open-ended
    ADD COLUMN work_permit_expires_at DATE;   -- NULL = not applicable

-- ============================================================================
-- 3. role CHECK extended with 'hr' (D-4)
--    PostgreSQL cannot alter CHECK constraints in place — drop and recreate.
-- ============================================================================
ALTER TABLE organization_memberships DROP CONSTRAINT IF EXISTS organization_memberships_role_check;
ALTER TABLE organization_memberships ADD CONSTRAINT organization_memberships_role_check
    CHECK (role IN ('employee', 'manager', 'finance', 'customer', 'hr'));
