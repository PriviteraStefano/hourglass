# Phase Pg-3: Wiring, cleanup & verification - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-07
**Phase:** Pg-3-Wiring
**Areas discussed:** Wiring order, internal/db/ package cleanup, Verification approach, CORS middleware location, Makefile cleanup specifics, Docker compose SurrealDB removal, Smoke test DB strategy

---

## Wiring Order

| Option | Description | Selected |
|--------|-------------|----------|
| Both active first | Add Postgres alongside SurrealDB, verify, then remove SurrealDB | |
| One-shot replacement | Replace all SurrealDB with Postgres in single pass | ✓ |
| You decide | Agent chooses | |

**User's choice:** One-shot replacement

| Option | Description | Selected |
|--------|-------------|----------|
| Direct pool per repo | `postgres.NewXxxRepo(pool)` — each repo in its own constructor | ✓ |
| Aggregate constructor | Single `postgres.NewAllRepos(pool)` returning all repos | |

**User's choice:** Direct pool per repo

| Option | Description | Selected |
|--------|-------------|----------|
| Remove entirely | Delete SURREALDB_* env var check and warning | ✓ |
| Keep warning | Keep deprecation notice in main.go | |

**User's choice:** Remove entirely

| Option | Description | Selected |
|--------|-------------|----------|
| Match existing constructors | Same line-by-line structure, swap surrealdb.NewXxx → postgres.NewXxx | ✓ |
| Aggregate setup func | One newRepos(pool) helper | |

**User's choice:** Match existing constructors. User asked about idiomatic Go approach — confirmed explicit per-repo construction is the standard Go pattern.

**Notes:** User asked for Go idiomatic approach comparison between direct construction and aggregate setup. Confirmed explicit per-repo construction is the most common Go pattern.

---

## internal/db/ Package Cleanup

| Option | Description | Selected |
|--------|-------------|----------|
| Keep as-is | internal/db/ with just pgpool.go | ✓ |
| Fold into cmd/server/ | Move pool creation into main.go | |
| Rename package | Rename to internal/dbpool/ | |

**User's choice:** Keep as-is

| Option | Description | Selected |
|--------|-------------|----------|
| Remove both | Remove surrealdb.go AND lib/pq, then go mod tidy | ✓ |
| Keep lib/pq | Remove only SurrealDB for now | |

**User's choice:** Remove both

**Notes:** db.go (old migrate CLI with lib/pq) is already gone — only surreal.go and pgpool.go remain.

---

## Verification Approach

| Option | Description | Selected |
|--------|-------------|----------|
| Automated smoke test | Go test in cmd/server/main_test.go | ✓ |
| Manual only | Rely on D-15 manual verification flow | |

**User's choice:** Automated smoke test

| Option | Description | Selected |
|--------|-------------|----------|
| cmd/server/main_test.go | Go test in server package | ✓ |
| Dedicated integration test | Separate directory | |
| Extend handler tests | Follow existing pattern | |

**User's choice:** cmd/server/main_test.go

| Option | Description | Selected |
|--------|-------------|----------|
| Health + 2-3 key flows | /health then /units (authenticated) | ✓ |
| Health only | Just verify server starts | |

**User's choice:** Health + 2-3 key flows

---

## CORS Middleware Location

| Option | Description | Selected |
|--------|-------------|----------|
| Extract to internal/middleware/ | Move to cors.go alongside other middleware | ✓ |
| Keep in main.go | Leave as inline closure | |

**User's choice:** Extract to internal/middleware/

---

## Makefile Cleanup Specifics

| Option | Description | Selected |
|--------|-------------|----------|
| make setup = go run ./cmd/migrate -all | Single target, applies and seeds | ✓ |
| make setup = migrate up + seed separately | Two explicit steps | |

**User's choice:** make setup = go run ./cmd/migrate -all

| Option | Description | Selected |
|--------|-------------|----------|
| Remove all schema/surreal targets | Delete everything surreal-related | ✓ |
| Keep as commented reference | Comment out rather than delete | |

**User's choice:** Remove all schema/surreal targets

---

## Docker Compose SurrealDB Removal

| Option | Description | Selected |
|--------|-------------|----------|
| Remove entirely | Delete SurrealDB service block | ✓ |
| Commented reference | Keep commented out for historical debugging | |

**User's choice:** Remove entirely

| Option | Description | Selected |
|--------|-------------|----------|
| Clean all references | Service, profiles section, volumes, env vars | ✓ |
| Service block only | Just remove the service | |

**User's choice:** Clean all references

---

## Smoke Test DB Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse Pg-2 test helpers | Use exported_test_helpers.go | ✓ |
| Dedicated server test DB | New test helper in cmd/server/ | |

**User's choice:** Reuse Pg-2 test helpers

| Option | Description | Selected |
|--------|-------------|----------|
| /health + /units | Health check then list units | ✓ |
| /health + /units + /projects | Add projects for JOIN patterns | |

**User's choice:** /health + /units

---

## the agent's Discretion

None — all decisions were user-directed.

## Deferred Ideas

None — discussion stayed within phase scope.
