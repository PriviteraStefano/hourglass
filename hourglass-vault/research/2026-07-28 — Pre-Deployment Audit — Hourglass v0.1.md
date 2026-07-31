# Research: Pre-Deployment Audit — Hourglass v0.1

---
tags: ["research", "audit", "pre-deployment", "hourglass"]
date: 2026-07-28
status: draft
---

> **Purpose:** Systematic research to inform what needs fixing before the first deployment iteration. This is a living document — findings will be decomposed into SPECs and Plans in subsequent phases.

---

## Corrections — 2026-07-31 Code Verification

Verified against the codebase during Phase 8 planning. Three findings were **stale at audit time** (static analysis missed existing migrations/tests):

| Item | Audit claim | Real state (verified 2026-07-31) | Action |
|------|-------------|-----------------------------------|--------|
| **B1 / P0-1** | Time-entry CHECK constraint allows only `('draft','submitted','approved')` | `migrations/004_time_entries_status_check.up.sql` already widens it to all six states (with matching `.down.sql`). The audit quoted the `000` baseline, not the corrective migration. | ✅ Closed — no work |
| **T8 / P0-6** | Reset code returned in response body; 3-digit, brute-forceable | Handler returns only `message` + `expires_at` (code discarded, `password_reset.go:44,54-57`); `generateResetCode()` emits 8 chars from a 62-char charset (62^8, crypto/rand). Regression tests already guard the absence (`password_reset_test.go:137-139`, `auth_integration_test.go:302-304` — "per D-16"). | ✅ Closed — no work |
| **T5 / P0-5** | Refresh token rotation entirely missing | Rotation exists — `Refresh` (`internal/core/services/auth/auth.go:349–404`) revokes the old hash via `RevokeByHash` and issues a new token. The real gap is the **reuse-detection layer**: no `family_id`/`rotated_at`/`revoked_at` in token persistence, replay of a rotated token is a silent generic 401 (no family revocation), and rotate runs as 3 untransacted statements (crash window + multi-tab race). | 🔶 Rescoped — Small |

**Phase 8 impact:** scope drops from 6 items to 3.5 — P0-2 (list views), P0-3 (`/customers` route), P0-4 (error boundaries), P0-5-lite (reuse detection + family revocation + atomic rotate), + S3 input caps. Plans 08-01/08-03 rewritten 2026-07-31 accordingly.

**Process note:** B1 and T8 were already fixed when this audit ran — the analysis didn't cross-check `migrations/` for corrective files or `*_test.go` for existing guards. Future audits: verify each "open bug" against migrations and tests before cataloguing.

---

## 1. High-Level Architecture

### System Shape

```
┌─────────────────────────────────────────────────────────────┐
│  FRONTEND  — React 19 SPA (Vite dev server, port 3000)     │
│  ┌──────────┐  ┌──────────────┐  ┌───────────────────────┐  │
│  │ Routes   │  │ API Modules  │  │ Components (shadcn/ui)│  │
│  │ TanStack │  │ TanStack     │  │ Tailwind v4           │  │
│  │ Router   │  │ React Query  │  │ Base UI primitives    │  │
│  └────┬─────┘  └──────┬───────┘  └───────────────────────┘  │
│       └────────┬──────┘                                     │
│        ┌───────▼────────┐                                   │
│        │ lib/api.ts     │  ← HTTP client, cookie auth,      │
│        │ (401 refresh)  │    401 auto-refresh               │
│        └───────┬────────┘                                   │
└────────────────┼────────────────────────────────────────────┘
                 │ HTTP/JSON (HttpOnly cookies)
                 ▼
┌─────────────────────────────────────────────────────────────┐
│  BACKEND  — Go 1.26.1 (net/http stdlib, port 8080)        │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ MIDDLEWARE STACK (outer → inner)                    │   │
│  │ TryAuth → RateLimiter → Logging → APIVersion → CORS │   │
│  └──────────────────────────┬───────────────────────────┘   │
│  ┌──────────────────────────▼───────────────────────────┐   │
│  │ PRIMARY ADAPTERS — internal/adapters/primary/http/  │   │
│  │ auth.go  project.go  contract.go  customer.go ...   │   │
│  └──────────────────────────┬───────────────────────────┘   │
│  ┌──────────────────────────▼───────────────────────────┐   │
│  │ CORE SERVICES — internal/core/services/              │   │
│  │ auth/  project/  contract/  customer/  org/         │   │
│  │ time_entry/  unit/  working_group/  export/ ...      │   │
│  └──────────────────────────┬───────────────────────────┘   │
│  ┌──────────────────────────▼───────────────────────────┐   │
│  │ PORTS — internal/core/ports/                        │   │
│  │ *Repository interfaces, TokenService, PasswordHasher │   │
│  └──────────────────────────┬───────────────────────────┘   │
│  ┌──────────────────────────▼───────────────────────────┐   │
│  │ DOMAIN — internal/core/domain/                      │   │
│  │ auth/  contract/  customer/  organization/  ...      │   │
│  │ Zero external dependencies                           │   │
│  └──────────────────────────┬───────────────────────────┘   │
│  ┌──────────────────────────▼───────────────────────────┐   │
│  │ SECONDARY ADAPTERS — internal/adapters/secondary/   │   │
│  │ postgres/  *Repository implementations, pgxpool     │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Technology Stack

| Layer | Technology | Version | Notes |
|-------|-----------|---------|-------|
| Language | Go | 1.26.1 | Backend |
| Language | TypeScript | 6.0.3 | Frontend |
| DB | PostgreSQL | 15+ | pgx/v5 driver, pgxpool |
| Auth | JWT + bcrypt | — | golang-jwt/v5, HttpOnly cookies |
| Frontend | React | 19.2.5 | SPA |
| Router | TanStack Router | 1.169.1 | File-based, lazy-loading |
| Data Fetching | TanStack Query | 5.100.8 | queryOptions/mutationOptions |
| State | Zustand | 5.0.13 | Complex client state |
| UI | shadcn/ui (Base UI) | 4.6.0 | Component primitives |
| Styling | Tailwind CSS | 4.2.4 | Utility-first |
| Forms | React Hook Form + Zod | 7.75.0 + 4.4.2 | Form state + validation |
| i18n | Paraglide | — | ADR-FE-012 (Accepted) |
| Testing (unit) | Vitest + MSW | 4.1.6 | jsdom + Testing Library |
| Testing (e2e) | Playwright | 1.59.1 | Browser automation |
| Build | Vite | 8.0.10 | Dev server + bundler |
| Package Manager | Bun | — | Frontend dependencies |
| Container | Docker (multi-stage) | — | golang:1.26.1-alpine → alpine |

### Key Architectural Decisions (from .planning/codebase/)

* **Hexagonal architecture** — services depend only on ports, not HTTP or PostgreSQL
* **Hand-written SQL** with pgx — no ORM, full query control
* **UUID PKs** with `gen_random_uuid()` — no sequential ID exposure
* **JWT in HttpOnly cookies** — `auth_token` (15min) + `refresh_token` (7d)
* **Two-token refresh flow** — frontend auto-calls `/auth/refresh` on 401, retries original request
* **Dual model layer** — `internal/models/` (legacy) + `internal/core/domain/` (canonical). **This is a known tech debt item.**

---

## 2. Feature Inventory

### Completed Phases (from ROADMAP.md)

| Phase | Feature | Status | Backend | Frontend | E2E |
|-------|---------|--------|---------|----------|-----|
| 0 | Testing Foundation | ✅ Complete | ✅ testcontainers | ✅ Vitest+MSW | ✅ Playwright |
| 1 | Authorization | ✅ Complete | ✅ JWT, refresh rotation | ✅ Login, register, guards | ✅ auth.spec |
| 2 | Org Hierarchy | ✅ Complete | ✅ Units CRUD, members | ✅ ReactFlow tree, CRUD | ✅ org-hierarchy.spec |
| 3 | Customers | ✅ Complete | ✅ Customer CRUD | ✅ List, form, detail | ✅ customers.spec |
| 4 | Contracts | ✅ Complete | ✅ Contract CRUD + customer FK | ✅ List, detail, create | ✅ contracts.spec |
| 5 | Projects | ✅ Complete | ✅ Project CRUD + subprojects | ✅ List, detail, edit | ✅ projects.spec |
| 6 | Time Entries + Expenses | ✅ Complete | ✅ Full CRUD + approval | ✅ Calendar, detail, forms | ✅ time-entries.spec |
| 7 | Exports | ✅ Complete | ✅ CSV/XLSX, count, filters | ✅ ExportForm, download | ✅ (implicit) |

### Route Map (Frontend)

```
/                          → redirect to /time-entries
├── /login                 → _auth/login/index.tsx
├── /register              → _auth/register/index.tsx
├── /password-reset        → _auth/password-reset/index.tsx
│   └── /verify            → _auth/password-reset/verify/index.tsx
├── /invite                → _auth/invite/index.tsx
├── /bootstrap             → _auth/bootstrap/index.tsx
└── (authenticated layout) → _authenticated.tsx (AppShell + auth guard)
    ├── /                  → _authenticated/index.tsx → Navigate to /time-entries
    ├── /time-entries      → _authenticated/time-entries/index.tsx
    │   ├── List (tab)     → placeholder ⚠️
    │   ├── Calendar (tab) → MiniCalendar + EntryDetail
    │   └── Export (tab)   → ExportForm (timesheets)
    ├── /expenses          → _authenticated/expenses/index.tsx
    │   ├── List (tab)     → placeholder ⚠️
    │   ├── Calendar (tab) → ExpenseCalendar + ExpenseDetail
    │   └── Export (tab)   → ExportForm (expenses)
    ├── /projects          → _authenticated/projects/index.tsx
    │   └── /$id           → _authenticated/projects/$id.tsx
    ├── /contracts         → _authenticated/contracts/index.tsx
    │   └── /$id           → _authenticated/contracts/$id/index.tsx
    ├── /customers         → _authenticated/customers/$id.tsx
    ├── /exports           → _authenticated/exports/index.tsx
    └── /org-hierarchy     → _authenticated/org-hierarchy/index.tsx
```

### API Route Map (Backend)

All routes under middleware chain: `TryAuth → RateLimiter → Logging → APIVersion → CORS`

| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | /health | No | healthHandler.ServeHTTP |
| POST | /auth/register | Rate-limited (5/min) | authHandler.Register |
| POST | /auth/login | Rate-limited (5/min) | authHandler.Login |
| POST | /auth/logout | No | authHandler.Logout |
| POST | /auth/refresh | No (by design) | authHandler.Refresh |
| GET | /auth/me | Auth | authHandler.GetProfile |
| POST | /auth/bootstrap | No | authHandler.Bootstrap |
| GET | /auth/bootstrap-check | No | authHandler.BootstrapCheck |
| POST | /auth/switch-organization | Auth | authHandler.SwitchOrganization |
| GET | /auth/memberships | Auth | authHandler.GetMemberships |
| POST | /auth/password-reset/request | Rate-limited (3/min) | passwordResetHandler.Request |
| POST | /auth/password-reset/verify | Rate-limited (3/min) | passwordResetHandler.Verify |
| POST | /invitations | No | invitationHandler.Create |
| GET | /invitations/validate/code/{code} | No | invitationHandler.ValidateCode |
| GET | /invitations/validate/token/{token} | No | invitationHandler.ValidateToken |
| POST | /invitations/accept | No | invitationHandler.Accept |
| GET/POST/PUT/DELETE | /units/* | Auth | unitHandler.* (12 routes) |
| GET/POST/PUT/DELETE | /working-groups/* | Auth | wgHandler.* (9 routes) |
| GET/POST/PUT/DELETE | /customers/* | Auth | customerHandler.* (5 routes) |
| POST/GET/PUT/DELETE | /organizations/* | Auth | orgHandler.* (10 routes) |
| GET/POST/PUT/DELETE | /projects/* | Auth | projectHandler.* (10 routes) |
| GET/POST/PUT/DELETE | /contracts/* | Auth | contractHandler.* (8 routes) |
| GET | /exports/* | Auth | exportHandler.* (6 routes) |
| GET/POST/PUT/DELETE | /time-entries/* | Auth | timeEntryHandler.* (8 routes) |
| GET/POST/PUT/DELETE | /expenses/* | Auth | expenseHandler.* (9 routes) |

**Total backend routes:** ~80 endpoints across 12 domains.

---

## 3. ADR Compliance Audit

### Frontend ADRs (22 records in `adr` vault)

| ADR | Layer | Status | Compliance | Notes |
|-----|-------|--------|------------|-------|
| ADR-FE-001 | Project Structure | Accepted | ⚠️ Partial | `components/app/` is empty — `ExportForm` lives in `components/exports/` (non-standard). `components/layout/` exists but is not in the ADR's taxonomy. |
| ADR-FE-002 | Schema-First with Zod | Accepted | ✅ Good | Zod schemas used in route `validateSearch` (e.g., time-entries route). Types in `web/src/types/` are hand-written interfaces, not Zod-derived. |
| ADR-FE-003 | API Layer as Grouped Objects | Accepted | ✅ Good | All API modules use `*Apis` export pattern with `queryOptions`/`mutationOptions`. Query keys follow `['domain', ...qualifiers]` convention. |
| ADR-FE-004 | Route-Level Data Loading | Accepted | ⚠️ Partial | Route loaders use `client.ensureQueryData()` correctly (time-entries route). However, no `<Suspense>` boundaries or skeleton fallbacks found in components. `pendingMs` is set (50ms) but `pendingComponent` is defined only on the layout route, not per-route. |
| ADR-FE-005 | Deferred Data with useQuery | Accepted | ⚠️ Not assessed | `useSuspenseQuery` used in org-hierarchy; `useQueries` used for member queries. No `useQuery` with deferred data found yet. |
| ADR-FE-006 | Compound Components | Accepted | ❌ Not followed | Org-hierarchy page uses a plain object `OrgHierarchy = { Root, Flow, Dialogs, Toolbar }` — no context-backed namespace pattern. No `Object.assign` pattern found. The ADR's canonical pattern is not used anywhere in the codebase. |
| ADR-FE-007 | Zustand for Complex State | Accepted | ✅ Good | `org-hierarchy-context.tsx` uses Zustand store with context provider. Follows the "exposed via ref" pattern. |
| ADR-FE-008 | React Hook Form + Zod | Accepted | ⚠️ Not assessed | Login form exists but not yet examined. |
| ADR-FE-009 | Effect.ts as Strategic Direction | Proposed | ⏸️ Deferred | Not applicable for current sprint. |
| ADR-FE-010 | ShadCN UI as Primitives | Accepted | ✅ Good | `components/ui/` contains shadcn-generated primitives. `components.json` present. |
| ADR-FE-011 | Testing Strategy | Accepted | ⚠️ Partial | Vitest+MSW for API tests ✅. Playwright E2E ✅. **Zero component-level tests** for 80+ UI components — significant gap. |
| ADR-FE-012 | i18n with Paraglide | Accepted | ❓ Unknown | Paraglide mentioned in ADR but no `paraglide/` directory or `.po`/`.ftl` files found in project. May not be implemented yet. |
| ADR-FE-013 | Authentication | Accepted | ✅ Good | HttpOnly cookie JWT with refresh rotation implemented. Better-Auth under evaluation. |
| ADR-FE-014 | Error Handling & Boundaries | Accepted | ❌ Missing | **No `errorComponent` defined on any route.** No custom error boundary component. The `_authenticated.tsx` layout has `pendingComponent` but no `errorComponent`. Mutation errors use `toast.error` ✅. But route-level errors have no consistent UI. |
| ADR-FE-015 | Mutation Flow & Cache Invalidation | Accepted | ⚠️ Partial | Targeted invalidation works (`invalidateQueries({queryKey: ['time-entries']})`). Toasts on `onSuccess`/`onError` ✅. **But no optimistic updates** — all mutations use invalidation only. Navigation-after-create pattern not verified. |
| ADR-FE-016 | Loading States & Skeletons | Accepted | ❌ Missing | **No skeleton components found.** No `<Skeleton />` usage in any route. `pendingMs: 50` is set on time-entries route but no skeleton fallback. The `pendingComponent` on `_authenticated` shows a spinner, not a skeleton. |
| ADR-FE-017 | Routing Patterns | Accepted | ⚠️ Partial | File-based routing ✅. `validateSearch` with Zod ✅. `loaderDeps` ✅. But some routes (customers) use `$id.tsx` directly under `_authenticated/` instead of directory-per-route. |
| ADR-FE-018 | Data Fetching Edge Cases | Accepted | ⚠️ Not assessed | Pagination, stale/gc time, refetching — not yet examined. |
| ADR-FE-019 | Build & Codegen Pipeline | Accepted | ✅ Good | Vite config with TanStack Router plugin, Tailwind plugin, path alias `@/`. Route tree auto-generated (`routeTree.gen.ts`). |
| ADR-FE-020 | Performance & Code-Splitting | Accepted | ⚠️ Partial | Route-level lazy loading via TanStack Router ✅. But no `React.lazy` for route-local components (ADR-FE-001 pattern not fully followed — components are imported eagerly). No manual code-splitting beyond router defaults. |
| ADR-FE-021 | Accessibility Strategy | Proposed | ⏸️ Deferred | Not yet enforced. |
| ADR-FE-022 | Dependency & Upgrade Policy | Accepted | ✅ Good | Dependencies pinned in `package.json`/`go.mod`. |

### ADR Compliance Summary

**Critical gaps (block deployment quality):**
1. **ADR-FE-014** — No error boundaries (`errorComponent`) on any route
2. **ADR-FE-016** — No skeleton loading states anywhere
3. **ADR-FE-011** — Zero component-level tests

**Moderate gaps (affect UX consistency):**
4. **ADR-FE-001** — Component organization inconsistent (ExportForm in wrong place)
5. **ADR-FE-004** — No Suspense boundaries for data loading
6. **ADR-FE-006** — Compound component pattern not used
7. **ADR-FE-015** — No optimistic updates for simple mutations

---

## 4. Bug & Issue Catalogue

### Known Bugs (from CONCERNS.md)

| # | Bug | Severity | Files | Status |
|---|-----|----------|-------|--------|
| B1 | **Time Entry DB Status Constraint Mismatch** — DB CHECK constraint allows `('draft', 'submitted', 'approved')` but code uses `pending_manager`, `pending_finance`, `rejected`. Approval workflow with these states cannot be persisted. | 🔴 Critical | `migrations/000_full_schema.up.sql`, `internal/models/models.go`, `internal/core/services/time_entry/time_entry.go` | ✅ **Stale at audit time** — already fixed by `004_time_entries_status_check.up.sql` (see Corrections) |
| B2 | **Bogus Error on Register with Bad OrgID** — `uuid.Parse(req.OrgID)` silently ignores parse errors. Invalid invite code falls through without telling the user. | 🟡 Medium | `internal/core/services/auth/auth.go` | Open |
| B3 | **Unhandled JSON Decode Error in Time Entry Reject** — Malformed JSON to `/time-entries/{id}/reject` proceeds without a reason. | 🟡 Medium | `internal/adapters/primary/http/time_entry.go` | Open |
| B4 | **MockOrgRepo.GetMembership Always Returns nil** — Tests using this mock cannot exercise membership-dependent paths. | 🟡 Medium | `internal/core/services/testdata/mocks.go` | Open |

### Tech Debt (from CONCERNS.md)

| # | Issue | Impact | Effort | Priority |
|---|-------|--------|--------|----------|
| T1 | **Dual Model Layer** — Types in `internal/models/models.go`, `internal/models/surreal_models.go`, AND `internal/core/domain/*/*.go`. Duplication and drift risk. | Silent data corruption when fields diverge | Medium | High |
| T2 | **SurrealDB Cleanup Remnant** — `internal/models/surreal_models.go` (251 lines) is dead code. No production code references it. | Maintenance surface, misleading to developers | Low | High |
| T3 | **Duplicate Project Type Column** — `projects` table has both `project_type` and `type` columns with same CHECK constraint. | Ambiguity in which column to use | Low | Medium |
| T4 | **In-Memory Rate Limiting** — Resets on server restart, no sharing across instances. Map grows unbounded. | Ineffective in multi-instance deployments | Medium | Low (single-instance for now) |
| T5 | **Refresh Token Lacks Reuse Detection** — Rotation exists (revoke + reissue), but no `family_id`/`rotated_at` state: replay of a rotated token is a silent 401 with no family revocation, and rotate is 3 untransacted statements (crash window + tab race). | Stolen-token window stays open after replay | Small | High |
| T6 | **Cookie Name Mismatch** — `cookies.go` defines `access_token` but handlers hard-code `auth_token`. Helper functions are dead code. | Confusion, dead code | Low | Medium |
| T7 | **Fire-and-Forget Audit Log** — `go s.auditRepo.Create(ctx, auditLog)` uses request context that may be cancelled. Errors silently discarded. | Audit entries silently dropped | Low | Medium |
| T8 | **Password Reset Code in Response Body** — Reset code returned in API response. 3-digit code is trivially brute-forceable. | Security vulnerability | ✅ **Stale at audit time** — code never in response (8-char/62-charset); regression-tested per D-16 (see Corrections) | — | — |
| T9 | **`/auth/refresh` Lacks Auth Middleware** — No CSRF protection or client fingerprint on refresh endpoint. | Single-point-of-failure for session security | Low | Medium |

### Security Considerations

| # | Risk | Current Mitigation | Recommendation |
|---|------|-------------------|----------------|
| S1 | Development-only default JWT secret | Rejected in production/staging | Log warning regardless of environment |
| S2 | CORS allows wildcard `*` origin | Default config only allows localhost | Remove wildcard match entirely |
| S3 | No input length validation on string fields | Domain value objects validate format, not max length | Add explicit max-length checks in handlers |
| S4 | No rate limiting differentiation on auth endpoints | Global rate limiter (20/100 per min) | Separate stricter limits for login/password reset |

### Frontend UX/UI Gaps

| # | Issue | Severity | Route/Component |
|---|-------|----------|-----------------|
| U1 | **List views are placeholders** — Both `/time-entries` and `/expenses` have `TabsContent value="list"` with only a comment `{/* Placeholder for future list view */}`. Users see an empty tab. | 🔴 Critical | `time-entries-page.tsx`, `expenses-page.tsx` |
| U2 | **No error boundaries** — No `errorComponent` on any route. A failed loader or query error crashes to a blank screen. | 🔴 Critical | All routes |
| U3 | **No skeleton loading states** — No `<Skeleton />` components used. Only a spinner on the layout route's `pendingComponent`. | 🟡 High | All routes |
| U4 | **No optimistic updates** — All mutations wait for server round-trip before UI feedback. Simple toggles feel sluggish. | 🟡 Medium | All mutation flows |
| U5 | **Missing route: `/customers` index** — Only `/customers/$id` exists. No `/customers` list page route. The customers page component exists but no route maps to it. | 🔴 Critical | `web/src/routes/_authenticated/customers/` |
| U6 | **No dashboard/home page** — `/` redirects to `/time-entries`. No summary view, no recent activity, no quick actions. | 🟡 Medium | `_authenticated/index.tsx` |
| U7 | **ExportForm in wrong directory** — `components/exports/export-form` doesn't match ADR-FE-001's taxonomy (should be `components/app/` or route-local). | 🟢 Low | `components/exports/` |
| U8 | **No `components/app/` directory** — ADR-FE-001 defines `components/app/` for shared components but it doesn't exist. Shared layout components are in `components/layout/` which isn't in the ADR taxonomy. | 🟢 Low | `components/` |

---

## 5. Missing Features (for v0.1 MVP)

From PROJECT.md's "Active" requirements:

| Feature | Backend | Frontend | Gap |
|---------|---------|----------|-----|
| Testcontainers for PG integration tests | ✅ Done | — | — |
| Fix known auth bugs | ✅ Done (Phase 0) | — | — |
| Org hierarchy ReactFlow | ✅ Done | ✅ Done | — |
| Customer CRUD | ✅ Done | ⚠️ Partial | **No `/customers` index route** |
| Contract CRUD | ✅ Done | ✅ Done | — |
| Project CRUD | ✅ Done | ✅ Done | — |
| Time entry + expense | ✅ Done | ⚠️ Partial | **List views are placeholders** |
| CSV/Excel export | ✅ Done | ✅ Done | — |

---

## 6. Pre-Deployment Priority Matrix

### Must Fix Before Deployment (P0)

| # | Item | Type | Effort | Impact |
|---|------|------|--------|--------|
| P0-1 | **Time Entry DB Status Constraint** — migration to add `pending_manager`, `pending_finance`, `rejected` to CHECK constraint | Bug | Small | ✅ **Fixed (pre-audit)** — `004_time_entries_status_check.up.sql` (2026-07-31 verification) |
| P0-2 | **List view placeholders** — implement actual list views for time-entries and expenses | UX | Medium | ✅ **Fixed (08-04)** — shared `EntriesTable` list views (`web/src/components/shared/entries-table.tsx` + `time-entries-list.tsx`/`expenses-list.tsx`) with URL-shareable filters; Vitest route tests + E2E (`web/e2e/time-entries.spec.ts`, `web/e2e/expenses.spec.ts`) |
| P0-3 | **`/customers` index route missing** — add route definition for customers list | UX | Small | ✅ **Fixed (08-04)** — `/customers` index route (`web/src/routes/_authenticated/customers/index.tsx`); E2E sidebar→list→detail + search + deep-link (`web/e2e/customers.spec.ts`) |
| P0-4 | **Error boundaries on routes** — add `errorComponent` to layout and per-route overrides | UX | Small | ✅ **Fixed (08-04)** — shared `RouteError` panel, leaf-level `errorComponent` on data routes (`web/src/components/layout/route-error.tsx`); component + E2E recovery tests incl. retry-after-outage |
| P0-5 | **Refresh token reuse detection** — add `family_id` + `rotated_at` state; replay of a rotated token → `ErrTokenReuse` + family revocation; make rotate atomic (rotation itself already exists) | Security | Small | ✅ **Fixed (08-01/08-03)** — `010_refresh_token_reuse_detection` tombstone model + atomic rotate; replay → `ErrTokenReuse` + family revocation; regression suites (`auth_test.go`, `refresh_token_rotate_test.go`, `web/e2e/auth.spec.ts`) |
| P0-6 | **Password reset code exposure** — remove code from response body, deliver via email only | Security | Small | ✅ **Fixed (pre-audit)** — never in response; 8-char/62-charset; regression-tested per D-16 |

### Should Fix Before Deployment (P1)

| # | Item | Type | Effort | Impact |
|---|------|------|--------|--------|
| P1-1 | **Skeleton loading states** — add `<Skeleton>` components to all data-loading routes | UX | Medium | Perceived performance |
| P1-2 | **Register OrgID validation** — return error for malformed invite code | Bug | Small | Silent failure confuses users |
| P1-3 | **JSON decode error handling** — return 400 for malformed JSON in reject endpoint | Bug | Small | Data integrity |
| P1-4 | **Cookie name unification** — fix `cookies.go` constants or remove dead helpers | Tech debt | Small | Code clarity |
| P1-5 | **Audit log context fix** — use `context.Background()` for goroutine, add error logging | Tech debt | Small | Data reliability |

### Nice to Have (P2)

| # | Item | Type | Effort | Impact |
|---|------|------|--------|--------|
| P2-1 | **Optimistic updates** for simple mutations (status toggles, deletes) | UX | Medium | UI responsiveness |
| P2-2 | **Component-level tests** for critical UI paths (org hierarchy, forms) | Testing | Large | Regression prevention |
| P2-3 | **Dual model layer cleanup** — consolidate into `internal/core/domain/` | Tech debt | Large | Maintainability |
| P2-4 | **SurrealDB remnant removal** — delete `surreal_models.go` | Tech debt | Small | Code hygiene |
| P2-5 | **Dashboard/home page** — summary view with recent activity | UX | Large | User experience |
| P2-6 | **i18n implementation** — Paraglide setup and message extraction | Feature | Large | Internationalisation |

---

## 7. Backend ADR Gaps

The `adr` vault has **zero backend ADRs**. The following decisions are undocumented:

| Topic | Current State | Needs ADR |
|-------|--------------|-----------|
| Error handling strategy | Sentinel errors + `wrapPGError` + HTTP status mapping | ✅ Yes |
| Repository pattern | `*Repository` struct with `pgxpool.Pool` | ✅ Yes |
| Service pattern | `Service` struct with port dependencies | ✅ Yes |
| Handler pattern | `*Handler` struct, thin adapter | ✅ Yes |
| Middleware composition | `func(http.Handler) http.Handler` chain | ✅ Yes |
| Database migration strategy | Sequential numbered `.up.sql`/`.down.sql` files | ✅ Yes |
| Authentication mechanism | JWT + bcrypt + HttpOnly cookies | ✅ Yes |
| Rate limiting approach | In-memory sliding window | ✅ Yes |
| API versioning | Accept header based | ✅ Yes |
| Logging approach | stdlib `log.Printf` | ✅ Yes |
| Testing strategy | testcontainers + testify + MSW | ✅ Yes |
| CORS policy | Configurable origins, credentials | ✅ Yes |

**Recommendation:** Create `adr/backend/` folder with ADR-BE-001 through ADR-BE-012 covering these topics before deployment. This ensures the backend decisions are documented and consistently applied as the codebase evolves.

---

## 8. Research Next Steps

This audit feeds into the following implementation phases:

1. **SPEC-001:** P0 bug fixes (DB constraint, list views, customers route, error boundaries, security fixes)
2. **SPEC-002:** P1 UX improvements (skeletons, validation, tech debt cleanup)
3. **SPEC-003:** P2 polish (optimistic updates, tests, dashboard)
4. **SPEC-004:** Backend ADR creation

Each SPEC will reference the specific findings in this document and define acceptance criteria, implementation approach, and verification steps.

---

*Research completed: 2026-07-28*
*Method: Static analysis of codebase, ADR vault, and planning documents*
*Scope: All layers — backend (Go), frontend (React), database (PostgreSQL), infrastructure (Docker)*
