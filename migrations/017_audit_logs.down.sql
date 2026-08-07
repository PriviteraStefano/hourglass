-- 017_audit_logs.down.sql — reverse the audit log (ADR-BE-004 cycle)
DROP TABLE IF EXISTS audit_logs;
