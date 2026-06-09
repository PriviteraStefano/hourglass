# Phase 01: Authorization - Pattern Map

**Mapped:** 2026-06-10
**Files analyzed:** 7
**Analogs found:** 7 / 7

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/adapters/primary/http/auth.go` | controller (handler) | request-response | `Login` handler in same file | exact |
| `internal/core/services/password_reset/password_reset.go` | service | CRUD | `auth.Service` in `internal/core/services/auth/auth.go` | role-match |
| `internal/adapters/primary/http/password_reset.go` | controller (handler) | request-response | `Login` handler in `internal/adapters/primary/http/auth.go` | role-match |
| `web/src/components/app/org-switcher.tsx` | component | request-response | `ProfileMenu` in `web/src/components/app/profile-menu.tsx` | exact |
| `web/src/components/layout/sidebar.tsx` | component | request-response | `AppSidebar` (self, line 60) | partial |
| `web/src/api/auth.ts` | utility (api module) | request-response | Existing `*MutationOpts` / `*QueryOpts` in same file | exact |
| `web/src/routes/_authenticated/index.tsx` | route | request-response | `web/src/routes/_auth/login/index.tsx` | role-match |

---

## Pattern Assignments

### `internal/adapters/primary/http/auth.go` (controller, request-response)

**Modification:** Register handler — add cookie setting after successful registration.

**Analog:** `Login` handler (lines 74-122) and `Bootstrap` handler (lines 170-200) in the **same file**.

**Cookie-setting pattern** (Login, lines 117-121):
```go
secure := cookies.IsSecureRequest(r)
cookies.SetAccessTokenCookie(w, resp.Token, secure)
cookies.SetRefreshTokenCookie(w, resp.RefreshToken, secure)

api.RespondWithJSON(w, http.StatusOK, resp)
```

**Cookie-setting pattern** (Bootstrap, lines 195-199):
```go
secure := cookies.IsSecureRequest(r)
cookies.SetAccessTokenCookie(w, resp.Token, secure)
cookies.SetRefreshTokenCookie(w, resp.RefreshToken, secure)

api.RespondWithJSON(w, http.StatusOK, resp)
```

**Current Register handler** (lines 36-67) — note the **missing** cookie calls:
```go
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.authService.Register(ctx, auth.RegisterRequest{...})
	if err != nil {
		// error handling...
		return
	}

	// BUG: Missing cookie setting — currently just returns JSON
	api.RespondWithJSON(w, http.StatusCreated, resp)
}
```

**What to do:** The `RegisterResponse` (in auth service, line 67-69) only embeds `UserWithMembership` — no `Token`/`RefreshToken` fields. Need to either:
- **Option A:** Extend `auth.Service.Register()` to generate tokens (like Bootstrap does — see auth.go lines 432-445) and add `Token`, `RefreshToken`, `ExpiresAt` fields to `RegisterResponse`. This is the cleaner approach matching the architecture.
- **Option B:** After Register succeeds, call `authService.Login` with the password from the request.

**Token generation pattern to copy** (from Bootstrap, `auth.go` lines 432-445):
```go
token, err := s.tokenService.GenerateToken(tokenUserID, org.ID, "employee", user.Email)
if err != nil {
	return nil, err
}
refreshToken, err := s.tokenService.GenerateRefreshToken()
if err != nil {
	return nil, err
}
refreshHash := s.tokenService.HashRefreshToken(refreshToken)
if err := s.refreshTokenRepo.Add(ctx, user.ID, org.ID, refreshHash, time.Now().Add(7*24*time.Hour)); err != nil {
	return nil, err
}
return &BootstrapResponse{
	Token:        token,
	RefreshToken: refreshToken,
	ExpiresAt:    time.Now().Add(15 * time.Minute),
	UserWithMembership: *buildUserWithMembershipPtr(user, org.ID, org, membership),
}, nil
```

**Error handling pattern** (existing Register, lines 54-65):
```go
if err != nil {
	switch err {
	case auth.ErrEmailExists:
		api.RespondWithError(w, http.StatusConflict, "email already registered")
	case auth.ErrUsernameExists:
		api.RespondWithError(w, http.StatusConflict, "username already taken")
	default:
		api.RespondWithError(w, http.StatusBadRequest, err.Error())
	}
	return
}
```

**Cookie helpers** (`internal/cookies/cookies.go`):
```go
func SetAccessTokenCookie(w http.ResponseWriter, token string, secure bool)
func SetRefreshTokenCookie(w http.ResponseWriter, token string, secure bool)
func IsSecureRequest(r *http.Request) bool  // r.TLS != nil || X-Forwarded-Proto == "https"
```

---

### `internal/core/services/password_reset/password_reset.go` (service, CRUD)

**Modification:** Improve reset code entropy (D-09) and add verification rate limiting (D-10).

**Analog:** `auth.Service` in `internal/core/services/auth/auth.go` — same hexagonal service pattern.

**Current code generation** (lines 100-109) — already 8-char alphanumeric, but has modulo bias:
```go
const resetCodeCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func generateResetCode() string {
	b := make([]byte, 8)
	rand.Read(b)               // crypto/rand — good, but has modulo bias
	result := make([]byte, len(b))
	for i, v := range b {
		result[i] = resetCodeCharset[int(v)%len(resetCodeCharset)]  // MODULO BIAS: 256 % 62 != 0
	}
	return string(result)
}
```

**Fix pattern** — use `crypto/rand.Int` for unbiased distribution:
```go
import "crypto/rand"
import "math/big"

func generateResetCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	code := make([]byte, 8)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(err) // or fall back to biased version
		}
		code[i] = charset[n.Int64()]
	}
	return string(code)
}
```

**Service constructor pattern** (from `password_reset.go` lines 22-38):
```go
type Service struct {
	repo             ports.PasswordResetRepository
	userRepo         ports.UserRepository
	userFinder       ports.UserFinder
	hasher           ports.PasswordHasher
	tokenService     ports.TokenService
	refreshTokenRepo ports.RefreshTokenRepository
}

func NewService(
	repo ports.PasswordResetRepository,
	userRepo ports.UserRepository,
	userFinder ports.UserFinder,
	hasher ports.PasswordHasher,
	tokenService ports.TokenService,
	refreshTokenRepo ports.RefreshTokenRepository,
) *Service {
	return &Service{...}
}
```

**Request handler pattern** (from `password_reset.go` lines 40-68):
```go
func (s *Service) Request(ctx context.Context, identifier string) (code string, expiresAt time.Time, err error) {
	userID, err := s.userFinder.FindByIdentifier(ctx, identifier)
	if err != nil {
		return "", time.Time{}, password_reset.ErrUserNotFound
	}
	code = generateResetCode()
	codeHash, err := s.hasher.Hash(code)
	// ... store in repo ...
	return code, expiresAt, nil
}
```

**Rate-limit pattern** (already wired in `cmd/server/main.go` line 85, apply the `passwordResetRateLimiter` middleware):
```go
passwordResetRateLimiter := middleware.NewRateLimiter(3, 60)
mux.Handle("POST /auth/password-reset/request", passwordResetRateLimiter.Middleware(stdhttp.HandlerFunc(passwordResetHandler.Request)))
mux.Handle("POST /auth/password-reset/verify", passwordResetRateLimiter.Middleware(stdhttp.HandlerFunc(passwordResetHandler.Verify)))
```

---

### `internal/adapters/primary/http/password_reset.go` (controller, request-response)

**Verification:** Ensure the reset code is NOT returned in the response body (D-08).

**Current state** — already compliant. The Request handler ignores the code return value (line 44):
```go
_, expiresAt, err := h.service.Request(ctx, req.Identifier)  // code discarded with _
if err != nil {
	// error handling...
	return
}

api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
	"message":    "reset code sent",
	"expires_at": expiresAt,  // No "code" field! Correct.
})
```

**The frontend type must be updated** to remove the `code?: string` field (see pattern under `web/src/api/auth.ts`).

**Verify handler pattern** (lines 60-99) — for reference:
```go
func (h *PasswordResetHandler) Verify(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req VerifyResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// ... validation ...

	err := h.service.Verify(ctx, req.Identifier, req.Code, req.Password)
	if err != nil {
		if err == password_reset.ErrUserNotFound {
			api.RespondWithError(w, http.StatusNotFound, "user not found")
			return
		}
		// ... more error cases ...
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "password reset successful",
	})
}
```

**Request/Response types** (same file, lines 20-28):
```go
type RequestResetRequest struct {
	Identifier string `json:"identifier"`
}

type VerifyResetRequest struct {
	Identifier string `json:"identifier"`
	Code       string `json:"code"`
	Password   string `json:"password"`
}
```

---

### `web/src/components/app/org-switcher.tsx` (component, request-response)

**Modification:** Fetch real memberships, wire org switch mutation, remove `organizations` prop.

**Analog:** `ProfileMenu` in `web/src/components/app/profile-menu.tsx` — exact component type, same auth data consumption pattern.

**ProfileMenu pattern** (lines 17-30) — shows how to consume auth data + use mutations:
```tsx
export function ProfileMenu() {
  const navigate = useNavigate()
  const {data: {user}} = useSuspenseQuery(AuthApis.profileQueryOpts)
  const {mutateAsync: logout} = useMutation(AuthApis.logoutMutationOpts)

  const handleLogout = () => {
    logout().then(() => navigate({to: '/login'}))
  }
  // ... JSX ...
}
```

**OrgSwitcher current state** (lines 16-76) — accepts `organizations` prop with empty array:
```tsx
export function OrgSwitcher({organizations}: {
  organizations: Array<Organization>
}) {
  const {isMobile} = useSidebar()
  const {data: {organization}} = useSuspenseQuery(AuthApis.profileQueryOpts)
  // ... renders current org, empty organizations dropdown ...
}
```

**What to change:**
1. Remove `organizations` prop
2. Add `useSuspenseQuery(AuthApis.membershipsQueryOpts)` to fetch memberships
3. Add `useMutation(AuthApis.switchOrganizationMutationOpts)` for switching
4. Map memberships to dropdown items
5. On switch: call mutation, then invalidate/refetch queries

**The query to use** (from `auth.ts` lines 129-133):
```typescript
const membershipsQueryOpts = queryOptions({
  queryKey: ['auth', 'memberships'],
  queryFn: async () => api<{
    memberships: Array<{
      membership: UserWithMembership['membership'];
      organization: UserWithMembership['organization']
    }>
  }>('/auth/memberships'),
  retry: false,
})
```

**The mutation to use** (from `auth.ts` lines 119-127):
```typescript
const switchOrganizationMutationOpts = mutationOptions({
  mutationFn: (data: { organization_id: string }) =>
    api<AuthResponse>('/auth/switch-organization', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  onSuccess: (data: AuthResponse, _, __, {client}) =>
    client.setQueryData(['auth', 'me'], data.user),
})
```

**Switch onClick pattern** (needs full cache refresh, D-05):
```typescript
// Inside dropdown item onClick:
const {mutateAsync: switchOrg} = useMutation(AuthApis.switchOrganizationMutationOpts)

const handleSwitch = async (orgId: string) => {
  await switchOrg({ organization_id: orgId })
  // D-05: Full data refresh for new org context
  client.invalidateQueries()  // or client.clear()
  // Also refetch memberships
}
```

**The `Organization` type** (from `types/models.ts` lines 14-19):
```typescript
export interface Organization {
  id: string
  name: string
  slug: string
  created_at: string
}
```

---

### `web/src/components/layout/sidebar.tsx` (component, request-response)

**Modification:** Remove hardcoded `organizations={[]}` prop (D-03).

**Current** (line 60):
```tsx
<OrgSwitcher organizations={[]}/>
```

**Target** — since OrgSwitcher will be self-sufficient (fetches its own memberships internally following ProfileMenu pattern), just drop the prop:
```tsx
<OrgSwitcher/>
```

**Analog:** `ProfileMenu` usage in the same file (line 151) — it's already used without any props:
```tsx
<SidebarFooter>
  <ProfileMenu/>
  <ThemeToggle/>
</SidebarFooter>
```

---

### `web/src/api/auth.ts` (utility, request-response)

**Modification:** Update `requestPasswordResetMutationOpts` response type to remove optional `code` field (D-11).

**Current** (lines 97-103):
```typescript
const requestPasswordResetMutationOpts = mutationOptions({
  mutationFn: (data: PasswordResetRequest) =>
    api<{ message: string; code?: string }>('/auth/password-reset/request', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
})
```

**Target** — remove `code?` from response type:
```typescript
const requestPasswordResetMutationOpts = mutationOptions({
  mutationFn: (data: PasswordResetRequest) =>
    api<{ message: string }>('/auth/password-reset/request', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
})
```

**Pattern for all auth APIs** in this file — every mutation/query follows the same shape:
```typescript
const someMutationOpts = mutationOptions({
  mutationFn: (data: SomeType) =>
    api<ResponseType>('/path', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  onSuccess: (result, _, __, {client}) => {
    // optional cache update
  },
})
```

---

### `web/src/routes/_authenticated/index.tsx` (new route, request-response)

**New file:** Index page at `/` that redirects to `/time-entries`.

**Analog:** `web/src/routes/_auth/login/index.tsx` — simple route file with a component.

**Login route pattern** (5 lines):
```tsx
import {createFileRoute} from '@tanstack/react-router'
import {LoginForm} from "@/routes/_auth/login/-components/login-form.tsx";

export const Route = createFileRoute('/_auth/login/')({
  component: () => (
    <div className="min-h-screen flex items-center justify-center bg-muted/30">
      <LoginForm />
    </div>
  ),
})
```

**Target pattern** — simple redirect using TanStack Router's `Navigate` component:
```tsx
import {createFileRoute, Navigate} from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/')({
  component: () => <Navigate to="/time-entries" replace />,
})
```

**Router path convention:** The route must match the file path convention. Since this is inside `_authenticated/`, the route ID will be `/_authenticated/` (the index route of the `_authenticated` layout).

---

## Shared Patterns

### Authentication Pattern (Backend — Middleware)
**Source:** `internal/middleware/middleware.go` lines 40-61
**Apply to:** All protected routes in `cmd/server/main.go`

```go
func Auth(authService *auth.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			api.RespondWithError(w, http.StatusUnauthorized, "missing access token")
			return
		}
		claims, err := authService.ValidateToken(cookie.Value)
		if err != nil {
			api.RespondWithError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, OrganizationIDKey, claims.OrganizationID)
		ctx = context.WithValue(ctx, RoleKey, claims.Role)
		ctx = context.WithValue(ctx, EmailKey, claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
```

**Route wiring pattern** (`cmd/server/main.go`):
```go
mux.HandleFunc("POST /auth/switch-organization", middleware.Auth(authService, authHandler.SwitchOrganization))
```

### Cookie Setting Pattern (Backend)
**Source:** `internal/adapters/primary/http/auth.go` (Login/Bootstrap handlers)
**Apply to:** Register handler fix

Follow the exact 3-line pattern after successful response:
```go
secure := cookies.IsSecureRequest(r)
cookies.SetAccessTokenCookie(w, resp.Token, secure)
cookies.SetRefreshTokenCookie(w, resp.RefreshToken, secure)
```

### API Response Envelope (Backend)
**Source:** `pkg/api/response.go`
**Apply to:** All handlers

```go
func RespondWithJSON(w http.ResponseWriter, status int, payload interface{}) { ... }
func RespondWithError(w http.ResponseWriter, status int, message string) { ... }
```

### Frontend Mutation Pattern (Frontend)
**Source:** `web/src/routes/_auth/login/-components/login-form.tsx`
**Apply to:** OrgSwitcher org-switch mutation

```tsx
const {mutateAsync: loginAsync, isError, error, isPending} = useMutation(AuthApis.loginMutationOpts)
const navigate = useNavigate()

const onSubmit = (data: LoginFormData) => {
  toast.promise(
    loginAsync(data),
    {
      loading: 'Logging in...',
      success: () => {
        navigate({to: '/', replace: true})
        return 'Authentication successful! Redirecting to dashboard...'
      },
      error: (err) => err?.message ?? 'Authentication failed',
    }
  )
}
```

### Frontend API Module Pattern (Frontend)
**Source:** `web/src/api/auth.ts`
**Apply to:** Any new API modules

Each endpoint is defined as either `queryOptions` or `mutationOptions`, then exported under a namespace object:
```typescript
const profileQueryOpts = queryOptions({ queryKey: ['auth', 'me'], queryFn: ... })
const loginMutationOpts = mutationOptions({ mutationFn: ..., onSuccess: ... })

export const AuthApis = {
  profileQueryOpts,
  loginMutationOpts,
  // ...
}
```

### TanStack Router Route Pattern (Frontend)
**Source:** `web/src/routes/_auth/login/index.tsx`
**Apply to:** New route at `/_authenticated/index.tsx`

File-based routing with co-located components in `-components/`:
```
routes/
  _auth/login/
    index.tsx            # Route definition (thin)
    -components/
      login-form.tsx     # Component
```

---

## No Analog Found

All files have close analogs in the codebase. No file requires external pattern references.

| File | Role | Data Flow | Analog Found |
|---|---|---|---|
| `internal/adapters/primary/http/auth.go` | handler | request-response | `Login()` in same file (line 74) |
| `internal/core/services/password_reset/password_reset.go` | service | CRUD | `auth.Service` in auth.go |
| `internal/adapters/primary/http/password_reset.go` | handler | request-response | Already compliant |
| `web/src/components/app/org-switcher.tsx` | component | request-response | `ProfileMenu` in profile-menu.tsx |
| `web/src/components/layout/sidebar.tsx` | component | request-response | Self (existing sidebar) |
| `web/src/api/auth.ts` | utility | request-response | Self (existing auth API) |
| `web/src/routes/_authenticated/index.tsx` | route | request-response | `login/index.tsx` |

---

## Metadata

**Analog search scope:**
- `internal/adapters/primary/http/*` (backend handlers)
- `internal/core/services/*` (backend services)
- `internal/middleware/*` (backend middleware)
- `web/src/components/app/*` (frontend components)
- `web/src/components/layout/*` (frontend layout)
- `web/src/api/*` (frontend API modules)
- `web/src/routes/*` (frontend routes)
- `web/src/routes/_auth/*/-components/*` (frontend form components)
- `web/src/lib/*` (frontend utilities)
- `web/src/types/*` (frontend types)
- `pkg/api/*` (shared response formats)
- `internal/cookies/*` (cookie helpers)
- `cmd/server/*` (route wiring)

**Files scanned:** 18
**Pattern extraction date:** 2026-06-10
