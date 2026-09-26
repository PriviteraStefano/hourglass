---
status: complete
quick_id: 260825-db-pool-size
date: 2026-08-25
---

# Summary — 260825-db-pool-size

**Concern:** CONCERNS.md #15 DB connection pool size not configured.

**Change:** `internal/db/db.go` `NewPool` now sets `connConfig.MaxConns` to `DB_MAX_CONNS` when set,
otherwise 20 (the pgxpool default of 4 was serializing all authenticated queries and starving
under concurrency). AGENTS.md documents `DB_MAX_CONNS`.

**Result:** The server's connection pool is bounded to a sane, configurable size instead of the
unsafe 4-default.

**Verification:** `go build`, `go vet` for `./internal/db` pass.
