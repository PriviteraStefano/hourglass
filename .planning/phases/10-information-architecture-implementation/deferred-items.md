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
