# Auth Completion Design

**Date**: 2026-05-01
**Status**: Draft
**Approach**: Fix in-place, extend existing handlers (Approach A)

## Context

The auth system has a cookie-based architecture (HttpOnly cookies, not Bearer tokens) with critical bugs and missing features. The backend uses SurrealDB via hexagonal architecture. The frontend uses TanStack Router + TanStack Query.

### Known Bugs (Critical)

| Bug | Location | Impact |
|-----|----------|--------|
| `GET /auth/me` context key mismatch | `auth.go:192` | Reads `ctx.Value("user_id")` as `string`, but middleware sets `contextKey("userID")` as `uuid.UUID`. Runtime panic. |
| `POST /auth/logout` doesn't revoke refresh tokens | `auth.go` | Clears cookies client-side but never calls `authService.Logout()`. Stolen tokens remain valid. |
| Login/Refresh hardcode role=`employee`, org=`uuid.Nil` | `auth.go` service | JWT always has wrong claims. No multi-org support. |
| Register doesn't create org membership | `auth.go` service | Users created orphaned from organizations. |
| `_authenticated.tsx` has no redirect on auth failure | Frontend | If `/auth/me` returns 401, app crashes or shows blank. |

### Missing Features

- No bootstrap UI page (API exists, no frontend)
- No org switching (users stuck on single org)
- No authenticated route guard (logged-in users can visit /login)
- No refresh dedup (race conditions with multiple tabs)
- `GET /auth/me` returns flat profile — frontend expects `UserWithMembership`
- Password reset returns code in response (acceptable for dev, needs email for prod)
- Invitation accept handler returns placeholder message

## User Stories

### Authentication Core

**US-1: Login** — As an employee, I can log in with email or username + password to access the app.

**US-2: Logout** — As a logged-in user, I can log out so my session ends and my refresh token is revoked server-side.

**US-3: Session Refresh** — As a logged-in user, when my access token expires, the system silently refreshes it using my HttpOnly refresh cookie.

**US-4: Bootstrap (First-Time Setup)** — As the very first visitor, when no users exist, I'm redirected to a bootstrap page where I create the initial organization and admin account.

### Registration & Onboarding

**US-5: Register (New Organization)** — As a new user, I can register by creating a new organization. I become an employee of that org.

**US-6: Register (Join via Invite)** — As an invited user, I can register with an invite code to join an existing organization.

### Password Reset

**US-7: Request Password Reset** — As a user who forgot my password, I can submit my email/username to receive a reset code (returned in API response for dev, via email for prod).

**US-8: Verify Password Reset** — As a user with a reset code, I can submit the code + new password to reset my password.

### Profile & Organization

**US-9: View Profile** — As a logged-in user, I can view my profile including name, email, username, role, and organization.

**US-10: Switch Active Organization** — As a user belonging to multiple organizations, I can switch my active organization so my JWT reflects the correct org and role.

### Security & Session

**US-11: Protected Route Guard** — As an unauthenticated user visiting a protected page, I'm redirected to /login.

**US-12: Authenticated Route Guard** — As a logged-in user visiting /login or /register, I'm redirected to /.

**US-13: Concurrent Refresh Dedup** — As a user with multiple tabs, when my access token expires, only one refresh request is made.

**US-14: Invalid Session Handling** — As a user whose refresh token has been revoked, when I try to refresh, I'm redirected to /login.

## Architecture

### Token Model: Single Active Org Per Token

JWT carries one `orgID` + `role` pair. User picks active org on login (default: first active membership). Switching org reissues JWT+refresh.

**Why not all orgs in JWT?** Heavier tokens. Roles change. Need DB query anyway for latest state.

**Why not slim JWT + DB lookup per request?** More DB load. Current middleware validates JWT without DB hit — keep that performance.

### Auth Flow

```
Register:
  POST /auth/register
  → Validate input
  → Create user (hash password)
  → Create organization (if new org)
  → Create OrganizationMembership (role=employee)
  → Generate JWT {userID, orgID, role, email} + refresh token
  → Set HttpOnly cookies
  → Return { user, membership, organization }

Login:
  POST /auth/login
  → Validate credentials (email or username + password)
  → Look up active memberships
  → Default to first active membership
  → Generate JWT {userID, orgID, role, email} + refresh token
  → Set HttpOnly cookies
  → Return { user, membership, organization }

Refresh:
  POST /auth/refresh
  → Validate refresh token from cookie
  → Rotate refresh token (issue new refresh token, revoke old one)
  → Look up membership for the org in the expired JWT's claims
  → Verify membership is still active
  → Generate new JWT (same org/role) + new refresh token
  → Set HttpOnly cookies

Logout:
  POST /auth/logout
  → Revoke refresh token server-side (hash lookup + set revoked_at)
  → Clear both HttpOnly cookies (expiry in past)
  → Frontend: navigate to /login

Switch Org:
  POST /auth/switch-organization
  → Verify user has active membership in target org
  → Generate new JWT {userID, newOrgID, newRole, email} + new refresh token
  → Set HttpOnly cookies
  → Return { user, membership, organization }
  → Frontend: update ['auth', 'me'] cache

Bootstrap:
  GET /auth/bootstrap-check → { needs_bootstrap: bool }
  POST /auth/bootstrap
  → Verify zero users exist
  → Create organization + user (role=admin) + membership
  → Generate JWT {userID, orgID, role=admin, email} + refresh token
  → Set HttpOnly cookies
  → Return { user, membership, organization }

View Profile:
  GET /auth/me (requires Auth middleware)
  → Extract userID from context via middleware.GetUserID(ctx)
  → Extract orgID from context via middleware.GetOrganizationID(ctx)
  → Fetch user + membership + organization
  → Return { user, membership, organization }

Password Reset:
  POST /auth/password-reset/request
  → Find user by identifier
  → Generate reset code, store with expiry
  → Return { message: "Reset code generated" } + code in response (dev only)

  POST /auth/password-reset/verify
  → Validate code + identifier
  → Update password hash
  → Revoke all refresh tokens for user (force re-login)
  → Return { message: "Password reset successful" }
```

### Cookie Configuration

| Cookie | Name | Max-Age | HttpOnly | Secure | SameSite | Path |
|--------|------|---------|----------|--------|----------|------|
| Access token | `auth_token` | 15 min | Yes | Prod only | Lax | / |
| Refresh token | `refresh_token` | 7 days | Yes | Prod only | Lax | /auth |

### JWT Claims

```json
{
  "user_id": "uuid",
  "organization_id": "uuid",
  "role": "employee",
  "email": "user@example.com",
  "exp": 1746140400,
  "iat": 1746139500
}
```

## Backend Changes

### Bug Fixes

1. **`GET /auth/me` context key fix** — Replace `ctx.Value("user_id").(string)` with `middleware.GetUserID(ctx)` in `internal/adapters/primary/http/auth.go`. The helper already exists and handles the typed context key correctly.

2. **`POST /auth/logout` revocation** — After clearing cookies, call `authService.Logout(ctx, refreshToken)` to revoke the refresh token server-side. Read the refresh token from the `refresh_token` cookie before clearing it.

3. **Login/Refresh membership lookup** — In `internal/core/services/auth/auth.go`:
   - Add `GetMemberships(ctx, userID)` call after credential validation
   - Default to first active membership if user has one
   - Return structured error if user has no memberships
   - Use the membership's orgID and role in JWT claims instead of hardcoded values

4. **Register creates membership** — In `AuthService.Register`:
   - After creating user and org, create `OrganizationMembership` record
   - Set `role=employee`, `is_active=true`, `activated_at=now`
   - Include membership in response

### New Repository Method

Add to `UserRepository` port (`internal/core/ports/user_repository.go`):

```go
GetMemberships(ctx context.Context, userID uuid.UUID) ([]domain.OrganizationMembership, error)
```

Add to `OrganizationRepository` port:

```go
GetMembership(ctx context.Context, userID uuid.UUID, orgID uuid.UUID) (*domain.OrganizationMembership, error)
```

Implement both in SurrealDB adapter.

### New Domain Type

Add `OrganizationMembership` to `internal/core/domain/auth/`:

```go
type OrganizationMembership struct {
    ID             uuid.UUID
    UserID         uuid.UUID
    OrganizationID uuid.UUID
    Role           string
    IsActive       bool
    InvitedBy      *uuid.UUID
    InvitedAt      *time.Time
    ActivatedAt    *time.Time
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

### New Endpoints

5. **`POST /auth/switch-organization`**
   - Request: `{ "organization_id": "uuid" }`
   - Requires Auth middleware
   - Service: `SwitchOrganization(ctx, userID, orgID)` — verify membership, generate new token pair
   - Response: `{ user, membership, organization }`

6. **`GET /auth/bootstrap-check`**
   - No auth required
   - Service: check if any users exist (reuse `UserRepository.AnyExists`)
   - Response: `{ "needs_bootstrap": true/false }`

### Response Shape Change

`GET /auth/me` and `POST /auth/login` return `UserWithMembership`:

```json
{
  "data": {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "username": "jdoe",
      "name": "John Doe",
      "is_active": true,
      "created_at": "2026-05-01T00:00:00Z"
    },
    "membership": {
      "id": "uuid",
      "user_id": "uuid",
      "organization_id": "uuid",
      "role": "employee",
      "is_active": true,
      "activated_at": "2026-05-01T00:00:00Z"
    },
    "organization": {
      "id": "uuid",
      "name": "Acme Corp",
      "slug": "acme-corp",
      "created_at": "2026-05-01T00:00:00Z"
    }
  }
}
```

## Frontend Changes

### Bug Fixes

1. **`_authenticated.tsx` redirect on auth failure** — Wrap `fetchQuery` in try/catch. On error, `throw redirect({ to: '/login' })`. Remove redundant `ensureQueryData` call since `fetchQuery` already caches.

2. **`profile-menu.tsx` redirect after logout** — In the logout mutation's `onSuccess`, navigate to `/login` using `useNavigate()`.

### New Features

3. **Refresh dedup in `api.ts`** — Track in-flight refresh promise globally. If a refresh is already underway, return the existing promise instead of firing a new one.

4. **Authenticated route guard** — Add `beforeLoad` to `(auth)` layout route: if `['auth', 'me']` cache has data, redirect to `/`. Alternatively, check in each auth page's `beforeLoad`.

5. **`types/api.ts` updates**:
   - Add `username: string` to `User` type
   - Verify `AuthResponse` shape matches backend `UserWithMembership`
   - Remove duplicate `Contract` interface
   - Remove unused `token` field from `AuthResponse` (auth is cookie-based)

6. **Bootstrap page** — New route `(auth)/bootstrap/`:
   - `beforeLoad`: call `GET /auth/bootstrap-check`. If `needs_bootstrap` is false, redirect to `/login`.
   - Form: org name, email, username, firstname, lastname, password
   - On success: navigate to `/`

7. **Org switcher component** — In profile menu or app header:
   - Fetch user's memberships (new API call or included in profile response)
   - Show current org name
   - Dropdown to switch org
   - Calls `POST /auth/switch-organization`, updates `['auth', 'me']` cache

8. **`Forgot password?` link** — Add link from login form to `/password-reset`.

### Auth Layout Route

Create `web/src/routes/(auth).tsx` with shared layout:
- Centered card wrapper (replaces duplicated `min-h-screen flex items-center justify-center` per page)
- `beforeLoad`: if user is authenticated, redirect to `/`

## Non-Goals (Deferred)

- Email delivery for password reset codes (use in-response for dev)
- Invitation accept handler completion (placeholder remains until invitations feature is complete)
- Stale PostgreSQL migration cleanup
- Debug printf removal in SurrealDB adapter
- Bearer token auth support
- Remember me / extended session
- Rate limiting on auth endpoints
- Account lockout after failed attempts
