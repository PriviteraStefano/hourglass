---
quick_id: 260825-compose-root
description: Extract composition root (buildGraph) from main.go + make time-entry approval port explicit (CONCERNS #2 #3)
date: 2026-08-25
status: complete
---

# Quick Task 260825-compose-root

## Plan

Address CONCERNS.md #2 (manual dependency wiring) and #3 (duplicated repo passed as two ports).

Tasks:
1. Move all repository/service/handler construction out of `main()` into `cmd/server/compose.go`
   `buildGraph(pool, jwtSecret) (*appGraph, error)` — a single typed composition root. `main.go`
   becomes a thin entry point: env/JWT check, pool, buildGraph, route registration, middleware, serve.
2. #3: `tesvc.NewService(timeEntryRepo, timeEntryRepo, ...)` passed the same concrete repo for two
   ports (TimeEntryRepository + TimeEntryApprovalRepository). Inject the approval role via a distinct,
   explicitly-named `timeEntryApprovalRepo := postgres.NewTimeEntryRepository(pool)` so the two roles
   are unambiguous at the call site.

## Verify

- `go build ./...` passes
- `go vet ./cmd/server` passes
- `go test ./cmd/server/...` passes
