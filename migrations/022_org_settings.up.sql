-- 022_org_settings.up.sql — Org policy key/value store + planning_mode override
--
-- Generic org policy storage per D-13-18 (user decision): new policy keys are
-- data rows in org_settings (key/value JSONB), never typed columns on the
-- legacy organization_settings table. The known-key vocabulary is enforced in
-- code (plan 13-04) — a CHECK on JSONB is infeasible by design (T-13-03).
-- organization_memberships.planning_mode (D-13-19): nullable per-member
-- override, no backfill — NULL falls back to the org default (D-X).

-- ============================================================================
-- 1. org_settings (D-13-18)
-- ============================================================================
CREATE TABLE org_settings (
    org_id     UUID NOT NULL REFERENCES organizations(id),
    key        VARCHAR(50) NOT NULL,
    value      JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (org_id, key)
);

-- ============================================================================
-- 2. planning_mode override on memberships (D-13-19)
-- ============================================================================
ALTER TABLE organization_memberships ADD COLUMN planning_mode VARCHAR(20);
