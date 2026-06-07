---
phase: pg-2-adapters
plan: "05"
name: "Token & Password Reset Repos"
subsystem: "postgres-adapters"
tags:
  - refresh-tokens
  - password-reset
  - postgres
  - repository
requires: []
provides:
  - RefreshTokenRepository
  - PasswordResetRepository
affects:
  - internal/core/ports/refresh_token_repo.go
  - internal/core/ports/password_reset_repository.go
  - internal/core/domain/password_reset/password_reset.go
decisions:
  - FindByHash returns nil,nil on no match (caller decides how to handle)
  - FindActiveByUserID returns domain ErrResetNotFound on no match
  - PasswordResetRepository parses UUID strings at the adapter boundary
metrics:
  duration: ~10m
  completed: 2026-06-07
---

# Phase pg-2-adapters Plan 05: Token & Password Reset Repos Summary

RefreshTokenRepository (4 methods) and PasswordResetRepository (4 methods) implemented against PostgreSQL using pgxpool, following the established repository pattern in the postgres adapter package.

## Files Created

| File | Purpose |
|------|---------|
| `internal/adapters/secondary/postgres/refresh_token_repo.go` | RefreshTokenRepository - Add, FindByHash, RevokeByHash, RevokeAllByUser |
| `internal/adapters/secondary/postgres/refresh_token_repo_test.go` | 5 tests covering happy path, expired, revoked, not found, RevokeAllByUser |
| `internal/adapters/secondary/postgres/password_reset_repository.go` | PasswordResetRepository - Create, FindActiveByUserID, MarkUsed, UpdateUserPassword |
| `internal/adapters/secondary/postgres/password_reset_repository_test.go` | 5 tests covering happy path, expired, used-after-MarkUsed, not found, password update |

## Implementation Details

### RefreshTokenRepository

- **Add** — INSERT with `gen_random_uuid()` for id, delegates error wrapping
- **FindByHash** — SELECT with `expires_at > NOW() AND revoked_at IS NULL` guard; `pgx.ErrNoRows` maps to `nil, nil` (caller decides absence handling)
- **RevokeByHash** — UPDATE setting `revoked_at = NOW()` scoped to `revoked_at IS NULL`
- **RevokeAllByUser** — UPDATE scoped to user_id and `revoked_at IS NULL`

### PasswordResetRepository

- **Create** — INSERT with RETURNING clause, scans full row including nullable `used_at`
- **FindActiveByUserID** — Parses `userID` string to `uuid.UUID`, queries `expires_at > NOW() AND used_at IS NULL`, orders by `created_at DESC LIMIT 1`; `pgx.ErrNoRows` maps to domain `password_reset.ErrResetNotFound`
- **MarkUsed** — Parses `id` string to `uuid.UUID`, UPDATE with `used_at IS NULL` guard
- **UpdateUserPassword** — UPDATE users SET password_hash, updated_at NOW()

## Deviations from Plan

None — plan executed exactly as written.

## Verification

- `go vet ./internal/adapters/secondary/postgres/` — PASS
- `go build ./internal/adapters/secondary/postgres/` — PASS
- Each task committed individually with descriptive messages

## Commits

- `f7a8e65` — feat(pg-2-05): RefreshTokenRepository with tests
- `5dcd632` — feat(pg-2-05): PasswordResetRepository with tests
