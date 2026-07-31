---
phase: 08-pre-deployment-hardening-p0-audit-fixes
verified: 2026-07-31T17:05:00Z
status: human_needed
score: 6/7 must-haves verified
overrides_applied: 0
gaps: []
deferred:
  - truth: "POST /auth/bootstrap applies S3 length caps (WR-02 — open review warning)"
    addressed_in: "Phase 9 (activity ontology rewrites the same handlers)"
    evidence: "08-REVIEW.md resolution: 'Warnings WR-01..05 remain open (deferred to deferred-items.md / Phase 9)' — bootstrap handler at internal/adapters/primary/http/auth.go:199 still decodes RegisterRequest without validateStringLengths"
human_verification:
  - test: "Run the error-boundary Playwright suite on an idle machine (no parallel load): `npx playwright test error-boundary` repeated 10+ times"
    expected: "4/4 passing on every run — specifically 'Try again re-runs the loader and recovers to data' must clear the error panel within the 15s assertion"
    why_human: "In this verification environment the recovery path failed ~25% of runs (7/26: alert stayed visible the full 15s after clicking Try again), contradicting the 08-04 SUMMARY claim of 15/15 stable. Need a human to decide whether the residual stale-boundary flake (TanStack Router v1.170 invalidate/error-clear race) is acceptable for the v0.1 deployment gate or requires a follow-up fix."
  - test: "Manual browser smoke on /time-entries and /expenses List tabs: apply status/date/category filters, reload the URL, confirm filter state persists; click a row and confirm the detail surface opens"
    expected: "Filters narrow rows, URL reflects listStatuses/listFrom/listTo/listCategory, reload restores them; row click opens EntryDetail/ExpenseDetail"
    why_human: "Filter persistence and row-click navigation are browser interactions; automated specs cover them but the feel and edge cases (single-bound date ranges after CR-01) need human confirmation."
  - test: "Manual browser smoke on /customers: navigate from sidebar, search, open a customer detail, deep-link a fresh session"
    expected: "List renders with data, search narrows, detail loads at /customers/$id, fresh session redirects through /login to the app without 404/blank screen"
    why_human: "Visual render quality and navigation feel of the customers route need human confirmation beyond the E2E assertions."
  - test: "Kill the backend mid-session on an authenticated page and confirm the error panel renders; restart the backend and click 'Try again'"
    expected: "Error panel with message + Try again + Go to Today appears (never a blank screen); after backend restart, Try again recovers to real data"
    why_human: "The real-outage recovery loop is the P0-4 contract; the automated 500-interception spec proved it most runs but flaked intermittently (see item 1)."
---

# Phase 8: Pre-Deployment Hardening (P0 audit fixes) Verification Report

**Phase Goal:** Close the remaining P0 findings from the 2026-07-28 Pre-Deployment Audit after code verification (2026-07-31): P0-2 list views, P0-3 `/customers` route, P0-4 error boundaries, P0-5-lite refresh-token reuse detection — plus folded-in S3 input length caps. P0-1 (status CHECK) and P0-6 (reset-code exposure) verified already-fixed pre-audit. Phase gates first deployment of v0.1.
**Verified:** 2026-07-31T17:05:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #   | Truth   | Status     | Evidence       |
| --- | ------- | ---------- | -------------- |
| 1   | P0-2: time-entries and expenses List tabs render real, filterable data (shared table shell, URL-driven filters) | ✓ VERIFIED | `web/src/components/shared/entries-table.tsx` (generic `EntriesTable<T>`, page size 25), `time-entries-list.tsx` + `expenses-list.tsx` pull real data via `useSuspenseQuery(TimeEntriesApis.timeEntriesForMonthQueryOpts / ExpensesApis.expensesForMonthQueryOpts)`; filters driven by `validateSearch` params (`listStatuses`/`listFrom`/`listTo`/`listCategory`) and written back via `navigate`; no placeholder comments remain in source (grep clean); live E2E: time-entries 7/7 + expenses 5/5 green; status-badge maps all six states with distinct colors |
| 2   | P0-3: `/customers` index route is reachable and renders list data | ✓ VERIFIED | `web/src/routes/_authenticated/customers/index.tsx` exists with `ensureQueryData(CustomersApis.customersQueryOpts())` loader + `component: CustomersPage`; registered in `routeTree.gen.ts` (`/customers/`); sidebar link `href: "/customers"`; missing customers-context store was recreated (P3 gap fixed in `4103ca1`); live E2E: customers 8/8 green incl. sidebar→list→detail, search, deep-link redirect chain |
| 3   | P0-4: a failed loader on any authenticated route shows a recoverable error panel, never a blank screen | ⚠️ UNCERTAIN | Artifacts exist and are wired: `route-error.tsx` (full `RouteError` + slim `AuthRouteError`), `errorComponent` on `_authenticated.tsx` layout, `_auth.tsx` (slim variant), and leaf data routes (time-entries/expenses/customers index); panel renders reliably (blank-screen half verified). **However:** the recovery path is intermittently unreliable — the E2E assertion `alert not.toBeVisible` after clicking "Try again" failed ~25% of runs in this verification (7/26 across suite + isolated runs; alert stayed visible the full 15s). The 08-04 SUMMARY claim of "stable 15/15 consecutive runs" was NOT reproducible. See Human Verification item 1. |
| 4   | P0-5: refresh-token reuse detection — family_id/rotated_at, replay → ErrTokenReuse + family revocation, atomic rotate | ✓ VERIFIED | Migration `010` adds `family_id` (backfilled one-per-row + DEFAULT gen_random_uuid), `rotated_at`, family index (up/down verified on disk); `ports.RefreshToken` gains FamilyID/RotatedAt/RevokedAt; `Rotate` is a single pgx `BeginTx` with `SELECT … FOR UPDATE` (old row → rotated_at, successor inserted same family, commit; replay → `RevokeFamily` + `ports.ErrTokenReuse`); service maps to `auth.ErrTokenReuse`; handler → 401 + `cookies.ClearAuthCookies` (both cookies). Live probe against running server: refresh rotates (new token), replay of original → `{"error":"refresh token reuse detected; session revoked"}`, **successor replay also 401** (family revoked). Real-PG repo tests pass: `TestRefreshTokenRepository_Rotate`, `_ReplayRevokesFamily`, `_ConcurrentRace`; service unit tests pass (replay revokes family, mid-rotate failure no partial state) |
| 5   | S3: request string length caps at handler boundary → 400 on violation | ✓ VERIFIED (WR-02 warning) | `internal/adapters/primary/http/validate.go` with per-field caps (email 320, name 200, description 4000, address 500, VAT 50, phone 50, password 128, short 500), rune-count semantics; wired into 10 handler files (auth register/login, password_reset, customer, contract, project, time_entry, expense, unit, working_group). Live probe: POST /auth/register with 10,000-char firstname → `400 {"error":"firstname exceeds maximum length of 200"}`; normal register → 200. Boundary + 7 endpoint over-limit handler tests pass. **Warning:** `POST /auth/bootstrap` (auth.go:199) decodes the same RegisterRequest shape with NO `validateStringLengths` gate — open review warning WR-02, deferred (see Deferred Items) |
| 6   | P0-1: time-entry status CHECK includes all six states (pre-existing fix, not regressed) | ✓ VERIFIED | `migrations/004_time_entries_status_check.up.sql` widens the CHECK to draft/submitted/pending_manager/pending_finance/approved/rejected (matching down reverts to 3 states); no Phase 8 regression (no migration re-narrows it; latest migration is 010) |
| 7   | P0-6: reset code absent from response body (pre-existing fix, not regressed) | ✓ VERIFIED | `internal/adapters/primary/http/password_reset.go` Request returns only `message` + `expires_at` (code discarded); regression test `password_reset_test.go:139` asserts `"code"` field absent from response; handler Reset path unchanged |

**Score:** 6/7 truths verified

### Deferred Items

Items not yet met but explicitly deferred by the phase's own code review (08-REVIEW.md resolution: "Warnings WR-01..05 remain open (deferred to deferred-items.md / Phase 9)").

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | WR-02: S3 length caps missing on POST /auth/bootstrap | Phase 9 (activity ontology rewrites the same auth handlers) | 08-REVIEW.md WR-02; code confirmed: `auth.go:199` Bootstrap has no validateStringLengths gate — same unbounded input shape as Register |
| 2 | WR-04: Service.Refresh rotates token before user/membership lookups — successor can be orphaned on post-commit failure | Phase 9 (auth surface rewrite) | 08-REVIEW.md WR-04: `auth.go:363-401` Rotate commits before `userRepo.GetByID`/`GenerateToken` |
| 3 | WR-05: multi-tab parallel refreshes revoke the family (strict-reuse semantics) | Documented T9, explicitly out of scope | 08-REVIEW.md WR-05; 08-01/08-03 decisions keep strict-reuse model |
| 4 | WR-01: E2E seed data hardcoded to 2026-07 — **expires 2026-08-01** | Deferred per review; **imminent** | `web/e2e/helpers.ts:142-171` + `time-entries.spec.ts:159` use fixed July 2026 dates while list views query the current month; the P0-2/P0-4 E2E evidence suites will fail from tomorrow |

### Required Artifacts

| Artifact | Expected    | Status | Details |
| -------- | ----------- | ------ | ------- |
| `migrations/010_refresh_token_reuse_detection.up.sql` / `.down.sql` | family_id + rotated_at + family index, backfill, rollback | ✓ VERIFIED | Columns, DEFAULT gen_random_uuid, UPDATE backfill, index; down drops index + columns |
| `internal/core/ports/refresh_token_repo.go` | FamilyID/RotatedAt/RevokedAt, Rotate, RevokeFamily, ErrTokenReuse | ✓ VERIFIED | Full interface present with docs |
| `internal/adapters/secondary/postgres/refresh_token_repo.go` | Atomic Rotate (BeginTx + FOR UPDATE), RevokeFamily, tombstone FindByHash | ✓ VERIFIED | Single tx, replay → family revocation + ErrTokenReuse, no partial-state window |
| `internal/core/services/auth/errors.go` | ErrTokenReuse sentinel | ✓ VERIFIED | Created per plan (package had no errors.go) |
| `internal/core/services/auth/auth.go` | Refresh delegates to Rotate; issue paths keep fresh families | ✓ VERIFIED | L349-394 |
| `internal/adapters/primary/http/auth.go` | ErrTokenReuse → 401 + ClearAuthCookies | ✓ VERIFIED | L163-169; live-probed |
| `internal/adapters/primary/http/validate.go` | Length-cap helper | ✓ VERIFIED | Rune-count, field-level 400 message |
| `internal/adapters/primary/http/{customer,contract,project,time_entry,expense,password_reset,unit,working_group}.go` | validateStringLengths wired into write handlers | ✓ VERIFIED | grep: 2-3 call sites each |
| `web/src/routes/_authenticated/customers/index.tsx` | Index route + loader | ✓ VERIFIED | ensureQueryData loader, RouteError boundary |
| `web/src/components/shared/entries-table.tsx` | Generic table shell | ✓ VERIFIED | No `any`, pagination, empty state |
| `web/src/components/shared/status-badge.tsx` | Six-state badge | ✓ VERIFIED | Distinct classes, approved=emerald |
| `web/src/components/shared/entries-filters.tsx` | Status multi-select + date range | ✓ VERIFIED | CR-01 fixed (single-bound label guards), 5 regression tests |
| `web/src/components/layout/route-error.tsx` | RouteError + AuthRouteError | ✓ VERIFIED | Retry via router.invalidate + QueryErrorResetBoundary reset + Go to Today |
| `_authenticated.tsx` / `_auth.tsx` / leaf route indexes | errorComponent wiring | ✓ VERIFIED | Layout fallback + leaf boundaries + slim auth variant |
| `web/src/lib/list-filters.ts` | listStatusesSchema | ✓ VERIFIED | single/repeated/JSON-array forms |
| `web/e2e/{helpers,time-entries,expenses,customers,error-boundary}.spec.ts` + `auth.spec.ts` | Browser-proven coverage | ✓ VERIFIED (with flake) | time-entries 7/7, expenses 5/5, customers 8/8, auth 7/7 passed live; error-boundary 4/4 most runs but recovery test flakes ~25% |

### Key Link Verification

| From | To  | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `auth.go` Refresh handler | service `Refresh` | `h.authService.Refresh` | WIRED | L161-176 |
| service `Refresh` | repo `Rotate` | `s.refreshTokenRepo.Rotate` | WIRED | L363; replay mapped to ErrTokenReuse L365-367 |
| repo `Rotate` | DB transaction | pgx BeginTx + FOR UPDATE | WIRED | L88-138; single commit |
| `validateStringLengths` | write handlers | call immediately after JSON decode | WIRED | 10 handler files; live 400 probe |
| `time-entries-list.tsx` | time-entries API hook | `useSuspenseQuery(TimeEntriesApis.timeEntriesForMonthQueryOpts)` | WIRED | Real API data (Level 4: FLOWING — seeded rows rendered in E2E) |
| `expenses-list.tsx` | expenses API hook | `useSuspenseQuery(ExpensesApis.expensesForMonthQueryOpts)` | WIRED | Real API data (Level 4: FLOWING — receipt/mileage rows rendered in E2E) |
| list filters | URL | `validateSearch` + `navigate` search patch | WIRED | Reload restores filters (E2E round-trip asserted) |
| `/customers` route | customers API | `ensureQueryData(CustomersApis.customersQueryOpts())` | WIRED | Level 4: FLOWING — seeded list rendered in E2E |
| sidebar Customers link | `/customers` route | `href: "/customers"` | WIRED | sidebar.tsx L60 |
| error boundary | leaf + layout routes | `errorComponent: RouteError/AuthRouteError` | WIRED | 3 leaf data routes + layout fallback + _auth slim |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `time-entries-list.tsx` | `entries` | `TimeEntriesApis.timeEntriesForMonthQueryOpts(month)` → GET /time-entries | Yes — live API, seeded rows rendered in E2E | ✓ FLOWING |
| `expenses-list.tsx` | `expenses` | `ExpensesApis.expensesForMonthQueryOpts(month)` → GET /expenses | Yes — live API, receipt/km rows rendered in E2E | ✓ FLOWING |
| `customers/index.tsx` loader | customers list | `CustomersApis.customersQueryOpts()` → GET /customers | Yes — seeded rows rendered in E2E | ✓ FLOWING |
| `route-error.tsx` | `error.message` | router loader error | Yes — real error propagated (500-intercepted E2E) | ✓ FLOWING |
| refresh-token `Rotate` | successor row | pgx tx SELECT FOR UPDATE + INSERT | Yes — real PG rows, live probe confirmed rotation + revocation | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Go build | `go build ./...` | exit 0 | ✓ PASS |
| P0-5 service unit: replay revokes family + mid-rotate failure | `go test ./internal/core/services/auth/ -run 'TestService_Refresh_ReplayRevokesFamily|TestService_Refresh_MidRotateFailure_NoPartialState'` | ok 0.9s | ✓ PASS |
| P0-5 real-PG: rotate / replay-revokes-family / concurrent race | `go test ./internal/adapters/secondary/postgres/ -run 'TestRefreshTokenRepository_Rotate$|..._ReplayRevokesFamily|..._ConcurrentRace'` | 3 PASS (3.8s, testcontainers) | ✓ PASS |
| S3 caps boundary + endpoint over-limit | `go test ./internal/adapters/primary/http/ -run 'TestValidateStringLengths_Boundaries|TestInputLengthCaps_RejectOversizedFields'` | ok 0.9s | ✓ PASS |
| Shared components + error boundary Vitest | `npx vitest run src/components/shared/__tests__/entries-table.test.tsx src/components/layout/__tests__/route-error.test.tsx` | 15/15 pass | ✓ PASS |
| P0-2/P0-3 E2E | `npx playwright test time-entries expenses customers` | 19/19 pass (live backend) | ✓ PASS |
| P0-4 E2E | `npx playwright test error-boundary` | 4/4 most runs; **recovery test flaked 7/26 runs** (alert stayed visible 15s after Try-again) | ⚠️ FLAKY |
| S3 live probe: over-length firstname | `curl POST /auth/register` with 10k-char firstname | `400 {"error":"firstname exceeds maximum length of 200"}` | ✓ PASS |
| P0-5 live probe: rotate → replay → family revocation | `curl POST /auth/refresh` chain | refresh rotates; replay → `401 refresh token reuse detected; session revoked`; successor replay → 401 | ✓ PASS |

### Probe Execution

No probe scripts are declared for this phase (no `scripts/*/tests/probe-*.sh` exist; not a migration/tooling phase). **SKIPPED** — Phase 8's verification is test-suite-based, and the executor's probe-style claims (E2E suites) were re-run independently above.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| P0-2 | 08-02, 08-04 | List views on time-entries/expenses List tabs | ✓ SATISFIED | Shared EntriesTable + URL filters; E2E 12/12 |
| P0-3 | 08-02, 08-04 | `/customers` index route | ✓ SATISFIED | Route + loader + sidebar; E2E 8/8 |
| P0-4 | 08-02, 08-04 | Route error boundaries | ? NEEDS HUMAN | Panel renders (blank-screen fixed); recovery path flakes ~25% in this environment |
| P0-5 | 08-01, 08-03 | Refresh-token reuse detection | ✓ SATISFIED | Migration + atomic rotate + live replay/revocation probe + real-PG tests |
| S3 | 08-01, 08-03 | Handler-boundary length caps | ✓ SATISFIED (WR-02 partial) | validate.go + 10 handler files + live 400 probe; bootstrap uncovered (deferred) |
| P0-1 | — (closed pre-audit) | Status CHECK all six states | ✓ SATISFIED (not regressed) | migration 004 present, latest migration 010 doesn't re-narrow |
| P0-6 | — (closed pre-audit) | Reset code out of response | ✓ SATISFIED (not regressed) | handler returns message+expires_at only; test asserts code absence |

**Requirement-ID accounting (important note):** The PLAN frontmatter `requirements:` fields reference **P0-1..P0-6 and S3 — these are AUDIT FINDING IDs from `hourglass-vault/research/2026-07-28 — Pre-Deployment Audit — Hourglass v0.1.md` (§4 security table for S3, §6 priority matrix for P0-1..P0-6), NOT `.planning/REQUIREMENTS.md` IDs.** Verified: REQUIREMENTS.md contains no P0-* or S3 entries (its IDs are TEST/AUTH/ORG/CUST/CTRT/PROJ/TIME/EXPN/APPR/EXPT), and its traceability table has no Phase 8 row. All seven audit findings referenced across the four plans are accounted for above; there are no orphaned REQUIREMENTS.md requirements for this phase, and no REQUIREMENTS.md requirement maps to Phase 8.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| `internal/adapters/primary/http/auth.go` | 199-229 | Bootstrap handler without S3 length caps (WR-02) | ⚠️ Warning | Public endpoint accepts unbounded strings on a fresh deployment; open review warning, deferred |
| `web/e2e/helpers.ts` + `time-entries.spec.ts` | 142-171, 159 | Hardcoded 2026-07 seed dates (WR-01) | ⚠️ Warning | **Time bomb:** the P0-2/P0-4 E2E evidence suites fail from 2026-08-01 when the list month rolls over |
| `web/src/components/shared/entries-table.tsx` | 55-59 | Page index not reset when row set shrinks (IN-01) | ℹ️ Info | Minor navigation surprise; clamped display works |
| `internal/adapters/primary/http/auth.go` | 163-171 | Generic refresh 401 leaves cookies set (IN-02) | ℹ️ Info | Extra refresh attempt per page load before redirect |

Debt-marker scan (TBD/FIXME/XXX) on all phase-modified key files: **clean**. Placeholder scan on list views / customers route: **clean** (no placeholder comments remain — the P0-2 stub is gone).

### Human Verification Required

### 1. P0-4 recovery reliability (deployment-gate decision)

**Test:** Run the error-boundary Playwright suite on an idle machine, repeated 10+ times: `npx playwright test error-boundary`
**Expected:** 4/4 passing on every run — specifically "Try again re-runs the loader and recovers to data" must clear the error panel within its 15s assertion.
**Why human:** This verification observed the recovery path fail ~25% of runs (7/26: the alert stayed visible the full 15s after clicking "Try again"), reproducing the exact stale-boundary symptom class 08-04 claimed to have fixed ("stable 15/15 consecutive runs" was not reproducible here). It is a TanStack Router v1.170 invalidate/error-clearing race that may be environment-sensitive (this machine was under load, load avg 6.8-9.5). A human must decide: accept for the v0.1 gate, or require a follow-up fix.

### 2. List-view manual smoke (P0-2)

**Test:** On /time-entries and /expenses, apply status/date/category filters, reload the URL, confirm filter state persists; click a row and confirm the detail surface opens; try a single-bound date range (one click in the calendar).
**Expected:** Filters narrow rows and survive reload; row click opens EntryDetail/ExpenseDetail; single-bound range renders without the pre-CR-01 RangeError.
**Why human:** Browser interaction feel and the CR-01 edge case need human confirmation beyond automated specs.

### 3. Customers route manual smoke (P0-3)

**Test:** Sidebar → Customers; search; open a detail; deep-link a fresh session to /customers.
**Expected:** List renders with data, search narrows, detail loads, no 404/blank; fresh session flows login → app → /customers.
**Why human:** Visual render and navigation feel of the newly reachable route.

### 4. Real-outage recovery loop (P0-4)

**Test:** Stop the backend while on an authenticated page; confirm the error panel; restart the backend; click "Try again".
**Expected:** Error panel with message + Try again + Go to Today (never a blank screen); recovery to real data after restart.
**Why human:** The real outage loop is the P0-4 contract; the automated 500-interception spec proved it most runs but flaked intermittently.

### Gaps Summary

**Six of seven must-haves are verified** with code-level, test-level, and live behavioral evidence. P0-5 (reuse detection), S3 (input caps), P0-2 (list views), P0-3 (customers route), and the pre-existing P0-1/P0-6 fixes are all confirmed in the codebase and — for the runnable ones — proven live against the running backend and real PostgreSQL.

**One must-have is UNCERTAIN and needs a human decision before the v0.1 deployment gate: P0-4 error-boundary recoverability.** The blank-screen defect is genuinely fixed (the panel renders reliably), but the "Try again" recovery path failed ~25% of runs in this verification environment (7/26), showing the same stale-panel symptom 08-04's leaf-level boundary change was meant to eliminate. The claimed 15/15 stability could not be reproduced. This may be environment-load-sensitive, but it is the exact property the phase's own E2E acceptance criterion asserts, so it should not be silently accepted.

**Secondary warnings (not blockers):** WR-01 is a hard time-bomb — the E2E seed data is hardcoded to July 2026 and expires tomorrow (2026-08-01), which will break the P0-2/P0-4 evidence suites exactly when they are needed for the deployment gate; fixing it is a one-line date-relative change that should be scheduled immediately. WR-02 (bootstrap without S3 caps) and WR-04/WR-05 remain open per the phase's own review resolution, though notably they are **not listed in `deferred-items.md`** (only the three original items are) — the review resolution claims a deferral location that does not contain them; recommend adding them so the deferral is auditable.

---

_Verified: 2026-07-31T17:05:00Z_
_Verifier: the agent (gsd-verifier)_
