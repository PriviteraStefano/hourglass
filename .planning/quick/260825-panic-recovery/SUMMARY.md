---
status: complete
quick_id: 260825-panic-recovery
date: 2026-08-25
---

# Summary — 260825-panic-recovery

**Concern:** CONCERNS.md #13 No centralized panic-recovery middleware.

**Change:** New `middleware.Recovery` wraps the entire handler chain (outermost in
`cmd/server/main.go`). It `recover()`s panics from any handler/middleware, logs the stack, and
returns a clean `500 {"error":"internal"}` instead of Go's default connection-closing behavior
with a stack trace in logs and a possibly-corrupted JSON envelope.

**Result:** An unexpected panic in a handler no longer drops the connection or writes a partial
response; clients get a structured 500 and the server keeps serving.

**Verification:** `go build`, `go vet` for `./cmd/server` and `./internal/middleware` pass.
