# Coding Conventions

**Analysis Date:** 2026-05-12

## Languages

**Frontend (TypeScript/TSX):**
- TypeScript 5 (~6.0.3) with strict mode
- JSX via `react-jsx` transform (React 19)

**Backend (Go):**
- Go 1.26.1

## Naming Patterns

**Files:**

- **Go handlers/services/domain:** lowercase with underscores (e.g., `auth.go`, `user_repository.go`)
- **Go tests:** `_test.go` suffix co-located (e.g., `auth_test.go`)
- **Go ports (interfaces):** `_repository.go`, `_service.go`, `_service.go` suffix
- **React components:** PascalCase (e.g., `LoginForm.tsx`, `AppShell.tsx`)
- **UI primitives:** lowercase kebab for shadcn-like components (e.g., `button.tsx`, `badge.tsx`)
- **Route components:** lowercase kebab within `-components/` subfolder convention
- **API modules:** lowercase (e.g., `auth.ts`, `time-entries.ts`)
- **Type files:** lowercase (e.g., `models.ts`, `api.ts`)
- **Lib/util files:** lowercase (e.g., `utils.ts`, `api.ts`, `query-client.ts`)

**Directories:**

- Go packages: lowercase, single words or underscore-separated
- Frontend: kebab-case for route subfolders (e.g., `org-hierarchy/`, `time-entries/`)

**Variables & Functions:**

- Go: camelCase for exported fields in structs; unexported fields may use underscores
- TypeScript: camelCase for variables and functions; PascalCase for components and types
- React hooks: camelCase with `use` prefix (e.g., `useMutation`, `useForm`)

**Types:**

- TypeScript: PascalCase interfaces and types (e.g., `User`, `EntryStatus`, `LoginFormData`)
- Go: PascalCase structs and fields; exported structs have JSON field tags

## Code Style

### TypeScript/React

**Formatting:** No explicit formatter config (no Prettier); rely on ESLint + editor defaults.

**Linting:** ESLint flat config (`web/eslint.config.js`) with:
- `@eslint/js` (recommended)
- `typescript-eslint` (recommended)
- `eslint-plugin-react-hooks` (flat config recommended)
- `eslint-plugin-react-refresh`

Key tsconfig settings (`web/tsconfig.json`):
- `strict: true`
- `noUnusedLocals: false` — unused locals allowed
- `noUnusedParameters: false` — unused parameters allowed
- `jsx: react-jsx`
- `moduleResolution: bundler`
- `erasableSyntaxOnly: true`

**Import organization:**
1. React/router imports
2. TanStack Query / React Query imports
3. Third-party UI libraries (e.g., `lucide-react`, `recharts`)
4. Internal `@/` alias imports (components, api, lib, types, hooks)
5. No explicit sorting within groups; alphabetical within groups recommended

**Path aliases:**
- `@/*` maps to `web/src/*` (configured in `web/vite.config.ts` and `web/tsconfig.json`)

**Component patterns:**
- Use named exports (e.g., `export function LoginForm`) not default
- Props destructured inline with explicit types
- `cn()` utility for className merging (clsx + tailwind-merge)
- CVA (`class-variance-authority`) for component variants (e.g., `button.tsx`, `badge.tsx`)

### Go

**Formatting:** `go fmt` (implicit via `gofmt`)

**Error handling:**
- Sentinel errors defined at package level: `var ErrEmailExists = errors.New("email already registered")`
- Domain-level errors in `internal/core/domain/<name>/errors.go`
- Service-level errors in `internal/core/services/<name>/<name>.go`
- `errors.Is()` / `errors.As()` for error matching
- `fmt.Errorf("op: %w", err)` for wrapping

**Handler pattern:** Thin handlers in `internal/adapters/primary/http/` delegate to services:
```go
type AuthHandler struct {
    authService       *auth.Service
    invitationService *invitation.Service
}

func NewAuthHandler(authService *auth.Service, invitationService *invitation.Service) *AuthHandler { ... }
```

**HTTP routing:** Go 1.22+ pattern matching:
```go
mux.HandleFunc("POST /auth/register", handler.Register)
mux.HandleFunc("GET /auth/me", handler.GetProfile)
```

**Response format:** All JSON responses via `pkg/api/response.go`:
```go
api.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{"data": resp})
api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
```

**Context values:** Context keys defined as `type contextKey string` const, injected via `context.WithValue()` and read via `middleware.GetUserID(ctx)` helper functions.

## Import Organization

### Frontend
```typescript
// 1. React/router
import { useForm } from 'react-hook-form'
import { Link, useNavigate } from '@tanstack/react-router'

// 2. TanStack
import { useMutation } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'

// 3. UI/icons
import { Card, CardContent } from '@/components/ui/card'
import { LoaderIcon } from 'lucide-react'

// 4. Internal
import { AuthApis } from '@/api/auth.ts'
import { cn } from '@/lib/utils'
import type { UserWithMembership } from '@/types'
```

### Backend (Go)
```go
import (
    "encoding/json"
    "net/http"
    "time"

    "github.com/google/uuid"
    authdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
    "github.com/stefanoprivitera/hourglass/internal/core/ports"
)
```

## Error Handling

**Frontend:**
- React Query mutations use `onError` callbacks; `isError` / `error` from `useMutation`
- Toast notifications for user-facing errors via `sonner` (`toast.error()`, `toast.promise()`)
- API layer (`web/src/lib/api.ts`) throws `Error` on non-OK responses; 401 triggers refresh flow
- Zod validation schemas on forms with `react-hook-form`

**Backend (Go):**
- Handlers return HTTP errors via `api.RespondWithError(w, status, "message")`
- Services return typed sentinel errors matched in handlers via `switch err`
- Repository errors wrapped with `wrapErr(err, "operation")` helper that converts SurrealDB "not found" to `ports.ErrUserNotFound`
- No panic recovery in handlers (let errors propagate)

## Logging

**Frontend:** No explicit logging framework. Use `console.error` sparingly for unexpected errors.

**Backend (Go):** Standard library `log` package in middleware:
```go
log.Printf("request: %s %s", r.Method, r.URL.Path)
```
No structured logging library currently in use.

## Comments

**When to comment:**
- Explain "why" not "what" for non-obvious logic
- Complex regex patterns (e.g., username validation)
- Business rule rationale in service layer

**JSDoc/TSDoc:** Minimal usage. Type annotations on function signatures provide documentation.

**Go comments:** Package-level comments on exported types and functions:
```go
// Service handles time entry business logic.
type Service struct { ... }
```

## Function Design

**Frontend:**
- Small, focused functions preferred
- React Query `queryOptions` / `mutationOptions` for data fetching (defined in `web/src/api/*.ts`)
- Mutations include `onSuccess` callbacks to update query cache: `client.setQueryData(['auth', 'me'], data.user)`
- Route `beforeLoad` for auth checks; `pendingComponent` for loading states

**Backend (Go):**
- Services take interfaces (ports) for dependency injection
- Repository methods accept `context.Context` as first parameter
- Validation at handler level before calling service
- Validation at service level before calling repository

## Module Design

### Frontend

**Barrel exports:** API modules export a named `AuthApis` / `ContractsApis` etc. object:
```typescript
export const AuthApis = {
  profileQueryOpts,
  loginMutationOpts,
  ...
}
```

**Shared query client:** Single `QueryClient` instance in `web/src/lib/query-client.ts` with defaults:
```typescript
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      staleTime: 30000,
      refetchOnWindowFocus: false,
    },
  },
})
```

### Backend (Go)

**Hexagonal architecture:**
- `internal/core/domain/` — domain models and value objects
- `internal/core/ports/` — interface definitions (repository ports)
- `internal/core/services/` — application logic
- `internal/adapters/primary/http/` — HTTP handlers (thin)
- `internal/adapters/secondary/surrealdb/` — SurrealDB implementations
- `internal/models/` — shared model structs and constants

**Exported packages:** Only the service packages are exported; ports are internal.

## API Client Pattern

The frontend `api<T>()` helper (`web/src/lib/api.ts`) handles:
- Cookie-based auth (`credentials: 'include'`)
- JSON serialization
- Auto-refresh on 401 with promise deduplication
- Response envelope unwrapping (`{ data: T }` → `T`)

## React Component Patterns

**TanStack Router:**
- File-based routing with `createFileRoute()`
- `__root.tsx` for root layout
- `_authenticated.tsx` for protected route guard with `beforeLoad`
- `(auth)/` folder for public auth routes

**TanStack React Query:**
- Query options pattern for type-safe queries
- Mutation options with `onSuccess` for optimistic updates
- `queryClient.invalidateQueries()` on mutations

**Forms:**
- `react-hook-form` with `zod` resolver
- Zod schemas co-located in the same file as the form component
- Validation errors displayed via `form.formState.errors`

---

*Convention analysis: 2026-05-12*
