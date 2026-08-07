-- 014_ticket_schema.up.sql — Tickets: internal issue tracking (P-003 rev)
--
-- First-class internal tickets per ADR-P-003 rev (D-A: lifecycle
-- open→triage→planned→in_progress→resolved→closed + reopen via 'open';
-- D-E: internal-only in v0.2 — no customer-facing portal). Named 014 —
-- one slot BEFORE 015_activity_origins — because 015 gains a tickets FK
-- and PostgreSQL resolves FKs at apply time (A8 ordering problem solved
-- by numbering tickets 014).
--
-- All additive per D-16 (no backfill, no rewrite):
--   * tickets — closed kind vocabulary, closed status vocabulary,
--     nullable dismissed_hours (TICK-04: "dismissed with N h logged" —
--     raw Σ value per D-13; the dismissal audit row lands in audit_logs)
--   * ticket_comments — append-only comments, CASCADE on ticket delete

-- ============================================================================
-- 1. tickets
-- ============================================================================
CREATE TABLE tickets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    kind            VARCHAR(50) NOT NULL
                    CHECK (kind IN ('question','bug','change','evolution')),        -- D-A: closed set
    status          VARCHAR(50) NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open','triage','planned','in_progress',
                                      'resolved','closed','dismissed')),            -- D-A: lifecycle
    requester_id    UUID NOT NULL REFERENCES users(id),
    assignee_id     UUID REFERENCES users(id),
    dismissed_hours DECIMAL(8,2),            -- TICK-04: hours logged before dismissal (D-13 raw Σ)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- FK-adjacent access paths: per-org listing and status-filtered queues.
CREATE INDEX idx_tickets_org_id ON tickets(org_id);
CREATE INDEX idx_tickets_status ON tickets(status);

-- ============================================================================
-- 2. ticket_comments (append-only)
-- ============================================================================
CREATE TABLE ticket_comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id  UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    author_id  UUID NOT NULL REFERENCES users(id),
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ticket_comments_ticket_id ON ticket_comments(ticket_id);
