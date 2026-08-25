---
quick_id: 260825-panic-recovery
description: Add Recovery middleware converting panics into clean 500 (CONCERNS #13)
date: 2026-08-25
status: complete
---

# Quick Task 260825-panic-recovery

## Plan

Address CONCERNS.md #13 "No centralized panic-recovery middleware".

Task: add `internal/middleware/recovery.go` `Recovery(next)` that recovers from panics in any
downstream handler, logs the stack, and writes a clean `500 {"error":"internal"}` JSON response.
Wire it as the outermost middleware in `cmd/server/main.go`.

## Verify

- `go build ./cmd/server ./internal/middleware` passes
- `go vet ./cmd/server ./internal/middleware` passes
