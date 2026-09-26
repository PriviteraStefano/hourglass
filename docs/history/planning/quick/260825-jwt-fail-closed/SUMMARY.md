---
status: complete
quick_id: 260825-jwt-fail-closed
date: 2026-08-25
---

# Summary — 260825-jwt-fail-closed

**Concern:** CONCERNS.md #11 Dev JWT secret still boots the server.

**Change:** `cmd/server/main.go` now fails closed: if `JWT_SECRET` is unset it only boots with the
insecure default secret when `ALLOW_INSECURE_AUTH=1` is explicitly set; otherwise it exits with
FATAL. The previous branch only blocked production when `GO_ENV` was literally "production"/"staging",
so an unset/misconfigured `GO_ENV` in production silently enabled auth bypass via a publicly-known
key. AGENTS.md env note updated to document `ALLOW_INSECURE_AUTH`.

**Result:** A production deployment with `JWT_SECRET` unset (and no explicit opt-in) can no longer
boot into a forgeable-JWT state.

**Verification:** `go build`, `go vet` for `./cmd/server` pass.
