-- 017_audit_logs.up.sql — General append-only audit log (D-05)
--
-- Generic audit trail scoped by org, addressed by (entity_type, entity_id):
--   * append-only by design — the schema exposes no UPDATE/DELETE paths
--     (threats T-11-05..T-11-08 enforce immutability in the code plans;
--     this table shapes exactly what those plans rely on)
--   * actor_id nullable (system-initiated events), comment free-form,
--     payload JSONB for event-specific data (D-05)
--   * used for ticket dismissal audit rows (TICK-04) among other events
--
-- Additive per D-16; no backfill.

-- ============================================================================
-- 1. audit_logs
-- ============================================================================
CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id),
    entity_type VARCHAR(50) NOT NULL,
    entity_id   UUID NOT NULL,
    action      VARCHAR(50) NOT NULL,
    actor_id    UUID REFERENCES users(id),
    comment     TEXT,
    payload     JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Entity-scoped history reads (the dominant access path).
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id, created_at);
