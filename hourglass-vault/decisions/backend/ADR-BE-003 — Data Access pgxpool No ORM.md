# ADR-BE-003 — Data Access: Hand-Written SQL on pgxpool, No ORM

---
tags: ["adr", "backend", "data-access", "postgres", "promote"]

---

# ADR-BE-003 — Data Access: Hand-Written SQL on pgxpool, No ORM

**Status:** Accepted · **(promote)** — candidate for the future global Go ADR set
**Date:** 2026-07-28
**Code:** `internal/db/pgpool.go`, `internal/adapters/secondary/postgres/*.go`, `go.mod`

## Context

v0.1 migrated SurrealDB → PostgreSQL and chose `pgx/v5` with hand-written SQL over an ORM (decision recorded in `.planning/PROJECT.md`: "Full query control, pgx is Go's gold standard PG driver" — outcome: good). That choice needs a written contract so it isn't gradually eroded by per-query shortcuts.

## Decision

* **Driver:** `github.com/jackc/pgx/v5` with `pgxpool`. No ORM, no query builder.
* **One shared pool** via `internal/db/pgpool.go` (`sync.Once` singleton; `MaxConns=25`, `MaxConnLifetime=30min`, `MaxConnIdleTime=5min`). Repositories receive `*pgxpool.Pool` by constructor injection.
* **SQL lives in repositories.** Queries are written inline in `internal/adapters/secondary/postgres/*_repository.go`; rows are mapped to domain entities inside the repository (no `pgx` types leave the layer — see [[ADR-BE-001 — Error Handling Sentinel Errors|ADR-BE-001]] `wrapPGError`).
* **Parameterised queries only** — never string-concatenated SQL.
* **Recursive CTEs** are sanctioned for hierarchy (units) queries.
* **No cross-repository transactions via shared magic** — when a write must span tables atomically (e.g. bootstrap: org + user + membership), it is done explicitly in one repository method on one connection, documented in that repo.

## Consequences

* Full control over queries and plans — important for hierarchy and approval queries.
* Predictable performance; no ORM magic to debug.
* ⚠️ Hand-written mapping is verbose — accepted; the mapping is the price of the hexagonal boundary.
* ⚠️ Single pool means no read/write splitting (audit §scaling) — accepted for v0.1; multi-pool is the documented scaling path, not a v0.1 need.

## Related

* [[ADR-BE-004 — Database Migrations]], [[ADR-BE-002 — Hexagonal Wiring]]
* [[ADR-BE-009 — Testing testcontainers testify]] (test pool pattern)
