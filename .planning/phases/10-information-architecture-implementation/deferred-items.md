# Deferred Items — 10-01 Foundation (Role type + activities API + route rename)

Out-of-scope discoveries logged during plan execution (scope boundary: do not
auto-fix issues not caused by the current task's changes).

## 1. Pre-existing typecheck rot (6 errors remaining, unrelated files)

- **Discovered:** 2026-07-31, Task 1 typecheck verification
- **Symptom:** `cd web && bun run typecheck` reports 6 errors in files
  unrelated to the projects→activities rename. Baseline (git stash of this
  plan's changes) shows **87 typecheck errors**; this plan eliminates the
  ~77 rename-related + TanStack v5.101 `queryFn`/`mutationFn` test-rot errors,
  leaving these 10.
- **Remaining errors (all pre-existing, none caused by Phase 10 plans):**
  - `src/lib/__tests__/api.test.ts(84)` — TS2339 `get` on type `never`
  - `src/routes/__root.tsx(16)` — TS2322 ThemeProviderProps `attribute`
  - `src/routes/_auth/bootstrap/-components/bootstrap-form.tsx(57)` — TS2741
    `org_name` missing (BootstrapRequest shape drift)
  - `src/routes/_auth/invite/-components/invitation-accept-form.tsx(13)` —
    TS2724 `useSearchParams` removed from @tanstack/react-router v1.170
  - `src/routes/_auth/password-reset/-components/password-reset-request-form.tsx(45)` —
    TS2345 navigate argument mismatch
  - `src/routes/_authenticated/org-hierarchy/-components/dialogs/unit-detail-panel.tsx(197)` —
    TS2345 string|undefined vs string
- **Resolved in 10-03 (contracts wrap):** the 4 contracts errors that were
  assigned to Plan 10-03 are fixed (contract-detail.tsx: currency
  `v ?? undefined`, `Customer.name` → `Customer.company_name`;
  create-contract-dialog.tsx: navigate `search: { from: "owned" }`, Select
  `onValueChange` wrapper) — see 10-03-SUMMARY.md.
- **Impact:** The plan's `typecheck exits 0` acceptance criterion is
  technically unattainable at baseline; Phase 10 plans contribute zero new
  errors. The 6 remaining errors touch out-of-phase surfaces (`_auth` forms,
  __root theme provider, api test, org-hierarchy unit-detail-panel) — fixing
  them is out of scope for Phase 10's IA work.

## 2. E2E seed date rollover (2026-08-01) — pre-existing, unrelated suites

- **Discovered:** 2026-08-01, Plan 10-05 full e2e run
- **Symptom:** 3 pre-existing suites fail after the calendar rolled to August:
  - `time-entries.spec.ts` "list tab shows seeded rows for all six workflow
    states" — seeds hard-code July dates (2026-07-15..20); the list view
    defaults to the current month (August) → "No time entries in this period."
  - `expenses.spec.ts` "list tab shows seeded rows with categories" — seeds
    July dates (2026-07-10..14); same default-month gap.
  - `error-boundary.spec.ts` "Try again re-runs the loader and recovers to
    data" — asserts `seeded-draft-*` visible after recovery; the seed's July
    date falls outside the default August month.
- **Baseline:** these suites passed 42/42 in Plan 10-04 on 2026-07-31 — the
  failure is the month rollover in the hard-coded seeds, NOT a Phase 10
  regression. Plan 10-05's own `e2e/approvals.spec.ts` seeds **current-month**
  (August) dates and passes 4/4; it does not depend on the July seeds.
- **Impact:** The "full e2e suite exits 0" criterion is technically
  unattainable at baseline on 2026-08-01 until the list-view seeds use
  current-month dates. The approvals spec — this plan's deliverable — is
  green. Fixing the July-hardcoded seeds in three unrelated suites is out of
  scope for Phase 10's IA work; a future maintenance plan should make
  `seedTimeEntries`/`seedExpenses` compute dates relative to the current
  month.
