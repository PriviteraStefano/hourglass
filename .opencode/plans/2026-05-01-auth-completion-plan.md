# Plan: Auth Completion

> Source PRD: `.opencode/plans/2026-05-01-auth-completion-design.md`

## Progress Tracker

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 1: Critical Bug Fixes | **In Progress** | 3/5 done. Profile-menu logout redirect + cookie name verification remaining |
| Phase 2: Login + Register + Membership | Pending | |
| Phase 3: Bootstrap UI | Pending | |
| Phase 4: Auth Route Guards + Session Resilience | Pending | |
| Phase 5: Org Switching | Pending | |
| Phase 6: Password Reset + UX Polish | Pending | |

### Phase 1 completed edits:

1. **`GET /auth/me` context key fix** — DONE. Replaced `ctx.Value("user_id").(string)` with `middleware.GetUserID(ctx)` + nil check. Added `uuid` and `middleware` imports. File: `internal/adapters/primary/http/auth.go`
2. **`POST /auth/logout` revocation** — DONE. Added refresh token read from cookie + `authService.Logout()` call before clearing cookies. File: `internal/adapters/primary/http/auth.go`
3. **`_authenticated.tsx` redirect** — DONE. Wrapped `fetchQuery` in try/catch with `redirect({ to: '/login' })`. Removed redundant `ensureQueryData`. File: `web/src/routes/_authenticated.tsx`
4. **`profile-menu.tsx` logout redirect** — NOT YET DONE. Needs `useNavigate()` + navigate to `/login` on logout success
5. **Cookie name verification** — NOT YET DONE

## Architectural decisions

- **Routes**: `/auth/login`, `/auth/register`, `/auth/logout`, `/auth/refresh`, `/auth/me`, `/auth/switch-organization`, `/auth/bootstrap`, `/auth/bootstrap-check`, `/auth/password-reset/request`, `/auth/password-reset/verify`
- **Auth model**: Cookie-based (HttpOnly). Access token 15min in `auth_token` cookie. Refresh token 7d in `refresh_token` cookie. No Bearer header.
- **JWT claims**: `{ user_id, organization_id, role, email, exp, iat }`. Single active org per token.
- **Response shape**: All auth responses return `UserWithMembership`: `{ user, membership, organization }`
- **Token rotation**: Refresh tokens are rotated on each refresh (old revoked, new issued)
- **DB**: SurrealDB via hexagonal ports/adapters

---

## Phase 1: Critical Bug Fixes

**User stories**: US-1 (Login), US-2 (Logout), US-3 (Session Refresh), US-11 (Protected Route Guard), US-14 (Invalid Session Handling)

### What to build

Fix the 5 runtime-breaking bugs so the existing login→refresh→profile→logout loop works end-to-end. No new features — just make what exists actually run.

### Backend fixes

1. **`GET /auth/me` context key** — In `internal/adapters/primary/http/auth.go`, replace `ctx.Value("user_id").(string)` with `middleware.GetUserID(ctx)`. Use `middleware.GetOrganizationID(ctx)` for org. This fixes the runtime panic.

2. **`POST /auth/logout` revocation** — In the Logout handler, read `refresh_token` cookie before clearing. Call `authService.Logout(ctx, refreshToken)` to revoke server-side. Then clear both cookies.

3. **`_authenticated.tsx` auth failure redirect** — Wrap `client.fetchQuery(AuthApis.profileQueryOpts)` in try/catch. On error, `throw redirect({ to: '/login' })`. Remove redundant `ensureQueryData` call.

4. **`profile-menu.tsx` logout redirect** — After logout mutation, navigate to `/login` via `useNavigate()`.

5. **Login handler refresh token cookie read** — Verify the login handler correctly sets both cookies. Ensure cookie names match between set (handler) and read (middleware, refresh, logout).

### Acceptance criteria

- [x] `GET /auth/me` returns 200 with user profile (no panic) when authenticated
- [x] `POST /auth/logout` revokes refresh token server-side and clears cookies
- [x] Visiting a protected page while unauthenticated redirects to `/login`
- [ ] Logging out redirects to `/login` — **profile-menu.tsx edit remaining**
- [ ] Login → profile → logout → redirect works end-to-end — **needs manual testing**

---

## Phase 2: Login + Register + Membership

**User stories**: US-1 (Login), US-5 (Register New Org), US-6 (Register Join via Invite), US-9 (View Profile)

### What to build

Fix the hardcoded role/org in Login and Refresh. Make Register create org memberships. Change `GET /auth/me` response shape to `UserWithMembership`. Wire the frontend types to match the new backend shape.

### Backend

1. **`OrganizationMembership` domain type** — Add to `internal/core/domain/auth/`. Struct with ID, UserID, OrganizationID, Role, IsActive, timestamps.

2. **Repository methods** — Add `GetMemberships(ctx, userID)` to `UserRepository` port. Add `GetMembership(ctx, userID, orgID)` to `OrganizationRepository` port. Add `AddMembership(ctx, membership)` to `OrganizationRepository` port. Implement all in SurrealDB adapter.

3. **`AuthService.Register`** — After creating user and org, create `OrganizationMembership` with `role=employee`, `is_active=true`. Return `UserWithMembership` (user + membership + org).

4. **`AuthService.Login`** — After credential validation, call `GetMemberships`. Default to first active membership. Use membership's orgID and role in JWT claims instead of hardcoded values. Return `UserWithMembership`.

5. **`AuthService.Refresh`** — Extract orgID from expired JWT claims. Look up membership for that orgID. Verify still active. Use membership's role in new JWT claims.

6. **`AuthService.GetProfile`** — Accept orgID (from JWT context). Fetch user + membership (by userID + orgID) + organization. Return `UserWithMembership`.

7. **`AuthHandler.GetProfile`** — Use `middleware.GetUserID(ctx)` and `middleware.GetOrganizationID(ctx)` to pass both to service.

8. **Response shape** — All auth endpoint responses now return:
   ```json
   { "data": { "user": {...}, "membership": {...}, "organization": {...} } }
   ```

### Frontend

9. **`types/api.ts`** — Add `username` to `User`. Add `UserWithMembership` type. Update `AuthResponse` to wrap `UserWithMembership` (drop `token` field). Remove duplicate `Contract` interface.

10. **`api/auth.ts`** — Update `profileQueryOpts`, `loginMutationOpts`, `registerMutationOpts`, `bootstrapMutationOpts` to use `UserWithMembership` response shape. Fix cache setters to store the nested structure.

### Acceptance criteria

- [ ] Register creates user + org + membership. User is not orphaned.
- [ ] Login returns JWT with correct orgID and role from membership
- [ ] `GET /auth/me` returns `{ user, membership, organization }` matching frontend type
- [ ] Frontend profile query correctly reads and caches `UserWithMembership`
- [ ] Refresh token uses correct org/role from membership

---

## Phase 3: Bootstrap UI

**User stories**: US-4 (Bootstrap)

### What to build

Add `GET /auth/bootstrap-check` endpoint. Add a bootstrap page that only shows when no users exist. First visitor creates the org and admin account.

### Backend

1. **`GET /auth/bootstrap-check`** — No auth required. Call `UserRepository.AnyExists`. Return `{ needs_bootstrap: true/false }`.

2. **`POST /auth/bootstrap`** — Update to create `OrganizationMembership` with `role=admin`. Return `UserWithMembership`. Set both cookies (access + refresh).

3. **Register route in `main.go`** — `GET /auth/bootstrap-check → hexAuthHandler.BootstrapCheck`

### Frontend

4. **Bootstrap API** — Add `bootstrapCheckQueryOpts` to `auth.ts`: `GET /auth/bootstrap-check`, returns `{ needs_bootstrap: boolean }`.

5. **Bootstrap page** — New route `(auth)/bootstrap/`:
   - `beforeLoad`: fetch `bootstrapCheckQueryOpts`. If `needs_bootstrap` is false, redirect to `/login`.
   - Form: org name, email, username, firstname, lastname, password
   - On success: navigate to `/`

6. **Root route bootstrap redirect** — In `__root.tsx` or `_authenticated.tsx` `beforeLoad`: if `bootstrap-check` says `needs_bootstrap: true`, redirect to `/bootstrap`. Only run this check when no auth cache exists.

### Acceptance criteria

- [ ] `GET /auth/bootstrap-check` returns `{ needs_bootstrap: true }` when no users exist
- [ ] `GET /auth/bootstrap-check` returns `{ needs_bootstrap: false }` after first user created
- [ ] Visiting app with no users redirects to `/bootstrap`
- [ ] Bootstrap form creates org + admin user + membership
- [ ] After bootstrap, user is logged in and redirected to `/`
- [ ] Bootstrap page is inaccessible after first user exists

---

## Phase 4: Auth Route Guards + Session Resilience

**User stories**: US-12 (Authenticated Route Guard), US-13 (Concurrent Refresh Dedup), US-14 (Invalid Session Handling)

### What to build

Prevent logged-in users from accessing auth pages. Deduplicate concurrent refresh requests. Handle invalid sessions gracefully.

### Frontend

1. **`(auth).tsx` layout route** — Create `web/src/routes/(auth).tsx`:
   - Shared centered card layout (replace duplicated layout CSS in each auth page)
   - `beforeLoad`: check if `['auth', 'me']` cache has data. If yes, redirect to `/`.
   - Use try/catch on cache check — if query throws, allow through (user is not authenticated)

2. **Refresh dedup in `api.ts`** — Add module-level `refreshPromise: Promise<void> | null`. In 401 handler:
   - If `refreshPromise` exists, await it instead of creating new one
   - If not, create refresh request, assign to `refreshPromise`, clear on settle
   - On refresh failure, redirect to `/login`

3. **Invalid session handling** — If refresh fails (401/403), clear auth cache, redirect to `/login`. This already partially exists in `api.ts` but needs the dedup integration.

4. **Remove per-page layout duplication** — Remove `min-h-screen flex items-center justify-center` from individual auth form pages. Use `(auth).tsx` layout instead.

### Acceptance criteria

- [ ] Logged-in user visiting `/login` is redirected to `/`
- [ ] Logged-in user visiting `/register` is redirected to `/`
- [ ] Multiple simultaneous 401s trigger only one `/auth/refresh` call
- [ ] Refreshed tab gets new access token and retries its request
- [ ] Revoked refresh token → redirect to `/login` on all tabs

---

## Phase 5: Org Switching

**User stories**: US-10 (Switch Active Organization)

### What to build

Add org switching endpoint. Add org switcher UI in the profile menu/header.

### Backend

1. **`SwitchOrganization` service method** — `SwitchOrganization(ctx, userID, targetOrgID)`:
   - Call `GetMembership(ctx, userID, targetOrgID)` to verify user belongs to org
   - Verify membership is active
   - Generate new JWT with updated orgID + role + new refresh token
   - Revoke old refresh token
   - Return `UserWithMembership`

2. **`POST /auth/switch-organization`** — New route. Requires Auth middleware. Request body: `{ "organization_id": "uuid" }`. Set both cookies. Return `UserWithMembership`.

3. **Register route in `main.go`** — `POST /auth/switch-organization → middleware.Auth(authService, hexAuthHandler.SwitchOrganization)`

### Frontend

4. **Switch org API** — Add `switchOrganizationMutationOpts` to `auth.ts`: `POST /auth/switch-organization`, body `{ organization_id }`. On success, update `['auth', 'me']` cache with returned `UserWithMembership`.

5. **`GetMemberships` query** — Add `membershipsQueryOpts` to `auth.ts`: endpoint to get user's memberships list. Alternatively, include memberships list in `GET /auth/me` response.

6. **Org switcher component** — In profile menu:
   - Display current org name from cached profile
   - Dropdown with all memberships
   - On select: call switch org mutation
   - On success: cache updates → UI refreshes with new org context

### Acceptance criteria

- [ ] `POST /auth/switch-organization` with valid orgID returns new `UserWithMembership` with correct org/role
- [ ] After switch, JWT cookie contains new orgID and role
- [ ] Org switcher shows current org and lists all memberships
- [ ] Switching org updates profile cache and all dependent UI
- [ ] Switching to an org user doesn't belong to returns 403

---

## Phase 6: Password Reset + UX Polish

**User stories**: US-7 (Request Password Reset), US-8 (Verify Password Reset), US-1 (Login UX)

### What to build

Fix password reset to revoke tokens on password change. Add forgot password link to login. Clean up frontend types and small UX issues.

### Backend

1. **`POST /auth/password-reset/verify`** — After updating password hash, revoke all refresh tokens for the user (force re-login on all devices). Add `RevokeAllByUser(ctx, userID)` to `RefreshTokenRepository` port. Implement in SurrealDB adapter.

### Frontend

2. **Forgot password link** — Add link from login form to `/password-reset`.

3. **`InvitationAcceptForm` async default** — Fix form default values: use `useEffect` + `reset()` to update email field when invitation data loads, instead of relying on initial `useForm` defaults.

4. **Type cleanup** — Remove unused `token` field from `AuthResponse`. Remove commented-out Zod schema from `types/api.ts`. Remove commented-out `header.tsx`.

### Acceptance criteria

- [ ] Password reset verify revokes all refresh tokens for user
- [ ] After resetting password, user must log in again (old sessions invalidated)
- [ ] Login form has "Forgot password?" link pointing to `/password-reset`
- [ ] Invitation accept form email field populates from async invitation data
- [ ] No dead code or unused type fields remain
