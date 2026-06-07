---
phase: pg-2-adapters
plan: 05
type: execute
wave: 2
depends_on: [pg-2-01]
files_modified:
  - internal/adapters/secondary/postgres/refresh_token_repo.go
  - internal/adapters/secondary/postgres/refresh_token_repo_test.go
  - internal/adapters/secondary/postgres/password_reset_repository.go
  - internal/adapters/secondary/postgres/password_reset_repository_test.go
autonomous: true
requirements: []
must_haves:
  truths:
    - RefreshTokenRepository supports Add, FindByHash (with expiry+revocation filtering), RevokeByHash, RevokeAllByUser
    - PasswordResetRepository supports Create (with RETURNING), FindActiveByUserID (expiry+used_at filtering), MarkUsed, UpdateUserPassword
    - Refresh token queries filter on `expires_at > NOW() AND revoked_at IS NULL`
    - Password reset queries filter on `expires_at > NOW() AND used_at IS NULL`
  artifacts:
    - path: internal/adapters/secondary/postgres/refresh_token_repo.go
      provides: implements ports.RefreshTokenRepository (4 methods with time-based filtering)
    - path: internal/adapters/secondary/postgres/password_reset_repository.go
      provides: implements ports.PasswordResetRepository (4 methods with nullable column handling)
  key_links:
    - from: refresh_token_repo.go FindByHash
      to: refresh_tokens table (006_refresh_tokens)
      via: WHERE token_hash = $1 AND expires_at > NOW() AND revoked_at IS NULL
    - from: password_reset_repository.go FindActiveByUserID
      to: password_resets table
      via: WHERE user_id = $1 AND expires_at > NOW() AND used_at IS NULL
---

<objective>
Implement RefreshTokenRepository and PasswordResetRepository for PostgreSQL.

Purpose: Port the session/token management repos from SurrealDB. These are auth-domain critical — refresh tokens enable seamless re-authentication, password resets enable account recovery. Both rely on time-based expiry filtering and nullable column handling.

Output:
- refresh_token_repo.go + tests
- password_reset_repository.go + tests
</objective>

<execution_context>
@/Users/stefanoprivitera/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/stefanoprivitera/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/pg-2-adapters/pg-2-PATTERNS.md
@.planning/phases/pg-2-adapters/pg-2-RESEARCH.md

# Port interfaces
@internal/core/ports/refresh_token_repo.go
@internal/core/ports/password_reset_repository.go

# SurrealDB analogs
@internal/adapters/secondary/surrealdb/refresh_token_repo.go
@internal/adapters/secondary/surrealdb/password_reset_repository.go

# Domain models
@internal/core/domain/password_reset/password_reset.go

# Foundation helpers
@internal/adapters/secondary/postgres/postgres.go
@internal/adapters/secondary/postgres/exported_test_helpers.go

# Schema
@migrations/002_full_schema.up.sql (password_resets table)
@migrations/006_refresh_tokens.up.sql (refresh_tokens table)

<interfaces>
From internal/core/ports/refresh_token_repo.go:
```go
type RefreshTokenRepository interface {
    Add(ctx context.Context, userID, organizationID uuid.UUID, tokenHash string, expiresAt time.Time) error
    FindByHash(ctx context.Context, hash string) (*RefreshToken, error)
    RevokeByHash(ctx context.Context, hash string) error
    RevokeAllByUser(ctx context.Context, userID uuid.UUID) error
}
```

From internal/core/ports/password_reset_repository.go:
```go
type PasswordResetRepository interface {
    Create(ctx context.Context, pr *password_reset.PasswordReset) (*password_reset.PasswordReset, error)
    FindActiveByUserID(ctx context.Context, userID string) (*password_reset.PasswordReset, error)
    MarkUsed(ctx context.Context, id string) error
    UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
}
```
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: RefreshTokenRepository + tests</name>

  <files>
    internal/adapters/secondary/postgres/refresh_token_repo.go
    internal/adapters/secondary/postgres/refresh_token_repo_test.go
  </files>

  <read_first>
    @internal/core/ports/refresh_token_repo.go (interface — 4 methods + RefreshToken struct)
    @internal/adapters/secondary/surrealdb/refresh_token_repo.go (analog)
    @migrations/006_refresh_tokens.up.sql (table: id, user_id, organization_id, token_hash, expires_at, revoked_at, created_at)
  </read_first>

  <action>
    **A) refresh_token_repo.go** — implements ports.RefreshTokenRepository (4 methods)
    Struct `RefreshTokenRepository` with `pool *pgxpool.Pool`, constructor `NewRefreshTokenRepository`.

    **SQL queries with time-based filtering** (expires_at > NOW() AND revoked_at IS NULL):

    1. **Add**: `INSERT INTO refresh_tokens (id, user_id, organization_id, token_hash, expires_at, created_at) VALUES ($1,$2,$3,$4,$5,$6)`
       - Generate uuid.New() for ID
       - Wrap with wrapPGError

    2. **FindByHash**: 
       ```sql
       SELECT user_id, organization_id, token_hash, expires_at, created_at 
       FROM refresh_tokens 
       WHERE token_hash = $1 AND expires_at > NOW() AND revoked_at IS NULL 
       LIMIT 1
       ```
       - Scan into ports.RefreshToken struct fields
       - ErrNoRows → return nil, nil (not an error — token not found or expired is a normal auth failure)

    3. **RevokeByHash**: `UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL`

    4. **RevokeAllByUser**: `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`

    **B) refresh_token_repo_test.go:**
    - Test Add → FindByHash (create token, find by hash, assert fields match)
    - Test FindByHash expired (create token with past expires_at, FindByHash returns nil)
    - Test FindByHash revoked (create token, RevokeByHash, then FindByHash returns nil)
    - Test RevokeByHash (create token, revoke, verify revoked_at is set via raw SQL)
    - Test RevokeAllByUser (create 2 tokens for same user, revoke all, verify both revoked)
    - Seed: create org + user for FK references
  </action>

  <verify>
    <automated>go vet ./internal/adapters/secondary/postgres/ && go build ./internal/adapters/secondary/postgres/</automated>
  </verify>

  <done>
    1. RefreshTokenRepository with 4 methods compiles
    2. Time-based filtering (expires_at > NOW(), revoked_at IS NULL) works correctly
    3. Test file compiles
    4. `go build ./internal/adapters/secondary/postgres/` passes
  </done>
</task>

<task type="auto">
  <name>Task 2: PasswordResetRepository + tests</name>

  <files>
    internal/adapters/secondary/postgres/password_reset_repository.go
    internal/adapters/secondary/postgres/password_reset_repository_test.go
  </files>

  <read_first>
    @internal/core/ports/password_reset_repository.go (interface — 4 methods)
    @internal/core/domain/password_reset/password_reset.go (PasswordReset struct: ID uuid.UUID, UsedAt *time.Time, etc.)
    @internal/adapters/secondary/surrealdb/password_reset_repository.go (analog)
    @migrations/002_full_schema.up.sql (password_resets table: id, user_id, code_hash, expires_at, used_at, created_at)
  </read_first>

  <action>
    **A) password_reset_repository.go** — implements ports.PasswordResetRepository (4 methods)
    Struct `PasswordResetRepository` with `pool *pgxpool.Pool`, constructor `NewPasswordResetRepository`.

    **Methods:**

    1. **Create**: `INSERT INTO password_resets (id, user_id, code_hash, expires_at, created_at) VALUES ($1,$2,$3,$4,$5) RETURNING id, user_id, code_hash, expires_at, used_at, created_at`
       - QueryRow + Scan into password_reset.PasswordReset
       - `used_at` is nullable — scan into *time.Time

    2. **FindActiveByUserID**: 
       ```sql
       SELECT id, user_id, code_hash, expires_at, used_at, created_at 
       FROM password_resets 
       WHERE user_id = $1 AND expires_at > NOW() AND used_at IS NULL 
       ORDER BY created_at DESC LIMIT 1
       ```
       - Parse `userID string` to uuid.UUID first
       - ErrNoRows → `password_reset.ErrResetNotFound`

    3. **MarkUsed**: `UPDATE password_resets SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`
       - Parse `id string` to uuid.UUID first

    4. **UpdateUserPassword**: `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`

    **B) password_reset_repository_test.go:**
    - Test Create → FindActiveByUserID (create reset, find active, assert fields match)
    - Test FindActiveByUserID expired (create with past expires_at, expect ErrResetNotFound)
    - Test FindActiveByUserID used (create reset, MarkUsed, then Find returns ErrResetNotFound)
    - Test MarkUsed (create reset, mark used, verify used_at set)
    - Test UpdateUserPassword (create user, update password, verify hash changed)
    - Seed: create user for FK reference
  </action>

  <verify>
    <automated>go vet ./internal/adapters/secondary/postgres/ && go build ./internal/adapters/secondary/postgres/</automated>
  </verify>

  <done>
    1. PasswordResetRepository with 4 methods compiles
    2. Nullable column scanning (used_at *time.Time) handled correctly
    3. Time-based expiry filtering (expires_at > NOW(), used_at IS NULL) works
    4. Test file compiles
    5. `go build ./internal/adapters/secondary/postgres/` passes
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Application → PostgreSQL token/reset queries | Auth tokens and password reset requests flow through parameterized queries |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-pg2-14 | Tampering | refresh_token bulk revoke | mitigate | RevokeAllByUser only updates rows WHERE revoked_at IS NULL — idempotent, no double-revoke |
| T-pg2-15 | Tampering | password_reset MarkUsed | mitigate | WHERE used_at IS NULL prevents double-use of reset codes |
| T-pg2-16 | Information Disclosure | FindByHash returns nil,nil on not-found | accept | Returns nil instead of error to avoid revealing whether a token hash exists |
</threat_model>

<verification>
```bash
go vet ./internal/adapters/secondary/postgres/
go build ./internal/adapters/secondary/postgres/
# Integration tests:
# DATABASE_URL=... go test ./internal/adapters/secondary/postgres/ -run "TestRefreshToken|TestPasswordReset" -count=1 -v
```
</verification>

<success_criteria>
1. RefreshTokenRepository implements all 4 methods from ports.RefreshTokenRepository
2. PasswordResetRepository implements all 4 methods from ports.PasswordResetRepository
3. Auth token/reset queries use time-based filtering correctly
4. All tests compile
5. Files committed to git
</success_criteria>

<output>
After completion, create `.planning/phases/pg-2-adapters/pg-2-05-SUMMARY.md`
</output>
