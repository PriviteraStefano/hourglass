# Hexagonal Architecture Migration Plan

## Overview

Migrating from handler-based architecture to hexagonal (ports & adapters) architecture.

**Reference repo**: [samverrall/hex-structure](https://github.com/samverrall/hex-structure)

## Hexagonal Core Concepts

| Concept | Description |
|---------|-------------|
| **Domain** | Pure business entities and value objects. Zero external dependencies. |
| **Ports** | Interfaces (contracts) that define what the application needs. |
| **Services** | Application logic. Depends on ports, not implementations. |
| **Primary Adapter** | Driving adapter (HTTP, CLI, queue consumer). Initiates action. |
| **Secondary Adapter** | Driven adapter (DB, external API, cache). Implements ports. |

### Naming Rationale
- **Primary/Secondary** = role (driving/driven), stable
- **http/surrealdb** = implementation, swappable
- Multiple primary adapters possible (HTTP + gRPC). Multiple secondary (DB + cache).

## Target Directory Structure

```
internal/
├── core/
│   ├── domain/
│   │   └── auth/
│   │       ├── user.go           # User entity
│   │       ├── credentials.go     # Email, password value objects
│   │       ├── errors.go         # Domain errors
│   │       └── refresh_token.go  # Refresh token entity
│   ├── services/
│   │   └── auth/
│   │       ├── auth.go           # AuthService struct + interface
│   │       ├── register.go       # Register use case
│   │       ├── login.go          # Login use case
│   │       ├── refresh.go        # Refresh use case
│   │       └── logout.go         # Logout use case
│   └── ports/
│       ├── user_repository.go    # UserRepository port interface
│       ├── token_service.go     # Token generation/validation port
│       ├── password_hasher.go   # Password hashing port
│       └── refresh_token_repo.go # Refresh token storage port
├── adapters/
│   ├── primary/
│   │   └── http/
│   │       └── auth.go           # HTTP handlers (thin, delegates to service)
│   └── secondary/
│       └── surrealdb/
│           ├── user_repository.go # Implements UserRepository
│           └── refresh_token_repo.go # Implements RefreshTokenRepository
└── ports/                        # Port interfaces (same as core/ports, for clarity)
```

## Migration: Auth Handler

### ✅ Phase 1: Create Domain Layer (COMPLETE)
**Files created:**
- `internal/core/domain/auth/user.go` — User entity
- `internal/core/domain/auth/credentials.go` — Email, Password, Username value objects
- `internal/core/domain/auth/errors.go` — Domain errors

### ✅ Phase 2: Create Ports Layer (COMPLETE)
**Files created:**
- `internal/core/ports/user_repository.go` — `UserRepository` interface
- `internal/core/ports/token_service.go` — `TokenService` interface
- `internal/core/ports/password_hasher.go` — `PasswordHasher` interface
- `internal/core/ports/refresh_token_repo.go` — `RefreshTokenRepository` interface

### ✅ Phase 3: Create Auth Service (COMPLETE)
**Files created:**
- `internal/core/services/auth/auth.go` — Service struct with Register, Login, GetProfile, Refresh, Logout

**Status:** All business logic in service layer. Depends on ports (interfaces), not implementations.

### ✅ Phase 4: Create Secondary Adapters (COMPLETE)
**Files created:**
- `internal/adapters/secondary/surrealdb/user_repository.go` — Implements UserRepository
- `internal/adapters/secondary/surrealdb/password_hasher.go` — Implements PasswordHasher
- `internal/adapters/secondary/surrealdb/token_service.go` — Implements TokenService
- `internal/adapters/secondary/surrealdb/refresh_token_repo.go` — Implements RefreshTokenRepository

### ✅ Phase 5: Create Primary Adapter (HTTP Handler) (COMPLETE)
**File created:**
- `internal/adapters/primary/http/auth.go` — Thin HTTP handler (Register, Login, GetProfile, Refresh, Logout)

### ✅ Phase 6: Update main.go Wiring (COMPLETE)
**File updated:**
- `cmd/server/main.go` — Wires hexagonal auth components

### ✅ Phase 7: Refresh Token Flow (COMPLETE)
- Login now returns refresh token + sets cookie
- Refresh endpoint validates token, creates new access token
- Logout revokes refresh token

### ✅ Phase 8: Remaining Handler Migrations (PARTIALLY COMPLETE)
- Completed: organization, project, contract, customer, export, approval, legacy time-entry cleanup, expense handler retirement
- Remaining live gaps: bootstrap stub, registration/org bootstrap, and tests

### 📋 Remaining Items
- [ ] Bootstrap endpoint (stubbed as 501)
- [ ] Organization creation during register
- [ ] Tests

## Handler Migration Pattern (for future handlers)

When migrating other handlers (Unit, WorkingGroup, TimeEntry), follow same pattern:

1. Extract domain entities/value objects
2. Define ports as interfaces
3. Move business logic to service layer
4. Create secondary adapters implementing ports
5. Create thin HTTP handler (primary adapter)
6. Update wiring in main.go

## Key Files After Migration

| File | Responsibility |
|------|----------------|
| `internal/core/domain/auth/user.go` | User entity |
| `internal/core/ports/*.go` | Interface definitions |
| `internal/core/services/auth/*.go` | All auth business logic |
| `internal/adapters/secondary/surrealdb/*.go` | SurrealDB implementations |
| `internal/adapters/primary/http/auth.go` | HTTP handler |
| `cmd/server/main.go` | Wiring |

## Notes

- `internal/auth/` (current JWT service) stays as-is initially. It will be wrapped by `TokenService` port.
- `internal/db/` stays as-is. Secondary adapters use it.
- `internal/middleware/` stays as-is. Works with primary adapter.
- `pkg/api/` stays as-is. Used by primary adapter for responses.
