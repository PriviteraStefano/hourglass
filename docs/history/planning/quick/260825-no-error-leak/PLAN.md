---
quick_id: 260825-no-error-leak
description: Stop leaking internal error strings to clients in auth handlers (CONCERNS #6)
date: 2026-08-25
status: complete
---

# Quick Task 260825-no-error-leak

## Plan

Address CONCERNS.md #6 "Internal error messages leaked to clients".

Task: the only two `err.Error()` leak sites in the HTTP handlers are `auth.go:72` (Register
default case) and `auth.go:220` (Bootstrap). Replace both with stable generic messages and log
the detailed error server-side via `log.Printf`. Adds the `log` import to `auth.go`.

## Verify

- `go build ./internal/adapters/primary/http` passes
- `go vet ./internal/adapters/primary/http` passes
- grep confirms no remaining `err.Error()` passed to `RespondWithError`
