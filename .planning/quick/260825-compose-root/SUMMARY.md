---
status: complete
quick_id: 260825-compose-root
date: 2026-08-25
---

# Summary — 260825-compose-root

**Concerns:** CONCERNS.md #2 (manual dependency wiring) + #3 (duplicated repo passed as two ports).

**Changes:**
- New `cmd/server/compose.go`: `appGraph` struct + `buildGraph(pool, jwtSecret)` composition root
  that assembles every repository, service, and handler in one typed place. `main.go` reduced from
  ~360 lines of wiring to a thin entry point (env/JWT, pool, buildGraph, route registration,
  middleware chain, `ListenAndServe`).
- #3: the time-entry service's two ports are now injected as `timeEntryRepo` (entry role) and a
  distinct `timeEntryApprovalRepo` (approval role), removing the positional same-variable reuse.

**Result:** Hexagonal graph is now composed in one explicit, compile-checked location; new endpoints
no longer require hand-editing `main()`. The two time-entry ports are clearly separated.

**Verification:** `go build ./...`, `go vet ./cmd/server`, `go test ./cmd/server/...` all pass.
