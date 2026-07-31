# Deferred Items — Phase 08 Plan 02

Out-of-scope discoveries logged per the executor scope boundary. Not fixed here;
candidates for a future backend or frontend plan.

## 1. Expense creation is broken backend-side (unit_id FK violation)

- **Found during:** Task 3 verification (browser smoke with real data)
- **Symptom:** `POST /expenses` always returns 500 "failed to create expense"
- **Root cause:** `expenses.unit_id` is `NOT NULL REFERENCES units(id)`, but
  `CreateExpenseRequest` (HTTP handler) and the domain/service request carry no
  `unit_id` — the service inserts the zero UUID, violating the FK. Verified in
  code: `internal/adapters/primary/http/expense.go` (CreateExpenseRequest has no
  unit_id), `internal/core/services/expense/expense.go` (never sets UnitID),
  `migrations/000_full_schema.up.sql` (unit_id NOT NULL + FK).
- **Impact:** The expenses calendar "create" flow fails for every org; P0-2
  expenses list itself works (reads fine) but new expenses can't be created.
- **Evidence:** `exp1 err: {"error":"failed to create expense"}` with a valid
  project/category/amount/date; 201 only unreachable.
- **Suggested fix:** add `unit_id` to the create request chain (handler →
  service → domain), require a valid unit for the org, or drop the NOT NULL/FK
  on unit_id if unit-less expenses are legitimate.

## 2. Mutation invalidation bug in contracts/projects/units API modules

- **Found during:** Task 1 (customers) — same bug fixed in `api/customers.ts`
- **Issue:** TanStack Query v5.101 passes the query client as the **4th**
  `onSuccess` argument (`(data, variables, context, { client })`). The 3-arg
  destructure `(_, __, { client })` reads the mutation context (usually
  undefined) → `client.invalidateQueries` throws inside the callback and is
  swallowed → lists never refetch after create/update/delete.
- **Affected files (pre-existing, untouched by this plan):**
  - `web/src/api/contracts.ts` (lines 46, 55, 80, 89)
  - `web/src/api/projects.ts` (lines 44, 53, 75)
  - `web/src/api/units.ts` (lines 69, 77, 95, 104)
- **Fix applied here only to:** `web/src/api/customers.ts` (in scope — P0-3 CRUD)

## 3. Pre-existing e2e flakiness (unrelated to this plan's changes)

- **Rate limiter vs. login-per-test:** backend caps anonymous `POST /auth/login`
  at 5/min per IP; every spec (except customers.spec.ts, fixed here) logs in via
  the UI once per test → full-suite parallel runs trip 429s. Evidence: batch run
  failures all at `waitForURL('/')` after login; specs pass individually.
- **`waitForURL('/')` race:** the authenticated index route redirects `/` →
  `/time-entries` (since `081e13c`, Phase 01-02). Specs racing the transient "/"
  are flaky. auth.spec.ts fails intermittently on this.
- **projects.spec.ts selector mismatch:** create-project dialog's name field is
  labeled "Project name" with no `name="name"` attribute; the spec queries
  `input[name="name"]` → timeout. Dialog untouched by this plan (last change:
  `c8068f4` formatting pass).
- **Frontend build debt:** `web` has ~85 pre-existing TypeScript errors across
  projects/contracts/org-hierarchy/auth routes and the api test files; `bun run
  build` (tsc first) cannot pass until those are fixed. This plan's files are
  type-clean (verified by targeted typecheck).
