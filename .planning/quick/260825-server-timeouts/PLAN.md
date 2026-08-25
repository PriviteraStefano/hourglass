---
quick_id: 260825-server-timeouts
description: Add HTTP timeouts + graceful shutdown (CONCERNS #14)
date: 2026-08-25
status: complete
---

# Quick Task 260825-server-timeouts

## Plan

Address CONCERNS.md #14 "HTTP server has no timeouts and no graceful shutdown".

Task: replace `stdhttp.ListenAndServe` in `cmd/server/main.go` with an explicit `&http.Server`
configured with `ReadTimeout` (15s), `WriteTimeout` (30s), `IdleTimeout` (120s), and a
SIGINT/SIGTERM handler that calls `srv.Shutdown` with a 30s context. ListenAndServe errors other
than `ErrServerClosed` are fatal.

## Verify

- `go build ./...` passes
- `go vet ./cmd/server` passes
