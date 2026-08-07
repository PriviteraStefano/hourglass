-- 014_ticket_schema.down.sql — reverse the ticket schema (ADR-BE-004 cycle)
-- Dependency order: comments before tickets (CASCADE would absorb it anyway).
DROP TABLE IF EXISTS ticket_comments;
DROP TABLE IF EXISTS tickets;
