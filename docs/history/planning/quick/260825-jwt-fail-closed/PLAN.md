---
quick_id: 260825-jwt-fail-closed
description: Require JWT_SECRET or explicit ALLOW_INSECURE_AUTH opt-in at boot (CONCERNS #11)
date: 2026-08-25
status: complete
---

# Quick Task 260825-jwt-fail-closed

## Plan

Address CONCERNS.md #11 "Dev JWT secret still boots the server".

Task: in `cmd/server/main.go`, the server must not boot with the known dev secret unless an
operator explicitly sets `ALLOW_INSECURE_AUTH=1`. Otherwise (JWT_SECRET unset, any environment
including production with a misconfigured/unset GO_ENV) it refuses to boot with FATAL. Removed the
previous `GO_ENV == production/staging` branch that let an unset GO_ENV in prod fall through to the
insecure secret. Updated the AGENTS.md env note.

## Verify

- `go build ./cmd/server` passes
- `go vet ./cmd/server` passes
