# Codebase Concerns

**Analysis Date:** 2026-05-12

## Tech Debt

**Postgres/SurrealDB Dual Persistence:**
- Issue: Project uses both PostgreSQL (migrations) and SurrealDB (primary datastore), creating dual persistence overhead
- Files: `cmd/migrate/main.go`, `internal/db/migrate.go`, `schema/`, `migrations/`
- Impact: Maintenance burden, potential data sync issues, additional infrastructure requirements
- Fix approach: Deprecate Postgres migrations, move to SurrealDB-only or implement proper data migration

**Auto-generated Files in Version Control:**
- Issue: `web/src/routeTree.gen.ts` (329 lines) is auto-generated but checked into git
- Files: `web/src/routeTree.gen.ts`
- Impact: Merge conflicts on route changes, cannot regenerate without overwriting
- Fix approach: Add to .gitignore, generate at build time via `bun run gen`

**Default JWT Secret in Production Detection:**
- Issue: Uses "dev-secret-change-in-production" fallback when JWT_SECRET not set, only fails in production/staging explicitly
- Files: `cmd/server/main.go:31-36`
- Impact: Accidental production deployments with weak JWT secret
- Fix approach: Require JWT_SECRET in all non-dev environments via environment validation

## Known Bugs

**Context Value Type Assertions Without Nil Checks:**
- Symptoms: `r.Context().Value(RoleKey).(string)` can panic if type assertion fails
- Files: `internal/middleware/middleware.go:48`, `internal/middleware/middleware.go:68-91`
- Trigger: Malformed context or middleware chain issues
- Workaround: Use type-safe getters with comma-ok idiom

**Error Message Leakage:**
- Symptoms: Internal error messages exposed to clients via `err.Error()`
- Files: `internal/adapters/primary/http/auth.go:62`, `internal/adapters/primary/http/auth.go:231`
- Trigger: Any unexpected error during register/bootstrap operations
- Workaround: None currently - errors bubble up directly

## Security Considerations

**Unused Role-Based Authorization Middleware:**
- Risk: `RequireRole` middleware exists but is never applied to any endpoint
- Files: `internal/middleware/middleware.go:46-65`, all handlers in `internal/adapters/primary/http/`
- Current mitigation: None - any authenticated user can access any endpoint
- Recommendations: Audit each endpoint's required role and apply `RequireRole` middleware

**No Rate Limiting on Auth Endpoints:**
- Risk: Login and register endpoints vulnerable to brute force attacks
- Files: `internal/adapters/primary/http/auth.go`, `cmd/server/main.go`
- Current mitigation: None
- Recommendations: Add rate limiting middleware specifically for `/auth/login` and `/auth/register`

**Missing Input Validation:**
- Risk: No validation layer for request payloads - directly passed to services
- Files: All handlers in `internal/adapters/primary/http/`
- Current mitigation: Basic JSON decode validation only
- Recommendations: Add validation library (e.g., go-playground/validator) or schema validation

## Performance Bottlenecks

**Large Service File:**
- Problem: `internal/core/services/auth/auth.go` at 540 lines handles registration, login, refresh, profile, membership
- Files: `internal/core/services/auth/auth.go`
- Cause: Single service handles all auth concerns with minimal separation
- Improvement path: Split into auth.Service, profile.Service, membership.Service

**Large Handler Files:**
- Problem: HTTP handlers exceed 300+ lines each
- Files: `internal/adapters/primary/http/time_entry.go` (477 lines), `internal/adapters/primary/http/auth.go` (378 lines), `internal/adapters/primary/http/working_group.go` (304 lines)
- Cause: Business logic embedded in handlers rather than delegated to services
- Improvement path: Extract more logic to service layer, reduce handler responsibility to request/response transformation

**N+1 Query Potential in Repositories:**
- Problem: Some list operations may produce multiple queries per item
- Files: `internal/adapters/secondary/surrealdb/time_entry_repository.go`, `internal/adapters/secondary/surrealdb/project_repository.go`
- Cause: Relationships not eagerly fetched
- Improvement path: Use SurrealDB RELATE queries with eager loading

## Fragile Areas

**Middleware Context Propagation:**
- Why fragile: All handlers rely on context values set by Auth middleware; if middleware chain breaks, handlers fail silently
- Files: `internal/middleware/middleware.go`, all handlers
- Safe modification: Ensure Auth middleware always runs before handlers that depend on context
- Test coverage: None - no middleware tests exist

**Database Connection Lifecycle:**
- Why fragile: Single SurrealDB connection, no connection pooling configuration visible
- Files: `internal/db/surreal.go`, `cmd/server/main.go:41-45`
- Safe modification: Add connection pool settings, implement graceful shutdown
- Test coverage: No connection pool tests

**No Graceful Shutdown:**
- Why fragile: Server does not handle SIGTERM, connections may be dropped mid-request
- Files: `cmd/server/main.go`
- Safe modification: Add signal handling for graceful shutdown
- Test coverage: None

## Scaling Limits

**Single Database Connection:**
- Current capacity: One SurrealDB connection instance
- Limit: Cannot scale horizontally without external connection pooling
- Scaling path: Implement connection pool or use SurrealDB cluster mode

**In-Memory Token Refresh:**
- Current capacity: Single refresh promise per browser tab
- Limit: Multiple tabs may cause race conditions on token refresh
- Scaling path: Move refresh state to shared storage or implement proper token coordination

## Dependencies at Risk

**Go 1.26.1:**
- Risk: Modern Go version but some dependencies may lag
- Impact: Security patches, performance improvements
- Migration plan: Keep current, monitor dependency updates

**SurrealDB:**
- Risk: Less mature than traditional RDBMS, less community support
- Impact: Driver issues, query optimization challenges
- Migration plan: Monitor SurrealDB releases, keep drivers updated

**React 19:**
- Risk: Very new, some libraries may have compatibility issues
- Impact: Component library issues, hook behavior changes
- Migration plan: Test thoroughly on updates, maintain narrow dependency range

## Missing Critical Features

**Test Infrastructure:**
- Problem: No centralized test setup, each test file has duplicated helpers
- Blocks: Consistent test coverage, integration testing

**Structured Logging:**
- Problem: Uses basic log.Printf instead of structured JSON logging
- Blocks: Log aggregation, error tracking, performance monitoring

**API Versioning:**
- Problem: No API versioning strategy
- Blocks: Backward-compatible changes, deprecated endpoints

## Test Coverage Gaps

**Service Layer - Zero Coverage:**
- What's not tested: All 11 services in `internal/core/services/*`
- Files: `contract/`, `unit/`, `organization/`, `password_reset/`, `project/`, `customer/`, `export/`, `working_group/`, `invitation/`
- Risk: Business logic regressions go undetected
- Priority: HIGH

**Repository Layer - Minimal Coverage:**
- What's not tested: 10+ repositories in `internal/adapters/secondary/surrealdb/*`
- Files: `contract_repository.go`, `project_repository.go`, `customer_repository.go`, `unit_repository.go`, etc.
- Risk: Data access bugs, query errors
- Priority: HIGH

**Frontend - Zero Coverage:**
- What's not tested: All React components, hooks, utilities
- Files: `web/src/components/`, `web/src/hooks/`, `web/src/lib/`
- Risk: UI regressions, broken interactions
- Priority: MEDIUM

**Handler Authorization Checks:**
- What's not tested: Role-based access validation in handlers
- Files: All files in `internal/adapters/primary/http/`
- Risk: Authorization bypasses
- Priority: HIGH

---

*Concerns audit: 2026-05-12*