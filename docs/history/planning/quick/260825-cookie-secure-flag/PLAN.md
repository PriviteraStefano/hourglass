---
quick_id: 260825-cookie-secure-flag
description: Derive cookie Secure flag from TLS/deployment flag, not X-Forwarded-Proto (CONCERNS #12)
date: 2026-08-25
status: complete
---

# Quick Task 260825-cookie-secure-flag

## Plan

Address CONCERNS.md #12 "Refresh-token Secure flag trusts X-Forwarded-Proto".

Task: `internal/cookies/cookies.go` `IsSecureRequest` no longer trusts the client-influenced
`X-Forwarded-Proto` header. It now returns secure only when `r.TLS != nil` or the operator sets
`SECURE_COOKIES=1|true` (deployment flag). Documented the flag in AGENTS.md.

## Verify

- `go build ./internal/cookies` passes
- `go vet ./internal/cookies` passes
- `go test ./internal/cookies/... ./internal/adapters/primary/http/...` passes
