---
quick_id: 260825-db-pool-size
description: Configure pgxpool MaxConns from DB_MAX_CONNS (default 20) (CONCERNS #15)
date: 2026-08-25
status: complete
---

# Quick Task 260825-db-pool-size

## Plan

Address CONCERNS.md #15 "DB connection pool size not configured".

Task: in `internal/db/db.go` `NewPool`, set `connConfig.MaxConns` from env `DB_MAX_CONNS` (tunable),
defaulting to 20. Previously unset → pgxpool default 4, serializing authenticated traffic. Updated
AGENTS.md env note.

## Verify

- `go build ./internal/db` passes
- `go vet ./internal/db` passes
