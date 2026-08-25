---
status: complete
quick_id: 260825-server-timeouts
date: 2026-08-25
---

# Summary — 260825-server-timeouts

**Concern:** CONCERNS.md #14 HTTP server has no timeouts and no graceful shutdown.

**Change:** `cmd/server/main.go` now constructs an explicit `&http.Server` with `ReadTimeout`,
`WriteTimeout`, and `IdleTimeout` (mitigating Slowloris / slow-reading clients), and runs
`srv.Shutdown(ctx)` on `SIGINT`/`SIGTERM` so in-flight requests are drained on deploy/restart
instead of dropped. `ErrServerClosed` is treated as a clean stop.

**Result:** The server is resilient to slow clients and shuts down gracefully.

**Verification:** `go build ./...` and `go vet ./cmd/server` pass.
