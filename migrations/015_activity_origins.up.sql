-- 015_activity_origins.up.sql — Origins: discriminator + typed refs on activities (P-013)
--
-- Adds the origin discriminator and its reference columns to activities per
-- ADR-P-013 (D-D: refs set once at creation; D-02: only proposed_by is
-- required on employee proposals — reviewed_by is deliberately
-- unconstrained, research OQ1).
--
-- Every new column is nullable and the multi-column CHECK is written as
-- `origin_type IS NULL OR (<per-type rules>)` — the three-valued-logic
-- guard per D-01 / research Pitfall 1: legacy rows (origin_type NULL) pass
-- every check. Additive per D-16; no backfill.
--
-- FK resolution: activities.ticket_id REFERENCES tickets(id) requires 014
-- to be applied first — 014 is numbered before 015 for exactly this reason.

-- ============================================================================
-- 1. Discriminator + reference columns (all nullable)
-- ============================================================================
ALTER TABLE activities ADD COLUMN origin_type  VARCHAR(50);
ALTER TABLE activities ADD COLUMN assigned_by  UUID REFERENCES users(id);
ALTER TABLE activities ADD COLUMN assigned_to  UUID REFERENCES users(id);
ALTER TABLE activities ADD COLUMN proposed_by  UUID REFERENCES users(id);
ALTER TABLE activities ADD COLUMN reviewed_by  UUID REFERENCES users(id);
ALTER TABLE activities ADD COLUMN ticket_id    UUID REFERENCES tickets(id);

-- ============================================================================
-- 2. Vocabulary check (closed origin set)
-- ============================================================================
ALTER TABLE activities ADD CONSTRAINT activities_origin_type_check
    CHECK (origin_type IN ('manager_assignment', 'employee_proposal', 'customer_ticket'));

-- ============================================================================
-- 3. Ref-set check — three-valued-logic guard for legacy rows
--    (origin_type IS NULL OR <per-type rules with explicit IS [NOT] NULL>)
-- ============================================================================
ALTER TABLE activities ADD CONSTRAINT activities_origin_refs_check
    CHECK (origin_type IS NULL OR
           (origin_type = 'manager_assignment' AND
            assigned_by IS NOT NULL AND assigned_to IS NOT NULL AND
            proposed_by IS NULL AND reviewed_by IS NULL AND ticket_id IS NULL) OR
           (origin_type = 'employee_proposal' AND
            proposed_by IS NOT NULL AND
            assigned_by IS NULL AND assigned_to IS NULL AND ticket_id IS NULL) OR
           (origin_type = 'customer_ticket' AND
            ticket_id IS NOT NULL AND
            assigned_by IS NULL AND assigned_to IS NULL AND
            proposed_by IS NULL AND reviewed_by IS NULL));

-- FK-adjacent access path: ticket → linked activities.
CREATE INDEX idx_activities_ticket_id ON activities(ticket_id);
