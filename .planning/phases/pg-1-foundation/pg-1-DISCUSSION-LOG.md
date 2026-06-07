# Phase Pg-1: Foundation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-07
**Phase:** Pg-1-Foundation
**Areas discussed:** Migrate CLI driver, Docker-compose env vars

---

## Migrate CLI Driver

| Option | Description | Selected |
|--------|-------------|----------|
| Keep database/sql+lib/pq | Migrate CLI stays on database/sql with lib/pq. Keep both drivers during Pg-1->Pg-3. Remove lib/pq in Pg-3 cleanup. | |
| Port migrate CLI to pgx | Update cmd/migrate to use pgxpool in Pg-1 (or Pg-3). Single pgx dep, cleaner story. More work in Pg-1 but simpler end state. | ✓ |

**User's choice:** Port migrate CLI to pgx
**Notes:** Single pgx dependency is cleaner end state. Port cmd/migrate, internal/db/db.go, and internal/db/migrate.go from database/sql+lib/pq to pgx in Pg-1.

---

## Docker-compose env vars

| Option | Description | Selected |
|--------|-------------|----------|
| Explicit DATABASE_URL in app service | App service gets explicit DATABASE_URL env var pointing to postgres container | ✓ |

**User's choice:** Add DATABASE_URL env var to app service in docker-compose.yml
**Notes:** Already covered by D-21, made explicit as D-26.

---

## Deferred Ideas

- None — discussion stayed within phase scope
