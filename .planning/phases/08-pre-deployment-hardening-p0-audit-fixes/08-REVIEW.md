---
phase: 08-pre-deployment-hardening-p0-audit-fixes
reviewed: 2026-07-31T00:00:00Z
depth: standard
files_reviewed: 42
files_reviewed_list:
  - cmd/server/main.go
  - internal/adapters/primary/http/auth.go
  - internal/adapters/primary/http/auth_integration_test.go
  - internal/adapters/primary/http/auth_test.go
  - internal/adapters/primary/http/validate.go
  - internal/adapters/primary/http/validate_test.go
  - internal/adapters/secondary/postgres/refresh_token_repo.go
  - internal/adapters/secondary/postgres/refresh_token_rotate_test.go
  - internal/core/ports/refresh_token_repo.go
  - internal/core/services/auth/auth.go
  - internal/core/services/auth/auth_integration_test.go
  - internal/core/services/auth/auth_test.go
  - internal/core/services/auth/errors.go
  - internal/core/services/testdata/mocks.go
  - migrations/010_refresh_token_reuse_detection.down.sql
  - migrations/010_refresh_token_reuse_detection.up.sql
  - web/src/components/layout/__tests__/route-error.test.tsx
  - web/src/components/layout/route-error.tsx
  - web/src/components/shared/entries-filters.tsx
  - web/src/components/shared/entries-table.tsx
  - web/src/components/shared/status-badge.tsx
  - web/src/lib/list-filters.ts
  - web/src/routes/_auth.tsx
  - web/src/routes/_authenticated.tsx
  - web/src/routes/_authenticated/customers/-context/customers-context.tsx
  - web/src/routes/_authenticated/customers/index.tsx
  - web/src/routes/_authenticated/expenses/-components/expenses-list.tsx
  - web/src/routes/_authenticated/expenses/-components/expenses-page.tsx
  - web/src/routes/_authenticated/expenses/index.tsx
  - web/src/routes/_authenticated/time-entries/-components/time-entries-list.tsx
  - web/src/routes/_authenticated/time-entries/-components/time-entries-page.tsx
  - web/src/routes/_authenticated/time-entries/index.tsx
  - web/e2e/auth.spec.ts
  - web/e2e/customers.spec.ts
  - web/e2e/error-boundary.spec.ts
  - web/e2e/expenses.spec.ts
  - web/e2e/helpers.ts
  - web/e2e/time-entries.spec.ts
  - web/src/routeTree.gen.ts
  - web/src/api/customers.ts
  - web/src/lib/__tests__/setup.ts
  - web/src/routes/_auth/password-reset/-components/password-reset-request-form.tsx
findings:
  critical: 1
  warning: 5
  info: 3
  total: 9
status: issues_found
---

# Phase 8: Code Review Report

**Reviewed:** 2026-07-31T00:00:00Z
**Depth:** standard
**Files Reviewed:** 42
**Status:** issues_found

## Summary

Reviewed the Phase 8 changes against the pre-phase commit `f6431ea`:
refresh-token reuse detection with family revocation (atomic rotate tx, `ErrTokenReuse`
→ 401 + cookie clear), the S3 request-string length caps (`validate.go`), the frontend
list views built on the shared `EntriesTable`/`EntriesFilters`/`StatusBadge`, the
`/customers` index route with its zustand context store, and the route error boundaries
(`RouteError`/`AuthRouteError`).

The backend reuse-detection work is solid: `Rotate` correctly serializes concurrent
rotations via `SELECT … FOR UPDATE`, the replay path revokes the whole family inside the
same transaction, the migration backfills existing rows into fresh families, and the
unit/integration/race tests actually exercise the intended semantics (including the
documented T9 multi-tab race tradeoff). The 401 + cookie-clear path for `ErrTokenReuse`
is wired end-to-end and covered by both Go integration tests and the Playwright spec.

The main defect found is on the frontend: the shared `DateRangeFilter` crashes with a
`RangeError` (reproduced against the installed date-fns) whenever a user selects only a
start date — a first-class interaction the component's own `onSelect` handler supports.
Secondary issues: S3 caps were not applied to the `/auth/bootstrap` handler (same input
surface as register), the e2e seed helpers are hardcoded to July 2026 dates and will
break next month, and the bcrypt 72-byte truncation is not addressed by the rune-count
password cap.

## Critical Issues

### CR-01: DateRangeFilter throws RangeError when only a start date is selected

**File:** `web/src/components/shared/entries-filters.tsx:129-130`
**Issue:** In range mode, clicking once in the DayPicker selects a single day — the
`onSelect` handler (lines 142-151) then calls `onChange(from, undefined)`. The trigger
label immediately renders:

```tsx
{format(new Date(`${from}T00:00:00`), "dd MMM")} -{" "}
{format(new Date(`${to}T00:00:00`), "dd MMM")}
```

With `to === undefined` this evaluates `new Date("undefinedT00:00:00")`, which is an
Invalid Date, and date-fns `format` throws `RangeError: Invalid time value` (verified
against the installed date-fns). The same crash occurs on a deep link / manual URL with
only `listTo` set (`from` undefined). Because this filter is shared by both the
time-entries and expenses list views (P0-2), a single-click date selection takes down
the whole page into the route error panel. This is a supported interaction — the
component itself implements the from-only branch — so it will be hit by any user who
clicks the calendar once.
**Fix:** Guard each side of the label independently:

```tsx
<span>
  {from && format(new Date(`${from}T00:00:00`), "dd MMM")}
  {from && to && " - "}
  {to && format(new Date(`${to}T00:00:00`), "dd MMM")}
</span>
```

## Warnings

### WR-01: E2E seed data is hardcoded to July 2026 — suites time-bomb on the next month

**File:** `web/e2e/helpers.ts:142-147` (and `web/e2e/time-entries.spec.ts:159`)
**Issue:** `seedTimeEntries`/`seedExpenses` insert rows with fixed dates (`2026-07-15`
through `2026-07-20`), while the list views query the *current* month
(`month` search param defaults to `new Date()`). `time-entries.spec.ts:159` also asserts
`date=.*2026-07-15`. The comment claims "all inside the current month", but the dates are
baked in — from 2026-08-01 the seeded rows silently fall outside the queried month and
every list-view and error-boundary spec fails (or the `desc(...)` assertions match
nothing).
**Fix:** Compute dates relative to the run time, e.g.
`const d = (day: number) => { const now = new Date(); return \`${now.getFullYear()}-${String(now.getMonth()+1).padStart(2,"0")}-${String(day).padStart(2,"0")}\`; }`
and use those in both `helpers.ts` and the `date=…` URL assertion in the specs.

### WR-02: S3 length caps missing on POST /auth/bootstrap

**File:** `internal/adapters/primary/http/auth.go:199-229`
**Issue:** The S3 hardening (audit item) added `validateStringLengths` gates to
Register, Login, and all entity handlers, but `Bootstrap` accepts the same unbounded
input shape (email, username, firstname, lastname, password, organization_name) with no
length caps. Bootstrap creates the admin user + org and is the first endpoint hit on a
fresh deployment — an oversized body bypasses the boundary hardening the phase
introduced.
**Fix:** Apply the same gate as Register:

```go
if !validateStringLengths(w,
    lengthField("email", req.Email, MaxEmailLength),
    lengthField("username", req.Username, MaxShortStringLength),
    lengthField("firstname", req.FirstName, MaxNameLength),
    lengthField("lastname", req.LastName, MaxNameLength),
    lengthField("password", req.Password, MaxPasswordLength),
    lengthField("organization_name", req.OrganizationName, MaxNameLength),
) {
    return
}
```

### WR-03: Password length cap is rune-based but bcrypt truncates at 72 bytes

**File:** `internal/adapters/primary/http/validate.go:21`
**Issue:** `MaxPasswordLength = 128` is enforced via `utf8.RuneCountInString`, so a
128-rune multi-byte password can be up to 512 bytes. bcrypt (used by
`auth.NewPasswordHasher`) silently truncates at 72 bytes, meaning two passwords sharing
their first 72 bytes authenticate identically — the cap neither prevents the truncation
nor bounds the bcrypt input, and the "128 chars accepted" contract gives users a false
sense of entropy. This is exactly the class of input the S3 cap was meant to bound.
**Fix:** Also reject passwords whose byte length exceeds 72 (or pre-hash with SHA-256
before bcrypt and document it). At minimum add a byte check:
`if len([]byte(f.value)) > 72 { … reject }` for the password field.

### WR-04: Service.Refresh rotates the token before user/token lookups — successor can be orphaned

**File:** `internal/core/services/auth/auth.go:363-401`
**Issue:** The repo transaction commits the rotation (old token consumed, successor
inserted) *before* `userRepo.GetByID` (line 374), `GetMembership`, and
`GenerateToken` (line 391) run. If any of those fail after the commit (user deleted
mid-refresh, token service error, DB blip), the client receives a 401 while the successor
row — the only valid token in the family — was never delivered. The client still holds
the consumed cookie; its next refresh replays it, is misread as token reuse, and revokes
the family. The user is force-logged-out from an error that had nothing to do with them.
**Fix:** Move the user/membership/org lookups *before* the rotation, or on the error path
after rotation explicitly clear the auth cookies (mirroring the `ErrTokenReuse` branch)
so the client doesn't replay the consumed token.

### WR-05: Multi-tab parallel refreshes kill the session via reuse detection

**File:** `internal/adapters/secondary/postgres/refresh_token_repo.go:88-138`
(interplay with `web/src/lib/api.ts:52-65`)
**Issue:** The phase documents (and the tests assert) that two simultaneous rotations of
the same token end with the loser revoking the whole family — the winner's fresh token
dies too. `api.ts` deduplicates refreshes *within one tab* via a module-level
`refreshPromise`, but two tabs each hold their own singleton; an expired access token
with two tabs open produces two near-simultaneous `/auth/refresh` calls → family
revocation → both tabs logged out. The race semantics are a deliberate, documented
security choice (T9, out of scope), but the practical consequence for legitimate
multi-tab users is a hard logout, and nothing on the frontend mitigates it.
**Fix:** Consider a cross-tab refresh lock (BroadcastChannel / localStorage lock, or a
short server-side grace period distinguishing near-simultaneous same-client refreshes).
If the strict semantics are kept, at least surface the 401 copy clearly (the current
"refresh token reuse detected" message is confusing for a legitimate two-tab user).

## Info

### IN-01: EntriesTable page index is not reset when the row set changes

**File:** `web/src/components/shared/entries-table.tsx:55-59`
**Issue:** `page` is raw state; `currentPage` clamps it when filters shrink the list, but
`page` itself is never reset. After filtering down to one page and then clearing the
filter, the user jumps back to the pre-filter page number instead of page 1. No crash,
but a small navigation surprise.
**Fix:** Reset `page` to 0 in a `useEffect` keyed on `rows.length` (or lift page state to
the caller), or track the last non-clamped page.

### IN-02: Refresh handler 401 paths are inconsistent about clearing cookies

**File:** `internal/adapters/primary/http/auth.go:163-171`
**Issue:** The `ErrTokenReuse` path clears both auth cookies, but the generic
`"invalid refresh token"` 401 (unknown/expired token) leaves them in place. The client
then keeps presenting the dead cookies on every subsequent request (each page load burns
a refresh attempt before redirecting to /login).
**Fix:** Clear `auth_token`/`refresh_token` on the generic 401 branch too — an unknown or
expired refresh token is equally unrecoverable.

### IN-03: MockRefreshTokenRepo doesn't model token expiry

**File:** `internal/core/services/testdata/mocks.go:856-867`
**Issue:** The real repo's `FindByHash`/`Rotate` filter on `expires_at > NOW()`, but the
mock never checks `ExpiresAt`, so service-level tests cannot exercise expired-token
paths (which the real DB treats as "unknown token"). Low risk today, but the mock will
silently diverge if expiry handling changes.
**Fix:** Add an expiry check to the mock's `FindByHash` and `Rotate` mirroring the real
repo.

---

_Reviewed: 2026-07-31T00:00:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
